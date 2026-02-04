package llm

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/cenkalti/backoff/v5"
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
	var attempt int

	operation := func() (*CompletionResponse, error) {
		attempt++

		resp, err := r.client.ChatCompletion(ctx, req)
		if err == nil {
			return resp, nil
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			slog.Debug("Non-retryable error encountered",
				slog.String("component", "retry_client"),
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()))
			return nil, backoff.Permanent(err)
		}

		slog.Info("Retrying LLM request after failure",
			slog.String("component", "retry_client"),
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", r.config.MaxAttempts),
			slog.String("error", err.Error()))

		return nil, err
	}

	response, err := backoff.Retry(ctx, operation, r.retryOptions()...)

	if err != nil {
		slog.Error("All retry attempts exhausted",
			slog.String("component", "retry_client"),
			slog.Int("attempts", attempt),
			slog.String("error", err.Error()))
	}

	return response, err
}

// ChatCompletionStream implements the LLMClient interface with retry logic for streaming
func (r *RetryingClient) ChatCompletionStream(ctx context.Context, req CompletionRequest, handler StreamHandler) error {
	var attempt int

	operation := func() (struct{}, error) {
		attempt++

		err := r.client.ChatCompletionStream(ctx, req, handler)
		if err == nil {
			return struct{}{}, nil
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			slog.Debug("Non-retryable error encountered in stream",
				slog.String("component", "retry_client"),
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()))
			return struct{}{}, backoff.Permanent(err)
		}

		slog.Info("Retrying LLM stream request after failure",
			slog.String("component", "retry_client"),
			slog.Int("attempt", attempt),
			slog.Int("max_attempts", r.config.MaxAttempts),
			slog.String("error", err.Error()))

		return struct{}{}, err
	}

	_, err := backoff.Retry(ctx, operation, r.retryOptions()...)

	if err != nil {
		slog.Error("All retry attempts exhausted for stream",
			slog.String("component", "retry_client"),
			slog.Int("attempts", attempt),
			slog.String("error", err.Error()))
	}

	return err
}

// GetModelInfo implements the LLMClient interface (no retry needed for this)
func (r *RetryingClient) GetModelInfo() ModelInfo {
	return r.client.GetModelInfo()
}

// retryOptions creates the retry options for backoff.Retry
func (r *RetryingClient) retryOptions() []backoff.RetryOption {
	// Create exponential backoff
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.InitialInterval = r.config.InitialBackoff
	expBackoff.MaxInterval = r.config.MaxBackoff
	expBackoff.Multiplier = r.config.BackoffFactor

	return []backoff.RetryOption{
		backoff.WithBackOff(expBackoff),
		backoff.WithMaxTries(uint(r.config.MaxAttempts)),
	}
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

	// Check for typed APIError
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRetryable()
	}

	// Check for net.OpError (network operation errors)
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true // Network operation errors are retryable
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Timeout errors are retryable
		if netErr.Timeout() {
			return true
		}
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Default: not retryable
	return false
}
