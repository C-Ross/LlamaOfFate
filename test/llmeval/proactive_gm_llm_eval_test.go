//go:build llmeval

package llmeval_test

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
)

// Proactive GM evaluation tests for issue #104.
//
// Per Fate Core SRD (https://fate-srd.com/fate-core/running-the-game):
// - The GM should make the world respond dynamically, not passively
// - Trouble aspects should create complications that make scenes interesting
// - NPCs should pursue their own agendas and react to the situation
// - Scenes should push toward their dramatic question / purpose
//
// These tests verify the scene_response_prompt.tmpl produces responses where
// the GM behaves proactively rather than only reacting to player statements.

// --- Judge questions ---

const troublePressureQuestion = "Does this GM response reference, hint at, or create a complication " +
	"connected to the player character's Trouble aspect? The Trouble aspect will be provided in the " +
	"context. A subtle allusion, environmental detail, or NPC remark that relates to the Trouble counts. " +
	"YES means the response connects to the Trouble. NO means the Trouble is completely ignored."

const scenePurposeAdvancementQuestion = "Does this GM response move the story toward, raise the stakes of, " +
	"or add new information relevant to the scene's dramatic question / purpose? The scene purpose is " +
	"provided in the context. YES means the response advances or engages with the scene purpose. " +
	"NO means the response is purely reactive small talk with no story progression."

const npcInitiativeQuestion = "Does an NPC in this GM response take an independent action, make a demand, " +
	"reveal information unprompted, or otherwise act on their own agenda rather than passively answering " +
	"the player? YES means at least one NPC shows initiative or drives the scene forward. " +
	"NO means NPCs only react to what the player said with no agenda of their own."

// --- Test case type ---

// ProactiveGMTestCase extends the scene response setup with judge evaluation criteria.
type ProactiveGMTestCase struct {
	Name                string
	PlayerInput         string
	SceneName           string
	SceneDescription    string
	ScenePurpose        string
	OtherCharacters     []*character.Character
	ConversationContext string
	PlayerName          string
	PlayerHighConcept   string
	PlayerTrouble       string
	PlayerAspects       []string
	Description         string

	// Which judge checks to run
	CheckTroublePressure bool
	CheckScenePurpose    bool
	CheckNPCInitiative   bool
}

// ProactiveGMResult stores judge verdicts for a single test case.
type ProactiveGMResult struct {
	TestCase              ProactiveGMTestCase
	Response              string
	TroublePressurePass   bool
	TroublePressureReason string
	ScenePurposePass      bool
	ScenePurposeReason    string
	NPCInitiativePass     bool
	NPCInitiativeReason   string
	Error                 error
}

// --- Test cases ---

