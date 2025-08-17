package llm

import "context"

// Message represents a single message in a conversation
type Message struct {
	Role    string `json:"role"`    // "system", "user", or "assistant"
	Content string `json:"content"` // The message content
}

// CompletionRequest represents a request to generate text completion
type CompletionRequest struct {
	Messages          []Message `json:"messages"`
	MaxTokens         int       `json:"max_tokens,omitempty"`
	Temperature       float64   `json:"temperature,omitempty"`
	TopP              float64   `json:"top_p,omitempty"`
	PresencePenalty   float64   `json:"presence_penalty,omitempty"`
	FrequencyPenalty  float64   `json:"frequency_penalty,omitempty"`
	Model             string    `json:"model,omitempty"`
	Stream            bool      `json:"stream,omitempty"`
}

// CompletionResponse represents the response from a completion request
type CompletionResponse struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []CompletionResponseChoice `json:"choices"`
	Usage   CompletionUsage          `json:"usage,omitempty"`
}

// CompletionResponseChoice represents a single completion choice
type CompletionResponseChoice struct {
	Index        int                    `json:"index"`
	Message      Message                `json:"message"`
	FinishReason string                 `json:"finish_reason"`
	Delta        *Message               `json:"delta,omitempty"` // For streaming responses
}

// CompletionUsage represents token usage information
type CompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamHandler is a function type for handling streaming responses
type StreamHandler func(chunk CompletionResponse) error

// LLMClient defines the interface for Large Language Model clients
type LLMClient interface {
	// ChatCompletion sends a chat completion request and returns the response
	ChatCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	
	// ChatCompletionStream sends a streaming chat completion request
	// The handler function is called for each chunk received
	ChatCompletionStream(ctx context.Context, req CompletionRequest, handler StreamHandler) error
	
	// GetModelInfo returns information about the model being used
	GetModelInfo() ModelInfo
}

// ModelInfo contains information about the LLM model
type ModelInfo struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	MaxTokens   int    `json:"max_tokens"`
	Description string `json:"description"`
}
