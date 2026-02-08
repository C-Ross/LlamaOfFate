package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
)

// ActionParseRequest represents a request to parse user input into an action
type ActionParseRequest struct {
	Character       *character.Character   `json:"character"`
	RawInput        string                 `json:"raw_input"`
	Context         string                 `json:"context,omitempty"`          // Scene description, recent events, etc.
	Scene           interface{}            `json:"scene,omitempty"`            // Current scene object
	OtherCharacters []*character.Character `json:"other_characters,omitempty"` // Other characters in the scene
}

// ActionParseResponse represents the LLM's response for action parsing
type ActionParseResponse struct {
	ActionType  string `json:"action_type"` // "Overcome", "Create an Advantage", "Attack", "Defend"
	Skill       string `json:"skill"`       // The Fate Core skill to use
	Description string `json:"description"` // Clean description of what they're trying to do
	Target      string `json:"target"`      // The target of the action (if any)
	Difficulty  int    `json:"difficulty"`  // Passive opposition rating on the Fate ladder (-2 to +8)
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

	slog.Debug("Action parser LLM request",
		"component", "action_parser",
		"system_prompt", systemPrompt,
		"user_prompt", userPrompt)

	// Get LLM response
	resp, err := ap.llmClient.ChatCompletion(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Parse the JSON response
	var parseResp ActionParseResponse
	if err := json.Unmarshal([]byte(resp.Content()), &parseResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w\nResponse was: %s", err, resp.Content())
	}

	// Convert string action type to enum
	actionType, err := parseActionType(parseResp.ActionType)
	if err != nil {
		return nil, fmt.Errorf("invalid action type: %w", err)
	}

	// Create the action
	actionID := generateActionID()

	var parsedAction *action.Action
	if parseResp.Target != "" {
		parsedAction = action.NewActionWithTarget(
			actionID,
			req.Character.ID,
			actionType,
			parseResp.Skill,
			parseResp.Description,
			parseResp.Target,
		)
	} else {
		parsedAction = action.NewAction(
			actionID,
			req.Character.ID,
			actionType,
			parseResp.Skill,
			parseResp.Description,
		)
	}
	parsedAction.RawInput = req.RawInput

	// Set difficulty from LLM response (attacks will override with active defense later)
	parsedAction.Difficulty = dice.Ladder(parseResp.Difficulty)

	return parsedAction, nil
}

// buildSystemPrompt creates the system prompt using templates
func (ap *ActionParser) buildSystemPrompt() (string, error) {
	return prompt.RenderActionParseSystem()
}

// buildUserPrompt creates the user prompt using templates with pre-computed difficulty guidance
func (ap *ActionParser) buildUserPrompt(req ActionParseRequest) (string, error) {
	// Compute difficulty range based on character's highest skill
	highestSkill := dice.Mediocre
	for _, level := range req.Character.Skills {
		if level > highestSkill {
			highestSkill = level
		}
	}

	// Fate guidance: 2 below skill = easy, at skill = moderate, 2 above = hard
	// Typical range: Fair (+2) for routine to skill+2 for challenging
	minDiff := int(dice.Average)     // Floor at Average (+1) for meaningful rolls
	defaultDiff := int(dice.Fair)    // Fair (+2) is the standard default
	maxDiff := int(highestSkill) + 2 // Up to 2 above their best skill
	if maxDiff < int(dice.Good) {
		maxDiff = int(dice.Good) // At least Good (+3) as upper bound
	}
	if maxDiff > int(dice.Fantastic) {
		maxDiff = int(dice.Fantastic) // Cap at Fantastic (+6) for normal play
	}

	// Build human-readable guide
	guide := fmt.Sprintf("%d=easy, %d=moderate, %d=hard", minDiff, defaultDiff, maxDiff)

	templateData := prompt.ActionParseTemplateData{
		Character:         req.Character,
		RawInput:          req.RawInput,
		Context:           req.Context,
		Scene:             req.Scene,
		OtherCharacters:   req.OtherCharacters,
		DifficultyMin:     minDiff,
		DifficultyMax:     maxDiff,
		DifficultyDefault: defaultDiff,
		DifficultyGuide:   guide,
	}

	return prompt.RenderActionParse(templateData)
}

// parseActionType converts string action type to enum
// Also handles common LLM mistakes like returning skill names instead of action types
func parseActionType(actionTypeStr string) (action.ActionType, error) {
	// Normalize to lower case for more flexible matching
	normalized := strings.ToLower(strings.TrimSpace(actionTypeStr))

	switch normalized {
	case "overcome":
		return action.Overcome, nil
	case "create an advantage", "create advantage", "createadvantage":
		return action.CreateAdvantage, nil
	case "attack":
		return action.Attack, nil
	case "defend", "defense":
		return action.Defend, nil
	}

	// Check if LLM returned a skill name instead of action type
	// Map aggressive/confrontational skills to Attack
	attackSkills := map[string]bool{
		"fight": true, "shoot": true, "provoke": true,
	}
	if attackSkills[normalized] {
		return action.Attack, nil
	}

	// Map defensive skills to Defend
	defendSkills := map[string]bool{
		"athletics": true, "will": true, "physique": true,
	}
	if defendSkills[normalized] {
		return action.Defend, nil
	}

	// Default to Overcome for any other skill names
	// CreateAdvantage should only be used when explicitly requested
	return action.Overcome, nil
}

// generateActionID creates a unique action ID
func generateActionID() string {
	// Simple implementation using timestamp
	return fmt.Sprintf("action-%d", time.Now().UnixNano())
}
