package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// TakenOutResult represents the outcome classification of being taken out
type TakenOutResult int

const (
	TakenOutContinue   TakenOutResult = iota // Continue in same scene (knocked down, stunned, etc.)
	TakenOutTransition                       // Transition to new scene (captured, driven out, etc.)
	TakenOutGameOver                         // Game ending (death, permanent incapacitation)
)

// parseConflictMarker extracts a conflict trigger from LLM response and returns cleaned text
func (sm *SceneManager) parseConflictMarker(response string) (*prompt.ConflictTrigger, string) {
	return prompt.ParseConflictMarker(response)
}

// parseConflictEndMarker extracts a conflict resolution from LLM response and returns cleaned text
func (sm *SceneManager) parseConflictEndMarker(response string) (*prompt.ConflictResolution, string) {
	return prompt.ParseConflictEndMarker(response)
}

// initiateConflict starts a conflict with all characters in the scene
func (sm *SceneManager) initiateConflict(conflictType scene.ConflictType, initiatorID string) error {
	if sm.currentScene.IsConflict {
		return fmt.Errorf("already in a conflict")
	}

	// Validate the initiator is a real character in this scene
	if sm.engine.GetCharacter(initiatorID) == nil {
		slog.Warn("Conflict trigger rejected: initiator ID does not match any character",
			"component", componentSceneManager,
			"initiator", initiatorID)
		return fmt.Errorf("initiator %q is not a known character", initiatorID)
	}

	// Check if the initiator was taken out earlier in this scene
	if sm.currentScene.IsCharacterTakenOut(initiatorID) {
		slog.Debug("Conflict initiator was previously taken out this scene",
			"component", componentSceneManager,
			"initiator", initiatorID)
		return fmt.Errorf("initiator %s was taken out this scene", initiatorID)
	}

	// Build participants from all characters in the scene
	participants := make([]scene.ConflictParticipant, 0)

	for _, charID := range sm.currentScene.Characters {
		char := sm.engine.GetCharacter(charID)
		if char == nil {
			continue
		}

		// Skip characters that have been taken out earlier in this scene
		if sm.currentScene.IsCharacterTakenOut(charID) {
			slog.Debug("Skipping taken-out character for conflict",
				"component", componentSceneManager,
				"character", charID)
			continue
		}

		// Calculate initiative based on conflict type
		initiative := sm.calculateInitiative(char, conflictType)

		participants = append(participants, scene.ConflictParticipant{
			CharacterID: charID,
			Initiative:  initiative,
			Status:      scene.StatusActive,
		})
	}

	if len(participants) < 2 {
		return fmt.Errorf("conflict requires at least 2 participants")
	}

	sm.currentScene.StartConflictWithInitiator(conflictType, participants, initiatorID)

	slog.Info("Conflict initiated",
		"component", componentSceneManager,
		"type", conflictType,
		"initiator", initiatorID,
		"participants", len(participants))

	return nil
}

// resolveConflictPeacefully ends a conflict through non-violent means
func (sm *SceneManager) resolveConflictPeacefully(reason string) string {
	if !sm.currentScene.IsConflict {
		return ""
	}

	// Format reason for display
	reasonMessage := ""
	switch reason {
	case "surrender":
		reasonMessage = "Your opponent surrenders!"
	case "agreement":
		reasonMessage = "You've reached an agreement!"
	case "retreat":
		reasonMessage = "Your opponent retreats!"
	case "resolved":
		reasonMessage = "The conflict has been resolved!"
	default:
		reasonMessage = "The conflict ends!"
	}

	sm.clearConflictStress()
	sm.currentScene.EndConflict()

	slog.Info("Conflict resolved peacefully",
		"component", componentSceneManager,
		"reason", reason)

	return reasonMessage
}

// clearConflictStress clears stress for all conflict participants.
// Per Fate Core: "After a conflict, when you get a minute to breathe,
// any stress boxes you checked off become available for your use again."
func (sm *SceneManager) clearConflictStress() {
	if sm.currentScene.ConflictState == nil {
		return
	}

	for _, p := range sm.currentScene.ConflictState.Participants {
		char := sm.engine.GetCharacter(p.CharacterID)
		if char != nil {
			char.ClearAllStress()
		}
	}

	slog.Info("Cleared stress for all conflict participants",
		"component", componentSceneManager,
		"participants", len(sm.currentScene.ConflictState.Participants))
}

// calculateInitiative returns the initiative value for a character based on conflict type
func (sm *SceneManager) calculateInitiative(char *character.Character, conflictType scene.ConflictType) int {
	return core.CalculateInitiative(char, conflictType)
}

// sortInitiativeOrder sorts the initiative order by participant initiative values
func (sm *SceneManager) sortInitiativeOrder() {
	if sm.currentScene.ConflictState == nil {
		return
	}

	sm.currentScene.ConflictState.SortByInitiative()
}

