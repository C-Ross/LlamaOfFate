package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

// setupConflictSM creates a SceneManager with a player and attacker in an
// active physical conflict. The engine optionally has an LLM client.
func setupConflictSM(t *testing.T, llmClient *capturingMockLLMClient) (*SceneManager, *character.Character, *character.Character) {
	t.Helper()

	var engine *Engine
	var err error
	if llmClient != nil {
		engine, err = NewWithLLM(llmClient)
	} else {
		engine, err = New()
	}
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	player := character.NewCharacter("player-1", "Hero")
	attacker := character.NewCharacter("npc-1", "Bandit")
	attacker.Aspects.HighConcept = "Ruthless Highwayman"

	engine.AddCharacter(player)
	engine.AddCharacter(attacker)

	testScene := scene.NewScene("test-scene", "Dusty Road", "A lonely stretch of highway.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(attacker.ID)
	sm.currentScene = testScene
	sm.player = player

	err = sm.initiateConflict(scene.PhysicalConflict, attacker.ID)
	require.NoError(t, err)

	return sm, player, attacker
}

// testAttackCtx returns a minimal AttackContext for test use.
func testAttackCtx() prompt.AttackContext {
	return prompt.AttackContext{
		Skill:       "Fight",
		Description: "swings a sword",
		Shifts:      2,
	}
}

// --- applyAttackDamageToPlayer ---

// Per Fate Core SRD (Resolving Attacks p.160): on a successful hit the attack
// deals shifts of stress equal to the number of shifts obtained (minimum 1).
// Stress is absorbed by checking the box equal to the shifts value on the
// appropriate track (1-indexed).
func TestApplyAttackDamageToPlayer_SuccessStressAbsorbed(t *testing.T) {
	sm, player, attacker := setupConflictSM(t, nil)

	// 1-shift hit → check stress box 1 (0-indexed box 0) on physical track.
	// Default character has 2 physical boxes, so this is absorbable.
	outcome := &dice.Outcome{Type: dice.Success, Shifts: 1}
	atkCtx := testAttackCtx()

	events := sm.applyAttackDamageToPlayer(context.Background(), outcome, attacker, atkCtx)
	require.Len(t, events, 1)

	stressEvt, ok := events[0].(PlayerStressEvent)
	require.True(t, ok, "expected PlayerStressEvent, got %T", events[0])
	assert.Equal(t, 1, stressEvt.Shifts)
	assert.Equal(t, "physical", stressEvt.StressType)

	// Verify the box is actually checked on the character.
	track := player.StressTracks["physical"]
	assert.True(t, track.Boxes[0], "box 1 should be checked")
}

// Fate Core SRD: Success with Style on an attack still deals shifts of stress
// (which will be >= 3) — there is no special bonus on attack SWS beyond high shifts.
func TestApplyAttackDamageToPlayer_SuccessWithStyle_HighShiftsAbsorbed(t *testing.T) {
	sm, player, attacker := setupConflictSM(t, nil)

	// Default character has 2 stress boxes, so a 2-shift hit uses box 2.
	outcome := &dice.Outcome{Type: dice.SuccessWithStyle, Shifts: 2}
	atkCtx := testAttackCtx()

	events := sm.applyAttackDamageToPlayer(context.Background(), outcome, attacker, atkCtx)
	require.Len(t, events, 1)

	stressEvt, ok := events[0].(PlayerStressEvent)
	require.True(t, ok, "expected PlayerStressEvent, got %T", events[0])
	assert.Equal(t, 2, stressEvt.Shifts)
	assert.Equal(t, "physical", stressEvt.StressType)

	track := player.StressTracks["physical"]
	assert.True(t, track.Boxes[1], "box 2 should be checked")
}

// Fate Core SRD: a hit with shifts equal to 0 on a Success still deals 1 shift
// (minimum 1 rule — the code clamps shifts < 1 to 1).
func TestApplyAttackDamageToPlayer_SuccessMinimumOneShift(t *testing.T) {
	sm, player, attacker := setupConflictSM(t, nil)

	outcome := &dice.Outcome{Type: dice.Success, Shifts: 0}
	atkCtx := testAttackCtx()

	events := sm.applyAttackDamageToPlayer(context.Background(), outcome, attacker, atkCtx)
	require.Len(t, events, 1)

	stressEvt, ok := events[0].(PlayerStressEvent)
	require.True(t, ok, "expected PlayerStressEvent, got %T", events[0])
	assert.Equal(t, 1, stressEvt.Shifts, "minimum 1 shift on success")

	track := player.StressTracks["physical"]
	assert.True(t, track.Boxes[0], "box 1 should be checked (min-1 rule)")
}

// Fate Core SRD: when stress exceeds the track, the character must take
// consequences or be taken out. Here we overflow a 2-box track with a 3-shift hit.
// This should trigger handleStressOverflow which sets pendingMidFlow.
func TestApplyAttackDamageToPlayer_StressOverflow_PromptsMidFlow(t *testing.T) {
	sm, player, attacker := setupConflictSM(t, nil)

	// 3-shift hit on a character with 2 physical stress boxes → overflow.
	outcome := &dice.Outcome{Type: dice.Success, Shifts: 3}
	atkCtx := testAttackCtx()

	events := sm.applyAttackDamageToPlayer(context.Background(), outcome, attacker, atkCtx)

	// Should contain a StressOverflowEvent.
	var hasOverflow bool
	for _, e := range events {
		if _, ok := e.(StressOverflowEvent); ok {
			hasOverflow = true
			break
		}
	}
	assert.True(t, hasOverflow, "expected StressOverflowEvent for 3-shift hit on 2-box track")

	// pendingMidFlow should be set with consequence choices per Fate Core.
	require.NotNil(t, sm.pendingMidFlow, "expected pendingMidFlow to be set for consequence choice")
	assert.Contains(t, sm.pendingMidFlow.event.Prompt, "choose")

	// Available consequences: mild(2), moderate(4), severe(6) + "Be Taken Out".
	slots := player.AvailableConsequenceSlots()
	expectedOptions := len(slots) + 1 // +1 for "Be Taken Out"
	assert.Equal(t, expectedOptions, len(sm.pendingMidFlow.event.Options))

	// Last option should be "Be Taken Out".
	lastOpt := sm.pendingMidFlow.event.Options[len(sm.pendingMidFlow.event.Options)-1]
	assert.Equal(t, "Be Taken Out", lastOpt.Label)
}

// Fate Core SRD: On a Tie the attacker gets a boost but deals no stress.
func TestApplyAttackDamageToPlayer_Tie_NoStress(t *testing.T) {
	sm, player, attacker := setupConflictSM(t, nil)

	outcome := &dice.Outcome{Type: dice.Tie, Shifts: 0}
	atkCtx := testAttackCtx()

	events := sm.applyAttackDamageToPlayer(context.Background(), outcome, attacker, atkCtx)
	require.Len(t, events, 1)

	defEvt, ok := events[0].(PlayerDefendedEvent)
	require.True(t, ok, "expected PlayerDefendedEvent, got %T", events[0])
	assert.True(t, defEvt.IsTie, "Tie should set IsTie=true (attacker gets a boost)")

	// No stress should be applied.
	track := player.StressTracks["physical"]
	for i, box := range track.Boxes {
		assert.False(t, box, "stress box %d should not be checked on a tie", i+1)
	}
}

// Fate Core SRD: On a Failure the attack misses entirely — no stress, no boost.
func TestApplyAttackDamageToPlayer_Failure_NoStress(t *testing.T) {
	sm, player, attacker := setupConflictSM(t, nil)

	outcome := &dice.Outcome{Type: dice.Failure, Shifts: -2}
	atkCtx := testAttackCtx()

	events := sm.applyAttackDamageToPlayer(context.Background(), outcome, attacker, atkCtx)
	require.Len(t, events, 1)

	defEvt, ok := events[0].(PlayerDefendedEvent)
	require.True(t, ok, "expected PlayerDefendedEvent, got %T", events[0])
	assert.False(t, defEvt.IsTie, "Failure should set IsTie=false")

	track := player.StressTracks["physical"]
	for i, box := range track.Boxes {
		assert.False(t, box, "stress box %d should not be checked on a failure", i+1)
	}
}

// Fate Core SRD: mental conflict uses the mental stress track.
func TestApplyAttackDamageToPlayer_MentalConflict_UsesMentalStress(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine)

	player := character.NewCharacter("player-1", "Hero")
	attacker := character.NewCharacter("npc-1", "Illusionist")

	engine.AddCharacter(player)
	engine.AddCharacter(attacker)

	testScene := scene.NewScene("test-scene", "Dream Realm", "A swirling void.")
	testScene.AddCharacter(player.ID)
	testScene.AddCharacter(attacker.ID)
	sm.currentScene = testScene
	sm.player = player

	// Start a MENTAL conflict.
	err = sm.initiateConflict(scene.MentalConflict, attacker.ID)
	require.NoError(t, err)

	outcome := &dice.Outcome{Type: dice.Success, Shifts: 1}
	atkCtx := prompt.AttackContext{Skill: "Provoke", Description: "mind assault", Shifts: 1}

	events := sm.applyAttackDamageToPlayer(context.Background(), outcome, attacker, atkCtx)
	require.Len(t, events, 1)

	stressEvt, ok := events[0].(PlayerStressEvent)
	require.True(t, ok, "expected PlayerStressEvent, got %T", events[0])
	assert.Equal(t, "mental", stressEvt.StressType)

	mentalTrack := player.StressTracks["mental"]
	assert.True(t, mentalTrack.Boxes[0], "mental box 1 should be checked")

	physTrack := player.StressTracks["physical"]
	for i, box := range physTrack.Boxes {
		assert.False(t, box, "physical box %d should NOT be checked in mental conflict", i+1)
	}
}

