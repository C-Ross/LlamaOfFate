package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
)

// ---------- Character loading ------------------------------------------------

func TestLoadCharacter(t *testing.T) {
	path := writeTemp(t, "char.yaml", `
id: test-char
name: "Test Hero"
aspects:
  high_concept: "Mighty Warrior"
  trouble: "Hot-Headed"
  other:
    - "Loyal Friend"
skills:
  Fight: 4
  Athletics: 3
fate_points: 3
refresh: 3
`)

	c, err := LoadCharacter(path)
	require.NoError(t, err)

	assert.Equal(t, "test-char", c.ID)
	assert.Equal(t, "Test Hero", c.Name)
	assert.Equal(t, "Mighty Warrior", c.Aspects.HighConcept)
	assert.Equal(t, "Hot-Headed", c.Aspects.Trouble)
	assert.Equal(t, []string{"Loyal Friend"}, c.Aspects.OtherAspects)
	assert.Equal(t, dice.Great, c.Skills["Fight"])
	assert.Equal(t, dice.Good, c.Skills["Athletics"])
	assert.Equal(t, 3, c.FatePoints)
	assert.Equal(t, 3, c.Refresh)

	// InitDefaults should have set up stress tracks and timestamps.
	require.NotNil(t, c.StressTracks)
	assert.Contains(t, c.StressTracks, "physical")
	assert.Contains(t, c.StressTracks, "mental")
	assert.False(t, c.CreatedAt.IsZero())
	assert.NotNil(t, c.Consequences)
}

func TestLoadCharacters_Directory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yaml"), `
id: alpha
name: "Alpha"
aspects:
  high_concept: "First"
  trouble: "One"
skills:
  Lore: 5
fate_points: 1
refresh: 1
`)
	writeFile(t, filepath.Join(dir, "b.yaml"), `
id: beta
name: "Beta"
aspects:
  high_concept: "Second"
  trouble: "Two"
skills:
  Fight: 3
fate_points: 2
refresh: 2
`)
	// Non-YAML file should be ignored.
	writeFile(t, filepath.Join(dir, "readme.txt"), "not yaml")

	chars, err := LoadCharacters(dir)
	require.NoError(t, err)
	assert.Len(t, chars, 2)
	assert.Equal(t, "Alpha", chars["alpha"].Name)
	assert.Equal(t, "Beta", chars["beta"].Name)
}

func TestLoadCharacter_MissingFile(t *testing.T) {
	_, err := LoadCharacter("/nonexistent/path.yaml")
	require.Error(t, err)
}

// ---------- Scenario loading -------------------------------------------------

func TestLoadScenario_Minimal(t *testing.T) {
	path := writeTemp(t, "scenario.yaml", `
id: test
title: "Test Scenario"
genre: Fantasy
problem: "Big bad problem"
setting: "A magical land"
story_questions:
  - "Can evil be stopped?"
`)

	ls, err := LoadScenario(path, nil)
	require.NoError(t, err)

	assert.Equal(t, "Test Scenario", ls.Scenario.Title)
	assert.Equal(t, "Fantasy", ls.Scenario.Genre)
	assert.Equal(t, "Big bad problem", ls.Scenario.Problem)
	assert.Equal(t, "A magical land", ls.Scenario.Setting)
	assert.Equal(t, []string{"Can evil be stopped?"}, ls.Scenario.StoryQuestions)
	assert.Nil(t, ls.Player)
	assert.Empty(t, ls.NPCs)
	assert.Nil(t, ls.Scene)
}

func TestLoadScenario_WithPlayer(t *testing.T) {
	dir := t.TempDir()

	charPath := filepath.Join(dir, "hero.yaml")
	writeFile(t, charPath, `
id: hero
name: "Hero"
aspects:
  high_concept: "Great Hero"
  trouble: "Scared of Spiders"
skills:
  Fight: 4
fate_points: 3
refresh: 3
`)

	scenPath := filepath.Join(dir, "scenario.yaml")
	writeFile(t, scenPath, `
id: quest
title: "The Quest"
genre: Fantasy
problem: "Dragons"
setting: "Mountains"
story_questions:
  - "Will dragons fall?"
