package scene

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScene(t *testing.T) {
	id := "test-scene"
	name := "The Abandoned Library"
	description := "Dust motes dance in shafts of light streaming through broken windows"

	scene := NewScene(id, name, description)

	assert.Equal(t, id, scene.ID)
	assert.Equal(t, name, scene.Name)
	assert.Equal(t, description, scene.Description)
	assert.Empty(t, scene.SituationAspects)
	assert.Empty(t, scene.Characters)
	assert.False(t, scene.IsConflict)
	assert.Nil(t, scene.ConflictState)

	// Check timestamps
	assert.WithinDuration(t, time.Now(), scene.CreatedAt, time.Second)
	assert.WithinDuration(t, time.Now(), scene.UpdatedAt, time.Second)
}

func TestScene_AddCharacter(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")
	originalTime := scene.UpdatedAt

	time.Sleep(1 * time.Millisecond) // Ensure time difference

	characterID := "char-123"
	scene.AddCharacter(characterID)

	assert.Len(t, scene.Characters, 1)
	assert.Equal(t, characterID, scene.Characters[0])
	assert.True(t, scene.UpdatedAt.After(originalTime))

	// Test adding duplicate character
	scene.AddCharacter(characterID)
	assert.Len(t, scene.Characters, 1, "Should not add duplicate characters")
}

func TestScene_RemoveCharacter(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	// Add characters
	scene.AddCharacter("char-1")
	scene.AddCharacter("char-2")
	scene.AddCharacter("char-3")
	scene.ActiveCharacter = "char-2"

	originalTime := scene.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	// Remove middle character
	scene.RemoveCharacter("char-2")

	assert.Len(t, scene.Characters, 2)

	// Check that char-2 is not in the list
	assert.NotContains(t, scene.Characters, "char-2")

	// Check that active character was cleared
	assert.Empty(t, scene.ActiveCharacter)
	assert.True(t, scene.UpdatedAt.After(originalTime))

	// Test removing non-existent character
	originalLength := len(scene.Characters)
	scene.RemoveCharacter("non-existent")
	assert.Len(t, scene.Characters, originalLength, "Should not change list when removing non-existent character")
}

func TestScene_AddSituationAspect(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")
	originalTime := scene.UpdatedAt

	time.Sleep(1 * time.Millisecond)

	aspect := NewSituationAspect("aspect-1", "Dark and Foreboding", "char-123", 2)
	scene.AddSituationAspect(aspect)

	assert.Len(t, scene.SituationAspects, 1)
	assert.Equal(t, "Dark and Foreboding", scene.SituationAspects[0].Aspect)
	assert.True(t, scene.UpdatedAt.After(originalTime))
}

func TestScene_RemoveSituationAspect(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	// Add aspects
	aspect1 := NewSituationAspect("aspect-1", "Dark", "char-1", 1)
	aspect2 := NewSituationAspect("aspect-2", "Creaky", "char-2", 1)
	scene.AddSituationAspect(aspect1)
	scene.AddSituationAspect(aspect2)

	originalTime := scene.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	// Remove first aspect
	scene.RemoveSituationAspect("aspect-1")

	assert.Len(t, scene.SituationAspects, 1)
	assert.Equal(t, "aspect-2", scene.SituationAspects[0].ID)
	assert.True(t, scene.UpdatedAt.After(originalTime))
}

func TestScene_GetSituationAspect(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	aspect := NewSituationAspect("aspect-1", "Mysterious", "char-1", 1)
	scene.AddSituationAspect(aspect)

	// Test finding existing aspect
	found := scene.GetSituationAspect("aspect-1")
	require.NotNil(t, found)
	assert.Equal(t, "Mysterious", found.Aspect)

	// Test finding non-existent aspect
	notFound := scene.GetSituationAspect("non-existent")
	assert.Nil(t, notFound)
}

func TestScene_StartConflict(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
		{CharacterID: "char-2", Initiative: 1},
	}

	originalTime := scene.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	scene.StartConflict(PhysicalConflict, participants)

	assert.True(t, scene.IsConflict)
	require.NotNil(t, scene.ConflictState)
	assert.Equal(t, PhysicalConflict, scene.ConflictState.Type)
	assert.Len(t, scene.ConflictState.Participants, 2)
	assert.Equal(t, 1, scene.ConflictState.Round)
	assert.Equal(t, 0, scene.ConflictState.CurrentTurn)
	// Participants should be set to active status
	assert.Equal(t, StatusActive, scene.ConflictState.Participants[0].Status)
	assert.True(t, scene.UpdatedAt.After(originalTime))
}

func TestScene_EndConflict(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	// Start a conflict first
	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
	}
	scene.StartConflict(PhysicalConflict, participants)

	originalTime := scene.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	scene.EndConflict()

	assert.False(t, scene.IsConflict)
	assert.Nil(t, scene.ConflictState)
	assert.True(t, scene.UpdatedAt.After(originalTime))
}

