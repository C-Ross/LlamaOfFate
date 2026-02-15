package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_SaveCascade_ThroughGameManagerRun verifies that after
// GameManager.Start() completes, calling Save() cascades the full snapshot
// through ScenarioManager → SceneManager and captures the correct state.
func TestIntegration_SaveCascade_ThroughGameManagerStart(t *testing.T) {
	// Set up engine with mock LLM (not used — initial scene is pre-configured and quit is immediate)
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	// Create a player with real state
	player := character.NewCharacter("player1", "Jesse Calhoun")
	player.Aspects.HighConcept = "Gunslinger With a Past"
	player.Aspects.Trouble = "Wanted Dead or Alive"
	player.Aspects.AddAspect("Quick Draw")
	player.SetSkill("Shoot", dice.Great)
	player.SetSkill("Notice", dice.Fair)
	player.FatePoints = 5

	// Create a pre-configured scene with NPCs
	testScene := scene.NewScene("saloon_1", "The Dusty Saloon", "A dimly lit saloon at the edge of town.")
	testScene.AddSituationAspect(scene.SituationAspect{ID: "aspect_1", Aspect: "Smoky Atmosphere", Duration: "scene"})
	testScene.AddSituationAspect(scene.SituationAspect{ID: "aspect_2", Aspect: "Swinging Doors", Duration: "scene"})

	bartender := character.NewCharacter("npc_bartender", "Old Pete")
	bartender.Aspects.HighConcept = "Grizzled Barkeep"
	bartender.CharacterType = character.CharacterTypeSupportingNPC

	// Recording saver to capture save calls
	recorder := &recordingSaver{}

	// Set up scenario
	scenario := &scene.Scenario{
		Title:          "The Showdown at High Noon",
		Problem:        "A gang of outlaws threatens the town",
		StoryQuestions: []string{"Will Jesse protect the town?", "Can the outlaws be stopped?"},
		Setting:        "Dusty frontier town",
		Genre:          "Western",
	}

	// Wire up GameManager
	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)
	gm.SetScenario(scenario)

	// Set initial scene and start — game initializes but doesn't loop
	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  []*character.Character{bartender},
	})
	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, events)

	// ScenarioManager should be stored on GameManager after Start
	require.NotNil(t, gm.scenarioManager, "scenarioManager should be stored after Start")

	// Now explicitly call Save
	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates, "expected at least one save call")

	saved := recorder.savedStates[len(recorder.savedStates)-1]

	// --- Verify scenario state ---
	assert.Equal(t, "Jesse Calhoun", saved.Scenario.Player.Name)
	assert.Equal(t, "Gunslinger With a Past", saved.Scenario.Player.Aspects.HighConcept)
	assert.Equal(t, "Wanted Dead or Alive", saved.Scenario.Player.Aspects.Trouble)
	assert.Equal(t, 5, saved.Scenario.Player.FatePoints)
	assert.Equal(t, dice.Great, saved.Scenario.Player.GetSkill("Shoot"))

	assert.Equal(t, "The Showdown at High Noon", saved.Scenario.Scenario.Title)
	assert.Equal(t, "A gang of outlaws threatens the town", saved.Scenario.Scenario.Problem)
	assert.Len(t, saved.Scenario.Scenario.StoryQuestions, 2)
	assert.Equal(t, "Western", saved.Scenario.Scenario.Genre)
	assert.False(t, saved.Scenario.Scenario.IsResolved)

	// --- Verify scene state ---
	require.NotNil(t, saved.Scene.CurrentScene, "current scene should be captured")
	assert.Equal(t, "The Dusty Saloon", saved.Scene.CurrentScene.Name)
	assert.Contains(t, saved.Scene.CurrentScene.Description, "dimly lit saloon")

	// Situation aspects should be present on the scene
	assert.GreaterOrEqual(t, len(saved.Scene.CurrentScene.SituationAspects), 2)

	// Player should be in the scene's character list
	assert.Contains(t, saved.Scene.CurrentScene.Characters, "player1")
}

