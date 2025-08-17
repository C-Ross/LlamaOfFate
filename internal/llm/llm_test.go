package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "Hello, world!",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello, world!", msg.Content)
}

func TestCompletionRequest(t *testing.T) {
	req := CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
		MaxTokens:        100,
		Temperature:      0.7,
		TopP:             0.1,
		PresencePenalty:  0.5,
		FrequencyPenalty: 0.3,
		Model:            "test-model",
		Stream:           false,
	}

	assert.Len(t, req.Messages, 2)
	assert.Equal(t, "system", req.Messages[0].Role)
	assert.Equal(t, "user", req.Messages[1].Role)
	assert.Equal(t, 100, req.MaxTokens)
	assert.Equal(t, 0.7, req.Temperature)
	assert.Equal(t, 0.1, req.TopP)
	assert.Equal(t, 0.5, req.PresencePenalty)
	assert.Equal(t, 0.3, req.FrequencyPenalty)
	assert.Equal(t, "test-model", req.Model)
	assert.False(t, req.Stream)
}

func TestCompletionResponse(t *testing.T) {
	resp := CompletionResponse{
		ID:      "test-id",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "Meta-Llama-3.1-405B-Instruct",
		Choices: []CompletionResponseChoice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "Hello! How can I help you today?",
				},
				FinishReason: "stop",
			},
		},
		Usage: CompletionUsage{
			PromptTokens:     10,
			CompletionTokens: 8,
			TotalTokens:      18,
		},
	}

	assert.Equal(t, "test-id", resp.ID)
	assert.Equal(t, "chat.completion", resp.Object)
	assert.Equal(t, int64(1234567890), resp.Created)
	assert.Equal(t, "Meta-Llama-3.1-405B-Instruct", resp.Model)
	assert.Len(t, resp.Choices, 1)
	assert.Equal(t, 0, resp.Choices[0].Index)
	assert.Equal(t, "assistant", resp.Choices[0].Message.Role)
	assert.Equal(t, "Hello! How can I help you today?", resp.Choices[0].Message.Content)
	assert.Equal(t, "stop", resp.Choices[0].FinishReason)
	assert.Equal(t, 10, resp.Usage.PromptTokens)
	assert.Equal(t, 8, resp.Usage.CompletionTokens)
	assert.Equal(t, 18, resp.Usage.TotalTokens)
}

func TestModelInfo(t *testing.T) {
	info := ModelInfo{
		Name:        "Meta-Llama-3.1-405B-Instruct",
		Provider:    "Azure ML",
		MaxTokens:   4096,
		Description: "Azure ML hosted Meta-Llama-3.1-405B-Instruct",
	}

	assert.Equal(t, "Meta-Llama-3.1-405B-Instruct", info.Name)
	assert.Equal(t, "Azure ML", info.Provider)
	assert.Equal(t, 4096, info.MaxTokens)
	assert.Equal(t, "Azure ML hosted Meta-Llama-3.1-405B-Instruct", info.Description)
}
