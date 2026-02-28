package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
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

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	response := "The orc swings his axe at you! [CONFLICT:physical:orc-1] Roll for initiative!"
	trigger, cleanedResponse := sm.conflict.parseConflictMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, scene.PhysicalConflict, trigger.Type)
	assert.Equal(t, "orc-1", trigger.InitiatorID)
	assert.Equal(t, "The orc swings his axe at you! Roll for initiative!", cleanedResponse)
}

func TestSceneManager_ParseConflictMarker_Mental(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	response := "The sorcerer locks eyes with you, attempting to dominate your mind. [CONFLICT:mental:sorcerer-1]"
	trigger, cleanedResponse := sm.conflict.parseConflictMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, scene.MentalConflict, trigger.Type)
	assert.Equal(t, "sorcerer-1", trigger.InitiatorID)
	assert.Equal(t, "The sorcerer locks eyes with you, attempting to dominate your mind.", cleanedResponse)
}

func TestSceneManager_ParseConflictMarker_NoMarker(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	response := "The merchant smiles and offers you a deal."
	trigger, cleanedResponse := sm.conflict.parseConflictMarker(response)

	assert.Nil(t, trigger)
	assert.Equal(t, "The merchant smiles and offers you a deal.", cleanedResponse)
}

func TestSceneManager_CalculateInitiative_Physical(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	char := character.NewCharacter("char1", "Fighter")
	char.SetSkill("Notice", 3)
	char.SetSkill("Athletics", 2)

	// Should use Notice for physical conflicts
	initiative := sm.conflict.calculateInitiative(char, scene.PhysicalConflict)
	assert.Equal(t, 3, initiative)
}

func TestSceneManager_CalculateInitiative_Physical_NoNotice(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	char := character.NewCharacter("char1", "Fighter")
	char.SetSkill("Athletics", 2)

	// Should fall back to Athletics when Notice is 0
	initiative := sm.conflict.calculateInitiative(char, scene.PhysicalConflict)
	assert.Equal(t, 2, initiative)
}

func TestSceneManager_CalculateInitiative_Mental(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	char := character.NewCharacter("char1", "Wizard")
	char.SetSkill("Empathy", 4)
	char.SetSkill("Rapport", 2)

	// Should use Empathy for mental conflicts
	initiative := sm.conflict.calculateInitiative(char, scene.MentalConflict)
	assert.Equal(t, 4, initiative)
}

func TestSceneManager_CalculateInitiative_Mental_NoEmpathy(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	char := character.NewCharacter("char1", "Diplomat")
	char.SetSkill("Rapport", 3)

	// Should fall back to Rapport when Empathy is 0
	initiative := sm.conflict.calculateInitiative(char, scene.MentalConflict)
	assert.Equal(t, 3, initiative)
}

func TestSceneManager_InitiateConflict(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
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

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	// Try to start another conflict
	err = sm.conflict.initiateConflict(scene.MentalConflict, player.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in a conflict")
}

func TestSceneManager_InitiateConflict_NotEnoughParticipants(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	// Create only one character
	player := character.NewCharacter("player1", "Hero")
	engine.AddCharacter(player)

	// Create scene with only player
	testScene := scene.NewScene("scene1", "Test Scene", "Alone")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	// Try to initiate conflict with only one participant
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, player.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 participants")
}

func TestSceneManager_InitiateConflict_UnknownInitiator(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, "none")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a known character")

	// Verify no conflict was started
	assert.False(t, sm.currentScene.IsConflict)
}

