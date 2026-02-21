package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Constructor / Interface ---

func TestNewYAMLSaver(t *testing.T) {
	saver := NewYAMLSaver("/tmp/test-saves")
	require.NotNil(t, saver)
	assert.Equal(t, "/tmp/test-saves", saver.dir)
}

func TestYAMLSaver_ImplementsInterface(t *testing.T) {
	var _ engine.GameStateSaver = (*YAMLSaver)(nil)
}

// --- Load with no save file ---

func TestYAMLSaver_Load_NoFile(t *testing.T) {
	dir := t.TempDir()
	saver := NewYAMLSaver(dir)

	state, err := saver.Load()
	assert.NoError(t, err)
	assert.Nil(t, state, "Load should return nil when no save file exists")
}

// validMinimalState returns a GameState with the minimum required fields to
// pass Validate(). Tests that don't care about specific field values should
// start from this and override what they need.
func validMinimalState() engine.GameState {
	player := character.NewCharacter("player1", "Test Hero")
	player.Aspects.HighConcept = "Test Concept"
	player.Aspects.Trouble = "Test Trouble"
	return engine.GameState{
		Scenario: engine.ScenarioState{
			Player:   player,
			Scenario: &scene.Scenario{Title: "Test", Genre: "Fantasy"},
		},
		Scene: engine.SceneState{
			CurrentScene: scene.NewScene("s1", "Test Room", "A test room"),
		},
	}
}

// --- Minimal round-trip ---

func TestYAMLSaver_RoundTrip_Minimal(t *testing.T) {
	dir := t.TempDir()
	saver := NewYAMLSaver(dir)

	original := validMinimalState()
	original.Scenario.ScenarioCount = 1
	original.Scenario.SceneCount = 2

	require.NoError(t, saver.Save(original))

	loaded, err := saver.Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, 1, loaded.Scenario.ScenarioCount)
	assert.Equal(t, 2, loaded.Scenario.SceneCount)
}

// --- Full round-trip with realistic game state ---

