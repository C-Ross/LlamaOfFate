//go:build llmeval

package llmeval_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	llmazure "github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ScenarioResTestCase represents a test case for scenario resolution checking
type ScenarioResTestCase struct {
	Name           string
	Scenario       *scene.Scenario
	SceneSummaries []promptpkg.SceneSummary
	LatestSummary  *promptpkg.SceneSummary
	PlayerName     string
	PlayerAspects  []string
	ExpectResolved bool
	Description    string // Why this should/shouldn't be resolved
}

// darkValleyScenario returns the scenario from the walkthrough_sword_and_sorcery session
// where the resolution checker failed to mark scenes 8-9 as resolved despite Malyster being dead.
func darkValleyScenario() *scene.Scenario {
	return &scene.Scenario{
		Title:   "Dark Valley's Burning Vengeance",
		Problem: "The ruthless sorcerer, Malyster, has returned to the Dark Valley, slaughtering Jon's family and burning his ranch to the ground. Jon must now stop Malyster before he slaughters the nearby village of Brindlemark, seeking revenge for some past injustice.",
		StoryQuestions: []string{
			"Will Jon's thirst for vengeance cloud his judgment against Malyster?",
			"Can Jon gather enough allies in Brindlemark before Malyster's dark magic corrupts the land?",
			"Will the village elder's secrets about Jon's past hinder or help his quest for justice?",
			"Can Jon defeat Malyster in a final confrontation?",
		},
		Setting: "The Dark Valley, a lawless frontier of rugged ranchlands and mystical energies, lies on the edge of a vast and mysterious forest. The nearby village of Brindlemark is a small settlement of hardy folk, living in the shadow of the valley's dark history and the looming threat of sorcery.",
		Genre:   "Sword and Sorcery",
	}
}