// --- handleStressOverflow ---

// SRD: When no consequence slots are available the character is taken out.
func TestHandleStressOverflow_NoConsequencesAvailable_TakenOut(t *testing.T) {
	// Use an LLM mock so handleTakenOut → generateTakenOutNarrativeAndOutcome
	// can complete without error.
	mockLLM := &capturingMockLLMClient{
		response: `{"narrative":"You fall.","outcome":"game_over","new_scene_hint":""}`,
	}
	sm, player, attacker := setupConflictSM(t, mockLLM)

	// Fill all consequence slots so none are available.
	player.AddConsequence(character.Consequence{ID: "c1", Type: character.MildConsequence, Aspect: "Bruised"})
	player.AddConsequence(character.Consequence{ID: "c2", Type: character.ModerateConsequence, Aspect: "Broken Arm"})
	player.AddConsequence(character.Consequence{ID: "c3", Type: character.SevereConsequence, Aspect: "Shattered"})

	events := sm.handleStressOverflow(context.Background(), 3, character.PhysicalStress, attacker, testAttackCtx())

	// Should include StressOverflowEvent with NoConsequences=true and PlayerTakenOutEvent.
	var hasNoConseq, hasTakenOut bool
	for _, e := range events {
		switch ev := e.(type) {
		case StressOverflowEvent:
			if ev.NoConsequences {
				hasNoConseq = true
			}
		case PlayerTakenOutEvent:
			hasTakenOut = true
		}
	}
	assert.True(t, hasNoConseq, "expected StressOverflowEvent with NoConsequences=true")
	assert.True(t, hasTakenOut, "expected PlayerTakenOutEvent when no consequences left")
}

