package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
)

func main() {
	fmt.Println("=== LlamaOfFate Enhanced Scene Demo with LLM ===")
	fmt.Println("This demonstrates the LLM-driven scene loop with intelligent")
	fmt.Println("classification of dialog, clarification, and actions.")
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

	// Display LLM information
	modelInfo := azureClient.GetModelInfo()
	fmt.Printf("Using LLM: %s (%s)\n", modelInfo.Name, modelInfo.Provider)
	fmt.Println()

	// Create a sample character
	player := createSampleCharacter()

	// Get the scene manager
	sceneManager := gameEngine.GetSceneManager()
	if sceneManager == nil {
		log.Fatal("Scene manager not available")
	}

	// Start a rich sample scene
	err = sceneManager.StartScene(
		"abandoned-tower",
		"The Abandoned Wizard's Tower",
		"You stand before a crumbling stone tower that stretches high into the mist. "+
			"Strange blue lights flicker through broken windows near the top. "+
			"The heavy wooden door hangs ajar, revealing darkness within. "+
			"Ancient runes are carved deep into the stone archway, still glowing faintly with magical energy. "+
			"The air hums with residual magic, and you hear the distant sound of something large moving inside.",
		player,
	)
	if err != nil {
		log.Fatalf("Failed to start scene: %v", err)
	}

	fmt.Println()
	fmt.Println("=== Enhanced Features ===")
	fmt.Println("• The LLM will classify your input as dialog, clarification, or action")
	fmt.Println("• Conversation history is maintained for context")
	fmt.Println("• Rich narrative responses based on scene context")
	fmt.Println("• Intelligent action parsing and resolution")
	fmt.Println()
	fmt.Println("Try saying:")
	fmt.Println(`  "What do I see in more detail?"`)
	fmt.Println(`  "I call out 'Hello, is anyone there?'"`)
	fmt.Println(`  "Carefully examine the runes"`)
	fmt.Println(`  "Sneak through the doorway"`)
	fmt.Println()

	// Run the scene loop
	ctx := context.Background()
	if err := sceneManager.RunSceneLoop(ctx); err != nil {
		log.Fatalf("Scene loop error: %v", err)
	}

	// Stop the engine
	if err := gameEngine.Stop(); err != nil {
		log.Printf("Warning: Failed to stop engine: %v", err)
	}

	fmt.Println("Thanks for exploring the tower!")
}

func createSampleCharacter() *character.Character {
	player := character.NewCharacter("player1", "Lyra Moonwhisper")

	// Set aspects
	player.Aspects.HighConcept = "Scholarly Arcane Investigator"
	player.Aspects.Trouble = "Curiosity About Forbidden Knowledge"
	player.Aspects.AddAspect("Trained by the College of Mages")
	player.Aspects.AddAspect("Sees Magic in Everything")
	player.Aspects.AddAspect("Protective Ward Tattoos")

	// Set skills focused on investigation and magic
	player.SetSkill("Lore", dice.Superb)       // +5 - Primary skill
	player.SetSkill("Investigate", dice.Great) // +4
	player.SetSkill("Notice", dice.Great)      // +4
	player.SetSkill("Will", dice.Good)         // +3
	player.SetSkill("Empathy", dice.Good)      // +3
	player.SetSkill("Rapport", dice.Good)      // +3
	player.SetSkill("Athletics", dice.Fair)    // +2
	player.SetSkill("Burglary", dice.Fair)     // +2
	player.SetSkill("Stealth", dice.Fair)      // +2
	player.SetSkill("Crafts", dice.Fair)       // +2
	player.SetSkill("Deceive", dice.Average)   // +1
	player.SetSkill("Drive", dice.Average)     // +1
	player.SetSkill("Fight", dice.Average)     // +1
	player.SetSkill("Physique", dice.Average)  // +1
	player.SetSkill("Provoke", dice.Average)   // +1

	// Set fate points and refresh
	player.FatePoints = 3
	player.Refresh = 3

	return player
}
