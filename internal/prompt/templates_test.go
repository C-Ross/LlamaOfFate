package prompt

import (
	"bytes"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputClassificationTemplate(t *testing.T) {
	// Create test data
	testScene := scene.NewScene("test-scene", "Test Scene", "A test scene description")
	data := InputClassificationData{
		Scene:       testScene,
		PlayerInput: "What do I see?",
	}

	// Execute the template
	var buf bytes.Buffer
	err := InputClassificationPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail")

	result := buf.String()

	// Verify the template was populated correctly
	assert.Contains(t, result, "Test Scene", "Scene name should be included")
	assert.Contains(t, result, "A test scene description", "Scene description should be included")
	assert.Contains(t, result, "What do I see?", "Player input should be included")
	assert.Contains(t, result, "Fate Core principle", "Template should contain Fate Core guidance")
}

func TestSceneResponseTemplate(t *testing.T) {
	// Create test data
	testScene := scene.NewScene("test-scene", "Test Scene", "A test scene description")
	data := SceneResponseData{
		Scene:               testScene,
		CharacterContext:    "Test Character Context",
		AspectsContext:      "Test Aspects Context",
		ConversationContext: "Test Conversation Context",
		PlayerInput:         "Look around",
		InteractionType:     "clarification",
	}

	// Execute the template
	var buf bytes.Buffer
	err := SceneResponsePrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail")

	result := buf.String()

	// Verify the template was populated correctly
	assert.Contains(t, result, "Test Scene", "Scene name should be included")
	assert.Contains(t, result, "Test Character Context", "Character context should be included")
	assert.Contains(t, result, "Look around", "Player input should be included")
	assert.Contains(t, result, "clarification", "Interaction type should be included")
	assert.Contains(t, result, "FATE CORE GM PRINCIPLES", "Template should contain GM guidance")
}

func TestConsequenceAspectTemplate(t *testing.T) {
	// Create test data
	data := ConsequenceAspectData{
		CharacterName: "Hero",
		AttackerName:  "Dark Knight",
		Severity:      "moderate",
		ConflictType:  "physical",
		AttackContext: AttackContext{
			Skill:       "Fight",
			Description: "The Dark Knight's sword crashes down on your shield",
			Shifts:      4,
		},
	}

	// Execute the template
	var buf bytes.Buffer
	err := ConsequenceAspectPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail")

	result := buf.String()

	// Verify the template was populated correctly
	assert.Contains(t, result, "Hero", "Character name should be included")
	assert.Contains(t, result, "Dark Knight", "Attacker name should be included")
	assert.Contains(t, result, "moderate", "Severity should be included")
	assert.Contains(t, result, "physical", "Conflict type should be included")
	assert.Contains(t, result, "Fight", "Attack skill should be included")
	assert.Contains(t, result, "The Dark Knight's sword crashes down on your shield", "Attack description should be included")
	assert.Contains(t, result, "4", "Attack shifts should be included")
}

func TestConsequenceAspectTemplateWithoutAttackContext(t *testing.T) {
	// Create test data without attack context (optional fields)
	data := ConsequenceAspectData{
		CharacterName: "Hero",
		AttackerName:  "Dark Knight",
		Severity:      "mild",
		ConflictType:  "mental",
	}

	// Execute the template
	var buf bytes.Buffer
	err := ConsequenceAspectPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail even without attack context")

	result := buf.String()

	// Verify the template was populated with basic fields
	assert.Contains(t, result, "Hero", "Character name should be included")
	assert.Contains(t, result, "Dark Knight", "Attacker name should be included")
	assert.Contains(t, result, "mild", "Severity should be included")
	assert.Contains(t, result, "mental", "Conflict type should be included")
}

func TestTakenOutTemplate(t *testing.T) {
	// Create test data
	data := TakenOutData{
		CharacterName:       "Hero",
		AttackerName:        "Dark Knight",
		AttackerHighConcept: "Corrupted Champion of Darkness",
		ConflictType:        "physical",
		SceneDescription:    "A dark throne room with shadowy pillars",
		AttackContext: AttackContext{
			Skill:       "Fight",
			Description: "The Dark Knight's final blow strikes true",
			Shifts:      6,
		},
	}

	// Execute the template
	var buf bytes.Buffer
	err := TakenOutPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail")

	result := buf.String()

	// Verify the template was populated correctly
	assert.Contains(t, result, "Hero", "Character name should be included")
	assert.Contains(t, result, "Dark Knight", "Attacker name should be included")
	assert.Contains(t, result, "Corrupted Champion of Darkness", "Attacker high concept should be included")
	assert.Contains(t, result, "physical", "Conflict type should be included")
	assert.Contains(t, result, "A dark throne room with shadowy pillars", "Scene description should be included")
	assert.Contains(t, result, "Fight", "Attack skill should be included")
	assert.Contains(t, result, "The Dark Knight's final blow strikes true", "Attack description should be included")
	assert.Contains(t, result, "6", "Attack shifts should be included")
}

func TestTakenOutTemplateWithoutAttackContext(t *testing.T) {
	// Create test data without attack context (optional fields)
	data := TakenOutData{
		CharacterName:       "Hero",
		AttackerName:        "Dark Knight",
		AttackerHighConcept: "Corrupted Champion of Darkness",
		ConflictType:        "mental",
		SceneDescription:    "A dark throne room",
	}

	// Execute the template
	var buf bytes.Buffer
	err := TakenOutPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail even without attack context")

	result := buf.String()

	// Verify the template was populated with basic fields
	assert.Contains(t, result, "Hero", "Character name should be included")
	assert.Contains(t, result, "Dark Knight", "Attacker name should be included")
	assert.Contains(t, result, "Corrupted Champion of Darkness", "Attacker high concept should be included")
	assert.Contains(t, result, "mental", "Conflict type should be included")
	assert.Contains(t, result, "A dark throne room", "Scene description should be included")
}

func TestRecoveryNarrativeTemplate(t *testing.T) {
	data := RecoveryNarrativeData{
		CharacterName: "Simon Falcon",
		SceneSetting:  "The crew rests after escaping the orbital station",
		Consequences: []RecoveryAttempt{
			{
				Aspect:     "Bruised Ribs",
				Severity:   "mild",
				Skill:      "Physique",
				RollResult: 3,
				Difficulty: "2",
				Outcome:    "success",
			},
			{
				Aspect:     "Shattered Confidence",
				Severity:   "moderate",
				Skill:      "Will",
				RollResult: 1,
				Difficulty: "4",
				Outcome:    "failure",
			},
		},
	}

	rendered, err := RenderRecoveryNarrative(data)
	require.NoError(t, err)

	assert.Contains(t, rendered, "Simon Falcon")
	assert.Contains(t, rendered, "Bruised Ribs")
	assert.Contains(t, rendered, "Shattered Confidence")
	assert.Contains(t, rendered, "success")
	assert.Contains(t, rendered, "failure")
}

func TestFateNarrationTemplate(t *testing.T) {
	data := FateNarrationData{
		SceneName:        "Warehouse Showdown",
		SceneDescription: "A tense fight in a dark warehouse",
		ConflictType:     "physical",
		TakenOutNPCs: []FateNarrationNPC{
			{ID: "npc-thug", Name: "Thug", HighConcept: "Hired Muscle"},
			{ID: "npc-boss", Name: "Boss", HighConcept: "Criminal Mastermind"},
		},
		PlayerNarration: "I knock them both out and tie them up for the authorities.",
	}

	rendered, err := RenderFateNarration(data)
	require.NoError(t, err)

	assert.Contains(t, rendered, "Warehouse Showdown")
	assert.Contains(t, rendered, "Thug")
	assert.Contains(t, rendered, "npc-thug")
	assert.Contains(t, rendered, "npc-boss")
	assert.Contains(t, rendered, "Criminal Mastermind")
	assert.Contains(t, rendered, "knock them both out")
	assert.Contains(t, rendered, "physical")
}

func TestChallengeBuildTemplate(t *testing.T) {
	data := ChallengeBuildData{
		Description:      "Break into the baron's vault",
		SceneName:        "The Baron's Keep",
		SceneDescription: "A fortified keep with guards and traps",
		PlayerSkills:     []string{"Athletics", "Stealth", "Burglary"},
		SituationAspects: []scene.SituationAspect{
			{ID: "asp-1", Aspect: "Patrolling Guards"},
			{ID: "asp-2", Aspect: "Moonless Night"},
		},
		DifficultyGuidance: DifficultyGuidance{
			DifficultyMin:     1,
			DifficultyMax:     5,
			DifficultyDefault: 2,
			DifficultyGuide:   "1=easy, 2=moderate, 5=hard",
		},
	}

	rendered, err := RenderChallengeBuild(data)
	require.NoError(t, err)

	assert.Contains(t, rendered, "Break into the baron's vault")
	assert.Contains(t, rendered, "The Baron's Keep")
	assert.Contains(t, rendered, "fortified keep")
	assert.Contains(t, rendered, "Athletics")
	assert.Contains(t, rendered, "Stealth")
	assert.Contains(t, rendered, "Burglary")
	assert.Contains(t, rendered, "1=easy, 2=moderate, 5=hard")
	assert.Contains(t, rendered, "DIFFICULTY RANGE")
	assert.Contains(t, rendered, "Patrolling Guards")
	assert.Contains(t, rendered, "Moonless Night")
	assert.Contains(t, rendered, "SITUATION ASPECTS")
}

func TestRenderInputClassification(t *testing.T) {
	testScene := scene.NewScene("test-scene", "Dark Forest", "A dark and foreboding forest.")
	data := InputClassificationData{
		Scene:       testScene,
		PlayerInput: "I search for hidden paths",
	}

	rendered, err := RenderInputClassification(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Dark Forest")
	assert.Contains(t, rendered, "I search for hidden paths")
}

func TestRenderSceneResponse(t *testing.T) {
	testScene := scene.NewScene("tavern", "The Old Tavern", "A noisy tavern.")
	data := SceneResponseData{
		Scene:               testScene,
		CharacterContext:    "Experienced adventurer",
		AspectsContext:      "On the Hunt",
		ConversationContext: "Previous conversation",
		PlayerInput:         "I ask the barkeep about the stranger",
		InteractionType:     "dialog",
	}

	rendered, err := RenderSceneResponse(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "The Old Tavern")
	assert.Contains(t, rendered, "I ask the barkeep about the stranger")
	assert.Contains(t, rendered, "dialog")
}

func TestRenderConflictResponse(t *testing.T) {
	testScene := scene.NewScene("arena", "The Arena", "A sandy combat arena.")
	testScene.StartConflict(scene.PhysicalConflict, []scene.ConflictParticipant{})
	data := ConflictResponseData{
		Scene:                testScene,
		CharacterContext:     "Brave warrior",
		AspectsContext:       "Ready to Fight",
		ConversationContext:  "",
		PlayerInput:          "I attack with my sword",
		CurrentCharacterName: "Hero",
		ParticipantMap:       map[string]*scene.ConflictParticipant{},
		CharacterMap:         map[string]*character.Character{},
	}

	rendered, err := RenderConflictResponse(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "The Arena")
	assert.Contains(t, rendered, "I attack with my sword")
}

func TestRenderActionNarrative(t *testing.T) {
	testScene := scene.NewScene("dungeon", "The Dungeon", "A damp stone dungeon.")
	char := character.NewCharacter("hero-1", "Aria")
	act := action.NewAction("action-1", "hero-1", action.Overcome, "Athletics", "I leap over the gap")
	act.Outcome = &dice.Outcome{Type: dice.Success, Shifts: 2}
	data := ActionNarrativeData{
		Scene:            testScene,
		CharacterContext: "Nimble rogue",
		AspectsContext:   "Fleeing the Guards",
		Action:           act,
	}

	rendered, err := RenderActionNarrative(data)
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
	_ = char // used for setup illustration
}

func TestRenderNPCAttack(t *testing.T) {
	data := NPCAttackData{
		ConflictType:       "physical",
		Round:              2,
		SceneName:          "Rooftop Chase",
		NPCName:            "Street Thug",
		NPCHighConcept:     "Ruthless Street Criminal",
		NPCAspects:         []string{"Quick Reflexes"},
		Skill:              "Fight",
		TargetName:         "Hero",
		TargetHighConcept:  "Daring Detective",
		SituationAspects:   []scene.SituationAspect{{ID: "asp-1", Aspect: "Slippery Rooftop"}},
		OutcomeDescription: "deals 3 shifts of damage",
	}

	rendered, err := RenderNPCAttack(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Street Thug")
	assert.Contains(t, rendered, "Hero")
	assert.Contains(t, rendered, "Slippery Rooftop")
}

func TestRenderNPCActionDecision(t *testing.T) {
	data := NPCActionDecisionData{
		ConflictType:     "physical",
		Round:            1,
		SceneName:        "Warehouse Fight",
		SceneDescription: "A dark warehouse with stacked crates.",
		NPCName:          "Guard",
		NPCHighConcept:   "Loyal Warehouse Guard",
		NPCSkills:        map[string]int{"Fight": 2, "Athletics": 1},
		NPCPhysicalStress: []bool{false, false},
		NPCMentalStress:  []bool{false},
		Targets: []NPCTargetInfo{
			{ID: "hero-1", Name: "Hero", HighConcept: "Daring Detective"},
		},
	}

	rendered, err := RenderNPCActionDecision(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Guard")
	assert.Contains(t, rendered, "Hero")
	assert.Contains(t, rendered, "Warehouse Fight")
}

func TestRenderActionParse(t *testing.T) {
	char := character.NewCharacter("hero-1", "Kai")
	char.Skills = map[string]dice.Ladder{"Fight": dice.Good, "Athletics": dice.Fair}
	data := ActionParseTemplateData{
		Character: char,
		RawInput:  "I punch the guard",
		Context:   "Physical conflict in a dungeon",
		DifficultyGuidance: DifficultyGuidance{
			DifficultyMin:     1,
			DifficultyMax:     5,
			DifficultyDefault: 2,
			DifficultyGuide:   "1=easy, 2=moderate, 5=hard",
		},
	}

	rendered, err := RenderActionParse(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Kai")
	assert.Contains(t, rendered, "I punch the guard")
}

func TestRenderActionParseSystem(t *testing.T) {
	rendered, err := RenderActionParseSystem()
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestRenderAspectGeneration(t *testing.T) {
	char := character.NewCharacter("hero-1", "Lena")
	char.Aspects.HighConcept = "Fearless Explorer"
	act := action.NewAction("action-1", "hero-1", action.CreateAdvantage, "Investigate", "I search the room")
	outcome := &dice.Outcome{Shifts: 2}
	data := AspectGenerationRequest{
		Character:   char,
		Action:      act,
		Outcome:     outcome,
		Context:     "Searching the ancient ruins",
		TargetType:  "scene",
	}

	rendered, err := RenderAspectGeneration(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Lena")
	assert.Contains(t, rendered, "Investigate")
}

func TestRenderAspectGenerationSystem(t *testing.T) {
	rendered, err := RenderAspectGenerationSystem()
	require.NoError(t, err)
	assert.NotEmpty(t, rendered)
}

func TestRenderSceneGeneration(t *testing.T) {
	scenario := &scene.Scenario{
		Title:   "The Missing Artifact",
		Problem: "A powerful relic has vanished from the museum.",
		Setting: "Victorian London",
		Genre:   "Steampunk",
	}
	data := SceneGenerationData{
		TransitionHint:    "a foggy London street",
		Scenario:          scenario,
		PlayerName:        "Inspector Wells",
		PlayerHighConcept: "Relentless Investigator",
		PlayerTrouble:     "Haunted by the Past",
	}

	rendered, err := RenderSceneGeneration(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Inspector Wells")
	assert.Contains(t, rendered, "Victorian London")
}

func TestRenderSceneSummary(t *testing.T) {
	data := SceneSummaryData{
		SceneName:        "The Museum Heist",
		SceneDescription: "A grand museum at night.",
		SituationAspects: []string{"Broken Alarm", "Moonlit Halls"},
		HowEnded:         "transition",
		TransitionHint:   "the rooftops",
	}

	rendered, err := RenderSceneSummary(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "The Museum Heist")
	assert.Contains(t, rendered, "transition")
}

func TestRenderScenarioGeneration(t *testing.T) {
	data := ScenarioGenerationData{
		PlayerName:        "Jade",
		PlayerHighConcept: "Cyberpunk Hacker",
		PlayerTrouble:     "Hunted by the Megacorp",
		PlayerAspects:     []string{"Street Smart", "Back Alley Connections"},
		Genre:             "Cyberpunk",
		Theme:             "Corporate conspiracy",
	}

	rendered, err := RenderScenarioGeneration(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Jade")
	assert.Contains(t, rendered, "Cyberpunk Hacker")
	assert.Contains(t, rendered, "Cyberpunk")
}

func TestRenderScenarioResolution(t *testing.T) {
	scenario := &scene.Scenario{
		Title:          "The Dark Conspiracy",
		Problem:        "A secret cult threatens the city.",
		StoryQuestions: []string{"Will the hero expose the cult?"},
	}
	data := ScenarioResolutionData{
		Scenario:   scenario,
		PlayerName: "Marcus",
		PlayerAspects: []string{"Determined Journalist"},
	}

	rendered, err := RenderScenarioResolution(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "The Dark Conspiracy")
	assert.Contains(t, rendered, "Marcus")
}

func TestRenderConsequenceAspect(t *testing.T) {
	data := ConsequenceAspectData{
		CharacterName: "Reya",
		AttackerName:  "Bandit Chief",
		Severity:      "moderate",
		ConflictType:  "physical",
		AttackContext: AttackContext{
			Skill:       "Fight",
			Description: "A crushing blow to the ribs",
			Shifts:      4,
		},
	}

	rendered, err := RenderConsequenceAspect(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Reya")
	assert.Contains(t, rendered, "Bandit Chief")
	assert.Contains(t, rendered, "moderate")
}

func TestRenderTakenOut(t *testing.T) {
	data := TakenOutData{
		CharacterName:       "Finn",
		AttackerName:        "Shadow Assassin",
		AttackerHighConcept: "Death-Dealer for Hire",
		ConflictType:        "physical",
		SceneDescription:    "A moonlit courtyard.",
		AttackContext: AttackContext{
			Skill:       "Stealth",
			Description: "A blade from the shadows",
			Shifts:      6,
		},
	}

	rendered, err := RenderTakenOut(data)
	require.NoError(t, err)
	assert.Contains(t, rendered, "Finn")
	assert.Contains(t, rendered, "Shadow Assassin")
	assert.Contains(t, rendered, "A moonlit courtyard.")
}

// TestTemplatesInit verifies the templates package init() loaded all templates.
func TestTemplatesInit(t *testing.T) {
	// All template globals should be non-nil after init()
	require.NotNil(t, AspectGenerationPrompt)
	require.NotNil(t, AspectGenerationSystemPrompt)
	require.NotNil(t, ActionParseSystemPrompt)
	require.NotNil(t, ActionParsePrompt)
	require.NotNil(t, InputClassificationPrompt)
	require.NotNil(t, SceneResponsePrompt)
	require.NotNil(t, ActionNarrativePrompt)
	require.NotNil(t, ConflictResponsePrompt)
	require.NotNil(t, NPCAttackPrompt)
	require.NotNil(t, NPCActionDecisionPrompt)
	require.NotNil(t, ConsequenceAspectPrompt)
	require.NotNil(t, TakenOutPrompt)
	require.NotNil(t, SceneGenerationPrompt)
	require.NotNil(t, SceneSummaryPrompt)
	require.NotNil(t, ScenarioGenerationPrompt)
	require.NotNil(t, ScenarioResolutionPrompt)
	require.NotNil(t, RecoveryNarrativePrompt)
	require.NotNil(t, FateNarrationPrompt)
	require.NotNil(t, ChallengeBuildPrompt)
}

// Prevent "imported and not used" error: bytes is used in existing tests above.
var _ = bytes.NewBuffer
