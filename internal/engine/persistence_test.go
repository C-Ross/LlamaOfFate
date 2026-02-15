package engine

import (
	"context"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- noopSaver tests ---

func TestNoopSaver_Save(t *testing.T) {
	saver := noopSaver{}
	err := saver.Save(GameState{})
	assert.NoError(t, err)
}

func TestNoopSaver_Load(t *testing.T) {
	saver := noopSaver{}
	state, err := saver.Load()
	assert.NoError(t, err)
	assert.Nil(t, state)
}

// --- recordingSaver for test assertions ---

type recordingSaver struct {
	savedStates []GameState
	loadResult  *GameState
	loadErr     error
	saveErr     error
}

func (s *recordingSaver) Save(state GameState) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.savedStates = append(s.savedStates, state)
	return nil
}

func (s *recordingSaver) Load() (*GameState, error) {
	return s.loadResult, s.loadErr
}

// --- SceneManager.Snapshot tests ---

func TestSceneManager_Snapshot_Empty(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	snapshot := sm.Snapshot()

	assert.Nil(t, snapshot.CurrentScene)
	assert.Empty(t, snapshot.ConversationHistory)
	assert.Empty(t, snapshot.ScenePurpose)
}

func TestSceneManager_Snapshot_WithState(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	testScene := scene.NewScene("scene1", "Saloon", "A dusty saloon")
	player := character.NewCharacter("player1", "Jesse")
	require.NoError(t, sm.StartScene(testScene, player))

	sm.SetScenePurpose("Find the informant")
	sm.conversationHistory = []prompt.ConversationEntry{
		{PlayerInput: "I look around", GMResponse: "You see a bartender", Timestamp: time.Now(), Type: "dialog"},
		{PlayerInput: "I approach", GMResponse: "He nods", Timestamp: time.Now(), Type: "dialog"},
	}

	snapshot := sm.Snapshot()

	assert.Equal(t, testScene, snapshot.CurrentScene)
	assert.Equal(t, "Find the informant", snapshot.ScenePurpose)
	assert.Len(t, snapshot.ConversationHistory, 2)
	assert.Equal(t, "I look around", snapshot.ConversationHistory[0].PlayerInput)
}

func TestSceneManager_Snapshot_CopiesConversationHistory(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	sm.conversationHistory = []prompt.ConversationEntry{
		{PlayerInput: "hello", GMResponse: "hi", Timestamp: time.Now(), Type: "dialog"},
	}

	snapshot := sm.Snapshot()

	// Mutate original — snapshot should be unaffected
	sm.conversationHistory = append(sm.conversationHistory,
		prompt.ConversationEntry{PlayerInput: "more", GMResponse: "stuff", Timestamp: time.Now(), Type: "dialog"},
	)
	assert.Len(t, snapshot.ConversationHistory, 1)
}

// --- ScenarioManager.Snapshot tests ---

func TestScenarioManager_Snapshot(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"
	player.SetSkill("shoot", dice.Great)

	sm := NewScenarioManager(engine, player)
	sm.SetScenario(&scene.Scenario{
		Title:          "The Train Heist",
		Problem:        "A train carrying gold",
		StoryQuestions: []string{"Will they get the gold?"},
		Setting:        "Wild West",
		Genre:          "Western",
	})
	sm.SetScenarioCount(2)
	sm.sceneCount = 3
	sm.lastGeneratedPurpose = "Infiltrate the train"
	sm.lastGeneratedHook = "You hear the whistle in the distance"
	sm.npcRegistry["marshal"] = character.NewCharacter("npc1", "Marshal Dan")
	sm.npcAttitudes["marshal"] = "hostile"
	sm.sceneSummaries = []prompt.SceneSummary{
		{SceneDescription: "The saloon", KeyEvents: []string{"Met the Marshal"}},
	}

	// Set up the scene manager with a scene
	sceneManager := engine.GetSceneManager()
	testScene := scene.NewScene("scene2", "Train Station", "A busy station")
	require.NoError(t, sceneManager.StartScene(testScene, player))
	sceneManager.SetScenePurpose("Board the train")

	scenarioState, sceneState := sm.Snapshot()

	// Verify scenario state
	assert.Equal(t, player, scenarioState.Player)
	assert.Equal(t, "The Train Heist", scenarioState.Scenario.Title)
	assert.Equal(t, 2, scenarioState.ScenarioCount)
	assert.Equal(t, 3, scenarioState.SceneCount)
	assert.Equal(t, "Infiltrate the train", scenarioState.LastPurpose)
	assert.Equal(t, "You hear the whistle in the distance", scenarioState.LastHook)
	assert.Len(t, scenarioState.NPCRegistry, 1)
	assert.Equal(t, "Marshal Dan", scenarioState.NPCRegistry["marshal"].Name)
	assert.Equal(t, "hostile", scenarioState.NPCAttitudes["marshal"])
	assert.Len(t, scenarioState.SceneSummaries, 1)
	assert.Equal(t, "The saloon", scenarioState.SceneSummaries[0].SceneDescription)

	// Verify scene state (cascaded from SceneManager)
	assert.Equal(t, testScene, sceneState.CurrentScene)
	assert.Equal(t, "Board the train", sceneState.ScenePurpose)
}

