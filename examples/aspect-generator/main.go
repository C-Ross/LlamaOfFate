package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
)

func main() {
	fmt.Println("=== Fate Core Aspect Generator Example ===")
	fmt.Println()

	// Load Azure configuration
	config, err := azure.LoadConfig("configs/azure-llm.yaml")
	if err != nil {
		log.Fatalf("Failed to load Azure config: %v", err)
	}

	// Create Azure ML client
	azureClient := azure.NewClient(*config)

	// Create aspect generator
	aspectGenerator := engine.NewAspectGenerator(azureClient)

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

	fmt.Printf("Character: %s\n", char.Name)
	fmt.Printf("High Concept: %s\n", char.Aspects.HighConcept)
	fmt.Printf("Trouble: %s\n", char.Aspects.Trouble)
	fmt.Printf("Other Aspects: %v\n", char.Aspects.OtherAspects)
	fmt.Printf("Key Skills: Athletics %s, Stealth %s, Burglary %s\n\n",
		char.GetSkill("Athletics").String(),
		char.GetSkill("Stealth").String(),
		char.GetSkill("Burglary").String())

	// Create some test scenarios
	scenarios := []struct {
		name        string
		skill       string
		description string
		rawInput    string
		difficulty  dice.Ladder
		context     string
		targetType  string
		existing    []string
		rollResult  dice.CheckResult
	}{
		{
			name:        "Rooftop Chase - Athletics",
			skill:       "Athletics",
			description: "Parkour across rooftops to gain advantage",
			rawInput:    "I want to use my parkour skills to get to higher ground and find the perfect spot to jump down on my target",
			difficulty:  dice.Fair,
			context:     "A chase scene across the rooftops of the old town. Narrow alleys below, various building heights, clotheslines and chimneys provide obstacles and opportunities.",
			targetType:  "situation",
			existing:    []string{"Narrow Alleyways Below", "Uneven Rooftop Heights"},
			rollResult:  dice.CheckResult{Roll: &dice.Roll{Dice: [4]dice.FateDie{dice.Plus, dice.Blank, dice.Plus, dice.Minus}, Total: 1}, BaseSkill: dice.Great, Modifier: 0, FinalValue: dice.Superb}, // Great(4) + 1 = Superb(5)
		},
		{
			name:        "Stealth Infiltration",
			skill:       "Stealth",
			description: "Find a hidden vantage point",
			rawInput:    "I want to scout the area and find a good hiding spot where I can observe the guards' patrol patterns",
			difficulty:  dice.Good,
			context:     "The grand estate's courtyard at night. Manicured gardens with topiary, a central fountain, and several guard patrols. Gas lamps provide pools of light with deep shadows between.",
			targetType:  "character",
			existing:    []string{"Patrolling Guards", "Pools of Lamplight", "Ornate Topiary"},
			rollResult:  dice.CheckResult{Roll: &dice.Roll{Dice: [4]dice.FateDie{dice.Blank, dice.Plus, dice.Blank, dice.Plus}, Total: 2}, BaseSkill: dice.Good, Modifier: 0, FinalValue: dice.Superb}, // Good(3) + 2 = Superb(5)
		},
		{
			name:        "Deception Distraction",
			skill:       "Deceive",
			description: "Create a diversion with false information",
			rawInput:    "I'm going to start a rumor about a fire in the east wing to draw the guards away from the vault",
			difficulty:  dice.Fair,
			context:     "Inside the estate during a fancy party. Well-dressed guests, servants carrying trays, guards trying to blend in while staying alert. Perfect cover for social engineering.",
			targetType:  "situation",
			existing:    []string{"Crowded Party", "Distracted Guards", "Gossiping Nobles"},
			rollResult:  dice.CheckResult{Roll: &dice.Roll{Dice: [4]dice.FateDie{dice.Minus, dice.Blank, dice.Blank, dice.Plus}, Total: 0}, BaseSkill: dice.Fair, Modifier: 0, FinalValue: dice.Fair}, // Fair(2) + 0 = Fair(2)
		},
	}

	ctx := context.Background()

	for i, scenario := range scenarios {
		fmt.Printf("=== Scenario %d: %s ===\n", i+1, scenario.name)

		// Create the action
		testAction := action.NewAction(
			fmt.Sprintf("action-%d", i+1),
			char.ID,
			action.CreateAdvantage,
			scenario.skill,
			scenario.description,
		)
		testAction.RawInput = scenario.rawInput
		testAction.Difficulty = scenario.difficulty

		// Use the predefined roll result and derive outcome using CompareAgainst
		testAction.CheckResult = &scenario.rollResult
		outcome := scenario.rollResult.CompareAgainst(scenario.difficulty)
		testAction.Outcome = outcome

		fmt.Printf("Action: %s\n", testAction.Description)
		fmt.Printf("Player Intent: %s\n", testAction.RawInput)
		fmt.Printf("Skill Used: %s (%s)\n", scenario.skill, scenario.rollResult.BaseSkill.String())
		fmt.Printf("Difficulty: %s\n", scenario.difficulty.String())
		fmt.Printf("Roll: %s (Total: %+d)\n", scenario.rollResult.Roll.String(), scenario.rollResult.Roll.Total)
		fmt.Printf("Final Result: %s vs %s = %s (%+d shifts)\n",
			scenario.rollResult.FinalValue.String(),
			scenario.difficulty.String(),
			outcome.Type.String(),
			outcome.Shifts)

		// Create the aspect generation request
		req := engine.AspectGenerationRequest{
			Character:       char,
			Action:          testAction,
			Outcome:         outcome,
			Context:         scenario.context,
			TargetType:      scenario.targetType,
			ExistingAspects: scenario.existing,
		}

		// Generate the aspect using the LLM
		fmt.Printf("\nGenerating aspect with Azure LLM...\n")
		response, err := aspectGenerator.GenerateAspect(ctx, req)
		if err != nil {
			fmt.Printf("Error generating aspect: %v\n", err)
			continue
		}

		// Display results
		fmt.Printf("\n--- Generated Aspect ---\n")
		fmt.Printf("Aspect: \"%s\"\n", response.AspectText)
		fmt.Printf("Description: %s\n", response.Description)
		fmt.Printf("Duration: %s\n", response.Duration)
		fmt.Printf("Free Invokes: %d\n", response.FreeInvokes)
		if response.IsBoost {
			fmt.Printf("Type: Boost (disappears after one use or end of scene)\n")
		} else {
			fmt.Printf("Type: Full Aspect\n")
		}
		fmt.Printf("Reasoning: %s\n", response.Reasoning)

		separator := strings.Repeat("=", 60)
		fmt.Printf("\n%s\n\n", separator)
	}

	fmt.Println("Example completed! Try different scenarios by modifying the code above.")
}
