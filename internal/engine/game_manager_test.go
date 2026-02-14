package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGameManager(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)

	assert.NotNil(t, gm)
	assert.Equal(t, engine, gm.engine)
}

func TestGameManager_SetPlayer(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	player := character.NewCharacter("player1", "Test Hero")

	gm.SetPlayer(player)

	assert.Equal(t, player, gm.player)
}

func TestGameManager_SetUI(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	mockUI := &MockUI{}

	gm.SetUI(mockUI)

	assert.Equal(t, mockUI, gm.ui)
}

func TestGameManager_SetScenario(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	scenario := &scene.Scenario{
		Title:   "Test Scenario",
		Problem: "A test problem",
		Genre:   "Fantasy",
		Setting: "A magical realm",
	}

	gm.SetScenario(scenario)

	assert.Equal(t, "Fantasy", gm.scenario.Genre)
	assert.Equal(t, "A magical realm", gm.scenario.Setting)
	assert.Equal(t, "A test problem", gm.scenario.Problem)
}

func TestGameManager_Run_RequiresEngine(t *testing.T) {
	gm := &GameManager{}
	err := gm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "engine is required")
}

func TestGameManager_Run_RequiresPlayer(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	gm.SetUI(&MockUI{})

	err = gm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player character is required")
}

func TestGameManager_Run_RequiresUI(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	gm.SetPlayer(character.NewCharacter("player1", "Test Hero"))

	err = gm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UI is required")
}

func TestGameManager_RunWithInitialScene_RequiresScene(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	gm.SetPlayer(character.NewCharacter("player1", "Test Hero"))
	gm.SetUI(&MockUI{})

	err = gm.RunWithInitialScene(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initial scene config is required")
}

func TestInitialSceneConfig_Fields(t *testing.T) {
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	npc1 := character.NewCharacter("npc1", "NPC One")
	npc2 := character.NewCharacter("npc2", "NPC Two")

	config := &InitialSceneConfig{
		Scene: testScene,
		NPCs:  []*character.Character{npc1, npc2},
	}

	assert.Equal(t, testScene, config.Scene)
	assert.Len(t, config.NPCs, 2)
}

func TestGameManager_HandleMilestone_ReturnsMilestoneEvent(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	player := character.NewCharacter("player1", "Test Hero")
	player.FatePoints = 1
	player.Refresh = 3
	gm.SetPlayer(player)
	gm.SetScenario(&scene.Scenario{Title: "The Heist"})
	gm.scenarioCount = 1

	events := gm.handleMilestone()

	require.NotEmpty(t, events)

	// Last event should be a MilestoneEvent
	milestone, ok := events[len(events)-1].(MilestoneEvent)
	require.True(t, ok, "last event should be MilestoneEvent")
	assert.Equal(t, "scenario_complete", milestone.Type)
	assert.Equal(t, "The Heist", milestone.ScenarioTitle)
	assert.Equal(t, 3, milestone.FatePoints) // Should be refreshed to 3

	// Fate points should be refreshed
	assert.Equal(t, 3, player.FatePoints)
}

func TestGameManager_HandleMilestone_WithConsequenceRecovery(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	player := character.NewCharacter("player1", "Test Hero")
	player.FatePoints = 2
	player.Refresh = 3
	// Add a recovering moderate consequence that should heal at scenario milestone
	player.Consequences = []character.Consequence{
		{
			ID:                    "c1",
			Type:                  character.ModerateConsequence,
			Aspect:                "Broken Arm",
			Recovering:            true,
			RecoveryStartScene:    0,
			RecoveryStartScenario: 0,
		},
	}
	gm.SetPlayer(player)
	gm.SetScenario(&scene.Scenario{Title: "The Heist"})
	gm.scenarioCount = 1

	events := gm.handleMilestone()

	require.Len(t, events, 2) // RecoveryEvent + MilestoneEvent

	// First event should be RecoveryEvent for the healed consequence
	recovery, ok := events[0].(RecoveryEvent)
	require.True(t, ok, "first event should be RecoveryEvent")
	assert.Equal(t, "healed", recovery.Action)
	assert.Equal(t, "Broken Arm", recovery.Aspect)
	assert.Equal(t, "moderate", recovery.Severity)

	// Second event should be MilestoneEvent
	milestone, ok := events[1].(MilestoneEvent)
	require.True(t, ok, "second event should be MilestoneEvent")
	assert.Equal(t, "scenario_complete", milestone.Type)
}

func TestGameManager_HandleMilestone_NilScenario(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	player := character.NewCharacter("player1", "Test Hero")
	player.Refresh = 3
	gm.SetPlayer(player)
	// No scenario set

	events := gm.handleMilestone()

	require.NotEmpty(t, events)
	milestone, ok := events[len(events)-1].(MilestoneEvent)
	require.True(t, ok)
	assert.Equal(t, "", milestone.ScenarioTitle)
}

func TestMilestoneEvent_Fields(t *testing.T) {
	event := MilestoneEvent{
		Type:          "scenario_complete",
		ScenarioTitle: "The Great Heist",
		FatePoints:    5,
	}
	assert.Equal(t, "scenario_complete", event.Type)
	assert.Equal(t, "The Great Heist", event.ScenarioTitle)
	assert.Equal(t, 5, event.FatePoints)
}

func TestGameResumedEvent_Fields(t *testing.T) {
	event := GameResumedEvent{
		ScenarioTitle: "The Great Heist",
		SceneName:     "The Vault",
	}
	assert.Equal(t, "The Great Heist", event.ScenarioTitle)
	assert.Equal(t, "The Vault", event.SceneName)
}

// --- GameManager.Start tests ---

func TestGameManager_Start_RequiresEngine(t *testing.T) {
	gm := &GameManager{saver: noopSaver{}}
	_, err := gm.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "engine is required")
}

