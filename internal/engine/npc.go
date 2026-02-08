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

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: promptText}},
		MaxTokens:   150,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if resp.Content() == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Parse JSON response
	content := resp.Content()
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

// processNPCTurn handles an NPC's action during conflict
func (sm *SceneManager) processNPCTurn(ctx context.Context, npcID string) {
	npc := sm.engine.GetCharacter(npcID)
	if npc == nil {
		slog.Warn("NPC not found for turn processing",
			"component", componentSceneManager,
			"npc_id", npcID)
		return
	}

	sm.ui.DisplayTurnAnnouncement(npc.Name, sm.currentScene.ConflictState.Round, false)

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
		sm.processNPCDefend(ctx, npc, decision)
	case "CREATE_ADVANTAGE":
		sm.processNPCCreateAdvantage(ctx, npc, decision)
	case "OVERCOME":
		sm.processNPCOvercome(ctx, npc, decision)
	default: // ATTACK or unknown
		sm.processNPCAttack(ctx, npc, decision)
	}
}

// getDefaultAttackSkill returns the default attack skill based on conflict type
func (sm *SceneManager) getDefaultAttackSkill() string {
	return core.DefaultAttackSkillForConflict(sm.currentScene.ConflictState.Type)
}

// processNPCDefend handles an NPC choosing full defense
func (sm *SceneManager) processNPCDefend(ctx context.Context, npc *character.Character, decision *prompt.NPCActionDecision) {
	// Set full defense flag
	sm.currentScene.SetFullDefense(npc.ID)

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s takes a defensive stance! (+2 to all defense rolls this exchange)",
		npc.Name,
	))

	// Generate narrative
	narrative := fmt.Sprintf("%s braces for incoming attacks, focusing entirely on defense.", npc.Name)
	if decision.Description != "" {
		narrative = decision.Description
	}
	sm.ui.DisplayNarrative(narrative)
}

// processNPCCreateAdvantage handles an NPC creating an advantage
func (sm *SceneManager) processNPCCreateAdvantage(ctx context.Context, npc *character.Character, decision *prompt.NPCActionDecision) {
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

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s attempts to Create an Advantage with %s (%s vs Fair)",
		npc.Name,
		skill,
		npcRoll.FinalValue.String(),
	))

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
		sm.ui.DisplaySystemMessage(fmt.Sprintf(
			"Created aspect: \"%s\" with %d free invoke(s)!",
			aspectName,
			freeInvokes,
		))
		sm.ui.DisplayNarrative(fmt.Sprintf("%s gains a tactical advantage!", npc.Name))
	case dice.Tie:
		sm.ui.DisplaySystemMessage("The attempt succeeds but grants a boost to opponents!")
		sm.ui.DisplayNarrative(fmt.Sprintf("%s's maneuver is partially successful.", npc.Name))
	default:
		sm.ui.DisplaySystemMessage("The attempt fails!")
		sm.ui.DisplayNarrative(fmt.Sprintf("%s's gambit doesn't pay off.", npc.Name))
	}
}

// processNPCOvercome handles an NPC attempting to overcome an obstacle
func (sm *SceneManager) processNPCOvercome(ctx context.Context, npc *character.Character, decision *prompt.NPCActionDecision) {
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

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s attempts to Overcome with %s (%s vs Fair)",
		npc.Name,
		skill,
		npcRoll.FinalValue.String(),
	))

	switch outcome.Type {
	case dice.Success, dice.SuccessWithStyle:
		sm.ui.DisplaySystemMessage("The obstacle is overcome!")
		narrative := decision.Description
		if narrative == "" {
			narrative = fmt.Sprintf("%s successfully overcomes the challenge.", npc.Name)
		}
		sm.ui.DisplayNarrative(narrative)
	case dice.Tie:
		sm.ui.DisplaySystemMessage("Success, but at a minor cost.")
		sm.ui.DisplayNarrative(fmt.Sprintf("%s manages to push through, but not without difficulty.", npc.Name))
	default:
		sm.ui.DisplaySystemMessage("The attempt fails!")
		sm.ui.DisplayNarrative(fmt.Sprintf("%s is unable to overcome the obstacle.", npc.Name))
	}
}

// processNPCAttack handles an NPC attacking a target
func (sm *SceneManager) processNPCAttack(ctx context.Context, npc *character.Character, decision *prompt.NPCActionDecision) {
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

	// Display the mechanical result
	defenseDisplay := defenseSkill
	if defenseBonus > 0 {
		defenseDisplay = fmt.Sprintf("%s+2 (Full Defense)", defenseSkill)
	}
	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s attacks %s with %s (%s) vs %s (%s)",
		npc.Name,
		target.Name,
		attackSkill,
		npcRoll.FinalValue.String(),
		defenseDisplay,
		targetDefense.FinalValue.String(),
	))

	// Initial outcome (before player invokes)
	initialOutcome := npcRoll.CompareAgainst(targetDefense.FinalValue)
	sm.ui.DisplaySystemMessage(fmt.Sprintf("Initial outcome: %s", initialOutcome.Type.String()))

	// If target is the player, allow them to invoke aspects to improve defense
	if target.ID == sm.player.ID {
		// Create a temporary action to track invokes for defense
		defenseAction := action.NewAction("defense-invoke", sm.player.ID, action.Defend, defenseSkill, "Defending against attack")

		// Player can invoke to improve their defense
		// isDefense=true means skip prompt if attack already fails
		targetDefense = sm.handlePostRollInvokes(targetDefense, npcRoll.FinalValue, defenseAction, true)

		// Recalculate outcome with potentially improved defense
		// Note: For defense, we compare attacker vs defender, so we still use npcRoll.CompareAgainst
		// but the targetDefense.FinalValue may have increased
	}

	// Compare results (final)
	outcome := npcRoll.CompareAgainst(targetDefense.FinalValue)

	// Display updated outcome if it changed
	if outcome.Type != initialOutcome.Type {
		sm.ui.DisplaySystemMessage(fmt.Sprintf("Final outcome: %s", outcome.Type.String()))
	}

	// Generate narrative for the attack
	npcNarrative, err := sm.generateNPCAttackNarrative(ctx, npc, attackSkill, outcome)
	if err != nil {
		slog.Error("Failed to generate NPC attack narrative",
			"component", componentSceneManager,
			"npc_id", npc.ID,
			"error", err)
		// Fallback narrative
		if outcome.Type == dice.Success || outcome.Type == dice.SuccessWithStyle {
			npcNarrative = fmt.Sprintf("%s's attack hits!", npc.Name)
		} else {
			npcNarrative = fmt.Sprintf("%s's attack misses.", npc.Name)
		}
	}
	sm.ui.DisplayNarrative(npcNarrative)

	// Only apply damage if target is the player (for now, NPC vs NPC damage not fully implemented)
	if target.ID == sm.player.ID {
		// Create attack context with the skill, narrative, and shifts
		attackCtx := prompt.AttackContext{
			Skill:       attackSkill,
			Description: npcNarrative,
			Shifts:      outcome.Shifts,
		}
		sm.applyAttackDamageToPlayer(ctx, outcome, npc, attackCtx)
	} else {
		// For NPC targets, just show the result
		if outcome.Type == dice.Success || outcome.Type == dice.SuccessWithStyle {
			sm.ui.DisplaySystemMessage(fmt.Sprintf("%s takes %d shifts of stress!", target.Name, outcome.Shifts))
		}
	}
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

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: promptText}},
		MaxTokens:   100,
		Temperature: 0.8,
	})
	if err != nil {
		return "", err
	}

	if resp.Content() == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	return resp.Content(), nil
}
