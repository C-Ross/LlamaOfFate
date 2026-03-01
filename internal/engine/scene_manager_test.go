package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockUI is an event recorder for engine tests.
// It embeds EventRecorder for Emit() / OfType() / RequireFirst() etc.
type MockUI struct {
	EventRecorder
}

func TestNewSceneManager(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	assert.NotNil(t, sm)
	assert.Equal(t, engine, sm.characters)
	assert.NotNil(t, sm.actions.roller)
}

func TestSceneManager_StartScene(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
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
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	// Test initial state
	assert.Empty(t, sm.conversationHistory)

	// Add some conversation entries
	sm.RecordConversationEntry("What do I see?", "You see a dark room.", "dialog")
	sm.RecordConversationEntry("Look around", "The room has stone walls.", "clarification")

	assert.Len(t, sm.conversationHistory, 2)
	assert.Equal(t, "What do I see?", sm.conversationHistory[0].PlayerInput)
	assert.Equal(t, "You see a dark room.", sm.conversationHistory[0].GMResponse)
	assert.Equal(t, "dialog", sm.conversationHistory[0].Type)
}

func TestSceneManager_BuildContexts(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
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
	sm.RecordConversationEntry("Hello", "Hello there!", "dialog")
	convContext = sm.buildConversationContext()
	assert.Contains(t, convContext, "Player: Hello")
	assert.Contains(t, convContext, "GM: Hello there!")

	// Test aspects context
	aspectsContext := sm.buildAspectsContext()
	assert.Contains(t, aspectsContext, "Brave Warrior")
	assert.Contains(t, aspectsContext, "Quick to Anger")
}

func TestBuildCharacterContext_OtherAspects(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})
	player := character.NewCharacter("p1", "Lyra")
	player.Aspects.HighConcept = "Wandering Wizard"
	player.Aspects.Trouble = "Haunted Past"
	player.Aspects.AddAspect("Well Connected")
	player.Aspects.AddAspect("Silver Tongue")
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	ctx := sm.buildCharacterContext()
	assert.Contains(t, ctx, "Well Connected")
	assert.Contains(t, ctx, "Silver Tongue")
	assert.Contains(t, ctx, "Other Aspects:")
}

func TestBuildAspectsContext_FreeInvokes(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})
	player := character.NewCharacter("p1", "Hero")
	player.Aspects.HighConcept = "Bold Knight"
	testScene := scene.NewScene("s1", "Hall", "A great hall")
	testScene.SituationAspects = append(testScene.SituationAspects, scene.SituationAspect{
		ID:          "sa-1",
		Aspect:      "On Fire",
		FreeInvokes: 2,
	})
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

	ctx := sm.buildAspectsContext()
	assert.Contains(t, ctx, "On Fire (2 free invokes)")
}

func TestBuildAspectsContext_Empty(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})
	sm.player = nil
	sm.conflict.player = nil
	sm.actions.player = nil
	testScene := scene.NewScene("s1", "Hall", "Empty hall")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

	ctx := sm.buildAspectsContext()
	assert.Equal(t, "No special aspects currently in play.", ctx)
}

func TestAddToConversationHistory_TrimsBeyond10(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})

	// Add 12 entries
	for i := 0; i < 12; i++ {
		sm.RecordConversationEntry("input", "response", "dialog")
	}

	assert.Len(t, sm.conversationHistory, 10, "should trim to last 10 entries")
}

func TestBuildSceneEndResult_PlayerTakenOut(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})
	sm.conflict.sceneEndReason = SceneEndPlayerTakenOut
	sm.conflict.playerTakenOutHint = "You collapse in a heap."

	result := sm.buildSceneEndResult()
	assert.Equal(t, SceneEndPlayerTakenOut, result.Reason)
	assert.Equal(t, "You collapse in a heap.", result.TransitionHint)
}

func TestBuildSceneEndResult_DefaultReason(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})
	// sceneEndReason is empty
	result := sm.buildSceneEndResult()
	assert.Equal(t, SceneEndQuit, result.Reason)
}