func getTroublePressureTestCases() []ProactiveGMTestCase {
	bartender := NewBartender()

	return []ProactiveGMTestCase{
		{
			Name:                 "Trouble should color a tense moment",
			PlayerInput:          `Jesse orders a whiskey and looks around the saloon carefully.`,
			SceneName:            "The Dusty Spur Saloon",
			SceneDescription:     "A dimly lit saloon packed with miners and cowboys. Wanted posters line the wall behind the bar.",
			ScenePurpose:         "Can Jesse find an ally in Redemption Gulch?",
			OtherCharacters:      []*character.Character{bartender},
			ConversationContext:  `GM: The saloon buzzes with low conversation. Maggie polishes a glass, watching the door.`,
			PlayerName:           "Jesse Calhoun",
			PlayerHighConcept:    "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:        "The Cortez Gang Burned My Life",
			Description:          "With Trouble 'The Cortez Gang Burned My Life' and wanted posters on the wall, the GM should weave in a reminder or complication related to the gang",
			CheckTroublePressure: true,
		},
		{
			Name:                 "Trouble allusion during friendly NPC chat",
			PlayerInput:          `Jesse asks Maggie "You been here long? What's this town really like?"`,
			SceneName:            "The Dusty Spur Saloon",
			SceneDescription:     "A saloon with creaky floorboards and a dusty chandelier. The crowd is thin tonight.",
			ScenePurpose:         "Can Jesse learn how deep the Cortez gang's control runs?",
			OtherCharacters:      []*character.Character{bartender},
			ConversationContext:  `GM: Maggie leans on the bar, looking tired. "Long enough to know trouble when I see it."`,
			PlayerName:           "Jesse Calhoun",
			PlayerHighConcept:    "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:        "The Cortez Gang Burned My Life",
			Description:          "When asked about the town, Maggie's answer should allude to the gang (the player's Trouble) since the scene purpose is about them",
			CheckTroublePressure: true,
		},
		{
			Name:                 "Trouble pressure with different character",
			PlayerInput:          `Zird examines the bookshelves, looking for anything about the Ritual of Stars.`,
			SceneName:            "The Collegia Library",
			SceneDescription:     "Towering shelves of ancient tomes. Arcane sigils glow faintly on the ceiling. A few scholars study at distant tables.",
			ScenePurpose:         "Can Zird find evidence of the conspiracy before the conspirators find him?",
			OtherCharacters:      nil,
			ConversationContext:  `GM: The library is quiet except for the scratch of quills and the hum of protective wards.`,
			PlayerName:           "Zird the Arcane",
			PlayerHighConcept:    "Wizard Detective on the Trail of Forbidden Knowledge",
			PlayerTrouble:        "The Lure of Ancient Mysteries",
			Description:          "With Trouble 'The Lure of Ancient Mysteries', browsing forbidden texts should tempt Zird — the GM should hint at dangerous but alluring knowledge",
			CheckTroublePressure: true,
		},
	}
}

func getScenePurposeTestCases() []ProactiveGMTestCase {
	sheriff := character.NewCharacter("sheriff", "Sheriff Daniels")
	sheriff.Aspects.HighConcept = "Tired Lawman Who Looks the Other Way"
	sheriff.Aspects.Trouble = "In Cortez's Pocket"

	rival := character.NewCharacter("rival", "Magister Voss")
	rival.Aspects.HighConcept = "Ambitious Arcanist with Hidden Loyalties"

	return []ProactiveGMTestCase{
		{
			Name:                "Scene purpose should emerge in NPC dialog",
			PlayerInput:         `Jesse tips his hat. "Sheriff, I'm new in town. Seems like a fine enough place."`,
			SceneName:           "The Sheriff's Office",
			SceneDescription:    "A small office with a desk, gun rack, and wanted posters. The sheriff sits behind his desk.",
			ScenePurpose:        "Will Jesse discover that the sheriff is working with the Cortez gang?",
			OtherCharacters:     []*character.Character{sheriff},
			ConversationContext: `GM: Sheriff Daniels looks up from a stack of papers. A half-empty whiskey bottle sits on his desk.`,
			PlayerName:          "Jesse Calhoun",
			PlayerHighConcept:   "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:       "The Cortez Gang Burned My Life",
			Description:         "Even with casual player input, the sheriff's response should contain tells or hints about his corruption (the scene purpose)",
			CheckScenePurpose:   true,
		},
		{
			Name:                "Idle chat should still nudge toward purpose",
			PlayerInput:         `Jesse looks out the window. "Nice sunset tonight."`,
			SceneName:           "The Sheriff's Office",
			SceneDescription:    "A small frontier office. Through the window, the sun sets over Redemption Gulch.",
			ScenePurpose:        "Will Jesse discover that the sheriff is working with the Cortez gang?",
			OtherCharacters:     []*character.Character{sheriff},
			ConversationContext: `GM: The sheriff glances at the window and shifts uncomfortably in his chair.`,
			PlayerName:          "Jesse Calhoun",
			PlayerHighConcept:   "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:       "The Cortez Gang Burned My Life",
			Description:         "Even with deliberately vague player input, the GM should steer toward the scene's dramatic question rather than just small-talking about the weather",
			CheckScenePurpose:   true,
		},
		{
			Name:                "Scene purpose in exploration context",
			PlayerInput:         `Zird opens a random tome and starts reading.`,
			SceneName:           "The Collegia Library",
			SceneDescription:    "Towering shelves of ancient tomes. Arcane sigils glow on the ceiling. A few scholars study quietly.",
			ScenePurpose:        "Can Zird find evidence of the conspiracy before the conspirators find him?",
			OtherCharacters:     []*character.Character{rival},
			ConversationContext: `GM: Magister Voss glances up from his own reading as Zird enters. His expression is unreadable.`,
			PlayerName:          "Zird the Arcane",
			PlayerHighConcept:   "Wizard Detective on the Trail of Forbidden Knowledge",
			PlayerTrouble:       "The Lure of Ancient Mysteries",
			Description:         "The tome Zird opens or Voss's behavior should feed into the conspiracy investigation (the scene purpose)",
			CheckScenePurpose:   true,
		},
	}
}

