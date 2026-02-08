package engine

import (
	"context"
	"strings"
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
	displayedMessages     []string
	lastInput             string
	lastExit              bool
	lastError             error
	conflictStartCalls    []string
	conflictEscalateCalls []string
	turnAnnouncementCalls []string
	conflictEndCalls      []string
	invokeChoice          *InvokeChoice // Pre-configured invoke response
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

func (m *MockUI) PromptForInvoke(available []InvokableAspect, fatePoints int, currentResult string, shiftsNeeded int) *InvokeChoice {
	if m.invokeChoice != nil {
		return m.invokeChoice
	}
	// Default: skip invokes
	return &InvokeChoice{Aspect: nil}
}

func (m *MockUI) DisplayConflictStart(conflictType string, initiatorName string, participants []ConflictParticipantInfo) {
	m.conflictStartCalls = append(m.conflictStartCalls, conflictType+":"+initiatorName)
}

func (m *MockUI) DisplayConflictEscalation(fromType, toType, triggerCharName string) {
	m.conflictEscalateCalls = append(m.conflictEscalateCalls, fromType+"->"+toType+":"+triggerCharName)
}

func (m *MockUI) DisplayTurnAnnouncement(characterName string, turnNumber int, isPlayer bool) {
	m.turnAnnouncementCalls = append(m.turnAnnouncementCalls, characterName)
}

func (m *MockUI) DisplayConflictEnd(reason string) {
	m.conflictEndCalls = append(m.conflictEndCalls, reason)
}

func (m *MockUI) DisplayGameOver(reason string) {
	m.displayedMessages = append(m.displayedMessages, "GAME OVER: "+reason)
}

func (m *MockUI) DisplaySceneTransition(narrative string, newSceneHint string) {
	m.displayedMessages = append(m.displayedMessages, "SCENE TRANSITION: "+narrative)
}

func (m *MockUI) DisplayCharacter() {
	m.displayedMessages = append(m.displayedMessages, "CHARACTER")
}

func TestNewSceneManager(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	assert.NotNil(t, sm)
	assert.Equal(t, engine, sm.engine)
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
	_, err = sm.RunSceneLoop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM client is required")

	// Even with LLM client, should fail because no UI is configured
	mockClient := &MockLLMClient{}
	engine.llmClient = mockClient

	_, err = sm.RunSceneLoop(ctx)
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
	sm.applyActionEffects(context.Background(), testAction, nil) // nil target for create advantage

	assert.Equal(t, initialAspectCount+1, len(sm.currentScene.SituationAspects))

	newAspect := sm.currentScene.SituationAspects[len(sm.currentScene.SituationAspects)-1]
	assert.Contains(t, newAspect.Aspect, "Advantage from")
	assert.Equal(t, player.ID, newAspect.CreatedBy)
	assert.True(t, newAspect.FreeInvokes > 0)

	// Verify UI was called
	assert.True(t, len(mockUI.displayedMessages) > 0)
	assert.Contains(t, mockUI.displayedMessages[0], "Created situation aspect")
}

func TestSceneManager_ApplyActionEffects_CreateAdvantage_WithLLM(t *testing.T) {
	// Create a mock LLM client that returns a creative aspect name
	mockLLM := &MockLLMClient{
		response: `{
			"aspect_text": "Perfect Vantage Point",
			"description": "The character has found an excellent position",
			"duration": "scene",
			"free_invokes": 1,
			"is_boost": false,
			"reasoning": "The jump gave them a tactical advantage"
		}`,
	}

	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	sm := engine.GetSceneManager()
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player1", "Test Character")
	player.SetSkill("Athletics", dice.Good)
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Create a successful Create Advantage action
	testAction := createTestAction(t, "Create an Advantage", "Athletics", "Jump over the obstacle")

	// Simulate success
	roller := dice.NewSeededRoller(12345)
	result := roller.RollWithModifier(dice.Good, 2)
	testAction.CheckResult = result
	testAction.Outcome = result.CompareAgainst(dice.Fair)

	initialAspectCount := len(sm.currentScene.SituationAspects)
	sm.applyActionEffects(context.Background(), testAction, nil)

	assert.Equal(t, initialAspectCount+1, len(sm.currentScene.SituationAspects))

	newAspect := sm.currentScene.SituationAspects[len(sm.currentScene.SituationAspects)-1]
	// With LLM, it should have the creative aspect name instead of the fallback
	assert.Equal(t, "Perfect Vantage Point", newAspect.Aspect)
	assert.Equal(t, player.ID, newAspect.CreatedBy)
	assert.True(t, newAspect.FreeInvokes > 0)

	// Verify UI was called with the creative name
	assert.True(t, len(mockUI.displayedMessages) > 0)
	assert.Contains(t, mockUI.displayedMessages[0], "Perfect Vantage Point")
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
	response, err := sm.generateSceneResponse(context.Background(), "Hello there", "dialog")

	// The response should not be empty (indicates template executed successfully)
	require.NoError(t, err)
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
	narrative, err := sm.generateActionNarrative(context.Background(), testAction)

	// The response should not be empty (indicates template executed successfully)
	require.NoError(t, err)
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
	narrative, err := sm.generateActionNarrative(context.Background(), testAction)

	// The response should not be empty (indicates template executed successfully)
	require.NoError(t, err)
	assert.NotEmpty(t, narrative)
	assert.Equal(t, "You successfully overcome the obstacle!", narrative)

	// The function completing without error means the template executed successfully
	// even when the Target field is empty (conditional {{- if .Action.Target}} works)
}

func TestSceneManager_ParseConflictMarker_Physical(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	response := "The orc swings his axe at you! [CONFLICT:physical:orc-1] Roll for initiative!"
	trigger, cleanedResponse := sm.parseConflictMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, scene.PhysicalConflict, trigger.Type)
	assert.Equal(t, "orc-1", trigger.InitiatorID)
	assert.Equal(t, "The orc swings his axe at you! Roll for initiative!", cleanedResponse)
}

