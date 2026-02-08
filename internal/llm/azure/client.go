package azure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
)

// Config holds the configuration for Azure ML LLM client
type Config struct {
	APIEndpoint string `yaml:"api_endpoint"` // e.g., "https://your-endpoint.inference.ai.azure.com"
	APIKey      string `yaml:"api_key"`      // Your Azure ML API key
	ModelName   string `yaml:"model_name"`   // e.g., "Meta-Llama-3.1-405B-Instruct"
	Timeout     int    `yaml:"timeout"`      // Request timeout in seconds (default: 30)
}

// Client implements the LLMClient interface for Azure ML
type Client struct {
	config     Config
	httpClient *http.Client
	modelInfo  llm.ModelInfo
}

// NewClient creates a new Azure ML LLM client
func NewClient(config Config) *Client {
	timeout := time.Duration(config.Timeout) * time.Second
	if config.Timeout == 0 {
		timeout = 60 * time.Second // Increased default timeout
	}

	modelInfo := llm.ModelInfo{
		Name:          config.ModelName,
		Provider:      "Azure ML",
		MaxTokens:     getMaxTokensForModel(config.ModelName),
		ContextWindow: getContextWindowForModel(config.ModelName),
		Description:   fmt.Sprintf("Azure ML hosted %s", config.ModelName),
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		modelInfo: modelInfo,
	}
}

// ChatCompletion implements the LLMClient interface
func (c *Client) ChatCompletion(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	// Force streaming to false for non-streaming requests
	req.Stream = false

	// Set the model name if not already set
	if req.Model == "" {
		req.Model = c.config.ModelName
	}

	url := c.config.APIEndpoint

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	slog.Debug("Azure ChatCompletion request",
		slog.String("component", "azure_llm"),
		slog.String("endpoint", url),
		slog.String("model", req.Model),
		slog.Bool("stream", req.Stream),
		slog.Int("messages", len(req.Messages)),
		slog.Int("max_tokens", req.MaxTokens),
		slog.Float64("temperature", req.Temperature),
		slog.String("payload", string(reqBody)))

	start := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("Azure ChatCompletion request error",
			slog.String("component", "azure_llm"),
			slog.String("endpoint", url),
			slog.String("model", req.Model),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // error on close is not actionable

	headers := resp.Header.Clone()

	slog.Debug("Azure ChatCompletion headers",
		slog.String("component", "azure_llm"),
		slog.String("endpoint", url),
		slog.String("model", req.Model),
		slog.Int("status", resp.StatusCode),
		slog.Any("headers", headers))

	duration := time.Since(start)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Azure ChatCompletion read error",
			slog.String("component", "azure_llm"),
			slog.String("endpoint", url),
			slog.String("model", req.Model),
			slog.Duration("duration", duration),
			slog.Any("headers", headers),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("Azure ChatCompletion non-200 response",
			slog.String("component", "azure_llm"),
			slog.String("endpoint", url),
			slog.String("model", req.Model),
			slog.Int("status", resp.StatusCode),
			slog.Duration("duration", duration),
			slog.Any("headers", headers),
			slog.String("body", string(bodyBytes)))
		return nil, &llm.APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(bodyBytes),
		}
	}

	var response llm.CompletionResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		slog.Error("Azure ChatCompletion decode error",
			slog.String("component", "azure_llm"),
			slog.String("endpoint", url),
			slog.String("model", req.Model),
			slog.Duration("duration", duration),
			slog.Any("headers", headers),
			slog.Any("error", err),
			slog.String("raw_body", string(bodyBytes)))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	slog.Debug("Azure ChatCompletion response",
		slog.String("component", "azure_llm"),
		slog.String("endpoint", url),
		slog.String("model", req.Model),
		slog.Int("status", resp.StatusCode),
		slog.Duration("duration", duration),
		slog.Any("headers", headers),
		slog.Int("choices", len(response.Choices)),
		slog.String("raw_body", string(bodyBytes)))

	c.logTokenUsage(response.Usage, req.Model)

	response.CleanContent()

	if response.Content() == "" {
		return nil, llm.ErrEmptyResponse
	}

	return &response, nil
}

// ChatCompletionStream implements the LLMClient interface for streaming
func (c *Client) ChatCompletionStream(ctx context.Context, req llm.CompletionRequest, handler llm.StreamHandler) error {
	// Force streaming to true
	req.Stream = true

	// Set the model name if not already set
	if req.Model == "" {
		req.Model = c.config.ModelName
	}

	url := c.config.APIEndpoint

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	slog.Debug("Azure ChatCompletionStream request",
		slog.String("component", "azure_llm"),
		slog.String("endpoint", url),
		slog.String("model", req.Model),
		slog.Bool("stream", req.Stream),
		slog.Int("messages", len(req.Messages)),
		slog.String("payload", string(reqBody)))

	start := time.Now()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		slog.Error("Azure ChatCompletionStream request error",
			slog.String("component", "azure_llm"),
			slog.String("endpoint", url),
			slog.String("model", req.Model),
			slog.Any("error", err))
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // error on close is not actionable

	headers := resp.Header.Clone()

	slog.Debug("Azure ChatCompletionStream headers",
		slog.String("component", "azure_llm"),
		slog.String("endpoint", url),
		slog.String("model", req.Model),
		slog.Int("status", resp.StatusCode),
		slog.Any("headers", headers))

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		slog.Error("Azure ChatCompletionStream non-200 response",
			slog.String("component", "azure_llm"),
			slog.String("endpoint", url),
			slog.String("model", req.Model),
			slog.Int("status", resp.StatusCode),
			slog.Duration("duration", time.Since(start)),
			slog.Any("headers", headers),
			slog.String("body", string(bodyBytes)))
		return &llm.APIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(bodyBytes),
		}
	}

	slog.Debug("Azure ChatCompletionStream response started",
		slog.String("component", "azure_llm"),
		slog.String("endpoint", url),
		slog.String("model", req.Model),
		slog.Duration("handshake_duration", time.Since(start)),
		slog.Any("headers", headers))

	return c.processStreamingResponse(resp.Body, handler)
}

