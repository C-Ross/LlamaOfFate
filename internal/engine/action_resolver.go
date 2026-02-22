package engine

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

// NarrativeProvider generates narrative text and records conversation history.
// SceneManager implements this interface so that action resolution can
// produce narrative without a direct dependency on scene-level state
// (conversation context, LLM prompt rendering, etc.).
type NarrativeProvider interface {
	// GenerateActionNarrative produces LLM-generated narrative for a resolved action.
	GenerateActionNarrative(ctx context.Context, a *action.Action) (string, error)
	// BuildMechanicalNarrative produces a fallback narrative from action data
	// when the LLM is unavailable.
	BuildMechanicalNarrative(a *action.Action) string
	// RecordConversationEntry records an exchange in the conversation history
	// so that subsequent LLM prompts have context.
	RecordConversationEntry(playerInput, gmResponse, interactionType string)
}

// diceFacesToInts converts a [4]FateDie array to a []int slice for event serialization.
func diceFacesToInts(dice [4]dice.FateDie) []int {
	out := make([]int, 4)
	for i, d := range dice {
		out[i] = int(d)
	}
	return out
}

// ActionResolver handles the generic action resolution pipeline: dice rolling,
// invoke loops, mid-flow prompts, narrative coordination, and applying
// mechanical effects (create-advantage aspects, attack damage delegation).
// Both SceneManager (for player actions) and ConflictManager (for NPC defense
// invokes) use ActionResolver.
type ActionResolver struct {
	// Shared dependencies — set once at construction.
	roller          dice.DiceRoller
	characters      CharacterResolver
	narrative       NarrativeProvider
	sessionLogger   *session.Logger
	aspectGenerator AspectGenerator

	// ConflictManager — used for conflict-specific side effects (initiate,
	// escalate, damage, advance turns). May be nil in tests.
	conflict *ConflictManager

	// Per-scene state — wired by SceneManager.StartScene.
	player       *character.Character
	currentScene *scene.Scene

	// Action-resolution mutable state — reset each scene.
	pendingInvoke  *invokeState
	pendingMidFlow *midFlowState
}

// newActionResolver creates an ActionResolver with the given shared dependencies.
// The conflict back-reference and NarrativeProvider are wired separately after
// construction to break the circular dependency.
func newActionResolver(roller dice.DiceRoller, characters CharacterResolver, ag AspectGenerator) *ActionResolver {
	return &ActionResolver{
		roller:          roller,
		characters:      characters,
		aspectGenerator: ag,
	}
}

// SetNarrativeProvider wires the narrative dependency. Called by SceneManager
// after construction.
func (ar *ActionResolver) SetNarrativeProvider(np NarrativeProvider) {
	ar.narrative = np
}

// setSceneState wires per-scene references. Called by SceneManager.StartScene.
func (ar *ActionResolver) setSceneState(s *scene.Scene, player *character.Character) {
	ar.currentScene = s
	ar.player = player
}

// resetState clears per-scene action-resolution state.
func (ar *ActionResolver) resetState() {
	ar.pendingInvoke = nil
	ar.pendingMidFlow = nil
}

// setSessionLogger updates the session logger (may be called after construction).
func (ar *ActionResolver) setSessionLogger(logger *session.Logger) {
	ar.sessionLogger = logger
}

// RecordConversation is a convenience method that delegates to the
// NarrativeProvider's RecordConversationEntry. ConflictManager uses this
// to record conflict-specific dialogue without directly depending on the
// NarrativeProvider interface.
func (ar *ActionResolver) RecordConversation(playerInput, gmResponse, interactionType string) {
	if ar.narrative != nil {
		ar.narrative.RecordConversationEntry(playerInput, gmResponse, interactionType)
	}
}

// --- Accessor methods ---

// HasPendingInvoke returns true when the engine is waiting for an InvokeResponse.
func (ar *ActionResolver) HasPendingInvoke() bool {
	return ar.pendingInvoke != nil
}

