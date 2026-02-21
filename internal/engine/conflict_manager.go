package engine

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

// ConflictManager owns conflict-specific state and will (in a later phase)
// receive all conflict/NPC/invoke/midflow methods. For now it is a struct
// definition that SceneManager holds and wires per-scene.
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
}

// newConflictManager creates a ConflictManager sharing the given dependencies.
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
}

// setSessionLogger updates the session logger (may be called after construction).
func (cm *ConflictManager) setSessionLogger(logger *session.Logger) {
	cm.sessionLogger = logger
}
