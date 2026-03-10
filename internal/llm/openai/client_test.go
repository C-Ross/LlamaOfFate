package openai

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	config := Config{
		APIEndpoint: "https://test.azure.endpoint.com",
		APIKey:      "test-key",
		ModelName:   "Meta-Llama-3.1-405B-Instruct",
		Timeout:     60,
	}

	client := NewClient(config)

	assert.NotNil(t, client)
	assert.Equal(t, config.APIEndpoint, client.config.APIEndpoint)
	assert.Equal(t, config.APIKey, client.config.APIKey)
	assert.Equal(t, config.ModelName, client.config.ModelName)
	assert.NotNil(t, client.httpClient)

	modelInfo := client.GetModelInfo()
	assert.Equal(t, "Meta-Llama-3.1-405B-Instruct", modelInfo.Name)
	assert.Equal(t, "Azure ML", modelInfo.Provider)
	assert.Equal(t, 2048, modelInfo.MaxTokens)
	assert.Equal(t, 128000, modelInfo.ContextWindow)
}

func TestNewClientWithDefaults(t *testing.T) {
	config := Config{
		APIEndpoint: "https://test.azure.endpoint.com",
		APIKey:      "test-key",
	}

	client := NewClient(config)

	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)

	modelInfo := client.GetModelInfo()
	assert.Contains(t, modelInfo.Description, "hosted")
}

func TestGetMaxTokensForModel(t *testing.T) {
	tests := []struct {
		modelName   string
		expectedMax int
	}{
		{"Meta-Llama-3.1-405B-Instruct", 2048},
		{"Meta-Llama-3.1-70B-Instruct", 2048},
		{"Meta-Llama-3.1-8B-Instruct", 2048},
		{"unknown-model", 2048},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			result := getMaxTokensForModel(tt.modelName)
			assert.Equal(t, tt.expectedMax, result)
		})
	}
}

