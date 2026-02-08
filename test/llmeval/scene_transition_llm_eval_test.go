//go:build llmeval

package llmeval_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SceneTransitionTestCase represents a test case for scene transition detection
type SceneTransitionTestCase struct {
	Name                   string
	PlayerInput            string
	SceneName              string
	SceneDescription       string
	OtherCharacters        []*character.Character
	ExpectTransitionMarker bool   // Should the LLM add a SCENE_TRANSITION marker?
	ExpectedHintContains   string // Optional: check if the hint contains this text
	Description            string // Human-readable description of why this should/shouldn't trigger transition
}

// getExitTestCases returns inputs that SHOULD trigger scene transition markers
func getExitTestCases() []SceneTransitionTestCase {
	bartender := character.NewCharacter("bartender", "Maggie Two-Rivers")
	bartender.Aspects.HighConcept = "Weathered Saloon Owner"

	return []SceneTransitionTestCase{
		{
			Name:                   "Walk out of saloon",
			PlayerInput:            "Jesse turns and walks out of the saloon",
			SceneName:              "The Dusty Spur Saloon",
			SceneDescription:       "A dimly lit saloon with swinging doors, a long bar, and the smell of whiskey",
			OtherCharacters:        []*character.Character{bartender},
			ExpectTransitionMarker: true,
			Description:            "Explicitly walking out should trigger scene transition",
		},
		{
			Name:                   "Saunter out with dialog",
			PlayerInput:            "\"Thanks for the drink.\" Jesse tips his hat and saunters out.",
			SceneName:              "The Dusty Spur Saloon",
			SceneDescription:       "A dimly lit saloon with swinging doors, a long bar, and the smell of whiskey",
			OtherCharacters:        []*character.Character{bartender},
			ExpectTransitionMarker: true,
			Description:            "Farewell dialog combined with leaving should trigger transition",
		},
		{
			Name:                   "Leave the building",
			PlayerInput:            "I leave the building",
			SceneName:              "Abandoned Warehouse",
			SceneDescription:       "A dusty warehouse with crates stacked against the walls",
			ExpectTransitionMarker: true,
			Description:            "Explicit 'leave' should trigger transition",
		},
		{
			Name:                   "Exit the tavern",
			PlayerInput:            "I exit the tavern and head outside",
			SceneName:              "The Rusty Anchor",
			SceneDescription:       "A busy dockside tavern filled with sailors",
			ExpectTransitionMarker: true,
			Description:            "Exit command should trigger transition",
		},
		{
			Name:                   "Ride away",
			PlayerInput:            "I mount my horse and ride off toward the canyon",
			SceneName:              "Town Square",
			SceneDescription:       "The central square of Redemption Gulch",
			ExpectTransitionMarker: true,
			ExpectedHintContains:   "canyon",
			Description:            "Riding away from location should trigger transition with destination hint",
		},
		{
			Name:                   "Step outside",
			PlayerInput:            "I step outside to get some fresh air",
			SceneName:              "Smoky Tavern",
			SceneDescription:       "The air is thick with pipe smoke inside this cramped tavern",
			ExpectTransitionMarker: true,
			Description:            "Stepping outside is leaving the location",
		},
		{
			Name:                   "Third person exit",
			PlayerInput:            "Magnus pushes through the crowd and exits through the back door",
			SceneName:              "Crowded Market",
			SceneDescription:       "A busy marketplace with vendors and shoppers everywhere",
			ExpectTransitionMarker: true,
			Description:            "Third person explicit exit should trigger transition",
		},
		{
			Name:                   "Depart the scene",
			PlayerInput:            "Having learned what I came for, I depart",
			SceneName:              "Scholar's Study",
			SceneDescription:       "A quiet study filled with books and scrolls",
			ExpectTransitionMarker: true,
			Description:            "Explicit departure should trigger transition",
		},
	}
}

