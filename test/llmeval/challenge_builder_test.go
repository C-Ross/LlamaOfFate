//go:build llmeval

package llmeval_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ChallengeBuilderTestCase defines a scenario for the challenge builder.
type ChallengeBuilderTestCase struct {
	Name             string
	Description      string
	SceneName        string
	SceneDescription string
	PlayerSkills     []string
	SituationAspects []scene.SituationAspect
	Skills           map[string]dice.Ladder // for DifficultyGuidance computation
	MinTasks         int                    // expected minimum task count
	MaxTasks         int                    // expected maximum task count
}

func getChallengeBuilderTestCases() []ChallengeBuilderTestCase {
	return []ChallengeBuilderTestCase{
		{
			Name:             "Heist_MultiSkill",
			Description:      "Break into the baron's vault to steal the cursed gem",
			SceneName:        "The Baron's Keep",
			SceneDescription: "A fortified keep with guards, traps, and a locked vault",
			PlayerSkills:     []string{"Athletics", "Stealth", "Burglary", "Contacts", "Notice"},
			SituationAspects: []scene.SituationAspect{
				{ID: "asp-1", Aspect: "Patrolling Guards"},
				{ID: "asp-2", Aspect: "Moonless Night"},
			},
			Skills:   map[string]dice.Ladder{"Athletics": dice.Good, "Stealth": dice.Great, "Burglary": dice.Fair, "Contacts": dice.Average, "Notice": dice.Good},
			MinTasks: 3,
			MaxTasks: 5,
		},
		{
			Name:             "SimplePursuit",
			Description:      "Chase the pickpocket through the crowded market",
			SceneName:        "Market Square",
			SceneDescription: "A bustling open-air market packed with vendors and shoppers",
			PlayerSkills:     []string{"Athletics", "Notice", "Physique"},
			SituationAspects: []scene.SituationAspect{
				{ID: "asp-3", Aspect: "Crowded Stalls"},
			},
			Skills:   map[string]dice.Ladder{"Athletics": dice.Good, "Notice": dice.Fair, "Physique": dice.Average},
			MinTasks: 2,
			MaxTasks: 3,
		},
		{
			Name:             "MagicalRitual",
			Description:      "Perform an ancient ritual to seal the dimensional rift before demons pour through",
			SceneName:        "The Shattered Temple",
			SceneDescription: "A crumbling temple atop a storm-lashed peak, with a glowing rift in the sky",
			PlayerSkills:     []string{"Lore", "Will", "Crafts", "Rapport", "Athletics"},
			SituationAspects: []scene.SituationAspect{
				{ID: "asp-4", Aspect: "Unstable Dimensional Rift"},
				{ID: "asp-5", Aspect: "Howling Gale"},
				{ID: "asp-6", Aspect: "Crumbling Foundations"},
			},
			Skills:   map[string]dice.Ladder{"Lore": dice.Superb, "Will": dice.Great, "Crafts": dice.Good, "Rapport": dice.Fair, "Athletics": dice.Average},
			MinTasks: 3,
			MaxTasks: 5,
		},
		{
			Name:             "SocialInfiltration",
			Description:      "Infiltrate the governor's masquerade ball to find evidence of corruption",
			SceneName:        "Governor's Mansion",
			SceneDescription: "A grand ballroom filled with masked nobles, servants, and hidden bodyguards",
			PlayerSkills:     []string{"Deceive", "Rapport", "Notice", "Stealth", "Empathy"},
			SituationAspects: []scene.SituationAspect{},
			Skills:           map[string]dice.Ladder{"Deceive": dice.Great, "Rapport": dice.Good, "Notice": dice.Fair, "Stealth": dice.Fair, "Empathy": dice.Average},
			MinTasks:         3,
			MaxTasks:         5,
		},
	}
}

// ChallengeBuilderResult tracks the evaluation outcome for one test case.
type ChallengeBuilderResult struct {
	TestCase ChallengeBuilderTestCase
	State    *scene.ChallengeState
	Passed   bool
	Failures []string
}

