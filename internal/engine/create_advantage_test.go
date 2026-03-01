package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCreateAdvantageSM creates a SceneManager with a mock LLM, a player with
// the given fate points, and an active scene. The roller is left as default;
// callers should replace sm.actions.roller with a PlannedRoller before calling
// resolveAction.
func setupCreateAdvantageSM(t *testing.T, fatePoints int) *SceneManager {
	t.Helper()

	mockClient := newTestLLMClient(`{"aspect_text": "Tactical Opening", "description": "A brief opening", "reasoning": "test"}`)
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := engine.GetSceneManager()

	player := character.NewCharacter("player-1", "Hero")
	player.Aspects.HighConcept = "Cunning Strategist"
	player.Aspects.Trouble = "Overconfident"
	player.FatePoints = fatePoints
	player.SetSkill("Notice", dice.Fair) // Fair (+2)

	engine.AddCharacter(player)

	testScene := scene.NewScene("test-scene", "Test Room", "A room for testing.")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	return sm
}

// --- resolveAction-level tests for Create Advantage ---

// Fate Core SRD (Create Advantage, Tie): "You get a boost instead of a full
// situation aspect." This test exercises the FULL resolveAction pipeline —
// dice roll, invoke loop, narrative generation, and effect application — to
// verify the boost propagates through every layer.
//
// Regression test for https://github.com/C-Ross/LlamaOfFate/issues/120
func TestResolveAction_CreateAdvantage_Tie_CreatesBoost(t *testing.T) {
	sm := setupCreateAdvantageSM(t, 0) // 0 FP → no invoke loop

	// PlannedRoller: dice total 0 → skill Fair(+2) + 0 = Fair(+2) vs difficulty Fair(+2) → Tie.
	sm.actions.roller = dice.NewPlannedRoller([]int{0})

	testAction := action.NewAction(
		"action-1",
		sm.actions.player.ID,
		action.CreateAdvantage,
		"Notice",
		"Look for an opening in the enemy's defenses",
	)
	testAction.Difficulty = dice.Fair // Fair (+2)

	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	// Should not be awaiting invoke (no FP, no free invokes available).
	assert.False(t, awaiting, "should not be awaiting invoke with 0 FP")

	// Outcome on the action should be Tie.
	require.NotNil(t, testAction.Outcome, "action outcome should be set")
	assert.Equal(t, dice.Tie, testAction.Outcome.Type, "outcome should be Tie")

	// An AspectCreatedEvent with IsBoost=true should appear in the events.
	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost, "CaA Tie should create a boost, not a full aspect")
	assert.Equal(t, 1, boostEvt.FreeInvokes, "boost should have 1 free invoke")
	assert.NotEmpty(t, boostEvt.AspectName, "boost should have a name")

	// The boost should be on the scene.
	require.Len(t, sm.currentScene.SituationAspects, 1,
		"exactly one situation aspect (the boost) should be on the scene")
	boost := sm.currentScene.SituationAspects[0]
	assert.True(t, boost.IsBoost, "scene aspect should be flagged as a boost")
	assert.Equal(t, 1, boost.FreeInvokes)
	assert.Equal(t, sm.actions.player.ID, boost.CreatedBy)
}

// Fate Core SRD (Create Advantage, Success): "You create a situation aspect
// with one free invoke." Full pipeline test.
func TestResolveAction_CreateAdvantage_Success_CreatesAspect(t *testing.T) {
	sm := setupCreateAdvantageSM(t, 0)

	// Dice total 1 → skill Fair(+2) + 1 = Good(+3) vs difficulty Fair(+2) → Success (1 shift).
	sm.actions.roller = dice.NewPlannedRoller([]int{1})

	testAction := action.NewAction(
		"action-1",
		sm.actions.player.ID,
		action.CreateAdvantage,
		"Notice",
		"Scout the perimeter",
	)
	testAction.Difficulty = dice.Fair

	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.Success, testAction.Outcome.Type)

	aspectEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.False(t, aspectEvt.IsBoost, "Success should create a full aspect, not a boost")
	assert.Equal(t, 1, aspectEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.False(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, 1, sm.currentScene.SituationAspects[0].FreeInvokes)
}