func TestSceneManager_ParseConflictMarker_Mental(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	response := "The sorcerer locks eyes with you, attempting to dominate your mind. [CONFLICT:mental:sorcerer-1]"
	trigger, cleanedResponse := sm.parseConflictMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, scene.MentalConflict, trigger.Type)
	assert.Equal(t, "sorcerer-1", trigger.InitiatorID)
	assert.Equal(t, "The sorcerer locks eyes with you, attempting to dominate your mind.", cleanedResponse)
}

func TestSceneManager_ParseConflictMarker_NoMarker(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	response := "The merchant smiles and offers you a deal."
	trigger, cleanedResponse := sm.parseConflictMarker(response)

	assert.Nil(t, trigger)
	assert.Equal(t, "The merchant smiles and offers you a deal.", cleanedResponse)
}

func TestSceneManager_CalculateInitiative_Physical(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	char := character.NewCharacter("char1", "Fighter")
	char.SetSkill("Notice", 3)
	char.SetSkill("Athletics", 2)

	// Should use Notice for physical conflicts
	initiative := sm.calculateInitiative(char, scene.PhysicalConflict)
	assert.Equal(t, 3, initiative)
}

func TestSceneManager_CalculateInitiative_Physical_NoNotice(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	char := character.NewCharacter("char1", "Fighter")
	char.SetSkill("Athletics", 2)

	// Should fall back to Athletics when Notice is 0
	initiative := sm.calculateInitiative(char, scene.PhysicalConflict)
	assert.Equal(t, 2, initiative)
}

func TestSceneManager_CalculateInitiative_Mental(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	char := character.NewCharacter("char1", "Wizard")
	char.SetSkill("Empathy", 4)
	char.SetSkill("Rapport", 2)

	// Should use Empathy for mental conflicts
	initiative := sm.calculateInitiative(char, scene.MentalConflict)
	assert.Equal(t, 4, initiative)
}

func TestSceneManager_CalculateInitiative_Mental_NoEmpathy(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	char := character.NewCharacter("char1", "Diplomat")
	char.SetSkill("Rapport", 3)

	// Should fall back to Rapport when Empathy is 0
	initiative := sm.calculateInitiative(char, scene.MentalConflict)
	assert.Equal(t, 3, initiative)
}

func TestSceneManager_InitiateConflict(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	mockUI := &MockUI{}
	sm := NewSceneManager(engine)
	sm.SetUI(mockUI)

	// Create player and enemy
	player := character.NewCharacter("player1", "Hero")
	player.SetSkill("Notice", 3)
	enemy := character.NewCharacter("enemy1", "Goblin")
	enemy.SetSkill("Notice", 1)

	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	// Create scene with both characters
	testScene := scene.NewScene("scene1", "Test Scene", "A dangerous encounter")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)

	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Initiate conflict
	err = sm.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	// Verify conflict state
	assert.True(t, sm.currentScene.IsConflict)
	require.NotNil(t, sm.currentScene.ConflictState)
	assert.Equal(t, scene.PhysicalConflict, sm.currentScene.ConflictState.Type)
	assert.Equal(t, enemy.ID, sm.currentScene.ConflictState.InitiatingCharacter)
	assert.Len(t, sm.currentScene.ConflictState.Participants, 2)

	// Verify UI was called
	require.Len(t, mockUI.conflictStartCalls, 1)
	assert.Contains(t, mockUI.conflictStartCalls[0], "physical")
}

