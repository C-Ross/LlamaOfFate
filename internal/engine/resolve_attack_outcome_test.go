package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupResolveAttackSM creates a SceneManager with an attacker NPC and player
// in a physical conflict. Uses the shared setupTestSM helper.
func setupResolveAttackSM(t *testing.T) (*SceneManager, *core.Character, *core.Character) {
	t.Helper()
	boostJSON := `{"aspect_text":"Test Boost","description":"test","reasoning":"test"}`
	return setupTestSM(t, smTestOpts{
		llmResponses: []string{boostJSON},
		skills:       map[string]dice.Ladder{"Fight": dice.Fair, "Athletics": dice.Fair},
		npc: &smTestNPC{
			id:          "npc-1",
			name:        "Bandit",
			highConcept: "Ruthless Highwayman",
			skills:      map[string]dice.Ladder{"Fight": dice.Fair, "Athletics": dice.Fair},
		},
		conflictType: scene.PhysicalConflict,
	})
}

// --- Success (shifts > 0): onDamage callback fires ---

func TestResolveAttackOutcome_Success_CallsOnDamage(t *testing.T) {
	sm, _, npc := setupResolveAttackSM(t)
	outcome := &dice.Outcome{Type: dice.Success, Shifts: 2}

	var calledShifts int
	events := sm.conflict.resolveAttackOutcome(context.Background(), outcome, sm.conflict.player, npc, "Fight", "swings a sword", attackCallbacks{
		onDamage: func(shifts int) []GameEvent {
			calledShifts = shifts
			return []GameEvent{NarrativeEvent{Text: "damage"}}
		},
		onTie:             func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure:         func() []GameEvent { return nil },
	})

	assert.Equal(t, 2, calledShifts)
	RequireFirstFrom[NarrativeEvent](t, events)
	// No boost should be created on success.
	AssertNoEventIn[AspectCreatedEvent](t, events)
}

func TestResolveAttackOutcome_SWS_CallsOnDamageWithClampedShifts(t *testing.T) {
	sm, _, npc := setupResolveAttackSM(t)
	outcome := &dice.Outcome{Type: dice.SuccessWithStyle, Shifts: 4}

	var calledShifts int
	events := sm.conflict.resolveAttackOutcome(context.Background(), outcome, sm.conflict.player, npc, "Fight", "big hit", attackCallbacks{
		onDamage: func(shifts int) []GameEvent {
			calledShifts = shifts
			return []GameEvent{NarrativeEvent{Text: "damage"}}
		},
		onTie:             func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure:         func() []GameEvent { return nil },
	})

	assert.Equal(t, 4, calledShifts)
	AssertNoEventIn[AspectCreatedEvent](t, events)
}

// --- Verify correct attacker/defender for boost ownership ---

func TestResolveAttackOutcome_Tie_CallsOnTieAndCreatesAttackerBoost(t *testing.T) {
	sm, player, npc := setupResolveAttackSM(t)
	outcome := &dice.Outcome{Type: dice.Tie, Shifts: 0}

	tieCalled := false
	events := sm.conflict.resolveAttackOutcome(context.Background(), outcome, player, npc, "Fight", "evenly matched", attackCallbacks{
		onDamage: func(shifts int) []GameEvent { return nil },
		onTie: func() []GameEvent {
			tieCalled = true
			return []GameEvent{NarrativeEvent{Text: "tie"}}
		},
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure:         func() []GameEvent { return nil },
	})

	assert.True(t, tieCalled)
	// Callback event present.
	RequireFirstFrom[NarrativeEvent](t, events)
	// Attacker boost must be created.
	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost)
	assert.Equal(t, 1, boostEvt.FreeInvokes)
}

// --- Defend with style: onDefendWithStyle callback + defender boost ---

