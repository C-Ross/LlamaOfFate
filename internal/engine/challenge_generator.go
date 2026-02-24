package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
)

// ChallengeGenerator builds a ChallengeState from a narrative description.
// The LLM determines the task breakdown (skills, difficulties, descriptions)
// while this generator ensures the result is valid core data.
type ChallengeGenerator interface {
	BuildChallenge(ctx context.Context, req prompt.ChallengeBuildData) (*scene.ChallengeState, error)
}

// Compile-time check that LLMChallengeGenerator satisfies ChallengeGenerator.
var _ ChallengeGenerator = (*LLMChallengeGenerator)(nil)

// LLMChallengeGenerator uses an LLM to turn a challenge description into
// a structured ChallengeState with tasks.
type LLMChallengeGenerator struct {
	llmClient llm.LLMClient
}

// NewChallengeGenerator creates a new LLM-backed challenge generator.
func NewChallengeGenerator(llmClient llm.LLMClient) *LLMChallengeGenerator {
	return &LLMChallengeGenerator{llmClient: llmClient}
}

// BuildChallenge renders the challenge build prompt, calls the LLM, and parses
// the JSON response into a ChallengeState. Task IDs and pending status are
// assigned automatically.
func (cg *LLMChallengeGenerator) BuildChallenge(ctx context.Context, req prompt.ChallengeBuildData) (*scene.ChallengeState, error) {
	if req.Description == "" {
		return nil, fmt.Errorf("challenge description cannot be empty")
	}

	promptText, err := prompt.RenderChallengeBuild(req)
	if err != nil {
		return nil, fmt.Errorf("failed to render challenge build prompt: %w", err)
	}

	llmReq := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: promptText},
		},
		MaxTokens:   1024,
		Temperature: 0.7,
		TopP:        0.9,
	}

	response, err := cg.llmClient.ChatCompletion(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate challenge tasks: %w", err)
	}

	return cg.parseResponse(response.Content(), req.Description)
}

// parseResponse extracts JSON from the LLM response and builds a ChallengeState.
// The LLM returns {"tasks": [...]}, which maps directly to ChallengeState.Tasks.
// We then assign sequential IDs and set each task to pending.
func (cg *LLMChallengeGenerator) parseResponse(content string, description string) (*scene.ChallengeState, error) {
	cleaned := llm.CleanJSONResponse(content)

	var state scene.ChallengeState
	if err := json.Unmarshal([]byte(cleaned), &state); err != nil {
		return nil, fmt.Errorf("failed to parse challenge JSON: %w", err)
	}

	if len(state.Tasks) == 0 {
		return nil, fmt.Errorf("challenge must have at least one task")
	}

	state.Description = description
	for i := range state.Tasks {
		state.Tasks[i].ID = fmt.Sprintf("task-%d", i+1)
		state.Tasks[i].Status = scene.TaskPending
	}

	return &state, nil
}