// recalculateInitiative recalculates initiative for all participants based on conflict type
func (sm *SceneManager) recalculateInitiative(conflictType scene.ConflictType) {
	if sm.currentScene.ConflictState == nil {
		return
	}

	for i := range sm.currentScene.ConflictState.Participants {
		p := &sm.currentScene.ConflictState.Participants[i]
		char := sm.engine.GetCharacter(p.CharacterID)
		if char != nil {
			p.Initiative = sm.calculateInitiative(char, conflictType)
		}
	}

	sm.sortInitiativeOrder()
}

// handleConflictEscalation changes the conflict type and recalculates initiative.
// Returns the escalation event.
func (sm *SceneManager) handleConflictEscalation(newType scene.ConflictType) []GameEvent {
	if !sm.currentScene.IsConflict {
		return nil
	}

	currentType := sm.currentScene.ConflictState.Type
	if currentType == newType {
		return nil
	}

	sm.currentScene.EscalateConflict(newType)
	sm.recalculateInitiative(newType)

	return []GameEvent{ConflictEscalationEvent{
		FromType:        string(currentType),
		ToType:          string(newType),
		TriggerCharName: sm.player.Name,
	}}
}

// advanceConflictTurns advances through turns and processes NPC actions until it's the player's turn.
// Returns all events generated during NPC turns.
func (sm *SceneManager) advanceConflictTurns(ctx context.Context) []GameEvent {
	if !sm.currentScene.IsConflict || sm.currentScene.ConflictState == nil {
		return nil
	}

	var events []GameEvent

	// Advance past the player's turn
	sm.currentScene.NextTurn()

	// Process NPC turns until we get back to the player or conflict ends
	for sm.currentScene.IsConflict {
		currentActor := sm.currentScene.GetCurrentActor()
		if currentActor == "" {
			break
		}

		// If it's the player's turn, stop and let them act
		if currentActor == sm.player.ID {
			events = append(events, TurnAnnouncementEvent{CharacterName: sm.player.Name, TurnNumber: sm.currentScene.ConflictState.Round, IsPlayer: true})
			break
		}

		// Process NPC turn
		npcEvents := sm.processNPCTurn(ctx, currentActor)
		events = append(events, npcEvents...)

		// Advance to next turn
		sm.currentScene.NextTurn()
	}

	return events
}

// getParticipantInfo returns information about all conflict participants for display
func (sm *SceneManager) getParticipantInfo() []ConflictParticipantInfo {
	if sm.currentScene.ConflictState == nil {
		return nil
	}

	info := make([]ConflictParticipantInfo, 0, len(sm.currentScene.ConflictState.Participants))
	for _, p := range sm.currentScene.ConflictState.Participants {
		name := p.CharacterID
		if char := sm.engine.GetCharacter(p.CharacterID); char != nil {
			name = char.Name
		}
		info = append(info, ConflictParticipantInfo{
			CharacterID:   p.CharacterID,
			CharacterName: name,
			Initiative:    p.Initiative,
			IsPlayer:      p.CharacterID == sm.player.ID,
		})
	}
	return info
}

