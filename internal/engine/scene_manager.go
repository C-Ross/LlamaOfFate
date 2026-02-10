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

	// User-facing messages
	msgLLMUnavailable = "[The mists of fate obscure my vision...]"
)

// SceneManager handles the main scene loop and player interactions
type SceneManager struct {
	engine                *Engine
	currentScene          *scene.Scene
	player                *character.Character
	roller                *dice.Roller
	conversationHistory   []prompt.ConversationEntry
	ui                    UI
	shouldExit            bool                    // Set to true when the game should end
	exitOnSceneTransition bool                    // Set to true to exit the loop on scene transition
	lastTransition        *prompt.SceneTransition // Captured transition hint when scene ends
	aspectGenerator       *AspectGenerator
	sessionLogger         *session.Logger
	takenOutChars         []string       // Characters taken out during this scene
	sceneEndReason        SceneEndReason // Why the scene ended
	playerTakenOutHint    string         // Transition hint if player was taken out
	scenePurpose          string         // Dramatic question driving the current scene
}

// SetScenePurpose sets the dramatic question driving the current scene,
// used to give the response LLM awareness of the scene's goal.
func (sm *SceneManager) SetScenePurpose(purpose string) {
	sm.scenePurpose = purpose
}

// NewSceneManager creates a new scene manager
func NewSceneManager(engine *Engine) *SceneManager {
	sm := &SceneManager{
		engine:              engine,
		roller:              dice.NewRoller(),
		conversationHistory: make([]prompt.ConversationEntry, 0),
	}
	// Initialize aspect generator if LLM client is available
	if engine.llmClient != nil {
		sm.aspectGenerator = NewAspectGenerator(engine.llmClient)
	}
	return sm
}

// SetUI sets the UI for the scene manager.
// If the UI implements SceneInfoSetter, it also sets the scene info
// so the UI can display character and scene status.
func (sm *SceneManager) SetUI(ui UI) {
	sm.ui = ui
	if setter, ok := ui.(SceneInfoSetter); ok {
		setter.SetSceneInfo(sm)
	}
}

// SetSessionLogger sets the session logger for recording game transcripts
func (sm *SceneManager) SetSessionLogger(logger *session.Logger) {
	sm.sessionLogger = logger
}

// SetExitOnSceneTransition configures whether the scene loop should exit on scene transition
func (sm *SceneManager) SetExitOnSceneTransition(exit bool) {
	sm.exitOnSceneTransition = exit
}

// StartScene begins a new scene with the given pre-configured scene
func (sm *SceneManager) StartScene(scene *scene.Scene, player *character.Character) error {
	sm.currentScene = scene
	sm.player = player

	// Ensure the player is in the scene
	sm.currentScene.AddCharacter(player.ID)

	// Set active character to player if not already set
	if sm.currentScene.ActiveCharacter == "" {
		sm.currentScene.ActiveCharacter = player.ID
	}

	// Log scene start
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("scene_start", map[string]any{
			"scene_name":        scene.Name,
			"scene_description": scene.Description,
			"characters":        scene.Characters,
			"player_id":         player.ID,
		})
	}

	// Scene description will be displayed by the terminal UI when needed

	return nil
}

// RunSceneLoop starts the interactive scene loop
func (sm *SceneManager) RunSceneLoop(ctx context.Context) (*SceneEndResult, error) {
	if sm.currentScene == nil {
		return nil, fmt.Errorf("no active scene")
	}

	if sm.engine.llmClient == nil {
		return nil, fmt.Errorf("LLM client is required for scene loop functionality")
	}

	if sm.ui == nil {
		return nil, fmt.Errorf("UI is required for scene loop functionality")
	}

	// Reset scene-specific state
	sm.takenOutChars = nil
	sm.sceneEndReason = ""
	sm.playerTakenOutHint = ""
	sm.lastTransition = nil
	sm.shouldExit = false

	// Display the initial scene
	sm.ui.DisplaySystemMessage(fmt.Sprintf("=== %s ===", sm.currentScene.Name))
	sm.ui.DisplayNarrative(sm.currentScene.Description)
	sm.ui.DisplaySystemMessage("Type 'help' for commands, 'exit' to end.")

	for !sm.shouldExit {
		input, isExit, err := sm.ui.ReadInput()
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}

		if input == "" {
			continue
		}

		// Check for scene exit
		if isExit {
			sm.sceneEndReason = SceneEndQuit
			break
		}

		sm.processInput(ctx, input)
	}

	// Build the scene end result
	result := &SceneEndResult{
		Reason:        sm.sceneEndReason,
		TakenOutChars: sm.takenOutChars,
	}

	// If no reason was set but we exited, default to quit
	if result.Reason == "" {
		result.Reason = SceneEndQuit
	}

	// Add transition hint based on reason
	switch result.Reason {
	case SceneEndTransition:
		if sm.lastTransition != nil {
			result.TransitionHint = sm.lastTransition.Hint
		}
	case SceneEndPlayerTakenOut:
		result.TransitionHint = sm.playerTakenOutHint
	}

	return result, nil
}

