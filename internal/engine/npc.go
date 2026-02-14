package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
)

// getNPCActionDecision uses the LLM to decide what action an NPC should take
func (sm *SceneManager) getNPCActionDecision(ctx context.Context, npc *character.Character) (*prompt.NPCActionDecision, error) {
	if sm.engine.llmClient == nil {
		return nil, fmt.Errorf("LLM client required for NPC decisions")
	}

	// Determine conflict type string
	conflictType := "physical"
	if sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	// Build target list (all active participants except this NPC)
	var targets []prompt.NPCTargetInfo
	for _, p := range sm.currentScene.ConflictState.Participants {
		if p.CharacterID == npc.ID || p.Status != scene.StatusActive {
			continue
		}
		char := sm.engine.GetCharacter(p.CharacterID)
		if char == nil {
			continue
		}
		target := prompt.NPCTargetInfo{
			ID:          char.ID,
			Name:        char.Name,
			HighConcept: char.Aspects.HighConcept,
		}
		if track, ok := char.StressTracks["physical"]; ok {
			target.PhysicalStress = track.Boxes
		}
		if track, ok := char.StressTracks["mental"]; ok {
			target.MentalStress = track.Boxes
		}
		targets = append(targets, target)
	}

	// Build skill map with integer values
	npcSkills := make(map[string]int)
	for skill, level := range npc.Skills {
		npcSkills[skill] = int(level)
	}

	// Get NPC stress
	var physicalStress, mentalStress []bool
	if track, ok := npc.StressTracks["physical"]; ok {
		physicalStress = track.Boxes
	}
	if track, ok := npc.StressTracks["mental"]; ok {
		mentalStress = track.Boxes
	}

	data := prompt.NPCActionDecisionData{
		ConflictType:      conflictType,
		Round:             sm.currentScene.ConflictState.Round,
		SceneName:         sm.currentScene.Name,
		SceneDescription:  sm.currentScene.Description,
		NPCName:           npc.Name,
		NPCHighConcept:    npc.Aspects.HighConcept,
		NPCTrouble:        npc.Aspects.Trouble,
		NPCAspects:        npc.Aspects.GetAll(),
		NPCSkills:         npcSkills,
		NPCPhysicalStress: physicalStress,
		NPCMentalStress:   mentalStress,
		Targets:           targets,
		SituationAspects:  sm.currentScene.SituationAspects,
	}

	promptText, err := prompt.RenderNPCActionDecision(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render NPC action decision template: %w", err)
	}

	content, err := llm.SimpleCompletion(ctx, sm.engine.llmClient, promptText, 150, 0.7)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var decision prompt.NPCActionDecision
	if err := json.Unmarshal([]byte(content), &decision); err != nil {
		slog.Warn("Failed to parse NPC action decision",
			"component", componentSceneManager,
			"npc", npc.Name,
			"response", content,
			"error", err)
		return nil, fmt.Errorf("failed to parse decision: %w", err)
	}

	slog.Info("NPC action decision",
		"component", componentSceneManager,
		"npc", npc.Name,
		"action_type", decision.ActionType,
		"skill", decision.Skill,
		"target", decision.TargetID,
		"description", decision.Description)

	return &decision, nil
}

// processNPCTurn handles an NPC's action during conflict.
// Returns (events, awaitingInvoke) where awaitingInvoke is true if the player
// needs to respond to a defense invoke prompt.
func (sm *SceneManager) processNPCTurn(ctx context.Context, npcID string) ([]GameEvent, bool) {
	npc := sm.engine.GetCharacter(npcID)
	if npc == nil {
		slog.Warn("NPC not found for turn processing",
			"component", componentSceneManager,
			"npc_id", npcID)
		return nil, false
	}

	var events []GameEvent
	events = append(events, TurnAnnouncementEvent{CharacterName: npc.Name, TurnNumber: sm.currentScene.ConflictState.Round, IsPlayer: false})

	// Get LLM decision for NPC action
	decision, err := sm.getNPCActionDecision(ctx, npc)
	if err != nil {
		slog.Warn("Failed to get NPC action decision, defaulting to attack",
			"component", componentSceneManager,
			"npc", npc.Name,
			"error", err)
		// Fallback to simple attack
		decision = &prompt.NPCActionDecision{
			ActionType:  "ATTACK",
			Skill:       sm.getDefaultAttackSkill(),
			TargetID:    sm.player.ID,
			Description: fmt.Sprintf("%s attacks!", npc.Name),
		}
	}

	// Process the decision based on action type
	switch strings.ToUpper(decision.ActionType) {
	case "DEFEND":
		events = append(events, sm.processNPCDefend(ctx, npc, decision)...)
	case "CREATE_ADVANTAGE":
		events = append(events, sm.processNPCCreateAdvantage(ctx, npc, decision)...)
	case "OVERCOME":
		events = append(events, sm.processNPCOvercome(ctx, npc, decision)...)
	default: // ATTACK or unknown
		attackEvents, awaiting := sm.processNPCAttack(ctx, npc, decision)
		events = append(events, attackEvents...)
		return events, awaiting
	}

	return events, false
}