// resolveAction fully resolves a parsed action
func (sm *SceneManager) resolveAction(ctx context.Context, parsedAction *action.Action) ([]GameEvent, bool) {
	var events []GameEvent

	// Check if this action should initiate or escalate a conflict
	if parsedAction.Type == action.Attack {
		actionConflictType := core.ConflictTypeForSkill(parsedAction.Skill)

		if !sm.currentScene.IsConflict {
			// Auto-initiate conflict for attack actions
			if err := sm.initiateConflict(actionConflictType, sm.player.ID); err != nil {
				slog.Warn("Failed to auto-initiate conflict",
					"component", componentSceneManager,
					"error", err)
			} else {
				events = append(events, ConflictStartEvent{
					ConflictType:  string(actionConflictType),
					InitiatorName: sm.player.Name,
					Participants:  sm.getParticipantInfo(),
				})
			}
		} else if sm.currentScene.ConflictState.Type != actionConflictType {
			// Escalate conflict if type changes
			escalateEvents := sm.handleConflictEscalation(actionConflictType)
			events = append(events, escalateEvents...)
		}
	}

	// Get character's skill level
	skillLevel := sm.player.GetSkill(parsedAction.Skill)

	// Calculate total bonus
	totalBonus := int(skillLevel) + parsedAction.CalculateBonus()

	// Roll dice
	result := sm.roller.RollWithModifier(dice.Mediocre, totalBonus)

	// For attacks against characters, use active defense instead of static difficulty
	var defenseResult *dice.CheckResult
	var targetChar *character.Character
	if parsedAction.Type == action.Attack && parsedAction.Target != "" {
		targetChar = sm.engine.ResolveCharacter(parsedAction.Target)
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
		defenseResult, defEvent = sm.rollTargetDefense(targetChar, parsedAction.Skill)
		events = append(events, defEvent)
		parsedAction.Difficulty = defenseResult.FinalValue
	}

	// Display initial result
	var resultString string
	if defenseResult != nil && targetChar != nil {
		resultString = fmt.Sprintf("%s (Total: %s vs %s's Defense %s)",
			result.String(), result.FinalValue.String(), targetChar.Name, defenseResult.FinalValue.String())
	} else {
		resultString = fmt.Sprintf("%s (Total: %s vs Difficulty %s)",
			result.String(), result.FinalValue.String(), parsedAction.Difficulty.String())
	}
	initialOutcome := result.CompareAgainst(parsedAction.Difficulty)
	events = append(events, ActionResultEvent{
		Skill:      parsedAction.Skill,
		SkillLevel: fmt.Sprintf("%s (%+d)", skillLevel.String(), int(skillLevel)),
		Bonuses:    parsedAction.CalculateBonus(),
		Result:     resultString,
		Outcome:    initialOutcome.Type.String(),
	})

	// Log the dice roll
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("dice_roll", map[string]any{
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
		return sm.finishResolveAction(finishCtx, finalResult, capturedParsedAction, capturedInitialOutcome, capturedTargetChar, accEvents)
	}

	// Post-roll invoke opportunity via event-driven loop
	return sm.beginInvokeLoop(ctx, result, parsedAction.Difficulty, parsedAction, false, events, finish)
}

// finishResolveAction is the continuation called after the invoke loop completes.
// It determines the final outcome, generates narrative, applies effects, and
// advances conflict turns. Returns all events for the InputResult.
func (sm *SceneManager) finishResolveAction(
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
	narrative, err := sm.generateActionNarrative(ctx, parsedAction)
	if err != nil {
		slog.Error("Action narrative generation failed",
			"component", componentSceneManager,
			"action_id", parsedAction.ID,
			"error", err)
		narrative = sm.buildMechanicalNarrative(parsedAction)
	}
	events = append(events, NarrativeEvent{Text: narrative})

	// Log the narrative
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("narrative", map[string]any{
			"text":    narrative,
			"action":  parsedAction.Type.String(),
			"outcome": outcome.Type.String(),
		})
	}

	// Apply mechanical effects based on action type and outcome
	effectEvents := sm.applyActionEffects(ctx, parsedAction, targetChar)
	events = append(events, effectEvents...)

	// If we're in a conflict, advance turn and process NPC turns
	if sm.currentScene.IsConflict {
		turnEvents := sm.advanceConflictTurns(ctx)
		events = append(events, turnEvents...)
	}

	return events
}

// rollTargetDefense rolls an active defense for a target character
// and returns the roll result plus a DefenseRollEvent.
func (sm *SceneManager) rollTargetDefense(target *character.Character, attackSkill string) (*dice.CheckResult, DefenseRollEvent) {
	// Determine defense skill based on attack skill type
	defenseSkill := core.DefenseSkillForAttack(attackSkill)
	defenseLevel := target.GetSkill(defenseSkill)

	// Roll defense
	defenseRoll := sm.roller.RollWithModifier(dice.Mediocre, int(defenseLevel))

	event := DefenseRollEvent{
		DefenderName: target.Name,
		Skill:        defenseSkill,
		Result:       defenseRoll.FinalValue.String(),
	}

	return defenseRoll, event
}