default_player: hero
`)

	chars, err := LoadCharacters(dir)
	require.NoError(t, err)

	// Load only the scenario file, but resolve the player reference.
	ls, err := LoadScenario(scenPath, chars)
	require.NoError(t, err)

	require.NotNil(t, ls.Player)
	assert.Equal(t, "Hero", ls.Player.Name)
}

func TestLoadScenario_UnknownPlayer(t *testing.T) {
	path := writeTemp(t, "scenario.yaml", `
id: bad
title: "Bad Ref"
genre: Fantasy
problem: "problem"
setting: "setting"
story_questions: []
default_player: nonexistent
`)

	_, err := LoadScenario(path, map[string]*character.Character{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown player")
}

func TestLoadScenario_WithNPCs(t *testing.T) {
	path := writeTemp(t, "scenario.yaml", `
id: npc-test
title: "NPC Test"
genre: Western
problem: "trouble"
setting: "dusty"
story_questions: []
npcs:
  - id: bart
    name: "Bartender"
    type: supporting
    high_concept: "Friendly Barkeep"
    aspects:
      - "Knows Everyone"
    skills:
      Rapport: 3
      Empathy: 2
    fate_points: 2
  - id: guard-1
    name: "Town Guard"
    type: nameless_fair
    high_concept: "Dutiful Watchman"
    primary_skill: Notice
    skills:
      Shoot: 1
  - id: goblin
    name: "Goblin"
    type: nameless_good
    high_concept: "Sneaky"
    primary_skill: Fight
  - id: peasant
    name: "Peasant"
    type: nameless_average
    high_concept: "Frightened Villager"
    primary_skill: Will
`)

	ls, err := LoadScenario(path, nil)
	require.NoError(t, err)
	require.Len(t, ls.NPCs, 4)

	// Supporting NPC
	bart := ls.NPCs[0]
	assert.Equal(t, "Bartender", bart.Name)
	assert.Equal(t, character.CharacterTypeSupportingNPC, bart.CharacterType)
	assert.Equal(t, "Friendly Barkeep", bart.Aspects.HighConcept)
	assert.Equal(t, []string{"Knows Everyone"}, bart.Aspects.OtherAspects)
	assert.Equal(t, dice.Good, bart.Skills["Rapport"])
	assert.Equal(t, 2, bart.FatePoints)

	// Nameless Fair
	guard := ls.NPCs[1]
	assert.Equal(t, character.CharacterTypeNamelessFair, guard.CharacterType)
	assert.Equal(t, dice.Fair, guard.Skills["Notice"])
	assert.Equal(t, dice.Average, guard.Skills["Shoot"])

	// Nameless Good
	goblin := ls.NPCs[2]
	assert.Equal(t, character.CharacterTypeNamelessGood, goblin.CharacterType)
	assert.Equal(t, dice.Good, goblin.Skills["Fight"])

	// Nameless Average
	peasant := ls.NPCs[3]
	assert.Equal(t, character.CharacterTypeNamelessAverage, peasant.CharacterType)
}

func TestLoadScenario_WithScene(t *testing.T) {
	path := writeTemp(t, "scenario.yaml", `
id: scene-test
title: "Scene Test"
genre: Fantasy
problem: "problem"
setting: "setting"
story_questions: []
initial_scene:
  id: tavern
  name: "The Tavern"
  description: "A cozy tavern."
  situation_aspects:
    - id: warm-fire
      aspect: "Roaring Fireplace"
      duration: scene
farewell: "You leave the tavern."
`)

	ls, err := LoadScenario(path, nil)
	require.NoError(t, err)

	require.NotNil(t, ls.Scene)
	assert.Equal(t, "tavern", ls.Scene.ID)
	assert.Equal(t, "The Tavern", ls.Scene.Name)
	assert.Equal(t, "A cozy tavern.", ls.Scene.Description)
	require.Len(t, ls.Scene.SituationAspects, 1)
	assert.Equal(t, "Roaring Fireplace", ls.Scene.SituationAspects[0].Aspect)
	assert.Equal(t, "scene", ls.Scene.SituationAspects[0].Duration)
	assert.Equal(t, "You leave the tavern.", ls.Farewell)

	// InitDefaults should have set up runtime fields.
	assert.NotNil(t, ls.Scene.Characters)
	assert.NotNil(t, ls.Scene.TakenOutCharacters)
	assert.False(t, ls.Scene.CreatedAt.IsZero())
}

func TestLoadScenario_NamelessNPC_MissingPrimarySkill(t *testing.T) {
	path := writeTemp(t, "scenario.yaml", `