func TestSceneManager_HandleConflictEscalation(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	err = sm.conflict.initiateConflict(scene.MentalConflict, enemy.ID)
	require.NoError(t, err)

	// Verify initial mental initiative (player with Empathy 4 should be first)
	assert.Equal(t, scene.MentalConflict, sm.currentScene.ConflictState.Type)
	// Player has Empathy 4, enemy has Empathy 1
	firstParticipant := sm.currentScene.ConflictState.Participants[0]
	assert.Equal(t, player.ID, firstParticipant.CharacterID)
	assert.Equal(t, 4, firstParticipant.Initiative)

	// Escalate to physical
	sm.conflict.handleConflictEscalation(scene.PhysicalConflict)

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

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	response := "The goblin drops his spear and raises his hands. \"I yield!\" [CONFLICT:end:surrender]"
	resolution, cleanedResponse := sm.conflict.parseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "surrender", resolution.Reason)
	assert.Equal(t, "The goblin drops his spear and raises his hands. \"I yield!\"", cleanedResponse)
}

func TestSceneManager_ParseConflictEndMarker_Agreement(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	response := "The merchant nods slowly. \"Very well, we have a deal.\" [CONFLICT:end:agreement]"
	resolution, cleanedResponse := sm.conflict.parseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "agreement", resolution.Reason)
	assert.Equal(t, "The merchant nods slowly. \"Very well, we have a deal.\"", cleanedResponse)
}

func TestSceneManager_ParseConflictEndMarker_Retreat(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	response := "The orc looks at his fallen comrades and flees into the forest. [CONFLICT:end:retreat]"
	resolution, cleanedResponse := sm.conflict.parseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "retreat", resolution.Reason)
	assert.Equal(t, "The orc looks at his fallen comrades and flees into the forest.", cleanedResponse)
}

func TestSceneManager_ParseConflictEndMarker_NoMarker(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	response := "The guard eyes you suspiciously but does not attack."
	resolution, cleanedResponse := sm.conflict.parseConflictEndMarker(response)

	assert.Nil(t, resolution)
	assert.Equal(t, "The guard eyes you suspiciously but does not attack.", cleanedResponse)
}

func TestSceneManager_ResolveConflictPeacefully(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	// Start a conflict
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)
	assert.True(t, sm.currentScene.IsConflict)

	// Resolve peacefully
	sm.conflict.resolveConflictPeacefully("surrender")

	// Verify conflict ended
	assert.False(t, sm.currentScene.IsConflict)
}

func TestSceneManager_ResolveConflictPeacefully_ClearsStress(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	// Start a conflict
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	// Resolve peacefully (should clear all stress)
	sm.conflict.resolveConflictPeacefully("surrender")

	// Verify stress was cleared for all participants
	assert.Equal(t, 2, player.GetStressTrack(character.PhysicalStress).AvailableBoxes())
	assert.Equal(t, 2, player.GetStressTrack(character.MentalStress).AvailableBoxes())
	assert.Equal(t, 2, enemy.GetStressTrack(character.PhysicalStress).AvailableBoxes())
}

func TestSceneManager_ResolveConflictPeacefully_NotInConflict(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	// Setup scene (no conflict)
	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

	// Attempting to resolve should not panic
	sm.conflict.resolveConflictPeacefully("surrender")

	// Still not in conflict
	assert.False(t, sm.currentScene.IsConflict)
}

func TestSceneManager_ResolveConflictPeacefully_AllReasons(t *testing.T) {
	tests := []struct {
		reason   string
		expected string
	}{
		{"surrender", "Your opponent surrenders!"},
		{"agreement", "You've reached an agreement!"},
		{"retreat", "Your opponent retreats!"},
		{"resolved", "The conflict has been resolved!"},
		{"something_else", "The conflict ends!"},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			engine, err := New()
			require.NoError(t, err)

			sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
			player := character.NewCharacter("p1", "Hero")
			player.SetSkill("Notice", 2)
			enemy := character.NewCharacter("e1", "Goblin")
			enemy.SetSkill("Notice", 1)
			engine.AddCharacter(player)
			engine.AddCharacter(enemy)

			testScene := scene.NewScene("s1", "Room", "A room")
			testScene.AddCharacter(player.ID)
			testScene.AddCharacter(enemy.ID)
			sm.currentScene = testScene
			sm.conflict.currentScene = testScene
			sm.actions.currentScene = testScene
			sm.player = player
			sm.conflict.player = player
			sm.actions.player = player

			err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
			require.NoError(t, err)

			msg := sm.conflict.resolveConflictPeacefully(tt.reason)
			assert.Equal(t, tt.expected, msg)
			assert.False(t, sm.currentScene.IsConflict)
		})
	}
}

