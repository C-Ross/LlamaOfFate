package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/ui/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLMClient returns canned responses for testing.
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) ChatCompletion(_ context.Context, _ llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.CompletionResponse{
		ID:      "test",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "test-model",
		Choices: []llm.CompletionResponseChoice{
			{
				Index:        0,
				Message:      llm.Message{Role: "assistant", Content: m.response},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (m *mockLLMClient) ChatCompletionStream(_ context.Context, _ llm.CompletionRequest, _ llm.StreamHandler) error {
	return fmt.Errorf("not implemented")
}

func (m *mockLLMClient) GetModelInfo() llm.ModelInfo {
	return llm.ModelInfo{Name: "test-model", Provider: "test"}
}

const validScenarioJSON = `{
	"title": "Neon Shadows",
	"problem": "A rogue AI threatens the city's power grid",
	"story_questions": [
		"Can the hacker stop the AI before the blackout?",
		"Will the megacorp cover up the truth?"
	],
	"setting": "A sprawling cyberpunk metropolis of neon and chrome",
	"genre": "Cyberpunk"
}`

func TestGenerateScenario_Success(t *testing.T) {
	client := &mockLLMClient{response: validScenarioJSON}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	scenario, err := generateScenario(context.Background(), client, custom)
	require.NoError(t, err)
	assert.Equal(t, "Neon Shadows", scenario.Title)
	assert.Equal(t, "A rogue AI threatens the city's power grid", scenario.Problem)
	assert.Equal(t, "Cyberpunk", scenario.Genre)
	assert.Len(t, scenario.StoryQuestions, 2)
}

func TestGenerateScenario_LLMError(t *testing.T) {
	client := &mockLLMClient{err: fmt.Errorf("service unavailable")}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	_, err := generateScenario(context.Background(), client, custom)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM completion")
}

func TestGenerateScenario_InvalidJSON(t *testing.T) {
	client := &mockLLMClient{response: "this is not json at all"}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	_, err := generateScenario(context.Background(), client, custom)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse scenario")
}

func TestGenerateScenario_MissingTitle(t *testing.T) {
	client := &mockLLMClient{response: `{"problem": "something bad"}`}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	_, err := generateScenario(context.Background(), client, custom)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse scenario")
}

func TestGenerateScenario_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := &mockLLMClient{err: ctx.Err()}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	_, err := generateScenario(ctx, client, custom)
	require.Error(t, err)
}

func TestGenerateScenario_MarkdownWrappedJSON(t *testing.T) {
	// LLMs sometimes wrap JSON in markdown code fences
	wrapped := "```json\n" + validScenarioJSON + "\n```"
	client := &mockLLMClient{response: wrapped}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	scenario, err := generateScenario(context.Background(), client, custom)
	require.NoError(t, err)
	assert.Equal(t, "Neon Shadows", scenario.Title)
}