// getResolvedTestCases returns scenarios that SHOULD be marked resolved
func getResolvedTestCases() []ScenarioResTestCase {
	return []ScenarioResTestCase{
		{
			Name:     "Antagonist killed and village saved — walkthrough scene 8",
			Scenario: darkValleyScenario(),
			SceneSummaries: []promptpkg.SceneSummary{
				{
					NarrativeProse: "Jon braved the mystical Cliffs of the Fae Witch, overcoming her mental assaults to claim a powerful talisman. The Fae Witch, impressed or defeated, yielded to Jon's determination. With this newfound tool, Jon is one step closer to confronting Malyster.",
					KeyEvents:      []string{"The Fae Witch engaged Jon in a mental battle with riddles and phantasms.", "Jon resisted her attacks and overcame her, gaining her respect or submission.", "The Fae Witch yielded and granted Jon a magical talisman to counter Malyster's magic."},
				},
				{
					NarrativeProse: "As darkness descended upon Brindlemark, Jon arrived to find the village ablaze and its people in a state of panic. Meeting Elara, he rallied the villagers and steeled himself for a confrontation with the source of the chaos: Malyster. With a newfound sense of determination, Jon prepared to face the darkness head-on.",
					KeyEvents:      []string{"Jon meets Elara amidst the chaos", "Jon rallies the villagers", "Jon prepares to confront Malyster"},
				},
			},
			LatestSummary: &promptpkg.SceneSummary{
				NarrativeProse:    "Jon bravely confronted Malyster in the heart of Brindlemark, shattering the dark ritual that had entranced the villagers. With the talisman's power, he freed the villagers, who then turned on Malyster in their rage. In the aftermath, Elara expressed her gratitude to Jon for his heroism.",
				KeyEvents:         []string{"Jon used a talisman to counter Malyster's magic and freed the villagers from their trance.", "Malyster was disarmed and subsequently torn apart by the freed villagers.", "The dark ritual was disrupted, and the villagers were freed from their spell."},
				UnresolvedThreads: []string{"The aftermath of the dark ritual and its effects on Brindlemark village", "Elara's personal thanks and potential future interactions"},
			},
			PlayerName:     "Jon of the Dark Valley",
			PlayerAspects:  []string{"Haunted Former Rancher Seeking Justice", "Vengeance Burns Hotter Than Reason"},
			ExpectResolved: true,
			Description:    "Extracted from walkthrough_sword_and_sorcery session. Malyster is dead, village saved, all 4 story questions answerable. Resolution checker returned is_resolved=false citing 'village elder secrets unanswered' despite that being answered in scene 5.",
		},
		{
			Name:     "Denouement after antagonist death — walkthrough scene 9",
			Scenario: darkValleyScenario(),
			SceneSummaries: []promptpkg.SceneSummary{
				{
					NarrativeProse: "As darkness descended upon Brindlemark, Jon arrived to find the village ablaze and its people in a state of panic. Meeting Elara, he rallied the villagers and steeled himself for a confrontation with the source of the chaos: Malyster. With a newfound sense of determination, Jon prepared to face the darkness head-on.",
					KeyEvents:      []string{"Jon meets Elara amidst the chaos", "Jon rallies the villagers", "Jon prepares to confront Malyster"},
				},
				{
					NarrativeProse: "Jon bravely confronted Malyster in the heart of Brindlemark, shattering the dark ritual that had entranced the villagers. With the talisman's power, he freed the villagers, who then turned on Malyster in their rage. In the aftermath, Elara expressed her gratitude to Jon for his heroism.",
					KeyEvents:      []string{"Jon used a talisman to counter Malyster's magic and freed the villagers from their trance.", "Malyster was disarmed and subsequently torn apart by the freed villagers.", "The dark ritual was disrupted, and the villagers were freed from their spell."},
				},
			},
			LatestSummary: &promptpkg.SceneSummary{
				NarrativeProse:    "As the village began to heal from the darkness of Malyster's ritual, Jon found solace in Elara's forgiveness and acceptance. With a newfound sense of belonging, Jon bid farewell to the village and set out to confront the remnants of the evil that had threatened his new home. The journey ahead would be fraught with danger, but Jon was determined to put the past behind him.",
				KeyEvents:         []string{"Elara thanks Jon and forgives him, sharing a personal connection to his past", "Elara declares Jon one of the village and they share an embrace", "The village starts to rebuild"},
				UnresolvedThreads: []string{"Finding and dismantling Malyster's tower and any remaining evils", "Understanding the full extent of the village's secrets and Jon's past"},
			},
			PlayerName:     "Jon of the Dark Valley",
			PlayerAspects:  []string{"Haunted Former Rancher Seeking Justice", "Vengeance Burns Hotter Than Reason"},
			ExpectResolved: true,
			Description:    "Extracted from walkthrough_sword_and_sorcery session. Malyster dead since scene 8, village rebuilding, denouement complete. Resolution checker returned is_resolved=false citing sequel hooks (Malyster's tower, village secrets) as blockers.",
		},
		{
			Name: "Villain defeated and town saved",
			Scenario: &scene.Scenario{
				Title:   "The Cortez Gang's Last Stand",
				Problem: "The Cortez Gang is terrorizing Redemption Gulch, threatening to burn it to the ground unless the town pays a massive ransom.",
				StoryQuestions: []string{
					"Can Jesse discover where the Cortez Gang is hiding?",
					"Will the townspeople rally behind Jesse?",
					"Can Jesse defeat El Diablo Cortez in a final showdown?",
				},
			},
			SceneSummaries: []promptpkg.SceneSummary{
				{
					NarrativeProse: "Jesse arrived in Redemption Gulch and discovered the townspeople living in fear. He learned from Maggie at the saloon that the gang hides in the old silver mine.",
					KeyEvents:      []string{"Jesse arrived in town", "Maggie revealed the gang's hideout location"},
				},
				{
					NarrativeProse: "Jesse rallied the townspeople by convincing the sheriff and the ranchers to stand together. They armed themselves and prepared for a confrontation.",
					KeyEvents:      []string{"Sheriff joined Jesse's cause", "Ranchers agreed to fight"},
				},
			},
			LatestSummary: &promptpkg.SceneSummary{
				NarrativeProse:    "In a dramatic showdown at the mine entrance, Jesse faced El Diablo Cortez. After a fierce gunfight, Jesse shot the gun from Cortez's hand and the remaining gang members surrendered. The town is safe.",
				KeyEvents:         []string{"Jesse defeated El Diablo Cortez", "Gang surrendered", "Town saved"},
				UnresolvedThreads: []string{},
			},
			PlayerName:     "Jesse Calhoun",
			PlayerAspects:  []string{"Vengeful Rancher", "Quick Draw"},
			ExpectResolved: true,
			Description:    "All story questions answered, villain defeated, town saved — clear resolution",
		},
		{
			Name: "Problem circumvented rather than solved directly",
			Scenario: &scene.Scenario{
				Title:   "The Stolen Jewel of Aetheria",
				Problem: "The Jewel of Aetheria has been stolen from the Collegia Arcana, and rival factions are racing to recover it.",
				StoryQuestions: []string{
					"Can Cynere discover who hired her rival?",
					"Will Cynere recover the Jewel before her competitor?",
					"Can Cynere avoid the wrath of the Collegia?",
				},
			},
			SceneSummaries: []promptpkg.SceneSummary{
				{
					NarrativeProse: "Cynere discovered her rival is Anna, an agent of the Cult of Tranquility. The Jewel's true purpose is as a component in a dark ritual.",
					KeyEvents:      []string{"Rival identified as Anna", "Jewel's ritual purpose discovered"},
				},
			},
			LatestSummary: &promptpkg.SceneSummary{
				NarrativeProse:    "Cynere and Anna joined forces and destroyed the Jewel rather than let it fall into the Cult's hands. The Collegia begrudgingly accepted this outcome since the alternative was catastrophic. The rival factions' race is over.",
				KeyEvents:         []string{"Cynere and Anna allied", "Jewel destroyed", "Collegia stood down"},
				UnresolvedThreads: []string{},
			},
			PlayerName:     "Cynere",
			PlayerAspects:  []string{"Infamous Girl With Sword", "Tempted by Shiny Things"},
			ExpectResolved: true,
			Description:    "Problem made moot by destroying the MacGuffin — a valid Fate Core resolution",
		},
	}
}

