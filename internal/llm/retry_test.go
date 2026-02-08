package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMClient is a mock implementation of LLMClient for testing
type MockLLMClient struct {
	chatCompletionFunc       func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	chatCompletionStreamFunc func(ctx context.Context, req CompletionRequest, handler StreamHandler) error
	modelInfo                ModelInfo
}

func (m *MockLLMClient) ChatCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if m.chatCompletionFunc != nil {
		return m.chatCompletionFunc(ctx, req)
	}
	return &CompletionResponse{
		ID:     "test-id",
		Model:  "test-model",
		Object: "chat.completion",
		Choices: []CompletionResponseChoice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: "test response",
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (m *MockLLMClient) ChatCompletionStream(ctx context.Context, req CompletionRequest, handler StreamHandler) error {
	if m.chatCompletionStreamFunc != nil {
		return m.chatCompletionStreamFunc(ctx, req, handler)
	}
	return nil
}

func (m *MockLLMClient) GetModelInfo() ModelInfo {
	if m.modelInfo.Name != "" {
		return m.modelInfo
	}
	return ModelInfo{
		Name:     "test-model",
		Provider: "test-provider",
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.InitialBackoff)
	assert.Equal(t, 10*time.Second, config.MaxBackoff)
	assert.Equal(t, 2.0, config.BackoffFactor)
}

func TestNewRetryingClient(t *testing.T) {
	mockClient := &MockLLMClient{}
	config := DefaultRetryConfig()

	retryClient := NewRetryingClient(mockClient, config)

	assert.NotNil(t, retryClient)
	assert.Equal(t, mockClient, retryClient.client)
	assert.Equal(t, config.MaxAttempts, retryClient.config.MaxAttempts)
}

func TestNewRetryingClientWithDefaults(t *testing.T) {
	mockClient := &MockLLMClient{}
	config := RetryConfig{} // Empty config

	retryClient := NewRetryingClient(mockClient, config)

	assert.NotNil(t, retryClient)
	assert.Equal(t, 3, retryClient.config.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, retryClient.config.InitialBackoff)
	assert.Equal(t, 10*time.Second, retryClient.config.MaxBackoff)
	assert.Equal(t, 2.0, retryClient.config.BackoffFactor)
}

func TestRetryingClient_ChatCompletion_Success(t *testing.T) {
	mockClient := &MockLLMClient{}
	config := DefaultRetryConfig()
	retryClient := NewRetryingClient(mockClient, config)

	ctx := context.Background()
	req := CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
	}

	resp, err := retryClient.ChatCompletion(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "test-id", resp.ID)
	assert.Equal(t, "test response", resp.Content())
}

func TestRetryingClient_ChatCompletion_RetriesOnNetworkError(t *testing.T) {
	attempts := 0
	mockClient := &MockLLMClient{
		chatCompletionFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			attempts++
			if attempts < 3 {
				return nil, &net.OpError{Op: "dial", Err: errors.New("connection refused")}
			}
			return &CompletionResponse{
				ID: "success-id",
				Choices: []CompletionResponseChoice{
					{Message: Message{Role: "assistant", Content: "success"}},
				},
			}, nil
		},
	}

	config := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}
	retryClient := NewRetryingClient(mockClient, config)

	ctx := context.Background()
	req := CompletionRequest{Messages: []Message{{Role: "user", Content: "test"}}}

	resp, err := retryClient.ChatCompletion(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "success-id", resp.ID)
	assert.Equal(t, 3, attempts)
}

func TestRetryingClient_ChatCompletion_RetriesOnRateLimit(t *testing.T) {
	attempts := 0
	mockClient := &MockLLMClient{
		chatCompletionFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			attempts++
			if attempts < 2 {
				return nil, &APIError{
					StatusCode: 429,
					Status:     "429 Too Many Requests",
					Body:       "rate limit exceeded",
				}
			}
			return &CompletionResponse{
				ID: "success-id",
				Choices: []CompletionResponseChoice{
					{Message: Message{Role: "assistant", Content: "success"}},
				},
			}, nil
		},
	}

	config := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}
	retryClient := NewRetryingClient(mockClient, config)

	ctx := context.Background()
	req := CompletionRequest{Messages: []Message{{Role: "user", Content: "test"}}}

	resp, err := retryClient.ChatCompletion(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "success-id", resp.ID)
	assert.Equal(t, 2, attempts)
}