func TestSceneManager_InitiateConflict_AlreadyInConflict(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	mockUI := &MockUI{}
	sm := NewSceneManager(engine)
	sm.SetUI(mockUI)

	// Create characters
	player := character.NewCharacter("player1", "Hero")
	enemy := character.NewCharacter("enemy1", "Goblin")
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	// Create scene
	testScene := scene.NewScene("scene1", "Test Scene", "A dangerous encounter")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Start first conflict
	err = sm.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	// Try to start another conflict
	err = sm.initiateConflict(scene.MentalConflict, player.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in a conflict")
}

func TestSceneManager_InitiateConflict_NotEnoughParticipants(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	mockUI := &MockUI{}
	sm := NewSceneManager(engine)
	sm.SetUI(mockUI)

	// Create only one character
	player := character.NewCharacter("player1", "Hero")
	engine.AddCharacter(player)

	// Create scene with only player
	testScene := scene.NewScene("scene1", "Test Scene", "Alone")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Try to initiate conflict with only one participant
	err = sm.initiateConflict(scene.PhysicalConflict, player.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 participants")
}

func TestSceneManager_HandleConflictEscalation(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	mockUI := &MockUI{}
	sm := NewSceneManager(engine)
	sm.SetUI(mockUI)

	// Create characters
	player := character.NewCharacter("player1", "Hero")
	player.SetSkill("Notice", 2)
	player.SetSkill("Empathy", 4)
	enemy := character.NewCharacter("enemy1", "Antagonist")
	enemy.SetSkill("Notice", 3)
	enemy.SetSkill("Empathy", 1)
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	// Create scene
	testScene := scene.NewScene("scene1", "Test Scene", "Tense negotiation")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Start a mental conflict
	err = sm.initiateConflict(scene.MentalConflict, enemy.ID)
	require.NoError(t, err)

	// Verify initial mental initiative (player with Empathy 4 should be first)
	assert.Equal(t, scene.MentalConflict, sm.currentScene.ConflictState.Type)
	// Player has Empathy 4, enemy has Empathy 1
	firstParticipant := sm.currentScene.ConflictState.Participants[0]
	assert.Equal(t, player.ID, firstParticipant.CharacterID)
	assert.Equal(t, 4, firstParticipant.Initiative)

	// Escalate to physical
	sm.handleConflictEscalation(scene.PhysicalConflict)

	// Verify escalation
	assert.Equal(t, scene.PhysicalConflict, sm.currentScene.ConflictState.Type)
	assert.Equal(t, scene.MentalConflict, sm.currentScene.ConflictState.OriginalType)

	// Initiative should be recalculated - enemy has Notice 3, player has Notice 2
	// So enemy should now be first
	firstParticipant = sm.currentScene.ConflictState.Participants[0]
	assert.Equal(t, enemy.ID, firstParticipant.CharacterID)
	assert.Equal(t, 3, firstParticipant.Initiative)
}

func TestSceneManager_ParseConflictEndMarker_Surrender(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	response := "The goblin drops his spear and raises his hands. \"I yield!\" [CONFLICT:end:surrender]"
	resolution, cleanedResponse := sm.parseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "surrender", resolution.Reason)
	assert.Equal(t, "The goblin drops his spear and raises his hands. \"I yield!\"", cleanedResponse)
}

func TestSceneManager_ParseConflictEndMarker_Agreement(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	response := "The merchant nods slowly. \"Very well, we have a deal.\" [CONFLICT:end:agreement]"
	resolution, cleanedResponse := sm.parseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "agreement", resolution.Reason)
	assert.Equal(t, "The merchant nods slowly. \"Very well, we have a deal.\"", cleanedResponse)
}

func TestSceneManager_ParseConflictEndMarker_Retreat(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	response := "The orc looks at his fallen comrades and flees into the forest. [CONFLICT:end:retreat]"
	resolution, cleanedResponse := sm.parseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "retreat", resolution.Reason)
	assert.Equal(t, "The orc looks at his fallen comrades and flees into the forest.", cleanedResponse)
}

