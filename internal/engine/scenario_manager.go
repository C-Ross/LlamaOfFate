package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

const (
	componentScenarioManager = "scenario_manager"
)

// ScenarioSettings holds configuration for a scenario
type ScenarioSettings struct {
	Genre          string // e.g., "Western", "Cyberpunk", "Fantasy"
	SettingContext string // World/setting description for LLM context
}

// ScenarioManager orchestrates multi-scene gameplay
type ScenarioManager struct {
	engine        *Engine
	player        *character.Character
	ui            UI
	sessionLogger *session.Logger
	settings      ScenarioSettings
	initialScene  *scene.Scene     // Optional pre-configured starting scene
	initialNPCs   []*character.Character // NPCs for initial scene
}

// NewScenarioManager creates a new scenario manager
func NewScenarioManager(engine *Engine, player *character.Character) *ScenarioManager {
	return &ScenarioManager{
		engine: engine,
		player: player,
	}
}

// SetUI sets the UI for the scenario manager
func (m *ScenarioManager) SetUI(ui UI) {
	m.ui = ui
}

// SetSessionLogger sets the session logger
func (m *ScenarioManager) SetSessionLogger(logger *session.Logger) {
	m.sessionLogger = logger
}

// SetSettings configures the scenario settings
func (m *ScenarioManager) SetSettings(settings ScenarioSettings) {
	m.settings = settings
}

// SetInitialScene sets a pre-configured starting scene
func (m *ScenarioManager) SetInitialScene(s *scene.Scene, npcs []*character.Character) {
	m.initialScene = s
	m.initialNPCs = npcs
}

// Run executes the scenario loop, transitioning between scenes until quit or player taken out
func (m *ScenarioManager) Run(ctx context.Context) error {
	if m.engine == nil {
		return fmt.Errorf("engine is required")
	}
	if m.engine.llmClient == nil {
		return fmt.Errorf("LLM client is required")
	}
	if m.ui == nil {
		return fmt.Errorf("UI is required")
	}
	if m.player == nil {
		return fmt.Errorf("player character is required")
	}

	// Register player with engine
	m.engine.AddCharacter(m.player)

	// Get or create initial scene
	currentScene, err := m.getInitialScene(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial scene: %w", err)
	}

	// Main scenario loop
	transitionHint := ""
	for {
		// Set up the scene manager for this scene
		sceneManager := m.engine.GetSceneManager()
		sceneManager.SetUI(m.ui)
		if m.sessionLogger != nil {
			sceneManager.SetSessionLogger(m.sessionLogger)
		}
		sceneManager.SetExitOnSceneTransition(true)

		// Start the scene
		if err := sceneManager.StartScene(currentScene, m.player); err != nil {
			return fmt.Errorf("failed to start scene: %w", err)
		}

		// Run the scene loop
		result, err := sceneManager.RunSceneLoop(ctx)
		if err != nil {
			return fmt.Errorf("scene loop error: %w", err)
		}

		// Log the scene end
		if m.sessionLogger != nil {
			m.sessionLogger.Log("scene_end", map[string]any{
				"reason":          result.Reason,
				"transition_hint": result.TransitionHint,
				"taken_out_chars": result.TakenOutChars,
			})
		}

		slog.Info("Scene ended",
			"component", componentScenarioManager,
			"reason", result.Reason,
			"transition_hint", result.TransitionHint,
		)

		// Handle the scene result
		switch result.Reason {
		case SceneEndQuit:
			// Player chose to quit
			return nil

		case SceneEndPlayerTakenOut:
			// Player was taken out - could continue or end based on context
			if result.TransitionHint != "" {
				// Player taken out but story continues (captured, etc.)
				transitionHint = result.TransitionHint
			} else {
				// Game over
				return nil
			}

		case SceneEndTransition:
			// Normal scene transition
			transitionHint = result.TransitionHint
		}

		// Generate the next scene
		currentScene, err = m.generateNextScene(ctx, transitionHint)
		if err != nil {
			slog.Error("Failed to generate next scene",
				"component", componentScenarioManager,
				"error", err,
			)
			return fmt.Errorf("failed to generate next scene: %w", err)
		}
	}
}

