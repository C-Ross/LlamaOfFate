package web

import (
	"encoding/json"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalEvent_NarrativeEvent(t *testing.T) {
	event := uicontract.NarrativeEvent{
		Text:      "The saloon doors creak open.",
		SceneName: "Dusty Saloon",
		Purpose:   "Establish the setting",
	}

	data, err := MarshalEvent(event)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "narrative", msg.Event)

	var parsed uicontract.NarrativeEvent
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	assert.Equal(t, "The saloon doors creak open.", parsed.Text)
	assert.Equal(t, "Dusty Saloon", parsed.SceneName)
	assert.Equal(t, "Establish the setting", parsed.Purpose)
}

func TestMarshalEvent_DialogEvent(t *testing.T) {
	event := uicontract.DialogEvent{
		PlayerInput: "I look around the room.",
		GMResponse:  "You see a dusty bar and three suspicious patrons.",
		IsRecap:     true,
		RecapType:   "dialog",
	}

	data, err := MarshalEvent(event)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "dialog", msg.Event)

	var parsed uicontract.DialogEvent
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	assert.Equal(t, "I look around the room.", parsed.PlayerInput)
	assert.Equal(t, "You see a dusty bar and three suspicious patrons.", parsed.GMResponse)
	assert.True(t, parsed.IsRecap)
	assert.Equal(t, "dialog", parsed.RecapType)
}

func TestMarshalEvent_ActionResultEvent(t *testing.T) {
	event := uicontract.ActionResultEvent{
		Skill:      "Fight",
		SkillRank:  "Good",
		SkillBonus: 3,
		Bonuses:    2,
		Result:     "[+][-][ ][+] (Total: Epic (+7) vs Difficulty Fair (+2))",
		Outcome:    "Success with Style",
		DiceFaces:  []int{1, -1, 0, 1},
		Total:      7,
		TotalRank:  "Epic",
		Difficulty: 2,
		DiffRank:   "Fair",
	}

	data, err := MarshalEvent(event)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "action_result", msg.Event)

	var parsed uicontract.ActionResultEvent
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	assert.Equal(t, "Fight", parsed.Skill)
	assert.Equal(t, "Good", parsed.SkillRank)
	assert.Equal(t, 3, parsed.SkillBonus)
	assert.Equal(t, 2, parsed.Bonuses)
	assert.Equal(t, "Success with Style", parsed.Outcome)
	assert.Equal(t, []int{1, -1, 0, 1}, parsed.DiceFaces)
	assert.Equal(t, 7, parsed.Total)
	assert.Equal(t, "Epic", parsed.TotalRank)
	assert.Equal(t, 2, parsed.Difficulty)
	assert.Equal(t, "Fair", parsed.DiffRank)
}

func TestMarshalEvent_ConflictStartEvent(t *testing.T) {
	event := uicontract.ConflictStartEvent{
		ConflictType:  "physical",
		InitiatorName: "Bandit",
		Participants: []uicontract.ConflictParticipantInfo{
			{CharacterID: "player-1", CharacterName: "Jesse", Initiative: 4, IsPlayer: true},
			{CharacterID: "npc-1", CharacterName: "Bandit", Initiative: 2, IsPlayer: false},
		},
	}

	data, err := MarshalEvent(event)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "conflict_start", msg.Event)

	var parsed uicontract.ConflictStartEvent
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	assert.Equal(t, "physical", parsed.ConflictType)
	require.Len(t, parsed.Participants, 2)
	assert.Equal(t, "Jesse", parsed.Participants[0].CharacterName)
	assert.True(t, parsed.Participants[0].IsPlayer)
}

func TestMarshalEvent_DefenseRollEvent(t *testing.T) {
	event := uicontract.DefenseRollEvent{
		DefenderName: "Bandit",
		Skill:        "Athletics",
		Result:       "Good (+3)",
		DiceFaces:    []int{1, 0, -1, 1},
	}

	data, err := MarshalEvent(event)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "defense_roll", msg.Event)

	var parsed uicontract.DefenseRollEvent
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	assert.Equal(t, "Bandit", parsed.DefenderName)
	assert.Equal(t, "Athletics", parsed.Skill)
	assert.Equal(t, "Good (+3)", parsed.Result)
	assert.Equal(t, []int{1, 0, -1, 1}, parsed.DiceFaces)
}