// getUnresolvedTestCases returns scenarios that should NOT be marked resolved
func getUnresolvedTestCases() []ScenarioResTestCase {
	return []ScenarioResTestCase{
		{
			Name: "Only first scene completed",
			Scenario: &scene.Scenario{
				Title:   "Shadow Over Tombstone",
				Problem: "A mysterious plague is spreading through Tombstone, and the local doctor has gone missing.",
				StoryQuestions: []string{
					"Can Jesse find the missing doctor?",
					"Will Jesse discover the source of the plague?",
					"Can the town be saved before the plague spreads?",
				},
			},
			LatestSummary: &promptpkg.SceneSummary{
				NarrativeProse:    "Jesse arrived in Tombstone and saw the effects of the plague firsthand. He spoke to the sheriff who mentioned the doctor was last seen heading toward the old cemetery.",
				KeyEvents:         []string{"Jesse saw plague victims", "Sheriff mentioned cemetery"},
				UnresolvedThreads: []string{"Doctor's whereabouts unknown", "Source of plague unknown"},
			},
			PlayerName:     "Jesse Calhoun",
			PlayerAspects:  []string{"Vengeful Rancher", "Old Friends in Low Places"},
			ExpectResolved: false,
			Description:    "Problem is still active, no story questions answered, significant threads remain",
		},
		{
			Name: "Partial progress with key questions unanswered",
			Scenario: &scene.Scenario{
				Title:   "The Arcane Conspiracy",
				Problem: "A conspiracy within the Collegia Arcana threatens to unleash forbidden magic that could destroy the city.",
				StoryQuestions: []string{
					"Can Zird identify the conspirators?",
					"Will Zird find an ally within the Collegia?",
					"Can the forbidden ritual be stopped before completion?",
				},
			},
			SceneSummaries: []promptpkg.SceneSummary{
				{
					NarrativeProse: "Zird discovered evidence of the conspiracy in the Collegia library but was discovered snooping. He narrowly escaped arrest.",
					KeyEvents:      []string{"Found conspiracy evidence", "Nearly arrested"},
				},
			},
			LatestSummary: &promptpkg.SceneSummary{
				NarrativeProse:    "Zird found a potential ally in Librarian Kael, but isn't sure if she can be trusted. The conspirators are still unknown, and the ritual preparations continue.",
				KeyEvents:         []string{"Met Librarian Kael", "Ritual preparations ongoing"},
				UnresolvedThreads: []string{"Conspirators still unidentified", "Ritual timeline unknown", "Kael's trustworthiness unclear"},
			},
			PlayerName:     "Zird the Arcane",
			PlayerAspects:  []string{"Wizard Detective", "Rivals in the Collegia Arcana"},
			ExpectResolved: false,
			Description:    "Central problem still active, conspirators unknown, ritual still in progress",
		},
		{
			Name: "One question answered but problem still urgent",
			Scenario: &scene.Scenario{
				Title:   "The Cortez Gang Rides Again",
				Problem: "The Cortez Gang has kidnapped the mayor's daughter and demands the town's gold reserves as ransom.",
				StoryQuestions: []string{
					"Can Jesse locate where the gang is holding the hostage?",
					"Will Jesse be able to rescue her without paying the ransom?",
					"Can Jesse bring El Diablo Cortez to justice?",
				},
			},
			SceneSummaries: []promptpkg.SceneSummary{
				{
					NarrativeProse: "Jesse tracked the gang to an abandoned mine south of town. He confirmed the hostage is alive but heavily guarded.",
					KeyEvents:      []string{"Gang tracked to abandoned mine", "Hostage confirmed alive"},
				},
			},
			LatestSummary: &promptpkg.SceneSummary{
				NarrativeProse:    "Jesse knows the mine location but his first rescue attempt was repelled. Cortez sent a warning that the next attempt will cost the hostage her life. Time is running out.",
				KeyEvents:         []string{"Rescue attempt failed", "Cortez issued ultimatum"},
				UnresolvedThreads: []string{"Hostage still captive", "Cortez at large", "Ransom deadline approaching"},
			},
			PlayerName:     "Jesse Calhoun",
			PlayerAspects:  []string{"Vengeful Rancher", "Quick Draw"},
			ExpectResolved: false,
			Description:    "Hostage located (Q1 answered YES) but rescue and justice questions unanswered, problem still urgent",
		},
	}
}

