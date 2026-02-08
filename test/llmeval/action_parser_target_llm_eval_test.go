//go:build llmeval

package llmeval_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TargetTestCase represents a test case that validates target resolution
type TargetTestCase struct {
	Name            string
	RawInput        string
	Context         string
	OtherCharacters []*character.Character
	ExpectedType    action.ActionType
	ExpectedSkills  []string
	// ValidTargets lists acceptable target values. The LLM should return one of these.
	// Use character IDs (preferred) or names. Empty slice means target should be empty.
	ValidTargets []string
	Description  string
}

// TargetEvalResult stores the result of a targeting eval
type TargetEvalResult struct {
	TestCase        TargetTestCase
	ActualType      action.ActionType
	ActualSkill     string
	ActualTarget    string
	TypeMatches     bool
	SkillAcceptable bool
	TargetResolved  bool // Whether the target can be resolved via Engine.ResolveCharacter
	TargetIsID      bool // Whether the target is a clean character ID (best case)
	Error           error
}

// getSceneNPCs creates a set of NPCs for targeting tests
func getSceneNPCs() []*character.Character {
	bandit := character.NewCharacter("scene_2_npc_0", "Bandit Leader")
	bandit.Aspects.HighConcept = "Ruthless Outlaw Boss"
	bandit.Aspects.Trouble = "Wanted Dead or Alive"
	bandit.SetSkill("Fight", dice.Good)
	bandit.SetSkill("Shoot", dice.Fair)
	bandit.SetSkill("Provoke", dice.Fair)

	scout := character.NewCharacter("scene_4_npc_1", "Outlaw Scout")
	scout.Aspects.HighConcept = "Sharp-Eyed Canyon Watcher"
	scout.Aspects.Trouble = "Jumpy and Paranoid"
	scout.SetSkill("Shoot", dice.Good)
	scout.SetSkill("Notice", dice.Good)
	scout.SetSkill("Stealth", dice.Fair)

	sheriff := character.NewCharacter("scene_1_npc_0", "Sheriff Morgan")
	sheriff.Aspects.HighConcept = "Weary Lawman Past His Prime"
	sheriff.Aspects.Trouble = "Too Old for This"
	sheriff.SetSkill("Shoot", dice.Fair)
	sheriff.SetSkill("Rapport", dice.Good)
	sheriff.SetSkill("Notice", dice.Fair)

	return []*character.Character{bandit, scout, sheriff}
}

// getAttackTargetTestCases returns test cases where the player attacks a specific NPC
func getAttackTargetTestCases() []TargetTestCase {
	npcs := getSceneNPCs()
	return []TargetTestCase{
		{
			Name:            "Shoot named NPC",
			RawInput:        "I shoot the Outlaw Scout",
			Context:         "In a canyon ambush, the Outlaw Scout is firing from behind a boulder",
			OtherCharacters: npcs,
			ExpectedType:    action.Attack,
			ExpectedSkills:  []string{"Shoot"},
			ValidTargets:    []string{"scene_4_npc_1", "Outlaw Scout"},
			Description:     "Direct attack on a named NPC should target their ID",
		},
		{
			Name:            "Attack bandit leader with fists",
			RawInput:        "I punch the Bandit Leader in the face",
			Context:         "Face to face with the Bandit Leader in the hideout",
			OtherCharacters: npcs,
			ExpectedType:    action.Attack,
			ExpectedSkills:  []string{"Fight"},
			ValidTargets:    []string{"scene_2_npc_0", "Bandit Leader"},
			Description:     "Melee attack on named NPC should resolve to their ID",
		},
		{
			Name:            "Shoot at the scout on the ridge",
			RawInput:        "I draw my rifle and take aim at the scout on the ridge",
			Context:         "The Outlaw Scout is perched on a rocky ridge, rifle in hand",
			OtherCharacters: npcs,
			ExpectedType:    action.Attack,
			ExpectedSkills:  []string{"Shoot"},
			ValidTargets:    []string{"scene_4_npc_1", "Outlaw Scout"},
			Description:     "Contextual reference to 'the scout' should resolve to Outlaw Scout",
		},
		{
			Name:            "Provoke the sheriff",
			RawInput:        "I get in Sheriff Morgan's face and call him a coward",
			Context:         "In the saloon, tempers are running hot",
			OtherCharacters: npcs,
			ExpectedType:    action.Attack,
			ExpectedSkills:  []string{"Provoke"},
			ValidTargets:    []string{"scene_1_npc_0", "Sheriff Morgan"},
			Description:     "Mental attack via Provoke should target the named character",
		},
		{
			Name:            "Ambush the bandit",
			RawInput:        "I leap from behind the rock and drive my knife into the bandit leader",
			Context:         "Hidden behind a boulder, the Bandit Leader walks past unaware",
			OtherCharacters: npcs,
			ExpectedType:    action.Attack,
			ExpectedSkills:  []string{"Fight"},
			ValidTargets:    []string{"scene_2_npc_0", "Bandit Leader"},
			Description:     "Ambush attack should still correctly target the NPC",
		},
	}
}

