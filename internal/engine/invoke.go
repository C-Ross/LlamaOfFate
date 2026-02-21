package engine

import (
	"context"
	"fmt"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// invokeState holds the context for an in-progress invoke loop that has been
// suspended while waiting for the player's InvokeResponse.
type invokeState struct {
	result       *dice.CheckResult // Current roll result (mutated by invokes)
	difficulty   dice.Ladder       // Target difficulty / defense value
	parsedAction *action.Action    // Action being resolved (tracks invokes)
	isDefense    bool              // True when player is invoking on a defense roll
	usedAspects  map[string]bool   // Aspects already invoked this roll
	available    []InvokableAspect // Aspects offered in the current prompt
	continuation invokeFinishFunc  // Called when the invoke loop completes
	resumeTurns  bool              // True when NPC turns should resume after invoke resolves
}

// invokeFinishFunc is called when the invoke loop completes (player skips or
// no more invokes available). It receives the final CheckResult and the events
// accumulated so far, and returns the full set of events for the InputResult.
type invokeFinishFunc func(ctx context.Context, result *dice.CheckResult, events []GameEvent) []GameEvent

// beginInvokeLoop checks whether a post-roll invoke prompt is needed and, if so,
// returns the events including an InvokePromptEvent and populates cm.pendingInvoke.
// Returns (events, awaitingInvoke).
// If no invoke is possible the continuation is called immediately and
// (events, false) is returned.
func (cm *ConflictManager) beginInvokeLoop(
	ctx context.Context,
	result *dice.CheckResult,
	difficulty dice.Ladder,
	parsedAction *action.Action,
	isDefense bool,
	preEvents []GameEvent,
	finish invokeFinishFunc,
) ([]GameEvent, bool) {
	usedAspects := make(map[string]bool)

	promptEvent, available := cm.buildInvokePrompt(result, difficulty, isDefense, usedAspects)
	if promptEvent == nil {
		// No invoke possible — finish immediately.
		events := finish(ctx, result, preEvents)
		// The continuation may have started its own invoke loop
		// (e.g. NPC defense invoke inside advanceConflictTurns).
		return events, cm.pendingInvoke != nil
	}

	cm.pendingInvoke = &invokeState{
		result:       result,
		difficulty:   difficulty,
		parsedAction: parsedAction,
		isDefense:    isDefense,
		usedAspects:  usedAspects,
		available:    available,
		continuation: finish,
	}

	events := append(preEvents, *promptEvent)
	return events, true
}

// ProvideInvokeResponse processes the player's invoke decision and either
// returns another InvokePromptEvent (more invokes available) or finalises the
// action and returns all remaining events.
func (cm *ConflictManager) ProvideInvokeResponse(ctx context.Context, resp InvokeResponse) (*InputResult, error) {
	if cm.pendingInvoke == nil {
		return nil, fmt.Errorf("ProvideInvokeResponse called with no pending invoke")
	}

	is := cm.pendingInvoke
	var events []GameEvent

	if resp.AspectIndex == uicontract.InvokeSkip {
		// Player chose to skip — finalise.
		// Pass empty events since preEvents were already rendered by the caller.
		cm.pendingInvoke = nil
		events = is.continuation(ctx, is.result, nil)
		// If this invoke was for a defense roll during NPC turns, resume turns.
		events = cm.maybeResumeConflictTurns(ctx, is, events)
		return cm.wrapInvokeResult(events), nil
	}

	// Validate index.
	if resp.AspectIndex < 0 || resp.AspectIndex >= len(is.available) {
		return nil, fmt.Errorf("ProvideInvokeResponse: aspect index %d out of range [0, %d)",
			resp.AspectIndex, len(is.available))
	}

	selected := &is.available[resp.AspectIndex]

	// Determine free vs paid.
	useFree := selected.FreeInvokes > 0

	// Spend fate point or use free invoke.
	invokeEvents := cm.applyInvokeChoice(is, selected, useFree, resp.IsReroll)

	// Mark used.
	is.usedAspects[selected.Name] = true

	// Track invoke on the action.
	is.parsedAction.AddAspectInvoke(action.AspectInvoke{
		AspectText: selected.Name,
		Source:     selected.Source,
		SourceID:   selected.SourceID,
		IsFree:     useFree,
		FatePointCost: func() int {
			if useFree {
				return 0
			}
			return 1
		}(),
		Bonus: func() int {
			if resp.IsReroll {
				return 0
			}
			return 2
		}(),
		IsReroll: resp.IsReroll,
	})

	// Check if another invoke is possible.
	promptEvent, available := cm.buildInvokePrompt(is.result, is.difficulty, is.isDefense, is.usedAspects)
	if promptEvent == nil {
		// No more invokes — finalise.
		// Pass only the invoke events from this round (prior events already rendered).
		cm.pendingInvoke = nil
		events = is.continuation(ctx, is.result, invokeEvents)
		// If this invoke was for a defense roll during NPC turns, resume turns.
		events = cm.maybeResumeConflictTurns(ctx, is, events)
		return cm.wrapInvokeResult(events), nil
	}

	// Another invoke prompt — return only new events from this round.
	is.available = available
	newEvents := append(invokeEvents, *promptEvent)
	return &InputResult{
		Events:         newEvents,
		AwaitingInvoke: true,
	}, nil
}

// wrapInvokeResult packages the final events after invoke completion, detecting
// whether the scene ended or a nested invoke loop was started by the
// continuation (e.g. NPC defense invoke inside advanceConflictTurns).
func (cm *ConflictManager) wrapInvokeResult(events []GameEvent) *InputResult {
	result := &InputResult{Events: events}
	if cm.pendingInvoke != nil {
		result.AwaitingInvoke = true
	}
	// Scene-end wrapping is handled by SceneManager.applySceneEnd via the
	// public delegator, not here.
	return result
}

// maybeResumeConflictTurns resumes NPC turn processing after a defense invoke
// resolves. Only called when is.resumeTurns is true and no nested invoke was
// started by the continuation.
func (cm *ConflictManager) maybeResumeConflictTurns(ctx context.Context, is *invokeState, events []GameEvent) []GameEvent {
	if !is.resumeTurns {
		return events
	}
	if cm.pendingInvoke != nil {
		// A nested invoke was started (shouldn't happen for the continuation,
		// but guard against it). Turn resumption will happen when that resolves.
		return events
	}
	if !cm.currentScene.IsConflict {
		return events
	}
	turnEvents, _ := cm.advanceConflictTurns(ctx)
	return append(events, turnEvents...)
}

// buildInvokePrompt determines whether the player can invoke an aspect and
// returns the prompt event and available aspects. Returns (nil, nil) when
// no invoke is possible.
func (cm *ConflictManager) buildInvokePrompt(
	result *dice.CheckResult,
	difficulty dice.Ladder,
	isDefense bool,
	usedAspects map[string]bool,
) (*InvokePromptEvent, []InvokableAspect) {
	outcome := result.CompareAgainst(difficulty)
	shiftsNeeded := 0

	if isDefense {
		if outcome.Shifts >= 0 {
			return nil, nil
		}
		shiftsNeeded = -outcome.Shifts
	} else {
		if outcome.Type == dice.SuccessWithStyle {
			return nil, nil
		}
		if outcome.Shifts < 0 {
			shiftsNeeded = -outcome.Shifts
		} else if outcome.Shifts < 3 {
			shiftsNeeded = 3 - outcome.Shifts
		}
	}

	available := cm.gatherInvokableAspects(usedAspects)

	canInvoke := false
	for _, aspect := range available {
		if aspect.AlreadyUsed {
			continue
		}
		if aspect.FreeInvokes > 0 || cm.player.FatePoints > 0 {
			canInvoke = true
			break
		}
	}
	if !canInvoke {
		return nil, nil
	}

	return &InvokePromptEvent{
		Available:     available,
		FatePoints:    cm.player.FatePoints,
		CurrentResult: result.FinalValue.String(),
		ShiftsNeeded:  shiftsNeeded,
	}, available
}

// applyInvokeChoice spends the resource and applies the +2 or reroll effect.
// Returns InvokeEvents describing what happened.
func (cm *ConflictManager) applyInvokeChoice(is *invokeState, selected *InvokableAspect, useFree bool, isReroll bool) []GameEvent {
	var events []GameEvent

	if useFree {
		for i := range cm.currentScene.SituationAspects {
			if cm.currentScene.SituationAspects[i].Aspect == selected.Name {
				cm.currentScene.SituationAspects[i].UseFreeInvoke()
				break
			}
		}
	} else {
		if !cm.player.SpendFatePoint() {
			events = append(events, InvokeEvent{
				AspectName: selected.Name,
				Failed:     true,
			})
			return events
		}
	}

	var newRoll, newTotal string
	if isReroll {
		is.result = cm.roller.Reroll(is.result)
		newRoll = is.result.Roll.String()
		newTotal = is.result.FinalValue.String()
	} else {
		is.result.ApplyInvokeBonus(2)
		newTotal = is.result.FinalValue.String()
	}

	events = append(events, InvokeEvent{
		AspectName:     selected.Name,
		IsFree:         useFree,
		IsReroll:       isReroll,
		FatePointsLeft: cm.player.FatePoints,
		NewRoll:        newRoll,
		NewTotal:       newTotal,
	})

	// Log the invoke
	if cm.sessionLogger != nil {
		cm.sessionLogger.Log("invoke", map[string]any{
			"aspect":    selected.Name,
			"source":    selected.Source,
			"is_free":   useFree,
			"is_reroll": isReroll,
			"new_total": is.result.FinalValue.String(),
		})
	}

	return events
}

// HasPendingInvoke returns true when the engine is waiting for an InvokeResponse.
func (cm *ConflictManager) HasPendingInvoke() bool {
	return cm.pendingInvoke != nil
}
