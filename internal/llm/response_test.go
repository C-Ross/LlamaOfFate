package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain JSON",
			input:    `{"action_type": "Overcome", "skill": "Athletics"}`,
			expected: `{"action_type": "Overcome", "skill": "Athletics"}`,
		},
		{
			name:     "JSON with markdown code block",
			input:    "```json\n{\"action_type\": \"Overcome\", \"skill\": \"Athletics\"}\n```",
			expected: `{"action_type": "Overcome", "skill": "Athletics"}`,
		},
		{
			name:     "JSON with generic code block",
			input:    "```\n{\"action_type\": \"Overcome\", \"skill\": \"Athletics\"}\n```",
			expected: `{"action_type": "Overcome", "skill": "Athletics"}`,
		},
		{
			name:     "JSON with extra whitespace",
			input:    "  \n  {\"action_type\": \"Overcome\", \"skill\": \"Athletics\"}  \n  ",
			expected: `{"action_type": "Overcome", "skill": "Athletics"}`,
		},
		{
			name:     "Multiple JSON blocks - should take last one",
			input:    "```json\n{\"action_type\": \"Investigate\", \"skill\": \"Investigate\"}\n```\n\nCorrected to match the exact action type:\n\n```json\n{\"action_type\": \"Create an Advantage\", \"skill\": \"Investigate\"}\n```",
			expected: `{"action_type": "Create an Advantage", "skill": "Investigate"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := CleanJSONResponse(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}
func TestSimpleCompletion_Success(t *testing.T) {
	mock := &MockLLMClient{
		chatCompletionFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			assert.Len(t, req.Messages, 1)
			assert.Equal(t, "user", req.Messages[0].Role)
			assert.Equal(t, "hello", req.Messages[0].Content)
			assert.Equal(t, 100, req.MaxTokens)
			assert.Equal(t, 0.5, req.Temperature)
			return &CompletionResponse{
				Choices: []CompletionResponseChoice{
					{Message: Message{Role: "assistant", Content: "world"}},
				},
			}, nil
		},
	}

	content, err := SimpleCompletion(context.Background(), mock, "hello", 100, 0.5)
	require.NoError(t, err)
	assert.Equal(t, "world", content)
}

func TestSimpleCompletion_EmptyResponse(t *testing.T) {
	mock := &MockLLMClient{
		chatCompletionFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{Choices: []CompletionResponseChoice{}}, nil
		},
	}

	_, err := SimpleCompletion(context.Background(), mock, "hello", 100, 0.5)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrEmptyResponse))
}

func TestSimpleCompletion_LLMError(t *testing.T) {
	mock := &MockLLMClient{
		chatCompletionFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			return nil, errors.New("connection refused")
		},
	}

	_, err := SimpleCompletion(context.Background(), mock, "hello", 100, 0.5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}
