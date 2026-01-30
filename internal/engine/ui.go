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

// ConflictParticipantInfo provides display information about a conflict participant
type ConflictParticipantInfo struct {
	CharacterID   string
	CharacterName string
	Initiative    int
	IsPlayer      bool
}

// InvokableAspect represents an aspect available for invocation
type InvokableAspect struct {
	Name        string // The aspect text
	Source      string // "character", "situation", "consequence"
	SourceID    string // ID of the source (character ID, aspect ID, etc.)
	FreeInvokes int    // Number of free invokes available (0 = requires fate point)
	AlreadyUsed bool   // True if already invoked on this roll
}

// InvokeChoice represents the player's invoke decision
type InvokeChoice struct {
	Aspect   *InvokableAspect // nil if player chose to skip
	UseFree  bool             // true = use free invoke, false = spend fate point
	IsReroll bool             // true = reroll dice, false = +2 bonus
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

	// Invoke methods
	PromptForInvoke(available []InvokableAspect, fatePoints int, currentResult string, shiftsNeeded int) *InvokeChoice

	// Conflict display methods
	DisplayConflictStart(conflictType string, initiatorName string, participants []ConflictParticipantInfo)
	DisplayConflictEscalation(fromType, toType, triggerCharName string)
	DisplayTurnAnnouncement(characterName string, turnNumber int, isPlayer bool)
	DisplayConflictEnd(reason string)

	// Game flow methods
	DisplayGameOver(reason string)
	DisplaySceneTransition(narrative string, newSceneHint string)
	DisplayCharacter()
}
