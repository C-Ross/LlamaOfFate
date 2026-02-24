package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChallengeGenerator(t *testing.T) {
	mockClient := &MockLLMClient{}
	generator := NewChallengeGenerator(mockClient)

	assert.NotNil(t, generator)
	assert.Equal(t, mockClient, generator.llmClient)
}

func TestBuildChallenge_Success(t *testing.T) {
	mockClient := &MockLLMClient{
		response: `{
			"tasks": [
				{"skill": "Athletics", "difficulty": 3, "description": "Scale the outer wall of the fortress"},
				{"skill": "Stealth", "difficulty": 2, "description": "Sneak past the guards unnoticed"},
				{"skill": "Burglary", "difficulty": 4, "description": "Pick the vault's complex lock"}
			]
		}`,
	}

	generator := NewChallengeGenerator(mockClient)

	req := prompt.ChallengeBuildData{
		Description:      "Break into the baron's vault",
		SceneName:        "The Baron's Keep",
		SceneDescription: "A fortified keep with guards and traps",
		PlayerSkills:     []string{"Athletics", "Stealth", "Burglary", "Fight"},
	}

	state, err := generator.BuildChallenge(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "Break into the baron's vault", state.Description)
	assert.Len(t, state.Tasks, 3)

	// Verify task IDs are auto-assigned
	assert.Equal(t, "task-1", state.Tasks[0].ID)
	assert.Equal(t, "task-2", state.Tasks[1].ID)
	assert.Equal(t, "task-3", state.Tasks[2].ID)

	// Verify all tasks start as pending
	for _, task := range state.Tasks {
		assert.Equal(t, scene.TaskPending, task.Status)
	}

	// Verify task content is preserved from LLM
	assert.Equal(t, "Athletics", state.Tasks[0].Skill)
	assert.Equal(t, 3, state.Tasks[0].Difficulty)
	assert.Equal(t, "Scale the outer wall of the fortress", state.Tasks[0].Description)
}

func TestBuildChallenge_CodeFencedJSON(t *testing.T) {
	// LLMs sometimes wrap JSON in code fences — CleanJSONResponse strips them
	mockClient := &MockLLMClient{
		response: "```json\n{\"tasks\": [{\"skill\": \"Lore\", \"difficulty\": 2, \"description\": \"Decode the ancient runes\"}]}\n```",
	}

	generator := NewChallengeGenerator(mockClient)

	req := prompt.ChallengeBuildData{
		Description:  "Decipher the ancient map",
		SceneName:    "The Library",
		PlayerSkills: []string{"Lore"},
	}

	state, err := generator.BuildChallenge(context.Background(), req)

	require.NoError(t, err)
	require.Len(t, state.Tasks, 1)
	assert.Equal(t, "Lore", state.Tasks[0].Skill)
}

func TestBuildChallenge_EmptyDescription(t *testing.T) {
	generator := NewChallengeGenerator(&MockLLMClient{})

	req := prompt.ChallengeBuildData{Description: ""}
	_, err := generator.BuildChallenge(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "challenge description cannot be empty")
}

func TestBuildChallenge_LLMError(t *testing.T) {
	mockClient := &MockLLMClient{err: fmt.Errorf("service unavailable")}

	generator := NewChallengeGenerator(mockClient)

	req := prompt.ChallengeBuildData{
		Description:  "Escape the dungeon",
		PlayerSkills: []string{"Athletics"},
	}

	_, err := generator.BuildChallenge(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate challenge tasks")
}

func TestBuildChallenge_InvalidJSON(t *testing.T) {
	mockClient := &MockLLMClient{response: "not valid json at all"}

	generator := NewChallengeGenerator(mockClient)

	req := prompt.ChallengeBuildData{
		Description:  "Escape the dungeon",
		PlayerSkills: []string{"Athletics"},
	}

	_, err := generator.BuildChallenge(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse challenge JSON")
}

func TestBuildChallenge_EmptyTasks(t *testing.T) {
	mockClient := &MockLLMClient{response: `{"tasks": []}`}

	generator := NewChallengeGenerator(mockClient)

	req := prompt.ChallengeBuildData{
		Description:  "Do something",
		PlayerSkills: []string{"Athletics"},
	}

	_, err := generator.BuildChallenge(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "challenge must have at least one task")
}

func TestBuildChallenge_ReusesCoreChallengeTask(t *testing.T) {
	// Verify the generator produces scene.ChallengeState directly —
	// no intermediate DTO involved.
	mockClient := &MockLLMClient{
		response: `{"tasks": [{"skill": "Rapport", "difficulty": 2, "description": "Negotiate passage"}]}`,
	}

	generator := NewChallengeGenerator(mockClient)

	req := prompt.ChallengeBuildData{
		Description:  "Cross the bridge",
		PlayerSkills: []string{"Rapport"},
	}

	state, err := generator.BuildChallenge(context.Background(), req)

	require.NoError(t, err)
	// The returned type IS scene.ChallengeState — call its methods
	pending := state.PendingTasks()
	assert.Len(t, pending, 1)
	assert.False(t, state.IsComplete())
}