// getDefaultAttackSkill returns the default attack skill based on conflict type
func (sm *SceneManager) getDefaultAttackSkill() string {
	return core.DefaultAttackSkillForConflict(sm.currentScene.ConflictState.Type)
}

// processNPCDefend handles an NPC choosing full defense
func (sm *SceneManager) processNPCDefend(ctx context.Context, npc *character.Character, decision *prompt.NPCActionDecision) []GameEvent {
	// Set full defense flag
	sm.currentScene.SetFullDefense(npc.ID)

	var events []GameEvent
	events = append(events, NPCActionResultEvent{
		NPCName:    npc.Name,
		ActionType: "defend",
	})

	// Generate narrative
	narrative := fmt.Sprintf("%s braces for incoming attacks, focusing entirely on defense.", npc.Name)
	if decision.Description != "" {
		narrative = decision.Description
	}
	events = append(events, NarrativeEvent{Text: narrative})

	return events
}

// processNPCCreateAdvantage handles an NPC creating an advantage
func (sm *SceneManager) processNPCCreateAdvantage(ctx context.Context, npc *character.Character, decision *prompt.NPCActionDecision) []GameEvent {
	skill := decision.Skill
	if skill == "" {
		skill = "Notice" // Default
	}

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(skill)

	// Roll against Fair (+2) difficulty for creating aspects
	npcRoll := sm.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))
	difficulty := dice.Fair
	outcome := npcRoll.CompareAgainst(difficulty)

	var events []GameEvent

	switch outcome.Type {
	case dice.Success, dice.SuccessWithStyle:
		// Create situation aspect
		aspectName := fmt.Sprintf("%s's Advantage", npc.Name)
		if decision.Description != "" {
			// Try to extract a short aspect name from the description
			aspectName = decision.Description
			if len(aspectName) > 40 {
				aspectName = aspectName[:40]
			}
		}

		freeInvokes := 1
		if outcome.Type == dice.SuccessWithStyle {
			freeInvokes = 2
		}

		aspectID := fmt.Sprintf("npc-advantage-%d", time.Now().UnixNano())
		situationAspect := scene.NewSituationAspect(aspectID, aspectName, npc.ID, freeInvokes)
		sm.currentScene.AddSituationAspect(situationAspect)
		events = append(events, NPCActionResultEvent{
			NPCName:       npc.Name,
			ActionType:    "create_advantage",
			Skill:         skill,
			RollResult:    npcRoll.FinalValue.String(),
			Difficulty:    "Fair",
			Outcome:       outcome.Type.String(),
			AspectCreated: aspectName,
			FreeInvokes:   freeInvokes,
		})
		events = append(events, NarrativeEvent{Text: fmt.Sprintf("%s gains a tactical advantage!", npc.Name)})
	case dice.Tie:
		events = append(events, NPCActionResultEvent{
			NPCName:    npc.Name,
			ActionType: "create_advantage",
			Skill:      skill,
			RollResult: npcRoll.FinalValue.String(),
			Difficulty: "Fair",
			Outcome:    outcome.Type.String(),
		})
		events = append(events, NarrativeEvent{Text: fmt.Sprintf("%s's maneuver is partially successful.", npc.Name)})
	default:
		events = append(events, NPCActionResultEvent{
			NPCName:    npc.Name,
			ActionType: "create_advantage",
			Skill:      skill,
			RollResult: npcRoll.FinalValue.String(),
			Difficulty: "Fair",
			Outcome:    outcome.Type.String(),
		})
		events = append(events, NarrativeEvent{Text: fmt.Sprintf("%s's gambit doesn't pay off.", npc.Name)})
	}

	return events
}

