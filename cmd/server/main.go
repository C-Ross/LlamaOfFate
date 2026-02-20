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

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
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

	factory := func(gameID string, setup *web.GameSetup) (engine.GameSessionManager, error) {
		return newGameSession(llmClient, gameID, setup)
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

func newGameSession(llmClient llm.LLMClient, gameID string, setup *web.GameSetup) (engine.GameSessionManager, error) {
	saver := newSaver(gameID)

	// If no setup is provided, try to resume a saved game.
	// Return nil (no driver) when no save exists — the caller enters setup flow.
	if setup == nil {
		gameEngine, err := engine.NewWithLLM(llmClient)
		if err != nil {
			return nil, fmt.Errorf("create engine: %w", err)
		}
		gm := engine.NewGameManager(gameEngine)
		gm.SetSaver(saver)

		// Attempt to load a saved game. If the save file exists, set a
		// placeholder player (Start will overwrite from the save). Otherwise
		// signal "no driver" so the handler triggers the setup flow.
		state, loadErr := saver.Load()
		if loadErr != nil || state == nil || state.Scene.CurrentScene == nil {
			return nil, nil // no saved game — enter setup flow
		}
		// Save exists: provide a placeholder player; Start() will hydrate from state.
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
	var player *character.Character

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
		// For Phase 2 we'll generate the scenario via LLM here.
		// For now, fall back to a generic scenario from the genre.
		scenario = genreScenarioFallback(setup.Custom.Genre)
	} else {
		return nil, fmt.Errorf("setup message has neither presetId nor custom data")
	}

	gameEngine, err := engine.NewWithLLM(llmClient)
	if err != nil {
		return nil, fmt.Errorf("create engine: %w", err)
	}

	gm := engine.NewGameManager(gameEngine)
	gm.SetPlayer(player)
	gm.SetScenario(scenario)
	gm.SetSaver(saver)

	return gm, nil
}

// genreScenarioFallback returns a preset scenario that matches the given genre,
// or the saloon scenario as a fallback. This is a temporary bridge until Phase 2
// wires up LLM scenario generation.
func genreScenarioFallback(genre string) *scene.Scenario {
	switch genre {
	case "Cyberpunk":
		return scene.PredefinedScenario("heist")
	case "Fantasy":
		return scene.PredefinedScenario("tower")
	default:
		return scene.PredefinedScenario("saloon")
	}
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