// HasPendingMidFlow returns true when the engine is waiting for a MidFlowResponse.
func (ar *ActionResolver) HasPendingMidFlow() bool {
	return ar.pendingMidFlow != nil
}

// PendingMidFlowEvent returns the InputRequestEvent for the pending mid-flow
// prompt. Only valid when HasPendingMidFlow() is true.
func (ar *ActionResolver) PendingMidFlowEvent() InputRequestEvent {
	if ar.pendingMidFlow == nil {
		return InputRequestEvent{}
	}
	return ar.pendingMidFlow.event
}

// --- Action resolution pipeline ---

// resolveAction fully resolves a parsed action: rolls dice, checks for
// conflict initiation/escalation, offers invoke opportunities, generates
// narrative, applies mechanical effects, and advances conflict turns.
func (ar *ActionResolver) resolveAction(ctx context.Context, parsedAction *action.Action) ([]GameEvent, bool) {
	var events []GameEvent

	// Check if this action should initiate or escalate a conflict
	if parsedAction.Type == action.Attack {
		actionConflictType := core.ConflictTypeForSkill(parsedAction.Skill)

		if ar.currentScene.ActiveSceneType() == scene.SceneTypeNone {
			// Auto-initiate conflict for attack actions
			if err := ar.conflict.initiateConflict(actionConflictType, ar.player.ID); err != nil {
				slog.Warn("Failed to auto-initiate conflict",
					"component", componentSceneManager,
					"error", err)
			} else {
				events = append(events, ConflictStartEvent{
					ConflictType:  string(actionConflictType),
					InitiatorName: ar.player.Name,
					Participants:  ar.conflict.getParticipantInfo(),
				})
				ar.narrative.RecordConversationEntry("",
					fmt.Sprintf("[%s conflict initiated by %s]", actionConflictType, ar.player.Name),
					inputTypeConflict)
			}
		} else if ar.currentScene.ConflictState.Type != actionConflictType {
			// Escalate conflict if type changes
			escalateEvents := ar.conflict.handleConflictEscalation(actionConflictType)
			events = append(events, escalateEvents...)
		}
	}

	// Get character's skill level
	skillLevel := ar.player.GetSkill(parsedAction.Skill)

	// Calculate total bonus
	totalBonus := int(skillLevel) + parsedAction.CalculateBonus()

	// Roll dice
	result := ar.roller.RollWithModifier(dice.Mediocre, totalBonus)

	// For attacks against characters, use active defense instead of static difficulty
	var defenseResult *dice.CheckResult
	var targetChar *character.Character
	if parsedAction.Type == action.Attack && parsedAction.Target != "" {
		targetChar = ar.characters.ResolveCharacter(parsedAction.Target)
		if targetChar == nil {
			slog.Warn("Attack target not found, action aborted",
				"component", componentSceneManager,
				"target", parsedAction.Target)
			events = append(events, SystemMessageEvent{
				Message: fmt.Sprintf("Could not find target '%s' — try again.", parsedAction.Target),
			})
			return events, false
		}
		var defEvent DefenseRollEvent
		defenseResult, defEvent = ar.rollTargetDefense(targetChar, parsedAction.Skill)
		events = append(events, defEvent)
		parsedAction.Difficulty = defenseResult.FinalValue
	}

	// For non-attack actions with active NPC opposition (Fate Core: active vs passive),
	// roll the NPC's skill as opposition instead of using a flat difficulty.
	var oppositionResult *dice.CheckResult
	var opposingChar *character.Character
	if parsedAction.Type != action.Attack && parsedAction.OpposingNPCID != "" {
		opposingChar = ar.characters.ResolveCharacter(parsedAction.OpposingNPCID)
		if opposingChar != nil {
			oppositionLevel := opposingChar.GetSkill(parsedAction.OpposingSkill)
			oppositionResult = ar.roller.RollWithModifier(dice.Mediocre, int(oppositionLevel))
			parsedAction.Difficulty = oppositionResult.FinalValue
			events = append(events, DefenseRollEvent{
				DefenderName: opposingChar.Name,
				Skill:        parsedAction.OpposingSkill,
				Result:       oppositionResult.FinalValue.String(),
				DiceFaces:    diceFacesToInts(oppositionResult.Roll.Dice),
			})
			slog.Debug("Active NPC opposition roll",
				"component", componentSceneManager,
				"npc", opposingChar.Name,
				"skill", parsedAction.OpposingSkill,
				"skill_level", int(oppositionLevel),
				"roll_result", oppositionResult.FinalValue.String())
		} else {
			slog.Warn("Active opposition NPC not found, falling back to passive difficulty",
				"component", componentSceneManager,
				"npc_id", parsedAction.OpposingNPCID)
		}
	}

	// Display initial result
	var resultString string
	var defenderName string
	if defenseResult != nil && targetChar != nil {
		defenderName = targetChar.Name
		resultString = fmt.Sprintf("%s (Total: %s vs %s's Defense %s)",
			result.String(), result.FinalValue.String(), targetChar.Name, defenseResult.FinalValue.String())
	} else if oppositionResult != nil && opposingChar != nil {
		defenderName = opposingChar.Name
		resultString = fmt.Sprintf("%s (Total: %s vs %s's %s %s)",
			result.String(), result.FinalValue.String(),
			opposingChar.Name, parsedAction.OpposingSkill, oppositionResult.FinalValue.String())
	} else {
		resultString = fmt.Sprintf("%s (Total: %s vs Difficulty %s)",
			result.String(), result.FinalValue.String(), parsedAction.Difficulty.String())
	}
	initialOutcome := result.CompareAgainst(parsedAction.Difficulty)
	events = append(events, ActionResultEvent{
		Skill:        parsedAction.Skill,
		SkillRank:    skillLevel.Name(),
		SkillBonus:   int(skillLevel),
		Bonuses:      parsedAction.CalculateBonus(),
		Result:       resultString,
		Outcome:      initialOutcome.Type.String(),
		DiceFaces:    diceFacesToInts(result.Roll.Dice),
		Total:        int(result.FinalValue),
		TotalRank:    result.FinalValue.Name(),
		Difficulty:   int(parsedAction.Difficulty),
		DiffRank:     parsedAction.Difficulty.Name(),
		DefenderName: defenderName,
	})

	// Log the dice roll
	if ar.sessionLogger != nil {
		logData := map[string]any{
			"skill":       parsedAction.Skill,
			"skill_level": int(skillLevel),
			"bonus":       parsedAction.CalculateBonus(),
			"roll_result": result.String(),
			"final_value": int(result.FinalValue),
			"difficulty":  int(parsedAction.Difficulty),
			"outcome":     initialOutcome.Type.String(),
			"shifts":      initialOutcome.Shifts,
		}
		if opposingChar != nil {
			logData["opposing_npc"] = opposingChar.Name
			logData["opposing_skill"] = parsedAction.OpposingSkill
		}
		ar.sessionLogger.Log("dice_roll", logData)
	}

	// Build the continuation that runs after invokes complete.
	// Capture the variables needed by the post-invoke logic.
	capturedInitialOutcome := initialOutcome
	capturedTargetChar := targetChar
	capturedParsedAction := parsedAction

	finish := func(finishCtx context.Context, finalResult *dice.CheckResult, accEvents []GameEvent) []GameEvent {
		return ar.finishResolveAction(finishCtx, finalResult, capturedParsedAction, capturedInitialOutcome, capturedTargetChar, accEvents)
	}

	// Post-roll invoke opportunity via event-driven loop
	return ar.beginInvokeLoop(ctx, result, parsedAction.Difficulty, parsedAction, false, events, finish)
}