// getCreateAdvantageTargetTestCases returns test cases where CaA targets a specific NPC
func getCreateAdvantageTargetTestCases() []TargetTestCase {
	npcs := getSceneNPCs()
	return []TargetTestCase{
		{
			Name:            "Study scout patrol pattern",
			RawInput:        "I watch the Outlaw Scout to learn his patrol route",
			Context:         "Observing from a distance, the Outlaw Scout paces along the ridge",
			OtherCharacters: npcs,
			ExpectedType:    action.CreateAdvantage,
			ExpectedSkills:  []string{"Notice", "Investigate"},
			ValidTargets:    []string{"scene_4_npc_1", "Outlaw Scout"},
			Description:     "Studying an NPC's behavior is CaA targeting that NPC",
		},
		{
			Name:            "Intimidate bandit leader",
			RawInput:        "I crack my knuckles and stare down the Bandit Leader to make him nervous",
			Context:         "Tense standoff in the canyon, the Bandit Leader is weighing his options",
			OtherCharacters: npcs,
			ExpectedType:    action.CreateAdvantage,
			ExpectedSkills:  []string{"Provoke"},
			ValidTargets:    []string{"scene_2_npc_0", "Bandit Leader"},
			Description:     "Intimidation to create an aspect targets the NPC",
		},
		{
			Name:            "Read the sheriff's mood",
			RawInput:        "I try to gauge whether Sheriff Morgan is sympathetic to our cause",
			Context:         "Meeting with Sheriff Morgan in his office to discuss the outlaw problem",
			OtherCharacters: npcs,
			ExpectedType:    action.CreateAdvantage,
			ExpectedSkills:  []string{"Empathy"},
			ValidTargets:    []string{"scene_1_npc_0", "Sheriff Morgan"},
			Description:     "Reading someone to discover an aspect targets that character",
		},
	}
}

// getNoTargetTestCases returns test cases where no character target should be set
func getNoTargetTestCases() []TargetTestCase {
	npcs := getSceneNPCs()
	return []TargetTestCase{
		{
			Name:            "Take cover behind rocks",
			RawInput:        "I dive behind the nearest boulder for cover",
			Context:         "Bullets are flying in the canyon ambush",
			OtherCharacters: npcs,
			ExpectedType:    action.Overcome,
			ExpectedSkills:  []string{"Athletics"},
			ValidTargets:    []string{}, // No character target
			Description:     "Taking cover targets the environment, not a character",
		},
		{
			Name:            "Climb the canyon wall",
			RawInput:        "I climb up the canyon wall to get to higher ground",
			Context:         "Pinned down in the canyon, need a better vantage point",
			OtherCharacters: npcs,
			ExpectedType:    action.Overcome,
			ExpectedSkills:  []string{"Athletics"},
			ValidTargets:    []string{}, // No character target
			Description:     "Environmental action has no character target",
		},
	}
}

// evaluateTargetTestCase runs a single targeting test case
func evaluateTargetTestCase(ctx context.Context, parser *engine.ActionParser, char *character.Character, tc TargetTestCase) TargetEvalResult {
	req := engine.ActionParseRequest{
		Character:       char,
		RawInput:        tc.RawInput,
		Context:         tc.Context,
		OtherCharacters: tc.OtherCharacters,
	}

	parsedAction, err := parser.ParseAction(ctx, req)
	if err != nil {
		return TargetEvalResult{
			TestCase: tc,
			Error:    err,
		}
	}

	skillAcceptable := false
	for _, s := range tc.ExpectedSkills {
		if strings.EqualFold(parsedAction.Skill, s) {
			skillAcceptable = true
			break
		}
	}

	// Check target resolution
	targetResolved := false
	targetIsID := false

	if len(tc.ValidTargets) == 0 {
		// Expect no character target
		targetResolved = parsedAction.Target == "" || !isCharacterTarget(parsedAction.Target, tc.OtherCharacters)
	} else {
		// Build a temporary engine to test resolution
		eng, _ := engine.New()
		for _, npc := range tc.OtherCharacters {
			eng.AddCharacter(npc)
		}

		resolved := eng.ResolveCharacter(parsedAction.Target)
		if resolved != nil {
			targetResolved = true
			// Check if any valid target matches
			for _, valid := range tc.ValidTargets {
				if resolved.ID == valid || strings.EqualFold(resolved.Name, valid) {
					targetResolved = true
					break
				}
			}
		}

		// Check if it's a clean ID (best case)
		for _, npc := range tc.OtherCharacters {
			if parsedAction.Target == npc.ID {
				targetIsID = true
				break
			}
		}
	}

	return TargetEvalResult{
		TestCase:        tc,
		ActualType:      parsedAction.Type,
		ActualSkill:     parsedAction.Skill,
		ActualTarget:    parsedAction.Target,
		TypeMatches:     parsedAction.Type == tc.ExpectedType,
		SkillAcceptable: skillAcceptable,
		TargetResolved:  targetResolved,
		TargetIsID:      targetIsID,
		Error:           nil,
	}
}

