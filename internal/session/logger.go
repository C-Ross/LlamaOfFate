// Package session provides session logging for game transcripts.
// The session log captures the full back-and-forth of gameplay in YAML format
// for analysis and test case extraction.
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// SessionsDir is the directory where session logs are stored
	SessionsDir = "sessions"
)

// GenerateLogPath creates a session log path in the sessions/ directory.
// The filename is constructed from a prefix and parts, each sanitized to be filesystem-safe.
// Parts are joined with underscores and a timestamp is appended.
// maxLen specifies the maximum length for each sanitized part (0 = no limit).
//
// Example: GenerateLogPath("session", []string{"western", "jesse calhoun"}, 20)
// Returns: "sessions/session_western_jesse_calhoun_20060102_150405.yaml"
func GenerateLogPath(prefix string, parts []string, maxLen int) (string, error) {
	// Ensure sessions directory exists
	if err := os.MkdirAll(SessionsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Sanitize and collect parts
	var sanitized []string
	sanitized = append(sanitized, sanitizePart(prefix, maxLen))
	for _, part := range parts {
		if part != "" {
			sanitized = append(sanitized, sanitizePart(part, maxLen))
		}
	}

	// Add timestamp
	sanitized = append(sanitized, time.Now().Format("20060102_150405"))

	// Build filename
	filename := strings.Join(sanitized, "_") + ".yaml"
	return filepath.Join(SessionsDir, filename), nil
}

// sanitizePart converts a string to a filesystem-safe identifier.
// Converts to lowercase, replaces spaces with underscores, removes non-alphanumeric chars.
// If maxLen > 0, truncates to that length.
func sanitizePart(s string, maxLen int) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, s)
	if maxLen > 0 && len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

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
