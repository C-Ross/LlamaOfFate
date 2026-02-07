package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

const (
	componentScenarioManager = "scenario_manager"
)

// ScenarioManager orchestrates multi-scene gameplay within a scenario
type ScenarioManager struct {
	engine               *Engine
	player               *character.Character
	ui                   UI
	sessionLogger        *session.Logger
	scenario             *prompt.Scenario                // The current scenario with problem and story questions
	initialScene         *scene.Scene                    // Optional pre-configured starting scene
	initialNPCs          []*character.Character          // NPCs for initial scene
	sceneSummaries       []prompt.SceneSummary           // Summaries of recent scenes (sliding window of last 3)
	lastGeneratedPurpose string                          // Purpose from the most recently generated scene
	lastGeneratedHook    string                          // Opening hook from the most recently generated scene
	sceneCount           int                             // Total scenes completed in this scenario
	scenarioCount        int                             // Current scenario number (set by GameManager)
	npcRegistry          map[string]*character.Character // Named NPCs persisted across scenes, keyed by normalized name
	npcAttitudes         map[string]string               // Last-known attitude per NPC (keyed by normalized name)
}

// NewScenarioManager creates a new scenario manager
func NewScenarioManager(engine *Engine, player *character.Character) *ScenarioManager {
	return &ScenarioManager{
		engine:       engine,
		player:       player,
		npcRegistry:  make(map[string]*character.Character),
		npcAttitudes: make(map[string]string),
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
func (m *ScenarioManager) SetScenario(scenario *prompt.Scenario) {
	m.scenario = scenario
}

// SetScenarioCount sets the current scenario number (for consequence recovery tracking)
func (m *ScenarioManager) SetScenarioCount(count int) {
	m.scenarioCount = count
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

		// Pass scene purpose so the response LLM is aware of it
		if m.lastGeneratedPurpose != "" {
			sceneManager.SetScenePurpose(m.lastGeneratedPurpose)
		}

		// Start the scene
		if err := sceneManager.StartScene(currentScene, m.player); err != nil {
			return nil, fmt.Errorf("failed to start scene: %w", err)
		}

		// Display scene purpose and opening hook to the player
		if m.lastGeneratedPurpose != "" {
			m.ui.DisplaySystemMessage("Scene Purpose: " + m.lastGeneratedPurpose)
			if m.sessionLogger != nil {
				m.sessionLogger.Log("scene_purpose", map[string]any{
					"purpose": m.lastGeneratedPurpose,
				})
			}
		}
		if m.lastGeneratedHook != "" {
			m.ui.DisplayNarrative(m.lastGeneratedHook)
			if m.sessionLogger != nil {
				m.sessionLogger.Log("opening_hook", map[string]any{
					"hook": m.lastGeneratedHook,
				})
			}
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

			// Update NPC attitudes from scene summary
			m.updateNPCAttitudes(summary)

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
					slog.Info("prompt.Scenario resolved",
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

		// Increment scene count and handle between-scene recovery
		m.sceneCount++
		m.handleBetweenSceneRecovery(ctx)

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
			// Register named NPCs for persistence across scenes
			if npc.CharacterType != character.CharacterTypeNamelessGood &&
				npc.CharacterType != character.CharacterTypeNamelessFair &&
				npc.CharacterType != character.CharacterTypeNamelessAverage {
				m.npcRegistry[normalizeNPCName(npc.Name)] = npc
				m.npcAttitudes[normalizeNPCName(npc.Name)] = "neutral"
			}
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

	data := prompt.SceneGenerationData{
		TransitionHint:    transitionHint,
		Scenario:          m.scenario,
		PlayerName:        m.player.Name,
		PlayerHighConcept: m.player.Aspects.HighConcept,
		PlayerTrouble:     m.player.Aspects.Trouble,
		PlayerAspects:     playerAspects,
		PreviousSummaries: m.sceneSummaries, // Include recent scene summaries for context
		Complications:     m.extractComplications(),
		KnownNPCs:         m.getKnownNPCSummaries(),
	}

	promptText, err := prompt.RenderSceneGeneration(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render scene generation prompt: %w", err)
	}

	resp, err := m.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: promptText}},
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
	generated, err := prompt.ParseGeneratedScene(resp.Choices[0].Message.Content)
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

	// Create and register NPCs, reusing known NPCs from previous scenes
	for i, npcData := range generated.NPCs {
		normalizedName := normalizeNPCName(npcData.Name)

		// Check if this NPC already exists in the registry
		if existingNPC, found := m.npcRegistry[normalizedName]; found {
			// Reuse existing NPC — they persist across scenes
			m.engine.AddCharacter(existingNPC) // Re-register in case engine was reset
			newScene.AddCharacter(existingNPC.ID)
			slog.Info("Recurring NPC added to scene",
				"component", componentScenarioManager,
				"npc_name", existingNPC.Name,
				"npc_id", existingNPC.ID,
			)
			continue
		}

		// Create new NPC
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

		// Register named NPCs (non-nameless) for persistence
		if npc.CharacterType != character.CharacterTypeNamelessGood &&
			npc.CharacterType != character.CharacterTypeNamelessFair &&
			npc.CharacterType != character.CharacterTypeNamelessAverage {
			m.npcRegistry[normalizedName] = npc
			m.npcAttitudes[normalizedName] = npcData.Disposition
		}
	}

	// Log the generated scene
	if m.sessionLogger != nil {
		m.sessionLogger.Log("scene_generated", map[string]any{
			"scene_id":          sceneID,
			"scene_name":        generated.SceneName,
			"description":       generated.Description,
			"purpose":           generated.Purpose,
			"opening_hook":      generated.OpeningHook,
			"situation_aspects": generated.SituationAspects,
			"npc_count":         len(generated.NPCs),
		})
	}

	slog.Info("Generated new scene",
		"component", componentScenarioManager,
		"scene_name", generated.SceneName,
		"purpose", generated.Purpose,
		"aspects", len(generated.SituationAspects),
		"npcs", len(generated.NPCs),
	)

	// Store generated purpose for the scene manager to use
	m.lastGeneratedPurpose = generated.Purpose
	m.lastGeneratedHook = generated.OpeningHook

	return newScene, nil
}

// addSceneSummary adds a summary to the sliding window (max 3 summaries)
func (m *ScenarioManager) addSceneSummary(summary *prompt.SceneSummary) {
	if summary == nil {
		return
	}
	m.sceneSummaries = append(m.sceneSummaries, *summary)
	// Keep only last 3 summaries (sliding window)
	if len(m.sceneSummaries) > 3 {
		m.sceneSummaries = m.sceneSummaries[len(m.sceneSummaries)-3:]
	}
}

// extractComplications gathers unresolved threads from recent scene summaries
// to pass as explicit complications for the next scene generation.
func (m *ScenarioManager) extractComplications() []string {
	seen := make(map[string]bool)
	var complications []string
	for _, summary := range m.sceneSummaries {
		for _, thread := range summary.UnresolvedThreads {
			if !seen[thread] {
				seen[thread] = true
				complications = append(complications, thread)
			}
		}
	}
	return complications
}

// handleBetweenSceneRecovery handles consequence recovery between scenes.
// Per Fate Core, recovery requires an overcome action, then waiting.
// This performs automatic rolls and generates LLM narrative.
func (m *ScenarioManager) handleBetweenSceneRecovery(ctx context.Context) {
	// First, check if any already-recovering consequences have healed
	cleared := m.player.CheckConsequenceRecovery(m.sceneCount, m.scenarioCount)
	for _, conseq := range cleared {
		m.ui.DisplaySystemMessage(fmt.Sprintf(
			"Your %s consequence \"%s\" has fully healed!",
			conseq.Type, conseq.Aspect,
		))
		if m.sessionLogger != nil {
			m.sessionLogger.Log("consequence_healed", map[string]any{
				"type":        conseq.Type,
				"aspect":      conseq.Aspect,
				"scene_count": m.sceneCount,
			})
		}
	}

	// Find consequences that haven't started recovery yet
	var needsRecovery []int
	for i, conseq := range m.player.Consequences {
		if !conseq.Recovering {
			needsRecovery = append(needsRecovery, i)
		}
	}
	if len(needsRecovery) == 0 {
		return
	}

	// Attempt automatic recovery rolls
	roller := dice.NewRoller()
	var attempts []prompt.RecoveryAttempt

	for _, idx := range needsRecovery {
		conseq := m.player.Consequences[idx]

		// Determine difficulty per Fate Core:
		// Mild: Fair (+2), Moderate: Great (+4), Severe: Fantastic (+6)
		// Self-treatment: +2 difficulty
		difficulty := conseq.Type.Value() // 2, 4, or 6
		difficulty += 2                   // Self-treatment penalty

		// Determine recovery skill
		skill, skillLevel := m.bestRecoverySkill(conseq)

		// Roll 4dF + skill
		result := roller.RollWithModifier(skillLevel, 0)

		outcome := "failure"
		if int(result.FinalValue) >= difficulty {
			outcome = "success"
			m.player.BeginConsequenceRecovery(conseq.ID, m.sceneCount, m.scenarioCount)
		}

		difficultyLabel := dice.Ladder(difficulty).String()

		attempts = append(attempts, prompt.RecoveryAttempt{
			Severity:   string(conseq.Type),
			Aspect:     conseq.Aspect,
			Difficulty: difficultyLabel,
			Skill:      skill,
			RollResult: int(result.FinalValue),
			Outcome:    outcome,
		})

		slog.Info("Recovery attempt",
			"component", componentScenarioManager,
			"consequence", conseq.Aspect,
			"severity", conseq.Type,
			"skill", skill,
			"roll", int(result.FinalValue),
			"difficulty", difficulty,
			"outcome", outcome,
		)
	}

	// Generate LLM narrative for recovery
	m.displayRecoveryNarrative(ctx, attempts)
}

// bestRecoverySkill determines the best skill and its level for recovery.
// Physical consequences use Lore (or Will as fallback), mental use Empathy (or Rapport).
func (m *ScenarioManager) bestRecoverySkill(conseq character.Consequence) (string, dice.Ladder) {
	// Determine which skills could help based on consequence aspect context
	// Physical keywords suggest physical recovery, otherwise mental
	physicalSkills := []string{"Lore", "Crafts", "Will"}
	mentalSkills := []string{"Empathy", "Rapport", "Will"}

	var candidates []string
	switch conseq.Duration {
	case "mild", "moderate", "severe":
		// Use consequence type to infer physical vs mental
		// Default to physical skills for now
		candidates = physicalSkills
	default:
		candidates = physicalSkills
	}

	// Also try mental skills
	candidates = append(candidates, mentalSkills...)

	bestSkill := "Will"
	bestLevel := m.player.GetSkill("Will")

	for _, skill := range candidates {
		level := m.player.GetSkill(skill)
		if level > bestLevel {
			bestSkill = skill
			bestLevel = level
		}
	}

	return bestSkill, bestLevel
}

// displayRecoveryNarrative generates and displays LLM-driven narrative for recovery
func (m *ScenarioManager) displayRecoveryNarrative(ctx context.Context, attempts []prompt.RecoveryAttempt) {
	if len(attempts) == 0 {
		return
	}

	// Display mechanical results
	m.ui.DisplaySystemMessage("\n--- Between Scenes: Recovery ---")
	for _, a := range attempts {
		if a.Outcome == "success" {
			m.ui.DisplaySystemMessage(fmt.Sprintf(
				"Recovery roll for \"%s\" (%s): %s +%d vs %s — Success! Recovery begins.",
				a.Aspect, a.Severity, a.Skill, a.RollResult, a.Difficulty,
			))
		} else {
			m.ui.DisplaySystemMessage(fmt.Sprintf(
				"Recovery roll for \"%s\" (%s): %s +%d vs %s — Failed. The wound persists.",
				a.Aspect, a.Severity, a.Skill, a.RollResult, a.Difficulty,
			))
		}
	}

	// Generate LLM narrative
	if m.engine.llmClient == nil {
		return
	}

	setting := ""
	if m.scenario != nil {
		setting = m.scenario.Setting
	}

	data := prompt.RecoveryNarrativeData{
		CharacterName: m.player.Name,
		SceneSetting:  setting,
		Consequences:  attempts,
	}

	promptText, err := prompt.RenderRecoveryNarrative(data)
	if err != nil {
		slog.Warn("Failed to render recovery narrative prompt",
			"component", componentScenarioManager,
			"error", err,
		)
		return
	}

	resp, err := m.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: promptText}},
		MaxTokens:   200,
		Temperature: 0.6,
	})
	if err != nil {
		slog.Warn("Failed to generate recovery narrative",
			"component", componentScenarioManager,
			"error", err,
		)
		return
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return
	}

	// Try to parse JSON response for narrative and renamed aspects
	type recoveryResponse struct {
		Narrative      string            `json:"narrative"`
		RenamedAspects map[string]string `json:"renamed_aspects"`
	}

	content := llm.CleanJSONResponse(resp.Choices[0].Message.Content)
	var parsed recoveryResponse
	if parseErr := json.Unmarshal([]byte(content), &parsed); parseErr != nil {
		// If parsing fails, display raw content
		m.ui.DisplayNarrative(strings.TrimSpace(resp.Choices[0].Message.Content))
	} else {
		if parsed.Narrative != "" {
			m.ui.DisplayNarrative(parsed.Narrative)
		}
		// Apply renamed aspects to recovering consequences
		for oldAspect, newAspect := range parsed.RenamedAspects {
			for i := range m.player.Consequences {
				if m.player.Consequences[i].Aspect == oldAspect && m.player.Consequences[i].Recovering {
					m.player.Consequences[i].Aspect = newAspect
					slog.Info("Consequence renamed during recovery",
						"component", componentScenarioManager,
						"old", oldAspect,
						"new", newAspect,
					)
				}
			}
		}
	}

	if m.sessionLogger != nil {
		m.sessionLogger.Log("recovery_attempt", map[string]any{
			"attempts": attempts,
		})
	}
}

