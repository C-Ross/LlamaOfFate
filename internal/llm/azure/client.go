package azure

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		Name:        config.ModelName,
		Provider:    "Azure ML",
		MaxTokens:   getMaxTokensForModel(config.ModelName),
		Description: fmt.Sprintf("Azure ML hosted %s", config.ModelName),
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response llm.CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

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
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Check for the end of stream
		if line == "data: [DONE]" {
			break
		}
		
		// Parse the data line
		if strings.HasPrefix(line, "data: ") {
			jsonData := strings.TrimPrefix(line, "data: ")
			
			var chunk llm.CompletionResponse
			if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
				// Log the error but continue processing
				continue
			}
			
			if err := handler(chunk); err != nil {
				return fmt.Errorf("handler error: %w", err)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
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
