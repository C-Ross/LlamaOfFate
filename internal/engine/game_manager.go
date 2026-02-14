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
	scenario        *scene.Scenario     // The scenario to run (can be provided or generated)
	initialScene    *InitialSceneConfig // Optional pre-configured starting scene (for demos/tests)
	scenarioCount   int                 // Number of scenarios completed
	saver           GameStateSaver      // Persistence interface (defaults to noopSaver)
	scenarioManager *ScenarioManager    // Current scenario manager (stored for Save access)
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

// Start initializes the game and returns the opening events. This is the async
// entry point for event-driven UIs (web). After Start, the caller drives the
// game by calling HandleInput for each player input.
//
// Start checks for saved games, configures the scenario manager, and delegates
// to ScenarioManager.Start. If resuming a saved game, a GameResumedEvent is
// prepended to the returned events.
func (g *GameManager) Start(ctx context.Context) ([]GameEvent, error) {
	if g.engine == nil {
		return nil, fmt.Errorf("engine is required")
	}
	if g.player == nil {
		return nil, fmt.Errorf("player character is required")
	}

	// Check for a saved game to resume (skip when an initial scene is
	// explicitly configured — RunWithInitialScene always starts fresh).
	var savedState *GameState
	if g.initialScene == nil {
		var err error
		savedState, err = g.saver.Load()
		if err != nil {
			slog.Warn("Failed to load saved game, starting fresh",
				"error", err,
			)
			savedState = nil
		}
	}

	// Resume from save if a valid, unfinished game exists
	if savedState != nil && savedState.Scene.CurrentScene != nil {
		if savedState.Scenario.Scenario == nil || !savedState.Scenario.Scenario.IsResolved {
			return g.startFromSave(ctx, savedState)
		}
		slog.Info("Previous scenario was completed, starting fresh")
	}

	// Fresh start
	g.scenarioManager = NewScenarioManager(g.engine, g.player)
	g.scenarioManager.SetScenario(g.scenario)
	g.scenarioManager.SetScenarioCount(g.scenarioCount)
	g.scenarioManager.SetSaveFunc(g.Save)
	if g.initialScene != nil {
		g.scenarioManager.SetInitialScene(g.initialScene.Scene, g.initialScene.NPCs)
		g.scenarioManager.SetExitAfterScene(g.initialScene.ExitAfterScene)
	}
	if g.sessionLogger != nil {
		g.scenarioManager.SetSessionLogger(g.sessionLogger)
	}

	return g.scenarioManager.Start(ctx)
}

// startFromSave restores game state and returns opening events with GameResumedEvent prepended.
func (g *GameManager) startFromSave(ctx context.Context, state *GameState) ([]GameEvent, error) {
	g.player = state.Scenario.Player

	g.scenarioManager = NewScenarioManager(g.engine, g.player)
	g.scenarioManager.SetSaveFunc(g.Save)
	if g.sessionLogger != nil {
		g.scenarioManager.SetSessionLogger(g.sessionLogger)
	}

	g.scenarioManager.Restore(state.Scenario, state.Scene)

	slog.Info("Resuming saved game",
		"scene", state.Scene.CurrentScene.Name,
		"scenario", state.Scenario.Scenario.Title,
	)

	if g.sessionLogger != nil {
		g.sessionLogger.Log("game_resumed", map[string]any{
			"scene_name":     state.Scene.CurrentScene.Name,
			"scenario_title": state.Scenario.Scenario.Title,
			"scene_count":    state.Scenario.SceneCount,
		})
	}

	scenarioEvents, err := g.scenarioManager.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("scenario start error: %w", err)
	}

	// Prepend the GameResumedEvent
	events := make([]GameEvent, 0, 1+len(scenarioEvents))
	events = append(events, GameResumedEvent{
		ScenarioTitle: state.Scenario.Scenario.Title,
		SceneName:     state.Scene.CurrentScene.Name,
	})
	events = append(events, scenarioEvents...)

	return events, nil
}

