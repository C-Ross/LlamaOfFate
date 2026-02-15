package web

import (
	"context"
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

// mockDriver implements GameSessionManager for testing. Each method records its call
// and returns the configured result.
type mockDriver struct {
	startEvents []uicontract.GameEvent
	startErr    error

	handleInputResult *engine.InputResult
	handleInputErr    error
	lastInput         string

	invokeResult *engine.InputResult
	invokeErr    error
	lastInvoke   uicontract.InvokeResponse

	midFlowResult *engine.InputResult
	midFlowErr    error
	lastMidFlow   uicontract.MidFlowResponse
}

func (m *mockDriver) Start(_ context.Context) ([]uicontract.GameEvent, error) {
	return m.startEvents, m.startErr
}

func (m *mockDriver) HandleInput(_ context.Context, input string) (*engine.InputResult, error) {
	m.lastInput = input
	return m.handleInputResult, m.handleInputErr
}

func (m *mockDriver) ProvideInvokeResponse(_ context.Context, resp uicontract.InvokeResponse) (*engine.InputResult, error) {
	m.lastInvoke = resp
	return m.invokeResult, m.invokeErr
}

func (m *mockDriver) ProvideMidFlowResponse(_ context.Context, resp uicontract.MidFlowResponse) (*engine.InputResult, error) {
	m.lastMidFlow = resp
	return m.midFlowResult, m.midFlowErr
}

// wsTestPair creates a WebSocket server/client pair for testing. The server
// side calls sessionFn with the accepted connection; the client side is
// returned for the test to use.
func wsTestPair(t *testing.T, sessionFn func(conn *websocket.Conn)) *websocket.Conn {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("ws accept: %v", err)
		}
		sessionFn(conn)
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	client, _, err := websocket.Dial(ctx, "ws"+server.URL[4:], nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.CloseNow() })

	return client
}

// readServerMessage reads and parses a ServerMessage from the client connection.
func readServerMessage(t *testing.T, ctx context.Context, client *websocket.Conn) ServerMessage {
	t.Helper()
	var msg ServerMessage
	err := wsjson.Read(ctx, client, &msg)
	require.NoError(t, err)
	return msg
}