func TestSceneManager_HandleConflictEscalation_NotInConflict(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	testScene := scene.NewScene("s1", "Room", "A room")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

	events := sm.conflict.handleConflictEscalation(scene.PhysicalConflict)
	assert.Nil(t, events)
}

func TestSceneManager_HandleConflictEscalation_SameType(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	player := character.NewCharacter("p1", "Hero")
	player.SetSkill("Notice", 2)
	enemy := character.NewCharacter("e1", "Goblin")
	enemy.SetSkill("Notice", 1)
	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("s1", "Room", "A room")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	events := sm.conflict.handleConflictEscalation(scene.PhysicalConflict)
	assert.Nil(t, events, "escalation to same type should be a no-op")
}

func TestSceneManager_GatherInvokableAspects_ConsequencesAndSituation(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	player := character.NewCharacter("p1", "Hero")
	player.Aspects.HighConcept = "Bold Knight"
	player.AddConsequence(character.Consequence{
		ID:     "c1",
		Type:   character.MildConsequence,
		Aspect: "Bruised Ribs",
	})
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	testScene := scene.NewScene("s1", "Room", "A room")
	testScene.SituationAspects = append(testScene.SituationAspects, scene.SituationAspect{
		ID:          "sa-1",
		Aspect:      "Slippery Floor",
		FreeInvokes: 1,
	})
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

	aspects := sm.actions.gatherInvokableAspects(map[string]bool{})
	var names []string
	for _, a := range aspects {
		names = append(names, a.Name)
	}
	assert.Contains(t, names, "Bold Knight")
	assert.Contains(t, names, "Bruised Ribs")
	assert.Contains(t, names, "Slippery Floor")

	// Verify sources
	for _, a := range aspects {
		switch a.Name {
		case "Bruised Ribs":
			assert.Equal(t, "consequence", a.Source)
		case "Slippery Floor":
			assert.Equal(t, "situation", a.Source)
			assert.Equal(t, 1, a.FreeInvokes)
		}
	}
}

func TestSceneManager_RollTargetDefense(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	sm.actions.roller = dice.NewSeededRoller(12345) // Predictable rolls

	target := character.NewCharacter("target-1", "Goblin")
	target.SetSkill("Athletics", 2)
	target.SetSkill("Will", 1)

	// Test physical attack defense (uses Athletics)
	defenseResult, defEvent := sm.actions.rollTargetDefense(target, "Fight")
	assert.NotNil(t, defenseResult)
	assert.NotEmpty(t, defEvent.DefenderName)
	// With seeded roller and Athletics +2, we get a predictable result

	// Test mental attack defense (uses Will)
	defenseResult, defEvent = sm.actions.rollTargetDefense(target, "Provoke")
	assert.NotNil(t, defenseResult)
	assert.NotEmpty(t, defEvent.DefenderName)
}

func TestSceneManager_ApplyDamageToTarget_StressAbsorbed(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	target := character.NewCharacter("target-1", "Goblin")
	// Default stress track should be able to absorb small hits

	dmgEvent := sm.conflict.applyDamageToTarget(context.Background(), target, 1, character.PhysicalStress)

	// Check that stress was absorbed
	assert.NotNil(t, dmgEvent.Absorbed, "Expected stress absorption")
	assert.Equal(t, "physical", dmgEvent.Absorbed.TrackType)
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

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	// Start a conflict
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, target.ID)
	require.NoError(t, err)

	// Take out the target
	dmgEvent := &DamageResolutionEvent{TargetName: target.Name}
	sm.conflict.applyTargetTakenOut(context.Background(), target, dmgEvent)
	assert.True(t, dmgEvent.TakenOut)

	// Check that target is marked as taken out (conflict still active because of otherEnemy)
	participant := sm.currentScene.ConflictState.GetParticipant(target.ID)
	require.NotNil(t, participant)
	assert.Equal(t, scene.StatusTakenOut, participant.Status)

	// Conflict should still be active since otherEnemy remains
	assert.True(t, sm.currentScene.IsConflict)
	assert.False(t, dmgEvent.VictoryEnd)
}

