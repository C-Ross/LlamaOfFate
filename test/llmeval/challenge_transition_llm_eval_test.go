//go:build llmeval

package llmeval_test

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
)

// ChallengeTransitionTestCase tests whether the LLM respects the
// "no scene transition during active challenge" instruction.
type ChallengeTransitionTestCase struct {
	Name             string
	PlayerInput      string
	SceneName        string
	SceneDescription string
	ChallengeTasks   []scene.ChallengeTask
	ExpectMarker     bool   // Should the response include [SCENE_TRANSITION]?
	Description      string // Why the expected outcome is correct
}

// getChallengeExitAttempts returns inputs where the player tries to leave but
// should be blocked because a challenge is active.
func getChallengeExitAttempts() []ChallengeTransitionTestCase {
	return []ChallengeTransitionTestCase{
		{
			Name:             "Walk away from active challenge",
			PlayerInput:      "I walk out of the mine entrance",
			SceneName:        "Collapsing Mine",
			SceneDescription: "A crumbling mineshaft with falling rocks and loose timbers",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "t1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending, Description: "Dodge falling rocks"},
				{ID: "t2", Skill: "Notice", Difficulty: 2, Status: scene.TaskPending, Description: "Spot safe path"},
			},
			ExpectMarker: false,
			Description:  "Cannot leave — active challenge requires resolution first",
		},
		{
			Name:             "Try to escape vault heist",
			PlayerInput:      "I leave through the back door",
			SceneName:        "Museum Vault",
			SceneDescription: "A high-security vault with laser grids and pressure plates",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "t1", Skill: "Burglary", Difficulty: 4, Status: scene.TaskPending, Description: "Crack the combination lock"},
				{ID: "t2", Skill: "Stealth", Difficulty: 3, Status: scene.TaskPending, Description: "Bypass laser grid"},
			},
			ExpectMarker: false,
			Description:  "Mid-heist challenge — leaving would orphan the challenge",
		},
		{
			Name:             "Flee burning building challenge",
			PlayerInput:      "I run outside to safety",
			SceneName:        "Burning Manor",
			SceneDescription: "A mansion engulfed in flames with smoke filling the halls",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "t1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending, Description: "Navigate through smoke"},
				{ID: "t2", Skill: "Will", Difficulty: 2, Status: scene.TaskPending, Description: "Stay calm under pressure"},
			},
			ExpectMarker: false,
			Description:  "Running outside IS the challenge goal — must resolve tasks first",
		},
		{
			Name:             "Ride away from ambush challenge",
			PlayerInput:      "I mount my horse and ride off",
			SceneName:        "Canyon Ambush",
			SceneDescription: "A narrow canyon with bandits firing from the ridges above",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "t1", Skill: "Ride", Difficulty: 3, Status: scene.TaskPending, Description: "Control panicked horse"},
				{ID: "t2", Skill: "Notice", Difficulty: 2, Status: scene.TaskPending, Description: "Spot escape route"},
			},
			ExpectMarker: false,
			Description:  "Escape requires challenge tasks, no free transition",
		},
	}
}

// ChallengeTransitionResult stores one evaluation result.
type ChallengeTransitionResult struct {
	TestCase  ChallengeTransitionTestCase
	Response  string
	HasMarker bool
	Matches   bool
	Error     error
}

func evaluateChallengeTransition(ctx context.Context, client llm.LLMClient, tc ChallengeTransitionTestCase) ChallengeTransitionResult {
	testScene := scene.NewScene("test-scene", tc.SceneName, tc.SceneDescription)
	err := testScene.StartChallenge("Active challenge", tc.ChallengeTasks)
	if err != nil {
		return ChallengeTransitionResult{TestCase: tc, Error: err}
	}

	player := core.NewCharacter("player-1", "Test Character")
	player.Aspects.HighConcept = "Wandering Stranger"

	charContext := BuildCharacterContext(player)
	aspectsContext := BuildAspectsContext(testScene, player, nil)

	data := promptpkg.SceneResponseData{
		Scene:               testScene,
		CharacterContext:    charContext,
		AspectsContext:      aspectsContext,
		ConversationContext: "No previous conversation.",
		PlayerInput:         tc.PlayerInput,
		InteractionType:     "dialog",
	}

	prompt, err := promptpkg.RenderSceneResponse(data)
	if err != nil {
		return ChallengeTransitionResult{TestCase: tc, Error: err}
	}

	response, err := llm.SimpleCompletion(ctx, client, prompt, 500, 0.3)
	if err != nil {
		return ChallengeTransitionResult{TestCase: tc, Error: err}
	}

	transition, _ := promptpkg.ParseSceneTransitionMarker(response)
	hasMarker := transition != nil

	return ChallengeTransitionResult{
		TestCase:  tc,
		Response:  response,
		HasMarker: hasMarker,
		Matches:   hasMarker == tc.ExpectMarker,
	}
}

// TestChallengeTransition_LLMEvaluation verifies that the scene response prompt
// suppresses [SCENE_TRANSITION] markers when a challenge is active.
// Run with: go test -v -tags=llmeval ./test/llmeval/ -run TestChallengeTransition
func TestChallengeTransition_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	cases := getChallengeExitAttempts()
	var results []ChallengeTransitionResult
	correct := 0

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateChallengeTransition(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.Matches {
				correct++
			}

			assert.Equal(t, tc.ExpectMarker, result.HasMarker,
				"Transition marker mismatch for '%s'. %s\nResponse: %s",
				tc.PlayerInput, tc.Description, TruncateResponse(result.Response, 300))
		})
	}

	// Summary
	t.Log("\n========== CHALLENGE TRANSITION SUPPRESSION SUMMARY ==========")
	t.Logf("Suppressed correctly: %d/%d (%.1f%%)",
		correct, len(cases), float64(correct)*100/float64(len(cases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: '%s'", r.TestCase.PlayerInput)
			t.Logf("      Scene: %s", r.TestCase.SceneDescription)
			t.Logf("      Why: %s", r.TestCase.Description)
			t.Logf("      Response: %s", TruncateResponse(r.Response, 200))
		}
	}
}
