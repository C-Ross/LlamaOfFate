package action

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
)

func TestResolveAttackOutcome(t *testing.T) {
	tests := []struct {
		name           string
		outcome        *dice.Outcome
		expectedShifts int
		expectedSide   AttackSideEffect
	}{
		{
			name:           "success with 3 shifts",
			outcome:        &dice.Outcome{Type: dice.Success, Shifts: 3},
			expectedShifts: 3,
			expectedSide:   NoSideEffect,
		},
		{
			name:           "success with style returns boost option",
			outcome:        &dice.Outcome{Type: dice.SuccessWithStyle, Shifts: 4},
			expectedShifts: 4,
			expectedSide:   AttackerBoostOption,
		},
		{
			name:           "success with style clamps minimum to 1 shift",
			outcome:        &dice.Outcome{Type: dice.SuccessWithStyle, Shifts: 0},
			expectedShifts: 1,
			expectedSide:   AttackerBoostOption,
		},
		{
			name:           "success clamps minimum to 1 shift",
			outcome:        &dice.Outcome{Type: dice.Success, Shifts: 0},
			expectedShifts: 1,
			expectedSide:   NoSideEffect,
		},
		{
			name:           "tie gives attacker boost",
			outcome:        &dice.Outcome{Type: dice.Tie, Shifts: 0},
			expectedShifts: 0,
			expectedSide:   AttackerBoost,
		},
		{
			name:           "failure by less than 3 has no side effect",
			outcome:        &dice.Outcome{Type: dice.Failure, Shifts: -1},
			expectedShifts: 0,
			expectedSide:   NoSideEffect,
		},
		{
			name:           "failure by exactly 3 gives defender boost",
			outcome:        &dice.Outcome{Type: dice.Failure, Shifts: -3},
			expectedShifts: 0,
			expectedSide:   DefenderBoost,
		},
		{
			name:           "failure by more than 3 gives defender boost",
			outcome:        &dice.Outcome{Type: dice.Failure, Shifts: -5},
			expectedShifts: 0,
			expectedSide:   DefenderBoost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shifts, side := ResolveAttackOutcome(tt.outcome)
			assert.Equal(t, tt.expectedShifts, shifts, "shifts")
			assert.Equal(t, tt.expectedSide, side, "side effect")
		})
	}
}
