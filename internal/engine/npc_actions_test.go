package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

// setupNPCConflictSM creates a SceneManager with a player and NPC in an active
// physical conflict. The roller is left as default; callers should replace it
// with a PlannedRoller before calling NPC action functions.
func setupNPCConflictSM(t *testing.T) (*SceneManager, *character.Character, *character.Character) {
	t.Helper()

	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	npc := character.NewCharacter("npc-1", "Goblin Scout")
	npc.Aspects.HighConcept = "Sneaky Goblin"
	npc.SetSkill("Notice", 2)
	npc.SetSkill("Athletics", 2)
	npc.SetSkill("Fight", 2)

	engine.AddCharacter(player)
	engine.AddCharacter(npc)

	testScene := scene.NewScene("test-scene", "Forest Clearing", "A dim clearing in the woods.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, npc.ID)
	require.NoError(t, err)

	return sm, player, npc
}

// --- processNPCDefend ---

// Fate Core SRD: Full Defense grants +2 to all defense rolls until your next
// turn. The NPC emits a defend action result and narrative.
func TestProcessNPCDefend_Default(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	decision := &prompt.NPCActionDecision{
		ActionType: "DEFEND",
	}

	events := sm.conflict.processNPCDefend(context.Background(), npc, decision)

	require.Len(t, events, 2)

	actionEvt, ok := events[0].(NPCActionResultEvent)
	require.True(t, ok, "expected NPCActionResultEvent, got %T", events[0])
	assert.Equal(t, npc.Name, actionEvt.NPCName)
	assert.Equal(t, "defend", actionEvt.ActionType)

	narrEvt, ok := events[1].(NarrativeEvent)
	require.True(t, ok, "expected NarrativeEvent, got %T", events[1])
	assert.Contains(t, narrEvt.Text, npc.Name)

	// Full defense flag should be set on the scene participant.
	assert.True(t, sm.currentScene.ConflictState.IsFullDefense(npc.ID))
}

func TestProcessNPCDefend_CustomDescription(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	decision := &prompt.NPCActionDecision{
		ActionType:  "DEFEND",
		Description: "The goblin ducks behind a rock.",
	}

	events := sm.conflict.processNPCDefend(context.Background(), npc, decision)

	narrEvt := events[1].(NarrativeEvent)
	assert.Equal(t, "The goblin ducks behind a rock.", narrEvt.Text)
}

// --- processNPCCreateAdvantage ---

// Fate Core SRD: Create an Advantage with Success creates a new situation
// aspect with one free invoke.
func TestProcessNPCCreateAdvantage_Success(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// NPC Notice +2, rolled against Fair (+2).
	// Dice total 1 → final Good(+3), shifts=1 → Success.
	sm.actions.roller = dice.NewPlannedRoller([]int{1})

	decision := &prompt.NPCActionDecision{
		ActionType:  "CREATE_ADVANTAGE",
		Skill:       "Notice",
		Description: "Hidden Snare",
	}

	events := sm.conflict.processNPCCreateAdvantage(context.Background(), npc, decision)

	require.Len(t, events, 2)

	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "create_advantage", actionEvt.ActionType)
	assert.Equal(t, "Notice", actionEvt.Skill)
	assert.Equal(t, "Success", actionEvt.Outcome)
	assert.Equal(t, "Hidden Snare", actionEvt.AspectCreated)
	assert.Equal(t, 1, actionEvt.FreeInvokes)

	// Situation aspect should be added to the scene.
	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.Equal(t, "Hidden Snare", sm.currentScene.SituationAspects[0].Aspect)
	assert.Equal(t, 1, sm.currentScene.SituationAspects[0].FreeInvokes)
}

// Fate Core SRD: Create an Advantage with Success with Style grants 2 free
// invokes instead of 1.
func TestProcessNPCCreateAdvantage_SuccessWithStyle(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// Dice total 3 → final Superb(+5), shifts=3 → SWS.
	sm.actions.roller = dice.NewPlannedRoller([]int{3})

	decision := &prompt.NPCActionDecision{
		ActionType:  "CREATE_ADVANTAGE",
		Skill:       "Notice",
		Description: "Perfect Ambush Position",
	}

	events := sm.conflict.processNPCCreateAdvantage(context.Background(), npc, decision)

	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "Success with Style", actionEvt.Outcome)
	assert.Equal(t, 2, actionEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.Equal(t, 2, sm.currentScene.SituationAspects[0].FreeInvokes)
}