// SRD: When consequence slots are available, the player gets a choice
// of consequence or being taken out.
func TestHandleStressOverflow_ConsequencesAvailable_SetsMidFlow(t *testing.T) {
	sm, _, attacker := setupConflictSM(t, nil)

	events := sm.handleStressOverflow(context.Background(), 3, character.PhysicalStress, attacker, testAttackCtx())

	// First event is StressOverflowEvent.
	require.NotEmpty(t, events)
	overflowEvt, ok := events[0].(StressOverflowEvent)
	require.True(t, ok)
	assert.Equal(t, 3, overflowEvt.Shifts)

	// pendingMidFlow should be set.
	require.NotNil(t, sm.pendingMidFlow)
	// Options: mild, moderate, severe, + Be Taken Out = 4.
	assert.Len(t, sm.pendingMidFlow.event.Options, 4)
}

// --- applyConsequence ---

// SRD (Consequences p.162): A mild consequence absorbs 2 shifts. If that's
// enough to cover the hit, no further stress is applied.
func TestApplyConsequence_MildAbsorbsAll(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: "Twisted Ankle",
	}
	sm, player, attacker := setupConflictSM(t, mockLLM)

	// 2-shift hit → mild (value=2) absorbs all.
	events := sm.applyConsequence(context.Background(), character.MildConsequence, 2, attacker, testAttackCtx())

	require.NotEmpty(t, events)
	pce, ok := events[0].(PlayerConsequenceEvent)
	require.True(t, ok, "expected PlayerConsequenceEvent, got %T", events[0])
	assert.Equal(t, "mild", pce.Severity)
	assert.Equal(t, "Twisted Ankle", pce.Aspect)
	assert.Equal(t, 2, pce.Absorbed)
	assert.Nil(t, pce.StressAbsorbed, "no remaining shifts to absorb")

	// Verify consequence was added to the player.
	require.Len(t, player.Consequences, 1)
	assert.Equal(t, character.MildConsequence, player.Consequences[0].Type)
}

