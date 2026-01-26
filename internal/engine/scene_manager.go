package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
)

const (
	// Component identifier for logging
	componentSceneManager = "scene_manager"

	// Input classification types
	inputTypeDialog        = "dialog"
	inputTypeClarification = "clarification"
	inputTypeAction        = "action"

	// User-facing messages
	msgLLMUnavailable = "[The mists of fate obscure my vision...]"
)

// conflictMarkerRegex matches [CONFLICT:type:character_id] markers
var conflictMarkerRegex = regexp.MustCompile(`\[CONFLICT:(physical|mental):([^\]]+)\]`)

// ConflictTrigger represents a detected conflict initiation
type ConflictTrigger struct {
	Type        scene.ConflictType
	InitiatorID string
}

// SceneManager handles the main scene loop and player interactions
type SceneManager struct {
	engine               *Engine
	currentScene         *scene.Scene
	player               *character.Character
	reader               *bufio.Reader
	roller               *dice.Roller
	conversationHistory  []ConversationEntry
	ui                   UI
	shouldExit           bool // Set to true when the game should end
	exitOnSceneTransition bool // Set to true to exit the loop on scene transition
}

// InputClassificationData holds the data for input classification template
type InputClassificationData struct {
	Scene       *scene.Scene
	PlayerInput string
}

// SceneResponseData holds the data for scene response template
type SceneResponseData struct {
	Scene               *scene.Scene
	CharacterContext    string
	AspectsContext      string
	ConversationContext string
	PlayerInput         string
	InteractionType     string
	OtherCharacters     []*character.Character
}

// ActionNarrativeData holds the data for action narrative template
type ActionNarrativeData struct {
	Scene               *scene.Scene
	CharacterContext    string
	AspectsContext      string
	ConversationContext string
	Action              *action.Action
	OtherCharacters     []*character.Character
}

// NewSceneManager creates a new scene manager
func NewSceneManager(engine *Engine) *SceneManager {
	return &SceneManager{
		engine:              engine,
		reader:              bufio.NewReader(os.Stdin),
		roller:              dice.NewRoller(),
		conversationHistory: make([]ConversationEntry, 0),
	}
}

// SetUI sets the UI for the scene manager
func (sm *SceneManager) SetUI(ui UI) {
	sm.ui = ui
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

	// Scene description will be displayed by the terminal UI when needed

	return nil
}

// RunSceneLoop starts the interactive scene loop
func (sm *SceneManager) RunSceneLoop(ctx context.Context) error {
	if sm.currentScene == nil {
		return fmt.Errorf("no active scene")
	}

	if sm.engine.llmClient == nil {
		return fmt.Errorf("LLM client is required for scene loop functionality")
	}

	if sm.ui == nil {
		return fmt.Errorf("UI is required for scene loop functionality")
	}

	// Display the initial scene
	sm.ui.DisplaySystemMessage(fmt.Sprintf("=== %s ===", sm.currentScene.Name))
	sm.ui.DisplayNarrative(sm.currentScene.Description)
	sm.ui.DisplaySystemMessage("Type 'help' for commands, 'exit' to end.")

	for {
		// Check if game should end (e.g., game over from being taken out)
		if sm.shouldExit {
			break
		}

		input, isExit, err := sm.ui.ReadInput()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		if input == "" {
			continue
		}

		// Check for scene exit
		if isExit {
			break
		}

		sm.processInput(ctx, input)
	}

	return nil
}

// processInput handles player input
func (sm *SceneManager) processInput(ctx context.Context, input string) {
	// Use LLM to determine the type of input
	inputType, err := sm.classifyInput(ctx, input)
	if err != nil {
		slog.Warn("Input classification failed, defaulting to dialog",
			"component", componentSceneManager,
			"input", input,
			"error", err)
		inputType = inputTypeDialog // Graceful fallback
	}

	slog.Debug("Input classified",
		"component", componentSceneManager,
		"input_type", inputType,
		"input", input)

	switch inputType {
	case inputTypeDialog, inputTypeClarification:
		sm.handleDialog(ctx, input)
	case inputTypeAction:
		sm.handleAction(ctx, input)
	default:
		// Default to dialog if classification is unclear
		sm.handleDialog(ctx, input)
	}
}