// isCharacterTarget checks if the target string references any of the given characters
func isCharacterTarget(target string, chars []*character.Character) bool {
	lower := strings.ToLower(target)
	for _, c := range chars {
		if strings.Contains(lower, strings.ToLower(c.ID)) ||
			strings.Contains(lower, strings.ToLower(c.Name)) {
			return true
		}
	}
	return false
}

// TestActionParser_TargetResolution validates that the LLM returns resolvable targets
// when characters are present in the scene.
// Run with: go test -v -tags=llmeval -run TestActionParser_TargetResolution ./test/llmeval/ -timeout 5m
func TestActionParser_TargetResolution(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	parser := engine.NewActionParser(client)
	char := getTestCharacter()
	ctx := context.Background()

	allTestCases := []struct {
		category string
		cases    []TargetTestCase
	}{
		{"AttackTargets", getAttackTargetTestCases()},
		{"CreateAdvantageTargets", getCreateAdvantageTargetTestCases()},
		{"NoTarget", getNoTargetTestCases()},
	}

	var results []TargetEvalResult
	totalTests := 0
	typeCorrect := 0
	targetResolved := 0
	targetCleanID := 0
	targetExpected := 0 // Tests where we expect a character target

	for _, category := range allTestCases {
		t.Run(category.category, func(t *testing.T) {
			for _, tc := range category.cases {
				t.Run(tc.Name, func(t *testing.T) {
					result := evaluateTargetTestCase(ctx, parser, char, tc)
					results = append(results, result)

					if result.Error != nil {
						t.Errorf("Error: %v", result.Error)
						return
					}

					totalTests++

					if result.TypeMatches {
						typeCorrect++
					}

					hasExpectedTarget := len(tc.ValidTargets) > 0
					if hasExpectedTarget {
						targetExpected++
					}

					if result.TargetResolved {
						targetResolved++
					}
					if result.TargetIsID {
						targetCleanID++
					}

					if verboseLogging {
						typeStatus := "✓"
						if !result.TypeMatches {
							typeStatus = "✗"
						}
						targetStatus := "✓"
						if !result.TargetResolved {
							targetStatus = "✗"
						}
						idNote := ""
						if result.TargetIsID {
							idNote = " (clean ID)"
						}

						t.Logf("%s Type: expected=%s, got=%s", typeStatus, tc.ExpectedType, result.ActualType)
						t.Logf("%s Target: got='%s'%s", targetStatus, result.ActualTarget, idNote)
					}

					// Assertions
					assert.Equal(t, tc.ExpectedType, result.ActualType,
						"Action type mismatch for '%s'", tc.RawInput)
					assert.True(t, result.SkillAcceptable,
						"Skill mismatch for '%s': expected one of %v, got %s",
						tc.RawInput, tc.ExpectedSkills, result.ActualSkill)
					assert.True(t, result.TargetResolved,
						"Target not resolvable for '%s': got target='%s', expected one of %v",
						tc.RawInput, result.ActualTarget, tc.ValidTargets)
				})
			}
		})
	}

	// Print summary
	t.Log("\n========== TARGET RESOLUTION SUMMARY ==========")
	t.Logf("Total Tests: %d", totalTests)
	t.Logf("Type Accuracy:     %d/%d (%.1f%%)",
		typeCorrect, totalTests,
		float64(typeCorrect)*100/float64(totalTests))
	if targetExpected > 0 {
		t.Logf("Target Resolvable: %d/%d (%.1f%%)",
			targetResolved, totalTests,
			float64(targetResolved)*100/float64(totalTests))
		t.Logf("Target Clean ID:   %d/%d (%.1f%%)",
			targetCleanID, targetExpected,
			float64(targetCleanID)*100/float64(targetExpected))
	}

	// Print failed cases
	t.Log("\n--- Failed Target Cases ---")
	failCount := 0
	for _, r := range results {
		if !r.TargetResolved {
			failCount++
			t.Logf("FAIL: '%s'", r.TestCase.RawInput)
			t.Logf("      Target: '%s'", r.ActualTarget)
			t.Logf("      Expected one of: %v", r.TestCase.ValidTargets)
			t.Logf("      Description: %s", r.TestCase.Description)
		}
	}
	if failCount == 0 {
		t.Log("  (none)")
	}

	// Report non-ID targets (resolved but not clean IDs) — informational, not failures
	t.Log("\n--- Targets Resolved via Fallback (not clean IDs) ---")
	fallbackCount := 0
	for _, r := range results {
		if r.TargetResolved && !r.TargetIsID && len(r.TestCase.ValidTargets) > 0 {
			fallbackCount++
			t.Logf("INFO: '%s' → target='%s' (resolved via name/composite match)",
				r.TestCase.RawInput, r.ActualTarget)
		}
	}
	if fallbackCount == 0 {
		t.Log("  (none — all targets were clean IDs)")
	}
}
