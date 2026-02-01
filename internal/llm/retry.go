package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"time"
)

// RetryConfig holds configuration for retry behavior
type RetryConfig struct {
	MaxAttempts    int           // Maximum number of retry attempts (default: 3)
	InitialBackoff time.Duration // Initial backoff duration (default: 100ms)
	MaxBackoff     time.Duration // Maximum backoff duration (default: 10s)
	BackoffFactor  float64       // Backoff multiplier (default: 2.0)
}

// DefaultRetryConfig returns a retry configuration with sensible defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		BackoffFactor:  2.0,
	}
}

// RetryingClient wraps an LLMClient and adds automatic retry logic
type RetryingClient struct {
	client LLMClient
	config RetryConfig
}

// NewRetryingClient creates a new RetryingClient that wraps the given client
func NewRetryingClient(client LLMClient, config RetryConfig) *RetryingClient {
	// Apply defaults if not set
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.InitialBackoff <= 0 {
		config.InitialBackoff = 100 * time.Millisecond
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = 10 * time.Second
	}
	if config.BackoffFactor <= 0 {
		config.BackoffFactor = 2.0
	}

	return &RetryingClient{
		client: client,
		config: config,
	}
}

// ChatCompletion implements the LLMClient interface with retry logic
func (r *RetryingClient) ChatCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Check context before attempting
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		response, err := r.client.ChatCompletion(ctx, req)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			slog.Debug("Non-retryable error encountered",
				slog.String("component", "retry_client"),
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()))
			return nil, err
		}

		// Don't sleep after last attempt
		if attempt < r.config.MaxAttempts {
			backoff := r.calculateBackoff(attempt)
			slog.Info("Retrying LLM request after failure",
				slog.String("component", "retry_client"),
				slog.Int("attempt", attempt),
				slog.Int("max_attempts", r.config.MaxAttempts),
				slog.Duration("backoff", backoff),
				slog.String("error", err.Error()))

			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	slog.Error("All retry attempts exhausted",
		slog.String("component", "retry_client"),
		slog.Int("attempts", r.config.MaxAttempts),
		slog.String("error", lastErr.Error()))

	return nil, fmt.Errorf("all %d retry attempts failed: %w", r.config.MaxAttempts, lastErr)
}

// ChatCompletionStream implements the LLMClient interface with retry logic for streaming
func (r *RetryingClient) ChatCompletionStream(ctx context.Context, req CompletionRequest, handler StreamHandler) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		// Check context before attempting
		if err := ctx.Err(); err != nil {
			return err
		}

		err := r.client.ChatCompletionStream(ctx, req, handler)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			slog.Debug("Non-retryable error encountered in stream",
				slog.String("component", "retry_client"),
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()))
			return err
		}

		// Don't sleep after last attempt
		if attempt < r.config.MaxAttempts {
			backoff := r.calculateBackoff(attempt)
			slog.Info("Retrying LLM stream request after failure",
				slog.String("component", "retry_client"),
				slog.Int("attempt", attempt),
				slog.Int("max_attempts", r.config.MaxAttempts),
				slog.Duration("backoff", backoff),
				slog.String("error", err.Error()))

			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	slog.Error("All retry attempts exhausted for stream",
		slog.String("component", "retry_client"),
		slog.Int("attempts", r.config.MaxAttempts),
		slog.String("error", lastErr.Error()))

	return fmt.Errorf("all %d retry attempts failed: %w", r.config.MaxAttempts, lastErr)
}

// GetModelInfo implements the LLMClient interface (no retry needed for this)
func (r *RetryingClient) GetModelInfo() ModelInfo {
	return r.client.GetModelInfo()
}

// calculateBackoff calculates the backoff duration with exponential backoff and jitter
func (r *RetryingClient) calculateBackoff(attempt int) time.Duration {
	// Calculate exponential backoff
	backoff := float64(r.config.InitialBackoff) * math.Pow(r.config.BackoffFactor, float64(attempt-1))

	// Cap at max backoff
	if backoff > float64(r.config.MaxBackoff) {
		backoff = float64(r.config.MaxBackoff)
	}

	// Add jitter (±25%) - using math/rand/v2 which is safe for concurrent use
	jitter := backoff * 0.25 * (rand.Float64()*2 - 1)
	backoff += jitter

	// Ensure non-negative
	if backoff < 0 {
		backoff = 0
	}

	return time.Duration(backoff)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation/timeout - not retryable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	errStr := err.Error()

	// Check for rate limiting (429)
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		return true
	}

	// Check for server errors (5xx)
	if strings.Contains(errStr, "500") || strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") || strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "status 5") {
		return true
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Timeout errors are retryable
		if netErr.Timeout() {
			return true
		}
		// Temporary network errors are retryable
		if netErr.Temporary() {
			return true
		}
	}

	// Check for connection errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") {
		return true
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Check for HTTP-specific errors
	if strings.Contains(errStr, "EOF") || strings.Contains(errStr, "unexpected EOF") {
		return true
	}

	// Default: not retryable
	return false
}

// IsRetryableHTTPStatus checks if an HTTP status code is retryable
func IsRetryableHTTPStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || // 429
		statusCode == http.StatusInternalServerError || // 500
		statusCode == http.StatusBadGateway || // 502
		statusCode == http.StatusServiceUnavailable || // 503
		statusCode == http.StatusGatewayTimeout // 504
}
