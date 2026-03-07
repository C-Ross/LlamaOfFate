//go:build llmeval

package llmeval_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AspectGeneratorTestCase defines a test case for aspect generation
type AspectGeneratorTestCase struct {
	Name            string
	Skill           string
	Description     string
	RawInput        string
	Difficulty      dice.Ladder
	Context         string
	TargetType      string
	ExistingAspects []string
	RollResult      dice.CheckResult
	ExpectedOutcome dice.OutcomeType
}

// AspectGeneratorEvaluationSummary tracks test results
type AspectGeneratorEvaluationSummary struct {
	TotalTests       int
	Passed           int
	Failed           int
	ByOutcome        map[dice.OutcomeType]struct{ Passed, Failed int }
	LengthViolations int
	ParseErrors      int
}

func TestAspectGeneratorLLMEvaluation(t *testing.T) {
	azureClient := RequireLLMClient(t)
	verboseLogging := VerboseLoggingEnabled()

	// Create aspect generator
	aspectGenerator := engine.NewAspectGenerator(azureClient)

	// Create a test character
	char := createTestCharacter()

	// Define test cases
	testCases := getAspectGeneratorTestCases()

	ctx := context.Background()

	summary := AspectGeneratorEvaluationSummary{
		ByOutcome: make(map[dice.OutcomeType]struct{ Passed, Failed int }),
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.Name, func(t *testing.T) {
			summary.TotalTests++

			// Create the action
			testAction := action.NewAction(
				fmt.Sprintf("action-%s", tc.Name),
				char.ID,
				action.CreateAdvantage,
				tc.Skill,
				tc.Description,
			)
			testAction.RawInput = tc.RawInput
			testAction.Difficulty = tc.Difficulty

			// Use the predefined roll result and derive outcome
			testAction.CheckResult = &tc.RollResult
			outcome := tc.RollResult.CompareAgainst(tc.Difficulty)
			testAction.Outcome = outcome

			if verboseLogging {
				t.Logf("=== Test Case: %s ===", tc.Name)
				t.Logf("Action: %s", tc.Description)
				t.Logf("Player Intent: %s", tc.RawInput)
				t.Logf("Skill Used: %s (%s)", tc.Skill, tc.RollResult.BaseSkill.String())
				t.Logf("Difficulty: %s", tc.Difficulty.String())
				t.Logf("Roll: %s (Total: %+d)", tc.RollResult.Roll.String(), tc.RollResult.Roll.Total)
				t.Logf("Final Result: %s vs %s = %s (%+d shifts)",
					tc.RollResult.FinalValue.String(),
					tc.Difficulty.String(),
					outcome.Type.String(),
					outcome.Shifts)
			}

			// Create the aspect generation request
			req := prompt.AspectGenerationRequest{
				Character:       char,
				Action:          testAction,
				Outcome:         outcome,
				Context:         tc.Context,
				TargetType:      tc.TargetType,
				ExistingAspects: tc.ExistingAspects,
			}

			// Generate the aspect using the LLM
			response, err := aspectGenerator.GenerateAspect(ctx, req)
			if err != nil {
				summary.Failed++
				summary.ParseErrors++
				stats := summary.ByOutcome[outcome.Type]
				stats.Failed++
				summary.ByOutcome[outcome.Type] = stats
				t.Errorf("Error generating aspect: %v", err)
				return
			}

			if verboseLogging {
				t.Logf("--- Generated Aspect ---")
				t.Logf("Aspect: \"%s\"", response.AspectText)
				t.Logf("Description: %s", response.Description)
				t.Logf("Duration: %s", response.Duration)
				t.Logf("Free Invokes: %d", response.FreeInvokes)
				if response.IsBoost {
					t.Logf("Type: Boost")
				} else {
					t.Logf("Type: Full Aspect")
				}
				t.Logf("Reasoning: %s", response.Reasoning)
			}

			// Validate the response
			testPassed := true

			// Check aspect text length (2-10 words) for non-failure outcomes
			if outcome.Type != dice.Failure {
				wordCount := len(strings.Fields(response.AspectText))
				if wordCount < 2 || wordCount > 10 {
					summary.LengthViolations++
					testPassed = false
					t.Errorf("Aspect length violation: '%s' has %d words (expected 2-10)",
						response.AspectText, wordCount)
				}

				// Check that aspect text is not empty
				if response.AspectText == "" {
					testPassed = false
					t.Error("Aspect text is empty for non-failure outcome")
				}
			}

			// Validate free invokes based on outcome type
			expectedFreeInvokes := getExpectedFreeInvokes(outcome.Type)
			if response.FreeInvokes != expectedFreeInvokes {
				testPassed = false
				t.Errorf("Free invokes mismatch: got %d, expected %d for %s",
					response.FreeInvokes, expectedFreeInvokes, outcome.Type.String())
			}

			// Validate boost flag based on outcome type
			expectedIsBoost := (outcome.Type == dice.Tie)
			if response.IsBoost != expectedIsBoost {
				testPassed = false
				t.Errorf("IsBoost mismatch: got %v, expected %v for %s",
					response.IsBoost, expectedIsBoost, outcome.Type.String())
			}

			// Validate duration is one of valid values
			validDurations := []string{"scene", "session", "permanent"}
			durationValid := false
			for _, d := range validDurations {
				if response.Duration == d {
					durationValid = true
					break
				}
			}
			if !durationValid {
				testPassed = false
				t.Errorf("Invalid duration: '%s' (expected one of: %v)",
					response.Duration, validDurations)
			}

			// Check aspect is not a duplicate of existing aspects
			for _, existing := range tc.ExistingAspects {
				if strings.EqualFold(response.AspectText, existing) {
					testPassed = false
					t.Errorf("Generated aspect '%s' duplicates existing aspect '%s'",
						response.AspectText, existing)
				}
			}

			// Update summary
			stats := summary.ByOutcome[outcome.Type]
			if testPassed {
				summary.Passed++
				stats.Passed++
			} else {
				summary.Failed++
				stats.Failed++
			}
			summary.ByOutcome[outcome.Type] = stats
		})
	}

	// Print summary
	t.Logf("\n=== Aspect Generator Evaluation Summary ===")
	t.Logf("Total Tests: %d", summary.TotalTests)
	t.Logf("Passed: %d (%.1f%%)", summary.Passed, float64(summary.Passed)/float64(summary.TotalTests)*100)
	t.Logf("Failed: %d", summary.Failed)
	t.Logf("Length Violations: %d", summary.LengthViolations)
	t.Logf("Parse Errors: %d", summary.ParseErrors)
	t.Logf("\nBy Outcome Type:")
	for outcomeType, stats := range summary.ByOutcome {
		total := stats.Passed + stats.Failed
		if total > 0 {
			t.Logf("  %s: %d/%d passed (%.1f%%)",
				outcomeType.String(), stats.Passed, total, float64(stats.Passed)/float64(total)*100)
		}
	}
}

