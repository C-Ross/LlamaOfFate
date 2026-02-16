package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_Health(t *testing.T) {
	h := NewHandler(func(_ string) (engine.GameSessionManager, error) {
		return nil, nil
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
}

func TestHandler_WebSocket_FullRoundTrip(t *testing.T) {
	factory := func(_ string) (engine.GameSessionManager, error) {
		return &mockDriver{
			startEvents: []uicontract.GameEvent{
				uicontract.NarrativeEvent{Text: "Welcome!", SceneName: "Test Scene"},
			},
			handleInputResult: &engine.InputResult{
				Events: []uicontract.GameEvent{
					uicontract.DialogEvent{PlayerInput: "hello", GMResponse: "world"},
				},
				GameOver: true,
			},
		}, nil
	}

	h := NewHandler(factory, nil)
	server := httptest.NewServer(h)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, _, err := websocket.Dial(ctx, "ws"+server.URL[4:]+"/ws", nil)
	require.NoError(t, err)
	defer func() { _ = client.CloseNow() }()

	// Read session_init
	var msg ServerMessage
	err = wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	assert.Equal(t, "session_init", msg.Event)

	// Read opening narrative
	err = wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	assert.Equal(t, "narrative", msg.Event)

	// Verify data content
	var narrative uicontract.NarrativeEvent
	require.NoError(t, json.Unmarshal(msg.Data, &narrative))
	assert.Equal(t, "Welcome!", narrative.Text)

	// Read initial result_meta
	err = wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	assert.Equal(t, "result_meta", msg.Event)

	// Send player input
	err = wsjson.Write(ctx, client, ClientMessage{Type: ClientInput, Text: "hello"})
	require.NoError(t, err)

	// Read dialog response
	err = wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	assert.Equal(t, "dialog", msg.Event)

	// Read final result_meta
	err = wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	assert.Equal(t, "result_meta", msg.Event)

	var meta ResultMeta
	require.NoError(t, json.Unmarshal(msg.Data, &meta))
	assert.True(t, meta.GameOver)
}

func TestHandler_WebSocket_MultipleEvents(t *testing.T) {
	factory := func(_ string) (engine.GameSessionManager, error) {
		return &mockDriver{
			startEvents: []uicontract.GameEvent{
				uicontract.NarrativeEvent{Text: "Scene begins."},
				uicontract.SystemMessageEvent{Message: "You have 3 fate points."},
			},
			handleInputResult: &engine.InputResult{
				Events:   []uicontract.GameEvent{uicontract.GameOverEvent{Reason: "done"}},
				GameOver: true,
			},
		}, nil
	}

	h := NewHandler(factory, nil)
	server := httptest.NewServer(h)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, _, err := websocket.Dial(ctx, "ws"+server.URL[4:]+"/ws", nil)
	require.NoError(t, err)
	defer func() { _ = client.CloseNow() }()

	// Read session_init
	var msg ServerMessage
	err = wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	assert.Equal(t, "session_init", msg.Event)

	// Read two opening events
	err = wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	assert.Equal(t, "narrative", msg.Event)

	err = wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	assert.Equal(t, "system_message", msg.Event)

	// Read result_meta
	err = wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	assert.Equal(t, "result_meta", msg.Event)
}

func TestIsNormalClose(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "StatusGoingAway is normal",
			err:      websocket.CloseError{Code: websocket.StatusGoingAway, Reason: ""},
			expected: true,
		},
		{
			name:     "StatusNormalClosure is normal",
			err:      websocket.CloseError{Code: websocket.StatusNormalClosure, Reason: "bye"},
			expected: true,
		},
		{
			name:     "EOF is normal",
			err:      io.EOF,
			expected: true,
		},
		{
			name:     "context canceled is normal",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "wrapped close error is normal",
			err:      fmt.Errorf("read client message: %w", websocket.CloseError{Code: websocket.StatusGoingAway}),
			expected: true,
		},
		{
			name:     "real error is not normal",
			err:      fmt.Errorf("something broke"),
			expected: false,
		},
		{
			name:     "StatusInternalError is not normal",
			err:      websocket.CloseError{Code: websocket.StatusInternalError, Reason: "crash"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isNormalClose(tt.err))
		})
	}
}