func TestRetryingClient_ChatCompletion_RetriesOnServerError(t *testing.T) {
	attempts := 0
	mockClient := &MockLLMClient{
		chatCompletionFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			attempts++
			if attempts < 2 {
				return nil, &APIError{
					StatusCode: 503,
					Status:     "503 Service Unavailable",
					Body:       "service unavailable",
				}
			}
			return &CompletionResponse{
				ID: "success-id",
				Choices: []CompletionResponseChoice{
					{Message: Message{Role: "assistant", Content: "success"}},
				},
			}, nil
		},
	}

	config := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}
	retryClient := NewRetryingClient(mockClient, config)

	ctx := context.Background()
	req := CompletionRequest{Messages: []Message{{Role: "user", Content: "test"}}}

	resp, err := retryClient.ChatCompletion(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "success-id", resp.ID)
	assert.Equal(t, 2, attempts)
}

func TestRetryingClient_ChatCompletion_DoesNotRetryOnNonRetryableError(t *testing.T) {
	attempts := 0
	mockClient := &MockLLMClient{
		chatCompletionFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			attempts++
			return nil, &APIError{
				StatusCode: 400,
				Status:     "400 Bad Request",
				Body:       "bad request",
			}
		},
	}

	config := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}
	retryClient := NewRetryingClient(mockClient, config)

	ctx := context.Background()
	req := CompletionRequest{Messages: []Message{{Role: "user", Content: "test"}}}

	resp, err := retryClient.ChatCompletion(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, 1, attempts) // Should only try once
	assert.Contains(t, err.Error(), "400")
}

func TestRetryingClient_ChatCompletion_ExhaustsRetries(t *testing.T) {
	attempts := 0
	mockClient := &MockLLMClient{
		chatCompletionFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			attempts++
			return nil, &APIError{
				StatusCode: 503,
				Status:     "503 Service Unavailable",
				Body:       "service unavailable",
			}
		},
	}

	config := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}
	retryClient := NewRetryingClient(mockClient, config)

	ctx := context.Background()
	req := CompletionRequest{Messages: []Message{{Role: "user", Content: "test"}}}

	resp, err := retryClient.ChatCompletion(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, 3, attempts) // Should try max attempts
}

func TestRetryingClient_ChatCompletion_RespectsContextCancellation(t *testing.T) {
	mockClient := &MockLLMClient{
		chatCompletionFunc: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			return nil, &APIError{
				StatusCode: 503,
				Status:     "503 Service Unavailable",
				Body:       "service unavailable",
			}
		},
	}

	config := RetryConfig{
		MaxAttempts:    5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		BackoffFactor:  2.0,
	}
	retryClient := NewRetryingClient(mockClient, config)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := CompletionRequest{Messages: []Message{{Role: "user", Content: "test"}}}

	resp, err := retryClient.ChatCompletion(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled))
}

func TestRetryingClient_ChatCompletionStream_Success(t *testing.T) {
	mockClient := &MockLLMClient{
		chatCompletionStreamFunc: func(ctx context.Context, req CompletionRequest, handler StreamHandler) error {
			return nil
		},
	}

	config := DefaultRetryConfig()
	retryClient := NewRetryingClient(mockClient, config)

	ctx := context.Background()
	req := CompletionRequest{Messages: []Message{{Role: "user", Content: "test"}}}
	handler := func(chunk CompletionResponse) error {
		return nil
	}

	err := retryClient.ChatCompletionStream(ctx, req, handler)

	require.NoError(t, err)
}

