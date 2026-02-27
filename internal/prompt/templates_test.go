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
	testScene := scene.NewScene("test-scene", "Harbor District", "A busy port")
	data := InputClassificationData{
		Scene:       testScene,
		PlayerInput: "I want to sneak past the guards",
	}

	result, err := RenderInputClassification(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Harbor District")
	assert.Contains(t, result, "I want to sneak past the guards")
}

func TestRenderSceneResponse(t *testing.T) {
	testScene := scene.NewScene("test-scene", "Moonlit Alley", "A narrow alley under the moon")
	data := SceneResponseData{
		Scene:               testScene,
		CharacterContext:    "A skilled rogue",
		AspectsContext:      "Shadows everywhere",
		ConversationContext: "Previous dialogue",
		PlayerInput:         "I look for a hiding spot",
		InteractionType:     "action",
	}

	result, err := RenderSceneResponse(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Moonlit Alley")
	assert.Contains(t, result, "A skilled rogue")
	assert.Contains(t, result, "I look for a hiding spot")
}

func TestRenderConflictResponse(t *testing.T) {
	testScene := scene.NewScene("conflict-scene", "Arena Floor", "An ancient arena")
	testScene.StartConflict(scene.PhysicalConflict, []scene.ConflictParticipant{})
	data := ConflictResponseData{
		Scene:                testScene,
		CharacterContext:     "Brave warrior",
		AspectsContext:       "Battle-hardened",
		ConversationContext:  "Round started",
		PlayerInput:          "I attack the goblin",
		CurrentCharacterName: "Hero",
		ParticipantMap:       map[string]*scene.ConflictParticipant{},
		CharacterMap:         map[string]*character.Character{},
	}

	result, err := RenderConflictResponse(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Arena Floor")
	assert.Contains(t, result, "I attack the goblin")
}

func TestRenderActionNarrative(t *testing.T) {
	testScene := scene.NewScene("scene-1", "Training Ground", "A place for training")
	act := action.NewAction("act-1", "hero-1", action.Overcome, "Athletics", "Sprint across the gap")
	act.Outcome = &dice.Outcome{Type: dice.Success, Shifts: 2}
	data := ActionNarrativeData{
		Scene:            testScene,
		CharacterContext: "A nimble hero",
		AspectsContext:   "No aspects",
		Action:           act,
	}

	result, err := RenderActionNarrative(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Sprint across the gap")
	assert.NotEmpty(t, result)
}

func TestRenderNPCActionDecision(t *testing.T) {
	data := NPCActionDecisionData{
		ConflictType:      "physical",
		Round:             2,
		SceneName:         "Tavern Brawl",
		SceneDescription:  "Tables and chairs flying",
		NPCName:           "Bandit",
		NPCHighConcept:    "Desperate Outlaw",
		NPCTrouble:        "Greedy to a Fault",
		NPCAspects:        []string{"Quick with a Blade"},
		NPCSkills:         map[string]int{"Fight": 3, "Athletics": 2},
		NPCPhysicalStress: []bool{false, false, false},
		NPCMentalStress:   []bool{false, false},
		Targets: []NPCTargetInfo{
			{ID: "player-1", Name: "Hero", HighConcept: "Wandering Knight"},
		},
		SituationAspects: []scene.SituationAspect{
			{ID: "asp-1", Aspect: "Broken Furniture"},
		},
	}

	result, err := RenderNPCActionDecision(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Bandit")
	assert.Contains(t, result, "Tavern Brawl")
	assert.Contains(t, result, "Hero")
}

func TestRenderNPCAttack(t *testing.T) {
	data := NPCAttackData{
		ConflictType:       "physical",
		Round:              1,
		SceneName:          "Dark Alley",
		NPCName:            "Shadow Assassin",
		NPCHighConcept:     "Silent Killer",
		NPCAspects:         []string{"Moves Like a Ghost"},
		Skill:              "Fight",
		TargetName:         "Detective",
		TargetHighConcept:  "World-Weary Investigator",
		SituationAspects:   []scene.SituationAspect{{ID: "asp-1", Aspect: "Heavy Fog"}},
		OutcomeDescription: "The assassin strikes from the shadows, landing a blow.",
	}

	result, err := RenderNPCAttack(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Shadow Assassin")
	assert.Contains(t, result, "Detective")
	assert.Contains(t, result, "Dark Alley")
}

func TestRenderActionParse(t *testing.T) {
	char := character.NewCharacter("player-1", "Aria Swift")
	char.Skills["Athletics"] = dice.Great
	char.Skills["Fight"] = dice.Good
	data := ActionParseTemplateData{
		Character: char,
		RawInput:  "I want to leap over the gap",
		Context:   "Rooftop chase",
		DifficultyGuidance: DifficultyGuidance{
			DifficultyMin:     1,
			DifficultyMax:     5,
			DifficultyDefault: 2,
			DifficultyGuide:   "1=easy, 2=moderate, 5=hard",
		},
	}

	result, err := RenderActionParse(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Aria Swift")
	assert.Contains(t, result, "I want to leap over the gap")
}

func TestRenderActionParseSystem(t *testing.T) {
	result, err := RenderActionParseSystem()
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRenderAspectGeneration(t *testing.T) {
	char := character.NewCharacter("player-1", "Rex Bold")
	act := action.NewAction("act-1", "player-1", action.CreateAdvantage, "Notice", "Search the crime scene")
	act.Outcome = &dice.Outcome{Type: dice.Success, Shifts: 1}
	outcome := &dice.Outcome{Type: dice.Success, Shifts: 1}
	data := AspectGenerationRequest{
		Character:       char,
		Action:          act,
		Outcome:         outcome,
		Context:         "A dusty office with overturned furniture",
		TargetType:      "scene",
		ExistingAspects: []string{"Dim Lighting"},
	}

	result, err := RenderAspectGeneration(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Rex Bold")
	assert.NotEmpty(t, result)
}

func TestRenderAspectGenerationSystem(t *testing.T) {
	result, err := RenderAspectGenerationSystem()
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRenderSceneGeneration(t *testing.T) {
	scenario := &scene.Scenario{
		Title:   "The Lost City",
		Problem: "An ancient evil has awakened beneath the city",
		Setting: "Fantasy city built on ruins",
	}
	data := SceneGenerationData{
		TransitionHint:    "The underground ruins",
		Scenario:          scenario,
		PlayerName:        "Zara",
		PlayerHighConcept: "Fearless Archaeologist",
		PlayerTrouble:     "Reckless Curiosity",
		PlayerAspects:     []string{"Ancient Languages Expert"},
		PreviousSummaries: []SceneSummary{},
		Complications:     []string{},
		KnownNPCs:         []NPCSummary{},
	}

	result, err := RenderSceneGeneration(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Zara")
	assert.Contains(t, result, "The Lost City")
}

func TestRenderSceneSummary(t *testing.T) {
	data := SceneSummaryData{
		SceneName:        "The Market Square",
		SceneDescription: "A busy marketplace",
		SituationAspects: []string{"Crowded Streets", "Merchant Stalls"},
		ConversationHistory: []ConversationEntry{
			{PlayerInput: "I ask the merchant about the theft.", GMResponse: "The merchant nervously denies knowing anything."},
		},
		NPCsInScene: []NPCSummary{
			{Name: "Merchant", Attitude: "nervous"},
		},
		HowEnded:       "transition",
		TransitionHint: "The docks",
	}

	result, err := RenderSceneSummary(data)
	require.NoError(t, err)
	assert.Contains(t, result, "The Market Square")
	assert.Contains(t, result, "Merchant")
}

func TestRenderScenarioGeneration(t *testing.T) {
	data := ScenarioGenerationData{
		PlayerName:        "Marcus",
		PlayerHighConcept: "Former Detective",
		PlayerTrouble:     "The Case That Broke Me",
		PlayerAspects:     []string{"Friends in Low Places"},
		Genre:             "noir",
		Theme:             "corruption",
	}

	result, err := RenderScenarioGeneration(data)
	require.NoError(t, err)
	assert.Contains(t, result, "Marcus")
	assert.Contains(t, result, "Former Detective")
}

func TestRenderScenarioResolution(t *testing.T) {
	scenario := &scene.Scenario{
		Title:          "The Conspiracy",
		Problem:        "A secret society controls the city government",
		StoryQuestions: []string{"Can the player expose them?", "Will the city be saved?"},
	}
	data := ScenarioResolutionData{
		Scenario:      scenario,
		PlayerName:    "Investigator",
		PlayerAspects: []string{"Dogged Reporter"},
		SceneSummaries: []SceneSummary{
			{NarrativeProse: "The player found evidence of the conspiracy."},
		},
	}

	result, err := RenderScenarioResolution(data)
	require.NoError(t, err)
	assert.Contains(t, result, "The Conspiracy")
	assert.Contains(t, result, "A secret society controls the city government")
}
