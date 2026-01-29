package main

import (
	"context"
	"fmt"
	"log"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
)

func main() {
	logging.SetupDefaultLogging()

	fmt.Println("=== Fate Core Action Parser Example ===")
	fmt.Println()

	// Load Azure configuration
	config, err := azure.LoadConfig("configs/azure-llm.yaml")
	if err != nil {
		log.Fatalf("Failed to load Azure config: %v", err)
	}

	// Create Azure ML client
	azureClient := azure.NewClient(*config)

	// Create action parser
	actionParser := engine.NewActionParser(azureClient)

	// Create a test character
	char := character.NewCharacter("player-001", "Zara the Swift")
	char.Aspects.HighConcept = "Acrobatic Cat Burglar"
	char.Aspects.Trouble = "Can't Resist a Shiny Challenge"
	char.Aspects.AddAspect("Friends in Low Places")
	char.Aspects.AddAspect("Parkour Expert")
	char.SetSkill("Athletics", dice.Great)
	char.SetSkill("Stealth", dice.Good)
	char.SetSkill("Burglary", dice.Good)
	char.SetSkill("Deceive", dice.Fair)
	char.SetSkill("Fight", dice.Average)

	fmt.Printf("Character: %s\n", char.Name)
	fmt.Printf("High Concept: %s\n", char.Aspects.HighConcept)
	fmt.Printf("Trouble: %s\n", char.Aspects.Trouble)
	fmt.Printf("Other Aspects: %v\n", char.Aspects.OtherAspects)
	fmt.Printf("Key Skills: Athletics %s, Stealth %s, Burglary %s, Deceive %s\n\n",
		char.GetSkill("Athletics").String(),
		char.GetSkill("Stealth").String(),
		char.GetSkill("Burglary").String(),
		char.GetSkill("Deceive").String())

	// Test various player inputs
	testInputs := []struct {
		name    string
		input   string
		context string
	}{
		{
			name:    "Physical Movement",
			input:   "I want to jump across the rooftops to get to the other building",
			context: "On a rooftop in the merchant district, with a narrow alley below and another building across the gap",
		},
		{
			name:    "Stealth Action",
			input:   "I'll sneak past the guards and hide in the shadows",
			context: "In the castle courtyard with two guards patrolling near the main entrance",
		},
		{
			name:    "Social Deception with Target",
			input:   "I'm going to pretend to be a servant and tell the guard that the lord wants to see him urgently",
			context: "Standing near the entrance to the lord's private chambers, with a single guard blocking the way",
		},
		{
			name:    "Combat Attack with Target",
			input:   "I attack the bandit with my dagger!",
			context: "In melee combat with a bandit who just drew his sword",
		},
		{
			name:    "Ranged Attack with Target",
			input:   "I shoot the crossbow at the orc captain",
			context: "In a battle against several orcs, with their captain commanding from the rear",
		},
		{
			name:    "Defensive Action",
			input:   "I try to dodge the incoming crossbow bolt",
			context: "A crossbow-wielding enemy is shooting at me from across the room",
		},
		{
			name:    "Investigation with Target",
			input:   "I want to carefully examine the desk for clues about who was here",
			context: "In the burglarized mansion's study, looking for evidence of the thief",
		},
		{
			name:    "Create Advantage with Target",
			input:   "I'll use my knowledge of the city to find a good vantage point where I can observe the noble's house",
			context: "Trying to stake out Lord Blackwood's mansion in a district I know well",
		},
		{
			name:    "Overcome Obstacle with Target",
			input:   "I need to pick the lock on this chest without making any noise",
			context: "In the noble's bedroom at night, with guards patrolling nearby",
		},
		{
			name:    "Social Attack with Target",
			input:   "I want to intimidate the informant into telling me what he knows about the heist",
			context: "Meeting with a nervous street contact in a dark alley",
		},
		{
			name:    "Create Advantage on Enemy",
			input:   "I'm going to try to trip the guard with my rope to knock him off balance",
			context: "Fighting a heavily armored guard in the castle corridor",
		},
		{
			name:    "Multi-target Area Effect",
			input:   "I throw a smoke bomb to obscure the vision of all the guards in the room",
			context: "Surrounded by three guards in the treasury, need to escape",
		},
	}

	// Create some other characters for scenarios involving multiple characters
	bandit := character.NewCharacter("bandit-1", "Bandit")
	bandit.Aspects.HighConcept = "Desperate Cutthroat"
	bandit.SetSkill("Fight", dice.Fair)

	orcCaptain := character.NewCharacter("orc-captain", "Orc Captain")
	orcCaptain.Aspects.HighConcept = "Battle-Hardened Leader"
	orcCaptain.SetSkill("Fight", dice.Good)

	guard := character.NewCharacter("guard-1", "Castle Guard")
	guard.Aspects.HighConcept = "Loyal Protector"
	guard.SetSkill("Fight", dice.Fair)

	ctx := context.Background()

	for i, test := range testInputs {
		fmt.Printf("=== Test %d: %s ===\n", i+1, test.name)
		fmt.Printf("Player Input: \"%s\"\n", test.input)
		fmt.Printf("Context: %s\n", test.context)

		// Add other characters for combat scenarios
		var otherCharacters []*character.Character
		switch test.name {
		case "Combat Attack with Target":
			otherCharacters = []*character.Character{bandit}
		case "Ranged Attack with Target":
			otherCharacters = []*character.Character{orcCaptain}
		case "Create Advantage on Enemy":
			otherCharacters = []*character.Character{guard}
		}

		// Parse the player input
		req := engine.ActionParseRequest{
			Character:       char,
			RawInput:        test.input,
			Context:         test.context,
			OtherCharacters: otherCharacters,
		}

		parsedAction, err := actionParser.ParseAction(ctx, req)
		if err != nil {
			fmt.Printf("❌ Error parsing action: %v\n\n", err)
			continue
		}

		fmt.Printf("✅ Parsed Action:\n")
		fmt.Printf("   Type: %s\n", parsedAction.Type.String())
		fmt.Printf("   Skill: %s (%s)\n", parsedAction.Skill, char.GetSkill(parsedAction.Skill).String())
		fmt.Printf("   Description: %s\n", parsedAction.Description)
		if parsedAction.Target != "" {
			fmt.Printf("   Target: %s\n", parsedAction.Target)
		}
		fmt.Printf("   Action ID: %s\n", parsedAction.ID)
		fmt.Println()
	}

	fmt.Println("=== Action Parser Example Complete ===")
}