// processInput handles player input
func (sm *SceneManager) processInput(ctx context.Context, input string) {
	// Log player input
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("player_input", map[string]any{"input": input})
	}

	// Handle meta-commands locally (no LLM round-trip)
	if sm.handleMetaCommand(input) {
		return
	}

	// Check for concession command during conflict (before any roll per Fate Core rules)
	if sm.currentScene.IsConflict && sm.isConcedeCommand(input) {
		sm.handleConcession(ctx)
		return
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
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("input_classification", map[string]any{
			"input":          input,
			"classification": inputType,
		})
	}

	slog.Debug("Input classified",
		"component", componentSceneManager,
		"input_type", inputType,
		"input", input)

	switch inputType {
	case inputTypeDialog, inputTypeClarification, inputTypeNarrative:
		sm.handleDialog(ctx, input)
	case inputTypeAction:
		sm.handleAction(ctx, input)
	default:
		// Default to dialog if classification is unclear
		sm.handleDialog(ctx, input)
	}
}

// handleMetaCommand checks for local commands that don't need the LLM.
// Returns true if the input was handled as a meta-command.
func (sm *SceneManager) handleMetaCommand(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	switch lower {
	case "help", "?":
		sm.displayHelp()
	case "scene":
		sm.ui.DisplaySystemMessage(fmt.Sprintf("=== %s ===", sm.currentScene.Name))
		sm.ui.DisplayNarrative(sm.currentScene.Description)
		if len(sm.currentScene.SituationAspects) > 0 {
			sm.ui.DisplaySystemMessage("Situation Aspects:")
			for _, a := range sm.currentScene.SituationAspects {
				sm.ui.DisplaySystemMessage(fmt.Sprintf("  - \"%s\"", a.Aspect))
			}
		}
	case "character", "char", "me":
		sm.ui.DisplayCharacter()
	case "status":
		sm.ui.DisplayCharacter()
		if sm.currentScene.IsConflict {
			sm.ui.DisplaySystemMessage("You are in a conflict.")
		}
	default:
		return false
	}
	return true
}

// displayHelp shows available commands
func (sm *SceneManager) displayHelp() {
	sm.ui.DisplaySystemMessage("Commands:")
	sm.ui.DisplaySystemMessage("  help, ?      - Show this help message")
	sm.ui.DisplaySystemMessage("  scene        - Redisplay the current scene")
	sm.ui.DisplaySystemMessage("  character    - Show your character sheet")
	sm.ui.DisplaySystemMessage("  status       - Show character and scene status")
	sm.ui.DisplaySystemMessage("  exit, quit   - Leave the game")
	if sm.currentScene.IsConflict {
		sm.ui.DisplaySystemMessage("  concede      - Concede the current conflict")
	}
	sm.ui.DisplaySystemMessage("")
	sm.ui.DisplaySystemMessage("Or type naturally to interact with the scene.")
}