func createTestCharacter() *core.Character {
	char := core.NewCharacter("player-001", "Zara the Swift")
	char.Aspects.HighConcept = "Acrobatic Cat Burglar"
	char.Aspects.Trouble = "Can't Resist a Shiny Challenge"
	char.Aspects.AddAspect("Friends in Low Places")
	char.Aspects.AddAspect("Parkour Expert")
	char.SetSkill("Athletics", dice.Great)
	char.SetSkill("Stealth", dice.Good)
	char.SetSkill("Burglary", dice.Good)
	char.SetSkill("Deceive", dice.Fair)
	char.SetSkill("Fight", dice.Fair)
	char.SetSkill("Notice", dice.Average)
	return char
}

func getExpectedFreeInvokes(outcomeType dice.OutcomeType) int {
	switch outcomeType {
	case dice.SuccessWithStyle:
		return 2
	case dice.Success:
		return 1
	case dice.Tie:
		return 1
	case dice.Failure:
		return 0
	default:
		return 0
	}
}

func getAspectGeneratorTestCases() []AspectGeneratorTestCase {
	return []AspectGeneratorTestCase{
		// Success cases
		{
			Name:            "Rooftop_Chase_Athletics_Success",
			Skill:           "Athletics",
			Description:     "Parkour across rooftops to gain advantage",
			RawInput:        "I want to use my parkour skills to get to higher ground and find the perfect spot to jump down on my target",
			Difficulty:      dice.Fair,
			Context:         "A chase scene across the rooftops of the old town. Narrow alleys below, various building heights, clotheslines and chimneys provide obstacles and opportunities.",
			TargetType:      "situation",
			ExistingAspects: []string{"Narrow Alleyways Below", "Uneven Rooftop Heights"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Blank, dice.Plus, dice.Minus}, Total: 1},
				BaseSkill:  dice.Great,
				Modifier:   0,
				FinalValue: dice.Superb,
			},
		},
		{
			Name:            "Stealth_Infiltration_Success",
			Skill:           "Stealth",
			Description:     "Find a hidden vantage point",
			RawInput:        "I want to scout the area and find a good hiding spot where I can observe the guards' patrol patterns",
			Difficulty:      dice.Good,
			Context:         "The grand estate's courtyard at night. Manicured gardens with topiary, a central fountain, and several guard patrols. Gas lamps provide pools of light with deep shadows between.",
			TargetType:      "character",
			ExistingAspects: []string{"Patrolling Guards", "Pools of Lamplight", "Ornate Topiary"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Blank, dice.Plus, dice.Blank, dice.Plus}, Total: 2},
				BaseSkill:  dice.Good,
				Modifier:   0,
				FinalValue: dice.Superb,
			},
		},
		{
			Name:            "Deception_Distraction_Tie",
			Skill:           "Deceive",
			Description:     "Create a diversion with false information",
			RawInput:        "I'm going to start a rumor about a fire in the east wing to draw the guards away from the vault",
			Difficulty:      dice.Fair,
			Context:         "Inside the estate during a fancy party. Well-dressed guests, servants carrying trays, guards trying to blend in while staying alert. Perfect cover for social engineering.",
			TargetType:      "situation",
			ExistingAspects: []string{"Crowded Party", "Distracted Guards", "Gossiping Nobles"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Minus, dice.Blank, dice.Blank, dice.Plus}, Total: 0},
				BaseSkill:  dice.Fair,
				Modifier:   0,
				FinalValue: dice.Fair,
			},
		},
		// Success with Style cases
		{
			Name:            "Notice_Scout_SuccessWithStyle",
			Skill:           "Notice",
			Description:     "Carefully observe the enemy camp",
			RawInput:        "I'm going to spend time carefully watching the camp to learn their routines and find weaknesses",
			Difficulty:      dice.Average,
			Context:         "Overlooking a bandit camp from a nearby ridge at dawn. Multiple tents, a central fire pit, and what appears to be a supply wagon.",
			TargetType:      "situation",
			ExistingAspects: []string{"Early Morning Light", "Bandit Camp Below"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Plus, dice.Plus, dice.Blank}, Total: 3},
				BaseSkill:  dice.Average,
				Modifier:   0,
				FinalValue: dice.Great,
			},
		},
		{
			Name:            "Burglary_CaseLock_SuccessWithStyle",
			Skill:           "Burglary",
			Description:     "Examine the vault lock mechanism",
			RawInput:        "I carefully examine the lock, looking for any weaknesses or unusual features that might help me crack it",
			Difficulty:      dice.Fair,
			Context:         "Standing before a massive vault door in the basement. The lock is intricate, with multiple tumblers and what appears to be a magical ward.",
			TargetType:      "object",
			ExistingAspects: []string{"Magical Ward", "Complex Mechanism"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Plus, dice.Blank, dice.Plus}, Total: 3},
				BaseSkill:  dice.Good,
				Modifier:   0,
				FinalValue: dice.Fantastic,
			},
		},
		// Tie cases (should create boosts)
		{
			Name:            "Fight_Feint_Tie",
			Skill:           "Fight",
			Description:     "Feint to create an opening",
			RawInput:        "I fake a high attack to get him to raise his guard, hoping to create an opening below",
			Difficulty:      dice.Good,
			Context:         "Dueling in a moonlit courtyard. Both combatants are skilled and evenly matched. The fight has been going back and forth.",
			TargetType:      "character",
			ExistingAspects: []string{"Moonlit Courtyard", "Evenly Matched"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Blank, dice.Blank, dice.Blank}, Total: 1},
				BaseSkill:  dice.Fair,
				Modifier:   0,
				FinalValue: dice.Good,
			},
		},
		{
			Name:            "Athletics_Positioning_Tie",
			Skill:           "Athletics",
			Description:     "Maneuver for better position",
			RawInput:        "I try to circle around to get the sun at my back",
			Difficulty:      dice.Fair,
			Context:         "An outdoor arena with the afternoon sun blazing. Sand underfoot, a cheering crowd surrounding the fighting pit.",
			TargetType:      "situation",
			ExistingAspects: []string{"Blazing Afternoon Sun", "Sandy Footing"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Blank, dice.Blank, dice.Blank, dice.Blank}, Total: 0},
				BaseSkill:  dice.Fair,
				Modifier:   0,
				FinalValue: dice.Fair,
			},
		},
		// Failure cases
		{
			Name:            "Stealth_Sneak_Failure",
			Skill:           "Stealth",
			Description:     "Sneak past the guards",
			RawInput:        "I try to slip past the guards while they're distracted by the commotion",
			Difficulty:      dice.Good,
			Context:         "The main hallway of the estate. Two guards stand at attention near the door to the treasury. A servant just dropped a tray nearby.",
			TargetType:      "situation",
			ExistingAspects: []string{"Dropped Tray Distraction", "Alert Guards"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Minus, dice.Minus, dice.Blank, dice.Blank}, Total: -2},
				BaseSkill:  dice.Good,
				Modifier:   0,
				FinalValue: dice.Average,
			},
		},
		{
			Name:            "Deceive_Bluff_Failure",
			Skill:           "Deceive",
			Description:     "Bluff my way past the checkpoint",
			RawInput:        "I try to convince the guard that I'm a kitchen servant who got lost",
			Difficulty:      dice.Good,
			Context:         "A checkpoint inside the estate. A suspicious guard is questioning everyone who passes. You're wearing clothes that don't quite fit the servant aesthetic.",
			TargetType:      "character",
			ExistingAspects: []string{"Suspicious Guard", "Poorly Fitting Disguise"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Minus, dice.Blank, dice.Minus, dice.Minus}, Total: -3},
				BaseSkill:  dice.Fair,
				Modifier:   0,
				FinalValue: dice.Terrible,
			},
		},
		// Additional varied scenarios
		{
			Name:            "Notice_ReadPerson_Success",
			Skill:           "Notice",
			Description:     "Read the noble's tells",
			RawInput:        "I watch the baron closely during our conversation, looking for any nervous habits or tells that might give away his true intentions",
			Difficulty:      dice.Fair,
			Context:         "A private meeting room in the baron's estate. Rich furnishings, a crackling fireplace, and the baron sitting across from you with a practiced smile.",
			TargetType:      "character",
			ExistingAspects: []string{"Practiced Smile", "Nervous Fingers"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Blank, dice.Plus, dice.Blank}, Total: 2},
				BaseSkill:  dice.Average,
				Modifier:   0,
				FinalValue: dice.Good,
			},
		},
		{
			Name:            "Athletics_CreateCover_Success",
			Skill:           "Athletics",
			Description:     "Overturn furniture for cover",
			RawInput:        "I kick over the heavy oak table to create cover from the crossbow bolts",
			Difficulty:      dice.Average,
			Context:         "A tavern common room turned into a combat zone. Tables, chairs, and broken mugs everywhere. Enemies with crossbows at the far end.",
			TargetType:      "situation",
			ExistingAspects: []string{"Tavern Brawl Chaos", "Crossbow Bolts Flying"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Plus, dice.Blank, dice.Minus}, Total: 1},
				BaseSkill:  dice.Great,
				Modifier:   0,
				FinalValue: dice.Superb,
			},
		},
		{
			Name:            "Burglary_DisableTrap_SuccessWithStyle",
			Skill:           "Burglary",
			Description:     "Disable the pressure plate trap",
			RawInput:        "I carefully examine the pressure plate and try to jam the mechanism so it won't trigger",
			Difficulty:      dice.Good,
			Context:         "A trapped corridor in an ancient tomb. The pressure plate is barely visible, and you can see small holes in the walls where darts would emerge.",
			TargetType:      "object",
			ExistingAspects: []string{"Ancient Tomb", "Dart Trap"},
			RollResult: dice.CheckResult{
				Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Plus, dice.Plus, dice.Blank}, Total: 3},
				BaseSkill:  dice.Good,
				Modifier:   0,
				FinalValue: dice.Fantastic,
			},
		},
	}
}

