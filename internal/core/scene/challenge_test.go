package scene

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ChallengeState method tests ---

func TestChallengeState_PendingTasks_AllPending(t *testing.T) {
	cs := &ChallengeState{
		Description: "Escape the burning building",
		Tasks: []ChallengeTask{
			{ID: "t1", Skill: "Athletics", Difficulty: 3, Status: TaskPending},
			{ID: "t2", Skill: "Investigate", Difficulty: 2, Status: TaskPending},
			{ID: "t3", Skill: "Rapport", Difficulty: 1, Status: TaskPending},
		},
	}

	pending := cs.PendingTasks()
	assert.Len(t, pending, 3)
}

func TestChallengeState_PendingTasks_SomeResolved(t *testing.T) {
	cs := &ChallengeState{
		Tasks: []ChallengeTask{
			{ID: "t1", Skill: "Athletics", Status: TaskSucceeded},
			{ID: "t2", Skill: "Investigate", Status: TaskPending},
			{ID: "t3", Skill: "Rapport", Status: TaskFailed},
		},
	}

	pending := cs.PendingTasks()
	require.Len(t, pending, 1)
	assert.Equal(t, "t2", pending[0].ID)
}

func TestChallengeState_PendingTasks_NoneLeft(t *testing.T) {
	cs := &ChallengeState{
		Tasks: []ChallengeTask{
			{ID: "t1", Status: TaskSucceeded},
			{ID: "t2", Status: TaskFailed},
		},
	}

	pending := cs.PendingTasks()
	assert.Empty(t, pending)
}

func TestChallengeState_Tally(t *testing.T) {
	cs := &ChallengeState{
		Tasks: []ChallengeTask{
			{ID: "t1", Status: TaskSucceeded},
			{ID: "t2", Status: TaskFailed},
			{ID: "t3", Status: TaskTied},
			{ID: "t4", Status: TaskSucceededWithStyle},
			{ID: "t5", Status: TaskPending},
		},
	}

	successes, failures, ties := cs.Tally()
	assert.Equal(t, 2, successes, "succeeded + succeeded_with_style both count as successes")
	assert.Equal(t, 1, failures)
	assert.Equal(t, 1, ties)
}

func TestChallengeState_Tally_AllPending(t *testing.T) {
	cs := &ChallengeState{
		Tasks: []ChallengeTask{
			{ID: "t1", Status: TaskPending},
			{ID: "t2", Status: TaskPending},
		},
	}

	successes, failures, ties := cs.Tally()
	assert.Equal(t, 0, successes)
	assert.Equal(t, 0, failures)
	assert.Equal(t, 0, ties)
}

func TestChallengeState_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		tasks    []ChallengeTask
		expected bool
	}{
		{
			name: "all resolved",
			tasks: []ChallengeTask{
				{ID: "t1", Status: TaskSucceeded},
				{ID: "t2", Status: TaskFailed},
				{ID: "t3", Status: TaskTied},
			},
			expected: true,
		},
		{
			name: "one pending",
			tasks: []ChallengeTask{
				{ID: "t1", Status: TaskSucceeded},
				{ID: "t2", Status: TaskPending},
			},
			expected: false,
		},
		{
			name: "all pending",
			tasks: []ChallengeTask{
				{ID: "t1", Status: TaskPending},
				{ID: "t2", Status: TaskPending},
			},
			expected: false,
		},
		{
			name:     "empty tasks",
			tasks:    []ChallengeTask{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChallengeState{Tasks: tt.tasks}
			assert.Equal(t, tt.expected, cs.IsComplete())
		})
	}
}