// classifyInput uses LLM to determine if input is dialog, clarification, or action
func (sm *SceneManager) classifyInput(ctx context.Context, input string) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("classifyInput: %w", ErrLLMUnavailable)
	}

	// Prepare template data
	data := InputClassificationData{
		Scene:       sm.currentScene,
		PlayerInput: input,
	}

	// Execute the template
	var buf bytes.Buffer
	if err := InputClassificationPrompt.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("classifyInput: %w: %v", ErrLLMInvalidResponse, err)
	}

	prompt := buf.String()

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   10,
		Temperature: 0.1, // Low temperature for consistent classification
	}

	// Debug output
	slog.Debug("Scene manager input classification LLM request",
		"component", componentSceneManager,
		"prompt", prompt)

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("classifyInput: %w: %v", ErrLLMUnavailable, err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("classifyInput: %w", ErrLLMInvalidResponse)
	}

	classification := strings.ToLower(strings.TrimSpace(resp.Choices[0].Message.Content))

	// Validate the response is one of our expected types
	switch classification {
	case inputTypeDialog, inputTypeClarification, inputTypeAction:
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

	// Check for conflict markers in the response
	conflictTrigger, cleanedResponse := sm.parseConflictMarker(response)

	sm.ui.DisplayDialog(input, cleanedResponse)
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

		sm.resolveAction(ctx, action)
	} else {
		sm.ui.DisplaySystemMessage("Action parser not available - LLM client required for action processing.")
	}
}

// resolveAction fully resolves a parsed action
func (sm *SceneManager) resolveAction(ctx context.Context, parsedAction *action.Action) {
	// Check if this action should initiate or escalate a conflict
	if parsedAction.Type == action.Attack {
		actionConflictType := sm.getConflictTypeForSkill(parsedAction.Skill)

		if !sm.currentScene.IsConflict {
			// Auto-initiate conflict for attack actions
			if err := sm.initiateConflict(actionConflictType, sm.player.ID); err != nil {
				slog.Warn("Failed to auto-initiate conflict",
					"component", componentSceneManager,
					"error", err)
			}
		} else if sm.currentScene.ConflictState.Type != actionConflictType {
			// Escalate conflict if type changes
			sm.handleConflictEscalation(actionConflictType)
		}
	}

	// Get character's skill level
	skillLevel := sm.player.GetSkill(parsedAction.Skill)

	// Calculate total bonus
	totalBonus := int(skillLevel) + parsedAction.CalculateBonus()

	// Roll dice
	result := sm.roller.RollWithModifier(dice.Mediocre, totalBonus)
	parsedAction.CheckResult = result

	// Determine outcome
	outcome := result.CompareAgainst(parsedAction.Difficulty)
	parsedAction.Outcome = outcome

	// Display result using UI
	resultString := fmt.Sprintf("%s (Total: %s vs Difficulty %s)",
		result.String(), result.FinalValue.String(), parsedAction.Difficulty.String())
	sm.ui.DisplayActionResult(parsedAction.Skill,
		fmt.Sprintf("%s (%+d)", skillLevel.String(), int(skillLevel)),
		parsedAction.CalculateBonus(),
		resultString,
		outcome.Type.String())

	// Generate narrative with error handling
	narrative, err := sm.generateActionNarrative(ctx, parsedAction)
	if err != nil {
		slog.Error("Action narrative generation failed",
			"component", componentSceneManager,
			"action_id", parsedAction.ID,
			"error", err)
		// Provide mechanical fallback
		narrative = sm.buildMechanicalNarrative(parsedAction)
	}
	sm.ui.DisplayNarrative(narrative)

	// Apply mechanical effects based on action type and outcome
	sm.applyActionEffects(parsedAction)

	// If we're in a conflict, advance turn and process NPC turns
	if sm.currentScene.IsConflict {
		sm.advanceConflictTurns(ctx)
	}
}