func TestSceneManager_ParseConflictEndMarker_NoMarker(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	response := "The guard eyes you suspiciously but does not attack."
	resolution, cleanedResponse := sm.parseConflictEndMarker(response)

	assert.Nil(t, resolution)
	assert.Equal(t, "The guard eyes you suspiciously but does not attack.", cleanedResponse)
}

func TestSceneManager_ResolveConflictPeacefully(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	// Create test characters
	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Notice", 2)
	enemy := character.NewCharacter("enemy-1", "Goblin Guard")
	enemy.SetSkill("Notice", 1)

	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	// Setup scene
	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	sm.currentScene = testScene
	sm.player = player

	// Start a conflict
	err = sm.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)
	assert.True(t, sm.currentScene.IsConflict)

	// Resolve peacefully
	sm.resolveConflictPeacefully("surrender")

	// Verify conflict ended
	assert.False(t, sm.currentScene.IsConflict)
}

func TestSceneManager_ResolveConflictPeacefully_ClearsStress(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	// Create test characters
	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Notice", 2)
	enemy := character.NewCharacter("enemy-1", "Goblin Guard")
	enemy.SetSkill("Notice", 1)

	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	// Take stress on both characters
	player.TakeStress(character.PhysicalStress, 1)
	player.TakeStress(character.MentalStress, 1)
	enemy.TakeStress(character.PhysicalStress, 2)

	assert.Equal(t, 1, player.GetStressTrack(character.PhysicalStress).AvailableBoxes())
	assert.Equal(t, 1, player.GetStressTrack(character.MentalStress).AvailableBoxes())

	// Setup scene
	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	sm.currentScene = testScene
	sm.player = player

	// Start a conflict
	err = sm.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	// Resolve peacefully (should clear all stress)
	sm.resolveConflictPeacefully("surrender")

	// Verify stress was cleared for all participants
	assert.Equal(t, 2, player.GetStressTrack(character.PhysicalStress).AvailableBoxes())
	assert.Equal(t, 2, player.GetStressTrack(character.MentalStress).AvailableBoxes())
	assert.Equal(t, 2, enemy.GetStressTrack(character.PhysicalStress).AvailableBoxes())
}

func TestSceneManager_ResolveConflictPeacefully_NotInConflict(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	// Setup scene (no conflict)
	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	sm.currentScene = testScene

	// Attempting to resolve should not panic
	sm.resolveConflictPeacefully("surrender")

	// Still not in conflict
	assert.False(t, sm.currentScene.IsConflict)
}

func TestSceneManager_RollTargetDefense(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	sm.roller = dice.NewSeededRoller(12345) // Predictable rolls
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	target := character.NewCharacter("target-1", "Goblin")
	target.SetSkill("Athletics", 2)
	target.SetSkill("Will", 1)

	// Test physical attack defense (uses Athletics)
	defenseResult := sm.rollTargetDefense(target, "Fight")
	assert.NotNil(t, defenseResult)
	// With seeded roller and Athletics +2, we get a predictable result

	// Test mental attack defense (uses Will)
	defenseResult = sm.rollTargetDefense(target, "Provoke")
	assert.NotNil(t, defenseResult)
}

func TestSceneManager_ApplyDamageToTarget_StressAbsorbed(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	target := character.NewCharacter("target-1", "Goblin")
	// Default stress track should be able to absorb small hits

	sm.applyDamageToTarget(target, 1, character.PhysicalStress)

	// Check that stress was absorbed message was displayed
	found := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "absorbs the damage") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected stress absorption message")
}