func TestYAMLSaver_RoundTrip_FullState(t *testing.T) {
	dir := t.TempDir()
	saver := NewYAMLSaver(dir)

	player := character.NewCharacter("player1", "Jesse Calhoun")
	player.Aspects.HighConcept = "Gunslinger With a Past"
	player.Aspects.Trouble = "Wanted Dead or Alive"
	player.Aspects.AddAspect("Quick Draw")
	player.SetSkill("Shoot", dice.Great)
	player.SetSkill("Notice", dice.Fair)
	player.FatePoints = 5

	bartender := character.NewCharacter("npc_bartender", "Old Pete")
	bartender.Aspects.HighConcept = "Grizzled Barkeep"
	bartender.CharacterType = character.CharacterTypeSupportingNPC

	testScene := scene.NewScene("saloon_1", "The Dusty Saloon", "A dimly lit saloon at the edge of town.")
	testScene.AddSituationAspect(scene.SituationAspect{
		ID: "aspect_1", Aspect: "Smoky Atmosphere", Duration: "scene",
	})

	original := engine.GameState{
		Scenario: engine.ScenarioState{
			Player: player,
			Scenario: &scene.Scenario{
				Title:          "The Showdown at High Noon",
				Problem:        "A gang of outlaws threatens the town",
				StoryQuestions: []string{"Will Jesse protect the town?", "Can the outlaws be stopped?"},
				Setting:        "Dusty frontier town",
				Genre:          "Western",
			},
			ScenarioCount: 1,
			SceneCount:    3,
			SceneSummaries: []prompt.SceneSummary{
				{
					SceneDescription:  "The dusty saloon",
					KeyEvents:         []string{"Met the bartender", "Overheard a conversation"},
					NPCsEncountered:   []prompt.NPCSummary{{Name: "Old Pete", Attitude: "friendly"}},
					AspectsDiscovered: []string{"Hidden Door Behind the Bar"},
					UnresolvedThreads: []string{"Who is the mysterious stranger?"},
					HowEnded:          "transition",
					NarrativeProse:    "Jesse pushed through the swinging doors...",
				},
			},
			NPCRegistry:  map[string]*character.Character{"old pete": bartender},
			NPCAttitudes: map[string]string{"old pete": "friendly"},
			LastPurpose:  "Find the informant",
			LastHook:     "The bartender gestures you over",
		},
		Scene: engine.SceneState{
			CurrentScene: testScene,
			ConversationHistory: []prompt.ConversationEntry{
				{
					PlayerInput: "I look around the saloon",
					GMResponse:  "You see a bartender polishing glasses",
					Timestamp:   time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
					Type:        "dialog",
				},
				{
					PlayerInput: "I approach the bar",
					GMResponse:  "The bartender eyes you warily",
					Timestamp:   time.Date(2026, 2, 10, 12, 1, 0, 0, time.UTC),
					Type:        "dialog",
				},
			},
			ScenePurpose: "Find the informant",
		},
	}

	require.NoError(t, saver.Save(original))

	loaded, err := saver.Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// --- Verify scenario-level state ---
	assert.Equal(t, "Jesse Calhoun", loaded.Scenario.Player.Name)
	assert.Equal(t, "Gunslinger With a Past", loaded.Scenario.Player.Aspects.HighConcept)
	assert.Equal(t, "Wanted Dead or Alive", loaded.Scenario.Player.Aspects.Trouble)
	assert.Contains(t, loaded.Scenario.Player.Aspects.OtherAspects, "Quick Draw")
	assert.Equal(t, 5, loaded.Scenario.Player.FatePoints)
	assert.Equal(t, dice.Great, loaded.Scenario.Player.GetSkill("Shoot"))
	assert.Equal(t, dice.Fair, loaded.Scenario.Player.GetSkill("Notice"))

	assert.Equal(t, "The Showdown at High Noon", loaded.Scenario.Scenario.Title)
	assert.Equal(t, "Western", loaded.Scenario.Scenario.Genre)
	assert.Len(t, loaded.Scenario.Scenario.StoryQuestions, 2)
	assert.False(t, loaded.Scenario.Scenario.IsResolved)

	assert.Equal(t, 1, loaded.Scenario.ScenarioCount)
	assert.Equal(t, 3, loaded.Scenario.SceneCount)

	assert.Equal(t, "Find the informant", loaded.Scenario.LastPurpose)
	assert.Equal(t, "The bartender gestures you over", loaded.Scenario.LastHook)

	// NPC registry
	require.Contains(t, loaded.Scenario.NPCRegistry, "old pete")
	assert.Equal(t, "Grizzled Barkeep", loaded.Scenario.NPCRegistry["old pete"].Aspects.HighConcept)

	// NPC attitudes
	assert.Equal(t, "friendly", loaded.Scenario.NPCAttitudes["old pete"])

	// Scene summaries
	require.Len(t, loaded.Scenario.SceneSummaries, 1)
	assert.Equal(t, "The dusty saloon", loaded.Scenario.SceneSummaries[0].SceneDescription)
	assert.Contains(t, loaded.Scenario.SceneSummaries[0].KeyEvents, "Met the bartender")
	assert.Len(t, loaded.Scenario.SceneSummaries[0].NPCsEncountered, 1)
	assert.Equal(t, "Old Pete", loaded.Scenario.SceneSummaries[0].NPCsEncountered[0].Name)

	// --- Verify scene-level state ---
	require.NotNil(t, loaded.Scene.CurrentScene)
	assert.Equal(t, "The Dusty Saloon", loaded.Scene.CurrentScene.Name)
	assert.Contains(t, loaded.Scene.CurrentScene.Description, "dimly lit saloon")
	assert.Equal(t, "Find the informant", loaded.Scene.ScenePurpose)

	// Conversation history
	require.Len(t, loaded.Scene.ConversationHistory, 2)
	assert.Equal(t, "I look around the saloon", loaded.Scene.ConversationHistory[0].PlayerInput)
	assert.Equal(t, "I approach the bar", loaded.Scene.ConversationHistory[1].PlayerInput)
}