func TestSceneManager_HandleTargetTakenOut_ConflictEnds(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	// Start a conflict
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, target.ID)
	require.NoError(t, err)

	// Take out the only target
	dmgEvent := &DamageResolutionEvent{TargetName: target.Name}
	sm.conflict.applyTargetTakenOut(context.Background(), target, dmgEvent)

	// Conflict should end since no active opponents remain
	assert.False(t, sm.currentScene.IsConflict)

	// Check victory via event
	assert.True(t, dmgEvent.VictoryEnd, "Expected victory end")
}

func TestSceneManager_HandleTargetTakenOut_MarksSceneLevelTakenOut(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	// Start a conflict
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, target.ID)
	require.NoError(t, err)

	// Take out the target
	dmgEvent := &DamageResolutionEvent{TargetName: target.Name}
	sm.conflict.applyTargetTakenOut(context.Background(), target, dmgEvent)

	// Conflict ends, but character should still be marked as taken out at scene level
	assert.False(t, sm.currentScene.IsConflict)
	assert.True(t, sm.currentScene.IsCharacterTakenOut(target.ID))
}

func TestSceneManager_InitiateConflict_ExcludesTakenOutCharacters(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	// Start first conflict and take out enemy1
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy1.ID)
	require.NoError(t, err)

	dmgEvent := &DamageResolutionEvent{TargetName: enemy1.Name}
	sm.conflict.applyTargetTakenOut(context.Background(), enemy1, dmgEvent)
	// Conflict still ongoing because enemy2 is active
	assert.True(t, sm.currentScene.IsConflict)

	// End the conflict manually (simulating peaceful resolution)
	sm.currentScene.EndConflict()
	assert.False(t, sm.currentScene.IsConflict)

	// Try to initiate a new conflict with enemy2
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy2.ID)
	require.NoError(t, err)

	// enemy1 should NOT be in the new conflict since they were taken out
	participant := sm.currentScene.ConflictState.GetParticipant(enemy1.ID)
	assert.Nil(t, participant, "Taken-out character should not be in new conflict")

	// enemy2 and player should be in the conflict
	assert.NotNil(t, sm.currentScene.ConflictState.GetParticipant(enemy2.ID))
	assert.NotNil(t, sm.currentScene.ConflictState.GetParticipant(player.ID))
}

func TestSceneManager_InitiateConflict_TakenOutInitiatorFails(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	enemy := character.NewCharacter("enemy-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(enemy)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(enemy.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	// Start first conflict and take out enemy
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	dmgEvent := &DamageResolutionEvent{TargetName: enemy.Name}
	sm.conflict.applyTargetTakenOut(context.Background(), enemy, dmgEvent)
	assert.False(t, sm.currentScene.IsConflict) // Conflict ended

	// Enemy is marked as taken out at scene level
	assert.True(t, sm.currentScene.IsCharacterTakenOut(enemy.ID))

	// Try to have the taken-out enemy initiate a new conflict - should fail
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "taken out")
}

func TestSceneManager_ApplyActionEffects_Attack(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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

	events := sm.actions.applyActionEffects(context.Background(), testAction, target)

	// Check that events contain attack result with shifts
	ar := RequireFirstFrom[PlayerAttackResultEvent](t, events)
	assert.Greater(t, ar.Shifts, 0)
}