func TestSession_StartsAndSendsOpeningEvents(t *testing.T) {
	driver := &mockDriver{
		startEvents: []uicontract.GameEvent{
			uicontract.NarrativeEvent{Text: "Welcome to the saloon.", SceneName: "Saloon"},
		},
		handleInputResult: &engine.InputResult{
			Events:   []uicontract.GameEvent{uicontract.DialogEvent{GMResponse: "reply"}},
			GameOver: true,
		},
	}

	sessionDone := make(chan error, 1)
	client := wsTestPair(t, func(conn *websocket.Conn) {
		s := NewSession(conn, driver, nil)
		sessionDone <- s.Run(context.Background())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Read opening narrative
	msg := readServerMessage(t, ctx, client)
	assert.Equal(t, "narrative", msg.Event)

	// Read initial result_meta
	msg = readServerMessage(t, ctx, client)
	assert.Equal(t, "result_meta", msg.Event)

	// Send input to trigger game over
	err := wsjson.Write(ctx, client, ClientMessage{Type: ClientInput, Text: "hello"})
	require.NoError(t, err)

	// Read dialog response
	msg = readServerMessage(t, ctx, client)
	assert.Equal(t, "dialog", msg.Event)

	// Read final result_meta (game over)
	msg = readServerMessage(t, ctx, client)
	assert.Equal(t, "result_meta", msg.Event)

	// Session should complete
	select {
	case err := <-sessionDone:
		assert.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("session did not complete in time")
	}
}

func TestSession_HandleInputRoundTrip(t *testing.T) {
	driver := &mockDriver{
		startEvents: []uicontract.GameEvent{
			uicontract.NarrativeEvent{Text: "Scene begins."},
		},
		handleInputResult: &engine.InputResult{
			Events: []uicontract.GameEvent{
				uicontract.DialogEvent{PlayerInput: "search", GMResponse: "You find gold."},
			},
			GameOver: true,
		},
	}

	sessionDone := make(chan error, 1)
	client := wsTestPair(t, func(conn *websocket.Conn) {
		s := NewSession(conn, driver, nil)
		sessionDone <- s.Run(context.Background())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip opening events
	_ = readServerMessage(t, ctx, client) // narrative
	_ = readServerMessage(t, ctx, client) // result_meta

	// Send player input
	err := wsjson.Write(ctx, client, ClientMessage{Type: ClientInput, Text: "I search the room"})
	require.NoError(t, err)

	// Read response
	msg := readServerMessage(t, ctx, client)
	assert.Equal(t, "dialog", msg.Event)

	_ = readServerMessage(t, ctx, client) // result_meta

	select {
	case err := <-sessionDone:
		assert.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("session did not complete in time")
	}

	assert.Equal(t, "I search the room", driver.lastInput)
}

func TestSession_InvokeResponseRoundTrip(t *testing.T) {
	callCount := 0
	driver := &mockDriver{
		startEvents: []uicontract.GameEvent{uicontract.NarrativeEvent{Text: "Start"}},
	}
	// First HandleInput returns awaiting invoke
	driver.handleInputResult = &engine.InputResult{
		Events: []uicontract.GameEvent{
			uicontract.InvokePromptEvent{FatePoints: 3},
		},
		AwaitingInvoke: true,
	}
	// Invoke response ends the game
	driver.invokeResult = &engine.InputResult{
		Events:   []uicontract.GameEvent{uicontract.SystemMessageEvent{Message: "Invoked!"}},
		GameOver: true,
	}

	sessionDone := make(chan error, 1)
	client := wsTestPair(t, func(conn *websocket.Conn) {
		// Override HandleInput to track call count and switch results
		origResult := driver.handleInputResult
		driver.handleInputResult = origResult
		s := NewSession(conn, driver, nil)
		sessionDone <- s.Run(context.Background())
	})
	_ = callCount

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip opening events
	_ = readServerMessage(t, ctx, client) // narrative
	_ = readServerMessage(t, ctx, client) // result_meta

	// Send input that triggers invoke prompt
	err := wsjson.Write(ctx, client, ClientMessage{Type: ClientInput, Text: "attack bandit"})
	require.NoError(t, err)

	// Read invoke prompt
	msg := readServerMessage(t, ctx, client)
	assert.Equal(t, "invoke_prompt", msg.Event)

	// Read result_meta with awaitingInvoke
	msg = readServerMessage(t, ctx, client)
	assert.Equal(t, "result_meta", msg.Event)

	// Send invoke response
	err = wsjson.Write(ctx, client, ClientMessage{
		Type:        ClientInvoke,
		AspectIndex: 0,
		IsReroll:    false,
	})
	require.NoError(t, err)

	// Read invoke result
	msg = readServerMessage(t, ctx, client)
	assert.Equal(t, "system_message", msg.Event)

	_ = readServerMessage(t, ctx, client) // result_meta

	select {
	case err := <-sessionDone:
		assert.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("session did not complete in time")
	}

	assert.Equal(t, 0, driver.lastInvoke.AspectIndex)
}

func TestSession_MidFlowResponseRoundTrip(t *testing.T) {
	driver := &mockDriver{
		startEvents: []uicontract.GameEvent{uicontract.NarrativeEvent{Text: "Start"}},
		handleInputResult: &engine.InputResult{
			Events: []uicontract.GameEvent{
				uicontract.InputRequestEvent{
					Type:   uicontract.InputRequestNumberedChoice,
					Prompt: "Choose consequence:",
					Options: []uicontract.InputOption{
						{Label: "Mild: Bruised"},
						{Label: "Moderate: Broken Arm"},
					},
				},
			},
			AwaitingMidFlow: true,
		},
		midFlowResult: &engine.InputResult{
			Events:   []uicontract.GameEvent{uicontract.SystemMessageEvent{Message: "Chose mild."}},
			GameOver: true,
		},
	}

	sessionDone := make(chan error, 1)
	client := wsTestPair(t, func(conn *websocket.Conn) {
		s := NewSession(conn, driver, nil)
		sessionDone <- s.Run(context.Background())
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip opening
	_ = readServerMessage(t, ctx, client)
	_ = readServerMessage(t, ctx, client)

	// Trigger mid-flow
	err := wsjson.Write(ctx, client, ClientMessage{Type: ClientInput, Text: "action"})
	require.NoError(t, err)

	_ = readServerMessage(t, ctx, client) // input_request
	_ = readServerMessage(t, ctx, client) // result_meta

	// Send mid-flow response
	err = wsjson.Write(ctx, client, ClientMessage{
		Type:        ClientMidFlow,
		ChoiceIndex: 0,
	})
	require.NoError(t, err)

	msg := readServerMessage(t, ctx, client)
	assert.Equal(t, "system_message", msg.Event)

	_ = readServerMessage(t, ctx, client) // result_meta

	select {
	case err := <-sessionDone:
		assert.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("session did not complete in time")
	}

	assert.Equal(t, 0, driver.lastMidFlow.ChoiceIndex)
}
