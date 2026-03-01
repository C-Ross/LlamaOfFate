package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAspectGenerator(t *testing.T) {
	mockClient := newTestLLMClient()
	generator := NewAspectGenerator(mockClient)

	assert.NotNil(t, generator)
	assert.Equal(t, mockClient, generator.llmClient)
}

func TestGenerateAspect_Success(t *testing.T) {
	mockClient := newTestLLMClient(`{
			"aspect_text": "High Ground Advantage",
			"description": "Character has positioned themselves advantageously",
			"duration": "scene",
			"free_invokes": 1,
			"is_boost": false,
			"reasoning": "Success on Athletics to gain positional advantage"
		}`)

	generator := NewAspectGenerator(mockClient)

	// Create test data
	char := character.NewCharacter("test-char", "Test Hero")
	char.Aspects.HighConcept = "Agile Fighter"
	char.Aspects.Trouble = "Reckless in Combat"
	char.SetSkill("Athletics", dice.Good)

	roller := dice.NewSeededRoller(12345)
	testAction := action.NewAction("test-action", "test-char", action.CreateAdvantage, "Athletics", "Climb to higher ground for advantage")
	testAction.RawInput = "I want to climb up the rocky outcropping to get a better position"
	testAction.Difficulty = dice.Fair

	result := roller.RollWithModifier(dice.Good, 0)
	outcome := result.CompareAgainst(dice.Fair)

	req := prompt.AspectGenerationRequest{
		Character:       char,
		Action:          testAction,
		Outcome:         outcome,
		Context:         "A rocky battlefield with various elevation changes and cover",
		TargetType:      "situation",
		ExistingAspects: []string{"Unstable Footing", "Limited Cover"},
	}

	response, err := generator.GenerateAspect(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "High Ground Advantage", response.AspectText)
	assert.Equal(t, "Character has positioned themselves advantageously", response.Description)
	assert.Equal(t, "scene", response.Duration)
	assert.Equal(t, 1, response.FreeInvokes)
	assert.False(t, response.IsBoost)
}

func TestGenerateAspect_SuccessWithStyle(t *testing.T) {
	mockClient := newTestLLMClient(`{
			"aspect_text": "Perfect Positioning",
			"description": "Masterful tactical advantage achieved",
			"duration": "scene",
			"free_invokes": 2,
			"is_boost": false,
			"reasoning": "Success with Style on a tactical maneuver"
		}`)

	generator := NewAspectGenerator(mockClient)

	char := character.NewCharacter("test-char", "Test Hero")
	char.SetSkill("Athletics", dice.Great)

	testAction := action.NewAction("test-action", "test-char", action.CreateAdvantage, "Athletics", "Parkour to perfect position")
	testAction.Difficulty = dice.Fair

	// Create an outcome that results in Success with Style
	roller := dice.NewSeededRoller(12345)
	result := roller.RollWithModifier(dice.Great, 2) // This should give us enough shifts
	outcome := result.CompareAgainst(dice.Fair)

	req := prompt.AspectGenerationRequest{
		Character:  char,
		Action:     testAction,
		Outcome:    outcome,
		Context:    "Urban rooftop chase scene",
		TargetType: "character",
	}

	response, err := generator.GenerateAspect(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, response)

	// For Success with Style, we expect 2 free invokes
	if outcome.Type == dice.SuccessWithStyle {
		assert.Equal(t, 2, response.FreeInvokes)
		assert.False(t, response.IsBoost)
	}
}

func TestGenerateAspect_Tie(t *testing.T) {
	mockClient := newTestLLMClient(`{
			"aspect_text": "Momentary Opening",
			"description": "Brief opportunity that won't last long",
			"duration": "scene",
			"free_invokes": 1,
			"is_boost": true,
			"reasoning": "Tie creates a boost"
		}`)

	generator := NewAspectGenerator(mockClient)

	char := character.NewCharacter("test-char", "Test Hero")
	char.SetSkill("Deceive", dice.Fair)

	testAction := action.NewAction("test-action", "test-char", action.CreateAdvantage, "Deceive", "Create a distraction")
	testAction.Difficulty = dice.Fair

	// Create a tie outcome
	result := &dice.CheckResult{
		Roll:       &dice.Roll{Total: 0},
		BaseSkill:  dice.Fair,
		Modifier:   0,
		FinalValue: dice.Fair,
	}
	outcome := result.CompareAgainst(dice.Fair)

	req := prompt.AspectGenerationRequest{
		Character:  char,
		Action:     testAction,
		Outcome:    outcome,
		Context:    "Crowded marketplace",
		TargetType: "situation",
	}

	response, err := generator.GenerateAspect(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, response.FreeInvokes)
	assert.True(t, response.IsBoost)
}