func TestScene_GetCurrentActor(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	// Test with no conflict
	actor := scene.GetCurrentActor()
	assert.Empty(t, actor)

	// Start conflict
	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
		{CharacterID: "char-2", Initiative: 1},
	}
	scene.StartConflict(PhysicalConflict, participants)

	// Test getting current actor
	actor = scene.GetCurrentActor()
	assert.Equal(t, "char-1", actor)
}

func TestScene_NextTurn(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	// Test with no conflict
	scene.NextTurn() // Should not crash

	// Start conflict
	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
		{CharacterID: "char-2", Initiative: 1},
	}
	scene.StartConflict(PhysicalConflict, participants)

	originalTime := scene.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	// Test advancing turn - first actor should be marked as having acted
	scene.NextTurn()
	assert.Equal(t, 1, scene.ConflictState.CurrentTurn)
	assert.Equal(t, 1, scene.ConflictState.Round)
	assert.True(t, scene.ConflictState.Participants[0].HasActed)

	// Test wrapping to next round
	scene.NextTurn()
	assert.Equal(t, 0, scene.ConflictState.CurrentTurn, "Should wrap to turn 0")
	assert.Equal(t, 2, scene.ConflictState.Round)
	// HasActed should be reset for new round
	assert.False(t, scene.ConflictState.Participants[0].HasActed)
	assert.False(t, scene.ConflictState.Participants[1].HasActed)
	assert.True(t, scene.UpdatedAt.After(originalTime))
}

func TestScene_NextTurn_SkipsInactiveParticipants(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
		{CharacterID: "char-2", Initiative: 2},
		{CharacterID: "char-3", Initiative: 1},
	}
	scene.StartConflict(PhysicalConflict, participants)

	// Mark char-2 as taken out
	scene.SetParticipantStatus("char-2", StatusTakenOut)

	// Advance from char-1
	scene.NextTurn()

	// Should skip char-2 and go to char-3
	assert.Equal(t, "char-3", scene.GetCurrentActor())
}

func TestScene_SetParticipantStatus(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	// Test with no conflict
	assert.False(t, scene.SetParticipantStatus("char-1", StatusConceded))

	// Start conflict
	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
		{CharacterID: "char-2", Initiative: 1},
	}
	scene.StartConflict(PhysicalConflict, participants)

	// Test setting status
	assert.True(t, scene.SetParticipantStatus("char-1", StatusConceded))
	assert.Equal(t, StatusConceded, scene.GetParticipant("char-1").Status)

	// Test setting status for non-existent character
	assert.False(t, scene.SetParticipantStatus("non-existent", StatusTakenOut))
}

func TestScene_FullDefense(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	// Test with no conflict
	assert.False(t, scene.SetFullDefense("char-1"))
	assert.False(t, scene.IsFullDefense("char-1"))

	// Start conflict
	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
		{CharacterID: "char-2", Initiative: 1},
	}
	scene.StartConflict(PhysicalConflict, participants)

	// Test setting full defense
	assert.True(t, scene.SetFullDefense("char-1"))
	assert.True(t, scene.IsFullDefense("char-1"))
	// Full defense should mark as having acted
	assert.True(t, scene.GetParticipant("char-1").HasActed)

	// Test non-existent character
	assert.False(t, scene.SetFullDefense("non-existent"))
	assert.False(t, scene.IsFullDefense("non-existent"))
}

func TestScene_CountActiveParticipants(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
		{CharacterID: "char-2", Initiative: 2},
		{CharacterID: "char-3", Initiative: 1},
	}
	scene.StartConflict(PhysicalConflict, participants)

	assert.Equal(t, 3, scene.CountActiveParticipants())

	scene.SetParticipantStatus("char-1", StatusConceded)
	assert.Equal(t, 2, scene.CountActiveParticipants())

	scene.SetParticipantStatus("char-2", StatusTakenOut)
	assert.Equal(t, 1, scene.CountActiveParticipants())
}

func TestScene_GetParticipant(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	// Test with no conflict
	assert.Nil(t, scene.GetParticipant("char-1"))

	// Start conflict
	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
	}
	scene.StartConflict(MentalConflict, participants)

	// Test getting participant
	p := scene.GetParticipant("char-1")
	require.NotNil(t, p)
	assert.Equal(t, "char-1", p.CharacterID)
	assert.Equal(t, 3, p.Initiative)

	// Test non-existent participant
	assert.Nil(t, scene.GetParticipant("non-existent"))
}

func TestNewSituationAspect(t *testing.T) {
	id := "aspect-123"
	aspect := "On Fire"
	createdBy := "char-456"
	freeInvokes := 2

	situationAspect := NewSituationAspect(id, aspect, createdBy, freeInvokes)

	assert.Equal(t, id, situationAspect.ID)
	assert.Equal(t, aspect, situationAspect.Aspect)
	assert.Equal(t, createdBy, situationAspect.CreatedBy)
	assert.Equal(t, freeInvokes, situationAspect.FreeInvokes)
	assert.Equal(t, "scene", situationAspect.Duration)
	assert.WithinDuration(t, time.Now(), situationAspect.CreatedAt, time.Second)
}