// gatherInvokableAspects collects all aspects available for the player to invoke
func (sm *SceneManager) gatherInvokableAspects(usedAspects map[string]bool) []InvokableAspect {
	var aspects []InvokableAspect

	// Character aspects (High Concept, Trouble, Other)
	for _, aspectText := range sm.player.Aspects.GetAll() {
		if aspectText == "" {
			continue
		}
		aspects = append(aspects, InvokableAspect{
			Name:        aspectText,
			Source:      "character",
			SourceID:    sm.player.ID,
			FreeInvokes: 0, // Character aspects don't have free invokes
			AlreadyUsed: usedAspects[aspectText],
		})
	}

	// Player's consequences (can be invoked against self for +2)
	for _, consequence := range sm.player.Consequences {
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
	for _, sitAspect := range sm.currentScene.SituationAspects {
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

// applyActionEffects applies mechanical effects based on action results
// and returns composite events describing what happened.
func (sm *SceneManager) applyActionEffects(ctx context.Context, parsedAction *action.Action, target *character.Character) []GameEvent {
	if parsedAction.Outcome == nil {
		return nil
	}

	var events []GameEvent

	switch parsedAction.Type {
	case action.CreateAdvantage:
		if parsedAction.IsSuccess() {
			aspectName, freeInvokes := sm.generateAspectName(ctx, parsedAction)

			situationAspect := scene.NewSituationAspect(
				fmt.Sprintf("aspect-%d", time.Now().UnixNano()),
				aspectName,
				sm.player.ID,
				freeInvokes,
			)

			sm.currentScene.AddSituationAspect(situationAspect)
			events = append(events, AspectCreatedEvent{
				AspectName:  aspectName,
				FreeInvokes: freeInvokes,
			})
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

			// Apply stress to target
			dmgEvent := sm.applyDamageToTarget(ctx, target, shifts, stressType)
			events = append(events, dmgEvent)
		} else if parsedAction.Outcome.Type == dice.Tie {
			// On a tie, attacker gets a boost
			events = append(events, PlayerAttackResultEvent{
				TargetName: target.Name,
				IsTie:      true,
			})
		}
	}

	return events
}

// generateAspectName uses the LLM to generate a creative aspect name for Create an Advantage
// Falls back to a simple description-based name if the LLM is unavailable or fails
func (sm *SceneManager) generateAspectName(ctx context.Context, parsedAction *action.Action) (string, int) {
	// Determine free invokes based on outcome
	freeInvokes := 1
	if parsedAction.IsSuccessWithStyle() {
		freeInvokes = 2
	}

	// Fallback name if LLM generation fails
	fallbackName := fmt.Sprintf("Advantage from %s", parsedAction.Description)

	// If no aspect generator available, use fallback
	if sm.aspectGenerator == nil {
		slog.Debug("No aspect generator available, using fallback name",
			"component", componentSceneManager)
		return fallbackName, freeInvokes
	}

	// Gather existing aspects for context
	existingAspects := make([]string, 0)
	for _, sa := range sm.currentScene.SituationAspects {
		existingAspects = append(existingAspects, sa.Aspect)
	}

	// Build the request
	req := prompt.AspectGenerationRequest{
		Character:       sm.player,
		Action:          parsedAction,
		Outcome:         parsedAction.Outcome,
		Context:         sm.currentScene.Description,
		TargetType:      "situation",
		ExistingAspects: existingAspects,
	}

	// Generate aspect via LLM with timeout derived from the caller's context
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := sm.aspectGenerator.GenerateAspect(ctx, req)
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

// applyDamageToTarget applies shifts as stress/consequences to a target
// and returns a DamageResolutionEvent describing everything that happened.
func (sm *SceneManager) applyDamageToTarget(ctx context.Context, target *character.Character, shifts int, stressType character.StressTrackType) DamageResolutionEvent {
	dmgEvent := DamageResolutionEvent{
		TargetName: target.Name,
	}

	// Try to absorb with stress track
	absorbed := target.TakeStress(stressType, shifts)
	if absorbed {
		dmgEvent.Absorbed = &StressAbsorptionDetail{
			TrackType:  string(stressType),
			Shifts:     shifts,
			TrackState: target.StressTracks[string(stressType)].String(),
		}
		return dmgEvent
	}

	// Target couldn't absorb all stress - check for consequences or taken out
	sm.fillTargetStressOverflow(ctx, target, shifts, stressType, &dmgEvent)
	return dmgEvent
}

// fillTargetStressOverflow handles when a target can't absorb stress, filling
// in the DamageResolutionEvent with consequence/taken-out information.
func (sm *SceneManager) fillTargetStressOverflow(ctx context.Context, target *character.Character, shifts int, stressType character.StressTrackType, dmgEvent *DamageResolutionEvent) {
	// Check if target has available consequences
	availableConseq := target.AvailableConsequenceSlots()

	if len(availableConseq) == 0 {
		// No way to absorb - target is taken out!
		sm.applyTargetTakenOut(ctx, target, dmgEvent)
		return
	}

	// NPC takes the most appropriate consequence automatically.
	bestConseq, _ := character.BestConsequenceFor(availableConseq, shifts)

	// Apply consequence to target
	consequence := character.Consequence{
		ID:        fmt.Sprintf("conseq-%d", time.Now().UnixNano()),
		Type:      bestConseq.Type,
		Aspect:    fmt.Sprintf("Wounded by %s", sm.player.Name),
		Duration:  string(bestConseq.Type),
		CreatedAt: time.Now(),
	}
	target.AddConsequence(consequence)

	absorbed := bestConseq.Value
	remaining := shifts - absorbed

	dmgEvent.Consequence = &ConsequenceDetail{
		Severity: string(bestConseq.Type),
		Aspect:   consequence.Aspect,
		Absorbed: absorbed,
	}

	// If there's remaining damage, try stress again or take out
	if remaining > 0 {
		if target.TakeStress(stressType, remaining) {
			dmgEvent.RemainingAbsorbed = &StressAbsorptionDetail{
				TrackType:  string(stressType),
				Shifts:     remaining,
				TrackState: target.StressTracks[string(stressType)].String(),
			}
		} else {
			sm.applyTargetTakenOut(ctx, target, dmgEvent)
		}
	}
}

// applyTargetTakenOut marks a target as taken out and updates the damage event.
// Side-effects: updates takenOutChars, scene participant status, checks victory,
// potentially sets pendingMidFlow for fate narration.
func (sm *SceneManager) applyTargetTakenOut(ctx context.Context, target *character.Character, dmgEvent *DamageResolutionEvent) {
	dmgEvent.TakenOut = true

	// Track this character as taken out during this scene
	sm.takenOutChars = append(sm.takenOutChars, target.ID)

	// Log the taken out event
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("taken_out", map[string]any{
			"character_id":   target.ID,
			"character_name": target.Name,
			"by_player":      sm.player.ID,
		})
	}

	// Mark the target as taken out for the duration of this scene
	sm.currentScene.MarkCharacterTakenOut(target.ID)

	// Mark the target as taken out in the conflict
	if sm.currentScene.IsConflict && sm.currentScene.ConflictState != nil {
		sm.currentScene.SetParticipantStatus(target.ID, scene.StatusTakenOut)

		// Check if conflict should end (all opponents taken out)
		activeOpponents := 0
		for _, p := range sm.currentScene.ConflictState.Participants {
			if p.CharacterID != sm.player.ID && p.Status == scene.StatusActive {
				activeOpponents++
			}
		}

		if activeOpponents == 0 {
			dmgEvent.VictoryEnd = true
			sm.promptPlayerForFates(ctx)
			sm.clearConflictStress()
			sm.currentScene.EndConflict()
		}
	}

	slog.Info("Target taken out",
		"component", componentSceneManager,
		"target", target.ID,
		"target_name", target.Name)
}

// promptPlayerForFates prompts the player to narrate the fates of all taken-out
// NPCs after a victory. Per Fate Core, the victor decides what the loss looks like.
// The player's free-text narration is sent to the LLM, which classifies each NPC's
// fate and whether they are permanently removed from the story.
func (sm *SceneManager) promptPlayerForFates(ctx context.Context) {
	if len(sm.takenOutChars) == 0 {
		return
	}

	// Collect taken-out NPC info
	var takenOutNPCs []prompt.FateNarrationNPC
	var npcNames []string
	for _, charID := range sm.takenOutChars {
		char := sm.engine.GetCharacter(charID)
		if char == nil || charID == sm.player.ID {
			continue
		}
		takenOutNPCs = append(takenOutNPCs, prompt.FateNarrationNPC{
			ID:          charID,
			Name:        char.Name,
			HighConcept: char.Aspects.HighConcept,
		})
		npcNames = append(npcNames, char.Name)
	}

	if len(takenOutNPCs) == 0 {
		return
	}

	// Build the prompt for the player
	nameList := strings.Join(npcNames, ", ")

	// Build NPC name list for context.
	npcContextList := make([]map[string]string, 0, len(takenOutNPCs))
	for _, npc := range takenOutNPCs {
		npcContextList = append(npcContextList, map[string]string{
			"id":           npc.ID,
			"name":         npc.Name,
			"high_concept": npc.HighConcept,
		})
	}

	event := InputRequestEvent{
		Type:   uicontract.InputRequestFreeText,
		Prompt: fmt.Sprintf("You decide their fate! What happens to %s?", nameList),
		Context: map[string]any{
			"request_type": "fate_narration",
			"npc_names":    npcNames,
			"npcs":         npcContextList,
		},
	}

	// Capture variables for the continuation closure.
	capturedNPCs := takenOutNPCs

	sm.pendingMidFlow = &midFlowState{
		event: event,
		continuation: func(ctx context.Context, resp MidFlowResponse) []GameEvent {
			if strings.TrimSpace(resp.Text) == "" {
				slog.Warn("Empty fate narration input",
					"component", componentSceneManager)
				return nil
			}

			return sm.processFateNarration(ctx, resp.Text, capturedNPCs)
		},
	}
}

// processFateNarration sends the player's fate narration to the LLM for parsing
// and applies the results. Extracted from promptPlayerForFates to serve as the
// mid-flow continuation.
func (sm *SceneManager) processFateNarration(ctx context.Context, input string, takenOutNPCs []prompt.FateNarrationNPC) []GameEvent {
	// Determine conflict type for context
	conflictType := "physical"
	if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	// Send to LLM for structured parsing
	data := prompt.FateNarrationData{
		SceneName:        sm.currentScene.Name,
		SceneDescription: sm.currentScene.Description,
		ConflictType:     conflictType,
		TakenOutNPCs:     takenOutNPCs,
		PlayerNarration:  input,
	}

	rendered, err := prompt.RenderFateNarration(data)
	if err != nil {
		slog.Error("Failed to render fate narration prompt",
			"component", componentSceneManager,
			"error", err)
		return nil
	}

	content, err := llm.SimpleCompletion(ctx, sm.engine.llmClient, rendered, 400, 0.4)
	if err != nil {
		slog.Error("Failed to get fate narration from LLM",
			"component", componentSceneManager,
			"error", err)
		return nil
	}

	result, err := prompt.ParseFateNarration(content)
	if err != nil {
		slog.Error("Failed to parse fate narration response",
			"component", componentSceneManager,
			"error", err)
		return nil
	}

	// Apply fates to characters
	for _, fate := range result.Fates {
		char := sm.engine.GetCharacter(fate.ID)
		if char == nil {
			slog.Warn("Could not resolve character for fate",
				"component", componentSceneManager,
				"id", fate.ID,
				"name", fate.Name)
			continue
		}
		char.Fate = &character.TakenOutFate{
			Description: fate.Description,
			Permanent:   fate.Permanent,
		}

		slog.Info("Applied fate to character",
			"component", componentSceneManager,
			"character", char.Name,
			"fate", fate.Description,
			"permanent", fate.Permanent)
	}

	// Log the fate narration
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("fate_narration", map[string]any{
			"player_input": input,
			"fates":        result.Fates,
			"narrative":    result.Narrative,
		})
	}

	return []GameEvent{NarrativeEvent{Text: result.Narrative}}
}

// applyAttackDamageToPlayer applies attack damage to the player and returns events.
func (sm *SceneManager) applyAttackDamageToPlayer(ctx context.Context, outcome *dice.Outcome, attacker *character.Character, attackCtx prompt.AttackContext) []GameEvent {
	var events []GameEvent

	// Apply stress if the attack hit
	switch outcome.Type {
	case dice.Success, dice.SuccessWithStyle:
		shifts := outcome.Shifts
		if shifts < 1 {
			shifts = 1
		}
		stressType := character.PhysicalStress
		if sm.currentScene.ConflictState.Type == scene.MentalConflict {
			stressType = character.MentalStress
		}

		// Try to absorb with stress track
		absorbed := sm.player.TakeStress(stressType, shifts)
		if absorbed {
			events = append(events, PlayerStressEvent{
				Shifts:     shifts,
				StressType: string(stressType),
				TrackState: sm.player.StressTracks[string(stressType)].String(),
			})
		} else {
			// Cannot absorb - need consequence or taken out
			overflowEvents := sm.handleStressOverflow(ctx, shifts, stressType, attacker, attackCtx)
			events = append(events, overflowEvents...)
		}
	case dice.Tie:
		events = append(events, PlayerDefendedEvent{IsTie: true})
	default:
		events = append(events, PlayerDefendedEvent{IsTie: false})
	}

	return events
}

// handleStressOverflow handles when the player cannot absorb stress with their stress track.
// Returns events emitted immediately; may set pendingMidFlow for consequence choice.
func (sm *SceneManager) handleStressOverflow(ctx context.Context, shifts int, stressType character.StressTrackType, attacker *character.Character, attackCtx prompt.AttackContext) []GameEvent {
	var events []GameEvent
	events = append(events, SystemMessageEvent{Message: fmt.Sprintf(
		"You cannot absorb %d shifts with your stress track!",
		shifts,
	)})

	// Determine available consequences
	availableConsequences := sm.player.AvailableConsequenceSlots()

	if len(availableConsequences) == 0 {
		// No consequences available - taken out
		events = append(events, SystemMessageEvent{Message: "You have no available consequences! You are taken out!"})
		takenOutEvents := sm.handleTakenOut(ctx, attacker, attackCtx)
		events = append(events, takenOutEvents...)
		return events
	}

	// Build numbered-choice options.
	options := make([]InputOption, 0, len(availableConsequences)+1)
	for _, conseq := range availableConsequences {
		options = append(options, InputOption{
			Label:       fmt.Sprintf("Take a %s consequence", conseq.Type),
			Description: fmt.Sprintf("absorbs %d shifts", conseq.Value),
		})
	}
	options = append(options, InputOption{
		Label:       "Be Taken Out",
		Description: "Your opponent decides your fate",
	})

	event := InputRequestEvent{
		Type:    uicontract.InputRequestNumberedChoice,
		Prompt:  "\nYou must choose how to handle this:",
		Options: options,
		Context: map[string]any{
			"request_type": "consequence_choice",
			"shifts":       shifts,
			"stress_type":  string(stressType),
		},
	}

	// Capture variables for the continuation closure.
	capturedConsequences := availableConsequences
	capturedShifts := shifts
	capturedAttacker := attacker
	capturedAttackCtx := attackCtx

	sm.pendingMidFlow = &midFlowState{
		event: event,
		continuation: func(ctx context.Context, resp MidFlowResponse) []GameEvent {
			takenOutIdx := len(capturedConsequences)

			if resp.ChoiceIndex >= 0 && resp.ChoiceIndex < takenOutIdx {
				return sm.applyConsequence(ctx, capturedConsequences[resp.ChoiceIndex].Type, capturedShifts, capturedAttacker, capturedAttackCtx)
			}
			var events []GameEvent
			events = append(events, SystemMessageEvent{Message: "You are taken out!"})
			takenOutEvents := sm.handleTakenOut(ctx, capturedAttacker, capturedAttackCtx)
			events = append(events, takenOutEvents...)
			return events
		},
	}

	return events
}

// applyConsequence applies a consequence to the player character and returns events.
func (sm *SceneManager) applyConsequence(ctx context.Context, conseqType character.ConsequenceType, shifts int, attacker *character.Character, attackCtx prompt.AttackContext) []GameEvent {
	// Generate a consequence aspect via LLM
	aspectName, err := sm.generateConsequenceAspect(ctx, conseqType, attacker, attackCtx)
	if err != nil {
		slog.Error("Failed to generate consequence aspect", "error", err)
		caser := cases.Title(language.English)
		aspectName = fmt.Sprintf("%s Wound", caser.String(string(conseqType)))
	}

	consequence := character.Consequence{
		ID:        fmt.Sprintf("conseq-%d", time.Now().UnixNano()),
		Type:      conseqType,
		Aspect:    aspectName,
		Duration:  string(conseqType),
		CreatedAt: time.Now(),
	}

	sm.player.AddConsequence(consequence)

	absorbed := conseqType.Value()
	remaining := shifts - absorbed

	pce := PlayerConsequenceEvent{
		Severity: string(conseqType),
		Aspect:   aspectName,
		Absorbed: absorbed,
	}

	var events []GameEvent

	// If there are remaining shifts, try to absorb with stress
	if remaining > 0 {
		stressType := character.PhysicalStress
		if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
			stressType = character.MentalStress
		}

		if sm.player.TakeStress(stressType, remaining) {
			pce.StressAbsorbed = &StressAbsorptionDetail{
				TrackType:  string(stressType),
				Shifts:     remaining,
				TrackState: sm.player.StressTracks[string(stressType)].String(),
			}
			events = append(events, pce)
		} else {
			events = append(events, pce)
			events = append(events, SystemMessageEvent{Message: fmt.Sprintf(
				"You cannot absorb the remaining %d shifts! You may need another consequence.",
				remaining,
			)})
			// Recursively handle remaining damage
			overflowEvents := sm.handleStressOverflow(ctx, remaining, stressType, attacker, attackCtx)
			events = append(events, overflowEvents...)
		}
	} else {
		events = append(events, pce)
	}

	return events
}

// generateConsequenceAspect uses LLM to generate a consequence aspect
func (sm *SceneManager) generateConsequenceAspect(ctx context.Context, conseqType character.ConsequenceType, attacker *character.Character, attackCtx prompt.AttackContext) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("LLM client required")
	}

	conflictType := "physical"
	if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	data := prompt.ConsequenceAspectData{
		CharacterName: sm.player.Name,
		AttackerName:  attacker.Name,
		Severity:      string(conseqType),
		ConflictType:  conflictType,
		AttackContext: attackCtx,
	}

	prompt, err := prompt.RenderConsequenceAspect(data)
	if err != nil {
		return "", fmt.Errorf("failed to render consequence aspect template: %w", err)
	}

	return llm.SimpleCompletion(ctx, sm.engine.llmClient, prompt, 20, 0.8)
}

