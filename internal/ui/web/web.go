package web

import (
	"fmt"
	"sync"

	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// Compile-time interface check.
var _ uicontract.UI = (*WebUI)(nil)

// WebUI implements uicontract.UI for event-driven WebSocket sessions.
//
// In the async engine path (Start → HandleInput → InputResult), the engine
// never calls ReadInput or Emit directly. All events flow through the
// returned InputResult.Events slices. WebUI exists to satisfy the SetUI
// contract on GameManager.
//
// If a future engine path does call Emit (e.g. streaming partial events
// during long LLM calls), WebUI captures those events in a buffer that
// the session can drain.
type WebUI struct {
	mu     sync.Mutex
	buffer []uicontract.GameEvent
}

// NewWebUI creates a new WebUI adapter.
func NewWebUI() *WebUI {
	return &WebUI{}
}

// Emit captures a GameEvent for later retrieval.
// In the current async path this is never called, but it provides a
// safety net if the engine emits events outside of InputResult.
func (w *WebUI) Emit(event uicontract.GameEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buffer = append(w.buffer, event)
}

// ReadInput is not used in the async path and always returns an error.
// The WebSocket session drives input via HandleInput directly.
func (w *WebUI) ReadInput() (string, bool, error) {
	return "", false, fmt.Errorf("ReadInput not supported in WebUI; use HandleInput")
}

// DrainBuffer returns and clears any events captured via Emit.
func (w *WebUI) DrainBuffer() []uicontract.GameEvent {
	w.mu.Lock()
	defer w.mu.Unlock()
	events := w.buffer
	w.buffer = nil
	return events
}
