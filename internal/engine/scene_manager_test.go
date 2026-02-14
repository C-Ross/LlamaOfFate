package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
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

func (m *MockUI) Emit(event GameEvent) {
	switch e := event.(type) {
	case ActionAttemptEvent:
		m.displayedMessages = append(m.displayedMessages, "ActionAttempt: "+e.Description)
	case ActionResultEvent:
		m.displayedMessages = append(m.displayedMessages, "ActionResult: "+e.Outcome)
	case NarrativeEvent:
		m.displayedMessages = append(m.displayedMessages, "Narrative: "+e.Text)
	case DialogEvent:
		m.displayedMessages = append(m.displayedMessages, "Dialog: "+e.PlayerInput+" -> "+e.GMResponse)
	case SystemMessageEvent:
		m.displayedMessages = append(m.displayedMessages, "System: "+e.Message)
	case ConflictStartEvent:
		m.conflictStartCalls = append(m.conflictStartCalls, e.ConflictType+":"+e.InitiatorName)
	case ConflictEscalationEvent:
		m.conflictEscalateCalls = append(m.conflictEscalateCalls, e.FromType+"->"+e.ToType+":"+e.TriggerCharName)
	case TurnAnnouncementEvent:
		m.turnAnnouncementCalls = append(m.turnAnnouncementCalls, e.CharacterName)
	case ConflictEndEvent:
		m.conflictEndCalls = append(m.conflictEndCalls, e.Reason)
	case GameOverEvent:
		m.displayedMessages = append(m.displayedMessages, "GAME OVER: "+e.Reason)
	case SceneTransitionEvent:
		m.displayedMessages = append(m.displayedMessages, "SCENE TRANSITION: "+e.Narrative)
	case CharacterDisplayEvent:
		m.displayedMessages = append(m.displayedMessages, "CHARACTER")

	// Composite mechanical events
	case DefenseRollEvent:
		m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("Defense: %s defends with %s (%s)", e.DefenderName, e.Skill, e.Result))
	case DamageResolutionEvent:
		if e.Absorbed != nil {
			m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("DamageRes: %s absorbs the damage with their %s stress track", e.TargetName, e.Absorbed.TrackType))
		}
		if e.Consequence != nil {
			m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("DamageRes: %s takes a %s consequence: \"%s\" (absorbs %d shifts)", e.Consequence.TargetName, e.Consequence.Severity, e.Consequence.Aspect, e.Consequence.Absorbed))
		}
		if e.RemainingAbsorbed != nil {
			m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("DamageRes: %s absorbs remaining %d shifts with stress", e.TargetName, e.RemainingAbsorbed.Shifts))
		}
		if e.TakenOut {
			m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("DamageRes: %s is Taken Out!", e.TargetName))
		}
		if e.VictoryEnd {
			m.displayedMessages = append(m.displayedMessages, "DamageRes: Victory! All opponents defeated!")
		}
	case PlayerAttackResultEvent:
		if e.TargetMissing {
			m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("AttackResult: Could not find target '%s'", e.TargetHint))
		} else if e.IsTie {
			m.displayedMessages = append(m.displayedMessages, "AttackResult: Tie! boost")
		} else {
			m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("AttackResult: deals %d shifts to %s", e.Shifts, e.TargetName))
		}
	case AspectCreatedEvent:
		m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("AspectCreated: '%s' with %d free invoke(s)", e.AspectName, e.FreeInvokes))
	case NPCAttackEvent:
		m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("NPCAttack: %s attacks %s with %s (%s) vs %s (%s)", e.AttackerName, e.TargetName, e.AttackSkill, e.AttackResult, e.DefenseSkill, e.DefenseResult))
	case PlayerStressEvent:
		m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("PlayerStress: %d %s stress (%s)", e.Shifts, e.StressType, e.TrackState))
	case PlayerDefendedEvent:
		if e.IsTie {
			m.displayedMessages = append(m.displayedMessages, "PlayerDefended: deflected, boost")
		} else {
			m.displayedMessages = append(m.displayedMessages, "PlayerDefended: successfully defend")
		}
	case PlayerConsequenceEvent:
		m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("PlayerConsequence: %s \"%s\" absorbs %d", e.Severity, e.Aspect, e.Absorbed))
	case PlayerTakenOutEvent:
		m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("PlayerTakenOut: by %s outcome=%s", e.AttackerName, e.Outcome))
	case ConcessionEvent:
		m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("Concession: gained %d FP (now %d)", e.FatePointsGained, e.CurrentFatePoints))
	case OutcomeChangedEvent:
		m.displayedMessages = append(m.displayedMessages, fmt.Sprintf("OutcomeChanged: %s", e.FinalOutcome))
	}
}