// GetModelInfo implements the LLMClient interface
func (c *Client) GetModelInfo() llm.ModelInfo {
	return c.modelInfo
}

// setHeaders sets the required headers for Azure ML API requests
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	// Handle different Azure authentication formats
	// If the key already starts with "Bearer " or "api-key ", use it as-is
	// Otherwise, assume it's a raw key and add "Bearer " prefix
	apiKey := c.config.APIKey
	if !strings.HasPrefix(apiKey, "Bearer ") && !strings.HasPrefix(apiKey, "api-key ") {
		apiKey = "Bearer " + apiKey
	}
	req.Header.Set("Authorization", apiKey)
}

// processStreamingResponse processes the streaming response from Azure ML
func (c *Client) processStreamingResponse(body io.Reader, handler llm.StreamHandler) error {
	scanner := bufio.NewScanner(body)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		slog.Debug("Azure ChatCompletionStream line",
			slog.String("component", "azure_llm"),
			slog.String("line", line))

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for the end of stream
		if line == "data: [DONE]" {
			slog.Debug("Azure ChatCompletionStream complete",
				slog.String("component", "azure_llm"))
			break
		}

		// Parse the data line
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")

			var chunk llm.CompletionResponse
			if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
				slog.Warn("Azure ChatCompletionStream decode failed",
					slog.String("component", "azure_llm"),
					slog.String("raw", jsonData),
					slog.Any("error", err))
				// Log the error but continue processing
				continue
			}

			slog.Debug("Azure ChatCompletionStream chunk",
				slog.String("component", "azure_llm"),
				slog.Int("choices", len(chunk.Choices)),
				slog.Any("chunk", chunk))

			if err := handler(chunk); err != nil {
				slog.Error("Azure ChatCompletionStream handler error",
					slog.String("component", "azure_llm"),
					slog.Any("error", err))
				return fmt.Errorf("handler error: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Azure ChatCompletionStream scanner error",
			slog.String("component", "azure_llm"),
			slog.Any("error", err))
		return fmt.Errorf("error reading stream: %w", err)
	}

	return nil
}

// getMaxTokensForModel returns the maximum tokens for different Llama models
func getMaxTokensForModel(modelName string) int {
	switch {
	case strings.Contains(modelName, "405B"):
		return 2048
	case strings.Contains(modelName, "70B"):
		return 2048
	case strings.Contains(modelName, "8B"):
		return 2048
	default:
		return 2048 // Default for Llama models
	}
}

// getContextWindowForModel returns the context window size for different Llama models.
// Model specs: https://huggingface.co/meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8#model-information
func getContextWindowForModel(modelName string) int {
	switch {
	case strings.Contains(modelName, "Maverick"):
		return 1000000 // Llama 4 Maverick supports 1M context
	case strings.Contains(modelName, "Scout"):
		return 10000000 // Llama 4 Scout supports 10M context
	case strings.Contains(modelName, "Llama-3.1"):
		return 128000 // Llama 3.1 supports 128k context
	case strings.Contains(modelName, "Llama-3"):
		return 8192 // Llama 3.0 supports 8k context
	default:
		return 8192 // Conservative default
	}
}

const tokenUsageWarningThreshold = 0.8 // Warn at 80% of context window

// logTokenUsage logs token usage at debug level and warns if approaching the context window limit
func (c *Client) logTokenUsage(usage llm.CompletionUsage, model string) {
	if usage.TotalTokens == 0 {
		return
	}

	slog.Debug("Token usage",
		slog.String("component", "azure_llm"),
		slog.String("model", model),
		slog.Int("prompt_tokens", usage.PromptTokens),
		slog.Int("completion_tokens", usage.CompletionTokens),
		slog.Int("total_tokens", usage.TotalTokens))

	contextWindow := c.modelInfo.ContextWindow
	if contextWindow <= 0 {
		return
	}

	usageRatio := float64(usage.TotalTokens) / float64(contextWindow)
	if usageRatio >= tokenUsageWarningThreshold {
		slog.Warn("Token usage approaching context window limit",
			slog.String("component", "azure_llm"),
			slog.String("model", model),
			slog.Int("total_tokens", usage.TotalTokens),
			slog.Int("context_window", contextWindow),
			slog.Float64("usage_percent", usageRatio*100))
	}
}
