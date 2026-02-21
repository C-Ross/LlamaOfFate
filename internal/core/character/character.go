package character

import (
	"fmt"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
)

// TakenOutFate records what happened to a character after being taken out.
// Per Fate Core, the victor narrates the defeated character's fate.
// The Description is a 1-3 word summary (e.g., "killed", "unconscious", "fled").
// Permanent indicates whether the character is permanently removed from the story.
type TakenOutFate struct {
	Description string `json:"description" yaml:"description"` // 1-3 word summary: "destroyed", "unconscious", "fled"
	Permanent   bool   `json:"permanent" yaml:"permanent"`     // true = permanently removed from the story
}

// Character represents a player character or NPC
type Character struct {
	ID            string        `json:"id" yaml:"id"`
	Name          string        `json:"name" yaml:"name"`
	Description   string        `json:"description" yaml:"description,omitempty"`
	CharacterType CharacterType `json:"character_type,omitempty" yaml:"character_type,omitempty"` // PC or NPC type (0 = PC)

	// Core Character Elements
	Aspects Aspects                `json:"aspects" yaml:"aspects"`
	Skills  map[string]dice.Ladder `json:"skills" yaml:"skills"`
	Stunts  []Stunt                `json:"stunts" yaml:"stunts,omitempty"`

	// Resources
	FatePoints int `json:"fate_points" yaml:"fate_points"`
	Refresh    int `json:"refresh" yaml:"refresh"`

	// Health and Status
	StressTracks map[string]*StressTrack `json:"stress_tracks" yaml:"stress_tracks,omitempty"`
	Consequences []Consequence           `json:"consequences" yaml:"consequences,omitempty"`
	Fate         *TakenOutFate           `json:"fate,omitempty" yaml:"fate,omitempty"` // Set when character is taken out

	// Metadata
	CreatedAt time.Time `json:"created_at" yaml:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at,omitempty"`
}

// IsTakenOut returns true if this character has been taken out and assigned a fate.
func (c *Character) IsTakenOut() bool {
	return c.Fate != nil
}

// IsPermanentlyRemoved returns true if this character was taken out with a permanent fate
// (e.g., killed, destroyed) and should not appear in future scenes.
func (c *Character) IsPermanentlyRemoved() bool {
	return c.Fate != nil && c.Fate.Permanent
}

// Aspects holds the character aspects
type Aspects struct {
	HighConcept  string   `json:"high_concept" yaml:"high_concept"`
	Trouble      string   `json:"trouble" yaml:"trouble"`
	OtherAspects []string `json:"other_aspects" yaml:"other"`
}

// GetAll returns all aspects as a slice
func (a *Aspects) GetAll() []string {
	aspects := make([]string, 0, 2+len(a.OtherAspects))
	if a.HighConcept != "" {
		aspects = append(aspects, a.HighConcept)
	}
	if a.Trouble != "" {
		aspects = append(aspects, a.Trouble)
	}
	for _, aspect := range a.OtherAspects {
		if aspect != "" {
			aspects = append(aspects, aspect)
		}
	}
	return aspects
}

// IsComplete returns true if High Concept and Trouble are filled
func (a *Aspects) IsComplete() bool {
	return a.HighConcept != "" && a.Trouble != ""
}

// AddAspect adds a new aspect to the character
func (a *Aspects) AddAspect(aspect string) {
	if aspect != "" {
		a.OtherAspects = append(a.OtherAspects, aspect)
	}
}

// RemoveAspect removes an aspect by index from the other aspects
func (a *Aspects) RemoveAspect(index int) bool {
	if index < 0 || index >= len(a.OtherAspects) {
		return false
	}
	a.OtherAspects = append(a.OtherAspects[:index], a.OtherAspects[index+1:]...)
	return true
}

// SetAspect sets an aspect at a specific index in the other aspects
func (a *Aspects) SetAspect(index int, aspect string) bool {
	if index < 0 || index >= len(a.OtherAspects) {
		return false
	}
	a.OtherAspects[index] = aspect
	return true
}

// Count returns the total number of non-empty aspects
func (a *Aspects) Count() int {
	count := 0
	if a.HighConcept != "" {
		count++
	}
	if a.Trouble != "" {
		count++
	}
	for _, aspect := range a.OtherAspects {
		if aspect != "" {
			count++
		}
	}
	return count
}

// Stunt represents a character stunt
type Stunt struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	SkillUsed   string   `json:"skill_used,omitempty"`
	Effect      string   `json:"effect"`
	Conditions  []string `json:"conditions,omitempty"`
}

// StressTrackType represents the type of stress track
type StressTrackType string

const (
	PhysicalStress StressTrackType = "physical"
	MentalStress   StressTrackType = "mental"
)

