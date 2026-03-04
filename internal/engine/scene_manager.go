package engine

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/prompt"
	"github.com/C-Ross/LlamaOfFate/internal/session"
)

const (
	// Component identifier for logging
	componentSceneManager = "scene_manager"

	// Input classification types
	inputTypeDialog        = "dialog"
	inputTypeClarification = "clarification"
	inputTypeAction        = "action"
	inputTypeNarrative     = "narrative"
	inputTypeConflict      = "conflict"

	// User-facing messages
	msgLLMUnavailable = "[The mists of fate obscure my vision...]"
)

// SceneManager handles the main scene loop and player interactions
type SceneManager struct {
	llmClient           llm.LLMClient
	characters          CharacterResolver
	actionParser        ActionParser
	currentScene        *scene.Scene
	player              *character.Character
	conversationHistory []prompt.ConversationEntry
	lastTransition      *prompt.SceneTransition // Captured transition hint when scene ends
	sessionLogger       session.SessionLogger
	scenePurpose        string            // Dramatic question driving the current scene
	actions             *ActionResolver   // Generic action resolution (dice, invokes, narrative)
	conflict            *ConflictManager  // Conflict lifecycle (turns, NPC, damage, taken-out)
	challenge           *ChallengeManager // Challenge lifecycle (initiation, task resolution)

	// Scene-exit state for transitions (owned by SM, not CM).
	// CM owns exit state for conflict outcomes (player taken out).
	shouldExit     bool
	sceneEndReason SceneEndReason
}

// SetScenePurpose sets the dramatic question driving the current scene,
// used to give the response LLM awareness of the scene's goal.
func (sm *SceneManager) SetScenePurpose(purpose string) {
	sm.scenePurpose = purpose
}

// NewSceneManager creates a new scene manager with injected dependencies.
// llmClient and actionParser may be nil when running without an LLM.
// sessionLogger must not be nil; use session.NullLogger{} when logging is not needed.
func NewSceneManager(characters CharacterResolver, llmClient llm.LLMClient, actionParser ActionParser, sessionLogger session.SessionLogger) *SceneManager {
	roller := dice.NewRoller()
	var ag AspectGenerator
	if llmClient != nil {
		ag = NewAspectGenerator(llmClient)
	}

	cm := newConflictManager(llmClient, characters, sessionLogger)
	chm := newChallengeManager(llmClient, characters, sessionLogger)
	ar := newActionResolver(roller, characters, ag, sessionLogger)
	// Wire cross-references: AR ↔ CM.
	ar.conflict = cm
	ar.challenge = chm
	cm.actions = ar
	chm.actions = ar

	sm := &SceneManager{
		llmClient:           llmClient,
		characters:          characters,
		actionParser:        actionParser,
		conversationHistory: make([]prompt.ConversationEntry, 0),
		actions:             ar,
		conflict:            cm,
		challenge:           chm,
		sessionLogger:       sessionLogger,
	}
	// SceneManager implements NarrativeProvider — wire it so ActionResolver
	// can generate narrative and record conversation history.
	sm.actions.SetNarrativeProvider(sm)
	return sm
}

// SetExitOnSceneTransition configures whether the scene loop should exit on scene transition
func (sm *SceneManager) SetExitOnSceneTransition(exit bool) {
	sm.conflict.exitOnSceneTransition = exit
}

// StartScene begins a new scene with the given pre-configured scene
func (sm *SceneManager) StartScene(scene *scene.Scene, player *character.Character) error {
	sm.currentScene = scene
	sm.player = player
	sm.actions.setSceneState(scene, player)
	sm.conflict.setSceneState(scene, player)
	sm.challenge.setSceneState(scene, player)

	// Clear conversation history from previous scene — recap should only
	// appear when restoring from a save (via Restore()), not on normal transitions.
	sm.conversationHistory = nil

	// Ensure the player is in the scene
	sm.currentScene.AddCharacter(player.ID)

	// Set active character to player if not already set
	if sm.currentScene.ActiveCharacter == "" {
		sm.currentScene.ActiveCharacter = player.ID
	}

	// Log scene start
	sm.sessionLogger.Log("scene_start", map[string]any{
		"scene_name":        scene.Name,
		"scene_description": scene.Description,
		"characters":        scene.Characters,
		"player_id":         player.ID,
	})

	// Scene description will be displayed by the terminal UI when needed

	return nil
}