// Fate Core SRD: Tie on Create Advantage — you get a boost (temporary aspect with
// one free invoke) instead of a full situation aspect.
func TestProcessNPCCreateAdvantage_Tie(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// Dice total 0 → final Fair(+2), shifts=0 → Tie.
	sm.actions.roller = dice.NewPlannedRoller([]int{0})

	decision := &prompt.NPCActionDecision{
		ActionType: "CREATE_ADVANTAGE",
		Skill:      "Notice",
	}

	events := sm.conflict.processNPCCreateAdvantage(context.Background(), npc, decision)

	// Tie produces 3 events: NPCActionResultEvent + AspectCreatedEvent (boost) + NarrativeEvent.
	require.Len(t, events, 3)
	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "Tie", actionEvt.Outcome)
	assert.NotEmpty(t, actionEvt.AspectCreated, "Tie should create a boost")
	assert.Equal(t, 1, actionEvt.FreeInvokes, "Boost grants 1 free invoke")

	// A boost (IsBoost=true) should be added to the scene.
	require.Len(t, sm.currentScene.SituationAspects, 1)
	boost := sm.currentScene.SituationAspects[0]
	assert.True(t, boost.IsBoost, "aspect should be flagged as a boost")
	assert.Equal(t, 1, boost.FreeInvokes)

	boostEvt, ok := events[1].(AspectCreatedEvent)
	require.True(t, ok, "expected AspectCreatedEvent, got %T", events[1])
	assert.True(t, boostEvt.IsBoost)
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	narrEvt := events[2].(NarrativeEvent)
	assert.Contains(t, narrEvt.Text, npc.Name)
}

// Fate Core SRD: Failure on Create Advantage — nothing happens, or the GM may
// create a situation aspect that works against you.
func TestProcessNPCCreateAdvantage_Failure(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// Dice total -1 → final Average(+1), shifts=-1 → Failure.
	sm.actions.roller = dice.NewPlannedRoller([]int{-1})

	decision := &prompt.NPCActionDecision{
		ActionType: "CREATE_ADVANTAGE",
		Skill:      "Notice",
	}

	events := sm.conflict.processNPCCreateAdvantage(context.Background(), npc, decision)

	require.Len(t, events, 2)
	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "Failure", actionEvt.Outcome)
	assert.Empty(t, actionEvt.AspectCreated)

	// No situation aspect added.
	assert.Empty(t, sm.currentScene.SituationAspects)
}

// Verify default skill "Notice" is used when decision has empty skill.
func TestProcessNPCCreateAdvantage_DefaultSkill(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	sm.actions.roller = dice.NewPlannedRoller([]int{1})

	decision := &prompt.NPCActionDecision{
		ActionType: "CREATE_ADVANTAGE",
		Skill:      "", // empty → defaults to Notice
	}

	events := sm.conflict.processNPCCreateAdvantage(context.Background(), npc, decision)
	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "Notice", actionEvt.Skill)
}

// --- processNPCOvercome ---

// Fate Core SRD: Overcome with Success — the obstacle is dealt with.
func TestProcessNPCOvercome_Success(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// Dice total 1 → Success (shifts=1).
	sm.actions.roller = dice.NewPlannedRoller([]int{1})

	decision := &prompt.NPCActionDecision{
		ActionType:  "OVERCOME",
		Skill:       "Athletics",
		Description: "The goblin leaps over the barricade.",
	}

	events := sm.conflict.processNPCOvercome(context.Background(), npc, decision)

	require.Len(t, events, 2)
	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "overcome", actionEvt.ActionType)
	assert.Equal(t, "Athletics", actionEvt.Skill)
	assert.Equal(t, "Success", actionEvt.Outcome)

	// Custom description should be used as narrative.
	narrEvt := events[1].(NarrativeEvent)
	assert.Equal(t, "The goblin leaps over the barricade.", narrEvt.Text)
}

// Fate Core SRD: Overcome with Tie — you succeed at a minor cost, or fail
// but gain a boost.
func TestProcessNPCOvercome_Tie(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// Dice total 0 → Tie.
	sm.actions.roller = dice.NewPlannedRoller([]int{0})

	decision := &prompt.NPCActionDecision{
		ActionType: "OVERCOME",
		Skill:      "Athletics",
	}

	events := sm.conflict.processNPCOvercome(context.Background(), npc, decision)

	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "Tie", actionEvt.Outcome)

	narrEvt := events[1].(NarrativeEvent)
	assert.Contains(t, narrEvt.Text, npc.Name)
}

