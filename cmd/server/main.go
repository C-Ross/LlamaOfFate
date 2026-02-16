package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/storage"
	"github.com/C-Ross/LlamaOfFate/internal/ui/web"
)

func main() {
	logging.SetupDefaultLogging()

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	configPath := "configs/azure-llm.yaml"
	if c := os.Getenv("LLM_CONFIG"); c != "" {
		configPath = c
	}

	llmClient, err := initLLMClient(configPath)
	if err != nil {
		log.Fatalf("LLM init failed: %v", err)
	}

	factory := func(gameID string) (engine.GameSessionManager, error) {
		return newGameSession(llmClient, gameID)
	}

	handler := web.NewHandler(factory, slog.Default())

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	slog.Info("server stopped")
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

func newGameSession(llmClient llm.LLMClient, gameID string) (engine.GameSessionManager, error) {
	gameEngine, err := engine.NewWithLLM(llmClient)
	if err != nil {
		return nil, fmt.Errorf("create engine: %w", err)
	}

	scenario := defaultScenario()
	player := defaultPlayer()

	gm := engine.NewGameManager(gameEngine)
	gm.SetPlayer(player)
	gm.SetScenario(scenario)
	gm.SetSaver(newSaver(gameID))

	return gm, nil
}

// newSaver creates a YAMLSaver that stores saves in a per-game subdirectory
// under the user's home directory.
func newSaver(gameID string) *storage.YAMLSaver {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Warn("could not determine home directory for saves, using current dir", "error", err)
		home = "."
	}
	saveDir := filepath.Join(home, ".llamaoffate", "saves", gameID)
	saver := storage.NewYAMLSaver(saveDir)
	slog.Info("save file location", "path", saver.Path(), "game_id", gameID)
	return saver
}
