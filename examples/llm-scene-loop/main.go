package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/session"
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
		logPath = fmt.Sprintf("session_%s_%s.yaml", selectedScene, time.Now().Format("20060102_150405"))
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
		// Multi-scene mode: use GameManager for continuous play
		runMultiSceneMode(ctx, gameEngine, sceneConfig, terminalUI, sessionLogger, selectedScene)
	} else {
		// Single-scene mode: original behavior
		runSingleSceneMode(ctx, gameEngine, sceneConfig, terminalUI, sessionLogger)
	}

	// Stop the engine
	if err := gameEngine.Stop(); err != nil {
		log.Printf("Warning: Failed to stop engine: %v", err)
	}

	// Display final character state
	terminalUI.DisplayCharacter()
	fmt.Printf("\n%s\n", sceneConfig.Farewell)
}

// runSingleSceneMode runs the original single-scene behavior
func runSingleSceneMode(ctx context.Context, gameEngine *engine.Engine, sceneConfig SceneConfig, terminalUI *terminal.TerminalUI, sessionLogger *session.Logger) {
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

	if sessionLogger != nil {
		sceneManager.SetSessionLogger(sessionLogger)
	}

	// Start the scene with the pre-configured scene
	err := sceneManager.StartScene(sceneConfig.Scene, sceneConfig.Player)
	if err != nil {
		log.Fatalf("Failed to start scene: %v", err)
	}

	// Run the scene loop
	sceneManager.SetUI(terminalUI)
	sceneManager.SetExitOnSceneTransition(true)
	terminalUI.SetSceneInfo(sceneManager)

	// Display initial character info
	terminalUI.DisplayCharacter()

	if _, err := sceneManager.RunSceneLoop(ctx); err != nil {
		log.Fatalf("Scene loop error: %v", err)
	}
}

// runMultiSceneMode uses GameManager for multi-scene continuous play
func runMultiSceneMode(ctx context.Context, gameEngine *engine.Engine, sceneConfig SceneConfig, terminalUI *terminal.TerminalUI, sessionLogger *session.Logger, sceneName string) {
	fmt.Println("*** Multi-scene mode enabled - scenes will continue on transition ***")
	fmt.Println()

	// Create and configure the game manager
	gameManager := engine.NewGameManager(gameEngine)
	gameManager.SetPlayer(sceneConfig.Player)
	gameManager.SetUI(terminalUI)
	if sessionLogger != nil {
		gameManager.SetSessionLogger(sessionLogger)
	}

	// Set genre-appropriate scenario
	gameManager.SetScenario(getScenario(sceneName))

	// Set up terminal UI with scene info callback
	terminalUI.SetSceneInfo(gameEngine.GetSceneManager())

	// Display initial character info
	terminalUI.DisplayCharacter()

	// Run with the initial scene
	initialScene := &engine.InitialSceneConfig{
		Scene: sceneConfig.Scene,
		NPCs:  sceneConfig.NPCs,
	}

	if err := gameManager.RunWithInitialScene(ctx, initialScene); err != nil {
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
