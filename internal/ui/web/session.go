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
	driver  engine.GameSessionManager
	conn    *websocket.Conn
	logger  *slog.Logger
	gameID  string
	factory GameSessionManagerFactory // non-nil only for setup sessions
	setup   SetupConfig               // preset list for setup_request
}

// NewSession creates a session for the given WebSocket connection and game driver.
// Use this when the driver is already created (e.g. resumed game).
func NewSession(conn *websocket.Conn, driver engine.GameSessionManager, logger *slog.Logger, gameID string) *Session {
	if logger == nil {
		logger = slog.Default()
	}
	return &Session{
		driver: driver,
		conn:   conn,
		logger: logger,
		gameID: gameID,
	}
}

// NewSetupSession creates a session that sends setup_request and waits
// for the player's choice before creating a game driver.
func NewSetupSession(conn *websocket.Conn, factory GameSessionManagerFactory, setup SetupConfig, logger *slog.Logger, gameID string) *Session {
	if logger == nil {
		logger = slog.Default()
	}
	return &Session{
		conn:    conn,
		logger:  logger,
		gameID:  gameID,
		factory: factory,
		setup:   setup,
	}
}

// Run executes the session lifecycle: start the game, then loop on client messages.
// It blocks until the context is cancelled, the client disconnects, or the game ends.
func (s *Session) Run(ctx context.Context) error {
	// 0. Send session_init with game ID so the client can store it for reconnection.
	if err := s.sendSessionInit(ctx); err != nil {
		return fmt.Errorf("send session_init: %w", err)
	}

	// 0b. If we don't have a driver yet, run the setup flow to get one.
	if s.driver == nil {
		if err := s.runSetup(ctx); err != nil {
			return fmt.Errorf("setup: %w", err)
		}
	}

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

// runSetup sends the setup_request event and waits for the client's
// setup message. On success it creates a GameSessionManager via the factory
// and stores it in s.driver.
func (s *Session) runSetup(ctx context.Context) error {
	// Send setup_request with available presets.
	if err := s.sendSetupRequest(ctx); err != nil {
		return fmt.Errorf("send setup_request: %w", err)
	}

	// Wait for the client's setup choice.
	msg, err := s.readClientMessage(ctx)
	if err != nil {
		return fmt.Errorf("read setup message: %w", err)
	}
	if msg.Type != ClientSetup {
		return fmt.Errorf("expected setup message, got %q", msg.Type)
	}

	setup := &GameSetup{
		PresetID: msg.PresetID,
		Custom:   msg.Custom,
	}

	s.logger.Info("setup received",
		"preset_id", setup.PresetID,
		"has_custom", setup.Custom != nil,
	)

	// If custom scenario requested, notify client of generation in progress.
	if setup.Custom != nil {
		if err := s.sendSetupGenerating(ctx, "Generating your scenario..."); err != nil {
			return fmt.Errorf("send setup_generating: %w", err)
		}
	}

	driver, err := s.factory(s.gameID, setup)
	if err != nil {
		// Send error to client and return
		sysErr := uicontract.SystemMessageEvent{Message: fmt.Sprintf("Setup failed: %v", err)}
		_ = s.sendEvents(ctx, []uicontract.GameEvent{sysErr})
		return fmt.Errorf("factory: %w", err)
	}
	s.driver = driver
	return nil
}

// sendSetupRequest sends a setup_request server message listing available presets.
func (s *Session) sendSetupRequest(ctx context.Context) error {
	req := SetupRequest{
		Presets:     s.setup.Presets,
		AllowCustom: s.setup.AllowCustom,
	}
	data, err := MarshalSetupRequest(req)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}

// sendSetupGenerating sends a setup_generating server message so the client
// can display a loading indicator during LLM scenario generation.
func (s *Session) sendSetupGenerating(ctx context.Context, message string) error {
	data, err := MarshalSetupGenerating(message)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
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

// sendSessionInit sends a session_init server message so the client knows
// the game ID for this session and can reconnect to it later.
func (s *Session) sendSessionInit(ctx context.Context) error {
	data, err := MarshalSessionInit(s.gameID)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, data)
}
