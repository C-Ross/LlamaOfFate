package engine

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// conflictMarkerRegex matches [CONFLICT:type:character_id] markers for escalation
var conflictMarkerRegex = regexp.MustCompile(`\[CONFLICT:(physical|mental):([^\]]+)\]`)

// conflictEndMarkerRegex matches [CONFLICT:end:reason] markers for de-escalation
var conflictEndMarkerRegex = regexp.MustCompile(`\[CONFLICT:end:(surrender|agreement|retreat|resolved)\]`)

// ConflictTrigger represents a detected conflict initiation
type ConflictTrigger struct {
	Type        scene.ConflictType
	InitiatorID string
}

// ConflictResolution represents a detected conflict de-escalation
type ConflictResolution struct {
	Reason string
}

// parseConflictMarker extracts a conflict trigger from LLM response and returns cleaned text
func (sm *SceneManager) parseConflictMarker(response string) (*ConflictTrigger, string) {
	matches := conflictMarkerRegex.FindStringSubmatch(response)
	if matches == nil {
		return nil, response
	}

	conflictType := scene.PhysicalConflict
	if matches[1] == "mental" {
		conflictType = scene.MentalConflict
	}

	trigger := &ConflictTrigger{
		Type:        conflictType,
		InitiatorID: strings.TrimSpace(matches[2]),
	}

	// Remove the marker from the response and clean up any double spaces
	cleanedResponse := conflictMarkerRegex.ReplaceAllString(response, "")
	// Replace multiple spaces with single space
	cleanedResponse = strings.Join(strings.Fields(cleanedResponse), " ")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	return trigger, cleanedResponse
}

// parseConflictEndMarker extracts a conflict resolution from LLM response and returns cleaned text
func (sm *SceneManager) parseConflictEndMarker(response string) (*ConflictResolution, string) {
	matches := conflictEndMarkerRegex.FindStringSubmatch(response)
	if matches == nil {
		return nil, response
	}

	resolution := &ConflictResolution{
		Reason: matches[1],
	}

	// Remove the marker from the response and clean up
	cleanedResponse := conflictEndMarkerRegex.ReplaceAllString(response, "")
	cleanedResponse = strings.Join(strings.Fields(cleanedResponse), " ")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	return resolution, cleanedResponse
}

// initiateConflict starts a conflict with all characters in the scene
func (sm *SceneManager) initiateConflict(conflictType scene.ConflictType, initiatorID string) error {
	if sm.currentScene.IsConflict {
		return fmt.Errorf("already in a conflict")
	}

	// Build participants from all characters in the scene
	participants := make([]scene.ConflictParticipant, 0)

	for _, charID := range sm.currentScene.Characters {
		char := sm.engine.GetCharacter(charID)
		if char == nil {
			continue
		}

		// Calculate initiative based on conflict type
		initiative := sm.calculateInitiative(char, conflictType)

		participants = append(participants, scene.ConflictParticipant{
			CharacterID: charID,
			Initiative:  initiative,
			Status:      scene.StatusActive,
		})
	}

	if len(participants) < 2 {
		return fmt.Errorf("conflict requires at least 2 participants")
	}

	// Sort by initiative (descending)
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].Initiative > participants[j].Initiative
	})

	sm.currentScene.StartConflictWithInitiator(conflictType, participants, initiatorID)

	// Re-sort initiative order after StartConflict
	sm.sortInitiativeOrder()

	// Display conflict start
	initiatorName := initiatorID
	if char := sm.engine.GetCharacter(initiatorID); char != nil {
		initiatorName = char.Name
	}

	sm.ui.DisplayConflictStart(string(conflictType), initiatorName, sm.getParticipantInfo())

	slog.Info("Conflict initiated",
		"component", componentSceneManager,
		"type", conflictType,
		"initiator", initiatorID,
		"participants", len(participants))

	return nil
}

