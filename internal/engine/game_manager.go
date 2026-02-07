package engine

import (
	"context"
	"fmt"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

// GameManager orchestrates the overall game flow, managing scenarios and game state
type GameManager struct {
	engine        *Engine
	player        *character.Character
	ui            UI
	sessionLogger *session.Logger
	scenario      *Scenario // The scenario to run (can be provided or generated)
	scenarioCount int       // Number of scenarios completed
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

// SetScenario sets the scenario to run
func (g *GameManager) SetScenario(scenario *Scenario) {
	g.scenario = scenario
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

	// Create and configure the scenario manager
	scenarioManager := NewScenarioManager(g.engine, g.player)
	scenarioManager.SetUI(g.ui)
	scenarioManager.SetScenario(g.scenario)
	scenarioManager.SetScenarioCount(g.scenarioCount)
	if g.sessionLogger != nil {
		scenarioManager.SetSessionLogger(g.sessionLogger)
	}

	// Run the scenario
	result, err := scenarioManager.Run(ctx)
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
	scenarioManager.SetScenario(g.scenario)
	scenarioManager.SetScenarioCount(g.scenarioCount)
	scenarioManager.SetInitialScene(initialScene.Scene, initialScene.NPCs)
	if g.sessionLogger != nil {
		scenarioManager.SetSessionLogger(g.sessionLogger)
	}

	// Run the scenario
	result, err := scenarioManager.Run(ctx)
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
		g.ui.DisplaySystemMessage(fmt.Sprintf(
			"Your %s consequence \"%s\" has fully healed!",
			conseq.Type, conseq.Aspect,
		))
		if g.sessionLogger != nil {
			g.sessionLogger.Log("consequence_healed", map[string]any{
				"type":      conseq.Type,
				"aspect":    conseq.Aspect,
				"healed_at": "scenario_milestone",
			})
		}
	}

	// Display milestone message
	g.ui.DisplaySystemMessage("\n=== MILESTONE: Scenario Complete! ===")
	g.ui.DisplaySystemMessage("Your fate points have been refreshed.\n")

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

// InitialSceneConfig holds configuration for starting with a pre-built scene
type InitialSceneConfig struct {
	Scene *scene.Scene
	NPCs  []*character.Character
}
