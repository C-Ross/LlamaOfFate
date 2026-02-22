package character

import (
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCharacter(t *testing.T) {
	id := "test-id"
	name := "Test Character"

	char := NewCharacter(id, name)

	assert.Equal(t, id, char.ID)
	assert.Equal(t, name, char.Name)
	assert.Equal(t, 3, char.FatePoints)
	assert.Equal(t, 3, char.Refresh)
	assert.Empty(t, char.Skills)
	assert.Empty(t, char.Stunts)

	// Check stress tracks
	physicalTrack := char.GetStressTrack(PhysicalStress)
	require.NotNil(t, physicalTrack, "Should have physical stress track")
	assert.Equal(t, 2, physicalTrack.MaxBoxes)

	mentalTrack := char.GetStressTrack(MentalStress)
	require.NotNil(t, mentalTrack, "Should have mental stress track")
	assert.Equal(t, 2, mentalTrack.MaxBoxes)
}

func TestCharacter_GetSkill(t *testing.T) {
	char := NewCharacter("test", "Test")

	// Test default skill (should be Mediocre)
	skill := char.GetSkill("Athletics")
	assert.Equal(t, dice.Mediocre, skill)

	// Set a skill and test retrieval
	char.SetSkill("Athletics", dice.Good)
	skill = char.GetSkill("Athletics")
	assert.Equal(t, dice.Good, skill)
}

func TestCharacter_SetSkill(t *testing.T) {
	char := NewCharacter("test", "Test")
	originalTime := char.UpdatedAt

	time.Sleep(1 * time.Millisecond) // Ensure time difference

	char.SetSkill("Fight", dice.Great)

	assert.Equal(t, dice.Great, char.Skills["Fight"])
	assert.True(t, char.UpdatedAt.After(originalTime))
}

func TestCharacter_FatePoints(t *testing.T) {
	char := NewCharacter("test", "Test")

	// Test spending fate points
	assert.True(t, char.SpendFatePoint(), "Should be able to spend fate point when points available")
	assert.Equal(t, 2, char.FatePoints)

	// Spend all remaining points
	char.SpendFatePoint()
	char.SpendFatePoint()
	assert.Equal(t, 0, char.FatePoints)

	// Test spending when no points available
	assert.False(t, char.SpendFatePoint(), "Should not be able to spend when no points available")

	// Test gaining fate points
	char.GainFatePoint()
	assert.Equal(t, 1, char.FatePoints)

	// Test refresh
	char.RefreshFatePoints()
	assert.Equal(t, char.Refresh, char.FatePoints)
}

func TestCharacter_TakeStress(t *testing.T) {
	char := NewCharacter("test", "Test")

	// Test taking physical stress
	assert.True(t, char.TakeStress(PhysicalStress, 1))

	track := char.GetStressTrack(PhysicalStress)
	assert.True(t, track.Boxes[0], "Should mark the stress box")

	// Test taking stress on already filled box
	assert.False(t, char.TakeStress(PhysicalStress, 1), "Should fail for already filled box")

	// Test taking stress beyond track capacity
	assert.False(t, char.TakeStress(PhysicalStress, 5), "Should fail for stress beyond track capacity")
}

func TestCharacter_HasAspect(t *testing.T) {
	char := NewCharacter("test", "Test")

	// Test with no aspects
	assert.False(t, char.HasAspect("Test Aspect"))

	// Set aspects
	char.Aspects.HighConcept = "Wizard Detective"
	char.Aspects.Trouble = "The Lure of Ancient Mysteries"

	// Test existing aspects
	assert.True(t, char.HasAspect("Wizard Detective"))
	assert.True(t, char.HasAspect("The Lure of Ancient Mysteries"))

	// Test consequence aspects
	consequence := Consequence{
		ID:     "test-consequence",
		Type:   MildConsequence,
		Aspect: "Bruised Ribs",
	}
	char.AddConsequence(consequence)

	assert.True(t, char.HasAspect("Bruised Ribs"))
}

func TestAspects_GetAll(t *testing.T) {
	aspects := Aspects{
		HighConcept:  "Wizard Detective",
		Trouble:      "The Lure of Ancient Mysteries",
		OtherAspects: []string{"The Case of the Cursed Curator", "Friends in Low Places"},
	}

	all := aspects.GetAll()

	assert.Len(t, all, 4)

	expectedAspects := []string{
		"Wizard Detective",
		"The Lure of Ancient Mysteries",
		"The Case of the Cursed Curator",
		"Friends in Low Places",
	}

	for _, expected := range expectedAspects {
		assert.Contains(t, all, expected)
	}
}

func TestAspects_IsComplete(t *testing.T) {
	aspects := Aspects{}

	assert.False(t, aspects.IsComplete())

	aspects.HighConcept = "Test"
	assert.False(t, aspects.IsComplete(), "Should return false when missing Trouble")

	aspects.Trouble = "Test"
	assert.True(t, aspects.IsComplete(), "Should return true when High Concept and Trouble are filled")
}

func TestAspects_AddAspect(t *testing.T) {
	aspects := Aspects{OtherAspects: make([]string, 0)}

	aspects.AddAspect("Quick Reflexes")

	assert.Len(t, aspects.OtherAspects, 1)
	assert.Equal(t, "Quick Reflexes", aspects.OtherAspects[0])

	// Test adding empty aspect
	aspects.AddAspect("")
	assert.Len(t, aspects.OtherAspects, 1, "Should not add empty aspects")
}

func TestAspects_RemoveAspect(t *testing.T) {
	aspects := Aspects{
		OtherAspects: []string{"First Aspect", "Second Aspect", "Third Aspect"},
	}

	// Remove middle aspect
	assert.True(t, aspects.RemoveAspect(1))
	assert.Len(t, aspects.OtherAspects, 2)
	assert.Equal(t, "Third Aspect", aspects.OtherAspects[1])

	// Test invalid indices
	assert.False(t, aspects.RemoveAspect(-1))
	assert.False(t, aspects.RemoveAspect(10))
}

func TestAspects_SetAspect(t *testing.T) {
	aspects := Aspects{
		OtherAspects: []string{"First Aspect", "Second Aspect"},
	}

	// Set existing aspect
	assert.True(t, aspects.SetAspect(0, "Updated First Aspect"))
	assert.Equal(t, "Updated First Aspect", aspects.OtherAspects[0])

	// Test invalid indices
	assert.False(t, aspects.SetAspect(-1, "Invalid"))
	assert.False(t, aspects.SetAspect(10, "Invalid"))
}

func TestAspects_Count(t *testing.T) {
	aspects := Aspects{OtherAspects: make([]string, 0)}

	// Test empty aspects
	assert.Equal(t, 0, aspects.Count())

	// Add High Concept
	aspects.HighConcept = "Test High Concept"
	assert.Equal(t, 1, aspects.Count())

	// Add Trouble
	aspects.Trouble = "Test Trouble"
	assert.Equal(t, 2, aspects.Count())

	// Add other aspects
	aspects.AddAspect("First Other")
	aspects.AddAspect("Second Other")
	assert.Equal(t, 4, aspects.Count())

	// Test with empty other aspect
	aspects.OtherAspects = append(aspects.OtherAspects, "")
	assert.Equal(t, 4, aspects.Count(), "Should not count empty aspects")
}

func TestNewStressTrack(t *testing.T) {
	track := NewStressTrack(PhysicalStress, 3)

	assert.Equal(t, PhysicalStress, track.Type)
	assert.Equal(t, 3, track.MaxBoxes)
	assert.Len(t, track.Boxes, 3)

	// All boxes should start empty
	for i, box := range track.Boxes {
		assert.False(t, box, "Box %d should start empty", i)
	}
}

func TestStressTrack_TakeStress(t *testing.T) {
	track := NewStressTrack(PhysicalStress, 3)

	// Test valid stress
	assert.True(t, track.TakeStress(2))
	assert.True(t, track.Boxes[1], "Should mark box 2") // 2-stress box is index 1

	// Test taking same stress again
	assert.False(t, track.TakeStress(2), "Should fail when box already filled")

	// Test invalid stress amounts
	assert.False(t, track.TakeStress(0))
	assert.False(t, track.TakeStress(4), "Should fail for track with only 3 boxes")
}

func TestStressTrack_IsFull(t *testing.T) {
	track := NewStressTrack(PhysicalStress, 2)

	assert.False(t, track.IsFull())

	track.TakeStress(1)
	assert.False(t, track.IsFull(), "Should return false when not all boxes filled")

	track.TakeStress(2)
	assert.True(t, track.IsFull(), "Should return true when all boxes filled")
}

func TestStressTrack_AvailableBoxes(t *testing.T) {
	track := NewStressTrack(PhysicalStress, 3)

	assert.Equal(t, 3, track.AvailableBoxes())

	track.TakeStress(2)
	assert.Equal(t, 2, track.AvailableBoxes(), "Should have 2 available after taking 1 box")
}

func TestConsequenceType_Value(t *testing.T) {
	tests := []struct {
		consequenceType ConsequenceType
		expectedValue   int
	}{
		{MildConsequence, 2},
		{ModerateConsequence, 4},
		{SevereConsequence, 6},
		{ExtremeConsequence, 8},
		{ConsequenceType("invalid"), 0},
	}

	for _, test := range tests {
		value := test.consequenceType.Value()
		assert.Equal(t, test.expectedValue, value,
			"ConsequenceType(%s).Value() should return %d", test.consequenceType, test.expectedValue)
	}
}

func TestStressTrack_ClearAll(t *testing.T) {
	track := NewStressTrack(PhysicalStress, 3)

	// Fill some boxes
	track.TakeStress(1)
	track.TakeStress(3)
	assert.Equal(t, 1, track.AvailableBoxes())

	// Clear all
	track.ClearAll()
	assert.Equal(t, 3, track.AvailableBoxes())
	for _, box := range track.Boxes {
		assert.False(t, box, "All boxes should be cleared")
	}
}

func TestStressTrack_ClearAll_EmptyTrack(t *testing.T) {
	track := NewStressTrack(MentalStress, 2)
	// Nothing filled, should be safe to call
	track.ClearAll()
	assert.Equal(t, 2, track.AvailableBoxes())
}

func TestCharacter_ClearAllStress(t *testing.T) {
	char := NewCharacter("test", "Test")

	// Fill stress on both tracks
	char.TakeStress(PhysicalStress, 1)
	char.TakeStress(PhysicalStress, 2)
	char.TakeStress(MentalStress, 1)

	assert.Equal(t, 0, char.GetStressTrack(PhysicalStress).AvailableBoxes())
	assert.Equal(t, 1, char.GetStressTrack(MentalStress).AvailableBoxes())

	char.ClearAllStress()

	assert.Equal(t, 2, char.GetStressTrack(PhysicalStress).AvailableBoxes())
	assert.Equal(t, 2, char.GetStressTrack(MentalStress).AvailableBoxes())
}

func TestCharacter_RemoveConsequence(t *testing.T) {
	char := NewCharacter("test", "Test")

	c1 := Consequence{ID: "c1", Type: MildConsequence, Aspect: "Bruised"}
	c2 := Consequence{ID: "c2", Type: ModerateConsequence, Aspect: "Deep Cut"}
	char.AddConsequence(c1)
	char.AddConsequence(c2)
	assert.Len(t, char.Consequences, 2)

	assert.True(t, char.RemoveConsequence("c1"))
	assert.Len(t, char.Consequences, 1)
	assert.Equal(t, "c2", char.Consequences[0].ID)

	assert.False(t, char.RemoveConsequence("nonexistent"))
	assert.Len(t, char.Consequences, 1)
}

func TestCharacter_BeginConsequenceRecovery(t *testing.T) {
	char := NewCharacter("test", "Test")

	c1 := Consequence{ID: "c1", Type: MildConsequence, Aspect: "Winded"}
	char.AddConsequence(c1)

	assert.True(t, char.BeginConsequenceRecovery("c1", 3, 1))
	assert.True(t, char.Consequences[0].Recovering)
	assert.Equal(t, 3, char.Consequences[0].RecoveryStartScene)
	assert.Equal(t, 1, char.Consequences[0].RecoveryStartScenario)

	// Non-existent ID
	assert.False(t, char.BeginConsequenceRecovery("nonexistent", 3, 1))
}

func TestCharacter_CheckConsequenceRecovery_MildClearsAfterOneScene(t *testing.T) {
	char := NewCharacter("test", "Test")

	c := Consequence{
		ID:                 "c1",
		Type:               MildConsequence,
		Aspect:             "Bruised Hand",
		Recovering:         true,
		RecoveryStartScene: 2,
	}
	char.AddConsequence(c)

	// Same scene — not healed yet
	cleared := char.CheckConsequenceRecovery(2, 0)
	assert.Empty(t, cleared)
	assert.Len(t, char.Consequences, 1)

	// Next scene — healed
	cleared = char.CheckConsequenceRecovery(3, 0)
	assert.Len(t, cleared, 1)
	assert.Equal(t, "Bruised Hand", cleared[0].Aspect)
	assert.Empty(t, char.Consequences)
}

func TestCharacter_CheckConsequenceRecovery_ModerateClearsAfterOneScenario(t *testing.T) {
	char := NewCharacter("test", "Test")

	c := Consequence{
		ID:                    "c2",
		Type:                  ModerateConsequence,
		Aspect:                "Deep Cut",
		Recovering:            true,
		RecoveryStartScenario: 1,
	}
	char.AddConsequence(c)

	// Same scenario — not healed
	cleared := char.CheckConsequenceRecovery(10, 1)
	assert.Empty(t, cleared)
	assert.Len(t, char.Consequences, 1)

	// Next scenario — healed
	cleared = char.CheckConsequenceRecovery(10, 2)
	assert.Len(t, cleared, 1)
	assert.Equal(t, "Deep Cut", cleared[0].Aspect)
	assert.Empty(t, char.Consequences)
}

func TestCharacter_CheckConsequenceRecovery_SevereClearsAfterOneScenario(t *testing.T) {
	char := NewCharacter("test", "Test")

	c := Consequence{
		ID:                    "c3",
		Type:                  SevereConsequence,
		Aspect:                "Broken Leg",
		Recovering:            true,
		RecoveryStartScenario: 0,
	}
	char.AddConsequence(c)

	// Same scenario — not healed
	cleared := char.CheckConsequenceRecovery(10, 0)
	assert.Empty(t, cleared)

	// Next scenario — healed
	cleared = char.CheckConsequenceRecovery(10, 1)
	assert.Len(t, cleared, 1)
	assert.Equal(t, "Broken Leg", cleared[0].Aspect)
}

func TestCharacter_CheckConsequenceRecovery_NonRecoveringNotCleared(t *testing.T) {
	char := NewCharacter("test", "Test")

	c := Consequence{
		ID:         "c1",
		Type:       MildConsequence,
		Aspect:     "Winded",
		Recovering: false,
	}
	char.AddConsequence(c)

	// Even many scenes later, non-recovering consequences stay
	cleared := char.CheckConsequenceRecovery(100, 100)
	assert.Empty(t, cleared)
	assert.Len(t, char.Consequences, 1)
}

func TestCharacter_CheckConsequenceRecovery_MixedConsequences(t *testing.T) {
	char := NewCharacter("test", "Test")

	// Mild recovering consequence (should clear at scene 4)
	char.AddConsequence(Consequence{
		ID: "mild", Type: MildConsequence, Aspect: "Bruised",
		Recovering: true, RecoveryStartScene: 3,
	})
	// Moderate not yet recovering
	char.AddConsequence(Consequence{
		ID: "moderate", Type: ModerateConsequence, Aspect: "Deep Cut",
		Recovering: false,
	})
	// Severe recovering (should clear at scenario 2)
	char.AddConsequence(Consequence{
		ID: "severe", Type: SevereConsequence, Aspect: "Broken Leg",
		Recovering: true, RecoveryStartScenario: 1,
	})

	// At scene 4, scenario 1 — mild clears, severe stays
	cleared := char.CheckConsequenceRecovery(4, 1)
	assert.Len(t, cleared, 1)
	assert.Equal(t, "Bruised", cleared[0].Aspect)
	assert.Len(t, char.Consequences, 2)

	// At scene 5, scenario 2 — severe clears
	cleared = char.CheckConsequenceRecovery(5, 2)
	assert.Len(t, cleared, 1)
	assert.Equal(t, "Broken Leg", cleared[0].Aspect)
	assert.Len(t, char.Consequences, 1)

	// Moderate stays because it's not recovering
	assert.Equal(t, "moderate", char.Consequences[0].ID)
}

func TestCharacter_IsTakenOut(t *testing.T) {
	char := NewCharacter("npc-1", "Goblin")

	assert.False(t, char.IsTakenOut(), "New character should not be taken out")

	char.Fate = &TakenOutFate{
		Description: "Killed",
		Permanent:   true,
	}
	assert.True(t, char.IsTakenOut(), "Character with Fate set should be taken out")
}

func TestCharacter_IsPermanentlyRemoved(t *testing.T) {
	char := NewCharacter("npc-1", "Goblin")

	assert.False(t, char.IsPermanentlyRemoved(), "New character should not be permanently removed")

	char.Fate = &TakenOutFate{
		Description: "Knocked unconscious",
		Permanent:   false,
	}
	assert.False(t, char.IsPermanentlyRemoved(), "Non-permanent fate should not be permanently removed")

	char.Fate = &TakenOutFate{
		Description: "Killed",
		Permanent:   true,
	}
	assert.True(t, char.IsPermanentlyRemoved(), "Permanent fate should be permanently removed")
}

// TestRecalculateStressTracks covers all Fate Core SRD tiers for Physique and Will,
// extra mild consequence slots at Superb+, NPC type behaviour, and InitDefaults
// hydration as sub-tests within a single table-driven function.
func TestRecalculateStressTracks(t *testing.T) {
	ladder := func(l dice.Ladder) *dice.Ladder { return &l }
	mildIDs := [4]string{"m0", "m1", "m2", "m3"}

	tests := []struct {
		name            string
		setup           func() *Character // nil → NewCharacter("test", "Test")
		physique        *dice.Ladder      // nil → not set
		will            *dice.Ladder      // nil → not set
		wantPhysBoxes   int
		wantMentalBoxes int
		wantMaxMild     int // total mild consequence slots available (0 = skip check)
	}{
		// No-skill default
		{
			name:            "no skills → 2 boxes each, 1 mild slot",
			wantPhysBoxes:   2,
			wantMentalBoxes: 2,
			wantMaxMild:     1,
		},
		// Average tier (3 boxes)
		{
			name:            "Physique Average → 3 physical boxes",
			physique:        ladder(dice.Average),
			wantPhysBoxes:   3,
			wantMentalBoxes: 2,
			wantMaxMild:     1,
		},
		{
			name:            "Will Average → 3 mental boxes",
			will:            ladder(dice.Average),
			wantPhysBoxes:   2,
			wantMentalBoxes: 3,
			wantMaxMild:     1,
		},
		{
			name:            "Physique Fair → 3 physical boxes",
			physique:        ladder(dice.Fair),
			wantPhysBoxes:   3,
			wantMentalBoxes: 2,
			wantMaxMild:     1,
		},
		{
			name:            "Will Fair → 3 mental boxes",
			will:            ladder(dice.Fair),
			wantPhysBoxes:   2,
			wantMentalBoxes: 3,
			wantMaxMild:     1,
		},
		// Good tier (4 boxes)
		{
			name:            "Physique Good → 4 physical boxes",
			physique:        ladder(dice.Good),
			wantPhysBoxes:   4,
			wantMentalBoxes: 2,
			wantMaxMild:     1,
		},
		{
			name:            "Will Good → 4 mental boxes",
			will:            ladder(dice.Good),
			wantPhysBoxes:   2,
			wantMentalBoxes: 4,
			wantMaxMild:     1,
		},
		{
			name:            "Physique Great → 4 physical boxes",
			physique:        ladder(dice.Great),
			wantPhysBoxes:   4,
			wantMentalBoxes: 2,
			wantMaxMild:     1,
		},
		{
			name:            "Will Great → 4 mental boxes",
			will:            ladder(dice.Great),
			wantPhysBoxes:   2,
			wantMentalBoxes: 4,
			wantMaxMild:     1,
		},
		// Superb tier (4 boxes + extra mild slot)
		{
			name:            "Physique Superb → 4 physical boxes and 1 extra mild slot",
			physique:        ladder(dice.Superb),
			wantPhysBoxes:   4,
			wantMentalBoxes: 2,
			wantMaxMild:     2,
		},
		{
			name:            "Will Superb → 4 mental boxes and 1 extra mild slot",
			will:            ladder(dice.Superb),
			wantPhysBoxes:   2,
			wantMentalBoxes: 4,
			wantMaxMild:     2,
		},
		{
			name:            "both Physique and Will Superb → 4 boxes each and 2 extra mild slots",
			physique:        ladder(dice.Superb),
			will:            ladder(dice.Superb),
			wantPhysBoxes:   4,
			wantMentalBoxes: 4,
			wantMaxMild:     3,
		},
		// NPC types
		{
			name:            "Main NPC with Physique Good and Will Fair",
			setup:           func() *Character { return NewMainNPC("boss-1", "Boss") },
			physique:        ladder(dice.Good),
			will:            ladder(dice.Fair),
			wantPhysBoxes:   4,
			wantMentalBoxes: 3,
			wantMaxMild:     1,
		},
		{
			name:            "Supporting NPC default → 2 boxes each",
			setup:           func() *Character { return NewSupportingNPC("npc", "NPC", "Concept") },
			wantPhysBoxes:   2,
			wantMentalBoxes: 2,
			wantMaxMild:     1,
		},
		{
			name:            "Supporting NPC with Physique Good → 4 physical boxes",
			setup:           func() *Character { return NewSupportingNPC("npc", "NPC", "Concept") },
			physique:        ladder(dice.Good),
			wantPhysBoxes:   4,
			wantMentalBoxes: 2,
			wantMaxMild:     1,
		},
		// InitDefaults hydration from saved skills
		{
			name: "InitDefaults hydrates tracks from existing skills",
			setup: func() *Character {
				char := &Character{
					ID:     "saved",
					Name:   "Loaded Hero",
					Skills: map[string]dice.Ladder{"Physique": dice.Great, "Will": dice.Average},
					Aspects: Aspects{
						OtherAspects: make([]string, 0),
					},
					Consequences: make([]Consequence, 0),
				}
				char.InitDefaults()
				return char
			},
			wantPhysBoxes:   4,
			wantMentalBoxes: 3,
			wantMaxMild:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var char *Character
			if tt.setup != nil {
				char = tt.setup()
			} else {
				char = NewCharacter("test", "Test")
			}
			if tt.physique != nil {
				char.SetSkill("Physique", *tt.physique)
			}
			if tt.will != nil {
				char.SetSkill("Will", *tt.will)
			}

			physTrack := char.GetStressTrack(PhysicalStress)
			mentalTrack := char.GetStressTrack(MentalStress)
			require.NotNil(t, physTrack)
			require.NotNil(t, mentalTrack)
			assert.Equal(t, tt.wantPhysBoxes, physTrack.MaxBoxes)
			assert.Equal(t, tt.wantMentalBoxes, mentalTrack.MaxBoxes)

			if tt.wantMaxMild > 0 && char.CharacterType.HasConsequences() {
				for i := 0; i < tt.wantMaxMild; i++ {
					assert.True(t, char.CanTakeConsequence(MildConsequence),
						"mild slot %d of %d should be available", i+1, tt.wantMaxMild)
					char.AddConsequence(Consequence{ID: mildIDs[i], Type: MildConsequence})
				}
				assert.False(t, char.CanTakeConsequence(MildConsequence),
					"should have no more mild slots after %d taken", tt.wantMaxMild)
			}
		})
	}
}

