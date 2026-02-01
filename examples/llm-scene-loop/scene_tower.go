package main

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// =============================================================================
// SCENE: The Abandoned Wizard's Tower (Fantasy)
// =============================================================================

func buildTowerScene() SceneConfig {
	player := createTowerCharacter()
	opponent := createTowerOpponent()

	towerScene := scene.NewScene(
		"abandoned-tower",
		"The Abandoned Wizard's Tower",
		"You stand before a crumbling stone tower that stretches high into the mist. "+
			"Strange blue lights flicker through broken windows near the top. "+
			"The heavy wooden door hangs ajar, revealing darkness within. "+
			"Ancient runes are carved deep into the stone archway, still glowing faintly with magical energy. "+
			"The air hums with residual magic, and you hear the distant sound of something large moving inside.",
	)

	return SceneConfig{
		Name:        "The Abandoned Wizard's Tower",
		Description: "A fantasy exploration scenario",
		Player:      player,
		NPCs:        []*character.Character{opponent},
		Scene:       towerScene,
		Farewell:    "Thanks for exploring the tower!",
	}
}

func createTowerCharacter() *character.Character {
	player := character.NewCharacter("player1", "Lyra Moonwhisper")

	// Set aspects
	player.Aspects.HighConcept = "Scholarly Arcane Investigator"
	player.Aspects.Trouble = "Curiosity About Forbidden Knowledge"
	player.Aspects.AddAspect("Trained by the College of Mages")
	player.Aspects.AddAspect("Sees Magic in Everything")
	player.Aspects.AddAspect("Protective Ward Tattoos")

	// Set skills focused on investigation and magic
	player.SetSkill("Lore", dice.Superb)       // +5 - Primary skill
	player.SetSkill("Investigate", dice.Great) // +4
	player.SetSkill("Notice", dice.Great)      // +4
	player.SetSkill("Will", dice.Good)         // +3
	player.SetSkill("Empathy", dice.Good)      // +3
	player.SetSkill("Rapport", dice.Good)      // +3
	player.SetSkill("Athletics", dice.Fair)    // +2
	player.SetSkill("Burglary", dice.Fair)     // +2
	player.SetSkill("Stealth", dice.Fair)      // +2
	player.SetSkill("Crafts", dice.Fair)       // +2
	player.SetSkill("Deceive", dice.Average)   // +1
	player.SetSkill("Drive", dice.Average)     // +1
	player.SetSkill("Fight", dice.Average)     // +1
	player.SetSkill("Physique", dice.Average)  // +1
	player.SetSkill("Provoke", dice.Average)   // +1

	// Set fate points and refresh
	player.FatePoints = 3
	player.Refresh = 3

	return player
}

func createTowerOpponent() *character.Character {
	// Create a nameless (Good) NPC - has 2 stress boxes, no consequences
	// Per Fate Core SRD: https://fate-srd.com/fate-core/creating-and-playing-opposition#good-nameless-npcs
	opponent := character.NewNamelessNPC("goblin-guard", "Goblin Guard", character.CharacterTypeNamelessGood, "Fight")

	// Nameless NPCs get a single aspect (their high concept)
	opponent.Aspects.HighConcept = "Sneaky Tower Guardian"

	// Good nameless NPCs have one skill at Good (+3) - already set by factory
	// Add a couple more skills at lower levels for variety
	opponent.SetSkill("Stealth", dice.Fair)   // +2
	opponent.SetSkill("Notice", dice.Average) // +1

	return opponent
}
