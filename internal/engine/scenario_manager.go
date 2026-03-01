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
	sessionLogger        session.SessionLogger
	scenario             *scene.Scenario                 // The current scenario with problem and story questions
	initialScene         *scene.Scene                    // Optional pre-configured starting scene
	initialNPCs          []*character.Character          // NPCs for initial scene
	sceneSummaries       []prompt.SceneSummary           // Summaries of recent scenes (sliding window of last 3)
	lastGeneratedPurpose string                          // Purpose from the most recently generated scene
	lastGeneratedHook    string                          // Opening hook from the most recently generated scene
	sceneCount           int                             // Total scenes completed in this scenario
	scenarioCount        int                             // Current scenario number (set by GameManager)
	npcRegistry          map[string]*character.Character // Named NPCs persisted across scenes, keyed by normalized name
	npcAttitudes         map[string]string               // Last-known attitude per NPC (keyed by normalized name)
	saveFunc             func() error                    // Optional callback to trigger a save via GameManager
	resumed              bool                            // True when restoring from a saved game
	started              bool                            // True after Start has been called
	currentScene         *scene.Scene                    // The active scene (set by Start, updated by HandleInput)
	lastTransitionHint   string                          // Transition hint from the last scene end
	exitAfterScene       bool                            // Exit after the first scene ends instead of generating next scenes
}

// NewScenarioManager creates a new scenario manager.
// sessionLogger must not be nil; use session.NullLogger{} when logging is not needed.
func NewScenarioManager(engine *Engine, player *character.Character, sessionLogger session.SessionLogger) *ScenarioManager {
	return &ScenarioManager{
		engine:        engine,
		player:        player,
		sessionLogger: sessionLogger,
		npcRegistry:   make(map[string]*character.Character),
		npcAttitudes:  make(map[string]string),
	}
}

