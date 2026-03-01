//go:build llmeval

package llmeval_test

import (
	"context"
	"os"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ChallengeClassificationTestCase validates that inputs matching challenge
// task skills are classified as "action" rather than "clarification" or "dialog".
type ChallengeClassificationTestCase struct {
	Name                  string
	RawInput              string
	SceneName             string
	SceneDescription      string
	ActiveChallengeSkills []string
	ExpectedType          string // "action", "dialog", "clarification", or "narrative"
	Description           string
}

func getChallengeActionCases() []ChallengeClassificationTestCase {
	return []ChallengeClassificationTestCase{
		{
			Name:                  "Notice during challenge — scan for path",
			RawInput:              "I look carefully for a safe path",
			SceneName:             "Collapsing Mine",
			SceneDescription:      "A crumbling mineshaft with falling rocks and loose timbers",
			ActiveChallengeSkills: []string{"Notice", "Athletics"},
			ExpectedType:          "action",
			Description:           "Notice is a pending challenge task — should be action, not clarification",
		},
		{
			Name:                  "Stealth during challenge — sneak past",
			RawInput:              "I move quietly through the shadows",
			SceneName:             "Museum Vault",
			SceneDescription:      "A high-security vault with laser grids and pressure plates",
			ActiveChallengeSkills: []string{"Stealth", "Burglary"},
			ExpectedType:          "action",
			Description:           "Stealth is a pending challenge task — must be action",
		},
		{
			Name:                  "Athletics during challenge — leap gap",
			RawInput:              "I jump over the gap in the floor",
			SceneName:             "Collapsing Mine",
			SceneDescription:      "A crumbling mineshaft with a wide crack splitting the floor",
			ActiveChallengeSkills: []string{"Notice", "Athletics"},
			ExpectedType:          "action",
			Description:           "Athletics is a pending challenge task — physical action",
		},
		{
			Name:                  "Burglary during challenge — pick the lock",
			RawInput:              "I examine the lock and try to pick it",
			SceneName:             "Museum Vault",
			SceneDescription:      "Standing before a massive vault door with a combination lock",
			ActiveChallengeSkills: []string{"Stealth", "Burglary"},
			ExpectedType:          "action",
			Description:           "Burglary is a pending challenge task — action",
		},
		{
			Name:                  "Will during challenge — resist fear",
			RawInput:              "I steel myself and push through the terror",
			SceneName:             "Haunted Crypt",
			SceneDescription:      "An ancient crypt where shadows move on their own",
			ActiveChallengeSkills: []string{"Will", "Lore"},
			ExpectedType:          "action",
			Description:           "Will is a pending challenge task — overcoming fear",
		},
		{
			Name:                  "Dialog still recognized during challenge",
			RawInput:              "\"We need to hurry!\" I shout to my companion",
			SceneName:             "Collapsing Mine",
			SceneDescription:      "A crumbling mineshaft with falling rocks and loose timbers",
			ActiveChallengeSkills: []string{"Notice", "Athletics"},
			ExpectedType:          "dialog",
			Description:           "Pure dialog remains dialog even during a challenge",
		},
	}
}

type ChallengeClassificationResult struct {
	TestCase   ChallengeClassificationTestCase
	ActualType string
	Matches    bool
	Error      error
}

func evaluateChallengeClassification(ctx context.Context, client llm.LLMClient, tc ChallengeClassificationTestCase) ChallengeClassificationResult {
	testScene := scene.NewScene("test-scene", tc.SceneName, tc.SceneDescription)

	data := promptpkg.InputClassificationData{
		Scene:                 testScene,
		PlayerInput:           tc.RawInput,
		ActiveChallengeSkills: tc.ActiveChallengeSkills,
	}

	prompt, err := promptpkg.RenderInputClassification(data)
	if err != nil {
		return ChallengeClassificationResult{TestCase: tc, Error: err}
	}

	classification, err := llm.SimpleCompletion(ctx, client, prompt, 10, 0.1)
	if err != nil {
		return ChallengeClassificationResult{TestCase: tc, Error: err}
	}

	classification = promptpkg.ParseClassification(classification)

	return ChallengeClassificationResult{
		TestCase:   tc,
		ActualType: classification,
		Matches:    classification == tc.ExpectedType,
	}
}

// TestChallengeClassification_LLMEvaluation verifies that inputs matching
// challenge task skills are classified as "action" when a challenge is active.
// Run with: go test -v -tags=llmeval ./test/llmeval/ -run TestChallengeClassification
func TestChallengeClassification_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	cases := getChallengeActionCases()
	var results []ChallengeClassificationResult
	correct := 0

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateChallengeClassification(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.Matches {
				correct++
			}

			assert.Equal(t, tc.ExpectedType, result.ActualType,
				"Classification mismatch for '%s'. %s",
				tc.RawInput, tc.Description)
		})
	}

	// Summary
	t.Log("\n========== CHALLENGE CLASSIFICATION SUMMARY ==========")
	t.Logf("Correct: %d/%d (%.1f%%)",
		correct, len(cases), float64(correct)*100/float64(len(cases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: '%s'", r.TestCase.RawInput)
			t.Logf("      Expected: %s, Got: %s", r.TestCase.ExpectedType, r.ActualType)
			t.Logf("      Skills: %v", r.TestCase.ActiveChallengeSkills)
			t.Logf("      Why: %s", r.TestCase.Description)
		}
	}
}
