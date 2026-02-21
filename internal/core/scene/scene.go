package scene

import (
	"sort"
	"time"
)

// Scene represents the current game scene
type Scene struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`

	// Scene Elements
	SituationAspects []SituationAspect `json:"situation_aspects" yaml:"situation_aspects"`
	Characters       []string          `json:"character_ids" yaml:"character_ids,omitempty"`
	ActiveCharacter  string            `json:"active_character_id,omitempty" yaml:"active_character_id,omitempty"`

	// Scene State
	IsConflict         bool            `json:"is_conflict" yaml:"is_conflict,omitempty"`
	ConflictState      *ConflictState  `json:"conflict_state,omitempty" yaml:"conflict_state,omitempty"`
	TakenOutCharacters map[string]bool `json:"taken_out_characters,omitempty" yaml:"taken_out_characters,omitempty"` // Characters taken out this scene

	// Metadata
	CreatedAt time.Time `json:"created_at" yaml:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at,omitempty"`
}

// SituationAspect represents environmental or temporary aspects
type SituationAspect struct {
	ID          string    `json:"id" yaml:"id"`
	Aspect      string    `json:"aspect" yaml:"aspect"`
	FreeInvokes int       `json:"free_invokes" yaml:"free_invokes,omitempty"`
	Duration    string    `json:"duration" yaml:"duration"`               // "scene", "scenario", "permanent"
	CreatedBy   string    `json:"created_by" yaml:"created_by,omitempty"` // character ID
	CreatedAt   time.Time `json:"created_at" yaml:"created_at,omitempty"`
}

// ConflictType distinguishes physical from mental conflicts
// Physical conflicts use Notice for initiative and target physical stress
// Mental conflicts use Empathy for initiative and target mental stress
type ConflictType string

const (
	PhysicalConflict ConflictType = "physical"
	MentalConflict   ConflictType = "mental"
)

// ParticipantStatus tracks a participant's state in a conflict
type ParticipantStatus string

const (
	StatusActive   ParticipantStatus = "active"    // Still fighting
	StatusConceded ParticipantStatus = "conceded"  // Voluntarily withdrew
	StatusTakenOut ParticipantStatus = "taken_out" // Removed by opponent
)

// ConflictState manages conflict mechanics
type ConflictState struct {
	Type                ConflictType          `json:"type"`                           // Current conflict type (physical or mental)
	OriginalType        ConflictType          `json:"original_type,omitempty"`        // Original type if escalated
	InitiatingCharacter string                `json:"initiating_character,omitempty"` // Who started the conflict
	Participants        []ConflictParticipant `json:"participants"`
	InitiativeOrder     []string              `json:"initiative_order"`
	CurrentTurn         int                   `json:"current_turn"`
	Round               int                   `json:"round"`
}

// ConflictParticipant represents a character in conflict
type ConflictParticipant struct {
	CharacterID string            `json:"character_id"`
	Initiative  int               `json:"initiative"`
	Status      ParticipantStatus `json:"status"`
	FullDefense bool              `json:"full_defense"` // Forgoes action for +2 to all defense rolls
	HasActed    bool              `json:"has_acted"`    // Whether they've taken their action this exchange
}

// SortByInitiative sorts participants by initiative (descending) and rebuilds
// the InitiativeOrder slice from active participants.
func (cs *ConflictState) SortByInitiative() {
	sort.Slice(cs.Participants, func(i, j int) bool {
		return cs.Participants[i].Initiative > cs.Participants[j].Initiative
	})

	cs.InitiativeOrder = make([]string, 0, len(cs.Participants))
	for _, p := range cs.Participants {
		if p.Status == StatusActive {
			cs.InitiativeOrder = append(cs.InitiativeOrder, p.CharacterID)
		}
	}
}

