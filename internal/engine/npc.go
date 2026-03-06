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
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
)

// getNPCActionDecision uses the LLM to decide what action an NPC should take
func (cm *ConflictManager) getNPCActionDecision(ctx context.Context, npc *core.Character) (*prompt.NPCActionDecision, error) {
	if cm.llmClient == nil {
		return nil, fmt.Errorf("LLM client required for NPC decisions")
	}

	conflictType := cm.conflictTypeString()

	// Build target list (all active participants except this NPC)
	var targets []prompt.NPCTargetInfo
	for _, p := range cm.currentScene.ConflictState.Participants {
		if p.CharacterID == npc.ID || p.Status != scene.StatusActive {
			continue
		}
		char := cm.characters.GetCharacter(p.CharacterID)
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
		Round:             cm.currentScene.ConflictState.Round,
		SceneName:         cm.currentScene.Name,
		SceneDescription:  cm.currentScene.Description,
		NPCName:           npc.Name,
		NPCHighConcept:    npc.Aspects.HighConcept,
		NPCTrouble:        npc.Aspects.Trouble,
		NPCAspects:        npc.Aspects.GetAll(),
		NPCSkills:         npcSkills,
		NPCPhysicalStress: physicalStress,
		NPCMentalStress:   mentalStress,
		Targets:           targets,
		SituationAspects:  cm.currentScene.SituationAspects,
	}

	promptText, err := prompt.RenderNPCActionDecision(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render NPC action decision template: %w", err)
	}

	content, err := llm.SimpleCompletion(ctx, cm.llmClient, promptText, 150, 0.7)
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
func (cm *ConflictManager) processNPCTurn(ctx context.Context, npcID string) ([]GameEvent, bool) {
	npc := cm.characters.GetCharacter(npcID)
	if npc == nil {
		slog.Warn("NPC not found for turn processing",
			"component", componentSceneManager,
			"npc_id", npcID)
		return nil, false
	}

	var events []GameEvent
	events = append(events, TurnAnnouncementEvent{CharacterName: npc.Name, TurnNumber: cm.currentScene.ConflictState.Round, IsPlayer: false})

	// Get LLM decision for NPC action
	decision, err := cm.getNPCActionDecision(ctx, npc)
	if err != nil {
		slog.Warn("Failed to get NPC action decision, defaulting to attack",
			"component", componentSceneManager,
			"npc", npc.Name,
			"error", err)
		// Fallback to simple attack
		decision = &prompt.NPCActionDecision{
			ActionType:  "ATTACK",
			Skill:       cm.getDefaultAttackSkill(),
			TargetID:    cm.player.ID,
			Description: fmt.Sprintf("%s attacks!", npc.Name),
		}
	}

	// Process the decision based on action type
	switch strings.ToUpper(decision.ActionType) {
	case "DEFEND":
		events = append(events, cm.processNPCDefend(ctx, npc, decision)...)
	case "CREATE_ADVANTAGE":
		events = append(events, cm.processNPCCreateAdvantage(ctx, npc, decision)...)
	case "OVERCOME":
		events = append(events, cm.processNPCOvercome(ctx, npc, decision)...)
	default: // ATTACK or unknown
		attackEvents, awaiting := cm.processNPCAttack(ctx, npc, decision)
		events = append(events, attackEvents...)
		return events, awaiting
	}

	return events, false
}

// getDefaultAttackSkill returns the default attack skill based on conflict type
func (cm *ConflictManager) getDefaultAttackSkill() string {
	return core.DefaultAttackSkillForConflict(cm.currentScene.ConflictState.Type)
}

// processNPCDefend handles an NPC choosing full defense
func (cm *ConflictManager) processNPCDefend(ctx context.Context, npc *core.Character, decision *prompt.NPCActionDecision) []GameEvent {
	// Set full defense flag
	cm.currentScene.ConflictState.SetFullDefense(npc.ID)

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
func (cm *ConflictManager) processNPCCreateAdvantage(ctx context.Context, npc *core.Character, decision *prompt.NPCActionDecision) []GameEvent {
	skill := decision.Skill
	if skill == "" {
		skill = core.SkillNotice // Default
	}

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(skill)

	// Roll against Fair (+2) difficulty for creating aspects
	npcRoll := cm.actions.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))
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
		cm.currentScene.AddSituationAspect(situationAspect)
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
		// On a tie, NPC gets a boost instead of a full aspect.
		boostDesc := decision.Description
		if boostDesc == "" {
			boostDesc = fmt.Sprintf("%s creates a fleeting advantage", npc.Name)
		}
		boostName := cm.actions.generateBoostName(ctx, npc, skill, boostDesc, fmt.Sprintf("%s's Opportunity", npc.Name))
		events = append(events, NPCActionResultEvent{
			NPCName:       npc.Name,
			ActionType:    "create_advantage",
			Skill:         skill,
			RollResult:    npcRoll.FinalValue.String(),
			Difficulty:    "Fair",
			Outcome:       outcome.Type.String(),
			AspectCreated: boostName,
			FreeInvokes:   1,
		})
		events = append(events, cm.actions.createBoost(boostName, npc.ID))
		events = append(events, NarrativeEvent{Text: fmt.Sprintf("%s's maneuver creates a fleeting opening.", npc.Name)})
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
func (cm *ConflictManager) processNPCOvercome(ctx context.Context, npc *core.Character, decision *prompt.NPCActionDecision) []GameEvent {
	skill := decision.Skill
	if skill == "" {
		skill = core.SkillAthletics // Default
	}

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(skill)

	// Roll against Fair (+2) difficulty
	npcRoll := cm.actions.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))
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
	case dice.SuccessWithStyle:
		narrative := decision.Description
		if narrative == "" {
			narrative = fmt.Sprintf("%s successfully overcomes the challenge with style.", npc.Name)
		}
		events = append(events, NarrativeEvent{Text: narrative})
		// Overcome SWS grants a boost in addition to achieving the goal.
		overcomeSWS := decision.Description
		if overcomeSWS == "" {
			overcomeSWS = fmt.Sprintf("%s overcomes with style", npc.Name)
		}
		boostName := cm.actions.generateBoostName(ctx, npc, skill, overcomeSWS, "Strong Momentum")
		events = append(events, cm.actions.createBoost(boostName, npc.ID))
	case dice.Success:
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
// invokes are available, cm.actions.pendingInvoke is set and awaitingInvoke is true.
// The invoke continuation finishes the attack (narrative, damage) and resumes
// processing remaining NPC turns via advanceConflictTurns.
func (cm *ConflictManager) processNPCAttack(ctx context.Context, npc *core.Character, decision *prompt.NPCActionDecision) ([]GameEvent, bool) {
	// Determine target
	target := cm.player // Default to player
	targetID := decision.TargetID
	if targetID != "" && targetID != cm.player.ID {
		if t := cm.characters.GetCharacter(targetID); t != nil {
			target = t
		}
	}

	// Use the skill from the decision, or default based on conflict type
	attackSkill := decision.Skill
	if attackSkill == "" {
		attackSkill = cm.getDefaultAttackSkill()
	}

	// Determine defense skill based on attack skill
	defenseSkill := core.DefenseSkillForAttack(attackSkill)

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(attackSkill)

	// Roll NPC's attack
	npcRoll := cm.actions.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))

	// Get target's defense (check for full defense bonus)
	targetDefenseLevel := target.GetSkill(defenseSkill)
	defenseBonus := 0
	if participant := cm.currentScene.ConflictState.GetParticipant(target.ID); participant != nil && participant.FullDefense {
		defenseBonus = 2
	}
	targetDefense := cm.actions.roller.RollWithModifier(dice.Mediocre, int(targetDefenseLevel)+defenseBonus)

	fullDefense := defenseBonus > 0

	// Initial outcome (before player invokes)
	initialOutcome := npcRoll.CompareAgainst(targetDefense.FinalValue)

	// If target is the player, use the event-driven invoke path for defense
	if target.ID == cm.player.ID {
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

		defenseAction := action.NewAction("defense-invoke", cm.player.ID, action.Defend, defenseSkill, "Defending against attack")

		// Capture variables for the continuation closure
		capturedNPC := npc
		capturedAttackSkill := attackSkill
		capturedNPCRoll := npcRoll
		capturedInitialOutcome := initialOutcome

		finish := func(finishCtx context.Context, result *dice.CheckResult, accEvents []GameEvent) []GameEvent {
			return cm.finishNPCAttackOnPlayer(finishCtx, result, accEvents, capturedNPC, capturedAttackSkill, capturedNPCRoll, capturedInitialOutcome)
		}

		evts, awaiting := cm.actions.beginInvokeLoop(ctx, targetDefense, npcRoll.FinalValue, defenseAction, true, preEvents, finish)
		// Mark the pending invoke for NPC turn resumption after it resolves.
		if awaiting && cm.actions.pendingInvoke != nil {
			cm.actions.pendingInvoke.resumeTurns = true
		}
		return evts, awaiting
	}

	// Non-player target: compute outcome directly (no invoke opportunity)
	outcome := npcRoll.CompareAgainst(targetDefense.FinalValue)

	npcNarrative, err := cm.generateNPCAttackNarrative(ctx, npc, attackSkill, outcome)
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
	} else if outcome.Type == dice.Tie {
		// On a tie, attacker (NPC) gets a boost (no damage to target).
		tieDesc := fmt.Sprintf("%s attacks %s but is evenly matched", npc.Name, target.Name)
		boostName := cm.actions.generateBoostName(ctx, npc, attackSkill, tieDesc, "Fleeting Opening")
		events = append(events, cm.actions.createBoost(boostName, npc.ID))
	} else if outcome.Type == dice.Failure && outcome.Shifts <= -3 {
		// Target defended with style — target gets a boost.
		defDesc := fmt.Sprintf("defending against %s's attack", npc.Name)
		defSkill := core.DefenseSkillForAttack(attackSkill)
		boostName := cm.actions.generateBoostName(ctx, target, defSkill, defDesc, "Deflected with Ease")
		events = append(events, cm.actions.createBoost(boostName, target.ID))
	}

	return events, false
}

