package character

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
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
