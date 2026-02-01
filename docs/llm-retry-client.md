# LLM Retry Client

## Overview

The retry client provides automatic retry logic for transient LLM API failures. It transparently wraps any `LLMClient` implementation and handles:

- Network connectivity problems
- Rate limiting (429 responses)
- Service unavailability (5xx errors)
- Request timeouts

## Features

- **Exponential backoff with jitter**: Prevents thundering herd problem
- **Configurable retry behavior**: Customize max attempts, backoff parameters
- **Smart error classification**: Only retries on transient errors
- **Transparent to callers**: Drop-in replacement for any `LLMClient`
- **Comprehensive logging**: Track retry attempts for observability

## Usage

### Basic Usage with Default Configuration

```go
import (
    "github.com/C-Ross/LlamaOfFate/internal/llm"
    "github.com/C-Ross/LlamaOfFate/internal/llm/azure"
)

// Create your base LLM client (e.g., Azure)
config, err := azure.LoadConfig("configs/azure-llm.yaml")
if err != nil {
    log.Fatalf("Failed to load Azure config: %v", err)
}
azureClient := azure.NewClient(*config)

// Wrap it with retry logic using default config (3 attempts, exponential backoff)
retryClient := llm.NewRetryingClient(azureClient, llm.DefaultRetryConfig())

// Use retryClient like any other LLMClient - retries are automatic!
ctx := context.Background()
req := llm.CompletionRequest{
    Messages: []llm.Message{
        {Role: "system", Content: "You are a helpful assistant."},
        {Role: "user", Content: "Hello!"},
    },
    MaxTokens:   100,
    Temperature: 0.7,
}

response, err := retryClient.ChatCompletion(ctx, req)
if err != nil {
    log.Printf("Request failed after retries: %v", err)
    return
}

fmt.Println(response.Choices[0].Message.Content)
```

### Custom Retry Configuration

```go
import (
    "time"
    "github.com/C-Ross/LlamaOfFate/internal/llm"
    "github.com/C-Ross/LlamaOfFate/internal/llm/azure"
)

// Create custom retry configuration
config := llm.RetryConfig{
    MaxAttempts:    5,                      // Try up to 5 times
    InitialBackoff: 200 * time.Millisecond, // Start with 200ms
    MaxBackoff:     30 * time.Second,       // Cap at 30s
    BackoffFactor:  2.5,                    // Increase by 2.5x each time
}

azureClient := azure.NewClient(*azureConfig)
retryClient := llm.NewRetryingClient(azureClient, config)

// Now retryClient will use your custom retry behavior
```

### Using with Game Engine

```go
// Load Azure LLM configuration
azureConfig, err := azure.LoadConfig("configs/azure-llm.yaml")
if err != nil {
    log.Fatalf("Failed to load Azure config: %v", err)
}

// Create Azure client
azureClient := azure.NewClient(*azureConfig)

// Wrap with retry logic
retryClient := llm.NewRetryingClient(azureClient, llm.DefaultRetryConfig())

// Create the game engine with the retry-enabled LLM client
gameEngine, err := engine.NewWithLLM(retryClient)
if err != nil {
    log.Fatalf("Failed to create engine: %v", err)
}

// The engine will now automatically retry on transient failures!
```

## Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `MaxAttempts` | int | 3 | Maximum number of retry attempts |
| `InitialBackoff` | time.Duration | 100ms | Initial backoff duration |
| `MaxBackoff` | time.Duration | 10s | Maximum backoff duration |
| `BackoffFactor` | float64 | 2.0 | Multiplier for exponential backoff |

## Retryable vs Non-Retryable Errors

### Retryable Errors (will retry)
- HTTP 429 (Too Many Requests / Rate Limit)
- HTTP 500 (Internal Server Error)
- HTTP 502 (Bad Gateway)
- HTTP 503 (Service Unavailable)
- HTTP 504 (Gateway Timeout)
- Network errors (connection refused, connection reset, etc.)
- DNS errors
- Timeout errors (from network layer, not context)
- EOF errors

### Non-Retryable Errors (fail immediately)
- HTTP 4xx (except 429) - client errors like 400, 401, 403, 404
- Context cancellation (context.Canceled)
- Context timeout (context.DeadlineExceeded)
- Validation errors

## Logging

The retry client logs retry attempts at INFO level and final failures at ERROR level:

```
INFO  Retrying LLM request after failure component=retry_client attempt=1 max_attempts=3 backoff=125ms error="API request failed with status 503"
INFO  Retrying LLM request after failure component=retry_client attempt=2 max_attempts=3 backoff=250ms error="API request failed with status 503"
ERROR All retry attempts exhausted component=retry_client attempts=3 error="API request failed with status 503"
```

## Backoff Strategy

The retry client uses **exponential backoff with jitter**:

1. **Exponential backoff**: Each retry waits longer than the previous one
   - Attempt 1: InitialBackoff
   - Attempt 2: InitialBackoff × BackoffFactor
   - Attempt 3: InitialBackoff × BackoffFactor²
   - ...capped at MaxBackoff

2. **Jitter**: Random variation (±25%) prevents multiple clients from retrying simultaneously

Example with defaults (InitialBackoff=100ms, BackoffFactor=2.0):
- Attempt 1 → Wait ~100ms (±25ms jitter)
- Attempt 2 → Wait ~200ms (±50ms jitter)
- Attempt 3 → Would wait ~400ms but request succeeds or fails

## Context Handling

The retry client respects context cancellation and timeouts:

```go
// Set a deadline for the entire operation (including retries)
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := retryClient.ChatCompletion(ctx, req)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Operation timed out")
    }
}
```

If the context is cancelled or times out:
- Current request is cancelled immediately
- No more retries are attempted
- Error is returned to caller

## Testing

To test with simulated failures:

```go
// Create a mock client that fails the first 2 attempts
mockClient := &llm.MockLLMClient{
    ChatCompletionFunc: func(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
        attempts++
        if attempts < 3 {
            return nil, fmt.Errorf("API request failed with status 503")
        }
        return &llm.CompletionResponse{...}, nil
    },
}

config := llm.RetryConfig{
    MaxAttempts:    3,
    InitialBackoff: 10 * time.Millisecond,
    MaxBackoff:     100 * time.Millisecond,
    BackoffFactor:  2.0,
}
retryClient := llm.NewRetryingClient(mockClient, config)

// This will succeed on the 3rd attempt
response, err := retryClient.ChatCompletion(ctx, req)
```

## Streaming Support

The retry client also supports streaming with the same retry logic:

```go
handler := func(chunk llm.CompletionResponse) error {
    // Process streaming chunk
    fmt.Print(chunk.Choices[0].Delta.Content)
    return nil
}

err := retryClient.ChatCompletionStream(ctx, req, handler)
if err != nil {
    log.Printf("Stream failed after retries: %v", err)
}
```

**Note**: When retrying a stream, the entire stream restarts from the beginning. The handler will be called again for all chunks.
