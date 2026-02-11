package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
)

// AspectGenerationResponse contains the generated aspect and related information
type AspectGenerationResponse struct {
	AspectText  string `json:"aspect_text"`
	Description string `json:"description"`
	Duration    string `json:"duration"`     // "scene", "session", "permanent"
	FreeInvokes int    `json:"free_invokes"` // Number of free invokes granted
	IsBoost     bool   `json:"is_boost"`     // True if this is a boost (one free invoke)
	Reasoning   string `json:"reasoning"`    // Explanation of why this aspect was chosen
}

// AspectGeneratorI is the interface for generating aspects from Create an Advantage actions.
// This enables dependency injection and testability by allowing mock implementations.
type AspectGeneratorI interface {
	GenerateAspect(ctx context.Context, req prompt.AspectGenerationRequest) (*AspectGenerationResponse, error)
}

// Compile-time check that AspectGenerator satisfies AspectGeneratorI.
var _ AspectGeneratorI = (*AspectGenerator)(nil)

// AspectGenerator handles generating aspects based on Create an Advantage attempts
type AspectGenerator struct {
	llmClient llm.LLMClient
}

// NewAspectGenerator creates a new aspect generator
func NewAspectGenerator(llmClient llm.LLMClient) *AspectGenerator {
	return &AspectGenerator{
		llmClient: llmClient,
	}
}

// GenerateAspect generates an aspect based on the Create an Advantage attempt and outcome
func (ag *AspectGenerator) GenerateAspect(ctx context.Context, req prompt.AspectGenerationRequest) (*AspectGenerationResponse, error) {
	if req.Action.Type != action.CreateAdvantage {
		return nil, fmt.Errorf("action type must be CreateAdvantage, got %s", req.Action.Type.String())
	}

	if req.Outcome == nil {
		return nil, fmt.Errorf("outcome cannot be nil")
	}

	prompt := ag.buildPrompt(req)

	llmReq := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: ag.getSystemPrompt()},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1024,
		Temperature: 0.8,
		TopP:        0.9,
	}

	response, err := ag.llmClient.ChatCompletion(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate aspect: %w", err)
	}

	// Parse the LLM response into an AspectGenerationResponse
	return ag.parseResponse(response.Content(), req.Outcome)
}

// getSystemPrompt returns the system prompt for aspect generation
func (ag *AspectGenerator) getSystemPrompt() string {
	promptText, err := prompt.RenderAspectGenerationSystem()
	if err != nil {
		// Fallback to hardcoded prompt if template fails
		return `You are an expert Game Master for the Fate Core RPG system. Your job is to generate appropriate aspects based on a character's "Create an Advantage" action and the outcome of their dice roll.

FATE CORE CREATE AN ADVANTAGE RULES:
- Success creates a new aspect with one free invoke
- Success with Style creates a new aspect with two free invokes OR allows you to discover an existing unknown aspect
- Tie creates a boost (temporary aspect with one free invoke)
- Failure means no aspect is created, but the GM may offer a success at a cost

ASPECT GUIDELINES:
- Aspects should be 2-4 words, descriptive, and invoking-worthy
- They should relate directly to what the character was trying to accomplish
- Consider the skill used and the character's approach
- Aspects can be situational advantages, character discoveries, environmental features, or opponent weaknesses
- Avoid making aspects too powerful or too specific

Your response must be in JSON format with these fields:
{
  "aspect_text": "The actual aspect text",
  "description": "A brief description of what this aspect represents",
  "duration": "scene|session|permanent",
  "free_invokes": 0-2,
  "is_boost": true/false,
  "reasoning": "Brief explanation of why this aspect fits the situation"
}`
	}
	return promptText
}

// buildPrompt constructs the detailed prompt for aspect generation
func (ag *AspectGenerator) buildPrompt(req prompt.AspectGenerationRequest) string {
	promptText, err := prompt.RenderAspectGeneration(req)
	if err != nil {
		// Fallback to the old string building method if template fails
		return ag.buildPromptFallback(req)
	}
	return promptText
}