// generateSceneResponse generates an LLM response for dialog/clarification
func (sm *SceneManager) generateSceneResponse(ctx context.Context, input string, interactionType string) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("generateSceneResponse: %w", ErrLLMUnavailable)
	}

	// Get other characters in the scene
	otherCharactersMap := sm.engine.GetCharactersByScene(sm.currentScene)
	delete(otherCharactersMap, sm.player.ID) // Remove the player

	var otherCharacters []*character.Character
	for _, char := range otherCharactersMap {
		otherCharacters = append(otherCharacters, char)
	}

	// Prepare template data
	data := SceneResponseData{
		Scene:               sm.currentScene,
		CharacterContext:    sm.buildCharacterContext(),
		AspectsContext:      sm.buildAspectsContext(),
		ConversationContext: sm.buildConversationContext(),
		PlayerInput:         input,
		InteractionType:     interactionType,
		OtherCharacters:     otherCharacters,
	}

	// Execute the template
	var buf bytes.Buffer
	if err := SceneResponsePrompt.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("generateSceneResponse: %w: %v", ErrLLMInvalidResponse, err)
	}

	prompt := buf.String()

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   300,
		Temperature: 0.7,
	}

	// Debug output
	slog.Debug("Scene manager scene response LLM request",
		"component", componentSceneManager,
		"prompt", prompt)

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("generateSceneResponse: %w: %v", ErrLLMUnavailable, err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("generateSceneResponse: %w", ErrLLMInvalidResponse)
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// generateActionNarrative generates narrative text for action results
func (sm *SceneManager) generateActionNarrative(ctx context.Context, parsedAction *action.Action) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("generateActionNarrative: %w", ErrLLMUnavailable)
	}

	// Get other characters in the scene
	otherCharactersMap := sm.engine.GetCharactersByScene(sm.currentScene)
	delete(otherCharactersMap, sm.player.ID) // Remove the player

	var otherCharacters []*character.Character
	for _, char := range otherCharactersMap {
		otherCharacters = append(otherCharacters, char)
	}

	// Prepare template data
	data := ActionNarrativeData{
		Scene:               sm.currentScene,
		CharacterContext:    sm.buildCharacterContext(),
		AspectsContext:      sm.buildAspectsContext(),
		ConversationContext: sm.buildConversationContext(),
		Action:              parsedAction,
		OtherCharacters:     otherCharacters,
	}

	// Execute the template
	var buf bytes.Buffer
	if err := ActionNarrativePrompt.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("generateActionNarrative: %w: %v", ErrLLMInvalidResponse, err)
	}

	prompt := buf.String()

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   200,
		Temperature: 0.8,
	}

	// Debug output
	slog.Debug("Scene manager action narrative LLM request",
		"component", componentSceneManager,
		"prompt", prompt)

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("generateActionNarrative: %w: %v", ErrLLMUnavailable, err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("generateActionNarrative: %w", ErrLLMInvalidResponse)
	}

	narrative := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Add this to conversation history as well
	actionDescription := fmt.Sprintf("Attempted: %s (Outcome: %s)", parsedAction.Description, parsedAction.Outcome.Type.String())
	sm.addToConversationHistory(actionDescription, narrative, inputTypeAction)

	return narrative, nil
}

// applyActionEffects applies mechanical effects based on action results
func (sm *SceneManager) applyActionEffects(parsedAction *action.Action) {
	if parsedAction.Outcome == nil {
		return
	}

	switch parsedAction.Type {
	case action.CreateAdvantage:
		if parsedAction.IsSuccess() {
			aspectName := fmt.Sprintf("Advantage from %s", parsedAction.Description)
			freeInvokes := 1
			if parsedAction.IsSuccessWithStyle() {
				freeInvokes = 2
			}

			situationAspect := scene.NewSituationAspect(
				fmt.Sprintf("aspect-%d", time.Now().UnixNano()),
				aspectName,
				sm.player.ID,
				freeInvokes,
			)

			sm.currentScene.AddSituationAspect(situationAspect)
			sm.ui.DisplaySystemMessage(fmt.Sprintf("Created situation aspect: '%s' with %d free invoke(s)",
				aspectName, freeInvokes))
		}
	}

	// TODO: Add more action type effects (Attack, Defend, etc.)
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
func (sm *SceneManager) GetConversationHistory() []ConversationEntry {
	return sm.conversationHistory
}

// Ensure SceneManager implements the SceneInfo interface
var _ SceneInfo = (*SceneManager)(nil)

// addToConversationHistory adds an exchange to the conversation history
func (sm *SceneManager) addToConversationHistory(playerInput, gmResponse, interactionType string) {
	entry := ConversationEntry{
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

// parseConflictMarker extracts a conflict trigger from LLM response and returns cleaned text
func (sm *SceneManager) parseConflictMarker(response string) (*ConflictTrigger, string) {
	matches := conflictMarkerRegex.FindStringSubmatch(response)
	if matches == nil {
		return nil, response
	}

	conflictType := scene.PhysicalConflict
	if matches[1] == "mental" {
		conflictType = scene.MentalConflict
	}

	trigger := &ConflictTrigger{
		Type:        conflictType,
		InitiatorID: strings.TrimSpace(matches[2]),
	}

	// Remove the marker from the response and clean up any double spaces
	cleanedResponse := conflictMarkerRegex.ReplaceAllString(response, "")
	// Replace multiple spaces with single space
	cleanedResponse = strings.Join(strings.Fields(cleanedResponse), " ")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	return trigger, cleanedResponse
}

// initiateConflict starts a conflict with all characters in the scene
func (sm *SceneManager) initiateConflict(conflictType scene.ConflictType, initiatorID string) error {
	if sm.currentScene.IsConflict {
		return fmt.Errorf("already in a conflict")
	}

	// Build participants from all characters in the scene
	participants := make([]scene.ConflictParticipant, 0)

	for _, charID := range sm.currentScene.Characters {
		char := sm.engine.GetCharacter(charID)
		if char == nil {
			continue
		}

		// Calculate initiative based on conflict type
		initiative := sm.calculateInitiative(char, conflictType)

		participants = append(participants, scene.ConflictParticipant{
			CharacterID: charID,
			Initiative:  initiative,
			Status:      scene.StatusActive,
		})
	}

	if len(participants) < 2 {
		return fmt.Errorf("conflict requires at least 2 participants")
	}

	// Sort by initiative (descending)
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].Initiative > participants[j].Initiative
	})

	sm.currentScene.StartConflictWithInitiator(conflictType, participants, initiatorID)

	// Re-sort initiative order after StartConflict
	sm.sortInitiativeOrder()

	// Display conflict start
	initiatorName := initiatorID
	if char := sm.engine.GetCharacter(initiatorID); char != nil {
		initiatorName = char.Name
	}

	sm.ui.DisplayConflictStart(string(conflictType), initiatorName, sm.getParticipantInfo())

	slog.Info("Conflict initiated",
		"component", componentSceneManager,
		"type", conflictType,
		"initiator", initiatorID,
		"participants", len(participants))

	return nil
}

