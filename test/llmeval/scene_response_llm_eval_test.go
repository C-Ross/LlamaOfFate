//go:build llmeval

package llmeval_test

import (
	"context"
	"os"
	"regexp"
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

// Regex patterns for detecting "choose your own adventure" style options
var (
	// Matches "Jesse must now decide:" or similar prompts
	decisionPromptRegex = regexp.MustCompile(`(?i)(must now decide|must decide|you must choose|you can choose|what will .* do\?|what do you do\?)`)
	// Matches bullet point options like "- Option 1" or "• Option"
	bulletOptionsRegex = regexp.MustCompile(`(?m)^[\s]*[-•*]\s+(Use|Take|Press|Attempt|Keep|Try|Draw|Reach|Spin|Slowly)`)
	// Matches numbered options like "1. Option" or "1) Option"
	numberedOptionsRegex = regexp.MustCompile(`(?m)^[\s]*\d+[.)]\s+\w+`)
)

// SceneResponseTestCase for testing scene response behaviors
type SceneResponseTestCase struct {
	Name                 string
	PlayerInput          string
	SceneName            string
	SceneDescription     string
	OtherCharacters      []*character.Character
	ConversationContext  string
	Description          string
	CheckNoOptions       bool   // Verify response doesn't contain options
	CheckTransition      bool   // Verify scene transition marker appears
	ExpectedHintContains string // For transition tests
}

// getNoOptionsTestCases returns cases where the LLM should NOT present options
func getNoOptionsTestCases() []SceneResponseTestCase {
	blackJack := character.NewCharacter("blackjack", "Black Jack McCoy")
	blackJack.Aspects.HighConcept = "Dangerous Outlaw with a Quick Draw"
	blackJack.Aspects.Trouble = "Wanted Dead or Alive"

	bartender := character.NewCharacter("bartender", "Maggie Two-Rivers")
	bartender.Aspects.HighConcept = "Weathered Saloon Owner"

	return []SceneResponseTestCase{
		{
			Name:                "Tense confrontation dialog",
			PlayerInput:         `Jesse cooly eyes Black Jack "Howdy there partner. Was looking to talk, thought we might have some similar interests."`,
			SceneName:           "Windmill on the Outskirts",
			SceneDescription:    "An old abandoned windmill on the outskirts of town. The creaking blades announce any arrival. Black Jack McCoy stands near the entrance, hand near his gun.",
			OtherCharacters:     []*character.Character{blackJack},
			ConversationContext: `GM: As Jesse approaches the old windmill, the creaking of its weathered blades echoes through the stillness. Black Jack McCoy stands near the windmill's entrance, eyeing Jesse with a mixture of curiosity and wariness.`,
			Description:         "Tense NPC confrontation should describe reaction without presenting player options",
			CheckNoOptions:      true,
		},
		{
			Name:                "Asking for information from hostile NPC",
			PlayerInput:         `Jesse nods "I'm looking for the Cortez gang, I have some unfriendly business with them. I hear you might know where I can find them."`,
			SceneName:           "Windmill on the Outskirts",
			SceneDescription:    "An old abandoned windmill on the outskirts of town. Black Jack McCoy stands with his hand on his gun.",
			OtherCharacters:     []*character.Character{blackJack},
			ConversationContext: `GM: Black Jack's eyes narrow as he takes a step forward. "Talk's free, but I'm listening with one ear open and my hand on my gun."`,
			Description:         "Information request to hostile NPC should get NPC response, not player options",
			CheckNoOptions:      true,
		},
		{
			Name:                "Casual bar conversation",
			PlayerInput:         `Jesse sips his drink and looks at Maggie. "I bet you know everything that goes on around here."`,
			SceneName:           "The Dusty Spur Saloon",
			SceneDescription:    "A dimly lit saloon with swinging doors, a long bar, and the smell of whiskey",
			OtherCharacters:     []*character.Character{bartender},
			ConversationContext: `GM: Maggie slides the whiskey down the bar to Jesse, her gaze lingering on his face.`,
			Description:         "Normal bar conversation should continue naturally without options",
			CheckNoOptions:      true,
		},
		{
			Name:                "Negotiation with dangerous character",
			PlayerInput:         `Jesse laughs wryly. "I may be both, but I'll take my chances. So you going to help me?" He meets Jack's eyes and stands his ground.`,
			SceneName:           "Windmill on the Outskirts",
			SceneDescription:    "An old abandoned windmill. Tension fills the air as Black Jack's hand rests on his pistol.",
			OtherCharacters:     []*character.Character{blackJack},
			ConversationContext: `GM: Black Jack's eyes flicker. "Cortez gang, huh? You're either very brave or very stupid."`,
			Description:         "Negotiation with dangerous NPC should continue dialog without presenting choices",
			CheckNoOptions:      true,
		},
		{
			Name:                "Brief farewell before leaving",
			PlayerInput:         `Jesse nods. "A man don't live forever. So, gonna point me in the right direction?"`,
			SceneName:           "Windmill on the Outskirts",
			SceneDescription:    "An old abandoned windmill on the outskirts of Redemption Gulch.",
			OtherCharacters:     []*character.Character{blackJack},
			ConversationContext: `GM: Black Jack says, "I might know a thing or two about the Cortez gang, but I'm not doing it out of the kindness of my heart."`,
			Description:         "Final question before potential departure should get direct answer",
			CheckNoOptions:      true,
		},
	}
}

// getTransitionAfterLeaveTestCases returns cases where player leaves and transition should occur
func getTransitionAfterLeaveTestCases() []SceneResponseTestCase {
	blackJack := character.NewCharacter("blackjack", "Black Jack McCoy")
	blackJack.Aspects.HighConcept = "Dangerous Outlaw with a Quick Draw"

	return []SceneResponseTestCase{
		{
			Name:             "Leave after getting information",
			PlayerInput:      "Jesse shakes his head and walks to his horse, mounts and rides off towards the mine.",
			SceneName:        "Windmill on the Outskirts",
			SceneDescription: "An old abandoned windmill on the outskirts of Redemption Gulch.",
			OtherCharacters:  []*character.Character{blackJack},
			ConversationContext: `GM: Black Jack glares at Jesse. "Alright, I'll give you a direction. They're holed up in the old abandoned mine on the other side of Redemption Gulch."
Player: Jesse tips his hat. "Thank you."`,
			Description:          "Player leaving after conversation should trigger scene transition",
			CheckTransition:      true,
			ExpectedHintContains: "mine",
		},
		{
			Name:                 "Explicit departure to known destination",
			PlayerInput:          "Jesse rides directly to the mine.",
			SceneName:            "Windmill on the Outskirts",
			SceneDescription:     "An old abandoned windmill. Black Jack watches from the entrance.",
			OtherCharacters:      []*character.Character{blackJack},
			ConversationContext:  `GM: Black Jack watches as Jesse turns toward his horse. "Reckon we'll see how long you last."`,
			Description:          "Explicit destination statement should trigger transition to that location",
			CheckTransition:      true,
			ExpectedHintContains: "mine",
		},
		{
			Name:                 "Ride away from windmill",
			PlayerInput:          "Jesse rides away from the windmill, heading straight to the mine.",
			SceneName:            "Windmill on the Outskirts",
			SceneDescription:     "An old abandoned windmill on the outskirts of town.",
			OtherCharacters:      []*character.Character{blackJack},
			ConversationContext:  `GM: Black Jack just told Jesse the Cortez gang is at the abandoned mine.`,
			Description:          "Riding away should trigger transition",
			CheckTransition:      true,
			ExpectedHintContains: "mine",
		},
	}
}

// SceneResponseResult stores the evaluation result
type SceneResponseResult struct {
	TestCase       SceneResponseTestCase
	Response       string
	HasOptions     bool
	OptionsFound   string
	HasTransition  bool
	TransitionHint string
	Matches        bool
	Error          error
}

// TestSceneResponse_NoOptions_LLMEvaluation tests that LLM doesn't present CYOA-style options
func TestSceneResponse_NoOptions_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	verboseLogging := os.Getenv("VERBOSE") == "1"
	testCases := getNoOptionsTestCases()

	var results []SceneResponseResult
	correct := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateSceneResponseBehavior(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.Matches {
				correct++
			}

			if verboseLogging || !result.Matches {
				status := "✓ PASS"
				if !result.Matches {
					status = "✗ FAIL"
				}
				t.Logf("%s: HasOptions=%v", status, result.HasOptions)
				t.Logf("  Input: %s", tc.PlayerInput)
				t.Logf("  Why: %s", tc.Description)
				if result.HasOptions {
					t.Logf("  Options found: %s", result.OptionsFound)
				}
				t.Logf("  Response: %s", truncateResponseText(result.Response, 300))
			}

			assert.False(t, result.HasOptions,
				"Response should NOT contain options/choices. Found: %s\nResponse: %s",
				result.OptionsFound, truncateResponseText(result.Response, 400))
		})
	}

	// Summary
	t.Log("\n========== NO OPTIONS TEST SUMMARY ==========")
	t.Logf("Cases without options: %d/%d (%.1f%%)",
		correct, len(testCases), float64(correct)*100/float64(len(testCases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: '%s'", r.TestCase.Name)
			t.Logf("      Input: %s", truncateResponseText(r.TestCase.PlayerInput, 80))
			t.Logf("      Options found: %s", r.OptionsFound)
		}
	}
}

