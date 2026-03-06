//go:build llmeval

package llmeval_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
)

// ActionNarrativeTestCase defines a test scenario for action narrative evaluation.
type ActionNarrativeTestCase struct {
	Name             string
	ActionType       action.ActionType
	Skill            string
	Description      string
	RawInput         string
	OutcomeType      dice.OutcomeType
	Shifts           int
	SceneName        string
	SceneDescription string
	TestDescription  string // Why this case matters
}

// ActionNarrativeEvalResult stores the outcome of a single evaluation.
type ActionNarrativeEvalResult struct {
	TestCase           ActionNarrativeTestCase
	RawResponse        string
	OutcomeMatch       bool // Narrative tone matches the outcome level
	NoCombatJargon     bool // No inappropriate damage/stress language
	Concise            bool // 1-3 sentences
	Error              error
	JudgeOutcomeResult JudgeResult
	JudgeCombatResult  JudgeResult
	Passed             bool
}

// --- Test case factories ---

func getOvercomeNarrativeTestCases() []ActionNarrativeTestCase {
	return []ActionNarrativeTestCase{
		{
			Name:             "Overcome_Fail_Physical",
			ActionType:       action.Overcome,
			Skill:            "Athletics",
			Description:      "Jump across the chasm",
			RawInput:         "I leap across the chasm to the other side",
			OutcomeType:      dice.Failure,
			Shifts:           -2,
			SceneName:        "The Broken Bridge",
			SceneDescription: "A crumbling stone bridge spans a deep chasm. Chunks of masonry have fallen away, leaving a dangerous gap.",
			TestDescription:  "Failed Overcome should convey setback or inability to pass the obstacle",
		},
		{
			Name:             "Overcome_Tie_Physical",
			ActionType:       action.Overcome,
			Skill:            "Athletics",
			Description:      "Climb the fortress wall",
			RawInput:         "I try to scale the wall using the cracks in the stonework",
			OutcomeType:      dice.Tie,
			Shifts:           0,
			SceneName:        "Fortress Approach",
			SceneDescription: "A towering stone fortress wall rises before you. Moss-covered stones offer some handholds, but the wall is slick with rain.",
			TestDescription:  "Tie on Overcome means success at a minor cost — narrative should reflect partial achievement",
		},
		{
			Name:             "Overcome_Success_Social",
			ActionType:       action.Overcome,
			Skill:            "Rapport",
			Description:      "Convince the merchant to share information",
			RawInput:         "I buy the merchant a drink and ask about the missing shipment",
			OutcomeType:      dice.Success,
			Shifts:           2,
			SceneName:        "Riverside Market",
			SceneDescription: "A bustling riverside market with stalls selling spices, cloth, and exotic goods. The merchant's tent is set apart from the crowd.",
			TestDescription:  "Successful social Overcome should convey achieving the goal through conversation",
		},
		{
			Name:             "Overcome_SWS_Mental",
			ActionType:       action.Overcome,
			Skill:            "Lore",
			Description:      "Decipher the ancient runes",
			RawInput:         "I study the runes carefully, cross-referencing with what I know of ancient symbology",
			OutcomeType:      dice.SuccessWithStyle,
			Shifts:           4,
			SceneName:        "The Sealed Vault",
			SceneDescription: "Deep beneath the temple, an obsidian vault door is inscribed with glowing runes in a long-dead language.",
			TestDescription:  "SWS on Overcome should convey exceptional, impressive achievement",
		},
		{
			Name:             "Overcome_Fail_Stealth",
			ActionType:       action.Overcome,
			Skill:            "Stealth",
			Description:      "Sneak past the sentries",
			RawInput:         "I stick to the shadows and try to slip past the guards",
			OutcomeType:      dice.Failure,
			Shifts:           -3,
			SceneName:        "Palace Perimeter",
			SceneDescription: "The palace grounds at night. Lanterns illuminate the paths, and pairs of sentries patrol on regular rotations.",
			TestDescription:  "Failed stealth should convey detection or inability to pass unnoticed",
		},
		{
			Name:             "Overcome_Success_Investigation",
			ActionType:       action.Overcome,
			Skill:            "Investigate",
			Description:      "Search for clues in the abandoned workshop",
			RawInput:         "I search the workshop thoroughly — under benches, inside drawers, anywhere evidence might be hidden",
			OutcomeType:      dice.Success,
			Shifts:           1,
			SceneName:        "The Clockmaker's Workshop",
			SceneDescription: "A dusty workshop filled with dismantled clockwork mechanisms, scattered gears, and half-finished automata.",
			TestDescription:  "Successful investigation should convey finding useful information",
		},
	}
}

