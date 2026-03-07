//go:build llmeval

package llmeval_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// markerLeakJudgeQuestion asks whether the response references internal system markers.
const markerLeakJudgeQuestion = `Does this response contain any out-of-fiction references to internal system tags, markers, or formatting instructions? Look for:
- Mentioning marker names like "SCENE_TRANSITION", "CONFLICT", or similar technical tags by name
- Explaining why a marker was or was not added (e.g., "No SCENE_TRANSITION because..." or "adding a scene transition marker")
- Referencing the marker system itself (e.g., "as per the scene markers" or "no transition marker needed")
- Any meta-commentary about the formatting or structure of the response itself
Note: The actual marker syntax [SCENE_TRANSITION:...] or [CONFLICT:...] at the end of a response is EXPECTED and should NOT count as a leak. Only flag plain-text references to marker names or explanations about marker logic.`

// markerLeakRegex matches literal references to marker names in narrative text.
// This catches the LLM writing things like "No SCENE_TRANSITION" or "CONFLICT marker".
// It deliberately excludes the actual bracketed marker syntax which is expected.
var markerLeakRegex = regexp.MustCompile(`(?i)\b(SCENE_TRANSITION|CONFLICT_END|CONFLICT_MARKER)\b|(?:no|adding|without|include|need|skip)\s+(?:a\s+)?(?:scene[_ ]?transition|conflict[_ ]?marker)`)

// MarkerLeakTestCase defines a scenario likely to provoke marker leaking.
type MarkerLeakTestCase struct {
	Name                string
	PlayerInput         string
	SceneName           string
	SceneDescription    string
	OtherCharacters     []*core.Character
	ConversationContext string
	Description         string
}

// MarkerLeakResult stores the evaluation result.
type MarkerLeakResult struct {
	TestCase       MarkerLeakTestCase
	Response       string
	CleanedText    string // Response with valid markers stripped
	RegexLeakFound bool
	RegexLeakText  string
	JudgeLeak      bool
	JudgeReasoning string
	Passed         bool
	Error          error
}

