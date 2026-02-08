package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
)

// ParseGeneratedScene parses an LLM response into a GeneratedScene.
func ParseGeneratedScene(content string) (*GeneratedScene, error) {
	cleaned := strings.TrimSpace(content)

	var generated GeneratedScene
	if err := json.Unmarshal([]byte(cleaned), &generated); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &generated); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	if generated.SceneName == "" {
		return nil, fmt.Errorf("missing scene_name")
	}
	if generated.Description == "" {
		return nil, fmt.Errorf("missing description")
	}
	if generated.Purpose == "" {
		return nil, fmt.Errorf("missing purpose")
	}

	return &generated, nil
}

// ParseSceneSummary parses an LLM response into a SceneSummary.
func ParseSceneSummary(content string) (*SceneSummary, error) {
	cleaned := strings.TrimSpace(content)

	var summary SceneSummary
	if err := json.Unmarshal([]byte(cleaned), &summary); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &summary); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	if summary.NarrativeProse == "" {
		return nil, fmt.Errorf("missing narrative_prose")
	}

	return &summary, nil
}

// ParseScenarioResolution parses an LLM response into a ScenarioResolutionResult.
func ParseScenarioResolution(content string) (*ScenarioResolutionResult, error) {
	cleaned := strings.TrimSpace(content)

	var result ScenarioResolutionResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	return &result, nil
}

// ParseScenario parses an LLM response into a Scenario.
func ParseScenario(content string) (*scene.Scenario, error) {
	cleaned := strings.TrimSpace(content)

	var scenario scene.Scenario
	if err := json.Unmarshal([]byte(cleaned), &scenario); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &scenario); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	if scenario.Title == "" {
		return nil, fmt.Errorf("missing title")
	}
	if scenario.Problem == "" {
		return nil, fmt.Errorf("missing problem")
	}

	return &scenario, nil
}

// ParseClassification extracts a single classification word from an LLM response.
// It handles common LLM formatting artifacts:
//   - Markdown formatting ("## narrative", "**action**")
//   - Trailing explanations ("dialog - the player is speaking")
//   - Quote/backtick wrapping ("`clarification`")
//
// Returns the lowercased first alphabetic word found in the response.
func ParseClassification(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))

	// Strip non-alphabetic characters from leading/trailing positions
	// to handle markdown artifacts like ## or **
	s = stripNonAlpha(s)

	// Extract first word (LLM sometimes appends reasoning)
	if idx := strings.IndexAny(s, " \n\t"); idx > 0 {
		s = s[:idx]
	}

	// Final strip in case inner content had trailing punctuation
	return stripNonAlpha(s)
}

// stripNonAlpha removes leading and trailing non-alphabetic characters from s.
func stripNonAlpha(s string) string {
	start := 0
	for start < len(s) && !isAlpha(s[start]) {
		start++
	}
	end := len(s)
	for end > start && !isAlpha(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