func getCreateAdvantageNarrativeTestCases() []ActionNarrativeTestCase {
	return []ActionNarrativeTestCase{
		{
			Name:             "CaA_Success_Stealth",
			ActionType:       action.CreateAdvantage,
			Skill:            "Stealth",
			Description:      "Find a hidden vantage point",
			RawInput:         "I scout ahead and find a spot overlooking their camp where I can observe without being seen",
			OutcomeType:      dice.Success,
			Shifts:           2,
			SceneName:        "Outlaw Camp Outskirts",
			SceneDescription: "The outskirts of a dusty outlaw camp nestled in a rocky canyon. Campfire smoke rises between canvas tents.",
			TestDescription:  "CaA success should convey establishing a strategic advantage",
		},
		{
			Name:             "CaA_Tie_Deceive",
			ActionType:       action.CreateAdvantage,
			Skill:            "Deceive",
			Description:      "Spread a rumor to distract the guards",
			RawInput:         "I start whispering to the other servants that there's a fire in the east wing",
			OutcomeType:      dice.Tie,
			Shifts:           0,
			SceneName:        "Grand Estate Ball",
			SceneDescription: "A lavish masquerade ball in a candlelit grand hall. Guests in elaborate masks mingle while servants move through the crowd.",
			TestDescription:  "CaA tie gives a boost — narrative should convey a fleeting, partial advantage",
		},
		{
			Name:             "CaA_SWS_Notice",
			ActionType:       action.CreateAdvantage,
			Skill:            "Notice",
			Description:      "Spot the enemy's weakness",
			RawInput:         "I watch the duellist's footwork carefully, looking for a pattern or vulnerability",
			OutcomeType:      dice.SuccessWithStyle,
			Shifts:           3,
			SceneName:        "Moonlit Dueling Ground",
			SceneDescription: "An open courtyard under a full moon. The air is tense as two combatants circle each other on the flagstones.",
			TestDescription:  "CaA SWS should convey discovering a significant, exploitable advantage",
		},
		{
			Name:             "CaA_Fail_Provoke",
			ActionType:       action.CreateAdvantage,
			Skill:            "Provoke",
			Description:      "Taunt the bandit leader into making a mistake",
			RawInput:         "I mock the bandit leader's swordsmanship, trying to make him angry and sloppy",
			OutcomeType:      dice.Failure,
			Shifts:           -1,
			SceneName:        "Canyon Standoff",
			SceneDescription: "A narrow canyon with steep walls. The bandit leader stands with his crew, outnumbering you three to one.",
			TestDescription:  "Failed CaA should convey the provocation backfiring or having no effect",
		},
		{
			Name:             "CaA_Success_Rapport",
			ActionType:       action.CreateAdvantage,
			Skill:            "Rapport",
			Description:      "Win over the crowd at the tavern",
			RawInput:         "I buy a round of drinks and tell a rousing story about my adventures to get the crowd on my side",
			OutcomeType:      dice.Success,
			Shifts:           1,
			SceneName:        "The Rusty Anchor Tavern",
			SceneDescription: "A lively seaside tavern packed with sailors, merchants, and dockworkers. A bard plays in the corner.",
			TestDescription:  "Social CaA success should convey gaining social leverage or goodwill",
		},
		{
			Name:             "CaA_Fail_Craft",
			ActionType:       action.CreateAdvantage,
			Skill:            "Crafts",
			Description:      "Rig a tripwire trap across the doorway",
			RawInput:         "I use the wire and bell from my pack to rig a quick alarm trap in the doorway",
			OutcomeType:      dice.Failure,
			Shifts:           -2,
			SceneName:        "Abandoned Watchtower",
			SceneDescription: "A crumbling stone watchtower. The door hangs off its hinges and the floor is littered with debris.",
			TestDescription:  "Failed crafting should convey the trap failing or being noticed",
		},
	}
}