// TestRecalculateStressTracks_SkillChangeMidGame verifies that changing Physique or Will
// mid-game resizes the track while preserving already-checked boxes.
func TestRecalculateStressTracks_SkillChangeMidGame(t *testing.T) {
	char := NewCharacter("test", "Test") // 2 boxes initially

	// Mark box 1 and 2
	assert.True(t, char.TakeStress(PhysicalStress, 1))
	assert.True(t, char.TakeStress(PhysicalStress, 2))

	// Upgrade Physique to Good → expands to 4 boxes
	char.SetSkill("Physique", dice.Good)
	physTrack := char.GetStressTrack(PhysicalStress)
	assert.Equal(t, 4, physTrack.MaxBoxes, "Should expand to 4 boxes")
	assert.True(t, physTrack.Boxes[0], "Box 1 should remain checked after resize")
	assert.True(t, physTrack.Boxes[1], "Box 2 should remain checked after resize")
	assert.False(t, physTrack.Boxes[2], "Box 3 should start empty")
	assert.False(t, physTrack.Boxes[3], "Box 4 should start empty")
}

// TestStressBoxesForSkill covers all tiers in the Fate Core table.
func TestStressBoxesForSkill(t *testing.T) {
	tests := []struct {
		level    dice.Ladder
		expected int
	}{
		{dice.Mediocre, 2},
		{dice.Average, 3},
		{dice.Fair, 3},
		{dice.Good, 4},
		{dice.Great, 4},
		{dice.Superb, 4},
		{dice.Fantastic, 4},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, stressBoxesForSkill(tt.level),
			"stressBoxesForSkill(%s)", tt.level)
	}
}
