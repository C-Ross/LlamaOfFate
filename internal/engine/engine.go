package engine

import (
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
)

// CharacterResolver provides read-only access to the character registry.
// Engine satisfies this interface — no new implementation code is needed.
type CharacterResolver interface {
	GetCharacter(id string) *character.Character
	ResolveCharacter(target string) *character.Character
	GetCharactersByScene(s *scene.Scene) map[string]*character.Character
}

// Engine represents the core game engine
type Engine struct {
	actionParser      ActionParser
	sceneManager      *SceneManager
	llmClient         llm.LLMClient
	characterRegistry map[string]*character.Character
}

// New creates a new game engine instance
func New() (*Engine, error) {
	engine := &Engine{
		characterRegistry: make(map[string]*character.Character),
	}
	engine.sceneManager = NewSceneManager(engine, nil, nil)
	return engine, nil
}

// NewWithLLM creates a new game engine instance with an LLM client
func NewWithLLM(llmClient llm.LLMClient) (*Engine, error) {
	engine := &Engine{
		llmClient:         llmClient,
		actionParser:      NewActionParser(llmClient),
		characterRegistry: make(map[string]*character.Character),
	}
	engine.sceneManager = NewSceneManager(engine, llmClient, engine.actionParser)
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
func (e *Engine) GetActionParser() ActionParser {
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

// GetCharacter retrieves a character from the registry by ID
func (e *Engine) GetCharacter(id string) *character.Character {
	return e.characterRegistry[id]
}

// GetCharacterByName retrieves a character from the registry by name.
// It performs a case-insensitive match against character names.
// Returns nil if no match is found.
func (e *Engine) GetCharacterByName(name string) *character.Character {
	lowerName := strings.ToLower(strings.TrimSpace(name))
	for _, char := range e.characterRegistry {
		if strings.ToLower(char.Name) == lowerName {
			return char
		}
	}
	return nil
}

// ResolveCharacter attempts to find a character using flexible matching.
// It handles the "Name (ID)" format that LLMs produce from prompt context,
// as well as plain ID or plain name lookups.
func (e *Engine) ResolveCharacter(target string) *character.Character {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}

	// Try exact ID lookup
	if char := e.GetCharacter(target); char != nil {
		return char
	}

	// Try exact name lookup
	if char := e.GetCharacterByName(target); char != nil {
		return char
	}

	// Try to extract ID from "Name (ID)" format
	if idx := strings.LastIndex(target, "("); idx > 0 {
		if end := strings.LastIndex(target, ")"); end > idx {
			extractedID := strings.TrimSpace(target[idx+1 : end])
			if char := e.GetCharacter(extractedID); char != nil {
				return char
			}
			// Also try extracted ID as a name
			if char := e.GetCharacterByName(extractedID); char != nil {
				return char
			}

			// Try the part before parentheses as a name
			extractedName := strings.TrimSpace(target[:idx])
			if char := e.GetCharacterByName(extractedName); char != nil {
				return char
			}
		}
	}

	return nil
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
