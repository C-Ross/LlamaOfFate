//go:build llmeval

package llmeval_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
)

// SceneGenTestCase represents a test case for scene generation
type SceneGenTestCase struct {
	Name              string
	TransitionHint    string
	Scenario          *scene.Scenario
	PlayerName        string
	PlayerHighConcept string
	PlayerTrouble     string
	PlayerAspects     []string
	PreviousSummaries []promptpkg.SceneSummary
	Complications     []string // Unresolved threads to weave in as complications
	Description       string
}

// SceneGenResult stores the result of a scene generation evaluation
type SceneGenResult struct {
	TestCase            SceneGenTestCase
	RawResponse         string
	Parsed              *promptpkg.GeneratedScene
	ValidJSON           bool
	HasSceneName        bool
	SceneNameLength     int // word count
	HasDescription      bool
	HasPurpose          bool
	HasOpeningHook      bool
	HasSituationAspects bool
	SituationAspectCt   int
	NPCCount            int
	NPCsHaveHighConc    bool
	NPCsHaveDisp        bool
	AdvancesStory       bool // Description or aspects relate to scenario
	Passed              bool
	Error               error
}

func getSceneGenTestCases() []SceneGenTestCase {
	westernScenario := &scene.Scenario{
		Title:   "The Cortez Gang's Last Stand",
		Problem: "The Cortez Gang is terrorizing Redemption Gulch and threatening to burn it down.",
		StoryQuestions: []string{
			"Can Jesse discover where the Cortez Gang is hiding?",
			"Will the townspeople rally behind Jesse?",
			"Can Jesse defeat El Diablo Cortez?",
		},
		Setting: "The dusty frontier town of Redemption Gulch in the American West, 1878.",
		Genre:   "Western",
	}

	fantasyScenario := &scene.Scenario{
		Title:   "The Arcane Conspiracy",
		Problem: "A conspiracy within the Collegia Arcana threatens to unleash forbidden magic.",
		StoryQuestions: []string{
			"Can Zird identify the conspirators?",
			"Will Zird find an ally within the Collegia?",
			"Can the forbidden ritual be stopped?",
		},
		Setting: "A bustling medieval city where magic is regulated by the Collegia Arcana.",
		Genre:   "Fantasy",
	}

	return []SceneGenTestCase{
		{
			Name:              "First scene of western scenario",
			TransitionHint:    "",
			Scenario:          westernScenario,
			PlayerName:        "Jesse Calhoun",
			PlayerHighConcept: "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:     "The Cortez Gang Burned My Life",
			PlayerAspects:     []string{"Quick Draw", "Old Friends in Low Places"},
			Description:       "Opening scene should establish the scenario setting and hook the player in",
		},
		{
			Name:              "Transition to specific location",
			TransitionHint:    "the old silver mine south of town",
			Scenario:          westernScenario,
			PlayerName:        "Jesse Calhoun",
			PlayerHighConcept: "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:     "The Cortez Gang Burned My Life",
			PlayerAspects:     []string{"Quick Draw"},
			PreviousSummaries: []promptpkg.SceneSummary{
				{
					NarrativeProse:    "Jesse arrived in Redemption Gulch and learned from Maggie at the saloon that the gang hides in the old silver mine.",
					KeyEvents:         []string{"Arrived in town", "Gang location discovered"},
					UnresolvedThreads: []string{"Gang still at mine"},
				},
			},
			Description: "Scene at specified transition destination should match the hint",
		},
		{
			Name:              "Fantasy scene with previous context",
			TransitionHint:    "the Collegia library",
			Scenario:          fantasyScenario,
			PlayerName:        "Zird the Arcane",
			PlayerHighConcept: "Wizard Detective on the Trail of Forbidden Knowledge",
			PlayerTrouble:     "The Lure of Ancient Mysteries",
			PlayerAspects:     []string{"Rivals in the Collegia Arcana"},
			PreviousSummaries: []promptpkg.SceneSummary{
				{
					NarrativeProse:    "Zird discovered coded messages in his colleague's office hinting at forbidden research. A rival wizard spotted him snooping.",
					KeyEvents:         []string{"Found coded messages", "Spotted by rival"},
					UnresolvedThreads: []string{"Coded messages not yet deciphered", "Rival may report him"},
				},
			},
			Description: "Scene should build on unresolved threads from previous scene",
		},
		{
			Name:              "Scene without transition hint (new adventure start)",
			Scenario:          westernScenario,
			PlayerName:        "Jesse Calhoun",
			PlayerHighConcept: "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:     "The Cortez Gang Burned My Life",
			Description:       "Without a transition hint, should generate an appropriate opening scene",
		},
		{
			Name:              "Complications from unresolved threads",
			TransitionHint:    "the town square",
			Scenario:          westernScenario,
			PlayerName:        "Jesse Calhoun",
			PlayerHighConcept: "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:     "The Cortez Gang Burned My Life",
			PlayerAspects:     []string{"Quick Draw"},
			PreviousSummaries: []promptpkg.SceneSummary{
				{
					NarrativeProse:    "Jesse learned the gang's lieutenant is in town. The sheriff was found dead.",
					KeyEvents:         []string{"Sheriff found dead", "Lieutenant spotted"},
					UnresolvedThreads: []string{"The gang lieutenant is still at large", "Who killed the sheriff?"},
				},
			},
			Complications: []string{"The gang lieutenant is still at large", "Who killed the sheriff?"},
			Description:   "Scene should incorporate unresolved threads as complications",
		},
	}
}