// StressTrack represents physical or mental stress
type StressTrack struct {
	Type     StressTrackType `json:"type"`
	Boxes    []bool          `json:"boxes"`
	MaxBoxes int             `json:"max_boxes"`
}

// NewStressTrack creates a new stress track with the specified number of boxes
func NewStressTrack(trackType StressTrackType, maxBoxes int) *StressTrack {
	return &StressTrack{
		Type:     trackType,
		Boxes:    make([]bool, maxBoxes),
		MaxBoxes: maxBoxes,
	}
}

// IsFull returns true if all stress boxes are filled
func (st *StressTrack) IsFull() bool {
	for _, box := range st.Boxes {
		if !box {
			return false
		}
	}
	return true
}

// AvailableBoxes returns the number of unfilled stress boxes
func (st *StressTrack) AvailableBoxes() int {
	count := 0
	for _, box := range st.Boxes {
		if !box {
			count++
		}
	}
	return count
}

// TakeStress marks stress boxes and returns true if successful
func (st *StressTrack) TakeStress(amount int) bool {
	if amount <= 0 || amount > len(st.Boxes) {
		return false
	}

	// Check if the specific box is available (1-indexed)
	boxIndex := amount - 1
	if boxIndex >= len(st.Boxes) || st.Boxes[boxIndex] {
		return false
	}

	st.Boxes[boxIndex] = true
	return true
}

// ClearStress clears a specific stress box (1-indexed)
func (st *StressTrack) ClearStress(box int) bool {
	if box <= 0 || box > len(st.Boxes) {
		return false
	}

	boxIndex := box - 1
	if !st.Boxes[boxIndex] {
		return false
	}

	st.Boxes[boxIndex] = false
	return true
}

// ClearAll resets all stress boxes to unchecked.
// Per Fate Core: "After a conflict, when you get a minute to breathe,
// any stress boxes you checked off become available for your use again."
func (st *StressTrack) ClearAll() {
	for i := range st.Boxes {
		st.Boxes[i] = false
	}
}

// String returns a visual representation of the stress track
func (st *StressTrack) String() string {
	result := fmt.Sprintf("%s Stress: ", st.Type)
	for i, box := range st.Boxes {
		if box {
			result += fmt.Sprintf("[%d]", i+1)
		} else {
			result += fmt.Sprintf(" %d ", i+1)
		}
	}
	return result
}

// ConsequenceType represents the severity of a consequence
type ConsequenceType string

const (
	MildConsequence     ConsequenceType = "mild"
	ModerateConsequence ConsequenceType = "moderate"
	SevereConsequence   ConsequenceType = "severe"
	ExtremeConsequence  ConsequenceType = "extreme"
)

// ConsequenceValue returns the stress value absorbed by this consequence type
func (ct ConsequenceType) Value() int {
	switch ct {
	case MildConsequence:
		return 2
	case ModerateConsequence:
		return 4
	case SevereConsequence:
		return 6
	case ExtremeConsequence:
		return 8
	default:
		return 0
	}
}

// Consequence represents character consequences
type Consequence struct {
	ID                    string          `json:"id"`
	Type                  ConsequenceType `json:"type"`
	Aspect                string          `json:"aspect"`
	Duration              string          `json:"duration"`
	CreatedAt             time.Time       `json:"created_at"`
	Recovering            bool            `json:"recovering"`              // Whether recovery action has been taken
	RecoveryStartScene    int             `json:"recovery_start_scene"`    // Scene count when recovery began
	RecoveryStartScenario int             `json:"recovery_start_scenario"` // Scenario count when recovery began
}