// calculateInitiative returns the initiative value for a character based on conflict type
func (sm *SceneManager) calculateInitiative(char *character.Character, conflictType scene.ConflictType) int {
	// Physical: Notice, then Athletics, then Physique
	// Mental: Empathy, then Rapport, then Will
	if conflictType == scene.PhysicalConflict {
		initiative := int(char.GetSkill("Notice"))
		if initiative == 0 {
			initiative = int(char.GetSkill("Athletics"))
		}
		return initiative
	}

	// Mental conflict
	initiative := int(char.GetSkill("Empathy"))
	if initiative == 0 {
		initiative = int(char.GetSkill("Rapport"))
	}
	return initiative
}

// sortInitiativeOrder sorts the initiative order by participant initiative values
func (sm *SceneManager) sortInitiativeOrder() {
	if sm.currentScene.ConflictState == nil {
		return
	}

	// Sort participants by initiative
	sort.Slice(sm.currentScene.ConflictState.Participants, func(i, j int) bool {
		return sm.currentScene.ConflictState.Participants[i].Initiative >
			sm.currentScene.ConflictState.Participants[j].Initiative
	})

	// Rebuild initiative order from sorted participants
	sm.currentScene.ConflictState.InitiativeOrder = make([]string, 0)
	for _, p := range sm.currentScene.ConflictState.Participants {
		if p.Status == scene.StatusActive {
			sm.currentScene.ConflictState.InitiativeOrder = append(
				sm.currentScene.ConflictState.InitiativeOrder, p.CharacterID)
		}
	}
}

// recalculateInitiative recalculates initiative for all participants based on conflict type
func (sm *SceneManager) recalculateInitiative(conflictType scene.ConflictType) {
	if sm.currentScene.ConflictState == nil {
		return
	}

	for i := range sm.currentScene.ConflictState.Participants {
		p := &sm.currentScene.ConflictState.Participants[i]
		char := sm.engine.GetCharacter(p.CharacterID)
		if char != nil {
			p.Initiative = sm.calculateInitiative(char, conflictType)
		}
	}

	sm.sortInitiativeOrder()
}

// handleConflictEscalation changes the conflict type and recalculates initiative
func (sm *SceneManager) handleConflictEscalation(newType scene.ConflictType) {
	if !sm.currentScene.IsConflict {
		return
	}

	currentType := sm.currentScene.ConflictState.Type
	if currentType == newType {
		return
	}

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"The conflict escalates from %s to %s!", currentType, newType))

	sm.currentScene.EscalateConflict(newType)
	sm.recalculateInitiative(newType)
}

// advanceConflictTurns advances through turns and processes NPC actions until it's the player's turn
func (sm *SceneManager) advanceConflictTurns(ctx context.Context) {
	if !sm.currentScene.IsConflict || sm.currentScene.ConflictState == nil {
		return
	}

	// Advance past the player's turn
	sm.currentScene.NextTurn()

	// Process NPC turns until we get back to the player or conflict ends
	for sm.currentScene.IsConflict {
		currentActor := sm.currentScene.GetCurrentActor()
		if currentActor == "" {
			break
		}

		// If it's the player's turn, stop and let them act
		if currentActor == sm.player.ID {
			sm.ui.DisplayTurnAnnouncement(sm.player.Name, sm.currentScene.ConflictState.Round, true)
			break
		}

		// Process NPC turn
		sm.processNPCTurn(ctx, currentActor)

		// Advance to next turn
		sm.currentScene.NextTurn()
	}
}

