package main

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// =============================================================================
// SCENE: Megacorp Data Vault (Cyberpunk) - Multiple Enemies + Dangerous Aspect
// =============================================================================

func buildHeistScene() SceneConfig {
	player := createHeistCharacter()

	// Create multiple enemies
	securityDrone1 := createSecurityDrone("drone-1", "Security Drone Alpha")
	securityDrone2 := createSecurityDrone("drone-2", "Security Drone Beta")
	corporateAgent := createCorporateAgent()

	heistScene := scene.NewScene(
		"megacorp-vault",
		"Megacorp Data Vault",
		"You've infiltrated the lower levels of Nexus Industries' headquarters. "+
			"The data vault stretches before you—a cathedral of humming server racks bathed in cold blue light. "+
			"Holographic warning signs flicker: 'LEVEL 5 CLEARANCE REQUIRED.' "+
			"Two hovering security drones patrol the aisles with scanning beams, "+
			"while a black-suited corporate agent reviews a datapad near the central access terminal. "+
			"The air is cold and sterile, and you can hear the distant thrum of the building's fusion reactor. "+
			"Your contact said the prototype AI core data is stored here—worth millions on the black market.",
	)

	// Add a dangerous scene aspect that poses a significant challenge
	heistScene.AddSituationAspect(scene.NewSituationAspect(
		"biometric-lockdown",
		"Biometric Lockdown Protocols Active",
		"scene",
		0,
	))

	return SceneConfig{
		Name:        "Megacorp Data Vault",
		Description: "A cyberpunk heist scenario with multiple enemies",
		Player:      player,
		NPCs:        []*character.Character{securityDrone1, securityDrone2, corporateAgent},
		Scene:       heistScene,
		Farewell:    "You slip back into the neon-lit streets. The data burns in your pocket.",
	}
}

func createHeistCharacter() *character.Character {
	player := character.NewCharacter("player1", "Zero")

	// Set aspects for a cyberpunk infiltrator
	player.Aspects.HighConcept = "Ghost in the Machine Netrunner"
	player.Aspects.Trouble = "Wanted by Three Megacorps"
	player.Aspects.AddAspect("Military-Grade Cybernetic Reflexes")
	player.Aspects.AddAspect("Nobody Gets Left Behind")
	player.Aspects.AddAspect("I Know a Guy for Everything")

	// Set skills focused on stealth and tech
	player.SetSkill("Burglary", dice.Superb)   // +5 - Primary skill (hacking/infiltration)
	player.SetSkill("Stealth", dice.Great)     // +4
	player.SetSkill("Notice", dice.Great)      // +4
	player.SetSkill("Crafts", dice.Good)       // +3 - Tech skills
	player.SetSkill("Athletics", dice.Good)    // +3
	player.SetSkill("Shoot", dice.Good)        // +3
	player.SetSkill("Deceive", dice.Fair)      // +2
	player.SetSkill("Will", dice.Fair)         // +2
	player.SetSkill("Investigate", dice.Fair)  // +2
	player.SetSkill("Contacts", dice.Fair)     // +2
	player.SetSkill("Fight", dice.Average)     // +1
	player.SetSkill("Physique", dice.Average)  // +1
	player.SetSkill("Provoke", dice.Average)   // +1
	player.SetSkill("Resources", dice.Average) // +1
	player.SetSkill("Empathy", dice.Mediocre)  // +0

	// Set fate points and refresh
	player.FatePoints = 3
	player.Refresh = 3

	return player
}

func createSecurityDrone(id, name string) *character.Character {
	// Fair (+2) nameless NPCs - basic security drones
	drone := character.NewNamelessNPC(id, name, character.CharacterTypeNamelessFair, "Shoot")
	drone.Aspects.HighConcept = "Automated Threat Response Unit"
	drone.SetSkill("Notice", dice.Average) // +1

	return drone
}

func createCorporateAgent() *character.Character {
	// Supporting NPC - more dangerous than nameless drones
	agent := character.NewSupportingNPC("corp-agent", "Agent Chen", "Nexus Industries Troubleshooter")
	agent.Aspects.AddAspect("Augmented Combat Implants")

	agent.SetSkill("Fight", dice.Good)       // +3
	agent.SetSkill("Shoot", dice.Good)       // +3
	agent.SetSkill("Notice", dice.Fair)      // +2
	agent.SetSkill("Athletics", dice.Fair)   // +2
	agent.SetSkill("Will", dice.Average)     // +1
	agent.SetSkill("Physique", dice.Average) // +1

	agent.FatePoints = 2

	return agent
}
