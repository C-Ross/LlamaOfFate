package terminal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTerminalUI(t *testing.T) {
	ui := NewTerminalUI()
	require.NotNil(t, ui)
	require.NotNil(t, ui.reader)
}

func TestTerminalUI_isExitCommand(t *testing.T) {
	ui := NewTerminalUI()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "exit command",
			input:    "exit",
			expected: true,
		},
		{
			name:     "quit command",
			input:    "quit",
			expected: true,
		},
		{
			name:     "end command",
			input:    "end",
			expected: true,
		},
		{
			name:     "leave command",
			input:    "leave",
			expected: true,
		},
		{
			name:     "resolve command",
			input:    "resolve",
			expected: true,
		},
		{
			name:     "uppercase exit",
			input:    "EXIT",
			expected: true,
		},
		{
			name:     "mixed case quit",
			input:    "QuIt",
			expected: true,
		},
		{
			name:     "exit with whitespace",
			input:    "  exit  ",
			expected: true,
		},
		{
			name:     "regular command",
			input:    "attack goblin",
			expected: false,
		},
		{
			name:     "help command",
			input:    "help",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "partial match",
			input:    "exiting",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.isExitCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTerminalUI_DisplayMethods(t *testing.T) {
	ui := NewTerminalUI()

	// These methods should not panic when called
	// We can't easily test the actual output without mocking stdout
	// but we can ensure they execute without errors
	require.NotPanics(t, func() {
		ui.DisplayActionAttempt("Attack the goblin")
	})

	require.NotPanics(t, func() {
		ui.DisplayActionResult("Fight", "Good (+3)", 2, "Fair (+2)", "Success")
	})

	require.NotPanics(t, func() {
		ui.DisplayNarrative("You successfully strike the goblin!")
	})

	require.NotPanics(t, func() {
		ui.DisplayDialog("Hello there", "The goblin grunts in response")
	})

	require.NotPanics(t, func() {
		ui.DisplaySystemMessage("Scene started")
	})
}
