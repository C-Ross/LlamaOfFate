package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActionParser(t *testing.T) {
	mockClient := &MockLLMClient{}
	parser := NewActionParser(mockClient)

	assert.NotNil(t, parser)
	assert.Equal(t, mockClient, parser.llmClient)
}

func TestActionParser_ParseAction_Overcome(t *testing.T) {
	// Setup mock LLM response for overcome action
	mockResponse := `{
		"action_type": "Overcome",
		"skill": "Athletics",
		"description": "Jump across the wide chasm",
		"target": "",
		"reasoning": "Player wants to get past an obstacle using physical movement",
		"confidence": 9
	}`

	mockClient := &MockLLMClient{response: mockResponse}
	parser := NewActionParser(mockClient)

	// Create test character
	char := character.NewCharacter("test-char", "Test Hero")
	char.Aspects.HighConcept = "Daring Adventurer"
	char.SetSkill("Athletics", dice.Good)

	// Test request
	req := ActionParseRequest{
		Character: char,
		RawInput:  "I want to jump across the chasm",
		Context:   "Standing at the edge of a deep rocky chasm",
	}

	// Parse the action
	parsedAction, err := parser.ParseAction(context.Background(), req)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, action.Overcome, parsedAction.Type)
	assert.Equal(t, "Athletics", parsedAction.Skill)
	assert.Equal(t, "Jump across the wide chasm", parsedAction.Description)
	assert.Equal(t, "", parsedAction.Target)
	assert.Equal(t, "I want to jump across the chasm", parsedAction.RawInput)
	assert.Equal(t, char.ID, parsedAction.CharacterID)
}

func TestActionParser_ParseAction_CreateAdvantage(t *testing.T) {
	// Setup mock LLM response for create advantage action
	mockResponse := `{
		"action_type": "Create an Advantage",
		"skill": "Stealth",
		"description": "Find a hidden vantage point to observe the guards",
		"target": "",
		"reasoning": "Player wants to set up an advantage for future actions using stealth",
		"confidence": 8
	}`

	mockClient := &MockLLMClient{response: mockResponse}
	parser := NewActionParser(mockClient)

	// Create test character
	char := character.NewCharacter("test-char", "Sneaky Rogue")
	char.Aspects.HighConcept = "Master Thief"
	char.SetSkill("Stealth", dice.Great)

	// Test request
	req := ActionParseRequest{
		Character: char,
		RawInput:  "I want to scout around and find a good hiding spot to watch the guards",
		Context:   "Outside the heavily guarded castle gates",
	}

	// Parse the action
	parsedAction, err := parser.ParseAction(context.Background(), req)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, action.CreateAdvantage, parsedAction.Type)
	assert.Equal(t, "Stealth", parsedAction.Skill)
	assert.Equal(t, "Find a hidden vantage point to observe the guards", parsedAction.Description)
	assert.Equal(t, "", parsedAction.Target)
	assert.Equal(t, char.ID, parsedAction.CharacterID)
}

func TestActionParser_ParseAction_Attack(t *testing.T) {
	// Setup mock LLM response for attack action
	mockResponse := `{
		"action_type": "Attack",
		"skill": "Fight",
		"description": "Strike the orc with my sword",
		"target": "orc",
		"reasoning": "Player is trying to harm an enemy in melee combat",
		"confidence": 10
	}`

	mockClient := &MockLLMClient{response: mockResponse}
	parser := NewActionParser(mockClient)

	// Create test character
	char := character.NewCharacter("test-char", "Brave Warrior")
	char.Aspects.HighConcept = "Skilled Swordsman"
	char.SetSkill("Fight", dice.Superb)

	// Test request
	req := ActionParseRequest{
		Character: char,
		RawInput:  "I attack the orc with my sword!",
		Context:   "In melee combat with a large orc",
	}

	// Parse the action
	parsedAction, err := parser.ParseAction(context.Background(), req)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, action.Attack, parsedAction.Type)
	assert.Equal(t, "Fight", parsedAction.Skill)
	assert.Equal(t, "Strike the orc with my sword", parsedAction.Description)
	assert.Equal(t, "orc", parsedAction.Target)
}

