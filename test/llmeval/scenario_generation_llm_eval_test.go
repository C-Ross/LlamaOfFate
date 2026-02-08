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
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ScenarioGenTestCase represents a test case for scenario generation
type ScenarioGenTestCase struct {
	Name              string
	PlayerName        string
	PlayerHighConcept string
	PlayerTrouble     string
	PlayerAspects     []string
	Genre             string
	Theme             string
	Description       string // Why this case is interesting
}

// ScenarioGenResult stores the result of a scenario generation evaluation
type ScenarioGenResult struct {
	TestCase         ScenarioGenTestCase
	RawResponse      string
	Parsed           *scene.Scenario
	ValidJSON        bool
	HasTitle         bool
	HasProblem       bool
	HasStoryQs       bool
	StoryQsCount     int
	StoryQsCanWill   bool // Story questions use Can/Will format
	ProblemIsUrgent  bool // Problem references urgency or consequence
	DrawsFromAspects bool // Problem or questions reference character aspects
	Passed           bool
	Error            error
}

func getScenarioGenTestCases() []ScenarioGenTestCase {
	return []ScenarioGenTestCase{
		{
			Name:              "Western outlaw character",
			PlayerName:        "Jesse Calhoun",
			PlayerHighConcept: "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:     "The Cortez Gang Burned My Life",
			PlayerAspects:     []string{"Quick Draw", "Old Friends in Low Places"},
			Genre:             "Western",
			Description:       "Classic western character with clear revenge motivation",
		},
		{
			Name:              "Fantasy wizard",
			PlayerName:        "Zird the Arcane",
			PlayerHighConcept: "Wizard Detective on the Trail of Forbidden Knowledge",
			PlayerTrouble:     "The Lure of Ancient Mysteries",
			PlayerAspects:     []string{"Rivals in the Collegia Arcana", "If I Haven't Been There, I've Read About It"},
			Genre:             "Fantasy",
			Description:       "Fantasy character with academic rivals and curiosity-driven trouble",
		},
		{
			Name:              "Cyberpunk hacker",
			PlayerName:        "Neon",
			PlayerHighConcept: "Underground Netrunner with a Conscience",
			PlayerTrouble:     "MegaCorp Has My Sister",
			PlayerAspects:     []string{"Ghost in the Machine", "I Know Where the Bodies Are Buried"},
			Genre:             "Cyberpunk",
			Description:       "Character with clear personal stakes and leverage",
		},
		{
			Name:              "Minimal character info",
			PlayerName:        "Ash",
			PlayerHighConcept: "Wandering Swordsman",
			PlayerTrouble:     "Haunted by the Past",
			Genre:             "Fantasy",
			Description:       "Character with minimal aspects — should still generate a valid scenario",
		},
		{
			Name:              "Character with theme hint",
			PlayerName:        "Detective Morgan",
			PlayerHighConcept: "Hard-Boiled Private Eye Who's Seen It All",
			PlayerTrouble:     "The Bottle Is My Only Friend",
			PlayerAspects:     []string{"Knows Every Snitch in Town", "Ex-Cop with a Badge-Shaped Scar"},
			Genre:             "Noir",
			Theme:             "corruption in high places",
			Description:       "Theme hint should influence the scenario's direction",
		},
	}
}