func TestAspectGeneratorEdgeCases(t *testing.T) {
	azureClient := RequireLLMClient(t)
	verboseLogging := VerboseLoggingEnabled()

	aspectGenerator := engine.NewAspectGenerator(azureClient)
	char := createTestCharacter()
	ctx := context.Background()

	t.Run("Many_Existing_Aspects", func(t *testing.T) {
		// Test with many existing aspects to ensure LLM doesn't duplicate
		testAction := action.NewAction("action-many", char.ID, action.CreateAdvantage, "Notice", "Survey the battlefield")
		testAction.RawInput = "I take a moment to survey the entire battlefield and look for any tactical advantage we might exploit"
		testAction.Difficulty = dice.Fair

		rollResult := dice.CheckResult{
			Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Blank, dice.Plus, dice.Blank}, Total: 2},
			BaseSkill:  dice.Average,
			Modifier:   0,
			FinalValue: dice.Good,
		}
		testAction.CheckResult = &rollResult
		outcome := rollResult.CompareAgainst(dice.Fair)

		existingAspects := []string{
			"High Ground",
			"Flanking Position",
			"Defensive Formation",
			"Scattered Forces",
			"Low Morale",
			"Fresh Reinforcements",
			"Muddy Terrain",
			"Setting Sun",
		}

		req := prompt.AspectGenerationRequest{
			Character:       char,
			Action:          testAction,
			Outcome:         outcome,
			Context:         "A large battlefield at sunset. Two armies face each other across a muddy field. Various terrain features provide tactical options.",
			TargetType:      "situation",
			ExistingAspects: existingAspects,
		}

		response, err := aspectGenerator.GenerateAspect(ctx, req)
		require.NoError(t, err, "Failed to generate aspect")

		if verboseLogging {
			t.Logf("Generated aspect with many existing: '%s'", response.AspectText)
		}

		// Check aspect is not a duplicate
		for _, existing := range existingAspects {
			assert.NotEqual(t, strings.ToLower(response.AspectText), strings.ToLower(existing),
				"Generated aspect should not duplicate existing aspect '%s'", existing)
		}

		// Check word count
		wordCount := len(strings.Fields(response.AspectText))
		assert.GreaterOrEqual(t, wordCount, 2, "Aspect should have at least 2 words")
		assert.LessOrEqual(t, wordCount, 10, "Aspect should have at most 10 words")
	})

	t.Run("Minimal_Context", func(t *testing.T) {
		// Test with minimal context
		testAction := action.NewAction("action-minimal", char.ID, action.CreateAdvantage, "Athletics", "Jump")
		testAction.RawInput = "I jump"
		testAction.Difficulty = dice.Average

		rollResult := dice.CheckResult{
			Roll:       &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Plus, dice.Blank, dice.Blank}, Total: 2},
			BaseSkill:  dice.Great,
			Modifier:   0,
			FinalValue: dice.Fantastic,
		}
		testAction.CheckResult = &rollResult
		outcome := rollResult.CompareAgainst(dice.Average)

		req := prompt.AspectGenerationRequest{
			Character:       char,
			Action:          testAction,
			Outcome:         outcome,
			Context:         "Combat.",
			TargetType:      "situation",
			ExistingAspects: []string{},
		}

		response, err := aspectGenerator.GenerateAspect(ctx, req)
		require.NoError(t, err, "Failed to generate aspect with minimal context")

		if verboseLogging {
			t.Logf("Generated aspect with minimal context: '%s'", response.AspectText)
		}

		// Should still produce a valid aspect
		assert.NotEmpty(t, response.AspectText, "Should generate aspect even with minimal context")
		wordCount := len(strings.Fields(response.AspectText))
		assert.GreaterOrEqual(t, wordCount, 2, "Aspect should have at least 2 words")
		assert.LessOrEqual(t, wordCount, 10, "Aspect should have at most 10 words")
	})
}
