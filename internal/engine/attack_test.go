package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAttackSM creates a SceneManager with a mock LLM, a player with Fight
// at Good (+3), a target NPC with Athletics at Fair (+2) for defense, and an
// active scene containing both characters.
//
// PlannedRoller sequence for a full attack turn cycle:
//
//	[0] player attack dice
//	[1] NPC defense dice
//	[2] NPC counter-attack dice (auto-processed after player's turn)
//	[3] player defense dice (against NPC counter-attack)
//
// Provide at least 4 planned rolls. Extra 0s are safe padding.
func setupAttackSM(t *testing.T, fatePoints int) (*SceneManager, *character.Character) {
	t.Helper()

	// The mock response must be valid JSON for the NPC action decision parser.
	mockClient := &MockLLMClient{response: `{"action":"attack","skill":"Fight","target":"player-1","description":"counter-attack","reasoning":"test"}`}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := engine.GetSceneManager()

	player := character.NewCharacter("player-1", "Hero")
	player.Aspects.HighConcept = "Fearsome Brawler"
	player.Aspects.Trouble = "Short Temper"
	player.FatePoints = fatePoints
	player.SetSkill("Fight", dice.Good)     // Good (+3)
	player.SetSkill("Athletics", dice.Fair) // Fair (+2) — defense against NPC attacks
	player.SetSkill("Notice", dice.Fair)    // Initiative skill

	npc := character.NewCharacter("npc-1", "Thug")
	npc.SetSkill("Fight", dice.Fair)      // Fair (+2) — NPC's attack skill
	npc.SetSkill("Athletics", dice.Fair)  // Fair (+2) — defense skill for Fight
	npc.SetSkill("Notice", dice.Mediocre) // Low initiative

	engine.AddCharacter(player)
	engine.AddCharacter(npc)

	testScene := scene.NewScene("test-scene", "Test Arena", "An arena for testing.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	return sm, npc
}

func makeAttackAction(playerID, targetName string) *action.Action {
	return action.NewActionWithTarget(
		"action-1",
		playerID,
		action.Attack,
		"Fight",
		"Punch the target",
		targetName,
	)
}

// --- resolveAction-level tests for Attack ---

// Fate Core SRD (Attack, Success): "If you succeed, you deal a hit equal to
// the number of shifts." Full pipeline test.
func TestResolveAction_Attack_Success_DealsDamage(t *testing.T) {
	sm, npc := setupAttackSM(t, 0)

	// [0] Attack dice 0 → Good(+3)+0 = +3.
	// [1] Defense dice 0 → Fair(+2)+0 = +2.
	// Outcome: +3 vs +2 = Success, 1 shift.
	// [2-3] NPC counter-attack: -1,0 → NPC fails (no boost created).
	sm.actions.roller = dice.NewPlannedRoller([]int{0, 0, -1, 0})

	testAction := makeAttackAction(sm.actions.player.ID, npc.Name)
	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.Success, testAction.Outcome.Type)
	assert.Equal(t, 1, testAction.Outcome.Shifts)

	// PlayerAttackResultEvent should show shifts dealt.
	atkEvt := RequireFirstFrom[PlayerAttackResultEvent](t, events)
	assert.Equal(t, npc.Name, atkEvt.TargetName)
	assert.Equal(t, 1, atkEvt.Shifts)
	assert.False(t, atkEvt.IsTie)

	// DamageResolutionEvent should exist (stress applied to NPC).
	dmgEvt := RequireFirstFrom[DamageResolutionEvent](t, events)
	assert.Equal(t, npc.Name, dmgEvt.TargetName)
	assert.False(t, dmgEvt.TakenOut)
}

