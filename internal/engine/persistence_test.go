package engine

import (
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
