package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/ui/terminal"
)

// SceneConfig holds all the data needed to set up a scene
type SceneConfig struct {
	Name        string
	Description string
	Player      *character.Character
	NPCs        []*character.Character
	Scene       *scene.Scene
	Farewell    string
}

func main() {
	logging.SetupDefaultLogging()

	// Parse command-line arguments
	sceneFlag := flag.String("scene", "", "Scene to play: tower, heist, saloon (default: random)")
	listFlag := flag.Bool("list", false, "List available scenes and exit")
	flag.Parse()

	availableScenes := []string{"tower", "heist", "saloon"}

	if *listFlag {
		fmt.Println("Available scenes:")
		fmt.Println("  tower  - Fantasy: The Abandoned Wizard's Tower")
		fmt.Println("  heist  - Cyberpunk: Megacorp Data Vault (multiple enemies, dangerous aspect)")
		fmt.Println("  saloon - Western: The Dusty Spur Saloon (non-hostile NPC, two aspects)")
		return
	}

	// Determine which scene to play
	selectedScene := *sceneFlag
	if selectedScene == "" {
		selectedScene = availableScenes[rand.Intn(len(availableScenes))]
		fmt.Printf("Randomly selected scene: %s\n\n", selectedScene)
	}

	fmt.Println("=== LlamaOfFate Scene Demo ===")
	fmt.Println()

	// Check if Azure config exists
	configPath := "configs/azure-llm.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Azure LLM config not found at %s\n", configPath)
		fmt.Println("Please copy configs/azure-llm.yaml.example to configs/azure-llm.yaml")
		fmt.Println("and configure your Azure OpenAI credentials.")
		return
	}

	// Load Azure LLM configuration
	config, err := azure.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load Azure config: %v", err)
	}

	// Create Azure ML client
	azureClient := azure.NewClient(*config)

	// Create the game engine with LLM
	gameEngine, err := engine.NewWithLLM(azureClient)
	if err != nil {
		log.Fatalf("Failed to create engine with LLM: %v", err)
	}

	// Start the engine
	if err := gameEngine.Start(); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	// Build the selected scene configuration
	var sceneConfig SceneConfig
	switch selectedScene {
	case "tower":
		sceneConfig = buildTowerScene()
	case "heist":
		sceneConfig = buildHeistScene()
	case "saloon":
		sceneConfig = buildSaloonScene()
	default:
		log.Fatalf("Unknown scene: %s. Use -list to see available scenes.", selectedScene)
	}

	// Register all characters with the engine
	gameEngine.AddCharacter(sceneConfig.Player)
	for _, npc := range sceneConfig.NPCs {
		gameEngine.AddCharacter(npc)
	}

	// Add all characters to the scene
	sceneConfig.Scene.AddCharacter(sceneConfig.Player.ID)
	for _, npc := range sceneConfig.NPCs {
		sceneConfig.Scene.AddCharacter(npc.ID)
	}

	// Get the scene manager
	sceneManager := gameEngine.GetSceneManager()
	if sceneManager == nil {
		log.Fatal("Scene manager not available")
	}

	// Start the scene with the pre-configured scene
	err = sceneManager.StartScene(sceneConfig.Scene, sceneConfig.Player)
	if err != nil {
		log.Fatalf("Failed to start scene: %v", err)
	}

	// Run the scene loop
	ctx := context.Background()
	terminal := terminal.NewTerminalUI()
	sceneManager.SetUI(terminal)
	sceneManager.SetExitOnSceneTransition(true)
	terminal.SetSceneInfo(sceneManager)

	// Display initial character info
	terminal.DisplayCharacter()

	if err := sceneManager.RunSceneLoop(ctx); err != nil {
		log.Fatalf("Scene loop error: %v", err)
	}

	// Stop the engine
	if err := gameEngine.Stop(); err != nil {
		log.Printf("Warning: Failed to stop engine: %v", err)
	}

	// Display final character state
	terminal.DisplayCharacter()
	fmt.Printf("\n%s\n", sceneConfig.Farewell)
}
