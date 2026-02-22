package engine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

// GameSessionManager is the interface for driving a game session. It exposes
// the async/event-driven API that callers (syncdriver, web handler, tests) use
// to run a game: Start returns opening events, then HandleInput /
// ProvideInvokeResponse / ProvideMidFlowResponse each return an InputResult
// with events. Save persists the current game state.
type GameSessionManager interface {
	Start(ctx context.Context) ([]GameEvent, error)
	HandleInput(ctx context.Context, input string) (*InputResult, error)
	ProvideInvokeResponse(ctx context.Context, resp InvokeResponse) (*InputResult, error)
	ProvideMidFlowResponse(ctx context.Context, resp MidFlowResponse) (*InputResult, error)
	Save() error
}

// Compile-time check: *GameManager satisfies GameSessionManager.
var _ GameSessionManager = (*GameManager)(nil)

// GameManager orchestrates the overall game flow, managing scenarios and game state.
// It exposes a purely async/event-driven API: Start returns opening events, then
// HandleInput / ProvideInvokeResponse / ProvideMidFlowResponse each return an
// InputResult with events. The caller (syncdriver, web handler, test) decides
// how to render events and collect input.
type GameManager struct {
	engine          *Engine
	player          *character.Character
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

// SetSessionLogger sets the session logger
func (g *GameManager) SetSessionLogger(logger *session.Logger) {
	g.sessionLogger = logger
}

// SetScenario sets the scenario to run
func (g *GameManager) SetScenario(scenario *scene.Scenario) {
	g.scenario = scenario
}

// SetInitialScene configures a pre-built starting scene. When set, Start will
// use this scene instead of generating one via LLM. Useful for demos and tests.
func (g *GameManager) SetInitialScene(config *InitialSceneConfig) {
	g.initialScene = config
}

// GetEngine returns the underlying Engine so callers can access SceneInfo,
// the character registry, etc.
func (g *GameManager) GetEngine() *Engine {
	return g.engine
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

// Start initializes the game and returns the opening events. After Start, the
// caller drives the game by calling HandleInput for each player input.
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
	var loadError error
	if g.initialScene == nil {
		var err error
		savedState, err = g.saver.Load()
		if err != nil {
			slog.Warn("Failed to load saved game, starting fresh",
				"error", err,
			)
			loadError = err
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

	events, err := g.scenarioManager.Start(ctx)
	if err != nil {
		return nil, err
	}

	// Notify the player if their saved game could not be loaded.
	if loadError != nil {
		notification := ErrorNotificationEvent{
			Message: "Your saved game could not be loaded and a new game has been started.",
		}
		events = append([]GameEvent{notification}, events...)
	}

	// Append a full-state snapshot so web UIs can initialise sidebar state.
	events = append(events, g.buildStateSnapshot())

	return events, nil
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

	// Append a full-state snapshot so web UIs can initialise sidebar state.
	events = append(events, g.buildStateSnapshot())

	return events, nil
}

// HandleInput processes a single player input through the game manager layer.
// It delegates to ScenarioManager.HandleInput and, when a scenario ends with
// resolution, appends milestone events (fate point refresh, consequence recovery).
func (g *GameManager) HandleInput(ctx context.Context, input string) (*InputResult, error) {
	if g.scenarioManager == nil {
		return nil, fmt.Errorf("HandleInput called before Start")
	}

	result, err := g.scenarioManager.HandleInput(ctx, input)
	if err != nil {
		return nil, err
	}

	g.appendPostInputSnapshot(result)

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

	g.appendPostInputSnapshot(result)

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

	g.appendPostInputSnapshot(result)

	return result, nil
}

// appendPostInputSnapshot appends milestone events when a scenario resolves and
// a GameStateSnapshotEvent whenever a scene transition occurred (so the web
// sidebar refreshes NPCs, stress, aspects, etc.).
func (g *GameManager) appendPostInputSnapshot(result *InputResult) {
	// Scenario resolution → milestone
	if result.GameOver && result.ScenarioResult != nil && result.ScenarioResult.Reason == ScenarioEndResolved {
		g.scenarioCount++
		result.Events = append(result.Events, g.handleMilestone()...)
	}

	// Scene transition (new scene started) → append fresh snapshot
	if result.SceneEnded && !result.GameOver {
		result.Events = append(result.Events, g.buildStateSnapshot())
	}
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

// buildStateSnapshot creates a GameStateSnapshotEvent from the current engine
// state.  It is safe to call after ScenarioManager.Start has returned.
func (g *GameManager) buildStateSnapshot() GameStateSnapshotEvent {
	snap := GameStateSnapshotEvent{}

	// Player
	if g.player != nil {
		snap.Player = PlayerSnapshot{
			Name:        g.player.Name,
			HighConcept: g.player.Aspects.HighConcept,
			Trouble:     g.player.Aspects.Trouble,
			Aspects:     g.player.Aspects.OtherAspects,
			FatePoints:  g.player.FatePoints,
			Refresh:     g.player.Refresh,
		}

		// Stress tracks
		tracks := make(map[string]StressTrackSnapshot, len(g.player.StressTracks))
		for name, st := range g.player.StressTracks {
			boxes := make([]bool, len(st.Boxes))
			copy(boxes, st.Boxes)
			tracks[name] = StressTrackSnapshot{
				Boxes:    boxes,
				MaxBoxes: st.MaxBoxes,
			}
		}
		snap.Player.StressTracks = tracks

		// Consequences
		for _, c := range g.player.Consequences {
			snap.Player.Consequences = append(snap.Player.Consequences, ConsequenceSnapshotEntry{
				Severity:   string(c.Type),
				Aspect:     c.Aspect,
				Recovering: c.Recovering,
			})
		}
	}

	// Scene
	currentScene := g.engine.GetSceneManager().GetCurrentScene()
	if currentScene != nil {
		snap.SceneName = currentScene.Name
		snap.InConflict = currentScene.IsConflict

		for _, sa := range currentScene.SituationAspects {
			snap.SituationAspects = append(snap.SituationAspects, SituationAspectSnapshot{
				Name:        sa.Aspect,
				FreeInvokes: sa.FreeInvokes,
				IsBoost:     sa.IsBoost,
			})
		}

		// NPCs in the scene
		npcs := g.engine.GetCharactersByScene(currentScene)
		for _, npc := range npcs {
			if npc.ID == g.player.ID {
				continue // skip the player
			}
			npcSnap := NPCSnapshot{
				Name:        npc.Name,
				HighConcept: npc.Aspects.HighConcept,
				Aspects:     npc.Aspects.OtherAspects,
				IsTakenOut:  npc.IsTakenOut(),
			}
			snap.NPCs = append(snap.NPCs, npcSnap)
		}
	}

	return snap
}

// InitialSceneConfig holds configuration for starting with a pre-built scene
type InitialSceneConfig struct {
	Scene          *scene.Scene
	NPCs           []*character.Character
	ExitAfterScene bool // Exit the game after the initial scene ends instead of generating next scenes
}