// TestSceneResponse_TransitionOnLeave_LLMEvaluation tests that scene transitions happen when player leaves
func TestSceneResponse_TransitionOnLeave_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	verboseLogging := os.Getenv("VERBOSE") == "1"
	testCases := getTransitionAfterLeaveTestCases()

	var results []SceneResponseResult
	correct := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateSceneResponseBehavior(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.Matches {
				correct++
			}

			if verboseLogging || !result.Matches {
				status := "✓ PASS"
				if !result.Matches {
					status = "✗ FAIL"
				}
				t.Logf("%s: HasTransition=%v", status, result.HasTransition)
				t.Logf("  Input: %s", tc.PlayerInput)
				t.Logf("  Why: %s", tc.Description)
				if result.HasTransition {
					t.Logf("  Transition hint: %s", result.TransitionHint)
				}
				t.Logf("  Response: %s", truncateResponseText(result.Response, 300))
			}

			assert.True(t, result.HasTransition,
				"Response should have SCENE_TRANSITION marker when player leaves.\nResponse: %s",
				truncateResponseText(result.Response, 400))

			// Check hint content if specified
			if tc.ExpectedHintContains != "" && result.HasTransition {
				assert.Contains(t, strings.ToLower(result.TransitionHint), strings.ToLower(tc.ExpectedHintContains),
					"Transition hint should contain '%s', got '%s'", tc.ExpectedHintContains, result.TransitionHint)
			}
		})
	}

	// Summary
	t.Log("\n========== TRANSITION ON LEAVE TEST SUMMARY ==========")
	t.Logf("Cases with transition: %d/%d (%.1f%%)",
		correct, len(testCases), float64(correct)*100/float64(len(testCases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: '%s'", r.TestCase.Name)
			t.Logf("      Input: %s", truncateResponseText(r.TestCase.PlayerInput, 80))
			t.Logf("      Response: %s", truncateResponseText(r.Response, 200))
		}
	}
}

