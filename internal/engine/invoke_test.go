package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

// buildInvokeTestSM creates a SceneManager with a player that has fate points,
// aspects, and a situation aspect with free invokes.
func buildInvokeTestSM(t *testing.T, fatePoints int) (*SceneManager, *MockUI) {
	t.Helper()
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	sm.actions.roller = dice.NewSeededRoller(42)

	player := character.NewCharacter("player-1", "Test Hero")
	player.Aspects.HighConcept = "Mighty Warrior"
	player.Aspects.Trouble = "Hot-Headed"
	player.FatePoints = fatePoints
	player.SetSkill("Fight", dice.Good)
	engine.AddCharacter(player)

	testScene := scene.NewScene("test-scene", "Test Arena", "A test arena")
	testScene.AddCharacter(player.ID)
	testScene.AddSituationAspect(scene.NewSituationAspect("sit-1", "Burning Rafters", "gm", 2))

	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	mockUI := &MockUI{}

	return sm, mockUI
}

// --- buildInvokePrompt tests ---

func TestBuildInvokePrompt_NoAspectsAvailable(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 0)
	// Zero FP and no free invokes on character aspects
	sm.currentScene.SituationAspects = nil // remove situation aspects

	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	prompt, _ := sm.actions.buildInvokePrompt(result, dice.Superb, false, nil)

	assert.Nil(t, prompt, "no prompt when no invokes are possible")
}

func TestBuildInvokePrompt_HasFatePoints(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	// Roll that needs improvement
	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	prompt, available := sm.actions.buildInvokePrompt(result, dice.Superb, false, nil)

	require.NotNil(t, prompt, "should prompt when FP available")
	assert.Equal(t, 3, prompt.FatePoints)
	assert.NotEmpty(t, available)
	assert.Greater(t, prompt.ShiftsNeeded, 0)
}

func TestBuildInvokePrompt_FreeInvokeAvailable_ZeroFP(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 0)

	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	prompt, available := sm.actions.buildInvokePrompt(result, dice.Superb, false, nil)

	require.NotNil(t, prompt, "should prompt when free invokes available")
	assert.Equal(t, 0, prompt.FatePoints)

	// At least one aspect should have free invokes
	hasFree := false
	for _, a := range available {
		if a.FreeInvokes > 0 {
			hasFree = true
			break
		}
	}
	assert.True(t, hasFree, "at least one aspect should have free invokes")
}

func TestBuildInvokePrompt_SkipsWhenSuccessWithStyle(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	// Use a very high modifier to guarantee success with style
	result := sm.actions.roller.RollWithModifier(dice.Mediocre, 10)
	prompt, _ := sm.actions.buildInvokePrompt(result, dice.Mediocre, false, nil)

	assert.Nil(t, prompt, "should not prompt when already success with style")
}

func TestBuildInvokePrompt_Defense_SkipsWhenDefenseWins(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	// Defense result is higher than attack difficulty
	result := sm.actions.roller.RollWithModifier(dice.Mediocre, 10)
	prompt, _ := sm.actions.buildInvokePrompt(result, dice.Good, true, nil)

	assert.Nil(t, prompt, "should not prompt for defense when already winning")
}

func TestBuildInvokePrompt_Defense_PromptsWhenLosing(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	// Defense result is lower than attack
	result := sm.actions.roller.RollWithModifier(dice.Mediocre, 0)
	prompt, _ := sm.actions.buildInvokePrompt(result, dice.Superb, true, nil)

	require.NotNil(t, prompt, "should prompt for defense when losing")
	assert.Greater(t, prompt.ShiftsNeeded, 0)
}

// --- beginInvokeLoop tests ---

func TestBeginInvokeLoop_NoInvokePossible_CallsFinishImmediately(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 0)
	sm.currentScene.SituationAspects = nil // no free invokes available

	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	parsedAction := action.NewAction("act-1", "player-1", action.Overcome, "Fight", "test")

	finishCalled := false
	finish := func(ctx context.Context, r *dice.CheckResult, events []GameEvent) []GameEvent {
		finishCalled = true
		return append(events, SystemMessageEvent{Message: "finished"})
	}

	events, awaiting := sm.actions.beginInvokeLoop(context.Background(), result, dice.Superb, parsedAction, false, nil, finish)

	assert.True(t, finishCalled, "finish should be called immediately")
	assert.False(t, awaiting, "should not be awaiting invoke")
	require.Len(t, events, 1)
	assert.Equal(t, "finished", events[0].(SystemMessageEvent).Message)
	assert.Nil(t, sm.actions.pendingInvoke, "no pending invoke state")
}

