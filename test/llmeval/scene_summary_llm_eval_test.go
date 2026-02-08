//go:build llmeval

package llmeval_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SceneSummaryTestCase represents a test case for scene summary generation
type SceneSummaryTestCase struct {
	Name        string
	Data        promptpkg.SceneSummaryData
	Description string
}

// SceneSummaryResult stores the result of a scene summary evaluation
type SceneSummaryResult struct {
	TestCase          SceneSummaryTestCase
	RawResponse       string
	Parsed            *promptpkg.SceneSummary
	ValidJSON         bool
	HasNarrativeProse bool
	HasKeyEvents      bool
	KeyEventCount     int
	HasHowEnded       bool
	NPCsMatch         bool // NPCs encountered match those in the scene
	HasUnresolved     bool // Has unresolved threads when expected
	ProseIsNarrative  bool // Reads like a recap, not a list
	Passed            bool
	Error             error
}

func getSceneSummaryTestCases() []SceneSummaryTestCase {
	return []SceneSummaryTestCase{
		{
			Name: "Saloon conversation with NPC",
			Data: promptpkg.SceneSummaryData{
				SceneName:        "The Dusty Spur Saloon",
				SceneDescription: "A dimly lit saloon with swinging doors, a long bar, and the smell of whiskey.",
				SituationAspects: []string{"Dim Flickering Lamplight", "Rowdy Crowd"},
				ConversationHistory: []promptpkg.ConversationEntry{
					{PlayerInput: "Jesse walks in and sits at the bar", GMResponse: "Maggie slides a glass toward you. 'New in town, stranger?'"},
					{PlayerInput: "Jesse nods. 'Looking for information about the Cortez Gang.'", GMResponse: "Maggie's face darkens. 'You don't want to go asking about them too loudly. They've got ears everywhere.' She leans in and whispers, 'Try the old mine south of here.'"},
					{PlayerInput: "Jesse tips his hat and walks out.", GMResponse: "Maggie watches you leave with a concerned look. The swinging doors creak shut behind you. [SCENE_TRANSITION:old mine south of town]"},
				},
				NPCsInScene:    []promptpkg.NPCSummary{{Name: "Maggie Two-Rivers", Attitude: "friendly"}},
				TakenOutChars:  []string{},
				HowEnded:       "transition",
				TransitionHint: "old mine south of town",
			},
			Description: "Simple conversation scene should capture key info exchange and transition",
		},
		{
			Name: "Combat scene with taken out NPC",
			Data: promptpkg.SceneSummaryData{
				SceneName:        "Ambush at Dead Man's Creek",
				SceneDescription: "A narrow creek crossing flanked by rocky outcrops — perfect ambush territory.",
				SituationAspects: []string{"Rocky Cover", "Rushing Water", "Narrow Crossing"},
				ConversationHistory: []promptpkg.ConversationEntry{
					{PlayerInput: "Jesse approaches the creek cautiously", GMResponse: "You hear the click of a rifle being cocked. 'That's far enough, Calhoun!' A voice calls from the rocks above."},
					{PlayerInput: "Jesse dives behind a boulder and draws his pistol", GMResponse: "Bullets chip the rock near your head. There are two bandits — one on each side of the creek. The one on the left is reloading."},
					{PlayerInput: "Jesse fires at the one reloading", GMResponse: "Your shot hits true. The bandit on the left collapses. The other one shouts a curse and starts retreating upstream."},
					{PlayerInput: "Jesse chases after the retreating bandit", GMResponse: "You pursue through the shallows. The bandit stumbles and you catch up. He throws down his weapons. 'Don't shoot! Cortez sent us! He knows you're coming!'"},
				},
				NPCsInScene:    []promptpkg.NPCSummary{{Name: "Creek Bandits", Attitude: "hostile"}},
				TakenOutChars:  []string{"Bandit Left Flank"},
				HowEnded:       "transition",
				TransitionHint: "continuing upstream toward the mine",
			},
			Description: "Combat scene should capture the fight outcome, casualties, and intelligence gained",
		},
		{
			Name: "Scene ending with player quitting",
			Data: promptpkg.SceneSummaryData{
				SceneName:        "The Sheriff's Office",
				SceneDescription: "A small frontier office with a desk, a gun rack, and wanted posters on the walls.",
				SituationAspects: []string{"Wanted Posters Everywhere", "Locked Gun Rack"},
				ConversationHistory: []promptpkg.ConversationEntry{
					{PlayerInput: "Jesse walks in to talk to the sheriff", GMResponse: "Sheriff Daniels looks up from his desk. 'What brings you in, stranger?'"},
					{PlayerInput: "Jesse asks about the Cortez Gang's recent activity", GMResponse: "The sheriff sighs. 'They hit the stagecoach again last week. Took everything, including the payroll for the mine workers. People are getting restless.'"},
				},
				NPCsInScene: []promptpkg.NPCSummary{{Name: "Sheriff Daniels", Attitude: "neutral"}},
				HowEnded:    "quit",
			},
			Description: "Scene ended by player quitting — should still capture what happened",
		},
		{
			Name: "Scene with multiple NPCs and unresolved threads",
			Data: promptpkg.SceneSummaryData{
				SceneName:        "The Mining Camp",
				SceneDescription: "A ramshackle mining camp with tents, pickaxes, and exhausted workers.",
				SituationAspects: []string{"Exhausted Workers", "Unstable Mine Entrance"},
				ConversationHistory: []promptpkg.ConversationEntry{
					{PlayerInput: "Jesse asks the foreman what happened here", GMResponse: "The foreman, Big Pete, wipes his brow. 'Cortez's men came through last night. Took our food supplies and beat up two of my workers.'"},
					{PlayerInput: "Jesse asks if anyone saw where they went", GMResponse: "A young woman steps forward. 'I'm Elena. I saw them head into the upper tunnels. But there's something else — I found this map in one of their saddlebags.' She holds out a weathered map."},
					{PlayerInput: "Jesse takes the map and examines it", GMResponse: "The map shows a network of tunnels beneath the mountain. One chamber is marked with an X and the word 'vault.' There are also strange symbols you don't recognize."},
					{PlayerInput: "Jesse asks Elena about the symbols", GMResponse: "Elena shakes her head. 'I've never seen anything like them. But the old prospector, Hank, might know. He lives in a cabin further up the mountain. Haven't seen him in days though.'"},
				},
				NPCsInScene: []promptpkg.NPCSummary{
					{Name: "Big Pete", Attitude: "friendly"},
					{Name: "Elena", Attitude: "friendly"},
				},
				TakenOutChars:  []string{},
				HowEnded:       "transition",
				TransitionHint: "Hank's cabin up the mountain",
			},
			Description: "Complex scene with multiple NPCs should have multiple unresolved threads (map symbols, Hank's whereabouts, vault)",
		},
	}
}

