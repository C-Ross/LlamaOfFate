package main

// Scenario and player character loading from YAML config files.
// Falls back to hardcoded defaults if configs are unavailable.

import (
	"log/slog"

	"github.com/C-Ross/LlamaOfFate/internal/config"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

const defaultScenarioID = "saloon"

// loadedScenarios caches scenarios loaded from YAML at startup.
var loadedScenarios map[string]*config.LoadedScenario

func init() {
	loaded, err := config.LoadAll("configs")
	if err != nil {
		slog.Warn("failed to load scenarios from YAML, will use hardcoded defaults", "error", err)
		return
	}
	loadedScenarios = loaded
}

// defaultScenario returns the default scenario (saloon).
func defaultScenario() *scene.Scenario {
	if ls, ok := loadedScenarios[defaultScenarioID]; ok {
		return ls.Scenario
	}
	slog.Warn("falling back to hardcoded default scenario")
	return &scene.Scenario{
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

// defaultPlayer returns the default player character for the saloon scenario.
func defaultPlayer() *character.Character {
	if ls, ok := loadedScenarios[defaultScenarioID]; ok && ls.Player != nil {
		return ls.Player
	}
	slog.Warn("falling back to hardcoded default player")
	player := character.NewCharacter("player1", "Jesse Calhoun")
	player.Aspects.HighConcept = "Haunted Former Rancher Seeking Justice"
	player.Aspects.Trouble = "Vengeance Burns Hotter Than Reason"
	return player
}
