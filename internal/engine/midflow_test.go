package engine

import (
	"context"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- handleStressOverflow tests ---

func TestStressOverflowEmitsChoiceEvent(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	attacker := character.NewCharacter("enemy-1", "Orc")
	engine.AddCharacter(player)
	engine.AddCharacter(attacker)

	testScene := scene.NewScene("test-scene", "Arena", "A dusty arena.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(attacker.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Call handleStressOverflow directly — player has default consequence slots.
	ctx := context.Background()
	attackCtx := prompt.AttackContext{Skill: "Fight", Description: "Slash", Shifts: 3}
	sm.conflict.handleStressOverflow(ctx, 3, character.PhysicalStress, attacker, attackCtx)

	// Should have set pendingMidFlow.
	require.NotNil(t, sm.actions.pendingMidFlow, "expected pendingMidFlow to be set")

	event := sm.actions.pendingMidFlow.event
	assert.Equal(t, uicontract.InputRequestNumberedChoice, event.Type)
	assert.Contains(t, event.Prompt, "choose how to handle")

	// Options should include consequence slots + "Be Taken Out".
	available := player.AvailableConsequenceSlots()
	assert.Len(t, event.Options, len(available)+1, "expected one option per consequence + taken out")

	// Last option should be "Be Taken Out".
	lastOpt := event.Options[len(event.Options)-1]
	assert.Contains(t, lastOpt.Label, "Be Taken Out")

	// Each consequence option should mention its type.
	for i, slot := range available {
		assert.Contains(t, event.Options[i].Label, string(slot.Type))
	}
}

func TestStressOverflowNoConsequences_TakenOut(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	mockLLM := &capturingMockLLMClient{
		response: `{"narrative": "You fall.", "outcome": "transition", "new_scene_hint": "You wake up later."}`,
	}
	engine.llmClient = mockLLM

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	attacker := character.NewCharacter("enemy-1", "Orc")
	// Fill all consequence slots so none are available.
	player.AddConsequence(character.Consequence{ID: "c1", Type: character.MildConsequence, Aspect: "Bruised"})
	player.AddConsequence(character.Consequence{ID: "c2", Type: character.ModerateConsequence, Aspect: "Broken Arm"})
	player.AddConsequence(character.Consequence{ID: "c3", Type: character.SevereConsequence, Aspect: "Shattered Leg"})

	engine.AddCharacter(player)
	engine.AddCharacter(attacker)

	testScene := scene.NewScene("test-scene", "Arena", "A dusty arena.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(attacker.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	ctx := context.Background()
	attackCtx := prompt.AttackContext{Skill: "Fight", Description: "Slash", Shifts: 3}
	events := sm.conflict.handleStressOverflow(ctx, 3, character.PhysicalStress, attacker, attackCtx)

	// No consequences available → should NOT set pendingMidFlow, should go straight to taken out.
	assert.Nil(t, sm.actions.pendingMidFlow, "expected no pending mid-flow when no consequences available")

	// Should have returned "taken out" events.
	AssertHasEventIn[PlayerTakenOutEvent](t, events)
}

func TestProvideMidFlowResponse_ConsequenceChoice(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	// LLM returns a consequence name.
	mockLLM := &capturingMockLLMClient{
		response: `[CONSEQUENCE_ASPECT:Bruised Ribs]`,
	}
	engine.llmClient = mockLLM

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Fight", 2)
	attacker := character.NewCharacter("enemy-1", "Orc")
	engine.AddCharacter(player)
	engine.AddCharacter(attacker)

	testScene := scene.NewScene("test-scene", "Arena", "A dusty arena.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(attacker.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	ctx := context.Background()
	attackCtx := prompt.AttackContext{Skill: "Fight", Description: "Slash", Shifts: 2}

	// Trigger stress overflow to get a pending mid-flow.
	sm.conflict.handleStressOverflow(ctx, 2, character.PhysicalStress, attacker, attackCtx)
	require.NotNil(t, sm.actions.pendingMidFlow)

	// Choose the first consequence (mild = absorbs 2).
	result, err := sm.ProvideMidFlowResponse(ctx, MidFlowResponse{ChoiceIndex: 0})
	require.NoError(t, err)
	require.NotNil(t, result)

	// pendingMidFlow should be cleared (unless recursive overflow created a new one).
	// The mild consequence should be applied.
	assert.Len(t, player.Consequences, 1, "expected one consequence applied")
	assert.Equal(t, character.MildConsequence, player.Consequences[0].Type)
}

func TestProvideMidFlowResponse_TakenOutChoice(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	mockLLM := &capturingMockLLMClient{
		response: `{"narrative": "You fall.", "outcome": "transition", "new_scene_hint": "You wake up later."}`,
	}
	engine.llmClient = mockLLM

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	attacker := character.NewCharacter("enemy-1", "Orc")
	engine.AddCharacter(player)
	engine.AddCharacter(attacker)

	testScene := scene.NewScene("test-scene", "Arena", "A dusty arena.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(attacker.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	ctx := context.Background()
	attackCtx := prompt.AttackContext{Skill: "Fight", Description: "Slash", Shifts: 3}
	sm.conflict.handleStressOverflow(ctx, 3, character.PhysicalStress, attacker, attackCtx)
	require.NotNil(t, sm.actions.pendingMidFlow)

	// Choose the last option (taken out). Available = 3 consequence slots, so index 3 = taken out.
	available := player.AvailableConsequenceSlots()
	takenOutIdx := len(available)
	result, err := sm.ProvideMidFlowResponse(ctx, MidFlowResponse{ChoiceIndex: takenOutIdx})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have returned PlayerTakenOutEvent in the result events
	AssertHasEventIn[PlayerTakenOutEvent](t, result.Events)
}

// --- promptPlayerForFates tests ---

func TestFateNarrationEmitsFreeTextEvent(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	npc1 := character.NewCharacter("npc-1", "Goblin Scout")
	npc1.Aspects.HighConcept = "Sneaky Goblin"
	npc2 := character.NewCharacter("npc-2", "Goblin Chief")
	npc2.Aspects.HighConcept = "Fearsome Leader"

	engine.AddCharacter(player)
	engine.AddCharacter(npc1)
	engine.AddCharacter(npc2)

	testScene := scene.NewScene("test-scene", "Forest Clearing", "A clearing in the forest.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc1.ID)
	testScene.AddCharacter(npc2.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Mark NPCs as taken out.
	sm.conflict.takenOutChars = []string{npc1.ID, npc2.ID}

	ctx := context.Background()
	sm.conflict.promptPlayerForFates(ctx)

	// Should have set pendingMidFlow with a free_text event.
	require.NotNil(t, sm.actions.pendingMidFlow, "expected pendingMidFlow for fate narration")
	event := sm.actions.pendingMidFlow.event

	assert.Equal(t, uicontract.InputRequestFreeText, event.Type)
	assert.Contains(t, event.Prompt, "Goblin Scout")
	assert.Contains(t, event.Prompt, "Goblin Chief")
	assert.Equal(t, "fate_narration", event.Context["request_type"])

	// Context should include NPC names.
	npcNames, ok := event.Context["npc_names"].([]string)
	require.True(t, ok, "expected npc_names in context")
	assert.Contains(t, npcNames, "Goblin Scout")
	assert.Contains(t, npcNames, "Goblin Chief")
}

func TestFateNarrationNoTakenOut_NoPendingMidFlow(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	engine.AddCharacter(player)

	testScene := scene.NewScene("test-scene", "Room", "A room.")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// No taken-out characters.
	sm.conflict.takenOutChars = nil

	ctx := context.Background()
	sm.conflict.promptPlayerForFates(ctx)

	assert.Nil(t, sm.actions.pendingMidFlow, "expected no pending mid-flow with no taken-out chars")
}

func TestProvideMidFlowResponse_FateNarration(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	mockLLM := &capturingMockLLMClient{
		response: `[FATE_NARRATION]{"narrative":"The goblins flee into the forest.","fates":[{"id":"npc-1","name":"Goblin Scout","description":"Fled into the woods","permanent":false}]}[/FATE_NARRATION]`,
	}
	engine.llmClient = mockLLM

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	npc1 := character.NewCharacter("npc-1", "Goblin Scout")
	npc1.Aspects.HighConcept = "Sneaky Goblin"
	engine.AddCharacter(player)
	engine.AddCharacter(npc1)

	testScene := scene.NewScene("test-scene", "Forest", "A dark forest.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(npc1.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	sm.conflict.takenOutChars = []string{npc1.ID}

	ctx := context.Background()
	sm.conflict.promptPlayerForFates(ctx)
	require.NotNil(t, sm.actions.pendingMidFlow)

	// Provide the narration.
	result, err := sm.ProvideMidFlowResponse(ctx, MidFlowResponse{Text: "I let the goblin flee"})
	require.NoError(t, err)
	require.NotNil(t, result)

	// pendingMidFlow should be cleared.
	assert.Nil(t, sm.actions.pendingMidFlow)

	// If parsing succeeded, events should include a NarrativeEvent.
	// (The actual LLM parsing may fail with our mock, but the continuation is exercised.)
}

// --- handleConcession tests ---

func TestConcessionEmitsFreeTextEvent(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	enemy := character.NewCharacter("enemy-1", "Goblin")
	player.FatePoints = 1
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	ctx := context.Background()
	sm.conflict.handleConcession(ctx)

	// Fate points should be awarded (synchronous part).
	assert.Equal(t, 2, player.FatePoints, "expected fate point for conceding")

	// Conflict should have ended (synchronous part).
	assert.False(t, sm.currentScene.IsConflict, "expected conflict to end")

	// Should have a pending mid-flow for the concession narration.
	require.NotNil(t, sm.actions.pendingMidFlow, "expected pendingMidFlow for concession narration")
	event := sm.actions.pendingMidFlow.event

	assert.Equal(t, uicontract.InputRequestFreeText, event.Type)
	assert.Contains(t, event.Prompt, "concede")
	assert.Equal(t, "concession_narration", event.Context["request_type"])

	// Concession should have been recorded in conversation history.
	history := sm.GetConversationHistory()
	require.NotEmpty(t, history, "expected conversation history entry for concession")
	last := history[len(history)-1]
	assert.Equal(t, "conflict", last.Type)
	assert.Contains(t, last.GMResponse, "conceded")
}

func TestProvideMidFlowResponse_ConcessionNarration(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	enemy := character.NewCharacter("enemy-1", "Goblin")
	player.FatePoints = 1
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	ctx := context.Background()
	sm.conflict.handleConcession(ctx)
	require.NotNil(t, sm.actions.pendingMidFlow)

	// Provide concession narration.
	result, err := sm.ProvideMidFlowResponse(ctx, MidFlowResponse{Text: "I raise my hands in surrender and back away."})
	require.NoError(t, err)
	require.NotNil(t, result)

	// pendingMidFlow should be cleared.
	assert.Nil(t, sm.actions.pendingMidFlow)

	// Events should include a NarrativeEvent with the player's narration.
	require.Len(t, result.Events, 1)
	narr, ok := result.Events[0].(NarrativeEvent)
	require.True(t, ok, "expected NarrativeEvent")
	assert.Contains(t, narr.Text, "surrender")
	assert.Contains(t, narr.Text, player.Name)

	// Conversation history should be updated.
	assert.NotEmpty(t, sm.conversationHistory)
}

func TestProvideMidFlowResponse_EmptyConcessionNarration(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	enemy := character.NewCharacter("enemy-1", "Goblin")
	player.FatePoints = 1
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	ctx := context.Background()
	sm.conflict.handleConcession(ctx)
	require.NotNil(t, sm.actions.pendingMidFlow)

	// Provide empty narration.
	result, err := sm.ProvideMidFlowResponse(ctx, MidFlowResponse{Text: ""})
	require.NoError(t, err)
	require.NotNil(t, result)

	// No events should be emitted for empty narration.
	assert.Empty(t, result.Events)
}

// --- ProvideMidFlowResponse error cases ---

func TestProvideMidFlowResponse_NoPending(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	ctx := context.Background()
	_, err = sm.ProvideMidFlowResponse(ctx, MidFlowResponse{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending mid-flow")
}

// --- HandleInput integration ---

func TestHandleInput_ConcessionSetsAwaitingMidFlow(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	// Need an LLM client for HandleInput (input classification).
	mockLLM := &capturingMockLLMClient{response: "dialog"}
	engine.llmClient = mockLLM
	engine.actionParser = NewActionParser(mockLLM)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	enemy := character.NewCharacter("enemy-1", "Goblin")
	player.FatePoints = 1
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("test-scene", "Arena", "An arena.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := sm.HandleInput(ctx, "concede")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.AwaitingMidFlow, "expected AwaitingMidFlow after concession")
	assert.False(t, result.SceneEnded, "scene should not end yet (waiting for narration)")

	// The events should include an InputRequestEvent so the UI can render the prompt.
	var foundInputRequest bool
	for _, ev := range result.Events {
		if ire, ok := ev.(InputRequestEvent); ok {
			foundInputRequest = true
			assert.Equal(t, uicontract.InputRequestFreeText, ire.Type)
			assert.Contains(t, ire.Prompt, "concede")
		}
	}
	assert.True(t, foundInputRequest, "expected InputRequestEvent in result events")
}

func TestHandleInput_RejectsInputWhileMidFlowPending(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	engine.AddCharacter(player)

	testScene := scene.NewScene("test-scene", "Room", "A room.")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Manually set a pending mid-flow.
	sm.actions.pendingMidFlow = &midFlowState{
		event: InputRequestEvent{
			Type:   uicontract.InputRequestFreeText,
			Prompt: "test",
		},
		continuation: func(_ context.Context, _ MidFlowResponse) []GameEvent { return nil },
	}

	ctx := context.Background()
	_, err = sm.HandleInput(ctx, "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mid-flow")
}

// --- ProvideMidFlowResponse integration tests ---

func TestProvideMidFlowResponse_NumberedChoice(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	mockLLM := &capturingMockLLMClient{
		response: `[CONSEQUENCE_ASPECT:Bruised Ribs]`,
	}
	engine.llmClient = mockLLM

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	attacker := character.NewCharacter("enemy-1", "Orc")
	engine.AddCharacter(player)
	engine.AddCharacter(attacker)

	testScene := scene.NewScene("test-scene", "Arena", "A dusty arena.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(attacker.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Trigger stress overflow to set pendingMidFlow with a numbered choice.
	ctx := context.Background()
	attackCtx := prompt.AttackContext{Skill: "Fight", Description: "Slash", Shifts: 2}
	sm.conflict.handleStressOverflow(ctx, 2, character.PhysicalStress, attacker, attackCtx)
	require.NotNil(t, sm.actions.pendingMidFlow)
	assert.Equal(t, uicontract.InputRequestNumberedChoice, sm.actions.pendingMidFlow.event.Type)

	// Provide the response directly (choose the first option — mild consequence).
	result, err := sm.ProvideMidFlowResponse(ctx, MidFlowResponse{ChoiceIndex: 0})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Mild consequence should have been applied.
	assert.Len(t, player.Consequences, 1)
	assert.Equal(t, character.MildConsequence, player.Consequences[0].Type)
}

func TestProvideMidFlowResponse_FreeText(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	enemy := character.NewCharacter("enemy-1", "Goblin")
	player.FatePoints = 1
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("test-scene", "Room", "A room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	ctx := context.Background()
	sm.conflict.handleConcession(ctx)
	require.NotNil(t, sm.actions.pendingMidFlow)
	assert.Equal(t, uicontract.InputRequestFreeText, sm.actions.pendingMidFlow.event.Type)

	// Provide the narration text directly.
	result, err := sm.ProvideMidFlowResponse(ctx, MidFlowResponse{Text: "I drop my weapon and back away."})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have a NarrativeEvent with the player's narration.
	require.Len(t, result.Events, 1)
	narr, ok := result.Events[0].(NarrativeEvent)
	require.True(t, ok, "expected NarrativeEvent, got %T", result.Events[0])
	assert.Contains(t, narr.Text, "drop my weapon")
}

// --- midFlowState helpers ---

func TestStressOverflowContextFields(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := character.NewCharacter("player-1", "Hero")
	attacker := character.NewCharacter("enemy-1", "Orc")
	engine.AddCharacter(player)
	engine.AddCharacter(attacker)

	testScene := scene.NewScene("test-scene", "Arena", "A dusty arena.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(attacker.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	ctx := context.Background()
	attackCtx := prompt.AttackContext{Skill: "Fight", Description: "A mighty blow", Shifts: 5}
	sm.conflict.handleStressOverflow(ctx, 5, character.PhysicalStress, attacker, attackCtx)
	require.NotNil(t, sm.actions.pendingMidFlow)

	event := sm.actions.pendingMidFlow.event

	// Verify context fields.
	assert.Equal(t, "consequence_choice", event.Context["request_type"])
	assert.Equal(t, 5, event.Context["shifts"])
	assert.Equal(t, "physical", event.Context["stress_type"])
}

// --- capturingMockLLMClient is defined in conflict_test.go, but we need it here too ---
// It's in the same package so we can reuse it.
// If it weren't available we'd define it here — but since both files are in
// package engine, Go compiles them together.

// Unused variable avoidance — reference imports that might appear unused.
var (
	_               = time.Now
	_ llm.LLMClient = (*capturingMockLLMClient)(nil)
)
