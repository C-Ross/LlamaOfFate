//go:build llmeval

package llmeval_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// InputClassificationTestCase represents a test case for input classification
type InputClassificationTestCase struct {
	Name             string
	RawInput         string
	SceneName        string
	SceneDescription string
	ExpectedType     string // "dialog", "clarification", or "action"
	Description      string // Human-readable description of why this should be classified this way
}

// getDialogTestCases returns inputs that should be classified as dialog
func getDialogTestCases() []InputClassificationTestCase {
	return []InputClassificationTestCase{
		{
			Name:             "Direct speech",
			RawInput:         "I say to the bartender 'Another round for my friends!'",
			SceneName:        "The Rusty Anchor Tavern",
			SceneDescription: "A busy tavern filled with sailors and merchants",
			ExpectedType:     "dialog",
			Description:      "Direct speech to an NPC - clear dialog",
		},
		{
			Name:             "Ask NPC a question",
			RawInput:         "I ask the guard what happened here",
			SceneName:        "City Gate",
			SceneDescription: "The main entrance to the city, guards patrol the area",
			ExpectedType:     "dialog",
			Description:      "Asking an NPC for information - dialog",
		},
		{
			Name:             "Greet someone",
			RawInput:         "Hello there, friend!",
			SceneName:        "Market Square",
			SceneDescription: "A bustling marketplace with vendors and shoppers",
			ExpectedType:     "dialog",
			Description:      "Simple greeting - dialog",
		},
		{
			Name:             "Request information from NPC",
			RawInput:         "Ask the bartender about rumors",
			SceneName:        "The Rusty Anchor Tavern",
			SceneDescription: "A busy tavern filled with sailors and merchants",
			ExpectedType:     "dialog",
			Description:      "Requesting information from NPC - dialog",
		},
		{
			Name:             "Introduce myself",
			RawInput:         "I introduce myself to the merchant",
			SceneName:        "Trading Post",
			SceneDescription: "A well-stocked trading post on the frontier",
			ExpectedType:     "dialog",
			Description:      "Social introduction - dialog",
		},
	}
}

// getClarificationTestCases returns inputs that should be classified as clarification
func getClarificationTestCases() []InputClassificationTestCase {
	return []InputClassificationTestCase{
		{
			Name:             "Look around",
			RawInput:         "What do I see?",
			SceneName:        "Abandoned Warehouse",
			SceneDescription: "A dusty warehouse with crates stacked against the walls",
			ExpectedType:     "clarification",
			Description:      "Basic observation request - clarification",
		},
		{
			Name:             "Examine environment",
			RawInput:         "Look around the room",
			SceneName:        "Study Chamber",
			SceneDescription: "A scholarly room filled with books and scrolls",
			ExpectedType:     "clarification",
			Description:      "General observation - clarification",
		},
		{
			Name:             "Examine object",
			RawInput:         "Examine the mysterious door",
			SceneName:        "Ancient Temple",
			SceneDescription: "A temple entrance with strange carvings",
			ExpectedType:     "clarification",
			Description:      "Examining something without opposition - clarification",
		},
		{
			Name:             "Check surroundings",
			RawInput:         "I check what's in this room",
			SceneName:        "Inn Room",
			SceneDescription: "A simple but clean room at the local inn",
			ExpectedType:     "clarification",
			Description:      "Basic environment check - clarification",
		},
		{
			Name:             "Read something visible",
			RawInput:         "What does the sign say?",
			SceneName:        "Town Square",
			SceneDescription: "The central square with a notice board and fountain",
			ExpectedType:     "clarification",
			Description:      "Reading visible text - clarification",
		},
		{
			Name:             "Passive observation",
			RawInput:         "I lean back in my chair and watch the crowd",
			SceneName:        "Cafe Terrace",
			SceneDescription: "An outdoor cafe overlooking the busy market",
			ExpectedType:     "clarification",
			Description:      "Passive watching without specific goal - clarification",
		},
		{
			Name:             "Simple movement",
			RawInput:         "I walk over to the window and look outside",
			SceneName:        "Inn Room",
			SceneDescription: "A cozy room at the inn, safe and quiet",
			ExpectedType:     "clarification",
			Description:      "Mundane movement to observe - clarification",
		},
		{
			Name:             "Thinking",
			RawInput:         "I think about what the old man said earlier",
			SceneName:        "City Streets",
			SceneDescription: "Walking through quiet evening streets",
			ExpectedType:     "clarification",
			Description:      "Internal reflection - clarification or dialog",
		},
	}
}