func getNPCInitiativeTestCases() []ProactiveGMTestCase {
	blackJack := NewBlackJack()

	bartender := NewBartender()

	gangLt := character.NewCharacter("gang_lt", "El Sombra")
	gangLt.Aspects.HighConcept = "Cortez's Ruthless Lieutenant"
	gangLt.Aspects.Trouble = "Enjoys Cruelty Too Much"

	return []ProactiveGMTestCase{
		{
			Name:                "Hostile NPC should act on own agenda",
			PlayerInput:         `Jesse pauses and waits, watching Black Jack carefully.`,
			SceneName:           "Windmill on the Outskirts",
			SceneDescription:    "An old abandoned windmill. The creaking blades echo. Black Jack McCoy stands near the entrance.",
			ScenePurpose:        "Can Jesse get Black Jack to reveal the gang's location?",
			OtherCharacters:     []*character.Character{blackJack},
			ConversationContext: `GM: Black Jack's hand rests on his holster. His eyes scan the horizon behind Jesse.`,
			PlayerName:          "Jesse Calhoun",
			PlayerHighConcept:   "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:       "The Cortez Gang Burned My Life",
			Description:         "When the player just waits, the NPC should take initiative — make a demand, ask a question, or make a move — not stand there passively",
			CheckNPCInitiative:  true,
		},
		{
			Name:                "NPC should push their own goals in dialog",
			PlayerInput:         `Jesse shrugs. "Maybe."`,
			SceneName:           "The Dusty Spur Saloon",
			SceneDescription:    "A dimly lit saloon. Maggie stands behind the bar.",
			ScenePurpose:        "Can Jesse find an ally in Redemption Gulch?",
			OtherCharacters:     []*character.Character{bartender},
			ConversationContext: `GM: Maggie narrows her eyes. "You look like you're carrying more than just a saddlebag, stranger."`,
			PlayerName:          "Jesse Calhoun",
			PlayerHighConcept:   "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:       "The Cortez Gang Burned My Life",
			Description:         "With a noncommittal player response, Maggie should push further — she has her own curiosity and concerns — not just accept the brush-off",
			CheckNPCInitiative:  true,
		},
		{
			Name:                "Dangerous NPC should escalate tension",
			PlayerInput:         `Jesse stays quiet, keeping his hand near his holster.`,
			SceneName:           "The Abandoned Mine Entrance",
			SceneDescription:    "A dark mine entrance flanked by rotting timber. El Sombra blocks the path, flanked by two thugs.",
			ScenePurpose:        "Can Jesse confront the gang's lieutenant and survive?",
			OtherCharacters:     []*character.Character{gangLt},
			ConversationContext: `GM: El Sombra grins, revealing a gold tooth. "Well, well. The rancher who won't stay dead."`,
			PlayerName:          "Jesse Calhoun",
			PlayerHighConcept:   "Vengeful Rancher with Nothing Left to Lose",
			PlayerTrouble:       "The Cortez Gang Burned My Life",
			Description:         "A cruel villain should escalate — taunt, threaten, or make a move — when the player stays silent, not just stand there",
			CheckNPCInitiative:  true,
		},
	}
}

// --- Evaluate function ---