// Fate Core SRD (Create Advantage, Success with Style): "You create a
// situation aspect with TWO free invokes." Full pipeline test.
func TestResolveAction_CreateAdvantage_SWS_CreatesAspectWith2FreeInvokes(t *testing.T) {
	sm := setupCreateAdvantageSM(t, 0)

	// Dice total 3 → skill Fair(+2) + 3 = Superb(+5) vs difficulty Fair(+2) → SWS (3 shifts).
	sm.actions.roller = dice.NewPlannedRoller([]int{3})

	testAction := action.NewAction(
		"action-1",
		sm.actions.player.ID,
		action.CreateAdvantage,
		"Notice",
		"Thoroughly case the joint",
	)
	testAction.Difficulty = dice.Fair

	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.SuccessWithStyle, testAction.Outcome.Type)

	aspectEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.False(t, aspectEvt.IsBoost)
	assert.Equal(t, 2, aspectEvt.FreeInvokes, "SWS should grant 2 free invokes")

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.Equal(t, 2, sm.currentScene.SituationAspects[0].FreeInvokes)
}

// Fate Core SRD (Create Advantage, Failure): No aspect or boost is created.
// Full pipeline test.
func TestResolveAction_CreateAdvantage_Failure_NoAspect(t *testing.T) {
	sm := setupCreateAdvantageSM(t, 0)

	// Dice total -1 → skill Fair(+2) + (-1) = Average(+1) vs difficulty Fair(+2) → Failure (-1 shift).
	sm.actions.roller = dice.NewPlannedRoller([]int{-1})

	testAction := action.NewAction(
		"action-1",
		sm.actions.player.ID,
		action.CreateAdvantage,
		"Notice",
		"Search for weaknesses",
	)
	testAction.Difficulty = dice.Fair

	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.Failure, testAction.Outcome.Type)

	// No AspectCreatedEvent should be emitted.
	AssertNoEventIn[AspectCreatedEvent](t, events)

	// No situation aspects on the scene.
	assert.Empty(t, sm.currentScene.SituationAspects,
		"Failure should not create any situation aspects")
}

// Tie with invokes available: player has FP and aspects, so the invoke loop
// fires. The player skips the invoke, and the boost should still be created.
//
// Regression test for https://github.com/C-Ross/LlamaOfFate/issues/120
func TestResolveAction_CreateAdvantage_Tie_WithInvokeSkip_CreatesBoost(t *testing.T) {
	sm := setupCreateAdvantageSM(t, 3) // 3 FP → invoke loop will fire

	// Dice total 0 → Tie.
	sm.actions.roller = dice.NewPlannedRoller([]int{0})

	testAction := action.NewAction(
		"action-1",
		sm.actions.player.ID,
		action.CreateAdvantage,
		"Notice",
		"Look for an opening",
	)
	testAction.Difficulty = dice.Fair

	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	// With FP available, invoke loop should fire — engine is awaiting invoke.
	assert.True(t, awaiting, "should be awaiting invoke when player has FP")
	require.NotNil(t, sm.actions.pendingInvoke, "pending invoke state should be set")

	// Last event should be an InvokePromptEvent.
	lastEvent := events[len(events)-1]
	_, isPrompt := lastEvent.(InvokePromptEvent)
	assert.True(t, isPrompt, "last event should be InvokePromptEvent, got %T", lastEvent)

	// Player skips the invoke.
	result, err := sm.ProvideInvokeResponse(ctx, InvokeResponse{
		AspectIndex: uicontract.InvokeSkip,
	})
	require.NoError(t, err)
	assert.False(t, result.AwaitingInvoke, "should no longer be awaiting invoke after skip")

	// After skipping, the boost should have been created.
	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, result.Events)
	assert.True(t, boostEvt.IsBoost, "CaA Tie should create a boost after invoke skip")
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, sm.actions.player.ID, sm.currentScene.SituationAspects[0].CreatedBy)
}
