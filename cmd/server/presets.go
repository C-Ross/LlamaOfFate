package main

import (
	"fmt"
	"log/slog"

	"github.com/C-Ross/LlamaOfFate/internal/config"
	"github.com/C-Ross/LlamaOfFate/internal/core"
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

// presetOrder defines the display order for presets.
var presetOrder = []string{"saloon", "heist", "tower"}

// loadedPresets caches presets loaded from YAML at startup.
var loadedPresets map[string]*config.LoadedScenario

func init() {
	loaded, err := config.LoadAll("configs")
	if err != nil {
		slog.Warn("failed to load scenarios from YAML, presets will be empty", "error", err)
		return
	}
	loadedPresets = loaded
}

// presetRegistry returns the preset entry for the given ID, building it from
// the loaded YAML data.
func getPreset(id string) (presetEntry, bool) {
	ls, ok := loadedPresets[id]
	if !ok {
		return presetEntry{}, false
	}
	return presetEntry{
		Scenario: ls.Scenario,
		Player:   ls.Player,
		Meta: web.ScenarioPreset{
			ID:          ls.Raw.ID,
			Title:       ls.Raw.Title,
			Genre:       ls.Raw.Genre,
			Description: ls.Raw.Description,
		},
	}, true
}

// allPresetMeta returns the list of ScenarioPreset metadata in a stable order
// suitable for sending to the client.
func allPresetMeta() []web.ScenarioPreset {
	out := make([]web.ScenarioPreset, 0, len(presetOrder))
	for _, id := range presetOrder {
		if p, ok := getPreset(id); ok {
			out = append(out, p.Meta)
		}
	}
	return out
}

// lookupPreset returns the scenario and player for a given preset ID,
// or an error if the ID is not recognized.
func lookupPreset(id string) (*scene.Scenario, *character.Character, error) {
	p, ok := getPreset(id)
	if !ok {
		return nil, nil, fmt.Errorf("unknown preset: %q", id)
	}
	return p.Scenario, p.Player, nil
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
		player.SetSkill(core.SkillBurglary, dice.Great)
		player.SetSkill(core.SkillStealth, dice.Good)
		player.SetSkill(core.SkillNotice, dice.Good)
		player.SetSkill(core.SkillShoot, dice.Fair)
		player.SetSkill(core.SkillAthletics, dice.Fair)
		player.SetSkill(core.SkillDeceive, dice.Fair)
		player.SetSkill(core.SkillInvestigate, dice.Average)
		player.SetSkill(core.SkillWill, dice.Average)
		player.SetSkill(core.SkillContacts, dice.Average)
		player.SetSkill(core.SkillLore, dice.Average)
	case "Fantasy":
		player.SetSkill(core.SkillLore, dice.Great)
		player.SetSkill(core.SkillInvestigate, dice.Good)
		player.SetSkill(core.SkillNotice, dice.Good)
		player.SetSkill(core.SkillWill, dice.Fair)
		player.SetSkill(core.SkillRapport, dice.Fair)
		player.SetSkill(core.SkillAthletics, dice.Fair)
		player.SetSkill(core.SkillEmpathy, dice.Average)
		player.SetSkill(core.SkillStealth, dice.Average)
		player.SetSkill(core.SkillShoot, dice.Average)
		player.SetSkill(core.SkillPhysique, dice.Average)
	default:
		// Western / generic fallback
		player.SetSkill(core.SkillShoot, dice.Great)
		player.SetSkill(core.SkillAthletics, dice.Good)
		player.SetSkill(core.SkillNotice, dice.Good)
		player.SetSkill(core.SkillFight, dice.Fair)
		player.SetSkill(core.SkillWill, dice.Fair)
		player.SetSkill(core.SkillInvestigate, dice.Fair)
		player.SetSkill(core.SkillPhysique, dice.Average)
		player.SetSkill(core.SkillProvoke, dice.Average)
		player.SetSkill(core.SkillStealth, dice.Average)
		player.SetSkill(core.SkillRapport, dice.Average)
	}
}
