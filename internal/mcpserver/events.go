// Package mcpserver provides an MCP (Model Context Protocol) server that wraps
// the game engine, exposing structured tool-based access to the game session.
package mcpserver

import (
	"encoding/json"

	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// eventEnvelope wraps a GameEvent with its type name for structured JSON output.
type eventEnvelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

// marshalEvents converts a slice of GameEvents into a JSON array of typed
// envelopes: [{"type":"narrative","data":{...}}, ...].
func marshalEvents(events []uicontract.GameEvent) (string, error) {
	envelopes := make([]eventEnvelope, 0, len(events))
	for _, ev := range events {
		envelopes = append(envelopes, eventEnvelope{
			Type: eventTypeName(ev),
			Data: ev,
		})
	}
	data, err := json.MarshalIndent(envelopes, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// eventTypeName returns the snake_case wire name for a GameEvent concrete type.
// This mirrors the mapping in internal/ui/web/messages.go.
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
	case uicontract.ErrorNotificationEvent:
		return "error_notification"
	case uicontract.BoostExpiredEvent:
		return "boost_expired"
	case uicontract.ChallengeStartEvent:
		return "challenge_start"
	case uicontract.ChallengeTaskResultEvent:
		return "challenge_task_result"
	case uicontract.ChallengeCompleteEvent:
		return "challenge_complete"
	default:
		return "unknown"
	}
}