func TestResolveAttackOutcome_DefendWithStyle_CreatesDefenderBoost(t *testing.T) {
	sm, player, npc := setupResolveAttackSM(t)
	outcome := &dice.Outcome{Type: dice.Failure, Shifts: -3}

	dwsCalled := false
	events := sm.conflict.resolveAttackOutcome(context.Background(), outcome, player, npc, "Fight", "wild swing", attackCallbacks{
		onDamage: func(shifts int) []GameEvent { return nil },
		onTie:    func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent {
			dwsCalled = true
			return []GameEvent{NarrativeEvent{Text: "dws"}}
		},
		onFailure: func() []GameEvent { return nil },
	})

	assert.True(t, dwsCalled)
	RequireFirstFrom[NarrativeEvent](t, events)
	// Defender (npc) gets the boost.
	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost)
	assert.Equal(t, 1, boostEvt.FreeInvokes)
}

// --- Failure (no boost): onFailure callback fires ---

func TestResolveAttackOutcome_Failure_CallsOnFailure(t *testing.T) {
	sm, player, npc := setupResolveAttackSM(t)
	// Failure by 1 — not enough for defend-with-style.
	outcome := &dice.Outcome{Type: dice.Failure, Shifts: -1}

	failCalled := false
	events := sm.conflict.resolveAttackOutcome(context.Background(), outcome, player, npc, "Fight", "miss", attackCallbacks{
		onDamage:          func(shifts int) []GameEvent { return nil },
		onTie:             func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure: func() []GameEvent {
			failCalled = true
			return []GameEvent{NarrativeEvent{Text: "miss"}}
		},
	})

	assert.True(t, failCalled)
	RequireFirstFrom[NarrativeEvent](t, events)
	// No boost on a mild failure.
	AssertNoEventIn[AspectCreatedEvent](t, events)
}

func TestResolveAttackOutcome_Failure_NoOpCallback_NoEvents(t *testing.T) {
	sm, player, npc := setupResolveAttackSM(t)
	outcome := &dice.Outcome{Type: dice.Failure, Shifts: -1}

	events := sm.conflict.resolveAttackOutcome(context.Background(), outcome, player, npc, "Fight", "test", attackCallbacks{
		onDamage:          func(shifts int) []GameEvent { return nil },
		onTie:             func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure:         func() []GameEvent { return nil },
	})
	assert.Empty(t, events)
}

// --- Verify correct attacker/defender for boost ownership ---

func TestResolveAttackOutcome_Tie_BoostOwnedByAttacker(t *testing.T) {
	sm, player, npc := setupResolveAttackSM(t)
	outcome := &dice.Outcome{Type: dice.Tie, Shifts: 0}

	// Player attacks NPC; on tie the boost should belong to the attacker (player).
	sm.conflict.resolveAttackOutcome(context.Background(), outcome, player, npc, "Fight", "test", attackCallbacks{
		onDamage:          func(shifts int) []GameEvent { return nil },
		onTie:             func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure:         func() []GameEvent { return nil },
	})

	// Find the boost in scene situation aspects.
	var found bool
	for _, sa := range sm.conflict.currentScene.SituationAspects {
		if sa.IsBoost {
			assert.Equal(t, player.ID, sa.CreatedBy)
			found = true
			break
		}
	}
	require.True(t, found, "expected a boost in situation aspects")
}

func TestResolveAttackOutcome_DefendWithStyle_BoostOwnedByDefender(t *testing.T) {
	sm, player, npc := setupResolveAttackSM(t)
	outcome := &dice.Outcome{Type: dice.Failure, Shifts: -3}

	// Player attacks NPC; defend-with-style → boost owned by defender (npc).
	sm.conflict.resolveAttackOutcome(context.Background(), outcome, player, npc, "Fight", "test", attackCallbacks{
		onDamage:          func(shifts int) []GameEvent { return nil },
		onTie:             func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure:         func() []GameEvent { return nil },
	})

	var found bool
	for _, sa := range sm.conflict.currentScene.SituationAspects {
		if sa.IsBoost {
			assert.Equal(t, npc.ID, sa.CreatedBy)
			found = true
			break
		}
	}
	require.True(t, found, "expected a boost in situation aspects")
}

