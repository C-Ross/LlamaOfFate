//go:build llmeval

package llmeval_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
	llmazure "github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FateNarrationTestCase represents a test case for fate narration parsing
type FateNarrationTestCase struct {
	Name          string
	Data          promptpkg.FateNarrationData
	ExpectedFates int      // Expected number of fate entries
	PermanentIDs  []string // NPC IDs expected to be permanently removed
	TemporaryIDs  []string // NPC IDs expected to NOT be permanently removed
	Description   string
}

// FateNarrationEvalResult stores the result of a fate narration evaluation
type FateNarrationEvalResult struct {
	TestCase         FateNarrationTestCase
	RawResponse      string
	Parsed           *promptpkg.FateNarrationResult
	ValidJSON        bool
	CorrectFateCount bool
	PermanentCorrect bool
	TemporaryCorrect bool
	HasNarrative     bool
	FatesHaveDesc    bool // All fates have non-empty descriptions
	IDsMatch         bool // Fate IDs match expected NPC IDs
	Passed           bool
	Error            error
}

func getFateNarrationTestCases() []FateNarrationTestCase {
	return []FateNarrationTestCase{
		{
			Name: "All killed - explicit violence",
			Data: promptpkg.FateNarrationData{
				SceneName:        "Warehouse Showdown",
				SceneDescription: "A tense fight in a dimly lit warehouse full of crates and shadows.",
				ConflictType:     "physical",
				TakenOutNPCs: []promptpkg.FateNarrationNPC{
					{ID: "npc-thug", Name: "Thug", HighConcept: "Hired Muscle"},
					{ID: "npc-enforcer", Name: "Enforcer", HighConcept: "Ruthless Bodyguard"},
				},
				PlayerNarration: "I kill them both. No survivors.",
			},
			ExpectedFates: 2,
			PermanentIDs:  []string{"npc-thug", "npc-enforcer"},
			TemporaryIDs:  []string{},
			Description:    "Explicit killing of all opponents should mark both as permanent",
		},
		{
			Name: "Mixed fates - one killed, one spared",
			Data: promptpkg.FateNarrationData{
				SceneName:        "Dusty Trail Ambush",
				SceneDescription: "A rocky canyon with walls closing in on both sides.",
				ConflictType:     "physical",
				TakenOutNPCs: []promptpkg.FateNarrationNPC{
					{ID: "npc-leader", Name: "Bandit Leader", HighConcept: "Scarred Outlaw"},
					{ID: "npc-young", Name: "Young Bandit", HighConcept: "Reluctant Criminal"},
				},
				PlayerNarration: "The bandit leader tried to stab me in the back — I put him down for good. But the young one was just a scared kid. I tie him up and leave him for the sheriff to find.",
			},
			ExpectedFates: 2,
			PermanentIDs:  []string{"npc-leader"},
			TemporaryIDs:  []string{"npc-young"},
			Description:    "Mixed narrative should differentiate permanent vs temporary fates",
		},
		{
			Name: "All spared - nonlethal takedown",
			Data: promptpkg.FateNarrationData{
				SceneName:        "Guard Station",
				SceneDescription: "A well-fortified guard station at the entrance to the compound.",
				ConflictType:     "physical",
				TakenOutNPCs: []promptpkg.FateNarrationNPC{
					{ID: "npc-captain", Name: "Guard Captain", HighConcept: "Dutiful Soldier"},
					{ID: "npc-guard", Name: "Guard", HighConcept: "Fresh Recruit"},
				},
				PlayerNarration: "I knock them both unconscious and hide them in the supply closet. They'll have headaches when they wake up, but they'll be fine.",
			},
			ExpectedFates: 2,
			PermanentIDs:  []string{},
			TemporaryIDs:  []string{"npc-captain", "npc-guard"},
			Description:    "Nonlethal takedown should mark all as temporary",
		},
		{
			Name: "Single opponent killed",
			Data: promptpkg.FateNarrationData{
				SceneName:        "Final Duel",
				SceneDescription: "The dusty main street of a frontier town at high noon.",
				ConflictType:     "physical",
				TakenOutNPCs: []promptpkg.FateNarrationNPC{
					{ID: "npc-bart", Name: "Black Bart", HighConcept: "Notorious Gunslinger"},
				},
				PlayerNarration: "My bullet finds its mark. Black Bart crumples to the dirt, his gun clattering away. He's done.",
			},
			ExpectedFates: 1,
			PermanentIDs:  []string{"npc-bart"},
			TemporaryIDs:  []string{},
			Description:    "Single opponent killed should produce one permanent fate",
		},
		{
			Name: "Vague narration - they're all done",
			Data: promptpkg.FateNarrationData{
				SceneName:        "Bar Brawl",
				SceneDescription: "A rowdy tavern with overturned tables and broken bottles.",
				ConflictType:     "physical",
				TakenOutNPCs: []promptpkg.FateNarrationNPC{
					{ID: "npc-joe", Name: "Brawler Joe", HighConcept: "Drunken Troublemaker"},
					{ID: "npc-pete", Name: "Slim Pete", HighConcept: "Wiry Pickpocket"},
				},
				PlayerNarration: "They're done. I leave them on the floor and walk out.",
			},
			ExpectedFates: 2,
			PermanentIDs:  []string{},
			TemporaryIDs:  []string{"npc-joe", "npc-pete"},
			Description:    "Vague narration without explicit killing should default to non-permanent",
		},
		{
			Name: "Mental conflict - intimidation",
			Data: promptpkg.FateNarrationData{
				SceneName:        "The Negotiation",
				SceneDescription: "A tense meeting room in a corporate tower overlooking the city.",
				ConflictType:     "mental",
				TakenOutNPCs: []promptpkg.FateNarrationNPC{
					{ID: "npc-lawyer", Name: "Corporate Lawyer", HighConcept: "Shrewd Negotiator"},
				},
				PlayerNarration: "I've broken their will completely. The lawyer agrees to all our terms and slinks out of the room defeated.",
			},
			ExpectedFates: 1,
			PermanentIDs:  []string{},
			TemporaryIDs:  []string{"npc-lawyer"},
			Description:    "Mental conflict taken out should not be permanent (no physical harm)",
		},
	}
}