// processNPCOvercome handles an NPC attempting to overcome an obstacle
func (sm *SceneManager) processNPCOvercome(ctx context.Context, npc *character.Character, decision *prompt.NPCActionDecision) []GameEvent {
	skill := decision.Skill
	if skill == "" {
		skill = "Athletics" // Default
	}

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(skill)

	// Roll against Fair (+2) difficulty
	npcRoll := sm.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))
	difficulty := dice.Fair
	outcome := npcRoll.CompareAgainst(difficulty)

	var events []GameEvent
	events = append(events, NPCActionResultEvent{
		NPCName:    npc.Name,
		ActionType: "overcome",
		Skill:      skill,
		RollResult: npcRoll.FinalValue.String(),
		Difficulty: "Fair",
		Outcome:    outcome.Type.String(),
	})

	switch outcome.Type {
	case dice.Success, dice.SuccessWithStyle:
		narrative := decision.Description
		if narrative == "" {
			narrative = fmt.Sprintf("%s successfully overcomes the challenge.", npc.Name)
		}
		events = append(events, NarrativeEvent{Text: narrative})
	case dice.Tie:
		events = append(events, NarrativeEvent{Text: fmt.Sprintf("%s manages to push through, but not without difficulty.", npc.Name)})
	default:
		events = append(events, NarrativeEvent{Text: fmt.Sprintf("%s is unable to overcome the obstacle.", npc.Name)})
	}

	return events
}

// processNPCAttack handles an NPC attacking a target.
// Returns (events, awaitingInvoke).
// When the target is the player and defense
// invokes are available, sm.pendingInvoke is set and awaitingInvoke is true.
// The invoke continuation finishes the attack (narrative, damage) and resumes
// processing remaining NPC turns via advanceConflictTurns.
func (sm *SceneManager) processNPCAttack(ctx context.Context, npc *character.Character, decision *prompt.NPCActionDecision) ([]GameEvent, bool) {
	// Determine target
	target := sm.player // Default to player
	targetID := decision.TargetID
	if targetID != "" && targetID != sm.player.ID {
		if t := sm.engine.GetCharacter(targetID); t != nil {
			target = t
		}
	}

	// Use the skill from the decision, or default based on conflict type
	attackSkill := decision.Skill
	if attackSkill == "" {
		attackSkill = sm.getDefaultAttackSkill()
	}

	// Determine defense skill based on attack skill
	defenseSkill := core.DefenseSkillForAttack(attackSkill)

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(attackSkill)

	// Roll NPC's attack
	npcRoll := sm.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))

	// Get target's defense (check for full defense bonus)
	targetDefenseLevel := target.GetSkill(defenseSkill)
	defenseBonus := 0
	if participant := sm.currentScene.GetParticipant(target.ID); participant != nil && participant.FullDefense {
		defenseBonus = 2
	}
	targetDefense := sm.roller.RollWithModifier(dice.Mediocre, int(targetDefenseLevel)+defenseBonus)

	fullDefense := defenseBonus > 0

	// Initial outcome (before player invokes)
	initialOutcome := npcRoll.CompareAgainst(targetDefense.FinalValue)

	// If target is the player, use the event-driven invoke path for defense
	if target.ID == sm.player.ID {
		// Show attack setup before the invoke prompt
		preEvents := []GameEvent{
			NPCAttackEvent{
				AttackerName:   npc.Name,
				TargetName:     target.Name,
				AttackSkill:    attackSkill,
				AttackResult:   npcRoll.FinalValue.String(),
				DefenseSkill:   defenseSkill,
				DefenseResult:  targetDefense.FinalValue.String(),
				FullDefense:    fullDefense,
				InitialOutcome: initialOutcome.Type.String(),
				FinalOutcome:   initialOutcome.Type.String(),
			},
		}

		defenseAction := action.NewAction("defense-invoke", sm.player.ID, action.Defend, defenseSkill, "Defending against attack")

		// Capture variables for the continuation closure
		capturedNPC := npc
		capturedAttackSkill := attackSkill
		capturedNPCRoll := npcRoll
		capturedInitialOutcome := initialOutcome

		finish := func(finishCtx context.Context, result *dice.CheckResult, accEvents []GameEvent) []GameEvent {
			return sm.finishNPCAttackOnPlayer(finishCtx, result, accEvents, capturedNPC, capturedAttackSkill, capturedNPCRoll, capturedInitialOutcome)
		}

		evts, awaiting := sm.beginInvokeLoop(ctx, targetDefense, npcRoll.FinalValue, defenseAction, true, preEvents, finish)
		// Mark the pending invoke for NPC turn resumption after it resolves.
		if awaiting && sm.pendingInvoke != nil {
			sm.pendingInvoke.resumeTurns = true
		}
		return evts, awaiting
	}

	// Non-player target: compute outcome directly (no invoke opportunity)
	outcome := npcRoll.CompareAgainst(targetDefense.FinalValue)

	npcNarrative, err := sm.generateNPCAttackNarrative(ctx, npc, attackSkill, outcome)
	if err != nil {
		slog.Error("Failed to generate NPC attack narrative",
			"component", componentSceneManager,
			"npc_id", npc.ID,
			"error", err)
		npcNarrative = npcAttackFallbackNarrative(npc.Name, outcome)
	}

	var events []GameEvent
	events = append(events, NPCAttackEvent{
		AttackerName:   npc.Name,
		TargetName:     target.Name,
		AttackSkill:    attackSkill,
		AttackResult:   npcRoll.FinalValue.String(),
		DefenseSkill:   defenseSkill,
		DefenseResult:  targetDefense.FinalValue.String(),
		FullDefense:    fullDefense,
		InitialOutcome: outcome.Type.String(),
		FinalOutcome:   outcome.Type.String(),
		Narrative:      npcNarrative,
	})

	if outcome.Type == dice.Success || outcome.Type == dice.SuccessWithStyle {
		events = append(events, NarrativeEvent{Text: fmt.Sprintf("%s takes %d shifts of stress!", target.Name, outcome.Shifts)})
	}

	return events, false
}