// TestIntegration_SaveFunc_WiredThroughStart verifies that the saveFunc callback
// set by GameManager on ScenarioManager correctly cascades to the recorder saver.
func TestIntegration_SaveFunc_WiredThroughStart(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	player.Aspects.HighConcept = "Reluctant Champion"

	testScene := scene.NewScene("scene1", "Test Arena", "A testing ground")

	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title:   "Test Scenario",
		Problem: "A test problem",
		Genre:   "Fantasy",
		Setting: "Arena",
	})

	// Start the game
	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, events)

	// Verify saveFunc was wired: call it through the scenarioManager
	require.NotNil(t, gm.scenarioManager)
	require.NotNil(t, gm.scenarioManager.saveFunc, "saveFunc should be wired by GameManager.Start")

	// Calling saveFunc should cascade through GameManager.Save to the recorder
	err = gm.scenarioManager.saveFunc()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates)

	// Verify the most recent save captured state from both layers
	saved := recorder.savedStates[len(recorder.savedStates)-1]
	assert.Equal(t, "Test Hero", saved.Scenario.Player.Name)
	assert.Equal(t, "Test Arena", saved.Scene.CurrentScene.Name)
}

// TestIntegration_SaveCascade_NPCRegistry verifies that NPCs registered during
// scene setup are captured in the snapshot's NPC registry.
func TestIntegration_SaveCascade_NPCRegistry(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"

	testScene := scene.NewScene("scene1", "Saloon", "The saloon")

	marshal := character.NewCharacter("npc_marshal", "Marshal Dan")
	marshal.Aspects.HighConcept = "Stern Lawman"
	marshal.CharacterType = character.CharacterTypeMainNPC

	bartender := character.NewCharacter("npc_bartender", "Old Pete")
	bartender.Aspects.HighConcept = "Grizzled Barkeep"
	bartender.CharacterType = character.CharacterTypeSupportingNPC

	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  []*character.Character{marshal, bartender},
	})
	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	_ = events

	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates)

	saved := recorder.savedStates[len(recorder.savedStates)-1]
	assert.Contains(t, saved.Scenario.NPCRegistry, "marshal dan")
	assert.Contains(t, saved.Scenario.NPCRegistry, "old pete")
	assert.Equal(t, "Stern Lawman", saved.Scenario.NPCRegistry["marshal dan"].Aspects.HighConcept)
	assert.Equal(t, "Grizzled Barkeep", saved.Scenario.NPCRegistry["old pete"].Aspects.HighConcept)
}

// TestIntegration_SaveCascade_MultipleSaves verifies that multiple Save() calls
// produce independent snapshots that reflect state changes between saves.
func TestIntegration_SaveCascade_MultipleSaves(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"
	player.FatePoints = 3

	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// Start triggers scene_start auto-save.
	// Record the baseline count so we can verify our manual saves.
	baseline := len(recorder.savedStates)

	// First manual save
	err = gm.Save()
	require.NoError(t, err)
	require.Len(t, recorder.savedStates, baseline+1)
	assert.Equal(t, 3, recorder.savedStates[baseline].Scenario.Player.FatePoints)

	// Mutate player state
	player.SpendFatePoint()

	// Second manual save — should reflect the change
	err = gm.Save()
	require.NoError(t, err)
	require.Len(t, recorder.savedStates, baseline+2)
	assert.Equal(t, 2, recorder.savedStates[baseline+1].Scenario.Player.FatePoints)
}

// TestIntegration_SaveCascade_NoopSaverByDefault verifies that without calling
// SetSaver, GameManager uses noopSaver and Save() succeeds silently.
func TestIntegration_SaveCascade_NoopSaverByDefault(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	// Deliberately NOT calling SetSaver

	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// Save should succeed silently with noop
	err = gm.Save()
	assert.NoError(t, err)
}