// isConcedeCommand checks if the input is a concession command
// Per Fate Core rules, concession must happen before a roll is made
func (sm *SceneManager) isConcedeCommand(input string) bool {
	normalized := strings.ToLower(strings.TrimSpace(input))
	concedeCommands := []string{"concede", "i concede", "concession", "i give up", "give up"}
	for _, cmd := range concedeCommands {
		if normalized == cmd {
			return true
		}
	}
	return false
}

// handleConcession handles when the player concedes the conflict.
// Returns events emitted immediately; sets pendingMidFlow for narration.
func (sm *SceneManager) handleConcession(ctx context.Context) []GameEvent {
	var events []GameEvent

	// Award fate points: 1 for conceding + 1 for each consequence taken in this conflict
	// Per Fate Core: "you get a fate point for choosing to concede.
	// On top of that, if you've sustained any consequences in this conflict,
	// you get an additional fate point for each consequence."
	consequenceCount := len(sm.player.Consequences)
	fatePointsGained := core.ConcessionFatePoints(consequenceCount)

	for i := 0; i < fatePointsGained; i++ {
		sm.player.GainFatePoint()
	}

	events = append(events, ConcessionEvent{
		FatePointsGained:  fatePointsGained,
		ConsequenceCount:  consequenceCount,
		CurrentFatePoints: sm.player.FatePoints,
	})

	// Mark player as conceded and end the conflict
	if sm.currentScene.ConflictState != nil {
		sm.currentScene.SetParticipantStatus(sm.player.ID, scene.StatusConceded)
		sm.clearConflictStress()
		sm.currentScene.EndConflict()
		events = append(events, ConflictEndEvent{Reason: "You have conceded the conflict."})
	}

	// Emit a free-text input request for the concession narration instead of
	// blocking on ReadInput.
	event := InputRequestEvent{
		Type:   uicontract.InputRequestFreeText,
		Prompt: "\nDescribe how you concede and exit the conflict:",
		Context: map[string]any{
			"request_type": "concession_narration",
			"player_name":  sm.player.Name,
		},
	}

	sm.pendingMidFlow = &midFlowState{
		event: event,
		continuation: func(_ context.Context, resp MidFlowResponse) []GameEvent {
			var events []GameEvent
			if resp.Text != "" {
				events = append(events, NarrativeEvent{
					Text: fmt.Sprintf("%s %s", sm.player.Name, resp.Text),
				})
				sm.addToConversationHistory(resp.Text, "You exit the conflict on your own terms.", inputTypeDialog)
			}
			return events
		},
	}

	return events
}