func TestChallengeBuilder_LLMEvaluation(t *testing.T) {
	endpoint := os.Getenv("AZURE_API_ENDPOINT")
	apiKey := os.Getenv("AZURE_API_KEY")
	if endpoint == "" || apiKey == "" {
		t.Skip("Skipping LLM evaluation: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	verbose := os.Getenv("VERBOSE_TESTS") != ""

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	generator := engine.NewChallengeGenerator(client)

	ctx := context.Background()
	testCases := getChallengeBuilderTestCases()

	var results []ChallengeBuilderResult

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			guidance := prompt.ComputeDifficultyGuidance(tc.Skills)

			req := prompt.ChallengeBuildData{
				Description:        tc.Description,
				SceneName:          tc.SceneName,
				SceneDescription:   tc.SceneDescription,
				PlayerSkills:       tc.PlayerSkills,
				SituationAspects:   tc.SituationAspects,
				DifficultyGuidance: guidance,
			}

			state, err := generator.BuildChallenge(ctx, req)
			require.NoError(t, err, "BuildChallenge should not error")
			require.NotNil(t, state, "Returned state should not be nil")

			if verbose {
				t.Logf("=== %s ===", tc.Name)
				t.Logf("Description: %s", tc.Description)
				t.Logf("Tasks returned: %d", len(state.Tasks))
				for i, task := range state.Tasks {
					t.Logf("  [%d] %s (skill=%s, difficulty=%d): %s",
						i+1, task.ID, task.Skill, task.Difficulty, task.Description)
				}
			}

			result := ChallengeBuilderResult{TestCase: tc, State: state, Passed: true}

			// --- Structural checks ---

			// Task count within expected range
			if len(state.Tasks) < tc.MinTasks || len(state.Tasks) > tc.MaxTasks {
				msg := "task count %d not in [%d, %d]"
				result.Failures = append(result.Failures, msg)
				t.Errorf(msg, len(state.Tasks), tc.MinTasks, tc.MaxTasks)
			}

			// Description propagated
			assert.Equal(t, tc.Description, state.Description, "Description should match request")

			// All tasks are pending with sequential IDs
			for i, task := range state.Tasks {
				assert.Equalf(t, scene.TaskPending, task.Status, "task-%d status", i+1)
				assert.NotEmpty(t, task.ID, "task ID should be set")
			}

			// --- Skill validity checks ---
			skillSet := makeStringSet(tc.PlayerSkills)
			for _, task := range state.Tasks {
				if !skillSet[task.Skill] {
					msg := "task uses unknown skill: " + task.Skill
					result.Failures = append(result.Failures, msg)
					t.Error(msg)
				}
			}

			// All tasks use different skills
			usedSkills := map[string]bool{}
			for _, task := range state.Tasks {
				if usedSkills[task.Skill] {
					msg := "duplicate skill: " + task.Skill
					result.Failures = append(result.Failures, msg)
					t.Error(msg)
				}
				usedSkills[task.Skill] = true
			}

			// --- Difficulty checks ---
			for _, task := range state.Tasks {
				if task.Difficulty < guidance.DifficultyMin || task.Difficulty > guidance.DifficultyMax {
					msg := "task %s difficulty %d outside range [%d, %d]"
					result.Failures = append(result.Failures, msg)
					t.Errorf(msg, task.Skill, task.Difficulty, guidance.DifficultyMin, guidance.DifficultyMax)
				}
			}

			// --- Description quality checks ---
			for _, task := range state.Tasks {
				words := len(strings.Fields(task.Description))
				if words < 3 || words > 20 {
					msg := "task description word count %d out of range [3,20]: %s"
					result.Failures = append(result.Failures, msg)
					t.Errorf(msg, words, task.Description)
				}
			}

			// --- Core type integration ---
			// Verify state IS a real ChallengeState by calling methods
			pending := state.PendingTasks()
			assert.Equal(t, len(state.Tasks), len(pending), "All tasks should start pending")
			assert.False(t, state.IsComplete(), "Challenge should not be complete initially")

			if len(result.Failures) > 0 {
				result.Passed = false
			}
			results = append(results, result)
		})
	}

	// --- Summary ---
	passed, failed := 0, 0
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	t.Log("\n========== CHALLENGE BUILDER TEST SUMMARY ==========")
	t.Logf("Total: %d  Passed: %d  Failed: %d  Accuracy: %.0f%%",
		len(results), passed, failed, float64(passed)*100/float64(len(results)))

	if failed > 0 {
		t.Log("\n--- Failed Cases ---")
		for _, r := range results {
			if !r.Passed {
				t.Logf("FAIL: %s — %v", r.TestCase.Name, r.Failures)
			}
		}
	}
}

// makeStringSet converts a string slice to a lookup set.
func makeStringSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}
