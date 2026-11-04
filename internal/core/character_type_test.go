package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCharacterType_String(t *testing.T) {
	tests := []struct {
		charType CharacterType
		expected string
	}{
		{CharacterTypePC, "Player Character"},
		{CharacterTypeMainNPC, "Main NPC"},
		{CharacterTypeSupportingNPC, "Supporting NPC"},
		{CharacterTypeNamelessGood, "Nameless (Good)"},
		{CharacterTypeNamelessFair, "Nameless (Fair)"},
		{CharacterTypeNamelessAverage, "Nameless (Average)"},
		{CharacterType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.charType.String())
		})
	}
}

func TestCharacterType_IsPC(t *testing.T) {
	tests := []struct {
		charType CharacterType
		isPC     bool
	}{
		{CharacterTypePC, true},
		{CharacterTypeMainNPC, false},
		{CharacterTypeSupportingNPC, false},
		{CharacterTypeNamelessGood, false},
		{CharacterTypeNamelessFair, false},
		{CharacterTypeNamelessAverage, false},
	}

	for _, tt := range tests {
		t.Run(tt.charType.String(), func(t *testing.T) {
			assert.Equal(t, tt.isPC, tt.charType.IsPC())
		})
	}
}

func TestCharacterType_IsNPC(t *testing.T) {
	tests := []struct {
		charType CharacterType
		isNPC    bool
	}{
		{CharacterTypePC, false},
		{CharacterTypeMainNPC, true},
		{CharacterTypeSupportingNPC, true},
		{CharacterTypeNamelessGood, true},
		{CharacterTypeNamelessFair, true},
		{CharacterTypeNamelessAverage, true},
	}

	for _, tt := range tests {
		t.Run(tt.charType.String(), func(t *testing.T) {
			assert.Equal(t, tt.isNPC, tt.charType.IsNPC())
		})
	}
}

func TestCharacterType_IsNameless(t *testing.T) {
	tests := []struct {
		charType   CharacterType
		isNameless bool
	}{
		{CharacterTypePC, false},
		{CharacterTypeMainNPC, false},
		{CharacterTypeSupportingNPC, false},
		{CharacterTypeNamelessGood, true},
		{CharacterTypeNamelessFair, true},
		{CharacterTypeNamelessAverage, true},
	}

	for _, tt := range tests {
		t.Run(tt.charType.String(), func(t *testing.T) {
			assert.Equal(t, tt.isNameless, tt.charType.IsNameless())
		})
	}
}

func TestCharacterType_HasConsequences(t *testing.T) {
	tests := []struct {
		charType  CharacterType
		hasConseq bool
	}{
		{CharacterTypePC, true},
		{CharacterTypeMainNPC, true},
		{CharacterTypeSupportingNPC, true},
		{CharacterTypeNamelessGood, false},
		{CharacterTypeNamelessFair, false},
		{CharacterTypeNamelessAverage, false},
	}

	for _, tt := range tests {
		t.Run(tt.charType.String(), func(t *testing.T) {
			assert.Equal(t, tt.hasConseq, tt.charType.HasConsequences())
		})
	}
}

func TestCharacterType_MaxConsequenceSeverity(t *testing.T) {
	tests := []struct {
		charType    CharacterType
		maxSeverity ConsequenceType
	}{
		{CharacterTypePC, SevereConsequence},
		{CharacterTypeMainNPC, SevereConsequence},
		{CharacterTypeSupportingNPC, MildConsequence},
		{CharacterTypeNamelessGood, ""},
		{CharacterTypeNamelessFair, ""},
		{CharacterTypeNamelessAverage, ""},
	}

	for _, tt := range tests {
		t.Run(tt.charType.String(), func(t *testing.T) {
			assert.Equal(t, tt.maxSeverity, tt.charType.MaxConsequenceSeverity())
		})
	}
}

func TestNewCharacter_IsPC(t *testing.T) {
	// Default NewCharacter creates a PC
	pc := NewCharacter("player-1", "Hero")

	assert.Equal(t, CharacterTypePC, pc.CharacterType)
	assert.True(t, pc.CharacterType.IsPC())
	assert.False(t, pc.CharacterType.IsNPC())
}

func TestNewNamelessNPC_Good(t *testing.T) {
	npc := NewNamelessNPC("goblin-1", "Goblin Warrior", CharacterTypeNamelessGood, "Fight")

	assert.Equal(t, "goblin-1", npc.ID)
	assert.Equal(t, "Goblin Warrior", npc.Name)
	assert.Equal(t, CharacterTypeNamelessGood, npc.CharacterType)
	assert.Equal(t, 0, npc.FatePoints)

	// Good nameless NPCs have one skill at Good (+3)
	require.NotNil(t, npc.Skills["Fight"])
	assert.Equal(t, 3, int(npc.Skills["Fight"]))

	// Good nameless NPCs have 2 stress boxes
	require.NotNil(t, npc.StressTracks["physical"])
	assert.Equal(t, 2, len(npc.StressTracks["physical"].Boxes))

	// Nameless NPCs can't take consequences
	assert.False(t, npc.CanTakeConsequence(MildConsequence))
}