func TestChallengeState_OverallOutcome(t *testing.T) {
	tests := []struct {
		name     string
		tasks    []ChallengeTask
		expected ChallengeResult
	}{
		{
			name: "all succeed → success",
			tasks: []ChallengeTask{
				{Status: TaskSucceeded},
				{Status: TaskSucceeded},
				{Status: TaskSucceeded},
			},
			expected: ChallengeSuccess,
		},
		{
			name: "all fail → failure",
			tasks: []ChallengeTask{
				{Status: TaskFailed},
				{Status: TaskFailed},
				{Status: TaskFailed},
			},
			expected: ChallengeFailure,
		},
		{
			name: "majority succeed (2 of 3) → success",
			tasks: []ChallengeTask{
				{Status: TaskSucceeded},
				{Status: TaskSucceeded},
				{Status: TaskFailed},
			},
			expected: ChallengeSuccess,
		},
		{
			name: "majority fail (2 of 3) → failure",
			tasks: []ChallengeTask{
				{Status: TaskFailed},
				{Status: TaskFailed},
				{Status: TaskSucceeded},
			},
			expected: ChallengeFailure,
		},
		{
			name: "even split (1 success 1 fail 1 tie) → partial",
			tasks: []ChallengeTask{
				{Status: TaskSucceeded},
				{Status: TaskFailed},
				{Status: TaskTied},
			},
			expected: ChallengePartial,
		},
		{
			name: "even count split (2 succeed 2 fail) → partial",
			tasks: []ChallengeTask{
				{Status: TaskSucceeded},
				{Status: TaskSucceeded},
				{Status: TaskFailed},
				{Status: TaskFailed},
			},
			expected: ChallengePartial,
		},
		{
			name: "even count majority succeed (3 of 4) → success",
			tasks: []ChallengeTask{
				{Status: TaskSucceeded},
				{Status: TaskSucceeded},
				{Status: TaskSucceeded},
				{Status: TaskFailed},
			},
			expected: ChallengeSuccess,
		},
		{
			name:     "empty tasks → success",
			tasks:    []ChallengeTask{},
			expected: ChallengeSuccess,
		},
		{
			name: "ties excluded from majority (2 succeed 1 tie 1 fail) → success",
			tasks: []ChallengeTask{
				{Status: TaskSucceeded},
				{Status: TaskSucceeded},
				{Status: TaskTied},
				{Status: TaskFailed},
			},
			expected: ChallengeSuccess,
		},
		{
			name: "ties excluded from majority (2 fail 1 tie 1 succeed) → failure",
			tasks: []ChallengeTask{
				{Status: TaskFailed},
				{Status: TaskFailed},
				{Status: TaskTied},
				{Status: TaskSucceeded},
			},
			expected: ChallengeFailure,
		},
		{
			name: "all ties → partial",
			tasks: []ChallengeTask{
				{Status: TaskTied},
				{Status: TaskTied},
				{Status: TaskTied},
			},
			expected: ChallengePartial,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ChallengeState{Tasks: tt.tasks}
			assert.Equal(t, tt.expected, cs.OverallOutcome())
		})
	}
}

func TestChallengeState_ResolveTask(t *testing.T) {
	cs := &ChallengeState{
		Tasks: []ChallengeTask{
			{ID: "t1", Skill: "Athletics", Difficulty: 3, Status: TaskPending},
			{ID: "t2", Skill: "Investigate", Difficulty: 2, Status: TaskPending},
		},
	}

	err := cs.ResolveTask("t1", TaskSucceeded, "player-1")
	require.NoError(t, err)

	assert.Equal(t, TaskSucceeded, cs.Tasks[0].Status)
	assert.Equal(t, "player-1", cs.Tasks[0].ActorID)

	// Second task still pending
	assert.Equal(t, TaskPending, cs.Tasks[1].Status)
}

func TestChallengeState_ResolveTask_NotFound(t *testing.T) {
	cs := &ChallengeState{
		Tasks: []ChallengeTask{
			{ID: "t1", Status: TaskPending},
		},
	}

	err := cs.ResolveTask("nonexistent", TaskSucceeded, "player-1")
	assert.ErrorContains(t, err, "not found")
}

func TestChallengeState_ResolveTask_AlreadyResolved(t *testing.T) {
	cs := &ChallengeState{
		Tasks: []ChallengeTask{
			{ID: "t1", Status: TaskSucceeded, ActorID: "player-1"},
		},
	}

	err := cs.ResolveTask("t1", TaskFailed, "player-2")
	assert.ErrorContains(t, err, "already resolved")
}

func TestChallengeState_FindTaskBySkill(t *testing.T) {
	cs := &ChallengeState{
		Tasks: []ChallengeTask{
			{ID: "t1", Skill: "Athletics", Status: TaskSucceeded},
			{ID: "t2", Skill: "Investigate", Status: TaskPending},
			{ID: "t3", Skill: "Rapport", Status: TaskPending},
		},
	}

	// Athletics already resolved — should not match
	task := cs.FindTaskBySkill("Athletics")
	assert.Nil(t, task)

	// Investigate is pending — should match
	task = cs.FindTaskBySkill("Investigate")
	require.NotNil(t, task)
	assert.Equal(t, "t2", task.ID)

	// Unknown skill returns nil
	task = cs.FindTaskBySkill("Fight")
	assert.Nil(t, task)
}

// --- Scene integration tests ---

