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
		// --- Cases from Europa session bug: cautious/examine inputs misclassified as clarification ---
		{
			Name:                  "Cautious examine — control panel during challenge",
			RawInput:              "John carefully examines the reactor control panel to identify the warning indicators",
			SceneName:             "Reactor Control Room",
			SceneDescription:      "The reactor control room aboard Europa Station. Alarms blare and warning lights flash across multiple panels.",
			ActiveChallengeSkills: []string{"Notice", "Engineering", "Will"},
			ExpectedType:          "action",
			Description:           "Carefully examining is an active attempt, not a question — must be action during challenge",
		},
		{
			Name:                  "Search for keycard — investigate during challenge",
			RawInput:              "John searches the room for a keycard or access device",
			SceneName:             "Maintenance Bay",
			SceneDescription:      "A cluttered maintenance bay with lockers, tool racks, and scattered equipment",
			ActiveChallengeSkills: []string{"Investigate", "Crafts", "Athletics"},
			ExpectedType:          "action",
			Description:           "Searching is active effort — must be action, not clarification",
		},
		{
			Name:                  "Move through debris — athletics during challenge",
			RawInput:              "John carefully moves through the debris toward the emergency exit",
			SceneName:             "Collapsed Corridor",
			SceneDescription:      "A corridor partially collapsed with structural debris and leaking pipes",
			ActiveChallengeSkills: []string{"Athletics", "Notice", "Will"},
			ExpectedType:          "action",
			Description:           "Cautious movement through obstacle is an action, not clarification",
		},
		{
			Name:                  "Brace against hazard — will during challenge",
			RawInput:              "John braces himself against the heat and focuses on shutting down the override",
			SceneName:             "Reactor Control Room",
			SceneDescription:      "Temperature rising in the reactor room as emergency systems fail",
			ActiveChallengeSkills: []string{"Will", "Engineering", "Notice"},
			ExpectedType:          "action",
			Description:           "Resisting and pushing through is clearly an action",
		},
		{
			Name:                  "Genuine clarification question during challenge",
			RawInput:              "What do the alarms say?",
			SceneName:             "Reactor Control Room",
			SceneDescription:      "The reactor control room aboard Europa Station with blaring alarms",
			ActiveChallengeSkills: []string{"Notice", "Engineering", "Will"},
			ExpectedType:          "clarification",
			Description:           "Bare question asking for information is still clarification even during challenge",
		},
		{
			Name:                  "Explicit OOC question during challenge",
			RawInput:              "OOC: Can I use Engineering to fix the coolant system, or do I need Crafts?",
			SceneName:             "Reactor Control Room",
			SceneDescription:      "Temperature rising in the reactor room as emergency systems fail",
			ActiveChallengeSkills: []string{"Will", "Engineering", "Notice"},
			ExpectedType:          "clarification",
			Description:           "Player explicitly flags out-of-character question about game mechanics — always clarification",
		},
		{
			Name:                  "Implicit OOC meta-question during challenge",
			RawInput:              "What skills can I use for the remaining tasks?",
			SceneName:             "Reactor Control Room",
			SceneDescription:      "Temperature rising in the reactor room as emergency systems fail",
			ActiveChallengeSkills: []string{"Will", "Engineering", "Notice"},
			ExpectedType:          "clarification",
			Description:           "Meta-question about game state and mechanics is clearly out of character — must be clarification, not action",
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
	client := RequireLLMClient(t)
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
