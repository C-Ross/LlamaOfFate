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
func (cm *ConflictManager) parseConflictMarker(response string) (*prompt.ConflictTrigger, string) {
	return prompt.ParseConflictMarker(response)
}

// parseConflictEndMarker extracts a conflict resolution from LLM response and returns cleaned text
func (cm *ConflictManager) parseConflictEndMarker(response string) (*prompt.ConflictResolution, string) {
	return prompt.ParseConflictEndMarker(response)
}

// initiateConflict starts a conflict with all characters in the scene
func (cm *ConflictManager) initiateConflict(conflictType scene.ConflictType, initiatorID string) error {
	if cm.currentScene.IsConflict {
		return fmt.Errorf("already in a conflict")
	}

	// Validate the initiator is a real character in this scene
	if cm.characters.GetCharacter(initiatorID) == nil {
		slog.Warn("Conflict trigger rejected: initiator ID does not match any character",
			"component", componentSceneManager,
			"initiator", initiatorID)
		return fmt.Errorf("initiator %q is not a known character", initiatorID)
	}

	// Check if the initiator was taken out earlier in this scene
	if cm.currentScene.IsCharacterTakenOut(initiatorID) {
		slog.Debug("Conflict initiator was previously taken out this scene",
			"component", componentSceneManager,
			"initiator", initiatorID)
		return fmt.Errorf("initiator %s was taken out this scene", initiatorID)
	}

	// Build participants from all characters in the scene
	participants := make([]scene.ConflictParticipant, 0)

	for _, charID := range cm.currentScene.Characters {
		char := cm.characters.GetCharacter(charID)
		if char == nil {
			continue
		}

		// Skip characters that have been taken out earlier in this scene
		if cm.currentScene.IsCharacterTakenOut(charID) {
			slog.Debug("Skipping taken-out character for conflict",
				"component", componentSceneManager,
				"character", charID)
			continue
		}

		// Calculate initiative based on conflict type
		initiative := cm.calculateInitiative(char, conflictType)

		participants = append(participants, scene.ConflictParticipant{
			CharacterID: charID,
			Initiative:  initiative,
			Status:      scene.StatusActive,
		})
	}

	if len(participants) < 2 {
		return fmt.Errorf("conflict requires at least 2 participants")
	}

	cm.currentScene.StartConflictWithInitiator(conflictType, participants, initiatorID)

	slog.Info("Conflict initiated",
		"component", componentSceneManager,
		"type", conflictType,
		"initiator", initiatorID,
		"participants", len(participants))

	return nil
}