// --- Evaluation logic ---

func buildNarrativeTestScene(tc ActionNarrativeTestCase) *scene.Scene {
	s := scene.NewScene("eval-scene", tc.SceneName, tc.SceneDescription)
	s.AddSituationAspect(scene.NewSituationAspect("sa-1", "Tense Atmosphere", "gm", 0))
	return s
}

func buildNarrativeTestAction(tc ActionNarrativeTestCase) *action.Action {
	act := action.NewAction(
		fmt.Sprintf("eval-%s", tc.Name),
		"eval-player",
		tc.ActionType,
		tc.Skill,
		tc.Description,
	)
	act.RawInput = tc.RawInput
	act.Difficulty = dice.Good // Standard opposition

	// Construct a CheckResult that produces the desired outcome
	act.Outcome = &dice.Outcome{Type: tc.OutcomeType, Shifts: tc.Shifts}
	return act
}

func buildNarrativeTestCharacter() *core.Character {
	char := core.NewCharacter("eval-player", "Kael the Wanderer")
	char.Aspects.HighConcept = "Resourceful Wanderer"
	char.Aspects.Trouble = "Can't Leave Well Enough Alone"
	char.Aspects.AddAspect("Eyes of the Hawk")
	char.SetSkill("Athletics", dice.Good)
	char.SetSkill("Stealth", dice.Good)
	char.SetSkill("Investigate", dice.Good)
	char.SetSkill("Rapport", dice.Fair)
	char.SetSkill("Deceive", dice.Fair)
	char.SetSkill("Notice", dice.Fair)
	char.SetSkill("Lore", dice.Fair)
	char.SetSkill("Provoke", dice.Average)
	char.SetSkill("Crafts", dice.Average)
	return char
}

