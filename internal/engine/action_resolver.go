package engine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
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

// ActionResolver handles the generic action resolution pipeline: dice rolling,
// invoke loops, mid-flow prompts, and narrative coordination. Both SceneManager
// (for player actions) and ConflictManager (for NPC defense invokes) use
// ActionResolver.
type ActionResolver struct {
	// Shared dependencies — set once at construction.
	roller        dice.DiceRoller
	characters    CharacterResolver
	narrative     NarrativeProvider
	sessionLogger *session.Logger

	// ConflictManager — used for conflict-specific side effects (initiate,
	// escalate, apply effects, advance turns). May be nil in tests.
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
func newActionResolver(roller dice.DiceRoller, characters CharacterResolver) *ActionResolver {
	return &ActionResolver{
		roller:     roller,
		characters: characters,
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

		if !ar.currentScene.IsConflict {
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

	// Display initial result
	var resultString string
	var defenderName string
	if defenseResult != nil && targetChar != nil {
		defenderName = targetChar.Name
		resultString = fmt.Sprintf("%s (Total: %s vs %s's Defense %s)",
			result.String(), result.FinalValue.String(), targetChar.Name, defenseResult.FinalValue.String())
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
		ar.sessionLogger.Log("dice_roll", map[string]any{
			"skill":       parsedAction.Skill,
			"skill_level": int(skillLevel),
			"bonus":       parsedAction.CalculateBonus(),
			"roll_result": result.String(),
			"final_value": int(result.FinalValue),
			"difficulty":  int(parsedAction.Difficulty),
			"outcome":     initialOutcome.Type.String(),
			"shifts":      initialOutcome.Shifts,
		})
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
	effectEvents := ar.conflict.applyActionEffects(ctx, parsedAction, targetChar)
	events = append(events, effectEvents...)

	// If we're in a conflict, advance turn and process NPC turns
	if ar.currentScene.IsConflict {
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
