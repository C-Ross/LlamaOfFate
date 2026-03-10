package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	gameconfig "github.com/C-Ross/LlamaOfFate/internal/config"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/openai"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"gopkg.in/yaml.v3"
)

func main() {
	logging.SetupDefaultLogging()

	// Required flags
	hintFlag := flag.String("hint", "", "Transition hint (where the player is heading) (required)")

	// Scenario context flags
	scenarioFlag := flag.String("scenario", "", "Predefined scenario: saloon, heist, tower")
	genreFlag := flag.String("genre", "", "Genre hint (e.g., Western, Cyberpunk, Fantasy)")

	// Player character flags (optional, uses defaults if not set)
	nameFlag := flag.String("name", "Jesse Calhoun", "Player character name")
	conceptFlag := flag.String("concept", "Haunted Former Rancher Seeking Justice", "Player high concept")
	troubleFlag := flag.String("trouble", "Vengeance Burns Hotter Than Reason", "Player trouble aspect")

	// Summary context
	summariesFlag := flag.String("summaries", "", "Path to YAML file with previous scene summaries")

	// Output flags
	logFlag := flag.String("log", "auto", "Session log path (auto=generated filename, empty=disabled)")

	// Debug flags
	debugFlag := flag.Bool("debug", false, "Show rendered prompt")
	rawFlag := flag.Bool("raw", false, "Show raw LLM response")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: scene-generator [options]\n\n")
		fmt.Fprintf(os.Stderr, "Generate a Fate Core scene from transition context.\n\n")
		fmt.Fprintf(os.Stderr, "Required:\n")
		fmt.Fprintf(os.Stderr, "  -hint string       Where the player is heading (transition hint)\n\n")
		fmt.Fprintf(os.Stderr, "Optional:\n")
		fmt.Fprintf(os.Stderr, "  -scenario string   Predefined scenario (saloon, heist, tower)\n")
		fmt.Fprintf(os.Stderr, "  -genre string      Genre hint (Western, Cyberpunk, Fantasy, etc.)\n")
		fmt.Fprintf(os.Stderr, "  -name string       Player character name (default: Jesse Calhoun)\n")
		fmt.Fprintf(os.Stderr, "  -concept string    Player high concept\n")
		fmt.Fprintf(os.Stderr, "  -trouble string    Player trouble aspect\n")
		fmt.Fprintf(os.Stderr, "  -summaries string  Path to YAML file with previous scene summaries\n\n")
		fmt.Fprintf(os.Stderr, "Output:\n")
		fmt.Fprintf(os.Stderr, "  -log string        Session log path (default: auto-generated, empty disables)\n\n")
		fmt.Fprintf(os.Stderr, "Debug:\n")
		fmt.Fprintf(os.Stderr, "  -debug             Show rendered prompt\n")
		fmt.Fprintf(os.Stderr, "  -raw               Show raw LLM response\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  scene-generator -hint \"the dusty main street\" -genre Western\n")
		fmt.Fprintf(os.Stderr, "  scene-generator -scenario saloon -hint \"the sheriff's office\"\n")
		fmt.Fprintf(os.Stderr, "  scene-generator -summaries sample_summaries.yaml -hint \"the canyon hideout\"\n")
		fmt.Fprintf(os.Stderr, "  scene-generator -hint \"the rooftop\" -genre Cyberpunk --debug\n")
	}

	flag.Parse()

	// Validate required flags
	if *hintFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: -hint is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load LLM config
	configPath := "configs/azure-llm.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("LLM config not found at %s\n", configPath)
		fmt.Println("Please copy configs/azure-llm.yaml.example to configs/azure-llm.yaml")
		fmt.Println("and configure your LLM credentials.")
		os.Exit(1)
	}

	config, err := openai.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load LLM config: %v", err)
	}

	llmClient := openai.NewClient(*config)

	// Set up session logging
	var sessionLogger *session.Logger
	logPath := *logFlag
	if logPath == "auto" {
		var err error
		logPath, err = session.GenerateLogPath("scene_gen", []string{*hintFlag, *genreFlag}, 30)
		if err != nil {
			log.Fatalf("Failed to generate log path: %v", err)
		}
	}
	if logPath != "" {
		sessionLogger, err = session.NewLogger(logPath)
		if err != nil {
			log.Fatalf("Failed to create session logger: %v", err)
		}
		defer func() {
			if err := sessionLogger.Close(); err != nil {
				log.Printf("Warning: Failed to close session logger: %v", err)
			}
		}()
		fmt.Printf("Session log: %s\n\n", logPath)
	}

	// Build scenario context
	var scenario *scene.Scenario
	if *scenarioFlag != "" {
		scenarios, loadErr := gameconfig.LoadAll("configs")
		if loadErr != nil {
			log.Fatalf("Failed to load configs: %v", loadErr)
		}
		ls, ok := scenarios[*scenarioFlag]
		if !ok {
			fmt.Fprintf(os.Stderr, "Unknown scenario: %s (use saloon, heist, or tower)\n", *scenarioFlag)
			os.Exit(1)
		}
		scenario = ls.Scenario
	} else if *genreFlag != "" {
		scenario = &scene.Scenario{
			Genre: *genreFlag,
		}
	}

	// Load previous summaries if provided
	var summaries []prompt.SceneSummary
	if *summariesFlag != "" {
		summaries, err = loadSummaries(*summariesFlag)
		if err != nil {
			log.Fatalf("Failed to load summaries from %s: %v", *summariesFlag, err)
		}
		fmt.Printf("Loaded %d previous scene summaries\n\n", len(summaries))
	}

	// Build scene generation data
	data := prompt.SceneGenerationData{
		TransitionHint:    *hintFlag,
		Scenario:          scenario,
		PlayerName:        *nameFlag,
		PlayerHighConcept: *conceptFlag,
		PlayerTrouble:     *troubleFlag,
		PreviousSummaries: summaries,
	}

	// Render the prompt
	promptText, err := prompt.RenderSceneGeneration(data)
	if err != nil {
		log.Fatalf("Failed to render prompt: %v", err)
	}

	// Log the input data and prompt
	if sessionLogger != nil {
		sessionLogger.Log("scene_generation_input", map[string]any{
			"transition_hint": data.TransitionHint,
			"scenario":        data.Scenario,
			"player_name":     data.PlayerName,
			"high_concept":    data.PlayerHighConcept,
			"trouble":         data.PlayerTrouble,
			"summaries_count": len(summaries),
			"rendered_prompt": promptText,
		})
	}

	if *debugFlag {
		fmt.Println("=== Rendered Prompt ===")
		fmt.Println(promptText)
		fmt.Println()
	}

	// Call LLM
	ctx := context.Background()
	rawResponse, err := llm.SimpleCompletion(ctx, llmClient, promptText, 500, 0.8)
	if err != nil {
		log.Fatalf("LLM request failed: %v", err)
	}

	if *rawFlag {
		fmt.Println("=== Raw LLM Response ===")
		fmt.Println(rawResponse)
		fmt.Println()
	}

	// Parse the response
	generated, err := prompt.ParseGeneratedScene(rawResponse)
	if err != nil {
		fmt.Println("=== Parse Error ===")
		fmt.Printf("Error: %v\n\n", err)
		fmt.Println("Raw response:")
		fmt.Println(rawResponse)
		os.Exit(1)
	}

	// Log the result
	if sessionLogger != nil {
		sessionLogger.Log("scene_generation_output", map[string]any{
			"raw_response":    rawResponse,
			"generated_scene": generated,
		})
	}

	// Display formatted output
	displayGeneratedScene(generated)
}

