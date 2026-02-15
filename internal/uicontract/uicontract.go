// Package uicontract defines the data types that form the contract between
// the game engine and any UI implementation.
//
// Both the engine and UI packages import this leaf package; neither imports
// the other, preserving inversion of control.
//
// Blocking UI interface (syncdriver.BlockingUI) lives in the syncdriver
// package, which wraps the engine's async API for terminal UIs.
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

// InputRequestType identifies what kind of mid-flow input the engine needs.
type InputRequestType string

const (
	// InputRequestNumberedChoice requests the player to select from a numbered list.
	InputRequestNumberedChoice InputRequestType = "numbered_choice"
	// InputRequestFreeText requests free-form text input from the player.
	InputRequestFreeText InputRequestType = "free_text"
)

// InputOption represents one selectable option in a numbered-choice request.
type InputOption struct {
	Label       string // Short label (e.g. "Take a mild consequence")
	Description string // Longer explanation (e.g. "absorbs 2 shifts")
}

// MidFlowResponse is the player's answer to an InputRequestEvent.
// For numbered_choice: ChoiceIndex is the 0-based index into InputRequestEvent.Options.
// For free_text: Text contains the player's input.
type MidFlowResponse struct {
	ChoiceIndex int    // 0-based option index (ignored for free_text)
	Text        string // Free-form text (ignored for numbered_choice)
}
