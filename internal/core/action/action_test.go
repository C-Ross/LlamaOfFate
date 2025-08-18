package action

import (
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
)

func TestActionType_String(t *testing.T) {
	tests := []struct {
		actionType ActionType
		expected   string
	}{
		{Overcome, "Overcome"},
		{CreateAdvantage, "Create an Advantage"},
		{Attack, "Attack"},
		{Defend, "Defend"},
		{ActionType(99), "Unknown"},
	}

	for _, test := range tests {
		result := test.actionType.String()
		assert.Equal(t, test.expected, result, "ActionType(%d).String()", test.actionType)
	}
}

func TestNewAction(t *testing.T) {
	id := "test-action"
	characterID := "test-character"
	actionType := Overcome
	skill := "Athletics"
	description := "Jump over the chasm"

	action := NewAction(id, characterID, actionType, skill, description)

	assert.Equal(t, id, action.ID)
	assert.Equal(t, characterID, action.CharacterID)
	assert.Equal(t, actionType, action.Type)
	assert.Equal(t, skill, action.Skill)
	assert.Equal(t, description, action.Description)
	assert.Equal(t, dice.Mediocre, action.Difficulty)
	assert.Empty(t, action.Aspects)
	assert.Empty(t, action.Stunts)
	assert.Empty(t, action.Effects)

	// Check timestamp is recent
	assert.WithinDuration(t, time.Now(), action.Timestamp, time.Second)
}

func TestAction_AddAspectInvoke(t *testing.T) {
	action := NewAction("test", "char", Overcome, "Athletics", "Test")

	invoke := AspectInvoke{
		AspectText:    "Athletic",
		Source:        "character",
		SourceID:      "char",
		IsFree:        false,
		FatePointCost: 1,
		Bonus:         2,
	}

	action.AddAspectInvoke(invoke)

	assert.Len(t, action.Aspects, 1)
	assert.Equal(t, "Athletic", action.Aspects[0].AspectText)
}

func TestAction_AddEffect(t *testing.T) {
	action := NewAction("test", "char", Attack, "Fight", "Test")

	effect := Effect{
		Type:        "stress",
		Target:      "target-char",
		Value:       3,
		Description: "3 physical stress",
	}

	action.AddEffect(effect)

	assert.Len(t, action.Effects, 1)
	assert.Equal(t, "stress", action.Effects[0].Type)
}

func TestAction_CalculateBonus(t *testing.T) {
	action := NewAction("test", "char", Overcome, "Athletics", "Test")

	// Test with no aspects
	bonus := action.CalculateBonus()
	assert.Equal(t, 0, bonus)

	// Add aspects with bonuses
	action.AddAspectInvoke(AspectInvoke{
		AspectText: "Athletic",
		Bonus:      2,
	})
	action.AddAspectInvoke(AspectInvoke{
		AspectText: "Determined",
		Bonus:      2,
	})

	bonus = action.CalculateBonus()
	assert.Equal(t, 4, bonus)
}

func TestAction_IsSuccess(t *testing.T) {
	action := NewAction("test", "char", Overcome, "Athletics", "Test")

	// Test with no outcome
	assert.False(t, action.IsSuccess())

	// Test with failure outcome
	action.Outcome = &dice.Outcome{Type: dice.Failure}
	assert.False(t, action.IsSuccess())

	// Test with tie outcome
	action.Outcome = &dice.Outcome{Type: dice.Tie}
	assert.False(t, action.IsSuccess())

	// Test with success outcome
	action.Outcome = &dice.Outcome{Type: dice.Success}
	assert.True(t, action.IsSuccess())

	// Test with success with style outcome
	action.Outcome = &dice.Outcome{Type: dice.SuccessWithStyle}
	assert.True(t, action.IsSuccess())
}

func TestAction_IsSuccessWithStyle(t *testing.T) {
	action := NewAction("test", "char", Overcome, "Athletics", "Test")

	// Test with no outcome
	assert.False(t, action.IsSuccessWithStyle())

	// Test with regular success outcome
	action.Outcome = &dice.Outcome{Type: dice.Success}
	assert.False(t, action.IsSuccessWithStyle())

	// Test with success with style outcome
	action.Outcome = &dice.Outcome{Type: dice.SuccessWithStyle}
	assert.True(t, action.IsSuccessWithStyle())
}

func TestAspectInvoke(t *testing.T) {
	invoke := AspectInvoke{
		AspectText:    "Wizard Detective",
		Source:        "character",
		SourceID:      "char-123",
		IsFree:        true,
		FatePointCost: 0,
		Bonus:         2,
	}

	// Test that all fields are set correctly
	assert.Equal(t, "Wizard Detective", invoke.AspectText)
	assert.Equal(t, "character", invoke.Source)
	assert.Equal(t, "char-123", invoke.SourceID)
	assert.True(t, invoke.IsFree)
	assert.Equal(t, 0, invoke.FatePointCost)
	assert.Equal(t, 2, invoke.Bonus)
}

func TestEffect(t *testing.T) {
	effect := Effect{
		Type:        "stress",
		Target:      "character-456",
		Value:       3,
		Description: "Takes 3 physical stress",
	}

	// Test that all fields are set correctly
	assert.Equal(t, "stress", effect.Type)
	assert.Equal(t, "character-456", effect.Target)
	assert.Equal(t, 3, effect.Value)
	assert.Equal(t, "Takes 3 physical stress", effect.Description)
}
