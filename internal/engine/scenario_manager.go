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

// ScenarioManager orchestrates multi-scene gameplay within a scenario
type ScenarioManager struct {
	engine         *Engine
	player         *character.Character
	ui             UI
	sessionLogger  *session.Logger
	scenario       *Scenario              // The current scenario with problem and story questions
	initialScene   *scene.Scene           // Optional pre-configured starting scene
	initialNPCs    []*character.Character // NPCs for initial scene
	sceneSummaries []SceneSummary         // Summaries of recent scenes (sliding window of last 3)
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

// SetScenario sets the scenario for the manager
func (m *ScenarioManager) SetScenario(scenario *Scenario) {
	m.scenario = scenario
}

// SetInitialScene sets a pre-configured starting scene
func (m *ScenarioManager) SetInitialScene(s *scene.Scene, npcs []*character.Character) {
	m.initialScene = s
	m.initialNPCs = npcs
}

// Run executes the scenario loop, transitioning between scenes until quit, player taken out, or scenario resolved
func (m *ScenarioManager) Run(ctx context.Context) (*ScenarioResult, error) {
	if m.engine == nil {
		return nil, fmt.Errorf("engine is required")
	}
	if m.engine.llmClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}
	if m.ui == nil {
		return nil, fmt.Errorf("UI is required")
	}
	if m.player == nil {
		return nil, fmt.Errorf("player character is required")
	}

	// Register player with engine
	m.engine.AddCharacter(m.player)

	// Get or create initial scene
	currentScene, err := m.getInitialScene(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial scene: %w", err)
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
			return nil, fmt.Errorf("failed to start scene: %w", err)
		}

		// Run the scene loop
		result, err := sceneManager.RunSceneLoop(ctx)
		if err != nil {
			return nil, fmt.Errorf("scene loop error: %w", err)
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
			return &ScenarioResult{Reason: ScenarioEndQuit, Scenario: m.scenario}, nil

		case SceneEndPlayerTakenOut:
			// Player was taken out - could continue or end based on context
			if result.TransitionHint != "" {
				// Player taken out but story continues (captured, etc.)
				transitionHint = result.TransitionHint
			} else {
				// Game over
				return &ScenarioResult{Reason: ScenarioEndPlayerTakenOut, Scenario: m.scenario}, nil
			}

		case SceneEndTransition:
			// Normal scene transition
			transitionHint = result.TransitionHint
		}

		// Generate and store scene summary for context continuity
		summary, err := m.generateSceneSummary(ctx, sceneManager, currentScene, result)
		if err != nil {
			slog.Warn("Failed to generate scene summary, continuing without it",
				"component", componentScenarioManager,
				"error", err,
			)
		} else {
			m.addSceneSummary(summary)

			// Log the scene summary
			if m.sessionLogger != nil {
				m.sessionLogger.Log("scene_summary", summary)
			}

			// Check if the scenario is resolved (only if we have a scenario with story questions)
			if m.scenario != nil && len(m.scenario.StoryQuestions) > 0 {
				resolved, err := m.checkScenarioResolution(ctx, summary)
				if err != nil {
					slog.Warn("Failed to check scenario resolution, continuing",
						"component", componentScenarioManager,
						"error", err,
					)
				} else if resolved {
					m.scenario.IsResolved = true
					slog.Info("Scenario resolved",
						"component", componentScenarioManager,
						"scenario_title", m.scenario.Title,
					)
					if m.sessionLogger != nil {
						m.sessionLogger.Log("scenario_resolved", map[string]any{
							"scenario_title": m.scenario.Title,
							"scenario":       m.scenario,
						})
					}
					return &ScenarioResult{Reason: ScenarioEndResolved, Scenario: m.scenario}, nil
				}
			}
		}

		// Generate the next scene
		currentScene, err = m.generateNextScene(ctx, transitionHint)
		if err != nil {
			slog.Error("Failed to generate next scene",
				"component", componentScenarioManager,
				"error", err,
			)
			return nil, fmt.Errorf("failed to generate next scene: %w", err)
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
		Scenario:          m.scenario,
		PlayerName:        m.player.Name,
		PlayerHighConcept: m.player.Aspects.HighConcept,
		PlayerTrouble:     m.player.Aspects.Trouble,
		PlayerAspects:     playerAspects,
		PreviousSummaries: m.sceneSummaries, // Include recent scene summaries for context
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

// addSceneSummary adds a summary to the sliding window (max 3 summaries)
func (m *ScenarioManager) addSceneSummary(summary *SceneSummary) {
	if summary == nil {
		return
	}
	m.sceneSummaries = append(m.sceneSummaries, *summary)
	// Keep only last 3 summaries (sliding window)
	if len(m.sceneSummaries) > 3 {
		m.sceneSummaries = m.sceneSummaries[len(m.sceneSummaries)-3:]
	}
}

// generateSceneSummary creates a summary of the completed scene using LLM
func (m *ScenarioManager) generateSceneSummary(ctx context.Context, sceneManager *SceneManager, completedScene *scene.Scene, result *SceneEndResult) (*SceneSummary, error) {
	// Gather situation aspects
	var aspects []string
	for _, sa := range completedScene.SituationAspects {
		aspects = append(aspects, sa.Aspect)
	}

	// Gather NPCs in scene
	var npcsInScene []NPCSummary
	for _, charID := range completedScene.Characters {
		if charID == m.player.ID {
			continue
		}
		char := m.engine.GetCharacter(charID)
		if char != nil {
			attitude := "neutral"
			// Check if NPC was taken out
			for _, takenOutID := range result.TakenOutChars {
				if takenOutID == charID {
					attitude = "defeated"
					break
				}
			}
			npcsInScene = append(npcsInScene, NPCSummary{
				Name:     char.Name,
				Attitude: attitude,
			})
		}
	}

	// Determine how ended string
	howEnded := string(result.Reason)

	data := SceneSummaryData{
		SceneName:           completedScene.Name,
		SceneDescription:    completedScene.Description,
		SituationAspects:    aspects,
		ConversationHistory: sceneManager.GetConversationHistory(),
		NPCsInScene:         npcsInScene,
		TakenOutChars:       result.TakenOutChars,
		HowEnded:            howEnded,
		TransitionHint:      result.TransitionHint,
	}

	prompt, err := RenderSceneSummary(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render scene summary prompt: %w", err)
	}

	resp, err := m.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   400,
		Temperature: 0.5, // More focused for summarization
	})
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty LLM response")
	}

	// Parse the generated summary
	summary, err := m.parseSceneSummary(resp.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse scene summary: %w", err)
	}

	slog.Info("Generated scene summary",
		"component", componentScenarioManager,
		"scene_name", completedScene.Name,
		"key_events", len(summary.KeyEvents),
		"npcs", len(summary.NPCsEncountered),
	)

	return summary, nil
}

