package engine

import (
	"context"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

// ConflictManager owns conflict resolution, NPC turns, invoke loops,
// and mid-flow prompts. SceneManager delegates to it and wraps the
// results with scene-level concerns (narrative, scene-end).
type ConflictManager struct {
	// Shared dependencies — set once at construction.
	llmClient       llm.LLMClient
	characters      CharacterResolver
	roller          dice.DiceRoller
	sessionLogger   *session.Logger
	aspectGenerator AspectGenerator

	// Per-scene state — wired by SceneManager.StartScene / resetConflictState.
	player       *character.Character
	currentScene *scene.Scene

	// Conflict-specific mutable state — reset each scene.
	pendingInvoke  *invokeState
	pendingMidFlow *midFlowState
	takenOutChars  []string

	// Scene-exit state — set by conflict methods, read by SceneManager to
	// build SceneEndResult. Moved here so conflict methods don't need a
	// back-pointer to SceneManager. Phase 4 will replace with return types.
	shouldExit            bool
	sceneEndReason        SceneEndReason
	playerTakenOutHint    string
	exitOnSceneTransition bool

	// Narrative callbacks — wired by SceneManager after construction so that
	// conflict methods can generate narrative and record conversation history
	// without a direct dependency on SceneManager.
	generateActionNarrative  func(ctx context.Context, a *action.Action) (string, error)
	buildMechanicalNarrative func(a *action.Action) string
	addToConversationHistory func(playerInput, gmResponse, interactionType string)
}

// newConflictManager creates a ConflictManager sharing the given dependencies.
// Narrative callbacks must be wired separately after construction.
func newConflictManager(
	llmClient llm.LLMClient,
	characters CharacterResolver,
	roller dice.DiceRoller,
	aspectGenerator AspectGenerator,
) *ConflictManager {
	return &ConflictManager{
		llmClient:       llmClient,
		characters:      characters,
		roller:          roller,
		aspectGenerator: aspectGenerator,
	}
}

// setSceneState wires per-scene references. Called by SceneManager.StartScene.
func (cm *ConflictManager) setSceneState(s *scene.Scene, player *character.Character) {
	cm.currentScene = s
	cm.player = player
}

// resetState clears per-scene conflict state. Called by SceneManager.resetSceneState.
func (cm *ConflictManager) resetState() {
	cm.pendingInvoke = nil
	cm.pendingMidFlow = nil
	cm.takenOutChars = nil
	cm.shouldExit = false
	cm.sceneEndReason = ""
	cm.playerTakenOutHint = ""
}

// setSessionLogger updates the session logger (may be called after construction).
func (cm *ConflictManager) setSessionLogger(logger *session.Logger) {
	cm.sessionLogger = logger
}