// processNPCTurn handles an NPC's action during conflict
func (sm *SceneManager) processNPCTurn(ctx context.Context, npcID string) {
	npc := sm.engine.GetCharacter(npcID)
	if npc == nil {
		slog.Warn("NPC not found for turn processing",
			"component", componentSceneManager,
			"npc_id", npcID)
		return
	}

	sm.ui.DisplayTurnAnnouncement(npc.Name, sm.currentScene.ConflictState.Round, false)

	// Determine attack skill based on conflict type
	attackSkill := "Fight" // Default for physical
	defenseSkill := "Athletics"
	if sm.currentScene.ConflictState.Type == scene.MentalConflict {
		attackSkill = "Provoke"
		defenseSkill = "Will"
	}

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(attackSkill)

	// Roll NPC's attack
	npcRoll := sm.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))

	// Get player's defense
	playerDefenseLevel := sm.player.GetSkill(defenseSkill)
	playerDefense := sm.roller.RollWithModifier(dice.Mediocre, int(playerDefenseLevel))

	// Compare results
	outcome := npcRoll.CompareAgainst(playerDefense.FinalValue)

	// Display the mechanical result
	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s attacks with %s (%s) vs your %s (%s)",
		npc.Name,
		attackSkill,
		npcRoll.FinalValue.String(),
		defenseSkill,
		playerDefense.FinalValue.String(),
	))

	// Generate narrative for the attack
	npcNarrative, err := sm.generateNPCAttackNarrative(ctx, npc, attackSkill, outcome)
	if err != nil {
		slog.Error("Failed to generate NPC attack narrative",
			"component", componentSceneManager,
			"npc_id", npcID,
			"error", err)
		// Fallback narrative
		if outcome.Type == dice.Success || outcome.Type == dice.SuccessWithStyle {
			npcNarrative = fmt.Sprintf("%s's attack hits!", npc.Name)
		} else {
			npcNarrative = fmt.Sprintf("%s's attack misses.", npc.Name)
		}
	}
	sm.ui.DisplayNarrative(npcNarrative)

	// Apply stress if the attack hit
	if outcome.Type == dice.Success || outcome.Type == dice.SuccessWithStyle {
		shifts := outcome.Shifts
		if shifts < 1 {
			shifts = 1
		}
		stressType := character.PhysicalStress
		if sm.currentScene.ConflictState.Type == scene.MentalConflict {
			stressType = character.MentalStress
		}

		// Try to absorb with stress track
		absorbed := sm.player.TakeStress(stressType, shifts)
		if absorbed {
			sm.ui.DisplaySystemMessage(fmt.Sprintf(
				"You take %d %s stress! (%s)",
				shifts,
				stressType,
				sm.player.StressTracks[string(stressType)].String(),
			))
		} else {
			// Cannot absorb - need consequence or taken out
			sm.handleStressOverflow(ctx, shifts, stressType, npc)
		}
	} else if outcome.Type == dice.Tie {
		sm.ui.DisplaySystemMessage("The attack is deflected, but grants a boost!")
	} else {
		sm.ui.DisplaySystemMessage("You successfully defend!")
	}
}

// handleStressOverflow handles when the player cannot absorb stress with their stress track
func (sm *SceneManager) handleStressOverflow(ctx context.Context, shifts int, stressType character.StressTrackType, attacker *character.Character) {
	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"You cannot absorb %d shifts with your stress track!",
		shifts,
	))

	// Determine available consequences
	availableConsequences := sm.getAvailableConsequences(shifts)

	if len(availableConsequences) == 0 {
		// No consequences available - must concede or be taken out
		sm.ui.DisplaySystemMessage("You have no available consequences!")
		sm.handleTakenOutOrConcede(ctx, attacker)
		return
	}

	// Display options
	sm.ui.DisplaySystemMessage("\nYou must choose how to handle this:")
	sm.ui.DisplaySystemMessage("  1. Concede - You choose to lose on your terms")

	for i, conseq := range availableConsequences {
		sm.ui.DisplaySystemMessage(fmt.Sprintf(
			"  %d. Take a %s consequence (absorbs %d shifts)",
			i+2, conseq.Type, conseq.Value,
		))
	}
	sm.ui.DisplaySystemMessage(fmt.Sprintf("  %d. Be Taken Out - Your opponent decides your fate", len(availableConsequences)+2))

	// Read player choice
	sm.ui.DisplaySystemMessage("\nEnter your choice (number):")
	input, _, err := sm.ui.ReadInput()
	if err != nil {
		slog.Error("Failed to read consequence choice", "error", err)
		sm.handleTakenOutOrConcede(ctx, attacker)
		return
	}

	choice := strings.TrimSpace(input)

	// Parse choice
	if choice == "1" {
		sm.handleConcession(ctx)
		return
	}

	// Check for consequence choices
	for i, conseq := range availableConsequences {
		if choice == fmt.Sprintf("%d", i+2) {
			sm.applyConsequence(ctx, conseq.Type, shifts, attacker)
			return
		}
	}

	// Check for taken out choice
	if choice == fmt.Sprintf("%d", len(availableConsequences)+2) {
		sm.handleTakenOut(ctx, attacker)
		return
	}

	// Invalid choice - default to taken out
	sm.ui.DisplaySystemMessage("Invalid choice. You are taken out!")
	sm.handleTakenOut(ctx, attacker)
}

