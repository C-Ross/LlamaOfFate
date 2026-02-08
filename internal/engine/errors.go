package engine

import "errors"

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