// resolveConflictPeacefully ends a conflict through non-violent means
func (sm *SceneManager) resolveConflictPeacefully(reason string) {
	if !sm.currentScene.IsConflict {
		return
	}

	// Format reason for display
	reasonMessage := ""
	switch reason {
	case "surrender":
		reasonMessage = "Your opponent surrenders!"
	case "agreement":
		reasonMessage = "You've reached an agreement!"
	case "retreat":
		reasonMessage = "Your opponent retreats!"
	case "resolved":
		reasonMessage = "The conflict has been resolved!"
	default:
		reasonMessage = "The conflict ends!"
	}

	sm.ui.DisplaySystemMessage("\n=== Conflict Resolved ===")
	sm.ui.DisplaySystemMessage(reasonMessage)

	sm.currentScene.EndConflict()

	slog.Info("Conflict resolved peacefully",
		"component", componentSceneManager,
		"reason", reason)
}

// calculateInitiative returns the initiative value for a character based on conflict type
func (sm *SceneManager) calculateInitiative(char *character.Character, conflictType scene.ConflictType) int {
	return core.CalculateInitiative(char, conflictType)
}

// sortInitiativeOrder sorts the initiative order by participant initiative values
func (sm *SceneManager) sortInitiativeOrder() {
	if sm.currentScene.ConflictState == nil {
		return
	}

	// Sort participants by initiative
	sort.Slice(sm.currentScene.ConflictState.Participants, func(i, j int) bool {
		return sm.currentScene.ConflictState.Participants[i].Initiative >
			sm.currentScene.ConflictState.Participants[j].Initiative
	})

	// Rebuild initiative order from sorted participants
	sm.currentScene.ConflictState.InitiativeOrder = make([]string, 0)
	for _, p := range sm.currentScene.ConflictState.Participants {
		if p.Status == scene.StatusActive {
			sm.currentScene.ConflictState.InitiativeOrder = append(
				sm.currentScene.ConflictState.InitiativeOrder, p.CharacterID)
		}
	}
}

// recalculateInitiative recalculates initiative for all participants based on conflict type
func (sm *SceneManager) recalculateInitiative(conflictType scene.ConflictType) {
	if sm.currentScene.ConflictState == nil {
		return
	}

	for i := range sm.currentScene.ConflictState.Participants {
		p := &sm.currentScene.ConflictState.Participants[i]
		char := sm.engine.GetCharacter(p.CharacterID)
		if char != nil {
			p.Initiative = sm.calculateInitiative(char, conflictType)
		}
	}

	sm.sortInitiativeOrder()
}

// handleConflictEscalation changes the conflict type and recalculates initiative
func (sm *SceneManager) handleConflictEscalation(newType scene.ConflictType) {
	if !sm.currentScene.IsConflict {
		return
	}

	currentType := sm.currentScene.ConflictState.Type
	if currentType == newType {
		return
	}

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"The conflict escalates from %s to %s!", currentType, newType))

	sm.currentScene.EscalateConflict(newType)
	sm.recalculateInitiative(newType)
}

// advanceConflictTurns advances through turns and processes NPC actions until it's the player's turn
func (sm *SceneManager) advanceConflictTurns(ctx context.Context) {
	if !sm.currentScene.IsConflict || sm.currentScene.ConflictState == nil {
		return
	}

	// Advance past the player's turn
	sm.currentScene.NextTurn()

	// Process NPC turns until we get back to the player or conflict ends
	for sm.currentScene.IsConflict {
		currentActor := sm.currentScene.GetCurrentActor()
		if currentActor == "" {
			break
		}

		// If it's the player's turn, stop and let them act
		if currentActor == sm.player.ID {
			sm.ui.DisplayTurnAnnouncement(sm.player.Name, sm.currentScene.ConflictState.Round, true)
			break
		}

		// Process NPC turn
		sm.processNPCTurn(ctx, currentActor)

		// Advance to next turn
		sm.currentScene.NextTurn()
	}
}

// getParticipantInfo returns information about all conflict participants for display
func (sm *SceneManager) getParticipantInfo() []ConflictParticipantInfo {
	if sm.currentScene.ConflictState == nil {
		return nil
	}

	info := make([]ConflictParticipantInfo, 0, len(sm.currentScene.ConflictState.Participants))
	for _, p := range sm.currentScene.ConflictState.Participants {
		name := p.CharacterID
		if char := sm.engine.GetCharacter(p.CharacterID); char != nil {
			name = char.Name
		}
		info = append(info, ConflictParticipantInfo{
			CharacterID:   p.CharacterID,
			CharacterName: name,
			Initiative:    p.Initiative,
			IsPlayer:      p.CharacterID == sm.player.ID,
		})
	}
	return info
}