// handleTakenOut handles when the player is taken out and returns events.
func (sm *SceneManager) handleTakenOut(ctx context.Context, attacker *character.Character, attackCtx prompt.AttackContext) []GameEvent {
	// Generate narrative and outcome classification for being taken out
	narrative, outcome, newSceneHint, err := sm.generateTakenOutNarrativeAndOutcome(ctx, attacker, attackCtx)
	if err != nil {
		narrative = fmt.Sprintf("You collapse, defeated by %s.", attacker.Name)
		outcome = TakenOutTransition
		newSceneHint = "You awaken later, unsure of your fate..."
	}

	// Mark player as taken out and end the conflict
	if sm.currentScene.ConflictState != nil {
		sm.currentScene.SetParticipantStatus(sm.player.ID, scene.StatusTakenOut)
		sm.clearConflictStress()
		sm.currentScene.EndConflict()
	}

	outcomeStr := "continue"
	switch outcome {
	case TakenOutGameOver:
		outcomeStr = "game_over"
	case TakenOutTransition:
		outcomeStr = "transition"
	}

	var events []GameEvent
	events = append(events, PlayerTakenOutEvent{
		AttackerName: attacker.Name,
		Narrative:    narrative,
		Outcome:      outcomeStr,
		NewSceneHint: newSceneHint,
	})

	// Handle scene-level side effects based on outcome type
	switch outcome {
	case TakenOutGameOver:
		sm.sceneEndReason = SceneEndPlayerTakenOut
		sm.playerTakenOutHint = ""
		sm.shouldExit = true

	case TakenOutTransition:
		sm.sceneEndReason = SceneEndPlayerTakenOut
		sm.playerTakenOutHint = newSceneHint
		if sm.exitOnSceneTransition {
			sm.shouldExit = true
		}

	default: // TakenOutContinue
		// Don't set sceneEndReason - scene continues
	}

	return events
}