// resolveConflictPeacefully ends a conflict through non-violent means
func (cm *ConflictManager) resolveConflictPeacefully(reason string) string {
	if !cm.currentScene.IsConflict {
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

	cm.clearConflictStress()
	cm.currentScene.EndConflict()

	slog.Info("Conflict resolved peacefully",
		"component", componentSceneManager,
		"reason", reason)

	return reasonMessage
}

// clearConflictStress clears stress for all conflict participants.
// Per Fate Core: "After a conflict, when you get a minute to breathe,
// any stress boxes you checked off become available for your use again."
func (cm *ConflictManager) clearConflictStress() {
	if cm.currentScene.ConflictState == nil {
		return
	}

	for _, p := range cm.currentScene.ConflictState.Participants {
		char := cm.characters.GetCharacter(p.CharacterID)
		if char != nil {
			char.ClearAllStress()
		}
	}

	slog.Info("Cleared stress for all conflict participants",
		"component", componentSceneManager,
		"participants", len(cm.currentScene.ConflictState.Participants))
}

// calculateInitiative returns the initiative value for a character based on conflict type
func (cm *ConflictManager) calculateInitiative(char *core.Character, conflictType scene.ConflictType) int {
	return core.CalculateInitiative(char, conflictType)
}

// sortInitiativeOrder sorts the initiative order by participant initiative values
func (cm *ConflictManager) sortInitiativeOrder() {
	if cm.currentScene.ConflictState == nil {
		return
	}

	cm.currentScene.ConflictState.SortByInitiative()
}

// recalculateInitiative recalculates initiative for all participants based on conflict type
func (cm *ConflictManager) recalculateInitiative(conflictType scene.ConflictType) {
	if cm.currentScene.ConflictState == nil {
		return
	}

	for i := range cm.currentScene.ConflictState.Participants {
		p := &cm.currentScene.ConflictState.Participants[i]
		char := cm.characters.GetCharacter(p.CharacterID)
		if char != nil {
			p.Initiative = cm.calculateInitiative(char, conflictType)
		}
	}

	cm.sortInitiativeOrder()
}

// handleConflictEscalation changes the conflict type and recalculates initiative.
// Returns the escalation event.
func (cm *ConflictManager) handleConflictEscalation(newType scene.ConflictType) []GameEvent {
	if !cm.currentScene.IsConflict {
		return nil
	}

	currentType := cm.currentScene.ConflictState.Type
	if currentType == newType {
		return nil
	}

	cm.currentScene.EscalateConflict(newType)
	cm.recalculateInitiative(newType)

	return []GameEvent{ConflictEscalationEvent{
		FromType:        string(currentType),
		ToType:          string(newType),
		TriggerCharName: cm.player.Name,
	}}
}

// advanceConflictTurns advances through turns and processes NPC actions until
// it's the player's turn or a defense invoke pauses the loop.
// Returns (events, awaitingInvoke). When awaitingInvoke is true,
// the ActionResolver's pendingInvoke is set; the invoke continuation will resume NPC turns.
func (cm *ConflictManager) advanceConflictTurns(ctx context.Context) ([]GameEvent, bool) {
	if !cm.currentScene.IsConflict || cm.currentScene.ConflictState == nil {
		return nil, false
	}

	var events []GameEvent

	// Advance past the current actor's turn
	cm.currentScene.ConflictState.NextTurn()

	// Process NPC turns until we get back to the player or conflict ends
	for cm.currentScene.IsConflict {
		currentActor := cm.currentScene.ConflictState.GetCurrentActor()
		if currentActor == "" {
			break
		}

		// If it's the player's turn, stop and let them act
		if currentActor == cm.player.ID {
			events = append(events, TurnAnnouncementEvent{CharacterName: cm.player.Name, TurnNumber: cm.currentScene.ConflictState.Round, IsPlayer: true})
			break
		}

		// Process NPC turn
		npcEvents, awaiting := cm.processNPCTurn(ctx, currentActor)
		events = append(events, npcEvents...)

		// If a defense invoke paused the loop, return immediately.
		// The invoke continuation will resume remaining NPC turns.
		if awaiting {
			return events, true
		}

		// Advance to next turn
		cm.currentScene.ConflictState.NextTurn()
	}

	return events, false
}

// getParticipantInfo returns information about all conflict participants for display
func (cm *ConflictManager) getParticipantInfo() []ConflictParticipantInfo {
	if cm.currentScene.ConflictState == nil {
		return nil
	}

	info := make([]ConflictParticipantInfo, 0, len(cm.currentScene.ConflictState.Participants))
	for _, p := range cm.currentScene.ConflictState.Participants {
		name := p.CharacterID
		if char := cm.characters.GetCharacter(p.CharacterID); char != nil {
			name = char.Name
		}
		info = append(info, ConflictParticipantInfo{
			CharacterID:   p.CharacterID,
			CharacterName: name,
			Initiative:    p.Initiative,
			IsPlayer:      p.CharacterID == cm.player.ID,
		})
	}
	return info
}

// applyDamageToTarget applies shifts as stress/consequences to a target
// and returns a DamageResolutionEvent describing everything that happened.
func (cm *ConflictManager) applyDamageToTarget(ctx context.Context, target *core.Character, shifts int, stressType core.StressTrackType) DamageResolutionEvent {
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
	cm.fillTargetStressOverflow(ctx, target, shifts, stressType, &dmgEvent)
	return dmgEvent
}

// fillTargetStressOverflow handles when a target can't absorb stress, filling
// in the DamageResolutionEvent with consequence/taken-out information.
func (cm *ConflictManager) fillTargetStressOverflow(ctx context.Context, target *core.Character, shifts int, stressType core.StressTrackType, dmgEvent *DamageResolutionEvent) {
	// Check if target has available consequences
	availableConseq := target.AvailableConsequenceSlots()

	if len(availableConseq) == 0 {
		// No way to absorb - target is taken out!
		cm.applyTargetTakenOut(ctx, target, dmgEvent)
		return
	}

	// NPC takes the most appropriate consequence automatically.
	bestConseq, _ := core.BestConsequenceFor(availableConseq, shifts)

	// Apply consequence to target
	consequence := core.Consequence{
		ID:        fmt.Sprintf("conseq-%d", time.Now().UnixNano()),
		Type:      bestConseq.Type,
		Aspect:    fmt.Sprintf("Wounded by %s", cm.player.Name),
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
			cm.applyTargetTakenOut(ctx, target, dmgEvent)
		}
	}
}