// getInitialScene returns the initial scene, either pre-configured or generated
func (m *ScenarioManager) getInitialScene(ctx context.Context) (*scene.Scene, error) {
	if m.initialScene != nil {
		// Use the pre-configured scene
		// Register NPCs with engine and add to scene
		for _, npc := range m.initialNPCs {
			m.engine.AddCharacter(npc)
			m.initialScene.AddCharacter(npc.ID)
		}
		m.initialScene.AddCharacter(m.player.ID)
		return m.initialScene, nil
	}

	// Generate an initial scene
	return m.generateNextScene(ctx, "")
}

// generateNextScene uses the LLM to create a new scene based on transition context
func (m *ScenarioManager) generateNextScene(ctx context.Context, transitionHint string) (*scene.Scene, error) {
	// Gather player aspects
	var playerAspects []string
	for _, a := range m.player.Aspects.GetAll() {
		if a != m.player.Aspects.HighConcept && a != m.player.Aspects.Trouble {
			playerAspects = append(playerAspects, a)
		}
	}

	data := SceneGenerationData{
		TransitionHint:    transitionHint,
		SettingContext:    m.settings.SettingContext,
		PlayerName:        m.player.Name,
		PlayerHighConcept: m.player.Aspects.HighConcept,
		PlayerTrouble:     m.player.Aspects.Trouble,
		PlayerAspects:     playerAspects,
	}

	prompt, err := RenderSceneGeneration(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render scene generation prompt: %w", err)
	}

	resp, err := m.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   500,
		Temperature: 0.8, // Higher creativity for scene generation
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty LLM response")
	}

	// Parse the generated scene
	generated, err := m.parseGeneratedScene(resp.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated scene: %w", err)
	}

	// Create the scene object
	sceneID := fmt.Sprintf("scene_%d", len(m.engine.GetAllCharacters())) // Simple ID generation
	newScene := scene.NewScene(sceneID, generated.SceneName, generated.Description)

	// Add situation aspects
	for _, aspectName := range generated.SituationAspects {
		newScene.AddSituationAspect(scene.SituationAspect{
			Aspect:   aspectName,
			Duration: "scene",
		})
	}

	// Add player to scene
	newScene.AddCharacter(m.player.ID)

	// Create and register NPCs
	for i, npcData := range generated.NPCs {
		npcID := fmt.Sprintf("%s_npc_%d", sceneID, i)
		npc := character.NewCharacter(npcID, npcData.Name)
		npc.Aspects.HighConcept = npcData.HighConcept

		// Set NPC type based on disposition
		switch npcData.Disposition {
		case "hostile":
			npc.CharacterType = character.CharacterTypeNamelessGood
		case "friendly", "neutral":
			npc.CharacterType = character.CharacterTypeSupportingNPC
		default:
			npc.CharacterType = character.CharacterTypeSupportingNPC
		}

		m.engine.AddCharacter(npc)
		newScene.AddCharacter(npc.ID)
	}

	// Log the generated scene
	if m.sessionLogger != nil {
		m.sessionLogger.Log("scene_generated", map[string]any{
			"scene_id":          sceneID,
			"scene_name":        generated.SceneName,
			"description":       generated.Description,
			"situation_aspects": generated.SituationAspects,
			"npc_count":         len(generated.NPCs),
		})
	}

	slog.Info("Generated new scene",
		"component", componentScenarioManager,
		"scene_name", generated.SceneName,
		"aspects", len(generated.SituationAspects),
		"npcs", len(generated.NPCs),
	)

	return newScene, nil
}

// parseGeneratedScene parses the LLM response into a GeneratedScene
func (m *ScenarioManager) parseGeneratedScene(content string) (*GeneratedScene, error) {
	// Clean the response - extract JSON from potential markdown code blocks
	cleaned := cleanJSONResponse(content)

	var generated GeneratedScene
	if err := json.Unmarshal([]byte(cleaned), &generated); err != nil {
		// Try to extract just the first JSON object if there's extra content
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &generated); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	// Validate required fields
	if generated.SceneName == "" {
		return nil, fmt.Errorf("missing scene_name")
	}
	if generated.Description == "" {
		return nil, fmt.Errorf("missing description")
	}

	return &generated, nil
}
