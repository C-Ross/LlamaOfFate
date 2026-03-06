//go:build llmeval

package llmeval_test

import (
	"context"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/stretchr/testify/assert"
)

// OppositionTestCase represents a test case for active vs passive opposition classification
type OppositionTestCase struct {
	Name               string
	RawInput           string
	Context            string
	OtherCharacters    []*core.Character
	ExpectedOpposition string   // "active" or "passive"
	ValidOpposingNPCs  []string // Acceptable NPC IDs if active (empty for passive)
	ValidOpposingSkill []string // Acceptable opposing skills if active (empty for passive)
	ExpectedType       action.ActionType
	Description        string
}

// OppositionEvalResult stores the result of an opposition eval
type OppositionEvalResult struct {
	TestCase              OppositionTestCase
	ActualType            action.ActionType
	ActualOpposingNPCID   string
	ActualOpposingSkill   string
	TypeMatches           bool
	OppositionCorrect     bool // Did the LLM pick the right opposition type?
	OpposingNPCAcceptable bool // Is the opposing NPC one of the valid targets?
	OpposingSkillMatches  bool // Did the LLM pick an acceptable opposing skill?
	Error                 error
}

// getOppositionNPCs creates NPCs with distinct skills for opposition testing
func getOppositionNPCs() []*core.Character {
	guard := core.NewCharacter("tavern-guard", "Tavern Guard")
	guard.Aspects.HighConcept = "Alert Doorman"
	guard.Aspects.Trouble = "Easily Bribed"
	guard.SetSkill("Notice", dice.Good)
	guard.SetSkill("Fight", dice.Fair)
	guard.SetSkill("Physique", dice.Fair)
	guard.SetSkill("Will", dice.Average)

	merchant := core.NewCharacter("scene_1_npc_0", "Grizzled Merchant")
	merchant.Aspects.HighConcept = "Shrewd Trader With a Sharp Eye"
	merchant.Aspects.Trouble = "Trusts No One"
	merchant.SetSkill("Empathy", dice.Good)
	merchant.SetSkill("Rapport", dice.Fair)
	merchant.SetSkill("Will", dice.Good)
	merchant.SetSkill("Notice", dice.Fair)
	merchant.SetSkill("Deceive", dice.Average)

	librarian := core.NewCharacter("old-librarian", "Elder Librarian")
	librarian.Aspects.HighConcept = "Keeper of Forbidden Knowledge"
	librarian.Aspects.Trouble = "Fiercely Protective of the Collection"
	librarian.SetSkill("Lore", dice.Great)
	librarian.SetSkill("Will", dice.Good)
	librarian.SetSkill("Notice", dice.Fair)

	return []*core.Character{guard, merchant, librarian}
}

// getActiveOppositionTestCases returns test cases where an NPC should actively oppose
func getActiveOppositionTestCases() []OppositionTestCase {
	npcs := getOppositionNPCs()
	return []OppositionTestCase{
		{
			Name:               "Sneak past alert guard",
			RawInput:           "I try to sneak past the Tavern Guard without being noticed",
			Context:            "The Tavern Guard stands watch at the entrance to the back rooms, scanning the crowd",
			OtherCharacters:    npcs,
			ExpectedOpposition: "active",
			ValidOpposingNPCs:  []string{"tavern-guard"},
			ValidOpposingSkill: []string{"Notice"},
			ExpectedType:       action.Overcome,
			Description:        "Sneaking past an alert guard — Guard opposes with Notice",
		},
		{
			Name:               "Lie to suspicious merchant",
			RawInput:           "I tell the Grizzled Merchant that these gems are worth twice what they are",
			Context:            "Negotiating in the Grizzled Merchant's shop; he watches you with narrowed eyes",
			OtherCharacters:    npcs,
			ExpectedOpposition: "active",
			ValidOpposingNPCs:  []string{"scene_1_npc_0"},
			ValidOpposingSkill: []string{"Empathy", "Notice", "Deceive"},
			ExpectedType:       action.Overcome,
			Description:        "Lying to a merchant with Empathy — NPC actively resists with Empathy",
		},
		{
			Name:               "Fast-talk the merchant past his suspicion",
			RawInput:           "I try to smooth-talk the Grizzled Merchant into giving me a discount on the supplies",
			Context:            "Haggling with the Grizzled Merchant at his market stall; he looks unimpressed",
			OtherCharacters:    npcs,
			ExpectedOpposition: "active",
			ValidOpposingNPCs:  []string{"scene_1_npc_0"},
			ValidOpposingSkill: []string{"Will", "Rapport", "Empathy"},
			ExpectedType:       action.Overcome,
			Description:        "Persuading a stubborn merchant — NPC actively resists with Will or Rapport",
		},
		{
			Name:               "Persuade librarian to share forbidden book",
			RawInput:           "I try to convince the Elder Librarian to let me see the restricted section",
			Context:            "The Elder Librarian guards the entrance to the restricted archives jealously",
			OtherCharacters:    npcs,
			ExpectedOpposition: "active",
			ValidOpposingNPCs:  []string{"old-librarian"},
			ValidOpposingSkill: []string{"Will", "Lore"},
			ExpectedType:       action.Overcome,
			Description:        "Persuading a protective librarian — she resists with Will",
		},
		{
			Name:               "Distract the guard to create opening",
			RawInput:           "I create a distraction to draw the Tavern Guard away from the door",
			Context:            "The Tavern Guard watches the restricted hallway entrance attentively",
			OtherCharacters:    npcs,
			ExpectedOpposition: "active",
			ValidOpposingNPCs:  []string{"tavern-guard"},
			ValidOpposingSkill: []string{"Notice", "Will"},
			ExpectedType:       action.CreateAdvantage,
			Description:        "Creating a distraction against an alert guard — opposed by Notice",
		},
		{
			Name:               "Feint to fool the guard",
			RawInput:           "I pretend to stumble and drop my bag to distract the Tavern Guard while my partner slips through",
			Context:            "The Tavern Guard watches the restricted hallway; your partner waits nearby",
			OtherCharacters:    npcs,
			ExpectedOpposition: "active",
			ValidOpposingNPCs:  []string{"tavern-guard"},
			ValidOpposingSkill: []string{"Notice", "Empathy", "Will"},
			ExpectedType:       action.CreateAdvantage,
			Description:        "Feint to deceive a watchful guard — guard opposes with Notice or Empathy",
		},
	}
}