// --- Save overwrites previous save ---

func TestYAMLSaver_Save_Overwrites(t *testing.T) {
	dir := t.TempDir()
	saver := NewYAMLSaver(dir)

	state1 := validMinimalState()
	state1.Scenario.ScenarioCount = 1

	state2 := validMinimalState()
	state2.Scenario.ScenarioCount = 2

	require.NoError(t, saver.Save(state1))
	require.NoError(t, saver.Save(state2))

	loaded, err := saver.Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, 2, loaded.Scenario.ScenarioCount, "second save should overwrite first")
}

// --- Creates directory on save ---

func TestYAMLSaver_Save_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "save", "dir")
	saver := NewYAMLSaver(dir)

	state := engine.GameState{
		Scenario: engine.ScenarioState{ScenarioCount: 1},
	}

	require.NoError(t, saver.Save(state))

	// Verify file exists
	_, err := os.Stat(filepath.Join(dir, saveFileName))
	assert.NoError(t, err, "save file should exist after Save")
}

// --- Delete ---

func TestYAMLSaver_Delete(t *testing.T) {
	dir := t.TempDir()
	saver := NewYAMLSaver(dir)

	// Save then delete
	require.NoError(t, saver.Save(engine.GameState{}))
	require.NoError(t, saver.Delete())

	// Load should return nil after delete
	loaded, err := saver.Load()
	assert.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestYAMLSaver_Delete_NoFile(t *testing.T) {
	dir := t.TempDir()
	saver := NewYAMLSaver(dir)

	err := saver.Delete()
	assert.NoError(t, err, "deleting non-existent file should not error")
}

// --- Path ---

func TestYAMLSaver_Path(t *testing.T) {
	saver := NewYAMLSaver("/home/user/.llamaoffate/saves")
	assert.Equal(t, "/home/user/.llamaoffate/saves/game_save.yaml", saver.Path())
}

// --- Error cases ---

func TestYAMLSaver_Load_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	saver := NewYAMLSaver(dir)

	// Write corrupt data
	path := filepath.Join(dir, saveFileName)
	require.NoError(t, os.WriteFile(path, []byte("{{{{invalid yaml not valid"), 0o644))

	state, err := saver.Load()
	assert.Error(t, err, "loading corrupt YAML should error")
	assert.Nil(t, state)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestYAMLSaver_Save_ReadOnlyDir(t *testing.T) {
	// Create a directory and make it read-only
	dir := filepath.Join(t.TempDir(), "readonly")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.Chmod(dir, 0o444))
	defer func() { _ = os.Chmod(dir, 0o755) }() // Cleanup

	saver := NewYAMLSaver(dir)

	err := saver.Save(engine.GameState{})
	assert.Error(t, err, "saving to read-only directory should error")
}

// --- Round-trip: stress tracks, consequences, situation aspects, NPCs ---