func evaluateFateNarration(ctx context.Context, client llm.LLMClient, tc FateNarrationTestCase) FateNarrationEvalResult {
	rendered, err := promptpkg.RenderFateNarration(tc.Data)
	if err != nil {
		return FateNarrationEvalResult{TestCase: tc, Error: err}
	}

	raw, err := llm.SimpleCompletion(ctx, client, rendered, 400, 0.3)
	if err != nil {
		return FateNarrationEvalResult{TestCase: tc, Error: err}
	}

	result := FateNarrationEvalResult{
		TestCase:    tc,
		RawResponse: raw,
	}

	parsed, err := promptpkg.ParseFateNarration(raw)
	if err != nil {
		result.ValidJSON = false
		return result
	}
	result.ValidJSON = true
	result.Parsed = parsed

	// Check fate count
	result.CorrectFateCount = len(parsed.Fates) == tc.ExpectedFates

	// Check narrative exists
	result.HasNarrative = parsed.Narrative != "" && len(parsed.Narrative) > 20

	// Check all fates have descriptions
	result.FatesHaveDesc = true
	for _, f := range parsed.Fates {
		if f.Description == "" {
			result.FatesHaveDesc = false
			break
		}
	}

	// Build lookup of parsed fates by ID
	fatesByID := make(map[string]promptpkg.NPCFateResult)
	for _, f := range parsed.Fates {
		fatesByID[f.ID] = f
	}

	// Check IDs match expected NPCs
	allExpectedIDs := append(tc.PermanentIDs, tc.TemporaryIDs...)
	matchCount := 0
	for _, id := range allExpectedIDs {
		if _, ok := fatesByID[id]; ok {
			matchCount++
		}
	}
	result.IDsMatch = matchCount == len(allExpectedIDs)

	// Check permanent flags
	result.PermanentCorrect = true
	for _, id := range tc.PermanentIDs {
		fate, ok := fatesByID[id]
		if !ok || !fate.Permanent {
			result.PermanentCorrect = false
		}
	}

	result.TemporaryCorrect = true
	for _, id := range tc.TemporaryIDs {
		fate, ok := fatesByID[id]
		if !ok || fate.Permanent {
			result.TemporaryCorrect = false
		}
	}

	// Overall pass
	result.Passed = result.ValidJSON &&
		result.CorrectFateCount &&
		result.PermanentCorrect &&
		result.TemporaryCorrect &&
		result.HasNarrative &&
		result.FatesHaveDesc &&
		result.IDsMatch

	return result
}

