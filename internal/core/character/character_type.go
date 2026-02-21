package character

import "github.com/C-Ross/LlamaOfFate/internal/core/dice"

// CharacterType represents the category of character according to Fate Core rules.
// See: https://fate-srd.com/fate-core/creating-and-playing-opposition
type CharacterType int

const (
	// CharacterTypePC represents a Player Character with full aspects, skills,
	// stress tracks, and consequences.
	CharacterTypePC CharacterType = iota

	// CharacterTypeMainNPC represents a full NPC with complete aspects, skills,
	// stress tracks, and consequences. Used for major antagonists and rivals.
	// See: https://fate-srd.com/fate-core/creating-and-playing-opposition#main-npcs
	CharacterTypeMainNPC

	// CharacterTypeSupportingNPC represents an NPC with limited aspects and consequences.
	// Has 1-2 aspects, a small skill set, light stress, and only Mild consequences.
	// See: https://fate-srd.com/fate-core/creating-and-playing-opposition#supporting-npcs
	CharacterTypeSupportingNPC

	// CharacterTypeNamelessGood represents a Good (+3) nameless NPC.
	// Has a single aspect, one skill at Good (+3), 2 stress boxes, no consequences.
	// See: https://fate-srd.com/fate-core/creating-and-playing-opposition#good-nameless-npcs
	CharacterTypeNamelessGood

	// CharacterTypeNamelessFair represents a Fair (+2) nameless NPC.
	// Has a single aspect, one skill at Fair (+2), 1 stress box, no consequences.
	// See: https://fate-srd.com/fate-core/creating-and-playing-opposition#fair-nameless-npcs
	CharacterTypeNamelessFair

	// CharacterTypeNamelessAverage represents an Average (+1) nameless NPC.
	// Has no aspects, one skill at Average (+1), no stress boxes, no consequences.
	// Taken out on any successful hit.
	// See: https://fate-srd.com/fate-core/creating-and-playing-opposition#average-nameless-npcs
	CharacterTypeNamelessAverage
)

// String returns the string representation of a character type
func (t CharacterType) String() string {
	switch t {
	case CharacterTypePC:
		return "Player Character"
	case CharacterTypeMainNPC:
		return "Main NPC"
	case CharacterTypeSupportingNPC:
		return "Supporting NPC"
	case CharacterTypeNamelessGood:
		return "Nameless (Good)"
	case CharacterTypeNamelessFair:
		return "Nameless (Fair)"
	case CharacterTypeNamelessAverage:
		return "Nameless (Average)"
	default:
		return "Unknown"
	}
}

// IsPC returns true if this is a player character
func (t CharacterType) IsPC() bool {
	return t == CharacterTypePC
}

// IsNPC returns true if this is any type of NPC
func (t CharacterType) IsNPC() bool {
	return t != CharacterTypePC
}

// IsNameless returns true if this is a nameless NPC
func (t CharacterType) IsNameless() bool {
	return t == CharacterTypeNamelessGood || t == CharacterTypeNamelessFair || t == CharacterTypeNamelessAverage
}

// HasConsequences returns true if this character type can take consequences
func (t CharacterType) HasConsequences() bool {
	// PCs, Main NPCs, and Supporting NPCs can take consequences
	// Nameless NPCs are taken out when they can't absorb stress
	return t == CharacterTypePC || t == CharacterTypeMainNPC || t == CharacterTypeSupportingNPC
}

// MaxConsequenceSeverity returns the maximum consequence severity for this character type
func (t CharacterType) MaxConsequenceSeverity() ConsequenceType {
	switch t {
	case CharacterTypePC, CharacterTypeMainNPC:
		return SevereConsequence // Can take Mild, Moderate, Severe
	case CharacterTypeSupportingNPC:
		return MildConsequence // Can only take Mild (maybe Moderate for important ones)
	default:
		return "" // Nameless NPCs have no consequences
	}
}

// NewNamelessNPC creates a nameless NPC with the given name, aspect, and primary skill.
// The skill level and stress boxes are determined by the character type.
func NewNamelessNPC(id, name string, charType CharacterType, primarySkill string) *Character {
	char := &Character{
		ID:            id,
		Name:          name,
		CharacterType: charType,
		Aspects:       Aspects{OtherAspects: make([]string, 0)},
		Skills:        make(map[string]dice.Ladder),
		Stunts:        make([]Stunt, 0),
		FatePoints:    0, // Nameless NPCs don't have fate points
		Refresh:       0,
		StressTracks:  make(map[string]*StressTrack),
		Consequences:  make([]Consequence, 0),
	}

	// Set skill level and stress based on type
	switch charType {
	case CharacterTypeNamelessGood:
		char.Skills[primarySkill] = dice.Good // +3
		char.StressTracks[string(PhysicalStress)] = NewStressTrack(PhysicalStress, 2)
	case CharacterTypeNamelessFair:
		char.Skills[primarySkill] = dice.Fair // +2
		char.StressTracks[string(PhysicalStress)] = NewStressTrack(PhysicalStress, 1)
	case CharacterTypeNamelessAverage:
		char.Skills[primarySkill] = dice.Average // +1
		// No stress track - taken out on any hit
	}

	return char
}

// NewSupportingNPC creates a supporting NPC with the given name and high concept.
// Supporting NPCs have stress tracks sized by Physique/Will (defaulting to 2 boxes
// each) and can only take Mild consequences.
func NewSupportingNPC(id, name, highConcept string) *Character {
	char := &Character{
		ID:            id,
		Name:          name,
		CharacterType: CharacterTypeSupportingNPC,
		Aspects: Aspects{
			HighConcept:  highConcept,
			OtherAspects: make([]string, 0),
		},
		Skills:       make(map[string]dice.Ladder),
		Stunts:       make([]Stunt, 0),
		FatePoints:   1, // Limited fate points
		Refresh:      1,
		StressTracks: make(map[string]*StressTrack),
		Consequences: make([]Consequence, 0),
	}
	char.RecalculateStressTracks()
	return char
}

// NewMainNPC creates a main NPC (villain/rival) with full character capabilities.
// Main NPCs function like player characters with full aspects, skills, and consequences.
func NewMainNPC(id, name string) *Character {
	char := NewCharacter(id, name)
	char.CharacterType = CharacterTypeMainNPC
	return char
}

// CanTakeConsequence returns true if the character can take a consequence of the given type
func (c *Character) CanTakeConsequence(conseqType ConsequenceType) bool {
	// Check character type restrictions
	maxSeverity := c.CharacterType.MaxConsequenceSeverity()
	if maxSeverity == "" {
		return false // Nameless NPCs can't take consequences
	}

	// Check if this severity is allowed for this character type
	if !isConsequenceSeverityAllowed(conseqType, maxSeverity) {
		return false
	}

	// Count how many of this type are already taken
	count := 0
	for _, existing := range c.Consequences {
		if existing.Type == conseqType {
			count++
		}
	}

	// Mild consequences can have additional slots from Physique/Will at Superb+
	if conseqType == MildConsequence {
		return count < 1+c.extraMildConsequences()
	}
	return count == 0
}

// isConsequenceSeverityAllowed checks if a consequence type is at or below the max severity
func isConsequenceSeverityAllowed(conseqType, maxSeverity ConsequenceType) bool {
	severityOrder := map[ConsequenceType]int{
		MildConsequence:     1,
		ModerateConsequence: 2,
		SevereConsequence:   3,
		ExtremeConsequence:  4,
	}

	return severityOrder[conseqType] <= severityOrder[maxSeverity]
}