func TestHandleTargetStressOverflow_ConsequenceSelection(t *testing.T) {
	tests := []struct {
		name           string
		available      []character.ConsequenceSlot
		shifts         int
		expectedType   character.ConsequenceType
		expectedReason string
	}{
		{
			name: "exact match picks that consequence",
			available: []character.ConsequenceSlot{
				{Type: character.MildConsequence, Value: 2},
				{Type: character.ModerateConsequence, Value: 4},
				{Type: character.SevereConsequence, Value: 6},
			},
			shifts:       4,
			expectedType: character.ModerateConsequence,
		},
		{
			name: "picks smallest consequence that covers shifts",
			available: []character.ConsequenceSlot{
				{Type: character.MildConsequence, Value: 2},
				{Type: character.ModerateConsequence, Value: 4},
				{Type: character.SevereConsequence, Value: 6},
			},
			shifts:       3,
			expectedType: character.ModerateConsequence,
		},
		{
			name: "large hit picks largest available when none covers",
			available: []character.ConsequenceSlot{
				{Type: character.MildConsequence, Value: 2},
				{Type: character.ModerateConsequence, Value: 4},
			},
			shifts:       5,
			expectedType: character.ModerateConsequence,
		},
		{
			name: "only mild available and shifts exceed it picks mild",
			available: []character.ConsequenceSlot{
				{Type: character.MildConsequence, Value: 2},
			},
			shifts:       5,
			expectedType: character.MildConsequence,
		},
		{
			name: "single consequence that covers",
			available: []character.ConsequenceSlot{
				{Type: character.SevereConsequence, Value: 6},
			},
			shifts:       3,
			expectedType: character.SevereConsequence,
		},
		{
			name: "prefers mild over severe when both cover",
			available: []character.ConsequenceSlot{
				{Type: character.SevereConsequence, Value: 6},
				{Type: character.MildConsequence, Value: 2},
			},
			shifts:       2,
			expectedType: character.MildConsequence,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bestConseq, ok := character.BestConsequenceFor(tc.available, tc.shifts)
			require.True(t, ok)

			assert.Equal(t, tc.expectedType, bestConseq.Type,
				"expected %s consequence for %d shifts", tc.expectedType, tc.shifts)
		})
	}
}

func TestSceneManager_HandleTargetTakenOut(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")
	otherEnemy := character.NewCharacter("enemy-2", "Orc") // Another enemy so conflict doesn't auto-end

	engine.AddCharacter(player)
	engine.AddCharacter(target)
	engine.AddCharacter(otherEnemy)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	testScene.AddCharacter(otherEnemy.ID)
	sm.currentScene = testScene
	sm.player = player

	// Start a conflict
	err = sm.initiateConflict(scene.PhysicalConflict, target.ID)
	require.NoError(t, err)

	// Take out the target
	sm.handleTargetTakenOut(target)

	// Check that target is marked as taken out (conflict still active because of otherEnemy)
	participant := sm.currentScene.GetParticipant(target.ID)
	require.NotNil(t, participant)
	assert.Equal(t, scene.StatusTakenOut, participant.Status)

	// Conflict should still be active since otherEnemy remains
	assert.True(t, sm.currentScene.IsConflict)
}

func TestSceneManager_HandleTargetTakenOut_ConflictEnds(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	sm.currentScene = testScene
	sm.player = player

	// Start a conflict
	err = sm.initiateConflict(scene.PhysicalConflict, target.ID)
	require.NoError(t, err)

	// Take out the only target
	sm.handleTargetTakenOut(target)

	// Conflict should end since no active opponents remain
	assert.False(t, sm.currentScene.IsConflict)

	// Check victory message was displayed
	found := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "Victory") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected victory message")
}

func TestSceneManager_HandleTargetTakenOut_MarksSceneLevelTakenOut(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	sm.currentScene = testScene
	sm.player = player

	// Start a conflict
	err = sm.initiateConflict(scene.PhysicalConflict, target.ID)
	require.NoError(t, err)

	// Take out the target
	sm.handleTargetTakenOut(target)

	// Conflict ends, but character should still be marked as taken out at scene level
	assert.False(t, sm.currentScene.IsConflict)
	assert.True(t, sm.currentScene.IsCharacterTakenOut(target.ID))
}

func TestSceneManager_InitiateConflict_ExcludesTakenOutCharacters(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	enemy1 := character.NewCharacter("enemy-1", "Goblin")
	enemy2 := character.NewCharacter("enemy-2", "Orc")

	engine.AddCharacter(player)
	engine.AddCharacter(enemy1)
	engine.AddCharacter(enemy2)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy1.ID)
	testScene.AddCharacter(enemy2.ID)
	sm.currentScene = testScene
	sm.player = player

	// Start first conflict and take out enemy1
	err = sm.initiateConflict(scene.PhysicalConflict, enemy1.ID)
	require.NoError(t, err)

	sm.handleTargetTakenOut(enemy1)
	// Conflict still ongoing because enemy2 is active
	assert.True(t, sm.currentScene.IsConflict)

	// End the conflict manually (simulating peaceful resolution)
	sm.currentScene.EndConflict()
	assert.False(t, sm.currentScene.IsConflict)

	// Try to initiate a new conflict with enemy2
	err = sm.initiateConflict(scene.PhysicalConflict, enemy2.ID)
	require.NoError(t, err)

	// enemy1 should NOT be in the new conflict since they were taken out
	participant := sm.currentScene.GetParticipant(enemy1.ID)
	assert.Nil(t, participant, "Taken-out character should not be in new conflict")

	// enemy2 and player should be in the conflict
	assert.NotNil(t, sm.currentScene.GetParticipant(enemy2.ID))
	assert.NotNil(t, sm.currentScene.GetParticipant(player.ID))
}