func TestChatCompletion(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/", r.URL.Path) // Changed from "/v1/chat/completions" since we use full endpoint URLs now
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		response := llm.CompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "Meta-Llama-3.1-405B-Instruct",
			Choices: []llm.CompletionResponseChoice{
				{
					Index: 0,
					Message: llm.Message{
						Role:    "assistant",
						Content: "Hello! How can I help you today?",
					},
					FinishReason: "stop",
				},
			},
			Usage: llm.CompletionUsage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{
			"id": "%s",
			"object": "%s",
			"created": %d,
			"model": "%s",
			"choices": [{
				"index": %d,
				"message": {
					"role": "%s",
					"content": "%s"
				},
				"finish_reason": "%s"
			}],
			"usage": {
				"prompt_tokens": %d,
				"completion_tokens": %d,
				"total_tokens": %d
			}
		}`, response.ID, response.Object, response.Created, response.Model,
			response.Choices[0].Index, response.Choices[0].Message.Role,
			response.Choices[0].Message.Content, response.Choices[0].FinishReason,
			response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens)
	}))
	defer server.Close()

	config := Config{
		APIEndpoint: server.URL,
		APIKey:      "test-key",
		ModelName:   "Meta-Llama-3.1-405B-Instruct",
	}

	client := NewClient(config)

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
		},
		MaxTokens:   50,
		Temperature: 0.7,
	}

	ctx := context.Background()
	response, err := client.ChatCompletion(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "test-id", response.ID)
	assert.Equal(t, "chat.completion", response.Object)
	assert.Equal(t, "Meta-Llama-3.1-405B-Instruct", response.Model)
	assert.Len(t, response.Choices, 1)
	assert.Equal(t, "assistant", response.Choices[0].Message.Role)
	assert.Equal(t, "Hello! How can I help you today?", response.Content())
	assert.Equal(t, "stop", response.Choices[0].FinishReason)
}

func TestChatCompletionError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	config := Config{
		APIEndpoint: server.URL,
		APIKey:      "invalid-key",
		ModelName:   "Meta-Llama-3.1-405B-Instruct",
	}

	client := NewClient(config)

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Hello!"},
		},
	}

	ctx := context.Background()
	response, err := client.ChatCompletion(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "401")
}

func TestChatCompletionStream(t *testing.T) {
	// Create a test server that returns streaming data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/", r.URL.Path) // Changed from "/v1/chat/completions" since we use full endpoint URLs now

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Simulate streaming response
		streamData := []string{
			`data: {"id":"test-id","object":"chat.completion.chunk","created":1234567890,"model":"Meta-Llama-3.1-405B-Instruct","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"test-id","object":"chat.completion.chunk","created":1234567890,"model":"Meta-Llama-3.1-405B-Instruct","choices":[{"index":0,"delta":{"content":" there!"},"finish_reason":null}]}`,
			`data: {"id":"test-id","object":"chat.completion.chunk","created":1234567890,"model":"Meta-Llama-3.1-405B-Instruct","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}

		for _, data := range streamData {
			_, _ = fmt.Fprintf(w, "%s\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	config := Config{
		APIEndpoint: server.URL,
		APIKey:      "test-key",
		ModelName:   "Meta-Llama-3.1-405B-Instruct",
	}

	client := NewClient(config)

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Hello!"},
		},
		MaxTokens:   50,
		Temperature: 0.7,
	}

	var chunks []llm.CompletionResponse
	handler := func(chunk llm.CompletionResponse) error {
		chunks = append(chunks, chunk)
		return nil
	}

	ctx := context.Background()
	err := client.ChatCompletionStream(ctx, req, handler)

	require.NoError(t, err)
	assert.Len(t, chunks, 3) // Should receive 3 chunks before [DONE]

	// Check first chunk
	assert.Equal(t, "test-id", chunks[0].ID)
	assert.Equal(t, "chat.completion.chunk", chunks[0].Object)
	assert.Len(t, chunks[0].Choices, 1)
	assert.NotNil(t, chunks[0].Choices[0].Delta)

	// Check that we got content in the chunks
	var content string
	for _, chunk := range chunks {
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
			content += chunk.Choices[0].Delta.Content
		}
	}
	assert.Equal(t, "Hello there!", content)
}

func TestGetContextWindowForModel(t *testing.T) {
	tests := []struct {
		modelName   string
		expectedMax int
	}{
		{"Llama-4-Maverick-17B-128E-Instruct-FP8", 1000000},
		{"Llama-4-Scout-17B-16E-Instruct", 10000000},
		{"Meta-Llama-3.1-405B-Instruct", 128000},
		{"Meta-Llama-3.1-70B-Instruct", 128000},
		{"Meta-Llama-3.1-8B-Instruct", 128000},
		{"Meta-Llama-3-70B-Instruct", 8192},
		{"unknown-model", 8192},
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			result := getContextWindowForModel(tt.modelName)
			assert.Equal(t, tt.expectedMax, result)
		})
	}
}

func TestLogTokenUsage_DebugLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	defer slog.SetDefault(slog.Default())

	client := NewClient(Config{
		APIEndpoint: "https://test.azure.endpoint.com",
		APIKey:      "test-key",
		ModelName:   "Meta-Llama-3.1-405B-Instruct",
	})

	usage := llm.CompletionUsage{
		PromptTokens:     500,
		CompletionTokens: 200,
		TotalTokens:      700,
	}

	client.logTokenUsage(usage, "Meta-Llama-3.1-405B-Instruct")

	output := buf.String()
	assert.Contains(t, output, "Token usage")
	assert.Contains(t, output, "prompt_tokens=500")
	assert.Contains(t, output, "completion_tokens=200")
	assert.Contains(t, output, "total_tokens=700")
	assert.NotContains(t, output, "approaching context window limit")
}

func TestLogTokenUsage_WarningNearLimit(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	defer slog.SetDefault(slog.Default())

	client := NewClient(Config{
		APIEndpoint: "https://test.azure.endpoint.com",
		APIKey:      "test-key",
		ModelName:   "Meta-Llama-3.1-405B-Instruct",
	})

	// 80% of 128000 = 102400; use something above that
	usage := llm.CompletionUsage{
		PromptTokens:     100000,
		CompletionTokens: 5000,
		TotalTokens:      105000,
	}

	client.logTokenUsage(usage, "Meta-Llama-3.1-405B-Instruct")

	output := buf.String()
	assert.Contains(t, output, "Token usage")
	assert.Contains(t, output, "approaching context window limit")
	assert.Contains(t, output, "context_window=128000")
}

func TestLogTokenUsage_NoLogOnZeroTokens(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	defer slog.SetDefault(slog.Default())

	client := NewClient(Config{
		APIEndpoint: "https://test.azure.endpoint.com",
		APIKey:      "test-key",
		ModelName:   "Meta-Llama-3.1-405B-Instruct",
	})

	usage := llm.CompletionUsage{}

	client.logTokenUsage(usage, "Meta-Llama-3.1-405B-Instruct")

	assert.Empty(t, buf.String())
}

func TestLogTokenUsage_NoWarningBelowThreshold(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	defer slog.SetDefault(slog.Default())

	client := NewClient(Config{
		APIEndpoint: "https://test.azure.endpoint.com",
		APIKey:      "test-key",
		ModelName:   "Meta-Llama-3.1-405B-Instruct",
	})

	// Well below 80% of 128000
	usage := llm.CompletionUsage{
		PromptTokens:     1000,
		CompletionTokens: 200,
		TotalTokens:      1200,
	}

	client.logTokenUsage(usage, "Meta-Llama-3.1-405B-Instruct")

	output := buf.String()
	assert.Contains(t, output, "Token usage")
	assert.NotContains(t, output, "approaching context window limit")
}

func TestInferProvider(t *testing.T) {
	tests := []struct {
		endpoint string
		expected string
	}{
		{"https://my-model.azure.inference.ai.azure.com", "Azure ML"},
		{"http://localhost:11434/v1/chat/completions", "Ollama"},
		{"http://127.0.0.1:11434/v1/chat/completions", "Ollama"},
		{"https://api.openai.com/v1/chat/completions", "OpenAI"},
		{"https://some-custom-api.example.com/v1/chat", "OpenAI-compatible"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := inferProvider(tt.endpoint)
			assert.Equal(t, tt.expected, result)
		})
	}
}
