// Package syncdriver provides a synchronous blocking game loop that wraps the
// engine's async API (Start/HandleInput/ProvideInvokeResponse/ProvideMidFlowResponse).
// This is used by the CLI and any other blocking UI that reads input in a loop.
//
// The game engine itself is purely event-driven: it returns []GameEvent and
// *InputResult. The syncdriver bridges that to a blocking terminal-style loop:
// ReadInput -> HandleInput -> Emit events -> drive prompts -> repeat.
package syncdriver

import (
	"context"
	"fmt"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// BlockingUI is the interface a terminal (or similar blocking) UI must
// implement to be driven by Run.
type BlockingUI interface {
	// ReadInput returns the player's input, whether they want to exit, and any error.
	ReadInput() (input string, isExit bool, err error)

	// Emit renders a single game event.
	Emit(event uicontract.GameEvent)

	// PromptForInvoke prompts the player to invoke an aspect after a roll.
	PromptForInvoke(available []uicontract.InvokableAspect, fatePoints int, currentResult string, shiftsNeeded int) uicontract.InvokeResponse

	// PromptForMidFlow handles a mid-flow input request (e.g. consequence choice, concession).
	PromptForMidFlow(event uicontract.InputRequestEvent) uicontract.MidFlowResponse
}

// Run drives the game in a synchronous blocking loop. It calls gm.Start to
// initialize, then loops: ReadInput -> HandleInput -> emit events -> drive
// pending invoke/mid-flow prompts -> repeat until exit or game over.
//
// If onStart is non-nil, it is called after Start succeeds (and before the
// first ReadInput) so the caller can wire up SceneInfo or do other setup.
func Run(ctx context.Context, gm engine.GameSessionManager, ui BlockingUI, onStart func()) error {
	events, err := gm.Start(ctx)
	if err != nil {
		return err
	}

	if onStart != nil {
		onStart()
	}

	emitAll(ui, events)

	for {
		input, isExit, err := ui.ReadInput()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		if input == "" {
			continue
		}

		if isExit {
			_ = gm.Save()
			return nil
		}

		result, err := gm.HandleInput(ctx, input)
		if err != nil {
			return err
		}
		emitAll(ui, result.Events)

		result, err = driveBlockingPrompts(ctx, gm, ui, result)
		if err != nil {
			return err
		}

		if result.GameOver {
			return nil
		}
	}
}

// driveBlockingPrompts resolves pending invoke and mid-flow prompts in a
// synchronous loop, collecting responses and feeding them back to the GameManager.
func driveBlockingPrompts(ctx context.Context, gm engine.GameSessionManager, ui BlockingUI, result *engine.InputResult) (*engine.InputResult, error) {
	for result.AwaitingInvoke {
		var prompt *uicontract.InvokePromptEvent
		for i := len(result.Events) - 1; i >= 0; i-- {
			if p, ok := result.Events[i].(uicontract.InvokePromptEvent); ok {
				prompt = &p
				break
			}
		}
		if prompt == nil {
			return nil, fmt.Errorf("AwaitingInvoke set but no InvokePromptEvent in events")
		}

		resp := ui.PromptForInvoke(prompt.Available, prompt.FatePoints, prompt.CurrentResult, prompt.ShiftsNeeded)

		var err error
		result, err = gm.ProvideInvokeResponse(ctx, resp)
		if err != nil {
			return nil, err
		}
		emitAll(ui, result.Events)
	}

	for result.AwaitingMidFlow {
		var prompt *uicontract.InputRequestEvent
		for i := len(result.Events) - 1; i >= 0; i-- {
			if p, ok := result.Events[i].(uicontract.InputRequestEvent); ok {
				prompt = &p
				break
			}
		}
		if prompt == nil {
			return nil, fmt.Errorf("AwaitingMidFlow set but no InputRequestEvent in events")
		}

		resp := ui.PromptForMidFlow(*prompt)

		var err error
		result, err = gm.ProvideMidFlowResponse(ctx, resp)
		if err != nil {
			return nil, err
		}
		emitAll(ui, result.Events)
	}

	return result, nil
}

// emitAll dispatches a slice of events to the UI for rendering.
func emitAll(ui BlockingUI, events []uicontract.GameEvent) {
	for _, event := range events {
		ui.Emit(event)
	}
}
