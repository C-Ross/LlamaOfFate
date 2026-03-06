package engine

import (
	"fmt"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
)

// GameState aggregates all saveable game state for persistence.
// It is composed of two layers matching the manager hierarchy:
// ScenarioState (scenario-level) and SceneState (scene-level).
type GameState struct {
	Scenario ScenarioState `yaml:"scenario"`
	Scene    SceneState    `yaml:"scene"`
}

// Validate checks that a loaded GameState contains the minimum required data
// for resuming a game. It returns an error describing all missing fields if
// the state is incomplete (e.g. from an incompatible older save format).
func (gs *GameState) Validate() error {
	var problems []string

	if gs.Scenario.Player == nil {
		problems = append(problems, "missing player")
	} else {
		p := gs.Scenario.Player
		if p.Name == "" {
			problems = append(problems, "player has no name")
		}
		if p.Aspects.HighConcept == "" {
			problems = append(problems, "player has no high concept")
		}
		if p.Aspects.Trouble == "" {
			problems = append(problems, "player has no trouble")
		}
		if len(p.StressTracks) == 0 {
			problems = append(problems, "player has no stress tracks")
		}
	}

	if gs.Scenario.Scenario == nil {
		problems = append(problems, "missing scenario")
	}

	if gs.Scene.CurrentScene == nil {
		problems = append(problems, "missing current scene")
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid save state: %s", strings.Join(problems, "; "))
	}
	return nil
}

// ScenarioState holds state managed by ScenarioManager and GameManager:
// the player character, scenario progress, NPC registry, and scene summaries.
type ScenarioState struct {
	Player         *core.Character            `yaml:"player"`
	Scenario       *scene.Scenario                 `yaml:"scenario"`
	ScenarioCount  int                             `yaml:"scenario_count"`
	SceneCount     int                             `yaml:"scene_count"`
	SceneSummaries []prompt.SceneSummary           `yaml:"scene_summaries"`
	NPCRegistry    map[string]*core.Character `yaml:"npc_registry"`
	NPCAttitudes   map[string]string               `yaml:"npc_attitudes"`
	LastPurpose    string                          `yaml:"last_generated_purpose,omitempty"`
	LastHook       string                          `yaml:"last_generated_hook,omitempty"`
}

// SceneState holds state managed by SceneManager:
// the current scene (including any active conflict), conversation history,
// and the scene's dramatic purpose.
type SceneState struct {
	CurrentScene        *scene.Scene               `yaml:"current_scene"`
	ConversationHistory []prompt.ConversationEntry `yaml:"conversation_history"`
	ScenePurpose        string                     `yaml:"scene_purpose,omitempty"`
}
