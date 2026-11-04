package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
)

// initiateChallenge starts a challenge in the current scene by calling the
// ChallengeGenerator to produce tasks from a description, then wiring the
// resulting ChallengeState onto the Scene. Returns a ChallengeStartEvent.
func (chm *ChallengeManager) initiateChallenge(ctx context.Context, description string) ([]GameEvent, error) {
	if chm.challengeGenerator == nil {
		return nil, fmt.Errorf("challenge generator not available")
	}

	// Build the request for the generator
	playerSkills := chm.playerSkillNames()
	guidance := prompt.ComputeDifficultyGuidance(chm.player.Skills)

	req := prompt.ChallengeBuildData{
		Description:        description,
		SceneName:          chm.currentScene.Name,
		SceneDescription:   chm.currentScene.Description,
		PlayerSkills:       playerSkills,
		SituationAspects:   chm.currentScene.SituationAspects,
		DifficultyGuidance: guidance,
	}

	state, err := chm.challengeGenerator.BuildChallenge(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to build challenge: %w", err)
	}

	// Wire the state onto the scene
	if err := chm.currentScene.StartChallenge(state.Description, state.Tasks); err != nil {
		return nil, fmt.Errorf("failed to start challenge on scene: %w", err)
	}

	slog.Info("Challenge initiated",
		"component", componentSceneManager,
		"description", description,
		"tasks", len(state.Tasks))

	// Session logging
	chm.sessionLogger.Log("challenge_start", map[string]any{
		"description": state.Description,
		"tasks":       state.Tasks,
	})

	taskInfos := buildChallengeTaskInfos(chm.currentScene.ChallengeState)
	return []GameEvent{ChallengeStartEvent{
		Description: state.Description,
		Tasks:       taskInfos,
	}}, nil
}

// resolveTask records the outcome of a single challenge task. It is called by
// ActionResolver after an overcome roll completes. Returns a
// ChallengeTaskResultEvent and, if all tasks are done, a
// ChallengeCompleteEvent.
func (chm *ChallengeManager) resolveTask(taskID string, outcome dice.OutcomeType, shifts int) []GameEvent {
	cs := chm.currentScene.ChallengeState
	if cs == nil {
		return nil
	}

	// Map dice outcome to task status
	status := scene.TaskStatusForOutcome(outcome)

	if err := cs.ResolveTask(taskID, status, chm.player.ID); err != nil {
		slog.Warn("Failed to resolve challenge task",
			"component", componentSceneManager,
			"task_id", taskID,
			"error", err)
		return nil
	}

	// Find the task we just resolved for event data
	var task *scene.ChallengeTask
	for i := range cs.Tasks {
		if cs.Tasks[i].ID == taskID {
			task = &cs.Tasks[i]
			break
		}
	}

	slog.Info("Challenge task resolved",
		"component", componentSceneManager,
		"task_id", taskID,
		"status", status,
		"shifts", shifts)

	chm.sessionLogger.Log("challenge_task_result", map[string]any{
		"task_id": taskID,
		"skill":   task.Skill,
		"outcome": string(status),
		"shifts":  shifts,
	})

	events := []GameEvent{ChallengeTaskResultEvent{
		TaskID:      taskID,
		Description: task.Description,
		Skill:       task.Skill,
		Outcome:     string(status),
		Shifts:      shifts,
	}}

	// Check if challenge is complete
	if cs.IsComplete() {
		events = append(events, chm.completeChallenge()...)
	}

	return events
}

// completeChallenge tallies results and ends the challenge. Returns a
// ChallengeCompleteEvent.
func (chm *ChallengeManager) completeChallenge() []GameEvent {
	cs := chm.currentScene.ChallengeState
	if cs == nil {
		return nil
	}

	successes, failures, ties := cs.Tally()
	overall := cs.OverallOutcome()

	slog.Info("Challenge complete",
		"component", componentSceneManager,
		"successes", successes,
		"failures", failures,
		"ties", ties,
		"overall", overall)

	chm.sessionLogger.Log("challenge_complete", map[string]any{
		"successes": successes,
		"failures":  failures,
		"ties":      ties,
		"overall":   string(overall),
	})

	chm.currentScene.EndChallenge()

	return []GameEvent{ChallengeCompleteEvent{
		Successes: successes,
		Failures:  failures,
		Ties:      ties,
		Overall:   string(overall),
	}}
}

// findTaskForSkill returns the best pending task for a skill. If no exact
// match, returns nil (the caller should handle disambiguation).
func (chm *ChallengeManager) findTaskForSkill(skill string) *scene.ChallengeTask {
	if chm.currentScene.ChallengeState == nil {
		return nil
	}
	return chm.currentScene.ChallengeState.FindTaskBySkill(skill)
}

// playerSkillNames returns sorted skill names for the player character.
func (chm *ChallengeManager) playerSkillNames() []string {
	names := make([]string, 0, len(chm.player.Skills))
	for name := range chm.player.Skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