func evaluateScenarioGeneration(ctx context.Context, client llm.LLMClient, tc ScenarioGenTestCase) ScenarioGenResult {
	data := promptpkg.ScenarioGenerationData{
		PlayerName:        tc.PlayerName,
		PlayerHighConcept: tc.PlayerHighConcept,
		PlayerTrouble:     tc.PlayerTrouble,
		PlayerAspects:     tc.PlayerAspects,
		Genre:             tc.Genre,
		Theme:             tc.Theme,
	}

	prompt, err := promptpkg.RenderScenarioGeneration(data)
	if err != nil {
		return ScenarioGenResult{TestCase: tc, Error: err}
	}

	resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   800,
		Temperature: 0.3,
	})
	if err != nil {
		return ScenarioGenResult{TestCase: tc, Error: err}
	}

	if len(resp.Choices) == 0 {
		return ScenarioGenResult{TestCase: tc, Error: err}
	}

	raw := resp.Choices[0].Message.Content

	// Strip markdown code fences if present
	cleaned := raw
	if idx := strings.Index(cleaned, "```json"); idx != -1 {
		cleaned = cleaned[idx+7:]
	} else if idx := strings.Index(cleaned, "```"); idx != -1 {
		cleaned = cleaned[idx+3:]
	}
	if idx := strings.LastIndex(cleaned, "```"); idx != -1 {
		cleaned = cleaned[:idx]
	}
	cleaned = strings.TrimSpace(cleaned)

	result := ScenarioGenResult{
		TestCase:    tc,
		RawResponse: raw,
	}

	// Parse JSON
	var scenario scene.Scenario
	if err := json.Unmarshal([]byte(cleaned), &scenario); err != nil {
		result.ValidJSON = false
		return result
	}
	result.ValidJSON = true
	result.Parsed = &scenario

	// Validate fields
	result.HasTitle = scenario.Title != "" && len(strings.Fields(scenario.Title)) >= 2
	result.HasProblem = scenario.Problem != "" && len(scenario.Problem) > 20
	result.HasStoryQs = len(scenario.StoryQuestions) >= 2
	result.StoryQsCount = len(scenario.StoryQuestions)

	// Check story questions use "Can" or "Will" format per Fate Core
	if len(scenario.StoryQuestions) > 0 {
		canWillCount := 0
		for _, q := range scenario.StoryQuestions {
			q = strings.TrimSpace(q)
			if strings.HasPrefix(q, "Can ") || strings.HasPrefix(q, "Will ") {
				canWillCount++
			}
		}
		// At least most should use Can/Will format
		result.StoryQsCanWill = canWillCount >= len(scenario.StoryQuestions)/2+1
	}

	// Check that the problem or questions reference character aspects
	aspectTerms := collectAspectTerms(tc)
	problemLower := strings.ToLower(scenario.Problem)
	questionsLower := strings.ToLower(strings.Join(scenario.StoryQuestions, " "))
	allText := problemLower + " " + questionsLower

	for _, term := range aspectTerms {
		if strings.Contains(allText, strings.ToLower(term)) {
			result.DrawsFromAspects = true
			break
		}
	}

	// Check problem urgency — should imply consequences or time pressure
	urgencyTerms := []string{
		"before", "must", "threat", "danger", "deadline", "urgent",
		"attack", "destroy", "kill", "steal", "kidnap", "taken", "missing",
		"war", "invasion", "collapse", "burn", "death", "die", "corrupt",
		"demands", "approaching", "imminent", "rising",
	}
	for _, term := range urgencyTerms {
		if strings.Contains(problemLower, term) {
			result.ProblemIsUrgent = true
			break
		}
	}

	// Overall pass: valid JSON with required fields and Fate Core compliance
	result.Passed = result.ValidJSON &&
		result.HasTitle &&
		result.HasProblem &&
		result.HasStoryQs &&
		result.StoryQsCount >= 2 && result.StoryQsCount <= 4 &&
		result.StoryQsCanWill

	return result
}

// collectAspectTerms extracts key terms from character aspects for matching
func collectAspectTerms(tc ScenarioGenTestCase) []string {
	var terms []string
	// Extract key nouns from aspects
	for _, aspect := range []string{tc.PlayerHighConcept, tc.PlayerTrouble} {
		if aspect == "" {
			continue
		}
		words := strings.Fields(aspect)
		for _, w := range words {
			w = strings.Trim(w, ",.!?'\"")
			// Skip short/common words
			if len(w) > 3 && !isCommonWord(w) {
				terms = append(terms, w)
			}
		}
	}
	for _, a := range tc.PlayerAspects {
		words := strings.Fields(a)
		for _, w := range words {
			w = strings.Trim(w, ",.!?'\"")
			if len(w) > 3 && !isCommonWord(w) {
				terms = append(terms, w)
			}
		}
	}
	// Also add the player name
	terms = append(terms, tc.PlayerName)
	return terms
}

