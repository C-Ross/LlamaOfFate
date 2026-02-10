package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/C-Ross/LlamaOfFate/internal/storage"
	"github.com/C-Ross/LlamaOfFate/internal/ui/terminal"
)

const (
	AppName    = "LlamaOfFate"
	AppVersion = "0.1.0"
)

// initializeEngine creates a game engine with LLM support
func initializeEngine() *engine.Engine {
	configPath := "configs/azure-llm.yaml"
	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("LLM config not found at %s", configPath)
	}

	config, err := azure.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Found LLM config but failed to load it: %v", err)
	}

	azureClient := azure.NewClient(*config)
	retryClient := llm.NewRetryingClient(azureClient, llm.DefaultRetryConfig())

	gameEngine, err := engine.NewWithLLM(retryClient)
	if err != nil {
		log.Fatalf("Failed to create engine with LLM: %v", err)
	}

	slog.Info("LLM integration enabled", slog.String("model", config.ModelName))
	return gameEngine
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

	gameEngine := initializeEngine()

	// Use hardcoded scenario and player (see scenario.go)
	scenario := defaultScenario()
	player := defaultPlayer()

	fmt.Printf("Scenario: %s (%s)\n", scenario.Title, scenario.Genre)
	fmt.Printf("Player:   %s — \"%s\"\n", player.Name, player.Aspects.HighConcept)
	fmt.Println()

	// Set up session logging
	sessionLogger, err := setupSessionLogger(scenario, player.Name)
	if err != nil {
		log.Fatalf("Failed to create session logger: %v", err)
	}
	if sessionLogger != nil {
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
		})
	}

	// Wire everything into the GameManager and run
	ui := terminal.NewTerminalUI()

	gm := engine.NewGameManager(gameEngine)
	gm.SetPlayer(player)
	gm.SetUI(ui)
	gm.SetScenario(scenario)
	gm.SetSaver(newSaver())
	if sessionLogger != nil {
		gm.SetSessionLogger(sessionLogger)
	}

	fmt.Println("Type naturally to interact, 'quit' to exit.")
	fmt.Println("====================================================")

	ctx := context.Background()
	if err := gm.Run(ctx); err != nil {
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
	label := strings.ToLower(scenario.Genre)
	if label == "" {
		label = "game"
	}
	safeName := strings.ToLower(strings.ReplaceAll(playerName, " ", "_"))
	safeName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, safeName)
	if len(safeName) > 20 {
		safeName = safeName[:20]
	}
	logPath := fmt.Sprintf("session_%s_%s_%s.yaml", label, safeName, time.Now().Format("20060102_150405"))

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
	fmt.Println("A Fate Core RPG with LLM integration")
	fmt.Println()
	fmt.Println("This work is based on Fate Core System,")
	fmt.Println("products of Evil Hat Productions, LLC, developed, authored, and edited")
	fmt.Println("by Leonard Balsera, Brian Engard, Jeremy Keller, Ryan Macklin, Mike Olson,")
	fmt.Println("Clark Valentine, Amanda Valentine, Fred Hicks, and Rob Donoghue, and")
	fmt.Println("licensed for our use under the Creative Commons Attribution 3.0 Unported")
	fmt.Println("license (http://creativecommons.org/licenses/by/3.0/).")
	fmt.Println()
	fmt.Println("Fate Core System Reference Document: https://fate-srd.com/")
}
