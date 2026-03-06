package main

// Scenario and player character loading from YAML config files.

import (
	"log"

	"github.com/C-Ross/LlamaOfFate/internal/config"
	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

const defaultScenarioID = "saloon"

// loadedScenarios caches scenarios loaded from YAML at startup.
var loadedScenarios map[string]*config.LoadedScenario

func init() {
	loaded, err := config.LoadAll("configs")
	if err != nil {
		log.Fatalf("failed to load scenarios from YAML: %v", err)
	}
	loadedScenarios = loaded
}

// defaultScenario returns the default scenario (saloon).
func defaultScenario() *scene.Scenario {
	return loadedScenarios[defaultScenarioID].Scenario
}

// defaultPlayer returns the default player character for the saloon scenario.
func defaultPlayer() *core.Character {
	return loadedScenarios[defaultScenarioID].Player
}
