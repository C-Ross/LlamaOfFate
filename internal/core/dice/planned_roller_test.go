package dice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlannedRoller_RollWithModifier(t *testing.T) {
	roller := NewPlannedRoller([]int{2, -1, 0})

	// First roll: dice total 2, skill Mediocre(0) + modifier 3 → final = 0+2+3 = 5 (Superb)
	r1 := roller.RollWithModifier(Mediocre, 3)
	assert.Equal(t, Superb, r1.FinalValue)
	assert.Equal(t, 2, r1.Roll.Total)

	// Second roll: dice total -1, skill Fair(2) + modifier 0 → final = 2+(-1)+0 = 1 (Average)
	r2 := roller.RollWithModifier(Fair, 0)
	assert.Equal(t, Average, r2.FinalValue)

	// Third roll: dice total 0, skill Good(3) + modifier 0 → final = 3 (Good)
	r3 := roller.RollWithModifier(Good, 0)
	assert.Equal(t, Good, r3.FinalValue)

	assert.Equal(t, 0, roller.Remaining())
}

func TestPlannedRoller_Reroll(t *testing.T) {
	roller := NewPlannedRoller([]int{1, 3})

	original := roller.RollWithModifier(Good, 0) // dice=1 → final=4 (Great)
	assert.Equal(t, Great, original.FinalValue)

	rerolled := roller.Reroll(original) // dice=3 → final=6 (Fantastic)
	assert.Equal(t, Fantastic, rerolled.FinalValue)
	assert.True(t, rerolled.Rerolled)
	assert.NotNil(t, rerolled.OriginalRoll)
}

func TestPlannedRoller_PanicsWhenExhausted(t *testing.T) {
	roller := NewPlannedRoller([]int{0})
	roller.RollWithModifier(Mediocre, 0) // consume the one roll

	require.Panics(t, func() {
		roller.RollWithModifier(Mediocre, 0)
	})
}

func TestPlannedRoller_RollFromTotal_DiceFaces(t *testing.T) {
	// +3 total → three Plus dice, one Blank
	roll := rollFromTotal(3)
	assert.Equal(t, 3, roll.Total)
	plusCount := 0
	for _, d := range roll.Dice {
		if d == Plus {
			plusCount++
		}
	}
	assert.Equal(t, 3, plusCount)

	// -2 total → two Minus dice, two Blank
	roll = rollFromTotal(-2)
	assert.Equal(t, -2, roll.Total)
	minusCount := 0
	for _, d := range roll.Dice {
		if d == Minus {
			minusCount++
		}
	}
	assert.Equal(t, 2, minusCount)
}

// Verify PlannedRoller satisfies DiceRoller.
var _ DiceRoller = (*PlannedRoller)(nil)

// Verify Roller satisfies DiceRoller.
var _ DiceRoller = (*Roller)(nil)