func TestScenarioManager_Snapshot_CopiesMaps(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	sm := NewScenarioManager(engine, player)
	sm.npcRegistry["bandit"] = character.NewCharacter("npc1", "Bandit")
	sm.npcAttitudes["bandit"] = "neutral"
	sm.sceneSummaries = []prompt.SceneSummary{
		{SceneDescription: "Scene one"},
	}

	scenarioState, _ := sm.Snapshot()

	// Mutate originals — snapshot should be unaffected
	sm.npcRegistry["new_npc"] = character.NewCharacter("npc2", "New NPC")
	sm.npcAttitudes["new_npc"] = "friendly"
	sm.sceneSummaries = append(sm.sceneSummaries, prompt.SceneSummary{SceneDescription: "Scene two"})

	assert.Len(t, scenarioState.NPCRegistry, 1)
	assert.Len(t, scenarioState.NPCAttitudes, 1)
	assert.Len(t, scenarioState.SceneSummaries, 1)
}

func TestScenarioManager_SetSaveFunc(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	sm := NewScenarioManager(engine, player)

	called := false
	sm.SetSaveFunc(func() error {
		called = true
		return nil
	})

	require.NotNil(t, sm.saveFunc)
	err = sm.saveFunc()
	assert.NoError(t, err)
	assert.True(t, called)
}

// --- GameManager.Save / SetSaver tests ---

func TestGameManager_SetSaver(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)

	// Default should be noopSaver
	assert.IsType(t, noopSaver{}, gm.saver)

	// Set a real saver
	recorder := &recordingSaver{}
	gm.SetSaver(recorder)
	assert.Equal(t, recorder, gm.saver)
}

func TestGameManager_SetSaver_NilFallsBackToNoop(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	gm.SetSaver(&recordingSaver{})
	gm.SetSaver(nil)

	assert.IsType(t, noopSaver{}, gm.saver)
}

func TestGameManager_Save_NoScenarioManager(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	recorder := &recordingSaver{}
	gm.SetSaver(recorder)

	// Should be a no-op when no scenario manager is running
	err = gm.Save()
	assert.NoError(t, err)
	assert.Empty(t, recorder.savedStates)
}

func TestGameManager_Save_CascadesSnapshot(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"

	gm := NewGameManager(engine)
	gm.SetPlayer(player)

	recorder := &recordingSaver{}
	gm.SetSaver(recorder)

	// Manually wire up scenario manager as Run() would
	gm.scenarioManager = NewScenarioManager(engine, player)
	gm.scenarioManager.SetScenario(&scene.Scenario{
		Title:   "Test Scenario",
		Problem: "A test problem",
		Genre:   "Western",
		Setting: "Frontier",
	})
	gm.scenarioManager.SetScenarioCount(1)

	// Set up a scene
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	sceneManager := engine.GetSceneManager()
	require.NoError(t, sceneManager.StartScene(testScene, player))
	sceneManager.SetScenePurpose("Test purpose")

	// Save
	err = gm.Save()
	require.NoError(t, err)
	require.Len(t, recorder.savedStates, 1)

	saved := recorder.savedStates[0]

	// Verify scenario state cascaded
	assert.Equal(t, "Jesse", saved.Scenario.Player.Name)
	assert.Equal(t, "Gunslinger", saved.Scenario.Player.Aspects.HighConcept)
	assert.Equal(t, "Test Scenario", saved.Scenario.Scenario.Title)
	assert.Equal(t, 1, saved.Scenario.ScenarioCount)

	// Verify scene state cascaded
	assert.Equal(t, "Test Scene", saved.Scene.CurrentScene.Name)
	assert.Equal(t, "Test purpose", saved.Scene.ScenePurpose)
}