// TestYAMLSaver_RoundTrip_StressTracksAndConsequences verifies that stress
// tracks and consequences survive serialisation — these were previously lost
// because StressTrack / Consequence lacked yaml struct tags.
func TestYAMLSaver_RoundTrip_StressTracksAndConsequences(t *testing.T) {
	dir := t.TempDir()
	saver := NewYAMLSaver(dir)

	player := character.NewCharacter("player1", "Zero")
	player.Aspects.HighConcept = "Ghost in the Machine Netrunner"
	player.Aspects.Trouble = "Wanted by Three Megacorps"
	player.Aspects.AddAspect("Military-Grade Cybernetic Reflexes")
	player.SetSkill("Burglary", dice.Superb)
	player.SetSkill("Stealth", dice.Great)
	player.FatePoints = 2
	player.Refresh = 3

	// Mark a stress box as used
	player.StressTracks[string(character.PhysicalStress)].Boxes[0] = true

	// Give the player a consequence
	player.Consequences = append(player.Consequences, character.Consequence{
		ID:     "c1",
		Type:   character.MildConsequence,
		Aspect: "Bruised Ribs",
	})

	npc := character.NewSupportingNPC("npc_nova", "Nova", "Slick Info Broker")

	testScene := scene.NewScene("scene_2", "Rainy Street", "Rain-soaked neon street.")
	testScene.AddSituationAspect(scene.SituationAspect{
		ID: "sa1", Aspect: "Rainy Night Visibility", Duration: "scene", FreeInvokes: 1,
	})
	testScene.AddSituationAspect(scene.SituationAspect{
		ID: "sa2", Aspect: "Hovercar at the Curb", Duration: "scene",
	})
	testScene.AddCharacter("player1")
	testScene.AddCharacter("npc_nova")

	original := engine.GameState{
		Scenario: engine.ScenarioState{
			Player: player,
			Scenario: &scene.Scenario{
				Title:   "The Prometheus Job",
				Problem: "Extract the data core",
				Genre:   "Cyberpunk",
				Setting: "Dark near-future city",
			},
			NPCRegistry:  map[string]*character.Character{"nova": npc},
			NPCAttitudes: map[string]string{"nova": "friendly"},
		},
		Scene: engine.SceneState{
			CurrentScene: testScene,
			ScenePurpose: "Uncover the buyer's identity",
		},
	}

	require.NoError(t, saver.Save(original))

	loaded, err := saver.Load()
	require.NoError(t, err)
	require.NotNil(t, loaded)

	lp := loaded.Scenario.Player

	// --- Player character fields ---
	assert.Equal(t, "Zero", lp.Name)
	assert.Equal(t, "Ghost in the Machine Netrunner", lp.Aspects.HighConcept, "high concept must round-trip")
	assert.Equal(t, "Wanted by Three Megacorps", lp.Aspects.Trouble, "trouble must round-trip")
	assert.Contains(t, lp.Aspects.OtherAspects, "Military-Grade Cybernetic Reflexes")
	assert.Equal(t, 2, lp.FatePoints)
	assert.Equal(t, 3, lp.Refresh)
	assert.Equal(t, dice.Superb, lp.GetSkill("Burglary"))
	assert.Equal(t, dice.Great, lp.GetSkill("Stealth"))

	// --- Stress tracks must survive ---
	require.NotNil(t, lp.StressTracks, "stress tracks must not be nil after load")
	require.Contains(t, lp.StressTracks, string(character.PhysicalStress), "physical stress track missing")
	require.Contains(t, lp.StressTracks, string(character.MentalStress), "mental stress track missing")

	physical := lp.StressTracks[string(character.PhysicalStress)]
	assert.Equal(t, character.PhysicalStress, physical.Type)
	assert.Equal(t, 2, physical.MaxBoxes, "physical max boxes")
	require.Len(t, physical.Boxes, 2, "physical boxes length")
	assert.True(t, physical.Boxes[0], "first physical box should still be checked")
	assert.False(t, physical.Boxes[1], "second physical box should be unchecked")

	mental := lp.StressTracks[string(character.MentalStress)]
	assert.Equal(t, character.MentalStress, mental.Type)
	assert.Equal(t, 2, mental.MaxBoxes, "mental max boxes")
	require.Len(t, mental.Boxes, 2, "mental boxes length")

	// --- Consequences must survive ---
	require.Len(t, lp.Consequences, 1, "consequences must round-trip")
	assert.Equal(t, character.MildConsequence, lp.Consequences[0].Type)
	assert.Equal(t, "Bruised Ribs", lp.Consequences[0].Aspect)

	// --- NPC registry ---
	require.Contains(t, loaded.Scenario.NPCRegistry, "nova")
	loadedNPC := loaded.Scenario.NPCRegistry["nova"]
	assert.Equal(t, "Nova", loadedNPC.Name)
	assert.Equal(t, "Slick Info Broker", loadedNPC.Aspects.HighConcept)
	require.NotNil(t, loadedNPC.StressTracks, "NPC stress tracks must not be nil")
	require.Contains(t, loadedNPC.StressTracks, string(character.PhysicalStress))

	// --- Scene ---
	require.NotNil(t, loaded.Scene.CurrentScene)
	assert.Equal(t, "Rainy Street", loaded.Scene.CurrentScene.Name)
	require.Len(t, loaded.Scene.CurrentScene.SituationAspects, 2, "situation aspects must round-trip")
	assert.Equal(t, "Rainy Night Visibility", loaded.Scene.CurrentScene.SituationAspects[0].Aspect)
	assert.Equal(t, 1, loaded.Scene.CurrentScene.SituationAspects[0].FreeInvokes)
	assert.Equal(t, "Hovercar at the Curb", loaded.Scene.CurrentScene.SituationAspects[1].Aspect)

	// Characters in scene
	assert.Contains(t, loaded.Scene.CurrentScene.Characters, "player1")
	assert.Contains(t, loaded.Scene.CurrentScene.Characters, "npc_nova")
}

