package web

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/coder/websocket"
)

// GameSetup carries the player's scenario + character choice from the setup
// screen. Exactly one of PresetID or Custom is set.
type GameSetup struct {
	PresetID string       // Non-empty when the player picked a preset
	Custom   *CustomSetup // Non-nil when the player chose \"Create Your Own\"
}

// GameSessionManagerFactory creates a new GameSessionManager for a WebSocket session.
// The ctx parameter carries request-scoped context (used for LLM calls during generation).
// The gameID parameter identifies the game. When non-empty, the factory should
// attempt to resume the game from a saved state. When empty, a fresh game is created.
// The setup parameter carries the player's scenario choice (nil when resuming).
type GameSessionManagerFactory func(ctx context.Context, gameID string, setup *GameSetup) (engine.GameSessionManager, error)

// SetupConfig holds the data the Session sends in the setup_request event.
type SetupConfig struct {
	Presets     []ScenarioPreset
	AllowCustom bool
}

// Handler provides HTTP endpoints for the web UI.
type Handler struct {
	factory     GameSessionManagerFactory
	setupConfig SetupConfig
	logger      *slog.Logger
	mux         *http.ServeMux
}

// NewHandler creates an HTTP handler with WebSocket, health, and optional
// static file serving endpoints. When staticFS is non-nil, the handler serves
// the embedded frontend files and falls back to index.html for SPA routing.
func NewHandler(factory GameSessionManagerFactory, setupCfg SetupConfig, logger *slog.Logger, staticFS fs.FS) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &Handler{
		factory:     factory,
		setupConfig: setupCfg,
		logger:      logger,
		mux:         http.NewServeMux(),
	}
	h.mux.HandleFunc("GET /ws", h.handleWebSocket)
	h.mux.HandleFunc("GET /health", h.handleHealth)
	if staticFS != nil {
		h.mux.Handle("GET /", spaHandler(staticFS))
	}
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
// If the client provides a ?game_id=<id> query parameter, the session
// will attempt to resume a previously saved game. Otherwise a new game
// (and a new game ID) is created.
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

	gameID := r.URL.Query().Get("game_id")
	if gameID == "" {
		gameID = generateGameID()
		h.logger.Info("new game", "game_id", gameID)
	} else {
		h.logger.Info("resuming game", "game_id", gameID)
	}

	var session *Session

	driver, err := h.factory(r.Context(), gameID, nil)
	if err != nil {
		// If the save file is corrupt/incompatible, enter setup flow
		// and notify the user with an error toast.
		var saveErr *engine.SaveCorruptError
		if errors.As(err, &saveErr) {
			h.logger.Warn("corrupt save, entering setup", "game_id", gameID, "error", err)
			session = NewSetupSession(conn, h.factory, h.setupConfig, h.logger, gameID)
			session.loadError = saveErr.Error()
		} else {
			h.logger.Error("failed to create game driver", "error", err)
			_ = conn.Close(websocket.StatusInternalError, "failed to initialize game")
			return
		}
	}

	// If the factory returned a driver (resumed game), skip setup.
	// Otherwise begin the setup flow.
	if session == nil {
		if driver != nil {
			session = NewSession(conn, driver, h.logger, gameID)
		} else {
			session = NewSetupSession(conn, h.factory, h.setupConfig, h.logger, gameID)
		}
	}

	h.logger.Info("websocket session started", "remote", r.RemoteAddr, "game_id", gameID)
	if err := session.Run(r.Context()); err != nil {
		if isNormalClose(err) {
			h.logger.Info("client disconnected", "remote", r.RemoteAddr)
			return
		}
		h.logger.Error("session ended with error", "error", err, "remote", r.RemoteAddr)
		_ = conn.Close(websocket.StatusInternalError, "session error")
		return
	}

	h.logger.Info("websocket session completed", "remote", r.RemoteAddr, "game_id", gameID)
	_ = conn.Close(websocket.StatusNormalClosure, "game over")
}

// generateGameID produces a short random hex game identifier.
func generateGameID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback: should never happen.
		return "fallback"
	}
	return fmt.Sprintf("%x", b)
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

// spaHandler returns an http.Handler that serves static files from the given
// filesystem. If the requested path doesn't match a real file (e.g. a
// client-side route like /game/123), it falls back to index.html so the
// React SPA can handle routing.
func spaHandler(staticFS fs.FS) http.Handler {
	fileServer := http.FileServerFS(staticFS)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the exact file. Clean the path to prevent traversal.
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(staticFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		// File not found — serve index.html for SPA routing.
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
