package web

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebUI_ImplementsUI(t *testing.T) {
	var _ uicontract.UI = (*WebUI)(nil)
}

func TestWebUI_ReadInput_ReturnsError(t *testing.T) {
	ui := NewWebUI()
	_, _, err := ui.ReadInput()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestWebUI_Emit_CapturesEvents(t *testing.T) {
	ui := NewWebUI()

	ui.Emit(uicontract.NarrativeEvent{Text: "first"})
	ui.Emit(uicontract.SystemMessageEvent{Message: "second"})

	events := ui.DrainBuffer()
	require.Len(t, events, 2)
	assert.Equal(t, "first", events[0].(uicontract.NarrativeEvent).Text)
	assert.Equal(t, "second", events[1].(uicontract.SystemMessageEvent).Message)
}

func TestWebUI_DrainBuffer_ClearsBuffer(t *testing.T) {
	ui := NewWebUI()

	ui.Emit(uicontract.NarrativeEvent{Text: "event"})
	events := ui.DrainBuffer()
	require.Len(t, events, 1)

	// Second drain should return nil
	events = ui.DrainBuffer()
	assert.Nil(t, events)
}

func TestWebUI_DrainBuffer_EmptyByDefault(t *testing.T) {
	ui := NewWebUI()
	events := ui.DrainBuffer()
	assert.Nil(t, events)
}

func TestWebUI_EmitConcurrentSafety(t *testing.T) {
	ui := NewWebUI()
	done := make(chan struct{})

	// Write from multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			ui.Emit(uicontract.NarrativeEvent{Text: "concurrent"})
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	events := ui.DrainBuffer()
	assert.Len(t, events, 10)
}
