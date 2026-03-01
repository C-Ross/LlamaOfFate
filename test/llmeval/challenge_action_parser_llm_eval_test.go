//go:build llmeval

package llmeval_test

import (
	"context"
	"os"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ChallengeActionParserTestCase tests that actions during active challenges
// are parsed as Overcome (not Create an Advantage) when matching a pending task.
type ChallengeActionParserTestCase struct {
	Name           string
	RawInput       string
	Context        string
	ChallengeTasks []scene.ChallengeTask
	ExpectedType   action.ActionType
	ExpectedSkills []string
	Description    string
}

func getChallengeOvercomeCases() []ChallengeActionParserTestCase {
	return []ChallengeActionParserTestCase{
		{
			Name:     "Notice scan during challenge",
			RawInput: "I carefully scan for a safe path through the rubble",
			Context:  "A crumbling mineshaft with falling rocks and loose timbers",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "t1", Skill: "Notice", Difficulty: 2, Status: scene.TaskPending, Description: "Spot safe path"},
				{ID: "t2", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending, Description: "Dodge falling rocks"},
			},
			ExpectedType:   action.Overcome,
			ExpectedSkills: []string{"Notice"},
			Description:    "Notice maps to a challenge task — should be Overcome, not Create Advantage",
		},
		{
			Name:     "Stealth bypass during challenge",
			RawInput: "I quietly slip past the laser grid",
			Context:  "A high-security vault with laser grids and pressure plates",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "t1", Skill: "Stealth", Difficulty: 3, Status: scene.TaskPending, Description: "Bypass laser grid"},
				{ID: "t2", Skill: "Burglary", Difficulty: 4, Status: scene.TaskPending, Description: "Crack the lock"},
			},
			ExpectedType:   action.Overcome,
			ExpectedSkills: []string{"Stealth"},
			Description:    "Stealth maps to a challenge task — should be Overcome",
		},
		{
			Name:     "Athletics dodge during challenge",
			RawInput: "I dodge the falling rocks and sprint for cover",
			Context:  "A crumbling mineshaft with rocks raining from the ceiling",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "t1", Skill: "Notice", Difficulty: 2, Status: scene.TaskPending, Description: "Spot safe path"},
				{ID: "t2", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending, Description: "Dodge falling rocks"},
			},
			ExpectedType:   action.Overcome,
			ExpectedSkills: []string{"Athletics"},
			Description:    "Athletics maps to a challenge task — should be Overcome",
		},
		{
			Name:     "Burglary crack lock during challenge",
			RawInput: "I work on cracking the combination lock",
			Context:  "Standing before a massive vault door with a combination lock",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "t1", Skill: "Stealth", Difficulty: 3, Status: scene.TaskPending, Description: "Bypass laser grid"},
				{ID: "t2", Skill: "Burglary", Difficulty: 4, Status: scene.TaskPending, Description: "Crack the combination lock"},
			},
			ExpectedType:   action.Overcome,
			ExpectedSkills: []string{"Burglary"},
			Description:    "Burglary maps to a challenge task — should be Overcome",
		},
		{
			Name:     "Will resist during challenge",
			RawInput: "I steel myself against the supernatural dread",
			Context:  "An ancient crypt where shadows move on their own and whispers fill the air",
			ChallengeTasks: []scene.ChallengeTask{
				{ID: "t1", Skill: "Will", Difficulty: 3, Status: scene.TaskPending, Description: "Resist supernatural dread"},
				{ID: "t2", Skill: "Lore", Difficulty: 2, Status: scene.TaskPending, Description: "Decipher warding glyphs"},
			},
			ExpectedType:   action.Overcome,
			ExpectedSkills: []string{"Will"},
			Description:    "Will maps to a challenge task — should be Overcome",
		},
	}
}

type ChallengeActionParserResult struct {
	TestCase        ChallengeActionParserTestCase
	ActualType      action.ActionType
	ActualSkill     string
	TypeMatches     bool
	SkillAcceptable bool
	Error           error
}

