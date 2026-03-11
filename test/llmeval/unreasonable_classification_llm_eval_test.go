//go:build llmeval

package llmeval_test

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
)

// UnreasonableTestCase represents a test case for unreasonable input classification.
// Positive cases should be classified as "unreasonable"; negative cases should NOT.
type UnreasonableTestCase struct {
	Name             string
	RawInput         string
	SceneName        string
	SceneDescription string
	CharacterAspects []string
	Genre            string
	ExpectedType     string // "unreasonable" for positive, the correct type for negative
	Description      string
}

// getUnreasonablePositiveCases returns inputs that SHOULD be classified as unreasonable.
func getUnreasonablePositiveCases() []UnreasonableTestCase {
	return []UnreasonableTestCase{
		{
			Name:             "Telekinesis with non-magical character",
			RawInput:         "I use my telekinetic powers to lift the table",
			SceneName:        "Reactor Control Room",
			SceneDescription: "A high-tech control room on a space station with warning lights flashing",
			CharacterAspects: []string{
				"Veteran Nuclear Engineer with Nerves of Steel",
				"First Assignment on Europa — Everything is Unfamiliar",
				"If It Has a Reactor, I Can Fix It",
				"Stubborn Enough to Outlast a Meltdown",
				"Old-School Hands-On Troubleshooter",
			},
			Genre:        "Sci-Fi",
			ExpectedType: "unreasonable",
			Description:  "Nuclear engineer has no supernatural aspects — telekinesis is impossible",
		},
		{
			Name:             "Smartphone in Western setting",
			RawInput:         "I pull out my smartphone and google the location of the gang",
			SceneName:        "The Dusty Spur Saloon",
			SceneDescription: "A frontier saloon in a Wild West town, dust motes in the air",
			CharacterAspects: []string{
				"Haunted Former Rancher Seeking Justice",
				"Vengeance Burns Hotter Than Reason",
				"Fastest Draw in Three Counties",
				"I Remember Every Face",
				"The Land Speaks to Me",
			},
			Genre:        "Western",
			ExpectedType: "unreasonable",
			Description:  "Smartphones don't exist in a Western setting",
		},
		{
			Name:             "Flying with no flight aspects",
			RawInput:         "I fly over the canyon to the other side",
			SceneName:        "Canyon Rim",
			SceneDescription: "Standing at the edge of a deep canyon, the opposite side is far away",
			CharacterAspects: []string{
				"Haunted Former Rancher Seeking Justice",
				"Vengeance Burns Hotter Than Reason",
				"Fastest Draw in Three Counties",
				"I Remember Every Face",
				"The Land Speaks to Me",
			},
			Genre:        "Western",
			ExpectedType: "unreasonable",
			Description:  "No aspects support flight — humans can't fly",
		},
		{
			Name:             "Hacking a computer in fantasy",
			RawInput:         "I hack into the castle's computer system to disable the defenses",
			SceneName:        "Castle Gate",
			SceneDescription: "A medieval castle with stone walls and iron portcullis",
			CharacterAspects: []string{
				"Scholarly Arcane Investigator",
				"Curiosity About Forbidden Knowledge",
				"Trained by the College of Mages",
				"Sees Magic in Everything",
				"Protective Ward Tattoos",
			},
			Genre:        "Fantasy",
			ExpectedType: "unreasonable",
			Description:  "Computers don't exist in a fantasy medieval setting",
		},
		{
			Name:             "Casting magic as an engineer",
			RawInput:         "I cast a fireball at the malfunctioning reactor",
			SceneName:        "Reactor Control Room",
			SceneDescription: "A high-tech control room on a space station",
			CharacterAspects: []string{
				"Veteran Nuclear Engineer with Nerves of Steel",
				"First Assignment on Europa — Everything is Unfamiliar",
				"If It Has a Reactor, I Can Fix It",
				"Stubborn Enough to Outlast a Meltdown",
				"Old-School Hands-On Troubleshooter",
			},
			Genre:        "Sci-Fi",
			ExpectedType: "unreasonable",
			Description:  "Nuclear engineer has no magical aspects — fireballs are impossible in hard sci-fi",
		},
		{
			Name:             "Laser gun in Western",
			RawInput:         "I draw my laser pistol and blast the outlaw",
			SceneName:        "Main Street",
			SceneDescription: "A dusty main street in a frontier town, high noon",
			CharacterAspects: []string{
				"Haunted Former Rancher Seeking Justice",
				"Vengeance Burns Hotter Than Reason",
				"Fastest Draw in Three Counties",
				"I Remember Every Face",
				"The Land Speaks to Me",
			},
			Genre:        "Western",
			ExpectedType: "unreasonable",
			Description:  "Laser pistols don't exist in a Western setting",
		},
		{
			Name:             "Teleportation with no magical aspects",
			RawInput:         "I teleport behind the enemy",
			SceneName:        "Saloon Brawl",
			SceneDescription: "A brawl has broken out in the saloon, chairs flying everywhere",
			CharacterAspects: []string{
				"Haunted Former Rancher Seeking Justice",
				"Vengeance Burns Hotter Than Reason",
				"Fastest Draw in Three Counties",
				"I Remember Every Face",
				"The Land Speaks to Me",
			},
			Genre:        "Western",
			ExpectedType: "unreasonable",
			Description:  "Teleportation is impossible without supernatural aspects",
		},
		{
			Name:             "Summoning dragons as a netrunner",
			RawInput:         "I summon a dragon to destroy the corporate guards",
			SceneName:        "Corporate Lobby",
			SceneDescription: "A sleek corporate lobby with armed security guards",
			CharacterAspects: []string{
				"Ghost in the Machine Netrunner",
				"Wanted by Three Megacorps",
				"Military-Grade Cybernetic Reflexes",
				"Nobody Gets Left Behind",
				"I Know a Guy for Everything",
			},
			Genre:        "Cyberpunk",
			ExpectedType: "unreasonable",
			Description:  "Dragons don't exist in cyberpunk; netrunner has no summoning aspects",
		},
	}
}

