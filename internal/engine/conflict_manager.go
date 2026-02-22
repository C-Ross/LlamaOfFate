package engine

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

// ConflictManager owns conflict lifecycle, NPC turns, damage resolution,
// taken-out/concession handling, and other combat-specific logic.
// It delegates generic action resolution (dice, invokes, mid-flow, narrative)
// to ActionResolver.
type ConflictManager struct {
	// Shared dependencies — set once at construction.
	llmClient     llm.LLMClient
	characters    CharacterResolver
	sessionLogger *session.Logger

	// ActionResolver — used for dice rolling, invoke loops, mid-flow prompts,
	// and narrative. Wired after construction by SceneManager.
	actions *ActionResolver

	// Per-scene state — wired by SceneManager.StartScene / resetConflictState.
	player       *character.Character
	currentScene *scene.Scene

	// Conflict-specific mutable state — reset each scene.
	takenOutChars []string

	// Scene-exit state — set by conflict methods (handleTakenOut), read by
	// SceneManager via accessors to build SceneEndResult.
	shouldExit            bool
	sceneEndReason        SceneEndReason
	playerTakenOutHint    string
	exitOnSceneTransition bool
}

// newConflictManager creates a ConflictManager sharing the given dependencies.
// The actions back-reference is wired separately after construction.
func newConflictManager(
	llmClient llm.LLMClient,
	characters CharacterResolver,
) *ConflictManager {
	return &ConflictManager{
		llmClient:  llmClient,
		characters: characters,
	}
}

// setSceneState wires per-scene references. Called by SceneManager.StartScene.
func (cm *ConflictManager) setSceneState(s *scene.Scene, player *character.Character) {
	cm.currentScene = s
	cm.player = player
}

// resetState clears per-scene conflict state. Called by SceneManager.resetSceneState.
func (cm *ConflictManager) resetState() {
	cm.takenOutChars = nil
	cm.shouldExit = false
	cm.sceneEndReason = ""
	cm.playerTakenOutHint = ""
}

// setSessionLogger updates the session logger (may be called after construction).
func (cm *ConflictManager) setSessionLogger(logger *session.Logger) {
	cm.sessionLogger = logger
}

// --- Accessor methods ---
// These encapsulate ConflictManager internal state so that SceneManager
// does not reach into struct fields directly.

// SceneExitRequested returns true when a conflict method (e.g. handleTakenOut)
// has signalled that the scene should end.
func (cm *ConflictManager) SceneExitRequested() bool {
	return cm.shouldExit
}

// SceneExitState returns the scene-end reason and transition hint set by
// conflict resolution. Only meaningful when SceneExitRequested() is true.
func (cm *ConflictManager) SceneExitState() (SceneEndReason, string) {
	return cm.sceneEndReason, cm.playerTakenOutHint
}

// GetTakenOutChars returns the IDs of characters taken out during this scene.
func (cm *ConflictManager) GetTakenOutChars() []string {
	return cm.takenOutChars
}

// conflictTypeString returns "physical" or "mental" based on the active conflict.
// Defaults to "physical" when no conflict is active.
func (cm *ConflictManager) conflictTypeString() string {
	if cm.currentScene.ConflictState != nil && cm.currentScene.ConflictState.Type == scene.MentalConflict {
		return "mental"
	}
	return "physical"
}