// classifyInput uses LLM to determine if input is dialog, clarification, or action
func (sm *SceneManager) classifyInput(ctx context.Context, input string) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("classifyInput: %w", llm.ErrUnavailable)
	}

	// Prepare template data and render
	data := prompt.InputClassificationData{
		Scene:       sm.currentScene,
		PlayerInput: input,
	}

	promptText, err := prompt.RenderInputClassification(data)
	if err != nil {
		return "", fmt.Errorf("classifyInput: %w: %v", llm.ErrInvalidResponse, err)
	}

	content, err := llm.SimpleCompletion(ctx, sm.engine.llmClient, promptText, 10, 0.1)
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
func (sm *SceneManager) handleDialog(ctx context.Context, input string) {
	// Generate LLM response
	response, err := sm.generateSceneResponse(ctx, input, inputTypeDialog)
	if err != nil {
		slog.Error("Dialog generation failed",
			"component", componentSceneManager,
			"input", input,
			"error", err)
		sm.ui.DisplayDialog(input, msgLLMUnavailable)
		return
	}

	// Check for markers in the response (conflict escalation, de-escalation, and scene transition)
	conflictTrigger, cleanedResponse := sm.parseConflictMarker(response)
	conflictResolution, cleanedResponse := sm.parseConflictEndMarker(cleanedResponse)
	sceneTransition, cleanedResponse := prompt.ParseSceneTransitionMarker(cleanedResponse)

	sm.ui.DisplayDialog(input, cleanedResponse)

	// Log the dialog exchange
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("dialog", map[string]any{
			"player_input": input,
			"gm_response":  cleanedResponse,
		})
	}

	// Record this exchange in conversation history
	sm.addToConversationHistory(input, cleanedResponse, inputTypeDialog)

	// Handle conflict initiation if triggered by NPC
	if conflictTrigger != nil && !sm.currentScene.IsConflict {
		if err := sm.initiateConflict(conflictTrigger.Type, conflictTrigger.InitiatorID); err != nil {
			slog.Warn("Failed to initiate conflict from dialog",
				"component", componentSceneManager,
				"error", err)
		}
	}

	// Handle conflict de-escalation if detected
	if conflictResolution != nil && sm.currentScene.IsConflict {
		sm.resolveConflictPeacefully(conflictResolution.Reason)
	}

	// Handle scene transition if detected
	if sceneTransition != nil {
		sm.handleSceneTransition(sceneTransition)
	}
}

// handleSceneTransition processes a scene transition marker
func (sm *SceneManager) handleSceneTransition(transition *prompt.SceneTransition) {
	// Log the scene transition
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("scene_transition", map[string]any{
			"hint": transition.Hint,
		})
	}

	slog.Info("Scene transition detected",
		"component", componentSceneManager,
		"hint", transition.Hint)

	// Capture the transition for the caller (ScenarioManager)
	sm.lastTransition = transition

	// Display the transition to the player
	sm.ui.DisplaySceneTransition("", transition.Hint)

	// Mark that we should exit the scene loop
	sm.sceneEndReason = SceneEndTransition
	sm.shouldExit = true
}

// GetLastTransition returns the last scene transition that occurred, if any
func (sm *SceneManager) GetLastTransition() *prompt.SceneTransition {
	return sm.lastTransition
}

// handleAction processes player actions
func (sm *SceneManager) handleAction(ctx context.Context, input string) {
	sm.ui.DisplayActionAttempt(input)

	// Parse the action using the action parser
	if sm.engine.actionParser != nil {
		// Get other characters in the scene from the engine's registry
		otherCharactersMap := sm.engine.GetCharactersByScene(sm.currentScene)
		// Remove the player from other characters (they're already the main character)
		delete(otherCharactersMap, sm.player.ID)

		// Convert map to slice for ActionParseRequest
		var otherCharacters []*character.Character
		for _, char := range otherCharactersMap {
			otherCharacters = append(otherCharacters, char)
		}

		action, err := sm.engine.actionParser.ParseAction(ctx, ActionParseRequest{
			Character:       sm.player,
			RawInput:        input,
			Context:         sm.currentScene.Description,
			Scene:           sm.currentScene,
			OtherCharacters: otherCharacters,
		})

		if err != nil {
			sm.ui.DisplaySystemMessage(fmt.Sprintf("Failed to parse action: %v", err))
			return
		}

		// Log the parsed action
		if sm.sessionLogger != nil {
			sm.sessionLogger.Log("action_parse", action)
		}

		sm.resolveAction(ctx, action)
	} else {
		sm.ui.DisplaySystemMessage("Action parser not available - LLM client required for action processing.")
	}
}

