package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScenarioManager(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	assert.NotNil(t, sm)
	assert.Equal(t, engine, sm.engine)
	assert.Equal(t, player, sm.player)
}

func TestScenarioManager_SetScenario(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	scenario := &scene.Scenario{
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
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	npc := core.NewCharacter("npc1", "Test NPC")

	sm.SetInitialScene(testScene, []*core.Character{npc})

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

	generated, err := prompt.ParseGeneratedScene(jsonResponse)
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

	generated, err := prompt.ParseGeneratedScene(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "Test", generated.SceneName)
	assert.Equal(t, "Can the hero prevail?", generated.Purpose)
}

func TestParseGeneratedScene_MissingFields(t *testing.T) {
	// Missing scene_name
	jsonResponse := `{"description": "A test scene.", "purpose": "Can the hero win?"}`

	_, err := prompt.ParseGeneratedScene(jsonResponse)
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

	_, err := prompt.ParseGeneratedScene(jsonResponse)
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

	generated, err := prompt.ParseGeneratedScene(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "Can the traveler survive?", generated.Purpose)
	assert.Equal(t, "", generated.OpeningHook)
}

func TestParseGeneratedScene_InvalidJSON(t *testing.T) {
	// Invalid JSON
	jsonResponse := "This is not JSON at all"

	_, err := prompt.ParseGeneratedScene(jsonResponse)
	assert.Error(t, err)
}

func TestScenario_Defaults(t *testing.T) {
	scenario := scene.Scenario{}
	assert.Equal(t, "", scenario.Genre)
	assert.Equal(t, "", scenario.Setting)
	assert.Equal(t, "", scenario.Problem)
	assert.False(t, scenario.IsResolved)
}

func TestGeneratedScene_EmptyNPCs(t *testing.T) {
	generated := prompt.GeneratedScene{
		SceneName:        "Empty Scene",
		Description:      "A scene with no NPCs",
		SituationAspects: []string{"Lonely"},
		NPCs:             []prompt.GeneratedNPC{},
	}

	assert.Equal(t, "Empty Scene", generated.SceneName)
	assert.Len(t, generated.NPCs, 0)
}

func TestScenarioManager_addSceneSummary(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Add first summary
	summary1 := &prompt.SceneSummary{NarrativeProse: "First scene"}
	sm.addSceneSummary(summary1)
	assert.Len(t, sm.sceneSummaries, 1)

	// Add second summary
	summary2 := &prompt.SceneSummary{NarrativeProse: "Second scene"}
	sm.addSceneSummary(summary2)
	assert.Len(t, sm.sceneSummaries, 2)

	// Add third summary
	summary3 := &prompt.SceneSummary{NarrativeProse: "Third scene"}
	sm.addSceneSummary(summary3)
	assert.Len(t, sm.sceneSummaries, 3)

	// Add fourth summary - should maintain sliding window of 3
	summary4 := &prompt.SceneSummary{NarrativeProse: "Fourth scene"}
	sm.addSceneSummary(summary4)
	assert.Len(t, sm.sceneSummaries, 3)
	assert.Equal(t, "Second scene", sm.sceneSummaries[0].NarrativeProse)
	assert.Equal(t, "Fourth scene", sm.sceneSummaries[2].NarrativeProse)
}

func TestScenarioManager_addSceneSummary_NilSummary(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Adding nil should not panic or add anything
	sm.addSceneSummary(nil)
	assert.Len(t, sm.sceneSummaries, 0)
}

func TestScenarioManager_extractComplications(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// No summaries — no complications
	assert.Empty(t, sm.extractComplications())

	// Add summaries with unresolved threads
	sm.addSceneSummary(&prompt.SceneSummary{
		NarrativeProse:    "First scene",
		UnresolvedThreads: []string{"Find the treasure", "Mysterious stranger"},
	})
	sm.addSceneSummary(&prompt.SceneSummary{
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
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	sm.addSceneSummary(&prompt.SceneSummary{
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

	summary, err := prompt.ParseSceneSummary(jsonResponse)
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

	summary, err := prompt.ParseSceneSummary(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "A test summary.", summary.NarrativeProse)
}

func TestParseSceneSummary_MissingNarrativeProse(t *testing.T) {
	// Missing narrative_prose
	jsonResponse := `{"scene_description": "A test scene."}`

	_, err := prompt.ParseSceneSummary(jsonResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing narrative_prose")
}

func TestSceneSummary_Fields(t *testing.T) {
	summary := prompt.SceneSummary{
		SceneDescription:  "A dusty saloon",
		KeyEvents:         []string{"Event 1", "Event 2"},
		NPCsEncountered:   []prompt.NPCSummary{{Name: "Test NPC", Attitude: "friendly"}},
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
	npc := prompt.NPCSummary{
		Name:     "Sheriff Dan",
		Attitude: "hostile",
	}

	assert.Equal(t, "Sheriff Dan", npc.Name)
	assert.Equal(t, "hostile", npc.Attitude)
}

func TestSceneSummaryData_Fields(t *testing.T) {
	data := prompt.SceneSummaryData{
		SceneName:        "Test Scene",
		SceneDescription: "A test scene",
		SituationAspects: []string{"Aspect 1"},
		NPCsInScene:      []prompt.NPCSummary{{Name: "NPC", Attitude: "neutral"}},
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
	scenario := scene.Scenario{
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
	data := prompt.ScenarioGenerationData{
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
	scenario := &scene.Scenario{
		Title:   "Test",
		Problem: "Problem",
	}

	data := prompt.ScenarioResolutionData{
		Scenario:       scenario,
		SceneSummaries: []prompt.SceneSummary{{NarrativeProse: "Test"}},
		LatestSummary:  &prompt.SceneSummary{NarrativeProse: "Latest"},
		PlayerName:     "Hero",
		PlayerAspects:  []string{"Aspect"},
	}

	assert.Equal(t, scenario, data.Scenario)
	assert.Len(t, data.SceneSummaries, 1)
	assert.NotNil(t, data.LatestSummary)
}

func TestScenarioResolutionResult_Fields(t *testing.T) {
	result := prompt.ScenarioResolutionResult{
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

	result, err := prompt.ParseScenarioResolution(jsonResponse)
	require.NoError(t, err)

	assert.True(t, result.IsResolved)
	assert.Len(t, result.AnsweredQuestions, 1)
	assert.Equal(t, "The hero defeated the villain", result.Reasoning)
}

func TestParseScenarioResolution_WithCodeBlock(t *testing.T) {
	jsonResponse := "```json\n{\"is_resolved\": false, \"answered_questions\": [], \"reasoning\": \"Not yet\"}\n```"

	result, err := prompt.ParseScenarioResolution(jsonResponse)
	require.NoError(t, err)

	assert.False(t, result.IsResolved)
	assert.Len(t, result.AnsweredQuestions, 0)
}

func TestParseScenarioResolution_InvalidJSON(t *testing.T) {
	jsonResponse := "This is not JSON"

	_, err := prompt.ParseScenarioResolution(jsonResponse)
	assert.Error(t, err)
}

func TestScenarioResult_Fields(t *testing.T) {
	scenario := &scene.Scenario{Title: "Test"}

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
	data := prompt.SceneGenerationData{
		TransitionHint:    "the old mine",
		PlayerName:        "Jesse",
		PlayerHighConcept: "Vengeful Rancher",
		Complications: []string{
			"The sheriff is missing",
			"Gang plans to burn the town",
		},
	}

	rendered, err := prompt.RenderSceneGeneration(data)
	require.NoError(t, err)

	assert.Contains(t, rendered, "HOOKS TO INCORPORATE")
	assert.Contains(t, rendered, "The sheriff is missing")
	assert.Contains(t, rendered, "Gang plans to burn the town")
}

func TestRenderSceneGeneration_WithoutComplications(t *testing.T) {
	data := prompt.SceneGenerationData{
		TransitionHint: "the old mine",
		PlayerName:     "Jesse",
	}

	rendered, err := prompt.RenderSceneGeneration(data)
	require.NoError(t, err)

	assert.NotContains(t, rendered, "HOOKS TO INCORPORATE")
}

func TestRenderSceneGeneration_WithKnownNPCs(t *testing.T) {
	data := prompt.SceneGenerationData{
		TransitionHint:    "the tavern",
		PlayerName:        "Jesse",
		PlayerHighConcept: "Vengeful Rancher",
		KnownNPCs: []prompt.NPCSummary{
			{Name: "Greta Ironheart", Attitude: "friendly"},
			{Name: "Dark Raven", Attitude: "hostile"},
		},
	}

	rendered, err := prompt.RenderSceneGeneration(data)
	require.NoError(t, err)

	assert.Contains(t, rendered, "Greta Ironheart")
	assert.Contains(t, rendered, "friendly")
	assert.Contains(t, rendered, "Dark Raven")
	assert.Contains(t, rendered, "hostile")
}

func TestRenderSceneResponse_WithPurpose(t *testing.T) {
	s := scene.NewScene("s1", "Test Scene", "A test scene")
	data := prompt.SceneResponseData{
		Scene:               s,
		CharacterContext:    "Test character",
		AspectsContext:      "Test aspects",
		ConversationContext: "Test conversation",
		PlayerInput:         "I look around",
		InteractionType:     "dialog",
		ScenePurpose:        "Can the hero find the hidden passage?",
	}

	rendered, err := prompt.RenderSceneResponse(data)
	require.NoError(t, err)

	assert.Contains(t, rendered, "SCENE PURPOSE")
	assert.Contains(t, rendered, "Can the hero find the hidden passage?")
}

func TestRenderSceneResponse_WithoutPurpose(t *testing.T) {
	s := scene.NewScene("s1", "Test Scene", "A test scene")
	data := prompt.SceneResponseData{
		Scene:               s,
		CharacterContext:    "Test character",
		AspectsContext:      "Test aspects",
		ConversationContext: "Test conversation",
		PlayerInput:         "I look around",
		InteractionType:     "dialog",
	}

	rendered, err := prompt.RenderSceneResponse(data)
	require.NoError(t, err)

	assert.NotContains(t, rendered, "SCENE PURPOSE")
}

func TestSceneManager_SetScenePurpose(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	assert.Equal(t, "", sm.scenePurpose)

	sm.SetScenePurpose("Can the hero escape?")
	assert.Equal(t, "Can the hero escape?", sm.scenePurpose)
}

func TestNormalizeNPCName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Greta Ironheart", "greta ironheart"},
		{"  Greta Ironheart  ", "greta ironheart"},
		{"GRETA", "greta"},
		{"", ""},
		{"mixed Case NAME", "mixed case name"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, normalizeNPCName(tt.input), "normalizeNPCName(%q)", tt.input)
	}
}

func TestNewScenarioManager_InitializesNPCRegistry(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	assert.NotNil(t, sm.npcRegistry)
	registry, attitudes := sm.npcRegistry.Snapshot()
	assert.Empty(t, registry)
	assert.Empty(t, attitudes)
}

func TestScenarioManager_SetScenarioCount(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	sm.SetScenarioCount(3)
	assert.Equal(t, 3, sm.scenarioCount)
}

func TestScenarioManager_GetKnownNPCSummaries(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Empty registry
	summaries := sm.npcRegistry.KnownSummaries()
	assert.Empty(t, summaries)

	// Add an NPC to registry
	npc := core.NewCharacter("npc1", "Greta Ironheart")
	npc.CharacterType = core.CharacterTypeMainNPC
	npc.Aspects.HighConcept = "Dwarven Smith"
	sm.npcRegistry.Register(npc, "friendly")

	summaries = sm.npcRegistry.KnownSummaries()
	assert.Len(t, summaries, 1)
	assert.Equal(t, "Greta Ironheart", summaries[0].Name)
	assert.Equal(t, "friendly", summaries[0].Attitude)
}

func TestScenarioManager_GetKnownNPCSummaries_ExcludesPermanentlyRemoved(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Add a normal NPC
	alive := core.NewCharacter("npc1", "Greta Ironheart")
	sm.npcRegistry.Register(alive, "friendly")

	// Add a permanently removed NPC (killed)
	dead := core.NewCharacter("npc2", "Bandit Leader")
	dead.Fate = &core.TakenOutFate{Description: "killed in the explosion", Permanent: true}
	sm.npcRegistry.Register(dead, "hostile")

	summaries := sm.npcRegistry.KnownSummaries()
	assert.Len(t, summaries, 1)
	assert.Equal(t, "Greta Ironheart", summaries[0].Name)
	assert.Equal(t, "friendly", summaries[0].Attitude)
}

func TestScenarioManager_GetKnownNPCSummaries_TemporaryDefeatShowsFate(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Add a temporarily defeated NPC (knocked unconscious)
	knocked := core.NewCharacter("npc1", "Guard Captain")
	knocked.Fate = &core.TakenOutFate{Description: "knocked unconscious", Permanent: false}
	sm.npcRegistry.Register(knocked, "hostile")

	summaries := sm.npcRegistry.KnownSummaries()
	require.Len(t, summaries, 1)
	assert.Equal(t, "Guard Captain", summaries[0].Name)
	assert.Equal(t, "defeated (knocked unconscious)", summaries[0].Attitude)
}

func TestScenarioManager_GetKnownNPCSummaries_MixedFates(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Normal NPC
	normal := core.NewCharacter("npc1", "Shopkeeper")
	sm.npcRegistry.Register(normal, "neutral")

	// Temporarily defeated NPC
	temp := core.NewCharacter("npc2", "Thug")
	temp.Fate = &core.TakenOutFate{Description: "fled in terror", Permanent: false}
	sm.npcRegistry.Register(temp, "")

	// Permanently removed NPC
	perm := core.NewCharacter("npc3", "Assassin")
	perm.Fate = &core.TakenOutFate{Description: "fell off the cliff", Permanent: true}
	sm.npcRegistry.Register(perm, "")

	summaries := sm.npcRegistry.KnownSummaries()
	assert.Len(t, summaries, 2, "should exclude permanently removed NPC")

	nameMap := make(map[string]string)
	for _, s := range summaries {
		nameMap[s.Name] = s.Attitude
	}
	assert.Equal(t, "neutral", nameMap["Shopkeeper"])
	assert.Equal(t, "defeated (fled in terror)", nameMap["Thug"])
	_, found := nameMap["Assassin"]
	assert.False(t, found, "permanently removed NPC should not appear")
}

func TestScenarioManager_GenerateNextScene_SkipsPermanentlyRemovedNPCs(t *testing.T) {
	// Mock LLM returns a scene that tries to reintroduce a permanently removed NPC
	sceneJSON := `{
		"scene_name": "The Aftermath",
		"description": "The dust settles.",
		"purpose": "Can the hero move on?",
		"situation_aspects": ["Eerie Silence"],
		"npcs": [
			{"name": "Bandit Leader", "high_concept": "Ruthless Outlaw", "disposition": "hostile"},
			{"name": "Innkeeper", "high_concept": "Friendly Barkeep", "disposition": "friendly"}
		]
	}`
	engine, err := NewWithLLM(newTestLLMClient(sceneJSON), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())
	sm.scenario = &scene.Scenario{
		Title:   "Test Scenario",
		Problem: "A test problem",
		Genre:   "Fantasy",
	}

	// Pre-register "Bandit Leader" as permanently removed
	dead := core.NewCharacter("old_npc_1", "Bandit Leader")
	dead.Fate = &core.TakenOutFate{Description: "killed", Permanent: true}
	sm.npcRegistry.Register(dead, "hostile")

	newScene, err := sm.generateNextScene(context.Background(), "moving on")
	require.NoError(t, err)
	require.NotNil(t, newScene)

	// The permanently removed NPC should NOT be in the scene's character list
	for _, charID := range newScene.Characters {
		if charID == player.ID {
			continue
		}
		char := engine.GetCharacter(charID)
		if char != nil {
			assert.NotEqual(t, normalizeNPCName("Bandit Leader"), normalizeNPCName(char.Name),
				"permanently removed NPC should not appear in new scene")
		}
	}

	// The Innkeeper should be in the scene
	foundInnkeeper := false
	for _, charID := range newScene.Characters {
		char := engine.GetCharacter(charID)
		if char != nil && normalizeNPCName(char.Name) == normalizeNPCName("Innkeeper") {
			foundInnkeeper = true
			break
		}
	}
	assert.True(t, foundInnkeeper, "non-removed NPC should appear in scene")
}

func TestScenarioManager_UpdateNPCAttitudes(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Register an NPC so the attitude update recognizes it
	npc := core.NewCharacter("npc1", "Greta Ironheart")
	sm.npcRegistry.Register(npc, "neutral")

	summary := &prompt.SceneSummary{
		NPCsEncountered: []prompt.NPCSummary{
			{Name: "Greta Ironheart", Attitude: "hostile"},
		},
	}

	sm.npcRegistry.UpdateAttitudesFromSummary(summary)
	_, attitudes := sm.npcRegistry.Snapshot()
	assert.Equal(t, "hostile", attitudes[normalizeNPCName("Greta Ironheart")])
}

func TestScenarioManager_UpdateNPCAttitudes_NilSummary(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Should not panic
	sm.npcRegistry.UpdateAttitudesFromSummary(nil)
}

func TestScenarioManager_BestRecoverySkill(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	player.SetSkill("Physique", 3)
	player.SetSkill("Will", 2)
	player.SetSkill("Athletics", 1)
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Physical consequence should prefer Physique
	physConseq := core.Consequence{ID: "c1", Type: core.MildConsequence, Aspect: "Bruised Ribs"}
	skill, rating := sm.bestRecoverySkill(physConseq)
	// Should return highest skill (Physique at +3) since we can't know consequence type from aspect alone
	assert.NotEmpty(t, skill)
	assert.True(t, rating >= 0, "Rating should be >= 0, got %d", rating)
}

func TestScenarioManager_HandleBetweenSceneRecovery_NoConsequences(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	events := sm.handleBetweenSceneRecovery(context.Background())

	assert.Empty(t, events)
}

func TestScenarioManager_HandleBetweenSceneRecovery_HealedConsequence(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	// Add a recovering mild consequence that should heal (recovery was started at scene 0)
	player.Consequences = []core.Consequence{
		{
			ID:                    "c1",
			Type:                  core.MildConsequence,
			Aspect:                "Bruised Ribs",
			Recovering:            true,
			RecoveryStartScene:    0,
			RecoveryStartScenario: 0,
		},
	}
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())
	sm.sceneCount = 2 // Mild consequences heal after 1 scene

	events := sm.handleBetweenSceneRecovery(context.Background())

	// Should have a healed event
	require.NotEmpty(t, events)
	recovery, ok := events[0].(RecoveryEvent)
	require.True(t, ok, "expected RecoveryEvent, got %T", events[0])
	assert.Equal(t, "healed", recovery.Action)
	assert.Equal(t, "Bruised Ribs", recovery.Aspect)
	assert.Equal(t, "mild", recovery.Severity)
}

func TestScenarioManager_HandleBetweenSceneRecovery_RollEvents(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	player.SetSkill("Will", 2)
	// Add a non-recovering consequence that needs a recovery roll
	player.Consequences = []core.Consequence{
		{
			ID:         "c1",
			Type:       core.MildConsequence,
			Aspect:     "Scratched Up",
			Recovering: false,
		},
	}
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())
	sm.sceneCount = 1

	events := sm.handleBetweenSceneRecovery(context.Background())

	// Should have at least one roll event (no LLM client = no narrative events)
	require.NotEmpty(t, events)
	recovery, ok := events[0].(RecoveryEvent)
	require.True(t, ok, "expected RecoveryEvent, got %T", events[0])
	assert.Equal(t, "roll", recovery.Action)
	assert.Equal(t, "Scratched Up", recovery.Aspect)
	assert.Equal(t, "mild", recovery.Severity)
	assert.NotEmpty(t, recovery.Skill)
	assert.NotEmpty(t, recovery.Difficulty)
}

func TestScenarioManager_BuildRecoveryNarrativeEvents_Empty(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	events := sm.buildRecoveryNarrativeEvents(context.Background(), nil)

	assert.Nil(t, events)
}

func TestScenarioManager_BuildRecoveryNarrativeEvents_ReturnsRollEvents(t *testing.T) {
	engine, err := NewWithLLM(nil, session.NullLogger{}) // nil LLM so no narrative generation
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	attempts := []prompt.RecoveryAttempt{
		{
			Severity:   "mild",
			Aspect:     "Scratched Up",
			Difficulty: "Great (+4)",
			Skill:      "Will",
			RollResult: 3,
			Outcome:    "failure",
		},
		{
			Severity:   "moderate",
			Aspect:     "Broken Arm",
			Difficulty: "Fantastic (+6)",
			Skill:      "Lore",
			RollResult: 6,
			Outcome:    "success",
		},
	}

	events := sm.buildRecoveryNarrativeEvents(context.Background(), attempts)

	require.Len(t, events, 2)

	roll1, ok := events[0].(RecoveryEvent)
	require.True(t, ok)
	assert.Equal(t, "roll", roll1.Action)
	assert.Equal(t, "Scratched Up", roll1.Aspect)
	assert.Equal(t, "mild", roll1.Severity)
	assert.Equal(t, "Will", roll1.Skill)
	assert.Equal(t, 3, roll1.RollResult)
	assert.Equal(t, "Great (+4)", roll1.Difficulty)
	assert.False(t, roll1.Success)

	roll2, ok := events[1].(RecoveryEvent)
	require.True(t, ok)
	assert.Equal(t, "roll", roll2.Action)
	assert.Equal(t, "Broken Arm", roll2.Aspect)
	assert.Equal(t, "moderate", roll2.Severity)
	assert.True(t, roll2.Success)
}

// --- ScenarioManager.Start tests ---

func TestScenarioManager_Start_RequiresEngine(t *testing.T) {
	sm := &ScenarioManager{}
	_, err := sm.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "engine is required")
}

func TestScenarioManager_Start_RequiresLLM(t *testing.T) {
	engine, err := New(session.NullLogger{}) // No LLM
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	_, err = sm.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM client is required")
}

func TestScenarioManager_Start_RequiresPlayer(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	sm := &ScenarioManager{engine: engine}

	_, err = sm.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player character is required")
}

func TestScenarioManager_Start_WithInitialScene(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	testScene := scene.NewScene("scene1", "The Dusty Trail", "A winding desert path")
	sm.SetInitialScene(testScene, nil)

	events, err := sm.Start(context.Background())
	require.NoError(t, err)

	// Should have a NarrativeEvent with the scene name and description
	require.NotEmpty(t, events)
	narrative := events[0].(NarrativeEvent)
	assert.Equal(t, "The Dusty Trail", narrative.SceneName)
	assert.Equal(t, "A winding desert path", narrative.Text)

	// Should be marked as started
	assert.True(t, sm.started)
}

func TestScenarioManager_Start_WithPurposeAndHook(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Set up a mock LLM that returns scene generation with purpose and hook
	sm.SetScenario(&scene.Scenario{Title: "Test", Genre: "Fantasy"})

	// Use initial scene and manually set purpose/hook to avoid LLM dependency
	testScene := scene.NewScene("scene1", "The Tavern", "A cozy tavern")
	sm.SetInitialScene(testScene, nil)
	sm.lastGeneratedPurpose = "Can the hero find allies?"
	sm.lastGeneratedHook = "A stranger beckons from the corner."

	events, err := sm.Start(context.Background())
	require.NoError(t, err)

	// First event: scene name/description
	require.Len(t, events, 2)
	sceneNarrative := events[0].(NarrativeEvent)
	assert.Equal(t, "The Tavern", sceneNarrative.SceneName)

	// Second event: purpose/hook
	hookNarrative := events[1].(NarrativeEvent)
	assert.Equal(t, "Can the hero find allies?", hookNarrative.Purpose)
	assert.Equal(t, "A stranger beckons from the corner.", hookNarrative.Text)
}

func TestScenarioManager_Start_Resume(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	// Set up a scene in the SceneManager and mark as resumed
	testScene := scene.NewScene("scene1", "The Vault", "A bank vault")
	sceneManager := engine.GetSceneManager()
	require.NoError(t, sceneManager.StartScene(testScene, player))

	sm.Restore(ScenarioState{
		Player:   player,
		Scenario: &scene.Scenario{Title: "The Heist"},
	}, SceneState{
		CurrentScene: testScene,
	})

	events, err := sm.Start(context.Background())
	require.NoError(t, err)

	// Should have the scene narrative
	require.NotEmpty(t, events)
	narrative := events[0].(NarrativeEvent)
	assert.Equal(t, "The Vault", narrative.SceneName)
	assert.Equal(t, "A bank vault", narrative.Text)
}

func TestScenarioManager_HandleInput_BeforeStart(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	_, err = sm.HandleInput(context.Background(), "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HandleInput called before Start")
}

func TestScenarioManager_ProvideInvokeResponse_BeforeStart(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	_, err = sm.ProvideInvokeResponse(context.Background(), InvokeResponse{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ProvideInvokeResponse called before Start")
}

func TestScenarioManager_ProvideMidFlowResponse_BeforeStart(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())

	_, err = sm.ProvideMidFlowResponse(context.Background(), MidFlowResponse{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ProvideMidFlowResponse called before Start")
}

func TestScenarioManager_HandleSceneEnd_Quit(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())
	sm.scenario = &scene.Scenario{Title: "Test"}

	sceneManager := engine.GetSceneManager()
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	require.NoError(t, sceneManager.StartScene(testScene, player))
	sm.currentScene = testScene

	events, result, err := sm.handleSceneEnd(context.Background(), sceneManager, &SceneEndResult{
		Reason: SceneEndQuit,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScenarioEndQuit, result.Reason)
	assert.Empty(t, events) // No extra events on quit
}

func TestScenarioManager_HandleSceneEnd_PlayerTakenOut(t *testing.T) {
	engine, err := NewWithLLM(newTestLLMClient(), session.NullLogger{})
	require.NoError(t, err)

	player := core.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player, session.NullLogger{}, NewNPCRegistry())
	sm.scenario = &scene.Scenario{Title: "Test"}

	sceneManager := engine.GetSceneManager()
	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	require.NoError(t, sceneManager.StartScene(testScene, player))
	sm.currentScene = testScene

	// Player taken out with no transition hint = game over
	events, result, err := sm.handleSceneEnd(context.Background(), sceneManager, &SceneEndResult{
		Reason: SceneEndPlayerTakenOut,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, ScenarioEndPlayerTakenOut, result.Reason)
	assert.Empty(t, events)
}
