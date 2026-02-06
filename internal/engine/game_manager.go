package engine

import (
	"context"
	"fmt"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

// GameManager orchestrates the overall game flow, managing scenarios and game state
// This is a thin wrapper for Phase 1; future phases will add scenario selection/generation
type GameManager struct {
	engine        *Engine
	player        *character.Character
	ui            UI
	sessionLogger *session.Logger
	settings      ScenarioSettings
}

// NewGameManager creates a new game manager
func NewGameManager(engine *Engine) *GameManager {
	return &GameManager{
		engine: engine,
	}
}

// SetPlayer sets the player character
func (g *GameManager) SetPlayer(player *character.Character) {
	g.player = player
}

// SetUI sets the UI for the game
func (g *GameManager) SetUI(ui UI) {
	g.ui = ui
}

// SetSessionLogger sets the session logger
func (g *GameManager) SetSessionLogger(logger *session.Logger) {
	g.sessionLogger = logger
}

// SetSettings configures the game/scenario settings
func (g *GameManager) SetSettings(settings ScenarioSettings) {
	g.settings = settings
}

// Run starts the game loop
// For Phase 1, this simply runs a single scenario
// Future phases will add scenario selection, character creation, etc.
func (g *GameManager) Run(ctx context.Context) error {
	if g.engine == nil {
		return fmt.Errorf("engine is required")
	}
	if g.player == nil {
		return fmt.Errorf("player character is required")
	}
	if g.ui == nil {
		return fmt.Errorf("UI is required")
	}

	// Create and configure the scenario manager
	scenarioManager := NewScenarioManager(g.engine, g.player)
	scenarioManager.SetUI(g.ui)
	scenarioManager.SetSettings(g.settings)
	if g.sessionLogger != nil {
		scenarioManager.SetSessionLogger(g.sessionLogger)
	}

	// Run the scenario
	// Phase 1: Single scenario, exit when done
	// Future: Loop for multiple scenarios, handle scenario selection
	if err := scenarioManager.Run(ctx); err != nil {
		return fmt.Errorf("scenario error: %w", err)
	}

	return nil
}

// RunWithInitialScene runs the game starting from a pre-configured scene
// This is useful for demos and testing specific scenarios
func (g *GameManager) RunWithInitialScene(ctx context.Context, initialScene *InitialSceneConfig) error {
	if g.engine == nil {
		return fmt.Errorf("engine is required")
	}
	if g.player == nil {
		return fmt.Errorf("player character is required")
	}
	if g.ui == nil {
		return fmt.Errorf("UI is required")
	}
	if initialScene == nil {
		return fmt.Errorf("initial scene config is required")
	}

	// Create and configure the scenario manager
	scenarioManager := NewScenarioManager(g.engine, g.player)
	scenarioManager.SetUI(g.ui)
	scenarioManager.SetSettings(g.settings)
	scenarioManager.SetInitialScene(initialScene.Scene, initialScene.NPCs)
	if g.sessionLogger != nil {
		scenarioManager.SetSessionLogger(g.sessionLogger)
	}

	// Run the scenario
	if err := scenarioManager.Run(ctx); err != nil {
		return fmt.Errorf("scenario error: %w", err)
	}

	return nil
}

// InitialSceneConfig holds configuration for starting with a pre-built scene
type InitialSceneConfig struct {
	Scene *scene.Scene
	NPCs  []*character.Character
}
