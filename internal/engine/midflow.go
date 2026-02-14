package engine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// midFlowState holds the suspended state while the engine waits for
// a MidFlowResponse from the UI. This is the mid-flow equivalent of
// invokeState — each blocking ReadInput call in the old code becomes
// an InputRequestEvent + a continuation stored here.
type midFlowState struct {
	event        InputRequestEvent   // The event sent to the UI
	continuation midFlowContinuation // Called when the response arrives
}

// midFlowContinuation is called with the player's response and the parent
// context. It returns any additional events to emit and may set shouldExit /
// sceneEndReason on the SceneManager.
type midFlowContinuation func(ctx context.Context, resp MidFlowResponse) []GameEvent

// ProvideMidFlowResponse processes the player's response to an InputRequestEvent
// and returns the resulting events. This is the mid-flow equivalent of
// ProvideInvokeResponse.
func (sm *SceneManager) ProvideMidFlowResponse(ctx context.Context, resp MidFlowResponse) (*InputResult, error) {
	if sm.pendingMidFlow == nil {
		return nil, fmt.Errorf("ProvideMidFlowResponse called with no pending mid-flow request")
	}

	mf := sm.pendingMidFlow
	sm.pendingMidFlow = nil

	// Validate numbered choice index.
	if mf.event.Type == uicontract.InputRequestNumberedChoice {
		if resp.ChoiceIndex < 0 || resp.ChoiceIndex >= len(mf.event.Options) {
			slog.Warn("Invalid mid-flow choice index, defaulting to last option",
				"component", componentSceneManager,
				"index", resp.ChoiceIndex,
				"numOptions", len(mf.event.Options))
			resp.ChoiceIndex = len(mf.event.Options) - 1
		}
	}

	events := mf.continuation(ctx, resp)

	result := &InputResult{Events: events}

	// Check for nested mid-flow (e.g. recursive stress overflow).
	if sm.pendingMidFlow != nil {
		result.AwaitingMidFlow = true
	}

	if sm.shouldExit && !result.AwaitingMidFlow {
		result.SceneEnded = true
		result.EndResult = sm.buildSceneEndResult()
	}

	return result, nil
}
