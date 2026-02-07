package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
)

// ParseGeneratedScene parses an LLM response into a GeneratedScene.
func ParseGeneratedScene(content string) (*GeneratedScene, error) {
	cleaned := llm.CleanJSONResponse(content)

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
	cleaned := llm.CleanJSONResponse(content)

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
	cleaned := llm.CleanJSONResponse(content)

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
func ParseScenario(content string) (*Scenario, error) {
	cleaned := llm.CleanJSONResponse(content)

	var scenario Scenario
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

// PredefinedScenario returns a predefined scenario by name.
// Supported names: "saloon", "heist", "tower" (case-insensitive).
// Returns nil if the name is not recognized.
func PredefinedScenario(name string) *Scenario {
	switch strings.ToLower(name) {
	case "tower":
		return &Scenario{
			Title:   "The Wizard's Tower",
			Problem: "A mysterious magical disturbance threatens the tower and its inhabitants",
			StoryQuestions: []string{
				"Can the source of the disturbance be discovered?",
				"Will the tower's secrets be revealed?",
			},
			Genre:   "Fantasy",
			Setting: "A medieval fantasy world of magic and mystery. Wizards study arcane arts in towers, adventurers seek treasure in ancient ruins, and supernatural forces are very real.",
		}
	case "heist":
		return &Scenario{
			Title:   "The Prometheus Job",
			Problem: "A high-value data core must be extracted from a heavily guarded corporate facility",
			StoryQuestions: []string{
				"Can the team breach the facility's security?",
				"Will the extraction succeed without casualties?",
				"What secrets does the data core contain?",
			},
			Genre:   "Cyberpunk",
			Setting: "A dark near-future where megacorporations rule, hackers breach digital fortresses, and chrome-enhanced mercenaries sell their skills to the highest bidder. Neon lights flicker over rain-slicked streets.",
		}
	case "saloon":
		return &Scenario{
			Title:   "Trouble in Redemption Gulch",
			Problem: "The town is under threat from outlaws and someone needs to stand up for the innocent",
			StoryQuestions: []string{
				"Will the outlaws be brought to justice?",
				"Can the town be saved?",
			},
			Genre:   "Western",
			Setting: "The American Old West in the late 1800s. Dusty frontier towns, lawless territories, and the struggle between civilization and the wild. Gunslingers, outlaws, and honest folk all seeking their fortune.",
		}
	default:
		return nil
	}
}
