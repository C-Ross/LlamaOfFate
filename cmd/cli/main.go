package main

import (
	"fmt"
	"log"
	"os"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/ui/text"
)

const (
	AppName    = "LlamaOfFate"
	AppVersion = "0.1.0"
)

func main() {
	// Setup logging early
	logging.SetupDefaultLogging()

	fmt.Printf("%s v%s - A Fate Core RPG with LLM integration\n", AppName, AppVersion)
	fmt.Println("====================================================")

	// Initialize the game engine
	gameEngine, err := engine.New()
	if err != nil {
		log.Fatalf("Failed to initialize game engine: %v", err)
	}

	// Initialize the CLI interface
	cli, err := text.NewCLI(gameEngine)
	if err != nil {
		log.Fatalf("Failed to initialize CLI: %v", err)
	}

	// Check for command line arguments
	if len(os.Args) > 1 {
		// Handle command line arguments
		handleArgs(os.Args[1:], cli)
		return
	}

	// Start interactive mode
	fmt.Println("Starting interactive mode...")
	fmt.Println("Type 'help' for available commands or 'quit' to exit.")
	fmt.Println()

	if err := cli.Run(); err != nil {
		log.Fatalf("CLI error: %v", err)
	}
}

func handleArgs(args []string, cli *text.CLI) {
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