// HandleInput processes a single player input and returns the resulting events.
// The caller (UI) is responsible for driving the loop and rendering events.
// This is the primary entry point for event-driven UIs (e.g. web via WebSocket).
//
// If the returned InputResult has AwaitingInvoke == true, the caller must
// collect an InvokeResponse and call ProvideInvokeResponse before sending
// another HandleInput.
func (sm *SceneManager) HandleInput(ctx context.Context, input string) (*InputResult, error) {
	if sm.actions.HasPendingInvoke() {
		return nil, fmt.Errorf("HandleInput called while awaiting invoke response; call ProvideInvokeResponse instead")
	}
	if sm.actions.HasPendingMidFlow() {
		return nil, fmt.Errorf("HandleInput called while awaiting mid-flow response; call ProvideMidFlowResponse instead")
	}

	result := &InputResult{}

	// Log player input
	sm.sessionLogger.Log("player_input", map[string]any{"input": input})

	// Check for concession command during conflict (before any roll per Fate Core rules).
	if sm.currentScene.IsConflict && sm.conflict.isConcedeCommand(input) {
		result.Events = sm.conflict.handleConcession(ctx)
		if sm.actions.HasPendingMidFlow() {
			result.Events = append(result.Events, sm.actions.PendingMidFlowEvent())
			result.AwaitingMidFlow = true
		}
		sm.applySceneEnd(result)
		return result, nil
	}

	// Use LLM to determine the type of input
	inputType, err := sm.classifyInput(ctx, input)
	if err != nil {
		slog.Warn("Input classification failed, defaulting to dialog",
			"component", componentSceneManager,
			"input", input,
			"error", err)
		inputType = inputTypeDialog // Graceful fallback
	}

	// Log classification result
	sm.sessionLogger.Log("input_classification", map[string]any{
		"input":          input,
		"classification": inputType,
	})

	slog.Debug("Input classified",
		"component", componentSceneManager,
		"input_type", inputType,
		"input", input)

	switch inputType {
	case inputTypeDialog, inputTypeClarification, inputTypeNarrative:
		result.Events = sm.handleDialog(ctx, input)
	case inputTypeAction:
		events, awaiting := sm.handleAction(ctx, input)
		result.Events = events
		result.AwaitingInvoke = awaiting
	default:
		result.Events = sm.handleDialog(ctx, input)
	}

	// Check if a mid-flow prompt was emitted during processing.
	if sm.actions.HasPendingMidFlow() {
		result.Events = append(result.Events, sm.actions.PendingMidFlowEvent())
		result.AwaitingMidFlow = true
	}

	sm.applySceneEnd(result)

	return result, nil
}

// resetSceneState clears per-scene state before a new scene loop begins.
func (sm *SceneManager) resetSceneState() {
	sm.lastTransition = nil
	sm.shouldExit = false
	sm.sceneEndReason = ""
	sm.actions.resetState()
	sm.conflict.resetState()
	sm.challenge.resetState()
}

// buildSceneEndResult constructs a SceneEndResult from the current state.
// It merges SM-owned exit state (scene transitions) with CM-owned exit state
// (player taken out).
func (sm *SceneManager) buildSceneEndResult() *SceneEndResult {
	// SM's scene-transition reason takes priority when set.
	reason := sm.sceneEndReason
	if reason == "" {
		reason, _ = sm.conflict.SceneExitState()
	}

	result := &SceneEndResult{
		Reason:        reason,
		TakenOutChars: sm.conflict.GetTakenOutChars(),
	}

	if result.Reason == "" {
		result.Reason = SceneEndQuit
	}

	switch result.Reason {
	case SceneEndTransition:
		if sm.lastTransition != nil {
			result.TransitionHint = sm.lastTransition.Hint
		}
	case SceneEndPlayerTakenOut:
		_, hint := sm.conflict.SceneExitState()
		result.TransitionHint = hint
	}

	return result
}

// applySceneEnd checks whether a scene exit has been signalled — either by
// the conflict subsystem (player taken out) or by SceneManager itself (scene
// transition). If so, populates the result with SceneEnded / EndResult.
func (sm *SceneManager) applySceneEnd(result *InputResult) {
	exitRequested := sm.shouldExit || sm.conflict.SceneExitRequested()
	if exitRequested && !result.AwaitingInvoke && !result.AwaitingMidFlow {
		result.SceneEnded = true
		result.EndResult = sm.buildSceneEndResult()
	}
}

