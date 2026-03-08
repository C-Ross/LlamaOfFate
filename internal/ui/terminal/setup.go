package terminal

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
)

// CharacterSetup holds the player's character creation choices.
type CharacterSetup struct {
	Name        string
	HighConcept string
	Trouble     string
	Aspects     []string
	Skills      map[string]dice.Ladder
}

// PromptForCharacterSetup walks the player through full character creation:
// identity (name, high concept, trouble), additional aspects, and skill pyramid.
// The preset is used as a starting point — pressing Enter at any prompt
// keeps the preset value.
func (ui *TerminalUI) PromptForCharacterSetup(preset *core.Character) CharacterSetup {
	fmt.Println("\n=== Character Creation ===")

	// --- Step 1: Identity ---
	fmt.Println("\n--- Identity ---")

	name := ui.promptWithDefault("Name", preset.Name)
	highConcept := ui.promptWithDefault("High Concept", preset.Aspects.HighConcept)
	trouble := ui.promptWithDefault("Trouble", preset.Aspects.Trouble)

	// --- Step 2: Additional Aspects ---
	fmt.Println("\n--- Additional Aspects ---")
	fmt.Println("Enter up to 3 additional aspects (press Enter to skip each).")

	presetAspects := preset.Aspects.OtherAspects
	aspects := make([]string, 0, 3)
	for i := range 3 {
		defaultVal := ""
		if i < len(presetAspects) {
			defaultVal = presetAspects[i]
		}
		label := fmt.Sprintf("Aspect %d", i+1)
		val := ui.promptWithDefault(label, defaultVal)
		if val != "" {
			aspects = append(aspects, val)
		}
	}

	// --- Step 3: Skill Pyramid ---
	skills := ui.promptForSkillPyramid(preset.Skills)

	return CharacterSetup{
		Name:        name,
		HighConcept: highConcept,
		Trouble:     trouble,
		Aspects:     aspects,
		Skills:      skills,
	}
}

// promptWithDefault reads a line, showing the default in brackets.
// If the user presses Enter, the default is returned.
func (ui *TerminalUI) promptWithDefault(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}

	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

// promptForSkillPyramid prompts the player to build a skill pyramid.
// Players can accept defaults, assign skills tier by tier, or use the
// preset's existing skills.
func (ui *TerminalUI) promptForSkillPyramid(presetSkills map[string]dice.Ladder) map[string]dice.Ladder {
	fmt.Println("\n--- Skill Pyramid ---")
	fmt.Println("Standard pyramid: 1 Great (+4), 2 Good (+3), 3 Fair (+2), 4 Average (+1)")

	// Show current skills
	if len(presetSkills) > 0 {
		fmt.Println("\nPreset skills:")
		ui.printSkillSummary(presetSkills)
	}

	fmt.Println("\nOptions:")
	fmt.Println("  [1] Keep preset skills")
	fmt.Println("  [2] Use default pyramid")
	fmt.Println("  [3] Assign skills manually")
	fmt.Print("Choice [1]: ")

	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return presetSkills
	}
	input = strings.TrimSpace(input)

	switch input {
	case "", "1":
		if len(presetSkills) > 0 {
			return presetSkills
		}
		// No preset skills — fall through to defaults
		fmt.Println("No preset skills available, using defaults.")
		return core.DefaultPyramid()
	case "2":
		skills := core.DefaultPyramid()
		fmt.Println("\nDefault pyramid assigned:")
		ui.printSkillSummary(skills)
		return skills
	case "3":
		return ui.assignSkillsManually()
	default:
		fmt.Println("Invalid choice, keeping preset skills.")
		if len(presetSkills) > 0 {
			return presetSkills
		}
		return core.DefaultPyramid()
	}
}

// assignSkillsManually lets the player pick skills for each tier of the pyramid.
func (ui *TerminalUI) assignSkillsManually() map[string]dice.Ladder {
	type tier struct {
		level dice.Ladder
		count int
		label string
	}
	tiers := []tier{
		{dice.Great, 1, "Great (+4)"},
		{dice.Good, 2, "Good (+3)"},
		{dice.Fair, 3, "Fair (+2)"},
		{dice.Average, 4, "Average (+1)"},
	}

	skills := make(map[string]dice.Ladder, 10)
	used := make(map[string]bool)

	for _, t := range tiers {
		fmt.Printf("\n--- %s: pick %d skill(s) ---\n", t.label, t.count)
		available := ui.availableSkills(used)
		for i, s := range available {
			fmt.Printf("  %2d. %s\n", i+1, s)
		}

		for picked := 0; picked < t.count; {
			remaining := t.count - picked
			fmt.Printf("Enter skill number (%d remaining): ", remaining)

			input, err := ui.reader.ReadString('\n')
			if err != nil {
				// On error, fill remaining slots from available
				ui.autoFillRemaining(skills, used, t.level, remaining)
				picked = t.count
				continue
			}
			input = strings.TrimSpace(input)
			if input == "" {
				// Auto-fill remaining slots from available skills
				ui.autoFillRemaining(skills, used, t.level, remaining)
				picked = t.count
				continue
			}

			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(available) {
				fmt.Println("Invalid choice, try again.")
				continue
			}

			skill := available[num-1]
			if used[skill] {
				fmt.Printf("%s is already assigned, try again.\n", skill)
				continue
			}

			skills[skill] = t.level
			used[skill] = true
			picked++
			fmt.Printf("  ✓ %s → %s\n", skill, t.level)

			// Refresh available list for next pick in same tier
			available = ui.availableSkills(used)
		}
	}

	fmt.Println("\nYour skill pyramid:")
	ui.printSkillSummary(skills)
	return skills
}

// availableSkills returns Fate Core skills not yet used, sorted alphabetically.
func (ui *TerminalUI) availableSkills(used map[string]bool) []string {
	var avail []string
	for _, s := range core.FateCoreSkills {
		if !used[s] {
			avail = append(avail, s)
		}
	}
	return avail
}

// autoFillRemaining assigns the first N available skills to the given level.
func (ui *TerminalUI) autoFillRemaining(skills map[string]dice.Ladder, used map[string]bool, level dice.Ladder, count int) {
	avail := ui.availableSkills(used)
	for i := range count {
		if i >= len(avail) {
			break
		}
		skills[avail[i]] = level
		used[avail[i]] = true
		fmt.Printf("  (auto) %s → %s\n", avail[i], level)
	}
}

// printSkillSummary prints skills grouped by tier, highest first.
func (ui *TerminalUI) printSkillSummary(skills map[string]dice.Ladder) {
	grouped := make(map[dice.Ladder][]string)
	for name, level := range skills {
		grouped[level] = append(grouped[level], name)
	}

	// Sort tiers descending
	var levels []dice.Ladder
	for level := range grouped {
		levels = append(levels, level)
	}
	sort.Slice(levels, func(i, j int) bool { return levels[i] > levels[j] })

	for _, level := range levels {
		names := grouped[level]
		sort.Strings(names)
		fmt.Printf("  %s: %s\n", level, strings.Join(names, ", "))
	}
}
