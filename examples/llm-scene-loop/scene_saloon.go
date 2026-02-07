package main

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// =============================================================================
// SCENE: The Dusty Spur Saloon (Western) - Non-Hostile NPC + Two Aspects
// =============================================================================

func buildSaloonScene() SceneConfig {
	player := createSaloonCharacter()
	bartender := createBartender()

	saloonScene := scene.NewScene(
		"dusty-spur-saloon",
		"The Dusty Spur Saloon",
		"The swinging doors creak as you step into the Dusty Spur, the only watering hole in Redemption Gulch. "+
			"Dust motes dance in shafts of afternoon light cutting through grimy windows. "+
			"A piano sits silent in the corner, its player long gone. "+
			"Behind the bar, a weathered woman with sharp eyes polishes a glass, watching you with guarded curiosity. "+
			"The smell of cheap whiskey and tobacco hangs heavy in the air. "+
			"You've heard rumors that someone here knows the whereabouts of the Cortez Gang—"+
			"the outlaws who burned your ranch and left you for dead.",
	)

	// Add two interesting scene aspects
	saloonScene.AddSituationAspect(scene.NewSituationAspect(
		"information-for-sale",
		"Everyone Here Has a Price",
		"scene",
		0,
	))

	saloonScene.AddSituationAspect(scene.NewSituationAspect(
		"nervous-tension",
		"Strangers Ain't Welcome",
		"scene",
		0,
	))

	return SceneConfig{
		Name:        "The Dusty Spur Saloon",
		Description: "A western scenario with social intrigue",
		Player:      player,
		NPCs:        []*character.Character{bartender},
		Scene:       saloonScene,
		Farewell:    "You tip your hat and step back into the dusty street. The hunt continues.",
	}
}

func createSaloonCharacter() *character.Character {
	player := character.NewCharacter("player1", "Jesse Calhoun")

	// Set aspects for a revenge-seeking gunslinger
	player.Aspects.HighConcept = "Haunted Former Rancher Seeking Justice"
	player.Aspects.Trouble = "Vengeance Burns Hotter Than Reason"
	player.Aspects.AddAspect("Fastest Draw in Three Counties")
	player.Aspects.AddAspect("I Remember Every Face")
	player.Aspects.AddAspect("The Land Speaks to Me")

	// Set skills - balanced for both social and combat
	player.SetSkill("Shoot", dice.Superb)       // +5 - Primary skill
	player.SetSkill("Notice", dice.Great)       // +4
	player.SetSkill("Empathy", dice.Great)      // +4
	player.SetSkill("Will", dice.Good)          // +3
	player.SetSkill("Athletics", dice.Good)     // +3
	player.SetSkill("Rapport", dice.Good)       // +3
	player.SetSkill("Investigate", dice.Fair)   // +2
	player.SetSkill("Ride", dice.Fair)          // +2 (Drive equivalent for western)
	player.SetSkill("Fight", dice.Fair)         // +2
	player.SetSkill("Provoke", dice.Fair)       // +2
	player.SetSkill("Physique", dice.Average)   // +1
	player.SetSkill("Stealth", dice.Average)    // +1
	player.SetSkill("Contacts", dice.Average)   // +1
	player.SetSkill("Deceive", dice.Average)    // +1
	player.SetSkill("Resources", dice.Mediocre) // +0

	// Set fate points and refresh
	player.FatePoints = 3
	player.Refresh = 3

	return player
}

func createBartender() *character.Character {
	// Supporting NPC - not immediately hostile, has information to trade
	bartender := character.NewSupportingNPC("bartender", "Maggie Two-Rivers", "Shrewd Saloon Owner with Secrets")
	bartender.Aspects.AddAspect("I've Heard Every Story in This Town")
	bartender.Aspects.AddAspect("Nobody Crosses Me Twice")

	bartender.SetSkill("Empathy", dice.Good)    // +3 - reads people well
	bartender.SetSkill("Rapport", dice.Good)    // +3 - knows how to talk
	bartender.SetSkill("Notice", dice.Fair)     // +2
	bartender.SetSkill("Will", dice.Fair)       // +2
	bartender.SetSkill("Deceive", dice.Average) // +1
	bartender.SetSkill("Shoot", dice.Average)   // +1 - keeps a shotgun under the bar

	bartender.FatePoints = 2

	return bartender
}