// getPassiveOppositionTestCases returns test cases where no NPC should actively oppose
func getPassiveOppositionTestCases() []OppositionTestCase {
	npcs := getOppositionNPCs()
	return []OppositionTestCase{
		{
			Name:               "Climb the wall — no NPC involved",
			RawInput:           "I climb up the stone wall to reach the balcony",
			Context:            "A tall stone wall separates the courtyard from the upper level",
			OtherCharacters:    npcs,
			ExpectedOpposition: "passive",
			ValidOpposingNPCs:  []string{},
			ValidOpposingSkill: []string{},
			ExpectedType:       action.Overcome,
			Description:        "Environmental obstacle — no NPC opposes, use static difficulty",
		},
		{
			Name:               "Pick a lock — no NPC watching",
			RawInput:           "I pick the lock on the treasure chest",
			Context:            "Alone in the storage room with a locked iron chest",
			OtherCharacters:    npcs,
			ExpectedOpposition: "passive",
			ValidOpposingNPCs:  []string{},
			ValidOpposingSkill: []string{},
			ExpectedType:       action.Overcome,
			Description:        "Lock picking with no one around — passive difficulty",
		},
		{
			Name:               "Decipher ancient text — environment only",
			RawInput:           "I try to decipher the runes carved into the altar",
			Context:            "Standing alone in an ancient temple, examining the stone altar",
			OtherCharacters:    npcs,
			ExpectedOpposition: "passive",
			ValidOpposingNPCs:  []string{},
			ValidOpposingSkill: []string{},
			ExpectedType:       action.Overcome,
			Description:        "Scholarly puzzle — no NPC opposes, just flat difficulty",
		},
		{
			Name:               "Swim across the river",
			RawInput:           "I swim across the raging river to the other bank",
			Context:            "A swollen river blocks the path; the current looks strong",
			OtherCharacters:    npcs,
			ExpectedOpposition: "passive",
			ValidOpposingNPCs:  []string{},
			ValidOpposingSkill: []string{},
			ExpectedType:       action.Overcome,
			Description:        "Physical environment obstacle — passive opposition",
		},
		{
			Name:               "Set a trap in empty room",
			RawInput:           "I rig a tripwire across the doorway for whoever comes through next",
			Context:            "Alone in the abandoned warehouse, preparing for intruders",
			OtherCharacters:    npcs,
			ExpectedOpposition: "passive",
			ValidOpposingNPCs:  []string{},
			ValidOpposingSkill: []string{},
			ExpectedType:       action.CreateAdvantage,
			Description:        "Setting a trap with no NPC watching — passive difficulty",
		},
	}
}

