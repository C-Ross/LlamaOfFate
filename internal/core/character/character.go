package character

import (
	"fmt"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
)

// Character represents a player character or NPC
type Character struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Core Character Elements
	Aspects Aspects                `json:"aspects"`
	Skills  map[string]dice.Ladder `json:"skills"`
	Stunts  []Stunt                `json:"stunts"`

	// Resources
	FatePoints int `json:"fate_points"`
	Refresh    int `json:"refresh"`

	// Health and Status
	StressTracks map[string]*StressTrack `json:"stress_tracks"`
	Consequences []Consequence           `json:"consequences"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Aspects holds the character aspects
type Aspects struct {
	HighConcept  string   `json:"high_concept"`
	Trouble      string   `json:"trouble"`
	OtherAspects []string `json:"other_aspects"`
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
	ID        string          `json:"id"`
	Type      ConsequenceType `json:"type"`
	Aspect    string          `json:"aspect"`
	Duration  string          `json:"duration"`
	CreatedAt time.Time       `json:"created_at"`
}

// NewCharacter creates a new character with default values
func NewCharacter(id, name string) *Character {
	return &Character{
		ID:         id,
		Name:       name,
		Aspects:    Aspects{OtherAspects: make([]string, 0)},
		Skills:     make(map[string]dice.Ladder),
		Stunts:     make([]Stunt, 0),
		FatePoints: 3, // Default starting fate points
		Refresh:    3, // Default refresh
		StressTracks: map[string]*StressTrack{
			string(PhysicalStress): NewStressTrack(PhysicalStress, 2),
			string(MentalStress):   NewStressTrack(MentalStress, 2),
		},
		Consequences: make([]Consequence, 0),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// GetSkill returns the character's skill level, defaulting to Mediocre if not found
func (c *Character) GetSkill(skillName string) dice.Ladder {
	if level, exists := c.Skills[skillName]; exists {
		return level
	}
	return dice.Mediocre
}

// SetSkill sets a character's skill level
func (c *Character) SetSkill(skillName string, level dice.Ladder) {
	c.Skills[skillName] = level
	c.UpdatedAt = time.Now()
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

// AddConsequence adds a consequence to the character
func (c *Character) AddConsequence(consequence Consequence) {
	c.Consequences = append(c.Consequences, consequence)
	c.UpdatedAt = time.Now()
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
