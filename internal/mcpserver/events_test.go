package mcpserver

import (
	"encoding/json"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventTypeName_AllTypes(t *testing.T) {
	tests := []struct {
		event    uicontract.GameEvent
		expected string
	}{
		{uicontract.NarrativeEvent{}, "narrative"},
		{uicontract.DialogEvent{}, "dialog"},
		{uicontract.SystemMessageEvent{}, "system_message"},
		{uicontract.ActionAttemptEvent{}, "action_attempt"},
		{uicontract.ActionResultEvent{}, "action_result"},
		{uicontract.SceneTransitionEvent{}, "scene_transition"},
		{uicontract.GameOverEvent{}, "game_over"},
		{uicontract.ConflictStartEvent{}, "conflict_start"},
		{uicontract.ConflictEscalationEvent{}, "conflict_escalation"},
		{uicontract.TurnAnnouncementEvent{}, "turn_announcement"},
		{uicontract.ConflictEndEvent{}, "conflict_end"},
		{uicontract.InvokePromptEvent{}, "invoke_prompt"},
		{uicontract.InputRequestEvent{}, "input_request"},
		{uicontract.DefenseRollEvent{}, "defense_roll"},
		{uicontract.DamageResolutionEvent{}, "damage_resolution"},
		{uicontract.PlayerAttackResultEvent{}, "player_attack_result"},
		{uicontract.AspectCreatedEvent{}, "aspect_created"},
		{uicontract.NPCAttackEvent{}, "npc_attack"},
		{uicontract.PlayerStressEvent{}, "player_stress"},
		{uicontract.PlayerDefendedEvent{}, "player_defended"},
		{uicontract.PlayerConsequenceEvent{}, "player_consequence"},
		{uicontract.PlayerTakenOutEvent{}, "player_taken_out"},
		{uicontract.ConcessionEvent{}, "concession"},
		{uicontract.OutcomeChangedEvent{}, "outcome_changed"},
		{uicontract.InvokeEvent{}, "invoke"},
		{uicontract.NPCActionResultEvent{}, "npc_action_result"},
		{uicontract.RecoveryEvent{}, "recovery"},
		{uicontract.StressOverflowEvent{}, "stress_overflow"},
		{uicontract.MilestoneEvent{}, "milestone"},
		{uicontract.GameResumedEvent{}, "game_resumed"},
		{uicontract.GameStateSnapshotEvent{}, "game_state_snapshot"},
		{uicontract.ErrorNotificationEvent{}, "error_notification"},
		{uicontract.BoostExpiredEvent{}, "boost_expired"},
		{uicontract.ChallengeStartEvent{}, "challenge_start"},
		{uicontract.ChallengeTaskResultEvent{}, "challenge_task_result"},
		{uicontract.ChallengeCompleteEvent{}, "challenge_complete"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, eventTypeName(tc.event))
		})
	}
}

func TestMarshalEvents_EmptySlice(t *testing.T) {
	result, err := marshalEvents(nil)
	require.NoError(t, err)
	assert.Equal(t, "[]", result)
}

func TestMarshalEvents_SingleNarrative(t *testing.T) {
	events := []uicontract.GameEvent{
		uicontract.NarrativeEvent{Text: "You enter the tavern.", SceneName: "Tavern"},
	}
	result, err := marshalEvents(events)
	require.NoError(t, err)

	var envelopes []eventEnvelope
	err = json.Unmarshal([]byte(result), &envelopes)
	require.NoError(t, err)
	require.Len(t, envelopes, 1)
	assert.Equal(t, "narrative", envelopes[0].Type)
}

func TestMarshalEvents_MultipleEvents(t *testing.T) {
	events := []uicontract.GameEvent{
		uicontract.NarrativeEvent{Text: "Scene intro"},
		uicontract.DialogEvent{PlayerInput: "Hello", GMResponse: "Greetings"},
		uicontract.ActionResultEvent{Skill: "Athletics", Outcome: "succeed"},
	}
	result, err := marshalEvents(events)
	require.NoError(t, err)

	var envelopes []eventEnvelope
	err = json.Unmarshal([]byte(result), &envelopes)
	require.NoError(t, err)
	require.Len(t, envelopes, 3)
	assert.Equal(t, "narrative", envelopes[0].Type)
	assert.Equal(t, "dialog", envelopes[1].Type)
	assert.Equal(t, "action_result", envelopes[2].Type)
}

func TestMarshalEvents_DataFieldsPreserved(t *testing.T) {
	events := []uicontract.GameEvent{
		uicontract.ActionResultEvent{
			Skill:      "Fight",
			SkillRank:  "Good",
			SkillBonus: 3,
			Outcome:    "succeed",
			DiceFaces:  []int{1, -1, 0, 1},
			Total:      4,
			TotalRank:  "Great",
			Difficulty: 2,
			DiffRank:   "Fair",
		},
	}
	result, err := marshalEvents(events)
	require.NoError(t, err)

	var envelopes []struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	err = json.Unmarshal([]byte(result), &envelopes)
	require.NoError(t, err)
	require.Len(t, envelopes, 1)

	var actionResult uicontract.ActionResultEvent
	err = json.Unmarshal(envelopes[0].Data, &actionResult)
	require.NoError(t, err)
	assert.Equal(t, "Fight", actionResult.Skill)
	assert.Equal(t, 3, actionResult.SkillBonus)
	assert.Equal(t, []int{1, -1, 0, 1}, actionResult.DiceFaces)
	assert.Equal(t, 4, actionResult.Total)
	assert.Equal(t, "succeed", actionResult.Outcome)
}