// TestIntegration_SaveCascade_ConversationHistory verifies that conversation
// history accumulated during the scene loop is captured in the snapshot.
func TestIntegration_SaveCascade_ConversationHistory(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// Inject conversation history into the scene manager (simulating exchanges
	// that would happen during a real scene loop before the player quits)
	sceneManager := engine.GetSceneManager()
	sceneManager.conversationHistory = []prompt.ConversationEntry{
		{PlayerInput: "I look around the saloon", GMResponse: "You see a bartender polishing glasses", Type: "dialog"},
		{PlayerInput: "I approach the bar", GMResponse: "The bartender eyes you warily", Type: "dialog"},
	}

	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates, "expected at least one save call")

	saved := recorder.savedStates[len(recorder.savedStates)-1]
	require.Len(t, saved.Scene.ConversationHistory, 2)
	assert.Equal(t, "I look around the saloon", saved.Scene.ConversationHistory[0].PlayerInput)
	assert.Equal(t, "I approach the bar", saved.Scene.ConversationHistory[1].PlayerInput)
}

// TestIntegration_SaveCascade_SceneSummaries verifies that scene summaries
// accumulated across scene transitions are captured in the snapshot.
func TestIntegration_SaveCascade_SceneSummaries(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// Inject scene summaries (simulating multi-scene progression)
	gm.scenarioManager.sceneSummaries = []prompt.SceneSummary{
		{
			SceneDescription:  "The dusty saloon",
			KeyEvents:         []string{"Met the bartender", "Overheard a conversation"},
			NPCsEncountered:   []prompt.NPCSummary{{Name: "Old Pete", Attitude: "friendly"}},
			AspectsDiscovered: []string{"Hidden Door Behind the Bar"},
			UnresolvedThreads: []string{"Who is the mysterious stranger?"},
			HowEnded:          "transition",
			NarrativeProse:    "Jesse pushed through the swinging doors...",
		},
		{
			SceneDescription: "The back alley",
			KeyEvents:        []string{"Found a clue"},
			HowEnded:         "transition",
		},
	}

	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates, "expected at least one save call")

	saved := recorder.savedStates[len(recorder.savedStates)-1]
	require.Len(t, saved.Scenario.SceneSummaries, 2)
	assert.Equal(t, "The dusty saloon", saved.Scenario.SceneSummaries[0].SceneDescription)
	assert.Equal(t, "The back alley", saved.Scenario.SceneSummaries[1].SceneDescription)
	assert.Contains(t, saved.Scenario.SceneSummaries[0].KeyEvents, "Met the bartender")
	assert.Len(t, saved.Scenario.SceneSummaries[0].NPCsEncountered, 1)
	assert.Equal(t, "Old Pete", saved.Scenario.SceneSummaries[0].NPCsEncountered[0].Name)
}

// TestIntegration_AutomaticSaveTriggers verifies that Start() triggers
// an automatic scene_start save.
func TestIntegration_AutomaticSaveTriggers(t *testing.T) {
	mockLLM := &MockLLMClientForScenario{}
	engine, err := NewWithLLM(mockLLM)
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	recorder := &recordingSaver{}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)
	gm.SetScenario(&scene.Scenario{
		Title: "Test", Problem: "Test", Genre: "Western", Setting: "Town",
	})

	// Start — triggers scene_start auto-save
	gm.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  nil,
	})
	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// scene_start auto-save should have fired
	require.Len(t, recorder.savedStates, 1, "expected scene_start save")

	assert.Equal(t, "Jesse", recorder.savedStates[0].Scenario.Player.Name)
	assert.Equal(t, "Saloon", recorder.savedStates[0].Scene.CurrentScene.Name)
}