func TestBeginInvokeLoop_InvokePossible_SetsPendingState(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	parsedAction := action.NewAction("act-1", "player-1", action.Overcome, "Fight", "test")

	finish := func(ctx context.Context, r *dice.CheckResult, events []GameEvent) []GameEvent {
		t.Fatal("finish should NOT be called when invoke is possible")
		return events
	}

	events, awaiting := sm.actions.beginInvokeLoop(context.Background(), result, dice.Superb, parsedAction, false, nil, finish)

	assert.True(t, awaiting, "should be awaiting invoke")
	require.NotNil(t, sm.actions.pendingInvoke, "pending invoke state should be set")
	assert.True(t, sm.HasPendingInvoke())

	// Last event should be InvokePromptEvent
	require.NotEmpty(t, events)
	lastEvent := events[len(events)-1]
	promptEvent, ok := lastEvent.(InvokePromptEvent)
	require.True(t, ok, "last event should be InvokePromptEvent, got %T", lastEvent)
	assert.NotEmpty(t, promptEvent.Available)
	assert.Equal(t, 3, promptEvent.FatePoints)
}

// --- ProvideInvokeResponse tests ---

func TestProvideInvokeResponse_Skip(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	parsedAction := action.NewAction("act-1", "player-1", action.Overcome, "Fight", "test")
	initialFP := sm.player.FatePoints

	finishCalled := false
	finish := func(ctx context.Context, r *dice.CheckResult, events []GameEvent) []GameEvent {
		finishCalled = true
		return append(events, SystemMessageEvent{Message: "done"})
	}

	sm.actions.beginInvokeLoop(context.Background(), result, dice.Superb, parsedAction, false, nil, finish)
	require.True(t, sm.HasPendingInvoke())

	// Skip
	resp, err := sm.ProvideInvokeResponse(context.Background(), InvokeResponse{
		AspectIndex: uicontract.InvokeSkip,
	})
	require.NoError(t, err)

	assert.True(t, finishCalled, "finish should be called on skip")
	assert.False(t, resp.AwaitingInvoke)
	assert.Nil(t, sm.actions.pendingInvoke, "pending state cleared")
	assert.Equal(t, initialFP, sm.player.FatePoints, "no FP should be spent on skip")
}

func TestProvideInvokeResponse_PlusTwoBonus(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	initialFinal := result.FinalValue
	parsedAction := action.NewAction("act-1", "player-1", action.Overcome, "Fight", "test")

	finishCalled := false
	var finalResult *dice.CheckResult
	finish := func(ctx context.Context, r *dice.CheckResult, events []GameEvent) []GameEvent {
		finishCalled = true
		finalResult = r
		return events
	}

	sm.actions.beginInvokeLoop(context.Background(), result, dice.Superb, parsedAction, false, nil, finish)
	require.True(t, sm.HasPendingInvoke())

	// Find a character aspect index (not a free invoke)
	var charIdx int
	for i, a := range sm.actions.pendingInvoke.available {
		if a.Source == "character" && a.FreeInvokes == 0 {
			charIdx = i
			break
		}
	}

	resp, err := sm.ProvideInvokeResponse(context.Background(), InvokeResponse{
		AspectIndex: charIdx,
		IsReroll:    false, // +2
	})
	require.NoError(t, err)

	// FP should be spent
	assert.Equal(t, 2, sm.player.FatePoints, "should spend 1 FP")

	// Events should include the +2 message
	invokeEvts := SliceOfType[InvokeEvent](resp.Events)
	hasBonus := false
	for _, invokeEv := range invokeEvts {
		if !invokeEv.IsReroll && invokeEv.NewTotal != "" {
			hasBonus = true
		}
	}
	assert.True(t, hasBonus, "should see +2 invoke event")

	// If finish was called, check the result was increased
	if finishCalled {
		assert.Equal(t, initialFinal+2, finalResult.FinalValue, "result should be +2")
	}
}