// finishResolveAction is the continuation called after the invoke loop completes.
// It determines the final outcome, generates narrative, applies effects, and
// advances conflict turns. Returns all events for the InputResult.
func (ar *ActionResolver) finishResolveAction(
	ctx context.Context,
	result *dice.CheckResult,
	parsedAction *action.Action,
	initialOutcome *dice.Outcome,
	targetChar *character.Character,
	events []GameEvent,
) []GameEvent {
	parsedAction.CheckResult = result

	// Determine final outcome after invokes
	outcome := result.CompareAgainst(parsedAction.Difficulty)
	parsedAction.Outcome = outcome

	// Display updated outcome if it changed
	if outcome.Type != initialOutcome.Type {
		events = append(events, OutcomeChangedEvent{
			FinalOutcome: outcome.Type.String(),
		})
	}

	// Generate narrative with error handling
	narrative, err := ar.narrative.GenerateActionNarrative(ctx, parsedAction)
	if err != nil {
		slog.Error("Action narrative generation failed",
			"component", componentSceneManager,
			"action_id", parsedAction.ID,
			"error", err)
		narrative = ar.narrative.BuildMechanicalNarrative(parsedAction)
	}
	events = append(events, NarrativeEvent{Text: narrative})

	// Log the narrative
	if ar.sessionLogger != nil {
		ar.sessionLogger.Log("narrative", map[string]any{
			"text":    narrative,
			"action":  parsedAction.Type.String(),
			"outcome": outcome.Type.String(),
		})
	}

	// Apply mechanical effects based on action type and outcome
	effectEvents := ar.applyActionEffects(ctx, parsedAction, targetChar)
	events = append(events, effectEvents...)

	// If we're in a conflict, advance turn and process NPC turns
	if ar.currentScene.ActiveSceneType() == scene.SceneTypeConflict {
		turnEvents, _ := ar.conflict.advanceConflictTurns(ctx)
		events = append(events, turnEvents...)
	}

	return events
}

