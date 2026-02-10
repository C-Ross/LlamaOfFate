package engine

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
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

// ScenarioState holds state managed by ScenarioManager and GameManager:
// the player character, scenario progress, NPC registry, and scene summaries.
type ScenarioState struct {
	Player         *character.Character            `yaml:"player"`
	Scenario       *scene.Scenario                 `yaml:"scenario"`
	ScenarioCount  int                             `yaml:"scenario_count"`
	SceneCount     int                             `yaml:"scene_count"`
	SceneSummaries []prompt.SceneSummary           `yaml:"scene_summaries"`
	NPCRegistry    map[string]*character.Character `yaml:"npc_registry"`
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