// buildPromptFallback is the original string building method as a fallback
func (ag *AspectGenerator) buildPromptFallback(req prompt.AspectGenerationRequest) string {
	var prompt strings.Builder

	prompt.WriteString("Generate an aspect for a Create an Advantage action with the following details:\n\n")

	// Character information
	prompt.WriteString(fmt.Sprintf("CHARACTER: %s\n", req.Character.Name))
	if req.Character.Aspects.HighConcept != "" {
		prompt.WriteString(fmt.Sprintf("High Concept: %s\n", req.Character.Aspects.HighConcept))
	}
	if req.Character.Aspects.Trouble != "" {
		prompt.WriteString(fmt.Sprintf("Trouble: %s\n", req.Character.Aspects.Trouble))
	}
	characterAspects := req.Character.Aspects.GetAll()
	if len(characterAspects) > 2 {
		prompt.WriteString("Other Aspects: ")
		for i, aspect := range characterAspects[2:] {
			if i > 0 {
				prompt.WriteString(", ")
			}
			prompt.WriteString(aspect)
		}
		prompt.WriteString("\n")
	}

	// Action details
	prompt.WriteString("\nACTION ATTEMPT:\n")
	prompt.WriteString(fmt.Sprintf("Skill Used: %s (%s)\n", req.Action.Skill, req.Character.GetSkill(req.Action.Skill).String()))
	prompt.WriteString(fmt.Sprintf("Description: %s\n", req.Action.Description))
	if req.Action.RawInput != "" {
		prompt.WriteString(fmt.Sprintf("Player's Intent: %s\n", req.Action.RawInput))
	}

	// Roll outcome
	prompt.WriteString("\nROLL OUTCOME:\n")
	prompt.WriteString(fmt.Sprintf("Result: %s (%+d shifts)\n", req.Outcome.Type.String(), req.Outcome.Shifts))
	if req.Outcome.Result != nil && req.Outcome.Result.Roll != nil {
		prompt.WriteString(fmt.Sprintf("Dice Roll: %s (Total: %+d)\n", req.Outcome.Result.Roll.String(), req.Outcome.Result.Roll.Total))
		prompt.WriteString(fmt.Sprintf("Final Value: %s vs Difficulty %s\n",
			req.Outcome.Result.FinalValue.String(), req.Outcome.Difficulty.String()))
	} else {
		prompt.WriteString(fmt.Sprintf("Difficulty: %s\n", req.Outcome.Difficulty.String()))
	}

	// Context and existing aspects
	if req.Context != "" {
		prompt.WriteString(fmt.Sprintf("\nSCENE CONTEXT:\n%s\n", req.Context))
	}

	if len(req.ExistingAspects) > 0 {
		prompt.WriteString("\nEXISTING ASPECTS IN PLAY:\n")
		for _, aspect := range req.ExistingAspects {
			prompt.WriteString(fmt.Sprintf("- %s\n", aspect))
		}
	}

	prompt.WriteString(fmt.Sprintf("\nTARGET TYPE: %s\n", req.TargetType))

	prompt.WriteString("\nGenerate an appropriate aspect based on this Create an Advantage attempt:")

	return prompt.String()
}

// parseResponse parses the LLM response and extracts aspect information
func (ag *AspectGenerator) parseResponse(content string, outcome *dice.Outcome) (*AspectGenerationResponse, error) {
	// This is a simplified parser. In a production system, you'd want more robust JSON parsing
	// For now, we'll extract key information and provide defaults based on the outcome

	response := &AspectGenerationResponse{
		Duration: "scene", // Default duration
	}

	// Set defaults based on outcome type
	switch outcome.Type {
	case dice.SuccessWithStyle:
		response.FreeInvokes = 2
		response.IsBoost = false
	case dice.Success:
		response.FreeInvokes = 1
		response.IsBoost = false
	case dice.Tie:
		response.FreeInvokes = 1
		response.IsBoost = true
		response.Duration = "scene" // Boosts last until used or end of scene
	case dice.Failure:
		response.FreeInvokes = 0
		response.IsBoost = false
		response.AspectText = ""
		response.Description = "No aspect created due to failure"
		response.Reasoning = "The Create an Advantage attempt failed"
		return response, nil
	}

	// Try to extract aspect text from the response
	// Look for JSON-like content or extract meaningful phrases
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"aspect_text"`) {
			// Try to extract JSON value
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				aspectText := strings.Trim(parts[1], ` "`)
				aspectText = strings.TrimSuffix(aspectText, ",")
				aspectText = strings.Trim(aspectText, `"`)
				if aspectText != "" {
					response.AspectText = aspectText
				}
			}
		} else if strings.Contains(line, `"description"`) {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				description := strings.Trim(parts[1], ` "`)
				description = strings.TrimSuffix(description, ",")
				description = strings.Trim(description, `"`)
				if description != "" {
					response.Description = description
				}
			}
		} else if strings.Contains(line, `"reasoning"`) {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				reasoning := strings.Trim(parts[1], ` "`)
				reasoning = strings.TrimSuffix(reasoning, ",")
				reasoning = strings.Trim(reasoning, `"`)
				if reasoning != "" {
					response.Reasoning = reasoning
				}
			}
		}
	}

	// If we couldn't extract from JSON, try to find the aspect in the text
	if response.AspectText == "" {
		// Look for quoted strings that might be aspects
		if start := strings.Index(content, `"`); start != -1 {
			if end := strings.Index(content[start+1:], `"`); end != -1 {
				potential := content[start+1 : start+1+end]
				if len(strings.Fields(potential)) <= 4 && len(potential) > 3 {
					response.AspectText = potential
				}
			}
		}
	}

	// Fallback: provide a generic aspect based on the outcome
	if response.AspectText == "" {
		switch outcome.Type {
		case dice.SuccessWithStyle:
			response.AspectText = "Perfect Advantage"
		case dice.Success:
			response.AspectText = "Temporary Advantage"
		case dice.Tie:
			response.AspectText = "Fleeting Opportunity"
		}
		response.Description = "Generated aspect based on roll outcome"
		response.Reasoning = "Fallback aspect when LLM response couldn't be parsed"
	}

	return response, nil
}