func TestSceneManager_IsConcedeCommand(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
			result := sm.conflict.isConcedeCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSceneManager_HandleConcession(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)
	require.True(t, sm.currentScene.IsConflict)

	// Handle concession
	ctx := context.Background()
	events := sm.conflict.handleConcession(ctx)

	// Check fate point was awarded (1 base + 1 for conceding = 2)
	assert.Equal(t, 2, player.FatePoints, "Expected fate point for conceding")

	// Check conflict ended
	assert.False(t, sm.currentScene.IsConflict, "Expected conflict to end")

	// Check events contain ConcessionEvent
	ce := RequireFirstFrom[ConcessionEvent](t, events)
	assert.Equal(t, 1, ce.FatePointsGained)
	assert.Equal(t, 2, ce.CurrentFatePoints)
}

func TestSceneManager_HandleConcession_WithConsequences(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, enemy.ID)
	require.NoError(t, err)

	// Handle concession
	ctx := context.Background()
	events := sm.conflict.handleConcession(ctx)

	// Check fate points: 1 base + 1 for conceding + 2 for consequences = 4
	assert.Equal(t, 4, player.FatePoints, "Expected fate points for conceding with consequences")

	// Check ConcessionEvent mentions consequences
	concessions := SliceOfType[ConcessionEvent](events)
	require.NotEmpty(t, concessions, "Expected ConcessionEvent with consequence count")
	var foundConsequenceBonus bool
	for _, ce := range concessions {
		if ce.ConsequenceCount > 0 {
			foundConsequenceBonus = true
			assert.Equal(t, 3, ce.FatePointsGained)
			assert.Equal(t, 2, ce.ConsequenceCount)
			break
		}
	}
	assert.True(t, foundConsequenceBonus, "Expected ConcessionEvent with consequence count")
}

func TestSceneManager_ApplyActionEffects_Attack_NilTarget_ShowsError(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	events := sm.actions.applyActionEffects(context.Background(), testAction, nil)

	// Should return a PlayerAttackResultEvent with TargetMissing
	ar := RequireFirstFrom[PlayerAttackResultEvent](t, events)
	assert.True(t, ar.TargetMissing)
	assert.Equal(t, "Bart the Outlaw", ar.TargetHint)
}

func TestSceneManager_ApplyActionEffects_Attack_DealsDamage(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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

	events := sm.actions.applyActionEffects(context.Background(), testAction, target)

	// Verify stress was actually applied
	afterAvailable := target.GetStressTrack(character.PhysicalStress).AvailableBoxes()
	assert.Less(t, afterAvailable, initialAvailable, "Target should have taken stress")

	// Verify events contain attack result and damage resolution
	attackResults := SliceOfType[PlayerAttackResultEvent](events)
	require.NotEmpty(t, attackResults, "Expected PlayerAttackResultEvent with shifts")
	assert.Greater(t, attackResults[0].Shifts, 0)

	damageResults := SliceOfType[DamageResolutionEvent](events)
	require.NotEmpty(t, damageResults, "Expected DamageResolutionEvent with absorption")
	assert.NotNil(t, damageResults[0].Absorbed)
}

func TestSceneManager_ApplyActionEffects_Attack_Tie_GrantsBoost(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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

	events := sm.actions.applyActionEffects(context.Background(), testAction, target)

	// Attack result reports a tie.
	ar := RequireFirstFrom[PlayerAttackResultEvent](t, events)
	assert.True(t, ar.IsTie, "Expected PlayerAttackResultEvent with IsTie")

	// Attacker (player) gets a boost on a tie.
	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost, "tie on attack should create a boost for the attacker")
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, player.ID, sm.currentScene.SituationAspects[0].CreatedBy)
}

