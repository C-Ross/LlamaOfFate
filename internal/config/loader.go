// Package config loads scenario and character definitions from YAML files.
//
// YAML files live under configs/scenarios/ and configs/characters/. The loader
// converts them into the core domain types (scene.Scenario, character.Character,
// etc.) used by the rest of the codebase.
//
// Character YAML files unmarshal directly into character.Character (which carries
// yaml struct tags). The loader then calls InitDefaults() to set up runtime
// fields (stress tracks, timestamps, etc.) that are not stored on disk.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// ---------- YAML data shapes -------------------------------------------------
//
// Character YAML unmarshals directly into character.Character — no wrapper needed.
// The types below cover scenario-level structures that have no direct domain equivalent.

// ScenarioFile is the on-disk YAML shape for a scenario.
type ScenarioFile struct {
	ID             string       `yaml:"id"`
	Title          string       `yaml:"title"`
	Genre          string       `yaml:"genre"`
	Description    string       `yaml:"description"`
	Problem        string       `yaml:"problem"`
	Setting        string       `yaml:"setting"`
	StoryQuestions []string     `yaml:"story_questions"`
	DefaultPlayer  string       `yaml:"default_player"`
	NPCs           []NPCDef     `yaml:"npcs"`
	InitialScene   *scene.Scene `yaml:"initial_scene"`
	Farewell       string       `yaml:"farewell"`
}

// NPCDef describes an NPC in the scenario YAML.
type NPCDef struct {
	ID           string         `yaml:"id"`
	Name         string         `yaml:"name"`
	Type         string         `yaml:"type"` // supporting, nameless_good, nameless_fair, nameless_average, main
	HighConcept  string         `yaml:"high_concept"`
	Aspects      []string       `yaml:"aspects"`
	PrimarySkill string         `yaml:"primary_skill"` // required for nameless NPCs
	Skills       map[string]int `yaml:"skills"`
	FatePoints   int            `yaml:"fate_points"`
}

// ---------- Loaded result types -----------------------------------------------

// LoadedScenario is the fully-resolved result of loading a scenario YAML,
// including the referenced default player character.
type LoadedScenario struct {
	// Raw YAML data (useful for metadata like Description, DefaultPlayer ID).
	Raw ScenarioFile

	// Converted domain objects.
	Scenario *scene.Scenario
	Player   *character.Character // nil when DefaultPlayer is empty
	NPCs     []*character.Character
	Scene    *scene.Scene // nil when InitialScene is absent
	Farewell string       // scenario-level farewell message
}

// ---------- Public loader functions -------------------------------------------

// LoadCharacter reads a single character YAML file and returns a Character.
// The YAML is unmarshaled directly into character.Character, then InitDefaults
// is called to set up stress tracks and other runtime fields.
func LoadCharacter(path string) (*character.Character, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read character file %s: %w", path, err)
	}
	var c character.Character
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse character file %s: %w", path, err)
	}
	c.InitDefaults()
	return &c, nil
}

// LoadCharacters reads all .yaml files in a directory and returns a map keyed
// by character ID.
func LoadCharacters(dir string) (map[string]*character.Character, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read characters directory %s: %w", dir, err)
	}
	chars := make(map[string]*character.Character)
	for _, e := range entries {
		if e.IsDir() || !isYAML(e.Name()) {
			continue
		}
		c, err := LoadCharacter(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		chars[c.ID] = c
	}
	return chars, nil
}

// LoadScenarioFile reads a single scenario YAML.
func LoadScenarioFile(path string) (*ScenarioFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scenario file %s: %w", path, err)
	}
	var sf ScenarioFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parse scenario file %s: %w", path, err)
	}
	return &sf, nil
}

// LoadScenario reads a scenario YAML and resolves its DefaultPlayer from the
// provided character map. Pass nil for characters if you don't need player
// resolution.
func LoadScenario(path string, characters map[string]*character.Character) (*LoadedScenario, error) {
	sf, err := LoadScenarioFile(path)
	if err != nil {
		return nil, err
	}
	return resolveScenario(sf, characters)
}