func TestGenerateAspect_Failure(t *testing.T) {
	mockClient := newTestLLMClient("Failure - no aspect created")

	generator := NewAspectGenerator(mockClient)

	char := character.NewCharacter("test-char", "Test Hero")
	char.SetSkill("Athletics", dice.Average)

	testAction := action.NewAction("test-action", "test-char", action.CreateAdvantage, "Athletics", "Attempt to scale wall")
	testAction.Difficulty = dice.Great

	// Create a failure outcome
	result := &dice.CheckResult{
		Roll:       &dice.Roll{Total: -2},
		BaseSkill:  dice.Average,
		Modifier:   0,
		FinalValue: dice.Poor,
	}
	outcome := result.CompareAgainst(dice.Great)

	req := prompt.AspectGenerationRequest{
		Character:  char,
		Action:     testAction,
		Outcome:    outcome,
		Context:    "High stone wall",
		TargetType: "character",
	}

	response, err := generator.GenerateAspect(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "", response.AspectText)
	assert.Equal(t, 0, response.FreeInvokes)
	assert.False(t, response.IsBoost)
	assert.Contains(t, response.Description, "No aspect created")
}

func TestGenerateAspect_WrongActionType(t *testing.T) {
	mockClient := newTestLLMClient()
	generator := NewAspectGenerator(mockClient)

	char := character.NewCharacter("test-char", "Test Hero")
	testAction := action.NewAction("test-action", "test-char", action.Attack, "Fight", "Punch the orc")

	result := &dice.CheckResult{FinalValue: dice.Good}
	outcome := result.CompareAgainst(dice.Fair)

	req := prompt.AspectGenerationRequest{
		Character: char,
		Action:    testAction,
		Outcome:   outcome,
	}

	_, err := generator.GenerateAspect(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action type must be CreateAdvantage")
}

func TestGenerateAspect_NilOutcome(t *testing.T) {
	mockClient := newTestLLMClient()
	generator := NewAspectGenerator(mockClient)

	char := character.NewCharacter("test-char", "Test Hero")
	testAction := action.NewAction("test-action", "test-char", action.CreateAdvantage, "Athletics", "Climb")

	req := prompt.AspectGenerationRequest{
		Character: char,
		Action:    testAction,
		Outcome:   nil,
	}

	_, err := generator.GenerateAspect(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "outcome cannot be nil")
}

func TestBuildPrompt(t *testing.T) {
	generator := NewAspectGenerator(newTestLLMClient())

	char := character.NewCharacter("test-char", "Zara the Swift")
	char.Aspects.HighConcept = "Acrobatic Thief"
	char.Aspects.Trouble = "Can't Resist a Challenge"
	char.Aspects.AddAspect("Friends in Low Places")
	char.SetSkill("Stealth", dice.Great)

	testAction := action.NewAction("test-action", "test-char", action.CreateAdvantage, "Stealth", "Hide in the shadows")
	testAction.RawInput = "I want to find a good hiding spot to ambush from"
	testAction.Difficulty = dice.Fair

	result := &dice.CheckResult{
		Roll: &dice.Roll{
			Dice:  [4]dice.FateDie{dice.Plus, dice.Blank, dice.Blank, dice.Plus},
			Total: 2,
		},
		BaseSkill:  dice.Great,
		Modifier:   0,
		FinalValue: dice.Legendary,
	}
	outcome := result.CompareAgainst(dice.Fair)

	req := prompt.AspectGenerationRequest{
		Character:       char,
		Action:          testAction,
		Outcome:         outcome,
		Context:         "Dark alleyway with plenty of shadows and debris",
		TargetType:      "situation",
		ExistingAspects: []string{"Dim Lighting", "Narrow Passage"},
	}

	prompt := generator.buildPrompt(req)

	assert.Contains(t, prompt, "Zara the Swift")
	assert.Contains(t, prompt, "Acrobatic Thief")
	assert.Contains(t, prompt, "Can't Resist a Challenge")
	assert.Contains(t, prompt, "Friends in Low Places")
	assert.Contains(t, prompt, "Stealth")
	assert.Contains(t, prompt, "Hide in the shadows")
	assert.Contains(t, prompt, "Dark alleyway")
	assert.Contains(t, prompt, "Dim Lighting")
	assert.Contains(t, prompt, "situation")
}
