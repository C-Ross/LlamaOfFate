package engine

import "errors"

// Sentinel errors for game operations
var (
	ErrLLMUnavailable     = errors.New("LLM service unavailable")
	ErrLLMTimeout         = errors.New("LLM request timed out")
	ErrLLMInvalidResponse = errors.New("LLM returned invalid response")
	ErrNoActiveScene      = errors.New("no active scene")
	ErrCharacterNotFound  = errors.New("character not found")

	// Conflict-related errors
	ErrConflictAlreadyActive    = errors.New("conflict is already active")
	ErrNoActiveConflict         = errors.New("no active conflict")
	ErrParticipantNotInConflict = errors.New("participant not in conflict")
	ErrInvalidConflictType      = errors.New("invalid conflict type")
	ErrCannotEscalateToSameType = errors.New("cannot escalate to same conflict type")
	ErrCharacterAlreadyActed    = errors.New("character has already acted this exchange")
	ErrNotCharacterTurn         = errors.New("not this character's turn")
)

// IsRetryable returns true if the error might succeed on retry
func IsRetryable(err error) bool {
	return errors.Is(err, ErrLLMTimeout) || errors.Is(err, ErrLLMUnavailable)
}