id: bad-npc
title: "Bad NPC"
genre: Fantasy
problem: "p"
setting: "s"
story_questions: []
npcs:
  - id: guard
    name: "Guard"
    type: nameless_fair
    high_concept: "Watchful"
`)

	_, err := LoadScenario(path, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "primary_skill")
}

func TestLoadScenario_UnknownNPCType(t *testing.T) {
	path := writeTemp(t, "scenario.yaml", `
id: bad-type
title: "Bad Type"
genre: Fantasy
problem: "p"
setting: "s"
story_questions: []
npcs:
  - id: x
    name: "X"
    type: alien
    high_concept: "Weird"
`)

	_, err := LoadScenario(path, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown NPC type")
}

// ---------- LoadAll ----------------------------------------------------------

func TestLoadAll(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "characters"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios"), 0o755))

	writeFile(t, filepath.Join(root, "characters", "hero.yaml"), `
id: hero
name: "Hero"
aspects:
  high_concept: "Brave"
  trouble: "Reckless"
skills:
  Fight: 4
fate_points: 3
refresh: 3
`)

	writeFile(t, filepath.Join(root, "scenarios", "adventure.yaml"), `
id: adventure
title: "Adventure"
genre: Fantasy
problem: "Evil"
setting: "Land"
story_questions:
  - "Will good triumph?"
default_player: hero
`)

	scenarios, err := LoadAll(root)
	require.NoError(t, err)
	require.Len(t, scenarios, 1)

	ls := scenarios["adventure"]
	require.NotNil(t, ls)
	assert.Equal(t, "Adventure", ls.Scenario.Title)
	require.NotNil(t, ls.Player)
	assert.Equal(t, "Hero", ls.Player.Name)
}

// ---------- Integration: real configs ----------------------------------------

func TestLoadAll_RealConfigs(t *testing.T) {
	// This test loads the actual YAML files from the repo.
	configRoot := filepath.Join("..", "..", "configs")
	if _, err := os.Stat(filepath.Join(configRoot, "scenarios")); err != nil {
		t.Skip("configs directory not found, skipping integration test")
	}

	scenarios, err := LoadAll(configRoot)
	require.NoError(t, err)
	require.Len(t, scenarios, 3, "expected 3 scenarios (saloon, heist, tower)")

	// Saloon
	saloon := scenarios["saloon"]
	require.NotNil(t, saloon)
	assert.Equal(t, "Trouble in Redemption Gulch", saloon.Scenario.Title)
	assert.Equal(t, "Western", saloon.Scenario.Genre)
	require.NotNil(t, saloon.Player)
	assert.Equal(t, "Jesse Calhoun", saloon.Player.Name)
	assert.Equal(t, dice.Superb, saloon.Player.Skills["Shoot"])
	require.Len(t, saloon.NPCs, 1)
	assert.Equal(t, "Maggie Two-Rivers", saloon.NPCs[0].Name)
	require.NotNil(t, saloon.Scene)
	assert.Equal(t, "The Dusty Spur Saloon", saloon.Scene.Name)
	assert.Len(t, saloon.Scene.SituationAspects, 2)

	// Heist
	heist := scenarios["heist"]
	require.NotNil(t, heist)
	assert.Equal(t, "The Prometheus Job", heist.Scenario.Title)
	require.NotNil(t, heist.Player)
	assert.Equal(t, "Zero", heist.Player.Name)
	require.Len(t, heist.NPCs, 3)
	require.NotNil(t, heist.Scene)

	// Tower
	tower := scenarios["tower"]
	require.NotNil(t, tower)
	assert.Equal(t, "The Wizard's Tower", tower.Scenario.Title)
	require.NotNil(t, tower.Player)
	assert.Equal(t, "Lyra Moonwhisper", tower.Player.Name)
	require.Len(t, tower.NPCs, 1)
	require.NotNil(t, tower.Scene)
}

// ---------- Helpers ----------------------------------------------------------

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	writeFile(t, path, content)
	return path
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