// SRD: A moderate consequence absorbs 4 shifts. If hit was 5 shifts,
// the remaining 1 shift must go to the stress track.
func TestApplyConsequence_ModerateWithRemainingStress(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: "Cracked Ribs",
	}
	sm, player, attacker := setupConflictSM(t, mockLLM)

	// 5-shift hit → moderate (value=4) absorbs 4, remaining 1 goes to stress.
	events := sm.applyConsequence(context.Background(), character.ModerateConsequence, 5, attacker, testAttackCtx())

	require.NotEmpty(t, events)
	pce, ok := events[0].(PlayerConsequenceEvent)
	require.True(t, ok, "expected PlayerConsequenceEvent, got %T", events[0])
	assert.Equal(t, "moderate", pce.Severity)
	assert.Equal(t, 4, pce.Absorbed)
	require.NotNil(t, pce.StressAbsorbed, "remaining 1 shift should be absorbed by stress")
	assert.Equal(t, 1, pce.StressAbsorbed.Shifts)
	assert.Equal(t, "physical", pce.StressAbsorbed.TrackType)

	// Verify stress box 1 is checked.
	track := player.StressTracks["physical"]
	assert.True(t, track.Boxes[0], "stress box 1 should be checked for remaining shift")
}

// When LLM fails to generate the consequence aspect, the code falls back to
// a default name like "Mild Wound".
func TestApplyConsequence_LLMFallbackNaming(t *testing.T) {
	// No LLM client → generateConsequenceAspect returns error → fallback.
	sm, _, attacker := setupConflictSM(t, nil)

	events := sm.applyConsequence(context.Background(), character.MildConsequence, 2, attacker, testAttackCtx())

	require.NotEmpty(t, events)
	pce, ok := events[0].(PlayerConsequenceEvent)
	require.True(t, ok)
	assert.Equal(t, "Mild Wound", pce.Aspect, "Should fall back to 'Mild Wound' when LLM unavailable")
}

// SRD: If the consequence doesn't absorb all damage and stress track can't
// absorb the remainder, it should overflow again (recursive). This tests
// the recursive path.
func TestApplyConsequence_RecursiveOverflow(t *testing.T) {
	sm, player, attacker := setupConflictSM(t, nil)

	// Fill both physical stress boxes so remaining can't be absorbed.
	player.TakeStress(character.PhysicalStress, 1)
	player.TakeStress(character.PhysicalStress, 2)

	// 4-shift hit with mild consequence (absorbs 2), remaining 2 can't go to
	// stress (both boxes full) → recursive overflow → pendingMidFlow.
	events := sm.applyConsequence(context.Background(), character.MildConsequence, 4, attacker, testAttackCtx())

	var hasOverflow bool
	for _, e := range events {
		if _, ok := e.(StressOverflowEvent); ok {
			hasOverflow = true
			break
		}
	}
	assert.True(t, hasOverflow, "should emit StressOverflowEvent for remaining shifts")
	// pendingMidFlow should be set (moderate and severe still available).
	require.NotNil(t, sm.pendingMidFlow, "recursive overflow should set pendingMidFlow")
}