// applyTargetTakenOut marks a target as taken out and updates the damage event.
// Side-effects: updates takenOutChars, scene participant status, checks victory,
// potentially sets pendingMidFlow for fate narration.
func (cm *ConflictManager) applyTargetTakenOut(ctx context.Context, target *core.Character, dmgEvent *DamageResolutionEvent) {
	dmgEvent.TakenOut = true

	// Mark the character as taken out immediately so IsTakenOut() returns true.
	// processFateNarration will later overwrite Fate with the player's narration.
	if target.Fate == nil {
		target.Fate = &core.TakenOutFate{
			Description: "taken out",
		}
	}

	// Track this character as taken out during this scene
	cm.takenOutChars = append(cm.takenOutChars, target.ID)

	// Log the taken out event
	cm.sessionLogger.Log("taken_out", map[string]any{
		"character_id":   target.ID,
		"character_name": target.Name,
		"by_player":      cm.player.ID,
	})

	// Mark the target as taken out for the duration of this scene
	cm.currentScene.MarkCharacterTakenOut(target.ID)

	// Mark the target as taken out in the conflict
	if cm.currentScene.IsConflict && cm.currentScene.ConflictState != nil {
		cm.currentScene.ConflictState.SetParticipantStatus(target.ID, scene.StatusTakenOut)

		// Check if conflict should end (all opponents taken out)
		activeOpponents := 0
		for _, p := range cm.currentScene.ConflictState.Participants {
			if p.CharacterID != cm.player.ID && p.Status == scene.StatusActive {
				activeOpponents++
			}
		}

		if activeOpponents == 0 {
			dmgEvent.VictoryEnd = true
			cm.promptPlayerForFates(ctx)
			cm.clearConflictStress()
			cm.currentScene.EndConflict()
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
func (cm *ConflictManager) promptPlayerForFates(ctx context.Context) {
	if len(cm.takenOutChars) == 0 {
		return
	}

	// Collect taken-out NPC info
	var takenOutNPCs []prompt.FateNarrationNPC
	var npcNames []string
	for _, charID := range cm.takenOutChars {
		char := cm.characters.GetCharacter(charID)
		if char == nil || charID == cm.player.ID {
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

	cm.actions.pendingMidFlow = &midFlowState{
		event: event,
		continuation: func(ctx context.Context, resp MidFlowResponse) []GameEvent {
			if strings.TrimSpace(resp.Text) == "" {
				slog.Warn("Empty fate narration input",
					"component", componentSceneManager)
				return nil
			}

			return cm.processFateNarration(ctx, resp.Text, capturedNPCs)
		},
	}
}

// processFateNarration sends the player's fate narration to the LLM for parsing
// and applies the results. Extracted from promptPlayerForFates to serve as the
// mid-flow continuation.
func (cm *ConflictManager) processFateNarration(ctx context.Context, input string, takenOutNPCs []prompt.FateNarrationNPC) []GameEvent {
	conflictType := cm.conflictTypeString()

	// Send to LLM for structured parsing
	data := prompt.FateNarrationData{
		SceneName:        cm.currentScene.Name,
		SceneDescription: cm.currentScene.Description,
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

	content, err := llm.SimpleCompletion(ctx, cm.llmClient, rendered, 400, 0.4)
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
		char := cm.characters.GetCharacter(fate.ID)
		if char == nil {
			slog.Warn("Could not resolve character for fate",
				"component", componentSceneManager,
				"id", fate.ID,
				"name", fate.Name)
			continue
		}
		char.Fate = &core.TakenOutFate{
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
	cm.sessionLogger.Log("fate_narration", map[string]any{
		"player_input": input,
		"fates":        result.Fates,
		"narrative":    result.Narrative,
	})

	return []GameEvent{NarrativeEvent{Text: result.Narrative}}
}

// applyAttackDamageToPlayer applies attack damage to the player and returns events.
func (cm *ConflictManager) applyAttackDamageToPlayer(ctx context.Context, outcome *dice.Outcome, attacker *core.Character, attackCtx prompt.AttackContext) []GameEvent {
	var events []GameEvent

	// Apply stress if the attack hit
	switch outcome.Type {
	case dice.Success, dice.SuccessWithStyle:
		shifts := outcome.Shifts
		if shifts < 1 {
			shifts = 1
		}
		stressType := core.StressTypeForConflict(cm.currentScene.ConflictState.Type)

		// Try to absorb with stress track
		absorbed := cm.player.TakeStress(stressType, shifts)
		if absorbed {
			events = append(events, PlayerStressEvent{
				Shifts:     shifts,
				StressType: string(stressType),
				TrackState: cm.player.StressTracks[string(stressType)].String(),
			})
		} else {
			// Cannot absorb - need consequence or taken out
			overflowEvents := cm.handleStressOverflow(ctx, shifts, stressType, attacker, attackCtx)
			events = append(events, overflowEvents...)
		}
	case dice.Tie:
		// Attacker gets a boost on a tie (no damage to player).
		events = append(events, PlayerDefendedEvent{IsTie: true})
		boostName := cm.actions.generateBoostName(ctx, attacker, attackCtx.Skill, attackCtx.Description, "Fleeting Opening")
		events = append(events, cm.actions.createBoost(boostName, attacker.ID))
	default:
		// Attack failed — check if player defended with style (3+ margin).
		events = append(events, PlayerDefendedEvent{IsTie: false})
		if outcome.Shifts <= -3 {
			defDesc := fmt.Sprintf("defending against %s's attack", attacker.Name)
			defSkill := core.DefenseSkillForAttack(attackCtx.Skill)
			boostName := cm.actions.generateBoostName(ctx, cm.player, defSkill, defDesc, "Deflected with Ease")
			events = append(events, cm.actions.createBoost(boostName, cm.player.ID))
		}
	}

	return events
}

// handleStressOverflow handles when the player cannot absorb stress with their stress track.
// Returns events emitted immediately; may set pendingMidFlow for consequence choice.
func (cm *ConflictManager) handleStressOverflow(ctx context.Context, shifts int, stressType core.StressTrackType, attacker *core.Character, attackCtx prompt.AttackContext) []GameEvent {
	var events []GameEvent
	events = append(events, StressOverflowEvent{
		Shifts: shifts,
	})

	// Determine available consequences
	availableConsequences := cm.player.AvailableConsequenceSlots()

	if len(availableConsequences) == 0 {
		// No consequences available - taken out
		events = append(events, StressOverflowEvent{
			Shifts:         shifts,
			NoConsequences: true,
		})
		takenOutEvents := cm.handleTakenOut(ctx, attacker, attackCtx)
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

	cm.actions.pendingMidFlow = &midFlowState{
		event: event,
		continuation: func(ctx context.Context, resp MidFlowResponse) []GameEvent {
			takenOutIdx := len(capturedConsequences)

			if resp.ChoiceIndex >= 0 && resp.ChoiceIndex < takenOutIdx {
				return cm.applyConsequence(ctx, capturedConsequences[resp.ChoiceIndex].Type, capturedShifts, capturedAttacker, capturedAttackCtx)
			}
			var events []GameEvent
			takenOutEvents := cm.handleTakenOut(ctx, capturedAttacker, capturedAttackCtx)
			events = append(events, takenOutEvents...)
			return events
		},
	}

	return events
}

// applyConsequence applies a consequence to the player character and returns events.
func (cm *ConflictManager) applyConsequence(ctx context.Context, conseqType core.ConsequenceType, shifts int, attacker *core.Character, attackCtx prompt.AttackContext) []GameEvent {
	// Generate a consequence aspect via LLM
	aspectName, err := cm.generateConsequenceAspect(ctx, conseqType, attacker, attackCtx)
	if err != nil {
		slog.Error("Failed to generate consequence aspect", "error", err)
		caser := cases.Title(language.English)
		aspectName = fmt.Sprintf("%s Wound", caser.String(string(conseqType)))
	}

	consequence := core.Consequence{
		ID:        fmt.Sprintf("conseq-%d", time.Now().UnixNano()),
		Type:      conseqType,
		Aspect:    aspectName,
		Duration:  string(conseqType),
		CreatedAt: time.Now(),
	}

	cm.player.AddConsequence(consequence)

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
		stressType := core.PhysicalStress
		if cm.currentScene.ConflictState != nil && cm.currentScene.ConflictState.Type == scene.MentalConflict {
			stressType = core.MentalStress
		}

		if cm.player.TakeStress(stressType, remaining) {
			pce.StressAbsorbed = &StressAbsorptionDetail{
				TrackType:  string(stressType),
				Shifts:     remaining,
				TrackState: cm.player.StressTracks[string(stressType)].String(),
			}
			events = append(events, pce)
		} else {
			events = append(events, pce)
			events = append(events, StressOverflowEvent{
				Shifts:            remaining,
				RemainingOverflow: true,
			})
			// Recursively handle remaining damage
			overflowEvents := cm.handleStressOverflow(ctx, remaining, stressType, attacker, attackCtx)
			events = append(events, overflowEvents...)
		}
	} else {
		events = append(events, pce)
	}

	return events
}

// generateConsequenceAspect uses LLM to generate a consequence aspect
func (cm *ConflictManager) generateConsequenceAspect(ctx context.Context, conseqType core.ConsequenceType, attacker *core.Character, attackCtx prompt.AttackContext) (string, error) {
	if cm.llmClient == nil {
		return "", fmt.Errorf("LLM client required")
	}

	conflictType := cm.conflictTypeString()

	data := prompt.ConsequenceAspectData{
		CharacterName: cm.player.Name,
		AttackerName:  attacker.Name,
		Severity:      string(conseqType),
		ConflictType:  conflictType,
		AttackContext: attackCtx,
	}

	prompt, err := prompt.RenderConsequenceAspect(data)
	if err != nil {
		return "", fmt.Errorf("failed to render consequence aspect template: %w", err)
	}

	return llm.SimpleCompletion(ctx, cm.llmClient, prompt, 20, 0.8)
}

// isConcedeCommand checks if the input is a concession command
// Per Fate Core rules, concession must happen before a roll is made
func (cm *ConflictManager) isConcedeCommand(input string) bool {
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
func (cm *ConflictManager) handleConcession(ctx context.Context) []GameEvent {
	var events []GameEvent

	// Award fate points: 1 for conceding + 1 for each consequence taken in this conflict
	// Per Fate Core: "you get a fate point for choosing to concede.
	// On top of that, if you've sustained any consequences in this conflict,
	// you get an additional fate point for each consequence."
	consequenceCount := len(cm.player.Consequences)
	fatePointsGained := core.ConcessionFatePoints(consequenceCount)

	for i := 0; i < fatePointsGained; i++ {
		cm.player.GainFatePoint()
	}

	events = append(events, ConcessionEvent{
		FatePointsGained:  fatePointsGained,
		ConsequenceCount:  consequenceCount,
		CurrentFatePoints: cm.player.FatePoints,
	})

	// Mark player as conceded and end the conflict
	if cm.currentScene.ConflictState != nil {
		cm.currentScene.ConflictState.SetParticipantStatus(cm.player.ID, scene.StatusConceded)
		cm.clearConflictStress()
		cm.currentScene.EndConflict()
		events = append(events, ConflictEndEvent{Reason: "You have conceded the conflict."})
	}

	// Record concession in conversation history for recap on resume
	cm.actions.RecordConversation("concede",
		fmt.Sprintf("[Conflict ended — %s conceded. Gained %d fate point(s).]", cm.player.Name, fatePointsGained),
		inputTypeConflict)

	// Emit a free-text input request for the concession narration instead of
	// blocking on ReadInput.
	event := InputRequestEvent{
		Type:   uicontract.InputRequestFreeText,
		Prompt: "\nDescribe how you concede and exit the conflict:",
		Context: map[string]any{
			"request_type": "concession_narration",
			"player_name":  cm.player.Name,
		},
	}

	cm.actions.pendingMidFlow = &midFlowState{
		event: event,
		continuation: func(_ context.Context, resp MidFlowResponse) []GameEvent {
			var events []GameEvent
			if resp.Text != "" {
				events = append(events, NarrativeEvent{
					Text: fmt.Sprintf("%s %s", cm.player.Name, resp.Text),
				})
				cm.actions.RecordConversation(resp.Text, "You exit the conflict on your own terms.", inputTypeDialog)
			}
			return events
		},
	}

	return events
}

// handleTakenOut handles when the player is taken out and returns events.
func (cm *ConflictManager) handleTakenOut(ctx context.Context, attacker *core.Character, attackCtx prompt.AttackContext) []GameEvent {
	// Generate narrative and outcome classification for being taken out
	narrative, outcome, newSceneHint, err := cm.generateTakenOutNarrativeAndOutcome(ctx, attacker, attackCtx)
	if err != nil {
		narrative = fmt.Sprintf("You collapse, defeated by %s.", attacker.Name)
		outcome = TakenOutTransition
		newSceneHint = "You awaken later, unsure of your fate..."
	}

	// Mark player as taken out and end the conflict
	if cm.currentScene.ConflictState != nil {
		cm.currentScene.ConflictState.SetParticipantStatus(cm.player.ID, scene.StatusTakenOut)
		cm.clearConflictStress()
		cm.currentScene.EndConflict()
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
		cm.sceneEndReason = SceneEndPlayerTakenOut
		cm.playerTakenOutHint = ""
		cm.shouldExit = true

	case TakenOutTransition:
		cm.sceneEndReason = SceneEndPlayerTakenOut
		cm.playerTakenOutHint = newSceneHint
		if cm.exitOnSceneTransition {
			cm.shouldExit = true
		}

	default: // TakenOutContinue
		// Don't set sceneEndReason - scene continues
	}

	return events
}

// generateTakenOutNarrativeAndOutcome generates narrative and classifies the outcome
func (cm *ConflictManager) generateTakenOutNarrativeAndOutcome(ctx context.Context, attacker *core.Character, attackCtx prompt.AttackContext) (narrative string, outcome TakenOutResult, newSceneHint string, err error) {
	if cm.llmClient == nil {
		return "", TakenOutTransition, "", fmt.Errorf("LLM client required")
	}

	conflictType := cm.conflictTypeString()

	data := prompt.TakenOutData{
		CharacterName:       cm.player.Name,
		AttackerName:        attacker.Name,
		AttackerHighConcept: attacker.Aspects.HighConcept,
		ConflictType:        conflictType,
		SceneDescription:    cm.currentScene.Description,
		AttackContext:       attackCtx,
	}

	prompt, err := prompt.RenderTakenOut(data)
	if err != nil {
		return "", TakenOutTransition, "", fmt.Errorf("failed to render taken out template: %w", err)
	}

	content, err := llm.SimpleCompletion(ctx, cm.llmClient, prompt, 200, 0.7)
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
