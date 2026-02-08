package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSentinelErrors(t *testing.T) {
	t.Run("errors have expected messages", func(t *testing.T) {
		assert.Equal(t, "no active scene", ErrNoActiveScene.Error())
		assert.Equal(t, "character not found", ErrCharacterNotFound.Error())
		assert.Equal(t, "conflict is already active", ErrConflictAlreadyActive.Error())
		assert.Equal(t, "no active conflict", ErrNoActiveConflict.Error())
		assert.Equal(t, "participant not in conflict", ErrParticipantNotInConflict.Error())
		assert.Equal(t, "invalid conflict type", ErrInvalidConflictType.Error())
		assert.Equal(t, "cannot escalate to same conflict type", ErrCannotEscalateToSameType.Error())
		assert.Equal(t, "character has already acted this exchange", ErrCharacterAlreadyActed.Error())
		assert.Equal(t, "not this character's turn", ErrNotCharacterTurn.Error())
	})

	t.Run("different errors are not equal", func(t *testing.T) {
		assert.False(t, errors.Is(ErrNoActiveScene, ErrCharacterNotFound))
		assert.False(t, errors.Is(ErrConflictAlreadyActive, ErrNoActiveConflict))
	})
}