func evaluateSceneSummary(ctx context.Context, client llm.LLMClient, tc SceneSummaryTestCase) SceneSummaryResult {
	prompt, err := promptpkg.RenderSceneSummary(tc.Data)
	if err != nil {
		return SceneSummaryResult{TestCase: tc, Error: err}
	}

	resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   800,
		Temperature: 0.3,
	})
	if err != nil {
		return SceneSummaryResult{TestCase: tc, Error: err}
	}

	if len(resp.Choices) == 0 {
		return SceneSummaryResult{TestCase: tc, Error: err}
	}

	raw := resp.Choices[0].Message.Content
	result := SceneSummaryResult{
		TestCase:    tc,
		RawResponse: raw,
	}

	// Strip code fences
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

	var summary promptpkg.SceneSummary
	if err := json.Unmarshal([]byte(cleaned), &summary); err != nil {
		result.ValidJSON = false
		return result
	}
	result.ValidJSON = true
	result.Parsed = &summary

	// Validate narrative_prose: should read like a recap, not bullet points
	result.HasNarrativeProse = summary.NarrativeProse != "" && len(summary.NarrativeProse) > 30
	// Prose should not start with bullet points or numbers
	result.ProseIsNarrative = result.HasNarrativeProse &&
		!strings.HasPrefix(strings.TrimSpace(summary.NarrativeProse), "-") &&
		!strings.HasPrefix(strings.TrimSpace(summary.NarrativeProse), "•") &&
		!strings.HasPrefix(strings.TrimSpace(summary.NarrativeProse), "1.")

	// Validate key_events: 1-4 items
	result.KeyEventCount = len(summary.KeyEvents)
	result.HasKeyEvents = result.KeyEventCount >= 1 && result.KeyEventCount <= 4

	// Validate how_ended
	result.HasHowEnded = summary.HowEnded != "" && len(summary.HowEnded) > 5

	// Validate NPCs encountered match those in the scene
	if len(tc.Data.NPCsInScene) > 0 {
		sceneNPCNames := make(map[string]bool)
		for _, npc := range tc.Data.NPCsInScene {
			sceneNPCNames[strings.ToLower(npc.Name)] = true
		}
		if len(summary.NPCsEncountered) > 0 {
			matchCount := 0
			for _, npc := range summary.NPCsEncountered {
				npcLower := strings.ToLower(npc.Name)
				for sceneName := range sceneNPCNames {
					if strings.Contains(npcLower, sceneName) || strings.Contains(sceneName, npcLower) {
						matchCount++
						break
					}
				}
			}
			// At least some NPCs should match
			result.NPCsMatch = matchCount > 0
		} else {
			result.NPCsMatch = false
		}
	} else {
		result.NPCsMatch = true // No NPCs expected
	}

	// Check for unresolved threads in complex scenes
	hasConversation := len(tc.Data.ConversationHistory) > 2
	result.HasUnresolved = len(summary.UnresolvedThreads) > 0 || !hasConversation

	// Overall pass
	result.Passed = result.ValidJSON &&
		result.HasNarrativeProse &&
		result.ProseIsNarrative &&
		result.HasKeyEvents &&
		result.HasHowEnded &&
		result.NPCsMatch

	return result
}