// ConsequenceOption represents an available consequence the player can take
type ConsequenceOption struct {
	Type  character.ConsequenceType
	Value int
}

// getAvailableConsequences returns consequences that can absorb the given shifts
func (sm *SceneManager) getAvailableConsequences(shifts int) []ConsequenceOption {
	available := []ConsequenceOption{}

	// Check which consequence slots are available
	hasMild := false
	hasModerate := false
	hasSevere := false

	for _, c := range sm.player.Consequences {
		switch c.Type {
		case character.MildConsequence:
			hasMild = true
		case character.ModerateConsequence:
			hasModerate = true
		case character.SevereConsequence:
			hasSevere = true
		}
	}

	// Add available consequences that can help absorb damage
	// In Fate, you can take a consequence even if it doesn't fully absorb the damage
	// (you'd also need to use stress boxes for the remainder)
	if !hasMild {
		available = append(available, ConsequenceOption{
			Type:  character.MildConsequence,
			Value: character.MildConsequence.Value(),
		})
	}
	if !hasModerate {
		available = append(available, ConsequenceOption{
			Type:  character.ModerateConsequence,
			Value: character.ModerateConsequence.Value(),
		})
	}
	if !hasSevere {
		available = append(available, ConsequenceOption{
			Type:  character.SevereConsequence,
			Value: character.SevereConsequence.Value(),
		})
	}

	return available
}

// applyConsequence applies a consequence to the player character
func (sm *SceneManager) applyConsequence(ctx context.Context, conseqType character.ConsequenceType, shifts int, attacker *character.Character) {
	// Generate a consequence aspect via LLM
	aspectName, err := sm.generateConsequenceAspect(ctx, conseqType, attacker)
	if err != nil {
		slog.Error("Failed to generate consequence aspect", "error", err)
		aspectName = fmt.Sprintf("%s Wound", strings.Title(string(conseqType)))
	}

	consequence := character.Consequence{
		ID:        fmt.Sprintf("conseq-%d", time.Now().UnixNano()),
		Type:      conseqType,
		Aspect:    aspectName,
		Duration:  string(conseqType),
		CreatedAt: time.Now(),
	}

	sm.player.AddConsequence(consequence)

	absorbed := conseqType.Value()
	remaining := shifts - absorbed

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"\nYou take a %s consequence: \"%s\"",
		conseqType, aspectName,
	))
	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"The consequence absorbs %d shifts.",
		absorbed,
	))

	// If there are remaining shifts, try to absorb with stress
	if remaining > 0 {
		stressType := character.PhysicalStress
		if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
			stressType = character.MentalStress
		}

		if sm.player.TakeStress(stressType, remaining) {
			sm.ui.DisplaySystemMessage(fmt.Sprintf(
				"You absorb the remaining %d shifts as stress. (%s)",
				remaining,
				sm.player.StressTracks[string(stressType)].String(),
			))
		} else {
			sm.ui.DisplaySystemMessage(fmt.Sprintf(
				"You cannot absorb the remaining %d shifts! You may need another consequence.",
				remaining,
			))
			// Recursively handle remaining damage
			sm.handleStressOverflow(ctx, remaining, stressType, attacker)
		}
	}
}

// generateConsequenceAspect uses LLM to generate a consequence aspect
func (sm *SceneManager) generateConsequenceAspect(ctx context.Context, conseqType character.ConsequenceType, attacker *character.Character) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("LLM client required")
	}

	conflictType := "physical"
	if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	prompt := fmt.Sprintf(`Generate a short Fate Core consequence aspect for this situation:

Character: %s
Attacker: %s
Consequence Severity: %s
Conflict Type: %s

Generate ONLY the aspect name (2-4 words), something that represents lasting harm appropriate to the severity. Examples:
- Mild physical: "Bruised Ribs", "Sprained Ankle", "Rattled"
- Moderate physical: "Broken Arm", "Deep Gash", "Concussed"  
- Severe physical: "Crushed Leg", "Internal Bleeding", "Near Death"
- Mild mental: "Shaken Confidence", "Momentary Doubt", "Rattled Nerves"
- Moderate mental: "Crisis of Faith", "Deep Shame", "Paranoid"
- Severe mental: "Broken Spirit", "Traumatized", "Lost All Hope"

Response should be ONLY the aspect name, nothing else.`,
		sm.player.Name,
		attacker.Name,
		conseqType,
		conflictType,
	)

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   20,
		Temperature: 0.8,
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// handleConcession handles when the player concedes the conflict
func (sm *SceneManager) handleConcession(ctx context.Context) {
	sm.ui.DisplaySystemMessage("\n=== You Concede! ===")
	sm.ui.DisplaySystemMessage("You choose to lose the conflict on your own terms.")
	sm.ui.DisplaySystemMessage("You get to narrate how you exit the scene and avoid the worst consequences.")

	// Award a fate point for conceding
	sm.player.GainFatePoint()
	sm.ui.DisplaySystemMessage(fmt.Sprintf("You gain a Fate Point for conceding! (Now: %d)", sm.player.FatePoints))

	// Mark player as conceded and end the conflict
	if sm.currentScene.ConflictState != nil {
		sm.currentScene.SetParticipantStatus(sm.player.ID, scene.StatusConceded)
		sm.currentScene.EndConflict()
		sm.ui.DisplayConflictEnd("You have conceded the conflict.")
	}

	sm.ui.DisplaySystemMessage("\nDescribe how you concede and exit the conflict:")
}

