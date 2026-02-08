package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
