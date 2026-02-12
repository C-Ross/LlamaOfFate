package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClassification(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Clean responses
		{"simple word", "dialog", "dialog"},
		{"uppercase", "DIALOG", "dialog"},
		{"mixed case", "Narrative", "narrative"},

		// Whitespace
		{"leading/trailing spaces", "  clarification  ", "clarification"},
		{"leading/trailing newlines", "\naction\n", "action"},

		// Trailing explanations
		{"trailing explanation", "dialog - the player is speaking", "dialog"},
		{"newline after type", "action\nbecause there is opposition", "action"},
		{"tab after type", "narrative\tthis is mundane", "narrative"},

		// Markdown formatting
		{"markdown heading", "## narrative", "narrative"},
		{"markdown bold", "**action**", "action"},
		{"markdown heading with explanation", "## dialog because they are speaking", "dialog"},
		{"backtick wrapped", "`clarification`", "clarification"},
		{"double backtick", "``narrative``", "narrative"},

		// Quotes
		{"double quotes", "\"narrative\"", "narrative"},
		{"single quotes", "'action'", "action"},

		// Edge cases
		{"empty string", "", ""},
		{"only punctuation", "##**", ""},
		{"only spaces", "   ", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseClassification(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseFateNarration_ValidJSON(t *testing.T) {
	input := `{
		"fates": [
			{"id": "npc-1", "name": "Goblin", "description": "Killed", "permanent": true},
			{"id": "npc-2", "name": "Orc", "description": "Fled", "permanent": false}
		],
		"narrative": "The goblin fell lifeless. The orc scrambled away into the darkness."
	}`

	result, err := ParseFateNarration(input)
	require.NoError(t, err)
	require.Len(t, result.Fates, 2)

	assert.Equal(t, "npc-1", result.Fates[0].ID)
	assert.Equal(t, "Goblin", result.Fates[0].Name)
	assert.Equal(t, "Killed", result.Fates[0].Description)
	assert.True(t, result.Fates[0].Permanent)

	assert.Equal(t, "npc-2", result.Fates[1].ID)
	assert.Equal(t, "Orc", result.Fates[1].Name)
	assert.Equal(t, "Fled", result.Fates[1].Description)
	assert.False(t, result.Fates[1].Permanent)

	assert.Contains(t, result.Narrative, "goblin fell lifeless")
}

func TestParseFateNarration_JSONEmbeddedInText(t *testing.T) {
	input := `Here is the result:
	{
		"fates": [{"id": "npc-guard", "name": "Guard", "description": "Captured", "permanent": false}],
		"narrative": "You tie up the guard."
	}
	That's the output.`

	result, err := ParseFateNarration(input)
	require.NoError(t, err)
	require.Len(t, result.Fates, 1)
	assert.Equal(t, "npc-guard", result.Fates[0].ID)
	assert.Equal(t, "Guard", result.Fates[0].Name)
}

func TestParseFateNarration_MissingFates(t *testing.T) {
	input := `{"fates": [], "narrative": "Nothing happened."}`

	_, err := ParseFateNarration(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing fates")
}

func TestParseFateNarration_MissingNarrative(t *testing.T) {
	input := `{"fates": [{"id": "npc-1", "name": "Goblin", "description": "Dead", "permanent": true}], "narrative": ""}`

	_, err := ParseFateNarration(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing narrative")
}

func TestParseFateNarration_InvalidJSON(t *testing.T) {
	input := `not json at all`

	_, err := ParseFateNarration(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}