// classifyInput uses LLM to determine if input is dialog, clarification, or action
func (sm *SceneManager) classifyInput(ctx context.Context, input string) (string, error) {
	if sm.llmClient == nil {
		return "", fmt.Errorf("classifyInput: %w", llm.ErrUnavailable)
	}

	// Prepare template data and render
	data := prompt.InputClassificationData{
		Scene:       sm.currentScene,
		PlayerInput: input,
	}
	if sm.currentScene.IsChallenge && sm.currentScene.ChallengeState != nil {
		for _, task := range sm.currentScene.ChallengeState.PendingTasks() {
			data.ActiveChallengeSkills = append(data.ActiveChallengeSkills, task.Skill)
		}
	}

	promptText, err := prompt.RenderInputClassification(data)
	if err != nil {
		return "", fmt.Errorf("classifyInput: %w: %v", llm.ErrInvalidResponse, err)
	}

	content, err := llm.SimpleCompletion(ctx, sm.llmClient, promptText, 10, 0.1)
	if err != nil {
		return "", fmt.Errorf("classifyInput: %w: %v", llm.ErrUnavailable, err)
	}

	classification := prompt.ParseClassification(content)

	if classification == "" {
		slog.Warn("Could not parse classification from LLM response",
			"component", componentSceneManager,
			"raw_response", content)
		return "", fmt.Errorf("unexpected classification: %s", content)
	}

	// Validate the response is one of our expected types
	switch classification {
	case inputTypeDialog, inputTypeClarification, inputTypeAction, inputTypeNarrative:
		return classification, nil
	default:
		slog.Warn("Unexpected classification from LLM",
			"component", componentSceneManager,
			"classification", classification)
		return "", fmt.Errorf("unexpected classification: %s", classification)
	}
}

// handleDialog processes dialog and clarification requests
func (sm *SceneManager) handleDialog(ctx context.Context, input string) []GameEvent {
	var events []GameEvent

	// Generate LLM response
	response, err := sm.generateSceneResponse(ctx, input, inputTypeDialog)
	if err != nil {
		slog.Error("Dialog generation failed",
			"component", componentSceneManager,
			"input", input,
			"error", err)
		return append(events, DialogEvent{PlayerInput: input, GMResponse: msgLLMUnavailable})
	}

	// Check for markers in the response (conflict, challenge, and scene transition)
	conflictTrigger, cleanedResponse := sm.conflict.parseConflictMarker(response)
	conflictResolution, cleanedResponse := sm.conflict.parseConflictEndMarker(cleanedResponse)
	challengeTrigger, cleanedResponse := prompt.ParseChallengeMarker(cleanedResponse)
	sceneTransition, cleanedResponse := prompt.ParseSceneTransitionMarker(cleanedResponse)

	events = append(events, DialogEvent{PlayerInput: input, GMResponse: cleanedResponse})

	// Log the dialog exchange
	sm.sessionLogger.Log("dialog", map[string]any{
		"player_input": input,
		"gm_response":  cleanedResponse,
	})

	// Record this exchange in conversation history
	sm.RecordConversationEntry(input, cleanedResponse, inputTypeDialog)

	// Handle conflict initiation if triggered by NPC
	if conflictTrigger != nil && !sm.currentScene.IsConflict {
		if err := sm.conflict.initiateConflict(conflictTrigger.Type, conflictTrigger.InitiatorID); err != nil {
			slog.Warn("Failed to initiate conflict from dialog",
				"component", componentSceneManager,
				"error", err)
		} else {
			initiatorName := conflictTrigger.InitiatorID
			if char := sm.characters.GetCharacter(conflictTrigger.InitiatorID); char != nil {
				initiatorName = char.Name
			}
			events = append(events, ConflictStartEvent{
				ConflictType:  string(conflictTrigger.Type),
				InitiatorName: initiatorName,
				Participants:  sm.conflict.getParticipantInfo(),
			})
			sm.RecordConversationEntry("",
				fmt.Sprintf("[%s conflict initiated by %s]", conflictTrigger.Type, initiatorName),
				inputTypeConflict)
		}
	}

	// Handle conflict de-escalation if detected
	if conflictResolution != nil && sm.currentScene.IsConflict {
		reasonMessage := sm.conflict.resolveConflictPeacefully(conflictResolution.Reason)
		if reasonMessage != "" {
			events = append(events, ConflictEndEvent{Reason: reasonMessage})
			sm.RecordConversationEntry("",
				fmt.Sprintf("[Conflict ended — %s]", reasonMessage),
				inputTypeConflict)
		}
	}

	// Handle challenge initiation if triggered
	if challengeTrigger != nil && sm.currentScene.ActiveSceneType() == scene.SceneTypeNone {
		challengeEvents, err := sm.challenge.initiateChallenge(ctx, challengeTrigger.Description)
		if err != nil {
			slog.Warn("Failed to initiate challenge from dialog",
				"component", componentSceneManager,
				"error", err)
		} else {
			events = append(events, challengeEvents...)
			sm.RecordConversationEntry("",
				fmt.Sprintf("[Challenge started: %s]", challengeTrigger.Description),
				inputTypeDialog)
		}
	}

	// Handle scene transition if detected (suppress during active challenges)
	if sceneTransition != nil && !sm.currentScene.IsChallenge {
		events = append(events, sm.handleSceneTransition(sceneTransition)...)
	} else if sceneTransition != nil {
		slog.Warn("Suppressed scene transition during active challenge",
			"component", componentSceneManager,
			"hint", sceneTransition.Hint)
	}

	return events
}

