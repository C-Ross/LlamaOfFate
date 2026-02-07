package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			result := CleanJSONResponse(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}
