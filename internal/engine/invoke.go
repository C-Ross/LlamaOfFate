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
// returns the events including an InvokePromptEvent and populates ar.pendingInvoke.
// Returns (events, awaitingInvoke).
// If no invoke is possible the continuation is called immediately and
// (events, false) is returned.
func (ar *ActionResolver) beginInvokeLoop(
	ctx context.Context,
	result *dice.CheckResult,
	difficulty dice.Ladder,
	parsedAction *action.Action,
	isDefense bool,
	preEvents []GameEvent,
	finish invokeFinishFunc,
) ([]GameEvent, bool) {
	usedAspects := make(map[string]bool)

	promptEvent, available := ar.buildInvokePrompt(result, difficulty, isDefense, usedAspects)
	if promptEvent == nil {
		// No invoke possible — finish immediately.
		events := finish(ctx, result, preEvents)
		// The continuation may have started its own invoke loop
		// (e.g. NPC defense invoke inside advanceConflictTurns).
		return events, ar.pendingInvoke != nil
	}

	ar.pendingInvoke = &invokeState{
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
func (ar *ActionResolver) ProvideInvokeResponse(ctx context.Context, resp InvokeResponse) (*InputResult, error) {
	if ar.pendingInvoke == nil {
		return nil, fmt.Errorf("ProvideInvokeResponse called with no pending invoke")
	}

	is := ar.pendingInvoke
	var events []GameEvent

	if resp.AspectIndex == uicontract.InvokeSkip {
		// Player chose to skip — finalise.
		// Pass empty events since preEvents were already rendered by the caller.
		ar.pendingInvoke = nil
		events = is.continuation(ctx, is.result, nil)
		// If this invoke was for a defense roll during NPC turns, resume turns.
		events = ar.maybeResumeConflictTurns(ctx, is, events)
		return ar.wrapInvokeResult(events), nil
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
	invokeEvents := ar.applyInvokeChoice(is, selected, useFree, resp.IsReroll)

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
	promptEvent, available := ar.buildInvokePrompt(is.result, is.difficulty, is.isDefense, is.usedAspects)
	if promptEvent == nil {
		// No more invokes — finalise.
		// Pass only the invoke events from this round (prior events already rendered).
		ar.pendingInvoke = nil
		events = is.continuation(ctx, is.result, invokeEvents)
		// If this invoke was for a defense roll during NPC turns, resume turns.
		events = ar.maybeResumeConflictTurns(ctx, is, events)
		return ar.wrapInvokeResult(events), nil
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
func (ar *ActionResolver) wrapInvokeResult(events []GameEvent) *InputResult {
	result := &InputResult{Events: events}
	if ar.pendingInvoke != nil {
		result.AwaitingInvoke = true
	}
	// Scene-end wrapping is handled by SceneManager.applySceneEnd via the
	// public delegator, not here.
	return result
}

// maybeResumeConflictTurns resumes NPC turn processing after a defense invoke
// resolves. Only called when is.resumeTurns is true and no nested invoke was
// started by the continuation.
func (ar *ActionResolver) maybeResumeConflictTurns(ctx context.Context, is *invokeState, events []GameEvent) []GameEvent {
	if !is.resumeTurns {
		return events
	}
	if ar.pendingInvoke != nil {
		// A nested invoke was started (shouldn't happen for the continuation,
		// but guard against it). Turn resumption will happen when that resolves.
		return events
	}
	if !ar.currentScene.IsConflict {
		return events
	}
	turnEvents, _ := ar.conflict.advanceConflictTurns(ctx)
	return append(events, turnEvents...)
}

// buildInvokePrompt determines whether the player can invoke an aspect and
// returns the prompt event and available aspects. Returns (nil, nil) when
// no invoke is possible.
func (ar *ActionResolver) buildInvokePrompt(
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

	available := ar.gatherInvokableAspects(usedAspects)

	canInvoke := false
	for _, aspect := range available {
		if aspect.AlreadyUsed {
			continue
		}
		if aspect.FreeInvokes > 0 || ar.player.FatePoints > 0 {
			canInvoke = true
			break
		}
	}
	if !canInvoke {
		return nil, nil
	}

	return &InvokePromptEvent{
		Available:     available,
		FatePoints:    ar.player.FatePoints,
		CurrentResult: result.FinalValue.String(),
		ShiftsNeeded:  shiftsNeeded,
	}, available
}

// applyInvokeChoice spends the resource and applies the +2 or reroll effect.
// Returns InvokeEvents describing what happened.
func (ar *ActionResolver) applyInvokeChoice(is *invokeState, selected *InvokableAspect, useFree bool, isReroll bool) []GameEvent {
	var events []GameEvent

	if useFree {
		for i := range ar.currentScene.SituationAspects {
			if ar.currentScene.SituationAspects[i].Aspect == selected.Name {
				ar.currentScene.SituationAspects[i].UseFreeInvoke()
				break
			}
		}
	} else {
		if !ar.player.SpendFatePoint() {
			events = append(events, InvokeEvent{
				AspectName: selected.Name,
				Failed:     true,
			})
			return events
		}
	}

	var newRoll, newTotal string
	if isReroll {
		is.result = ar.roller.Reroll(is.result)
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
		FatePointsLeft: ar.player.FatePoints,
		NewRoll:        newRoll,
		NewTotal:       newTotal,
	})

	// Log the invoke
	if ar.sessionLogger != nil {
		ar.sessionLogger.Log("invoke", map[string]any{
			"aspect":    selected.Name,
			"source":    selected.Source,
			"is_free":   useFree,
			"is_reroll": isReroll,
			"new_total": is.result.FinalValue.String(),
		})
	}

	return events
}
