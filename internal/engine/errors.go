package engine

import "errors"

// Sentinel errors for game operations
var (
	ErrLLMUnavailable     = errors.New("LLM service unavailable")
	ErrLLMTimeout         = errors.New("LLM request timed out")
	ErrLLMInvalidResponse = errors.New("LLM returned invalid response")
	ErrNoActiveScene      = errors.New("no active scene")
	ErrCharacterNotFound  = errors.New("character not found")
)

// IsRetryable returns true if the error might succeed on retry
func IsRetryable(err error) bool {
	return errors.Is(err, ErrLLMTimeout) || errors.Is(err, ErrLLMUnavailable)
}
