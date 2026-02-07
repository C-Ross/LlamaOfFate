package main

// Hardcoded scenario and player character for development/testing.
// Replace this file with scenario generation or selection logic later.

import (
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
)

// defaultScenario returns the hardcoded scenario used for testing.
func defaultScenario() *prompt.Scenario {
	return &prompt.Scenario{
		Title:   "Trouble in Redemption Gulch",
		Genre:   "Western",
		Problem: "The town is under threat from outlaws and someone needs to stand up for the innocent",
		StoryQuestions: []string{
			"Will the outlaws be brought to justice?",
			"Can the town be saved?",
		},
		Setting: "The American Old West in the late 1800s. Dusty frontier towns, " +
			"lawless territories, and the struggle between civilization and the wild. " +
			"Gunslingers, outlaws, and honest folk all seeking their fortune.",
	}
}

// defaultPlayer returns the hardcoded player character used for testing.
// Skill pyramid follows Fate Core defaults: 1 Great, 2 Good, 3 Fair, 4 Average.
func defaultPlayer() *character.Character {
	player := character.NewCharacter("player1", "Jesse Calhoun")
	player.Aspects.HighConcept = "Haunted Former Rancher Seeking Justice"
	player.Aspects.Trouble = "Vengeance Burns Hotter Than Reason"

	// Great (+4)
	player.SetSkill("Shoot", dice.Great)
	// Good (+3)
	player.SetSkill("Athletics", dice.Good)
	player.SetSkill("Notice", dice.Good)
	// Fair (+2)
	player.SetSkill("Fight", dice.Fair)
	player.SetSkill("Will", dice.Fair)
	player.SetSkill("Drive", dice.Fair) // Horsemanship in this setting
	// Average (+1)
	player.SetSkill("Physique", dice.Average)
	player.SetSkill("Provoke", dice.Average)
	player.SetSkill("Stealth", dice.Average)
	player.SetSkill("Investigate", dice.Average)

	return player
}