func TestSceneManager_InitiateConflict_TakenOutInitiatorFails(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	enemy := character.NewCharacter("enemy-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	sm.currentScene = testScene
	sm.player = player

	// Start first conflict and take out enemy
	err = sm.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	sm.handleTargetTakenOut(enemy)
	assert.False(t, sm.currentScene.IsConflict) // Conflict ended

	// Enemy is marked as taken out at scene level
	assert.True(t, sm.currentScene.IsCharacterTakenOut(enemy.ID))

	// Try to have the taken-out enemy initiate a new conflict - should fail
	err = sm.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "taken out")
}

func TestSceneManager_ApplyActionEffects_Attack(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Create a successful attack action
	testAction := action.NewActionWithTarget(
		"action-1",
		player.ID,
		action.Attack,
		"Fight",
		"Strike the goblin",
		target.ID,
	)

	// Simulate a successful attack with 3 shifts
	roller := dice.NewSeededRoller(12345)
	result := roller.RollWithModifier(dice.Good, 2)
	testAction.CheckResult = result
	testAction.Outcome = &dice.Outcome{
		Type:   dice.Success,
		Shifts: 3,
	}

	sm.applyActionEffects(context.Background(), testAction, target)

	// Check that damage message was displayed
	found := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "deals") && strings.Contains(msg, "shifts") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected damage message")
}

func TestSceneManager_IsConcedeCommand(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"concede lowercase", "concede", true},
		{"concede uppercase", "CONCEDE", true},
		{"concede mixed case", "Concede", true},
		{"i concede", "i concede", true},
		{"I concede", "I Concede", true},
		{"concession", "concession", true},
		{"i give up", "i give up", true},
		{"give up", "give up", true},
		{"with whitespace", "  concede  ", true},
		{"partial match attack", "I concede to attack", false},
		{"regular action", "I attack the goblin", false},
		{"empty string", "", false},
		{"random input", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sm.isConcedeCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSceneManager_HandleConcession(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	enemy := character.NewCharacter("enemy-1", "Goblin")
	player.FatePoints = 1 // Start with 1 fate point

	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Start a conflict
	err = sm.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)
	require.True(t, sm.currentScene.IsConflict)

	// Handle concession
	ctx := context.Background()
	sm.handleConcession(ctx)

	// Check fate point was awarded (1 base + 1 for conceding = 2)
	assert.Equal(t, 2, player.FatePoints, "Expected fate point for conceding")

	// Check conflict ended
	assert.False(t, sm.currentScene.IsConflict, "Expected conflict to end")

	// Check appropriate messages were displayed
	foundConcedeMsg := false
	foundFatePointMsg := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "Concede") {
			foundConcedeMsg = true
		}
		if strings.Contains(msg, "Fate Point") {
			foundFatePointMsg = true
		}
	}
	assert.True(t, foundConcedeMsg, "Expected concede message")
	assert.True(t, foundFatePointMsg, "Expected fate point message")
}

func TestSceneManager_HandleConcession_WithConsequences(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	enemy := character.NewCharacter("enemy-1", "Goblin")
	player.FatePoints = 1 // Start with 1 fate point

	// Add a consequence to the player
	player.AddConsequence(character.Consequence{
		ID:   "conseq-1",
		Type: character.MildConsequence,
	})
	player.AddConsequence(character.Consequence{
		ID:   "conseq-2",
		Type: character.ModerateConsequence,
	})

	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Start a conflict
	err = sm.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	// Handle concession
	ctx := context.Background()
	sm.handleConcession(ctx)

	// Check fate points: 1 base + 1 for conceding + 2 for consequences = 4
	assert.Equal(t, 4, player.FatePoints, "Expected fate points for conceding with consequences")

	// Check message mentions consequences
	foundConsequenceBonus := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "consequences") {
			foundConsequenceBonus = true
			break
		}
	}
	assert.True(t, foundConsequenceBonus, "Expected message about bonus fate points for consequences")
}

