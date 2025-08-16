package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// TestIntegration_BasicGameplay demonstrates core systems working together
func TestIntegration_BasicGameplay(t *testing.T) {
	// Create a character
	char := character.NewCharacter("hero-1", "Zara the Bold")
	char.Aspects.HighConcept = "Daring Sky Pirate"
	char.Aspects.Trouble = "Wanted by the Imperial Navy"
	char.Aspects.AddAspect("The Case of the Cursed Curator")
	char.SetSkill("Athletics", dice.Good)
	char.SetSkill("Fight", dice.Great)

	// Create a scene
	gameScene := scene.NewScene("scene-1", "The Burning Airship", "The deck tilts precariously as flames lick at the rigging")
	gameScene.AddCharacter(char.ID)

	// Add a situation aspect to the scene
	situationAspect := scene.NewSituationAspect("aspect-1", "Deck Tilting Dangerously", char.ID, 2)
	gameScene.AddSituationAspect(situationAspect)

	// Create an action
	testAction := action.NewAction("action-1", char.ID, action.Overcome, "Athletics", "Leap across the burning gap")
	testAction.Difficulty = dice.Great // Challenging difficulty

	// Add an aspect invocation
	aspectInvoke := action.AspectInvoke{
		AspectText:    "Daring Sky Pirate",
		Source:        "character",
		SourceID:      char.ID,
		IsFree:        false,
		FatePointCost: 1,
		Bonus:         2,
	}
	testAction.AddAspectInvoke(aspectInvoke)

	// Simulate the action resolution
	roller := dice.NewSeededRoller(12345) // Use seeded roller for predictable test
	baseSkill := char.GetSkill("Athletics") // Good (+3)
	bonus := testAction.CalculateBonus()    // +2 from aspect
	
	checkResult := roller.RollWithModifier(baseSkill, bonus)
	testAction.CheckResult = checkResult
	testAction.Outcome = checkResult.CompareAgainst(testAction.Difficulty)

	// Verify the integration works
	assert.Equal(t, dice.Good, char.GetSkill("Athletics"))
	assert.Len(t, gameScene.Characters, 1)
	assert.Equal(t, char.ID, gameScene.Characters[0])
	assert.Len(t, gameScene.SituationAspects, 1)
	assert.Equal(t, 2, testAction.CalculateBonus())
	require.NotNil(t, testAction.CheckResult)
	require.NotNil(t, testAction.Outcome)

	// Verify fate point spending
	assert.True(t, char.SpendFatePoint())
	assert.Equal(t, 2, char.FatePoints, "Should have 2 fate points remaining after spending 1")

	t.Logf("Action result: %s with %d shifts", testAction.Outcome.Type.String(), testAction.Outcome.Shifts)
	t.Logf("Final roll value: %s", testAction.CheckResult.FinalValue.String())
}

// TestIntegration_StressAndConsequences demonstrates damage system
func TestIntegration_StressAndConsequences(t *testing.T) {
	// Create a character
	fighter := character.NewCharacter("fighter-1", "Grimm the Barbarian")
	fighter.SetSkill("Fight", dice.Superb)
	fighter.SetSkill("Physique", dice.Great)

	// Test stress system
	physicalTrack := fighter.GetStressTrack(character.PhysicalStress)
	require.NotNil(t, physicalTrack)

	// Take some stress
	assert.True(t, fighter.TakeStress(character.PhysicalStress, 1))
	assert.True(t, fighter.TakeStress(character.PhysicalStress, 2))

	// Try to take stress when track is full
	assert.False(t, fighter.TakeStress(character.PhysicalStress, 1), 
		"Should not be able to take 1-stress when box already filled")

	// Verify track state
	assert.True(t, physicalTrack.IsFull())
	assert.Equal(t, 0, physicalTrack.AvailableBoxes())

	// Add a consequence
	consequence := character.Consequence{
		ID:     "cons-1",
		Type:   character.MildConsequence,
		Aspect: "Bruised and Battered",
	}
	fighter.AddConsequence(consequence)

	// Verify consequence was added and creates an aspect
	assert.True(t, fighter.HasAspect("Bruised and Battered"))
	assert.Len(t, fighter.Consequences, 1)

	// Test consequence value
	assert.Equal(t, 2, consequence.Type.Value())
}