// getNoExitTestCases returns inputs that should NOT trigger scene transition markers
func getNoExitTestCases() []SceneTransitionTestCase {
	bartender := character.NewCharacter("bartender", "Maggie Two-Rivers")
	bartender.Aspects.HighConcept = "Weathered Saloon Owner"

	return []SceneTransitionTestCase{
		{
			Name:                   "Move within location",
			PlayerInput:            "I walk over to the window",
			SceneName:              "Inn Room",
			SceneDescription:       "A cozy room at the local inn",
			ExpectTransitionMarker: false,
			Description:            "Moving within a location is not leaving",
		},
		{
			Name:                   "Approach NPC",
			PlayerInput:            "I walk up to the bar and sit down",
			SceneName:              "The Dusty Spur Saloon",
			SceneDescription:       "A dimly lit saloon with a long bar",
			OtherCharacters:        []*character.Character{bartender},
			ExpectTransitionMarker: false,
			Description:            "Approaching something within the scene is not leaving",
		},
		{
			Name:                   "Ask about leaving",
			PlayerInput:            "Is there a back exit from here?",
			SceneName:              "Warehouse",
			SceneDescription:       "A dusty warehouse with crates everywhere",
			ExpectTransitionMarker: false,
			Description:            "Asking about exits is not actually leaving",
		},
		{
			Name:                   "Look outside",
			PlayerInput:            "I look out the window at the street",
			SceneName:              "Inn Room",
			SceneDescription:       "A room with a window overlooking the street",
			ExpectTransitionMarker: false,
			Description:            "Looking outside is not leaving",
		},
		{
			Name:                   "Order a drink",
			PlayerInput:            "I order a whiskey",
			SceneName:              "The Dusty Spur Saloon",
			SceneDescription:       "A dimly lit saloon",
			OtherCharacters:        []*character.Character{bartender},
			ExpectTransitionMarker: false,
			Description:            "Normal interaction should not trigger transition",
		},
		{
			Name:                   "Dialog only",
			PlayerInput:            "\"Tell me about the Cortez Gang\"",
			SceneName:              "The Dusty Spur Saloon",
			SceneDescription:       "A dimly lit saloon",
			OtherCharacters:        []*character.Character{bartender},
			ExpectTransitionMarker: false,
			Description:            "Dialog without movement should not trigger transition",
		},
		{
			Name:                   "Pace around",
			PlayerInput:            "I pace around the room, thinking",
			SceneName:              "Study Chamber",
			SceneDescription:       "A scholarly study with books",
			ExpectTransitionMarker: false,
			Description:            "Moving within a room is not leaving",
		},
		{
			Name:                   "Examine door",
			PlayerInput:            "I examine the exit door",
			SceneName:              "Warehouse",
			SceneDescription:       "A dusty warehouse",
			ExpectTransitionMarker: false,
			Description:            "Examining the exit is not using it",
		},
		{
			Name:                   "Consider leaving",
			PlayerInput:            "I consider whether I should leave",
			SceneName:              "Tavern",
			SceneDescription:       "A busy tavern",
			ExpectTransitionMarker: false,
			Description:            "Thinking about leaving is not actually leaving",
		},
	}
}

// SceneTransitionResult stores the result of a scene transition evaluation
type SceneTransitionResult struct {
	TestCase      SceneTransitionTestCase
	Response      string
	HasMarker     bool
	ExtractedHint string
	Matches       bool
	Error         error
}

// TestSceneTransition_LLMEvaluation evaluates whether the LLM correctly adds scene transition markers
// Run with: go test -v -tags=llmeval ./test/llmeval/ -run TestSceneTransition
// Requires AZURE_API_ENDPOINT and AZURE_API_KEY environment variables
func TestSceneTransition_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	verboseLogging := os.Getenv("VERBOSE") == "1"

	allTestCases := []struct {
		category string
		cases    []SceneTransitionTestCase
	}{
		{"ShouldExit", getExitTestCases()},
		{"ShouldNotExit", getNoExitTestCases()},
	}

	var results []SceneTransitionResult

	exitResults := struct{ total, correct int }{}
	noExitResults := struct{ total, correct int }{}

	for _, category := range allTestCases {
		t.Run(category.category, func(t *testing.T) {
			for _, tc := range category.cases {
				t.Run(tc.Name, func(t *testing.T) {
					result := evaluateSceneTransition(ctx, client, tc)
					results = append(results, result)

					if result.Error != nil {
						t.Errorf("Error: %v", result.Error)
						return
					}

					// Track results by expected outcome
					if tc.ExpectTransitionMarker {
						exitResults.total++
						if result.Matches {
							exitResults.correct++
						}
					} else {
						noExitResults.total++
						if result.Matches {
							noExitResults.correct++
						}
					}

					if verboseLogging {
						status := "✓ PASS"
						if !result.Matches {
							status = "✗ FAIL"
						}
						t.Logf("%s: Expected marker=%v, Got marker=%v", status, tc.ExpectTransitionMarker, result.HasMarker)
						t.Logf("  Input: %s", tc.PlayerInput)
						t.Logf("  Scene: %s", tc.SceneDescription)
						if result.HasMarker {
							t.Logf("  Hint: %s", result.ExtractedHint)
						}
						t.Logf("  Why: %s", tc.Description)
						if !result.Matches {
							t.Logf("  Response: %s", truncateResponse(result.Response, 200))
						}
					}

					assert.Equal(t, tc.ExpectTransitionMarker, result.HasMarker,
						"Transition marker mismatch for '%s'. %s\nResponse: %s",
						tc.PlayerInput, tc.Description, truncateResponse(result.Response, 300))

					// If we expected a marker with specific hint content, verify it
					if tc.ExpectTransitionMarker && tc.ExpectedHintContains != "" && result.HasMarker {
						assert.Contains(t, strings.ToLower(result.ExtractedHint), strings.ToLower(tc.ExpectedHintContains),
							"Hint should contain '%s', got '%s'", tc.ExpectedHintContains, result.ExtractedHint)
					}
				})
			}
		})
	}

	// Print summary
	t.Log("\n========== SCENE TRANSITION DETECTION SUMMARY ==========")
	if exitResults.total > 0 {
		t.Logf("Exit cases (should have marker): %d/%d (%.1f%%)",
			exitResults.correct, exitResults.total,
			float64(exitResults.correct)*100/float64(exitResults.total))
	}
	if noExitResults.total > 0 {
		t.Logf("Non-exit cases (should NOT have marker): %d/%d (%.1f%%)",
			noExitResults.correct, noExitResults.total,
			float64(noExitResults.correct)*100/float64(noExitResults.total))
	}

	totalCorrect := exitResults.correct + noExitResults.correct
	totalTests := exitResults.total + noExitResults.total
	if totalTests > 0 {
		t.Logf("Overall accuracy: %d/%d (%.1f%%)",
			totalCorrect, totalTests,
			float64(totalCorrect)*100/float64(totalTests))
	}

	// Print failed cases
	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: '%s'", r.TestCase.PlayerInput)
			t.Logf("      Expected marker: %v, Got marker: %v", r.TestCase.ExpectTransitionMarker, r.HasMarker)
			t.Logf("      Scene: %s", r.TestCase.SceneDescription)
			t.Logf("      Why: %s", r.TestCase.Description)
			t.Logf("      Response: %s", truncateResponse(r.Response, 200))
		}
	}
}