func isCommonWord(w string) bool {
	common := map[string]bool{
		"with": true, "that": true, "this": true, "from": true,
		"have": true, "been": true, "they": true, "their": true,
		"what": true, "when": true, "where": true, "which": true,
		"there": true, "about": true, "would": true, "could": true,
		"should": true, "only": true, "very": true, "just": true,
		"than": true, "then": true, "also": true, "into": true,
		"some": true, "more": true, "other": true, "most": true,
		"Nothing": true, "nothing": true, "Left": true, "left": true,
	}
	return common[strings.ToLower(w)]
}

// TestScenarioGeneration_LLMEvaluation verifies scenario generation produces valid Fate Core scenarios.
//
// Per Fate Core SRD (https://fate-srd.com/fate-core/defining-scenarios):
// - A scenario presents "some kind of big, urgent, open-ended problem"
// - Story questions are yes/no, in "Can/Will X accomplish Y?" format
// - Problems should draw from character aspects, especially Trouble
// - 2-4 story questions that add nuance before the final resolution question
// - The problem should have no single "right" solution
func TestScenarioGeneration_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	verboseLogging := os.Getenv("VERBOSE") == "1"
	testCases := getScenarioGenTestCases()

	var results []ScenarioGenResult
	passed := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateScenarioGeneration(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.Passed {
				passed++
			}

			if verboseLogging || !result.Passed {
				status := "✓ PASS"
				if !result.Passed {
					status = "✗ FAIL"
				}
				t.Logf("%s: %s", status, tc.Name)
				t.Logf("  Character: %s (%s)", tc.PlayerName, tc.PlayerHighConcept)
				t.Logf("  ValidJSON=%v Title=%v Problem=%v StoryQs=%d CanWill=%v Urgent=%v AspectsRef=%v",
					result.ValidJSON, result.HasTitle, result.HasProblem,
					result.StoryQsCount, result.StoryQsCanWill, result.ProblemIsUrgent, result.DrawsFromAspects)
				if result.Parsed != nil {
					t.Logf("  Title: %s", result.Parsed.Title)
					t.Logf("  Problem: %s", truncateScenarioText(result.Parsed.Problem, 150))
					for i, q := range result.Parsed.StoryQuestions {
						t.Logf("  Q%d: %s", i+1, q)
					}
				}
				if !result.ValidJSON {
					t.Logf("  Raw: %s", truncateScenarioText(result.RawResponse, 300))
				}
			}

			assert.True(t, result.ValidJSON, "Response should be valid JSON")
			assert.True(t, result.HasTitle, "Scenario should have a title (2+ words)")
			assert.True(t, result.HasProblem, "Scenario should have a substantive problem")
			assert.True(t, result.HasStoryQs, "Scenario should have at least 2 story questions")
			assert.LessOrEqual(t, result.StoryQsCount, 4, "Scenario should have at most 4 story questions")
			assert.True(t, result.StoryQsCanWill,
				"Per Fate Core, story questions should use 'Can/Will' format")
		})
	}

	// Summary
	t.Log("\n========== SCENARIO GENERATION SUMMARY ==========")
	t.Logf("Passed: %d/%d (%.1f%%)", passed, len(testCases), float64(passed)*100/float64(len(testCases)))

	// Aspect reference stats
	aspectRefCount := 0
	urgentCount := 0
	for _, r := range results {
		if r.DrawsFromAspects {
			aspectRefCount++
		}
		if r.ProblemIsUrgent {
			urgentCount++
		}
	}
	t.Logf("Problems referencing character aspects: %d/%d", aspectRefCount, len(results))
	t.Logf("Problems with urgency language: %d/%d", urgentCount, len(results))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Passed {
			t.Logf("FAIL: %s — ValidJSON=%v Title=%v Problem=%v StoryQs=%d CanWill=%v",
				r.TestCase.Name, r.ValidJSON, r.HasTitle, r.HasProblem,
				r.StoryQsCount, r.StoryQsCanWill)
		}
	}
}

func truncateScenarioText(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