// getActionTestCases returns inputs that should be classified as action
func getActionTestCases() []InputClassificationTestCase {
	return []InputClassificationTestCase{
		{
			Name:             "Attack enemy",
			RawInput:         "I attack the goblin with my sword",
			SceneName:        "Forest Clearing",
			SceneDescription: "A clearing where goblins have ambushed you",
			ExpectedType:     "action",
			Description:      "Combat attack - clear action",
		},
		{
			Name:             "Sneak past guards",
			RawInput:         "I try to sneak past the guards",
			SceneName:        "Castle Corridor",
			SceneDescription: "A corridor patrolled by alert guards",
			ExpectedType:     "action",
			Description:      "Stealth with opposition - action",
		},
		{
			Name:             "Pick lock under pressure",
			RawInput:         "I pick the lock before the guards return",
			SceneName:        "Noble's Estate",
			SceneDescription: "Outside a locked door, guards patrol nearby",
			ExpectedType:     "action",
			Description:      "Lock picking with time pressure - action",
		},
		{
			Name:             "Jump across chasm",
			RawInput:         "I jump across the chasm",
			SceneName:        "Mountain Pass",
			SceneDescription: "A deep chasm blocks the mountain path",
			ExpectedType:     "action",
			Description:      "Physical obstacle - action",
		},
		{
			Name:             "Climb wall",
			RawInput:         "I climb up the wall to get over it",
			SceneName:        "City Walls",
			SceneDescription: "A 20-foot stone wall blocks your path",
			ExpectedType:     "action",
			Description:      "Physical challenge - action",
		},
		{
			Name:             "Convince guard",
			RawInput:         "I try to talk my way past the guard",
			SceneName:        "Restricted Area",
			SceneDescription: "A guard blocks entry to the restricted section",
			ExpectedType:     "action",
			Description:      "Social challenge with opposition - action",
		},
		{
			Name:             "Create distraction",
			RawInput:         "I create a distraction to draw the guards away",
			SceneName:        "Palace Entrance",
			SceneDescription: "Guards block the palace entrance you need to pass",
			ExpectedType:     "action",
			Description:      "Tactical maneuver - action",
		},
		{
			Name:             "Study opponent",
			RawInput:         "I study the duelist's fighting style to find a weakness",
			SceneName:        "Dueling Arena",
			SceneDescription: "About to face a renowned swordsman in a duel",
			ExpectedType:     "action",
			Description:      "Tactical preparation - action (Create Advantage)",
		},
		{
			Name:             "Flee pursuers",
			RawInput:         "I run as fast as I can to escape",
			SceneName:        "City Streets",
			SceneDescription: "Being chased through narrow streets by angry guards",
			ExpectedType:     "action",
			Description:      "Escape with opposition - action",
		},
		{
			Name:             "Break chains",
			RawInput:         "I struggle to break free from these chains",
			SceneName:        "Dungeon Cell",
			SceneDescription: "Chained to the wall in a dark dungeon",
			ExpectedType:     "action",
			Description:      "Physical obstacle - action",
		},
		{
			Name:             "Intimidate prisoner",
			RawInput:         "I intimidate the prisoner to make him talk",
			SceneName:        "Interrogation Room",
			SceneDescription: "Questioning a captured spy",
			ExpectedType:     "action",
			Description:      "Social attack - action",
		},
		{
			Name:             "Scout ahead",
			RawInput:         "I sneak closer to scout the bandit camp",
			SceneName:        "Forest Edge",
			SceneDescription: "Approaching a bandit camp, guards patrol the perimeter",
			ExpectedType:     "action",
			Description:      "Stealth reconnaissance with risk of detection - action",
		},
	}
}

