package terminal

import (
	"bufio"
	"strings"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSceneInfo implements uicontract.SceneInfo for testing.
type mockSceneInfo struct {
	scene               *scene.Scene
	player              *core.Character
	conversationHistory []uicontract.ConversationEntry
}

func (m *mockSceneInfo) GetCurrentScene() *scene.Scene { return m.scene }
func (m *mockSceneInfo) GetPlayer() *core.Character    { return m.player }
func (m *mockSceneInfo) GetConversationHistory() []uicontract.ConversationEntry {
	return m.conversationHistory
}

func TestNewTerminalUI(t *testing.T) {
	ui := NewTerminalUI()
	require.NotNil(t, ui)
	require.NotNil(t, ui.reader)
}

func TestTerminalUI_isExitCommand(t *testing.T) {
	ui := NewTerminalUI()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "exit command",
			input:    "exit",
			expected: true,
		},
		{
			name:     "quit command",
			input:    "quit",
			expected: true,
		},
		{
			name:     "end command",
			input:    "end",
			expected: true,
		},
		{
			name:     "leave command",
			input:    "leave",
			expected: true,
		},
		{
			name:     "resolve command",
			input:    "resolve",
			expected: true,
		},
		{
			name:     "uppercase exit",
			input:    "EXIT",
			expected: true,
		},
		{
			name:     "mixed case quit",
			input:    "QuIt",
			expected: true,
		},
		{
			name:     "exit with whitespace",
			input:    "  exit  ",
			expected: true,
		},
		{
			name:     "regular command",
			input:    "attack goblin",
			expected: false,
		},
		{
			name:     "help command",
			input:    "help",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "partial match",
			input:    "exiting",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.isExitCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTerminalUI_EmitMethods(t *testing.T) {
	ui := NewTerminalUI()

	// These events should not panic when emitted
	// We can't easily test the actual output without mocking stdout
	// but we can ensure they execute without errors
	require.NotPanics(t, func() {
		ui.Emit(uicontract.ActionAttemptEvent{Description: "Attack the goblin"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ActionResultEvent{Skill: "Fight", SkillRank: "Good", SkillBonus: 3, Bonuses: 2, Result: "Fair (+2)", Outcome: "Success"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NarrativeEvent{Text: "You successfully strike the goblin!"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.DialogEvent{PlayerInput: "Hello there", GMResponse: "The goblin grunts in response"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.SystemMessageEvent{Message: "Scene started"})
	})
}

func TestHandleSpecialCommands_Recognized(t *testing.T) {
	ui := NewTerminalUI()
	player := core.NewCharacter("player1", "Test Hero")
	player.Aspects.HighConcept = "Brave Warrior"
	player.Aspects.Trouble = "Quick to Anger"

	testScene := scene.NewScene("scene1", "Test Scene", "A dark room")
	testScene.AddSituationAspect(scene.SituationAspect{Aspect: "Dim Lighting", FreeInvokes: 1})

	ui.SetSceneInfo(&mockSceneInfo{
		scene:  testScene,
		player: player,
	})

	recognized := []string{
		"help", "?",
		"scene",
		"character", "char", "me",
		"status",
		"aspects",
		"history", "conversation",
	}

	for _, cmd := range recognized {
		t.Run(cmd, func(t *testing.T) {
			require.NotPanics(t, func() {
				handled := ui.handleSpecialCommands(cmd)
				assert.True(t, handled, "command %q should be handled", cmd)
			})
		})
	}
}

func TestHandleSpecialCommands_NotRecognized(t *testing.T) {
	ui := NewTerminalUI()
	player := core.NewCharacter("player1", "Test Hero")

	ui.SetSceneInfo(&mockSceneInfo{
		scene:  scene.NewScene("s1", "S", "D"),
		player: player,
	})

	unrecognized := []string{
		"attack goblin",
		"look around",
		"",
		"exit",
	}

	for _, cmd := range unrecognized {
		t.Run(cmd, func(t *testing.T) {
			handled := ui.handleSpecialCommands(cmd)
			assert.False(t, handled, "command %q should not be handled", cmd)
		})
	}
}

func TestHandleSpecialCommands_CaseInsensitive(t *testing.T) {
	ui := NewTerminalUI()
	ui.SetSceneInfo(&mockSceneInfo{
		scene:  scene.NewScene("s1", "S", "D"),
		player: core.NewCharacter("p1", "P"),
	})

	assert.True(t, ui.handleSpecialCommands("HELP"))
	assert.True(t, ui.handleSpecialCommands("Scene"))
	assert.True(t, ui.handleSpecialCommands("CHARACTER"))
	assert.True(t, ui.handleSpecialCommands("Status"))
	assert.True(t, ui.handleSpecialCommands("ASPECTS"))
	assert.True(t, ui.handleSpecialCommands("History"))
}

// --- PromptForMidFlow tests ---

func newUIWithInput(input string) *TerminalUI {
	ui := NewTerminalUI()
	ui.reader = bufio.NewReader(strings.NewReader(input))
	return ui
}

func TestPromptForMidFlow_NumberedChoice_ValidSelection(t *testing.T) {
	ui := newUIWithInput("2\n")

	event := uicontract.InputRequestEvent{
		Type:   uicontract.InputRequestNumberedChoice,
		Prompt: "Choose how to handle 3 shifts:",
		Options: []uicontract.InputOption{
			{Label: "Take a mild consequence", Description: "absorbs 2 shifts"},
			{Label: "Take a moderate consequence", Description: "absorbs 4 shifts"},
			{Label: "Be Taken Out"},
		},
	}

	resp := ui.PromptForMidFlow(event)
	assert.Equal(t, 1, resp.ChoiceIndex, "input '2' should map to 0-based index 1")
}

func TestPromptForMidFlow_NumberedChoice_FirstOption(t *testing.T) {
	ui := newUIWithInput("1\n")

	event := uicontract.InputRequestEvent{
		Type:   uicontract.InputRequestNumberedChoice,
		Prompt: "Choose:",
		Options: []uicontract.InputOption{
			{Label: "Option A"},
			{Label: "Option B"},
		},
	}

	resp := ui.PromptForMidFlow(event)
	assert.Equal(t, 0, resp.ChoiceIndex)
}

func TestPromptForMidFlow_NumberedChoice_LastOption(t *testing.T) {
	ui := newUIWithInput("3\n")

	event := uicontract.InputRequestEvent{
		Type:   uicontract.InputRequestNumberedChoice,
		Prompt: "Choose:",
		Options: []uicontract.InputOption{
			{Label: "Option A"},
			{Label: "Option B"},
			{Label: "Option C"},
		},
	}

	resp := ui.PromptForMidFlow(event)
	assert.Equal(t, 2, resp.ChoiceIndex)
}

func TestPromptForMidFlow_NumberedChoice_InvalidInput_DefaultsToLast(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"out of range high", "99\n"},
		{"zero", "0\n"},
		{"negative", "-1\n"},
		{"non-numeric", "abc\n"},
		{"empty", "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := newUIWithInput(tt.input)

			event := uicontract.InputRequestEvent{
				Type:   uicontract.InputRequestNumberedChoice,
				Prompt: "Choose:",
				Options: []uicontract.InputOption{
					{Label: "Option A"},
					{Label: "Option B"},
					{Label: "Be Taken Out"},
				},
			}

			resp := ui.PromptForMidFlow(event)
			assert.Equal(t, 2, resp.ChoiceIndex, "invalid input should default to last option")
		})
	}
}

func TestPromptForMidFlow_NumberedChoice_ReadError_DefaultsToLast(t *testing.T) {
	// Empty reader simulates a read error (EOF).
	ui := newUIWithInput("")

	event := uicontract.InputRequestEvent{
		Type:   uicontract.InputRequestNumberedChoice,
		Prompt: "Choose:",
		Options: []uicontract.InputOption{
			{Label: "Option A"},
			{Label: "Option B"},
		},
	}

	resp := ui.PromptForMidFlow(event)
	assert.Equal(t, 1, resp.ChoiceIndex, "read error should default to last option")
}

func TestPromptForMidFlow_FreeText(t *testing.T) {
	ui := newUIWithInput("I surrender and drop my sword.\n")

	event := uicontract.InputRequestEvent{
		Type:   uicontract.InputRequestFreeText,
		Prompt: "Describe how you concede:",
	}

	resp := ui.PromptForMidFlow(event)
	assert.Equal(t, "I surrender and drop my sword.", resp.Text)
}

func TestPromptForMidFlow_FreeText_TrimsWhitespace(t *testing.T) {
	ui := newUIWithInput("  some text with spaces  \n")

	event := uicontract.InputRequestEvent{
		Type:   uicontract.InputRequestFreeText,
		Prompt: "Enter text:",
	}

	resp := ui.PromptForMidFlow(event)
	assert.Equal(t, "some text with spaces", resp.Text)
}

func TestPromptForMidFlow_FreeText_Empty(t *testing.T) {
	ui := newUIWithInput("\n")

	event := uicontract.InputRequestEvent{
		Type:   uicontract.InputRequestFreeText,
		Prompt: "Enter text:",
	}

	resp := ui.PromptForMidFlow(event)
	assert.Equal(t, "", resp.Text)
}

func TestPromptForMidFlow_FreeText_ReadError(t *testing.T) {
	ui := newUIWithInput("")

	event := uicontract.InputRequestEvent{
		Type:   uicontract.InputRequestFreeText,
		Prompt: "Enter text:",
	}

	resp := ui.PromptForMidFlow(event)
	assert.Equal(t, "", resp.Text)
}

func TestPromptForMidFlow_UnknownType_ReturnsZeroValue(t *testing.T) {
	ui := newUIWithInput("")

	event := uicontract.InputRequestEvent{
		Type:   "unknown_type",
		Prompt: "Something:",
	}

	resp := ui.PromptForMidFlow(event)
	assert.Equal(t, uicontract.MidFlowResponse{}, resp)
}

// --- Emit coverage for event types not tested above ---

func TestEmit_ConflictEvents(t *testing.T) {
	ui := NewTerminalUI()

	participants := []uicontract.ConflictParticipantInfo{
		{CharacterName: "Hero", Initiative: 5, IsPlayer: true},
		{CharacterName: "Goblin", Initiative: 3, IsPlayer: false},
	}

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ConflictStartEvent{
			ConflictType:  "physical",
			InitiatorName: "Goblin",
			Participants:  participants,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ConflictEscalationEvent{
			FromType:        "social",
			ToType:          "physical",
			TriggerCharName: "Goblin",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.TurnAnnouncementEvent{CharacterName: "Hero", TurnNumber: 1, IsPlayer: true})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.TurnAnnouncementEvent{CharacterName: "Goblin", TurnNumber: 1, IsPlayer: false})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ConflictEndEvent{Reason: "The goblin surrenders."})
	})
}

