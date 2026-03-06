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

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/llm/azure"
	"github.com/C-Ross/LlamaOfFate/internal/logging"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
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

	factory := func(ctx context.Context, gameID string, setup *web.GameSetup) (engine.GameSessionManager, error) {
		return newGameSession(ctx, llmClient, gameID, setup)
	}

	setupCfg := web.SetupConfig{
		Presets:     allPresetMeta(),
		AllowCustom: true,
	}

	handler := web.NewHandler(factory, setupCfg, slog.Default())

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

func newGameSession(ctx context.Context, llmClient llm.LLMClient, gameID string, setup *web.GameSetup) (engine.GameSessionManager, error) {
	saver := newSaver(gameID)

	// If no setup is provided, try to resume a saved game.
	// Return nil (no driver) when no save exists — the caller enters setup flow.
	if setup == nil {
		// Attempt to load a saved game. If the file doesn't exist, saver.Load
		// returns (nil, nil) and we signal "no driver" for the setup flow.
		// If the file exists but is corrupt/incompatible, return a SaveCorruptError
		// so the handler can notify the user before entering setup.
		state, loadErr := saver.Load()
		if loadErr != nil {
			return nil, &engine.SaveCorruptError{Cause: loadErr}
		}
		if state == nil || state.Scene.CurrentScene == nil {
			return nil, nil // no saved game — enter setup flow
		}

		sl := createSessionLogger("resume", gameID)
		gameEngine, err := engine.NewWithLLM(llmClient, sl)
		if err != nil {
			return nil, fmt.Errorf("create engine: %w", err)
		}
		gm := engine.NewGameManager(gameEngine, sl)
		gm.SetSaver(saver)

		// Save exists: provide the player from state; Start() will hydrate from state.
		placeholder := state.Scenario.Player
		if placeholder == nil {
			return nil, nil
		}
		gm.SetPlayer(placeholder)
		if state.Scenario.Scenario != nil {
			gm.SetScenario(state.Scenario.Scenario)
		}
		return gm, nil
	}

	// Setup provided — create fresh game with chosen scenario + player.
	var scenario *scene.Scenario
	var player *core.Character

	if setup.PresetID != "" {
		var err error
		scenario, player, err = lookupPreset(setup.PresetID)
		if err != nil {
			return nil, err
		}
	} else if setup.Custom != nil {
		player = buildCustomPlayer(
			setup.Custom.Name,
			setup.Custom.HighConcept,
			setup.Custom.Trouble,
			setup.Custom.Genre,
		)
		var err error
		scenario, err = generateScenario(ctx, llmClient, setup.Custom)
		if err != nil {
			return nil, fmt.Errorf("scenario generation: %w", err)
		}
	} else {
		return nil, fmt.Errorf("setup message has neither presetId nor custom data")
	}

	sl := createSessionLogger(scenario.Genre, player.Name)
	gameEngine, err := engine.NewWithLLM(llmClient, sl)
	if err != nil {
		return nil, fmt.Errorf("create engine: %w", err)
	}

	gm := engine.NewGameManager(gameEngine, sl)
	gm.SetPlayer(player)
	gm.SetScenario(scenario)
	gm.SetSaver(saver)

	return gm, nil
}

// createSessionLogger builds a session logger for a web game session.
// On failure it logs a warning and returns a NullLogger so the game can proceed.
func createSessionLogger(label, name string) session.SessionLogger {
	logPath, err := session.GenerateLogPath("session", []string{label, name}, 20)
	if err != nil {
		slog.Warn("session logging disabled: failed to generate log path", "error", err)
		return session.NullLogger{}
	}
	logger, err := session.NewLogger(logPath)
	if err != nil {
		slog.Warn("session logging disabled: failed to create logger", "error", err)
		return session.NullLogger{}
	}
	slog.Info("Session log started", "path", logPath)
	return logger
}

// scenarioGenerationTimeout is the maximum time allowed for LLM scenario generation.
const scenarioGenerationTimeout = 30 * time.Second

// generateScenario uses the LLM to create a custom scenario from the player's
// character data and chosen genre.
func generateScenario(ctx context.Context, client llm.LLMClient, custom *web.CustomSetup) (*scene.Scenario, error) {
	genCtx, cancel := context.WithTimeout(ctx, scenarioGenerationTimeout)
	defer cancel()

	data := prompt.ScenarioGenerationData{
		PlayerName:        custom.Name,
		PlayerHighConcept: custom.HighConcept,
		PlayerTrouble:     custom.Trouble,
		Genre:             custom.Genre,
	}

	promptText, err := prompt.RenderScenarioGeneration(data)
	if err != nil {
		return nil, fmt.Errorf("render prompt: %w", err)
	}

	slog.Info("generating scenario via LLM",
		"genre", custom.Genre,
		"player", custom.Name,
	)

	rawResponse, err := llm.SimpleCompletion(genCtx, client, promptText, 500, 0.8)
	if err != nil {
		return nil, fmt.Errorf("LLM completion: %w", err)
	}

	scenario, err := prompt.ParseScenario(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("parse scenario: %w", err)
	}

	slog.Info("scenario generated",
		"title", scenario.Title,
		"genre", scenario.Genre,
	)
	return scenario, nil
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
