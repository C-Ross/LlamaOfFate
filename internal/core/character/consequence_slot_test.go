package character

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- AvailableConsequenceSlots ---

func TestAvailableConsequenceSlots_PC_AllOpen(t *testing.T) {
	char := NewCharacter("pc-1", "Hero")
	// PC has no consequences yet — all three slots should be available
	slots := char.AvailableConsequenceSlots()

	require.Len(t, slots, 3)
	assert.Equal(t, MildConsequence, slots[0].Type)
	assert.Equal(t, 2, slots[0].Value)
	assert.Equal(t, ModerateConsequence, slots[1].Type)
	assert.Equal(t, 4, slots[1].Value)
	assert.Equal(t, SevereConsequence, slots[2].Type)
	assert.Equal(t, 6, slots[2].Value)
}

func TestAvailableConsequenceSlots_PC_MildTaken(t *testing.T) {
	char := NewCharacter("pc-1", "Hero")
	char.AddConsequence(Consequence{ID: "c1", Type: MildConsequence, Aspect: "Bruised"})

	slots := char.AvailableConsequenceSlots()

	require.Len(t, slots, 2)
	assert.Equal(t, ModerateConsequence, slots[0].Type)
	assert.Equal(t, SevereConsequence, slots[1].Type)
}

func TestAvailableConsequenceSlots_PC_AllTaken(t *testing.T) {
	char := NewCharacter("pc-1", "Hero")
	char.AddConsequence(Consequence{ID: "c1", Type: MildConsequence, Aspect: "Bruised"})
	char.AddConsequence(Consequence{ID: "c2", Type: ModerateConsequence, Aspect: "Broken Arm"})
	char.AddConsequence(Consequence{ID: "c3", Type: SevereConsequence, Aspect: "Shattered"})

	slots := char.AvailableConsequenceSlots()
	assert.Empty(t, slots)
}

func TestAvailableConsequenceSlots_SupportingNPC_OnlyMild(t *testing.T) {
	char := NewSupportingNPC("npc-1", "Guard", "Loyal Guard")

	slots := char.AvailableConsequenceSlots()
	require.Len(t, slots, 1)
	assert.Equal(t, MildConsequence, slots[0].Type)
}

func TestAvailableConsequenceSlots_SupportingNPC_MildTaken(t *testing.T) {
	char := NewSupportingNPC("npc-1", "Guard", "Loyal Guard")
	char.AddConsequence(Consequence{ID: "c1", Type: MildConsequence, Aspect: "Scratched"})

	slots := char.AvailableConsequenceSlots()
	assert.Empty(t, slots)
}

func TestAvailableConsequenceSlots_NamelessNPC_NoSlots(t *testing.T) {
	char := NewNamelessNPC("npc-2", "Thug", CharacterTypeNamelessGood, "Fight")

	slots := char.AvailableConsequenceSlots()
	assert.Empty(t, slots, "Nameless NPCs should have no consequence slots")
}

func TestAvailableConsequenceSlots_MainNPC_AllOpen(t *testing.T) {
	char := NewMainNPC("npc-3", "Villain")

	slots := char.AvailableConsequenceSlots()
	require.Len(t, slots, 3, "Main NPCs have full consequence slots like PCs")
}

// --- BestConsequenceFor ---

func TestBestConsequenceFor_EmptySlots(t *testing.T) {
	_, ok := BestConsequenceFor(nil, 3)
	assert.False(t, ok)

	_, ok = BestConsequenceFor([]ConsequenceSlot{}, 3)
	assert.False(t, ok)
}

func TestBestConsequenceFor_ExactMatch(t *testing.T) {
	slots := []ConsequenceSlot{
		{Type: MildConsequence, Value: 2},
		{Type: ModerateConsequence, Value: 4},
		{Type: SevereConsequence, Value: 6},
	}

	best, ok := BestConsequenceFor(slots, 4)
	require.True(t, ok)
	assert.Equal(t, ModerateConsequence, best.Type, "Should pick moderate for exactly 4 shifts")
}

func TestBestConsequenceFor_SmallestCoveringSlot(t *testing.T) {
	slots := []ConsequenceSlot{
		{Type: MildConsequence, Value: 2},
		{Type: ModerateConsequence, Value: 4},
		{Type: SevereConsequence, Value: 6},
	}

	best, ok := BestConsequenceFor(slots, 3)
	require.True(t, ok)
	assert.Equal(t, ModerateConsequence, best.Type,
		"Should pick moderate (4) as smallest that covers 3 shifts")
}

func TestBestConsequenceFor_NoneCovers_PicksLargest(t *testing.T) {
	slots := []ConsequenceSlot{
		{Type: MildConsequence, Value: 2},
		{Type: ModerateConsequence, Value: 4},
	}

	best, ok := BestConsequenceFor(slots, 5)
	require.True(t, ok)
	assert.Equal(t, ModerateConsequence, best.Type,
		"When nothing covers 5 shifts, pick largest available (moderate=4)")
}

func TestBestConsequenceFor_SingleSlotCovers(t *testing.T) {
	slots := []ConsequenceSlot{
		{Type: SevereConsequence, Value: 6},
	}

	best, ok := BestConsequenceFor(slots, 3)
	require.True(t, ok)
	assert.Equal(t, SevereConsequence, best.Type)
}

func TestBestConsequenceFor_SingleSlotDoesNotCover(t *testing.T) {
	slots := []ConsequenceSlot{
		{Type: MildConsequence, Value: 2},
	}

	best, ok := BestConsequenceFor(slots, 5)
	require.True(t, ok)
	assert.Equal(t, MildConsequence, best.Type,
		"Only option even though it doesn't cover the shifts")
}

func TestBestConsequenceFor_PrefersMildOverSevere(t *testing.T) {
	// Reverse order to ensure it's not just picking the first one
	slots := []ConsequenceSlot{
		{Type: SevereConsequence, Value: 6},
		{Type: MildConsequence, Value: 2},
	}

	best, ok := BestConsequenceFor(slots, 2)
	require.True(t, ok)
	assert.Equal(t, MildConsequence, best.Type,
		"Should prefer mild (exact fit) over severe (overkill)")
}

func TestBestConsequenceFor_OneShift(t *testing.T) {
	slots := []ConsequenceSlot{
		{Type: MildConsequence, Value: 2},
		{Type: ModerateConsequence, Value: 4},
		{Type: SevereConsequence, Value: 6},
	}

	best, ok := BestConsequenceFor(slots, 1)
	require.True(t, ok)
	assert.Equal(t, MildConsequence, best.Type,
		"For 1 shift, mild is the tightest fit")
}

func TestBestConsequenceFor_LargeHitAllAvailable(t *testing.T) {
	slots := []ConsequenceSlot{
		{Type: MildConsequence, Value: 2},
		{Type: ModerateConsequence, Value: 4},
		{Type: SevereConsequence, Value: 6},
	}

	best, ok := BestConsequenceFor(slots, 6)
	require.True(t, ok)
	assert.Equal(t, SevereConsequence, best.Type,
		"6 shifts: severe is exact match")
}

func TestBestConsequenceFor_MassiveHitBeyondAll(t *testing.T) {
	slots := []ConsequenceSlot{
		{Type: MildConsequence, Value: 2},
		{Type: ModerateConsequence, Value: 4},
		{Type: SevereConsequence, Value: 6},
	}

	best, ok := BestConsequenceFor(slots, 10)
	require.True(t, ok)
	assert.Equal(t, SevereConsequence, best.Type,
		"Nothing covers 10 shifts, picks largest available")
}