// getMarkerLeakTestCases returns scenarios designed to provoke marker leaking.
// These are borderline cases where the LLM might feel the need to explain its
// marker reasoning rather than just writing fiction.
func getMarkerLeakTestCases() []MarkerLeakTestCase {
	bartender := NewBartender()
	blackJack := NewBlackJack()

	guard := core.NewCharacter("guard", "Gate Guard")
	guard.Aspects.HighConcept = "Dutiful Town Watchman"

	return []MarkerLeakTestCase{
		{
			Name:             "Entering a building - not leaving",
			PlayerInput:      "Zero pushes open the heavy door and steps inside the building.",
			SceneName:        "Outside the Abandoned Mine",
			SceneDescription: "The entrance to an old mine shaft, boarded up and covered in warning signs. Shadows pool around the entrance.",
			Description:      "Entering a building is ambiguous — the LLM might explain that no SCENE_TRANSITION applies",
		},
		{
			Name:             "Moving to a different part of the area",
			PlayerInput:      "Jesse walks across the street to the general store porch.",
			SceneName:        "Main Street of Redemption Gulch",
			SceneDescription: "The dusty main street runs through the center of town, lined with wooden buildings, a saloon, and the sheriff's office.",
			Description:      "Walking across the street is within-scene movement, LLM might explain why no transition",
		},
		{
			Name:                "Partial departure language",
			PlayerInput:         "Jesse heads toward the door but stops to look back at Maggie.",
			SceneName:           "The Dusty Spur Saloon",
			SceneDescription:    "A dimly lit saloon with swinging doors, a long bar, and the smell of whiskey",
			OtherCharacters:     []*core.Character{bartender},
			ConversationContext: "GM: Maggie slides the whiskey across the bar. \"Something on your mind, stranger?\"",
			Description:         "Ambiguous departure intent - the LLM might explain its marker decision",
		},
		{
			Name:                "Threatening but not attacking",
			PlayerInput:         "Jesse puts his hand on his holster and stares Black Jack down. \"I'd choose my next words carefully if I were you.\"",
			SceneName:           "Windmill on the Outskirts",
			SceneDescription:    "An old abandoned windmill on the outskirts of town. Tension fills the air.",
			OtherCharacters:     []*core.Character{blackJack},
			ConversationContext: "GM: Black Jack sneers. \"You think you can just waltz in here and make demands?\"",
			Description:         "Tense but not violent — LLM might explain why no CONFLICT marker was added",
		},
		{
			Name:                "Player asks about exits",
			PlayerInput:         "Is there another way out of here?",
			SceneName:           "The Abandoned Warehouse",
			SceneDescription:    "A dusty warehouse filled with old crates. A single door leads to the alley, and moonlight streams through broken windows.",
			ConversationContext: "GM: The warehouse is quiet except for the scurrying of rats.",
			Description:         "Asking about exits could confuse the LLM into discussing transition markers",
		},
		{
			Name:                "NPC blocks the exit",
			PlayerInput:         "Jesse tries to push past the guard and leave.",
			SceneName:           "Town Gate",
			SceneDescription:    "The wooden gate at the edge of town, manned by a bored-looking guard.",
			OtherCharacters:     []*core.Character{guard},
			ConversationContext: "GM: The guard steps in front of Jesse, blocking his path. \"No one leaves town after dark. Sheriff's orders.\"",
			Description:         "Player wants to leave but can't — LLM might discuss transition or conflict marker logic",
		},
		{
			Name:                "De-escalation conversation",
			PlayerInput:         "\"Alright, alright. Let's all calm down here. Nobody needs to get hurt.\" Jesse slowly raises his hands.",
			SceneName:           "Windmill on the Outskirts",
			SceneDescription:    "An old abandoned windmill. The air crackles with tension.",
			OtherCharacters:     []*core.Character{blackJack},
			ConversationContext: "GM: Black Jack has his hand on his gun, eyes narrowed. \"Give me one good reason not to draw.\"",
			Description:         "De-escalation might cause the LLM to narrate about conflict markers",
		},
		{
			Name:             "Arriving at destination",
			PlayerInput:      "Lyra arrives at the tower and pushes through the entrance.",
			SceneName:        "Road to the Tower",
			SceneDescription: "A winding road through dark forest, leading to an ancient stone tower looming in the distance.",
			Description:      "Arriving somewhere is entering, not leaving — LLM might talk about SCENE_TRANSITION logic",
		},
		{
			Name:                "Looking out a window at a far location",
			PlayerInput:         "Jesse peers through the window toward the hills. \"That's where we need to go.\"",
			SceneName:           "Sheriff's Office",
			SceneDescription:    "A small office with a desk, a cell, and a window overlooking the town.",
			ConversationContext: "GM: The sheriff points to a map on the wall. \"The Cortez gang is holed up in those hills.\"",
			Description:         "Looking at a distant destination without leaving could trigger marker explanation",
		},
		{
			Name:                "Contemplating departure",
			PlayerInput:         "Jesse finishes his drink and weighs his options. Should he stay or go?",
			SceneName:           "The Dusty Spur Saloon",
			SceneDescription:    "A dimly lit saloon with the buzz of conversation.",
			OtherCharacters:     []*core.Character{bartender},
			ConversationContext: "GM: Maggie watches Jesse from behind the bar, polishing the same glass she's held for five minutes.",
			Description:         "Internal deliberation about leaving — LLM might reason about whether to add transition",
		},
	}
}