// NewCharacter creates a new character with default values.
// Stress track sizes are set by RecalculateStressTracks based on Physique and Will.
func NewCharacter(id, name string) *Character {
	char := &Character{
		ID:           id,
		Name:         name,
		Aspects:      Aspects{OtherAspects: make([]string, 0)},
		Skills:       make(map[string]dice.Ladder),
		Stunts:       make([]Stunt, 0),
		FatePoints:   3, // Default starting fate points
		Refresh:      3, // Default refresh
		StressTracks: make(map[string]*StressTrack),
		Consequences: make([]Consequence, 0),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	char.RecalculateStressTracks()
	return char
}

// InitDefaults fills in runtime fields (stress tracks, timestamps, nil slices/maps)
// that are not present in serialized data. Call this after unmarshaling from YAML/JSON
// into a zero-value Character. Stress track sizes are always recalculated from the
// character's current Physique and Will skills.
func (c *Character) InitDefaults() {
	if c.Skills == nil {
		c.Skills = make(map[string]dice.Ladder)
	}
	if c.Stunts == nil {
		c.Stunts = make([]Stunt, 0)
	}
	if c.Aspects.OtherAspects == nil {
		c.Aspects.OtherAspects = make([]string, 0)
	}
	if c.Consequences == nil {
		c.Consequences = make([]Consequence, 0)
	}
	if c.StressTracks == nil {
		c.StressTracks = make(map[string]*StressTrack)
	}
	c.RecalculateStressTracks()
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
}

// GetSkill returns the character's skill level, defaulting to Mediocre if not found
func (c *Character) GetSkill(skillName string) dice.Ladder {
	if level, exists := c.Skills[skillName]; exists {
		return level
	}
	return dice.Mediocre
}

// SetSkill sets a character's skill level and recalculates stress tracks when
// Physique or Will changes, keeping them in sync with the new skill rating.
func (c *Character) SetSkill(skillName string, level dice.Ladder) {
	c.Skills[skillName] = level
	c.UpdatedAt = time.Now()
	if skillName == "Physique" || skillName == "Will" {
		c.RecalculateStressTracks()
	}
}

// stressBoxesForSkill returns the number of stress boxes granted by a skill rating.
// Per Fate Core SRD (Physique and Will sections):
//   - Mediocre (+0) or no skill → 2 boxes
//   - Average (+1) or Fair (+2) → 3 boxes
//   - Good (+3) or higher       → 4 boxes
func stressBoxesForSkill(level dice.Ladder) int {
	switch {
	case level >= dice.Good:
		return 4
	case level >= dice.Average:
		return 3
	default:
		return 2
	}
}

// RecalculateStressTracks resizes the physical and mental stress tracks based on
// the character's Physique and Will skills respectively, per the Fate Core SRD.
// Checked boxes are preserved when expanding; the track is trimmed to the new
// maximum when shrinking (preserving any already-checked boxes up to the new size).
func (c *Character) RecalculateStressTracks() {
	if c.StressTracks == nil {
		c.StressTracks = make(map[string]*StressTrack)
	}
	c.recalculateTrack(PhysicalStress, c.GetSkill("Physique"))
	c.recalculateTrack(MentalStress, c.GetSkill("Will"))
}

// recalculateTrack resizes a single stress track to match the size dictated by
// the given skill level, preserving any already-checked boxes.
func (c *Character) recalculateTrack(trackType StressTrackType, skillLevel dice.Ladder) {
	newMax := stressBoxesForSkill(skillLevel)
	track := c.GetStressTrack(trackType)
	if track == nil {
		c.StressTracks[string(trackType)] = NewStressTrack(trackType, newMax)
		return
	}
	if track.MaxBoxes == newMax {
		return
	}
	newBoxes := make([]bool, newMax)
	copy(newBoxes, track.Boxes)
	track.Boxes = newBoxes
	track.MaxBoxes = newMax
}

// extraMildConsequences returns the number of additional mild consequence slots
// granted by Physique (physical) and Will (mental) at Superb (+5) or higher,
// per the Fate Core SRD.
func (c *Character) extraMildConsequences() int {
	extra := 0
	if c.GetSkill("Physique") >= dice.Superb {
		extra++
	}
	if c.GetSkill("Will") >= dice.Superb {
		extra++
	}
	return extra
}

// AddStunt adds a stunt to the character
func (c *Character) AddStunt(stunt Stunt) {
	c.Stunts = append(c.Stunts, stunt)
	c.UpdatedAt = time.Now()
}

// SpendFatePoint spends a fate point if available
func (c *Character) SpendFatePoint() bool {
	if c.FatePoints > 0 {
		c.FatePoints--
		c.UpdatedAt = time.Now()
		return true
	}
	return false
}

// GainFatePoint adds a fate point to the character
func (c *Character) GainFatePoint() {
	c.FatePoints++
	c.UpdatedAt = time.Now()
}

// RefreshFatePoints resets fate points to refresh rate
func (c *Character) RefreshFatePoints() {
	c.FatePoints = c.Refresh
	c.UpdatedAt = time.Now()
}

// GetStressTrack returns the specified stress track
func (c *Character) GetStressTrack(trackType StressTrackType) *StressTrack {
	return c.StressTracks[string(trackType)]
}

// TakeStress attempts to absorb stress using the appropriate track
func (c *Character) TakeStress(trackType StressTrackType, amount int) bool {
	track := c.GetStressTrack(trackType)
	if track != nil && track.TakeStress(amount) {
		c.UpdatedAt = time.Now()
		return true
	}
	return false
}

// ClearAllStress resets all stress tracks.
// Per Fate Core, stress clears after each conflict.
func (c *Character) ClearAllStress() {
	for _, track := range c.StressTracks {
		track.ClearAll()
	}
	c.UpdatedAt = time.Now()
}

// AddConsequence adds a consequence to the character
func (c *Character) AddConsequence(consequence Consequence) {
	c.Consequences = append(c.Consequences, consequence)
	c.UpdatedAt = time.Now()
}

// RemoveConsequence removes a consequence by ID
func (c *Character) RemoveConsequence(id string) bool {
	for i, conseq := range c.Consequences {
		if conseq.ID == id {
			c.Consequences = append(c.Consequences[:i], c.Consequences[i+1:]...)
			c.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// BeginConsequenceRecovery marks a consequence as recovering and records the
// current scene and scenario counts so elapsed time can be measured later.
func (c *Character) BeginConsequenceRecovery(id string, sceneCount, scenarioCount int) bool {
	for i := range c.Consequences {
		if c.Consequences[i].ID == id {
			c.Consequences[i].Recovering = true
			c.Consequences[i].RecoveryStartScene = sceneCount
			c.Consequences[i].RecoveryStartScenario = scenarioCount
			c.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// CheckConsequenceRecovery checks all recovering consequences and removes those
// that have healed based on Fate Core timing rules:
//   - Mild: 1 whole scene after recovery action
//   - Moderate: 1 whole scenario after recovery action
//   - Severe: 1 whole scenario after recovery action (mapped to scenario boundary)
//
// Returns the list of cleared consequences.
func (c *Character) CheckConsequenceRecovery(currentScene, currentScenario int) []Consequence {
	var cleared []Consequence
	var remaining []Consequence

	for _, conseq := range c.Consequences {
		if !conseq.Recovering {
			remaining = append(remaining, conseq)
			continue
		}

		healed := false
		switch conseq.Type {
		case MildConsequence:
			// Clears after 1 whole scene post recovery action
			if currentScene > conseq.RecoveryStartScene {
				healed = true
			}
		case ModerateConsequence:
			// Clears after 1 whole scenario post recovery action
			if currentScenario > conseq.RecoveryStartScenario {
				healed = true
			}
		case SevereConsequence:
			// Clears after 1 whole scenario post recovery action
			// Note: per Fate Core this should be a longer period; see issue for improvement
			if currentScenario > conseq.RecoveryStartScenario {
				healed = true
			}
		}

		if healed {
			cleared = append(cleared, conseq)
		} else {
			remaining = append(remaining, conseq)
		}
	}

	if len(cleared) > 0 {
		c.Consequences = remaining
		c.UpdatedAt = time.Now()
	}

	return cleared
}

// ConsequenceSlot represents an available consequence slot and how much stress it absorbs.
type ConsequenceSlot struct {
	Type  ConsequenceType
	Value int
}

// AvailableConsequenceSlots returns all consequence slots available for the character.
// The result respects the character's NPC type restrictions (e.g., nameless NPCs
// have no consequence slots, supporting NPCs only have mild).
func (c *Character) AvailableConsequenceSlots() []ConsequenceSlot {
	var slots []ConsequenceSlot

	if c.CanTakeConsequence(MildConsequence) {
		slots = append(slots, ConsequenceSlot{Type: MildConsequence, Value: MildConsequence.Value()})
	}
	if c.CanTakeConsequence(ModerateConsequence) {
		slots = append(slots, ConsequenceSlot{Type: ModerateConsequence, Value: ModerateConsequence.Value()})
	}
	if c.CanTakeConsequence(SevereConsequence) {
		slots = append(slots, ConsequenceSlot{Type: SevereConsequence, Value: SevereConsequence.Value()})
	}

	return slots
}

// BestConsequenceFor picks the most appropriate consequence slot to absorb shifts.
// Prefers the smallest consequence that fully covers the shifts.
// If none covers the shifts, returns the largest available to minimise leftover damage.
// Returns the chosen slot and true, or zero-value and false when no slots are provided.
func BestConsequenceFor(slots []ConsequenceSlot, shifts int) (ConsequenceSlot, bool) {
	if len(slots) == 0 {
		return ConsequenceSlot{}, false
	}

	best := slots[0]
	for _, s := range slots[1:] {
		if s.Value >= shifts {
			// s covers the damage; prefer it when best doesn't cover shifts
			// or when s is a tighter fit.
			if best.Value < shifts || s.Value < best.Value {
				best = s
			}
		} else if best.Value < shifts && s.Value > best.Value {
			// Neither covers shifts — track the largest to minimise remainder.
			best = s
		}
	}
	return best, true
}

// HasAspect checks if the character has a specific aspect
func (c *Character) HasAspect(aspect string) bool {
	aspects := c.Aspects.GetAll()
	for _, a := range aspects {
		if a == aspect {
			return true
		}
	}

	// Also check consequences as they create temporary aspects
	for _, consequence := range c.Consequences {
		if consequence.Aspect == aspect {
			return true
		}
	}

	return false
}
