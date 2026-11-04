package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/openai"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

// arrayFlags allows multiple -aspect flags
type arrayFlags []string

func (a *arrayFlags) String() string {
	return strings.Join(*a, ", ")
}

func (a *arrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func main() {
	logging.SetupDefaultLogging()

	// Required flags
	nameFlag := flag.String("name", "", "Character name (required)")
	conceptFlag := flag.String("concept", "", "High concept aspect (required)")

	// Optional character flags
	troubleFlag := flag.String("trouble", "", "Trouble aspect")
	var aspects arrayFlags
	flag.Var(&aspects, "aspect", "Additional aspect (can be repeated)")

	// Generation hints
	genreFlag := flag.String("genre", "", "Genre hint (e.g., Western, Cyberpunk, Fantasy)")
	themeFlag := flag.String("theme", "", "Theme hint for scenario")

	// Output flags
	logFlag := flag.String("log", "auto", "Session log path (auto=generated filename, empty=disabled)")

	// Debug flags
	debugFlag := flag.Bool("debug", false, "Show rendered prompt")
	rawFlag := flag.Bool("raw", false, "Show raw LLM response")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: scenario-generator [options]\n\n")
		fmt.Fprintf(os.Stderr, "Generate a Fate Core scenario from character aspects.\n\n")
		fmt.Fprintf(os.Stderr, "Required:\n")
		fmt.Fprintf(os.Stderr, "  -name string      Character name\n")
		fmt.Fprintf(os.Stderr, "  -concept string   High concept aspect\n\n")
		fmt.Fprintf(os.Stderr, "Optional:\n")
		fmt.Fprintf(os.Stderr, "  -trouble string   Trouble aspect\n")
		fmt.Fprintf(os.Stderr, "  -aspect string    Additional aspect (repeatable)\n")
		fmt.Fprintf(os.Stderr, "  -genre string     Genre hint (Western, Cyberpunk, Fantasy, etc.)\n")
		fmt.Fprintf(os.Stderr, "  -theme string     Theme hint for scenario\n\n")
		fmt.Fprintf(os.Stderr, "Output:\n")
		fmt.Fprintf(os.Stderr, "  -log string       Session log path (default: auto-generated, empty disables)\n\n")
		fmt.Fprintf(os.Stderr, "Debug:\n")
		fmt.Fprintf(os.Stderr, "  -debug            Show rendered prompt\n")
		fmt.Fprintf(os.Stderr, "  -raw              Show raw LLM response\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  scenario-generator -name \"Jesse Calhoun\" \\\n")
		fmt.Fprintf(os.Stderr, "    -concept \"Haunted Former Rancher Seeking Justice\" \\\n")
		fmt.Fprintf(os.Stderr, "    -trouble \"Vengeance Burns Hotter Than Reason\" \\\n")
		fmt.Fprintf(os.Stderr, "    -genre Western\n")
	}

	flag.Parse()

	// Validate required flags
	if *nameFlag == "" || *conceptFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: -name and -concept are required\n\n")
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
		logPath, err = session.GenerateLogPath("scenario_gen", []string{*nameFlag, *genreFlag}, 20)
		if err != nil {
			log.Fatalf("Failed to generate log path: %v", err)
		}
	}
	if logPath != "" {
		var err error
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

	// Build scenario generation data
	data := prompt.ScenarioGenerationData{
		PlayerName:        *nameFlag,
		PlayerHighConcept: *conceptFlag,
		PlayerTrouble:     *troubleFlag,
		PlayerAspects:     aspects,
		Genre:             *genreFlag,
		Theme:             *themeFlag,
	}

	// Render the prompt
	promptText, err := prompt.RenderScenarioGeneration(data)
	if err != nil {
		log.Fatalf("Failed to render prompt: %v", err)
	}

	// Log the input data and prompt
	if sessionLogger != nil {
		sessionLogger.Log("scenario_generation_input", map[string]any{
			"player_name":     data.PlayerName,
			"high_concept":    data.PlayerHighConcept,
			"trouble":         data.PlayerTrouble,
			"aspects":         data.PlayerAspects,
			"genre":           data.Genre,
			"theme":           data.Theme,
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
	scenario, err := prompt.ParseScenario(rawResponse)
	if err != nil {
		fmt.Println("=== Parse Error ===")
		fmt.Printf("Error: %v\n\n", err)
		fmt.Println("Raw response:")
		fmt.Println(rawResponse)
		os.Exit(1)
	}

	// Log the result
	if sessionLogger != nil {
		sessionLogger.Log("scenario_generation_output", map[string]any{
			"raw_response": rawResponse,
			"scenario":     scenario,
		})
	}

	// Display formatted output
	displayScenario(scenario)
}

// displayScenario prints the scenario in a readable format
func displayScenario(s *scene.Scenario) {
	fmt.Println("=== Generated Scenario ===")
	fmt.Printf("Title: %s\n", s.Title)
	if s.Genre != "" {
		fmt.Printf("Genre: %s\n", s.Genre)
	}
	fmt.Println()

	fmt.Println("Problem:")
	fmt.Printf("  %s\n", s.Problem)
	fmt.Println()

	if len(s.StoryQuestions) > 0 {
		fmt.Println("Story Questions:")
		for i, q := range s.StoryQuestions {
			fmt.Printf("  %d. %s\n", i+1, q)
		}
		fmt.Println()
	}

	if s.Setting != "" {
		fmt.Println("Setting:")
		// Word wrap the setting at ~70 chars
		words := strings.Fields(s.Setting)
		line := "  "
		for _, word := range words {
			if len(line)+len(word)+1 > 75 {
				fmt.Println(line)
				line = "  " + word
			} else {
				if line == "  " {
					line += word
				} else {
					line += " " + word
				}
			}
		}
		if line != "  " {
			fmt.Println(line)
		}
	}
}
