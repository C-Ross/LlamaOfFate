package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/mark3labs/mcp-go/server"

	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/mcpserver"
)

func main() {
	logging.SetupDefaultLogging()

	configPath := "configs/azure-llm.yaml"
	if c := os.Getenv("LLM_CONFIG"); c != "" {
		configPath = c
	}

	configRoot := "configs"
	if c := os.Getenv("CONFIG_ROOT"); c != "" {
		configRoot = c
	}

	llmClient, err := initLLMClient(configPath)
	if err != nil {
		slog.Warn("LLM client unavailable - game tools will fail to start games", "error", err)
	}

	gs, err := mcpserver.New(llmClient, configRoot)
	if err != nil {
		slog.Error("failed to create MCP game server", "error", err)
		os.Exit(1)
	}

	// Run as stdio MCP server
	stdio := server.NewStdioServer(gs.MCPServer())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	defer func() {
		if closeErr := gs.Close(); closeErr != nil {
			slog.Warn("failed to close game server", "error", closeErr)
		}
	}()

	slog.Info("LlamaOfFate MCP server starting on stdio")
	if err := stdio.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func initLLMClient(configPath string) (llm.LLMClient, error) {
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("LLM config not found at %s", configPath)
	}
	config, err := azure.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load LLM config: %w", err)
	}
	azureClient := azure.NewClient(*config)
	retryClient := llm.NewRetryingClient(azureClient, llm.DefaultRetryConfig())
	slog.Info("LLM integration enabled", "model", config.ModelName)
	return retryClient, nil
}
