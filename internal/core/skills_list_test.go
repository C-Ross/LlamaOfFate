package core

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFateCoreSkills_Count(t *testing.T) {
	require.Len(t, FateCoreSkills, 18, "Fate Core defines exactly 18 default skills")
}

func TestFateCoreSkills_Sorted(t *testing.T) {
	for i := 1; i < len(FateCoreSkills); i++ {
		assert.Less(t, FateCoreSkills[i-1], FateCoreSkills[i],
			"FateCoreSkills must be alphabetically sorted: %q should come before %q",
			FateCoreSkills[i-1], FateCoreSkills[i])
	}
}

func TestFateCoreSkills_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool, len(FateCoreSkills))
	for _, s := range FateCoreSkills {
		assert.False(t, seen[s], "duplicate skill: %q", s)
		seen[s] = true
	}
}

func TestIsValidSkill(t *testing.T) {
	tests := []struct {
		name  string
		skill string
		want  bool
	}{
		{"valid — Athletics", "Athletics", true},
		{"valid — Will", "Will", true},
		{"valid — Shoot", "Shoot", true},
		{"invalid — lowercase", "athletics", false},
		{"invalid — empty", "", false},
		{"invalid — nonsense", "Swordfighting", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsValidSkill(tt.skill))
		})
	}
}

func TestFateCoreSkills_ContainsAllExpected(t *testing.T) {
	expected := []string{
		"Athletics", "Burglary", "Contacts", "Crafts",
		"Deceive", "Drive", "Empathy", "Fight",
		"Investigate", "Lore", "Notice", "Physique",
		"Provoke", "Rapport", "Resources", "Shoot",
		"Stealth", "Will",
	}
	for _, skill := range expected {
		assert.True(t, IsValidSkill(skill), "expected skill %q to be valid", skill)
	}
}

func TestDefaultSkillPriority_Count(t *testing.T) {
	assert.Len(t, DefaultSkillPriority, 10, "default priority should list 10 skills")
}

func TestDefaultSkillPriority_AllValid(t *testing.T) {
	for _, skill := range DefaultSkillPriority {
		assert.True(t, IsValidSkill(skill), "priority skill %q should be valid", skill)
	}
}

func TestDefaultSkillPriority_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, skill := range DefaultSkillPriority {
		assert.False(t, seen[skill], "duplicate skill %q in DefaultSkillPriority", skill)
		seen[skill] = true
	}
}

func TestDefaultPyramid_Shape(t *testing.T) {
	pyramid := DefaultPyramid()
	require.Len(t, pyramid, 10)

	// Count tiers
	counts := make(map[dice.Ladder]int)
	for _, level := range pyramid {
		counts[level]++
	}
	assert.Equal(t, 1, counts[dice.Great], "1 Great skill")
	assert.Equal(t, 2, counts[dice.Good], "2 Good skills")
	assert.Equal(t, 3, counts[dice.Fair], "3 Fair skills")
	assert.Equal(t, 4, counts[dice.Average], "4 Average skills")
}

func TestDefaultPyramid_PassesValidation(t *testing.T) {
	pyramid := DefaultPyramid()
	err := ValidateStandardSkillPyramid(pyramid)
	assert.NoError(t, err)
}

func TestDefaultPyramid_ReturnsFreshMap(t *testing.T) {
	p1 := DefaultPyramid()
	p2 := DefaultPyramid()
	p1["Notice"] = dice.Average // mutate first
	assert.Equal(t, dice.Great, p2["Notice"], "mutation should not leak between calls")
}
