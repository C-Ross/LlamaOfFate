package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"gopkg.in/yaml.v3"
)

const saveFileName = "game_save.yaml"

// YAMLSaver persists game state as a single YAML file in the configured directory.
// It implements engine.GameStateSaver.
type YAMLSaver struct {
	dir string
}

// NewYAMLSaver creates a YAMLSaver that stores saves in the given directory.
// The directory is created on the first Save if it does not exist.
func NewYAMLSaver(dir string) *YAMLSaver {
	return &YAMLSaver{dir: dir}
}

// Save writes the game state to a YAML file, creating the directory if needed.
func (s *YAMLSaver) Save(state engine.GameState) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create save directory: %w", err)
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal game state: %w", err)
	}

	path := filepath.Join(s.dir, saveFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write save file: %w", err)
	}

	return nil
}

// Load reads a previously saved game state from the YAML file.
// Returns (nil, nil) if no save file exists.
func (s *YAMLSaver) Load() (*engine.GameState, error) {
	path := filepath.Join(s.dir, saveFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read save file: %w", err)
	}

	var state engine.GameState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal game state: %w", err)
	}

	return &state, nil
}

// Path returns the full path to the save file.
func (s *YAMLSaver) Path() string {
	return filepath.Join(s.dir, saveFileName)
}

// Delete removes the save file. Returns nil if the file does not exist.
func (s *YAMLSaver) Delete() error {
	path := filepath.Join(s.dir, saveFileName)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete save file: %w", err)
	}
	return nil
}

// Compile-time interface check.
var _ engine.GameStateSaver = (*YAMLSaver)(nil)