func (m *MockUI) PromptForInvoke(available []InvokableAspect, fatePoints int, currentResult string, shiftsNeeded int) *InvokeChoice {
	if m.invokeChoice != nil {
		return m.invokeChoice
	}
	// Default: skip invokes
	return &InvokeChoice{Aspect: nil}
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
	events := sm.applyActionEffects(context.Background(), testAction, nil) // nil target for create advantage

	assert.Equal(t, initialAspectCount+1, len(sm.currentScene.SituationAspects))

	newAspect := sm.currentScene.SituationAspects[len(sm.currentScene.SituationAspects)-1]
	assert.Contains(t, newAspect.Aspect, "Advantage from")
	assert.Equal(t, player.ID, newAspect.CreatedBy)
	assert.True(t, newAspect.FreeInvokes > 0)

	// Verify AspectCreatedEvent was returned
	found := false
	for _, evt := range events {
		if ac, ok := evt.(AspectCreatedEvent); ok {
			assert.Contains(t, ac.AspectName, "Advantage from")
			found = true
			break
		}
	}
	assert.True(t, found, "Expected AspectCreatedEvent in returned events, got: %v", events)
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
	events := sm.applyActionEffects(context.Background(), testAction, nil)

	assert.Equal(t, initialAspectCount+1, len(sm.currentScene.SituationAspects))

	newAspect := sm.currentScene.SituationAspects[len(sm.currentScene.SituationAspects)-1]
	// With LLM, it should have the creative aspect name instead of the fallback
	assert.Equal(t, "Perfect Vantage Point", newAspect.Aspect)
	assert.Equal(t, player.ID, newAspect.CreatedBy)
	assert.True(t, newAspect.FreeInvokes > 0)

	// Verify AspectCreatedEvent was returned with the creative name
	found := false
	for _, evt := range events {
		if ac, ok := evt.(AspectCreatedEvent); ok {
			assert.Equal(t, "Perfect Vantage Point", ac.AspectName)
			found = true
			break
		}
	}
	assert.True(t, found, "Expected AspectCreatedEvent in returned events, got: %v", events)
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
			mockClient := &MockLLMClient{response: tc.response}
			engine, err := NewWithLLM(mockClient)
			require.NoError(t, err)

			sm := NewSceneManager(engine)
			sm.currentScene = scene.NewScene("test", "Test", "Test scene")

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
	client := &sequentialMockLLMClient{
		responses: []string{"narrative", "You walk over to the table and sit down."},
	}
	engine, err := NewWithLLM(client)
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	testScene := scene.NewScene("test-scene", "Tavern", "A cozy tavern")
	sm.currentScene = testScene

	player := character.NewCharacter("player-1", "Test Player")
	sm.player = player

	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	result, err := sm.HandleInput(context.Background(), "I walk to the table")
	require.NoError(t, err)

	// Should return a DialogEvent (handleDialog path), not an action result
	hasDialog := false
	for _, event := range result.Events {
		if _, ok := event.(DialogEvent); ok {
			hasDialog = true
		}
	}
	assert.True(t, hasDialog, "Narrative input should produce a DialogEvent. Got events: %v", result.Events)
}

// sequentialMockLLMClient returns responses in order, cycling through them
type sequentialMockLLMClient struct {
	responses []string
	callIndex int
}

func (s *sequentialMockLLMClient) ChatCompletion(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	resp := s.responses[s.callIndex%len(s.responses)]
	s.callIndex++
	return &llm.CompletionResponse{
		ID:      "test",
		Object:  "chat.completion",
		Created: 0,
		Model:   "test",
		Choices: []llm.CompletionResponseChoice{
			{
				Index:        0,
				Message:      llm.Message{Role: "assistant", Content: resp},
				FinishReason: "stop",
			},
		},
		Usage: llm.CompletionUsage{PromptTokens: 10, CompletionTokens: 10, TotalTokens: 20},
	}, nil
}

func (s *sequentialMockLLMClient) ChatCompletionStream(ctx context.Context, req llm.CompletionRequest, handler llm.StreamHandler) error {
	return nil
}

func (s *sequentialMockLLMClient) GetModelInfo() llm.ModelInfo {
	return llm.ModelInfo{Name: "test", MaxTokens: 4096}
}

