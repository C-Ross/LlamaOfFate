package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_SaveCascade_ThroughGameManagerRun verifies that after
// GameManager.Run() completes, calling Save() cascades the full snapshot
// through ScenarioManager → SceneManager and captures the correct state.
func TestIntegration_SaveCascade_ThroughGameManagerRun(t *testing.T) {
	// Set up engine with mock LLM (not used — initial scene is pre-configured and quit is immediate)
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	// Create a player with real state
	player := character.NewCharacter("player1", "Jesse Calhoun")
	player.Aspects.HighConcept = "Gunslinger With a Past"
	player.Aspects.Trouble = "Wanted Dead or Alive"
	player.Aspects.AddAspect("Quick Draw")
	player.SetSkill("Shoot", dice.Great)
	player.SetSkill("Notice", dice.Fair)
	player.FatePoints = 5

	// Create a pre-configured scene with NPCs
	testScene := scene.NewScene("saloon_1", "The Dusty Saloon", "A dimly lit saloon at the edge of town.")
	testScene.AddSituationAspect(scene.SituationAspect{ID: "aspect_1", Aspect: "Smoky Atmosphere", Duration: "scene"})
	testScene.AddSituationAspect(scene.SituationAspect{ID: "aspect_2", Aspect: "Swinging Doors", Duration: "scene"})

	bartender := character.NewCharacter("npc_bartender", "Old Pete")
	bartender.Aspects.HighConcept = "Grizzled Barkeep"
	bartender.CharacterType = character.CharacterTypeSupportingNPC

	// UI that quits immediately
	mockUI := &MockUI{
		lastInput: "exit",
		lastExit:  true,
	}

	// Recording saver to capture save calls
	recorder := &recordingSaver{}

	// Set up scenario
	scenario := &scene.Scenario{
		Title:          "The Showdown at High Noon",
		Problem:        "A gang of outlaws threatens the town",
		StoryQuestions: []string{"Will Jesse protect the town?", "Can the outlaws be stopped?"},
		Setting:        "Dusty frontier town",
		Genre:          "Western",
	}

	// Wire up GameManager
	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetUI(mockUI)
	gm.SetSaver(recorder)
	gm.SetScenario(scenario)

	// Run with initial scene — player quits immediately
	err = gm.RunWithInitialScene(context.Background(), &InitialSceneConfig{
		Scene: testScene,
		NPCs:  []*character.Character{bartender},
	})
	require.NoError(t, err)

	// ScenarioManager should be stored on GameManager after Run
	require.NotNil(t, gm.scenarioManager, "scenarioManager should be stored after Run")

	// Now explicitly call Save — automatic triggers already saved during Run
	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates, "expected at least one save call")

	saved := recorder.savedStates[len(recorder.savedStates)-1]

	// --- Verify scenario state ---
	assert.Equal(t, "Jesse Calhoun", saved.Scenario.Player.Name)
	assert.Equal(t, "Gunslinger With a Past", saved.Scenario.Player.Aspects.HighConcept)
	assert.Equal(t, "Wanted Dead or Alive", saved.Scenario.Player.Aspects.Trouble)
	assert.Equal(t, 5, saved.Scenario.Player.FatePoints)
	assert.Equal(t, dice.Great, saved.Scenario.Player.GetSkill("Shoot"))

	assert.Equal(t, "The Showdown at High Noon", saved.Scenario.Scenario.Title)
	assert.Equal(t, "A gang of outlaws threatens the town", saved.Scenario.Scenario.Problem)
	assert.Len(t, saved.Scenario.Scenario.StoryQuestions, 2)
	assert.Equal(t, "Western", saved.Scenario.Scenario.Genre)
	assert.False(t, saved.Scenario.Scenario.IsResolved)

	// --- Verify scene state ---
	require.NotNil(t, saved.Scene.CurrentScene, "current scene should be captured")
	assert.Equal(t, "The Dusty Saloon", saved.Scene.CurrentScene.Name)
	assert.Contains(t, saved.Scene.CurrentScene.Description, "dimly lit saloon")

	// Situation aspects should be present on the scene
	assert.GreaterOrEqual(t, len(saved.Scene.CurrentScene.SituationAspects), 2)

	// Player should be in the scene's character list
	assert.Contains(t, saved.Scene.CurrentScene.Characters, "player1")
}