// TestMarkerLeak_LLMEvaluation tests that the LLM never references internal
// marker names or explains marker reasoning in its narrative responses.
//
// Run with: VERBOSE=1 go test -v -tags=llmeval -run TestMarkerLeak ./test/llmeval/ -timeout 5m
func TestMarkerLeak_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	verboseLogging := VerboseLoggingEnabled()
	testCases := getMarkerLeakTestCases()

	var results []MarkerLeakResult
	passed := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateMarkerLeak(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.Passed {
				passed++
			}

			if verboseLogging || !result.Passed {
				status := "✓ PASS"
				if !result.Passed {
					status := "✗ FAIL"
					t.Logf("%s: %s", status, tc.Name)
				} else {
					t.Logf("%s: %s", status, tc.Name)
				}
				t.Logf("  Input: %s", tc.PlayerInput)
				t.Logf("  Why: %s", tc.Description)
				if result.RegexLeakFound {
					t.Logf("  Regex leak: %q", result.RegexLeakText)
				}
				if result.JudgeLeak {
					t.Logf("  Judge found leak: %s", result.JudgeReasoning)
				}
				t.Logf("  Cleaned text: %s", TruncateResponse(result.CleanedText, 300))
			}

			assert.True(t, result.Passed,
				"Response should not contain out-of-fiction marker references.\n"+
					"Regex leak: %v (%s)\nJudge leak: %v (%s)\nResponse: %s",
				result.RegexLeakFound, result.RegexLeakText,
				result.JudgeLeak, result.JudgeReasoning,
				TruncateResponse(result.Response, 400))
		})
	}

	// Summary
	t.Log("\n========== MARKER LEAK EVALUATION SUMMARY ==========")
	t.Logf("Clean responses (no marker leak): %d/%d (%.1f%%)",
		passed, len(testCases), float64(passed)*100/float64(len(testCases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.Passed {
			t.Logf("FAIL: %s", r.TestCase.Name)
			t.Logf("      Input: %s", TruncateResponse(r.TestCase.PlayerInput, 80))
			if r.RegexLeakFound {
				t.Logf("      Regex: %q", r.RegexLeakText)
			}
			if r.JudgeLeak {
				t.Logf("      Judge: %s", r.JudgeReasoning)
			}
			t.Logf("      Response: %s", TruncateResponse(r.Response, 200))
		}
	}
}

// evaluateMarkerLeak renders the scene_response prompt, calls the LLM,
// then checks the response for marker references via regex and LLM judge.
func evaluateMarkerLeak(ctx context.Context, client llm.LLMClient, tc MarkerLeakTestCase) MarkerLeakResult {
	testScene := scene.NewScene("test-scene", tc.SceneName, tc.SceneDescription)

	player := core.NewCharacter("player1", "Zero")
	player.Aspects.HighConcept = "Shadow Operative with a Hidden Past"
	player.Aspects.Trouble = "Trust No One — Not Even Yourself"

	charContext := BuildCharacterContext(player)
	aspectsContext := BuildAspectsContext(testScene, player, tc.OtherCharacters)

	conversationContext := tc.ConversationContext
	if conversationContext == "" {
		conversationContext = "No previous conversation."
	}

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
		return MarkerLeakResult{TestCase: tc, Error: err}
	}

	// Use production temperature (0.7) since marker leaks are more likely at higher temps
	response, err := llm.SimpleCompletion(ctx, client, prompt, 600, 0.7)
	if err != nil {
		return MarkerLeakResult{TestCase: tc, Error: err}
	}

	// Strip valid markers to get the "player-facing" text
	cleanedText := stripValidMarkers(response)

	// Check 1: Regex for literal marker name references in cleaned text
	regexLeakFound := false
	regexLeakText := ""
	if match := markerLeakRegex.FindString(cleanedText); match != "" {
		regexLeakFound = true
		regexLeakText = match
	}

	// Check 2: LLM judge for subtler out-of-fiction references
	judgeLeak := false
	judgeReasoning := ""
	judgeResult, err := LLMJudge(ctx, client, cleanedText, markerLeakJudgeQuestion)
	if err != nil {
		return MarkerLeakResult{TestCase: tc, Error: fmt.Errorf("judge call failed: %w", err)}
	}
	judgeLeak = judgeResult.Pass // pass=true means YES, it found a leak
	judgeReasoning = judgeResult.Reasoning

	passed := !regexLeakFound && !judgeLeak

	return MarkerLeakResult{
		TestCase:       tc,
		Response:       response,
		CleanedText:    cleanedText,
		RegexLeakFound: regexLeakFound,
		RegexLeakText:  regexLeakText,
		JudgeLeak:      judgeLeak,
		JudgeReasoning: judgeReasoning,
		Passed:         passed,
	}
}

// Regexes matching valid marker syntax that should be stripped before checking for leaks.
var (
	validSceneTransitionRegex = regexp.MustCompile(`\[SCENE_TRANSITION:[^\]]+\]`)
	validConflictRegex        = regexp.MustCompile(`\[CONFLICT:(physical|mental):[^\]]+\]`)
	validConflictEndRegex     = regexp.MustCompile(`\[CONFLICT:end:(surrender|agreement|retreat|resolved)\]`)
)

// stripValidMarkers removes expected marker syntax from the response,
// returning only the narrative text that players would see.
func stripValidMarkers(response string) string {
	cleaned := validSceneTransitionRegex.ReplaceAllString(response, "")
	cleaned = validConflictRegex.ReplaceAllString(cleaned, "")
	cleaned = validConflictEndRegex.ReplaceAllString(cleaned, "")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return strings.TrimSpace(cleaned)
}

