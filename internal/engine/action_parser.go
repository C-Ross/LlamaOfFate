package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
)

// ActionParseRequest represents a request to parse user input into an action
type ActionParseRequest struct {
	Character *character.Character `json:"character"`
	RawInput  string               `json:"raw_input"`
	Context   string               `json:"context,omitempty"` // Scene description, recent events, etc.
}

// ActionParseResponse represents the LLM's response for action parsing
type ActionParseResponse struct {
	ActionType  string `json:"action_type"` // "Overcome", "Create an Advantage", "Attack", "Defend"
	Skill       string `json:"skill"`       // The Fate Core skill to use
	Description string `json:"description"` // Clean description of what they're trying to do
	Target      string `json:"target"`      // The target of the action (character, object, or description)
	Reasoning   string `json:"reasoning"`   // Explanation of the choice
	Confidence  int    `json:"confidence"`  // 1-10 scale of how confident the LLM is
}

// ActionParser handles parsing user input into structured actions using LLM
type ActionParser struct {
	llmClient llm.LLMClient
}

// NewActionParser creates a new action parser with the given LLM client
func NewActionParser(llmClient llm.LLMClient) *ActionParser {
	return &ActionParser{
		llmClient: llmClient,
	}
}

// ParseAction analyzes user input and returns a structured action using LLM
func (ap *ActionParser) ParseAction(ctx context.Context, req ActionParseRequest) (*action.Action, error) {
	// Build the LLM prompt using templates
	systemPrompt, err := ap.buildSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to build system prompt: %w", err)
	}

	userPrompt, err := ap.buildUserPrompt(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build user prompt: %w", err)
	}

	// Create the LLM request
	llmReq := llm.CompletionRequest{
		Messages: []llm.Message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		MaxTokens:   300,
		Temperature: 0.3, // Lower temperature for more consistent parsing
	}

	// Get LLM response
	resp, err := ap.llmClient.ChatCompletion(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse the JSON response
	var parseResp ActionParseResponse
	cleanedContent := cleanJSONResponse(resp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(cleanedContent), &parseResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w\nResponse was: %s", err, resp.Choices[0].Message.Content)
	}

	// Convert string action type to enum
	actionType, err := parseActionType(parseResp.ActionType)
	if err != nil {
		return nil, fmt.Errorf("invalid action type: %w", err)
	}

	// Create the action
	actionID := generateActionID()
	parsedAction := action.NewAction(
		actionID,
		req.Character.ID,
		actionType,
		parseResp.Skill,
		parseResp.Description,
	)
	parsedAction.RawInput = req.RawInput
	parsedAction.Target = parseResp.Target

	return parsedAction, nil
}

// buildSystemPrompt creates the system prompt using templates
func (ap *ActionParser) buildSystemPrompt() (string, error) {
	var buf bytes.Buffer
	if err := ActionParseSystemPrompt.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("failed to execute system prompt template: %w", err)
	}
	return buf.String(), nil
}

// buildUserPrompt creates the user prompt using templates
func (ap *ActionParser) buildUserPrompt(req ActionParseRequest) (string, error) {
	var buf bytes.Buffer
	if err := ActionParsePrompt.Execute(&buf, req); err != nil {
		return "", fmt.Errorf("failed to execute user prompt template: %w", err)
	}
	return buf.String(), nil
}

// parseActionType converts string action type to enum
func parseActionType(actionTypeStr string) (action.ActionType, error) {
	switch actionTypeStr {
	case "Overcome":
		return action.Overcome, nil
	case "Create an Advantage":
		return action.CreateAdvantage, nil
	case "Attack":
		return action.Attack, nil
	case "Defend":
		return action.Defend, nil
	default:
		return action.Overcome, fmt.Errorf("unknown action type: %s", actionTypeStr)
	}
}

// generateActionID creates a unique action ID
func generateActionID() string {
	// Simple implementation using timestamp
	return fmt.Sprintf("action-%d", time.Now().UnixNano())
}

// cleanJSONResponse removes markdown formatting from LLM JSON responses
func cleanJSONResponse(content string) string {
	// Remove markdown code block formatting
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

	// If we found JSON blocks, use the last one
	if len(jsonBlocks) > 0 {
		return jsonBlocks[len(jsonBlocks)-1]
	}

	// Fallback: simple cleanup
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
	}

	if strings.HasSuffix(content, "```") {
		content = strings.TrimSuffix(content, "```")
	}

	// Trim any remaining whitespace
	content = strings.TrimSpace(content)

	return content
}
