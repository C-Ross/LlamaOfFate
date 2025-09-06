package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUI implements the UI interface for testing
type MockUI struct {
	displayedMessages []string
	lastInput         string
	lastExit          bool
	lastError         error
}

func (m *MockUI) ReadInput() (input string, isExit bool, err error) {
	return m.lastInput, m.lastExit, m.lastError
}

func (m *MockUI) DisplayActionAttempt(description string) {
	m.displayedMessages = append(m.displayedMessages, "ActionAttempt: "+description)
}

func (m *MockUI) DisplayActionResult(skill string, skillLevel string, bonuses int, result string, outcome string) {
	m.displayedMessages = append(m.displayedMessages, "ActionResult: "+outcome)
}

func (m *MockUI) DisplayNarrative(narrative string) {
	m.displayedMessages = append(m.displayedMessages, "Narrative: "+narrative)
}

func (m *MockUI) DisplayDialog(playerInput, gmResponse string) {
	m.displayedMessages = append(m.displayedMessages, "Dialog: "+playerInput+" -> "+gmResponse)
}

func (m *MockUI) DisplaySystemMessage(message string) {
	m.displayedMessages = append(m.displayedMessages, "System: "+message)
}

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

	testScene := scene.NewScene("scene1", "Test Scene", "A test scene description")
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	assert.NotNil(t, sm.currentScene)
	assert.Equal(t, "scene1", sm.currentScene.ID)
	assert.Equal(t, "Test Scene", sm.currentScene.Name)
	assert.Equal(t, "A test scene description", sm.currentScene.Description)
	assert.Equal(t, player, sm.player)
	assert.Contains(t, sm.currentScene.Characters, player.ID)
	assert.Equal(t, player.ID, sm.currentScene.ActiveCharacter)
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

	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
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

	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	ctx := context.Background()

	// Should fail because no LLM client is configured
	err = sm.RunSceneLoop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM client is required")

	// Even with LLM client, should fail because no UI is configured
	mockClient := &MockLLMClient{}
	engine.llmClient = mockClient

	err = sm.RunSceneLoop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UI is required")
}

func TestSceneManager_ApplyActionEffects_CreateAdvantage(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player1", "Test Character")
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
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

	// Verify UI was called
	assert.True(t, len(mockUI.displayedMessages) > 0)
	assert.Contains(t, mockUI.displayedMessages[0], "Created situation aspect")
}

func TestSceneManager_GetCurrentScene(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	assert.Nil(t, sm.GetCurrentScene())

	player := character.NewCharacter("player1", "Test Character")
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
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
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
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

func TestSceneManager_OtherCharactersInTemplateData(t *testing.T) {
	// Create engine with mock LLM client
	mockClient := &MockLLMClient{response: "Test response"}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	// Create test characters
	player := character.NewCharacter("player1", "Player Character")
	player.Aspects.HighConcept = "Brave Hero"

	npc1 := character.NewCharacter("npc1", "Guard Captain")
	npc1.Aspects.HighConcept = "Experienced Soldier"

	npc2 := character.NewCharacter("npc2", "Merchant")
	npc2.Aspects.HighConcept = "Shrewd Trader"

	// Add characters to engine
	engine.AddCharacter(player)
	engine.AddCharacter(npc1)
	engine.AddCharacter(npc2)

	// Create scene and add all characters
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene with multiple characters")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc1.ID)
	testScene.AddCharacter(npc2.ID)

	// Start scene
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Test generateSceneResponse to verify OtherCharacters is populated
	response := sm.generateSceneResponse(context.Background(), "Hello there", "dialog")

	// The response should not be empty (indicates template executed successfully)
	assert.NotEmpty(t, response)
	assert.Equal(t, "Test response", response)

	// We can't easily test the exact template data without exposing internal methods,
	// but if the function completes without error, it means the template executed
	// successfully with the OtherCharacters field populated
}

func TestSceneManager_GenerateActionNarrativeWithTarget(t *testing.T) {
	// Create engine with mock LLM client
	mockClient := &MockLLMClient{response: "The attack strikes true!"}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	// Create test characters
	player := character.NewCharacter("player1", "Player Character")
	player.Aspects.HighConcept = "Brave Hero"

	enemy := character.NewCharacter("enemy1", "Orc Warrior")
	enemy.Aspects.HighConcept = "Brutal Fighter"

	// Add characters to engine
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	// Create scene and add characters
	testScene := scene.NewScene("scene1", "Test Scene", "A combat scene")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)

	// Start scene
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Create an action with a target
	testAction := action.NewAction("test-action-1", "player1", action.Attack, "Fight", "Attack the orc with sword")
	testAction.Target = "enemy1"
	testAction.Difficulty = dice.Fair

	// Set up action outcome
	testAction.Outcome = &dice.Outcome{
		Type:   dice.Success,
		Shifts: 2,
	}

	// Test generateActionNarrative with target
	narrative := sm.generateActionNarrative(context.Background(), testAction)

	// The response should not be empty (indicates template executed successfully)
	assert.NotEmpty(t, narrative)
	assert.Equal(t, "The attack strikes true!", narrative)

	// The function completing without error means the template executed successfully
	// with the Target field included in the ACTION DETAILS section
}

func TestSceneManager_GenerateActionNarrativeWithoutTarget(t *testing.T) {
	// Create engine with mock LLM client
	mockClient := &MockLLMClient{response: "You successfully overcome the obstacle!"}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	// Create test character
	player := character.NewCharacter("player1", "Player Character")
	player.Aspects.HighConcept = "Brave Hero"

	// Add character to engine
	engine.AddCharacter(player)

	// Create scene and add character
	testScene := scene.NewScene("scene1", "Test Scene", "A scene with obstacles")
	testScene.AddCharacter(player.ID)

	// Start scene
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Create an action without a target (like overcoming an environmental obstacle)
	testAction := action.NewAction("test-action-2", "player1", action.Overcome, "Athletics", "Jump across the chasm")
	// Note: No target set - should be empty string
	testAction.Difficulty = dice.Fair

	// Set up action outcome
	testAction.Outcome = &dice.Outcome{
		Type:   dice.Success,
		Shifts: 1,
	}

	// Test generateActionNarrative without target
	narrative := sm.generateActionNarrative(context.Background(), testAction)

	// The response should not be empty (indicates template executed successfully)
	assert.NotEmpty(t, narrative)
	assert.Equal(t, "You successfully overcome the obstacle!", narrative)

	// The function completing without error means the template executed successfully
	// even when the Target field is empty (conditional {{- if .Action.Target}} works)
}
