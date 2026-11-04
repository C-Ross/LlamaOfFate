package terminal

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptForCharacterSetup_AllDefaults(t *testing.T) {
	// Player presses Enter for everything → keeps all preset values.
	// Identity: 3 Enters, Aspects: 3 Enters, Skill choice: Enter (keep preset)
	ui := newUIWithInput("\n\n\n\n\n\n\n")

	preset := core.NewCharacter("p1", "Jesse Calhoun")
	preset.Aspects.HighConcept = "Quick-Draw Gunslinger"
	preset.Aspects.Trouble = "Wanted Dead or Alive"
	preset.Aspects.AddAspect("Old Partner's Badge")
	preset.SetSkill("Shoot", dice.Great)

	setup := ui.PromptForCharacterSetup(preset)

	assert.Equal(t, "Jesse Calhoun", setup.Name)
	assert.Equal(t, "Quick-Draw Gunslinger", setup.HighConcept)
	assert.Equal(t, "Wanted Dead or Alive", setup.Trouble)
	assert.Equal(t, []string{"Old Partner's Badge"}, setup.Aspects)
	assert.Equal(t, dice.Great, setup.Skills["Shoot"])
}

func TestPromptForCharacterSetup_CustomIdentity(t *testing.T) {
	// Player overrides name and high concept, keeps trouble and aspects.
	ui := newUIWithInput("Seraphina\nBattle-Scarred Mage\n\n\n\n\n\n")

	preset := core.NewCharacter("p1", "Default")
	preset.Aspects.HighConcept = "Boring Hero"
	preset.Aspects.Trouble = "Default Trouble"
	preset.SetSkill("Fight", dice.Good)

	setup := ui.PromptForCharacterSetup(preset)

	assert.Equal(t, "Seraphina", setup.Name)
	assert.Equal(t, "Battle-Scarred Mage", setup.HighConcept)
	assert.Equal(t, "Default Trouble", setup.Trouble)
}

func TestPromptForCharacterSetup_CustomAspects(t *testing.T) {
	// Player enters custom aspects, skips the third.
	ui := newUIWithInput("\n\n\nSworn Protector\nOld Debts\n\n\n")

	preset := core.NewCharacter("p1", "Hero")
	preset.Aspects.HighConcept = "HC"
	preset.Aspects.Trouble = "T"
	preset.SetSkill("Fight", dice.Good)

	setup := ui.PromptForCharacterSetup(preset)

	assert.Equal(t, []string{"Sworn Protector", "Old Debts"}, setup.Aspects)
}

func TestPromptForCharacterSetup_UseDefaultPyramid(t *testing.T) {
	// Identity: 3 Enters, Aspects: 3 Enters, Skill choice: "2" (use defaults)
	ui := newUIWithInput("\n\n\n\n\n\n2\n")

	preset := core.NewCharacter("p1", "Hero")
	preset.Aspects.HighConcept = "HC"
	preset.Aspects.Trouble = "T"

	setup := ui.PromptForCharacterSetup(preset)

	defaultPyramid := core.DefaultPyramid()
	assert.Equal(t, defaultPyramid, setup.Skills)
}

func TestPromptForSkillPyramid_KeepPreset(t *testing.T) {
	ui := newUIWithInput("\n") // Enter = choice 1 = keep preset

	presetSkills := map[string]dice.Ladder{
		"Shoot": dice.Great,
		"Fight": dice.Good,
	}

	result := ui.promptForSkillPyramid(presetSkills)

	assert.Equal(t, presetSkills, result)
}

func TestPromptForSkillPyramid_UseDefaults(t *testing.T) {
	ui := newUIWithInput("2\n")

	result := ui.promptForSkillPyramid(map[string]dice.Ladder{"Shoot": dice.Great})

	expected := core.DefaultPyramid()
	assert.Equal(t, expected, result)
}

func TestPromptForSkillPyramid_NoPresetFallsToDefault(t *testing.T) {
	// Enter with empty preset → should use DefaultPyramid.
	ui := newUIWithInput("\n")

	result := ui.promptForSkillPyramid(nil)

	expected := core.DefaultPyramid()
	assert.Equal(t, expected, result)
}

func TestAssignSkillsManually_FullInput(t *testing.T) {
	// Pick skill #1 (Athletics) for Great,
	// then #1 (Burglary), #1 (Contacts) for Good (after Athletics is used),
	// then #1 (Crafts), #1 (Deceive), #1 (Drive) for Fair,
	// then #1 (Empathy), #1 (Fight), #1 (Investigate), #1 (Lore) for Average.
	ui := newUIWithInput("1\n1\n1\n1\n1\n1\n1\n1\n1\n1\n")

	skills := ui.assignSkillsManually()

	require.Len(t, skills, 10)
	// Athletics = first alphabetically → Great
	assert.Equal(t, dice.Great, skills["Athletics"])
	// Burglary, Contacts → Good
	assert.Equal(t, dice.Good, skills["Burglary"])
	assert.Equal(t, dice.Good, skills["Contacts"])
}

func TestAssignSkillsManually_AutoFillOnEnter(t *testing.T) {
	// Press Enter at every tier → auto-fill from available skills
	ui := newUIWithInput("\n\n\n\n")

	skills := ui.assignSkillsManually()

	require.Len(t, skills, 10)
	// All 10 skills should be assigned
	total := 0
	for _, level := range skills {
		assert.True(t, level >= dice.Average && level <= dice.Great)
		total++
	}
	assert.Equal(t, 10, total)
}

func TestPromptWithDefault_ReturnsDefault(t *testing.T) {
	ui := newUIWithInput("\n")

	result := ui.promptWithDefault("Name", "Jesse")

	assert.Equal(t, "Jesse", result)
}

func TestPromptWithDefault_ReturnsCustom(t *testing.T) {
	ui := newUIWithInput("Custom Name\n")

	result := ui.promptWithDefault("Name", "Jesse")

	assert.Equal(t, "Custom Name", result)
}

func TestPromptWithDefault_EmptyDefault(t *testing.T) {
	ui := newUIWithInput("\n")

	result := ui.promptWithDefault("Name", "")

	assert.Equal(t, "", result)
}

func TestPrintSkillSummary_DoesNotPanic(t *testing.T) {
	ui := NewTerminalUI()
	skills := map[string]dice.Ladder{
		"Notice":    dice.Great,
		"Athletics": dice.Good,
		"Will":      dice.Good,
		"Fight":     dice.Fair,
	}

	require.NotPanics(t, func() {
		ui.printSkillSummary(skills)
	})
}

func TestAvailableSkills_ExcludesUsed(t *testing.T) {
	ui := NewTerminalUI()
	used := map[string]bool{
		"Athletics": true,
		"Fight":     true,
	}

	avail := ui.availableSkills(used)

	assert.NotContains(t, avail, "Athletics")
	assert.NotContains(t, avail, "Fight")
	assert.Contains(t, avail, "Notice")
	assert.Len(t, avail, 16) // 18 - 2 = 16
}
