package engine

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// ChallengeManager owns challenge lifecycle: initiation, task resolution,
// and completion. It delegates individual overcome rolls to ActionResolver.
type ChallengeManager struct {
	// Shared dependencies — set once at construction.
	llmClient          llm.LLMClient
	characters         CharacterResolver
	challengeGenerator ChallengeGenerator
	sessionLogger      session.SessionLogger

	// ActionResolver — used for dice rolling, invoke loops, mid-flow.
	// Wired after construction by SceneManager.
	actions *ActionResolver

	// Per-scene state — wired by SceneManager.StartScene.
	player       *character.Character
	currentScene *scene.Scene
}

// newChallengeManager creates a ChallengeManager sharing the given dependencies.
func newChallengeManager(
	llmClient llm.LLMClient,
	characters CharacterResolver,
	sessionLogger session.SessionLogger,
) *ChallengeManager {
	var cg ChallengeGenerator
	if llmClient != nil {
		cg = NewChallengeGenerator(llmClient)
	}
	return &ChallengeManager{
		llmClient:          llmClient,
		characters:         characters,
		challengeGenerator: cg,
		sessionLogger:      sessionLogger,
	}
}

// setSceneState wires per-scene references. Called by SceneManager.StartScene.
func (chm *ChallengeManager) setSceneState(s *scene.Scene, player *character.Character) {
	chm.currentScene = s
	chm.player = player
}

// resetState clears per-scene challenge state.
func (chm *ChallengeManager) resetState() {
	// No mutable state to clear beyond what's on the Scene itself.
}

// buildChallengeTaskInfos converts ChallengeState tasks to UI-friendly infos.
func buildChallengeTaskInfos(cs *scene.ChallengeState) []uicontract.ChallengeTaskInfo {
	infos := make([]uicontract.ChallengeTaskInfo, len(cs.Tasks))
	for i, t := range cs.Tasks {
		infos[i] = uicontract.ChallengeTaskInfo{
			ID:          t.ID,
			Description: t.Description,
			Skill:       t.Skill,
			Difficulty:  dice.Ladder(t.Difficulty).String(),
			Status:      string(t.Status),
		}
	}
	return infos
}