func TestScene_StartChallenge(t *testing.T) {
	s := NewScene("test", "Test Scene", "Test")
	tasks := []ChallengeTask{
		{Description: "Dodge debris", Skill: "Athletics", Difficulty: 3},
		{Description: "Find exit", Skill: "Investigate", Difficulty: 2},
		{Description: "Calm crowd", Skill: "Rapport", Difficulty: 1},
	}

	err := s.StartChallenge("Escape the burning building", tasks)
	require.NoError(t, err)

	assert.True(t, s.IsChallenge)
	require.NotNil(t, s.ChallengeState)
	assert.Equal(t, "Escape the burning building", s.ChallengeState.Description)
	assert.Len(t, s.ChallengeState.Tasks, 3)
	assert.False(t, s.ChallengeState.Resolved)

	// Check IDs were auto-assigned
	assert.Equal(t, "task-1", s.ChallengeState.Tasks[0].ID)
	assert.Equal(t, "task-2", s.ChallengeState.Tasks[1].ID)
	assert.Equal(t, "task-3", s.ChallengeState.Tasks[2].ID)

	// Check all tasks start pending
	for _, task := range s.ChallengeState.Tasks {
		assert.Equal(t, TaskPending, task.Status)
	}
}

func TestScene_StartChallenge_PreservesExplicitIDs(t *testing.T) {
	s := NewScene("test", "Test Scene", "Test")
	tasks := []ChallengeTask{
		{ID: "custom-1", Skill: "Athletics", Difficulty: 3},
		{ID: "custom-2", Skill: "Investigate", Difficulty: 2},
	}

	err := s.StartChallenge("Test", tasks)
	require.NoError(t, err)

	assert.Equal(t, "custom-1", s.ChallengeState.Tasks[0].ID)
	assert.Equal(t, "custom-2", s.ChallengeState.Tasks[1].ID)
}

func TestScene_StartChallenge_MutualExclusion_DuringConflict(t *testing.T) {
	s := NewScene("test", "Test Scene", "Test")
	s.StartConflict(PhysicalConflict, []ConflictParticipant{
		{CharacterID: "player", Initiative: 3},
		{CharacterID: "npc", Initiative: 1},
	})

	err := s.StartChallenge("Test", []ChallengeTask{
		{Skill: "Athletics", Difficulty: 3},
	})
	assert.ErrorContains(t, err, "active conflict")
	assert.False(t, s.IsChallenge)
	assert.Nil(t, s.ChallengeState)
}

func TestScene_StartChallenge_MutualExclusion_DuringChallenge(t *testing.T) {
	s := NewScene("test", "Test Scene", "Test")
	err := s.StartChallenge("First", []ChallengeTask{
		{Skill: "Athletics", Difficulty: 3},
	})
	require.NoError(t, err)

	err = s.StartChallenge("Second", []ChallengeTask{
		{Skill: "Investigate", Difficulty: 2},
	})
	assert.ErrorContains(t, err, "active challenge")
}

func TestScene_EndChallenge(t *testing.T) {
	s := NewScene("test", "Test Scene", "Test")
	err := s.StartChallenge("Test", []ChallengeTask{
		{Skill: "Athletics", Difficulty: 3},
	})
	require.NoError(t, err)
	require.True(t, s.IsChallenge)

	s.EndChallenge()

	assert.False(t, s.IsChallenge)
	assert.Nil(t, s.ChallengeState)
	assert.Equal(t, SceneTypeNone, s.ActiveSceneType())
}

func TestScene_ActiveSceneType_Challenge(t *testing.T) {
	s := NewScene("test", "Test Scene", "Test")
	err := s.StartChallenge("Test", []ChallengeTask{
		{Skill: "Athletics", Difficulty: 3},
	})
	require.NoError(t, err)

	assert.Equal(t, SceneTypeChallenge, s.ActiveSceneType())
}

func TestScene_ActiveSceneType_AfterChallengeEnds(t *testing.T) {
	s := NewScene("test", "Test Scene", "Test")
	err := s.StartChallenge("Test", []ChallengeTask{
		{Skill: "Athletics", Difficulty: 3},
	})
	require.NoError(t, err)
	require.Equal(t, SceneTypeChallenge, s.ActiveSceneType())

	s.EndChallenge()
	assert.Equal(t, SceneTypeNone, s.ActiveSceneType())
}

func TestScene_ChallengeAfterConflictEnds(t *testing.T) {
	s := NewScene("test", "Test Scene", "Test")

	// Start and end a conflict
	s.StartConflict(PhysicalConflict, []ConflictParticipant{
		{CharacterID: "player", Initiative: 3},
	})
	s.EndConflict()

	// Now a challenge should be allowed
	err := s.StartChallenge("Aftermath", []ChallengeTask{
		{Skill: "Athletics", Difficulty: 3},
	})
	assert.NoError(t, err)
	assert.Equal(t, SceneTypeChallenge, s.ActiveSceneType())
}