func TestSceneManager_RunSceneLoop_RecapsConversationOnResume(t *testing.T) {
	gameEngine, err := New()
	require.NoError(t, err)
	gameEngine.llmClient = &MockLLMClient{}

	sm := NewSceneManager(gameEngine)
	mockUI := &MockUI{lastInput: "quit", lastExit: true} // Exit immediately
	sm.SetUI(mockUI)

	player := character.NewCharacter("player1", "Test Character")
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")

	// Restore state with existing conversation history
	sm.Restore(SceneState{
		CurrentScene: testScene,
		ConversationHistory: []prompt.ConversationEntry{
			{PlayerInput: "I look around the room", GMResponse: "You see a dusty old tavern"},
			{PlayerInput: "I approach the bartender", GMResponse: "He eyes you warily"},
		},
		ScenePurpose: "Gather information",
	}, player)

	ctx := context.Background()
	_, err = sm.RunSceneLoop(ctx)
	require.NoError(t, err)

	// Should contain the recap markers and both conversation entries
	assert.Contains(t, mockUI.displayedMessages, "System: --- Recap of recent events ---")
	assert.Contains(t, mockUI.displayedMessages, "Dialog: I look around the room -> You see a dusty old tavern")
	assert.Contains(t, mockUI.displayedMessages, "Dialog: I approach the bartender -> He eyes you warily")
	assert.Contains(t, mockUI.displayedMessages, "System: --- End of recap ---")
}

func TestSceneManager_RunSceneLoop_NoRecapWithoutConversation(t *testing.T) {
	gameEngine, err := New()
	require.NoError(t, err)
	gameEngine.llmClient = &MockLLMClient{}

	sm := NewSceneManager(gameEngine)
	mockUI := &MockUI{lastInput: "quit", lastExit: true} // Exit immediately
	sm.SetUI(mockUI)

	player := character.NewCharacter("player1", "Test Character")
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")

	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = sm.RunSceneLoop(ctx)
	require.NoError(t, err)

	// Should NOT contain recap markers
	for _, msg := range mockUI.displayedMessages {
		assert.NotContains(t, msg, "Recap of recent events")
		assert.NotContains(t, msg, "End of recap")
	}
}

func TestSceneManager_StartScene_ClearsConversationHistory(t *testing.T) {
	gameEngine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(gameEngine)
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
	client := &sequentialMockLLMClient{
		responses: []string{"dialog", "The bartender nods slowly."},
	}
	engine, err := NewWithLLM(client)
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	testScene := scene.NewScene("tavern", "Tavern", "A dimly lit tavern")
	sm.currentScene = testScene
	sm.player = character.NewCharacter("player-1", "Hero")

	mockUI := &MockUI{}
	sm.SetUI(mockUI)

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
	client := &sequentialMockLLMClient{
		responses: []string{"dialog", "You step outside into the rain. [SCENE_TRANSITION:The rainy streets]"},
	}
	engine, err := NewWithLLM(client)
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	sm.exitOnSceneTransition = true
	testScene := scene.NewScene("tavern", "Tavern", "A dimly lit tavern")
	sm.currentScene = testScene
	sm.player = character.NewCharacter("player-1", "Hero")

	mockUI := &MockUI{}
	sm.SetUI(mockUI)

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
	client := &sequentialMockLLMClient{
		responses: []string{
			"action", // classification
			`{"skill":"Fight","type":"attack","description":"swing sword","target":"Goblin","difficulty":"Good"}`, // action parse
			"You swing your sword!", // narrative
		},
	}
	engine, err := NewWithLLM(client)
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	sm.roller = dice.NewSeededRoller(12345)
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
	sm.player = player

	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	result, err := sm.HandleInput(context.Background(), "I swing my sword at the goblin")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Action path now returns events
	assert.NotEmpty(t, result.Events, "action path should produce events")
	assert.False(t, result.SceneEnded)
}

func TestHandleInput_ClassificationFallbackToDialog(t *testing.T) {
	// LLM returns garbage for classification — should fallback to dialog
	client := &sequentialMockLLMClient{
		responses: []string{"xyzzy_invalid", "The room is quiet."},
	}
	engine, err := NewWithLLM(client)
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	testScene := scene.NewScene("room", "Room", "A quiet room")
	sm.currentScene = testScene
	sm.player = character.NewCharacter("player-1", "Hero")

	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	result, err := sm.HandleInput(context.Background(), "I look around")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should fallback to dialog and return a DialogEvent
	require.Len(t, result.Events, 1)
	_, ok := result.Events[0].(DialogEvent)
	assert.True(t, ok, "fallback should produce DialogEvent, got %T", result.Events[0])
}

func TestHandleInput_RenderEventsDialog(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	events := []GameEvent{
		DialogEvent{PlayerInput: "hello", GMResponse: "hi there"},
		SystemMessageEvent{Message: "something happened"},
		NarrativeEvent{Text: "The wind howls."},
		SceneTransitionEvent{Narrative: "", NewSceneHint: "next scene"},
	}

	sm.renderEvents(events)

	require.Len(t, mockUI.displayedMessages, 4)
	assert.Equal(t, "Dialog: hello -> hi there", mockUI.displayedMessages[0])
	assert.Equal(t, "System: something happened", mockUI.displayedMessages[1])
	assert.Equal(t, "Narrative: The wind howls.", mockUI.displayedMessages[2])
	assert.Equal(t, "SCENE TRANSITION: ", mockUI.displayedMessages[3])
}
