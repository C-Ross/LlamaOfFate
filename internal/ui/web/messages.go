// Package web provides a WebSocket-based UI implementation for browser gameplay.
package web

import (
	"encoding/json"
	"fmt"

	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// ServerMessage is the JSON envelope sent from server to client.
//
//	{"event": "<type>", "data": { ... }}
type ServerMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// ResultMeta is sent after each InputResult to communicate flow-control state.
type ResultMeta struct {
	AwaitingInvoke  bool `json:"awaitingInvoke"`
	AwaitingMidFlow bool `json:"awaitingMidFlow"`
	GameOver        bool `json:"gameOver"`
	SceneEnded      bool `json:"sceneEnded"`
}

// ClientMessageType identifies the kind of message the client sends.
type ClientMessageType string

const (
	ClientInput   ClientMessageType = "input"
	ClientInvoke  ClientMessageType = "invoke_response"
	ClientMidFlow ClientMessageType = "mid_flow_response"
)

// ClientMessage is the JSON envelope received from the client.
type ClientMessage struct {
	Type ClientMessageType `json:"type"`

	// For "input" messages
	Text string `json:"text,omitempty"`

	// For "invoke_response" messages
	AspectIndex int  `json:"aspectIndex,omitempty"`
	IsReroll    bool `json:"isReroll,omitempty"`

	// For "mid_flow_response" messages
	ChoiceIndex int    `json:"choiceIndex,omitempty"`
	FreeText    string `json:"freeText,omitempty"`
}

// ParseClientMessage deserializes a JSON byte slice into a ClientMessage.
func ParseClientMessage(data []byte) (ClientMessage, error) {
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return ClientMessage{}, fmt.Errorf("parse client message: %w", err)
	}
	if msg.Type == "" {
		return ClientMessage{}, fmt.Errorf("parse client message: missing type field")
	}
	switch msg.Type {
	case ClientInput, ClientInvoke, ClientMidFlow:
		return msg, nil
	default:
		return ClientMessage{}, fmt.Errorf("parse client message: unknown type %q", msg.Type)
	}
}

// eventTypeName returns the snake_case wire name for a GameEvent concrete type.
func eventTypeName(event uicontract.GameEvent) string {
	switch event.(type) {
	case uicontract.NarrativeEvent:
		return "narrative"
	case uicontract.DialogEvent:
		return "dialog"
	case uicontract.SystemMessageEvent:
		return "system_message"
	case uicontract.ActionAttemptEvent:
		return "action_attempt"
	case uicontract.ActionResultEvent:
		return "action_result"
	case uicontract.SceneTransitionEvent:
		return "scene_transition"
	case uicontract.GameOverEvent:
		return "game_over"
	case uicontract.ConflictStartEvent:
		return "conflict_start"
	case uicontract.ConflictEscalationEvent:
		return "conflict_escalation"
	case uicontract.TurnAnnouncementEvent:
		return "turn_announcement"
	case uicontract.ConflictEndEvent:
		return "conflict_end"
	case uicontract.InvokePromptEvent:
		return "invoke_prompt"
	case uicontract.InputRequestEvent:
		return "input_request"
	case uicontract.DefenseRollEvent:
		return "defense_roll"
	case uicontract.DamageResolutionEvent:
		return "damage_resolution"
	case uicontract.PlayerAttackResultEvent:
		return "player_attack_result"
	case uicontract.AspectCreatedEvent:
		return "aspect_created"
	case uicontract.NPCAttackEvent:
		return "npc_attack"
	case uicontract.PlayerStressEvent:
		return "player_stress"
	case uicontract.PlayerDefendedEvent:
		return "player_defended"
	case uicontract.PlayerConsequenceEvent:
		return "player_consequence"
	case uicontract.PlayerTakenOutEvent:
		return "player_taken_out"
	case uicontract.ConcessionEvent:
		return "concession"
	case uicontract.OutcomeChangedEvent:
		return "outcome_changed"
	case uicontract.InvokeEvent:
		return "invoke"
	case uicontract.NPCActionResultEvent:
		return "npc_action_result"
	case uicontract.RecoveryEvent:
		return "recovery"
	case uicontract.StressOverflowEvent:
		return "stress_overflow"
	case uicontract.MilestoneEvent:
		return "milestone"
	case uicontract.GameResumedEvent:
		return "game_resumed"
	case uicontract.GameStateSnapshotEvent:
		return "game_state_snapshot"
	default:
		return "unknown"
	}
}

// MarshalEvent serializes a GameEvent to a JSON ServerMessage envelope.
func MarshalEvent(event uicontract.GameEvent) ([]byte, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event data: %w", err)
	}
	msg := ServerMessage{
		Event: eventTypeName(event),
		Data:  data,
	}
	return json.Marshal(msg)
}

// MarshalResultMeta serializes a ResultMeta as a ServerMessage with event "result_meta".
func MarshalResultMeta(meta ResultMeta) ([]byte, error) {
	data, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal result meta: %w", err)
	}
	msg := ServerMessage{
		Event: "result_meta",
		Data:  data,
	}
	return json.Marshal(msg)
}

// SessionInit is the payload for the session_init event sent immediately after
// WebSocket connection. It tells the client which game ID this session belongs
// to, so the client can store it and reconnect to the same game later.
type SessionInit struct {
	GameID string `json:"gameId"`
}

// MarshalSessionInit serializes a session_init ServerMessage.
func MarshalSessionInit(gameID string) ([]byte, error) {
	data, err := json.Marshal(SessionInit{GameID: gameID})
	if err != nil {
		return nil, fmt.Errorf("marshal session init: %w", err)
	}
	msg := ServerMessage{
		Event: "session_init",
		Data:  data,
	}
	return json.Marshal(msg)
}
