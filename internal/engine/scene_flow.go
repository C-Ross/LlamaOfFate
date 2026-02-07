package engine

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// SceneEndReason indicates why a scene ended
type SceneEndReason string

const (
	// SceneEndTransition indicates the player moved to a new location/scene
	SceneEndTransition SceneEndReason = "transition"
	// SceneEndQuit indicates the player chose to quit
	SceneEndQuit SceneEndReason = "quit"
	// SceneEndPlayerTakenOut indicates the player was taken out
	SceneEndPlayerTakenOut SceneEndReason = "player_taken_out"
)

// SceneEndResult contains information about how and why a scene ended
type SceneEndResult struct {
	Reason         SceneEndReason
	TransitionHint string   // From [SCENE_TRANSITION:hint] marker, empty if not a transition
	TakenOutChars  []string // Character IDs taken out during the scene
}

// ScenarioEndReason indicates why a scenario ended
type ScenarioEndReason string

const (
	// ScenarioEndResolved indicates the scenario's story questions were answered
	ScenarioEndResolved ScenarioEndReason = "resolved"
	// ScenarioEndQuit indicates the player chose to quit
	ScenarioEndQuit ScenarioEndReason = "quit"
	// ScenarioEndPlayerTakenOut indicates the player was taken out permanently
	ScenarioEndPlayerTakenOut ScenarioEndReason = "player_taken_out"
)

// ScenarioResult contains information about how and why a scenario ended
type ScenarioResult struct {
	Reason   ScenarioEndReason
	Scenario *scene.Scenario // The scenario that was run
}