// TestMarkerLeak_NeoTechEntrance_LLMEvaluation reproduces the exact scenario from
// save a9d335d8115517ec where the LLM wrote "No SCENE_TRANSITION as Zero is entering
// the building." — an out-of-fiction reference to an internal marker name.
//
// Run with: VERBOSE=1 go test -v -tags=llmeval -run TestMarkerLeak_NeoTechEntrance -count=10 ./test/llmeval/ -timeout 10m
func TestMarkerLeak_NeoTechEntrance_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	verboseLogging := VerboseLoggingEnabled()

	// Reproduce the exact save state from a9d335d8115517ec
	player := core.NewCharacter("zero", "Zero")
	player.Aspects.HighConcept = "Ghost in the Machine Netrunner"
	player.Aspects.Trouble = "Wanted by Three Megacorps"
	player.Aspects.AddAspect("Military-Grade Cybernetic Reflexes")
	player.Aspects.AddAspect("Nobody Gets Left Behind")
	player.Aspects.AddAspect("I Know a Guy for Everything")

	raven := core.NewCharacter("scene_1_npc_0", "Raven")
	raven.Aspects.HighConcept = "Street-Savvy Information Broker"

	testScene := scene.NewScene("scene_2", "Outside NeoTech Tower",
		"Rain-soaked streets glisten under the tower's imposing shadow, the sleek, high-tech facade of NeoTech's headquarters piercing the night sky like a shard of glass. The sound of distant hover-drones and the hum of security systems create a constant, ominous background thrum. Zero stands at the edge of the crowd, a sea of faceless pedestrians flowing around them.")

	testScene.AddSituationAspect(scene.SituationAspect{Aspect: "Crowd Provides Cover", Duration: "scene"})
	testScene.AddSituationAspect(scene.SituationAspect{Aspect: "Security Drone Patrol", Duration: "scene"})
	testScene.AddSituationAspect(scene.SituationAspect{Aspect: "Tower's Main Entrance Well-Lit", Duration: "scene"})
	// The boost from the prior action
	testScene.AddSituationAspect(scene.SituationAspect{Aspect: "Blended with the Crowd", FreeInvokes: 1, IsBoost: true, Duration: "scene", CreatedBy: "zero"})

	charContext := BuildCharacterContext(player)
	aspectsContext := BuildAspectsContext(testScene, player, []*core.Character{raven})

	conversationHistory := `GM: As Zero weaves through the crowd with an ease that belies their bulky cybernetic enhancements, they expertly evade the probing spotlight of a security drone, its hum receding into the distance as they slip into the shadows cast by NeoTech's warehouses. The rain-soaked air is filled with the scent of wet pavement and ozone as Zero disappears into the darkness, the thrum of the city's underbelly growing louder. With their newfound position at the warehouse's rear, the soft glow of a service entrance beckons, a potential backdoor into the heart of NeoTech's operations.`

	playerInput := "Zero verifies there are no security drones watching, then uses his cyberpick to open the service entrance."

	data := promptpkg.SceneResponseData{
		Scene:               testScene,
		CharacterContext:    charContext,
		AspectsContext:      aspectsContext,
		ConversationContext: conversationHistory,
		PlayerInput:         playerInput,
		InteractionType:     "dialog",
		OtherCharacters:     []*core.Character{raven},
		ScenePurpose:        "Can Zero breach the NeoTech facility's outer security perimeter undetected?",
	}

	prompt, err := promptpkg.RenderSceneResponse(data)
	require.NoError(t, err, "Failed to render prompt")

	// Use exact production temperature
	response, err := llm.SimpleCompletion(ctx, client, prompt, 600, 0.7)
	require.NoError(t, err, "LLM call failed")

	cleanedText := stripValidMarkers(response)

	// Check 1: Regex for literal marker name references
	regexLeakFound := false
	regexLeakText := ""
	if match := markerLeakRegex.FindString(cleanedText); match != "" {
		regexLeakFound = true
		regexLeakText = match
	}

	// Check 2: LLM judge
	judgeResult, err := LLMJudge(ctx, client, cleanedText, markerLeakJudgeQuestion)
	require.NoError(t, err, "Judge call failed")

	passed := !regexLeakFound && !judgeResult.Pass

	if verboseLogging || !passed {
		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		t.Logf("%s | regex_leak=%v judge_leak=%v", status, regexLeakFound, judgeResult.Pass)
		if regexLeakFound {
			t.Logf("  Regex match: %q", regexLeakText)
		}
		if judgeResult.Pass {
			t.Logf("  Judge: %s", judgeResult.Reasoning)
		}
		t.Logf("  Cleaned: %s", TruncateResponse(cleanedText, 400))
	}

	assert.True(t, passed,
		"Response contains out-of-fiction marker reference.\nRegex: %v (%s)\nJudge: %v (%s)\nResponse: %s",
		regexLeakFound, regexLeakText,
		judgeResult.Pass, judgeResult.Reasoning,
		TruncateResponse(response, 500))
}
