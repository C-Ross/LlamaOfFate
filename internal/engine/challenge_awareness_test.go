package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Issue #124: Challenge awareness tests ---

func TestHandleInput_SceneTransitionSuppressedDuringChallenge(t *testing.T) {
	// The LLM returns a scene transition marker, but a challenge is active.
	// The code guard should suppress the transition.
	//
	// Sequence:
	//  1. classification → "dialog"
	//  2. scene response → narrative with [SCENE_TRANSITION:...] marker
	client := newTestLLMClient(
		"dialog",
		"You glance at the door but the corridor still rumbles. [SCENE_TRANSITION:the corridor outside]",
	)

	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	sm.conflict.exitOnSceneTransition = true

	testScene := scene.NewScene("mine", "Collapsing Mine", "A mine with crumbling walls")
	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Athletics", dice.Good)
	player.SetSkill("Notice", dice.Fair)
	engine.AddCharacter(player)
	testScene.AddCharacter(player.ID)

	// Start a challenge on this scene
	tasks := []scene.ChallengeTask{
		{ID: "task-1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending, Description: "Dodge falling rocks"},
		{ID: "task-2", Skill: "Notice", Difficulty: 2, Status: scene.TaskPending, Description: "Spot safe path"},
	}
	err = testScene.StartChallenge("Escape the collapsing mine", tasks)
	require.NoError(t, err)

	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.challenge.setSceneState(testScene, player)
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I try to leave")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should get a DialogEvent with cleaned response (marker stripped)
	AssertHasEventIn[DialogEvent](t, result.Events)

	// Should NOT get a SceneTransitionEvent — suppressed by challenge guard
	AssertNoEventIn[SceneTransitionEvent](t, result.Events)

	// Scene should not have ended
	assert.False(t, result.SceneEnded)

	// Challenge should still be active
	assert.True(t, testScene.IsChallenge)
}

func TestHandleInput_SceneTransitionAllowedWithoutChallenge(t *testing.T) {
	// Same scenario without a challenge — transition should proceed normally.
	client := newTestLLMClient(
		"dialog",
		"You step out into the rain. [SCENE_TRANSITION:the rainy streets]",
	)

	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	sm.conflict.exitOnSceneTransition = true

	testScene := scene.NewScene("tavern", "Tavern", "A dimly lit tavern")
	player := character.NewCharacter("player-1", "Hero")
	engine.AddCharacter(player)
	testScene.AddCharacter(player.ID)

	// No challenge active
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I leave the tavern")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have SceneTransitionEvent
	AssertHasEventIn[SceneTransitionEvent](t, result.Events)

	// Scene should have ended
	assert.True(t, result.SceneEnded)
}

func TestHandleInput_ChallengeCreateAdvantageOverriddenToOvercome(t *testing.T) {
	// The LLM classifies a challenge-matching action as Create an Advantage.
	// The code guard should override it to Overcome and emit a warning.
	//
	// Sequence:
	//  1. classification → "action"
	//  2. action parse   → Create an Advantage with Notice (matches challenge task)
	//  3. narrative       → flavor text
	client := newTestLLMClient(
		"action",
		`{"action_type":"Create an Advantage","skill":"Notice","description":"identify hidden hazards","difficulty":2}`,
		"You scan the corridor and spot the hazards!",
	)

	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	sm.actions.roller = dice.NewSeededRoller(42)

	testScene := scene.NewScene("mine", "Collapsing Mine", "A mine with crumbling walls")
	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Notice", dice.Fair) // +2
	engine.AddCharacter(player)
	testScene.AddCharacter(player.ID)

	// Start a challenge with a Notice task
	tasks := []scene.ChallengeTask{
		{ID: "task-1", Skill: "Notice", Difficulty: 2, Status: scene.TaskPending, Description: "Spot safe path"},
		{ID: "task-2", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending, Description: "Dodge falling rocks"},
	}
	err = testScene.StartChallenge("Escape the collapsing mine", tasks)
	require.NoError(t, err)

	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.challenge.setSceneState(testScene, player)
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I scan for hidden hazards")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have resolved the challenge task — this proves the CaA was
	// overridden to Overcome, because only Overcome resolves challenge tasks.
	taskResult := RequireFirstFrom[ChallengeTaskResultEvent](t, result.Events)
	assert.Equal(t, "task-1", taskResult.TaskID)
	assert.Equal(t, "Notice", taskResult.Skill)

	// Should also have a dice-roll result for the action
	AssertHasEventIn[ActionResultEvent](t, result.Events)
}

func TestHandleInput_ChallengeOvercomeNotOverridden(t *testing.T) {
	// When the LLM correctly classifies as Overcome during a challenge,
	// no override should happen — just normal resolution.
	client := newTestLLMClient(
		"action",
		`{"action_type":"Overcome","skill":"Athletics","description":"dodge the rocks","difficulty":3}`,
		"You dodge the falling debris!",
	)

	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	sm.actions.roller = dice.NewSeededRoller(42)

	testScene := scene.NewScene("mine", "Collapsing Mine", "A mine with crumbling walls")
	player := character.NewCharacter("player-1", "Hero")
	player.SetSkill("Athletics", dice.Good) // +3
	engine.AddCharacter(player)
	testScene.AddCharacter(player.ID)

	tasks := []scene.ChallengeTask{
		{ID: "task-1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending, Description: "Dodge falling rocks"},
		{ID: "task-2", Skill: "Notice", Difficulty: 2, Status: scene.TaskPending, Description: "Spot safe path"},
	}
	err = testScene.StartChallenge("Escape the collapsing mine", tasks)
	require.NoError(t, err)

	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.challenge.setSceneState(testScene, player)
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I dodge the rocks")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should resolve the challenge task (Overcome used directly — no override)
	taskResult := RequireFirstFrom[ChallengeTaskResultEvent](t, result.Events)
	assert.Equal(t, "task-1", taskResult.TaskID)
	assert.Equal(t, "Athletics", taskResult.Skill)

	AssertHasEventIn[ActionResultEvent](t, result.Events)
}