// generateSceneResponse generates an LLM response for dialog/clarification
func (sm *SceneManager) generateSceneResponse(ctx context.Context, input string, interactionType string) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("generateSceneResponse: %w", llm.ErrUnavailable)
	}

	// Get other characters in the scene
	otherCharactersMap := sm.engine.GetCharactersByScene(sm.currentScene)
	delete(otherCharactersMap, sm.player.ID) // Remove the player

	// Separate active characters from taken-out characters
	var otherCharacters []*character.Character
	var takenOutCharacters []*character.Character
	for _, char := range otherCharactersMap {
		if sm.currentScene.IsCharacterTakenOut(char.ID) {
			takenOutCharacters = append(takenOutCharacters, char)
		} else {
			otherCharacters = append(otherCharacters, char)
		}
	}

	var promptText string
	var renderErr error

	// Use conflict-specific template when in conflict
	if sm.currentScene.IsConflict && sm.currentScene.ConflictState != nil {
		// Build participant and character maps for conflict template
		participantMap := make(map[string]*scene.ConflictParticipant)
		for i := range sm.currentScene.ConflictState.Participants {
			p := &sm.currentScene.ConflictState.Participants[i]
			participantMap[p.CharacterID] = p
		}

		characterMap := make(map[string]*character.Character)
		characterMap[sm.player.ID] = sm.player
		for id, char := range otherCharactersMap {
			characterMap[id] = char
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

		promptText, renderErr = prompt.RenderConflictResponse(conflictData)
	} else {
		// Use standard scene response template
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

		promptText, renderErr = prompt.RenderSceneResponse(data)
	}

	if renderErr != nil {
		return "", fmt.Errorf("generateSceneResponse: %w: %v", llm.ErrInvalidResponse, renderErr)
	}

	content, err := llm.SimpleCompletion(ctx, sm.engine.llmClient, promptText, 300, 0.7)
	if err != nil {
		return "", fmt.Errorf("generateSceneResponse: %w: %v", llm.ErrUnavailable, err)
	}

	return content, nil
}

// generateActionNarrative generates narrative text for action results
func (sm *SceneManager) generateActionNarrative(ctx context.Context, parsedAction *action.Action) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("generateActionNarrative: %w", llm.ErrUnavailable)
	}

	// Get other characters in the scene
	otherCharactersMap := sm.engine.GetCharactersByScene(sm.currentScene)
	delete(otherCharactersMap, sm.player.ID) // Remove the player

	var otherCharacters []*character.Character
	for _, char := range otherCharactersMap {
		otherCharacters = append(otherCharacters, char)
	}

	// Prepare template data and render
	data := prompt.ActionNarrativeData{
		Scene:               sm.currentScene,
		CharacterContext:    sm.buildCharacterContext(),
		AspectsContext:      sm.buildAspectsContext(),
		ConversationContext: sm.buildConversationContext(),
		Action:              parsedAction,
		OtherCharacters:     otherCharacters,
	}

	promptText, err := prompt.RenderActionNarrative(data)
	if err != nil {
		return "", fmt.Errorf("generateActionNarrative: %w: %v", llm.ErrInvalidResponse, err)
	}

	narrative, err := llm.SimpleCompletion(ctx, sm.engine.llmClient, promptText, 200, 0.8)
	if err != nil {
		return "", fmt.Errorf("generateActionNarrative: %w: %v", llm.ErrUnavailable, err)
	}

	// Add this to conversation history as well
	actionDescription := fmt.Sprintf("Attempted: %s (Outcome: %s)", parsedAction.Description, parsedAction.Outcome.Type.String())
	sm.addToConversationHistory(actionDescription, narrative, inputTypeAction)

	return narrative, nil
}

// GetCurrentScene returns the current scene
func (sm *SceneManager) GetCurrentScene() *scene.Scene {
	return sm.currentScene
}

// GetPlayer returns the current player character
func (sm *SceneManager) GetPlayer() *character.Character {
	return sm.player
}

// GetConversationHistory returns the conversation history
func (sm *SceneManager) GetConversationHistory() []prompt.ConversationEntry {
	return sm.conversationHistory
}

// Ensure SceneManager implements the SceneInfo interface
var _ SceneInfo = (*SceneManager)(nil)

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

// addToConversationHistory adds an exchange to the conversation history
func (sm *SceneManager) addToConversationHistory(playerInput, gmResponse, interactionType string) {
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

// buildMechanicalNarrative creates a basic narrative from action data when LLM is unavailable
func (sm *SceneManager) buildMechanicalNarrative(a *action.Action) string {
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
