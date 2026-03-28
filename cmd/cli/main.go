package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/openai"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/C-Ross/LlamaOfFate/internal/storage"
	"github.com/C-Ross/LlamaOfFate/internal/syncdriver"
	"github.com/C-Ross/LlamaOfFate/internal/ui/terminal"
)

const (
	AppName    = "LlamaOfFate"
	AppVersion = "0.1.0"
)

// initializeEngine creates a game engine with LLM support
func initializeEngine(llmClient llm.LLMClient, sessionLogger session.SessionLogger) *engine.Engine {
	gameEngine, err := engine.NewWithLLM(llmClient, sessionLogger)
	if err != nil {
		log.Fatalf("Failed to create engine with LLM: %v", err)
	}
	return gameEngine
}

func initLLMClient(configPath string) (llm.LLMClient, string, error) {
	config, err := openai.LoadConfig(configPath)
	if err != nil {
		return nil, "", err
	}
	llmClient := openai.NewClient(*config)
	retryClient := llm.NewRetryingClient(llmClient, llm.DefaultRetryConfig())
	return retryClient, config.ModelName, nil
}

func main() {
	logging.SetupDefaultLogging()

	// Handle subcommands before engine init
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			showVersion()
			return
		case "help":
			showHelp()
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
			showHelp()
			os.Exit(1)
		}
	}

	fmt.Printf("%s v%s - A Fate Core RPG with LLM integration\n", AppName, AppVersion)
	fmt.Println("====================================================")

	configPath := "configs/azure-llm.yaml"
	if c := os.Getenv("LLM_CONFIG"); c != "" {
		configPath = c
	}

	// Fail fast on LLM configuration before character creation prompts.
	llmClient, modelName, err := initLLMClient(configPath)
	if err != nil {
		log.Fatalf("LLM init failed: %v", err)
	}
	slog.Info("LLM integration enabled", slog.String("model", modelName))

	// Use hardcoded scenario and player (see scenario.go)
	scenario := defaultScenario()
	preset := defaultPlayer()

	fmt.Printf("Scenario: %s (%s)\n", scenario.Title, scenario.Genre)
	fmt.Println()

	// Character creation — prompt for customisation using preset as defaults
	terminalUI := terminal.NewTerminalUI()
	setup := terminalUI.PromptForCharacterSetup(preset)
	player := applySetup(preset, setup)

	fmt.Printf("\nPlaying as: %s — \"%s\"\n", player.Name, player.Aspects.HighConcept)
	fmt.Println()

	// Set up session logging
	sessionLogger, err := setupSessionLogger(scenario, player.Name)
	if err != nil {
		log.Fatalf("Failed to create session logger: %v", err)
	}
	var sl session.SessionLogger
	if sessionLogger != nil {
		sl = sessionLogger
		defer func() {
			if closeErr := sessionLogger.Close(); closeErr != nil {
				log.Printf("Warning: Failed to close session logger: %v", closeErr)
			}
		}()
		sessionLogger.Log("scenario", scenario)
		sessionLogger.Log("player", map[string]any{
			"name":         player.Name,
			"high_concept": player.Aspects.HighConcept,
			"trouble":      player.Aspects.Trouble,
			"aspects":      player.Aspects.OtherAspects,
		})
	} else {
		sl = session.NullLogger{}
	}

	gameEngine := initializeEngine(llmClient, sl)

	// Wire everything into the GameManager and run
	gm := engine.NewGameManager(gameEngine, sl)
	gm.SetPlayer(player)
	gm.SetScenario(scenario)
	gm.SetSaver(newSaver())

	fmt.Println("Type naturally to interact, 'quit' to exit.")
	fmt.Println("====================================================")

	ctx := context.Background()
	onStart := func() {
		terminalUI.SetSceneInfo(gm.GetEngine().GetSceneManager())
	}
	if err := syncdriver.Run(ctx, gm, terminalUI, onStart); err != nil {
		log.Fatalf("Game error: %v", err)
	}

	fmt.Println("\nThanks for playing!")
}

// newSaver creates a YAMLSaver that stores saves in ~/.llamaoffate/saves.
func newSaver() *storage.YAMLSaver {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not determine home directory for saves: %v", err)
		home = "."
	}
	saveDir := filepath.Join(home, ".llamaoffate", "saves")
	saver := storage.NewYAMLSaver(saveDir)
	slog.Info("Save file location", slog.String("path", saver.Path()))
	return saver
}

// setupSessionLogger creates a session logger with an auto-generated filename
func setupSessionLogger(scenario *scene.Scenario, playerName string) (*session.Logger, error) {
	label := scenario.Genre
	if label == "" {
		label = "game"
	}

	logPath, err := session.GenerateLogPath("session", []string{label, playerName}, 20)
	if err != nil {
		return nil, err
	}

	logger, err := session.NewLogger(logPath)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Session log: %s\n", logPath)
	return logger, nil
}

func showHelp() {
	fmt.Printf("Usage: %s [command]\n\n", AppName)
	fmt.Println("Commands:")
	fmt.Println("  help     - Show this help message")
	fmt.Println("  version  - Show version information")
	fmt.Println()
	fmt.Println("If no command is provided, starts the game.")
}

func showVersion() {
	fmt.Printf("%s v%s\n", AppName, AppVersion)
	fmt.Println("by C. Ross (https://github.com/C-Ross)")
	fmt.Println("GitHub: https://github.com/C-Ross/LlamaOfFate")
	fmt.Println("A Fate Core RPG with LLM integration")
	fmt.Println()
	fmt.Println("This work is based on Fate Core System,")
	fmt.Println("a product of Evil Hat Productions, LLC, developed, authored, and edited")
	fmt.Println("by Leonard Balsera, Brian Engard, Jeremy Keller, Ryan Macklin, Mike Olson,")
	fmt.Println("Clark Valentine, Amanda Valentine, Fred Hicks, and Rob Donoghue, and")
	fmt.Println("licensed for our use under the Creative Commons Attribution 3.0 Unported")
	fmt.Println("license (https://creativecommons.org/licenses/by/3.0/).")
	fmt.Println()
	fmt.Println("Fate Core System Reference Document: https://fate-srd.com/")
}