// TakenOutResult represents the outcome classification of being taken out
type TakenOutResult int

const (
	TakenOutContinue   TakenOutResult = iota // Continue in same scene (knocked down, stunned, etc.)
	TakenOutTransition                       // Transition to new scene (captured, driven out, etc.)
	TakenOutGameOver                         // Game ending (death, permanent incapacitation)
)

// handleTakenOut handles when the player is taken out
func (sm *SceneManager) handleTakenOut(ctx context.Context, attacker *character.Character) {
	sm.ui.DisplaySystemMessage("\n=== You Are Taken Out! ===")
	sm.ui.DisplaySystemMessage(fmt.Sprintf("%s decides your fate.", attacker.Name))

	// Generate narrative and outcome classification for being taken out
	narrative, outcome, newSceneHint, err := sm.generateTakenOutNarrativeAndOutcome(ctx, attacker)
	if err != nil {
		narrative = fmt.Sprintf("You collapse, defeated by %s.", attacker.Name)
		outcome = TakenOutTransition
		newSceneHint = "You awaken later, unsure of your fate..."
	}

	// Mark player as taken out and end the conflict
	if sm.currentScene.ConflictState != nil {
		sm.currentScene.SetParticipantStatus(sm.player.ID, scene.StatusTakenOut)
		sm.currentScene.EndConflict()
	}

	// Handle based on outcome type
	switch outcome {
	case TakenOutGameOver:
		sm.ui.DisplayNarrative(narrative)
		sm.ui.DisplayGameOver(fmt.Sprintf("%s has met their end.", sm.player.Name))
		sm.shouldExit = true

	case TakenOutTransition:
		sm.ui.DisplaySceneTransition(narrative, newSceneHint)
		sm.ui.DisplaySystemMessage("\nThe scene shifts around you...")
		if sm.exitOnSceneTransition {
			sm.shouldExit = true
		}
		// Scene continues but context has changed

	default: // TakenOutContinue
		sm.ui.DisplayNarrative(narrative)
		sm.ui.DisplayConflictEnd(fmt.Sprintf("%s has won the conflict.", attacker.Name))
	}
}

// handleTakenOutOrConcede when there are no options left
func (sm *SceneManager) handleTakenOutOrConcede(ctx context.Context, attacker *character.Character) {
	sm.ui.DisplaySystemMessage("\nYou have no way to absorb this damage!")
	sm.ui.DisplaySystemMessage("  1. Concede (you choose how you lose)")
	sm.ui.DisplaySystemMessage("  2. Be Taken Out (opponent chooses)")
	sm.ui.DisplaySystemMessage("\nEnter your choice:")

	input, _, err := sm.ui.ReadInput()
	if err != nil || strings.TrimSpace(input) != "1" {
		sm.handleTakenOut(ctx, attacker)
		return
	}

	sm.handleConcession(ctx)
}