// --- handleTakenOut ---

// SRD (Being Taken Out p.166): When taken out the opponent decides your fate.
// Outcome "game_over" should set shouldExit=true, sceneEndReason=player_taken_out.
func TestHandleTakenOut_GameOver(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: `{"narrative":"The bandit delivers the final blow.","outcome":"game_over","new_scene_hint":""}`,
	}
	sm, _, attacker := setupConflictSM(t, mockLLM)

	events := sm.handleTakenOut(context.Background(), attacker, testAttackCtx())

	require.NotEmpty(t, events)
	ptoEvt, ok := events[0].(PlayerTakenOutEvent)
	require.True(t, ok, "expected PlayerTakenOutEvent, got %T", events[0])
	assert.Equal(t, attacker.Name, ptoEvt.AttackerName)
	assert.Equal(t, "game_over", ptoEvt.Outcome)
	assert.Equal(t, "The bandit delivers the final blow.", ptoEvt.Narrative)

	// Side effects.
	assert.True(t, sm.shouldExit, "game_over should set shouldExit")
	assert.Equal(t, SceneEndPlayerTakenOut, sm.sceneEndReason)
	assert.Empty(t, sm.playerTakenOutHint)
}

// SRD: Outcome "transition" means the scene changes (captured, driven out).
func TestHandleTakenOut_Transition(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: `{"narrative":"You collapse and are dragged away.","outcome":"transition","new_scene_hint":"You awaken in a dark cell."}`,
	}
	sm, _, attacker := setupConflictSM(t, mockLLM)
	sm.exitOnSceneTransition = true

	events := sm.handleTakenOut(context.Background(), attacker, testAttackCtx())

	require.NotEmpty(t, events)
	ptoEvt, ok := events[0].(PlayerTakenOutEvent)
	require.True(t, ok)
	assert.Equal(t, "transition", ptoEvt.Outcome)
	assert.Equal(t, "You awaken in a dark cell.", ptoEvt.NewSceneHint)

	assert.True(t, sm.shouldExit, "exitOnSceneTransition=true should set shouldExit")
	assert.Equal(t, SceneEndPlayerTakenOut, sm.sceneEndReason)
	assert.Equal(t, "You awaken in a dark cell.", sm.playerTakenOutHint)
}

// Transition with exitOnSceneTransition=false should NOT set shouldExit.
func TestHandleTakenOut_Transition_NoExitFlag(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: `{"narrative":"You fall.","outcome":"transition","new_scene_hint":"Later..."}`,
	}
	sm, _, attacker := setupConflictSM(t, mockLLM)
	sm.exitOnSceneTransition = false

	events := sm.handleTakenOut(context.Background(), attacker, testAttackCtx())

	require.NotEmpty(t, events)
	ptoEvt := events[0].(PlayerTakenOutEvent)
	assert.Equal(t, "transition", ptoEvt.Outcome)

	assert.False(t, sm.shouldExit, "exitOnSceneTransition=false should NOT set shouldExit")
	assert.Equal(t, SceneEndPlayerTakenOut, sm.sceneEndReason)
}

// Outcome "continue" means the scene keeps going (stunned, knocked down).
func TestHandleTakenOut_Continue(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: `{"narrative":"You stumble and fall, dazed.","outcome":"continue","new_scene_hint":""}`,
	}
	sm, _, attacker := setupConflictSM(t, mockLLM)

	events := sm.handleTakenOut(context.Background(), attacker, testAttackCtx())

	require.NotEmpty(t, events)
	ptoEvt := events[0].(PlayerTakenOutEvent)
	assert.Equal(t, "continue", ptoEvt.Outcome)

	assert.False(t, sm.shouldExit, "continue should NOT set shouldExit")
	assert.Empty(t, string(sm.sceneEndReason), "continue should NOT set sceneEndReason")
}

