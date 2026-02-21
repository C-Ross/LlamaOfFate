package engine

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestSaveCorruptError(t *testing.T) {
	cause := fmt.Errorf("save file is corrupt or incompatible: player has no high concept")
	saveErr := &SaveCorruptError{Cause: cause}

	t.Run("Error message wraps cause", func(t *testing.T) {
		assert.Equal(t, "saved game could not be loaded: save file is corrupt or incompatible: player has no high concept", saveErr.Error())
	})

	t.Run("Unwrap returns cause", func(t *testing.T) {
		assert.Equal(t, cause, saveErr.Unwrap())
	})

	t.Run("errors.As matches SaveCorruptError", func(t *testing.T) {
		wrapped := fmt.Errorf("factory: %w", saveErr)
		var target *SaveCorruptError
		require.True(t, errors.As(wrapped, &target))
		assert.Equal(t, cause, target.Cause)
	})
}