func evaluateProactiveGM(ctx context.Context, client llm.LLMClient, tc ProactiveGMTestCase) ProactiveGMResult {
	// Create test scene
	testScene := scene.NewScene("test-scene", tc.SceneName, tc.SceneDescription)

	// Create player character
	player := character.NewCharacter("player1", tc.PlayerName)
	player.Aspects.HighConcept = tc.PlayerHighConcept
	player.Aspects.Trouble = tc.PlayerTrouble
	for _, a := range tc.PlayerAspects {
		player.Aspects.AddAspect(a)
	}

	// Build contexts using existing helpers from scene_response_llm_eval_test.go
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
		ScenePurpose:        tc.ScenePurpose,
	}

	prompt, err := promptpkg.RenderSceneResponse(data)
	if err != nil {
		return ProactiveGMResult{TestCase: tc, Error: err}
	}

	response, err := llm.SimpleCompletion(ctx, client, prompt, 600, 0.3)
	if err != nil {
		return ProactiveGMResult{TestCase: tc, Error: err}
	}

	result := ProactiveGMResult{
		TestCase: tc,
		Response: response,
	}

	troubleContext := "Player character Trouble aspect: " + tc.PlayerTrouble
	purposeContext := "Scene purpose (dramatic question): " + tc.ScenePurpose

	if tc.CheckTroublePressure {
		judge, judgeErr := LLMJudgeWithContext(ctx, client, response, troublePressureQuestion, troubleContext)
		if judgeErr != nil {
			result.Error = judgeErr
			return result
		}
		result.TroublePressurePass = judge.Pass
		result.TroublePressureReason = judge.Reasoning
	}

	if tc.CheckScenePurpose {
		judge, judgeErr := LLMJudgeWithContext(ctx, client, response, scenePurposeAdvancementQuestion, purposeContext)
		if judgeErr != nil {
			result.Error = judgeErr
			return result
		}
		result.ScenePurposePass = judge.Pass
		result.ScenePurposeReason = judge.Reasoning
	}

	if tc.CheckNPCInitiative {
		judge, judgeErr := LLMJudge(ctx, client, response, npcInitiativeQuestion)
		if judgeErr != nil {
			result.Error = judgeErr
			return result
		}
		result.NPCInitiativePass = judge.Pass
		result.NPCInitiativeReason = judge.Reasoning
	}

	return result
}

// --- Test functions ---