func TestSceneManager_ApplyActionEffects_CreateAdvantage(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

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
	events := sm.actions.applyActionEffects(context.Background(), testAction, nil) // nil target for create advantage

	assert.Equal(t, initialAspectCount+1, len(sm.currentScene.SituationAspects))

	newAspect := sm.currentScene.SituationAspects[len(sm.currentScene.SituationAspects)-1]
	assert.Contains(t, newAspect.Aspect, "Advantage from")
	assert.Equal(t, player.ID, newAspect.CreatedBy)
	assert.True(t, newAspect.FreeInvokes > 0)

	// Verify AspectCreatedEvent was returned
	ac := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.Contains(t, ac.AspectName, "Advantage from")
}

func TestSceneManager_ApplyActionEffects_CreateAdvantage_WithLLM(t *testing.T) {
	// Create a mock LLM client that returns a creative aspect name
	mockLLM := newTestLLMClient(`{
			"aspect_text": "Perfect Vantage Point",
			"description": "The character has found an excellent position",
			"duration": "scene",
			"free_invokes": 1,
			"is_boost": false,
			"reasoning": "The jump gave them a tactical advantage"
		}`)

	engine, err := NewWithLLM(mockLLM, session.NullLogger{})
	require.NoError(t, err)

	sm := engine.GetSceneManager()

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
	events := sm.actions.applyActionEffects(context.Background(), testAction, nil)

	assert.Equal(t, initialAspectCount+1, len(sm.currentScene.SituationAspects))

	newAspect := sm.currentScene.SituationAspects[len(sm.currentScene.SituationAspects)-1]
	// With LLM, it should have the creative aspect name instead of the fallback
	assert.Equal(t, "Perfect Vantage Point", newAspect.Aspect)
	assert.Equal(t, player.ID, newAspect.CreatedBy)
	assert.True(t, newAspect.FreeInvokes > 0)

	// Verify AspectCreatedEvent was returned with the creative name
	ac := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.Equal(t, "Perfect Vantage Point", ac.AspectName)
}

func TestActionResolver_ApplyActionEffects_Overcome_NoMechanicalEffect(t *testing.T) {
	// Overcome actions currently produce no mechanical side effects via
	// applyActionEffects — this test documents that expectation so a future
	// ChallengeManager can add its own branch without breaking existing logic.
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player1", "Test Character")
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Create a successful Overcome action
	testAction := action.NewAction("test-action-1", "player1", action.Overcome, "Athletics", "Leap over the chasm")
	testAction.Outcome = &dice.Outcome{Type: dice.Success, Shifts: 2}

	events := sm.actions.applyActionEffects(context.Background(), testAction, nil)

	assert.Empty(t, events, "Overcome actions should produce no mechanical events (yet)")
}

func TestActionResolver_ApplyActionEffects_NilOutcome_ReturnsNil(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player1", "Test Character")
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Action with nil outcome
	testAction := action.NewAction("test-action-1", "player1", action.Attack, "Fight", "Strike")
	testAction.Outcome = nil

	events := sm.actions.applyActionEffects(context.Background(), testAction, nil)

	assert.Nil(t, events)
}

func TestActionResolver_AspectGeneratorWiring(t *testing.T) {
	// When created with an LLM, the ActionResolver should have an AspectGenerator.
	mockLLM := newTestLLMClient("test")
	engine, err := NewWithLLM(mockLLM, session.NullLogger{})
	require.NoError(t, err)

	sm := engine.GetSceneManager()
	assert.NotNil(t, sm.actions.aspectGenerator, "ActionResolver should have aspectGenerator when LLM is present")
}

func TestActionResolver_AspectGeneratorWiring_NoLLM(t *testing.T) {
	// Without an LLM, AspectGenerator should be nil.
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})
	assert.Nil(t, sm.actions.aspectGenerator, "ActionResolver should have nil aspectGenerator without LLM")
}