// --- GameStateSaver interface compliance ---

func TestNoopSaver_ImplementsInterface(t *testing.T) {
	var _ GameStateSaver = noopSaver{}
}

func TestRecordingSaver_ImplementsInterface(t *testing.T) {
	var _ GameStateSaver = &recordingSaver{}
}

// --- SceneManager.Restore tests ---

func TestSceneManager_Restore_Empty(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	player := character.NewCharacter("player1", "Jesse")

	sm.Restore(SceneState{}, player)

	assert.Nil(t, sm.currentScene)
	assert.Empty(t, sm.conversationHistory)
	assert.Empty(t, sm.scenePurpose)
	assert.Equal(t, player, sm.player)
}

func TestSceneManager_Restore_WithState(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	player := character.NewCharacter("player1", "Jesse")

	testScene := scene.NewScene("scene1", "Saloon", "A dusty saloon")
	history := []prompt.ConversationEntry{
		{PlayerInput: "look around", GMResponse: "you see a bartender", Type: "dialog"},
		{PlayerInput: "approach bar", GMResponse: "he nods", Type: "dialog"},
	}

	sm.Restore(SceneState{
		CurrentScene:        testScene,
		ConversationHistory: history,
		ScenePurpose:        "Find the informant",
	}, player)

	assert.Equal(t, testScene, sm.currentScene)
	assert.Equal(t, player, sm.player)
	assert.Len(t, sm.conversationHistory, 2)
	assert.Equal(t, "look around", sm.conversationHistory[0].PlayerInput)
	assert.Equal(t, "Find the informant", sm.scenePurpose)
}

func TestSceneManager_Restore_AddsPlayerToScene(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	player := character.NewCharacter("player1", "Jesse")

	testScene := scene.NewScene("scene1", "Saloon", "A dusty saloon")
	// Scene starts with no characters
	assert.Empty(t, testScene.Characters)

	sm.Restore(SceneState{CurrentScene: testScene}, player)

	assert.Contains(t, testScene.Characters, "player1")
}

func TestSceneManager_Restore_RoundTrip(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	// Create and populate a scene manager
	sm := NewSceneManager(engine)
	player := character.NewCharacter("player1", "Jesse")
	testScene := scene.NewScene("scene1", "Saloon", "A dusty saloon")
	require.NoError(t, sm.StartScene(testScene, player))
	sm.SetScenePurpose("Find the informant")
	sm.conversationHistory = []prompt.ConversationEntry{
		{PlayerInput: "hello", GMResponse: "greetings", Type: "dialog"},
	}

	// Snapshot
	snapshot := sm.Snapshot()

	// Restore into a fresh scene manager
	sm2 := NewSceneManager(engine)
	sm2.Restore(snapshot, player)

	assert.Equal(t, testScene, sm2.currentScene)
	assert.Equal(t, "Find the informant", sm2.scenePurpose)
	assert.Len(t, sm2.conversationHistory, 1)
	assert.Equal(t, "hello", sm2.conversationHistory[0].PlayerInput)
}

// --- ScenarioManager.Restore tests ---