func TestMarshalEvent_InvokePromptEvent(t *testing.T) {
	event := uicontract.InvokePromptEvent{
		Available: []uicontract.InvokableAspect{
			{Name: "Quick Draw", Source: "character", FreeInvokes: 0},
			{Name: "Dark Alley", Source: "situation", FreeInvokes: 1},
		},
		FatePoints:    3,
		CurrentResult: "Fair (+2)",
		ShiftsNeeded:  2,
	}

	data, err := MarshalEvent(event)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "invoke_prompt", msg.Event)

	var parsed uicontract.InvokePromptEvent
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	require.Len(t, parsed.Available, 2)
	assert.Equal(t, "Quick Draw", parsed.Available[0].Name)
	assert.Equal(t, 3, parsed.FatePoints)
}

func TestMarshalEvent_DamageResolutionEvent(t *testing.T) {
	event := uicontract.DamageResolutionEvent{
		TargetName: "Bandit",
		Absorbed: &uicontract.StressAbsorptionDetail{
			TrackType:  "physical",
			Shifts:     2,
			TrackState: "[X][X][ ]",
		},
		Consequence: &uicontract.ConsequenceDetail{
			Severity: "mild",
			Aspect:   "Bruised Ribs",
			Absorbed: 2,
		},
		TakenOut:   false,
		VictoryEnd: false,
	}

	data, err := MarshalEvent(event)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "damage_resolution", msg.Event)

	var parsed uicontract.DamageResolutionEvent
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	assert.Equal(t, "Bandit", parsed.TargetName)
	require.NotNil(t, parsed.Absorbed)
	assert.Equal(t, "physical", parsed.Absorbed.TrackType)
	require.NotNil(t, parsed.Consequence)
	assert.Equal(t, "Bruised Ribs", parsed.Consequence.Aspect)
}

func TestMarshalEvent_NPCAttackEvent(t *testing.T) {
	event := uicontract.NPCAttackEvent{
		AttackerName:   "Bandit Leader",
		TargetName:     "Jesse",
		AttackSkill:    "Fight",
		AttackResult:   "Great (+4)",
		DefenseSkill:   "Athletics",
		DefenseResult:  "Fair (+2)",
		FullDefense:    false,
		InitialOutcome: "Success",
		FinalOutcome:   "Success",
		Narrative:      "The bandit swings his blade.",
	}

	data, err := MarshalEvent(event)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "npc_attack", msg.Event)
}

func TestMarshalEvent_AllEventTypes(t *testing.T) {
	// Verify every event type maps to a known wire name (not "unknown").
	events := []uicontract.GameEvent{
		uicontract.NarrativeEvent{},
		uicontract.DialogEvent{},
		uicontract.SystemMessageEvent{},
		uicontract.ActionAttemptEvent{},
		uicontract.ActionResultEvent{},
		uicontract.SceneTransitionEvent{},
		uicontract.GameOverEvent{},
		uicontract.ConflictStartEvent{},
		uicontract.ConflictEscalationEvent{},
		uicontract.TurnAnnouncementEvent{},
		uicontract.ConflictEndEvent{},
		uicontract.InvokePromptEvent{},
		uicontract.InputRequestEvent{},
		uicontract.DefenseRollEvent{},
		uicontract.DamageResolutionEvent{},
		uicontract.PlayerAttackResultEvent{},
		uicontract.AspectCreatedEvent{},
		uicontract.NPCAttackEvent{},
		uicontract.PlayerStressEvent{},
		uicontract.PlayerDefendedEvent{},
		uicontract.PlayerConsequenceEvent{},
		uicontract.PlayerTakenOutEvent{},
		uicontract.ConcessionEvent{},
		uicontract.OutcomeChangedEvent{},
		uicontract.InvokeEvent{},
		uicontract.NPCActionResultEvent{},
		uicontract.RecoveryEvent{},
		uicontract.StressOverflowEvent{},
		uicontract.MilestoneEvent{},
		uicontract.GameResumedEvent{},
	}

	seen := make(map[string]bool)
	for _, event := range events {
		name := eventTypeName(event)
		assert.NotEqual(t, "unknown", name, "unregistered event type: %T", event)
		assert.False(t, seen[name], "duplicate wire name: %s", name)
		seen[name] = true

		data, err := MarshalEvent(event)
		require.NoError(t, err, "failed to marshal %T", event)
		assert.NotEmpty(t, data)
	}
	assert.Len(t, seen, 30, "expected 30 unique event wire names")
}

func TestMarshalResultMeta(t *testing.T) {
	meta := ResultMeta{
		AwaitingInvoke:  true,
		AwaitingMidFlow: false,
		GameOver:        false,
		SceneEnded:      false,
	}

	data, err := MarshalResultMeta(meta)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "result_meta", msg.Event)

	var parsed ResultMeta
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	assert.True(t, parsed.AwaitingInvoke)
	assert.False(t, parsed.AwaitingMidFlow)
}