// loadSummaries loads scene summaries from a YAML file
func loadSummaries(path string) ([]prompt.SceneSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var summaries []prompt.SceneSummary
	if err := yaml.Unmarshal(data, &summaries); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return summaries, nil
}

// displayGeneratedScene prints the scene in a readable format
func displayGeneratedScene(s *prompt.GeneratedScene) {
	fmt.Println("=== Generated Scene ===")
	fmt.Printf("Name: %s\n", s.SceneName)
	fmt.Println()

	if s.Purpose != "" {
		fmt.Println("Purpose:")
		printWrapped(s.Purpose, 75, "  ")
		fmt.Println()
	}

	fmt.Println("Description:")
	printWrapped(s.Description, 75, "  ")
	fmt.Println()

	if s.OpeningHook != "" {
		fmt.Println("Opening Hook:")
		printWrapped(s.OpeningHook, 75, "  ")
		fmt.Println()
	}

	if len(s.SituationAspects) > 0 {
		fmt.Println("Situation Aspects:")
		for _, a := range s.SituationAspects {
			fmt.Printf("  - \"%s\"\n", a)
		}
		fmt.Println()
	}

	if len(s.NPCs) > 0 {
		fmt.Println("NPCs:")
		for _, npc := range s.NPCs {
			fmt.Printf("  - %s (%s) [%s]\n", npc.Name, npc.HighConcept, npc.Disposition)
		}
		fmt.Println()
	} else {
		fmt.Println("NPCs: (none)")
		fmt.Println()
	}
}

func printWrapped(text string, width int, indent string) {
	words := strings.Fields(text)
	line := indent
	for _, word := range words {
		if len(line)+len(word)+1 > width {
			fmt.Println(line)
			line = indent + word
		} else {
			if line == indent {
				line += word
			} else {
				line += " " + word
			}
		}
	}
	if line != indent {
		fmt.Println(line)
	}
}