func TestActionResolver_ResolveAction_ActiveOpposition_Overcome(t *testing.T) {
	// When an Overcome action has OpposingNPCID set, the resolver should roll
	// the NPC's skill and use it as the difficulty instead of the static value.
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	mockLLM := newTestLLMClient("The guard nearly spots you as you slip past.")
	sm := NewSceneManager(engine, mockLLM, engine.actionParser, session.NullLogger{})
	sm.actions.roller = dice.NewSeededRoller(42) // Predictable rolls

	player := character.NewCharacter("player1", "Sneaky Rogue")
	player.SetSkill("Stealth", dice.Good)
	engine.AddCharacter(player)

	guard := character.NewCharacter("guard-1", "Stern Guard")
	guard.SetSkill("Notice", dice.Fair) // +2
	engine.AddCharacter(guard)

	testScene := scene.NewScene("scene1", "Corridor", "A guarded corridor")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(guard.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Create an Overcome action with active NPC opposition
	overcomeAction := action.NewAction("test-action-1", "player1", action.Overcome, "Stealth", "Sneak past the guard")
	overcomeAction.OpposingNPCID = "guard-1"
	overcomeAction.OpposingSkill = "Notice"
	overcomeAction.Difficulty = dice.Mediocre // Will be overridden by NPC roll

	events, _ := sm.actions.resolveAction(context.Background(), overcomeAction)

	// Should have a DefenseRollEvent for the NPC opposition
	defEvents := SliceOfType[DefenseRollEvent](events)
	require.Len(t, defEvents, 1, "Should have exactly one DefenseRollEvent for NPC opposition")
	assert.Equal(t, "Stern Guard", defEvents[0].DefenderName)
	assert.Equal(t, "Notice", defEvents[0].Skill)

	// Should have an ActionResultEvent with the NPC's name as defender
	actionResults := SliceOfType[ActionResultEvent](events)
	require.Len(t, actionResults, 1)
	assert.Equal(t, "Stern Guard", actionResults[0].DefenderName)

	// The difficulty should have been replaced by the NPC's roll, not the original static value
	assert.NotEqual(t, int(dice.Mediocre), actionResults[0].Difficulty,
		"Difficulty should be overridden by NPC's active opposition roll")
}

func TestActionResolver_ResolveAction_ActiveOpposition_CreateAdvantage(t *testing.T) {
	// Create Advantage with active opposition should also roll the NPC's skill.
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	mockLLM := newTestLLMClient("You study the merchant's body language.")
	sm := NewSceneManager(engine, mockLLM, engine.actionParser, session.NullLogger{})
	sm.actions.roller = dice.NewSeededRoller(99)

	player := character.NewCharacter("player1", "Observant Detective")
	player.SetSkill("Empathy", dice.Good)
	engine.AddCharacter(player)

	merchant := character.NewCharacter("merchant-1", "Cagey Merchant")
	merchant.SetSkill("Deceive", dice.Good) // +3
	engine.AddCharacter(merchant)

	testScene := scene.NewScene("scene1", "Market", "A bustling marketplace")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(merchant.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	caAction := action.NewAction("test-action-2", "player1", action.CreateAdvantage, "Empathy", "Read the merchant's tells")
	caAction.OpposingNPCID = "merchant-1"
	caAction.OpposingSkill = "Deceive"

	events, _ := sm.actions.resolveAction(context.Background(), caAction)

	defEvents := SliceOfType[DefenseRollEvent](events)
	require.Len(t, defEvents, 1)
	assert.Equal(t, "Cagey Merchant", defEvents[0].DefenderName)
	assert.Equal(t, "Deceive", defEvents[0].Skill)
}

func TestActionResolver_ResolveAction_ActiveOpposition_NPCNotFound(t *testing.T) {
	// If the opposing NPC ID can't be resolved, fall back to passive difficulty.
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	mockLLM := newTestLLMClient("You attempt the action.")
	sm := NewSceneManager(engine, mockLLM, engine.actionParser, session.NullLogger{})
	sm.actions.roller = dice.NewSeededRoller(42)

	player := character.NewCharacter("player1", "Test Hero")
	player.SetSkill("Stealth", dice.Good)
	engine.AddCharacter(player)

	testScene := scene.NewScene("scene1", "Room", "A room")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	overcomeAction := action.NewAction("test-action-3", "player1", action.Overcome, "Stealth", "Sneak past")
	overcomeAction.OpposingNPCID = "nonexistent-npc"
	overcomeAction.OpposingSkill = "Notice"
	overcomeAction.Difficulty = dice.Good // Should remain as passive fallback

	events, _ := sm.actions.resolveAction(context.Background(), overcomeAction)

	// Should NOT have a DefenseRollEvent since NPC wasn't found
	defEvents := SliceOfType[DefenseRollEvent](events)
	assert.Empty(t, defEvents, "Should have no DefenseRollEvent when NPC not found")

	// ActionResultEvent should use the original passive difficulty
	actionResults := SliceOfType[ActionResultEvent](events)
	require.Len(t, actionResults, 1)
	assert.Equal(t, int(dice.Good), actionResults[0].Difficulty)
	assert.Empty(t, actionResults[0].DefenderName)
}

func TestActionResolver_ResolveAction_PassiveOpposition_NoNPCRoll(t *testing.T) {
	// Standard passive opposition should NOT trigger any NPC roll.
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	mockLLM := newTestLLMClient("You climb the wall.")
	sm := NewSceneManager(engine, mockLLM, engine.actionParser, session.NullLogger{})
	sm.actions.roller = dice.NewSeededRoller(42)

	player := character.NewCharacter("player1", "Test Hero")
	player.SetSkill("Athletics", dice.Good)
	engine.AddCharacter(player)

	// NPC present but not opposing
	guard := character.NewCharacter("guard-1", "Guard")
	guard.SetSkill("Notice", dice.Fair)
	engine.AddCharacter(guard)

	testScene := scene.NewScene("scene1", "Wall", "A wall to climb")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(guard.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	overcomeAction := action.NewAction("test-action-4", "player1", action.Overcome, "Athletics", "Climb the wall")
	// No OpposingNPCID — passive opposition
	overcomeAction.Difficulty = dice.Good

	events, _ := sm.actions.resolveAction(context.Background(), overcomeAction)

	defEvents := SliceOfType[DefenseRollEvent](events)
	assert.Empty(t, defEvents, "Passive opposition should not produce DefenseRollEvent")

	actionResults := SliceOfType[ActionResultEvent](events)
	require.Len(t, actionResults, 1)
	assert.Equal(t, int(dice.Good), actionResults[0].Difficulty)
	assert.Empty(t, actionResults[0].DefenderName)
}

func TestActionResolver_GenerateAspectName_Fallback(t *testing.T) {
	// Without an aspect generator, generateAspectName should return a fallback.
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player1", "Test Character")
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	testAction := action.NewAction("test-action-1", "player1", action.CreateAdvantage, "Athletics", "Jump to high ground")
	testAction.Outcome = &dice.Outcome{Type: dice.Success, Shifts: 2}

	name, freeInvokes := sm.actions.generateAspectName(context.Background(), testAction)

	assert.Contains(t, name, "Advantage from")
	assert.Equal(t, 1, freeInvokes)
}

func TestActionResolver_GenerateAspectName_SuccessWithStyle_TwoFreeInvokes(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player1", "Test Character")
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	testAction := action.NewAction("test-action-1", "player1", action.CreateAdvantage, "Athletics", "Jump to high ground")
	testAction.Outcome = &dice.Outcome{Type: dice.SuccessWithStyle, Shifts: 3}

	_, freeInvokes := sm.actions.generateAspectName(context.Background(), testAction)

	assert.Equal(t, 2, freeInvokes, "Success with Style should grant 2 free invokes")
}

func TestSceneManager_GetCurrentScene(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

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
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

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
	mockClient := newTestLLMClient("Test response")
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

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
	mockClient := newTestLLMClient("The attack strikes true!")
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

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
	narrative, err := sm.GenerateActionNarrative(context.Background(), testAction)

	// The response should not be empty (indicates template executed successfully)
	require.NoError(t, err)
	assert.NotEmpty(t, narrative)
	assert.Equal(t, "The attack strikes true!", narrative)

	// The function completing without error means the template executed successfully
	// with the Target field included in the ACTION DETAILS section
}

func TestSceneManager_GenerateActionNarrativeWithoutTarget(t *testing.T) {
	// Create engine with mock LLM client
	mockClient := newTestLLMClient("You successfully overcome the obstacle!")
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

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
	narrative, err := sm.GenerateActionNarrative(context.Background(), testAction)

	// The response should not be empty (indicates template executed successfully)
	require.NoError(t, err)
	assert.NotEmpty(t, narrative)
	assert.Equal(t, "You successfully overcome the obstacle!", narrative)

	// The function completing without error means the template executed successfully
	// even when the Target field is empty (conditional {{- if .Action.Target}} works)
}

func TestEngine_GetCharacterByName_ExactMatch(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	engine.AddCharacter(npc)

	result := engine.GetCharacterByName("Bart the Outlaw")
	require.NotNil(t, result)
	assert.Equal(t, "scene-abc_npc_0", result.ID)
	assert.Equal(t, "Bart the Outlaw", result.Name)
}

func TestEngine_GetCharacterByName_CaseInsensitive(t *testing.T) {
	engine, err := New(session.NullLogger{})
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
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	engine.AddCharacter(npc)

	result := engine.GetCharacterByName("  Bart the Outlaw  ")
	require.NotNil(t, result)
	assert.Equal(t, "scene-abc_npc_0", result.ID)
}

func TestEngine_GetCharacterByName_NoMatch(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	npc := character.NewCharacter("scene-abc_npc_0", "Bart the Outlaw")
	engine.AddCharacter(npc)

	result := engine.GetCharacterByName("Nobody")
	assert.Nil(t, result)
}

func TestEngine_GetCharacterByName_IDDoesNotMatch(t *testing.T) {
	// Reproduces issue #25: the LLM returns a name, not an ID
	engine, err := New(session.NullLogger{})
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

// --- classifyInput unit tests (complement scene_manager_error_test.go) ---

func TestClassifyInput_TrimsExtraText(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
	}{
		{"trailing explanation", "dialog - the player is speaking", "dialog"},
		{"newline after type", "action\nbecause there is opposition", "action"},
		{"tab after type", "narrative\tthis is mundane", "narrative"},
		{"whitespace padded", "  clarification  ", "clarification"},
		{"markdown heading", "## narrative", "narrative"},
		{"markdown bold", "**action**", "action"},
		{"markdown heading with explanation", "## dialog because they are speaking", "dialog"},
		{"backtick wrapped", "`clarification`", "clarification"},
		{"quotes wrapped", "\"narrative\"", "narrative"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := newTestLLMClient(tc.response)
			engine, err := NewWithLLM(mockClient, session.NullLogger{})
			require.NoError(t, err)

			sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
			sm.currentScene = scene.NewScene("test", "Test", "Test scene")
			sm.conflict.currentScene = scene.NewScene("test", "Test", "Test scene")
			sm.actions.currentScene = scene.NewScene("test", "Test", "Test scene")

			result, err := sm.classifyInput(context.Background(), "test input")
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestProcessInput_NarrativeRoutesToDialog verifies that narrative classification
// goes through handleDialog (same as dialog/clarification, no dice roll).
func TestProcessInput_NarrativeRoutesToDialog(t *testing.T) {
	// First call returns "narrative" (classification), second returns scene response
	client := newTestLLMClient("narrative", "You walk over to the table and sit down.")
	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	testScene := scene.NewScene("test-scene", "Tavern", "A cozy tavern")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

	player := character.NewCharacter("player-1", "Test Player")
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I walk to the table")
	require.NoError(t, err)

	// Should return a DialogEvent (handleDialog path), not an action result
	AssertHasEventIn[DialogEvent](t, result.Events)
}

func TestSceneManager_StartScene_ClearsConversationHistory(t *testing.T) {
	gameEngine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(gameEngine, gameEngine.llmClient, gameEngine.actionParser, session.NullLogger{})
	player := character.NewCharacter("player1", "Test Character")

	// Restore with conversation history (simulating a previous scene)
	firstScene := scene.NewScene("scene1", "First Scene", "First scene desc")
	sm.Restore(SceneState{
		CurrentScene: firstScene,
		ConversationHistory: []prompt.ConversationEntry{
			{PlayerInput: "hello", GMResponse: "hi there"},
		},
	}, player)
	assert.Len(t, sm.GetConversationHistory(), 1)

	// StartScene should clear the old conversation
	secondScene := scene.NewScene("scene2", "Second Scene", "Second scene desc")
	err = sm.StartScene(secondScene, player)
	require.NoError(t, err)

	assert.Empty(t, sm.GetConversationHistory())
}

// --- HandleInput tests ---

func TestHandleInput_DialogReturnsDialogEvent(t *testing.T) {
	// First call: classification → "dialog", second call: scene response
	client := newTestLLMClient("dialog", "The bartender nods slowly.")
	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	testScene := scene.NewScene("tavern", "Tavern", "A dimly lit tavern")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = character.NewCharacter("player-1", "Hero")
	sm.conflict.player = character.NewCharacter("player-1", "Hero")
	sm.actions.player = character.NewCharacter("player-1", "Hero")

	result, err := sm.HandleInput(context.Background(), "I greet the bartender")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have exactly one DialogEvent
	require.Len(t, result.Events, 1)
	de, ok := result.Events[0].(DialogEvent)
	require.True(t, ok, "expected DialogEvent, got %T", result.Events[0])
	assert.Equal(t, "I greet the bartender", de.PlayerInput)
	assert.Equal(t, "The bartender nods slowly.", de.GMResponse)

	// Scene should not have ended
	assert.False(t, result.SceneEnded)
	assert.Nil(t, result.EndResult)
}

func TestHandleInput_DialogWithSceneTransition(t *testing.T) {
	// Response includes a scene transition marker
	client := newTestLLMClient("dialog", "You step outside into the rain. [SCENE_TRANSITION:The rainy streets]")
	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	sm.conflict.exitOnSceneTransition = true
	testScene := scene.NewScene("tavern", "Tavern", "A dimly lit tavern")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = character.NewCharacter("player-1", "Hero")
	sm.conflict.player = character.NewCharacter("player-1", "Hero")
	sm.actions.player = character.NewCharacter("player-1", "Hero")

	result, err := sm.HandleInput(context.Background(), "I leave the tavern")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have a DialogEvent and a SceneTransitionEvent
	require.Len(t, result.Events, 2)

	de, ok := result.Events[0].(DialogEvent)
	require.True(t, ok, "expected DialogEvent, got %T", result.Events[0])
	assert.Equal(t, "You step outside into the rain.", de.GMResponse)

	ste, ok := result.Events[1].(SceneTransitionEvent)
	require.True(t, ok, "expected SceneTransitionEvent, got %T", result.Events[1])
	assert.Equal(t, "The rainy streets", ste.NewSceneHint)

	// Scene should have ended with transition
	assert.True(t, result.SceneEnded)
	require.NotNil(t, result.EndResult)
	assert.Equal(t, SceneEndTransition, result.EndResult.Reason)
	assert.Equal(t, "The rainy streets", result.EndResult.TransitionHint)
}

func TestHandleInput_ActionPath_ReturnsEvents(t *testing.T) {
	// Classification returns "action"; the action path now returns events.
	client := newTestLLMClient(
		"action", // classification
		`{"skill":"Fight","type":"attack","description":"swing sword","target":"Goblin","difficulty":"Good"}`, // action parse
		"You swing your sword!", // narrative
	)
	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	sm.actions.roller = dice.NewSeededRoller(12345)
	testScene := scene.NewScene("arena", "Arena", "A fighting arena")

	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Fight", 2)
	enemy := character.NewCharacter("goblin-1", "Goblin")
	enemy.SetSkill("Fight", 1)

	engine.AddCharacter(player)
	engine.AddCharacter(enemy)
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I swing my sword at the goblin")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Action path now returns events
	assert.NotEmpty(t, result.Events, "action path should produce events")
	assert.False(t, result.SceneEnded)
}

func TestHandleInput_ClassificationFallbackToDialog(t *testing.T) {
	// LLM returns garbage for classification — should fallback to dialog
	client := newTestLLMClient("xyzzy_invalid", "The room is quiet.")
	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	testScene := scene.NewScene("room", "Room", "A quiet room")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = character.NewCharacter("player-1", "Hero")
	sm.conflict.player = character.NewCharacter("player-1", "Hero")
	sm.actions.player = character.NewCharacter("player-1", "Hero")

	result, err := sm.HandleInput(context.Background(), "I look around")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should fallback to dialog and return a DialogEvent
	require.Len(t, result.Events, 1)
	_, ok := result.Events[0].(DialogEvent)
	assert.True(t, ok, "fallback should produce DialogEvent, got %T", result.Events[0])
}

// --- Challenge integration tests ---

func TestHandleInput_DialogWithChallengeMarker_InitiatesChallenge(t *testing.T) {
	// Sequence:
	//  1. classification → "dialog"
	//  2. scene response → narrative with [CHALLENGE:...] marker
	//  3. challenge generator (BuildChallenge) → JSON tasks
	client := newTestLLMClient(
		"dialog",
		"The vault door looms before you. [CHALLENGE:Break into the vault]",
		`{"tasks": [
			{"skill": "Athletics", "difficulty": 3, "description": "Scale the wall"},
			{"skill": "Stealth",   "difficulty": 2, "description": "Sneak past guards"},
			{"skill": "Burglary",  "difficulty": 4, "description": "Pick the lock"}
		]}`,
	)

	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	testScene := scene.NewScene("vault", "Vault", "A massive vault")
	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Athletics", dice.Good)
	player.SetSkill("Stealth", dice.Fair)
	player.SetSkill("Burglary", dice.Great)
	engine.AddCharacter(player)
	testScene.AddCharacter(player.ID)

	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.challenge.setSceneState(testScene, player)
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I examine the vault")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have DialogEvent + ChallengeStartEvent
	AssertHasEventIn[DialogEvent](t, result.Events)
	startEvent := RequireFirstFrom[ChallengeStartEvent](t, result.Events)

	assert.Equal(t, "Break into the vault", startEvent.Description)
	assert.Len(t, startEvent.Tasks, 3)

	// Scene should now be in challenge mode
	assert.True(t, testScene.IsChallenge)
	require.NotNil(t, testScene.ChallengeState)
	assert.Len(t, testScene.ChallengeState.Tasks, 3)
}

func TestHandleInput_ActionDuringChallenge_ResolvesTask(t *testing.T) {
	// Scene already has an active challenge. Player takes an action whose
	// skill matches a pending task → should produce ChallengeTaskResultEvent.
	//
	// Sequence:
	//  1. classification → "action"
	//  2. action parse   → overcome with Athletics
	//  3. narrative       → flavor text
	client := newTestLLMClient(
		"action",
		`{"action_type":"Overcome","skill":"Athletics","description":"climb the wall","difficulty":3}`,
		"You scale the wall with ease!",
	)

	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	sm.actions.roller = dice.NewSeededRoller(42)

	testScene := scene.NewScene("vault", "Vault", "A massive vault")
	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Athletics", dice.Good) // +3
	engine.AddCharacter(player)
	testScene.AddCharacter(player.ID)

	// Manually start a challenge on the scene
	tasks := []scene.ChallengeTask{
		{ID: "task-1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending, Description: "Scale the wall"},
		{ID: "task-2", Skill: "Stealth", Difficulty: 2, Status: scene.TaskPending, Description: "Sneak past guards"},
	}
	err = testScene.StartChallenge("Break into the vault", tasks)
	require.NoError(t, err)

	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.challenge.setSceneState(testScene, player)
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I climb the wall")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have an ActionResultEvent, NarrativeEvent, and a ChallengeTaskResultEvent
	AssertHasEventIn[ActionResultEvent](t, result.Events)
	taskResult := RequireFirstFrom[ChallengeTaskResultEvent](t, result.Events)
	assert.Equal(t, "task-1", taskResult.TaskID)
	assert.Equal(t, "Athletics", taskResult.Skill)

	// task-2 is still pending so challenge should not be complete
	assert.True(t, testScene.IsChallenge)
	AssertNoEventIn[ChallengeCompleteEvent](t, result.Events)
}

func TestHandleInput_ActionDuringChallenge_CompletesChallenge(t *testing.T) {
	// Only one task left pending. Resolving it should produce both
	// ChallengeTaskResultEvent and ChallengeCompleteEvent.
	client := newTestLLMClient(
		"action",
		`{"action_type":"Overcome","skill":"Stealth","description":"sneak past the guards","difficulty":2}`,
		"You slip past unnoticed!",
	)

	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	sm.actions.roller = dice.NewSeededRoller(42)

	testScene := scene.NewScene("vault", "Vault", "A massive vault")
	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Stealth", dice.Fair) // +2
	engine.AddCharacter(player)
	testScene.AddCharacter(player.ID)

	// Challenge with one already-resolved task and one pending
	tasks := []scene.ChallengeTask{
		{ID: "task-1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskSucceeded, ActorID: "player-1"},
		{ID: "task-2", Skill: "Stealth", Difficulty: 2, Status: scene.TaskPending, Description: "Sneak past guards"},
	}
	err = testScene.StartChallenge("Break into the vault", tasks)
	require.NoError(t, err)

	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.challenge.setSceneState(testScene, player)
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I sneak past the guards")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have task result and challenge complete events
	AssertHasEventIn[ChallengeTaskResultEvent](t, result.Events)
	completeEvent := RequireFirstFrom[ChallengeCompleteEvent](t, result.Events)

	// task-1 succeeded, task-2 result depends on dice — but challenge should be done
	assert.GreaterOrEqual(t, completeEvent.Successes+completeEvent.Failures+completeEvent.Ties, 2)
	assert.NotEmpty(t, completeEvent.Overall)

	// Scene should no longer be in challenge
	assert.False(t, testScene.IsChallenge)
}