// parseSceneSummary parses the LLM response into a SceneSummary
func (m *ScenarioManager) parseSceneSummary(content string) (*SceneSummary, error) {
	// Clean the response - extract JSON from potential markdown code blocks
	cleaned := cleanJSONResponse(content)

	var summary SceneSummary
	if err := json.Unmarshal([]byte(cleaned), &summary); err != nil {
		// Try to extract just the first JSON object if there's extra content
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &summary); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	// Validate required field
	if summary.NarrativeProse == "" {
		return nil, fmt.Errorf("missing narrative_prose")
	}

	return &summary, nil
}

// checkScenarioResolution uses the LLM to determine if the scenario's story questions have been answered
func (m *ScenarioManager) checkScenarioResolution(ctx context.Context, latestSummary *SceneSummary) (bool, error) {
	if m.scenario == nil {
		return false, nil
	}

	// Gather player aspects
	playerAspects := m.player.Aspects.GetAll()

	data := ScenarioResolutionData{
		Scenario:       m.scenario,
		SceneSummaries: m.sceneSummaries,
		LatestSummary:  latestSummary,
		PlayerName:     m.player.Name,
		PlayerAspects:  playerAspects,
	}

	prompt, err := RenderScenarioResolution(data)
	if err != nil {
		return false, fmt.Errorf("failed to render scenario resolution prompt: %w", err)
	}

	resp, err := m.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   300,
		Temperature: 0.3, // Low temperature for more consistent analysis
	})
	if err != nil {
		return false, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return false, fmt.Errorf("empty LLM response")
	}

	// Parse the resolution result
	result, err := m.parseScenarioResolution(resp.Choices[0].Message.Content)
	if err != nil {
		return false, fmt.Errorf("failed to parse scenario resolution: %w", err)
	}

	slog.Info("Scenario resolution check",
		"component", componentScenarioManager,
		"is_resolved", result.IsResolved,
		"answered_questions", result.AnsweredQuestions,
		"reasoning", result.Reasoning,
	)

	// Log the resolution check
	if m.sessionLogger != nil {
		m.sessionLogger.Log("scenario_resolution_check", map[string]any{
			"is_resolved":        result.IsResolved,
			"answered_questions": result.AnsweredQuestions,
			"reasoning":          result.Reasoning,
		})
	}

	return result.IsResolved, nil
}

// parseScenarioResolution parses the LLM response into a ScenarioResolutionResult
func (m *ScenarioManager) parseScenarioResolution(content string) (*ScenarioResolutionResult, error) {
	// Clean the response - extract JSON from potential markdown code blocks
	cleaned := cleanJSONResponse(content)

	var result ScenarioResolutionResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		// Try to extract just the first JSON object if there's extra content
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			cleaned = content[start : end+1]
			if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
				return nil, fmt.Errorf("JSON parse error: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in response")
		}
	}

	return &result, nil
}