// TestIntegration_SaveFunc_WiredThroughRun verifies that the saveFunc callback
// set by GameManager on ScenarioManager correctly cascades to the recorder saver.
func TestIntegration_SaveFunc_WiredThroughRun(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	player.Aspects.HighConcept = "Reluctant Champion"

	testScene := scene.NewScene("scene1", "Test Arena", "A testing ground")

	mockUI := &MockUI{lastInput: "exit", lastExit: true}
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetUI(mockUI)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title:   "Test Scenario",
		Problem: "A test problem",
		Genre:   "Fantasy",
		Setting: "Arena",
	})

	// Run the game — player quits immediately
	err = gm.RunWithInitialScene(context.Background(), &InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	require.NoError(t, err)

	// Verify saveFunc was wired: call it through the scenarioManager
	require.NotNil(t, gm.scenarioManager)
	require.NotNil(t, gm.scenarioManager.saveFunc, "saveFunc should be wired by GameManager.Run")

	// Calling saveFunc should cascade through GameManager.Save to the recorder
	err = gm.scenarioManager.saveFunc()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates)

	// Verify the most recent save captured state from both layers
	saved := recorder.savedStates[len(recorder.savedStates)-1]
	assert.Equal(t, "Test Hero", saved.Scenario.Player.Name)
	assert.Equal(t, "Test Arena", saved.Scene.CurrentScene.Name)
}

// TestIntegration_SaveCascade_NPCRegistry verifies that NPCs registered during
// scene setup are captured in the snapshot's NPC registry.
func TestIntegration_SaveCascade_NPCRegistry(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"

	testScene := scene.NewScene("scene1", "Saloon", "The saloon")

	marshal := character.NewCharacter("npc_marshal", "Marshal Dan")
	marshal.Aspects.HighConcept = "Stern Lawman"
	marshal.CharacterType = character.CharacterTypeMainNPC

	bartender := character.NewCharacter("npc_bartender", "Old Pete")
	bartender.Aspects.HighConcept = "Grizzled Barkeep"
	bartender.CharacterType = character.CharacterTypeSupportingNPC

	mockUI := &MockUI{lastInput: "exit", lastExit: true}
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetUI(mockUI)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	err = gm.RunWithInitialScene(context.Background(), &InitialSceneConfig{
		Scene: testScene,
		NPCs:  []*character.Character{marshal, bartender},
	})
	require.NoError(t, err)

	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates)

	saved := recorder.savedStates[len(recorder.savedStates)-1]
	assert.Contains(t, saved.Scenario.NPCRegistry, "marshal dan")
	assert.Contains(t, saved.Scenario.NPCRegistry, "old pete")
	assert.Equal(t, "Stern Lawman", saved.Scenario.NPCRegistry["marshal dan"].Aspects.HighConcept)
	assert.Equal(t, "Grizzled Barkeep", saved.Scenario.NPCRegistry["old pete"].Aspects.HighConcept)
}

