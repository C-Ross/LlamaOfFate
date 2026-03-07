package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
)

// SkillDistribution controls which skill arrangement shape is considered valid.
// See: https://fate-srd.com/fate-core/skills
type SkillDistribution int

const (
	// DistributionPyramid requires each tier to have exactly one more skill
	// than the tier above. This is the default Fate Core character creation mode.
	// For cap=Great(+4): 1×Great, 2×Good, 3×Fair, 4×Average = 10 skills.
	// See: https://fate-srd.com/fate-core/skills
	DistributionPyramid SkillDistribution = iota

	// DistributionColumns requires that no tier has more skills than the tier
	// below it (i.e., count at rank N ≤ count at rank N−1). This is the looser
	// rule used during advancement and as an alternative to the pyramid.
	// See: https://fate-srd.com/fate-core/advancement-change#skill-columns
	DistributionColumns

	// DistributionSingleColumn requires exactly one skill at each tier from
	// the cap down to Average(+1). Used for supporting NPCs.
	// See: https://fate-srd.com/fate-core/creating-and-playing-opposition#supporting-npcs
	DistributionSingleColumn
)

// String implements fmt.Stringer, returning a human-readable label for the
// distribution mode. Used in error messages and logging.
func (d SkillDistribution) String() string {
	switch d {
	case DistributionPyramid:
		return "pyramid"
	case DistributionColumns:
		return "columns"
	case DistributionSingleColumn:
		return "single column"
	default:
		return fmt.Sprintf("SkillDistribution(%d)", int(d))
	}
}

// ValidateSkillDistribution checks that skills form a valid Fate Core
// arrangement according to the given mode and skill cap.
//
// All per-skill errors (unknown names, out-of-range ratings) are accumulated
// and returned together to avoid iterative fix-and-retry cycles.
// Shape validation runs only when individual skills are all valid.
//
// Parameters:
//   - skills: map of skill name → ladder value (e.g. "Fight" → dice.Great).
//     A nil or empty map is always valid (means "use genre defaults").
//   - cap: highest allowed ladder value (e.g. dice.Great for standard games,
//     dice.Superb for superhero games, dice.Fantastic for beyond-human).
//   - mode: Pyramid, Columns, or SingleColumn.
//   - validSkills: allowlist of skill names. Pass nil to skip name validation.
//
// Returns nil if valid, or a combined error containing all violations.
func ValidateSkillDistribution(skills map[string]dice.Ladder, cap dice.Ladder, mode SkillDistribution, validSkills []string) error {
	if len(skills) == 0 {
		return nil
	}

	if cap < dice.Average {
		return fmt.Errorf("skill cap %s must be at least Average (+1)", cap)
	}

	// Build allowlist set if provided.
	var allowSet map[string]bool
	if validSkills != nil {
		allowSet = make(map[string]bool, len(validSkills))
		for _, s := range validSkills {
			allowSet[s] = true
		}
	}

	// Validate each skill entry and bucket by tier.
	// Errors are accumulated so callers see all problems at once.
	tierCounts := make(map[dice.Ladder]int)
	var errs []error
	var unknownSkills []string

	for name, level := range skills {
		if allowSet != nil && !allowSet[name] {
			unknownSkills = append(unknownSkills, name)
			continue
		}
		if level < dice.Average {
			errs = append(errs, fmt.Errorf("skill %q rated at %s: minimum assignable rating is Average (+1)", name, level))
			continue
		}
		if level > cap {
			errs = append(errs, fmt.Errorf("skill %q rated at %s exceeds the skill cap of %s", name, level, cap))
			continue
		}
		tierCounts[level]++
	}

	if len(unknownSkills) > 0 {
		sort.Strings(unknownSkills)
		errs = append(errs, fmt.Errorf("unknown skill(s): %s", strings.Join(unknownSkills, ", ")))
	}

	// Shape validation only makes sense when all individual skills are valid,
	// because invalid entries are excluded from tierCounts.
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// Validate the shape according to the chosen distribution mode.
	switch mode {
	case DistributionPyramid:
		return validatePyramidShape(tierCounts, cap)
	case DistributionColumns:
		return validateColumnsShape(tierCounts, cap)
	case DistributionSingleColumn:
		return validateSingleColumnShape(tierCounts, cap)
	default:
		return fmt.Errorf("unsupported skill distribution mode: %s", mode)
	}
}

// validatePyramidShape requires each tier to have exactly (cap − rank + 1) skills.
// E.g., cap=Great(+4): 1×Great, 2×Good, 3×Fair, 4×Average.
func validatePyramidShape(tierCounts map[dice.Ladder]int, cap dice.Ladder) error {
	for rank := cap; rank >= dice.Average; rank-- {
		expected := int(cap-rank) + 1
		got := tierCounts[rank]
		if got != expected {
			return fmt.Errorf("pyramid requires %d skill(s) at %s, got %d", expected, rank, got)
		}
	}
	return nil
}

// validateColumnsShape requires that no tier has more skills than the tier
// below it. The bottom tier (Average) has no lower bound on count.
func validateColumnsShape(tierCounts map[dice.Ladder]int, cap dice.Ladder) error {
	// Must have at least one skill at the cap.
	if tierCounts[cap] == 0 {
		return fmt.Errorf("columns require at least one skill at the cap (%s)", cap)
	}

	// Every tier from cap down to Average must be populated.
	for rank := cap; rank >= dice.Average; rank-- {
		if tierCounts[rank] == 0 {
			return fmt.Errorf("columns require at least one skill at %s (no gaps allowed)", rank)
		}
	}

	// Each tier must have count ≤ the tier below it.
	for rank := cap; rank > dice.Average; rank-- {
		upper := tierCounts[rank]
		lower := tierCounts[rank-1]
		if upper > lower {
			return fmt.Errorf("columns violated: %d skill(s) at %s but only %d at %s below",
				upper, rank, lower, rank-1)
		}
	}

	return nil
}

// validateSingleColumnShape requires exactly one skill at each tier from
// cap down to Average.
func validateSingleColumnShape(tierCounts map[dice.Ladder]int, cap dice.Ladder) error {
	expectedTiers := int(cap - dice.Average + 1)
	totalSkills := 0
	for _, c := range tierCounts {
		totalSkills += c
	}
	if totalSkills != expectedTiers {
		return fmt.Errorf("single column requires exactly %d skills (one per tier from %s to Average), got %d",
			expectedTiers, cap, totalSkills)
	}

	for rank := cap; rank >= dice.Average; rank-- {
		got := tierCounts[rank]
		if got != 1 {
			return fmt.Errorf("single column requires exactly 1 skill at %s, got %d", rank, got)
		}
	}
	return nil
}

// ValidateStandardSkillPyramid is a convenience wrapper that validates skills
// as a standard Fate Core pyramid (cap=Great, pyramid mode, canonical skill
// names). A nil or empty map is valid (use genre defaults).
func ValidateStandardSkillPyramid(skills map[string]dice.Ladder) error {
	return ValidateSkillDistribution(skills, dice.Great, DistributionPyramid, FateCoreSkills)
}

// pyramidSkillCount returns the total number of skills in a pyramid with the
// given cap. The formula is the triangular number: cap*(cap+1)/2 when cap is
// expressed as tiers above Mediocre.
// E.g., Great(+4) → 1+2+3+4 = 10, Superb(+5) → 15, Fantastic(+6) → 21.
func pyramidSkillCount(cap dice.Ladder) int {
	tiers := int(cap - dice.Average + 1)
	count := 0
	for i := range tiers {
		count += i + 1
	}
	return count
}