// rollTargetDefense rolls an active defense for a target character
// and returns the roll result plus a DefenseRollEvent.
func (ar *ActionResolver) rollTargetDefense(target *character.Character, attackSkill string) (*dice.CheckResult, DefenseRollEvent) {
	// Determine defense skill based on attack skill type
	defenseSkill := core.DefenseSkillForAttack(attackSkill)
	defenseLevel := target.GetSkill(defenseSkill)

	// Roll defense
	defenseRoll := ar.roller.RollWithModifier(dice.Mediocre, int(defenseLevel))

	event := DefenseRollEvent{
		DefenderName: target.Name,
		Skill:        defenseSkill,
		Result:       defenseRoll.FinalValue.String(),
		DiceFaces:    diceFacesToInts(defenseRoll.Roll.Dice),
	}

	return defenseRoll, event
}

// gatherInvokableAspects collects all aspects available for the player to invoke.
func (ar *ActionResolver) gatherInvokableAspects(usedAspects map[string]bool) []InvokableAspect {
	var aspects []InvokableAspect

	// Character aspects (High Concept, Trouble, Other)
	for _, aspectText := range ar.player.Aspects.GetAll() {
		if aspectText == "" {
			continue
		}
		aspects = append(aspects, InvokableAspect{
			Name:        aspectText,
			Source:      "character",
			SourceID:    ar.player.ID,
			FreeInvokes: 0, // Character aspects don't have free invokes
			AlreadyUsed: usedAspects[aspectText],
		})
	}

	// Player's consequences (can be invoked against self for +2)
	for _, consequence := range ar.player.Consequences {
		if consequence.Aspect == "" {
			continue
		}
		aspects = append(aspects, InvokableAspect{
			Name:        consequence.Aspect,
			Source:      "consequence",
			SourceID:    consequence.ID,
			FreeInvokes: 0,
			AlreadyUsed: usedAspects[consequence.Aspect],
		})
	}

	// Situation aspects
	for _, sitAspect := range ar.currentScene.SituationAspects {
		if sitAspect.Aspect == "" {
			continue
		}
		aspects = append(aspects, InvokableAspect{
			Name:        sitAspect.Aspect,
			Source:      "situation",
			SourceID:    sitAspect.ID,
			FreeInvokes: sitAspect.FreeInvokes,
			AlreadyUsed: usedAspects[sitAspect.Aspect],
		})
	}

	return aspects
}

// --- Action effect application ---