// TestProactiveGM_TroublePressure_LLMEvaluation verifies the GM weaves the player's
// Trouble aspect into scene responses as complications or allusions.
//
// Per Fate Core SRD (https://fate-srd.com/fate-core/running-the-game):
// "Think about the aspects on the PCs' character sheets... use them to generate
// problems for the PCs." Trouble aspects especially should create ongoing pressure.
func TestProactiveGM_TroublePressure_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	verboseLogging := VerboseLoggingEnabled()
	testCases := getTroublePressureTestCases()

	var results []ProactiveGMResult
	passed := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateProactiveGM(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.TroublePressurePass {
				passed++
			}

			if verboseLogging || !result.TroublePressurePass {
				status := "✓ PASS"
				if !result.TroublePressurePass {
					status = "✗ FAIL"
				}
				t.Logf("%s: %s", status, tc.Name)
				t.Logf("  Trouble: %s", tc.PlayerTrouble)
				t.Logf("  Judge reasoning: %s", result.TroublePressureReason)
				t.Logf("  Response: %s", TruncateResponse(result.Response, 400))
			}

			assert.True(t, result.TroublePressurePass,
				"GM should weave Trouble aspect '%s' into the response.\nJudge: %s\nResponse: %s",
				tc.PlayerTrouble, result.TroublePressureReason, TruncateResponse(result.Response, 400))
		})
	}

	t.Log("\n========== TROUBLE PRESSURE SUMMARY ==========")
	t.Logf("Passed: %d/%d (%.1f%%)", passed, len(testCases), float64(passed)*100/float64(len(testCases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.TroublePressurePass {
			t.Logf("FAIL: %s — %s", r.TestCase.Name, r.TroublePressureReason)
		}
	}
}

// TestProactiveGM_ScenePurpose_LLMEvaluation verifies the GM pushes the story toward
// the scene's dramatic question, even when the player's input is vague or off-topic.
//
// Per Fate Core SRD (https://fate-srd.com/fate-core/defining-scenes):
// Every scene should have a purpose — a dramatic question to resolve. The GM should
// drive toward that purpose, not let scenes drift into aimless conversation.
func TestProactiveGM_ScenePurpose_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	verboseLogging := VerboseLoggingEnabled()
	testCases := getScenePurposeTestCases()

	var results []ProactiveGMResult
	passed := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateProactiveGM(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.ScenePurposePass {
				passed++
			}

			if verboseLogging || !result.ScenePurposePass {
				status := "✓ PASS"
				if !result.ScenePurposePass {
					status = "✗ FAIL"
				}
				t.Logf("%s: %s", status, tc.Name)
				t.Logf("  Scene Purpose: %s", tc.ScenePurpose)
				t.Logf("  Player Input: %s", tc.PlayerInput)
				t.Logf("  Judge reasoning: %s", result.ScenePurposeReason)
				t.Logf("  Response: %s", TruncateResponse(result.Response, 400))
			}

			assert.True(t, result.ScenePurposePass,
				"GM should advance scene purpose '%s' even with vague player input.\nJudge: %s\nResponse: %s",
				tc.ScenePurpose, result.ScenePurposeReason, TruncateResponse(result.Response, 400))
		})
	}

	t.Log("\n========== SCENE PURPOSE ADVANCEMENT SUMMARY ==========")
	t.Logf("Passed: %d/%d (%.1f%%)", passed, len(testCases), float64(passed)*100/float64(len(testCases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.ScenePurposePass {
			t.Logf("FAIL: %s — %s", r.TestCase.Name, r.ScenePurposeReason)
		}
	}
}

// TestProactiveGM_NPCInitiative_LLMEvaluation verifies NPCs act on their own agendas
// rather than passively waiting for the player to drive all action.
//
// Per Fate Core SRD (https://fate-srd.com/fate-core/running-the-game):
// NPCs should "pursue their own goals and react naturally." A hostile NPC shouldn't
// just stand there when the player is silent — they should threaten, demand, or act.
func TestProactiveGM_NPCInitiative_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	verboseLogging := VerboseLoggingEnabled()
	testCases := getNPCInitiativeTestCases()

	var results []ProactiveGMResult
	passed := 0

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateProactiveGM(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.NPCInitiativePass {
				passed++
			}

			if verboseLogging || !result.NPCInitiativePass {
				status := "✓ PASS"
				if !result.NPCInitiativePass {
					status = "✗ FAIL"
				}
				t.Logf("%s: %s", status, tc.Name)
				t.Logf("  NPC(s): %s", describeNPCs(tc.OtherCharacters))
				t.Logf("  Player Input: %s", tc.PlayerInput)
				t.Logf("  Judge reasoning: %s", result.NPCInitiativeReason)
				t.Logf("  Response: %s", TruncateResponse(result.Response, 400))
			}

			assert.True(t, result.NPCInitiativePass,
				"NPCs should show initiative, not passively wait.\nJudge: %s\nResponse: %s",
				result.NPCInitiativeReason, TruncateResponse(result.Response, 400))
		})
	}

	t.Log("\n========== NPC INITIATIVE SUMMARY ==========")
	t.Logf("Passed: %d/%d (%.1f%%)", passed, len(testCases), float64(passed)*100/float64(len(testCases)))

	t.Log("\n--- Failed Cases ---")
	for _, r := range results {
		if !r.NPCInitiativePass {
			t.Logf("FAIL: %s — %s", r.TestCase.Name, r.NPCInitiativeReason)
		}
	}
}

func describeNPCs(chars []*character.Character) string {
	if len(chars) == 0 {
		return "(none)"
	}
	names := ""
	for i, c := range chars {
		if i > 0 {
			names += ", "
		}
		names += c.Name
		if c.Aspects.HighConcept != "" {
			names += " (" + c.Aspects.HighConcept + ")"
		}
	}
	return names
}
