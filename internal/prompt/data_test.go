package prompt

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
)

func TestComputeDifficultyGuidance_TypicalSkills(t *testing.T) {
	skills := map[string]dice.Ladder{
		"Athletics": dice.Good,    // +3
		"Fight":     dice.Fair,    // +2
		"Notice":    dice.Average, // +1
	}

	g := ComputeDifficultyGuidance(skills)

	assert.Equal(t, int(dice.Average), g.DifficultyMin, "min should be Average (+1)")
	assert.Equal(t, int(dice.Fair), g.DifficultyDefault, "default should be Fair (+2)")
	assert.Equal(t, int(dice.Superb), g.DifficultyMax, "max should be peak+2 = Superb (+5)")
	assert.Contains(t, g.DifficultyGuide, "1=easy")
}

func TestComputeDifficultyGuidance_NoSkills(t *testing.T) {
	g := ComputeDifficultyGuidance(nil)

	assert.Equal(t, int(dice.Average), g.DifficultyMin)
	assert.Equal(t, int(dice.Fair), g.DifficultyDefault)
	// Mediocre(0)+2=2 but floor is Good(+3)
	assert.Equal(t, int(dice.Good), g.DifficultyMax, "should floor at Good (+3)")
}

func TestComputeDifficultyGuidance_HighSkillsCapped(t *testing.T) {
	skills := map[string]dice.Ladder{
		"Lore": dice.Legendary, // +8
	}

	g := ComputeDifficultyGuidance(skills)

	// 8+2=10 but capped at Fantastic (+6)
	assert.Equal(t, int(dice.Fantastic), g.DifficultyMax, "should cap at Fantastic (+6)")
}

func TestComputeDifficultyGuidance_LowSkillsFloored(t *testing.T) {
	skills := map[string]dice.Ladder{
		"Notice": dice.Average, // +1
	}

	g := ComputeDifficultyGuidance(skills)

	// 1+2=3 = Good, which is the floor anyway
	assert.Equal(t, int(dice.Good), g.DifficultyMax, "should floor at Good (+3)")
}