// handleSceneTransition processes a scene transition marker
func (sm *SceneManager) handleSceneTransition(transition *prompt.SceneTransition) []GameEvent {
	// Log the scene transition
	sm.sessionLogger.Log("scene_transition", map[string]any{
		"hint": transition.Hint,
	})

	slog.Info("Scene transition detected",
		"component", componentSceneManager,
		"hint", transition.Hint)

	// Capture the transition for the caller (ScenarioManager)
	sm.lastTransition = transition

	// Mark that we should exit the scene loop (SM-owned state for transitions)
	sm.sceneEndReason = SceneEndTransition
	sm.shouldExit = true

	return []GameEvent{SceneTransitionEvent{Narrative: "", NewSceneHint: transition.Hint}}
}

// GetLastTransition returns the last scene transition that occurred, if any
func (sm *SceneManager) GetLastTransition() *prompt.SceneTransition {
	return sm.lastTransition
}

// handleAction processes player actions and returns (events, awaitingInvoke).
func (sm *SceneManager) handleAction(ctx context.Context, input string) ([]GameEvent, bool) {
	var events []GameEvent
	events = append(events, ActionAttemptEvent{Description: input})

	// Parse the action using the action parser
	if sm.actionParser != nil {
		otherCharacters, _ := sm.sceneCharacters()

		action, err := sm.actionParser.ParseAction(ctx, ActionParseRequest{
			Character:       sm.player,
			RawInput:        input,
			Context:         sm.currentScene.Description,
			Scene:           sm.currentScene,
			OtherCharacters: otherCharacters,
		})

		if err != nil {
			events = append(events, SystemMessageEvent{
				Message: fmt.Sprintf("Failed to parse action: %v", err),
			})
			return events, false
		}

		// Use stored challenge task difficulty when applicable.
		if sm.challenge != nil {
			sm.challenge.EnforceTaskDifficulty(action)
		}

		// Log the parsed action
		sm.sessionLogger.Log("action_parse", action)

		return sm.actions.resolveAction(ctx, action)
	}

	events = append(events, SystemMessageEvent{
		Message: "Action parser not available - LLM client required for action processing.",
	})
	return events, false
}

// generateSceneResponse generates an LLM response for dialog/clarification
func (sm *SceneManager) generateSceneResponse(ctx context.Context, input string, interactionType string) (string, error) {
	if sm.llmClient == nil {
		return "", fmt.Errorf("generateSceneResponse: %w", llm.ErrUnavailable)
	}

	// Get other characters in the scene
	otherCharacters, takenOutCharacters := sm.sceneCharacters()

	// Select prompt template based on active scene type.
	var promptText string
	var renderErr error

	switch sm.currentScene.ActiveSceneType() {
	case scene.SceneTypeConflict:
		promptText, renderErr = sm.buildConflictPrompt(input, otherCharacters, takenOutCharacters)
	default:
		promptText, renderErr = sm.buildScenePrompt(input, interactionType, otherCharacters, takenOutCharacters)
	}

	if renderErr != nil {
		return "", fmt.Errorf("generateSceneResponse: %w: %v", llm.ErrInvalidResponse, renderErr)
	}

	content, err := llm.SimpleCompletion(ctx, sm.llmClient, promptText, 300, 0.7)
	if err != nil {
		return "", fmt.Errorf("generateSceneResponse: %w: %v", llm.ErrUnavailable, err)
	}

	return content, nil
}

