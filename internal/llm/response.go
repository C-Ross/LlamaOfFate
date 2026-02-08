package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// SimpleCompletion sends a single user-message prompt and returns the text content.
// Returns ErrEmptyResponse if the response is empty after cleaning.
func SimpleCompletion(ctx context.Context, client LLMClient, prompt string, maxTokens int, temperature float64) (string, error) {
	resp, err := client.ChatCompletion(ctx, CompletionRequest{
		Messages:    []Message{{Role: "user", Content: prompt}},
		MaxTokens:   maxTokens,
		Temperature: temperature,
	})
	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}
	if resp.Content() == "" {
		return "", ErrEmptyResponse
	}
	return resp.Content(), nil
}

// Content returns the message content from the first completion choice.
// Returns an empty string when no choices are present.
// Logs a warning if the response contains multiple choices (only the first is used).
func (r *CompletionResponse) Content() string {
	if len(r.Choices) == 0 {
		return ""
	}
	if len(r.Choices) > 1 {
		slog.Warn("LLM response contained multiple choices, using first",
			slog.String("component", "llm"),
			slog.Int("choice_count", len(r.Choices)),
		)
	}
	return r.Choices[0].Message.Content
}

// CleanContent strips markdown code fences and trims whitespace from all
// choice messages in the response. This is called automatically by LLM
// client implementations so callers don't need to clean responses manually.
func (r *CompletionResponse) CleanContent() {
	for i := range r.Choices {
		r.Choices[i].Message.Content = CleanJSONResponse(r.Choices[i].Message.Content)
	}
}

// CleanJSONResponse removes markdown formatting from LLM JSON responses.
// LLMs often wrap JSON output in ```json code blocks; this extracts the raw JSON.
// When multiple JSON blocks are present, the last one is returned (the corrected response).
func CleanJSONResponse(content string) string {
	content = strings.TrimSpace(content)

	// If there are multiple JSON blocks, take the last one (the corrected response)
	blocks := strings.Split(content, "```")
	var jsonBlocks []string

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if strings.HasPrefix(block, "json\n") {
			block = strings.TrimPrefix(block, "json\n")
			block = strings.TrimSpace(block)
			if strings.HasPrefix(block, "{") && strings.HasSuffix(block, "}") {
				jsonBlocks = append(jsonBlocks, block)
			}
		} else if strings.HasPrefix(block, "{") && strings.HasSuffix(block, "}") {
			jsonBlocks = append(jsonBlocks, block)
		}
	}

	if len(jsonBlocks) > 0 {
		return jsonBlocks[len(jsonBlocks)-1]
	}

	// Fallback: simple cleanup
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
	}

	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	return content
}
