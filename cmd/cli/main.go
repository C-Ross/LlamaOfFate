package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/ui/terminal"
)

const (
	AppName    = "LlamaOfFate"
	AppVersion = "0.1.0"
)

// initializeEngine attempts to create a game engine with LLM support if available
func initializeEngine() *engine.Engine {
	// Try to load LLM configuration
	configPath := "configs/azure-llm.yaml"
	if _, err := os.Stat(configPath); err != nil {

		log.Fatalf("LLM config not found at %s", configPath)
	}
	// Config file exists, try to load it
	config, err := azure.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Found LLM config but failed to load it: %v", err)
	}

	// Try to create engine with LLM
	azureClient := azure.NewClient(*config)

	// Wrap the Azure client with retry logic for resilience
	retryClient := llm.NewRetryingClient(azureClient, llm.DefaultRetryConfig())

	gameEngine, err := engine.NewWithLLM(retryClient)
	if err != nil {
		log.Fatalf("Failed to create engine with LLM: %v", err)
	}

	slog.Info("LLM integration enabled", slog.String("model", config.ModelName))
	return gameEngine
}

func main() {
	// Setup logging early
	logging.SetupDefaultLogging()

	fmt.Printf("%s v%s - A Fate Core RPG with LLM integration\n", AppName, AppVersion)
	fmt.Println("====================================================")

	// Initialize the game engine, preferably with LLM if available
	gameEngine := initializeEngine()

	// Check for command line arguments
	if len(os.Args) > 1 {
		// Handle command line arguments
		handleArgs(os.Args[1:])
		return
	}

	if err := runInteractiveMode(gameEngine); err != nil {
		log.Fatalf("Interactive mode error: %v", err)
	}
}

func handleArgs(args []string) {
	switch args[0] {
	case "version":
		showVersion()
	case "help":
		showHelp()
	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		showHelp()
		os.Exit(1)
	}
}

func runInteractiveMode(gameEngine *engine.Engine) error {
	ui := terminal.NewTerminalUI()

	return startSampleScene(gameEngine, ui)
}

// startSampleScene starts a sample scene for testing
func startSampleScene(gameEngine *engine.Engine, ui *terminal.TerminalUI) error {
	fmt.Println("Starting sample scene...")

	// Check if LLM is available
	sceneManager := gameEngine.GetSceneManager()
	if sceneManager == nil {
		return fmt.Errorf("scene manager not available")
	}

	hasLLM := gameEngine.GetActionParser() != nil
	if hasLLM {
		fmt.Println("✓ Enhanced scene with LLM support - natural language actions available!")
		fmt.Println("  You can try commands like:")
		fmt.Println("    \"What do I see?\" \"Examine the symbols\" \"Enter the cave cautiously\"")
	} else {
		fmt.Println("⚠ Basic scene mode - LLM not configured")
		fmt.Println("  Scene commands available: help, scene, character, status, aspects")
		fmt.Println("  Configure LLM for natural language interactions")
	}
	fmt.Println()

	// Create a sample character
	player := character.NewCharacter("player1", "Test Character")
	player.Aspects.HighConcept = "Brave Adventurer"
	player.Aspects.Trouble = "Too Curious for My Own Good"
	player.SetSkill("Athletics", dice.Good)
	player.SetSkill("Fight", dice.Fair)
	player.SetSkill("Notice", dice.Fair)

	// Create a sample scene
	sampleScene := scene.NewScene(
		"test-scene",
		"A Mysterious Cave",
		"You stand at the entrance of a dark cave. Cool air flows from within, "+
			"and you can hear the distant sound of dripping water. "+
			"Strange symbols are carved into the stone archway above.",
	)

	// Add the player to the scene
	sampleScene.AddCharacter(player.ID)

	// Set up the UI for the scene manager
	sceneManager.SetUI(ui)

	// Set scene info for the UI so it can display character and scene details
	ui.SetSceneInfo(sceneManager)

	// Start scene
	if err := sceneManager.StartScene(sampleScene, player); err != nil {
		return fmt.Errorf("failed to start scene: %w", err)
	}

	// Run scene loop
	ctx := context.Background()
	if err := sceneManager.RunSceneLoop(ctx); err != nil {
		return fmt.Errorf("scene loop error: %w", err)
	}

	return nil
}

func showHelp() {
	fmt.Printf("Usage: %s [command]\n\n", AppName)
	fmt.Println("Commands:")
	fmt.Println("  help     - Show this help message")
	fmt.Println("  version  - Show version information")
	fmt.Println()
	fmt.Println("If no command is provided, starts in interactive mode.")
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