func TestActionParser_ParseAction_Defend(t *testing.T) {
	// Setup mock LLM response for defend action
	mockResponse := `{
		"action_type": "Defend",
		"skill": "Athletics",
		"description": "Dodge the incoming arrow",
		"target": "",
		"reasoning": "Player is trying to avoid an incoming attack using agility",
		"confidence": 9
	}`

	mockClient := &MockLLMClient{response: mockResponse}
	parser := NewActionParser(mockClient)

	// Create test character
	char := character.NewCharacter("test-char", "Agile Scout")
	char.SetSkill("Athletics", dice.Good)

	// Test request
	req := ActionParseRequest{
		Character: char,
		RawInput:  "I try to dodge the arrow!",
		Context:   "Archer is shooting at me",
	}

	// Parse the action
	parsedAction, err := parser.ParseAction(context.Background(), req)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, action.Defend, parsedAction.Type)
	assert.Equal(t, "Athletics", parsedAction.Skill)
	assert.Equal(t, "Dodge the incoming arrow", parsedAction.Description)
	assert.Equal(t, "", parsedAction.Target)
}

func TestActionParser_ParseAction_InvalidJSON(t *testing.T) {
	// Setup mock LLM response with invalid JSON
	mockResponse := `invalid json response`

	mockClient := &MockLLMClient{response: mockResponse}
	parser := NewActionParser(mockClient)

	// Create test character
	char := character.NewCharacter("test-char", "Test Hero")

	// Test request
	req := ActionParseRequest{
		Character: char,
		RawInput:  "I want to do something",
	}

	// Parse the action
	_, err := parser.ParseAction(context.Background(), req)

	// Should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse LLM response")
}

func TestActionParser_ParseAction_InvalidActionType(t *testing.T) {
	// Setup mock LLM response with an unrecognized action type
	// Now the parser defaults to Overcome for unknown types instead of erroring
	mockResponse := `{
		"action_type": "InvalidType",
		"skill": "Athletics",
		"description": "Do something",
		"target": "",
		"reasoning": "Test invalid action type",
		"confidence": 5
	}`

	mockClient := &MockLLMClient{response: mockResponse}
	parser := NewActionParser(mockClient)

	// Create test character
	char := character.NewCharacter("test-char", "Test Hero")

	// Test request
	req := ActionParseRequest{
		Character: char,
		RawInput:  "I want to do something",
	}

	// Parse the action - now defaults to Overcome instead of erroring
	result, err := parser.ParseAction(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, action.Overcome, result.Type)
}

func TestBuildPrompts(t *testing.T) {
	parser := NewActionParser(&MockLLMClient{})

	// Create test character with various attributes
	char := character.NewCharacter("test-char", "Zara the Swift")
	char.Aspects.HighConcept = "Acrobatic Cat Burglar"
	char.Aspects.Trouble = "Can't Resist a Shiny Challenge"
	char.Aspects.AddAspect("Friends in Low Places")
	char.SetSkill("Athletics", dice.Great)
	char.SetSkill("Stealth", dice.Good)

	req := ActionParseRequest{
		Character: char,
		RawInput:  "I want to sneak past the guards",
		Context:   "In the castle hallway with patrolling guards",
	}

	// Test system prompt
	systemPrompt, err := parser.buildSystemPrompt()
	require.NoError(t, err)
	assert.Contains(t, systemPrompt, "Fate Core")
	assert.Contains(t, systemPrompt, "Overcome")
	assert.Contains(t, systemPrompt, "Create an Advantage")
	assert.Contains(t, systemPrompt, "Athletics")
	assert.Contains(t, systemPrompt, "Stealth")
	assert.Contains(t, systemPrompt, "JSON")

	// Test user prompt
	userPrompt, err := parser.buildUserPrompt(req)
	require.NoError(t, err)
	assert.Contains(t, userPrompt, "Zara the Swift")
	assert.Contains(t, userPrompt, "Acrobatic Cat Burglar")
	assert.Contains(t, userPrompt, "Can't Resist a Shiny Challenge")
	assert.Contains(t, userPrompt, "Friends in Low Places")
	assert.Contains(t, userPrompt, "Athletics Great")
	assert.Contains(t, userPrompt, "Stealth Good")
	assert.Contains(t, userPrompt, "I want to sneak past the guards")
	assert.Contains(t, userPrompt, "In the castle hallway with patrolling guards")
}