// Fate Core SRD (Defend): "If you succeed with style on a defend action, you
// get a boost instead of just succeeding." When the NPC player attacks and the
// NPC defender rolls 3+ more than the attacker, the NPC gets a boost.
func TestSceneManager_ApplyActionEffects_Attack_Failure_DefendWithStyle_GrantsTargetBoost(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	testAction := action.NewActionWithTarget("action-1", player.ID, action.Attack, "Fight", "Strike", target.ID)
	// Shifts = -3 → attacker lost by 3, defender succeeded with style.
	testAction.Outcome = &dice.Outcome{Type: dice.Failure, Shifts: -3}

	events := sm.actions.applyActionEffects(context.Background(), testAction, target)

	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost, "defending with style should create a boost for the defender")
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, target.ID, sm.currentScene.SituationAspects[0].CreatedBy)
}

// Fate Core SRD (Create Advantage, Tie): "You get a boost instead of the full aspect."
func TestSceneManager_ApplyActionEffects_CreateAdvantage_Tie_CreatesBoost(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	engine.AddCharacter(player)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	testAction := action.NewAction("action-1", player.ID, action.CreateAdvantage, "Notice", "Look for an opening")
	testAction.Outcome = &dice.Outcome{Type: dice.Tie, Shifts: 0}

	events := sm.actions.applyActionEffects(context.Background(), testAction, nil)

	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost, "CaA tie should create a boost, not a full aspect")
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, player.ID, sm.currentScene.SituationAspects[0].CreatedBy)
}

// Fate Core SRD (Overcome, SWS): "You succeed with style and... can be used to
// gain a boost." Overcome with SWS grants the player a boost.
func TestSceneManager_ApplyActionEffects_Overcome_SWS_CreatesBoost(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	engine.AddCharacter(player)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	testAction := action.NewAction("action-1", player.ID, action.Overcome, "Athletics", "Vault the obstacle")
	testAction.Outcome = &dice.Outcome{Type: dice.SuccessWithStyle, Shifts: 3}

	events := sm.actions.applyActionEffects(context.Background(), testAction, nil)

	boostEvt := RequireFirstFrom[AspectCreatedEvent](t, events)
	assert.True(t, boostEvt.IsBoost, "Overcome SWS should create a boost for the player")
	assert.Equal(t, 1, boostEvt.FreeInvokes)

	require.Len(t, sm.currentScene.SituationAspects, 1)
	assert.True(t, sm.currentScene.SituationAspects[0].IsBoost)
	assert.Equal(t, player.ID, sm.currentScene.SituationAspects[0].CreatedBy)
}

// errorAspectGenerator is a mock AspectGenerator that always returns an error,
// exercising the fallback path inside generateBoostName.
type errorAspectGenerator struct{}

func (e *errorAspectGenerator) GenerateAspect(_ context.Context, _ prompt.AspectGenerationRequest) (*AspectGenerationResponse, error) {
	return nil, errors.New("LLM unavailable")
}

// emptyAspectGenerator returns a response with an empty AspectText, exercising
// the empty-string fallback branch.
type emptyAspectGenerator struct{}

func (e *emptyAspectGenerator) GenerateAspect(_ context.Context, _ prompt.AspectGenerationRequest) (*AspectGenerationResponse, error) {
	return &AspectGenerationResponse{AspectText: ""}, nil
}

func TestGenerateBoostName_FallbackOnError(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	sm.actions.aspectGenerator = &errorAspectGenerator{}

	player := character.NewCharacter("player-1", "Hero")
	engine.AddCharacter(player)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	name := sm.actions.generateBoostName(context.Background(), player, "Fight", "strike hard", "Fleeting Opening")
	assert.Equal(t, "Fleeting Opening", name, "should return fallback when LLM errors")
}

func TestGenerateBoostName_FallbackOnEmptyResponse(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	sm.actions.aspectGenerator = &emptyAspectGenerator{}

	player := character.NewCharacter("player-1", "Hero")
	engine.AddCharacter(player)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	err = sm.StartScene(testScene, player)
	require.NoError(t, err)

	name := sm.actions.generateBoostName(context.Background(), player, "Athletics", "vault", "Strong Momentum")
	assert.Equal(t, "Strong Momentum", name, "should return fallback when LLM returns empty aspect text")
}