func TestEngine_GetCharacterByName_ExactMatch(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	engine.AddCharacter(npc)

	result := engine.GetCharacterByName("Bart the Outlaw")
	require.NotNil(t, result)
	assert.Equal(t, "scene-abc_npc_0", result.ID)
	assert.Equal(t, "Bart the Outlaw", result.Name)
}

func TestEngine_GetCharacterByName_CaseInsensitive(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	engine.AddCharacter(npc)

	result := engine.GetCharacterByName("bart the outlaw")
	require.NotNil(t, result)
	assert.Equal(t, "scene-abc_npc_0", result.ID)

	result = engine.GetCharacterByName("BART THE OUTLAW")
	require.NotNil(t, result)
	assert.Equal(t, "scene-abc_npc_0", result.ID)
}

func TestEngine_GetCharacterByName_Trimmed(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	engine.AddCharacter(npc)

	result := engine.GetCharacterByName("  Bart the Outlaw  ")
	require.NotNil(t, result)
	assert.Equal(t, "scene-abc_npc_0", result.ID)
}

func TestEngine_GetCharacterByName_NoMatch(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	engine.AddCharacter(npc)

	result := engine.GetCharacterByName("Nobody")
	assert.Nil(t, result)
}

func TestEngine_GetCharacterByName_IDDoesNotMatch(t *testing.T) {
	// Reproduces issue #25: the LLM returns a name, not an ID
	engine, err := New()
	require.NoError(t, err)

	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	engine.AddCharacter(npc)

	// ID-based lookup fails with a name
	byID := engine.GetCharacter("Bart the Outlaw")
	assert.Nil(t, byID, "Name should not match ID-based lookup")

	// Name-based lookup succeeds
	byName := engine.GetCharacterByName("Bart the Outlaw")
	require.NotNil(t, byName, "Name-based lookup should find the character")
	assert.Equal(t, "scene-abc_npc_0", byName.ID)
}

func TestSceneManager_ApplyActionEffects_Attack_NilTarget_ShowsError(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Create an attack action with a target name that can't be resolved
	testAction := action.NewActionWithTarget(
		"action-1",
		player.ID,
		action.Attack,
		"Fight",
		"Strike the bandit",
		"Bart the Outlaw",
	)
	testAction.Outcome = &dice.Outcome{
		Type:   dice.Success,
		Shifts: 3,
	}

	// Call with nil target (simulating failed resolution)
	sm.applyActionEffects(context.Background(), testAction, nil)

	// Should display an error message to the player, not silently skip
	found := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "Could not find target") && strings.Contains(msg, "Bart the Outlaw") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected error message about missing target, got: %v", mockUI.displayedMessages)
}

func TestSceneManager_ApplyActionEffects_Attack_DealsDamage(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Verify target starts with full stress
	initialAvailable := target.GetStressTrack(character.PhysicalStress).AvailableBoxes()

	// Create a successful attack action
	testAction := action.NewActionWithTarget(
		"action-1",
		player.ID,
		action.Attack,
		"Fight",
		"Strike the goblin",
		target.ID,
	)
	testAction.Outcome = &dice.Outcome{
		Type:   dice.Success,
		Shifts: 1,
	}

	sm.applyActionEffects(context.Background(), testAction, target)

	// Verify stress was actually applied
	afterAvailable := target.GetStressTrack(character.PhysicalStress).AvailableBoxes()
	assert.Less(t, afterAvailable, initialAvailable, "Target should have taken stress")

	// Verify damage message was displayed
	foundDamageMsg := false
	foundAbsorbMsg := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "deals") && strings.Contains(msg, "shifts") {
			foundDamageMsg = true
		}
		if strings.Contains(msg, "absorbs the damage") {
			foundAbsorbMsg = true
		}
	}
	assert.True(t, foundDamageMsg, "Expected damage message")
	assert.True(t, foundAbsorbMsg, "Expected stress absorption message")
}

func TestSceneManager_ApplyActionEffects_Attack_Tie_GrantsBoost(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Create an attack action that ties
	testAction := action.NewActionWithTarget(
		"action-1",
		player.ID,
		action.Attack,
		"Fight",
		"Strike the goblin",
		target.ID,
	)
	testAction.Outcome = &dice.Outcome{
		Type:   dice.Tie,
		Shifts: 0,
	}

	sm.applyActionEffects(context.Background(), testAction, target)

	// Verify boost message was displayed
	found := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "boost") {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected boost message on tie, got: %v", mockUI.displayedMessages)
}