// SetScenario sets the scenario for the manager
func (m *ScenarioManager) SetScenario(scenario *scene.Scenario) {
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

// SetSaveFunc sets a callback that triggers a game state save.
// The callback is provided by GameManager to avoid upward coupling.
func (m *ScenarioManager) SetSaveFunc(fn func() error) {
	m.saveFunc = fn
}

// SetExitAfterScene configures the manager to exit after the first scene ends
// instead of generating subsequent scenes. Used for single-scene demos.
func (m *ScenarioManager) SetExitAfterScene(exit bool) {
	m.exitAfterScene = exit
}

// triggerSave calls the save callback if configured, logging any errors.
// Save failures are non-fatal — the game continues even if persistence fails.
func (m *ScenarioManager) triggerSave(trigger string) {
	if m.saveFunc == nil {
		return
	}
	if err := m.saveFunc(); err != nil {
		slog.Warn("Failed to save game state",
			"component", componentScenarioManager,
			"trigger", trigger,
			"error", err,
		)
	} else {
		slog.Info("Game state saved",
			"component", componentScenarioManager,
			"trigger", trigger,
		)
	}
}

// emitSceneOpeningEvents returns the narrative events for a newly started scene:
// the scene name/description, and optionally the dramatic purpose and opening hook.
// It also logs purpose/hook to the session logger when present.
func (m *ScenarioManager) emitSceneOpeningEvents() []GameEvent {
	events := []GameEvent{
		NarrativeEvent{
			SceneName: m.currentScene.Name,
			Text:      m.currentScene.Description,
		},
	}
	if m.lastGeneratedPurpose != "" || m.lastGeneratedHook != "" {
		events = append(events, NarrativeEvent{
			Purpose: m.lastGeneratedPurpose,
			Text:    m.lastGeneratedHook,
		})
		if m.lastGeneratedPurpose != "" {
			m.sessionLogger.Log("scene_purpose", map[string]any{
				"purpose": m.lastGeneratedPurpose,
			})
		}
		if m.lastGeneratedHook != "" {
			m.sessionLogger.Log("opening_hook", map[string]any{
				"hook": m.lastGeneratedHook,
			})
		}
	}
	return events
}

// Snapshot returns the scenario-level and scene-level state for persistence.
// It cascades to SceneManager.Snapshot() for the scene layer.
func (m *ScenarioManager) Snapshot() (ScenarioState, SceneState) {
	// Copy NPC registry to avoid aliasing
	npcRegistry := make(map[string]*character.Character, len(m.npcRegistry))
	for k, v := range m.npcRegistry {
		npcRegistry[k] = v
	}

	// Copy NPC attitudes to avoid aliasing
	npcAttitudes := make(map[string]string, len(m.npcAttitudes))
	for k, v := range m.npcAttitudes {
		npcAttitudes[k] = v
	}

	// Copy scene summaries to avoid aliasing
	summaries := make([]prompt.SceneSummary, len(m.sceneSummaries))
	copy(summaries, m.sceneSummaries)

	scenarioState := ScenarioState{
		Player:         m.player,
		Scenario:       m.scenario,
		ScenarioCount:  m.scenarioCount,
		SceneCount:     m.sceneCount,
		SceneSummaries: summaries,
		NPCRegistry:    npcRegistry,
		NPCAttitudes:   npcAttitudes,
		LastPurpose:    m.lastGeneratedPurpose,
		LastHook:       m.lastGeneratedHook,
	}

	sceneState := m.engine.GetSceneManager().Snapshot()
	return scenarioState, sceneState
}

// Restore sets the scenario manager's state from a previously saved game state,
// enabling session resume. It restores all scenario-level fields, delegates
// scene restoration to the engine's SceneManager, and registers NPCs with the
// engine. After calling Restore, the next call to Run will resume mid-scene
// instead of generating a fresh initial scene.
func (m *ScenarioManager) Restore(scenarioState ScenarioState, sceneState SceneState) {
	m.player = scenarioState.Player
	m.scenario = scenarioState.Scenario
	m.scenarioCount = scenarioState.ScenarioCount
	m.sceneCount = scenarioState.SceneCount
	m.sceneSummaries = scenarioState.SceneSummaries
	m.npcRegistry = scenarioState.NPCRegistry
	m.npcAttitudes = scenarioState.NPCAttitudes
	m.lastGeneratedPurpose = scenarioState.LastPurpose
	m.lastGeneratedHook = scenarioState.LastHook

	// Initialize maps if nil (defensive against empty saves)
	if m.npcRegistry == nil {
		m.npcRegistry = make(map[string]*character.Character)
	}
	if m.npcAttitudes == nil {
		m.npcAttitudes = make(map[string]string)
	}

	// Register NPCs with the engine so scene character lookups work
	for _, npc := range m.npcRegistry {
		m.engine.AddCharacter(npc)
	}

	// Restore scene-level state via the engine's SceneManager
	m.engine.GetSceneManager().Restore(sceneState, m.player)

	// Mark as resumed so Run() skips initial scene generation
	m.resumed = true
}

// Start initializes the scenario and first scene, returning the opening events.
// This is the async entry point for event-driven UIs (web). After Start, the
// caller drives the game by calling HandleInput for each player input.
//
// Returned events include the scene name/description NarrativeEvent, any
// purpose/hook NarrativeEvent, and conversation recap DialogEvents if resuming.
func (m *ScenarioManager) Start(ctx context.Context) ([]GameEvent, error) {
	if m.engine == nil {
		return nil, fmt.Errorf("engine is required")
	}
	if m.engine.llmClient == nil {
		return nil, fmt.Errorf("LLM client is required")
	}
	if m.player == nil {
		return nil, fmt.Errorf("player character is required")
	}

	// Register player with engine
	m.engine.AddCharacter(m.player)

	var events []GameEvent

	// Get the initial scene — either restored from save, pre-configured, or generated
	resuming := m.resumed
	m.resumed = false

	if resuming {
		// Scene already restored via Restore() — retrieve it from the scene manager
		m.currentScene = m.engine.GetSceneManager().GetCurrentScene()
	} else {
		var err error
		m.currentScene, err = m.getInitialScene(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get initial scene: %w", err)
		}
	}

	// Set up the scene manager for this scene
	sceneManager := m.engine.GetSceneManager()
	sceneManager.SetExitOnSceneTransition(true)

	if resuming {
		// Scene already restored — reset state and emit scene info + recap
		sceneManager.resetSceneState()

		// Emit the scene name and description so the caller has the opening context
		events = append(events, NarrativeEvent{
			SceneName: m.currentScene.Name,
			Text:      m.currentScene.Description,
		})

		// Replay conversation history so the player has context
		history := sceneManager.GetConversationHistory()
		for _, entry := range history {
			events = append(events, DialogEvent{
				PlayerInput: entry.PlayerInput,
				GMResponse:  entry.GMResponse,
				IsRecap:     true,
				RecapType:   entry.Type,
			})
		}
	} else {
		// Normal path: set purpose, start scene, emit opening events
		if m.lastGeneratedPurpose != "" {
			sceneManager.SetScenePurpose(m.lastGeneratedPurpose)
		}

		// Start the scene
		if err := sceneManager.StartScene(m.currentScene, m.player); err != nil {
			return nil, fmt.Errorf("failed to start scene: %w", err)
		}

		// Reset scene state for the first scene
		sceneManager.resetSceneState()

		events = append(events, m.emitSceneOpeningEvents()...)

		// Save state at scene start
		m.triggerSave("scene_start")
	}

	m.started = true
	return events, nil
}

// HandleInput processes a single player input through the scenario layer.
// It delegates to SceneManager.HandleInput and, when a scene ends, performs
// the between-scene work (summary, resolution check, recovery, next scene
// generation) inline — bundling all resulting events into the returned
// InputResult.
//
// If InputResult.GameOver is true, the scenario has ended and ScenarioResult
// is populated. The caller should not send further inputs.
//
// This is the primary entry point for event-driven UIs (web via WebSocket).
// Terminal UIs use Run() which wraps this in a blocking loop.
func (m *ScenarioManager) HandleInput(ctx context.Context, input string) (*InputResult, error) {
	if !m.started {
		return nil, fmt.Errorf("HandleInput called before Start")
	}

	sceneManager := m.engine.GetSceneManager()
	result, err := sceneManager.HandleInput(ctx, input)
	if err != nil {
		return nil, err
	}

	// If scene hasn't ended or we're awaiting a response, save and return
	if !result.SceneEnded || result.AwaitingInvoke || result.AwaitingMidFlow {
		m.triggerSave("dialog")
		return result, nil
	}

	// Scene ended — do between-scene work and bundle events
	return m.completeSceneTransition(ctx, result)
}

// ProvideInvokeResponse forwards an invoke response to the SceneManager and
// handles any resulting scene-end transition, just like HandleInput.
func (m *ScenarioManager) ProvideInvokeResponse(ctx context.Context, resp InvokeResponse) (*InputResult, error) {
	if !m.started {
		return nil, fmt.Errorf("ProvideInvokeResponse called before Start")
	}

	sceneManager := m.engine.GetSceneManager()
	result, err := sceneManager.ProvideInvokeResponse(ctx, resp)
	if err != nil {
		return nil, err
	}

	if !result.SceneEnded || result.AwaitingInvoke || result.AwaitingMidFlow {
		m.triggerSave("dialog")
		return result, nil
	}

	return m.completeSceneTransition(ctx, result)
}

// ProvideMidFlowResponse forwards a mid-flow response to the SceneManager and
// handles any resulting scene-end transition, just like HandleInput.
func (m *ScenarioManager) ProvideMidFlowResponse(ctx context.Context, resp MidFlowResponse) (*InputResult, error) {
	if !m.started {
		return nil, fmt.Errorf("ProvideMidFlowResponse called before Start")
	}

	sceneManager := m.engine.GetSceneManager()
	result, err := sceneManager.ProvideMidFlowResponse(ctx, resp)
	if err != nil {
		return nil, err
	}

	if !result.SceneEnded || result.AwaitingInvoke || result.AwaitingMidFlow {
		m.triggerSave("dialog")
		return result, nil
	}

	return m.completeSceneTransition(ctx, result)
}

// completeSceneTransition performs all between-scene work after a scene ends
// and bundles the resulting events into the InputResult.
func (m *ScenarioManager) completeSceneTransition(ctx context.Context, result *InputResult) (*InputResult, error) {
	sceneManager := m.engine.GetSceneManager()
	sceneResult := result.EndResult

	extraEvents, scenarioResult, err := m.handleSceneEnd(ctx, sceneManager, sceneResult)
	if err != nil {
		return nil, err
	}
	result.Events = append(result.Events, extraEvents...)

	if scenarioResult != nil {
		result.GameOver = true
		result.ScenarioResult = scenarioResult
	}

	return result, nil
}

// handleSceneEnd performs between-scene work when a scene ends: logging, summary
// generation, resolution check, recovery, and next scene generation. Returns
// additional events and an optional ScenarioResult if the scenario is over.
func (m *ScenarioManager) handleSceneEnd(ctx context.Context, sceneManager *SceneManager, sceneResult *SceneEndResult) ([]GameEvent, *ScenarioResult, error) {
	var events []GameEvent

	// Log the scene end
	m.sessionLogger.Log("scene_end", map[string]any{
		"reason":          sceneResult.Reason,
		"transition_hint": sceneResult.TransitionHint,
		"taken_out_chars": sceneResult.TakenOutChars,
	})

	slog.Info("Scene ended",
		"component", componentScenarioManager,
		"reason", sceneResult.Reason,
		"transition_hint", sceneResult.TransitionHint,
	)

	// In single-scene mode, exit immediately after the scene ends
	if m.exitAfterScene {
		return events, &ScenarioResult{Reason: ScenarioEndSceneComplete, Scenario: m.scenario}, nil
	}

	// Handle the scene result
	switch sceneResult.Reason {
	case SceneEndQuit:
		m.triggerSave("player_quit")
		return events, &ScenarioResult{Reason: ScenarioEndQuit, Scenario: m.scenario}, nil

	case SceneEndPlayerTakenOut:
		if sceneResult.TransitionHint != "" {
			m.lastTransitionHint = sceneResult.TransitionHint
		} else {
			return events, &ScenarioResult{Reason: ScenarioEndPlayerTakenOut, Scenario: m.scenario}, nil
		}

	case SceneEndTransition:
		m.lastTransitionHint = sceneResult.TransitionHint
	}

	// Generate and store scene summary for context continuity
	summary, err := m.generateSceneSummary(ctx, sceneManager, m.currentScene, sceneResult)
	if err != nil {
		slog.Warn("Failed to generate scene summary, continuing without it",
			"component", componentScenarioManager,
			"error", err,
		)
	} else {
		m.addSceneSummary(summary)
		m.updateNPCAttitudes(summary)

		m.sessionLogger.Log("scene_summary", summary)

		// Check if the scenario is resolved
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
				m.sessionLogger.Log("scenario_resolved", map[string]any{
					"scenario_title": m.scenario.Title,
					"scenario":       m.scenario,
				})
				m.triggerSave("scenario_complete")
				return events, &ScenarioResult{Reason: ScenarioEndResolved, Scenario: m.scenario}, nil
			}
		}
	}

	// Between-scene recovery
	m.sceneCount++
	recoveryEvents := m.handleBetweenSceneRecovery(ctx)
	events = append(events, recoveryEvents...)

	// Save at scene transition
	m.triggerSave("scene_transition")

	// Generate the next scene
	nextScene, err := m.generateNextScene(ctx, m.lastTransitionHint)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate next scene: %w", err)
	}
	m.currentScene = nextScene

	// Set up the new scene
	newSceneManager := m.engine.GetSceneManager()
	newSceneManager.SetExitOnSceneTransition(true)

	if m.lastGeneratedPurpose != "" {
		newSceneManager.SetScenePurpose(m.lastGeneratedPurpose)
	}

	if err := newSceneManager.StartScene(m.currentScene, m.player); err != nil {
		return nil, nil, fmt.Errorf("failed to start scene: %w", err)
	}

	newSceneManager.resetSceneState()

	events = append(events, m.emitSceneOpeningEvents()...)

	m.triggerSave("scene_start")

	return events, nil, nil
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
			if !npc.CharacterType.IsNameless() {
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

	rawResponse, err := llm.SimpleCompletion(ctx, m.engine.llmClient, promptText, 500, 0.8)
	if err != nil {
		return nil, err
	}

	// Parse the generated scene
	generated, err := prompt.ParseGeneratedScene(rawResponse)
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
			// Skip permanently removed NPCs — they should never reappear
			if existingNPC.IsPermanentlyRemoved() {
				slog.Info("Skipping permanently removed NPC",
					"component", componentScenarioManager,
					"npc_name", existingNPC.Name,
					"npc_id", existingNPC.ID,
					"fate", existingNPC.Fate.Description,
				)
				continue
			}

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
		if !npc.CharacterType.IsNameless() {
			m.npcRegistry[normalizedName] = npc
			m.npcAttitudes[normalizedName] = npcData.Disposition
		}
	}

	// Log the generated scene
	m.sessionLogger.Log("scene_generated", map[string]any{
		"scene_id":          sceneID,
		"scene_name":        generated.SceneName,
		"description":       generated.Description,
		"purpose":           generated.Purpose,
		"opening_hook":      generated.OpeningHook,
		"situation_aspects": generated.SituationAspects,
		"npc_count":         len(generated.NPCs),
	})

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

