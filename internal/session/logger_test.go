package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger_WithPath(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_session.yaml")

	logger, err := NewLogger(logPath)
	require.NoError(t, err)
	require.NotNil(t, logger)
	assert.True(t, logger.IsEnabled())

	err = logger.Close()
	assert.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(logPath)
	assert.NoError(t, err)
}

func TestNewLogger_EmptyPath(t *testing.T) {
	logger, err := NewLogger("")
	require.NoError(t, err)
	require.NotNil(t, logger)
	assert.False(t, logger.IsEnabled())

	// Should be safe to call Close on disabled logger
	err = logger.Close()
	assert.NoError(t, err)
}

func TestLogger_LogMap(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_session.yaml")

	logger, err := NewLogger(logPath)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	logger.Log("player_input", map[string]any{"input": "I attack the goblin"})

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "---")
	assert.Contains(t, content, "type: player_input")
	assert.Contains(t, content, "input: I attack the goblin")
}

func TestLogger_LogStruct(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_session.yaml")

	logger, err := NewLogger(logPath)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	type TestAction struct {
		Skill      string `yaml:"skill"`
		Target     string `yaml:"target"`
		Difficulty int    `yaml:"difficulty"`
	}

	logger.Log("action_parse", TestAction{
		Skill:      "Fight",
		Target:     "goblin",
		Difficulty: 2,
	})

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "type: action_parse")
	assert.Contains(t, content, "skill: Fight")
	assert.Contains(t, content, "target: goblin")
	assert.Contains(t, content, "difficulty: 2")
}

func TestLogger_LogMultilineText(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_session.yaml")

	logger, err := NewLogger(logPath)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	longText := `Your fingers dance across the holographic interface.
A vulnerability emerges in the firmware update channel.
The drone's defenses begin to falter.`

	logger.Log("narrative", map[string]any{"text": longText})

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "type: narrative")
	// YAML should handle the multiline text
	assert.Contains(t, content, "holographic interface")
	assert.Contains(t, content, "defenses begin to falter")
}

func TestLogger_DisabledLogger_NoOp(t *testing.T) {
	logger, err := NewLogger("")
	require.NoError(t, err)
	assert.False(t, logger.IsEnabled())

	// These should all be no-ops and not panic
	logger.Log("player_input", map[string]any{"input": "test"})
	logger.Log("dice_roll", map[string]any{"result": 5})

	err = logger.Close()
	assert.NoError(t, err)
}

func TestLogger_MultipleEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_session.yaml")

	logger, err := NewLogger(logPath)
	require.NoError(t, err)

	logger.Log("player_input", map[string]any{"input": "First"})
	logger.Log("player_input", map[string]any{"input": "Second"})
	logger.Log("player_input", map[string]any{"input": "Third"})

	err = logger.Close()
	require.NoError(t, err)

	data, err := os.ReadFile(logPath)
	require.NoError(t, err)

	content := string(data)
	// Should have 3 YAML document separators
	assert.Equal(t, 3, strings.Count(content, "---"))
	assert.Contains(t, content, "First")
	assert.Contains(t, content, "Second")
	assert.Contains(t, content, "Third")
}

func TestGenerateLogPath(t *testing.T) {
	// Clean up any sessions dir created by tests
	defer func() {
		_ = os.RemoveAll(SessionsDir)
	}()

	tests := []struct {
		name      string
		prefix    string
		parts     []string
		maxLen    int
		wantMatch string // regex pattern to match
	}{
		{
			name:      "simple parts",
			prefix:    "session",
			parts:     []string{"western", "jesse"},
			maxLen:    0,
			wantMatch: `^sessions/session_western_jesse_\d{8}_\d{6}\.yaml$`,
		},
		{
			name:      "sanitize spaces",
			prefix:    "walkthrough",
			parts:     []string{"space opera", "simon falcon"},
			maxLen:    20,
			wantMatch: `^sessions/walkthrough_space_opera_simon_falcon_\d{8}_\d{6}\.yaml$`,
		},
		{
			name:      "sanitize special chars",
			prefix:    "session",
			parts:     []string{"sci-fi!", "player@123"},
			maxLen:    0,
			wantMatch: `^sessions/session_scifi_player123_\d{8}_\d{6}\.yaml$`,
		},
		{
			name:      "truncate long parts",
			prefix:    "scene_gen",
			parts:     []string{"a_very_long_description_that_exceeds_limit"},
			maxLen:    20,
			wantMatch: `^sessions/scene_gen_a_very_long_descript_\d{8}_\d{6}\.yaml$`,
		},
		{
			name:      "empty parts filtered",
			prefix:    "session",
			parts:     []string{"", "western", "", "jesse"},
			maxLen:    0,
			wantMatch: `^sessions/session_western_jesse_\d{8}_\d{6}\.yaml$`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GenerateLogPath(tt.prefix, tt.parts, tt.maxLen)
			require.NoError(t, err)
			assert.Regexp(t, tt.wantMatch, path)

			// Verify sessions directory was created
			info, err := os.Stat(SessionsDir)
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestNullLogger_Log(t *testing.T) {
	var nl NullLogger
	// Should not panic or produce any output
	nl.Log("event_type", map[string]any{"key": "value"})
	nl.Log("another_event", nil)
}

func TestNullLogger_Close(t *testing.T) {
	var nl NullLogger
	err := nl.Close()
	assert.NoError(t, err)
}

func TestNullLogger_ImplementsSessionLogger(t *testing.T) {
	// Compile-time check via interface assertion
	var _ SessionLogger = NullLogger{}
	var _ SessionLogger = (*Logger)(nil)
}

func TestLogger_Log_WriteError(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_write_error.yaml")

	logger, err := NewLogger(logPath)
	require.NoError(t, err)
	require.NotNil(t, logger)

	// Manually close the underlying file to trigger a write error
	require.NoError(t, logger.file.Close())
	logger.file = nil // prevent double-close in deferred cleanup

	// Should not panic, just log to stderr
	logger.Log("some_event", map[string]any{"data": "test"})
}

func TestSanitizePart(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "lowercase conversion",
			input:  "MyCharacter",
			maxLen: 0,
			want:   "mycharacter",
		},
		{
			name:   "spaces to underscores",
			input:  "space opera hero",
			maxLen: 0,
			want:   "space_opera_hero",
		},
		{
			name:   "remove special chars",
			input:  "name-with@special!chars",
			maxLen: 0,
			want:   "namewithspecialchars",
		},
		{
			name:   "truncate",
			input:  "verylongname",
			maxLen: 8,
			want:   "verylong",
		},
		{
			name:   "mixed operations",
			input:  "Jesse Calhoun!",
			maxLen: 20,
			want:   "jesse_calhoun",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePart(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got)
		})
	}
}
