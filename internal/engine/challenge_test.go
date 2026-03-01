package engine

import (
	"context"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ChallengeManager unit tests ---

func newChallengeTestSetup() (*ChallengeManager, *scene.Scene, *character.Character) {
	player := character.NewCharacter("player-1", "Test Hero")
	player.Skills = map[string]dice.Ladder{
		"Athletics": dice.Good,
		"Stealth":   dice.Fair,
		"Burglary":  dice.Average,
	}
	s := scene.NewScene("test-scene", "Test Scene", "A test scene for challenges")
	s.AddCharacter(player.ID)

	chm := &ChallengeManager{
		player:        player,
		currentScene:  s,
		sessionLogger: session.NullLogger{},
	}
	return chm, s, player
}

func TestChallengeManager_InitiateChallenge(t *testing.T) {
	mockClient := &MockLLMClient{
		response: `{
			"tasks": [
				{"skill": "Athletics", "difficulty": 3, "description": "Scale the wall"},
				{"skill": "Stealth", "difficulty": 2, "description": "Sneak past guards"},
				{"skill": "Burglary", "difficulty": 4, "description": "Pick the lock"}
			]
		}`,
	}

	chm, s, player := newChallengeTestSetup()
	chm.llmClient = mockClient
	chm.challengeGenerator = NewChallengeGenerator(mockClient)

	_ = player // used by chm via constructor

	events, err := chm.initiateChallenge(context.Background(), "Break into the vault")
	require.NoError(t, err)
	require.Len(t, events, 1)

	// Verify event type
	startEvent, ok := events[0].(ChallengeStartEvent)
	require.True(t, ok, "expected ChallengeStartEvent")
	assert.Equal(t, "Break into the vault", startEvent.Description)
	assert.Len(t, startEvent.Tasks, 3)

	// Verify scene state was updated
	assert.True(t, s.IsChallenge)
	require.NotNil(t, s.ChallengeState)
	assert.Equal(t, "Break into the vault", s.ChallengeState.Description)
	assert.Len(t, s.ChallengeState.Tasks, 3)
}

func TestChallengeManager_InitiateChallenge_NoGenerator(t *testing.T) {
	chm, _, _ := newChallengeTestSetup()
	// challengeGenerator is nil

	_, err := chm.initiateChallenge(context.Background(), "Do something")
	assert.ErrorContains(t, err, "challenge generator not available")
}

func TestChallengeManager_InitiateChallenge_MutualExclusion(t *testing.T) {
	mockClient := &MockLLMClient{
		response: `{"tasks": [{"skill": "Athletics", "difficulty": 3, "description": "Run"}]}`,
	}

	chm, s, _ := newChallengeTestSetup()
	chm.llmClient = mockClient
	chm.challengeGenerator = NewChallengeGenerator(mockClient)

	// Start a conflict first
	s.StartConflict(scene.PhysicalConflict, []scene.ConflictParticipant{
		{CharacterID: "player-1", Initiative: 3},
		{CharacterID: "npc-1", Initiative: 1},
	})

	_, err := chm.initiateChallenge(context.Background(), "Escape")
	assert.ErrorContains(t, err, "active conflict")
}

func TestChallengeManager_ResolveTask_Success(t *testing.T) {
	chm, s, _ := newChallengeTestSetup()

	// Manually start a challenge
	tasks := []scene.ChallengeTask{
		{ID: "task-1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending, Description: "Scale the wall"},
		{ID: "task-2", Skill: "Stealth", Difficulty: 2, Status: scene.TaskPending, Description: "Sneak past"},
	}
	err := s.StartChallenge("Test challenge", tasks)
	require.NoError(t, err)

	events := chm.resolveTask("task-1", dice.Success, 2)
	require.Len(t, events, 1)

	taskResult, ok := events[0].(ChallengeTaskResultEvent)
	require.True(t, ok)
	assert.Equal(t, "task-1", taskResult.TaskID)
	assert.Equal(t, "succeeded", taskResult.Outcome)
	assert.Equal(t, "Athletics", taskResult.Skill)
	assert.Equal(t, 2, taskResult.Shifts)
	assert.Equal(t, "Scale the wall", taskResult.Description)

	// Challenge should not be complete (task-2 still pending)
	assert.True(t, s.IsChallenge)
}

func TestChallengeManager_ResolveTask_CompletesChallenge(t *testing.T) {
	chm, s, _ := newChallengeTestSetup()

	tasks := []scene.ChallengeTask{
		{ID: "task-1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskSucceeded, ActorID: "player-1"},
		{ID: "task-2", Skill: "Stealth", Difficulty: 2, Status: scene.TaskPending, Description: "Sneak past"},
	}
	err := s.StartChallenge("Test challenge", tasks)
	require.NoError(t, err)

	events := chm.resolveTask("task-2", dice.SuccessWithStyle, 4)

	// Should get both task result and challenge complete events
	require.Len(t, events, 2)

	_, isTaskResult := events[0].(ChallengeTaskResultEvent)
	assert.True(t, isTaskResult)

	completeEvent, ok := events[1].(ChallengeCompleteEvent)
	require.True(t, ok)
	assert.Equal(t, 2, completeEvent.Successes)
	assert.Equal(t, 0, completeEvent.Failures)
	assert.Equal(t, "success", completeEvent.Overall)

	// Scene should no longer be in challenge
	assert.False(t, s.IsChallenge)
}

func TestChallengeManager_ResolveTask_NoChallengeActive(t *testing.T) {
	chm, _, _ := newChallengeTestSetup()

	events := chm.resolveTask("task-1", dice.Success, 2)
	assert.Empty(t, events)
}

func TestChallengeManager_FindTaskForSkill(t *testing.T) {
	chm, s, _ := newChallengeTestSetup()

	tasks := []scene.ChallengeTask{
		{ID: "task-1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskSucceeded},
		{ID: "task-2", Skill: "Stealth", Difficulty: 2, Status: scene.TaskPending},
	}
	err := s.StartChallenge("Test", tasks)
	require.NoError(t, err)

	// Athletics is already resolved
	task := chm.findTaskForSkill("Athletics")
	assert.Nil(t, task)

	// Stealth is pending
	task = chm.findTaskForSkill("Stealth")
	require.NotNil(t, task)
	assert.Equal(t, "task-2", task.ID)

	// Unknown skill
	task = chm.findTaskForSkill("Fight")
	assert.Nil(t, task)
}

func TestChallengeManager_PlayerSkillNames(t *testing.T) {
	chm, _, _ := newChallengeTestSetup()

	names := chm.playerSkillNames()
	assert.Equal(t, []string{"Athletics", "Burglary", "Stealth"}, names)
}

func TestOutcomeToTaskStatus(t *testing.T) {
	assert.Equal(t, scene.TaskSucceededWithStyle, outcomeToTaskStatus(dice.SuccessWithStyle))
	assert.Equal(t, scene.TaskSucceeded, outcomeToTaskStatus(dice.Success))
	assert.Equal(t, scene.TaskTied, outcomeToTaskStatus(dice.Tie))
	assert.Equal(t, scene.TaskFailed, outcomeToTaskStatus(dice.Failure))
}

func TestBuildChallengeTaskInfos(t *testing.T) {
	cs := &scene.ChallengeState{
		Tasks: []scene.ChallengeTask{
			{ID: "task-1", Description: "Scale wall", Skill: "Athletics", Difficulty: 3, Status: scene.TaskPending},
			{ID: "task-2", Description: "Pick lock", Skill: "Burglary", Difficulty: 4, Status: scene.TaskSucceeded},
		},
	}

	infos := buildChallengeTaskInfos(cs)
	require.Len(t, infos, 2)

	assert.Equal(t, "task-1", infos[0].ID)
	assert.Equal(t, "Scale wall", infos[0].Description)
	assert.Equal(t, "Athletics", infos[0].Skill)
	assert.Equal(t, "Good (+3)", infos[0].Difficulty)
	assert.Equal(t, "pending", infos[0].Status)

	assert.Equal(t, "task-2", infos[1].ID)
	assert.Equal(t, "Great (+4)", infos[1].Difficulty)
	assert.Equal(t, "succeeded", infos[1].Status)
}

func TestChallengeManager_ResolveTask_PartialOutcome(t *testing.T) {
	chm, s, _ := newChallengeTestSetup()

	tasks := []scene.ChallengeTask{
		{ID: "task-1", Skill: "Athletics", Difficulty: 3, Status: scene.TaskSucceeded, ActorID: "player-1"},
		{ID: "task-2", Skill: "Stealth", Difficulty: 2, Status: scene.TaskFailed, ActorID: "player-1"},
		{ID: "task-3", Skill: "Burglary", Difficulty: 4, Status: scene.TaskPending, Description: "Pick the lock"},
	}
	err := s.StartChallenge("Test challenge", tasks)
	require.NoError(t, err)

	events := chm.resolveTask("task-3", dice.Tie, 0)

	// Should get task result + challenge complete
	require.Len(t, events, 2)

	completeEvent, ok := events[1].(ChallengeCompleteEvent)
	require.True(t, ok)
	assert.Equal(t, 1, completeEvent.Successes)
	assert.Equal(t, 1, completeEvent.Failures)
	assert.Equal(t, 1, completeEvent.Ties)
	assert.Equal(t, "partial", completeEvent.Overall)
}

func TestChallengeStartEvent_ImplementsGameEvent(t *testing.T) {
	// Compile-time check that challenge events satisfy GameEvent
	var _ uicontract.GameEvent = ChallengeStartEvent{}
	var _ uicontract.GameEvent = ChallengeTaskResultEvent{}
	var _ uicontract.GameEvent = ChallengeCompleteEvent{}
}
