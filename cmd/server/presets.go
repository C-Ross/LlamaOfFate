package main

import (
	"fmt"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/ui/web"
)

// presetEntry bundles a scenario with its matching default player character.
type presetEntry struct {
	Scenario *scene.Scenario
	Player   *character.Character
	Meta     web.ScenarioPreset // Wire-format metadata for the setup screen
}

// presetRegistry maps preset IDs to their full configurations. The three
// entries correspond to the predefined scenarios in scene.PredefinedScenario.
var presetRegistry = map[string]presetEntry{
	"saloon": {
		Scenario: scene.PredefinedScenario("saloon"),
		Player:   westernPlayer(),
		Meta: web.ScenarioPreset{
			ID:          "saloon",
			Title:       "Trouble in Redemption Gulch",
			Genre:       "Western",
			Description: "Outlaws threaten a frontier town. Take justice into your own hands.",
		},
	},
	"heist": {
		Scenario: scene.PredefinedScenario("heist"),
		Player:   cyberpunkPlayer(),
		Meta: web.ScenarioPreset{
			ID:          "heist",
			Title:       "The Prometheus Job",
			Genre:       "Cyberpunk",
			Description: "Extract a data core from a corporate fortress. Trust no one.",
		},
	},
	"tower": {
		Scenario: scene.PredefinedScenario("tower"),
		Player:   fantasyPlayer(),
		Meta: web.ScenarioPreset{
			ID:          "tower",
			Title:       "The Wizard's Tower",
			Genre:       "Fantasy",
			Description: "Investigate a magical disturbance in an ancient tower.",
		},
	},
}

// allPresetMeta returns the list of ScenarioPreset metadata in a stable order
// suitable for sending to the client.
func allPresetMeta() []web.ScenarioPreset {
	order := []string{"saloon", "heist", "tower"}
	out := make([]web.ScenarioPreset, 0, len(order))
	for _, id := range order {
		out = append(out, presetRegistry[id].Meta)
	}
	return out
}

// lookupPreset returns the scenario and player for a given preset ID,
// or an error if the ID is not recognized.
func lookupPreset(id string) (*scene.Scenario, *character.Character, error) {
	entry, ok := presetRegistry[id]
	if !ok {
		return nil, nil, fmt.Errorf("unknown preset: %q", id)
	}
	return entry.Scenario, entry.Player, nil
}

// --- Preset player characters -------------------------------------------------

// westernPlayer returns the default western player character.
// Skill pyramid: 1 Great, 2 Good, 3 Fair, 4 Average.
func westernPlayer() *character.Character {
	player := character.NewCharacter("player1", "Jesse Calhoun")
	player.Aspects.HighConcept = "Haunted Former Rancher Seeking Justice"
	player.Aspects.Trouble = "Vengeance Burns Hotter Than Reason"
	player.SetSkill("Shoot", dice.Great)
	player.SetSkill("Athletics", dice.Good)
	player.SetSkill("Notice", dice.Good)
	player.SetSkill("Fight", dice.Fair)
	player.SetSkill("Will", dice.Fair)
	player.SetSkill("Drive", dice.Fair)
	player.SetSkill("Physique", dice.Average)
	player.SetSkill("Provoke", dice.Average)
	player.SetSkill("Stealth", dice.Average)
	player.SetSkill("Investigate", dice.Average)
	return player
}

// cyberpunkPlayer returns the default cyberpunk player character.
// Skill pyramid: 1 Great, 2 Good, 3 Fair, 4 Average.
func cyberpunkPlayer() *character.Character {
	player := character.NewCharacter("player1", "Zero")
	player.Aspects.HighConcept = "Chrome-Enhanced Freelance Hacker"
	player.Aspects.Trouble = "Corporate Enemies With Long Memories"
	player.SetSkill("Burglary", dice.Great)
	player.SetSkill("Stealth", dice.Good)
	player.SetSkill("Notice", dice.Good)
	player.SetSkill("Shoot", dice.Fair)
	player.SetSkill("Athletics", dice.Fair)
	player.SetSkill("Deceive", dice.Fair)
	player.SetSkill("Investigate", dice.Average)
	player.SetSkill("Will", dice.Average)
	player.SetSkill("Contacts", dice.Average)
	player.SetSkill("Lore", dice.Average)
	return player
}

// fantasyPlayer returns the default fantasy player character.
// Skill pyramid: 1 Great, 2 Good, 3 Fair, 4 Average.
func fantasyPlayer() *character.Character {
	player := character.NewCharacter("player1", "Lyra Moonwhisper")
	player.Aspects.HighConcept = "Scholarly Arcane Investigator"
	player.Aspects.Trouble = "Curiosity About Forbidden Knowledge"
	player.SetSkill("Lore", dice.Great)
	player.SetSkill("Investigate", dice.Good)
	player.SetSkill("Notice", dice.Good)
	player.SetSkill("Will", dice.Fair)
	player.SetSkill("Rapport", dice.Fair)
	player.SetSkill("Athletics", dice.Fair)
	player.SetSkill("Empathy", dice.Average)
	player.SetSkill("Stealth", dice.Average)
	player.SetSkill("Shoot", dice.Average)
	player.SetSkill("Physique", dice.Average)
	return player
}

// buildCustomPlayer creates a player character from custom setup data with
// a default skill pyramid. The genre determines which skills are selected.
func buildCustomPlayer(name, highConcept, trouble, genre string) *character.Character {
	player := character.NewCharacter("player1", name)
	player.Aspects.HighConcept = highConcept
	player.Aspects.Trouble = trouble
	applyDefaultSkillPyramid(player, genre)
	return player
}

// applyDefaultSkillPyramid assigns a genre-appropriate default skill pyramid.
func applyDefaultSkillPyramid(player *character.Character, genre string) {
	switch genre {
	case "Cyberpunk":
		player.SetSkill("Burglary", dice.Great)
		player.SetSkill("Stealth", dice.Good)
		player.SetSkill("Notice", dice.Good)
		player.SetSkill("Shoot", dice.Fair)
		player.SetSkill("Athletics", dice.Fair)
		player.SetSkill("Deceive", dice.Fair)
		player.SetSkill("Investigate", dice.Average)
		player.SetSkill("Will", dice.Average)
		player.SetSkill("Contacts", dice.Average)
		player.SetSkill("Lore", dice.Average)
	case "Fantasy":
		player.SetSkill("Lore", dice.Great)
		player.SetSkill("Investigate", dice.Good)
		player.SetSkill("Notice", dice.Good)
		player.SetSkill("Will", dice.Fair)
		player.SetSkill("Rapport", dice.Fair)
		player.SetSkill("Athletics", dice.Fair)
		player.SetSkill("Empathy", dice.Average)
		player.SetSkill("Stealth", dice.Average)
		player.SetSkill("Shoot", dice.Average)
		player.SetSkill("Physique", dice.Average)
	default:
		// Western / generic fallback
		player.SetSkill("Shoot", dice.Great)
		player.SetSkill("Athletics", dice.Good)
		player.SetSkill("Notice", dice.Good)
		player.SetSkill("Fight", dice.Fair)
		player.SetSkill("Will", dice.Fair)
		player.SetSkill("Investigate", dice.Fair)
		player.SetSkill("Physique", dice.Average)
		player.SetSkill("Provoke", dice.Average)
		player.SetSkill("Stealth", dice.Average)
		player.SetSkill("Rapport", dice.Average)
	}
}
