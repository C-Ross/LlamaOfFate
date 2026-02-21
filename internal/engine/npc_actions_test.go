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
	sm.player = player
	sm.conflict.player = player

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
	assert.True(t, sm.currentScene.IsFullDefense(npc.ID))
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
	sm.conflict.roller = dice.NewPlannedRoller([]int{1})

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
	sm.conflict.roller = dice.NewPlannedRoller([]int{3})

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

// Fate Core SRD: Tie on Create Advantage — you get a boost (temporary aspect)
// but no full situation aspect.
func TestProcessNPCCreateAdvantage_Tie(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// Dice total 0 → final Fair(+2), shifts=0 → Tie.
	sm.conflict.roller = dice.NewPlannedRoller([]int{0})

	decision := &prompt.NPCActionDecision{
		ActionType: "CREATE_ADVANTAGE",
		Skill:      "Notice",
	}

	events := sm.conflict.processNPCCreateAdvantage(context.Background(), npc, decision)

	require.Len(t, events, 2)
	actionEvt := events[0].(NPCActionResultEvent)
	assert.Equal(t, "Tie", actionEvt.Outcome)
	assert.Empty(t, actionEvt.AspectCreated, "Tie should not create a situation aspect")
	assert.Equal(t, 0, actionEvt.FreeInvokes)

	// No situation aspect added.
	assert.Empty(t, sm.currentScene.SituationAspects)

	narrEvt := events[1].(NarrativeEvent)
	assert.Contains(t, narrEvt.Text, npc.Name)
}

// Fate Core SRD: Failure on Create Advantage — nothing happens, or the GM may
// create a situation aspect that works against you.
func TestProcessNPCCreateAdvantage_Failure(t *testing.T) {
	sm, _, npc := setupNPCConflictSM(t)

	// Dice total -1 → final Average(+1), shifts=-1 → Failure.
	sm.conflict.roller = dice.NewPlannedRoller([]int{-1})

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

	sm.conflict.roller = dice.NewPlannedRoller([]int{1})

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
	sm.conflict.roller = dice.NewPlannedRoller([]int{1})

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
	sm.conflict.roller = dice.NewPlannedRoller([]int{0})

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
	sm.conflict.roller = dice.NewPlannedRoller([]int{-1})

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

	sm.conflict.roller = dice.NewPlannedRoller([]int{1})

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
	sm.player = player
	sm.conflict.player = player

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

	assert.True(t, sm.currentScene.IsFullDefense(npc.ID))
}

// When LLM fails, processNPCTurn should fall back to an attack.
func TestProcessNPCTurn_FallbackToAttack(t *testing.T) {
	// No LLM → getNPCActionDecision errors → fallback to attack.
	sm, player, npc := setupNPCConflictSM(t)

	// PlannedRoller: first roll for NPC attack, second for player defense.
	// NPC Fight +2: dice -4 → final = 0+(-4)+2 = -2 (Terrible).
	// Player defense: dice +4 → high defense. Attack will fail.
	sm.conflict.roller = dice.NewPlannedRoller([]int{-4, 4})

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
