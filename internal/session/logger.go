// Package session provides session logging for game transcripts.
// The session log captures the full back-and-forth of gameplay in YAML format
// for analysis and test case extraction.
package session

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Entry represents a single log entry in the session
type Entry struct {
	Timestamp time.Time `yaml:"timestamp"`
	Type      string    `yaml:"type"`
	Data      any       `yaml:"data"`
}

// Logger handles session logging to a YAML file
type Logger struct {
	file    *os.File
	mu      sync.Mutex
	enabled bool
}

// NewLogger creates a new session logger that writes to the specified file.
// Returns a disabled no-op logger if path is empty.
func NewLogger(path string) (*Logger, error) {
	if path == "" {
		return &Logger{enabled: false}, nil
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create session log file: %w", err)
	}

	return &Logger{
		file:    file,
		enabled: true,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// IsEnabled returns true if logging is active
func (l *Logger) IsEnabled() bool {
	return l.enabled
}

// Log writes an entry to the session log. Data can be any struct or map.
func (l *Logger) Log(entryType string, data any) {
	if !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := Entry{
		Timestamp: time.Now(),
		Type:      entryType,
		Data:      data,
	}

	// Write YAML document separator
	if _, err := l.file.WriteString("---\n"); err != nil {
		fmt.Fprintf(os.Stderr, "session log write error: %v\n", err)
		return
	}

	// Marshal and write the entry
	out, err := yaml.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "session log marshal error: %v\n", err)
		return
	}

	if _, err := l.file.Write(out); err != nil {
		fmt.Fprintf(os.Stderr, "session log write error: %v\n", err)
		return
	}

	// Flush to ensure entries are written immediately
	if err := l.file.Sync(); err != nil {
		fmt.Fprintf(os.Stderr, "session log sync error: %v\n", err)
	}
}