func getChallengeTestCharacter() *character.Character {
	char := character.NewCharacter("eval-char", "Magnus the Versatile")
	char.Aspects.HighConcept = "Resourceful Problem Solver"
	char.Aspects.Trouble = "Curiosity Killed the Cat"
	char.SetSkill("Athletics", dice.Good)
	char.SetSkill("Notice", dice.Fair)
	char.SetSkill("Stealth", dice.Good)
	char.SetSkill("Burglary", dice.Good)
	char.SetSkill("Will", dice.Fair)
	char.SetSkill("Lore", dice.Fair)
	char.SetSkill("Fight", dice.Fair)
	return char
}

func evaluateChallengeAction(ctx context.Context, parser engine.ActionParser, char *character.Character, tc ChallengeActionParserTestCase) ChallengeActionParserResult {
	// Build a Scene with an active challenge so the parser populates ChallengeContext
	testScene := scene.NewScene("test-scene", "Challenge Scene", tc.Context)
	err := testScene.StartChallenge("Active challenge", tc.ChallengeTasks)
	if err != nil {
		return ChallengeActionParserResult{TestCase: tc, Error: err}
	}

	req := engine.ActionParseRequest{
		Character: char,
		RawInput:  tc.RawInput,
		Context:   tc.Context,
		Scene:     testScene,
	}

	parsed, err := parser.ParseAction(ctx, req)
	if err != nil {
		return ChallengeActionParserResult{TestCase: tc, Error: err}
	}

	skillOK := false
	for _, s := range tc.ExpectedSkills {
		if parsed.Skill == s {
			skillOK = true
			break
		}
	}

	return ChallengeActionParserResult{
		TestCase:        tc,
		ActualType:      parsed.Type,
		ActualSkill:     parsed.Skill,
		TypeMatches:     parsed.Type == tc.ExpectedType,
		SkillAcceptable: skillOK,
	}
}

// TestChallengeActionParser_LLMEvaluation verifies that the action parser
// returns Overcome (not Create Advantage) for actions matching challenge tasks.
// Run with: go test -v -tags=llmeval ./test/llmeval/ -run TestChallengeActionParser
func TestChallengeActionParser_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	parser := engine.NewActionParser(client)
	char := getChallengeTestCharacter()
	ctx := context.Background()

	cases := getChallengeOvercomeCases()
	var results []ChallengeActionParserResult
	typeCorrect, skillCorrect := 0, 0

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateChallengeAction(ctx, parser, char, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.TypeMatches {
				typeCorrect++
			}
			if result.SkillAcceptable {
				skillCorrect++
			}

			assert.Equal(t, tc.ExpectedType, result.ActualType,
				"Action type mismatch for '%s'. %s", tc.RawInput, tc.Description)
			assert.True(t, result.SkillAcceptable,
				"Skill mismatch for '%s': expected one of %v, got %s",
				tc.RawInput, tc.ExpectedSkills, result.ActualSkill)
		})
	}

	// Summary
	total := len(cases)
	t.Log("\n========== CHALLENGE ACTION PARSER SUMMARY ==========")
	t.Logf("Type accuracy:  %d/%d (%.1f%%)", typeCorrect, total, float64(typeCorrect)*100/float64(total))
	t.Logf("Skill accuracy: %d/%d (%.1f%%)", skillCorrect, total, float64(skillCorrect)*100/float64(total))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.TypeMatches {
			t.Logf("FAIL: '%s'", r.TestCase.RawInput)
			t.Logf("      Expected: %s, Got: %s", r.TestCase.ExpectedType, r.ActualType)
			t.Logf("      Skill: expected %v, got %s", r.TestCase.ExpectedSkills, r.ActualSkill)
			t.Logf("      Why: %s", r.TestCase.Description)
		}
	}
}
