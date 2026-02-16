// Package web implements the WebSocket-based web UI for LlamaOfFate.
//
// The game engine exposes a purely async/event-driven API via
// engine.GameSessionManager (Start → HandleInput → InputResult). The web
// package's Session type drives that API over a WebSocket connection —
// no blocking UI adapter is needed.
package web