// TestScenarioResolution_LLMEvaluation verifies the LLM correctly identifies resolved vs unresolved scenarios.
//
// Per Fate Core SRD (https://fate-srd.com/fate-core/defining-scenarios):
// - "Once the problem is resolved (or it can no longer be resolved), the scenario is over"
// - Resolution can mean: solving the problem, circumventing it, or the situation changing
// - Story questions should have clear yes/no answers when resolved
// - The LLM should be conservative — only mark resolved with clear closure
func TestScenarioResolution_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := llmazure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := llmazure.NewClient(*config)
	ctx := context.Background()

	verboseLogging := os.Getenv("VERBOSE") == "1"

	allCategories := []struct {
		category string
		cases    []ScenarioResTestCase
	}{
		{"ShouldBeResolved", getResolvedTestCases()},
		{"ShouldNotBeResolved", getUnresolvedTestCases()},
	}

	var results []ScenarioResResult

	resolvedResults := struct{ total, correct int }{}
	unresolvedResults := struct{ total, correct int }{}

	for _, category := range allCategories {
		t.Run(category.category, func(t *testing.T) {
			for _, tc := range category.cases {
				t.Run(tc.Name, func(t *testing.T) {
					result := evaluateScenarioResolution(ctx, client, tc)
					results = append(results, result)

					if result.Error != nil {
						t.Errorf("Error: %v", result.Error)
						return
					}

					if tc.ExpectResolved {
						resolvedResults.total++
						if result.Matches {
							resolvedResults.correct++
						}
					} else {
						unresolvedResults.total++
						if result.Matches {
							unresolvedResults.correct++
						}
					}

					if verboseLogging || !result.Matches {
						status := "✓ PASS"
						if !result.Matches {
							status = "✗ FAIL"
						}
						t.Logf("%s: Expected resolved=%v, Got resolved=%v",
							status, tc.ExpectResolved, result.GotResolved)
						t.Logf("  Scenario: %s", tc.Scenario.Title)
						t.Logf("  Why: %s", tc.Description)
						if result.Parsed != nil {
							t.Logf("  Reasoning: %s", result.Parsed.Reasoning)
							for _, aq := range result.Parsed.AnsweredQuestions {
								t.Logf("    %s", aq)
							}
						}
						if !result.Matches {
							t.Logf("  Raw: %s", truncateResolutionText(result.RawResponse, 300))
						}
					}

					assert.True(t, result.ValidJSON, "Response should be valid JSON")
					assert.Equal(t, tc.ExpectResolved, result.GotResolved,
						"Resolution mismatch for '%s': %s", tc.Name, tc.Description)
					assert.True(t, result.HasReasoning,
						"Response should include reasoning for the resolution decision")
				})
			}
		})
	}

	// Summary
	t.Log("\n========== SCENARIO RESOLUTION SUMMARY ==========")
	if resolvedResults.total > 0 {
		t.Logf("Resolved cases (should be resolved): %d/%d (%.1f%%)",
			resolvedResults.correct, resolvedResults.total,
			float64(resolvedResults.correct)*100/float64(resolvedResults.total))
	}
	if unresolvedResults.total > 0 {
		t.Logf("Unresolved cases (should NOT be resolved): %d/%d (%.1f%%)",
			unresolvedResults.correct, unresolvedResults.total,
			float64(unresolvedResults.correct)*100/float64(unresolvedResults.total))
	}
	totalCorrect := resolvedResults.correct + unresolvedResults.correct
	totalTests := resolvedResults.total + unresolvedResults.total
	if totalTests > 0 {
		t.Logf("Overall accuracy: %d/%d (%.1f%%)",
			totalCorrect, totalTests,
			float64(totalCorrect)*100/float64(totalTests))
	}

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: %s — Expected resolved=%v, Got=%v",
				r.TestCase.Name, r.TestCase.ExpectResolved, r.GotResolved)
			t.Logf("      %s", r.TestCase.Description)
		}
	}
}