func TestSceneManager_ResolveAction_TargetByName(t *testing.T) {
	// Tests the core fix for issue #25: resolveAction should find targets by name
	mockClient := &MockLLMClient{response: "The attack strikes true!"}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := engine.GetSceneManager()
	sm.roller = dice.NewSeededRoller(42)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	// Create player and NPC with different ID and name
	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Fight", dice.Good)
	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	npc.SetSkill("Athletics", dice.Fair)

	engine.AddCharacter(player)
	engine.AddCharacter(npc)

	testScene := scene.NewScene("scene-abc", "Saloon", "A dusty saloon.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Simulate what the LLM action parser returns: target is the NPC's NAME, not ID
	testAction := action.NewActionWithTarget(
		"action-1",
		player.ID,
		action.Attack,
		"Fight",
		"Punch Bart",
		"Bart the Outlaw", // Name, not ID — this is what the LLM returns
	)
	testAction.Difficulty = dice.Fair

	ctx := context.Background()
	sm.resolveAction(ctx, testAction)

	// The attack should have resolved against Bart — check defense was rolled
	// and damage was applied (if target wasn't found, no damage messages would appear)
	foundDefenseResult := false
	foundDamageApplied := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "Bart the Outlaw") && strings.Contains(msg, "defends") {
			foundDefenseResult = true
		}
		if strings.Contains(msg, "shifts") && strings.Contains(msg, "Bart the Outlaw") {
			foundDamageApplied = true
		}
	}
	assert.True(t, foundDefenseResult,
		"Expected defense roll against 'Bart the Outlaw' via name lookup, got: %v",
		mockUI.displayedMessages)
	assert.True(t, foundDamageApplied,
		"Expected damage applied to 'Bart the Outlaw', got: %v",
		mockUI.displayedMessages)
}

func TestSceneManager_ResolveAction_UnknownTarget_AbortsWithoutConsumingTurn(t *testing.T) {
	// When a target can't be resolved, the action should abort early:
	// no dice roll, no narrative, no turn advancement.
	mockClient := &MockLLMClient{response: "Should not be called"}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := engine.GetSceneManager()
	sm.roller = dice.NewSeededRoller(42)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Fight", dice.Good)
	player.SetSkill("Notice", dice.Fair)
	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	npc.SetSkill("Notice", dice.Average)

	engine.AddCharacter(player)
	engine.AddCharacter(npc)

	testScene := scene.NewScene("scene-abc", "Saloon", "A dusty saloon.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Attack a target that doesn't exist in the scene
	testAction := action.NewActionWithTarget(
		"action-1",
		player.ID,
		action.Attack,
		"Fight",
		"Attack the ghost",
		"The Ghost", // No such character
	)
	testAction.Difficulty = dice.Fair

	ctx := context.Background()
	sm.resolveAction(ctx, testAction)

	// Should see the "try again" message
	foundTryAgain := false
	for _, msg := range mockUI.displayedMessages {
		if strings.Contains(msg, "Could not find target") && strings.Contains(msg, "try again") {
			foundTryAgain = true
		}
	}
	assert.True(t, foundTryAgain,
		"Expected 'try again' message for unknown target, got: %v", mockUI.displayedMessages)

	// Should NOT see any dice results, narratives, or damage messages
	for _, msg := range mockUI.displayedMessages {
		assert.False(t, strings.HasPrefix(msg, "ActionResult:"),
			"Should not roll dice when target is unknown, got: %v", mockUI.displayedMessages)
		assert.False(t, strings.HasPrefix(msg, "Narrative:"),
			"Should not generate narrative when target is unknown, got: %v", mockUI.displayedMessages)
	}
}

// MockSceneInfoAwareUI implements both UI and SceneInfoSetter for testing
type MockSceneInfoAwareUI struct {
	MockUI
	sceneInfo SceneInfo
}

func (m *MockSceneInfoAwareUI) SetSceneInfo(info SceneInfo) {
	m.sceneInfo = info
}

func TestSetUI_CallsSetSceneInfo_WhenUIImplementsSceneInfoSetter(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockSceneInfoAwareUI{}

	sm.SetUI(mockUI)

	assert.Equal(t, SceneInfo(sm), mockUI.sceneInfo, "SetUI should call SetSceneInfo on UIs that implement SceneInfoSetter")
}

func TestSetUI_DoesNotPanic_WhenUIDoesNotImplementSceneInfoSetter(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}

	// Should not panic
	sm.SetUI(mockUI)
	assert.NotNil(t, sm.ui)
}