// getNarrativeHazardTestCases returns test cases where a hazard is described in
// the narrative/context but is NOT an NPC in the scene. An unrelated NPC IS present.
// The LLM must choose passive opposition — not drag in the unrelated NPC.
// This reproduces the real-world bug: Zero tried to dodge a drone (set dressing),
// but the LLM picked Raven (an info broker NPC) as active opposition.
func getNarrativeHazardTestCases() []OppositionTestCase {
	// Only NPC is an info broker — unrelated to the hazard
	raven := core.NewCharacter("scene_1_npc_0", "Raven")
	raven.Aspects.HighConcept = "Street-Savvy Information Broker"
	// intentionally no skills set — skills: {}

	npcs := []*core.Character{raven}

	return []OppositionTestCase{
		{
			Name:               "Dodge security drone — NPC is unrelated broker",
			RawInput:           "I dodge the security drone's spotlight and slip through the crowd",
			Context:            "Rain-soaked streets outside NeoTech Tower. A security patrol drone swoops down, its spotlight scanning the crowd. Raven's voice crackles in your earpiece giving directions.",
			OtherCharacters:    npcs,
			ExpectedOpposition: "passive",
			ValidOpposingNPCs:  []string{},
			ValidOpposingSkill: []string{},
			ExpectedType:       action.Overcome,
			Description:        "Drone is narrative set dressing, not an NPC. Raven is an ally on comms. Must be passive.",
		},
		{
			Name:               "Sneak past cameras — NPC is friendly contact",
			RawInput:           "I creep along the wall to avoid the security cameras",
			Context:            "A corridor lined with security cameras. Raven monitors the feed remotely and warns you about blind spots.",
			OtherCharacters:    npcs,
			ExpectedOpposition: "passive",
			ValidOpposingNPCs:  []string{},
			ValidOpposingSkill: []string{},
			ExpectedType:       action.Overcome,
			Description:        "Cameras aren't NPCs. Raven is helping, not opposing. Must be passive.",
		},
		{
			Name:               "Disable alarm system — NPC not involved in action",
			RawInput:           "I hack the alarm panel to disable it before it triggers",
			Context:            "Standing at a security panel in a corporate hallway. Raven waits in the getaway vehicle outside.",
			OtherCharacters:    npcs,
			ExpectedOpposition: "passive",
			ValidOpposingNPCs:  []string{},
			ValidOpposingSkill: []string{},
			ExpectedType:       action.Overcome,
			Description:        "Alarm panel is environmental. Raven is elsewhere. Must be passive.",
		},
	}
}

// evaluateOppositionTestCase runs a single opposition test case
func evaluateOppositionTestCase(ctx context.Context, parser engine.ActionParser, char *core.Character, tc OppositionTestCase) OppositionEvalResult {
	req := engine.ActionParseRequest{
		Character:       char,
		RawInput:        tc.RawInput,
		Context:         tc.Context,
		OtherCharacters: tc.OtherCharacters,
	}

	parsedAction, err := parser.ParseAction(ctx, req)
	if err != nil {
		return OppositionEvalResult{
			TestCase: tc,
			Error:    err,
		}
	}

	hasActiveOpposition := parsedAction.OpposingNPCID != ""

	oppositionCorrect := false
	if tc.ExpectedOpposition == "active" {
		oppositionCorrect = hasActiveOpposition
	} else {
		oppositionCorrect = !hasActiveOpposition
	}

	npcAcceptable := false
	if len(tc.ValidOpposingNPCs) == 0 {
		// Passive — NPC ID should be empty
		npcAcceptable = parsedAction.OpposingNPCID == ""
	} else {
		for _, valid := range tc.ValidOpposingNPCs {
			if strings.EqualFold(parsedAction.OpposingNPCID, valid) {
				npcAcceptable = true
				break
			}
		}
	}

	skillMatches := false
	if len(tc.ValidOpposingSkill) == 0 {
		// Passive — skill should be empty
		skillMatches = parsedAction.OpposingSkill == ""
	} else {
		for _, valid := range tc.ValidOpposingSkill {
			if strings.EqualFold(parsedAction.OpposingSkill, valid) {
				skillMatches = true
				break
			}
		}
	}

	return OppositionEvalResult{
		TestCase:              tc,
		ActualType:            parsedAction.Type,
		ActualOpposingNPCID:   parsedAction.OpposingNPCID,
		ActualOpposingSkill:   parsedAction.OpposingSkill,
		TypeMatches:           parsedAction.Type == tc.ExpectedType,
		OppositionCorrect:     oppositionCorrect,
		OpposingNPCAcceptable: npcAcceptable,
		OpposingSkillMatches:  skillMatches,
		Error:                 nil,
	}
}