// handleBetweenSceneRecovery handles consequence recovery between scenes
// and returns the events for the caller to render.
// Per Fate Core, recovery requires an overcome action, then waiting.
// This performs automatic rolls and generates LLM narrative.
func (m *ScenarioManager) handleBetweenSceneRecovery(ctx context.Context) []GameEvent {
	var events []GameEvent

	// First, check if any already-recovering consequences have healed
	cleared := m.player.CheckConsequenceRecovery(m.sceneCount, m.scenarioCount)
	for _, conseq := range cleared {
		events = append(events, RecoveryEvent{
			Action:   "healed",
			Aspect:   conseq.Aspect,
			Severity: string(conseq.Type),
		})
		m.sessionLogger.Log("consequence_healed", map[string]any{
			"type":        conseq.Type,
			"aspect":      conseq.Aspect,
			"scene_count": m.sceneCount,
		})
	}

	// Find consequences that haven't started recovery yet
	var needsRecovery []int
	for i, conseq := range m.player.Consequences {
		if !conseq.Recovering {
			needsRecovery = append(needsRecovery, i)
		}
	}
	if len(needsRecovery) == 0 {
		return events
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
	events = append(events, m.buildRecoveryNarrativeEvents(ctx, attempts)...)

	return events
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

// buildRecoveryNarrativeEvents generates recovery roll events and LLM-driven
// narrative events, returning them for the caller to render.
func (m *ScenarioManager) buildRecoveryNarrativeEvents(ctx context.Context, attempts []prompt.RecoveryAttempt) []GameEvent {
	if len(attempts) == 0 {
		return nil
	}

	var events []GameEvent

	// Mechanical results — the terminal UI renders a header automatically.
	for _, a := range attempts {
		events = append(events, RecoveryEvent{
			Action:     "roll",
			Aspect:     a.Aspect,
			Severity:   a.Severity,
			Skill:      a.Skill,
			RollResult: a.RollResult,
			Difficulty: a.Difficulty,
			Success:    a.Outcome == "success",
		})
	}

	// Generate LLM narrative
	if m.engine.llmClient == nil {
		return events
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
		return events
	}

	content, err := llm.SimpleCompletion(ctx, m.engine.llmClient, promptText, 200, 0.6)
	if err != nil {
		slog.Warn("Failed to generate recovery narrative",
			"component", componentScenarioManager,
			"error", err,
		)
		return events
	}

	// Try to parse JSON response for narrative and renamed aspects
	type recoveryResponse struct {
		Narrative      string            `json:"narrative"`
		RenamedAspects map[string]string `json:"renamed_aspects"`
	}

	var parsed recoveryResponse
	if parseErr := json.Unmarshal([]byte(content), &parsed); parseErr != nil {
		// If parsing fails, use raw content
		events = append(events, NarrativeEvent{Text: content})
	} else {
		if parsed.Narrative != "" {
			events = append(events, NarrativeEvent{Text: parsed.Narrative})
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

	m.sessionLogger.Log("recovery_attempt", map[string]any{
		"attempts": attempts,
	})

	return events
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

// getKnownNPCSummaries returns summaries of known NPCs for scene generation prompts.
// Permanently removed NPCs (killed, destroyed) are excluded entirely.
// Temporarily defeated NPCs include their fate description in the attitude.
func (m *ScenarioManager) getKnownNPCSummaries() []prompt.NPCSummary {
	var summaries []prompt.NPCSummary
	for normalizedName, npc := range m.npcRegistry {
		// Exclude permanently removed NPCs — they should never reappear
		if npc.IsPermanentlyRemoved() {
			continue
		}

		attitude := m.npcAttitudes[normalizedName]
		if attitude == "" {
			attitude = "neutral"
		}

		// For temporarily defeated NPCs, include the fate description
		if npc.IsTakenOut() && !npc.Fate.Permanent {
			attitude = fmt.Sprintf("defeated (%s)", npc.Fate.Description)
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

	rawResponse, err := llm.SimpleCompletion(ctx, m.engine.llmClient, promptText, 400, 0.5)
	if err != nil {
		return nil, err
	}

	// Parse the generated summary
	summary, err := prompt.ParseSceneSummary(rawResponse)
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

	rawResponse, err := llm.SimpleCompletion(ctx, m.engine.llmClient, promptText, 300, 0.3)
	if err != nil {
		return false, err
	}

	// Parse the resolution result
	result, err := prompt.ParseScenarioResolution(rawResponse)
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
	m.sessionLogger.Log("scenario_resolution_check", map[string]any{
		"is_resolved":        result.IsResolved,
		"answered_questions": result.AnsweredQuestions,
		"reasoning":          result.Reasoning,
	})

	return result.IsResolved, nil
}
