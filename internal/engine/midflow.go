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

// resolveMidFlowBlocking bridges the event-driven mid-flow system with the
// blocking terminal UI. It reads the pending InputRequestEvent, displays the
// prompt, collects input via ReadInput, and feeds it back via
// ProvideMidFlowResponse.
func (sm *SceneManager) resolveMidFlowBlocking(ctx context.Context) (*InputResult, error) {
	if sm.pendingMidFlow == nil {
		return &InputResult{}, nil
	}

	event := sm.pendingMidFlow.event

	switch event.Type {
	case uicontract.InputRequestNumberedChoice:
		return sm.resolveMidFlowNumberedChoice(ctx, event)
	case uicontract.InputRequestFreeText:
		return sm.resolveMidFlowFreeText(ctx, event)
	default:
		return nil, fmt.Errorf("resolveMidFlowBlocking: unknown request type %q", event.Type)
	}
}

// resolveMidFlowNumberedChoice handles a numbered-choice prompt for the terminal UI.
func (sm *SceneManager) resolveMidFlowNumberedChoice(ctx context.Context, event InputRequestEvent) (*InputResult, error) {
	// Display the prompt and options (these mirror the old DisplaySystemMessage calls).
	var promptEvents []GameEvent
	promptEvents = append(promptEvents, SystemMessageEvent{Message: event.Prompt})
	for i, opt := range event.Options {
		if opt.Description != "" {
			promptEvents = append(promptEvents, SystemMessageEvent{Message: fmt.Sprintf("  %d. %s (%s)", i+1, opt.Label, opt.Description)})
		} else {
			promptEvents = append(promptEvents, SystemMessageEvent{Message: fmt.Sprintf("  %d. %s", i+1, opt.Label)})
		}
	}
	promptEvents = append(promptEvents, SystemMessageEvent{Message: "\nEnter your choice (number):"})
	sm.renderEvents(promptEvents)

	input, _, err := sm.ui.ReadInput()
	if err != nil {
		slog.Error("Failed to read mid-flow choice", "error", err)
		// Default to last option (typically "taken out" / worst outcome).
		return sm.ProvideMidFlowResponse(ctx, MidFlowResponse{ChoiceIndex: len(event.Options) - 1})
	}

	// Parse the 1-based number to 0-based index.
	input = trimSpace(input)
	for i := range event.Options {
		if input == fmt.Sprintf("%d", i+1) {
			return sm.ProvideMidFlowResponse(ctx, MidFlowResponse{ChoiceIndex: i})
		}
	}

	// Invalid input — default to last option.
	sm.renderEvents([]GameEvent{SystemMessageEvent{Message: "Invalid choice."}})
	return sm.ProvideMidFlowResponse(ctx, MidFlowResponse{ChoiceIndex: len(event.Options) - 1})
}

// resolveMidFlowFreeText handles a free-text prompt for the terminal UI.
func (sm *SceneManager) resolveMidFlowFreeText(ctx context.Context, event InputRequestEvent) (*InputResult, error) {
	sm.renderEvents([]GameEvent{SystemMessageEvent{Message: event.Prompt}})

	input, _, err := sm.ui.ReadInput()
	if err != nil {
		slog.Error("Failed to read mid-flow free text", "error", err)
		return sm.ProvideMidFlowResponse(ctx, MidFlowResponse{Text: ""})
	}

	return sm.ProvideMidFlowResponse(ctx, MidFlowResponse{Text: trimSpace(input)})
}

// trimSpace is a small helper to keep the import list clean in this file.
func trimSpace(s string) string {
	// Trim leading/trailing whitespace.
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