func evaluateSceneGeneration(ctx context.Context, client llm.LLMClient, tc SceneGenTestCase) SceneGenResult {
	data := promptpkg.SceneGenerationData{
		TransitionHint:    tc.TransitionHint,
		Scenario:          tc.Scenario,
		PlayerName:        tc.PlayerName,
		PlayerHighConcept: tc.PlayerHighConcept,
		PlayerTrouble:     tc.PlayerTrouble,
		PlayerAspects:     tc.PlayerAspects,
		PreviousSummaries: tc.PreviousSummaries,
		Complications:     tc.Complications,
	}

	prompt, err := promptpkg.RenderSceneGeneration(data)
	if err != nil {
		return SceneGenResult{TestCase: tc, Error: err}
	}

	raw, err := llm.SimpleCompletion(ctx, client, prompt, 800, 0.3)
	if err != nil {
		return SceneGenResult{TestCase: tc, Error: err}
	}
	result := SceneGenResult{
		TestCase:    tc,
		RawResponse: raw,
	}

	generated, err := promptpkg.ParseGeneratedScene(raw)
	if err != nil {
		result.ValidJSON = false
		return result
	}
	result.ValidJSON = true
	result.Parsed = generated

	// Validate scene_name: 2-5 words per template rules
	nameWords := len(strings.Fields(generated.SceneName))
	result.HasSceneName = generated.SceneName != "" && nameWords >= 2
	result.SceneNameLength = nameWords

	// Validate description: immersive, 2-3 sentences
	result.HasDescription = generated.Description != "" && len(generated.Description) > 30

	// Validate purpose: required, should be a dramatic question or goal
	result.HasPurpose = generated.Purpose != "" && len(generated.Purpose) > 10

	// Validate opening_hook: optional but preferred
	result.HasOpeningHook = generated.OpeningHook != ""

	// Validate situation_aspects: 2-3 per Fate Core and template rules
	result.SituationAspectCt = len(generated.SituationAspects)
	result.HasSituationAspects = result.SituationAspectCt >= 2 && result.SituationAspectCt <= 3

	// Validate NPCs
	result.NPCCount = len(generated.NPCs)
	if result.NPCCount > 0 {
		allHaveHC := true
		allHaveDisp := true
		for _, npc := range generated.NPCs {
			if npc.HighConcept == "" {
				allHaveHC = false
			}
			if npc.Disposition == "" {
				allHaveDisp = false
			} else {
				validDisp := npc.Disposition == "friendly" || npc.Disposition == "neutral" || npc.Disposition == "hostile"
				if !validDisp {
					allHaveDisp = false
				}
			}
		}
		result.NPCsHaveHighConc = allHaveHC
		result.NPCsHaveDisp = allHaveDisp
	} else {
		// No NPCs is valid
		result.NPCsHaveHighConc = true
		result.NPCsHaveDisp = true
	}

	// Check if scene advances the scenario story using LLM judge
	if tc.Scenario != nil {
		sceneText := generated.SceneName + "\n" + generated.Description + "\n" + strings.Join(generated.SituationAspects, "\n")
		scenarioContext := fmt.Sprintf("Scenario: %s\nProblem: %s\nStory Questions: %s",
			tc.Scenario.Title, tc.Scenario.Problem, strings.Join(tc.Scenario.StoryQuestions, "; "))
		judge, err := LLMJudgeWithContext(ctx, client, sceneText,
			"Does this scene content (name, description, and situation aspects) relate to or advance the scenario's story by incorporating elements from the scenario's problem or story questions?",
			scenarioContext)
		if err == nil {
			result.AdvancesStory = judge.Pass
		}
	}

	// Overall pass
	result.Passed = result.ValidJSON &&
		result.HasSceneName &&
		result.HasDescription &&
		result.HasPurpose &&
		result.HasSituationAspects &&
		result.NPCsHaveHighConc &&
		result.NPCsHaveDisp &&
		result.NPCCount <= 2

	return result
}