func TestScenarioManager_Restore(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"
	player.SetSkill("Shoot", dice.Great)
	player.FatePoints = 5

	sm := NewScenarioManager(engine, player)

	bartender := character.NewCharacter("npc_bartender", "Old Pete")
	bartender.Aspects.HighConcept = "Grizzled Barkeep"

	testScene := scene.NewScene("scene1", "Saloon", "The saloon")

	scenarioState := ScenarioState{
		Player: player,
		Scenario: &scene.Scenario{
			Title:   "The Showdown",
			Problem: "Outlaws threaten the town",
			Genre:   "Western",
		},
		ScenarioCount: 2,
		SceneCount:    4,
		SceneSummaries: []prompt.SceneSummary{
			{SceneDescription: "The dusty saloon"},
		},
		NPCRegistry:  map[string]*character.Character{"old pete": bartender},
		NPCAttitudes: map[string]string{"old pete": "friendly"},
		LastPurpose:  "Confront the marshal",
		LastHook:     "The doors swing open...",
	}

	sceneState := SceneState{
		CurrentScene:        testScene,
		ConversationHistory: []prompt.ConversationEntry{{PlayerInput: "hi", GMResponse: "hey", Type: "dialog"}},
		ScenePurpose:        "Confront the marshal",
	}

	sm.Restore(scenarioState, sceneState)

	// Verify scenario-level state
	assert.Equal(t, player, sm.player)
	assert.Equal(t, "The Showdown", sm.scenario.Title)
	assert.Equal(t, 2, sm.scenarioCount)
	assert.Equal(t, 4, sm.sceneCount)
	assert.Len(t, sm.sceneSummaries, 1)
	assert.Equal(t, "The dusty saloon", sm.sceneSummaries[0].SceneDescription)
	assert.Contains(t, sm.npcRegistry, "old pete")
	assert.Equal(t, "friendly", sm.npcAttitudes["old pete"])
	assert.Equal(t, "Confront the marshal", sm.lastGeneratedPurpose)
	assert.Equal(t, "The doors swing open...", sm.lastGeneratedHook)
	assert.True(t, sm.resumed)
}

func TestScenarioManager_Restore_RegistersNPCsWithEngine(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	sm := NewScenarioManager(engine, player)

	bartender := character.NewCharacter("npc_bartender", "Old Pete")
	marshal := character.NewCharacter("npc_marshal", "Marshal Dan")

	// Engine should not have these NPCs yet
	assert.Nil(t, engine.GetCharacter("npc_bartender"))
	assert.Nil(t, engine.GetCharacter("npc_marshal"))

	scenarioState := ScenarioState{
		Player: player,
		NPCRegistry: map[string]*character.Character{
			"old pete":    bartender,
			"marshal dan": marshal,
		},
	}

	sm.Restore(scenarioState, SceneState{})

	// NPCs should now be registered with the engine
	assert.NotNil(t, engine.GetCharacter("npc_bartender"))
	assert.NotNil(t, engine.GetCharacter("npc_marshal"))
	assert.Equal(t, "Old Pete", engine.GetCharacter("npc_bartender").Name)
}

func TestScenarioManager_Restore_NilMaps(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	sm := NewScenarioManager(engine, player)

	// Restore with nil maps — should not panic
	sm.Restore(ScenarioState{
		Player:       player,
		NPCRegistry:  nil,
		NPCAttitudes: nil,
	}, SceneState{})

	assert.NotNil(t, sm.npcRegistry)
	assert.NotNil(t, sm.npcAttitudes)
}

func TestScenarioManager_Restore_CascadesToSceneManager(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	sm := NewScenarioManager(engine, player)

	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	sceneState := SceneState{
		CurrentScene:        testScene,
		ConversationHistory: []prompt.ConversationEntry{{PlayerInput: "test", GMResponse: "ok", Type: "dialog"}},
		ScenePurpose:        "Test purpose",
	}

	sm.Restore(ScenarioState{Player: player}, sceneState)

	// Verify scene manager received the state
	sceneManager := engine.GetSceneManager()
	assert.Equal(t, testScene, sceneManager.GetCurrentScene())
	assert.Equal(t, "Test purpose", sceneManager.scenePurpose)
	assert.Len(t, sceneManager.conversationHistory, 1)
}

