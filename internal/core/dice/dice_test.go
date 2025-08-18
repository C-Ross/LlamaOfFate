package dice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLadder_String(t *testing.T) {
	tests := []struct {
		ladder   Ladder
		expected string
	}{
		{Terrible, "Terrible (-2)"},
		{Mediocre, "Mediocre (+0)"},
		{Great, "Great (+4)"},
		{Legendary, "Legendary (+8)"},
		{Ladder(10), "Legendary+ (+10)"},
		{Ladder(-3), "Terrible- (-3)"},
	}

	for _, test := range tests {
		result := test.ladder.String()
		assert.Equal(t, test.expected, result, "Ladder(%d).String()", test.ladder)
	}
}

func TestLadder_IsValid(t *testing.T) {
	tests := []struct {
		ladder Ladder
		valid  bool
	}{
		{Terrible, true},
		{Legendary, true},
		{Ladder(-3), true},
		{Ladder(10), true},
		{Ladder(-4), false},
		{Ladder(11), false},
	}

	for _, test := range tests {
		result := test.ladder.IsValid()
		assert.Equal(t, test.valid, result, "Ladder(%d).IsValid()", test.ladder)
	}
}

func TestLadder_Add(t *testing.T) {
	tests := []struct {
		base     Ladder
		value    int
		expected Ladder
	}{
		{Mediocre, 2, Fair},
		{Good, -1, Fair},
		{Terrible, 5, Good},
	}

	for _, test := range tests {
		result := test.base.Add(test.value)
		assert.Equal(t, test.expected, result, "Ladder(%d).Add(%d)", test.base, test.value)
	}
}

func TestLadder_Compare(t *testing.T) {
	tests := []struct {
		a        Ladder
		b        Ladder
		expected int
	}{
		{Good, Fair, 1},
		{Fair, Good, -1},
		{Great, Great, 0},
		{Superb, Mediocre, 5},
	}

	for _, test := range tests {
		result := test.a.Compare(test.b)
		assert.Equal(t, test.expected, result, "Ladder(%d).Compare(%d)", test.a, test.b)
	}
}

func TestParseLadder(t *testing.T) {
	tests := []struct {
		input    string
		expected Ladder
		hasError bool
	}{
		{"Great", Great, false},
		{"Mediocre", Mediocre, false},
		{"Invalid", Mediocre, true},
		{"", Mediocre, true},
	}

	for _, test := range tests {
		result, err := ParseLadder(test.input)
		if test.hasError {
			assert.Error(t, err, "ParseLadder(%s) expected error", test.input)
		} else {
			assert.NoError(t, err, "ParseLadder(%s) unexpected error", test.input)
			assert.Equal(t, test.expected, result, "ParseLadder(%s)", test.input)
		}
	}
}

func TestFateDie_String(t *testing.T) {
	tests := []struct {
		die      FateDie
		expected string
	}{
		{Minus, "[-]"},
		{Blank, "[ ]"},
		{Plus, "[+]"},
		{FateDie(99), "[?]"},
	}

	for _, test := range tests {
		result := test.die.String()
		assert.Equal(t, test.expected, result, "FateDie(%d).String()", test.die)
	}
}

func TestRoller_Roll4dF(t *testing.T) {
	// Test with seeded roller for predictable results
	roller := NewSeededRoller(12345)

	roll := roller.Roll4dF()

	// Verify roll structure
	require.NotNil(t, roll)
	assert.Len(t, roll.Dice, 4)

	// Verify dice values are valid
	for i, die := range roll.Dice {
		assert.GreaterOrEqual(t, int(die), -1, "dice[%d] should be >= -1", i)
		assert.LessOrEqual(t, int(die), 1, "dice[%d] should be <= 1", i)
	}

	// Verify total is sum of dice
	expectedTotal := 0
	for _, die := range roll.Dice {
		expectedTotal += int(die)
	}
	assert.Equal(t, expectedTotal, roll.Total)

	// Verify timestamp is recent
	assert.WithinDuration(t, time.Now(), roll.RolledAt, time.Second)
}

func TestRoller_RollWithModifier(t *testing.T) {
	roller := NewSeededRoller(54321)

	skill := Good
	modifier := 2

	result := roller.RollWithModifier(skill, modifier)

	require.NotNil(t, result)
	assert.Equal(t, skill, result.BaseSkill)
	assert.Equal(t, modifier, result.Modifier)

	expectedFinal := Ladder(int(skill) + result.Roll.Total + modifier)
	assert.Equal(t, expectedFinal, result.FinalValue)
}

func TestCheckResult_CompareAgainst(t *testing.T) {
	// Create a mock check result
	roll := &Roll{Total: 1} // +1 from dice
	checkResult := &CheckResult{
		Roll:       roll,
		BaseSkill:  Good,  // +3
		Modifier:   1,     // +1
		FinalValue: Great, // Total: +5
	}

	tests := []struct {
		difficulty     Ladder
		expectedType   OutcomeType
		expectedShifts int
	}{
		{Great, Tie, 0},                 // 5 vs 5 = tie
		{Good, Success, 1},              // 5 vs 4 = 1 shift success
		{Fair, Success, 2},              // 5 vs 3 = 2 shift success
		{Mediocre, SuccessWithStyle, 4}, // 5 vs 1 = 4 shift success with style
		{Superb, Failure, -1},           // 5 vs 6 = failure
	}

	for _, test := range tests {
		outcome := checkResult.CompareAgainst(test.difficulty)

		assert.Equal(t, test.expectedType, outcome.Type,
			"CompareAgainst(%d) Type", test.difficulty)
		assert.Equal(t, test.expectedShifts, outcome.Shifts,
			"CompareAgainst(%d) Shifts", test.difficulty)
		assert.Equal(t, test.difficulty, outcome.Difficulty,
			"CompareAgainst(%d) Difficulty", test.difficulty)
	}
}

func TestOutcomeType_String(t *testing.T) {
	tests := []struct {
		outcome  OutcomeType
		expected string
	}{
		{Failure, "Failure"},
		{Tie, "Tie"},
		{Success, "Success"},
		{SuccessWithStyle, "Success with Style"},
		{OutcomeType(99), "Unknown"},
	}

	for _, test := range tests {
		result := test.outcome.String()
		assert.Equal(t, test.expected, result, "OutcomeType(%d).String()", test.outcome)
	}
}

func TestNewRoller(t *testing.T) {
	roller := NewRoller()
	require.NotNil(t, roller)
	require.NotNil(t, roller.rng)
}

func TestNewSeededRoller(t *testing.T) {
	seed := int64(98765)
	roller := NewSeededRoller(seed)

	require.NotNil(t, roller)
	require.NotNil(t, roller.rng)

	// Test that seeded roller produces consistent results
	roll1 := roller.Roll4dF()

	roller2 := NewSeededRoller(seed)
	roll2 := roller2.Roll4dF()

	assert.Equal(t, roll1.Total, roll2.Total,
		"Seeded rollers with same seed should produce same results")
}