// applyActionEffects applies mechanical effects based on action results
// and returns composite events describing what happened.
// Create Advantage effects (aspect creation) are handled here directly.
// Attack damage is delegated to ConflictManager since it touches
// conflict-specific state (stress, consequences, taken-out, victory).
func (ar *ActionResolver) applyActionEffects(ctx context.Context, parsedAction *action.Action, target *character.Character) []GameEvent {
	if parsedAction.Outcome == nil {
		return nil
	}

	var events []GameEvent

	switch parsedAction.Type {
	case action.CreateAdvantage:
		if parsedAction.IsSuccess() {
			aspectName, freeInvokes := ar.generateAspectName(ctx, parsedAction)

			situationAspect := scene.NewSituationAspect(
				fmt.Sprintf("aspect-%d", time.Now().UnixNano()),
				aspectName,
				ar.player.ID,
				freeInvokes,
			)

			ar.currentScene.AddSituationAspect(situationAspect)
			events = append(events, AspectCreatedEvent{
				AspectName:  aspectName,
				FreeInvokes: freeInvokes,
			})
		} else if parsedAction.Outcome.Type == dice.Tie {
			// On a tie, player gets a boost instead of a full aspect (Fate Core SRD).
			aspectName, _ := ar.generateAspectName(ctx, parsedAction)
			events = append(events, ar.createBoost(aspectName, ar.player.ID))
		}

	case action.Attack:
		if target == nil {
			slog.Warn("Attack has no valid target, cannot apply damage",
				"component", componentSceneManager,
				"action_id", parsedAction.ID,
				"target", parsedAction.Target)
			events = append(events, PlayerAttackResultEvent{
				TargetMissing: true,
				TargetHint:    parsedAction.Target,
			})
			return events
		}

		if parsedAction.IsSuccess() {
			shifts := parsedAction.Outcome.Shifts
			if shifts < 1 {
				shifts = 1 // Minimum 1 shift on success
			}

			// Determine stress type based on attack skill
			stressType := core.StressTypeForAttack(parsedAction.Skill)

			events = append(events, PlayerAttackResultEvent{
				TargetName: target.Name,
				Shifts:     shifts,
			})

			// Delegate damage application to ConflictManager (conflict-specific state)
			dmgEvent := ar.conflict.applyDamageToTarget(ctx, target, shifts, stressType)
			events = append(events, dmgEvent)
		} else if parsedAction.Outcome.Type == dice.Tie {
			// On a tie, attacker gets a boost (no damage dealt) — Fate Core SRD Attack.
			events = append(events, PlayerAttackResultEvent{
				TargetName: target.Name,
				IsTie:      true,
			})
			boostName := ar.generateBoostName(ctx, ar.player, parsedAction.Skill, parsedAction.Description, "Fleeting Opening")
			events = append(events, ar.createBoost(boostName, ar.player.ID))
		} else if parsedAction.Outcome.Type == dice.Failure && parsedAction.Outcome.Shifts <= -3 {
			// Target defended with style — defender gets a boost (Fate Core SRD Defend).
			defDesc := fmt.Sprintf("defending against %s's attack", ar.player.Name)
			defSkill := core.DefenseSkillForAttack(parsedAction.Skill)
			boostName := ar.generateBoostName(ctx, target, defSkill, defDesc, "Deflected with Ease")
			events = append(events, ar.createBoost(boostName, target.ID))
		}

	case action.Overcome:
		if parsedAction.IsSuccessWithStyle() {
			// Overcome SWS grants a boost in addition to achieving the goal (Fate Core SRD).
			boostName := ar.generateBoostName(ctx, ar.player, parsedAction.Skill, parsedAction.Description, "Strong Momentum")
			events = append(events, ar.createBoost(boostName, ar.player.ID))
		}
	}

	return events
}

// createBoost creates a boost aspect on the scene and returns an AspectCreatedEvent.
// A boost is a temporary situation aspect with 1 free invoke that is removed
// after its free invoke is consumed (per Fate Core SRD: Types of Aspects — Boosts).
func (ar *ActionResolver) createBoost(name, createdByID string) AspectCreatedEvent {
	boost := scene.NewBoost(fmt.Sprintf("boost-%d", time.Now().UnixNano()), name, createdByID)
	ar.currentScene.AddSituationAspect(boost)
	return AspectCreatedEvent{AspectName: name, FreeInvokes: 1, IsBoost: true}
}