// TestSceneSummary_LLMEvaluation verifies scene summary generation captures narratively important details.
//
// Per the template rules:
// - key_events: 1-4 items, most important only
// - npcs_encountered: only NPCs actually interacted with
// - unresolved_threads: story hooks for future scenes
// - narrative_prose: "Previously on..." style recap, not a list
// - Focus on what's important for continuity between scenes
func TestSceneSummary_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	verboseLogging := os.Getenv("VERBOSE") == "1"
	testCases := getSceneSummaryTestCases()

	var results []SceneSummaryResult
	passed := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateSceneSummary(ctx, client, tc)
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
				t.Logf("  ValidJSON=%v Prose=%v Narrative=%v Events=%d HowEnded=%v NPCs=%v Unresolved=%v",
					result.ValidJSON, result.HasNarrativeProse, result.ProseIsNarrative,
					result.KeyEventCount, result.HasHowEnded, result.NPCsMatch, result.HasUnresolved)
				if result.Parsed != nil {
					t.Logf("  Prose: %s", truncateSummaryText(result.Parsed.NarrativeProse, 200))
					for i, e := range result.Parsed.KeyEvents {
						t.Logf("  Event %d: %s", i+1, e)
					}
					for _, npc := range result.Parsed.NPCsEncountered {
						t.Logf("  NPC: %s (%s)", npc.Name, npc.Attitude)
					}
					for _, ut := range result.Parsed.UnresolvedThreads {
						t.Logf("  Unresolved: %s", ut)
					}
					t.Logf("  How ended: %s", result.Parsed.HowEnded)
				}
				if !result.ValidJSON {
					t.Logf("  Raw: %s", truncateSummaryText(result.RawResponse, 300))
				}
			}

			assert.True(t, result.ValidJSON, "Response should be valid JSON")
			assert.True(t, result.HasNarrativeProse,
				"Summary should have narrative prose for recap")
			assert.True(t, result.ProseIsNarrative,
				"Narrative prose should read like a story recap, not a list")
			assert.True(t, result.HasKeyEvents,
				"Summary should have 1-4 key events (got %d)", result.KeyEventCount)
			assert.True(t, result.HasHowEnded,
				"Summary should describe how the scene ended")
			assert.True(t, result.NPCsMatch,
				"NPCs encountered should match NPCs present in the scene")
		})
	}

	// Summary
	t.Log("\n========== SCENE SUMMARY GENERATION SUMMARY ==========")
	t.Logf("Passed: %d/%d (%.1f%%)", passed, len(testCases), float64(passed)*100/float64(len(testCases)))

	unresolvedCount := 0
	for _, r := range results {
		if r.HasUnresolved {
			unresolvedCount++
		}
	}
	t.Logf("Summaries with unresolved threads: %d/%d", unresolvedCount, len(results))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Passed {
			t.Logf("FAIL: %s — ValidJSON=%v Prose=%v Events=%d NPCs=%v",
				r.TestCase.Name, r.ValidJSON, r.HasNarrativeProse,
				r.KeyEventCount, r.NPCsMatch)
		}
	}
}

func truncateSummaryText(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