// getThirdPersonClassificationTestCases returns inputs using third-person language (character names/pronouns)
// Players often describe their character's actions in third person
func getThirdPersonClassificationTestCases() []InputClassificationTestCase {
	return []InputClassificationTestCase{
		// Dialog - third person
		{
			Name:             "Third person - character greets",
			RawInput:         "Magnus greets the innkeeper warmly",
			SceneName:        "The Rusty Anchor Tavern",
			SceneDescription: "A cozy tavern with a friendly atmosphere",
			ExpectedType:     "dialog",
			Description:      "Third person greeting NPC - dialog",
		},
		{
			Name:             "Third person - she asks",
			RawInput:         "She asks the merchant about his wares",
			SceneName:        "Market Square",
			SceneDescription: "A bustling marketplace with various vendors",
			ExpectedType:     "dialog",
			Description:      "Third person question to NPC - dialog",
		},
		{
			Name:             "Third person - character says",
			RawInput:         "Magnus says 'I'm looking for work, friend'",
			SceneName:        "Guild Hall",
			SceneDescription: "The local adventurer's guild, job postings on the wall",
			ExpectedType:     "dialog",
			Description:      "Third person direct speech - dialog",
		},
		// Clarification - third person
		{
			Name:             "Third person - character looks around",
			RawInput:         "Magnus looks around the room",
			SceneName:        "Study Chamber",
			SceneDescription: "A quiet study filled with books and papers",
			ExpectedType:     "clarification",
			Description:      "Third person basic observation - clarification",
		},
		{
			Name:             "Third person - she examines",
			RawInput:         "She examines the strange markings on the wall",
			SceneName:        "Ancient Ruins",
			SceneDescription: "Crumbling stone walls covered in mysterious symbols",
			ExpectedType:     "clarification",
			Description:      "Third person examining without opposition - clarification",
		},
		{
			Name:             "Third person - character thinks",
			RawInput:         "Magnus thinks about the old wizard's warning",
			SceneName:        "Forest Path",
			SceneDescription: "A quiet trail through peaceful woods",
			ExpectedType:     "narrative",
			Description:      "Third person internal reflection - narrative (no roll)",
		},
		// Action - third person
		{
			Name:             "Third person - character attacks",
			RawInput:         "Magnus attacks the goblin with his sword",
			SceneName:        "Forest Clearing",
			SceneDescription: "A clearing where goblins have ambushed you",
			ExpectedType:     "action",
			Description:      "Third person combat attack - action",
		},
		{
			Name:             "Third person - she sneaks",
			RawInput:         "She sneaks past the sleeping guard",
			SceneName:        "Castle Corridor",
			SceneDescription: "A dimly lit corridor with a guard dozing at his post",
			ExpectedType:     "action",
			Description:      "Third person stealth with opposition - action",
		},
		{
			Name:             "Third person - character climbs",
			RawInput:         "Magnus climbs the wall to escape the pursuers",
			SceneName:        "City Alley",
			SceneDescription: "A narrow alley, angry guards closing in behind",
			ExpectedType:     "action",
			Description:      "Third person physical challenge - action",
		},
		{
			Name:             "Third person - he jumps",
			RawInput:         "He leaps across the chasm",
			SceneName:        "Mountain Pass",
			SceneDescription: "A deep ravine blocks the mountain path",
			ExpectedType:     "action",
			Description:      "Third person overcoming obstacle - action",
		},
		{
			Name:             "Third person - character persuades",
			RawInput:         "Magnus tries to convince the guard to let him pass",
			SceneName:        "City Gate",
			SceneDescription: "A stern guard blocking entry to the noble quarter",
			ExpectedType:     "action",
			Description:      "Third person social challenge with opposition - action",
		},
	}
}

// InputClassificationResult stores the result of classification evaluation
type InputClassificationResult struct {
	TestCase   InputClassificationTestCase
	ActualType string
	Matches    bool
	Error      error
}

// InputClassificationSummary provides aggregate statistics
type InputClassificationSummary struct {
	TotalTests      int
	Matches         int
	ByExpectedType  map[string]*ClassificationTypeSummary
	MisclassifiedAs map[string]int
}

// ClassificationTypeSummary provides per-type statistics
type ClassificationTypeSummary struct {
	Total   int
	Correct int
}

