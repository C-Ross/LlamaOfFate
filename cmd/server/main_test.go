package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/ui/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLMClient returns canned responses for testing.
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) ChatCompletion(_ context.Context, _ llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.CompletionResponse{
		ID:      "test",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "test-model",
		Choices: []llm.CompletionResponseChoice{
			{
				Index:        0,
				Message:      llm.Message{Role: "assistant", Content: m.response},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (m *mockLLMClient) ChatCompletionStream(_ context.Context, _ llm.CompletionRequest, _ llm.StreamHandler) error {
	return fmt.Errorf("not implemented")
}

func (m *mockLLMClient) GetModelInfo() llm.ModelInfo {
	return llm.ModelInfo{Name: "test-model", Provider: "test"}
}

const validScenarioJSON = `{
	"title": "Neon Shadows",
	"problem": "A rogue AI threatens the city's power grid",
	"story_questions": [
		"Can the hacker stop the AI before the blackout?",
		"Will the megacorp cover up the truth?"
	],
	"setting": "A sprawling cyberpunk metropolis of neon and chrome",
	"genre": "Cyberpunk"
}`

func TestGenerateScenario_Success(t *testing.T) {
	client := &mockLLMClient{response: validScenarioJSON}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	scenario, err := generateScenario(context.Background(), client, custom)
	require.NoError(t, err)
	assert.Equal(t, "Neon Shadows", scenario.Title)
	assert.Equal(t, "A rogue AI threatens the city's power grid", scenario.Problem)
	assert.Equal(t, "Cyberpunk", scenario.Genre)
	assert.Len(t, scenario.StoryQuestions, 2)
}

func TestGenerateScenario_LLMError(t *testing.T) {
	client := &mockLLMClient{err: fmt.Errorf("service unavailable")}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	_, err := generateScenario(context.Background(), client, custom)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM completion")
}

func TestGenerateScenario_InvalidJSON(t *testing.T) {
	client := &mockLLMClient{response: "this is not json at all"}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	_, err := generateScenario(context.Background(), client, custom)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse scenario")
}

func TestGenerateScenario_MissingTitle(t *testing.T) {
	client := &mockLLMClient{response: `{"problem": "something bad"}`}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	_, err := generateScenario(context.Background(), client, custom)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse scenario")
}

func TestGenerateScenario_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := &mockLLMClient{err: ctx.Err()}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	_, err := generateScenario(ctx, client, custom)
	require.Error(t, err)
}

func TestGenerateScenario_MarkdownWrappedJSON(t *testing.T) {
	// LLMs sometimes wrap JSON in markdown code fences
	wrapped := "```json\n" + validScenarioJSON + "\n```"
	client := &mockLLMClient{response: wrapped}
	custom := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	scenario, err := generateScenario(context.Background(), client, custom)
	require.NoError(t, err)
	assert.Equal(t, "Neon Shadows", scenario.Title)
}

func TestWebUIURL(t *testing.T) {
	assert.Equal(t, "http://localhost:8080", webUIURL("8080"))
}

// ---------------------------------------------------------------------------
// buildCustomPlayerFromSetup tests
// ---------------------------------------------------------------------------

func TestBuildCustomPlayerFromSetup_DefaultSkills(t *testing.T) {
	setup := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
	}

	player, err := buildCustomPlayerFromSetup(setup)
	require.NoError(t, err)
	assert.Equal(t, "Nova", player.Name)
	assert.Equal(t, "Rogue AI Whisperer", player.Aspects.HighConcept)
	assert.Equal(t, "Trusts Machines More Than People", player.Aspects.Trouble)
	// Default cyberpunk pyramid: Burglary at Great
	assert.Equal(t, dice.Great, player.GetSkill(core.SkillBurglary))
}

func TestBuildCustomPlayerFromSetup_WithAspects(t *testing.T) {
	setup := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
		Aspects:     []string{"Connected to the Underground", "Cybernetic Eye"},
	}

	player, err := buildCustomPlayerFromSetup(setup)
	require.NoError(t, err)
	assert.Contains(t, player.Aspects.OtherAspects, "Connected to the Underground")
	assert.Contains(t, player.Aspects.OtherAspects, "Cybernetic Eye")
}

func TestBuildCustomPlayerFromSetup_WithValidSkills(t *testing.T) {
	setup := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
		Skills: map[string]int{
			core.SkillShoot:       int(dice.Great),
			core.SkillAthletics:   int(dice.Good),
			core.SkillFight:       int(dice.Good),
			core.SkillNotice:      int(dice.Fair),
			core.SkillStealth:     int(dice.Fair),
			core.SkillWill:        int(dice.Fair),
			core.SkillPhysique:    int(dice.Average),
			core.SkillProvoke:     int(dice.Average),
			core.SkillRapport:     int(dice.Average),
			core.SkillInvestigate: int(dice.Average),
		},
	}

	player, err := buildCustomPlayerFromSetup(setup)
	require.NoError(t, err)
	assert.Equal(t, dice.Great, player.GetSkill(core.SkillShoot))
	assert.Equal(t, dice.Good, player.GetSkill(core.SkillAthletics))
	// Default pyramid NOT applied — Burglary should be zero.
	assert.Equal(t, dice.Ladder(0), player.GetSkill(core.SkillBurglary))
}

