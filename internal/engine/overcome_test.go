package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupOvercomeSM creates a SceneManager with a mock LLM, a player with
// Athletics at Fair (+2), and an active scene. Passive opposition only.
func setupOvercomeSM(t *testing.T, fatePoints int) *SceneManager {
	t.Helper()
	sm, _, _ := setupTestSM(t, smTestOpts{
		llmResponses: []string{"You push through!"},
		fatePoints:   fatePoints,
		highConcept:  "Nimble Acrobat",
		trouble:      "Fear of Heights",
		skills:       map[string]dice.Ladder{"Athletics": dice.Fair},
	})
	return sm
}

func makeOvercomeAction(playerID string) *action.Action {
	a := action.NewAction(
		"action-1",
		playerID,
		action.Overcome,
		"Athletics",
		"Vault over the obstacle",
	)
	a.Difficulty = dice.Fair // Fair (+2)
	return a
}

// --- resolveAction-level tests for Overcome ---

// Fate Core SRD (Overcome, Failure): "You either simply fail, gain a success
// at a serious cost, or suffer some other major negative outcome."
// No mechanical effect is applied by the engine (cost is narrative).
func TestResolveAction_Overcome_Failure_NoEffect(t *testing.T) {
	sm := setupOvercomeSM(t, 0)

	// Dice total -1 → Fair(+2)+(-1) = Average(+1) vs Fair(+2) → Failure, -1 shift.
	sm.actions.roller = dice.NewPlannedRoller([]int{-1})

	testAction := makeOvercomeAction(sm.actions.player.ID)
	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.Failure, testAction.Outcome.Type)

	// No aspects or boosts created.
	AssertNoEventIn[AspectCreatedEvent](t, events)
	assert.Empty(t, sm.currentScene.SituationAspects)
}

// Fate Core SRD (Overcome, Tie): "You attain your goal or get what you're
// after, but at a minor cost." No mechanical effect (cost is narrative).
func TestResolveAction_Overcome_Tie_NoMechanicalEffect(t *testing.T) {
	sm := setupOvercomeSM(t, 0)

	// Dice total 0 → Fair(+2)+0 = Fair(+2) vs Fair(+2) → Tie.
	sm.actions.roller = dice.NewPlannedRoller([]int{0})

	testAction := makeOvercomeAction(sm.actions.player.ID)
	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.Tie, testAction.Outcome.Type)

	// Overcome Tie has no mechanical effect — no boost, no aspect.
	AssertNoEventIn[AspectCreatedEvent](t, events)
	assert.Empty(t, sm.currentScene.SituationAspects)
}

// Fate Core SRD (Overcome, Success): "You accomplish your goal."
// No additional mechanical effect beyond success.
func TestResolveAction_Overcome_Success_NoBoost(t *testing.T) {
	sm := setupOvercomeSM(t, 0)

	// Dice total 1 → Fair(+2)+1 = Good(+3) vs Fair(+2) → Success, 1 shift.
	sm.actions.roller = dice.NewPlannedRoller([]int{1})

	testAction := makeOvercomeAction(sm.actions.player.ID)
	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.Success, testAction.Outcome.Type)

	// Regular success on Overcome has no bonus effect.
	AssertNoEventIn[AspectCreatedEvent](t, events)
	assert.Empty(t, sm.currentScene.SituationAspects)
}

// Fate Core SRD (Overcome, SWS): "You gain a boost in addition to achieving
// your goal." Full pipeline test.
func TestResolveAction_Overcome_SWS_CreatesBoost(t *testing.T) {
	sm := setupOvercomeSM(t, 0)

	// Dice total 3 → Fair(+2)+3 = Superb(+5) vs Fair(+2) → SWS, 3 shifts.
	sm.actions.roller = dice.NewPlannedRoller([]int{3})

	testAction := makeOvercomeAction(sm.actions.player.ID)
	ctx := context.Background()
	events, awaiting := sm.actions.resolveAction(ctx, testAction)

	assert.False(t, awaiting)
	require.NotNil(t, testAction.Outcome)
	assert.Equal(t, dice.SuccessWithStyle, testAction.Outcome.Type)

	// SWS on Overcome grants a boost.
	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost, "Overcome SWS should create a boost")
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, sm.actions.player.ID, sm.currentScene.SituationAspects[0].CreatedBy)
}
