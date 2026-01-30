package core

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
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
		want        character.StressTrackType
	}{
		// Physical attacks -> Physical stress
		{"Fight targets physical", "Fight", character.PhysicalStress},
		{"Shoot targets physical", "Shoot", character.PhysicalStress},
		{"Physique targets physical", "Physique", character.PhysicalStress},

		// Mental attacks -> Mental stress
		{"Provoke targets mental", "Provoke", character.MentalStress},
		{"Deceive targets mental", "Deceive", character.MentalStress},
		{"Rapport targets mental", "Rapport", character.MentalStress},
		{"Lore targets mental", "Lore", character.MentalStress},

		// Unknown defaults to Physical
		{"Unknown skill defaults to physical", "Crafts", character.PhysicalStress},
		{"Empty string defaults to physical", "", character.PhysicalStress},
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