// TestIntegration_SaveThenResume verifies the full save → load → resume flow:
// play a session (gets saved), then create a new GameManager that
// loads the save and resumes mid-scene.
func TestIntegration_SaveThenResume(t *testing.T) {
	// --- Session 1: Start and save ---
	engine1, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse Calhoun")
	player.Aspects.HighConcept = "Gunslinger With a Past"
	player.Aspects.Trouble = "Wanted Dead or Alive"
	player.FatePoints = 5
	player.SetSkill("Shoot", dice.Great)

	testScene := scene.NewScene("saloon_1", "The Dusty Saloon", "A dimly lit saloon.")
	testScene.AddSituationAspect(scene.SituationAspect{ID: "a1", Aspect: "Smoky Atmosphere", Duration: "scene"})

	bartender := character.NewCharacter("npc_bartender", "Old Pete")
	bartender.Aspects.HighConcept = "Grizzled Barkeep"
	bartender.CharacterType = character.CharacterTypeSupportingNPC

	recorder1 := &recordingSaver{}

	scenario := &scene.Scenario{
		Title:   "The Showdown",
		Problem: "Outlaws threaten the town",
		Genre:   "Western",
		Setting: "Frontier town",
	}

	gm1 := NewGameManager(engine1)
	gm1.SetPlayer(player)
	gm1.SetSaver(recorder1)
	gm1.SetScenario(scenario)

	gm1.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  []*character.Character{bartender},
	})
	events, err := gm1.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// Manual save to capture state
	err = gm1.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder1.savedStates)

	// Get the last saved state
	lastSave := recorder1.savedStates[len(recorder1.savedStates)-1]

	// --- Session 2: Resume from save ---
	engine2, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	recorder2 := &recordingSaver{loadResult: &lastSave}

	gm2 := NewGameManager(engine2)
	gm2.SetPlayer(character.NewCharacter("dummy", "Should Be Overridden"))
	gm2.SetSaver(recorder2)
	gm2.SetScenario(&scene.Scenario{Title: "Should Be Overridden"})

	// Start will load the save and resume
	events, err = gm2.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// Player should be the saved one, not the dummy
	assert.Equal(t, "Jesse Calhoun", gm2.player.Name)
	assert.Equal(t, "Gunslinger With a Past", gm2.player.Aspects.HighConcept)
	assert.Equal(t, 5, gm2.player.FatePoints)
	assert.Equal(t, dice.Great, gm2.player.GetSkill("Shoot"))

	// Save from session 2 should reference the restored state
	err = gm2.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder2.savedStates)
	session2Save := recorder2.savedStates[len(recorder2.savedStates)-1]
	assert.Equal(t, "Jesse Calhoun", session2Save.Scenario.Player.Name)
	assert.Equal(t, "The Showdown", session2Save.Scenario.Scenario.Title)
	assert.Equal(t, "The Dusty Saloon", session2Save.Scene.CurrentScene.Name)
}

// TestIntegration_Resume_PreservesNPCRegistry verifies that NPCs from the NPC
// registry are available after resume and appear in subsequent saves.
func TestIntegration_Resume_PreservesNPCRegistry(t *testing.T) {
	// Session 1: Start with NPCs
	engine1, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"

	testScene := scene.NewScene("scene1", "Saloon", "The saloon")

	marshal := character.NewCharacter("npc_marshal", "Marshal Dan")
	marshal.Aspects.HighConcept = "Stern Lawman"
	marshal.CharacterType = character.CharacterTypeMainNPC

	recorder := &recordingSaver{}

	gm1 := NewGameManager(engine1)
	gm1.SetPlayer(player)
	gm1.SetSaver(recorder)
	gm1.SetScenario(&scene.Scenario{Title: "Test", Problem: "Test", Genre: "Western"})

	gm1.SetInitialScene(&InitialSceneConfig{
		Scene: testScene,
		NPCs:  []*character.Character{marshal},
	})
	events, err := gm1.Start(context.Background())
	require.NoError(t, err)
	_ = events

	err = gm1.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates)

	lastSave := recorder.savedStates[len(recorder.savedStates)-1]

	// Verify NPC in the save
	require.Contains(t, lastSave.Scenario.NPCRegistry, "marshal dan")

	// Session 2: Resume
	engine2, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	recorder2 := &recordingSaver{loadResult: &lastSave}

	gm2 := NewGameManager(engine2)
	gm2.SetPlayer(player)
	gm2.SetSaver(recorder2)

	events, err = gm2.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// NPC should be in the engine's registry after resume
	assert.NotNil(t, engine2.GetCharacter("npc_marshal"))
	assert.Equal(t, "Marshal Dan", engine2.GetCharacter("npc_marshal").Name)

	// And in subsequent saves
	err = gm2.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder2.savedStates)
	resumeSave := recorder2.savedStates[len(recorder2.savedStates)-1]
	assert.Contains(t, resumeSave.Scenario.NPCRegistry, "marshal dan")
}