func TestEmit_DamageAndAttackEvents(t *testing.T) {
	ui := NewTerminalUI()

	require.NotPanics(t, func() {
		ui.Emit(uicontract.DefenseRollEvent{DefenderName: "Goblin", Skill: "Athletics", Result: "Fair (+2)"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.DamageResolutionEvent{
			TargetName: "Goblin",
			Absorbed:   &uicontract.StressAbsorptionDetail{TrackType: "physical", Shifts: 2, TrackState: "[X][ ][ ]"},
			TakenOut:   false,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.DamageResolutionEvent{
			TargetName: "Goblin",
			Consequence: &uicontract.ConsequenceDetail{
				Severity: "mild",
				Aspect:   "Bruised Arm",
				Absorbed: 2,
			},
			TakenOut:   true,
			VictoryEnd: true,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.DamageResolutionEvent{
			TargetName:        "Goblin",
			RemainingAbsorbed: &uicontract.StressAbsorptionDetail{TrackType: "physical", Shifts: 1},
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerAttackResultEvent{Shifts: 3, TargetName: "Goblin"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerAttackResultEvent{IsTie: true})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerAttackResultEvent{TargetMissing: true, TargetHint: "Dragon"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NPCAttackEvent{
			AttackerName:   "Goblin",
			TargetName:     "Hero",
			AttackSkill:    "Fight",
			AttackResult:   "Good (+3)",
			DefenseSkill:   "Athletics",
			DefenseResult:  "Fair (+2)",
			FullDefense:    false,
			InitialOutcome: "Success",
			FinalOutcome:   "Success",
			Narrative:      "The goblin slashes!",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NPCAttackEvent{
			AttackerName:   "Goblin",
			TargetName:     "Hero",
			AttackSkill:    "Fight",
			AttackResult:   "Good (+3)",
			DefenseSkill:   "Athletics",
			DefenseResult:  "Mediocre (+0)",
			FullDefense:    true,
			InitialOutcome: "Success",
			FinalOutcome:   "Failure",
		})
	})
}

func TestEmit_PlayerStatusEvents(t *testing.T) {
	ui := NewTerminalUI()

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerStressEvent{Shifts: 2, StressType: "physical", TrackState: "[X][X][ ]"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerDefendedEvent{IsTie: false})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerDefendedEvent{IsTie: true})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerConsequenceEvent{
			Severity: "mild",
			Aspect:   "Bruised Ribs",
			Absorbed: 2,
			StressAbsorbed: &uicontract.StressAbsorptionDetail{
				Shifts:     1,
				TrackState: "[X][ ][ ]",
			},
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerConsequenceEvent{
			Severity: "moderate",
			Aspect:   "Broken Arm",
			Absorbed: 4,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerTakenOutEvent{
			AttackerName: "Goblin",
			Narrative:    "You are defeated.",
			Outcome:      "game_over",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerTakenOutEvent{
			AttackerName: "Goblin",
			Narrative:    "You are dragged away.",
			Outcome:      "transition",
			NewSceneHint: "Prison cell",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.PlayerTakenOutEvent{
			AttackerName: "Goblin",
			Narrative:    "You retreat.",
			Outcome:      "continue",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ConcessionEvent{
			FatePointsGained:  2,
			ConsequenceCount:  1,
			CurrentFatePoints: 5,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ConcessionEvent{
			FatePointsGained:  1,
			ConsequenceCount:  0,
			CurrentFatePoints: 4,
		})
	})
}

func TestEmit_AspectAndInvokeEvents(t *testing.T) {
	ui := NewTerminalUI()

	require.NotPanics(t, func() {
		ui.Emit(uicontract.AspectCreatedEvent{AspectName: "On Fire!", FreeInvokes: 2, IsBoost: false})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.AspectCreatedEvent{AspectName: "Distracted", FreeInvokes: 1, IsBoost: true})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.BoostExpiredEvent{AspectName: "Distracted"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.OutcomeChangedEvent{FinalOutcome: "Success with Style"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.InvokeEvent{
			AspectName:     "On Fire!",
			IsFree:         true,
			IsReroll:       false,
			FatePointsLeft: 3,
			NewTotal:       "Great (+4)",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.InvokeEvent{
			AspectName:     "On Fire!",
			IsFree:         false,
			IsReroll:       true,
			FatePointsLeft: 2,
			NewRoll:        "+2",
			NewTotal:       "Superb (+5)",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.InvokeEvent{Failed: true})
	})
}

func TestEmit_NPCActionResultEvent(t *testing.T) {
	ui := NewTerminalUI()

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NPCActionResultEvent{NPCName: "Goblin", ActionType: "defend"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NPCActionResultEvent{
			NPCName:       "Goblin",
			ActionType:    "create_advantage",
			Skill:         "Deceive",
			RollResult:    "Good (+3)",
			Difficulty:    "Fair (+2)",
			Outcome:       "Success",
			AspectCreated: "Flanked",
			FreeInvokes:   1,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NPCActionResultEvent{
			NPCName:    "Goblin",
			ActionType: "create_advantage",
			Skill:      "Deceive",
			RollResult: "Fair (+2)",
			Difficulty: "Fair (+2)",
			Outcome:    "Tie",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NPCActionResultEvent{
			NPCName:    "Goblin",
			ActionType: "create_advantage",
			Skill:      "Deceive",
			RollResult: "Poor (+0)",
			Difficulty: "Fair (+2)",
			Outcome:    "Failure",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NPCActionResultEvent{
			NPCName:    "Goblin",
			ActionType: "overcome",
			Skill:      "Athletics",
			RollResult: "Good (+3)",
			Difficulty: "Average (+1)",
			Outcome:    "Success with Style",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NPCActionResultEvent{
			NPCName:    "Goblin",
			ActionType: "overcome",
			Skill:      "Athletics",
			RollResult: "Fair (+2)",
			Difficulty: "Fair (+2)",
			Outcome:    "Tie",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NPCActionResultEvent{
			NPCName:    "Goblin",
			ActionType: "overcome",
			Skill:      "Athletics",
			RollResult: "Mediocre (+0)",
			Difficulty: "Fair (+2)",
			Outcome:    "Failure",
		})
	})
}

func TestEmit_RecoveryAndOverflowEvents(t *testing.T) {
	ui := NewTerminalUI()

	require.NotPanics(t, func() {
		ui.Emit(uicontract.RecoveryEvent{
			Action:   "healed",
			Aspect:   "Bruised Ribs",
			Severity: "mild",
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.RecoveryEvent{
			Action:     "roll",
			Aspect:     "Broken Arm",
			Severity:   "moderate",
			Skill:      "Physique",
			RollResult: 3,
			Difficulty: "Fair (+2)",
			Success:    true,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.RecoveryEvent{
			Action:     "roll",
			Aspect:     "Broken Arm",
			Severity:   "moderate",
			Skill:      "Physique",
			RollResult: 1,
			Difficulty: "Fair (+2)",
			Success:    false,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.StressOverflowEvent{Shifts: 3, NoConsequences: true})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.StressOverflowEvent{Shifts: 2, RemainingOverflow: true})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.StressOverflowEvent{Shifts: 4})
	})
}

func TestEmit_MilestoneAndResumedEvents(t *testing.T) {
	ui := NewTerminalUI()

	require.NotPanics(t, func() {
		ui.Emit(uicontract.MilestoneEvent{
			Type:          "scenario_complete",
			ScenarioTitle: "The Dark Tower",
			FatePoints:    3,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.GameResumedEvent{
			ScenarioTitle: "The Dark Tower",
			SceneName:     "The Entrance Hall",
		})
	})
}

func TestEmit_SceneAndGameEvents(t *testing.T) {
	ui := NewTerminalUI()

	require.NotPanics(t, func() {
		ui.Emit(uicontract.SceneTransitionEvent{Narrative: "You move on.", NewSceneHint: "Next area"})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.SceneTransitionEvent{Narrative: "You move on.", NewSceneHint: ""})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.GameOverEvent{Reason: "You were defeated."})
	})
}

func TestEmit_ChallengeEvents(t *testing.T) {
	ui := NewTerminalUI()

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ChallengeStartEvent{
			Description: "Escape the burning building!",
			Tasks: []uicontract.ChallengeTaskInfo{
				{Skill: "Athletics", Description: "Jump across the gap", Difficulty: "Good (+3)"},
				{Skill: "Physique", Description: "Break down the door", Difficulty: "Fair (+2)"},
			},
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ChallengeTaskResultEvent{
			Description: "Jump across the gap",
			Skill:       "Athletics",
			Outcome:     "succeeded",
			Shifts:      2,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ChallengeTaskResultEvent{
			Description: "Break down the door",
			Skill:       "Physique",
			Outcome:     "succeeded_with_style",
			Shifts:      3,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ChallengeTaskResultEvent{
			Description: "Convince the guard",
			Skill:       "Rapport",
			Outcome:     "tied",
			Shifts:      0,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ChallengeTaskResultEvent{
			Description: "Pick the lock",
			Skill:       "Burglary",
			Outcome:     "failed",
			Shifts:      -2,
		})
	})

	require.NotPanics(t, func() {
		ui.Emit(uicontract.ChallengeCompleteEvent{
			Successes: 2,
			Failures:  1,
			Ties:      0,
			Overall:   "partial",
		})
	})
}

func TestEmit_RecapAndRecoveryFlow(t *testing.T) {
	ui := NewTerminalUI()

	// A DialogEvent in recap mode followed by NarrativeEvent should close recap
	require.NotPanics(t, func() {
		ui.Emit(uicontract.DialogEvent{PlayerInput: "what happened?", GMResponse: "Recap info", IsRecap: true})
	})
	assert.True(t, ui.inRecap, "inRecap should be true after recap dialog")

	require.NotPanics(t, func() {
		ui.Emit(uicontract.NarrativeEvent{Text: "New scene starts."})
	})
	assert.False(t, ui.inRecap, "inRecap should be false after NarrativeEvent")

	// RecoveryEvent sets inRecovery; MilestoneEvent clears it
	require.NotPanics(t, func() {
		ui.Emit(uicontract.RecoveryEvent{Action: "healed", Aspect: "Bruised Ribs", Severity: "mild"})
	})
	assert.True(t, ui.inRecovery, "inRecovery should be true after recovery event")

	require.NotPanics(t, func() {
		ui.Emit(uicontract.MilestoneEvent{Type: "scenario_complete"})
	})
	assert.False(t, ui.inRecovery, "inRecovery should be false after milestone")
}

// --- PromptForInvoke tests ---

func TestPromptForInvoke_NoUsableAspects_Skips(t *testing.T) {
	ui := newUIWithInput("")

	// All aspects already used and no FP → nothing to invoke
	available := []uicontract.InvokableAspect{
		{Name: "On Fire!", FreeInvokes: 0, AlreadyUsed: true},
	}
	resp := ui.PromptForInvoke(available, 0, "Fair (+2)", 1)
	assert.Equal(t, uicontract.InvokeSkip, resp.AspectIndex)
}

func TestPromptForInvoke_NoFPAndNoFreeInvokes_Skips(t *testing.T) {
	ui := newUIWithInput("")

	available := []uicontract.InvokableAspect{
		{Name: "Brave", FreeInvokes: 0, AlreadyUsed: false},
	}
	resp := ui.PromptForInvoke(available, 0, "Fair (+2)", 1)
	assert.Equal(t, uicontract.InvokeSkip, resp.AspectIndex)
}

func TestPromptForInvoke_PlayerSkips_EnterKey(t *testing.T) {
	// Player hits Enter (empty input) → skip
	ui := newUIWithInput("\n")

	available := []uicontract.InvokableAspect{
		{Name: "On Fire!", FreeInvokes: 1, AlreadyUsed: false},
	}
	resp := ui.PromptForInvoke(available, 3, "Fair (+2)", 1)
	assert.Equal(t, uicontract.InvokeSkip, resp.AspectIndex)
}

func TestPromptForInvoke_ValidChoice_PlusTwoBonus(t *testing.T) {
	// Choose aspect 1, then "b" for +2
	ui := newUIWithInput("1\nb\n")

	available := []uicontract.InvokableAspect{
		{Name: "On Fire!", FreeInvokes: 1, AlreadyUsed: false},
	}
	resp := ui.PromptForInvoke(available, 3, "Fair (+2)", 1)
	assert.Equal(t, 0, resp.AspectIndex)
	assert.False(t, resp.IsReroll)
}

func TestPromptForInvoke_ValidChoice_Reroll(t *testing.T) {
	// Choose aspect 1, then "r" for reroll
	ui := newUIWithInput("1\nr\n")

	available := []uicontract.InvokableAspect{
		{Name: "On Fire!", FreeInvokes: 1, AlreadyUsed: false},
	}
	resp := ui.PromptForInvoke(available, 3, "Fair (+2)", 1)
	assert.Equal(t, 0, resp.AspectIndex)
	assert.True(t, resp.IsReroll)
}

func TestPromptForInvoke_ValidChoice_RerollLongForm(t *testing.T) {
	// "reroll" as the full word should also set IsReroll
	ui := newUIWithInput("1\nreroll\n")

	available := []uicontract.InvokableAspect{
		{Name: "On Fire!", FreeInvokes: 1, AlreadyUsed: false},
	}
	resp := ui.PromptForInvoke(available, 3, "Fair (+2)", 1)
	assert.True(t, resp.IsReroll)
}

func TestPromptForInvoke_InvalidChoice_Skips(t *testing.T) {
	// "99" is out of range
	ui := newUIWithInput("99\n")

	available := []uicontract.InvokableAspect{
		{Name: "On Fire!", FreeInvokes: 1, AlreadyUsed: false},
	}
	resp := ui.PromptForInvoke(available, 3, "Fair (+2)", 1)
	assert.Equal(t, uicontract.InvokeSkip, resp.AspectIndex)
}

func TestPromptForInvoke_OnlyFPUsable_WhenHasFP(t *testing.T) {
	// Aspect has no free invokes but player has FP
	ui := newUIWithInput("1\nb\n")

	available := []uicontract.InvokableAspect{
		{Name: "Brave", FreeInvokes: 0, AlreadyUsed: false},
	}
	resp := ui.PromptForInvoke(available, 2, "Fair (+2)", 1)
	assert.Equal(t, 0, resp.AspectIndex)
}

func TestPromptForInvoke_ReadError_Skips(t *testing.T) {
	// Empty reader causes EOF on first read
	ui := newUIWithInput("")

	available := []uicontract.InvokableAspect{
		{Name: "On Fire!", FreeInvokes: 1, AlreadyUsed: false},
	}
	resp := ui.PromptForInvoke(available, 3, "Fair (+2)", 1)
	assert.Equal(t, uicontract.InvokeSkip, resp.AspectIndex)
}

func TestPromptForInvoke_ShiftsNeededZero(t *testing.T) {
	// shiftsNeeded=0 should not panic (different display branch)
	ui := newUIWithInput("\n")

	available := []uicontract.InvokableAspect{
		{Name: "Brave", FreeInvokes: 1, AlreadyUsed: false},
	}
	require.NotPanics(t, func() {
		ui.PromptForInvoke(available, 1, "Good (+3)", 0)
	})
}

// --- ReadInput tests ---

func TestReadInput_ExitCommand(t *testing.T) {
	ui := newUIWithInput("exit\n")

	input, isExit, err := ui.ReadInput()
	require.NoError(t, err)
	assert.True(t, isExit)
	assert.Equal(t, "exit", input)
}

func TestReadInput_MetaCommand_ReturnsEmpty(t *testing.T) {
	ui := newUIWithInput("help\n")
	ui.SetSceneInfo(&mockSceneInfo{
		scene:  scene.NewScene("s1", "Scene", "Desc"),
		player: character.NewCharacter("p1", "Hero"),
	})

	input, isExit, err := ui.ReadInput()
	require.NoError(t, err)
	assert.False(t, isExit)
	assert.Equal(t, "", input, "meta-commands return empty string to engine")
}

func TestReadInput_NormalInput_Returned(t *testing.T) {
	ui := newUIWithInput("attack the goblin\n")
	ui.SetSceneInfo(&mockSceneInfo{
		scene:  scene.NewScene("s1", "Scene", "Desc"),
		player: character.NewCharacter("p1", "Hero"),
	})

	input, isExit, err := ui.ReadInput()
	require.NoError(t, err)
	assert.False(t, isExit)
	assert.Equal(t, "attack the goblin", input)
}

func TestReadInput_EmptyLine_ReturnsEmpty(t *testing.T) {
	ui := newUIWithInput("\n")

	input, isExit, err := ui.ReadInput()
	require.NoError(t, err)
	assert.False(t, isExit)
	assert.Equal(t, "", input)
}

func TestReadInput_HintShownOnce(t *testing.T) {
	ui := newUIWithInput("hello\nhello\n")
	assert.False(t, ui.shownHint)

	ui.ReadInput() //nolint:errcheck
	assert.True(t, ui.shownHint)

	ui.ReadInput() //nolint:errcheck
	assert.True(t, ui.shownHint)
}

func TestReadInput_NoSceneInfo_DoesNotHandleMetaCommands(t *testing.T) {
	// Without sceneInfo, meta-commands fall through to engine
	ui := newUIWithInput("help\n")
	// sceneInfo is nil, so handleSpecialCommands is skipped

	input, isExit, err := ui.ReadInput()
	require.NoError(t, err)
	assert.False(t, isExit)
	assert.Equal(t, "help", input)
}

// --- DisplayCharacter tests ---

func TestDisplayCharacter_NoSceneInfo(t *testing.T) {
	ui := NewTerminalUI()
	require.NotPanics(t, func() { ui.DisplayCharacter() })
}

func TestDisplayCharacter_NoPlayer(t *testing.T) {
	ui := NewTerminalUI()
	ui.SetSceneInfo(&mockSceneInfo{player: nil})
	require.NotPanics(t, func() { ui.DisplayCharacter() })
}

func TestDisplayCharacter_WithAspects(t *testing.T) {
	ui := NewTerminalUI()
	player := character.NewCharacter("p1", "Hero")
	player.Aspects.HighConcept = "Brave Warrior"
	player.Aspects.Trouble = "Quick to Anger"
	player.Aspects.AddAspect("Well Connected")
	ui.SetSceneInfo(&mockSceneInfo{player: player})
	require.NotPanics(t, func() { ui.DisplayCharacter() })
}
