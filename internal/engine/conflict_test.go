package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingMockLLMClient captures prompts sent to the LLM and returns a fixed response
type capturingMockLLMClient struct {
	response        string
	capturedPrompts []string
}

func (c *capturingMockLLMClient) ChatCompletion(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	for _, msg := range req.Messages {
		c.capturedPrompts = append(c.capturedPrompts, msg.Content)
	}
	return &llm.CompletionResponse{
		ID:      "test",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "test",
		Choices: []llm.CompletionResponseChoice{
			{
				Index:        0,
				Message:      llm.Message{Role: "assistant", Content: c.response},
				FinishReason: "stop",
			},
		},
		Usage: llm.CompletionUsage{PromptTokens: 10, CompletionTokens: 10, TotalTokens: 20},
	}, nil
}

func (c *capturingMockLLMClient) ChatCompletionStream(ctx context.Context, req llm.CompletionRequest, handler llm.StreamHandler) error {
	return nil
}

func (c *capturingMockLLMClient) GetModelInfo() llm.ModelInfo {
	return llm.ModelInfo{Name: "test", MaxTokens: 4096}
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

func TestSceneManager_InitiateConflict_UnknownInitiator(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	mockUI := &MockUI{}
	sm := NewSceneManager(engine)
	sm.SetUI(mockUI)

	// Create player and enemy
	player := character.NewCharacter("player1", "Hero")
	enemy := character.NewCharacter("enemy1", "Goblin")
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	// Create scene with both characters
	testScene := scene.NewScene("scene1", "Test Scene", "A dangerous encounter")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Try to initiate conflict with an ID that doesn't match any character (LLM hallucination)
	err = sm.initiateConflict(scene.PhysicalConflict, "none")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a known character")

	// Verify no conflict was started
	assert.False(t, sm.currentScene.IsConflict)
	assert.Empty(t, mockUI.conflictStartCalls)
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

	sm.applyDamageToTarget(context.Background(), target, 1, character.PhysicalStress)

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
	sm.handleTargetTakenOut(context.Background(), target)

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
	sm.handleTargetTakenOut(context.Background(), target)

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
	sm.handleTargetTakenOut(context.Background(), target)

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

	sm.handleTargetTakenOut(context.Background(), enemy1)
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

	sm.handleTargetTakenOut(context.Background(), enemy)
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
	events, awaiting := sm.resolveAction(ctx, testAction)
	assert.False(t, awaiting, "unknown target should not await invoke")

	// Should see the "try again" message in events
	foundTryAgain := false
	for _, event := range events {
		if sysMsg, ok := event.(SystemMessageEvent); ok {
			if strings.Contains(sysMsg.Message, "Could not find target") && strings.Contains(sysMsg.Message, "try again") {
				foundTryAgain = true
			}
		}
	}
	assert.True(t, foundTryAgain,
		"Expected 'try again' message for unknown target, got events: %v", events)

	// Should NOT see any dice results or narratives in events
	for _, event := range events {
		_, isActionResult := event.(ActionResultEvent)
		assert.False(t, isActionResult,
			"Should not roll dice when target is unknown")
		_, isNarrative := event.(NarrativeEvent)
		assert.False(t, isNarrative,
			"Should not generate narrative when target is unknown")
	}
}

func TestSceneManager_HandleAction_ExcludesTakenOutFromTargets(t *testing.T) {
	// Mock LLM returns a valid action parse response targeting the active NPC
	actionResponse := `{
		"action_type": "Attack",
		"skill": "Fight",
		"description": "Punch the orc",
		"target": "active-npc",
		"difficulty": 2,
		"reasoning": "Physical attack",
		"confidence": 9
	}`
	mockClient := &capturingMockLLMClient{response: actionResponse}

	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	mockUI := &MockUI{}
	sm.SetUI(mockUI)

	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Fight", dice.Good)
	activeNPC := character.NewCharacter("active-npc", "Angry Orc")
	takenOutNPC := character.NewCharacter("taken-out-npc", "Defeated Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(activeNPC)
	engine.AddCharacter(takenOutNPC)

	testScene := scene.NewScene("test-scene", "Battle Room", "A room with enemies.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(activeNPC.ID)
	testScene.AddCharacter(takenOutNPC.ID)

	sm.currentScene = testScene
	sm.player = player
	sm.roller = dice.NewSeededRoller(12345)

	// Mark the goblin as taken out
	testScene.MarkCharacterTakenOut(takenOutNPC.ID)

	// Call handleAction — this will invoke the action parser with the LLM
	sm.handleAction(context.Background(), "I punch the orc")

	// Verify the taken-out NPC name does NOT appear in any prompt sent to the LLM
	allPrompts := strings.Join(mockClient.capturedPrompts, "\n")
	assert.NotContains(t, allPrompts, "Defeated Goblin",
		"Taken-out NPC should not appear in action parser prompt")
	assert.Contains(t, allPrompts, "Angry Orc",
		"Active NPC should appear in action parser prompt")
}
