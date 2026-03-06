package engine

import (
	"context"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/stretchr/testify/require"
)

// testLLMClient is a configurable mock LLM client that replaces the four
// separate mock types previously scattered across test files:
//
//   - MockLLMClient            → newTestLLMClient("response")
//   - MockLLMClientForScenario → newTestLLMClient()           (uses default scene JSON)
//   - capturingMockLLMClient   → newTestLLMClient("resp")     (check .capturedPrompts)
//   - sequentialMockLLMClient  → newTestLLMClient("a", "b")   (cycles through responses)
type testLLMClient struct {
	responses       []string // responses to cycle through; empty → use defaultSceneJSON
	err             error    // if set, ChatCompletion returns this error
	capturedPrompts []string // all message contents sent to ChatCompletion
	callIndex       int
}

// defaultSceneJSON is the fallback response when no responses are configured,
// matching the previous MockLLMClientForScenario behaviour.
const defaultSceneJSON = `{"scene_name": "Test Scene", "description": "A test scene.", "purpose": "Can the hero survive?", "situation_aspects": ["Aspect 1"], "npcs": []}`

func newTestLLMClient(responses ...string) *testLLMClient {
	return &testLLMClient{responses: responses}
}

func (m *testLLMClient) ChatCompletion(_ context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	// Capture every message for prompt-inspection tests.
	for _, msg := range req.Messages {
		m.capturedPrompts = append(m.capturedPrompts, msg.Content)
	}

	if m.err != nil {
		return nil, m.err
	}

	response := defaultSceneJSON
	if len(m.responses) > 0 {
		response = m.responses[m.callIndex%len(m.responses)]
		m.callIndex++
	}

	return &llm.CompletionResponse{
		ID:      "test-response",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "test-model",
		Choices: []llm.CompletionResponseChoice{
			{
				Index:        0,
				Message:      llm.Message{Role: "assistant", Content: response},
				FinishReason: "stop",
			},
		},
		Usage: llm.CompletionUsage{PromptTokens: 10, CompletionTokens: 10, TotalTokens: 20},
	}, nil
}

func (m *testLLMClient) ChatCompletionStream(_ context.Context, _ llm.CompletionRequest, _ llm.StreamHandler) error {
	return nil
}

func (m *testLLMClient) GetModelInfo() llm.ModelInfo {
	return llm.ModelInfo{
		Name:      "test-model",
		Provider:  "test",
		MaxTokens: 4096,
	}
}

// --- Unified SceneManager test setup ---

// smTestNPC configures an NPC for setupTestSM.
type smTestNPC struct {
	id          string
	name        string
	highConcept string
	skills      map[string]dice.Ladder
}

// smTestOpts configures setupTestSM. Zero values give sensible defaults.
type smTestOpts struct {
	llmResponses []string               // empty → no LLM client
	fatePoints   int                    // player fate points
	highConcept  string                 // player high concept
	trouble      string                 // player trouble
	skills       map[string]dice.Ladder // player skills (name → level)
	npc          *smTestNPC             // optional NPC
	conflictType scene.ConflictType     // non-empty → initiate conflict with NPC
}

// setupTestSM builds a SceneManager with a player (and optional NPC) for testing.
// When conflictType is set, the NPC is enrolled in a conflict.
// The second return is always the player; the third is the NPC (nil if none).
func setupTestSM(t *testing.T, opts smTestOpts) (*SceneManager, *core.Character, *core.Character) {
	t.Helper()

	var client *testLLMClient
	if len(opts.llmResponses) > 0 {
		client = newTestLLMClient(opts.llmResponses...)
	}

	var engine *Engine
	var err error
	if client != nil {
		engine, err = NewWithLLM(client, session.NullLogger{})
	} else {
		engine, err = New(session.NullLogger{})
	}
	require.NoError(t, err)

	// Player
	player := core.NewCharacter("player-1", "Hero")
	player.Aspects.HighConcept = opts.highConcept
	player.Aspects.Trouble = opts.trouble
	player.FatePoints = opts.fatePoints
	for skill, level := range opts.skills {
		player.SetSkill(skill, level)
	}
	engine.AddCharacter(player)

	// Scene
	testScene := scene.NewScene("test-scene", "Test Room", "A room for testing.")
	testScene.AddCharacter(player.ID)

	// NPC
	var npc *core.Character
	if opts.npc != nil {
		npc = core.NewCharacter(opts.npc.id, opts.npc.name)
		npc.Aspects.HighConcept = opts.npc.highConcept
		for skill, level := range opts.npc.skills {
			npc.SetSkill(skill, level)
		}
		engine.AddCharacter(npc)
		testScene.AddCharacter(npc.ID)
	}

	// Conflict path: manual wiring + initiateConflict
	if opts.conflictType != "" {
		sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
		sm.currentScene = testScene
		sm.conflict.currentScene = testScene
		sm.actions.currentScene = testScene
		sm.player = player
		sm.conflict.player = player
		sm.actions.player = player
		if npc != nil {
			err = sm.conflict.initiateConflict(opts.conflictType, npc.ID)
			require.NoError(t, err)
		}
		return sm, player, npc
	}

	// Normal path: public API
	sm := engine.GetSceneManager()
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)
	return sm, player, npc
}