// evaluateSceneTransition runs a single scene transition test
func evaluateSceneTransition(ctx context.Context, client llm.LLMClient, tc SceneTransitionTestCase) SceneTransitionResult {
	// Create test scene
	testScene := scene.NewScene("test-scene", tc.SceneName, tc.SceneDescription)

	// Create a test player character
	player := character.NewCharacter("player1", "Test Character")
	player.Aspects.HighConcept = "Wandering Stranger"

	// Build character context
	charContext := buildCharacterContext(player)

	// Build aspects context
	aspectsContext := buildAspectsContext(testScene, player, tc.OtherCharacters)

	// Prepare template data
	data := promptpkg.SceneResponseData{
		Scene:               testScene,
		CharacterContext:    charContext,
		AspectsContext:      aspectsContext,
		ConversationContext: "No previous conversation.",
		PlayerInput:         tc.PlayerInput,
		InteractionType:     "dialog",
		OtherCharacters:     tc.OtherCharacters,
	}

	prompt, err := promptpkg.RenderSceneResponse(data)
	if err != nil {
		return SceneTransitionResult{
			TestCase: tc,
			Error:    err,
		}
	}

	resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   500,
		Temperature: 0.3, // Low temperature for more consistent behavior
	})
	if err != nil {
		return SceneTransitionResult{
			TestCase: tc,
			Error:    err,
		}
	}

	if len(resp.Choices) == 0 {
		return SceneTransitionResult{
			TestCase: tc,
			Error:    err,
		}
	}

	response := resp.Choices[0].Message.Content

	// Check for scene transition marker using production parser
	transition, _ := promptpkg.ParseSceneTransitionMarker(response)
	hasMarker := transition != nil
	hint := ""
	if transition != nil {
		hint = transition.Hint
	}

	return SceneTransitionResult{
		TestCase:      tc,
		Response:      response,
		HasMarker:     hasMarker,
		ExtractedHint: hint,
		Matches:       hasMarker == tc.ExpectTransitionMarker,
	}
}

// buildCharacterContext creates a character context string for the template
func buildCharacterContext(player *character.Character) string {
	var sb strings.Builder
	sb.WriteString("Name: ")
	sb.WriteString(player.Name)
	sb.WriteString("\n")
	if player.Aspects.HighConcept != "" {
		sb.WriteString("High Concept: ")
		sb.WriteString(player.Aspects.HighConcept)
		sb.WriteString("\n")
	}
	if player.Aspects.Trouble != "" {
		sb.WriteString("Trouble: ")
		sb.WriteString(player.Aspects.Trouble)
		sb.WriteString("\n")
	}
	return sb.String()
}

// buildAspectsContext creates an aspects context string for the template
func buildAspectsContext(s *scene.Scene, player *character.Character, others []*character.Character) string {
	var sb strings.Builder
	sb.WriteString("Scene Aspects:\n")
	for _, aspect := range s.SituationAspects {
		sb.WriteString("  - ")
		sb.WriteString(aspect.Aspect)
		sb.WriteString("\n")
	}
	sb.WriteString("\nCharacter Aspects:\n")
	if player.Aspects.HighConcept != "" {
		sb.WriteString("  - ")
		sb.WriteString(player.Aspects.HighConcept)
		sb.WriteString(" (")
		sb.WriteString(player.Name)
		sb.WriteString(")\n")
	}
	for _, other := range others {
		if other.Aspects.HighConcept != "" {
			sb.WriteString("  - ")
			sb.WriteString(other.Aspects.HighConcept)
			sb.WriteString(" (")
			sb.WriteString(other.Name)
			sb.WriteString(")\n")
		}
	}
	return sb.String()
}

// truncateResponse truncates a response string for display
func truncateResponse(s string, maxLen int) string {
	// Remove newlines for cleaner display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