// Fate Core SRD: Overcome with Failure — the obstacle stands.
func TestProcessNPCOvercome_Failure(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// Dice total -1 → Failure.
	sm.actions.roller = dice.NewPlannedRoller([]int{-1})

	decision := &prompt.NPCActionDecision{
		ActionType: "OVERCOME",
		Skill:      "Athletics",
	}

	events := sm.conflict.processNPCOvercome(context.Background(), npc, decision)

	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "Failure", actionEvt.Outcome)

	narrEvt := events[1].(NarrativeEvent)
	assert.Contains(t, narrEvt.Text, "unable to overcome")
}

// Verify default skill "Athletics" is used when decision has empty skill.
func TestProcessNPCOvercome_DefaultSkill(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	sm.actions.roller = dice.NewPlannedRoller([]int{1})

	decision := &prompt.NPCActionDecision{
		ActionType: "OVERCOME",
		Skill:      "",
	}

	events := sm.conflict.processNPCOvercome(context.Background(), npc, decision)
	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "Athletics", actionEvt.Skill)
}

// --- processNPCTurn dispatch ---

// processNPCTurn should dispatch to the correct handler based on ActionType.
// We test this with an LLM-driven decision by providing a mock LLM that
// returns a DEFEND decision.
func TestProcessNPCTurn_DispatchDefend(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: `{"action_type":"DEFEND","skill":"","target_id":"","description":"hunkers down"}`,
	}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	npc := character.NewCharacter("npc-1", "Goblin Scout")
	npc.Aspects.HighConcept = "Sneaky Goblin"

	engine.AddCharacter(player)
	engine.AddCharacter(npc)

	testScene := scene.NewScene("test-scene", "Forest", "Woods.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, npc.ID)
	require.NoError(t, err)

	events, awaiting := sm.conflict.processNPCTurn(context.Background(), npc.ID)
	assert.False(t, awaiting, "defend should not await invoke")

	// Should contain TurnAnnouncementEvent + NPCActionResultEvent + NarrativeEvent.
	var hasAnnounce, hasAction bool
	for _, e := range events {
		switch e.(type) {
		case TurnAnnouncementEvent:
			hasAnnounce = true
		case NPCActionResultEvent:
			hasAction = true
		}
	}
	assert.True(t, hasAnnounce, "expected TurnAnnouncementEvent")
	assert.True(t, hasAction, "expected NPCActionResultEvent for defend")

	assert.True(t, sm.currentScene.ConflictState.IsFullDefense(npc.ID))
}

// When LLM fails, processNPCTurn should fall back to an attack.
func TestProcessNPCTurn_FallbackToAttack(t *testing.T) {
	// No LLM → getNPCActionDecision errors → fallback to attack.
	sm, player, npc := setupNPCConflictSM(t)

	// PlannedRoller: first roll for NPC attack, second for player defense.
	// NPC Fight +2: dice -4 → final = 0+(-4)+2 = -2 (Terrible).
	// Player defense: dice +4 → high defense. Attack will fail.
	sm.actions.roller = dice.NewPlannedRoller([]int{-4, 4})

	events, awaiting := sm.conflict.processNPCTurn(context.Background(), npc.ID)

	// Without invoke-eligible aspects, attack resolves immediately.
	_ = awaiting
	_ = player

	var hasAnnounce bool
	for _, e := range events {
		if _, ok := e.(TurnAnnouncementEvent); ok {
			hasAnnounce = true
		}
	}
	assert.True(t, hasAnnounce, "expected TurnAnnouncementEvent")
	// We don't assert the full attack chain here — that's covered by separate
	// attack tests. Just verifying the dispatch falls through to attack.
}

// --- Boost creation paths (Fate Core SRD) ---

