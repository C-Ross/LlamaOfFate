package terminal

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSceneInfo implements uicontract.SceneInfo for testing.
type mockSceneInfo struct {
	scene               *scene.Scene
	player              *character.Character
	conversationHistory []uicontract.ConversationEntry
}

func (m *mockSceneInfo) GetCurrentScene() *scene.Scene   { return m.scene }
func (m *mockSceneInfo) GetPlayer() *character.Character { return m.player }
func (m *mockSceneInfo) GetConversationHistory() []uicontract.ConversationEntry {
	return m.conversationHistory
}

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

func TestTerminalUI_EmitMethods(t *testing.T) {
	ui := NewTerminalUI()

	// These events should not panic when emitted
	// We can't easily test the actual output without mocking stdout
	// but we can ensure they execute without errors
	require.NotPanics(t, func() {
		ui.Emit(uicontract.ActionAttemptEvent{Description: "Attack the goblin"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ActionResultEvent{Skill: "Fight", SkillLevel: "Good (+3)", Bonuses: 2, Result: "Fair (+2)", Outcome: "Success"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NarrativeEvent{Text: "You successfully strike the goblin!"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.DialogEvent{PlayerInput: "Hello there", GMResponse: "The goblin grunts in response"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.SystemMessageEvent{Message: "Scene started"})
	})
}

func TestHandleSpecialCommands_Recognized(t *testing.T) {
	ui := NewTerminalUI()
	player := character.NewCharacter("player1", "Test Hero")
	player.Aspects.HighConcept = "Brave Warrior"
	player.Aspects.Trouble = "Quick to Anger"

	testScene := scene.NewScene("scene1", "Test Scene", "A dark room")
	testScene.AddSituationAspect(scene.SituationAspect{Aspect: "Dim Lighting", FreeInvokes: 1})

	ui.SetSceneInfo(&mockSceneInfo{
		scene:  testScene,
		player: player,
	})

	recognized := []string{
		"help", "?",
		"scene",
		"character", "char", "me",
		"status",
		"aspects",
		"history", "conversation",
	}

	for _, cmd := range recognized {
		t.Run(cmd, func(t *testing.T) {
			require.NotPanics(t, func() {
				handled := ui.handleSpecialCommands(cmd)
				assert.True(t, handled, "command %q should be handled", cmd)
			})
		})
	}
}

func TestHandleSpecialCommands_NotRecognized(t *testing.T) {
	ui := NewTerminalUI()
	player := character.NewCharacter("player1", "Test Hero")

	ui.SetSceneInfo(&mockSceneInfo{
		scene:  scene.NewScene("s1", "S", "D"),
		player: player,
	})

	unrecognized := []string{
		"attack goblin",
		"look around",
		"",
		"exit",
	}

	for _, cmd := range unrecognized {
		t.Run(cmd, func(t *testing.T) {
			handled := ui.handleSpecialCommands(cmd)
			assert.False(t, handled, "command %q should not be handled", cmd)
		})
	}
}

func TestHandleSpecialCommands_CaseInsensitive(t *testing.T) {
	ui := NewTerminalUI()
	ui.SetSceneInfo(&mockSceneInfo{
		scene:  scene.NewScene("s1", "S", "D"),
		player: character.NewCharacter("p1", "P"),
	})

	assert.True(t, ui.handleSpecialCommands("HELP"))
	assert.True(t, ui.handleSpecialCommands("Scene"))
	assert.True(t, ui.handleSpecialCommands("CHARACTER"))
	assert.True(t, ui.handleSpecialCommands("Status"))
	assert.True(t, ui.handleSpecialCommands("ASPECTS"))
	assert.True(t, ui.handleSpecialCommands("History"))
}