func TestProvideInvokeResponse_Reroll(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	parsedAction := action.NewAction("act-1", "player-1", action.Overcome, "Fight", "test")

	finish := func(ctx context.Context, r *dice.CheckResult, events []GameEvent) []GameEvent {
		return events
	}

	sm.actions.beginInvokeLoop(context.Background(), result, dice.Superb, parsedAction, false, nil, finish)
	require.True(t, sm.HasPendingInvoke())

	// Find a character aspect
	var charIdx int
	for i, a := range sm.actions.pendingInvoke.available {
		if a.Source == "character" && a.FreeInvokes == 0 {
			charIdx = i
			break
		}
	}

	resp, err := sm.ProvideInvokeResponse(context.Background(), InvokeResponse{
		AspectIndex: charIdx,
		IsReroll:    true,
	})
	require.NoError(t, err)

	// FP should be spent
	assert.Equal(t, 2, sm.player.FatePoints, "should spend 1 FP")

	// Events should include reroll message
	invokeEvts := SliceOfType[InvokeEvent](resp.Events)
	hasReroll := false
	for _, invokeEv := range invokeEvts {
		if invokeEv.IsReroll {
			hasReroll = true
		}
	}
	assert.True(t, hasReroll, "should see reroll invoke event")
}

func TestProvideInvokeResponse_FreeInvoke(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 0) // Zero FP — must use free invoke

	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	parsedAction := action.NewAction("act-1", "player-1", action.Overcome, "Fight", "test")

	finish := func(ctx context.Context, r *dice.CheckResult, events []GameEvent) []GameEvent {
		return events
	}

	sm.actions.beginInvokeLoop(context.Background(), result, dice.Superb, parsedAction, false, nil, finish)
	require.True(t, sm.HasPendingInvoke())

	// Find the free invoke aspect
	var freeIdx int
	found := false
	for i, a := range sm.actions.pendingInvoke.available {
		if a.FreeInvokes > 0 {
			freeIdx = i
			found = true
			break
		}
	}
	require.True(t, found, "should have a free invoke available")

	initialFreeInvokes := sm.currentScene.SituationAspects[0].FreeInvokes

	resp, err := sm.ProvideInvokeResponse(context.Background(), InvokeResponse{
		AspectIndex: freeIdx,
		IsReroll:    false,
	})
	require.NoError(t, err)

	// FP should NOT be spent
	assert.Equal(t, 0, sm.player.FatePoints, "should not spend FP for free invoke")

	// Free invoke count should be decremented
	assert.Equal(t, initialFreeInvokes-1, sm.currentScene.SituationAspects[0].FreeInvokes)

	// Events should include free invoke message
	invokeEvts := SliceOfType[InvokeEvent](resp.Events)
	hasFreeMsg := false
	for _, invokeEv := range invokeEvts {
		if invokeEv.IsFree {
			hasFreeMsg = true
		}
	}
	assert.True(t, hasFreeMsg, "should see free invoke event")
}

func TestProvideInvokeResponse_InvalidIndex(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	result := sm.actions.roller.RollWithModifier(dice.Mediocre, int(dice.Good))
	parsedAction := action.NewAction("act-1", "player-1", action.Overcome, "Fight", "test")

	finish := func(ctx context.Context, r *dice.CheckResult, events []GameEvent) []GameEvent {
		return events
	}

	sm.actions.beginInvokeLoop(context.Background(), result, dice.Superb, parsedAction, false, nil, finish)
	require.True(t, sm.HasPendingInvoke())

	_, err := sm.ProvideInvokeResponse(context.Background(), InvokeResponse{
		AspectIndex: 999,
	})
	assert.Error(t, err, "should error on invalid aspect index")
	assert.Contains(t, err.Error(), "out of range")
}

func TestProvideInvokeResponse_NoPendingInvoke(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	_, err := sm.ProvideInvokeResponse(context.Background(), InvokeResponse{
		AspectIndex: uicontract.InvokeSkip,
	})
	assert.Error(t, err, "should error when no pending invoke")
	assert.Contains(t, err.Error(), "no pending invoke")
}

func TestHandleInput_RejectsDuringPendingInvoke(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 3)

	// Manually set a pending invoke
	sm.actions.pendingInvoke = &invokeState{}

	_, err := sm.HandleInput(context.Background(), "hello")
	assert.Error(t, err, "should reject HandleInput during pending invoke")
	assert.Contains(t, err.Error(), "awaiting invoke response")
}

// --- HasPendingInvoke tests ---

func TestHasPendingInvoke_False(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 0)
	assert.False(t, sm.HasPendingInvoke())
}

func TestHasPendingInvoke_True(t *testing.T) {
	sm, _ := buildInvokeTestSM(t, 0)
	sm.actions.pendingInvoke = &invokeState{}
	assert.True(t, sm.HasPendingInvoke())
}

// --- InvokeSkip constant test ---

func TestInvokeSkipConstant(t *testing.T) {
	assert.Equal(t, -1, uicontract.InvokeSkip)
}