// getNoFalseTransitionTestCases returns cases where the player moves WITHIN the scene
// and should NOT trigger a scene transition. Extracted from session_western_jesse_calhoun_20260208_190311.yaml
// where "steps into the eaves" falsely triggered [SCENE_TRANSITION:the eaves outside the saloon].
func getNoFalseTransitionTestCases() []SceneResponseTestCase {
	dustyCowboy := character.NewCharacter("scene_1_npc_1", "The Dusty Cowboy")
	dustyCowboy.Aspects.HighConcept = "Mysterious Drifter with a Watchful Eye"

	bartender := character.NewCharacter("bartender", "Maggie Two-Rivers")
	bartender.Aspects.HighConcept = "Weathered Saloon Owner"

	blackJack := character.NewCharacter("blackjack", "Black Jack McCoy")
	blackJack.Aspects.HighConcept = "Dangerous Outlaw with a Quick Draw"

	return []SceneResponseTestCase{
		{
			Name:                "Steps into the eaves - within scene movement",
			PlayerInput:         `Jesse tips his hat to the stranger "Howdy partner, mind if I join you for a smoke?"  He steps into the eaves and pulls out a matchbook and a hand rolled cigar.`,
			SceneName:           "Roping a Plan",
			SceneDescription:    "The moon casts long shadows across the dusty main street of Redemption Gulch as Jesse walks away from the saloon. The sound of crickets and distant laughter from the saloon's patrons fill the night air, punctuated by the creaking of wooden signs and the occasional bark of a dog.",
			OtherCharacters:     []*character.Character{dustyCowboy},
			ConversationContext: `GM: As Jesse turns a corner, he notices a figure watching him from the shadows near the local livery stable, the faint glow of a cigar illuminating their features.`,
			Description:         "Walking to a nearby person under the eaves is movement WITHIN the scene, not leaving. Reproduced from session_western_jesse_calhoun_20260208_190311.yaml",
			CheckNoOptions:      false,
			CheckTransition:     false, // We use the dedicated NoFalseTransition check below
		},
		{
			Name:                "Walks to the bar - within scene movement",
			PlayerInput:         `Jesse walks over to the bar and sits down on a stool. "What'll it be tonight?"`,
			SceneName:           "The Dusty Spur Saloon",
			SceneDescription:    "A dimly lit saloon with swinging doors, a long bar, and the smell of whiskey and tobacco. Miners and cowboys crowd the tables.",
			OtherCharacters:     []*character.Character{bartender},
			ConversationContext: `GM: Jesse pushes through the swinging doors and surveys the room. The piano player pauses briefly, then continues.`,
			Description:         "Walking to the bar inside a saloon is not leaving the saloon",
			CheckNoOptions:      false,
			CheckTransition:     false,
		},
		{
			Name:                "Moves to a corner table",
			PlayerInput:         `Jesse takes his drink and moves to the corner table where the card game is happening.`,
			SceneName:           "The Dusty Spur Saloon",
			SceneDescription:    "A dimly lit saloon with swinging doors, a long bar, poker tables, and a stage for entertainment.",
			OtherCharacters:     []*character.Character{bartender},
			ConversationContext: `GM: Maggie slides the whiskey down the bar. "Careful with that crowd in the corner, stranger."`,
			Description:         "Moving to a different spot within the same room is not leaving",
			CheckNoOptions:      false,
			CheckTransition:     false,
		},
		{
			Name:                "Steps behind cover during conversation",
			PlayerInput:         `Jesse steps behind the windmill wall, keeping his hand near his holster. "Let's talk where we can't be seen from the road."`,
			SceneName:           "Windmill on the Outskirts",
			SceneDescription:    "An old abandoned windmill on the outskirts of town. The creaking blades announce any arrival. Black Jack McCoy stands near the entrance.",
			OtherCharacters:     []*character.Character{blackJack},
			ConversationContext: `GM: Black Jack eyes Jesse warily. "I don't much like being out in the open neither."`,
			Description:         "Moving behind cover at the same location is not departing the scene",
			CheckNoOptions:      false,
			CheckTransition:     false,
		},
		{
			Name:                "Approaches NPC on the porch",
			PlayerInput:         `Jesse walks up the porch steps and leans against the railing next to the old man. "Evening. Nice night for it."`,
			SceneName:           "The General Store",
			SceneDescription:    "The general store sits at the end of the main street. An old man rocks in a chair on the front porch, watching the town.",
			OtherCharacters:     nil,
			ConversationContext: `GM: The rocking chair creaks rhythmically as the old man watches Jesse approach.`,
			Description:         "Walking up porch steps to talk to someone at the current location is not leaving",
			CheckNoOptions:      false,
			CheckTransition:     false,
		},
	}
}

