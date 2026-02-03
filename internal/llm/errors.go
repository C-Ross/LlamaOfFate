package llm

import (
	"fmt"
	"net/http"
)

// APIError represents an error response from an LLM API
type APIError struct {
	StatusCode int
	Status     string
	Body       string
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("API request failed with status %d: %s", e.StatusCode, e.Body)
}

// IsRetryable returns true if the error represents a transient failure
// that should be retried
func (e *APIError) IsRetryable() bool {
	switch e.StatusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}
	return false
}