// updateNPCAttitudes updates the NPC registry with attitude changes from a scene summary
func (m *ScenarioManager) updateNPCAttitudes(summary *prompt.SceneSummary) {
	if summary == nil {
		return
	}
	for _, npc := range summary.NPCsEncountered {
		normalizedName := normalizeNPCName(npc.Name)
		m.npcAttitudes[normalizedName] = npc.Attitude
	}
}

// normalizeNPCName normalizes an NPC name for matching (lowercase, trimmed)
func normalizeNPCName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// getKnownNPCSummaries returns summaries of known NPCs for scene generation prompts
func (m *ScenarioManager) getKnownNPCSummaries() []prompt.NPCSummary {
	var summaries []prompt.NPCSummary
	for normalizedName, npc := range m.npcRegistry {
		attitude := m.npcAttitudes[normalizedName]
		if attitude == "" {
			attitude = "neutral"
		}
		summaries = append(summaries, prompt.NPCSummary{
			Name:     npc.Name,
			Attitude: attitude,
		})
	}
	return summaries
}

// generateSceneSummary creates a summary of the completed scene using LLM
func (m *ScenarioManager) generateSceneSummary(ctx context.Context, sceneManager *SceneManager, completedScene *scene.Scene, result *SceneEndResult) (*prompt.SceneSummary, error) {
	// Gather situation aspects
	var aspects []string
	for _, sa := range completedScene.SituationAspects {
		aspects = append(aspects, sa.Aspect)
	}

	// Gather NPCs in scene
	var npcsInScene []prompt.NPCSummary
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
			npcsInScene = append(npcsInScene, prompt.NPCSummary{
				Name:     char.Name,
				Attitude: attitude,
			})
		}
	}

	// Determine how ended string
	howEnded := string(result.Reason)

	data := prompt.SceneSummaryData{
		SceneName:           completedScene.Name,
		SceneDescription:    completedScene.Description,
		SituationAspects:    aspects,
		ConversationHistory: sceneManager.GetConversationHistory(),
		NPCsInScene:         npcsInScene,
		TakenOutChars:       result.TakenOutChars,
		HowEnded:            howEnded,
		TransitionHint:      result.TransitionHint,
	}

	promptText, err := prompt.RenderSceneSummary(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render scene summary prompt: %w", err)
	}

	resp, err := m.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: promptText}},
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
	summary, err := prompt.ParseSceneSummary(resp.Choices[0].Message.Content)
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

// checkScenarioResolution uses the LLM to determine if the scenario's story questions have been answered
func (m *ScenarioManager) checkScenarioResolution(ctx context.Context, latestSummary *prompt.SceneSummary) (bool, error) {
	if m.scenario == nil {
		return false, nil
	}

	// Gather player aspects
	playerAspects := m.player.Aspects.GetAll()

	data := prompt.ScenarioResolutionData{
		Scenario:       m.scenario,
		SceneSummaries: m.sceneSummaries,
		LatestSummary:  latestSummary,
		PlayerName:     m.player.Name,
		PlayerAspects:  playerAspects,
	}

	promptText, err := prompt.RenderScenarioResolution(data)
	if err != nil {
		return false, fmt.Errorf("failed to render scenario resolution prompt: %w", err)
	}

	resp, err := m.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: promptText}},
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
	result, err := prompt.ParseScenarioResolution(resp.Choices[0].Message.Content)
	if err != nil {
		return false, fmt.Errorf("failed to parse scenario resolution: %w", err)
	}

	slog.Info("prompt.Scenario resolution check",
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