// finishNPCAttackOnPlayer is the invoke continuation for NPC attacks against
// the player. It computes the final outcome after defense invokes, generates
// narrative, and applies damage. NPC turn resumption is handled separately
// by maybeResumeConflictTurns in ProvideInvokeResponse.
func (cm *ConflictManager) finishNPCAttackOnPlayer(
	ctx context.Context,
	defenseResult *dice.CheckResult,
	accEvents []GameEvent,
	npc *core.Character,
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
	npcNarrative, err := cm.generateNPCAttackNarrative(ctx, npc, attackSkill, outcome)
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
	damageEvents := cm.applyAttackDamageToPlayer(ctx, outcome, npc, attackCtx)
	events = append(events, damageEvents...)

	return events
}

// generateNPCAttackNarrative generates narrative for an NPC's attack
func (cm *ConflictManager) generateNPCAttackNarrative(ctx context.Context, npc *core.Character, skill string, outcome *dice.Outcome) (string, error) {
	if cm.llmClient == nil {
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

	conflictType := cm.conflictTypeString()

	// Get round number
	round := 1
	if cm.currentScene.ConflictState != nil {
		round = cm.currentScene.ConflictState.Round
	}

	// Build template data with full context
	data := prompt.NPCAttackData{
		ConflictType:       conflictType,
		Round:              round,
		SceneName:          cm.currentScene.Name,
		NPCName:            npc.Name,
		NPCHighConcept:     npc.Aspects.HighConcept,
		NPCAspects:         npc.Aspects.GetAll(),
		Skill:              skill,
		TargetName:         cm.player.Name,
		TargetHighConcept:  cm.player.Aspects.HighConcept,
		SituationAspects:   cm.currentScene.SituationAspects,
		OutcomeDescription: outcomeDesc,
	}

	promptText, err := prompt.RenderNPCAttack(data)
	if err != nil {
		return "", fmt.Errorf("failed to render NPC attack template: %w", err)
	}

	return llm.SimpleCompletion(ctx, cm.llmClient, promptText, 100, 0.8)
}

// npcAttackFallbackNarrative returns a simple hit/miss narrative when LLM
// narrative generation is unavailable.
func npcAttackFallbackNarrative(npcName string, outcome *dice.Outcome) string {
	if outcome.Type == dice.Success || outcome.Type == dice.SuccessWithStyle {
		return npcName + "'s attack hits!"
	}
	return npcName + "'s attack misses."
}
