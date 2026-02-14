// Package uicontract defines the interface and data types that form the
// contract between the game engine and any UI implementation.
//
// Both the engine and UI packages import this leaf package; neither imports
// the other, preserving inversion of control.
package uicontract

import (
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// SceneInfo provides read-only access to scene state for display purposes.
type SceneInfo interface {
	GetCurrentScene() *scene.Scene
	GetPlayer() *character.Character
	GetConversationHistory() []ConversationEntry
}

// SceneInfoSetter is an optional interface that UIs can implement to receive
// a reference back to the SceneInfo provider (e.g., for character/status display).
type SceneInfoSetter interface {
	SetSceneInfo(sceneInfo SceneInfo)
}

// UI defines the interface for user interaction during scene management.
type UI interface {
	// Input methods - returns the cleaned input and whether it's an exit command
	ReadInput() (input string, isExit bool, err error)

	// Scene interaction feedback
	DisplayActionAttempt(description string)
	DisplayActionResult(skill string, skillLevel string, bonuses int, result string, outcome string)
	DisplayNarrative(narrative string)
	DisplayDialog(playerInput, gmResponse string)
	DisplaySystemMessage(message string)

	// Invoke methods — used by the synchronous terminal path (RunSceneLoop).
	// Event-driven UIs (web) use InvokePromptEvent/InvokeResponse instead.
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

// ConflictParticipantInfo provides display information about a conflict participant.
type ConflictParticipantInfo struct {
	CharacterID   string
	CharacterName string
	Initiative    int
	IsPlayer      bool
}

// InvokableAspect represents an aspect available for invocation.
type InvokableAspect struct {
	Name        string // The aspect text
	Source      string // "character", "situation", "consequence"
	SourceID    string // ID of the source (character ID, aspect ID, etc.)
	FreeInvokes int    // Number of free invokes available (0 = requires fate point)
	AlreadyUsed bool   // True if already invoked on this roll
}

// InvokeChoice represents the player's invoke decision.
type InvokeChoice struct {
	Aspect   *InvokableAspect // nil if player chose to skip
	UseFree  bool             // true = use free invoke, false = spend fate point
	IsReroll bool             // true = reroll dice, false = +2 bonus
}

const (
	// InvokeSkip means the player chose not to invoke any aspect.
	InvokeSkip = -1
)

// InvokeResponse is the player's response to an InvokePromptEvent.
// AspectIndex indexes into InvokePromptEvent.Available; InvokeSkip means skip.
type InvokeResponse struct {
	AspectIndex int  // InvokeSkip = skip, 0..N = selected aspect index
	IsReroll    bool // true = reroll dice, false = +2 bonus
}

// ConversationEntry represents a single exchange in the conversation history.
type ConversationEntry struct {
	PlayerInput string    `json:"player_input"`
	GMResponse  string    `json:"gm_response"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // "dialog", "action", "clarification"
}