// TestIntegration_Resume_SkipsSceneStartSave verifies that resuming does NOT
// trigger a redundant "scene_start" save.
func TestIntegration_Resume_SkipsSceneStartSave(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")

	savedState := &GameState{
		Scenario: ScenarioState{
			Player:   player,
			Scenario: &scene.Scenario{Title: "Test", Problem: "Test", Genre: "Western"},
		},
		Scene: SceneState{
			CurrentScene: testScene,
		},
	}

	recorder := &recordingSaver{loadResult: savedState}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)

	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// On resume, no scene_start save should be triggered
	require.Len(t, recorder.savedStates, 0, "expected no auto-saves on resume")
}

// TestIntegration_Resume_MidConflict verifies that resuming into a scene with
// an active conflict preserves the conflict state.
func TestIntegration_Resume_MidConflict(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Jesse")
	player.Aspects.HighConcept = "Gunslinger"

	// Create a scene with an active conflict
	testScene := scene.NewScene("scene1", "Saloon", "The saloon")
	testScene.IsConflict = true
	testScene.ConflictState = &scene.ConflictState{
		Type:  scene.PhysicalConflict,
		Round: 2,
		Participants: []scene.ConflictParticipant{
			{CharacterID: "player1", Initiative: 3},
			{CharacterID: "npc_bandit", Initiative: 1},
		},
		InitiativeOrder: []string{"player1", "npc_bandit"},
		CurrentTurn:     0,
	}

	savedState := &GameState{
		Scenario: ScenarioState{
			Player:   player,
			Scenario: &scene.Scenario{Title: "Test", Problem: "Test", Genre: "Western"},
			NPCRegistry: map[string]*character.Character{
				"bandit": {ID: "npc_bandit", Name: "Bandit"},
			},
		},
		Scene: SceneState{
			CurrentScene: testScene,
			ConversationHistory: []prompt.ConversationEntry{
				{PlayerInput: "I draw my gun!", GMResponse: "The bandit snarls!", Type: "action"},
			},
			ScenePurpose: "Survive the ambush",
		},
	}

	recorder := &recordingSaver{loadResult: savedState}

	gm := NewGameManager(engine)
	gm.SetPlayer(player)
	gm.SetSaver(recorder)

	events, err := gm.Start(context.Background())
	require.NoError(t, err)
	_ = events

	// Save should preserve conflict state
	err = gm.Save()
	require.NoError(t, err)
	require.NotEmpty(t, recorder.savedStates)
	lastSave := recorder.savedStates[len(recorder.savedStates)-1]

	assert.True(t, lastSave.Scene.CurrentScene.IsConflict, "conflict should still be active")
	require.NotNil(t, lastSave.Scene.CurrentScene.ConflictState)
	assert.Equal(t, scene.PhysicalConflict, lastSave.Scene.CurrentScene.ConflictState.Type)
	assert.Equal(t, 2, lastSave.Scene.CurrentScene.ConflictState.Round)
	assert.Len(t, lastSave.Scene.CurrentScene.ConflictState.Participants, 2)
	assert.Equal(t, "Survive the ambush", lastSave.Scene.ScenePurpose)
	assert.Len(t, lastSave.Scene.ConversationHistory, 1)
}
