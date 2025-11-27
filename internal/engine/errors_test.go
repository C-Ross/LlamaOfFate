package engine

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "LLM timeout is retryable",
			err:      ErrLLMTimeout,
			expected: true,
		},
		{
			name:     "LLM unavailable is retryable",
			err:      ErrLLMUnavailable,
			expected: true,
		},
		{
			name:     "wrapped LLM timeout is retryable",
			err:      fmt.Errorf("operation failed: %w", ErrLLMTimeout),
			expected: true,
		},
		{
			name:     "wrapped LLM unavailable is retryable",
			err:      fmt.Errorf("classifyInput: %w", ErrLLMUnavailable),
			expected: true,
		},
		{
			name:     "invalid response is not retryable",
			err:      ErrLLMInvalidResponse,
			expected: false,
		},
		{
			name:     "no active scene is not retryable",
			err:      ErrNoActiveScene,
			expected: false,
		},
		{
			name:     "character not found is not retryable",
			err:      ErrCharacterNotFound,
			expected: false,
		},
		{
			name:     "generic error is not retryable",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "nil error is not retryable",
			err:      nil,
			expected: false,
		},
		{
			name:     "deeply wrapped timeout is retryable",
			err:      fmt.Errorf("outer: %w", fmt.Errorf("middle: %w", ErrLLMTimeout)),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryable(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	t.Run("errors have expected messages", func(t *testing.T) {
		assert.Equal(t, "LLM service unavailable", ErrLLMUnavailable.Error())
		assert.Equal(t, "LLM request timed out", ErrLLMTimeout.Error())
		assert.Equal(t, "LLM returned invalid response", ErrLLMInvalidResponse.Error())
		assert.Equal(t, "no active scene", ErrNoActiveScene.Error())
		assert.Equal(t, "character not found", ErrCharacterNotFound.Error())
	})

	t.Run("wrapped errors can be identified with errors.Is", func(t *testing.T) {
		wrappedTimeout := fmt.Errorf("operation: %w", ErrLLMTimeout)
		assert.True(t, errors.Is(wrappedTimeout, ErrLLMTimeout))

		wrappedUnavailable := fmt.Errorf("classifyInput: %w", ErrLLMUnavailable)
		assert.True(t, errors.Is(wrappedUnavailable, ErrLLMUnavailable))

		wrappedInvalid := fmt.Errorf("parsing failed: %w", ErrLLMInvalidResponse)
		assert.True(t, errors.Is(wrappedInvalid, ErrLLMInvalidResponse))
	})

	t.Run("different errors are not equal", func(t *testing.T) {
		assert.False(t, errors.Is(ErrLLMTimeout, ErrLLMUnavailable))
		assert.False(t, errors.Is(ErrLLMInvalidResponse, ErrNoActiveScene))
		assert.False(t, errors.Is(ErrCharacterNotFound, ErrLLMTimeout))
	})
}
