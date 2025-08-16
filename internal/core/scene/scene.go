package scene

import (
	"time"
)

// Scene represents the current game scene
type Scene struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	
	// Scene Elements
	SituationAspects []SituationAspect `json:"situation_aspects"`
	Characters       []string          `json:"character_ids"`
	ActiveCharacter  string            `json:"active_character_id,omitempty"`
	
	// Scene State
	IsConflict       bool              `json:"is_conflict"`
	ConflictState    *ConflictState    `json:"conflict_state,omitempty"`
	
	// Metadata
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// SituationAspect represents environmental or temporary aspects
type SituationAspect struct {
	ID           string    `json:"id"`
	Aspect       string    `json:"aspect"`
	FreeInvokes  int       `json:"free_invokes"`
	Duration     string    `json:"duration"` // "scene", "scenario", "permanent"
	CreatedBy    string    `json:"created_by"` // character ID
	CreatedAt    time.Time `json:"created_at"`
}

// ConflictState manages conflict mechanics
type ConflictState struct {
	Participants    []ConflictParticipant `json:"participants"`
	InitiativeOrder []string              `json:"initiative_order"`
	CurrentTurn     int                   `json:"current_turn"`
	Round           int                   `json:"round"`
	Zones           []Zone                `json:"zones,omitempty"`
}

// ConflictParticipant represents a character in conflict
type ConflictParticipant struct {
	CharacterID string `json:"character_id"`
	Initiative  int    `json:"initiative"`
	Active      bool   `json:"active"`
}

// Zone represents spatial positioning in conflicts
type Zone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Characters  []string `json:"character_ids"`
	Aspects     []string `json:"aspect_ids"`
}

// NewScene creates a new scene
func NewScene(id, name, description string) *Scene {
	return &Scene{
		ID:               id,
		Name:             name,
		Description:      description,
		SituationAspects: make([]SituationAspect, 0),
		Characters:       make([]string, 0),
		IsConflict:       false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// AddCharacter adds a character to the scene
func (s *Scene) AddCharacter(characterID string) {
	// Check if character is already in scene
	for _, id := range s.Characters {
		if id == characterID {
			return
		}
	}
	
	s.Characters = append(s.Characters, characterID)
	s.UpdatedAt = time.Now()
}

// RemoveCharacter removes a character from the scene
func (s *Scene) RemoveCharacter(characterID string) {
	for i, id := range s.Characters {
		if id == characterID {
			s.Characters = append(s.Characters[:i], s.Characters[i+1:]...)
			s.UpdatedAt = time.Now()
			break
		}
	}
	
	// If this was the active character, clear it
	if s.ActiveCharacter == characterID {
		s.ActiveCharacter = ""
	}
}

// AddSituationAspect adds a situation aspect to the scene
func (s *Scene) AddSituationAspect(aspect SituationAspect) {
	s.SituationAspects = append(s.SituationAspects, aspect)
	s.UpdatedAt = time.Now()
}

// RemoveSituationAspect removes a situation aspect by ID
func (s *Scene) RemoveSituationAspect(aspectID string) {
	for i, aspect := range s.SituationAspects {
		if aspect.ID == aspectID {
			s.SituationAspects = append(s.SituationAspects[:i], s.SituationAspects[i+1:]...)
			s.UpdatedAt = time.Now()
			break
		}
	}
}

// GetSituationAspect finds a situation aspect by ID
func (s *Scene) GetSituationAspect(aspectID string) *SituationAspect {
	for i, aspect := range s.SituationAspects {
		if aspect.ID == aspectID {
			return &s.SituationAspects[i]
		}
	}
	return nil
}

// StartConflict begins a conflict in this scene
func (s *Scene) StartConflict(participants []ConflictParticipant) {
	s.IsConflict = true
	s.ConflictState = &ConflictState{
		Participants:    participants,
		InitiativeOrder: make([]string, 0),
		CurrentTurn:     0,
		Round:           1,
		Zones:           make([]Zone, 0),
	}
	
	// TODO: Sort participants by initiative
	for _, participant := range participants {
		s.ConflictState.InitiativeOrder = append(s.ConflictState.InitiativeOrder, participant.CharacterID)
	}
	
	s.UpdatedAt = time.Now()
}

// EndConflict ends the conflict in this scene
func (s *Scene) EndConflict() {
	s.IsConflict = false
	s.ConflictState = nil
	s.UpdatedAt = time.Now()
}

// GetCurrentActor returns the character ID of the current actor in conflict
func (s *Scene) GetCurrentActor() string {
	if !s.IsConflict || s.ConflictState == nil || len(s.ConflictState.InitiativeOrder) == 0 {
		return ""
	}
	
	return s.ConflictState.InitiativeOrder[s.ConflictState.CurrentTurn]
}

// NextTurn advances to the next turn in conflict
func (s *Scene) NextTurn() {
	if !s.IsConflict || s.ConflictState == nil {
		return
	}
	
	s.ConflictState.CurrentTurn++
	if s.ConflictState.CurrentTurn >= len(s.ConflictState.InitiativeOrder) {
		s.ConflictState.CurrentTurn = 0
		s.ConflictState.Round++
	}
	
	s.UpdatedAt = time.Now()
}

// NewSituationAspect creates a new situation aspect
func NewSituationAspect(id, aspect, createdBy string, freeInvokes int) SituationAspect {
	return SituationAspect{
		ID:          id,
		Aspect:      aspect,
		FreeInvokes: freeInvokes,
		Duration:    "scene", // Default duration
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}
}

// UseFreeInvoke uses one free invoke from the aspect
func (sa *SituationAspect) UseFreeInvoke() bool {
	if sa.FreeInvokes > 0 {
		sa.FreeInvokes--
		return true
	}
	return false
}
