package engine

import (
	"context"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMClientForScenario provides predictable responses for scenario manager tests
type MockLLMClientForScenario struct {
	responses []string
	callIndex int
}

func (m *MockLLMClientForScenario) ChatCompletion(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	response := `{"scene_name": "Test Scene", "description": "A test scene.", "purpose": "Can the hero survive?", "situation_aspects": ["Aspect 1"], "npcs": []}`
	if m.callIndex < len(m.responses) {
		response = m.responses[m.callIndex]
		m.callIndex++
	}
	return &llm.CompletionResponse{
		ID:      "test-response",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "test-model",
		Choices: []llm.CompletionResponseChoice{
			{
				Index: 0,
				Message: llm.Message{
					Role:    "assistant",
					Content: response,
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (m *MockLLMClientForScenario) ChatCompletionStream(ctx context.Context, req llm.CompletionRequest, handler llm.StreamHandler) error {
	return nil
}

func (m *MockLLMClientForScenario) GetModelInfo() llm.ModelInfo {
	return llm.ModelInfo{Name: "test-model"}
}

func TestNewScenarioManager(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	assert.NotNil(t, sm)
	assert.Equal(t, engine, sm.engine)
	assert.Equal(t, player, sm.player)
}

func TestScenarioManager_Run_RequiresEngine(t *testing.T) {
	sm := &ScenarioManager{}
	_, err := sm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "engine is required")
}

func TestScenarioManager_Run_RequiresLLM(t *testing.T) {
	engine, err := New() // No LLM
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	_, err = sm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM client is required")
}

func TestScenarioManager_Run_RequiresUI(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	_, err = sm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UI is required")
}

func TestScenarioManager_Run_RequiresPlayer(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	sm := &ScenarioManager{
		engine: engine,
		ui:     &MockUI{},
	}

	_, err = sm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player character is required")
}

func TestScenarioManager_SetScenario(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	scenario := &Scenario{
		Title:   "Test Scenario",
		Problem: "A test problem",
		Genre:   "Western",
		Setting: "The Old West",
	}
	sm.SetScenario(scenario)

	assert.Equal(t, "Western", sm.scenario.Genre)
	assert.Equal(t, "The Old West", sm.scenario.Setting)
	assert.Equal(t, "A test problem", sm.scenario.Problem)
}

func TestScenarioManager_SetInitialScene(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	npc := character.NewCharacter("npc1", "Test NPC")

	sm.SetInitialScene(testScene, []*character.Character{npc})

	assert.Equal(t, testScene, sm.initialScene)
	assert.Len(t, sm.initialNPCs, 1)
	assert.Equal(t, npc, sm.initialNPCs[0])
}

func TestParseGeneratedScene_Valid(t *testing.T) {
	jsonResponse := `{
		"scene_name": "The Dusty Trail",
		"description": "A winding path through the desert, heat waves shimmer in the distance.",
		"purpose": "Can the traveler survive the scorching desert crossing?",
		"opening_hook": "A vulture circles overhead as the path narrows between two boulders.",
		"situation_aspects": ["Blazing Sun", "Rocky Terrain"],
		"npcs": [
			{"name": "Old Prospector", "high_concept": "Grizzled Desert Wanderer", "disposition": "friendly"}
		]
	}`

	generated, err := ParseGeneratedScene(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "The Dusty Trail", generated.SceneName)
	assert.Contains(t, generated.Description, "winding path")
	assert.Equal(t, "Can the traveler survive the scorching desert crossing?", generated.Purpose)
	assert.Equal(t, "A vulture circles overhead as the path narrows between two boulders.", generated.OpeningHook)
	assert.Len(t, generated.SituationAspects, 2)
	assert.Equal(t, "Blazing Sun", generated.SituationAspects[0])
	assert.Len(t, generated.NPCs, 1)
	assert.Equal(t, "Old Prospector", generated.NPCs[0].Name)
	assert.Equal(t, "friendly", generated.NPCs[0].Disposition)
}

func TestParseGeneratedScene_WithCodeBlock(t *testing.T) {
	// LLMs sometimes wrap JSON in markdown code blocks
	jsonResponse := "```json\n{\"scene_name\": \"Test\", \"description\": \"A test.\", \"purpose\": \"Can the hero prevail?\", \"situation_aspects\": [], \"npcs\": []}\n```"

	generated, err := ParseGeneratedScene(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "Test", generated.SceneName)
	assert.Equal(t, "Can the hero prevail?", generated.Purpose)
}

func TestParseGeneratedScene_MissingFields(t *testing.T) {
	// Missing scene_name
	jsonResponse := `{"description": "A test scene.", "purpose": "Can the hero win?"}`

	_, err := ParseGeneratedScene(jsonResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing scene_name")
}

func TestParseGeneratedScene_MissingPurpose(t *testing.T) {
	jsonResponse := `{
		"scene_name": "The Dusty Trail",
		"description": "A winding path through the desert.",
		"situation_aspects": ["Blazing Sun"],
		"npcs": []
	}`

	_, err := ParseGeneratedScene(jsonResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing purpose")
}

func TestParseGeneratedScene_OptionalOpeningHook(t *testing.T) {
	// opening_hook is optional — parsing should succeed without it
	jsonResponse := `{
		"scene_name": "The Dusty Trail",
		"description": "A winding path through the desert.",
		"purpose": "Can the traveler survive?",
		"situation_aspects": ["Blazing Sun"],
		"npcs": []
	}`

	generated, err := ParseGeneratedScene(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "Can the traveler survive?", generated.Purpose)
	assert.Equal(t, "", generated.OpeningHook)
}

func TestParseGeneratedScene_InvalidJSON(t *testing.T) {
	// Invalid JSON
	jsonResponse := "This is not JSON at all"

	_, err := ParseGeneratedScene(jsonResponse)
	assert.Error(t, err)
}

func TestScenario_Defaults(t *testing.T) {
	scenario := Scenario{}
	assert.Equal(t, "", scenario.Genre)
	assert.Equal(t, "", scenario.Setting)
	assert.Equal(t, "", scenario.Problem)
	assert.False(t, scenario.IsResolved)
}

func TestGeneratedScene_EmptyNPCs(t *testing.T) {
	generated := GeneratedScene{
		SceneName:        "Empty Scene",
		Description:      "A scene with no NPCs",
		SituationAspects: []string{"Lonely"},
		NPCs:             []GeneratedNPC{},
	}

	assert.Equal(t, "Empty Scene", generated.SceneName)
	assert.Len(t, generated.NPCs, 0)
}

func TestScenarioManager_addSceneSummary(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	// Add first summary
	summary1 := &SceneSummary{NarrativeProse: "First scene"}
	sm.addSceneSummary(summary1)
	assert.Len(t, sm.sceneSummaries, 1)

	// Add second summary
	summary2 := &SceneSummary{NarrativeProse: "Second scene"}
	sm.addSceneSummary(summary2)
	assert.Len(t, sm.sceneSummaries, 2)

	// Add third summary
	summary3 := &SceneSummary{NarrativeProse: "Third scene"}
	sm.addSceneSummary(summary3)
	assert.Len(t, sm.sceneSummaries, 3)

	// Add fourth summary - should maintain sliding window of 3
	summary4 := &SceneSummary{NarrativeProse: "Fourth scene"}
	sm.addSceneSummary(summary4)
	assert.Len(t, sm.sceneSummaries, 3)
	assert.Equal(t, "Second scene", sm.sceneSummaries[0].NarrativeProse)
	assert.Equal(t, "Fourth scene", sm.sceneSummaries[2].NarrativeProse)
}

func TestScenarioManager_addSceneSummary_NilSummary(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	// Adding nil should not panic or add anything
	sm.addSceneSummary(nil)
	assert.Len(t, sm.sceneSummaries, 0)
}

func TestScenarioManager_extractComplications(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	// No summaries — no complications
	assert.Empty(t, sm.extractComplications())

	// Add summaries with unresolved threads
	sm.addSceneSummary(&SceneSummary{
		NarrativeProse:    "First scene",
		UnresolvedThreads: []string{"Find the treasure", "Mysterious stranger"},
	})
	sm.addSceneSummary(&SceneSummary{
		NarrativeProse:    "Second scene",
		UnresolvedThreads: []string{"Find the treasure", "Gang hideout location"}, // duplicate thread
	})

	complications := sm.extractComplications()
	assert.Len(t, complications, 3, "Should deduplicate threads across summaries")
	assert.Contains(t, complications, "Find the treasure")
	assert.Contains(t, complications, "Mysterious stranger")
	assert.Contains(t, complications, "Gang hideout location")
}

func TestScenarioManager_extractComplications_NoThreads(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	sm.addSceneSummary(&SceneSummary{
		NarrativeProse: "A scene with no threads",
	})

	assert.Empty(t, sm.extractComplications())
}

func TestParseSceneSummary_Valid(t *testing.T) {
	jsonResponse := `{
		"scene_description": "A dusty saloon",
		"key_events": ["Met the bartender", "Learned about the outlaw"],
		"npcs_encountered": [{"name": "Bartender Bill", "attitude": "friendly"}],
		"aspects_discovered": ["Wanted Posters Everywhere"],
		"unresolved_threads": ["Find the outlaw"],
		"how_ended": "Left through the back door",
		"narrative_prose": "The stranger walked into the saloon and learned of a dangerous outlaw terrorizing the town."
	}`

	summary, err := ParseSceneSummary(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "A dusty saloon", summary.SceneDescription)
	assert.Len(t, summary.KeyEvents, 2)
	assert.Equal(t, "Met the bartender", summary.KeyEvents[0])
	assert.Len(t, summary.NPCsEncountered, 1)
	assert.Equal(t, "Bartender Bill", summary.NPCsEncountered[0].Name)
	assert.Equal(t, "friendly", summary.NPCsEncountered[0].Attitude)
	assert.Len(t, summary.AspectsDiscovered, 1)
	assert.Len(t, summary.UnresolvedThreads, 1)
	assert.Contains(t, summary.NarrativeProse, "stranger walked into")
}

func TestParseSceneSummary_WithCodeBlock(t *testing.T) {
	jsonResponse := "```json\n{\"narrative_prose\": \"A test summary.\"}\n```"

	summary, err := ParseSceneSummary(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "A test summary.", summary.NarrativeProse)
}

func TestParseSceneSummary_MissingNarrativeProse(t *testing.T) {
	// Missing narrative_prose
	jsonResponse := `{"scene_description": "A test scene."}`

	_, err := ParseSceneSummary(jsonResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing narrative_prose")
}

func TestSceneSummary_Fields(t *testing.T) {
	summary := SceneSummary{
		SceneDescription:  "A dusty saloon",
		KeyEvents:         []string{"Event 1", "Event 2"},
		NPCsEncountered:   []NPCSummary{{Name: "Test NPC", Attitude: "friendly"}},
		AspectsDiscovered: []string{"Test Aspect"},
		UnresolvedThreads: []string{"Find the treasure"},
		HowEnded:          "transition",
		NarrativeProse:    "A test narrative",
	}

	assert.Equal(t, "A dusty saloon", summary.SceneDescription)
	assert.Len(t, summary.KeyEvents, 2)
	assert.Len(t, summary.NPCsEncountered, 1)
	assert.Equal(t, "Test NPC", summary.NPCsEncountered[0].Name)
	assert.Equal(t, "A test narrative", summary.NarrativeProse)
}

func TestNPCSummary_Fields(t *testing.T) {
	npc := NPCSummary{
		Name:     "Sheriff Dan",
		Attitude: "hostile",
	}

	assert.Equal(t, "Sheriff Dan", npc.Name)
	assert.Equal(t, "hostile", npc.Attitude)
}

func TestSceneSummaryData_Fields(t *testing.T) {
	data := SceneSummaryData{
		SceneName:        "Test Scene",
		SceneDescription: "A test scene",
		SituationAspects: []string{"Aspect 1"},
		NPCsInScene:      []NPCSummary{{Name: "NPC", Attitude: "neutral"}},
		TakenOutChars:    []string{"enemy1"},
		HowEnded:         "transition",
		TransitionHint:   "the forest",
	}

	assert.Equal(t, "Test Scene", data.SceneName)
	assert.Len(t, data.SituationAspects, 1)
	assert.Len(t, data.NPCsInScene, 1)
	assert.Len(t, data.TakenOutChars, 1)
}

func TestScenario_Fields(t *testing.T) {
	scenario := Scenario{
		Title:          "Test Scenario",
		Problem:        "A test problem",
		StoryQuestions: []string{"Can the hero win?", "Will the villain escape?"},
		Setting:        "A fantasy world",
		Genre:          "Fantasy",
		IsResolved:     false,
	}

	assert.Equal(t, "Test Scenario", scenario.Title)
	assert.Equal(t, "A test problem", scenario.Problem)
	assert.Len(t, scenario.StoryQuestions, 2)
	assert.Equal(t, "Fantasy", scenario.Genre)
	assert.False(t, scenario.IsResolved)
}

func TestScenarioGenerationData_Fields(t *testing.T) {
	data := ScenarioGenerationData{
		PlayerName:        "Test Hero",
		PlayerHighConcept: "Brave Knight",
		PlayerTrouble:     "Hot-Headed",
		PlayerAspects:     []string{"Loyal to Friends"},
		Genre:             "Fantasy",
		Theme:             "Redemption",
	}

	assert.Equal(t, "Test Hero", data.PlayerName)
	assert.Equal(t, "Brave Knight", data.PlayerHighConcept)
	assert.Equal(t, "Fantasy", data.Genre)
}

func TestScenarioResolutionData_Fields(t *testing.T) {
	scenario := &Scenario{
		Title:   "Test",
		Problem: "Problem",
	}

	data := ScenarioResolutionData{
		Scenario:       scenario,
		SceneSummaries: []SceneSummary{{NarrativeProse: "Test"}},
		LatestSummary:  &SceneSummary{NarrativeProse: "Latest"},
		PlayerName:     "Hero",
		PlayerAspects:  []string{"Aspect"},
	}

	assert.Equal(t, scenario, data.Scenario)
	assert.Len(t, data.SceneSummaries, 1)
	assert.NotNil(t, data.LatestSummary)
}

func TestScenarioResolutionResult_Fields(t *testing.T) {
	result := ScenarioResolutionResult{
		IsResolved:        true,
		AnsweredQuestions: []string{"Can the hero win? - YES"},
		Reasoning:         "The hero defeated the villain",
	}

	assert.True(t, result.IsResolved)
	assert.Len(t, result.AnsweredQuestions, 1)
	assert.Equal(t, "The hero defeated the villain", result.Reasoning)
}

func TestParseScenarioResolution_Valid(t *testing.T) {
	jsonResponse := `{
		"is_resolved": true,
		"answered_questions": ["Can the hero win? - YES"],
		"reasoning": "The hero defeated the villain"
	}`

	result, err := ParseScenarioResolution(jsonResponse)
	require.NoError(t, err)

	assert.True(t, result.IsResolved)
	assert.Len(t, result.AnsweredQuestions, 1)
	assert.Equal(t, "The hero defeated the villain", result.Reasoning)
}

func TestParseScenarioResolution_WithCodeBlock(t *testing.T) {
	jsonResponse := "```json\n{\"is_resolved\": false, \"answered_questions\": [], \"reasoning\": \"Not yet\"}\n```"

	result, err := ParseScenarioResolution(jsonResponse)
	require.NoError(t, err)

	assert.False(t, result.IsResolved)
	assert.Len(t, result.AnsweredQuestions, 0)
}

func TestParseScenarioResolution_InvalidJSON(t *testing.T) {
	jsonResponse := "This is not JSON"

	_, err := ParseScenarioResolution(jsonResponse)
	assert.Error(t, err)
}

func TestScenarioResult_Fields(t *testing.T) {
	scenario := &Scenario{Title: "Test"}

	result := ScenarioResult{
		Reason:   ScenarioEndResolved,
		Scenario: scenario,
	}

	assert.Equal(t, ScenarioEndResolved, result.Reason)
	assert.Equal(t, scenario, result.Scenario)
}

func TestScenarioEndReason_Constants(t *testing.T) {
	assert.Equal(t, ScenarioEndReason("resolved"), ScenarioEndResolved)
	assert.Equal(t, ScenarioEndReason("quit"), ScenarioEndQuit)
	assert.Equal(t, ScenarioEndReason("player_taken_out"), ScenarioEndPlayerTakenOut)
}

func TestRenderSceneGeneration_WithComplications(t *testing.T) {
	data := SceneGenerationData{
		TransitionHint:    "the old mine",
		PlayerName:        "Jesse",
		PlayerHighConcept: "Vengeful Rancher",
		Complications: []string{
			"The sheriff is missing",
			"Gang plans to burn the town",
		},
	}

	rendered, err := RenderSceneGeneration(data)
	require.NoError(t, err)

	assert.Contains(t, rendered, "HOOKS TO INCORPORATE")
	assert.Contains(t, rendered, "The sheriff is missing")
	assert.Contains(t, rendered, "Gang plans to burn the town")
}

func TestRenderSceneGeneration_WithoutComplications(t *testing.T) {
	data := SceneGenerationData{
		TransitionHint: "the old mine",
		PlayerName:     "Jesse",
	}

	rendered, err := RenderSceneGeneration(data)
	require.NoError(t, err)

	assert.NotContains(t, rendered, "HOOKS TO INCORPORATE")
}

func TestRenderSceneResponse_WithPurpose(t *testing.T) {
	s := scene.NewScene("s1", "Test Scene", "A test scene")
	data := SceneResponseData{
		Scene:               s,
		CharacterContext:    "Test character",
		AspectsContext:      "Test aspects",
		ConversationContext: "Test conversation",
		PlayerInput:         "I look around",
		InteractionType:     "dialog",
		ScenePurpose:        "Can the hero find the hidden passage?",
	}

	rendered, err := RenderSceneResponse(data)
	require.NoError(t, err)

	assert.Contains(t, rendered, "SCENE PURPOSE")
	assert.Contains(t, rendered, "Can the hero find the hidden passage?")
}

func TestRenderSceneResponse_WithoutPurpose(t *testing.T) {
	s := scene.NewScene("s1", "Test Scene", "A test scene")
	data := SceneResponseData{
		Scene:               s,
		CharacterContext:    "Test character",
		AspectsContext:      "Test aspects",
		ConversationContext: "Test conversation",
		PlayerInput:         "I look around",
		InteractionType:     "dialog",
	}

	rendered, err := RenderSceneResponse(data)
	require.NoError(t, err)

	assert.NotContains(t, rendered, "SCENE PURPOSE")
}

func TestSceneManager_SetScenePurpose(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	sm := NewSceneManager(engine)
	assert.Equal(t, "", sm.scenePurpose)

	sm.SetScenePurpose("Can the hero escape?")
	assert.Equal(t, "Can the hero escape?", sm.scenePurpose)
}
