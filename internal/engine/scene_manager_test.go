package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSceneManager(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	assert.NotNil(t, sm)
	assert.Equal(t, engine, sm.engine)
	assert.NotNil(t, sm.reader)
	assert.NotNil(t, sm.roller)
}

func TestSceneManager_StartScene(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	player := character.NewCharacter("player1", "Test Character")

	err = sm.StartScene("scene1", "Test Scene", "A test scene description", player)
	require.NoError(t, err)

	assert.NotNil(t, sm.currentScene)
	assert.Equal(t, "scene1", sm.currentScene.ID)
	assert.Equal(t, "Test Scene", sm.currentScene.Name)
	assert.Equal(t, "A test scene description", sm.currentScene.Description)
	assert.Equal(t, player, sm.player)
	assert.Contains(t, sm.currentScene.Characters, player.ID)
	assert.Equal(t, player.ID, sm.currentScene.ActiveCharacter)
}

func TestSceneManager_IsExitCommand(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	testCases := []struct {
		input    string
		expected bool
	}{
		{"exit", true},
		{"quit", true},
		{"end", true},
		{"leave", true},
		{"resolve", true},
		{"EXIT", true},
		{"QUIT", true},
		{"  exit  ", true},
		{"help", false},
		{"attack the goblin", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := sm.isExitCommand(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSceneManager_ConversationHistory(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	// Test initial state
	assert.Empty(t, sm.conversationHistory)

	// Add some conversation entries
	sm.addToConversationHistory("What do I see?", "You see a dark room.", "dialog")
	sm.addToConversationHistory("Look around", "The room has stone walls.", "clarification")

	assert.Len(t, sm.conversationHistory, 2)
	assert.Equal(t, "What do I see?", sm.conversationHistory[0].PlayerInput)
	assert.Equal(t, "You see a dark room.", sm.conversationHistory[0].GMResponse)
	assert.Equal(t, "dialog", sm.conversationHistory[0].Type)
}

func TestSceneManager_BuildContexts(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	player := character.NewCharacter("player1", "Test Character")
	player.Aspects.HighConcept = "Brave Warrior"
	player.Aspects.Trouble = "Quick to Anger"

	err = sm.StartScene("scene1", "Test Scene", "A test scene", player)
	require.NoError(t, err)

	// Test character context
	charContext := sm.buildCharacterContext()
	assert.Contains(t, charContext, "Test Character")
	assert.Contains(t, charContext, "Brave Warrior")
	assert.Contains(t, charContext, "Quick to Anger")

	// Test conversation context (empty initially)
	convContext := sm.buildConversationContext()
	assert.Contains(t, convContext, "No previous conversation")

	// Add conversation and test again
	sm.addToConversationHistory("Hello", "Hello there!", "dialog")
	convContext = sm.buildConversationContext()
	assert.Contains(t, convContext, "Player: Hello")
	assert.Contains(t, convContext, "GM: Hello there!")

	// Test aspects context
	aspectsContext := sm.buildAspectsContext()
	assert.Contains(t, aspectsContext, "Brave Warrior")
	assert.Contains(t, aspectsContext, "Quick to Anger")
}

func TestSceneManager_RunSceneLoop_RequiresLLM(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	player := character.NewCharacter("player1", "Test Character")

	err = sm.StartScene("scene1", "Test Scene", "A test scene", player)
	require.NoError(t, err)

	ctx := context.Background()

	// Should fail because no LLM client is configured
	err = sm.RunSceneLoop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM client is required")
}

func TestSceneManager_ApplyActionEffects_CreateAdvantage(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	player := character.NewCharacter("player1", "Test Character")
	err = sm.StartScene("scene1", "Test Scene", "A test scene", player)
	require.NoError(t, err)

	// Create a successful Create Advantage action
	testAction := createTestAction(t, "Create an Advantage", "Athletics", "Jump over the obstacle")

	// Simulate success
	roller := dice.NewSeededRoller(12345)
	result := roller.RollWithModifier(dice.Good, 2) // Should be successful
	testAction.CheckResult = result
	testAction.Outcome = result.CompareAgainst(dice.Fair)

	initialAspectCount := len(sm.currentScene.SituationAspects)
	sm.applyActionEffects(testAction)

	assert.Equal(t, initialAspectCount+1, len(sm.currentScene.SituationAspects))

	newAspect := sm.currentScene.SituationAspects[len(sm.currentScene.SituationAspects)-1]
	assert.Contains(t, newAspect.Aspect, "Advantage from")
	assert.Equal(t, player.ID, newAspect.CreatedBy)
	assert.True(t, newAspect.FreeInvokes > 0)
}

func TestSceneManager_GetCurrentScene(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	assert.Nil(t, sm.GetCurrentScene())

	player := character.NewCharacter("player1", "Test Character")
	err = sm.StartScene("scene1", "Test Scene", "A test scene", player)
	require.NoError(t, err)

	scene := sm.GetCurrentScene()
	assert.NotNil(t, scene)
	assert.Equal(t, "scene1", scene.ID)
}

func TestSceneManager_GetPlayer(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	assert.Nil(t, sm.GetPlayer())

	player := character.NewCharacter("player1", "Test Character")
	err = sm.StartScene("scene1", "Test Scene", "A test scene", player)
	require.NoError(t, err)

	retrievedPlayer := sm.GetPlayer()
	assert.NotNil(t, retrievedPlayer)
	assert.Equal(t, player.ID, retrievedPlayer.ID)
	assert.Equal(t, player.Name, retrievedPlayer.Name)
}

// Helper function to create test actions
func createTestAction(t *testing.T, actionType, skill, description string) *action.Action {
	// Create a basic action for testing
	testAction := action.NewAction("test-action-1", "player1", action.CreateAdvantage, skill, description)
	testAction.Difficulty = dice.Fair
	return testAction
}
