package engine

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
)

// Engine represents the core game engine
type Engine struct {
	actionParser    *ActionParser
	sceneManager    *SceneManager
	llmClient       llm.LLMClient
	characterRegistry map[string]*character.Character
}

// New creates a new game engine instance
func New() (*Engine, error) {
	engine := &Engine{
		characterRegistry: make(map[string]*character.Character),
	}
	engine.sceneManager = NewSceneManager(engine)
	return engine, nil
}

// NewWithLLM creates a new game engine instance with an LLM client
func NewWithLLM(llmClient llm.LLMClient) (*Engine, error) {
	engine := &Engine{
		llmClient:         llmClient,
		actionParser:      NewActionParser(llmClient),
		characterRegistry: make(map[string]*character.Character),
	}
	engine.sceneManager = NewSceneManager(engine)
	return engine, nil
}

// Start initializes the game engine
func (e *Engine) Start() error {
	return nil
}

// Stop shuts down the game engine
func (e *Engine) Stop() error {
	return nil
}

// GetVersion returns the engine version
func (e *Engine) GetVersion() string {
	return "0.1.0"
}

// GetActionParser returns the action parser instance
func (e *Engine) GetActionParser() *ActionParser {
	return e.actionParser
}

// GetSceneManager returns the scene manager instance
func (e *Engine) GetSceneManager() *SceneManager {
	return e.sceneManager
}

// AddCharacter adds a character to the registry
func (e *Engine) AddCharacter(char *character.Character) {
	e.characterRegistry[char.ID] = char
}

// GetCharacter retrieves a character from the registry
func (e *Engine) GetCharacter(id string) *character.Character {
	return e.characterRegistry[id]
}

// GetAllCharacters returns all characters in the registry
func (e *Engine) GetAllCharacters() map[string]*character.Character {
	return e.characterRegistry
}

// GetCharactersByScene returns characters that are present in a given scene
func (e *Engine) GetCharactersByScene(scene *scene.Scene) map[string]*character.Character {
	characters := make(map[string]*character.Character)
	for _, charID := range scene.Characters {
		if char, exists := e.characterRegistry[charID]; exists {
			characters[charID] = char
		}
	}
	return characters
}