func evaluateActionNarrative(ctx context.Context, client llm.LLMClient, tc ActionNarrativeTestCase) ActionNarrativeEvalResult {
	result := ActionNarrativeEvalResult{TestCase: tc}

	char := buildNarrativeTestCharacter()
	testScene := buildNarrativeTestScene(tc)
	act := buildNarrativeTestAction(tc)

	charCtx := fmt.Sprintf("Name: %s\nHigh Concept: %s\nTrouble: %s",
		char.Name, char.Aspects.HighConcept, char.Aspects.Trouble)
	aspectsCtx := "Situation Aspects: Tense Atmosphere"

	data := promptpkg.ActionNarrativeData{
		Scene:               testScene,
		CharacterContext:    charCtx,
		AspectsContext:      aspectsCtx,
		ConversationContext: "",
		Action:              act,
		OtherCharacters:     nil,
	}

	promptText, err := promptpkg.RenderActionNarrative(data)
	if err != nil {
		result.Error = fmt.Errorf("render failed: %w", err)
		return result
	}

	narrative, err := llm.SimpleCompletion(ctx, client, promptText, 200, 0.8)
	if err != nil {
		result.Error = fmt.Errorf("LLM call failed: %w", err)
		return result
	}
	result.RawResponse = narrative

	// === Check 1: Conciseness (1-3 sentences) ===
	sentences := countSentences(narrative)
	result.Concise = sentences >= 1 && sentences <= 4 // allow slight tolerance

	// === Check 2: No combat jargon for non-attack actions ===
	combatTerms := []string{"damage", "hit points", "stress box", "wound", "health"}
	result.NoCombatJargon = true
	lower := strings.ToLower(narrative)
	for _, term := range combatTerms {
		if strings.Contains(lower, term) {
			result.NoCombatJargon = false
			break
		}
	}

	// === Check 3: LLM Judge — does the narrative tone match the outcome? ===
	outcomeQuestion := outcomeJudgeQuestion(tc.OutcomeType, tc.ActionType)
	judgeCtx := fmt.Sprintf("The player attempted: %s\nAction Type: %s\nSkill: %s\nOutcome: %s (shifts: %+d)\nScene: %s",
		tc.Description, tc.ActionType.String(), tc.Skill,
		tc.OutcomeType.String(), tc.Shifts, tc.SceneDescription)

	judgeResult, err := LLMJudgeWithContext(ctx, client, narrative, outcomeQuestion, judgeCtx)
	if err != nil {
		// Non-fatal: log but don't fail the whole test
		result.JudgeOutcomeResult = JudgeResult{Pass: false, Reasoning: fmt.Sprintf("judge error: %v", err)}
	} else {
		result.JudgeOutcomeResult = judgeResult
	}
	result.OutcomeMatch = result.JudgeOutcomeResult.Pass

	// === Check 4: LLM Judge — no combat mechanics in non-attack narration ===
	// Phrased positively so pass=true means "free of jargon" (good).
	combatQuestion := "Is this narrative free of game-mechanical terminology such as stress boxes, consequences, damage numbers, hit points, or combat-specific jargon? (Answer YES if the text reads like pure fiction with no mechanical terms.)"
	combatJudge, err := LLMJudge(ctx, client, narrative, combatQuestion)
	if err != nil {
		result.JudgeCombatResult = JudgeResult{Pass: true, Reasoning: fmt.Sprintf("judge error: %v", err)}
	} else {
		result.JudgeCombatResult = combatJudge
		result.NoCombatJargon = result.NoCombatJargon && combatJudge.Pass
	}

	result.Passed = result.OutcomeMatch && result.NoCombatJargon && result.Concise
	return result
}

// outcomeJudgeQuestion returns a yes/no question appropriate for judging whether
// the narrative matches the given outcome type.
func outcomeJudgeQuestion(outcomeType dice.OutcomeType, actionType action.ActionType) string {
	actionLabel := "action"
	if actionType == action.Overcome {
		actionLabel = "attempt to overcome the obstacle"
	} else if actionType == action.CreateAdvantage {
		actionLabel = "attempt to create a tactical advantage"
	}

	switch outcomeType {
	case dice.Failure:
		return fmt.Sprintf("Does this narrative clearly convey that the character's %s failed, was thwarted, or resulted in a setback?", actionLabel)
	case dice.Tie:
		return fmt.Sprintf("Does this narrative convey that the character's %s partially succeeded or came at a cost or compromise?", actionLabel)
	case dice.Success:
		return fmt.Sprintf("Does this narrative convey that the character's %s succeeded or achieved the intended goal?", actionLabel)
	case dice.SuccessWithStyle:
		return fmt.Sprintf("Does this narrative convey that the character's %s was an impressive, exceptional, or decisive success — going beyond merely achieving the goal?", actionLabel)
	default:
		return "Does this narrative describe the result of an action?"
	}
}

// countSentences estimates sentence count by counting sentence-ending punctuation.
func countSentences(text string) int {
	count := 0
	for _, r := range text {
		if r == '.' || r == '!' || r == '?' {
			count++
		}
	}
	// Handle edge case: ellipsis "..." counts as one
	count -= strings.Count(text, "...") * 2
	if count < 1 && len(strings.TrimSpace(text)) > 0 {
		count = 1
	}
	return count
}

// --- Main test function ---