// Fate Core SRD (Attack, SWS): "You deal a hit just like a success, but you
// also have the option to reduce ... shifts by one and create a boost."
// Full pipeline test — verifies extra shifts are dealt.
func TestResolveAction_Attack_SWS_DealsExtraDamage(t *testing.T) {
	sm, npc := setupAttackSM(t, 0)

	// [0] Attack dice 2 → Good(+3)+2 = +5.
	// [1] Defense dice 0 → Fair(+2)+0 = +2.
	// Outcome: +5 vs +2 = SWS, 3 shifts.
	// [2-3] NPC counter-attack: -1,0 → NPC fails (no boost created).
	sm.actions.roller = dice.NewPlannedRoller([]int{2, 0, -1, 0})

	testAction := makeAttackAction(sm.actions.player.ID, npc.Name)
	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.SuccessWithStyle, testAction.Outcome.Type)
	assert.Equal(t, 3, testAction.Outcome.Shifts)

	atkEvt := RequireFirstFrom[PlayerAttackResultEvent](t, events)
	assert.Equal(t, npc.Name, atkEvt.TargetName)
	assert.Equal(t, 3, atkEvt.Shifts)

	dmgEvt := RequireFirstFrom[DamageResolutionEvent](t, events)
	assert.Equal(t, npc.Name, dmgEvt.TargetName)
}

// Fate Core SRD (Attack, Tie): "You don't do any damage, but you get a boost."
// Full pipeline test.
func TestResolveAction_Attack_Tie_GrantsBoost(t *testing.T) {
	sm, npc := setupAttackSM(t, 0)

	// [0] Attack dice -1 → Good(+3)+(-1) = +2.
	// [1] Defense dice 0 → Fair(+2)+0 = +2.
	// Outcome: +2 vs +2 = Tie, 0 shifts.
	// [2-3] NPC counter-attack: -1,0 → NPC fails (no boost created).
	sm.actions.roller = dice.NewPlannedRoller([]int{-1, 0, -1, 0})

	testAction := makeAttackAction(sm.actions.player.ID, npc.Name)
	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.Tie, testAction.Outcome.Type)

	// PlayerAttackResultEvent should flag the tie.
	atkEvt := RequireFirstFrom[PlayerAttackResultEvent](t, events)
	assert.True(t, atkEvt.IsTie)
	assert.Equal(t, npc.Name, atkEvt.TargetName)

	// A boost should be created for the attacker.
	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost)
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, sm.actions.player.ID, sm.currentScene.SituationAspects[0].CreatedBy)
}

// Fate Core SRD (Attack, Failure): "You don't deal any damage. ... If the
// defender succeeds with style, they get a boost."
// This test covers simple failure (defender does not get SWS).
func TestResolveAction_Attack_Failure_NoDamage(t *testing.T) {
	sm, npc := setupAttackSM(t, 0)

	// [0] Attack dice -2 → Good(+3)+(-2) = +1.
	// [1] Defense dice 0 → Fair(+2)+0 = +2.
	// Outcome: +1 vs +2 = Failure, -1 shift.
	// [2-3] NPC counter-attack: -1,0 → NPC fails (no boost created).
	sm.actions.roller = dice.NewPlannedRoller([]int{-2, 0, -1, 0})

	testAction := makeAttackAction(sm.actions.player.ID, npc.Name)
	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.Failure, testAction.Outcome.Type)

	// No damage events.
	AssertNoEventIn[DamageResolutionEvent](t, events)

	// No boost created (defender didn't get SWS).
	AssertNoEventIn[AspectCreatedEvent](t, events)
}

// Fate Core SRD (Defend, SWS): "If the defender succeeds with style, the
// defender gets a boost." This happens when the attacker fails by 3+ shifts.
func TestResolveAction_Attack_Failure_DefenderSWS_GrantsTargetBoost(t *testing.T) {
	sm, npc := setupAttackSM(t, 0)

	// [0] Attack dice -4 → Good(+3)+(-4) = -1.
	// [1] Defense dice 0 → Fair(+2)+0 = +2.
	// Outcome: -1 vs +2 = Failure, -3 shifts (defender SWS).
	// [2-3] NPC counter-attack: -1,0 → NPC fails (no boost created).
	sm.actions.roller = dice.NewPlannedRoller([]int{-4, 0, -1, 0})

	testAction := makeAttackAction(sm.actions.player.ID, npc.Name)
	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.Failure, testAction.Outcome.Type)
	assert.True(t, testAction.Outcome.Shifts <= -3, "shifts should be -3 or worse")

	// Defender gets a boost.
	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost)
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	// Boost belongs to the defender (NPC), not the attacker.
	assert.Equal(t, npc.ID, sm.currentScene.SituationAspects[0].CreatedBy)
}
