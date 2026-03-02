//go:build llmeval

package llmeval_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
)

// ChallengeNarrativeTestCase validates that action narratives during a challenge
// focus on the player character's own attempt at the specific task, without
// having NPCs solve the task or narrating unrelated challenge tasks.
type ChallengeNarrativeTestCase struct {
	Name                 string
	ActionType           action.ActionType
	Skill                string
	Description          string
	RawInput             string
	OutcomeType          dice.OutcomeType
	Shifts               int
	SceneName            string
	SceneDescription     string
	ChallengeDescription string
	ChallengeTaskDesc    string
	TestDescription      string
}

func getChallengeNarrativeTestCases() []ChallengeNarrativeTestCase {
	return []ChallengeNarrativeTestCase{
		{
			Name:                 "Notice task — identify failing subsystem (success)",
			ActionType:           action.Overcome,
			Skill:                "Notice",
			Description:          "Identify the failing subsystem from the control panel warnings",
			RawInput:             "John carefully examines the reactor control panel to identify the warning indicators",
			OutcomeType:          dice.Success,
			Shifts:               2,
			SceneName:            "Reactor Control Room",
			SceneDescription:     "The reactor control room aboard Europa Station. Alarms blare and warning lights flash.",
			ChallengeDescription: "Stabilize the reactor before a catastrophic meltdown",
			ChallengeTaskDesc:    "Identify the failing subsystem from the control panel warnings",
			TestDescription:      "Narrative should describe the PLAYER identifying the subsystem, not an NPC doing it for them",
		},
		{
			Name:                 "Engineering task — override lockout (failure)",
			ActionType:           action.Overcome,
			Skill:                "Engineering",
			Description:          "Override the safety lockout to restore coolant flow",
			RawInput:             "John works frantically to bypass the safety lockout and restore coolant flow",
			OutcomeType:          dice.Failure,
			Shifts:               -2,
			SceneName:            "Reactor Control Room",
			SceneDescription:     "The reactor control room. Temperature is climbing as coolant remains offline.",
			ChallengeDescription: "Stabilize the reactor before a catastrophic meltdown",
			ChallengeTaskDesc:    "Override the safety lockout to restore coolant flow",
			TestDescription:      "Failed narrative should describe the PLAYER's own failed attempt, not an NPC failing",
		},
		{
			Name:                 "Stealth task — move past guards (success with style)",
			ActionType:           action.Overcome,
			Skill:                "Stealth",
			Description:          "Move through the corridors without alerting guards",
			RawInput:             "I stick to the shadows and time my movements to the patrol rotation",
			OutcomeType:          dice.SuccessWithStyle,
			Shifts:               4,
			SceneName:            "Museum Service Corridor",
			SceneDescription:     "A dimly lit service corridor behind the main gallery. Security cameras rotate on fixed intervals.",
			ChallengeDescription: "Infiltrate the museum vault and steal the artifact",
			ChallengeTaskDesc:    "Move through the corridors without alerting guards",
			TestDescription:      "SWS narrative should describe the player's exceptional stealth, not guards simply being absent",
		},
		{
			Name:                 "Will task — resist heat and pressure (tie)",
			ActionType:           action.Overcome,
			Skill:                "Will",
			Description:          "Maintain focus through the rising heat and pressure",
			RawInput:             "John braces himself against the heat and forces himself to concentrate",
			OutcomeType:          dice.Tie,
			Shifts:               0,
			SceneName:            "Reactor Control Room",
			SceneDescription:     "The reactor room temperature is dangerously high. Sweat pours and concentration wavers.",
			ChallengeDescription: "Stabilize the reactor before a catastrophic meltdown",
			ChallengeTaskDesc:    "Maintain focus through the rising heat and pressure",
			TestDescription:      "Tie narrative should show the player pushing through at a cost — their own struggle, not NPC help",
		},
	}
}

