package engine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

// GameManager orchestrates the overall game flow, managing scenarios and game state
type GameManager struct {
	engine          *Engine
	player          *character.Character
	ui              UI
	sessionLogger   *session.Logger
	scenario        *scene.Scenario  // The scenario to run (can be provided or generated)
	scenarioCount   int              // Number of scenarios completed
	saver           GameStateSaver   // Persistence interface (defaults to noopSaver)
	scenarioManager *ScenarioManager // Current scenario manager (stored for Save access)
}

// NewGameManager creates a new game manager
func NewGameManager(engine *Engine) *GameManager {
	return &GameManager{
		engine: engine,
		saver:  noopSaver{},
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

// SetScenario sets the scenario to run
func (g *GameManager) SetScenario(scenario *scene.Scenario) {
	g.scenario = scenario
}

// SetSaver sets the persistence implementation for saving and loading game state.
// If not called, a no-op saver is used and persistence is silently skipped.
func (g *GameManager) SetSaver(saver GameStateSaver) {
	if saver == nil {
		g.saver = noopSaver{}
		return
	}
	g.saver = saver
}

// Save persists the current game state by cascading through the manager hierarchy:
// GameManager → ScenarioManager.Snapshot() → SceneManager.Snapshot()
func (g *GameManager) Save() error {
	if g.scenarioManager == nil {
		return nil
	}
	scenarioState, sceneState := g.scenarioManager.Snapshot()
	return g.saver.Save(GameState{
		Scenario: scenarioState,
		Scene:    sceneState,
	})
}

// Run starts the game loop
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

	// Check for a saved game to resume
	savedState, err := g.saver.Load()
	if err != nil {
		slog.Warn("Failed to load saved game, starting fresh",
			"error", err,
		)
		savedState = nil
	}

	// Resume from save if a valid, unfinished game exists
	if savedState != nil && savedState.Scene.CurrentScene != nil {
		if savedState.Scenario.Scenario == nil || !savedState.Scenario.Scenario.IsResolved {
			return g.resumeFromSave(ctx, savedState)
		}
		slog.Info("Previous scenario was completed, starting fresh")
	}

	// Fresh start
	g.scenarioManager = NewScenarioManager(g.engine, g.player)
	g.scenarioManager.SetUI(g.ui)
	g.scenarioManager.SetScenario(g.scenario)
	g.scenarioManager.SetScenarioCount(g.scenarioCount)
	g.scenarioManager.SetSaveFunc(g.Save)
	if g.sessionLogger != nil {
		g.scenarioManager.SetSessionLogger(g.sessionLogger)
	}

	// Run the scenario
	result, err := g.scenarioManager.Run(ctx)
	if err != nil {
		return fmt.Errorf("scenario error: %w", err)
	}

	// Handle milestone if scenario was resolved
	if result != nil && result.Reason == ScenarioEndResolved {
		g.scenarioCount++
		g.handleMilestone()
	}

	return nil
}

// resumeFromSave restores game state from a saved snapshot and resumes
// the scene loop where the player left off.
func (g *GameManager) resumeFromSave(ctx context.Context, state *GameState) error {
	// Use the saved player (has updated stress, consequences, fate points, etc.)
	g.player = state.Scenario.Player

	// Create and configure ScenarioManager
	g.scenarioManager = NewScenarioManager(g.engine, g.player)
	g.scenarioManager.SetUI(g.ui)
	g.scenarioManager.SetSaveFunc(g.Save)
	if g.sessionLogger != nil {
		g.scenarioManager.SetSessionLogger(g.sessionLogger)
	}

	// Restore full state — this also restores the SceneManager and marks resumed
	g.scenarioManager.Restore(state.Scenario, state.Scene)

	slog.Info("Resuming saved game",
		"scene", state.Scene.CurrentScene.Name,
		"scenario", state.Scenario.Scenario.Title,
	)
	g.renderEvents([]GameEvent{
		SystemMessageEvent{Message: "=== Resuming saved game ==="},
		SystemMessageEvent{Message: fmt.Sprintf("Scenario: %s", state.Scenario.Scenario.Title)},
	})

	if g.sessionLogger != nil {
		g.sessionLogger.Log("game_resumed", map[string]any{
			"scene_name":     state.Scene.CurrentScene.Name,
			"scenario_title": state.Scenario.Scenario.Title,
			"scene_count":    state.Scenario.SceneCount,
		})
	}

	// Run the scenario — ScenarioManager.Run() will skip initial scene generation
	result, err := g.scenarioManager.Run(ctx)
	if err != nil {
		return fmt.Errorf("scenario error: %w", err)
	}

	if result != nil && result.Reason == ScenarioEndResolved {
		g.scenarioCount++
		g.handleMilestone()
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
	g.scenarioManager = NewScenarioManager(g.engine, g.player)
	g.scenarioManager.SetUI(g.ui)
	g.scenarioManager.SetScenario(g.scenario)
	g.scenarioManager.SetScenarioCount(g.scenarioCount)
	g.scenarioManager.SetInitialScene(initialScene.Scene, initialScene.NPCs)
	g.scenarioManager.SetSaveFunc(g.Save)
	if g.sessionLogger != nil {
		g.scenarioManager.SetSessionLogger(g.sessionLogger)
	}

	// Run the scenario
	result, err := g.scenarioManager.Run(ctx)
	if err != nil {
		return fmt.Errorf("scenario error: %w", err)
	}

	// Handle milestone if scenario was resolved
	if result != nil && result.Reason == ScenarioEndResolved {
		g.scenarioCount++
		g.handleMilestone()
	}

	return nil
}

// handleMilestone processes a scenario milestone (fate point refresh, consequence recovery, etc.)
func (g *GameManager) handleMilestone() {
	// Refresh fate points per Fate Core rules
	g.player.RefreshFatePoints()

	// Check for consequence recovery at scenario boundary
	// Moderate and severe consequences that are recovering clear after a whole scenario
	cleared := g.player.CheckConsequenceRecovery(0, g.scenarioCount)
	for _, conseq := range cleared {
		g.renderEvents([]GameEvent{SystemMessageEvent{Message: fmt.Sprintf(
			"Your %s consequence \"%s\" has fully healed!",
			conseq.Type, conseq.Aspect,
		)}})
		if g.sessionLogger != nil {
			g.sessionLogger.Log("consequence_healed", map[string]any{
				"type":      conseq.Type,
				"aspect":    conseq.Aspect,
				"healed_at": "scenario_milestone",
			})
		}
	}

	// Display milestone message
	g.renderEvents([]GameEvent{
		SystemMessageEvent{Message: "\n=== MILESTONE: Scenario Complete! ==="},
		SystemMessageEvent{Message: "Your fate points have been refreshed.\n"},
	})

	// Log the milestone
	if g.sessionLogger != nil {
		g.sessionLogger.Log("milestone", map[string]any{
			"type":           "scenario_complete",
			"fate_points":    g.player.FatePoints,
			"player":         g.player.Name,
			"scenario_title": g.scenario.Title,
		})
	}
}

// renderEvents dispatches events to the UI for display (terminal path).
func (g *GameManager) renderEvents(events []GameEvent) {
	renderEventsToUI(g.ui, events)
}

// InitialSceneConfig holds configuration for starting with a pre-built scene
type InitialSceneConfig struct {
	Scene *scene.Scene
	NPCs  []*character.Character
}