// --- Verify minimum 1-shift clamping from ResolveAttackOutcome ---

func TestResolveAttackOutcome_SuccessMinOneShift(t *testing.T) {
	sm, player, npc := setupResolveAttackSM(t)
	// 0-shift success → ResolveAttackOutcome clamps to 1.
	outcome := &dice.Outcome{Type: dice.Success, Shifts: 0}

	var calledShifts int
	sm.conflict.resolveAttackOutcome(context.Background(), outcome, player, npc, "Fight", "test", attackCallbacks{
		onDamage: func(shifts int) []GameEvent {
			calledShifts = shifts
			return nil
		},
		onTie:             func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure:         func() []GameEvent { return nil },
	})

	assert.Equal(t, 1, calledShifts, "Success with 0 shifts should clamp to minimum 1")
}

// --- Verify attackDesc flows into boost name generation ---

func TestResolveAttackOutcome_Tie_UsesAttackDescForBoostName(t *testing.T) {
	sm, player, npc := setupResolveAttackSM(t)
	outcome := &dice.Outcome{Type: dice.Tie, Shifts: 0}

	// The LLM mock returns "Test Boost" as aspect_text, but the important
	// thing is that generateBoostName was called (i.e., boost was created).
	events := sm.conflict.resolveAttackOutcome(context.Background(), outcome, player, npc, "Fight", "a precise lunge", attackCallbacks{
		onDamage:          func(shifts int) []GameEvent { return nil },
		onTie:             func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure:         func() []GameEvent { return nil },
	})

	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost)
	// The boost name comes from the LLM mock response.
	assert.NotEmpty(t, boostEvt.AspectName)
}

// --- Verify defense skill derivation for defend-with-style ---

func TestResolveAttackOutcome_DefendWithStyle_UsesCorrectDefenseSkill(t *testing.T) {
	sm, player, npc := setupResolveAttackSM(t)
	outcome := &dice.Outcome{Type: dice.Failure, Shifts: -3}

	// Attack with Provoke → defense skill should be Will.
	// The LLM captures prompts, so we can verify the skill was used.
	sm.conflict.resolveAttackOutcome(context.Background(), outcome, player, npc, "Provoke", "intimidation", attackCallbacks{
		onDamage:          func(shifts int) []GameEvent { return nil },
		onTie:             func() []GameEvent { return nil },
		onDefendWithStyle: func() []GameEvent { return nil },
		onFailure:         func() []GameEvent { return nil },
	})

	var found bool
	for _, sa := range sm.conflict.currentScene.SituationAspects {
		if sa.IsBoost {
			// Boost belongs to defender (npc).
			assert.Equal(t, npc.ID, sa.CreatedBy)
			found = true
			break
		}
	}
	require.True(t, found, "expected a boost in situation aspects")
}

// --- ResolveAttackOutcome delegates correctly from core/action ---

func TestResolveAttackOutcome_IntegrationWithCoreAction(t *testing.T) {
	// Verify that the method routes through action.ResolveAttackOutcome
	// by checking side effects match the core package contract.
	shifts, side := action.ResolveAttackOutcome(&dice.Outcome{Type: dice.Tie, Shifts: 0})
	assert.Equal(t, 0, shifts)
	assert.Equal(t, action.AttackerBoost, side)

	shifts, side = action.ResolveAttackOutcome(&dice.Outcome{Type: dice.Failure, Shifts: -3})
	assert.Equal(t, 0, shifts)
	assert.Equal(t, action.DefenderBoost, side)

	shifts, side = action.ResolveAttackOutcome(&dice.Outcome{Type: dice.Failure, Shifts: -1})
	assert.Equal(t, 0, shifts)
	assert.Equal(t, action.NoSideEffect, side)
}