// TestFateNarration_LLMEvaluation verifies that the fate narration prompt correctly
// parses player free-text into structured per-NPC fates with accurate permanent flags.
//
// Per Fate Core rules: "the person who took you out gets to decide what your loss looks like."
// The LLM must classify the player's narration into individual NPC outcomes and determine
// whether each NPC is permanently removed from the story (killed, destroyed) or temporarily
// defeated (knocked out, fled, captured).
func TestFateNarration_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := llmazure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := llmazure.NewClient(*config)
	ctx := context.Background()

	verboseLogging := os.Getenv("VERBOSE") == "1"
	testCases := getFateNarrationTestCases()

	var results []FateNarrationEvalResult
	passed := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateFateNarration(ctx, client, tc)
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
				t.Logf("  ValidJSON=%v FateCount=%v(%d) Permanent=%v Temporary=%v Narrative=%v Desc=%v IDs=%v",
					result.ValidJSON, result.CorrectFateCount, len(result.Parsed.Fates),
					result.PermanentCorrect, result.TemporaryCorrect,
					result.HasNarrative, result.FatesHaveDesc, result.IDsMatch)
				if result.Parsed != nil {
					for _, f := range result.Parsed.Fates {
					t.Logf("  Fate: %s [%s] — %s (permanent=%v)", f.Name, f.ID, f.Description, f.Permanent)
					}
					t.Logf("  Narrative: %s", truncateFateText(result.Parsed.Narrative, 200))
				}
				if !result.ValidJSON {
					t.Logf("  Raw: %s", truncateFateText(result.RawResponse, 300))
				}
			}

			assert.True(t, result.ValidJSON, "Response should be valid JSON")
			assert.True(t, result.CorrectFateCount,
				"Should have %d fates, got %d", tc.ExpectedFates, len(result.Parsed.Fates))
			assert.True(t, result.PermanentCorrect,
				"Permanent flags should match expected for killed/destroyed NPCs")
			assert.True(t, result.TemporaryCorrect,
				"Temporary flags should match expected for spared/incapacitated NPCs")
			assert.True(t, result.HasNarrative,
				"Response should include a narrative description")
			assert.True(t, result.FatesHaveDesc,
				"All fates should have non-empty descriptions")
			assert.True(t, result.IDsMatch,
				"Fate IDs should match input NPC IDs")
		})
	}

	// Summary
	t.Log("\n========== FATE NARRATION EVALUATION SUMMARY ==========")
	t.Logf("Passed: %d/%d (%.1f%%)", passed, len(testCases), float64(passed)*100/float64(len(testCases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Passed {
			t.Logf("FAIL: %s — JSON=%v Count=%v Perm=%v Temp=%v (%s)",
				r.TestCase.Name, r.ValidJSON, r.CorrectFateCount,
				r.PermanentCorrect, r.TemporaryCorrect, r.TestCase.Description)
		}
	}
}

func truncateFateText(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