// finishNPCAttackOnPlayer is the invoke continuation for NPC attacks against
// the player. It computes the final outcome after defense invokes, generates
// narrative, and applies damage. NPC turn resumption is handled separately
// by maybeResumeConflictTurns in ProvideInvokeResponse.
func (sm *SceneManager) finishNPCAttackOnPlayer(
	ctx context.Context,
	defenseResult *dice.CheckResult,
	accEvents []GameEvent,
	npc *character.Character,
	attackSkill string,
	npcRoll *dice.CheckResult,
	initialOutcome *dice.Outcome,
) []GameEvent {
	var events []GameEvent
	events = append(events, accEvents...)

	// Compare final defense against NPC attack
	outcome := npcRoll.CompareAgainst(defenseResult.FinalValue)

	// Show outcome change if invokes altered the result
	if outcome.Type != initialOutcome.Type {
		events = append(events, OutcomeChangedEvent{
			FinalOutcome: outcome.Type.String(),
		})
	}

	// Generate narrative for the final outcome
	npcNarrative, err := sm.generateNPCAttackNarrative(ctx, npc, attackSkill, outcome)
	if err != nil {
		slog.Error("Failed to generate NPC attack narrative",
			"component", componentSceneManager,
			"npc_id", npc.ID,
			"error", err)
		npcNarrative = npcAttackFallbackNarrative(npc.Name, outcome)
	}
	events = append(events, NarrativeEvent{Text: npcNarrative})

	// Apply damage to the player
	attackCtx := prompt.AttackContext{
		Skill:       attackSkill,
		Description: npcNarrative,
		Shifts:      outcome.Shifts,
	}
	damageEvents := sm.applyAttackDamageToPlayer(ctx, outcome, npc, attackCtx)
	events = append(events, damageEvents...)

	return events
}

// generateNPCAttackNarrative generates narrative for an NPC's attack
func (sm *SceneManager) generateNPCAttackNarrative(ctx context.Context, npc *character.Character, skill string, outcome *dice.Outcome) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("LLM client required for NPC narratives")
	}

	// Build outcome description based on result
	outcomeDesc := "misses completely"
	switch outcome.Type {
	case dice.SuccessWithStyle:
		outcomeDesc = "lands a devastating blow"
	case dice.Success:
		outcomeDesc = "connects solidly"
	case dice.Tie:
		outcomeDesc = "is barely deflected, but creates an opening"
	case dice.Failure:
		outcomeDesc = "misses completely"
	}

	// Determine conflict type string
	conflictType := "physical"
	if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	// Get round number
	round := 1
	if sm.currentScene.ConflictState != nil {
		round = sm.currentScene.ConflictState.Round
	}

	// Build template data with full context
	data := prompt.NPCAttackData{
		ConflictType:       conflictType,
		Round:              round,
		SceneName:          sm.currentScene.Name,
		NPCName:            npc.Name,
		NPCHighConcept:     npc.Aspects.HighConcept,
		NPCAspects:         npc.Aspects.GetAll(),
		Skill:              skill,
		TargetName:         sm.player.Name,
		TargetHighConcept:  sm.player.Aspects.HighConcept,
		SituationAspects:   sm.currentScene.SituationAspects,
		OutcomeDescription: outcomeDesc,
	}

	promptText, err := prompt.RenderNPCAttack(data)
	if err != nil {
		return "", fmt.Errorf("failed to render NPC attack template: %w", err)
	}

	return llm.SimpleCompletion(ctx, sm.engine.llmClient, promptText, 100, 0.8)
}

// npcAttackFallbackNarrative returns a simple hit/miss narrative when LLM
// narrative generation is unavailable.
func npcAttackFallbackNarrative(npcName string, outcome *dice.Outcome) string {
	if outcome.Type == dice.Success || outcome.Type == dice.SuccessWithStyle {
		return npcName + "'s attack hits!"
	}
	return npcName + "'s attack misses."
}
