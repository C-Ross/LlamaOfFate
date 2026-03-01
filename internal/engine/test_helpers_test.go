package engine

import (
	"context"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
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
