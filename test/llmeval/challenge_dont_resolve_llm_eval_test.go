//go:build llmeval

package llmeval_test

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
)

// DontResolveTestCase validates that during an active challenge, the GM's scene
// response to dialog/clarification does NOT narrate any pending tasks being
// completed, solved, or bypassed. Extracted from the Meltdown on Europa session
// where the GM pre-resolved challenge tasks in dialog responses.
type DontResolveTestCase struct {
	Name                string
	PlayerInput         string
	InteractionType     string // "dialog" or "clarification"
	SceneName           string
	SceneDescription    string
	ChallengeDesc       string
	ChallengeTasks      []scene.ChallengeTask
	ConversationContext string
	OtherCharacters     []*character.Character
	Description         string
}

func getDontResolveTestCases() []DontResolveTestCase {
	drPatel := character.NewCharacter("dr-patel", "Dr. Patel")
	drPatel.Aspects.HighConcept = "Brilliant Research Lead Under Pressure"

	return []DontResolveTestCase{
		{
			Name:            "Dialog during challenge — reactor control room",
			PlayerInput:     "\"Dr. Patel, what's the status of the coolant pumps?\"",
			InteractionType: "dialog",
			SceneName:       "Reactor Control Room",
			SceneDescription: "The reactor control room aboard Europa Station. Alarms blare, warning lights flash, " +
				"and the temperature is climbing. Emergency procedures have failed.",
			ChallengeDesc: "Stabilize the reactor before a catastrophic meltdown",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "task-1", Description: "Identify the failing subsystem from the control panel warnings", Skill: "Notice", Difficulty: int(dice.Good), Status: scene.TaskPending},
				{ID: "task-2", Description: "Override the safety lockout to restore coolant flow", Skill: "Engineering", Difficulty: int(dice.Great), Status: scene.TaskPending},
				{ID: "task-3", Description: "Maintain focus through the rising heat and pressure", Skill: "Will", Difficulty: int(dice.Fair), Status: scene.TaskPending},
			},
			ConversationContext: "GM: The reactor control room is chaos. Warning indicators flash across every panel as the temperature gauge climbs steadily into the red zone.",
			OtherCharacters:     []*character.Character{drPatel},
			Description:         "NPC response should describe the situation but NOT narrate any task being completed — the player still needs to roll for each task",
		},
		{
			Name:             "Clarification during challenge — general environment question",
			PlayerInput:      "How much time do we have before the reactor goes critical?",
			InteractionType:  "clarification",
			SceneName:        "Reactor Control Room",
			SceneDescription: "The reactor control room aboard Europa Station. Multiple systems are in alert status.",
			ChallengeDesc:    "Stabilize the reactor before a catastrophic meltdown",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "task-1", Description: "Identify the failing subsystem from the control panel warnings", Skill: "Notice", Difficulty: int(dice.Good), Status: scene.TaskPending},
				{ID: "task-2", Description: "Override the safety lockout to restore coolant flow", Skill: "Engineering", Difficulty: int(dice.Great), Status: scene.TaskPending},
				{ID: "task-3", Description: "Maintain focus through the rising heat and pressure", Skill: "Will", Difficulty: int(dice.Fair), Status: scene.TaskPending},
			},
			ConversationContext: "GM: Alarms blare across the control room.",
			OtherCharacters:     nil,
			Description:         "Time question doesn't overlap any pending task — GM can answer without resolving Notice, Engineering, or Will tasks",
		},
		{
			Name:             "Dialog during heist challenge — asking guard schedule",
			PlayerInput:      "\"Hey, do you know when the next patrol comes through?\"",
			InteractionType:  "dialog",
			SceneName:        "Museum Service Corridor",
			SceneDescription: "A dimly lit service corridor behind the main gallery. Security cameras rotate on fixed intervals.",
			ChallengeDesc:    "Infiltrate the museum vault and steal the artifact",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "task-1", Description: "Bypass the security camera system", Skill: "Burglary", Difficulty: int(dice.Good), Status: scene.TaskPending},
				{ID: "task-2", Description: "Move through the corridors without alerting guards", Skill: "Stealth", Difficulty: int(dice.Great), Status: scene.TaskPending},
				{ID: "task-3", Description: "Crack the vault combination lock", Skill: "Crafts", Difficulty: int(dice.Good), Status: scene.TaskPending},
			},
			ConversationContext: "GM: You're crouched behind a supply cart as a guard walks past the far end of the corridor.",
			OtherCharacters:     nil,
			Description:         "Response should NOT narrate the player bypassing cameras, sneaking past guards, or cracking the vault — those are pending tasks",
		},
	}
}

// DontResolveResult stores the evaluation result for one test case.
type DontResolveResult struct {
	TestCase     DontResolveTestCase
	Response     string
	ResolvesTask bool // true = BAD: the response narrates a task being completed/solved
	JudgeResult  JudgeResult
	Matches      bool
	Error        error
}