// getUnreasonableNegativeCases returns inputs that should NOT be classified as unreasonable.
// These are either legitimate actions, or edge cases where aspects plausibly support the action.
func getUnreasonableNegativeCases() []UnreasonableTestCase {
	return []UnreasonableTestCase{
		{
			Name:             "Casting spell as a wizard",
			RawInput:         "I cast a protective ward around myself",
			SceneName:        "Ancient Tower",
			SceneDescription: "A crumbling tower radiating magical energy",
			CharacterAspects: []string{
				"Scholarly Arcane Investigator",
				"Curiosity About Forbidden Knowledge",
				"Trained by the College of Mages",
				"Sees Magic in Everything",
				"Protective Ward Tattoos",
			},
			Genre:        "Fantasy",
			ExpectedType: "action",
			Description:  "Character has explicit magical aspects — casting spells is legitimate",
		},
		{
			Name:             "Sensing spirits with haunted aspects",
			RawInput:         "I reach out with my senses to feel if there are spirits here",
			SceneName:        "Abandoned Homestead",
			SceneDescription: "A burned-out ranch, charred timbers in the moonlight",
			CharacterAspects: []string{
				"Haunted Former Rancher Seeking Justice",
				"Vengeance Burns Hotter Than Reason",
				"Fastest Draw in Three Counties",
				"I Remember Every Face",
				"The Land Speaks to Me",
			},
			Genre:        "Western",
			ExpectedType: "action",
			Description:  "\"Haunted\" and \"The Land Speaks to Me\" are ambiguous enough to support supernatural sensing",
		},
		{
			Name:             "Punching down a door",
			RawInput:         "I punch the wooden door as hard as I can to break it open",
			SceneName:        "Abandoned Warehouse",
			SceneDescription: "An old warehouse with a rotting wooden door",
			CharacterAspects: []string{
				"Haunted Former Rancher Seeking Justice",
				"Vengeance Burns Hotter Than Reason",
				"Fastest Draw in Three Counties",
				"I Remember Every Face",
				"The Land Speaks to Me",
			},
			Genre:        "Western",
			ExpectedType: "action",
			Description:  "Extreme but physically possible — let the dice decide",
		},
		{
			Name:             "Intimidating the whole room",
			RawInput:         "I slam my fist on the bar and demand everyone tell me what they know",
			SceneName:        "The Dusty Spur Saloon",
			SceneDescription: "A frontier saloon with rough patrons",
			CharacterAspects: []string{
				"Haunted Former Rancher Seeking Justice",
				"Vengeance Burns Hotter Than Reason",
				"Fastest Draw in Three Counties",
				"I Remember Every Face",
				"The Land Speaks to Me",
			},
			Genre:        "Western",
			ExpectedType: "action",
			Description:  "Social skill (Provoke) — dramatic but not supernatural",
		},
		{
			Name:             "Leaping across a chasm",
			RawInput:         "I take a running leap across the gap",
			SceneName:        "Mountain Pass",
			SceneDescription: "A narrow gap in the mountain trail, maybe 10 feet across",
			CharacterAspects: []string{
				"Veteran Nuclear Engineer with Nerves of Steel",
				"First Assignment on Europa — Everything is Unfamiliar",
				"If It Has a Reactor, I Can Fix It",
				"Stubborn Enough to Outlast a Meltdown",
				"Old-School Hands-On Troubleshooter",
			},
			Genre:        "Sci-Fi",
			ExpectedType: "action",
			Description:  "Athletic action — difficult but physically possible",
		},
		{
			Name:             "Using cybernetics in cyberpunk",
			RawInput:         "I jack into the security system with my neural interface",
			SceneName:        "Corporate Server Room",
			SceneDescription: "Rows of servers humming in a climate-controlled room",
			CharacterAspects: []string{
				"Ghost in the Machine Netrunner",
				"Wanted by Three Megacorps",
				"Military-Grade Cybernetic Reflexes",
				"Nobody Gets Left Behind",
				"I Know a Guy for Everything",
			},
			Genre:        "Cyberpunk",
			ExpectedType: "action",
			Description:  "Netrunner with cybernetic aspects — jacking in is genre-appropriate and aspect-supported",
		},
		{
			Name:             "Ordering a drink",
			RawInput:         "I order a whiskey from the bartender",
			SceneName:        "The Dusty Spur Saloon",
			SceneDescription: "A frontier saloon",
			CharacterAspects: []string{
				"Haunted Former Rancher Seeking Justice",
				"Vengeance Burns Hotter Than Reason",
				"Fastest Draw in Three Counties",
			},
			Genre:        "Western",
			ExpectedType: "dialog",
			Description:  "Mundane social interaction — should be dialog, not unreasonable",
		},
		{
			Name:             "Looking around",
			RawInput:         "I look around the control room for anything useful",
			SceneName:        "Reactor Control Room",
			SceneDescription: "A high-tech control room with blinking consoles",
			CharacterAspects: []string{
				"Veteran Nuclear Engineer with Nerves of Steel",
				"If It Has a Reactor, I Can Fix It",
			},
			Genre:        "Sci-Fi",
			ExpectedType: "clarification",
			Description:  "Basic observation — should be clarification, not unreasonable",
		},
	}
}