func TestSituationAspect_UseFreeInvoke(t *testing.T) {
	aspect := NewSituationAspect("test", "Test Aspect", "char", 2)

	// Test using free invoke
	assert.True(t, aspect.UseFreeInvoke())
	assert.Equal(t, 1, aspect.FreeInvokes)

	// Use last free invoke
	aspect.UseFreeInvoke()
	assert.Equal(t, 0, aspect.FreeInvokes)

	// Test using when no free invokes available
	assert.False(t, aspect.UseFreeInvoke())
}

func TestScene_StartConflictWithInitiator(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
		{CharacterID: "char-2", Initiative: 1},
	}

	scene.StartConflictWithInitiator(PhysicalConflict, participants, "char-1")

	assert.True(t, scene.IsConflict)
	require.NotNil(t, scene.ConflictState)
	assert.Equal(t, PhysicalConflict, scene.ConflictState.Type)
	assert.Equal(t, "char-1", scene.ConflictState.InitiatingCharacter)
	assert.Len(t, scene.ConflictState.Participants, 2)
}

func TestScene_StartConflictWithInitiator_EmptyInitiator(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
	}

	scene.StartConflictWithInitiator(MentalConflict, participants, "")

	assert.True(t, scene.IsConflict)
	require.NotNil(t, scene.ConflictState)
	assert.Equal(t, MentalConflict, scene.ConflictState.Type)
	assert.Empty(t, scene.ConflictState.InitiatingCharacter)
}

func TestScene_EscalateConflict(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	// Test escalation with no active conflict
	assert.False(t, scene.EscalateConflict(PhysicalConflict))

	// Start a mental conflict
	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
		{CharacterID: "char-2", Initiative: 1},
	}
	scene.StartConflict(MentalConflict, participants)

	// Escalate to physical
	assert.True(t, scene.EscalateConflict(PhysicalConflict))
	assert.Equal(t, PhysicalConflict, scene.ConflictState.Type)
	assert.Equal(t, MentalConflict, scene.ConflictState.OriginalType)
}

func TestScene_EscalateConflict_SameType(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
	}
	scene.StartConflict(PhysicalConflict, participants)

	// Try to escalate to same type
	assert.False(t, scene.EscalateConflict(PhysicalConflict))
	// OriginalType should still be empty since no escalation occurred
	assert.Empty(t, scene.ConflictState.OriginalType)
}

func TestScene_EscalateConflict_PreservesOriginalType(t *testing.T) {
	scene := NewScene("test", "Test Scene", "Test")

	participants := []ConflictParticipant{
		{CharacterID: "char-1", Initiative: 3},
	}
	scene.StartConflict(MentalConflict, participants)

	// First escalation
	scene.EscalateConflict(PhysicalConflict)
	assert.Equal(t, MentalConflict, scene.ConflictState.OriginalType)

	// Simulate another escalation back (should preserve original)
	scene.EscalateConflict(MentalConflict)
	assert.Equal(t, MentalConflict, scene.ConflictState.OriginalType)
}

func TestScene_MarkCharacterTakenOut(t *testing.T) {
	s := NewScene("test", "Test Scene", "Test")

	// Initially no characters are taken out
	assert.False(t, s.IsCharacterTakenOut("char-1"))

	// Mark character as taken out
	originalTime := s.UpdatedAt
	time.Sleep(1 * time.Millisecond)
	s.MarkCharacterTakenOut("char-1")

	// Character should now be marked as taken out
	assert.True(t, s.IsCharacterTakenOut("char-1"))
	assert.False(t, s.IsCharacterTakenOut("char-2"))
	assert.True(t, s.UpdatedAt.After(originalTime))
}

func TestScene_MarkCharacterTakenOut_NilMap(t *testing.T) {
	// Create scene and manually nil out the map to test defensive handling
	s := NewScene("test", "Test Scene", "Test")
	s.TakenOutCharacters = nil

	// Should not panic and should return false
	assert.False(t, s.IsCharacterTakenOut("char-1"))

	// Mark should initialize the map
	s.MarkCharacterTakenOut("char-1")
	assert.True(t, s.IsCharacterTakenOut("char-1"))
}

func TestIsExpiredBoost_TrueWhenBoostWithZeroInvokes(t *testing.T) {
	boost := NewBoost("b-1", "Fleeting Opening", "char-1")
	assert.False(t, boost.IsExpiredBoost(), "fresh boost still has 1 free invoke")

	boost.UseFreeInvoke()
	assert.True(t, boost.IsExpiredBoost(), "boost with 0 free invokes should be expired")
}

func TestIsExpiredBoost_FalseForNonBoostAspect(t *testing.T) {
	sa := NewSituationAspect("sa-1", "On Fire", "char-1", 1)
	sa.UseFreeInvoke()
	assert.False(t, sa.IsExpiredBoost(), "regular aspect with 0 invokes is not an expired boost")
}

func TestIsExpiredBoost_FalseWhenInvokesRemain(t *testing.T) {
	boost := NewBoost("b-2", "Momentary Edge", "char-1")
	assert.False(t, boost.IsExpiredBoost(), "boost with remaining invokes is not expired")
}