// When LLM errors, handleTakenOut should fall back to a default narrative
// and "transition" outcome.
func TestHandleTakenOut_LLMError_Fallback(t *testing.T) {
	sm, _, attacker := setupConflictSM(t, nil) // No LLM → error path

	events := sm.handleTakenOut(context.Background(), attacker, testAttackCtx())

	require.NotEmpty(t, events)
	ptoEvt, ok := events[0].(PlayerTakenOutEvent)
	require.True(t, ok)
	assert.Equal(t, "transition", ptoEvt.Outcome, "should default to transition on LLM error")
	assert.Contains(t, ptoEvt.Narrative, attacker.Name, "fallback narrative should mention attacker")
}

// handleTakenOut should end the conflict and clear stress.
func TestHandleTakenOut_EndsConflictAndClearsStress(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: `{"narrative":"Defeated.","outcome":"game_over","new_scene_hint":""}`,
	}
	sm, player, attacker := setupConflictSM(t, mockLLM)

	// Give player some stress first.
	player.TakeStress(character.PhysicalStress, 1)
	assert.True(t, player.StressTracks["physical"].Boxes[0])

	sm.handleTakenOut(context.Background(), attacker, testAttackCtx())

	// Conflict should be ended.
	assert.False(t, sm.currentScene.IsConflict, "conflict should be ended after taken out")

	// Stress should be cleared (clearConflictStress is called).
	for _, box := range player.StressTracks["physical"].Boxes {
		assert.False(t, box, "physical stress should be cleared after taken out")
	}
}

// --- MidFlow continuation (consequence choice from overflow) ---

// Verifies the pendingMidFlow continuation correctly applies a consequence
// when the player selects one from the overflow prompt.
func TestStressOverflow_MidFlowContinuation_SelectConsequence(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: "Winded",
	}
	sm, player, attacker := setupConflictSM(t, mockLLM)

	// Trigger overflow.
	sm.handleStressOverflow(context.Background(), 3, character.PhysicalStress, attacker, testAttackCtx())
	require.NotNil(t, sm.pendingMidFlow)

	// Player picks choice 0 (mild consequence).
	contEvents := sm.pendingMidFlow.continuation(context.Background(), MidFlowResponse{ChoiceIndex: 0})

	var hasConseq bool
	for _, e := range contEvents {
		if pce, ok := e.(PlayerConsequenceEvent); ok {
			assert.Equal(t, "mild", pce.Severity)
			hasConseq = true
		}
	}
	assert.True(t, hasConseq, "selecting consequence should emit PlayerConsequenceEvent")

	// Consequence should exist on the player.
	assert.Len(t, player.Consequences, 1)
}

// When player selects "Be Taken Out" from overflow prompt.
func TestStressOverflow_MidFlowContinuation_SelectTakenOut(t *testing.T) {
	mockLLM := &capturingMockLLMClient{
		response: `{"narrative":"You choose to yield.","outcome":"game_over","new_scene_hint":""}`,
	}
	sm, _, _ := setupConflictSM(t, mockLLM)

	sm.handleStressOverflow(context.Background(), 3, character.PhysicalStress,
		sm.engine.GetCharacter("npc-1"), testAttackCtx())
	require.NotNil(t, sm.pendingMidFlow)

	// "Be Taken Out" is the last option (index = len(consequences)).
	takenOutIdx := len(sm.pendingMidFlow.event.Options) - 1
	contEvents := sm.pendingMidFlow.continuation(context.Background(), MidFlowResponse{ChoiceIndex: takenOutIdx})

	var hasTakenOut bool
	for _, e := range contEvents {
		if _, ok := e.(PlayerTakenOutEvent); ok {
			hasTakenOut = true
		}
	}
	assert.True(t, hasTakenOut, "selecting Be Taken Out should emit PlayerTakenOutEvent")
}
