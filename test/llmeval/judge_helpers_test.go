//go:build llmeval

package llmeval_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
)

// JudgeResult holds the LLM judge's verdict on a behavioral evaluation.
type JudgeResult struct {
	Pass      bool   `json:"pass"`
	Reasoning string `json:"reasoning"`
}

// LLMJudge sends a second LLM call to evaluate whether a response meets a behavioral criterion.
// The question should be phrased so that YES/true = pass.
func LLMJudge(ctx context.Context, client llm.LLMClient, response string, question string) (JudgeResult, error) {
	prompt := fmt.Sprintf(`You are an evaluator for an RPG game engine's LLM output. You will be given a response and a yes/no question about it. Answer strictly based on the text provided.

<response>
%s
</response>

<question>
%s
</question>

Respond with ONLY a JSON object, no other text:
{"pass": true, "reasoning": "one sentence explanation"}

Where "pass" is true if the answer to the question is YES, false if NO.`, response, question)

	raw, err := llm.SimpleCompletion(ctx, client, prompt, 200, 0.1)
	if err != nil {
		return JudgeResult{}, fmt.Errorf("judge LLM call failed: %w", err)
	}
	return parseJudgeResponse(raw)
}

// LLMJudgeWithContext is like LLMJudge but adds a context block before the response block.
// Used when the judge needs reference material to evaluate against.
func LLMJudgeWithContext(ctx context.Context, client llm.LLMClient, response string, question string, additionalContext string) (JudgeResult, error) {
	prompt := fmt.Sprintf(`You are an evaluator for an RPG game engine's LLM output. You will be given a response and a yes/no question about it. Answer strictly based on the text provided.

<context>
%s
</context>

<response>
%s
</response>

<question>
%s
</question>

Respond with ONLY a JSON object, no other text:
{"pass": true, "reasoning": "one sentence explanation"}

Where "pass" is true if the answer to the question is YES, false if NO.`, additionalContext, response, question)

	raw, err := llm.SimpleCompletion(ctx, client, prompt, 200, 0.1)
	if err != nil {
		return JudgeResult{}, fmt.Errorf("judge LLM call failed: %w", err)
	}
	return parseJudgeResponse(raw)
}

// parseJudgeResponse parses the judge's raw text response into a JudgeResult.
func parseJudgeResponse(raw string) (JudgeResult, error) {
	raw = strings.TrimSpace(raw)

	// Strip markdown code fences
	if strings.HasPrefix(raw, "```") {
		lines := strings.SplitN(raw, "\n", 2)
		if len(lines) > 1 {
			raw = lines[1]
		}
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}

	// Find JSON by locating { and } boundaries
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return JudgeResult{}, fmt.Errorf("no JSON object found in judge response: %s", truncateForError(raw))
	}
	raw = raw[start : end+1]

	var result JudgeResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return JudgeResult{}, fmt.Errorf("failed to parse judge response: %w (raw: %s)", err, truncateForError(raw))
	}
	return result, nil
}

// truncateForError truncates to 200 chars for error messages.
func truncateForError(s string) string {
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