// UnreasonableClassificationResult stores the result of a single test.
type UnreasonableClassificationResult struct {
	TestCase   UnreasonableTestCase
	ActualType string
	Matches    bool
	Error      error
}

// evaluateUnreasonableClassification runs a single classification test with character and genre context.
func evaluateUnreasonableClassification(ctx context.Context, client llm.LLMClient, tc UnreasonableTestCase) UnreasonableClassificationResult {
	testScene := scene.NewScene("test-scene", tc.SceneName, tc.SceneDescription)

	data := promptpkg.InputClassificationData{
		Scene:            testScene,
		PlayerInput:      tc.RawInput,
		CharacterAspects: tc.CharacterAspects,
		Genre:            tc.Genre,
	}

	prompt, err := promptpkg.RenderInputClassification(data)
	if err != nil {
		return UnreasonableClassificationResult{TestCase: tc, Error: err}
	}

	classification, err := llm.SimpleCompletion(ctx, client, prompt, 10, 0.1)
	if err != nil {
		return UnreasonableClassificationResult{TestCase: tc, Error: err}
	}

	classification = promptpkg.ParseClassification(classification)

	return UnreasonableClassificationResult{
		TestCase:   tc,
		ActualType: classification,
		Matches:    classification == tc.ExpectedType,
	}
}