// NewScene creates a new scene
func NewScene(id, name, description string) *Scene {
	return &Scene{
		ID:                 id,
		Name:               name,
		Description:        description,
		SituationAspects:   make([]SituationAspect, 0),
		Characters:         make([]string, 0),
		TakenOutCharacters: make(map[string]bool),
		IsConflict:         false,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

// InitDefaults fills in runtime fields (nil slices/maps, timestamps) that are
// not present in serialized data. Call after unmarshaling from YAML/JSON.
func (s *Scene) InitDefaults() {
	if s.SituationAspects == nil {
		s.SituationAspects = make([]SituationAspect, 0)
	}
	if s.Characters == nil {
		s.Characters = make([]string, 0)
	}
	if s.TakenOutCharacters == nil {
		s.TakenOutCharacters = make(map[string]bool)
	}
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
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

// MarkCharacterTakenOut marks a character as taken out for the duration of this scene
func (s *Scene) MarkCharacterTakenOut(characterID string) {
	if s.TakenOutCharacters == nil {
		s.TakenOutCharacters = make(map[string]bool)
	}
	s.TakenOutCharacters[characterID] = true
	s.UpdatedAt = time.Now()
}

// IsCharacterTakenOut returns true if the character has been taken out this scene
func (s *Scene) IsCharacterTakenOut(characterID string) bool {
	if s.TakenOutCharacters == nil {
		return false
	}
	return s.TakenOutCharacters[characterID]
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
func (s *Scene) StartConflict(conflictType ConflictType, participants []ConflictParticipant) {
	s.StartConflictWithInitiator(conflictType, participants, "")
}

// StartConflictWithInitiator begins a conflict with a specified initiating character
func (s *Scene) StartConflictWithInitiator(conflictType ConflictType, participants []ConflictParticipant, initiator string) {
	s.IsConflict = true
	s.ConflictState = &ConflictState{
		Type:                conflictType,
		InitiatingCharacter: initiator,
		Participants:        participants,
		InitiativeOrder:     make([]string, 0),
		CurrentTurn:         0,
		Round:               1,
	}

	// Ensure all participants have active status if not set
	for i := range s.ConflictState.Participants {
		if s.ConflictState.Participants[i].Status == "" {
			s.ConflictState.Participants[i].Status = StatusActive
		}
	}

	// Sort participants by initiative and build the initiative order
	s.ConflictState.SortByInitiative()

	s.UpdatedAt = time.Now()
}

// EscalateConflict changes the conflict type (e.g., mental to physical)
func (s *Scene) EscalateConflict(newType ConflictType) bool {
	if !s.IsConflict || s.ConflictState == nil {
		return false
	}

	if s.ConflictState.Type == newType {
		return false // Already this type
	}

	// Store original type if this is the first escalation
	if s.ConflictState.OriginalType == "" {
		s.ConflictState.OriginalType = s.ConflictState.Type
	}

	s.ConflictState.Type = newType
	s.UpdatedAt = time.Now()
	return true
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
// Skips participants who are no longer active (conceded or taken out)
func (s *Scene) NextTurn() {
	if !s.IsConflict || s.ConflictState == nil {
		return
	}

	// Mark current actor as having acted
	currentActor := s.GetCurrentActor()
	for i := range s.ConflictState.Participants {
		if s.ConflictState.Participants[i].CharacterID == currentActor {
			s.ConflictState.Participants[i].HasActed = true
			break
		}
	}

	// Find next active participant
	for {
		s.ConflictState.CurrentTurn++
		if s.ConflictState.CurrentTurn >= len(s.ConflictState.InitiativeOrder) {
			s.ConflictState.CurrentTurn = 0
			s.ConflictState.Round++
			s.resetExchangeState()
		}

		// Check if current actor is still active
		actorID := s.ConflictState.InitiativeOrder[s.ConflictState.CurrentTurn]
		if s.isParticipantActive(actorID) {
			break
		}

		// Safety check to prevent infinite loop if no active participants
		if s.CountActiveParticipants() == 0 {
			break
		}
	}

	s.UpdatedAt = time.Now()
}

// resetExchangeState resets per-exchange state at the start of a new round
func (s *Scene) resetExchangeState() {
	for i := range s.ConflictState.Participants {
		s.ConflictState.Participants[i].HasActed = false
		s.ConflictState.Participants[i].FullDefense = false
	}
}

// isParticipantActive returns true if the participant is still active in the conflict
func (s *Scene) isParticipantActive(characterID string) bool {
	for _, p := range s.ConflictState.Participants {
		if p.CharacterID == characterID {
			return p.Status == StatusActive
		}
	}
	return false
}

// CountActiveParticipants returns the number of active participants
func (s *Scene) CountActiveParticipants() int {
	count := 0
	for _, p := range s.ConflictState.Participants {
		if p.Status == StatusActive {
			count++
		}
	}
	return count
}

// SetParticipantStatus updates a participant's status (conceded, taken out)
func (s *Scene) SetParticipantStatus(characterID string, status ParticipantStatus) bool {
	if !s.IsConflict || s.ConflictState == nil {
		return false
	}

	for i := range s.ConflictState.Participants {
		if s.ConflictState.Participants[i].CharacterID == characterID {
			s.ConflictState.Participants[i].Status = status
			s.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// SetFullDefense sets a participant to full defense mode for this exchange
func (s *Scene) SetFullDefense(characterID string) bool {
	if !s.IsConflict || s.ConflictState == nil {
		return false
	}

	for i := range s.ConflictState.Participants {
		if s.ConflictState.Participants[i].CharacterID == characterID {
			s.ConflictState.Participants[i].FullDefense = true
			s.ConflictState.Participants[i].HasActed = true // Full defense uses your action
			s.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// IsFullDefense returns true if the participant is in full defense mode
func (s *Scene) IsFullDefense(characterID string) bool {
	if !s.IsConflict || s.ConflictState == nil {
		return false
	}

	for _, p := range s.ConflictState.Participants {
		if p.CharacterID == characterID {
			return p.FullDefense
		}
	}
	return false
}

// GetParticipant returns a pointer to a participant by character ID
func (s *Scene) GetParticipant(characterID string) *ConflictParticipant {
	if !s.IsConflict || s.ConflictState == nil {
		return nil
	}

	for i := range s.ConflictState.Participants {
		if s.ConflictState.Participants[i].CharacterID == characterID {
			return &s.ConflictState.Participants[i]
		}
	}
	return nil
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
