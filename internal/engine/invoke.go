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
}

// invokeFinishFunc is called when the invoke loop completes (player skips or
// no more invokes available). It receives the final CheckResult and the events
// accumulated so far, and returns the full set of events for the InputResult.
type invokeFinishFunc func(ctx context.Context, result *dice.CheckResult, events []GameEvent) []GameEvent

// beginInvokeLoop checks whether a post-roll invoke prompt is needed and, if so,
// returns the events including an InvokePromptEvent and populates sm.pendingInvoke.
// Returns (events, awaitingInvoke).
// If no invoke is possible the continuation is called immediately and
// (events, false) is returned.
func (sm *SceneManager) beginInvokeLoop(
	ctx context.Context,
	result *dice.CheckResult,
	difficulty dice.Ladder,
	parsedAction *action.Action,
	isDefense bool,
	preEvents []GameEvent,
	finish invokeFinishFunc,
) ([]GameEvent, bool) {
	usedAspects := make(map[string]bool)

	promptEvent, available := sm.buildInvokePrompt(result, difficulty, isDefense, usedAspects)
	if promptEvent == nil {
		// No invoke possible — finish immediately.
		events := finish(ctx, result, preEvents)
		return events, false
	}

	sm.pendingInvoke = &invokeState{
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
func (sm *SceneManager) ProvideInvokeResponse(ctx context.Context, resp InvokeResponse) (*InputResult, error) {
	if sm.pendingInvoke == nil {
		return nil, fmt.Errorf("ProvideInvokeResponse called with no pending invoke")
	}

	is := sm.pendingInvoke
	var events []GameEvent

	if resp.AspectIndex == uicontract.InvokeSkip {
		// Player chose to skip — finalise.
		// Pass empty events since preEvents were already rendered by the caller.
		sm.pendingInvoke = nil
		events = is.continuation(ctx, is.result, nil)
		return sm.wrapInvokeResult(events), nil
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
	invokeEvents := sm.applyInvokeChoice(is, selected, useFree, resp.IsReroll)

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
	promptEvent, available := sm.buildInvokePrompt(is.result, is.difficulty, is.isDefense, is.usedAspects)
	if promptEvent == nil {
		// No more invokes — finalise.
		// Pass only the invoke events from this round (prior events already rendered).
		sm.pendingInvoke = nil
		events = is.continuation(ctx, is.result, invokeEvents)
		return sm.wrapInvokeResult(events), nil
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
// whether the scene ended.
func (sm *SceneManager) wrapInvokeResult(events []GameEvent) *InputResult {
	result := &InputResult{Events: events}
	if sm.shouldExit {
		result.SceneEnded = true
		result.EndResult = sm.buildSceneEndResult()
	}
	return result
}

// buildInvokePrompt determines whether the player can invoke an aspect and
// returns the prompt event and available aspects. Returns (nil, nil) when
// no invoke is possible.
func (sm *SceneManager) buildInvokePrompt(
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

	available := sm.gatherInvokableAspects(usedAspects)

	canInvoke := false
	for _, aspect := range available {
		if aspect.AlreadyUsed {
			continue
		}
		if aspect.FreeInvokes > 0 || sm.player.FatePoints > 0 {
			canInvoke = true
			break
		}
	}
	if !canInvoke {
		return nil, nil
	}

	return &InvokePromptEvent{
		Available:     available,
		FatePoints:    sm.player.FatePoints,
		CurrentResult: result.FinalValue.String(),
		ShiftsNeeded:  shiftsNeeded,
	}, available
}

// applyInvokeChoice spends the resource and applies the +2 or reroll effect.
// Returns InvokeEvents describing what happened.
func (sm *SceneManager) applyInvokeChoice(is *invokeState, selected *InvokableAspect, useFree bool, isReroll bool) []GameEvent {
	var events []GameEvent

	if useFree {
		for i := range sm.currentScene.SituationAspects {
			if sm.currentScene.SituationAspects[i].Aspect == selected.Name {
				sm.currentScene.SituationAspects[i].UseFreeInvoke()
				break
			}
		}
	} else {
		if !sm.player.SpendFatePoint() {
			events = append(events, InvokeEvent{
				AspectName: selected.Name,
				Failed:     true,
			})
			return events
		}
	}

	var newRoll, newTotal string
	if isReroll {
		is.result = sm.roller.Reroll(is.result)
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
		FatePointsLeft: sm.player.FatePoints,
		NewRoll:        newRoll,
		NewTotal:       newTotal,
	})

	// Log the invoke
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("invoke", map[string]any{
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
func (sm *SceneManager) HasPendingInvoke() bool {
	return sm.pendingInvoke != nil
}

// legacyHandlePostRollInvokes is the original blocking invoke loop, used
// only by the synchronous terminal path until callers are fully migrated.
func (sm *SceneManager) legacyHandlePostRollInvokes(result *dice.CheckResult, difficulty dice.Ladder, parsedAction *action.Action, isDefense bool) *dice.CheckResult {
	usedAspects := make(map[string]bool)

	for {
		// Calculate outcome and shifts needed
		outcome := result.CompareAgainst(difficulty)
		shiftsNeeded := 0

		if isDefense {
			if outcome.Shifts >= 0 {
				break
			}
			shiftsNeeded = -outcome.Shifts
		} else {
			if outcome.Type == dice.SuccessWithStyle {
				break
			}
			if outcome.Shifts < 0 {
				shiftsNeeded = -outcome.Shifts
			} else if outcome.Shifts < 3 {
				shiftsNeeded = 3 - outcome.Shifts
			}
		}

		available := sm.gatherInvokableAspects(usedAspects)

		canInvoke := false
		for _, aspect := range available {
			if aspect.AlreadyUsed {
				continue
			}
			if aspect.FreeInvokes > 0 || sm.player.FatePoints > 0 {
				canInvoke = true
				break
			}
		}
		if !canInvoke {
			break
		}

		choice := sm.ui.PromptForInvoke(available, sm.player.FatePoints, result.FinalValue.String(), shiftsNeeded)

		if choice.Aspect == nil {
			break
		}

		if choice.UseFree {
			for i := range sm.currentScene.SituationAspects {
				if sm.currentScene.SituationAspects[i].Aspect == choice.Aspect.Name {
					sm.currentScene.SituationAspects[i].UseFreeInvoke()
					break
				}
			}
		} else {
			if !sm.player.SpendFatePoint() {
				sm.renderEvents([]GameEvent{InvokeEvent{AspectName: choice.Aspect.Name, Failed: true}})
				continue
			}
		}

		usedAspects[choice.Aspect.Name] = true

		var newRoll, newTotal string
		if choice.IsReroll {
			result = sm.roller.Reroll(result)
			newRoll = result.Roll.String()
			newTotal = result.FinalValue.String()
		} else {
			result.ApplyInvokeBonus(2)
			newTotal = result.FinalValue.String()
		}

		sm.renderEvents([]GameEvent{InvokeEvent{
			AspectName:     choice.Aspect.Name,
			IsFree:         choice.UseFree,
			IsReroll:       choice.IsReroll,
			FatePointsLeft: sm.player.FatePoints,
			NewRoll:        newRoll,
			NewTotal:       newTotal,
		}})

		parsedAction.AddAspectInvoke(action.AspectInvoke{
			AspectText: choice.Aspect.Name,
			Source:     choice.Aspect.Source,
			SourceID:   choice.Aspect.SourceID,
			IsFree:     choice.UseFree,
			FatePointCost: func() int {
				if choice.UseFree {
					return 0
				}
				return 1
			}(),
			Bonus: func() int {
				if choice.IsReroll {
					return 0
				}
				return 2
			}(),
			IsReroll: choice.IsReroll,
		})
	}

	return result
}