// TestIntegration_SaveCascade_MultipleSaves verifies that multiple Save() calls
// produce independent snapshots that reflect state changes between saves.
func TestIntegration_SaveCascade_MultipleSaves(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"
	player.FatePoints = 3

	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	mockUI := &MockUI{lastInput: "exit", lastExit: true}
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetUI(mockUI)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	err = gm.RunWithInitialScene(context.Background(), &InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	require.NoError(t, err)

	// Automatic saves happen during Run (scene_start, player_quit).
	// Record the baseline count so we can verify our manual saves.
	baseline := len(recorder.savedStates)

	// First manual save
	err = gm.Save()
	require.NoError(t, err)
	require.Len(t, recorder.savedStates, baseline+1)
	assert.Equal(t, 3, recorder.savedStates[baseline].Scenario.Player.FatePoints)

	// Mutate player state
	player.SpendFatePoint()

	// Second manual save — should reflect the change
	err = gm.Save()
	require.NoError(t, err)
	require.Len(t, recorder.savedStates, baseline+2)
	assert.Equal(t, 2, recorder.savedStates[baseline+1].Scenario.Player.FatePoints)
}

// TestIntegration_SaveCascade_NoopSaverByDefault verifies that without calling
// SetSaver, GameManager uses noopSaver and Save() succeeds silently.
func TestIntegration_SaveCascade_NoopSaverByDefault(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	mockUI := &MockUI{lastInput: "exit", lastExit: true}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetUI(mockUI)
	// Deliberately NOT calling SetSaver

	err = gm.RunWithInitialScene(context.Background(), &InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	require.NoError(t, err)

	// Save should succeed silently with noop
	err = gm.Save()
	assert.NoError(t, err)
}

// TestIntegration_SaveCascade_ConversationHistory verifies that conversation
// history accumulated during the scene loop is captured in the snapshot.
func TestIntegration_SaveCascade_ConversationHistory(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	mockUI := &MockUI{lastInput: "exit", lastExit: true}
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetUI(mockUI)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	err = gm.RunWithInitialScene(context.Background(), &InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	require.NoError(t, err)

	// Inject conversation history into the scene manager (simulating exchanges
	// that would happen during a real scene loop before the player quits)
	sceneManager := engine.GetSceneManager()
	sceneManager.conversationHistory = []prompt.ConversationEntry{
		{PlayerInput: "I look around the saloon", GMResponse: "You see a bartender polishing glasses", Type: "dialog"},
		{PlayerInput: "I approach the bar", GMResponse: "The bartender eyes you warily", Type: "dialog"},
	}

	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates, "expected at least one save call")

	// Check the most recent save (automatic triggers from the scene loop
	// produce earlier saves; our manual Save() with injected history is last)
	saved := recorder.savedStates[len(recorder.savedStates)-1]
	require.Len(t, saved.Scene.ConversationHistory, 2)
	assert.Equal(t, "I look around the saloon", saved.Scene.ConversationHistory[0].PlayerInput)
	assert.Equal(t, "I approach the bar", saved.Scene.ConversationHistory[1].PlayerInput)
}

// TestIntegration_SaveCascade_SceneSummaries verifies that scene summaries
// accumulated across scene transitions are captured in the snapshot.
func TestIntegration_SaveCascade_SceneSummaries(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	mockUI := &MockUI{lastInput: "exit", lastExit: true}
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetUI(mockUI)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	err = gm.RunWithInitialScene(context.Background(), &InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	require.NoError(t, err)

	// Inject scene summaries (simulating multi-scene progression)
	gm.scenarioManager.sceneSummaries = []prompt.SceneSummary{
		{
			SceneDescription:  "The dusty saloon",
			KeyEvents:         []string{"Met the bartender", "Overheard a conversation"},
			NPCsEncountered:   []prompt.NPCSummary{{Name: "Old Pete", Attitude: "friendly"}},
			AspectsDiscovered: []string{"Hidden Door Behind the Bar"},
			UnresolvedThreads: []string{"Who is the mysterious stranger?"},
			HowEnded:          "transition",
			NarrativeProse:    "Jesse pushed through the swinging doors...",
		},
		{
			SceneDescription: "The back alley",
			KeyEvents:        []string{"Found a clue"},
			HowEnded:         "transition",
		},
	}

	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates, "expected at least one save call")

	// Check the most recent save (automatic triggers from the scene loop
	// produce earlier saves; our manual Save() with injected summaries is last)
	saved := recorder.savedStates[len(recorder.savedStates)-1]
	require.Len(t, saved.Scenario.SceneSummaries, 2)
	assert.Equal(t, "The dusty saloon", saved.Scenario.SceneSummaries[0].SceneDescription)
	assert.Equal(t, "The back alley", saved.Scenario.SceneSummaries[1].SceneDescription)
	assert.Contains(t, saved.Scenario.SceneSummaries[0].KeyEvents, "Met the bartender")
	assert.Len(t, saved.Scenario.SceneSummaries[0].NPCsEncountered, 1)
	assert.Equal(t, "Old Pete", saved.Scenario.SceneSummaries[0].NPCsEncountered[0].Name)
}

// TestIntegration_AutomaticSaveTriggers verifies that the scene loop
// automatically triggers saves at scene_start and player_quit.
func TestIntegration_AutomaticSaveTriggers(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	mockUI := &MockUI{lastInput: "exit", lastExit: true}
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetUI(mockUI)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	// Run — player quits immediately, no manual Save() call
	err = gm.RunWithInitialScene(context.Background(), &InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	require.NoError(t, err)

	// Automatic triggers: scene_start + player_quit = 2 saves
	require.Len(t, recorder.savedStates, 2, "expected scene_start and player_quit saves")

	// Both saves should capture valid state
	for i, saved := range recorder.savedStates {
		assert.Equal(t, "Jesse", saved.Scenario.Player.Name, "save %d: player name", i)
		assert.Equal(t, "Saloon", saved.Scene.CurrentScene.Name, "save %d: scene name", i)
	}
}
