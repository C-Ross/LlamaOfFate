//go:build llmeval

package llmeval_test

import (
	"os"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/openai"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Client setup
// ---------------------------------------------------------------------------

// RequireLLMClient returns a ready-to-use LLM client, using either Ollama or
// an OpenAI-compatible endpoint depending on configuration. Set LLM_PROVIDER=ollama
// to use a local Ollama instance; otherwise API credentials are required.
func RequireLLMClient(tb testing.TB) llm.LLMClient {
	tb.Helper()

	provider := os.Getenv("LLM_PROVIDER")

	switch strings.ToLower(provider) {
	case "ollama":
		config, err := openai.LoadConfig("../../configs/ollama-llm.yaml")
		require.NoError(tb, err, "Failed to load Ollama config")
		return openai.NewClient(*config)
	default:
		if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
			tb.Skip("Skipping: set LLM_PROVIDER=ollama or AZURE_API_ENDPOINT and AZURE_API_KEY")
		}
		config, err := openai.LoadConfig("../../configs/azure-llm.yaml")
		require.NoError(tb, err, "Failed to load LLM config")
		return openai.NewClient(*config)
	}
}

// VerboseLoggingEnabled returns true when VERBOSE=1 is set.
// Use in place of per-file verbose variables for consistent behavior.
func VerboseLoggingEnabled() bool {
	return os.Getenv("VERBOSE") == "1"
}

// ---------------------------------------------------------------------------
// Context builders — used by scene response / transition / proactive-GM evals
// ---------------------------------------------------------------------------

// BuildCharacterContext creates a character context string for prompt templates.
func BuildCharacterContext(player *core.Character) string {
	var sb strings.Builder
	sb.WriteString("Name: ")
	sb.WriteString(player.Name)
	sb.WriteString("\n")
	if player.Aspects.HighConcept != "" {
		sb.WriteString("High Concept: ")
		sb.WriteString(player.Aspects.HighConcept)
		sb.WriteString("\n")
	}
	if player.Aspects.Trouble != "" {
		sb.WriteString("Trouble: ")
		sb.WriteString(player.Aspects.Trouble)
		sb.WriteString("\n")
	}
	return sb.String()
}

// BuildAspectsContext creates an aspects context string for prompt templates.
func BuildAspectsContext(s *scene.Scene, player *core.Character, others []*core.Character) string {
	var sb strings.Builder
	sb.WriteString("Scene Aspects:\n")
	for _, aspect := range s.SituationAspects {
		sb.WriteString("  - ")
		sb.WriteString(aspect.Aspect)
		sb.WriteString("\n")
	}
	sb.WriteString("\nCharacter Aspects:\n")
	if player.Aspects.HighConcept != "" {
		sb.WriteString("  - ")
		sb.WriteString(player.Aspects.HighConcept)
		sb.WriteString(" (")
		sb.WriteString(player.Name)
		sb.WriteString(")\n")
	}
	for _, other := range others {
		if other.Aspects.HighConcept != "" {
			sb.WriteString("  - ")
			sb.WriteString(other.Aspects.HighConcept)
			sb.WriteString(" (")
			sb.WriteString(other.Name)
			sb.WriteString(")\n")
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// Display helpers
// ---------------------------------------------------------------------------

// TruncateResponse truncates a response string for display, collapsing whitespace.
func TruncateResponse(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// ---------------------------------------------------------------------------
// NPC factories — reusable across scene response, transition, marker-leak, etc.
// ---------------------------------------------------------------------------

// NewBlackJack creates the Black Jack McCoy NPC used across multiple test files.
func NewBlackJack() *core.Character {
	blackJack := core.NewCharacter("blackjack", "Black Jack McCoy")
	blackJack.Aspects.HighConcept = "Dangerous Outlaw with a Quick Draw"
	blackJack.Aspects.Trouble = "Wanted Dead or Alive"
	return blackJack
}

// NewBartender creates the Maggie Two-Rivers bartender NPC used across multiple test files.
func NewBartender() *core.Character {
	bartender := core.NewCharacter("bartender", "Maggie Two-Rivers")
	bartender.Aspects.HighConcept = "Weathered Saloon Owner"
	return bartender
}

// NewHeistNPCs creates NPCs matching the heist scenario (Agent Chen + Security Drone).
func NewHeistNPCs() []*core.Character {
	chen := core.NewCharacter("corp-agent", "Agent Chen")
	chen.Aspects.HighConcept = "Nexus Industries Troubleshooter"
	chen.Aspects.AddAspect("Augmented Combat Implants")
	chen.SetSkill("Fight", dice.Good)
	chen.SetSkill("Shoot", dice.Good)
	chen.SetSkill("Notice", dice.Fair)
	chen.SetSkill("Athletics", dice.Fair)
	chen.SetSkill("Will", dice.Average)
	chen.SetSkill("Physique", dice.Average)

	drone := core.NewCharacter("drone-1", "Security Drone Alpha")
	drone.Aspects.HighConcept = "Automated Threat Response Unit"
	drone.SetSkill("Shoot", dice.Fair)
	drone.SetSkill("Notice", dice.Average)

	return []*core.Character{chen, drone}
}

// NewHeistPlayer creates the Zero / Ghost character from the heist preset.
func NewHeistPlayer() *core.Character {
	char := core.NewCharacter("zero", "Ghost")
	char.Aspects.HighConcept = "Ex-Corporate Netrunner Gone Rogue"
	char.Aspects.Trouble = "Every Megacorp Wants Me Dead"
	char.Aspects.AddAspect("Military-Grade Cybernetic Reflexes")
	char.Aspects.AddAspect("Nobody Gets Left Behind")
	char.Aspects.AddAspect("I Know a Guy for Everything")

	char.SetSkill("Burglary", dice.Superb)
	char.SetSkill("Stealth", dice.Great)
	char.SetSkill("Notice", dice.Great)
	char.SetSkill("Crafts", dice.Good)
	char.SetSkill("Athletics", dice.Good)
	char.SetSkill("Shoot", dice.Good)
	char.SetSkill("Deceive", dice.Fair)
	char.SetSkill("Will", dice.Fair)
	char.SetSkill("Investigate", dice.Fair)
	char.SetSkill("Contacts", dice.Fair)
	char.SetSkill("Fight", dice.Average)
	char.SetSkill("Physique", dice.Average)
	char.SetSkill("Provoke", dice.Average)
	char.SetSkill("Resources", dice.Average)

	return char
}

// ---------------------------------------------------------------------------
// Test character factories
// ---------------------------------------------------------------------------

// NewEvalCharacter creates a well-rounded test character with a range of skills.
func NewEvalCharacter() *core.Character {
	char := core.NewCharacter("eval-char", "Magnus the Versatile")
	char.Aspects.HighConcept = "Resourceful Problem Solver"
	char.Aspects.Trouble = "Curiosity Killed the Cat"
	char.Aspects.AddAspect("Former Street Urchin")
	char.Aspects.AddAspect("Quick on My Feet")

	char.SetSkill("Athletics", dice.Good)
	char.SetSkill("Fight", dice.Fair)
	char.SetSkill("Shoot", dice.Average)
	char.SetSkill("Stealth", dice.Good)
	char.SetSkill("Notice", dice.Fair)
	char.SetSkill("Investigate", dice.Good)
	char.SetSkill("Deceive", dice.Fair)
	char.SetSkill("Rapport", dice.Average)
	char.SetSkill("Will", dice.Fair)
	char.SetSkill("Provoke", dice.Average)
	char.SetSkill("Burglary", dice.Good)
	char.SetSkill("Lore", dice.Fair)

	return char
}
