package core

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/stretchr/testify/assert"
)

func TestDefenseSkillForAttack(t *testing.T) {
	tests := []struct {
		name        string
		attackSkill string
		want        string
	}{
		// Physical attacks -> Athletics defense
		{"Fight uses Athletics", "Fight", "Athletics"},
		{"Shoot uses Athletics", "Shoot", "Athletics"},
		{"Physique uses Athletics", "Physique", "Athletics"},

		// Mental attacks -> Will defense
		{"Provoke uses Will", "Provoke", "Will"},
		{"Deceive uses Will", "Deceive", "Will"},
		{"Rapport uses Will", "Rapport", "Will"},
		{"Lore uses Will", "Lore", "Will"},

		// Unknown defaults to Athletics
		{"Unknown skill defaults to Athletics", "Crafts", "Athletics"},
		{"Empty string defaults to Athletics", "", "Athletics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefenseSkillForAttack(tt.attackSkill)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStressTypeForAttack(t *testing.T) {
	tests := []struct {
		name        string
		attackSkill string
		want        StressTrackType
	}{
		// Physical attacks -> Physical stress
		{"Fight targets physical", "Fight", PhysicalStress},
		{"Shoot targets physical", "Shoot", PhysicalStress},
		{"Physique targets physical", "Physique", PhysicalStress},

		// Mental attacks -> Mental stress
		{"Provoke targets mental", "Provoke", MentalStress},
		{"Deceive targets mental", "Deceive", MentalStress},
		{"Rapport targets mental", "Rapport", MentalStress},
		{"Lore targets mental", "Lore", MentalStress},

		// Unknown defaults to Physical
		{"Unknown skill defaults to physical", "Crafts", PhysicalStress},
		{"Empty string defaults to physical", "", PhysicalStress},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StressTypeForAttack(tt.attackSkill)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConflictTypeForSkill(t *testing.T) {
	tests := []struct {
		name  string
		skill string
		want  scene.ConflictType
	}{
		// Physical skills -> Physical conflict
		{"Fight is physical", "Fight", scene.PhysicalConflict},
		{"Shoot is physical", "Shoot", scene.PhysicalConflict},
		{"Athletics is physical", "Athletics", scene.PhysicalConflict},
		{"Physique is physical", "Physique", scene.PhysicalConflict},

		// Mental skills -> Mental conflict
		{"Provoke is mental", "Provoke", scene.MentalConflict},
		{"Deceive is mental", "Deceive", scene.MentalConflict},
		{"Rapport is mental", "Rapport", scene.MentalConflict},
		{"Will is mental", "Will", scene.MentalConflict},
		{"Empathy is mental", "Empathy", scene.MentalConflict},

		// Unknown defaults to Physical
		{"Unknown skill defaults to physical", "Crafts", scene.PhysicalConflict},
		{"Empty string defaults to physical", "", scene.PhysicalConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConflictTypeForSkill(tt.skill)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsPhysicalAttackSkill(t *testing.T) {
	assert.True(t, IsPhysicalAttackSkill("Fight"))
	assert.True(t, IsPhysicalAttackSkill("Shoot"))
	assert.True(t, IsPhysicalAttackSkill("Physique"))
	assert.False(t, IsPhysicalAttackSkill("Provoke"))
	assert.False(t, IsPhysicalAttackSkill("Lore"))
	assert.False(t, IsPhysicalAttackSkill("Unknown"))
}

func TestIsMentalAttackSkill(t *testing.T) {
	assert.True(t, IsMentalAttackSkill("Provoke"))
	assert.True(t, IsMentalAttackSkill("Deceive"))
	assert.True(t, IsMentalAttackSkill("Rapport"))
	assert.True(t, IsMentalAttackSkill("Lore"))
	assert.False(t, IsMentalAttackSkill("Fight"))
	assert.False(t, IsMentalAttackSkill("Shoot"))
	assert.False(t, IsMentalAttackSkill("Unknown"))
}

func TestInitiativeSkillsForConflict(t *testing.T) {
	// Physical conflicts use Notice, then Athletics
	physSkills := InitiativeSkillsForConflict(scene.PhysicalConflict)
	assert.Equal(t, []string{"Notice", "Athletics"}, physSkills)

	// Mental conflicts use Empathy, then Rapport
	mentalSkills := InitiativeSkillsForConflict(scene.MentalConflict)
	assert.Equal(t, []string{"Empathy", "Rapport"}, mentalSkills)
}

func TestDefaultAttackSkillForConflict(t *testing.T) {
	// Physical conflicts default to Fight
	assert.Equal(t, "Fight", DefaultAttackSkillForConflict(scene.PhysicalConflict))

	// Mental conflicts default to Provoke
	assert.Equal(t, "Provoke", DefaultAttackSkillForConflict(scene.MentalConflict))
}

func TestCalculateInitiative(t *testing.T) {
	// Physical conflict - uses Notice primarily
	char := NewCharacter("test-1", "Test Character")
	char.SetSkill("Notice", 3)
	char.SetSkill("Athletics", 2)
	assert.Equal(t, 3, CalculateInitiative(char, scene.PhysicalConflict))

	// Physical conflict - falls back to Athletics when Notice is 0
	char2 := NewCharacter("test-2", "Test Character 2")
	char2.SetSkill("Athletics", 4)
	assert.Equal(t, 4, CalculateInitiative(char2, scene.PhysicalConflict))

	// Mental conflict - uses Empathy primarily
	char3 := NewCharacter("test-3", "Test Character 3")
	char3.SetSkill("Empathy", 5)
	char3.SetSkill("Rapport", 2)
	assert.Equal(t, 5, CalculateInitiative(char3, scene.MentalConflict))

	// Mental conflict - falls back to Rapport when Empathy is 0
	char4 := NewCharacter("test-4", "Test Character 4")
	char4.SetSkill("Rapport", 3)
	assert.Equal(t, 3, CalculateInitiative(char4, scene.MentalConflict))

	// Returns 0 when no relevant skills
	char5 := NewCharacter("test-5", "Test Character 5")
	assert.Equal(t, 0, CalculateInitiative(char5, scene.PhysicalConflict))
}

func TestConcessionFatePoints(t *testing.T) {
	tests := []struct {
		name             string
		consequenceCount int
		expected         int
	}{
		{"no consequences grants 1 FP", 0, 1},
		{"one consequence grants 2 FP", 1, 2},
		{"two consequences grants 3 FP", 2, 3},
		{"three consequences grants 4 FP", 3, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConcessionFatePoints(tt.consequenceCount)
			assert.Equal(t, tt.expected, got)
		})
	}
}