// TestIntegration_ConflictManagement demonstrates conflict system
func TestIntegration_ConflictManagement(t *testing.T) {
	// Create characters
	hero := character.NewCharacter("hero", "Hero")
	villain := character.NewCharacter("villain", "Villain")

	// Create scene
	battleScene := scene.NewScene("battle", "The Throne Room", "A grand chamber with marble pillars")
	battleScene.AddCharacter(hero.ID)
	battleScene.AddCharacter(villain.ID)

	// Start conflict
	participants := []scene.ConflictParticipant{
		{CharacterID: hero.ID, Initiative: 3, Active: true},
		{CharacterID: villain.ID, Initiative: 1, Active: true},
	}
	battleScene.StartConflict(participants)

	// Verify conflict state
	assert.True(t, battleScene.IsConflict)
	require.NotNil(t, battleScene.ConflictState)

	// Test turn management
	currentActor := battleScene.GetCurrentActor()
	assert.Equal(t, hero.ID, currentActor, "Current actor should be hero (higher initiative)")

	// Advance turn
	battleScene.NextTurn()
	currentActor = battleScene.GetCurrentActor()
	assert.Equal(t, villain.ID, currentActor, "After next turn, current actor should be villain")

	// Advance to next round
	battleScene.NextTurn()
	assert.Equal(t, 2, battleScene.ConflictState.Round)
	assert.Equal(t, 0, battleScene.ConflictState.CurrentTurn)

	// End conflict
	battleScene.EndConflict()
	assert.False(t, battleScene.IsConflict)
	assert.Nil(t, battleScene.ConflictState)
}

// TestIntegration_CompleteActionResolution demonstrates a full action resolution
func TestIntegration_CompleteActionResolution(t *testing.T) {
	// Setup
	character := character.NewCharacter("test-char", "Test Character")
	character.SetSkill("Deceive", dice.Fair)
	character.Aspects.HighConcept = "Master of Disguise"
	character.Aspects.AddAspect("Well Connected")

	gameScene := scene.NewScene("test-scene", "The Masquerade Ball", "Elegant figures in masks dance through the ballroom")
	gameScene.AddCharacter(character.ID)

	// Create situation aspect
	situationAspect := scene.NewSituationAspect("crowded", "Crowded Dance Floor", character.ID, 1)
	gameScene.AddSituationAspect(situationAspect)

	// Create action
	infiltrateAction := action.NewAction("infiltrate", character.ID, action.CreateAdvantage, "Deceive", "Blend into the crowd and overhear conversations")
	infiltrateAction.Difficulty = dice.Fair

	// Invoke character aspect (spend fate point)
	aspectInvoke := action.AspectInvoke{
		AspectText:    "Master of Disguise",
		Source:        "character",
		SourceID:      character.ID,
		IsFree:        false,
		FatePointCost: 1,
		Bonus:         2,
	}
	infiltrateAction.AddAspectInvoke(aspectInvoke)

	// Use free invoke from situation aspect
	freeInvoke := action.AspectInvoke{
		AspectText:    "Crowded Dance Floor",
		Source:        "situation",
		SourceID:      "crowded",
		IsFree:        true,
		FatePointCost: 0,
		Bonus:         2,
	}
	infiltrateAction.AddAspectInvoke(freeInvoke)

	// Process the action
	roller := dice.NewSeededRoller(54321)
	baseSkill := character.GetSkill("Deceive") // Fair (+2)
	totalBonus := infiltrateAction.CalculateBonus() // +4 total

	// Spend fate point
	assert.True(t, character.SpendFatePoint())

	// Use free invoke
	aspect := gameScene.GetSituationAspect("crowded")
	require.NotNil(t, aspect)
	assert.True(t, aspect.UseFreeInvoke())

	// Roll and resolve
	checkResult := roller.RollWithModifier(baseSkill, totalBonus)
	infiltrateAction.CheckResult = checkResult
	infiltrateAction.Outcome = checkResult.CompareAgainst(infiltrateAction.Difficulty)

	// Add effect for successful Create Advantage
	if infiltrateAction.IsSuccess() {
		effect := action.Effect{
			Type:        "aspect",
			Target:      gameScene.ID,
			Value:       "Overheard Secrets",
			Description: "Created situation aspect: Overheard Secrets",
		}
		infiltrateAction.AddEffect(effect)
	}

	// Verify results
	assert.Equal(t, 4, totalBonus)
	assert.Equal(t, 2, character.FatePoints)
	assert.Equal(t, 0, aspect.FreeInvokes)
	require.NotNil(t, infiltrateAction.CheckResult)
	require.NotNil(t, infiltrateAction.Outcome)

	t.Logf("Complete action resolution:")
	t.Logf("  Base skill: %s", baseSkill.String())
	t.Logf("  Total bonus: +%d", totalBonus)
	t.Logf("  Roll result: %s", infiltrateAction.CheckResult.Roll.String())
	t.Logf("  Final value: %s", infiltrateAction.CheckResult.FinalValue.String())
	t.Logf("  Outcome: %s (%d shifts)", infiltrateAction.Outcome.Type.String(), infiltrateAction.Outcome.Shifts)
}