// TestInputClassification_LLMEvaluation evaluates the input classification prompt
// Run with: go test -v -tags=llmeval ./test/llmeval/
// Requires AZURE_API_ENDPOINT and AZURE_API_KEY environment variables
func TestInputClassification_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	allTestCases := []struct {
		category string
		cases    []InputClassificationTestCase
	}{
		{"Dialog", getDialogTestCases()},
		{"Clarification", getClarificationTestCases()},
		{"Action", getActionTestCases()},
		{"ThirdPerson", getThirdPersonClassificationTestCases()},
	}

	var results []InputClassificationResult
	summary := InputClassificationSummary{
		ByExpectedType:  make(map[string]*ClassificationTypeSummary),
		MisclassifiedAs: make(map[string]int),
	}

	for _, expectedType := range []string{"dialog", "clarification", "narrative", "action"} {
		summary.ByExpectedType[expectedType] = &ClassificationTypeSummary{}
	}

	for _, category := range allTestCases {
		t.Run(category.category, func(t *testing.T) {
			for _, tc := range category.cases {
				t.Run(tc.Name, func(t *testing.T) {
					result := evaluateInputClassification(ctx, client, tc)
					results = append(results, result)

					summary.TotalTests++
					if result.Matches {
						summary.Matches++
					} else {
						summary.MisclassifiedAs[result.ActualType]++
					}

					typeSummary := summary.ByExpectedType[tc.ExpectedType]
					typeSummary.Total++
					if result.Matches {
						typeSummary.Correct++
					}

					if result.Error != nil {
						t.Errorf("Error: %v", result.Error)
						return
					}

					if verboseLogging {
						status := "✓"
						if !result.Matches {
							status = "✗"
						}
						t.Logf("%s Classification: expected=%s, got=%s", status, tc.ExpectedType, result.ActualType)
						t.Logf("  Input: %s", tc.RawInput)
						t.Logf("  Scene: %s", tc.SceneDescription)
					}

					assert.Equal(t, tc.ExpectedType, result.ActualType,
						"Classification mismatch for '%s'", tc.RawInput)
				})
			}
		})
	}

	// Print summary
	t.Log("\n========== INPUT CLASSIFICATION SUMMARY ==========")
	t.Logf("Total Tests: %d", summary.TotalTests)
	t.Logf("Accuracy: %d/%d (%.1f%%)",
		summary.Matches, summary.TotalTests,
		float64(summary.Matches)*100/float64(summary.TotalTests))

	t.Log("\n--- By Expected Type ---")
	for classType, ts := range summary.ByExpectedType {
		if ts.Total > 0 {
			t.Logf("%s: %d/%d correct (%.1f%%)",
				classType, ts.Correct, ts.Total,
				float64(ts.Correct)*100/float64(ts.Total))
		}
	}

	t.Log("\n--- Misclassification Breakdown ---")
	for classType, count := range summary.MisclassifiedAs {
		t.Logf("Misclassified as %s: %d times", classType, count)
	}

	// Print failed cases
	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: '%s'", r.TestCase.RawInput)
			t.Logf("      Expected: %s, Got: %s", r.TestCase.ExpectedType, r.ActualType)
			t.Logf("      Scene: %s", r.TestCase.SceneDescription)
			t.Logf("      Why expected: %s", r.TestCase.Description)
		}
	}
}

// evaluateInputClassification runs a single classification test
func evaluateInputClassification(ctx context.Context, client llm.LLMClient, tc InputClassificationTestCase) InputClassificationResult {
	// Create a test scene
	testScene := scene.NewScene("test-scene", tc.SceneName, tc.SceneDescription)

	// Prepare template data
	data := promptpkg.InputClassificationData{
		Scene:       testScene,
		PlayerInput: tc.RawInput,
	}

	prompt, err := promptpkg.RenderInputClassification(data)
	if err != nil {
		return InputClassificationResult{
			TestCase: tc,
			Error:    err,
		}
	}

	resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   10,
		Temperature: 0.1,
	})
	if err != nil {
		return InputClassificationResult{
			TestCase: tc,
			Error:    err,
		}
	}

	if len(resp.Choices) == 0 {
		return InputClassificationResult{
			TestCase: tc,
			Error:    fmt.Errorf("no response from LLM"),
		}
	}

	classification := strings.ToLower(strings.TrimSpace(resp.Choices[0].Message.Content))

	return InputClassificationResult{
		TestCase:   tc,
		ActualType: classification,
		Matches:    classification == tc.ExpectedType,
	}
}