// HandleInput processes a single player input through the game manager layer.
// It delegates to ScenarioManager.HandleInput and, when a scenario ends with
// resolution, appends milestone events (fate point refresh, consequence recovery).
//
// This is the primary entry point for event-driven UIs (web via WebSocket).
// Terminal UIs use Run() which drives the game in a blocking loop.
func (g *GameManager) HandleInput(ctx context.Context, input string) (*InputResult, error) {
	if g.scenarioManager == nil {
		return nil, fmt.Errorf("HandleInput called before Start")
	}

	result, err := g.scenarioManager.HandleInput(ctx, input)
	if err != nil {
		return nil, err
	}

	// If the scenario resolved, append milestone events
	if result.GameOver && result.ScenarioResult != nil && result.ScenarioResult.Reason == ScenarioEndResolved {
		g.scenarioCount++
		milestoneEvents := g.handleMilestone()
		result.Events = append(result.Events, milestoneEvents...)
	}

	return result, nil
}

// ProvideInvokeResponse forwards an invoke response and handles milestone if scenario ends.
func (g *GameManager) ProvideInvokeResponse(ctx context.Context, resp InvokeResponse) (*InputResult, error) {
	if g.scenarioManager == nil {
		return nil, fmt.Errorf("ProvideInvokeResponse called before Start")
	}

	result, err := g.scenarioManager.ProvideInvokeResponse(ctx, resp)
	if err != nil {
		return nil, err
	}

	if result.GameOver && result.ScenarioResult != nil && result.ScenarioResult.Reason == ScenarioEndResolved {
		g.scenarioCount++
		result.Events = append(result.Events, g.handleMilestone()...)
	}

	return result, nil
}

// ProvideMidFlowResponse forwards a mid-flow response and handles milestone if scenario ends.
func (g *GameManager) ProvideMidFlowResponse(ctx context.Context, resp MidFlowResponse) (*InputResult, error) {
	if g.scenarioManager == nil {
		return nil, fmt.Errorf("ProvideMidFlowResponse called before Start")
	}

	result, err := g.scenarioManager.ProvideMidFlowResponse(ctx, resp)
	if err != nil {
		return nil, err
	}

	if result.GameOver && result.ScenarioResult != nil && result.ScenarioResult.Reason == ScenarioEndResolved {
		g.scenarioCount++
		result.Events = append(result.Events, g.handleMilestone()...)
	}

	return result, nil
}

// Run starts the game loop.
// This is the terminal-only blocking convenience. Event-driven UIs (web)
// should use Start + HandleInput instead.
//
// Run delegates to Start for initialization, then drives the game in a
// blocking loop: ReadInput → HandleInput → render events → repeat.
// Invoke and mid-flow prompts are resolved synchronously by type-asserting
// the UI to InvokePrompter / MidFlowPrompter.
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

	events, err := g.Start(ctx)
	if err != nil {
		return err
	}

	// Wire up SceneInfo so the UI can handle special commands (help, character, etc.)
	if setter, ok := g.ui.(SceneInfoSetter); ok {
		setter.SetSceneInfo(g.engine.GetSceneManager())
	}

	g.renderEvents(events)

	for {
		input, isExit, err := g.ui.ReadInput()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		if input == "" {
			continue
		}

		if isExit {
			if g.sessionLogger != nil {
				g.sessionLogger.Log("player_quit", nil)
			}
			_ = g.Save()
			return nil
		}

		result, err := g.HandleInput(ctx, input)
		if err != nil {
			return err
		}
		g.renderEvents(result.Events)

		// Resolve any pending invoke / mid-flow prompts via blocking UI
		result, err = g.driveBlockingPrompts(ctx, result)
		if err != nil {
			return err
		}

		if result.GameOver {
			return nil
		}
	}
}

// RunWithInitialScene runs the game starting from a pre-configured scene.
// This is useful for demos and testing specific scenarios.
func (g *GameManager) RunWithInitialScene(ctx context.Context, initialScene *InitialSceneConfig) error {
	if initialScene == nil {
		return fmt.Errorf("initial scene config is required")
	}
	g.initialScene = initialScene
	return g.Run(ctx)
}