// TestSceneResponse_NoFalseTransition_LLMEvaluation tests that within-scene movement
// does NOT trigger a scene transition marker. Reproduces the bug from
// session_western_jesse_calhoun_20260208_190311.yaml where "steps into the eaves"
// caused a false [SCENE_TRANSITION:the eaves outside the saloon].
func TestSceneResponse_NoFalseTransition_LLMEvaluation(t *testing.T) {
	if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
		t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
	}

	config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
	require.NoError(t, err, "Failed to load Azure config")

	client := azure.NewClient(*config)
	ctx := context.Background()

	verboseLogging := os.Getenv("VERBOSE") == "1"
	testCases := getNoFalseTransitionTestCases()

	var results []SceneResponseResult
	correct := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateSceneResponseBehavior(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			// For these tests, success = NO transition marker
			passed := !result.HasTransition
			result.Matches = passed
			if passed {
				correct++
			}

			if verboseLogging || !passed {
				status := "✓ PASS"
				if !passed {
					status = "✗ FAIL"
				}
				t.Logf("%s: HasTransition=%v", status, result.HasTransition)
				t.Logf("  Input: %s", tc.PlayerInput)
				t.Logf("  Why: %s", tc.Description)
				if result.HasTransition {
					t.Logf("  False transition hint: %s", result.TransitionHint)
				}
				t.Logf("  Response: %s", truncateResponseText(result.Response, 300))
			}

			assert.False(t, result.HasTransition,
				"Response should NOT have SCENE_TRANSITION marker for within-scene movement.\nInput: %s\nTransition hint: %s\nResponse: %s",
				tc.PlayerInput, result.TransitionHint, truncateResponseText(result.Response, 400))
		})
	}

	// Summary
	t.Log("\n========== NO FALSE TRANSITION TEST SUMMARY ==========")
	t.Logf("Cases without false transition: %d/%d (%.1f%%)",
		correct, len(testCases), float64(correct)*100/float64(len(testCases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Matches {
			t.Logf("FAIL: '%s'", r.TestCase.Name)
			t.Logf("      Input: %s", truncateResponseText(r.TestCase.PlayerInput, 80))
			t.Logf("      False transition: %s", r.TransitionHint)
		}
	}
}

// evaluateSceneResponseBehavior runs a single scene response test
func evaluateSceneResponseBehavior(ctx context.Context, client llm.LLMClient, tc SceneResponseTestCase) SceneResponseResult {
	// Create test scene
	testScene := scene.NewScene("test-scene", tc.SceneName, tc.SceneDescription)

	// Create player character
	player := character.NewCharacter("player1", "Jesse Calhoun")
	player.Aspects.HighConcept = "Vengeful Rancher with Nothing Left to Lose"
	player.Aspects.Trouble = "The Cortez Gang Burned My Life"

	// Build contexts
	charContext := buildSceneResponseCharContext(player)
	aspectsContext := buildSceneResponseAspectsContext(testScene, player, tc.OtherCharacters)

	conversationContext := tc.ConversationContext
	if conversationContext == "" {
		conversationContext = "No previous conversation."
	}

	// Prepare template data
	data := promptpkg.SceneResponseData{
		Scene:               testScene,
		CharacterContext:    charContext,
		AspectsContext:      aspectsContext,
		ConversationContext: conversationContext,
		PlayerInput:         tc.PlayerInput,
		InteractionType:     "dialog",
		OtherCharacters:     tc.OtherCharacters,
	}

	prompt, err := promptpkg.RenderSceneResponse(data)
	if err != nil {
		return SceneResponseResult{TestCase: tc, Error: err}
	}

	response, err := llm.SimpleCompletion(ctx, client, prompt, 600, 0.3)
	if err != nil {
		return SceneResponseResult{TestCase: tc, Error: err}
	}

	// Check for options patterns
	hasOptions := false
	optionsFound := ""

	if decisionPromptRegex.MatchString(response) {
		hasOptions = true
		match := decisionPromptRegex.FindString(response)
		optionsFound = "Decision prompt: " + match
	}
	if bulletOptionsRegex.MatchString(response) {
		hasOptions = true
		matches := bulletOptionsRegex.FindAllString(response, 3)
		optionsFound += " Bullet options: " + strings.Join(matches, ", ")
	}
	if numberedOptionsRegex.MatchString(response) {
		hasOptions = true
		matches := numberedOptionsRegex.FindAllString(response, 3)
		optionsFound += " Numbered options: " + strings.Join(matches, ", ")
	}

	// Check for transition marker using production parser
	transition, _ := promptpkg.ParseSceneTransitionMarker(response)
	hasTransition := transition != nil
	transitionHint := ""
	if transition != nil {
		transitionHint = transition.Hint
	}

	// Determine if test passed
	matches := true
	if tc.CheckNoOptions && hasOptions {
		matches = false
	}
	if tc.CheckTransition && !hasTransition {
		matches = false
	}

	return SceneResponseResult{
		TestCase:       tc,
		Response:       response,
		HasOptions:     hasOptions,
		OptionsFound:   strings.TrimSpace(optionsFound),
		HasTransition:  hasTransition,
		TransitionHint: transitionHint,
		Matches:        matches,
	}
}

// buildSceneResponseCharContext creates character context for scene response
func buildSceneResponseCharContext(player *character.Character) string {
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

// buildSceneResponseAspectsContext creates aspects context for scene response
func buildSceneResponseAspectsContext(s *scene.Scene, player *character.Character, others []*character.Character) string {
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

// truncateResponseText truncates a response string for display
func truncateResponseText(s string, maxLen int) string {
	// Remove newlines for cleaner display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