// TestInputClassification_EdgeCases focuses on ambiguous inputs
func TestInputClassification_EdgeCases(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	edgeCases := []InputClassificationTestCase{
		// These are tricky - context determines classification
		{
			Name:             "Notice with no pressure",
			RawInput:         "I check if anyone is watching",
			SceneName:        "Quiet Library",
			SceneDescription: "A peaceful library with a few scholars reading",
			ExpectedType:     "clarification",
			Description:      "Basic awareness in safe environment - clarification",
		},
		{
			Name:             "Notice with danger",
			RawInput:         "I check if anyone is following me",
			SceneName:        "Dark Alley",
			SceneDescription: "A shadowy alley after you've stolen the artifact",
			ExpectedType:     "action",
			Description:      "Awareness when being pursued - action with consequences",
		},
		{
			Name:             "Open door - no obstacle",
			RawInput:         "I open the door",
			SceneName:        "Inn Hallway",
			SceneDescription: "The hallway outside your room at the inn",
			ExpectedType:     "narrative",
			Description:      "Simple mundane action - narrative (no roll)",
		},
		{
			Name:             "Open door - obstacle",
			RawInput:         "I try to force the door open",
			SceneName:        "Burning Building",
			SceneDescription: "Trapped in a burning building, the door is jammed",
			ExpectedType:     "action",
			Description:      "Overcoming obstacle with consequences - action",
		},
		{
			Name:             "Persuasion vs small talk",
			RawInput:         "I chat with the merchant about the weather",
			SceneName:        "Market Stall",
			SceneDescription: "A friendly merchant selling trinkets",
			ExpectedType:     "dialog",
			Description:      "Casual conversation without persuasion intent - dialog",
		},
		{
			Name:             "Persuasion with stakes",
			RawInput:         "I try to convince the merchant to lower his prices",
			SceneName:        "Market Stall",
			SceneDescription: "A shrewd merchant who drives hard bargains",
			ExpectedType:     "action",
			Description:      "Persuasion with opposition - action",
		},
	}

	if verboseLogging {
		t.Log("Testing edge cases where context determines classification")
		t.Log(strings.Repeat("=", 60))
	}

	for _, tc := range edgeCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateInputClassification(ctx, client, tc)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if verboseLogging {
				status := "✓ PASS"
				if !result.Matches {
					status = "✗ FAIL"
				}
				t.Logf("%s: Expected %s, Got %s", status, tc.ExpectedType, result.ActualType)
				t.Logf("  Input: %s", tc.RawInput)
				t.Logf("  Scene: %s", tc.SceneDescription)
				t.Logf("  Why: %s", tc.Description)
			}

			assert.Equal(t, tc.ExpectedType, result.ActualType,
				"Classification mismatch for '%s'", tc.RawInput)
		})
	}
}

