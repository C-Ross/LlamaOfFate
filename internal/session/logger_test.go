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