// driveBlockingPrompts resolves pending invoke and mid-flow prompts in a
// synchronous loop for terminal UIs. It type-asserts the UI to the optional
// InvokePrompter / MidFlowPrompter interfaces, collects responses, and feeds
// them back via ProvideInvokeResponse / ProvideMidFlowResponse.
func (g *GameManager) driveBlockingPrompts(ctx context.Context, result *InputResult) (*InputResult, error) {
	for result.AwaitingInvoke {
		prompter, ok := g.ui.(InvokePrompter)
		if !ok {
			return nil, fmt.Errorf("UI does not implement InvokePrompter")
		}

		// Find the InvokePromptEvent in the last batch of events
		var prompt *InvokePromptEvent
		for i := len(result.Events) - 1; i >= 0; i-- {
			if p, ok := result.Events[i].(InvokePromptEvent); ok {
				prompt = &p
				break
			}
		}
		if prompt == nil {
			return nil, fmt.Errorf("AwaitingInvoke set but no InvokePromptEvent in events")
		}

		resp := prompter.PromptForInvoke(prompt.Available, prompt.FatePoints, prompt.CurrentResult, prompt.ShiftsNeeded)

		var err error
		result, err = g.ProvideInvokeResponse(ctx, resp)
		if err != nil {
			return nil, err
		}
		g.renderEvents(result.Events)
	}

	for result.AwaitingMidFlow {
		prompter, ok := g.ui.(MidFlowPrompter)
		if !ok {
			return nil, fmt.Errorf("UI does not implement MidFlowPrompter")
		}

		var prompt *InputRequestEvent
		for i := len(result.Events) - 1; i >= 0; i-- {
			if p, ok := result.Events[i].(InputRequestEvent); ok {
				prompt = &p
				break
			}
		}
		if prompt == nil {
			return nil, fmt.Errorf("AwaitingMidFlow set but no InputRequestEvent in events")
		}

		resp := prompter.PromptForMidFlow(*prompt)

		var err error
		result, err = g.ProvideMidFlowResponse(ctx, resp)
		if err != nil {
			return nil, err
		}
		g.renderEvents(result.Events)
	}

	return result, nil
}

// handleMilestone processes a scenario milestone and returns the events
// for the caller to render. Performs fate point refresh and consequence
// recovery per Fate Core rules.
func (g *GameManager) handleMilestone() []GameEvent {
	// Refresh fate points per Fate Core rules
	g.player.RefreshFatePoints()

	var events []GameEvent

	// Check for consequence recovery at scenario boundary
	// Moderate and severe consequences that are recovering clear after a whole scenario
	cleared := g.player.CheckConsequenceRecovery(0, g.scenarioCount)
	for _, conseq := range cleared {
		events = append(events, RecoveryEvent{
			Action:   "healed",
			Severity: string(conseq.Type),
			Aspect:   conseq.Aspect,
		})
		if g.sessionLogger != nil {
			g.sessionLogger.Log("consequence_healed", map[string]any{
				"type":      conseq.Type,
				"aspect":    conseq.Aspect,
				"healed_at": "scenario_milestone",
			})
		}
	}

	// Milestone event
	scenarioTitle := ""
	if g.scenario != nil {
		scenarioTitle = g.scenario.Title
	}
	events = append(events, MilestoneEvent{
		Type:          "scenario_complete",
		ScenarioTitle: scenarioTitle,
		FatePoints:    g.player.FatePoints,
	})

	// Log the milestone
	if g.sessionLogger != nil {
		g.sessionLogger.Log("milestone", map[string]any{
			"type":           "scenario_complete",
			"fate_points":    g.player.FatePoints,
			"player":         g.player.Name,
			"scenario_title": scenarioTitle,
		})
	}

	return events
}

// renderEvents dispatches events to the UI for display (terminal path).
func (g *GameManager) renderEvents(events []GameEvent) {
	renderEventsToUI(g.ui, events)
}

// InitialSceneConfig holds configuration for starting with a pre-built scene
type InitialSceneConfig struct {
	Scene          *scene.Scene
	NPCs           []*character.Character
	ExitAfterScene bool // Exit the game after the initial scene ends instead of generating next scenes
}
