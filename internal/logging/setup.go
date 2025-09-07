package logging

import (
	"log"
	"log/slog"
	"os"
)

// SetupDefaultLogging configures structured logging with slog
// Uses DEBUG environment variable to enable debug mode, otherwise defaults to error level
// Only supports text format output to stdout
// Also ensures standard log package outputs are visible
func SetupDefaultLogging() {
	// Check for debug mode, default to error level
	var level slog.Level
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	} else {
		level = slog.LevelError
	}

	// Use text format only
	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(os.Stdout, opts)

	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Ensure standard log package (used by log.Fatalf) outputs to stderr
	// and uses a visible format
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags)
}
