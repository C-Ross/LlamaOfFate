package main

// Scenario and player character loading from YAML config files.

import (
	"log"

	"github.com/C-Ross/LlamaOfFate/internal/config"
	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/ui/terminal"
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

// applySetup applies the CharacterSetup choices to a copy of the preset player.
func applySetup(preset *core.Character, setup terminal.CharacterSetup) *core.Character {
	player := core.NewCharacter(preset.ID, setup.Name)
	player.Aspects.HighConcept = setup.HighConcept
	player.Aspects.Trouble = setup.Trouble
	for _, a := range setup.Aspects {
		player.Aspects.AddAspect(a)
	}
	player.FatePoints = preset.FatePoints
	player.Refresh = preset.Refresh

	// Apply skills (triggers stress track recalculation via SetSkill)
	for name, level := range setup.Skills {
		player.SetSkill(name, level)
	}
	return player
}