// ScenarioResResult stores a scenario resolution evaluation result
type ScenarioResResult struct {
	TestCase     ScenarioResTestCase
	RawResponse  string
	Parsed       *promptpkg.ScenarioResolutionResult
	ValidJSON    bool
	GotResolved  bool
	HasReasoning bool
	Matches      bool
	Error        error
}

func evaluateScenarioResolution(ctx context.Context, client llm.LLMClient, tc ScenarioResTestCase) ScenarioResResult {
	data := promptpkg.ScenarioResolutionData{
		Scenario:       tc.Scenario,
		SceneSummaries: tc.SceneSummaries,
		LatestSummary:  tc.LatestSummary,
		PlayerName:     tc.PlayerName,
		PlayerAspects:  tc.PlayerAspects,
	}

	prompt, err := promptpkg.RenderScenarioResolution(data)
	if err != nil {
		return ScenarioResResult{TestCase: tc, Error: err}
	}

	resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   600,
		Temperature: 0.2, // Low temperature for consistent judgment
	})
	if err != nil {
		return ScenarioResResult{TestCase: tc, Error: err}
	}

	if len(resp.Choices) == 0 {
		return ScenarioResResult{TestCase: tc, Error: err}
	}

	raw := resp.Choices[0].Message.Content
	result := ScenarioResResult{
		TestCase:    tc,
		RawResponse: raw,
	}

	// Use production JSON cleaning to prevent parsing divergence
	cleaned := llm.CleanJSONResponse(raw)

	var parsed promptpkg.ScenarioResolutionResult
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		result.ValidJSON = false
		return result
	}
	result.ValidJSON = true
	result.Parsed = &parsed
	result.GotResolved = parsed.IsResolved
	result.HasReasoning = parsed.Reasoning != "" && len(parsed.Reasoning) > 10
	result.Matches = parsed.IsResolved == tc.ExpectResolved

	return result
}

func truncateResolutionText(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