// TestYAMLSaver_Load_OldFormatSave_Rejected verifies that a save file created
// by an older version of the code (before yaml struct tags were added) is
// rejected by Validate. Old saves used Go's default lowercase keys (e.g.
// "stresstracks", "highconcept") which no longer deserialize into the current
// struct tags ("stress_tracks", "high_concept"), producing empty fields.
func TestYAMLSaver_Load_OldFormatSave_Rejected(t *testing.T) {
	dir := t.TempDir()
	oldFormatYAML := `scenario:
    player:
        id: player1
        name: Jesse Calhoun
        charactertype: 0
        aspects:
            highconcept: Haunted Former Rancher
            trouble: Vengeance Burns Hotter Than Reason
            otheraspects: []
        skills:
            Shoot: 4
        fatepoints: 4
        refresh: 3
        stresstracks:
            physical:
                type: physical
                boxes:
                    - true
                    - false
                maxboxes: 2
            mental:
                type: mental
                boxes:
                    - false
                    - false
                maxboxes: 2
    scenario:
        title: Trouble in Redemption Gulch
        problem: The town is under threat
        genre: Western
        is_resolved: false
    scenario_count: 0
    scene_count: 1
scene:
    current_scene:
        id: scene_1
        name: The Saloon
        description: A dusty saloon
        characters:
            - player1
    conversation_history: []
`
	err := os.WriteFile(filepath.Join(dir, saveFileName), []byte(oldFormatYAML), 0o644)
	require.NoError(t, err)

	saver := NewYAMLSaver(dir)
	loaded, loadErr := saver.Load()

	// The old-format save should be rejected because its YAML keys don't match
	// the current struct tags, leaving high concept and stress tracks empty.
	assert.Error(t, loadErr, "old-format save should be rejected by validation")
	assert.Nil(t, loaded)
	assert.Contains(t, loadErr.Error(), "invalid save state")
	assert.Contains(t, loadErr.Error(), "high concept")
	assert.Contains(t, loadErr.Error(), "stress tracks")
}

// --- Save produces valid YAML ---

func TestYAMLSaver_Save_ProducesReadableYAML(t *testing.T) {
	dir := t.TempDir()
	saver := NewYAMLSaver(dir)

	state := engine.GameState{
		Scenario: engine.ScenarioState{
			Scenario: &scene.Scenario{
				Title: "Test Scenario",
				Genre: "Western",
			},
			ScenarioCount: 1,
		},
		Scene: engine.SceneState{
			ScenePurpose: "Find the clue",
		},
	}

	require.NoError(t, saver.Save(state))

	// Read raw file and verify it's human-readable YAML
	data, err := os.ReadFile(filepath.Join(dir, saveFileName))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "scenario:")
	assert.Contains(t, content, "scene:")
	assert.Contains(t, content, "Test Scenario")
	assert.Contains(t, content, "Find the clue")
}