func TestSceneManager_ResolveAction_TargetByName(t *testing.T) {
	// Tests the core fix for issue #25: resolveAction should find targets by name
	mockClient := &MockLLMClient{response: "The attack strikes true!"}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := engine.GetSceneManager()
	sm.actions.roller = dice.NewSeededRoller(42)

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
	events, _ := sm.actions.resolveAction(ctx, testAction)

	// The attack should have resolved against Bart — check defense was rolled
	// and damage was applied (if target wasn't found, no damage messages would appear)
	defenses := SliceOfType[DefenseRollEvent](events)
	var foundDefenseResult bool
	for _, def := range defenses {
		if def.DefenderName == "Bart the Outlaw" {
			foundDefenseResult = true
			break
		}
	}
	assert.True(t, foundDefenseResult,
		"Expected DefenseRollEvent for 'Bart the Outlaw' via name lookup, got: %v",
		events)

	attacks := SliceOfType[PlayerAttackResultEvent](events)
	var foundDamageApplied bool
	for _, atk := range attacks {
		if atk.TargetName == "Bart the Outlaw" {
			foundDamageApplied = true
			break
		}
	}
	assert.True(t, foundDamageApplied,
		"Expected PlayerAttackResultEvent for 'Bart the Outlaw', got: %v",
		events)
}

func TestSceneManager_ResolveAction_UnknownTarget_AbortsWithoutConsumingTurn(t *testing.T) {
	// When a target can't be resolved, the action should abort early:
	// no dice roll, no narrative, no turn advancement.
	mockClient := &MockLLMClient{response: "Should not be called"}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := engine.GetSceneManager()
	sm.actions.roller = dice.NewSeededRoller(42)

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
	events, awaiting := sm.actions.resolveAction(ctx, testAction)
	assert.False(t, awaiting, "unknown target should not await invoke")

	// Should see the "try again" message in events
	sysMsgs := SliceOfType[SystemMessageEvent](events)
	foundTryAgain := false
	for _, sysMsg := range sysMsgs {
		if strings.Contains(sysMsg.Message, "Could not find target") && strings.Contains(sysMsg.Message, "try again") {
			foundTryAgain = true
		}
	}
	assert.True(t, foundTryAgain,
		"Expected 'try again' message for unknown target, got events: %v", events)

	// Should NOT see any dice results or narratives in events
	AssertNoEventIn[ActionResultEvent](t, events)
	AssertNoEventIn[NarrativeEvent](t, events)
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

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

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
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player
	sm.actions.roller = dice.NewSeededRoller(12345)

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

// TestApplyTargetTakenOut_SetsFateOnCharacter verifies that applyTargetTakenOut
// sets target.Fate so that IsTakenOut() returns true immediately. This is
// required for BuildStateSnapshot to report isTakenOut correctly.
// Regression test for https://github.com/C-Ross/LlamaOfFate/issues/118
func TestApplyTargetTakenOut_SetsFateOnCharacter(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)

	player := character.NewCharacter("player-1", "Hero")
	target := character.NewCharacter("target-1", "Goblin")

	engine.AddCharacter(player)
	engine.AddCharacter(target)

	testScene := scene.NewScene("test-scene", "Test Room", "A test room.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(target.ID)
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	// Start a conflict
	err = sm.conflict.initiateConflict(scene.PhysicalConflict, target.ID)
	require.NoError(t, err)

	// Precondition: target is not taken out
	assert.False(t, target.IsTakenOut(), "target should not be taken out before damage")

	// Take out the target
	dmgEvent := &DamageResolutionEvent{TargetName: target.Name}
	sm.conflict.applyTargetTakenOut(context.Background(), target, dmgEvent)

	// The character's Fate must be set so IsTakenOut() is true immediately
	assert.True(t, target.IsTakenOut(),
		"target.IsTakenOut() should be true after applyTargetTakenOut (issue #118)")
	require.NotNil(t, target.Fate,
		"target.Fate must be set by applyTargetTakenOut")
}
