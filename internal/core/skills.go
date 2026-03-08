// Package core provides Fate Core skill classification utilities.
// These functions encode the Fate Core rules for skill categorization
// as described in the Fate SRD at https://fate-srd.com/fate-core
package core

import (
	"fmt"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

func init() {
	// Verify that every skill referenced in classification maps is a valid
	// Fate Core skill. This catches typos and drift at startup.
	maps := []map[string]bool{
		physicalAttackSkills,
		mentalAttackSkills,
		physicalConflictSkills,
		mentalConflictSkills,
	}
	for _, m := range maps {
		for skill := range m {
			if !IsValidSkill(skill) {
				panic(fmt.Sprintf("core: skill classification references unknown skill %q", skill))
			}
		}
	}
}

// physicalAttackSkills are skills that deal physical stress when used to attack.
// Per Fate Core SRD: only Fight (melee) and Shoot (ranged) have the Attack action.
// Physique is Overcome/Create Advantage only.
// See: https://fate-srd.com/fate-core/default-skill-list
var physicalAttackSkills = map[string]bool{
	SkillFight: true,
	SkillShoot: true,
}

// mentalAttackSkills are skills that deal mental stress when used to attack.
// Per Fate Core SRD: only Provoke has the Attack action for mental conflicts.
// Deceive, Rapport, and Lore are Overcome/Create Advantage/Defend only.
// See: https://fate-srd.com/fate-core/default-skill-list
var mentalAttackSkills = map[string]bool{
	SkillProvoke: true,
}

// physicalConflictSkills trigger or indicate physical conflicts
var physicalConflictSkills = map[string]bool{
	SkillFight:     true,
	SkillShoot:     true,
	SkillAthletics: true,
	SkillPhysique:  true,
}

// mentalConflictSkills trigger or indicate mental conflicts
var mentalConflictSkills = map[string]bool{
	SkillProvoke: true,
	SkillDeceive: true,
	SkillRapport: true,
	SkillWill:    true,
	SkillEmpathy: true,
}

// DefenseSkillForAttack returns the appropriate defense skill for an attack skill.
// Per Fate Core rules:
// - Physical attacks (Fight, Shoot) are defended with Athletics
// - Mental/social attacks (Provoke) are defended with Will
func DefenseSkillForAttack(attackSkill string) string {
	if physicalAttackSkills[attackSkill] {
		return SkillAthletics
	}
	if mentalAttackSkills[attackSkill] {
		return SkillWill
	}
	// Default to Athletics for unknown attack types
	return SkillAthletics
}

// StressTypeForAttack determines which stress track an attack skill targets.
// Per Fate Core rules:
// - Physical attacks target physical stress
// - Mental/social attacks target mental stress
func StressTypeForAttack(attackSkill string) StressTrackType {
	if mentalAttackSkills[attackSkill] {
		return MentalStress
	}
	return PhysicalStress
}

// ConflictTypeForSkill determines the conflict type based on the skill used.
// Per Fate Core rules:
// - Physical skills (Fight, Shoot, Athletics, Physique) indicate physical conflict
// - Mental skills (Provoke, Deceive, Rapport, Will, Empathy) indicate mental conflict
func ConflictTypeForSkill(skill string) scene.ConflictType {
	if physicalConflictSkills[skill] {
		return scene.PhysicalConflict
	}
	if mentalConflictSkills[skill] {
		return scene.MentalConflict
	}
	// Default to physical for unknown skills
	return scene.PhysicalConflict
}

// IsPhysicalAttackSkill returns true if the skill deals physical damage
func IsPhysicalAttackSkill(skill string) bool {
	return physicalAttackSkills[skill]
}

// IsMentalAttackSkill returns true if the skill deals mental damage
func IsMentalAttackSkill(skill string) bool {
	return mentalAttackSkills[skill]
}

// InitiativeSkillsForConflict returns the ordered list of skills to check for initiative.
// Per Fate Core rules:
// - Physical conflicts: Notice, then Athletics as fallback
// - Mental conflicts: Empathy, then Rapport as fallback
func InitiativeSkillsForConflict(conflictType scene.ConflictType) []string {
	if conflictType == scene.PhysicalConflict {
		return []string{SkillNotice, SkillAthletics}
	}
	return []string{SkillEmpathy, SkillRapport}
}

// SkillGetter is an interface for types that can retrieve skill values.
// This allows CalculateInitiative to work with Character without creating import cycles.
type SkillGetter interface {
	GetSkill(name string) dice.Ladder
}

// CalculateInitiative returns the initiative value for a character based on conflict type.
// It checks skills in priority order and returns the first non-zero value.
func CalculateInitiative(char SkillGetter, conflictType scene.ConflictType) int {
	for _, skill := range InitiativeSkillsForConflict(conflictType) {
		if initiative := int(char.GetSkill(skill)); initiative > 0 {
			return initiative
		}
	}
	return 0
}

// DefaultAttackSkillForConflict returns the default attack skill for a conflict type.
// Per Fate Core rules:
// - Physical conflicts default to Fight
// - Mental conflicts default to Provoke
func DefaultAttackSkillForConflict(conflictType scene.ConflictType) string {
	if conflictType == scene.MentalConflict {
		return SkillProvoke
	}
	return SkillFight
}

// StressTypeForConflict returns the stress track type for a given conflict type.
// Per Fate Core rules:
// - Physical conflicts deal physical stress
// - Mental conflicts deal mental stress
func StressTypeForConflict(conflictType scene.ConflictType) StressTrackType {
	if conflictType == scene.MentalConflict {
		return MentalStress
	}
	return PhysicalStress
}

// ConcessionFatePoints returns the number of fate points awarded for conceding a conflict.
// Per Fate Core: "you get a fate point for choosing to concede. On top of that,
// if you've sustained any consequences in this conflict, you get an additional
// fate point for each consequence."
// See: https://fate-srd.com/fate-core/conflicts#conceding-the-conflict
func ConcessionFatePoints(consequenceCount int) int {
	return 1 + consequenceCount
}