func evaluateDontResolve(ctx context.Context, client llm.LLMClient, tc DontResolveTestCase) DontResolveResult {
	// Build a scene with an active challenge
	testScene := scene.NewScene("test-scene", tc.SceneName, tc.SceneDescription)
	err := testScene.StartChallenge(tc.ChallengeDesc, tc.ChallengeTasks)
	if err != nil {
		return DontResolveResult{TestCase: tc, Error: err}
	}

	// Create player character
	player := character.NewCharacter("player1", "John MacDougal")
	player.Aspects.HighConcept = "Resourceful Station Engineer"
	player.Aspects.Trouble = "First Day on Europa"

	charContext := BuildCharacterContext(player)
	aspectsContext := BuildAspectsContext(testScene, player, tc.OtherCharacters)

	conversationContext := tc.ConversationContext
	if conversationContext == "" {
		conversationContext = "No previous conversation."
	}

	data := promptpkg.SceneResponseData{
		Scene:               testScene,
		CharacterContext:    charContext,
		AspectsContext:      aspectsContext,
		ConversationContext: conversationContext,
		PlayerInput:         tc.PlayerInput,
		InteractionType:     tc.InteractionType,
		OtherCharacters:     tc.OtherCharacters,
	}

	prompt, err := promptpkg.RenderSceneResponse(data)
	if err != nil {
		return DontResolveResult{TestCase: tc, Error: err}
	}

	response, err := llm.SimpleCompletion(ctx, client, prompt, 500, 0.3)
	if err != nil {
		return DontResolveResult{TestCase: tc, Error: err}
	}

	// Build a context-aware judge question that lists the pending tasks
	taskList := ""
	for _, task := range tc.ChallengeTasks {
		if task.Status == scene.TaskPending {
			taskList += "- " + task.Description + " (" + task.Skill + ")\n"
		}
	}

	judgeQuestion := "Does this GM response narrate or describe any of the following challenge tasks being completed, solved, overcome, or bypassed? " +
		"(Answer YES if the response shows any task outcome being resolved WITHOUT a dice roll — e.g., describing a door being forced open, " +
		"a system being fixed, a problem being identified, a lock being picked, or guards being avoided. Answer NO if the response only describes " +
		"the environment, builds tension, voices NPCs, or provides partial information without resolving a task.)"

	judgeCtx := "The following challenge tasks are PENDING and require dice rolls to resolve:\n" + taskList +
		"\nThe player's input was: \"" + tc.PlayerInput + "\"\n" +
		"The interaction type is: " + tc.InteractionType + " (not an action roll)"

	judgeResult, err := LLMJudgeWithContext(ctx, client, response, judgeQuestion, judgeCtx)
	if err != nil {
		return DontResolveResult{TestCase: tc, Response: response, Error: err}
	}

	// pass=true means the judge found tasks being resolved (BAD)
	resolvesTask := judgeResult.Pass

	return DontResolveResult{
		TestCase:     tc,
		Response:     response,
		ResolvesTask: resolvesTask,
		JudgeResult:  judgeResult,
		Matches:      !resolvesTask, // success = tasks are NOT resolved
	}
}

// TestSceneResponse_DontResolveChallengeTasks_LLMEvaluation verifies that during
// an active challenge, the GM's scene response to dialog/clarification does NOT
// narrate any pending task being completed, bypassed, or solved.
//
// Extracted from the Meltdown on Europa session where dialog responses
// pre-resolved challenge tasks (e.g., describing the failing subsystem being identified
// before the player rolled Notice).
//
// Run with: go test -v -tags=llmeval ./test/llmeval/ -run TestSceneResponse_DontResolveChallengeTasks
func TestSceneResponse_DontResolveChallengeTasks_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	verboseLogging := VerboseLoggingEnabled()
	testCases := getDontResolveTestCases()

	var results []DontResolveResult
	correct := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateDontResolve(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.Matches {
				correct++
			}

			if verboseLogging || !result.Matches {
				status := "PASS"
				if !result.Matches {
					status = "FAIL"
				}
				t.Logf("%s: ResolvesTask=%v", status, result.ResolvesTask)
				t.Logf("  Input: %s", tc.PlayerInput)
				t.Logf("  Type: %s", tc.InteractionType)
				t.Logf("  Why: %s", tc.Description)
				t.Logf("  Judge: pass=%v — %s", result.JudgeResult.Pass, result.JudgeResult.Reasoning)
				t.Logf("  Response: %s", TruncateResponse(result.Response, 400))
			}

			assert.False(t, result.ResolvesTask,
				"GM response should NOT resolve pending challenge tasks during %s.\nJudge: %s\nResponse: %s",
				tc.InteractionType, result.JudgeResult.Reasoning, TruncateResponse(result.Response, 400))
		})
	}

	// Summary
	t.Log("\n========== DONT RESOLVE CHALLENGE TASKS SUMMARY ==========")
	t.Logf("Cases without task resolution: %d/%d (%.1f%%)",
		correct, len(testCases), float64(correct)*100/float64(len(testCases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: '%s'", r.TestCase.Name)
			t.Logf("      Input: %s", TruncateResponse(r.TestCase.PlayerInput, 80))
			t.Logf("      Judge: %s", r.JudgeResult.Reasoning)
		}
	}
}
