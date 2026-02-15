package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Session manages a single WebSocket ↔ GameSessionManager interaction.
type Session struct {
	driver engine.GameSessionManager
	conn   *websocket.Conn
	logger *slog.Logger
}

// NewSession creates a session for the given WebSocket connection and game driver.
func NewSession(conn *websocket.Conn, driver engine.GameSessionManager, logger *slog.Logger) *Session {
	if logger == nil {
		logger = slog.Default()
	}
	return &Session{
		driver: driver,
		conn:   conn,
		logger: logger,
	}
}

// Run executes the session lifecycle: start the game, then loop on client messages.
// It blocks until the context is cancelled, the client disconnects, or the game ends.
func (s *Session) Run(ctx context.Context) error {
	// 1. Start the game and send opening events
	events, err := s.driver.Start(ctx)
	if err != nil {
		return fmt.Errorf("game start: %w", err)
	}

	if err := s.sendEvents(ctx, events); err != nil {
		return fmt.Errorf("send opening events: %w", err)
	}
	// Send initial result_meta (no flags set — ready for input)
	if err := s.sendResultMeta(ctx, ResultMeta{}); err != nil {
		return fmt.Errorf("send initial result_meta: %w", err)
	}

	// 2. Message loop
	for {
		msg, err := s.readClientMessage(ctx)
		if err != nil {
			return fmt.Errorf("read client message: %w", err)
		}

		result, err := s.dispatch(ctx, msg)
		if err != nil {
			s.logger.Error("dispatch error", "type", msg.Type, "error", err)
			// Send the error as a system message so the client sees feedback
			sysErr := uicontract.SystemMessageEvent{Message: fmt.Sprintf("Error: %v", err)}
			_ = s.sendEvents(ctx, []uicontract.GameEvent{sysErr})
			continue
		}

		if err := s.sendEvents(ctx, result.Events); err != nil {
			return fmt.Errorf("send result events: %w", err)
		}

		meta := ResultMeta{
			AwaitingInvoke:  result.AwaitingInvoke,
			AwaitingMidFlow: result.AwaitingMidFlow,
			GameOver:        result.GameOver,
			SceneEnded:      result.SceneEnded,
		}
		if err := s.sendResultMeta(ctx, meta); err != nil {
			return fmt.Errorf("send result_meta: %w", err)
		}

		if result.GameOver {
			s.logger.Info("game over, closing session")
			return nil
		}
	}
}

// dispatch routes a client message to the appropriate GameSessionManager method.
func (s *Session) dispatch(ctx context.Context, msg ClientMessage) (*engine.InputResult, error) {
	switch msg.Type {
	case ClientInput:
		return s.driver.HandleInput(ctx, msg.Text)
	case ClientInvoke:
		resp := uicontract.InvokeResponse{
			AspectIndex: msg.AspectIndex,
			IsReroll:    msg.IsReroll,
		}
		return s.driver.ProvideInvokeResponse(ctx, resp)
	case ClientMidFlow:
		resp := uicontract.MidFlowResponse{
			ChoiceIndex: msg.ChoiceIndex,
			Text:        msg.FreeText,
		}
		return s.driver.ProvideMidFlowResponse(ctx, resp)
	default:
		return nil, fmt.Errorf("unhandled message type: %s", msg.Type)
	}
}

// readClientMessage reads and parses a single JSON message from the WebSocket.
func (s *Session) readClientMessage(ctx context.Context) (ClientMessage, error) {
	var raw json.RawMessage
	if err := wsjson.Read(ctx, s.conn, &raw); err != nil {
		return ClientMessage{}, err
	}
	return ParseClientMessage(raw)
}

// sendEvents marshals and writes each event as a server message.
func (s *Session) sendEvents(ctx context.Context, events []uicontract.GameEvent) error {
	for _, event := range events {
		data, err := MarshalEvent(event)
		if err != nil {
			s.logger.Error("marshal event failed", "error", err)
			continue
		}
		if err := s.conn.Write(ctx, websocket.MessageText, data); err != nil {
			return err
		}
	}
	return nil
}

// sendResultMeta marshals and writes a result_meta server message.
func (s *Session) sendResultMeta(ctx context.Context, meta ResultMeta) error {
	data, err := MarshalResultMeta(meta)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}
