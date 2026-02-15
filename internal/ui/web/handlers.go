package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/coder/websocket"
)

// GameSessionManagerFactory creates a new GameSessionManager for each WebSocket session.
// The caller is responsible for wiring the engine, player, scenario, etc.
type GameSessionManagerFactory func() (engine.GameSessionManager, error)

// Handler provides HTTP endpoints for the web UI.
type Handler struct {
	factory GameSessionManagerFactory
	logger  *slog.Logger
	mux     *http.ServeMux
}

// NewHandler creates an HTTP handler with WebSocket and health endpoints.
// TODO: serve static files from web/dist for production builds (no Vite proxy).
func NewHandler(factory GameSessionManagerFactory, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &Handler{
		factory: factory,
		logger:  logger,
		mux:     http.NewServeMux(),
	}
	h.mux.HandleFunc("GET /ws", h.handleWebSocket)
	h.mux.HandleFunc("GET /health", h.handleHealth)
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// handleHealth returns a simple health check response.
func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// handleWebSocket upgrades to WebSocket and runs a game session.
func (h *Handler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Allow any origin for development; tighten in production.
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.logger.Error("websocket accept failed", "error", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	driver, err := h.factory()
	if err != nil {
		h.logger.Error("failed to create game driver", "error", err)
		_ = conn.Close(websocket.StatusInternalError, "failed to initialize game")
		return
	}

	session := NewSession(conn, driver, h.logger)

	h.logger.Info("websocket session started", "remote", r.RemoteAddr)
	if err := session.Run(r.Context()); err != nil {
		if isNormalClose(err) {
			h.logger.Info("client disconnected", "remote", r.RemoteAddr)
			return
		}
		h.logger.Error("session ended with error", "error", err, "remote", r.RemoteAddr)
		_ = conn.Close(websocket.StatusInternalError, "session error")
		return
	}

	h.logger.Info("websocket session completed", "remote", r.RemoteAddr)
	_ = conn.Close(websocket.StatusNormalClosure, "game over")
}

// isNormalClose returns true when the error indicates the client disconnected
// normally (tab close, page reload, graceful close) rather than a real failure.
func isNormalClose(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
		return true
	}
	status := websocket.CloseStatus(err)
	return status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway
}