// generateBoostName generates a contextual name for a boost using the LLM aspect generator.
// It constructs a synthetic CreateAdvantage action with a Tie outcome so the existing
// aspect generation prompt can produce a thematic boost name.
// Falls back to the provided fallback string if no generator is available or the call fails.
func (ar *ActionResolver) generateBoostName(ctx context.Context, char *character.Character, skill, description, fallback string) string {
	if ar.aspectGenerator == nil {
		return fallback
	}

	existingAspects := make([]string, 0, len(ar.currentScene.SituationAspects))
	for _, sa := range ar.currentScene.SituationAspects {
		existingAspects = append(existingAspects, sa.Aspect)
	}

	// Synthetic CreateAdvantage action with a Tie outcome so the prompt knows this is a boost.
	syntheticAction := action.NewAction(
		fmt.Sprintf("boost-gen-%d", time.Now().UnixNano()),
		char.ID, action.CreateAdvantage, skill, description,
	)
	syntheticAction.Outcome = &dice.Outcome{Type: dice.Tie, Shifts: 0}

	req := prompt.AspectGenerationRequest{
		Character:       char,
		Action:          syntheticAction,
		Outcome:         syntheticAction.Outcome,
		Context:         ar.currentScene.Description,
		TargetType:      "situation",
		ExistingAspects: existingAspects,
	}

	genCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := ar.aspectGenerator.GenerateAspect(genCtx, req)
	if err != nil || resp == nil || resp.AspectText == "" {
		slog.Warn("Failed to generate boost name via LLM, using fallback",
			"component", componentSceneManager,
			"error", err)
		return fallback
	}

	return resp.AspectText
}

// generateAspectName uses the LLM to generate a creative aspect name for Create an Advantage.
// Falls back to a simple description-based name if the LLM is unavailable or fails.
func (ar *ActionResolver) generateAspectName(ctx context.Context, parsedAction *action.Action) (string, int) {
	// Determine free invokes based on outcome
	freeInvokes := 1
	if parsedAction.IsSuccessWithStyle() {
		freeInvokes = 2
	}

	// Fallback name if LLM generation fails
	fallbackName := fmt.Sprintf("Advantage from %s", parsedAction.Description)

	// If no aspect generator available, use fallback
	if ar.aspectGenerator == nil {
		slog.Debug("No aspect generator available, using fallback name",
			"component", componentSceneManager)
		return fallbackName, freeInvokes
	}

	// Gather existing aspects for context
	existingAspects := make([]string, 0)
	for _, sa := range ar.currentScene.SituationAspects {
		existingAspects = append(existingAspects, sa.Aspect)
	}

	// Build the request
	req := prompt.AspectGenerationRequest{
		Character:       ar.player,
		Action:          parsedAction,
		Outcome:         parsedAction.Outcome,
		Context:         ar.currentScene.Description,
		TargetType:      "situation",
		ExistingAspects: existingAspects,
	}

	// Generate aspect via LLM with timeout derived from the caller's context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := ar.aspectGenerator.GenerateAspect(ctx, req)
	if err != nil {
		slog.Warn("Failed to generate aspect via LLM, using fallback",
			"component", componentSceneManager,
			"error", err)
		return fallbackName, freeInvokes
	}

	// Use generated aspect text, or fallback if empty
	if response.AspectText == "" {
		return fallbackName, freeInvokes
	}

	// Override free invokes from LLM response if it makes sense
	if response.FreeInvokes > 0 {
		freeInvokes = response.FreeInvokes
	}

	slog.Debug("Generated aspect via LLM",
		"component", componentSceneManager,
		"aspect", response.AspectText,
		"freeInvokes", freeInvokes,
		"reasoning", response.Reasoning)

	return response.AspectText, freeInvokes
}
