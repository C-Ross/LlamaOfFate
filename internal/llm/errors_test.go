package llm

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
			err:      ErrTimeout,
			expected: true,
		},
		{
			name:     "LLM unavailable is retryable",
			err:      ErrUnavailable,
			expected: true,
		},
		{
			name:     "wrapped LLM timeout is retryable",
			err:      fmt.Errorf("operation failed: %w", ErrTimeout),
			expected: true,
		},
		{
			name:     "wrapped LLM unavailable is retryable",
			err:      fmt.Errorf("classifyInput: %w", ErrUnavailable),
			expected: true,
		},
		{
			name:     "invalid response is not retryable",
			err:      ErrInvalidResponse,
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
			err:      fmt.Errorf("outer: %w", fmt.Errorf("middle: %w", ErrTimeout)),
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

func TestLLMSentinelErrors(t *testing.T) {
	t.Run("errors have expected messages", func(t *testing.T) {
		assert.Equal(t, "LLM service unavailable", ErrUnavailable.Error())
		assert.Equal(t, "LLM request timed out", ErrTimeout.Error())
		assert.Equal(t, "LLM returned invalid response", ErrInvalidResponse.Error())
		assert.Equal(t, "empty LLM response", ErrEmptyResponse.Error())
	})

	t.Run("wrapped errors can be identified with errors.Is", func(t *testing.T) {
		wrappedTimeout := fmt.Errorf("operation: %w", ErrTimeout)
		assert.True(t, errors.Is(wrappedTimeout, ErrTimeout))

		wrappedUnavailable := fmt.Errorf("classifyInput: %w", ErrUnavailable)
		assert.True(t, errors.Is(wrappedUnavailable, ErrUnavailable))

		wrappedInvalid := fmt.Errorf("parsing failed: %w", ErrInvalidResponse)
		assert.True(t, errors.Is(wrappedInvalid, ErrInvalidResponse))
	})

	t.Run("different errors are not equal", func(t *testing.T) {
		assert.False(t, errors.Is(ErrTimeout, ErrUnavailable))
		assert.False(t, errors.Is(ErrInvalidResponse, ErrEmptyResponse))
	})
}