// buildConflictPrompt assembles the data for the conflict-specific response
// template and renders it. Extracted from generateSceneResponse for clarity.
func (sm *SceneManager) buildConflictPrompt(input string, otherCharacters, takenOutCharacters []*character.Character) (string, error) {
	// Build participant and character maps for conflict template
	participantMap := make(map[string]*scene.ConflictParticipant)
	for i := range sm.currentScene.ConflictState.Participants {
		p := &sm.currentScene.ConflictState.Participants[i]
		participantMap[p.CharacterID] = p
	}

	characterMap := make(map[string]*character.Character)
	characterMap[sm.player.ID] = sm.player
	for _, char := range otherCharacters {
		characterMap[char.ID] = char
	}
	for _, char := range takenOutCharacters {
		characterMap[char.ID] = char
	}

	// Determine current character name
	currentCharName := "Unknown"
	if sm.currentScene.ConflictState.CurrentTurn < len(sm.currentScene.ConflictState.InitiativeOrder) {
		currentID := sm.currentScene.ConflictState.InitiativeOrder[sm.currentScene.ConflictState.CurrentTurn]
		if char, ok := characterMap[currentID]; ok {
			currentCharName = char.Name
		}
	}

	conflictData := prompt.ConflictResponseData{
		Scene:                sm.currentScene,
		CharacterContext:     sm.buildCharacterContext(),
		AspectsContext:       sm.buildAspectsContext(),
		ConversationContext:  sm.buildConversationContext(),
		PlayerInput:          input,
		OtherCharacters:      otherCharacters,
		TakenOutCharacters:   takenOutCharacters,
		CurrentCharacterName: currentCharName,
		ParticipantMap:       participantMap,
		CharacterMap:         characterMap,
		ScenePurpose:         sm.scenePurpose,
	}

	return prompt.RenderConflictResponse(conflictData)
}

// buildScenePrompt assembles the data for the standard scene response template
// and renders it. Extracted from generateSceneResponse for clarity.
func (sm *SceneManager) buildScenePrompt(input, interactionType string, otherCharacters, takenOutCharacters []*character.Character) (string, error) {
	data := prompt.SceneResponseData{
		Scene:               sm.currentScene,
		CharacterContext:    sm.buildCharacterContext(),
		AspectsContext:      sm.buildAspectsContext(),
		ConversationContext: sm.buildConversationContext(),
		PlayerInput:         input,
		InteractionType:     interactionType,
		OtherCharacters:     otherCharacters,
		TakenOutCharacters:  takenOutCharacters,
		ScenePurpose:        sm.scenePurpose,
	}

	return prompt.RenderSceneResponse(data)
}

// GenerateActionNarrative generates narrative text for action results.
// Implements NarrativeProvider.
func (sm *SceneManager) GenerateActionNarrative(ctx context.Context, parsedAction *action.Action) (string, error) {
	if sm.llmClient == nil {
		return "", fmt.Errorf("generateActionNarrative: %w", llm.ErrUnavailable)
	}

	// Get other characters in the scene
	otherCharacters, _ := sm.sceneCharacters()

	// Prepare template data and render
	data := prompt.ActionNarrativeData{
		Scene:               sm.currentScene,
		CharacterContext:    sm.buildCharacterContext(),
		AspectsContext:      sm.buildAspectsContext(),
		ConversationContext: sm.buildConversationContext(),
		Action:              parsedAction,
		OtherCharacters:     otherCharacters,
	}

	// Add challenge context if a challenge is active and this skill matches a task
	if sm.currentScene.IsChallenge && sm.currentScene.ChallengeState != nil {
		data.ChallengeDescription = sm.currentScene.ChallengeState.Description
		if task := sm.currentScene.ChallengeState.FindTaskBySkill(parsedAction.Skill); task != nil {
			data.ChallengeTaskDesc = task.Description
		}
	}

	promptText, err := prompt.RenderActionNarrative(data)
	if err != nil {
		return "", fmt.Errorf("generateActionNarrative: %w: %v", llm.ErrInvalidResponse, err)
	}

	narrative, err := llm.SimpleCompletion(ctx, sm.llmClient, promptText, 200, 0.8)
	if err != nil {
		return "", fmt.Errorf("generateActionNarrative: %w: %v", llm.ErrUnavailable, err)
	}

	// Add this to conversation history as well
	actionDescription := fmt.Sprintf("Attempted: %s (Outcome: %s)", parsedAction.Description, parsedAction.Outcome.Type.String())
	sm.RecordConversationEntry(actionDescription, narrative, inputTypeAction)

	return narrative, nil
}

