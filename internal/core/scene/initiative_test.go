package scene

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflictState_SortByInitiative_BasicOrdering(t *testing.T) {
	cs := &ConflictState{
		Participants: []ConflictParticipant{
			{CharacterID: "slow", Initiative: 1, Status: StatusActive},
			{CharacterID: "fast", Initiative: 5, Status: StatusActive},
			{CharacterID: "mid", Initiative: 3, Status: StatusActive},
		},
	}

	cs.SortByInitiative()

	// Participants should be sorted descending
	require.Len(t, cs.Participants, 3)
	assert.Equal(t, "fast", cs.Participants[0].CharacterID)
	assert.Equal(t, "mid", cs.Participants[1].CharacterID)
	assert.Equal(t, "slow", cs.Participants[2].CharacterID)

	// Initiative order should match
	assert.Equal(t, []string{"fast", "mid", "slow"}, cs.InitiativeOrder)
}

func TestConflictState_SortByInitiative_ExcludesTakenOut(t *testing.T) {
	cs := &ConflictState{
		Participants: []ConflictParticipant{
			{CharacterID: "active-low", Initiative: 1, Status: StatusActive},
			{CharacterID: "taken-out", Initiative: 10, Status: StatusTakenOut},
			{CharacterID: "active-high", Initiative: 5, Status: StatusActive},
		},
	}

	cs.SortByInitiative()

	// Taken-out participant should sort first by initiative but not appear in order
	assert.Equal(t, "taken-out", cs.Participants[0].CharacterID)
	assert.Equal(t, []string{"active-high", "active-low"}, cs.InitiativeOrder,
		"Only active participants should appear in initiative order")
}

func TestConflictState_SortByInitiative_ExcludesConceded(t *testing.T) {
	cs := &ConflictState{
		Participants: []ConflictParticipant{
			{CharacterID: "conceded", Initiative: 3, Status: StatusConceded},
			{CharacterID: "fighter", Initiative: 2, Status: StatusActive},
		},
	}

	cs.SortByInitiative()

	assert.Equal(t, []string{"fighter"}, cs.InitiativeOrder,
		"Conceded participants should not be in initiative order")
}

func TestConflictState_SortByInitiative_TiedInitiative(t *testing.T) {
	cs := &ConflictState{
		Participants: []ConflictParticipant{
			{CharacterID: "a", Initiative: 3, Status: StatusActive},
			{CharacterID: "b", Initiative: 3, Status: StatusActive},
		},
	}

	cs.SortByInitiative()

	// Both should be in the order (stable tie doesn't matter, just both present)
	require.Len(t, cs.InitiativeOrder, 2)
	assert.Contains(t, cs.InitiativeOrder, "a")
	assert.Contains(t, cs.InitiativeOrder, "b")
}

func TestConflictState_SortByInitiative_SingleParticipant(t *testing.T) {
	cs := &ConflictState{
		Participants: []ConflictParticipant{
			{CharacterID: "solo", Initiative: 4, Status: StatusActive},
		},
	}

	cs.SortByInitiative()

	assert.Equal(t, []string{"solo"}, cs.InitiativeOrder)
}

func TestConflictState_SortByInitiative_AllInactive(t *testing.T) {
	cs := &ConflictState{
		Participants: []ConflictParticipant{
			{CharacterID: "a", Initiative: 5, Status: StatusTakenOut},
			{CharacterID: "b", Initiative: 3, Status: StatusConceded},
		},
	}

	cs.SortByInitiative()

	assert.Empty(t, cs.InitiativeOrder, "No active participants means empty order")
}

func TestConflictState_SortByInitiative_EmptyParticipants(t *testing.T) {
	cs := &ConflictState{
		Participants: []ConflictParticipant{},
	}

	cs.SortByInitiative()

	assert.Empty(t, cs.InitiativeOrder)
}

func TestConflictState_SortByInitiative_ReplacesExistingOrder(t *testing.T) {
	cs := &ConflictState{
		Participants: []ConflictParticipant{
			{CharacterID: "a", Initiative: 1, Status: StatusActive},
			{CharacterID: "b", Initiative: 5, Status: StatusActive},
		},
		InitiativeOrder: []string{"stale-data"},
	}

	cs.SortByInitiative()

	assert.Equal(t, []string{"b", "a"}, cs.InitiativeOrder,
		"Should replace stale initiative order entirely")
}

func TestStartConflict_SortsByInitiative(t *testing.T) {
	s := NewScene("test", "Test", "Test scene")

	participants := []ConflictParticipant{
		{CharacterID: "slow", Initiative: 1, Status: StatusActive},
		{CharacterID: "fast", Initiative: 5, Status: StatusActive},
		{CharacterID: "mid", Initiative: 3, Status: StatusActive},
	}

	s.StartConflict(PhysicalConflict, participants)

	require.NotNil(t, s.ConflictState)
	assert.Equal(t, []string{"fast", "mid", "slow"}, s.ConflictState.InitiativeOrder,
		"StartConflict should sort participants by initiative")
}
