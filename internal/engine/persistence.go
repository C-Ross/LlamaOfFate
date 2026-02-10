package engine

// GameStateSaver defines the interface for game state persistence.
// It is defined in the engine package (the consumer) to avoid coupling
// the engine to any specific storage implementation.
//
// Implementations live in separate packages (e.g., internal/storage/)
// and import engine.GameState — the engine never imports storage.
type GameStateSaver interface {
	// Save persists the current game state.
	Save(state GameState) error

	// Load retrieves the most recently saved game state.
	// Returns (nil, nil) if no save exists.
	Load() (*GameState, error)
}

// noopSaver is the default GameStateSaver when no persistence is configured.
// All operations succeed silently, making persistence opt-in with zero
// behavior change to the engine.
type noopSaver struct{}

func (noopSaver) Save(GameState) error      { return nil }
func (noopSaver) Load() (*GameState, error) { return nil, nil }
