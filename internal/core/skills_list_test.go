package core

import (
	"testing"

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