func TestRetryingClient_ChatCompletionStream_RetriesOnError(t *testing.T) {
	attempts := 0
	mockClient := &MockLLMClient{
		chatCompletionStreamFunc: func(ctx context.Context, req CompletionRequest, handler StreamHandler) error {
			attempts++
			if attempts < 2 {
				return &APIError{
					StatusCode: 503,
					Status:     "503 Service Unavailable",
					Body:       "service unavailable",
				}
			}
			return nil
		},
	}

	config := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}
	retryClient := NewRetryingClient(mockClient, config)

	ctx := context.Background()
	req := CompletionRequest{Messages: []Message{{Role: "user", Content: "test"}}}
	handler := func(chunk CompletionResponse) error {
		return nil
	}

	err := retryClient.ChatCompletionStream(ctx, req, handler)

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
}

func TestRetryingClient_ChatCompletionStream_DoesNotRetryOnNonRetryableError(t *testing.T) {
	attempts := 0
	mockClient := &MockLLMClient{
		chatCompletionStreamFunc: func(ctx context.Context, req CompletionRequest, handler StreamHandler) error {
			attempts++
			return &APIError{
				StatusCode: 401,
				Status:     "401 Unauthorized",
				Body:       "unauthorized",
			}
		},
	}

	config := RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		BackoffFactor:  2.0,
	}
	retryClient := NewRetryingClient(mockClient, config)

	ctx := context.Background()
	req := CompletionRequest{Messages: []Message{{Role: "user", Content: "test"}}}
	handler := func(chunk CompletionResponse) error {
		return nil
	}

	err := retryClient.ChatCompletionStream(ctx, req, handler)

	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // Should only try once
	assert.Contains(t, err.Error(), "401")
}

func TestRetryingClient_GetModelInfo(t *testing.T) {
	mockClient := &MockLLMClient{
		modelInfo: ModelInfo{
			Name:        "test-model",
			Provider:    "test-provider",
			MaxTokens:   4096,
			Description: "test description",
		},
	}

	config := DefaultRetryConfig()
	retryClient := NewRetryingClient(mockClient, config)

	info := retryClient.GetModelInfo()

	assert.Equal(t, "test-model", info.Name)
	assert.Equal(t, "test-provider", info.Provider)
	assert.Equal(t, 4096, info.MaxTokens)
	assert.Equal(t, "test description", info.Description)
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "context canceled",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "context deadline exceeded",
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name: "rate limit error (APIError)",
			err: &APIError{
				StatusCode: 429,
				Status:     "429 Too Many Requests",
				Body:       "rate limit exceeded",
			},
			retryable: true,
		},
		{
			name: "500 error (APIError)",
			err: &APIError{
				StatusCode: 500,
				Status:     "500 Internal Server Error",
				Body:       "internal server error",
			},
			retryable: true,
		},
		{
			name: "502 error (APIError)",
			err: &APIError{
				StatusCode: 502,
				Status:     "502 Bad Gateway",
				Body:       "bad gateway",
			},
			retryable: true,
		},
		{
			name: "503 error (APIError)",
			err: &APIError{
				StatusCode: 503,
				Status:     "503 Service Unavailable",
				Body:       "service unavailable",
			},
			retryable: true,
		},
		{
			name: "504 error (APIError)",
			err: &APIError{
				StatusCode: 504,
				Status:     "504 Gateway Timeout",
				Body:       "gateway timeout",
			},
			retryable: true,
		},
		{
			name: "400 error (APIError)",
			err: &APIError{
				StatusCode: 400,
				Status:     "400 Bad Request",
				Body:       "bad request",
			},
			retryable: false,
		},
		{
			name: "401 error (APIError)",
			err: &APIError{
				StatusCode: 401,
				Status:     "401 Unauthorized",
				Body:       "unauthorized",
			},
			retryable: false,
		},
		{
			name:      "validation error",
			err:       fmt.Errorf("invalid request: missing required field"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	tests := []struct {
		statusCode int
		retryable  bool
	}{
		{200, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			apiErr := &APIError{
				StatusCode: tt.statusCode,
				Status:     fmt.Sprintf("%d", tt.statusCode),
				Body:       "error body",
			}
			result := apiErr.IsRetryable()
			assert.Equal(t, tt.retryable, result)
		})
	}
}