func TestParseClientMessage_Input(t *testing.T) {
	raw := `{"type": "input", "text": "I search the room"}`

	msg, err := ParseClientMessage([]byte(raw))
	require.NoError(t, err)
	assert.Equal(t, ClientInput, msg.Type)
	assert.Equal(t, "I search the room", msg.Text)
}

func TestParseClientMessage_InvokeResponse(t *testing.T) {
	raw := `{"type": "invoke_response", "aspectIndex": 1, "isReroll": true}`

	msg, err := ParseClientMessage([]byte(raw))
	require.NoError(t, err)
	assert.Equal(t, ClientInvoke, msg.Type)
	assert.Equal(t, 1, msg.AspectIndex)
	assert.True(t, msg.IsReroll)
}

func TestParseClientMessage_InvokeSkip(t *testing.T) {
	raw := `{"type": "invoke_response", "aspectIndex": -1}`

	msg, err := ParseClientMessage([]byte(raw))
	require.NoError(t, err)
	assert.Equal(t, ClientInvoke, msg.Type)
	assert.Equal(t, -1, msg.AspectIndex)
}

func TestParseClientMessage_MidFlowResponse(t *testing.T) {
	raw := `{"type": "mid_flow_response", "choiceIndex": 2, "freeText": "I surrender"}`

	msg, err := ParseClientMessage([]byte(raw))
	require.NoError(t, err)
	assert.Equal(t, ClientMidFlow, msg.Type)
	assert.Equal(t, 2, msg.ChoiceIndex)
	assert.Equal(t, "I surrender", msg.FreeText)
}

func TestParseClientMessage_InvalidJSON(t *testing.T) {
	_, err := ParseClientMessage([]byte(`not json`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse client message")
}

func TestParseClientMessage_MissingType(t *testing.T) {
	_, err := ParseClientMessage([]byte(`{"text": "hello"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing type")
}

func TestParseClientMessage_UnknownType(t *testing.T) {
	_, err := ParseClientMessage([]byte(`{"type": "bogus"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

func TestParseClientMessage_SetupPreset(t *testing.T) {
	raw := `{"type": "setup", "presetId": "heist"}`

	msg, err := ParseClientMessage([]byte(raw))
	require.NoError(t, err)
	assert.Equal(t, ClientSetup, msg.Type)
	assert.Equal(t, "heist", msg.PresetID)
	assert.Nil(t, msg.Custom)
}

func TestParseClientMessage_SetupCustom(t *testing.T) {
	raw := `{
		"type": "setup",
		"custom": {
			"name": "Ada",
			"highConcept": "Rogue AI Whisperer",
			"trouble": "Trusts Machines More Than People",
			"genre": "Cyberpunk"
		}
	}`

	msg, err := ParseClientMessage([]byte(raw))
	require.NoError(t, err)
	assert.Equal(t, ClientSetup, msg.Type)
	assert.Empty(t, msg.PresetID)
	require.NotNil(t, msg.Custom)
	assert.Equal(t, "Ada", msg.Custom.Name)
	assert.Equal(t, "Rogue AI Whisperer", msg.Custom.HighConcept)
	assert.Equal(t, "Trusts Machines More Than People", msg.Custom.Trouble)
	assert.Equal(t, "Cyberpunk", msg.Custom.Genre)
}

func TestMarshalSetupRequest(t *testing.T) {
	req := SetupRequest{
		Presets: []ScenarioPreset{
			{ID: "saloon", Title: "Trouble in Redemption Gulch", Genre: "Western", Description: "A frontier town."},
		},
		AllowCustom: true,
	}

	data, err := MarshalSetupRequest(req)
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "setup_request", msg.Event)

	var parsed SetupRequest
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	require.Len(t, parsed.Presets, 1)
	assert.Equal(t, "saloon", parsed.Presets[0].ID)
	assert.True(t, parsed.AllowCustom)
}

func TestMarshalSetupGenerating(t *testing.T) {
	data, err := MarshalSetupGenerating("Generating scenario...")
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "setup_generating", msg.Event)

	var parsed SetupGenerating
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	assert.Equal(t, "Generating scenario...", parsed.Message)
}

func TestMarshalSessionInit(t *testing.T) {
	data, err := MarshalSessionInit("game-42")
	require.NoError(t, err)

	var msg ServerMessage
	require.NoError(t, json.Unmarshal(data, &msg))
	assert.Equal(t, "session_init", msg.Event)

	var parsed SessionInit
	require.NoError(t, json.Unmarshal(msg.Data, &parsed))
	assert.Equal(t, "game-42", parsed.GameID)
}