// Fate Core SRD (Overcome, SWS): NPC overcome with SWS grants the NPC a boost.
func TestProcessNPCOvercome_SuccessWithStyle_CreatesBoost(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// Dice total 3 → final Superb(+5), shifts=3 → SWS.
	sm.actions.roller = dice.NewPlannedRoller([]int{3})

	decision := &prompt.NPCActionDecision{
		ActionType: "OVERCOME",
		Skill:      "Athletics",
	}

	events := sm.conflict.processNPCOvercome(context.Background(), npc, decision)

	// SWS produces 3 events: NPCActionResultEvent + NarrativeEvent + AspectCreatedEvent.
	require.Len(t, events, 3)
	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "Success with Style", actionEvt.Outcome)

	narrEvt := events[1].(NarrativeEvent)
	assert.Contains(t, narrEvt.Text, npc.Name)

	boostEvt, ok := events[2].(AspectCreatedEvent)
	require.True(t, ok, "expected AspectCreatedEvent for NPC overcome SWS boost, got %T", events[2])
	assert.True(t, boostEvt.IsBoost)
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, npc.ID, sm.currentScene.SituationAspects[0].CreatedBy)
}

// Fate Core SRD (Attack, Tie): NPC attack on non-player target ties — attacker gets boost.
func TestProcessNPCAttack_NonPlayerTarget_Tie_CreatesBoost(t *testing.T) {
	eng, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(eng, eng.llmClient, eng.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	npc := character.NewCharacter("npc-1", "Goblin Scout")
	npc.SetSkill("Fight", 2)
	target := character.NewCharacter("npc-target", "Orc Warrior")
	target.SetSkill("Athletics", dice.Fair)

	eng.AddCharacter(player)
	eng.AddCharacter(npc)
	eng.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Forest Clearing", "A dim clearing.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc.ID)
	testScene.AddCharacter(target.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, npc.ID)
	require.NoError(t, err)

	decision := &prompt.NPCActionDecision{
		ActionType: "ATTACK",
		Skill:      "Fight",
		TargetID:   target.ID,
	}

	// NPC Fight +2 roll 0 → Fair(+2). Target Athletics +2 roll 0 → Fair(+2). Tie (shifts=0).
	sm.actions.roller = dice.NewPlannedRoller([]int{0, 0})

	events, _ := sm.conflict.processNPCAttack(context.Background(), npc, decision)

	var boostEvt AspectCreatedEvent
	var found bool
	for _, e := range events {
		if b, ok := e.(AspectCreatedEvent); ok {
			boostEvt = b
			found = true
			break
		}
	}

	require.True(t, found, "expected AspectCreatedEvent for NPC attack tie on non-player target")
	assert.True(t, boostEvt.IsBoost)
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, npc.ID, sm.currentScene.SituationAspects[0].CreatedBy, "attacker NPC gets the boost")
}

// Fate Core SRD (Defend): When the defender succeeds with style (attacker fails
// by ≥3 shifts), the defender gets a boost.
func TestProcessNPCAttack_NonPlayerTarget_DefendWithStyle_GrantsTargetBoost(t *testing.T) {
	eng, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(eng, eng.llmClient, eng.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	npc := character.NewCharacter("npc-1", "Goblin Scout")
	npc.SetSkill("Fight", 2)
	target := character.NewCharacter("npc-target", "Orc Warrior")
	target.SetSkill("Athletics", dice.Fair)

	eng.AddCharacter(player)
	eng.AddCharacter(npc)
	eng.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Forest Clearing", "A dim clearing.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc.ID)
	testScene.AddCharacter(target.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, npc.ID)
	require.NoError(t, err)

	decision := &prompt.NPCActionDecision{
		ActionType: "ATTACK",
		Skill:      "Fight",
		TargetID:   target.ID,
	}

	// NPC Fight +2 roll -4 → Terrible(-2). Target Athletics +2 roll 0 → Fair(+2).
	// Outcome: -2 vs +2 = -4 shifts (≤ -3) → Failure, defender succeeded with style.
	sm.actions.roller = dice.NewPlannedRoller([]int{-4, 0})

	events, _ := sm.conflict.processNPCAttack(context.Background(), npc, decision)

	var boostEvt AspectCreatedEvent
	var found bool
	for _, e := range events {
		if b, ok := e.(AspectCreatedEvent); ok {
			boostEvt = b
			found = true
			break
		}
	}

	require.True(t, found, "expected AspectCreatedEvent when target defends with style")
	assert.True(t, boostEvt.IsBoost)
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, target.ID, sm.currentScene.SituationAspects[0].CreatedBy, "defending target gets the boost")
}
