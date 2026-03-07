package core

import (
	"fmt"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper: build a valid standard pyramid (cap=Great, 10 skills)
// ---------------------------------------------------------------------------

func validGreatPyramid() map[string]dice.Ladder {
	return map[string]dice.Ladder{
		SkillFight:       dice.Great, // 1×Great
		SkillShoot:       dice.Good,  // 2×Good
		SkillAthletics:   dice.Good,
		SkillNotice:      dice.Fair, // 3×Fair
		SkillStealth:     dice.Fair,
		SkillWill:        dice.Fair,
		SkillPhysique:    dice.Average, // 4×Average
		SkillProvoke:     dice.Average,
		SkillRapport:     dice.Average,
		SkillInvestigate: dice.Average,
	}
}

func validSuperbPyramid() map[string]dice.Ladder {
	return map[string]dice.Ladder{
		// 1×Superb
		SkillFight: dice.Superb,
		// 2×Great
		SkillShoot:     dice.Great,
		SkillAthletics: dice.Great,
		// 3×Good
		SkillNotice:  dice.Good,
		SkillStealth: dice.Good,
		SkillWill:    dice.Good,
		// 4×Fair
		SkillPhysique:    dice.Fair,
		SkillProvoke:     dice.Fair,
		SkillRapport:     dice.Fair,
		SkillInvestigate: dice.Fair,
		// 5×Average
		SkillDeceive:   dice.Average,
		SkillEmpathy:   dice.Average,
		SkillLore:      dice.Average,
		SkillContacts:  dice.Average,
		SkillResources: dice.Average,
	}
}

// ---------------------------------------------------------------------------
// SkillDistribution.String
// ---------------------------------------------------------------------------

func TestSkillDistribution_String(t *testing.T) {
	assert.Equal(t, "pyramid", DistributionPyramid.String())
	assert.Equal(t, "columns", DistributionColumns.String())
	assert.Equal(t, "single column", DistributionSingleColumn.String())
	assert.Contains(t, SkillDistribution(99).String(), "99")
}

// ---------------------------------------------------------------------------
// Empty / nil input
// ---------------------------------------------------------------------------

func TestValidateSkillDistribution_NilMap(t *testing.T) {
	err := ValidateSkillDistribution(nil, dice.Great, DistributionPyramid, FateCoreSkills)
	assert.NoError(t, err, "nil map should be valid (use defaults)")
}

func TestValidateSkillDistribution_EmptyMap(t *testing.T) {
	err := ValidateSkillDistribution(map[string]dice.Ladder{}, dice.Great, DistributionPyramid, FateCoreSkills)
	assert.NoError(t, err, "empty map should be valid (use defaults)")
}

// ---------------------------------------------------------------------------
// Skill cap validation
// ---------------------------------------------------------------------------

func TestValidateSkillDistribution_CapBelowAverage(t *testing.T) {
	skills := map[string]dice.Ladder{SkillFight: dice.Average}
	err := ValidateSkillDistribution(skills, dice.Mediocre, DistributionPyramid, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least Average")
}

// ---------------------------------------------------------------------------
// Pyramid mode — valid cases
// ---------------------------------------------------------------------------

func TestValidatePyramid_GreatCap_Valid(t *testing.T) {
	err := ValidateSkillDistribution(validGreatPyramid(), dice.Great, DistributionPyramid, FateCoreSkills)
	assert.NoError(t, err)
}

func TestValidatePyramid_SuperbCap_Valid(t *testing.T) {
	err := ValidateSkillDistribution(validSuperbPyramid(), dice.Superb, DistributionPyramid, FateCoreSkills)
	assert.NoError(t, err)
}

func TestValidatePyramid_FantasticCap_Valid_NoNameCheck(t *testing.T) {
	// Fantastic(+6) pyramid = 21 skills, but we only have 18 canonical names.
	// Use nil validSkills to skip name validation (suitable for custom skill lists).
	skills := make(map[string]dice.Ladder)
	idx := 0
	for rank := dice.Fantastic; rank >= dice.Average; rank-- {
		count := int(dice.Fantastic-rank) + 1
		for i := 0; i < count; i++ {
			skills[fmt.Sprintf("Skill%d", idx)] = rank
			idx++
		}
	}
	require.Len(t, skills, 21)
	err := ValidateSkillDistribution(skills, dice.Fantastic, DistributionPyramid, nil)
	assert.NoError(t, err)
}

func TestValidatePyramid_AverageCap_Valid(t *testing.T) {
	// Minimal pyramid: cap=Average → just 1×Average skill.
	skills := map[string]dice.Ladder{SkillFight: dice.Average}
	err := ValidateSkillDistribution(skills, dice.Average, DistributionPyramid, FateCoreSkills)
	assert.NoError(t, err)
}

func TestValidatePyramid_FairCap_Valid(t *testing.T) {
	// cap=Fair → 1×Fair, 2×Average = 3 skills
	skills := map[string]dice.Ladder{
		SkillFight:     dice.Fair,
		SkillShoot:     dice.Average,
		SkillAthletics: dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Fair, DistributionPyramid, FateCoreSkills)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Pyramid mode — invalid cases
// ---------------------------------------------------------------------------

func TestValidatePyramid_TooManyAtCap(t *testing.T) {
	skills := validGreatPyramid()
	skills[SkillDeceive] = dice.Great // now 2×Great instead of 1
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Great")
}

func TestValidatePyramid_MissingTier(t *testing.T) {
	// Remove all Fair skills from a Great pyramid.
	skills := map[string]dice.Ladder{
		SkillFight:       dice.Great,
		SkillShoot:       dice.Good,
		SkillAthletics:   dice.Good,
		SkillPhysique:    dice.Average,
		SkillProvoke:     dice.Average,
		SkillRapport:     dice.Average,
		SkillInvestigate: dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Fair")
}

func TestValidatePyramid_UnknownSkill(t *testing.T) {
	skills := validGreatPyramid()
	delete(skills, SkillFight)
	skills["Swordfighting"] = dice.Great
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown skill")
	assert.Contains(t, err.Error(), "Swordfighting")
}

func TestValidatePyramid_MultipleUnknownSkills(t *testing.T) {
	skills := map[string]dice.Ladder{
		"Hacking":  dice.Great,
		"Piloting": dice.Good,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Hacking")
	assert.Contains(t, err.Error(), "Piloting")
}

func TestValidatePyramid_SkillAboveCap(t *testing.T) {
	skills := map[string]dice.Ladder{
		SkillFight: dice.Superb, // cap is Great
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds the skill cap")
}

func TestValidatePyramid_SkillBelowAverage(t *testing.T) {
	skills := map[string]dice.Ladder{
		SkillFight: dice.Mediocre,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minimum assignable rating")
}

func TestValidatePyramid_NegativeSkillValue(t *testing.T) {
	skills := map[string]dice.Ladder{
		SkillFight: dice.Poor,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minimum assignable rating")
}

// ---------------------------------------------------------------------------
// Columns mode — valid cases
// ---------------------------------------------------------------------------

func TestValidateColumns_Valid_EvenDistribution(t *testing.T) {
	// 2×Great, 2×Good, 2×Fair, 2×Average — valid columns.
	skills := map[string]dice.Ladder{
		SkillFight:     dice.Great,
		SkillShoot:     dice.Great,
		SkillAthletics: dice.Good,
		SkillNotice:    dice.Good,
		SkillStealth:   dice.Fair,
		SkillWill:      dice.Fair,
		SkillPhysique:  dice.Average,
		SkillProvoke:   dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionColumns, FateCoreSkills)
	assert.NoError(t, err)
}

func TestValidateColumns_Valid_PyramidIsAlsoValidColumns(t *testing.T) {
	// A pyramid shape also satisfies column rules.
	err := ValidateSkillDistribution(validGreatPyramid(), dice.Great, DistributionColumns, FateCoreSkills)
	assert.NoError(t, err)
}

func TestValidateColumns_Valid_WideBase(t *testing.T) {
	// 1×Great, 1×Good, 1×Fair, 5×Average — valid columns (each tier ≤ below).
	skills := map[string]dice.Ladder{
		SkillFight:       dice.Great,
		SkillShoot:       dice.Good,
		SkillAthletics:   dice.Fair,
		SkillNotice:      dice.Average,
		SkillStealth:     dice.Average,
		SkillWill:        dice.Average,
		SkillPhysique:    dice.Average,
		SkillInvestigate: dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionColumns, FateCoreSkills)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Columns mode — invalid cases
// ---------------------------------------------------------------------------

func TestValidateColumns_Invalid_MoreHigherThanLower(t *testing.T) {
	// 3×Great, 2×Good, 2×Fair, 2×Average — Great > Good, violates columns.
	skills := map[string]dice.Ladder{
		SkillFight:     dice.Great,
		SkillShoot:     dice.Great,
		SkillAthletics: dice.Great,
		SkillNotice:    dice.Good,
		SkillStealth:   dice.Good,
		SkillWill:      dice.Fair,
		SkillPhysique:  dice.Fair,
		SkillProvoke:   dice.Average,
		SkillRapport:   dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionColumns, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "columns violated")
}

func TestValidateColumns_Invalid_GapInTiers(t *testing.T) {
	// 1×Great, 0×Good, 1×Fair, 1×Average — gap at Good.
	skills := map[string]dice.Ladder{
		SkillFight:     dice.Great,
		SkillStealth:   dice.Fair,
		SkillAthletics: dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionColumns, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Good")
}

func TestValidateColumns_Invalid_NothingAtCap(t *testing.T) {
	// Skills only at Good and below, cap is Great.
	skills := map[string]dice.Ladder{
		SkillFight:   dice.Good,
		SkillShoot:   dice.Fair,
		SkillNotice:  dice.Average,
		SkillStealth: dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionColumns, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at the cap")
}

// ---------------------------------------------------------------------------
// Single column mode — valid cases
// ---------------------------------------------------------------------------

func TestValidateSingleColumn_GreatCap_Valid(t *testing.T) {
	// 1×Great, 1×Good, 1×Fair, 1×Average = 4 skills.
	skills := map[string]dice.Ladder{
		SkillFight:     dice.Great,
		SkillShoot:     dice.Good,
		SkillAthletics: dice.Fair,
		SkillNotice:    dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionSingleColumn, FateCoreSkills)
	assert.NoError(t, err)
}

func TestValidateSingleColumn_SuperbCap_Valid(t *testing.T) {
	// 1×Superb, 1×Great, 1×Good, 1×Fair, 1×Average = 5 skills.
	skills := map[string]dice.Ladder{
		SkillFight:     dice.Superb,
		SkillShoot:     dice.Great,
		SkillAthletics: dice.Good,
		SkillNotice:    dice.Fair,
		SkillStealth:   dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Superb, DistributionSingleColumn, FateCoreSkills)
	assert.NoError(t, err)
}

func TestValidateSingleColumn_AverageCap_Valid(t *testing.T) {
	// cap=Average → just 1×Average.
	skills := map[string]dice.Ladder{
		SkillFight: dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Average, DistributionSingleColumn, FateCoreSkills)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Single column mode — invalid cases
// ---------------------------------------------------------------------------

func TestValidateSingleColumn_TooManyAtOneTier(t *testing.T) {
	// 2×Great instead of 1.
	skills := map[string]dice.Ladder{
		SkillFight:     dice.Great,
		SkillShoot:     dice.Great,
		SkillAthletics: dice.Good,
		SkillNotice:    dice.Fair,
		SkillStealth:   dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionSingleColumn, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "single column")
}

func TestValidateSingleColumn_MissingTier(t *testing.T) {
	// Missing Good tier.
	skills := map[string]dice.Ladder{
		SkillFight:   dice.Great,
		SkillShoot:   dice.Fair,
		SkillNotice:  dice.Average,
		SkillStealth: dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionSingleColumn, FateCoreSkills)
	require.Error(t, err)
}

func TestValidateSingleColumn_WrongTotalCount(t *testing.T) {
	// Too few skills overall.
	skills := map[string]dice.Ladder{
		SkillFight: dice.Great,
		SkillShoot: dice.Good,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionSingleColumn, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "4 skills")
}

// ---------------------------------------------------------------------------
// Name validation
// ---------------------------------------------------------------------------

func TestValidateSkillDistribution_NilValidSkills_SkipsNameCheck(t *testing.T) {
	skills := map[string]dice.Ladder{
		"CustomMagic": dice.Great,
		"CustomCraft": dice.Good,
		"CustomLore":  dice.Good,
		"CustomFight": dice.Fair,
		"CustomShoot": dice.Fair,
		"CustomSneak": dice.Fair,
		"CustomTalk":  dice.Average,
		"CustomWill":  dice.Average,
		"CustomWatch": dice.Average,
		"CustomRun":   dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, nil)
	assert.NoError(t, err, "nil validSkills should skip name validation")
}

func TestValidateSkillDistribution_CustomSkillList(t *testing.T) {
	customSkills := []string{"Magic", "Sword", "Shield", "Dodge"}
	skills := map[string]dice.Ladder{
		"Magic":  dice.Great,
		"Sword":  dice.Good,
		"Shield": dice.Good,
		"Dodge":  dice.Fair,
		// Missing rest of pyramid — will fail shape, but name check passes.
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, customSkills)
	// Name check passes, but shape fails.
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "unknown skill")
	assert.Contains(t, err.Error(), "pyramid requires")
}

// ---------------------------------------------------------------------------
// ValidateStandardSkillPyramid convenience wrapper
// ---------------------------------------------------------------------------

func TestValidateStandardSkillPyramid_Valid(t *testing.T) {
	err := ValidateStandardSkillPyramid(validGreatPyramid())
	assert.NoError(t, err)
}

func TestValidateStandardSkillPyramid_Nil(t *testing.T) {
	err := ValidateStandardSkillPyramid(nil)
	assert.NoError(t, err)
}

func TestValidateStandardSkillPyramid_Invalid(t *testing.T) {
	skills := map[string]dice.Ladder{
		SkillFight: dice.Great,
		SkillShoot: dice.Great, // two at Great
	}
	err := ValidateStandardSkillPyramid(skills)
	require.Error(t, err)
}

func TestValidateStandardSkillPyramid_UnknownSkill(t *testing.T) {
	skills := validGreatPyramid()
	delete(skills, SkillFight)
	skills["Hacking"] = dice.Great
	err := ValidateStandardSkillPyramid(skills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

// ---------------------------------------------------------------------------
// pyramidSkillCount
// ---------------------------------------------------------------------------

func Test_pyramidSkillCount(t *testing.T) {
	tests := []struct {
		cap  dice.Ladder
		want int
	}{
		{dice.Average, 1},    // 1
		{dice.Fair, 3},       // 1+2
		{dice.Good, 6},       // 1+2+3
		{dice.Great, 10},     // 1+2+3+4
		{dice.Superb, 15},    // 1+2+3+4+5
		{dice.Fantastic, 21}, // 1+2+3+4+5+6
	}
	for _, tt := range tests {
		t.Run(tt.cap.String(), func(t *testing.T) {
			assert.Equal(t, tt.want, pyramidSkillCount(tt.cap))
		})
	}
}

// ---------------------------------------------------------------------------
// Unsupported distribution mode
// ---------------------------------------------------------------------------

func TestValidateSkillDistribution_UnsupportedMode(t *testing.T) {
	skills := map[string]dice.Ladder{SkillFight: dice.Great}
	err := ValidateSkillDistribution(skills, dice.Great, SkillDistribution(99), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestValidatePyramid_GoodCap_Valid(t *testing.T) {
	// cap=Good → 1×Good, 2×Fair, 3×Average = 6 skills.
	skills := map[string]dice.Ladder{
		SkillFight:     dice.Good,
		SkillShoot:     dice.Fair,
		SkillAthletics: dice.Fair,
		SkillNotice:    dice.Average,
		SkillStealth:   dice.Average,
		SkillWill:      dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Good, DistributionPyramid, FateCoreSkills)
	assert.NoError(t, err)
}

func TestValidateColumns_SuperbCap_Valid(t *testing.T) {
	// Verify columns work with a higher cap.
	skills := map[string]dice.Ladder{
		SkillFight:       dice.Superb,
		SkillShoot:       dice.Great,
		SkillAthletics:   dice.Great,
		SkillNotice:      dice.Good,
		SkillStealth:     dice.Good,
		SkillWill:        dice.Good,
		SkillPhysique:    dice.Fair,
		SkillProvoke:     dice.Fair,
		SkillRapport:     dice.Fair,
		SkillInvestigate: dice.Average,
		SkillDeceive:     dice.Average,
		SkillEmpathy:     dice.Average,
		SkillLore:        dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Superb, DistributionColumns, FateCoreSkills)
	assert.NoError(t, err)
}

func TestValidateColumns_ColumnsViolatedInMiddle(t *testing.T) {
	// 1×Great, 2×Good, 1×Fair, 2×Average — Good(2) > Fair(1) violates columns.
	skills := map[string]dice.Ladder{
		SkillFight:     dice.Great,
		SkillShoot:     dice.Good,
		SkillAthletics: dice.Good,
		SkillNotice:    dice.Fair,
		SkillStealth:   dice.Average,
		SkillWill:      dice.Average,
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionColumns, FateCoreSkills)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "columns violated")
}

// ---------------------------------------------------------------------------
// Error accumulation — multiple errors returned together
// ---------------------------------------------------------------------------

func TestValidateSkillDistribution_AccumulatesMultipleErrors(t *testing.T) {
	// Mix of unknown skill, above cap, and below average — all reported at once.
	skills := map[string]dice.Ladder{
		"Hacking":  dice.Great,    // unknown
		SkillFight: dice.Superb,   // above Great cap
		SkillShoot: dice.Mediocre, // below Average
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "unknown skill")
	assert.Contains(t, msg, "Hacking")
	assert.Contains(t, msg, "exceeds the skill cap")
	assert.Contains(t, msg, "minimum assignable rating")
}

func TestValidateSkillDistribution_AccumulatesUnknownAndAboveCap(t *testing.T) {
	skills := map[string]dice.Ladder{
		"Piloting": dice.Good,
		SkillFight: dice.Superb, // above Great cap
	}
	err := ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "Piloting")
	assert.Contains(t, msg, "exceeds the skill cap")
}