func TestParseActionType(t *testing.T) {
	tests := []struct {
		input    string
		expected action.ActionType
		hasError bool
	}{
		{"Overcome", action.Overcome, false},
		{"Create an Advantage", action.CreateAdvantage, false},
		{"Attack", action.Attack, false},
		{"Defend", action.Defend, false},
		// Unknown types now default to Overcome instead of erroring
		{"Invalid", action.Overcome, false},
		{"", action.Overcome, false},
		// Skill names map to appropriate actions
		{"fight", action.Attack, false},
		{"provoke", action.Attack, false},
		{"athletics", action.Defend, false},
		{"will", action.Defend, false},
		{"investigation", action.Overcome, false},
		{"stealth", action.Overcome, false},
	}

	for _, test := range tests {
		result, err := parseActionType(test.input)
		if test.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, test.expected, result)
		}
	}
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain JSON",
			input:    `{"action_type": "Overcome", "skill": "Athletics"}`,
			expected: `{"action_type": "Overcome", "skill": "Athletics"}`,
		},
		{
			name:     "JSON with markdown code block",
			input:    "```json\n{\"action_type\": \"Overcome\", \"skill\": \"Athletics\"}\n```",
			expected: `{"action_type": "Overcome", "skill": "Athletics"}`,
		},
		{
			name:     "JSON with generic code block",
			input:    "```\n{\"action_type\": \"Overcome\", \"skill\": \"Athletics\"}\n```",
			expected: `{"action_type": "Overcome", "skill": "Athletics"}`,
		},
		{
			name:     "JSON with extra whitespace",
			input:    "  \n  {\"action_type\": \"Overcome\", \"skill\": \"Athletics\"}  \n  ",
			expected: `{"action_type": "Overcome", "skill": "Athletics"}`,
		},
		{
			name:     "Multiple JSON blocks - should take last one",
			input:    "```json\n{\"action_type\": \"Investigate\", \"skill\": \"Investigate\"}\n```\n\nCorrected to match the exact action type:\n\n```json\n{\"action_type\": \"Create an Advantage\", \"skill\": \"Investigate\"}\n```",
			expected: `{"action_type": "Create an Advantage", "skill": "Investigate"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := cleanJSONResponse(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestActionParser_ParseAction_WithOtherCharacters(t *testing.T) {
	// Setup mock LLM response
	mockResponse := `{
		"action_type": "Attack",
		"skill": "Fight",
		"description": "Attack the orc warrior with my sword",
		"target": "orc-warrior",
		"reasoning": "Player wants to attack a specific enemy",
		"confidence": 9
	}`

	mockClient := &MockLLMClient{response: mockResponse}
	parser := NewActionParser(mockClient)

	// Create test character
	char := character.NewCharacter("test-char", "Test Hero")
	char.Aspects.HighConcept = "Brave Fighter"
	char.SetSkill("Fight", dice.Good)

	// Create other characters in scene
	orc := character.NewCharacter("orc-warrior", "Orc Warrior")
	orc.Aspects.HighConcept = "Brutal Fighter"

	goblin := character.NewCharacter("goblin-scout", "Goblin Scout")
	goblin.Aspects.HighConcept = "Sneaky Archer"

	// Test request with other characters - this should trigger the template error
	req := ActionParseRequest{
		Character:       char,
		RawInput:        "I attack the orc warrior with my sword",
		Context:         "In combat with multiple enemies",
		OtherCharacters: []*character.Character{orc, goblin},
	}

	// This should now work successfully with the fixed template
	parsedAction, err := parser.ParseAction(context.Background(), req)

	// We expect this to succeed now
	require.NoError(t, err)
	assert.Equal(t, action.Attack, parsedAction.Type)
	assert.Equal(t, "Fight", parsedAction.Skill)
	assert.Equal(t, "Attack the orc warrior with my sword", parsedAction.Description)
	assert.Equal(t, "orc-warrior", parsedAction.Target)
	assert.Equal(t, char.ID, parsedAction.CharacterID)
}

func TestActionParser_TemplateErrorRegression(t *testing.T) {
	// This test verifies the fix for the template error:
	// "failed to execute user prompt template: template: action_parse:18:5:
	// executing "action_parse" at <ne $id $.Character.ID>: error calling ne: incompatible types for comparison"

	// The error occurred because the template was trying to range over OtherCharacters
	// as if it was a map[string]*character.Character when it's actually []*character.Character

	mockResponse := `{
		"action_type": "Attack",
		"skill": "Fight",
		"description": "Attack with sword",
		"target": "enemy",
		"reasoning": "Combat action",
		"confidence": 9
	}`

	mockClient := &MockLLMClient{response: mockResponse}
	parser := NewActionParser(mockClient)

	char := character.NewCharacter("hero", "Hero")
	enemy := character.NewCharacter("enemy", "Enemy")

	// This used to cause the template error when OtherCharacters was populated
	req := ActionParseRequest{
		Character:       char,
		RawInput:        "attack enemy",
		Context:         "combat",
		OtherCharacters: []*character.Character{enemy}, // This triggers the template section that was broken
	}

	// This should work now (would have failed before the fix)
	_, err := parser.ParseAction(context.Background(), req)
	require.NoError(t, err, "Template should handle OtherCharacters slice correctly")
}
