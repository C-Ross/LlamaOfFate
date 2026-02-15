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
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/C-Ross/LlamaOfFate/internal/syncdriver"
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

	ctx := context.Background()
	terminalUI := terminal.NewTerminalUI()

	if *multiFlag {
		fmt.Println("*** Multi-scene mode enabled - scenes will continue on transition ***")
		fmt.Println()
	}

	runGame(ctx, gameEngine, sceneConfig, terminalUI, sessionLogger, selectedScene, *multiFlag)

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
func runGame(ctx context.Context, gameEngine *engine.Engine, sceneConfig SceneConfig, terminalUI *terminal.TerminalUI, sessionLogger *session.Logger, sceneName string, multi bool) {
	// Create and configure the game manager
	gameManager := engine.NewGameManager(gameEngine)
	gameManager.SetPlayer(sceneConfig.Player)
	if sessionLogger != nil {
		gameManager.SetSessionLogger(sessionLogger)
	}

	// Set genre-appropriate scenario
	gameManager.SetScenario(getScenario(sceneName))

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

// getScenario returns appropriate scenario for each scene type
func getScenario(sceneName string) *scene.Scenario {
	scenario := scene.PredefinedScenario(sceneName)
	if scenario != nil {
		return scenario
	}
	return &scene.Scenario{
		Title:   "A New Adventure",
		Problem: "Danger lurks and heroes are needed",
		StoryQuestions: []string{
			"Can the heroes prevail?",
		},
		Genre:   "Adventure",
		Setting: "A world of adventure and danger where heroes rise to meet challenges.",
	}
}