// TestSceneGeneration_LLMEvaluation verifies scene generation produces valid Fate Core scenes.
//
// Per Fate Core SRD (https://fate-srd.com/fate-core/defining-scenes):
// - Scenes have a clear purpose (dramatic question or action to resolve)
// - Include situation aspects that players can interact with (invoke/compel)
// - NPCs introduced only when dramatically appropriate
// - Descriptions should be evocative but concise
// - Scenes should advance the scenario's story questions
func TestSceneGeneration_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	verboseLogging := VerboseLoggingEnabled()
	testCases := getSceneGenTestCases()

	var results []SceneGenResult
	passed := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateSceneGeneration(ctx, client, tc)
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
				t.Logf("  ValidJSON=%v Name=%v(%d words) Desc=%v Purpose=%v Hook=%v Aspects=%d NPCs=%d",
					result.ValidJSON, result.HasSceneName, result.SceneNameLength,
					result.HasDescription, result.HasPurpose, result.HasOpeningHook,
					result.SituationAspectCt, result.NPCCount)
				if result.Parsed != nil {
					t.Logf("  Scene: %s", result.Parsed.SceneName)
					t.Logf("  Purpose: %s", truncateSceneGenText(result.Parsed.Purpose, 150))
					t.Logf("  Description: %s", truncateSceneGenText(result.Parsed.Description, 150))
					if result.Parsed.OpeningHook != "" {
						t.Logf("  Opening Hook: %s", truncateSceneGenText(result.Parsed.OpeningHook, 150))
					}
					for i, a := range result.Parsed.SituationAspects {
						t.Logf("  Aspect %d: %s", i+1, a)
					}
					for _, npc := range result.Parsed.NPCs {
						t.Logf("  NPC: %s (%s, %s)", npc.Name, npc.HighConcept, npc.Disposition)
					}
				}
				if !result.ValidJSON {
					t.Logf("  Raw: %s", truncateSceneGenText(result.RawResponse, 300))
				}
			}

			assert.True(t, result.ValidJSON, "Response should be valid JSON")
			assert.True(t, result.HasSceneName, "Scene should have a name (2+ words)")
			assert.True(t, result.HasDescription, "Scene should have a substantive description")
			assert.True(t, result.HasPurpose,
				"Scene should have a purpose (dramatic question) per Fate Core scene guidelines")
			assert.True(t, result.HasSituationAspects,
				"Scene should have 2-3 situation aspects per Fate Core (got %d)", result.SituationAspectCt)
			assert.LessOrEqual(t, result.NPCCount, 2,
				"Scene should have at most 2 NPCs per template rules")
			if result.NPCCount > 0 {
				assert.True(t, result.NPCsHaveHighConc, "NPCs should have high concepts")
				assert.True(t, result.NPCsHaveDisp,
					"NPCs should have valid disposition (friendly/neutral/hostile)")
			}
		})
	}

	// Summary
	t.Log("\n========== SCENE GENERATION SUMMARY ==========")
	t.Logf("Passed: %d/%d (%.1f%%)", passed, len(testCases), float64(passed)*100/float64(len(testCases)))

	advancesCount := 0
	for _, r := range results {
		if r.AdvancesStory {
			advancesCount++
		}
	}
	t.Logf("Scenes advancing scenario story: %d/%d", advancesCount, len(results))

	hookCount := 0
	for _, r := range results {
		if r.HasOpeningHook {
			hookCount++
		}
	}
	t.Logf("Scenes with opening hooks: %d/%d", hookCount, len(results))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Passed {
			t.Logf("FAIL: %s — ValidJSON=%v Name=%v Desc=%v Purpose=%v Aspects=%d NPCs=%d",
				r.TestCase.Name, r.ValidJSON, r.HasSceneName, r.HasDescription,
				r.HasPurpose, r.SituationAspectCt, r.NPCCount)
		}
	}
}

func truncateSceneGenText(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
