package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"

	gameconfig "github.com/C-Ross/LlamaOfFate/internal/config"
	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/C-Ross/LlamaOfFate/internal/syncdriver"
	"github.com/C-Ross/LlamaOfFate/internal/ui/terminal"
)

// SceneConfig holds all the data needed to set up a scene
type SceneConfig struct {
	Name        string
	Description string
	Scenario    *scene.Scenario
	Player      *core.Character
	NPCs        []*core.Character
	Scene       *scene.Scene
	Farewell    string
}

func main() {
	logging.SetupDefaultLogging()

	// Parse command-line arguments
	sceneFlag := flag.String("scene", "", "Scene to play: tower, heist, saloon (default: random)")
	listFlag := flag.Bool("list", false, "List available scenes and exit")
	logFlag := flag.String("log", "auto", "Session log path (default: auto-generated, empty string disables)")
	multiFlag := flag.Bool("multi", false, "Enable multi-scene mode (continues to new scenes on transition)")
	flag.Parse()

	availableScenes := []string{"tower", "heist", "saloon"}

	if *listFlag {
		fmt.Println("Available scenes:")
		fmt.Println("  tower  - Fantasy: The Abandoned Wizard's Tower")
		fmt.Println("  heist  - Cyberpunk: Megacorp Data Vault (multiple enemies, dangerous aspect)")
		fmt.Println("  saloon - Western: The Dusty Spur Saloon (non-hostile NPC, two aspects)")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  -multi   Enable multi-scene mode (generates new scenes on transition)")
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

	// Set up session logging (default: enabled with auto-generated filename)
	logPath := *logFlag
	if logPath == "auto" {
		var err error
		logPath, err = session.GenerateLogPath("session", []string{selectedScene}, 0)
		if err != nil {
			log.Fatalf("Failed to generate log path: %v", err)
		}
	}
	var sessionLogger *session.Logger
	var sl session.SessionLogger
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
		sl = sessionLogger
	} else {
		sl = session.NullLogger{}
	}

	// Create the game engine with LLM
	gameEngine, err := engine.NewWithLLM(azureClient, sl)
	if err != nil {
		log.Fatalf("Failed to create engine with LLM: %v", err)
	}

	// Start the engine
	if err := gameEngine.Start(); err != nil {
		log.Fatalf("Failed to start engine: %v", err)
	}

	// Load scenarios from YAML
	scenarios, err := gameconfig.LoadAll("configs")
	if err != nil {
		log.Fatalf("Failed to load scenario configs: %v", err)
	}

	// Build the selected scene configuration from YAML
	ls, ok := scenarios[selectedScene]
	if !ok {
		log.Fatalf("Unknown scene: %s. Use -list to see available scenes.", selectedScene)
	}
	sceneConfig := SceneConfig{
		Name:        ls.Raw.Title,
		Description: ls.Raw.Description,
		Scenario:    ls.Scenario,
		Player:      ls.Player,
		NPCs:        ls.NPCs,
		Scene:       ls.Scene,
		Farewell:    ls.Farewell,
	}

	ctx := context.Background()
	terminalUI := terminal.NewTerminalUI()

	if *multiFlag {
		fmt.Println("*** Multi-scene mode enabled - scenes will continue on transition ***")
		fmt.Println()
	}

	runGame(ctx, gameEngine, sceneConfig, terminalUI, sl, *multiFlag)

	// Stop the engine
	if err := gameEngine.Stop(); err != nil {
		log.Printf("Warning: Failed to stop engine: %v", err)
	}

	// Display final character state
	terminalUI.DisplayCharacter()
	fmt.Printf("\n%s\n", sceneConfig.Farewell)
}

// runGame sets up a GameManager and runs the game. In single-scene mode
// (multi=false), ExitAfterScene causes the game to end when the first scene
// completes. In multi-scene mode, scenes continue to be generated on transition.
func runGame(ctx context.Context, gameEngine *engine.Engine, sceneConfig SceneConfig, terminalUI *terminal.TerminalUI, sessionLogger session.SessionLogger, multi bool) {
	// Create and configure the game manager
	gameManager := engine.NewGameManager(gameEngine, sessionLogger)
	gameManager.SetPlayer(sceneConfig.Player)

	// Set genre-appropriate scenario
	gameManager.SetScenario(sceneConfig.Scenario)

	// Set up the initial scene
	gameManager.SetInitialScene(&engine.InitialSceneConfig{
		Scene:          sceneConfig.Scene,
		NPCs:           sceneConfig.NPCs,
		ExitAfterScene: !multi,
	})

	// Display initial character info
	terminalUI.SetSceneInfo(gameEngine.GetSceneManager())
	terminalUI.DisplayCharacter()

	// Run the blocking game loop
	onStart := func() {
		terminalUI.SetSceneInfo(gameEngine.GetSceneManager())
	}
	if err := syncdriver.Run(ctx, gameManager, terminalUI, onStart); err != nil {
		log.Fatalf("Game error: %v", err)
	}
}