// TestInputClassification_MundaneMovement verifies that ordinary movement
// is classified as clarification (no roll needed), not action.
// This addresses over-classification of mundane actions as requiring dice rolls.
// Run with: go test -v -tags=llmeval -run TestInputClassification_MundaneMovement ./test/llmeval/
func TestInputClassification_MundaneMovement(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	testCases := []InputClassificationTestCase{
		// === Mundane movement - should be NARRATIVE or CLARIFICATION (no roll) ===
		{
			Name:             "Push open door and stride in",
			RawInput:         "Lyra pushes open the door and strides inside",
			SceneName:        "Tavern Entrance",
			SceneDescription: "Standing outside a tavern, ready to enter",
			ExpectedType:     "narrative",
			Description:      "Confident entry through a door is mundane - just narrate it",
		},
		{
			Name:             "Walk through doorway",
			RawInput:         "I walk through the doorway into the next room",
			SceneName:        "Abandoned Building",
			SceneDescription: "Exploring an abandoned building, currently alone",
			ExpectedType:     "narrative",
			Description:      "Simple walking through a doorway - just narrate it",
		},
		{
			Name:             "Walk down street",
			RawInput:         "I walk down the street toward the market",
			SceneName:        "City Street",
			SceneDescription: "A busy city street during the day",
			ExpectedType:     "narrative",
			Description:      "Walking down a public street is mundane - just narrate it",
		},
		{
			Name:             "Enter shop",
			RawInput:         "I enter the blacksmith's shop",
			SceneName:        "Market District",
			SceneDescription: "Visiting the local blacksmith to buy a sword",
			ExpectedType:     "narrative",
			Description:      "Entering a shop as a customer - just narrate it",
		},
		{
			Name:             "Walk into mist",
			RawInput:         "Lyra walks off into the mist",
			SceneName:        "Moorland",
			SceneDescription: "A foggy morning on the moors",
			ExpectedType:     "narrative",
			Description:      "Walking into atmospheric conditions - just narrate it",
		},
		{
			Name:             "Careful entry",
			RawInput:         "Lyra carefully opens the door and steps inside",
			SceneName:        "Unknown Room",
			SceneDescription: "Entering a room in a dungeon",
			ExpectedType:     "narrative",
			Description:      "Being careful is cautious, not stealthy - just narrate it",
		},
		{
			Name:             "Careful movement with traps",
			RawInput:         "I carefully make my way across the floor",
			SceneName:        "Trapped Corridor",
			SceneDescription: "A corridor in the dungeon, pressure plates and tripwires are visible",
			ExpectedType:     "action",
			Description:      "Careful movement when traps are present requires Athletics roll",
		},
		{
			Name:             "March into throne room",
			RawInput:         "Magnus marches into the throne room to address the king",
			SceneName:        "Royal Palace",
			SceneDescription: "Arriving at the royal court for an audience",
			ExpectedType:     "narrative",
			Description:      "Bold entry for an audience - just narrate it",
		},
		// === Stealth movement - should be ACTION (requires roll) ===
		{
			Name:             "Sneak through door",
			RawInput:         "I sneak through the doorway, trying not to be seen",
			SceneName:        "Guard Barracks",
			SceneDescription: "Infiltrating the guard barracks at night",
			ExpectedType:     "action",
			Description:      "Explicitly sneaking to avoid detection requires a roll",
		},
		{
			Name:             "Slip inside unnoticed",
			RawInput:         "I slip inside while the guard's back is turned",
			SceneName:        "Restricted Area",
			SceneDescription: "Trying to enter the restricted area without being caught",
			ExpectedType:     "action",
			Description:      "Slipping past a guard requires a stealth roll",
		},
		{
			Name:             "Creep into room",
			RawInput:         "I creep into the darkened room",
			SceneName:        "Sleeping Quarters",
			SceneDescription: "Entering a room where someone might be sleeping",
			ExpectedType:     "action",
			Description:      "Creeping implies stealth intent - requires roll",
		},
		{
			Name:             "Tiptoe past guard",
			RawInput:         "I tiptoe past the sleeping guard",
			SceneName:        "Guard Post",
			SceneDescription: "A guard has fallen asleep at his post",
			ExpectedType:     "action",
			Description:      "Tiptoeing past someone requires a stealth roll",
		},
		{
			Name:             "Evade into mist",
			RawInput:         "Lyra tries to lose her pursuers by slipping into the mist",
			SceneName:        "Moorland Chase",
			SceneDescription: "Guards are chasing you across the foggy moors",
			ExpectedType:     "action",
			Description:      "Actively evading pursuers requires a stealth roll",
		},
		{
			Name:             "Silently enter shadows",
			RawInput:         "I silently enter the chamber, keeping to the shadows",
			SceneName:        "Wizard Tower",
			SceneDescription: "Infiltrating the wizard's tower",
			ExpectedType:     "action",
			Description:      "Silent shadow movement is explicit stealth - requires roll",
		},
	}

	if verboseLogging {
		t.Log("Testing mundane movement vs stealth classification")
		t.Log(strings.Repeat("=", 60))
	}

	mundaneResults := struct{ total, correct int }{}
	stealthResults := struct{ total, correct int }{}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateInputClassification(ctx, client, tc)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			// Track results by expected type
			if tc.ExpectedType == "narrative" {
				mundaneResults.total++
				if result.Matches {
					mundaneResults.correct++
				}
			} else if tc.ExpectedType == "action" {
				stealthResults.total++
				if result.Matches {
					stealthResults.correct++
				}
			}

			if verboseLogging {
				status := "✓ PASS"
				if !result.Matches {
					status = "✗ FAIL"
				}
				t.Logf("%s: Expected %s, Got %s", status, tc.ExpectedType, result.ActualType)
				t.Logf("  Input: %s", tc.RawInput)
				t.Logf("  Scene: %s", tc.SceneDescription)
				t.Logf("  Why: %s", tc.Description)
			}

			assert.Equal(t, tc.ExpectedType, result.ActualType,
				"Classification mismatch for '%s'. %s", tc.RawInput, tc.Description)
		})
	}

	// Print summary
	t.Log("\n========== MUNDANE VS STEALTH CLASSIFICATION SUMMARY ==========")
	if mundaneResults.total > 0 {
		t.Logf("Mundane movement (narrative): %d/%d (%.1f%%)",
			mundaneResults.correct, mundaneResults.total,
			float64(mundaneResults.correct)*100/float64(mundaneResults.total))
	}
	if stealthResults.total > 0 {
		t.Logf("Stealth movement (action): %d/%d (%.1f%%)",
			stealthResults.correct, stealthResults.total,
			float64(stealthResults.correct)*100/float64(stealthResults.total))
	}
}
