package engine

import (
	"errors"
	"fmt"
)

// Sentinel errors for game operations
var (
	ErrNoActiveScene     = errors.New("no active scene")
	ErrCharacterNotFound = errors.New("character not found")

	// Conflict-related errors
	ErrConflictAlreadyActive    = errors.New("conflict is already active")
	ErrNoActiveConflict         = errors.New("no active conflict")
	ErrParticipantNotInConflict = errors.New("participant not in conflict")
	ErrInvalidConflictType      = errors.New("invalid conflict type")
	ErrCannotEscalateToSameType = errors.New("cannot escalate to same conflict type")
	ErrCharacterAlreadyActed    = errors.New("character has already acted this exchange")
	ErrNotCharacterTurn         = errors.New("not this character's turn")
)

// SaveCorruptError indicates that a save file exists but could not be loaded
// because it is corrupt or incompatible. Callers should enter the setup flow
// and display the error message to the user.
type SaveCorruptError struct {
	Cause error
}

func (e *SaveCorruptError) Error() string {
	return fmt.Sprintf("saved game could not be loaded: %v", e.Cause)
}

func (e *SaveCorruptError) Unwrap() error {
	return e.Cause
}