func TestNewNamelessNPC_Fair(t *testing.T) {
	npc := NewNamelessNPC("guard-1", "Town Guard", CharacterTypeNamelessFair, "Notice")

	assert.Equal(t, CharacterTypeNamelessFair, npc.CharacterType)

	// Fair nameless NPCs have one skill at Fair (+2)
	require.NotNil(t, npc.Skills["Notice"])
	assert.Equal(t, 2, int(npc.Skills["Notice"]))

	// Fair nameless NPCs have 1 stress box
	require.NotNil(t, npc.StressTracks["physical"])
	assert.Equal(t, 1, len(npc.StressTracks["physical"].Boxes))
}

func TestNewNamelessNPC_Average(t *testing.T) {
	npc := NewNamelessNPC("peasant-1", "Frightened Peasant", CharacterTypeNamelessAverage, "Will")

	assert.Equal(t, CharacterTypeNamelessAverage, npc.CharacterType)

	// Average nameless NPCs have one skill at Average (+1)
	require.NotNil(t, npc.Skills["Will"])
	assert.Equal(t, 1, int(npc.Skills["Will"]))

	// Average nameless NPCs have NO stress boxes
	assert.Nil(t, npc.StressTracks["physical"])
}

func TestNewSupportingNPC(t *testing.T) {
	npc := NewSupportingNPC("innkeeper-1", "Boris the Innkeeper", "Knows Everyone's Business")

	assert.Equal(t, "innkeeper-1", npc.ID)
	assert.Equal(t, "Boris the Innkeeper", npc.Name)
	assert.Equal(t, CharacterTypeSupportingNPC, npc.CharacterType)
	assert.Equal(t, "Knows Everyone's Business", npc.Aspects.HighConcept)
	assert.Equal(t, 1, npc.FatePoints)

	// Supporting NPCs have stress tracks
	require.NotNil(t, npc.StressTracks["physical"])
	require.NotNil(t, npc.StressTracks["mental"])

	// Supporting NPCs can only take Mild consequences
	assert.True(t, npc.CanTakeConsequence(MildConsequence))
	assert.False(t, npc.CanTakeConsequence(ModerateConsequence))
	assert.False(t, npc.CanTakeConsequence(SevereConsequence))
}

func TestNewMainNPC(t *testing.T) {
	npc := NewMainNPC("villain-1", "The Dark Lord")

	assert.Equal(t, "villain-1", npc.ID)
	assert.Equal(t, "The Dark Lord", npc.Name)
	assert.Equal(t, CharacterTypeMainNPC, npc.CharacterType)
	assert.Equal(t, 3, npc.FatePoints)

	// Main NPCs have full stress tracks
	require.NotNil(t, npc.StressTracks["physical"])
	require.NotNil(t, npc.StressTracks["mental"])

	// Main NPCs can take all consequence types
	assert.True(t, npc.CanTakeConsequence(MildConsequence))
	assert.True(t, npc.CanTakeConsequence(ModerateConsequence))
	assert.True(t, npc.CanTakeConsequence(SevereConsequence))
}

func TestCharacter_CanTakeConsequence_AlreadyTaken(t *testing.T) {
	npc := NewMainNPC("villain-1", "The Dark Lord")

	// Initially can take Mild
	assert.True(t, npc.CanTakeConsequence(MildConsequence))

	// Add a Mild consequence
	npc.AddConsequence(Consequence{
		ID:   "conseq-1",
		Type: MildConsequence,
	})

	// Now can't take another Mild
	assert.False(t, npc.CanTakeConsequence(MildConsequence))

	// But can still take Moderate
	assert.True(t, npc.CanTakeConsequence(ModerateConsequence))
}

func TestCharacter_CanTakeConsequence_NamelessNPC(t *testing.T) {
	npc := NewNamelessNPC("goblin-1", "Goblin", CharacterTypeNamelessGood, "Fight")

	// Nameless NPCs cannot take any consequences
	assert.False(t, npc.CanTakeConsequence(MildConsequence))
	assert.False(t, npc.CanTakeConsequence(ModerateConsequence))
	assert.False(t, npc.CanTakeConsequence(SevereConsequence))
	assert.False(t, npc.CanTakeConsequence(ExtremeConsequence))
}

func TestNamelessNPC_TakenOutOnHit_Average(t *testing.T) {
	// Average nameless NPCs have no stress track
	npc := NewNamelessNPC("minion-1", "Minion", CharacterTypeNamelessAverage, "Fight")

	// No stress track means any hit takes them out
	stressTrack := npc.StressTracks["physical"]
	assert.Nil(t, stressTrack, "Average nameless NPCs should have no stress track")
}

func TestNamelessNPC_LimitedStress(t *testing.T) {
	// Good nameless NPCs can absorb some damage
	npc := NewNamelessNPC("thug-1", "Thug", CharacterTypeNamelessGood, "Fight")

	// Can absorb 1 or 2 shifts
	assert.True(t, npc.TakeStress(PhysicalStress, 1))
	assert.True(t, npc.TakeStress(PhysicalStress, 2))

	// Now stress track is full - can't absorb more
	assert.False(t, npc.TakeStress(PhysicalStress, 1))
	assert.False(t, npc.TakeStress(PhysicalStress, 2))
}
