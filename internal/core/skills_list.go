package core

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
