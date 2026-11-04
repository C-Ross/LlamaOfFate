package core

import "github.com/C-Ross/LlamaOfFate/internal/core/dice"

// Named constants for the 18 default Fate Core skills.
// See: https://fate-srd.com/fate-core/default-skill-list
const (
	SkillAthletics   = "Athletics"
	SkillBurglary    = "Burglary"
	SkillContacts    = "Contacts"
	SkillCrafts      = "Crafts"
	SkillDeceive     = "Deceive"
	SkillDrive       = "Drive"
	SkillEmpathy     = "Empathy"
	SkillFight       = "Fight"
	SkillInvestigate = "Investigate"
	SkillLore        = "Lore"
	SkillNotice      = "Notice"
	SkillPhysique    = "Physique"
	SkillProvoke     = "Provoke"
	SkillRapport     = "Rapport"
	SkillResources   = "Resources"
	SkillShoot       = "Shoot"
	SkillStealth     = "Stealth"
	SkillWill        = "Will"
)

// FateCoreSkills is the canonical, alphabetically sorted list of the 18
// default Fate Core skills.
var FateCoreSkills = []string{
	SkillAthletics,
	SkillBurglary,
	SkillContacts,
	SkillCrafts,
	SkillDeceive,
	SkillDrive,
	SkillEmpathy,
	SkillFight,
	SkillInvestigate,
	SkillLore,
	SkillNotice,
	SkillPhysique,
	SkillProvoke,
	SkillRapport,
	SkillResources,
	SkillShoot,
	SkillStealth,
	SkillWill,
}

// DefaultSkillPriority lists the 10 most generally useful Fate Core skills,
// ordered by versatility. Used by DefaultPyramid to fill a standard pyramid.
// Mirrors web/src/lib/skills.ts DEFAULT_SKILL_PRIORITY.
var DefaultSkillPriority = []string{
	SkillNotice,
	SkillAthletics,
	SkillWill,
	SkillInvestigate,
	SkillRapport,
	SkillFight,
	SkillStealth,
	SkillPhysique,
	SkillEmpathy,
	SkillShoot,
}

// DefaultPyramid returns a standard Fate Core skill pyramid (cap=Great)
// using DefaultSkillPriority: 1×Great, 2×Good, 3×Fair, 4×Average.
// Returns a fresh map each call so callers can mutate freely.
func DefaultPyramid() map[string]dice.Ladder {
	shape := []struct {
		level dice.Ladder
		count int
	}{
		{dice.Great, 1},
		{dice.Good, 2},
		{dice.Fair, 3},
		{dice.Average, 4},
	}
	result := make(map[string]dice.Ladder, 10)
	i := 0
	for _, tier := range shape {
		for range tier.count {
			result[DefaultSkillPriority[i]] = tier.level
			i++
		}
	}
	return result
}

// fateCoreSkillSet is a precomputed set for O(1) lookups.
// Built at var-init time (before any init() functions run) so that
// IsValidSkill is safe to call from other init() functions in this package.
var fateCoreSkillSet = buildSkillSet(FateCoreSkills)

func buildSkillSet(skills []string) map[string]bool {
	m := make(map[string]bool, len(skills))
	for _, s := range skills {
		m[s] = true
	}
	return m
}

// IsValidSkill reports whether name is one of the 18 default Fate Core skills.
func IsValidSkill(name string) bool {
	return fateCoreSkillSet[name]
}