// GetCurrentScene returns the current scene
func (sm *SceneManager) GetCurrentScene() *scene.Scene {
	return sm.currentScene
}

// sceneCharacters returns non-player characters in the current scene, partitioned
// into active and taken-out slices. Callers that only need active characters can
// ignore the second return value.
func (sm *SceneManager) sceneCharacters() (active, takenOut []*character.Character) {
	others := sm.characters.GetCharactersByScene(sm.currentScene)
	delete(others, sm.player.ID)
	for _, c := range others {
		if sm.currentScene.IsCharacterTakenOut(c.ID) {
			takenOut = append(takenOut, c)
		} else {
			active = append(active, c)
		}
	}
	return
}

// GetPlayer returns the current player character
func (sm *SceneManager) GetPlayer() *character.Character {
	return sm.player
}

// GetConversationHistory returns the conversation history
func (sm *SceneManager) GetConversationHistory() []prompt.ConversationEntry {
	return sm.conversationHistory
}

// ProvideInvokeResponse delegates to ActionResolver and wraps the result
// with scene-end detection. This is the public API called by ScenarioManager.
func (sm *SceneManager) ProvideInvokeResponse(ctx context.Context, resp InvokeResponse) (*InputResult, error) {
	result, err := sm.actions.ProvideInvokeResponse(ctx, resp)
	if err != nil {
		return nil, err
	}
	sm.applySceneEnd(result)
	return result, nil
}

// ProvideMidFlowResponse delegates to ActionResolver and wraps the result
// with scene-end detection. This is the public API called by ScenarioManager.
func (sm *SceneManager) ProvideMidFlowResponse(ctx context.Context, resp MidFlowResponse) (*InputResult, error) {
	result, err := sm.actions.ProvideMidFlowResponse(ctx, resp)
	if err != nil {
		return nil, err
	}
	sm.applySceneEnd(result)
	return result, nil
}

// HasPendingInvoke delegates to ActionResolver.
func (sm *SceneManager) HasPendingInvoke() bool {
	return sm.actions.HasPendingInvoke()
}

// Ensure SceneManager implements the SceneInfo interface
var _ SceneInfo = (*SceneManager)(nil)

// Ensure SceneManager satisfies NarrativeProvider.
var _ NarrativeProvider = (*SceneManager)(nil)

// Snapshot returns a SceneState capturing the current scene-level state
// for persistence. This includes the scene itself (with any active conflict),
// the conversation history, and the scene's dramatic purpose.
func (sm *SceneManager) Snapshot() SceneState {
	// Copy conversation history to avoid aliasing
	history := make([]prompt.ConversationEntry, len(sm.conversationHistory))
	copy(history, sm.conversationHistory)

	return SceneState{
		CurrentScene:        sm.currentScene,
		ConversationHistory: history,
		ScenePurpose:        sm.scenePurpose,
	}
}

// Restore sets the scene manager's state from a previously saved SceneState,
// enabling mid-scene (and mid-conflict) resume. The player must be provided
// separately since it lives in ScenarioState, not SceneState.
func (sm *SceneManager) Restore(state SceneState, player *character.Character) {
	sm.currentScene = state.CurrentScene
	sm.player = player
	sm.conversationHistory = state.ConversationHistory
	sm.scenePurpose = state.ScenePurpose
	sm.conflict.setSceneState(state.CurrentScene, player)
	sm.challenge.setSceneState(state.CurrentScene, player)
	sm.actions.setSceneState(state.CurrentScene, player)

	// Ensure player is in the scene (defensive — should already be from saved state)
	if sm.currentScene != nil {
		sm.currentScene.AddCharacter(player.ID)
		if sm.currentScene.ActiveCharacter == "" {
			sm.currentScene.ActiveCharacter = player.ID
		}
	}
}