// generateTakenOutNarrativeAndOutcome generates narrative and classifies the outcome
func (sm *SceneManager) generateTakenOutNarrativeAndOutcome(ctx context.Context, attacker *character.Character) (narrative string, outcome TakenOutResult, newSceneHint string, err error) {
	if sm.engine.llmClient == nil {
		return "", TakenOutTransition, "", fmt.Errorf("LLM client required")
	}

	conflictType := "physical"
	if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	prompt := fmt.Sprintf(`The player character %s has been taken out in a %s conflict by %s.

As the GM, decide their fate and provide a narrative. The attacker gets to decide what happens.

Consider the context:
- Scene: %s
- Attacker: %s (%s)
- Conflict type: %s

Respond in JSON format:
{
  "narrative": "A 2-3 sentence dramatic narrative describing what happens to the defeated character",
  "outcome": "one of: game_over, scene_transition, continue",
  "new_scene_hint": "If scene_transition, a brief hint about the new situation (e.g., 'You awaken in a dungeon cell...'). Empty string otherwise."
}

Outcome guidelines:
- "game_over": Only for truly fatal or permanently ending situations (death, soul destruction, etc.). Use sparingly!
- "scene_transition": Character is captured, driven off, knocked unconscious and moved, etc. Most common outcome.
- "continue": Character is knocked down but scene continues (rare, usually for dramatic moments).

Prefer scene_transition for most defeats. Reserve game_over only when death/permanent end is truly appropriate to the story.`,
		sm.player.Name,
		conflictType,
		attacker.Name,
		sm.currentScene.Description,
		attacker.Name,
		attacker.Aspects.HighConcept,
		conflictType,
	)

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   200,
		Temperature: 0.7,
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil {
		return "", TakenOutTransition, "", err
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", TakenOutTransition, "", fmt.Errorf("empty response")
	}

	// Parse the JSON response
	type takenOutResponse struct {
		Narrative    string `json:"narrative"`
		Outcome      string `json:"outcome"`
		NewSceneHint string `json:"new_scene_hint"`
	}

	content := cleanJSONResponse(resp.Choices[0].Message.Content)
	var parsed takenOutResponse
	if parseErr := json.Unmarshal([]byte(content), &parsed); parseErr != nil {
		// If parsing fails, use the raw content as narrative and default to transition
		slog.Warn("Failed to parse taken out response as JSON, using raw content",
			"error", parseErr,
			"content", content,
		)
		return strings.TrimSpace(resp.Choices[0].Message.Content), TakenOutTransition, "You awaken later...", nil
	}

	// Map outcome string to enum
	switch strings.ToLower(parsed.Outcome) {
	case "game_over":
		outcome = TakenOutGameOver
	case "continue":
		outcome = TakenOutContinue
	default:
		outcome = TakenOutTransition
	}

	return parsed.Narrative, outcome, parsed.NewSceneHint, nil
}

// generateNPCAttackNarrative generates narrative for an NPC's attack
func (sm *SceneManager) generateNPCAttackNarrative(ctx context.Context, npc *character.Character, skill string, outcome *dice.Outcome) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("LLM client required for NPC narratives")
	}

	outcomeDesc := "misses"
	switch outcome.Type {
	case dice.SuccessWithStyle:
		outcomeDesc = fmt.Sprintf("hits with devastating effect (%d shifts)", outcome.Shifts)
	case dice.Success:
		outcomeDesc = fmt.Sprintf("hits (%d shifts)", outcome.Shifts)
	case dice.Tie:
		outcomeDesc = "is barely deflected, granting a boost"
	case dice.Failure:
		outcomeDesc = "misses"
	}

	prompt := fmt.Sprintf(`Generate a brief (1-2 sentence) narrative description for this attack in a Fate Core RPG:

Attacker: %s (%s)
Skill: %s
Target: %s
Result: The attack %s

Be dramatic and describe the action cinematically. Do not mention dice or mechanical terms.`,
		npc.Name,
		npc.Aspects.HighConcept,
		skill,
		sm.player.Name,
		outcomeDesc,
	)

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   100,
		Temperature: 0.8,
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// getConflictTypeForSkill determines the conflict type based on the skill used
func (sm *SceneManager) getConflictTypeForSkill(skill string) scene.ConflictType {
	// Physical skills
	physicalSkills := map[string]bool{
		"Fight": true, "Shoot": true, "Athletics": true, "Physique": true,
	}

	// Mental skills
	mentalSkills := map[string]bool{
		"Provoke": true, "Deceive": true, "Rapport": true, "Will": true, "Empathy": true,
	}

	if physicalSkills[skill] {
		return scene.PhysicalConflict
	}
	if mentalSkills[skill] {
		return scene.MentalConflict
	}

	// Default to physical for unknown skills
	return scene.PhysicalConflict
}

// getParticipantInfo returns information about all conflict participants for display
func (sm *SceneManager) getParticipantInfo() []ConflictParticipantInfo {
	if sm.currentScene.ConflictState == nil {
		return nil
	}

	info := make([]ConflictParticipantInfo, 0, len(sm.currentScene.ConflictState.Participants))
	for _, p := range sm.currentScene.ConflictState.Participants {
		name := p.CharacterID
		if char := sm.engine.GetCharacter(p.CharacterID); char != nil {
			name = char.Name
		}
		info = append(info, ConflictParticipantInfo{
			CharacterID:   p.CharacterID,
			CharacterName: name,
			Initiative:    p.Initiative,
			IsPlayer:      p.CharacterID == sm.player.ID,
		})
	}
	return info
}
