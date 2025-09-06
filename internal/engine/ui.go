package engine

import (
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// ConversationEntry represents a single exchange in the scene
type ConversationEntry struct {
	PlayerInput string    `json:"player_input"`
	GMResponse  string    `json:"gm_response"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // "dialog", "action", "clarification"
}

// SceneInfo provides access to scene information for display purposes
type SceneInfo interface {
	GetCurrentScene() *scene.Scene
	GetPlayer() *character.Character
	GetConversationHistory() []ConversationEntry
}

// UI defines the interface for user interaction during scene management
type UI interface {
	// Input methods - returns the cleaned input and whether it's an exit command
	ReadInput() (input string, isExit bool, err error)

	// Scene interaction feedback
	DisplayActionAttempt(description string)
	DisplayActionResult(skill string, skillLevel string, bonuses int, result string, outcome string)
	DisplayNarrative(narrative string)
	DisplayDialog(playerInput, gmResponse string)
	DisplaySystemMessage(message string)
}