// TestActionParser_OppositionClassification validates that the LLM correctly
// identifies when an NPC should actively oppose a player action vs when static
// difficulty should be used (Fate Core: active vs passive opposition).
//
// Run with: go test -v -tags=llmeval -run TestActionParser_OppositionClassification ./test/llmeval/ -timeout 5m
func TestActionParser_OppositionClassification(t *testing.T) {
	client := RequireLLMClient(t)
	parser := engine.NewActionParser(client)
	char := getTestCharacter()
	ctx := context.Background()
	verboseLogging := VerboseLoggingEnabled()

	allTestCases := []struct {
		category string
		cases    []OppositionTestCase
	}{
		{"ActiveOpposition", getActiveOppositionTestCases()},
		{"PassiveOpposition", getPassiveOppositionTestCases()},
		{"NarrativeHazards", getNarrativeHazardTestCases()},
	}

	var results []OppositionEvalResult
	totalTests := 0
	typeCorrect := 0
	oppositionCorrect := 0
	npcCorrect := 0
	skillCorrect := 0

	for _, category := range allTestCases {
		t.Run(category.category, func(t *testing.T) {
			for _, tc := range category.cases {
				t.Run(tc.Name, func(t *testing.T) {
					result := evaluateOppositionTestCase(ctx, parser, char, tc)
					results = append(results, result)

					if result.Error != nil {
						t.Errorf("Error: %v", result.Error)
						return
					}

					totalTests++

					if result.TypeMatches {
						typeCorrect++
					}
					if result.OppositionCorrect {
						oppositionCorrect++
					}
					if result.OpposingNPCAcceptable {
						npcCorrect++
					}
					if result.OpposingSkillMatches {
						skillCorrect++
					}

					if verboseLogging || !result.OppositionCorrect {
						oppStatus := "✓"
						if !result.OppositionCorrect {
							oppStatus = "✗"
						}
						typeStatus := "✓"
						if !result.TypeMatches {
							typeStatus = "✗"
						}
						t.Logf("%s Opposition: expected=%s, got_npc='%s', got_skill='%s'",
							oppStatus, tc.ExpectedOpposition, result.ActualOpposingNPCID, result.ActualOpposingSkill)
						t.Logf("%s Type: expected=%s, got=%s",
							typeStatus, tc.ExpectedType, result.ActualType)
					}

					// Assertions
					assert.Equal(t, tc.ExpectedType, result.ActualType,
						"Action type mismatch for '%s'", tc.RawInput)
					assert.True(t, result.OppositionCorrect,
						"Opposition type mismatch for '%s': expected %s, got npc_id='%s' skill='%s'",
						tc.RawInput, tc.ExpectedOpposition, result.ActualOpposingNPCID, result.ActualOpposingSkill)
					if tc.ExpectedOpposition == "active" {
						assert.True(t, result.OpposingNPCAcceptable,
							"Opposing NPC mismatch for '%s': got '%s', expected one of %v",
							tc.RawInput, result.ActualOpposingNPCID, tc.ValidOpposingNPCs)
						assert.True(t, result.OpposingSkillMatches,
							"Opposing skill mismatch for '%s': got '%s', expected one of %v",
							tc.RawInput, result.ActualOpposingSkill, tc.ValidOpposingSkill)
					}
				})
			}
		})
	}

	// Print summary
	t.Log("\n========== OPPOSITION CLASSIFICATION SUMMARY ==========")
	t.Logf("Total Tests:        %d", totalTests)
	t.Logf("Type Accuracy:      %d/%d (%.1f%%)",
		typeCorrect, totalTests, safePercent(typeCorrect, totalTests))
	t.Logf("Opposition Accuracy: %d/%d (%.1f%%)",
		oppositionCorrect, totalTests, safePercent(oppositionCorrect, totalTests))
	t.Logf("NPC ID Accuracy:    %d/%d (%.1f%%)",
		npcCorrect, totalTests, safePercent(npcCorrect, totalTests))
	t.Logf("Skill Accuracy:     %d/%d (%.1f%%)",
		skillCorrect, totalTests, safePercent(skillCorrect, totalTests))

	// Print failed cases
	t.Log("\n--- Failed Cases ---")
	failCount := 0
	for _, r := range results {
		if r.Error != nil || !r.OppositionCorrect || !r.OpposingNPCAcceptable || !r.OpposingSkillMatches {
			failCount++
			t.Logf("FAIL: '%s'", r.TestCase.RawInput)
			t.Logf("      Expected: opposition=%s, npcs=%v, skills=%v",
				r.TestCase.ExpectedOpposition, r.TestCase.ValidOpposingNPCs, r.TestCase.ValidOpposingSkill)
			t.Logf("      Got:      npc_id='%s', skill='%s'",
				r.ActualOpposingNPCID, r.ActualOpposingSkill)
			t.Logf("      Reason:   %s", r.TestCase.Description)
			if r.Error != nil {
				t.Logf("      Error:    %v", r.Error)
			}
		}
	}
	if failCount == 0 {
		t.Log("  (none)")
	}
}

// safePercent computes a percentage avoiding division by zero
func safePercent(num, denom int) float64 {
	if denom == 0 {
		return 0
	}
	return float64(num) * 100 / float64(denom)
}
