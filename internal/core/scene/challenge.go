package scene

import "fmt"

// TaskStatus tracks the resolution state of a single challenge task.
type TaskStatus string

const (
	TaskPending            TaskStatus = "pending"
	TaskSucceeded          TaskStatus = "succeeded"
	TaskSucceededWithStyle TaskStatus = "succeeded_with_style"
	TaskFailed             TaskStatus = "failed"
	TaskTied               TaskStatus = "tied"
)

// ChallengeResult describes the overall outcome of a completed challenge.
type ChallengeResult string

const (
	ChallengeSuccess ChallengeResult = "success" // Majority of tasks succeeded
	ChallengePartial ChallengeResult = "partial" // Mixed results
	ChallengeFailure ChallengeResult = "failure" // Majority of tasks failed
)

// ChallengeTask represents one overcome action within a challenge.
type ChallengeTask struct {
	ID          string     `json:"id" yaml:"id"`
	Description string     `json:"description" yaml:"description"`
	Skill       string     `json:"skill" yaml:"skill"`
	Difficulty  int        `json:"difficulty" yaml:"difficulty"`                 // Ladder value
	Status      TaskStatus `json:"status" yaml:"status"`                         // pending, succeeded, succeeded_with_style, failed, tied
	ActorID     string     `json:"actor_id,omitempty" yaml:"actor_id,omitempty"` // Who attempted it
}

// ChallengeState tracks a multi-task challenge within a scene.
// A challenge is a series of overcome actions using different skills to
// resolve a complex, multi-part situation. See:
// https://fate-srd.com/fate-core/challenges
type ChallengeState struct {
	Description string          `json:"description" yaml:"description"`
	Tasks       []ChallengeTask `json:"tasks" yaml:"tasks"`
	Resolved    bool            `json:"resolved" yaml:"resolved"`
}

// PendingTasks returns tasks not yet attempted.
func (cs *ChallengeState) PendingTasks() []ChallengeTask {
	var pending []ChallengeTask
	for _, t := range cs.Tasks {
		if t.Status == TaskPending {
			pending = append(pending, t)
		}
	}
	return pending
}

// Tally returns (successes, failures, ties) across all tasks.
func (cs *ChallengeState) Tally() (int, int, int) {
	var successes, failures, ties int
	for _, t := range cs.Tasks {
		switch t.Status {
		case TaskSucceeded, TaskSucceededWithStyle:
			successes++
		case TaskFailed:
			failures++
		case TaskTied:
			ties++
		}
	}
	return successes, failures, ties
}

// IsComplete returns true when all tasks have been attempted
// (no pending tasks remain).
func (cs *ChallengeState) IsComplete() bool {
	for _, t := range cs.Tasks {
		if t.Status == TaskPending {
			return false
		}
	}
	return true
}

// OverallOutcome returns an overall result based on the tally:
//   - ChallengeSuccess if more than half the decisive tasks succeeded
//   - ChallengeFailure if more than half the decisive tasks failed
//   - ChallengePartial otherwise (mixed results)
//
// Ties are excluded from the majority threshold because in Fate Core a tie
// means "you achieve your goal but at a minor cost" — it is neither a clear
// success nor a failure, so it should not inflate the denominator.
func (cs *ChallengeState) OverallOutcome() ChallengeResult {
	successes, failures, ties := cs.Tally()
	total := len(cs.Tasks)
	if total == 0 {
		return ChallengeSuccess
	}

	decisive := total - ties
	half := decisive / 2
	if successes > half {
		return ChallengeSuccess
	}
	if failures > half {
		return ChallengeFailure
	}
	return ChallengePartial
}

// ResolveTask updates a task's outcome. Returns an error if the task ID
// is not found or the task has already been resolved.
func (cs *ChallengeState) ResolveTask(taskID string, status TaskStatus, actorID string) error {
	for i := range cs.Tasks {
		if cs.Tasks[i].ID == taskID {
			if cs.Tasks[i].Status != TaskPending {
				return fmt.Errorf("task %q already resolved with status %q", taskID, cs.Tasks[i].Status)
			}
			cs.Tasks[i].Status = status
			cs.Tasks[i].ActorID = actorID
			return nil
		}
	}
	return fmt.Errorf("task %q not found in challenge", taskID)
}

// FindTaskBySkill returns the first pending task that matches the given
// skill (case-sensitive). Returns nil if no pending task uses that skill.
func (cs *ChallengeState) FindTaskBySkill(skill string) *ChallengeTask {
	for i := range cs.Tasks {
		if cs.Tasks[i].Status == TaskPending && cs.Tasks[i].Skill == skill {
			return &cs.Tasks[i]
		}
	}
	return nil
}