// RecordConversationEntry adds an exchange to the conversation history.
// Implements NarrativeProvider.
func (sm *SceneManager) RecordConversationEntry(playerInput, gmResponse, interactionType string) {
	entry := prompt.ConversationEntry{
		PlayerInput: playerInput,
		GMResponse:  gmResponse,
		Timestamp:   time.Now(),
		Type:        interactionType,
	}

	sm.conversationHistory = append(sm.conversationHistory, entry)

	// Keep only the last 10 exchanges to avoid overly long context
	if len(sm.conversationHistory) > 10 {
		sm.conversationHistory = sm.conversationHistory[len(sm.conversationHistory)-10:]
	}
}

// buildConversationContext builds a summary of recent conversation
func (sm *SceneManager) buildConversationContext() string {
	if len(sm.conversationHistory) == 0 {
		return "No previous conversation in this scene."
	}

	var context strings.Builder
	context.WriteString("Recent exchanges:\n")

	// Show last 5 exchanges
	start := len(sm.conversationHistory) - 5
	if start < 0 {
		start = 0
	}

	for i := start; i < len(sm.conversationHistory); i++ {
		entry := sm.conversationHistory[i]
		context.WriteString(fmt.Sprintf("Player: %s\nGM: %s\n", entry.PlayerInput, entry.GMResponse))
	}

	return context.String()
}

// BuildMechanicalNarrative creates a basic narrative from action data when LLM is unavailable.
// Implements NarrativeProvider.
func (sm *SceneManager) BuildMechanicalNarrative(a *action.Action) string {
	if a.Outcome == nil {
		return fmt.Sprintf("Your attempt to %s...", a.Description)
	}

	switch a.Outcome.Type {
	case dice.SuccessWithStyle:
		return fmt.Sprintf("Your attempt to %s succeeds brilliantly!", a.Description)
	case dice.Success:
		return fmt.Sprintf("Your attempt to %s succeeds.", a.Description)
	case dice.Tie:
		return fmt.Sprintf("Your attempt to %s partially succeeds.", a.Description)
	case dice.Failure:
		return fmt.Sprintf("Your attempt to %s fails.", a.Description)
	default:
		return fmt.Sprintf("Your attempt to %s completes.", a.Description)
	}
}

// buildCharacterContext builds character information for LLM context
func (sm *SceneManager) buildCharacterContext() string {
	if sm.player == nil {
		return "No active character."
	}

	var context strings.Builder
	context.WriteString(fmt.Sprintf("Name: %s\n", sm.player.Name))
	context.WriteString(fmt.Sprintf("High Concept: %s\n", sm.player.Aspects.HighConcept))
	context.WriteString(fmt.Sprintf("Trouble: %s\n", sm.player.Aspects.Trouble))

	if len(sm.player.Aspects.OtherAspects) > 0 {
		context.WriteString("Other Aspects: ")
		for i, aspect := range sm.player.Aspects.OtherAspects {
			if aspect != "" {
				if i > 0 {
					context.WriteString(", ")
				}
				context.WriteString(aspect)
			}
		}
		context.WriteString("\n")
	}

	return context.String()
}

// buildAspectsContext builds available aspects for LLM context
func (sm *SceneManager) buildAspectsContext() string {
	var context strings.Builder

	// Character aspects
	if sm.player != nil {
		aspects := sm.player.Aspects.GetAll()
		if len(aspects) > 0 {
			context.WriteString("Character Aspects: ")
			context.WriteString(strings.Join(aspects, ", "))
			context.WriteString("\n")
		}
	}

	// Situation aspects
	if len(sm.currentScene.SituationAspects) > 0 {
		context.WriteString("Situation Aspects: ")
		var situationAspects []string
		for _, aspect := range sm.currentScene.SituationAspects {
			aspectText := aspect.Aspect
			if aspect.FreeInvokes > 0 {
				aspectText += fmt.Sprintf(" (%d free invokes)", aspect.FreeInvokes)
			}
			situationAspects = append(situationAspects, aspectText)
		}
		context.WriteString(strings.Join(situationAspects, ", "))
		context.WriteString("\n")
	}

	if context.Len() == 0 {
		return "No special aspects currently in play."
	}

	return context.String()
}