// ChallengeNarrativeResult stores the evaluation result.
type ChallengeNarrativeResult struct {
	TestCase         ChallengeNarrativeTestCase
	RawResponse      string
	PlayerFocused    bool // true = narrative focuses on the player's attempt
	OutcomeMatch     bool // narrative tone matches outcome level
	JudgePlayerFocus JudgeResult
	JudgeOutcome     JudgeResult
	Passed           bool
	Error            error
}

func evaluateChallengeNarrative(ctx context.Context, client llm.LLMClient, tc ChallengeNarrativeTestCase) ChallengeNarrativeResult {
	result := ChallengeNarrativeResult{TestCase: tc}

	testScene := scene.NewScene("eval-scene", tc.SceneName, tc.SceneDescription)

	charCtx := "Name: John MacDougal\nHigh Concept: Resourceful Station Engineer\nTrouble: First Day on Europa"
	aspectsCtx := "Situation Aspects: Tense Atmosphere"

	act := action.NewAction("eval-action", "eval-player", tc.ActionType, tc.Skill, tc.Description)
	act.RawInput = tc.RawInput
	act.Difficulty = dice.Good
	act.Outcome = &dice.Outcome{Type: tc.OutcomeType, Shifts: tc.Shifts}

	data := promptpkg.ActionNarrativeData{
		Scene:                testScene,
		CharacterContext:     charCtx,
		AspectsContext:       aspectsCtx,
		ConversationContext:  "",
		Action:               act,
		OtherCharacters:      nil,
		ChallengeDescription: tc.ChallengeDescription,
		ChallengeTaskDesc:    tc.ChallengeTaskDesc,
	}

	promptText, err := promptpkg.RenderActionNarrative(data)
	if err != nil {
		result.Error = fmt.Errorf("render failed: %w", err)
		return result
	}

	narrative, err := llm.SimpleCompletion(ctx, client, promptText, 200, 0.8)
	if err != nil {
		result.Error = fmt.Errorf("LLM call failed: %w", err)
		return result
	}
	result.RawResponse = narrative

	// === Check 1: Player-focused — is the player character the one taking action? ===
	playerFocusQuestion := "Does this narrative describe the player character (John MacDougal / the protagonist) " +
		"performing the action themselves? Answer YES if the player character is the one making the attempt. " +
		"Answer NO if an NPC solves the task for them, or the narrative describes the task being resolved " +
		"by someone other than the player."
	playerJudgeCtx := fmt.Sprintf("The challenge task is: %s\nThe player's input was: %s\nSkill used: %s",
		tc.ChallengeTaskDesc, tc.RawInput, tc.Skill)

	judgeResult, err := LLMJudgeWithContext(ctx, client, narrative, playerFocusQuestion, playerJudgeCtx)
	if err != nil {
		result.JudgePlayerFocus = JudgeResult{Pass: false, Reasoning: fmt.Sprintf("judge error: %v", err)}
	} else {
		result.JudgePlayerFocus = judgeResult
	}
	result.PlayerFocused = result.JudgePlayerFocus.Pass

	// === Check 2: Outcome match — does the tone match the dice result? ===
	outcomeQuestion := challengeOutcomeJudgeQuestion(tc.OutcomeType)
	outcomeCtx := fmt.Sprintf("The player attempted: %s\nSkill: %s\nOutcome: %s (shifts: %+d)",
		tc.Description, tc.Skill, tc.OutcomeType.String(), tc.Shifts)

	outcomeJudge, err := LLMJudgeWithContext(ctx, client, narrative, outcomeQuestion, outcomeCtx)
	if err != nil {
		result.JudgeOutcome = JudgeResult{Pass: false, Reasoning: fmt.Sprintf("judge error: %v", err)}
	} else {
		result.JudgeOutcome = outcomeJudge
	}
	result.OutcomeMatch = result.JudgeOutcome.Pass

	result.Passed = result.PlayerFocused && result.OutcomeMatch
	return result
}

