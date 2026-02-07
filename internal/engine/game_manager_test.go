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
	scenario := &Scenario{
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