// LoadAll loads every scenario and character from the given config root
// (e.g. "configs"). It expects subdirectories "scenarios/" and "characters/".
func LoadAll(configRoot string) (map[string]*LoadedScenario, error) {
	charDir := filepath.Join(configRoot, "characters")
	scenDir := filepath.Join(configRoot, "scenarios")

	characters, err := LoadCharacters(charDir)
	if err != nil {
		return nil, fmt.Errorf("load characters: %w", err)
	}

	entries, err := os.ReadDir(scenDir)
	if err != nil {
		return nil, fmt.Errorf("read scenarios directory %s: %w", scenDir, err)
	}

	scenarios := make(map[string]*LoadedScenario)
	for _, e := range entries {
		if e.IsDir() || !isYAML(e.Name()) {
			continue
		}
		ls, err := LoadScenario(filepath.Join(scenDir, e.Name()), characters)
		if err != nil {
			return nil, err
		}
		scenarios[ls.Raw.ID] = ls
	}
	return scenarios, nil
}

// ---------- Internal helpers --------------------------------------------------

func resolveScenario(sf *ScenarioFile, characters map[string]*character.Character) (*LoadedScenario, error) {
	ls := &LoadedScenario{Raw: *sf}

	// Build core Scenario
	ls.Scenario = &scene.Scenario{
		Title:          sf.Title,
		Problem:        sf.Problem,
		StoryQuestions: sf.StoryQuestions,
		Setting:        sf.Setting,
		Genre:          sf.Genre,
	}

	// Resolve default player
	if sf.DefaultPlayer != "" {
		if characters == nil {
			return nil, fmt.Errorf("scenario %q references player %q but no characters were loaded", sf.ID, sf.DefaultPlayer)
		}
		p, ok := characters[sf.DefaultPlayer]
		if !ok {
			return nil, fmt.Errorf("scenario %q references unknown player %q", sf.ID, sf.DefaultPlayer)
		}
		ls.Player = p
	}

	// Build NPCs
	for _, nd := range sf.NPCs {
		npc, err := buildNPC(nd)
		if err != nil {
			return nil, fmt.Errorf("scenario %q npc %q: %w", sf.ID, nd.ID, err)
		}
		ls.NPCs = append(ls.NPCs, npc)
	}

	// Build initial scene
	if sf.InitialScene != nil {
		sf.InitialScene.InitDefaults()
		ls.Scene = sf.InitialScene
	}
	ls.Farewell = sf.Farewell

	return ls, nil
}

func buildNPC(nd NPCDef) (*character.Character, error) {
	var npc *character.Character

	switch nd.Type {
	case "supporting":
		npc = character.NewSupportingNPC(nd.ID, nd.Name, nd.HighConcept)
	case "nameless_good":
		if nd.PrimarySkill == "" {
			return nil, fmt.Errorf("nameless NPC requires primary_skill")
		}
		npc = character.NewNamelessNPC(nd.ID, nd.Name, character.CharacterTypeNamelessGood, nd.PrimarySkill)
		npc.Aspects.HighConcept = nd.HighConcept
	case "nameless_fair":
		if nd.PrimarySkill == "" {
			return nil, fmt.Errorf("nameless NPC requires primary_skill")
		}
		npc = character.NewNamelessNPC(nd.ID, nd.Name, character.CharacterTypeNamelessFair, nd.PrimarySkill)
		npc.Aspects.HighConcept = nd.HighConcept
	case "nameless_average":
		if nd.PrimarySkill == "" {
			return nil, fmt.Errorf("nameless NPC requires primary_skill")
		}
		npc = character.NewNamelessNPC(nd.ID, nd.Name, character.CharacterTypeNamelessAverage, nd.PrimarySkill)
		npc.Aspects.HighConcept = nd.HighConcept
	case "main":
		npc = character.NewMainNPC(nd.ID, nd.Name)
		npc.Aspects.HighConcept = nd.HighConcept
	default:
		return nil, fmt.Errorf("unknown NPC type %q", nd.Type)
	}

	for _, a := range nd.Aspects {
		npc.Aspects.AddAspect(a)
	}
	for skill, level := range nd.Skills {
		npc.SetSkill(skill, dice.Ladder(level))
	}
	if nd.FatePoints > 0 {
		npc.FatePoints = nd.FatePoints
	}

	return npc, nil
}

func isYAML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}