func TestBuildCustomPlayerFromSetup_InvalidSkills(t *testing.T) {
	setup := &web.CustomSetup{
		Name:        "Nova",
		HighConcept: "Rogue AI Whisperer",
		Trouble:     "Trusts Machines More Than People",
		Genre:       "Cyberpunk",
		Skills: map[string]int{
			"Hacking": int(dice.Great),
		},
	}

	_, err := buildCustomPlayerFromSetup(setup)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid skill pyramid")
}

func TestBuildCustomPlayerFromSetup_WithAspectsAndSkills(t *testing.T) {
	setup := &web.CustomSetup{
		Name:        "Lyra",
		HighConcept: "Wandering Bard",
		Trouble:     "Can't Say No to a Good Story",
		Genre:       "Fantasy",
		Aspects:     []string{"Silver Tongue", "Lute of the Ancients"},
		Skills: map[string]int{
			core.SkillRapport:     int(dice.Great),
			core.SkillEmpathy:     int(dice.Good),
			core.SkillLore:        int(dice.Good),
			core.SkillNotice:      int(dice.Fair),
			core.SkillWill:        int(dice.Fair),
			core.SkillDeceive:     int(dice.Fair),
			core.SkillAthletics:   int(dice.Average),
			core.SkillStealth:     int(dice.Average),
			core.SkillInvestigate: int(dice.Average),
			core.SkillContacts:    int(dice.Average),
		},
	}

	player, err := buildCustomPlayerFromSetup(setup)
	require.NoError(t, err)
	assert.Equal(t, "Lyra", player.Name)
	assert.Equal(t, "Wandering Bard", player.Aspects.HighConcept)
	assert.Contains(t, player.Aspects.OtherAspects, "Silver Tongue")
	assert.Contains(t, player.Aspects.OtherAspects, "Lute of the Ancients")
	assert.Equal(t, dice.Great, player.GetSkill(core.SkillRapport))
}

func TestBuildCustomPlayerFromSetup_BackwardCompatible(t *testing.T) {
	// Old-style setup with just 4 fields — should produce same result as buildCustomPlayer.
	setup := &web.CustomSetup{
		Name:        "Jesse",
		HighConcept: "Outlaw with a Heart of Gold",
		Trouble:     "Wanted Dead or Alive",
		Genre:       "Western",
	}

	newPlayer, err := buildCustomPlayerFromSetup(setup)
	require.NoError(t, err)

	oldPlayer := buildCustomPlayer(setup.Name, setup.HighConcept, setup.Trouble, setup.Genre)

	assert.Equal(t, oldPlayer.Name, newPlayer.Name)
	assert.Equal(t, oldPlayer.Aspects.HighConcept, newPlayer.Aspects.HighConcept)
	assert.Equal(t, oldPlayer.Aspects.Trouble, newPlayer.Aspects.Trouble)
	assert.Equal(t, oldPlayer.GetSkill(core.SkillShoot), newPlayer.GetSkill(core.SkillShoot))
	assert.Equal(t, oldPlayer.GetSkill(core.SkillAthletics), newPlayer.GetSkill(core.SkillAthletics))
}