// generateTakenOutNarrativeAndOutcome generates narrative and classifies the outcome
func (sm *SceneManager) generateTakenOutNarrativeAndOutcome(ctx context.Context, attacker *character.Character, attackCtx prompt.AttackContext) (narrative string, outcome TakenOutResult, newSceneHint string, err error) {
	if sm.engine.llmClient == nil {
		return "", TakenOutTransition, "", fmt.Errorf("LLM client required")
	}

	conflictType := "physical"
	if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	data := prompt.TakenOutData{
		CharacterName:       sm.player.Name,
		AttackerName:        attacker.Name,
		AttackerHighConcept: attacker.Aspects.HighConcept,
		ConflictType:        conflictType,
		SceneDescription:    sm.currentScene.Description,
		AttackContext:       attackCtx,
	}

	prompt, err := prompt.RenderTakenOut(data)
	if err != nil {
		return "", TakenOutTransition, "", fmt.Errorf("failed to render taken out template: %w", err)
	}

	content, err := llm.SimpleCompletion(ctx, sm.engine.llmClient, prompt, 200, 0.7)
	if err != nil {
		return "", TakenOutTransition, "", err
	}

	// Parse the JSON response
	type takenOutResponse struct {
		Narrative    string `json:"narrative"`
		Outcome      string `json:"outcome"`
		NewSceneHint string `json:"new_scene_hint"`
	}

	var parsed takenOutResponse
	if parseErr := json.Unmarshal([]byte(content), &parsed); parseErr != nil {
		// If parsing fails, use the raw content as narrative and default to transition
		slog.Warn("Failed to parse taken out response as JSON, using raw content",
			"error", parseErr,
			"content", content,
		)
		return content, TakenOutTransition, "You awaken later...", nil
	}

	// Map outcome string to enum
	switch strings.ToLower(parsed.Outcome) {
	case "game_over":
		outcome = TakenOutGameOver
	case "continue":
		outcome = TakenOutContinue
	default:
		outcome = TakenOutTransition
	}

	return parsed.Narrative, outcome, parsed.NewSceneHint, nil
}