func TestScenarioManager_Restore_RoundTrip(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"

	sm := NewScenarioManager(engine, player)
	sm.SetScenario(&scene.Scenario{Title: "Train Heist", Genre: "Western"})
	sm.SetScenarioCount(1)
	sm.sceneCount = 2
	sm.npcRegistry["bandit"] = character.NewCharacter("npc1", "Bandit")
	sm.npcAttitudes["bandit"] = "hostile"
	sm.lastGeneratedPurpose = "Board the train"
	sm.lastGeneratedHook = "All aboard!"

	// Set up a scene
	testScene := scene.NewScene("scene1", "Station", "A busy station")
	sceneManager := engine.GetSceneManager()
	require.NoError(t, sceneManager.StartScene(testScene, player))
	sceneManager.SetScenePurpose("Board the train")

	// Snapshot
	scenarioState, sceneState := sm.Snapshot()

	// Create a new engine and scenario manager, then restore
	engine2, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	sm2 := NewScenarioManager(engine2, player)
	sm2.Restore(scenarioState, sceneState)

	// Verify round-trip
	assert.Equal(t, "Train Heist", sm2.scenario.Title)
	assert.Equal(t, 1, sm2.scenarioCount)
	assert.Equal(t, 2, sm2.sceneCount)
	assert.Contains(t, sm2.npcRegistry, "bandit")
	assert.Equal(t, "hostile", sm2.npcAttitudes["bandit"])
	assert.Equal(t, "Board the train", sm2.lastGeneratedPurpose)
	assert.Equal(t, "All aboard!", sm2.lastGeneratedHook)
	assert.True(t, sm2.resumed)

	// Scene should be restored on the new engine's scene manager
	assert.Equal(t, "Station", engine2.GetSceneManager().GetCurrentScene().Name)
}

// --- GameManager.Start load-on-startup tests ---

func TestGameManager_Start_LoadsOnStartup(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"

	testScene := scene.NewScene("scene1", "Saloon", "The saloon")

	// Pre-populate a SaveState that the saver will return
	savedState := &GameState{
		Scenario: ScenarioState{
			Player: player,
			Scenario: &scene.Scenario{
				Title:   "Saved Scenario",
				Problem: "Test",
				Genre:   "Western",
			},
			ScenarioCount: 1,
			SceneCount:    3,
		},
		Scene: SceneState{
			CurrentScene: testScene,
			ScenePurpose: "Find the clue",
		},
	}

	recorder := &recordingSaver{loadResult: savedState}

	gm := NewGameManager(engine)
	gm.SetPlayer(character.NewCharacter("different", "Different Player"))
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{Title: "Different Scenario"})

	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, events)

	// Should have used the saved player, not the one from SetPlayer
	assert.Equal(t, player, gm.player)

	// Manual save should capture the loaded state
	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates)
	lastSave := recorder.savedStates[len(recorder.savedStates)-1]
	assert.Equal(t, "Jesse", lastSave.Scenario.Player.Name)
	assert.Equal(t, "Saved Scenario", lastSave.Scenario.Scenario.Title)
}

func TestGameManager_Start_CompletedScenario_StartsFresh(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")

	// Save has a completed scenario
	savedState := &GameState{
		Scenario: ScenarioState{
			Player: player,
			Scenario: &scene.Scenario{
				Title:      "Completed Scenario",
				IsResolved: true,
			},
		},
		Scene: SceneState{
			CurrentScene: scene.NewScene("s1", "Scene", "Desc"),
		},
	}

	recorder := &recordingSaver{loadResult: savedState}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)

	// SetInitialScene to ensure a fresh start
	testScene := scene.NewScene("fresh_scene", "Fresh Start", "A new beginning")
	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
	})

	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, events)

	// Should NOT have used the completed save — fresh start should use the initial scene
	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates)
	lastSave := recorder.savedStates[len(recorder.savedStates)-1]
	assert.Equal(t, "Fresh Start", lastSave.Scene.CurrentScene.Name)
}

func TestGameManager_Start_NoSave_FreshStart(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	recorder := &recordingSaver{loadResult: nil} // No saved state

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)

	// SetInitialScene since we don't have an LLM for scene generation
	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
	})

	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, events)

	// Should have started fresh
	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates)
	lastSave := recorder.savedStates[len(recorder.savedStates)-1]
	assert.Equal(t, "Saloon", lastSave.Scene.CurrentScene.Name)
}