func TestGameManager_Start_RequiresPlayer(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)

	_, err = gm.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player character is required")
}

func TestGameManager_Start_FreshStart(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	player.Aspects.HighConcept = "Brave Knight"

	gm := NewGameManager(engine)
	gm.SetPlayer(player)

	gm.SetScenario(&scene.Scenario{Title: "The Tournament", Genre: "Fantasy"})

	// Start creates the scenario manager, which will call getInitialScene
	// which generates via LLM. The mock returns a valid scene.
	events, err := gm.Start(context.Background())
	require.NoError(t, err)

	// Should have opening events (scene narrative)
	require.NotEmpty(t, events)
	narrative := events[0].(NarrativeEvent)
	assert.NotEmpty(t, narrative.SceneName)
}

func TestGameManager_HandleInput_BeforeStart(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)
	gm.SetPlayer(character.NewCharacter("player1", "Test Hero"))

	_, err = gm.HandleInput(context.Background(), "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HandleInput called before Start")
}

func TestGameManager_ProvideInvokeResponse_BeforeStart(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)

	_, err = gm.ProvideInvokeResponse(context.Background(), InvokeResponse{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ProvideInvokeResponse called before Start")
}

func TestGameManager_ProvideMidFlowResponse_BeforeStart(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	gm := NewGameManager(engine)

	_, err = gm.ProvideMidFlowResponse(context.Background(), MidFlowResponse{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ProvideMidFlowResponse called before Start")
}

func TestGameManager_Start_ResumeFromSave(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	player.FatePoints = 3

	testScene := scene.NewScene("scene1", "The Vault", "A bank vault")
	testScenario := &scene.Scenario{Title: "The Heist", Genre: "Crime"}

	savedState := &GameState{
		Scenario: ScenarioState{
			Player:   player,
			Scenario: testScenario,
		},
		Scene: SceneState{
			CurrentScene: testScene,
		},
	}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(&mockSaver{savedState: savedState})

	events, err := gm.Start(context.Background())
	require.NoError(t, err)

	// Should have GameResumedEvent prepended
	require.NotEmpty(t, events)
	resumed, ok := events[0].(GameResumedEvent)
	require.True(t, ok, "first event should be GameResumedEvent, got %T", events[0])
	assert.Equal(t, "The Heist", resumed.ScenarioTitle)
	assert.Equal(t, "The Vault", resumed.SceneName)

	// Rest should include scene narrative
	require.True(t, len(events) >= 2, "expected at least 2 events (resumed + narrative)")
}

func TestInputResult_GameOverFields(t *testing.T) {
	result := InputResult{
		GameOver: true,
		ScenarioResult: &ScenarioResult{
			Reason: ScenarioEndResolved,
		},
	}
	assert.True(t, result.GameOver)
	assert.Equal(t, ScenarioEndResolved, result.ScenarioResult.Reason)
}

// mockSaver implements GameStateSaver for testing
type mockSaver struct {
	savedState *GameState
	lastSaved  *GameState
}

func (m *mockSaver) Save(state GameState) error {
	m.lastSaved = &state
	return nil
}

func (m *mockSaver) Load() (*GameState, error) {
	return m.savedState, nil
}