// TestUnreasonableClassification_LLMEvaluation tests that clearly impossible inputs
// are classified as "unreasonable" while legitimate creative actions are not.
// Run with: go test -v -tags=llmeval -run TestUnreasonableClassification_LLMEvaluation ./test/llmeval/ -timeout 5m
func TestUnreasonableClassification_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()
	verboseLogging := VerboseLoggingEnabled()

	allTestCases := []struct {
		category string
		cases    []UnreasonableTestCase
	}{
		{"Positive_ShouldBeUnreasonable", getUnreasonablePositiveCases()},
		{"Negative_ShouldNotBeUnreasonable", getUnreasonableNegativeCases()},
	}

	var results []UnreasonableClassificationResult
	positiveTotal, positiveCorrect := 0, 0
	negativeTotal, negativeCorrect := 0, 0

	for _, category := range allTestCases {
		t.Run(category.category, func(t *testing.T) {
			for _, tc := range category.cases {
				t.Run(tc.Name, func(t *testing.T) {
					result := evaluateUnreasonableClassification(ctx, client, tc)
					results = append(results, result)

					if result.Error != nil {
						t.Errorf("Error: %v", result.Error)
						return
					}

					if tc.ExpectedType == "unreasonable" {
						positiveTotal++
						if result.Matches {
							positiveCorrect++
						}
					} else {
						negativeTotal++
						if result.Matches {
							negativeCorrect++
						}
					}

					if verboseLogging {
						status := "✓"
						if !result.Matches {
							status = "✗"
						}
						t.Logf("%s Classification: expected=%s, got=%s", status, tc.ExpectedType, result.ActualType)
						t.Logf("  Input: %s", tc.RawInput)
						t.Logf("  Aspects: %v", tc.CharacterAspects)
						t.Logf("  Genre: %s", tc.Genre)
					}

					assert.Equal(t, tc.ExpectedType, result.ActualType,
						"Classification mismatch for '%s': %s", tc.RawInput, tc.Description)
				})
			}
		})
	}

	// Print summary
	t.Log("\n========== UNREASONABLE CLASSIFICATION SUMMARY ==========")
	t.Logf("Positive (should be unreasonable): %d/%d (%.1f%%)",
		positiveCorrect, positiveTotal,
		float64(positiveCorrect)*100/float64(positiveTotal))
	t.Logf("Negative (should NOT be unreasonable): %d/%d (%.1f%%)",
		negativeCorrect, negativeTotal,
		float64(negativeCorrect)*100/float64(negativeTotal))
	total := positiveTotal + negativeTotal
	correct := positiveCorrect + negativeCorrect
	t.Logf("Overall: %d/%d (%.1f%%)",
		correct, total,
		float64(correct)*100/float64(total))

	// Print failed cases
	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: '%s'", r.TestCase.RawInput)
			t.Logf("      Expected: %s, Got: %s", r.TestCase.ExpectedType, r.ActualType)
			t.Logf("      Aspects: %v", r.TestCase.CharacterAspects)
			t.Logf("      Genre: %s", r.TestCase.Genre)
			t.Logf("      Why: %s", r.TestCase.Description)
		}
	}
}