// TestActionNarrative_LLMEvaluation verifies that the action narrative prompt
// produces outcome-appropriate, concise, and non-mechanical narratives for
// Overcome and Create Advantage actions outside of conflict.
//
// Per Fate Core SRD, the four outcomes (Fail, Tie, Succeed, Succeed with Style)
// each have distinct narrative weight that the GM narration should reflect.
func TestActionNarrative_LLMEvaluation(t *testing.T) {
	client := RequireLLMClient(t)
	ctx := context.Background()

	verboseLogging := VerboseLoggingEnabled()

	overcomeTests := getOvercomeNarrativeTestCases()
	caATests := getCreateAdvantageNarrativeTestCases()
	allTests := append(overcomeTests, caATests...)

	var results []ActionNarrativeEvalResult
	passed := 0

	for _, tc := range allTests {
		t.Run(tc.Name, func(t *testing.T) {
			result := evaluateActionNarrative(ctx, client, tc)
			results = append(results, result)

			if result.Error != nil {
				t.Errorf("Error: %v", result.Error)
				return
			}

			if result.Passed {
				passed++
			}

			if verboseLogging || !result.Passed {
				status := "PASS"
				if !result.Passed {
					status = "FAIL"
				}
				t.Logf("%s: %s [%s %s → %s]",
					status, tc.Name, tc.ActionType.String(), tc.Skill, tc.OutcomeType.String())
				t.Logf("  Narrative: %s", truncateNarrative(result.RawResponse, 300))
				t.Logf("  OutcomeMatch=%v Concise=%v NoCombat=%v",
					result.OutcomeMatch, result.Concise, result.NoCombatJargon)
				t.Logf("  Judge(outcome): pass=%v — %s",
					result.JudgeOutcomeResult.Pass, result.JudgeOutcomeResult.Reasoning)
				t.Logf("  Judge(combat):  hasMechanics=%v — %s",
					result.JudgeCombatResult.Pass, result.JudgeCombatResult.Reasoning)
			}

			assert.True(t, result.OutcomeMatch,
				"Narrative should match %s outcome tone", tc.OutcomeType.String())
			assert.True(t, result.NoCombatJargon,
				"Non-attack narrative should not contain combat mechanical jargon")
			assert.True(t, result.Concise,
				"Narrative should be 1-3 sentences (got ~%d)", countSentences(result.RawResponse))
		})
	}

	// === Summary ===
	t.Log("\n========== ACTION NARRATIVE EVALUATION SUMMARY ==========")
	t.Logf("Passed: %d/%d (%.1f%%)", passed, len(allTests), float64(passed)*100/float64(len(allTests)))

	// By action type
	byType := map[action.ActionType]struct{ pass, total int }{}
	for _, r := range results {
		stats := byType[r.TestCase.ActionType]
		stats.total++
		if r.Passed {
			stats.pass++
		}
		byType[r.TestCase.ActionType] = stats
	}
	t.Log("\nBy Action Type:")
	for at, stats := range byType {
		t.Logf("  %s: %d/%d (%.1f%%)", at.String(), stats.pass, stats.total, float64(stats.pass)*100/float64(stats.total))
	}

	// By outcome
	byOutcome := map[dice.OutcomeType]struct{ pass, total int }{}
	for _, r := range results {
		stats := byOutcome[r.TestCase.OutcomeType]
		stats.total++
		if r.Passed {
			stats.pass++
		}
		byOutcome[r.TestCase.OutcomeType] = stats
	}
	t.Log("\nBy Outcome:")
	for ot, stats := range byOutcome {
		t.Logf("  %s: %d/%d (%.1f%%)", ot.String(), stats.pass, stats.total, float64(stats.pass)*100/float64(stats.total))
	}

	// Failed cases
	t.Log("\n--- Failed Cases ---")
	anyFailed := false
	for _, r := range results {
		if !r.Passed && r.Error == nil {
			anyFailed = true
			t.Logf("FAIL: %s [%s → %s] — outcome=%v concise=%v noCombat=%v",
				r.TestCase.Name, r.TestCase.ActionType.String(), r.TestCase.OutcomeType.String(),
				r.OutcomeMatch, r.Concise, r.NoCombatJargon)
			t.Logf("      Narrative: %s", truncateNarrative(r.RawResponse, 200))
		}
	}
	if !anyFailed {
		t.Log("  (none)")
	}
}

func truncateNarrative(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