func challengeOutcomeJudgeQuestion(outcomeType dice.OutcomeType) string {
	switch outcomeType {
	case dice.Failure:
		return "Does this narrative clearly convey that the character's attempt failed or resulted in a setback?"
	case dice.Tie:
		return "Does this narrative convey that the character's attempt partially succeeded or came at a cost?"
	case dice.Success:
		return "Does this narrative convey that the character's attempt succeeded?"
	case dice.SuccessWithStyle:
		return "Does this narrative convey an impressive or exceptional success?"
	default:
		return "Does this narrative describe the result of an action?"
	}
}

// TestActionNarrative_ChallengeContext_LLMEvaluation verifies that action narratives
// during an active challenge focus on the player character's own attempt at the
// specific task, rather than having NPCs solve it or narrating unrelated tasks.
//
// Extracted from the Meltdown on Europa session where action narratives during
// challenges didn't have enough context about the challenge task being resolved.
//
// Run with: go test -v -tags=llmeval ./test/llmeval/ -run TestActionNarrative_ChallengeContext
func TestActionNarrative_ChallengeContext_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	verboseLogging := VerboseLoggingEnabled()
	testCases := getChallengeNarrativeTestCases()

	var results []ChallengeNarrativeResult
	passed := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateChallengeNarrative(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.Passed {
				passed++
			}

			if verboseLogging || !result.Passed {
				status := "PASS"
				if !result.Passed {
					status = "FAIL"
				}
				t.Logf("%s: PlayerFocused=%v OutcomeMatch=%v",
					status, result.PlayerFocused, result.OutcomeMatch)
				t.Logf("  Task: %s (%s)", tc.ChallengeTaskDesc, tc.Skill)
				t.Logf("  Outcome: %s (shifts: %+d)", tc.OutcomeType.String(), tc.Shifts)
				t.Logf("  JudgePlayer: pass=%v — %s", result.JudgePlayerFocus.Pass, result.JudgePlayerFocus.Reasoning)
				t.Logf("  JudgeOutcome: pass=%v — %s", result.JudgeOutcome.Pass, result.JudgeOutcome.Reasoning)
				t.Logf("  Narrative: %s", TruncateResponse(result.RawResponse, 300))
			}

			assert.True(t, result.PlayerFocused,
				"Narrative should focus on the PLAYER's attempt, not NPCs solving the task.\nJudge: %s\nNarrative: %s",
				result.JudgePlayerFocus.Reasoning, TruncateResponse(result.RawResponse, 300))
			assert.True(t, result.OutcomeMatch,
				"Narrative tone should match %s outcome.\nJudge: %s",
				tc.OutcomeType.String(), result.JudgeOutcome.Reasoning)
		})
	}

	// Summary
	t.Log("\n========== CHALLENGE NARRATIVE CONTEXT SUMMARY ==========")
	t.Logf("Passed: %d/%d (%.1f%%)", passed, len(testCases), float64(passed)*100/float64(len(testCases)))

	// By outcome type
	byOutcome := map[dice.OutcomeType]struct{ pass, total int }{}
	for _, r := range results {
		stats := byOutcome[r.TestCase.OutcomeType]
		stats.total++
		if r.Passed {
			stats.pass++
		}
		byOutcome[r.TestCase.OutcomeType] = stats
	}
	t.Log("\nBy Outcome:")
	for ot, stats := range byOutcome {
		t.Logf("  %s: %d/%d (%.1f%%)", ot.String(), stats.pass, stats.total, float64(stats.pass)*100/float64(stats.total))
	}

	// Failed cases
	t.Log("\n--- Failed Cases ---")
	anyFailed := false
	for _, r := range results {
		if !r.Passed && r.Error == nil {
			anyFailed = true
			t.Logf("FAIL: %s [%s → %s] — player=%v outcome=%v",
				r.TestCase.Name, r.TestCase.Skill, r.TestCase.OutcomeType.String(),
				r.PlayerFocused, r.OutcomeMatch)
			t.Logf("      Narrative: %s", TruncateResponse(r.RawResponse, 200))
		}
	}
	if !anyFailed {
		t.Log("  (none)")
	}
}
