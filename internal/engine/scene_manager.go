package engine

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
)

// SceneManager handles the main scene loop and player interactions
type SceneManager struct {
	engine              *Engine
	currentScene        *scene.Scene
	player              *character.Character
	reader              *bufio.Reader
	roller              *dice.Roller
	conversationHistory []ConversationEntry
	debug               bool
	ui                  UI
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
		debug:               false,
	}
}

// SetDebug enables or disables debug mode for prompt logging
func (sm *SceneManager) SetDebug(enabled bool) {
	sm.debug = enabled
}

// SetUI sets the UI for the scene manager
func (sm *SceneManager) SetUI(ui UI) {
	sm.ui = ui
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

	sm.ui.DisplaySystemMessage("--- Scene Loop Started ---")
	sm.ui.DisplaySystemMessage("Type 'help' for commands, 'exit' to end the scene, or describe what you want to do.")

	for {
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
	inputType := sm.classifyInput(ctx, input)

	if sm.debug {
		slog.Debug("Input classified",
			"component", "scene_manager",
			"input_type", inputType,
			"input", input)
	}

	switch inputType {
	case "dialog", "clarification":
		sm.handleDialog(ctx, input)
	case "action":
		sm.handleAction(ctx, input)
	default:
		// Default to dialog if classification is unclear
		sm.handleDialog(ctx, input)
	}
}

// classifyInput uses LLM to determine if input is dialog, clarification, or action
func (sm *SceneManager) classifyInput(ctx context.Context, input string) string {
	if sm.engine.llmClient == nil {
		return "dialog" // Default fallback
	}

	// Prepare template data
	data := InputClassificationData{
		Scene:       sm.currentScene,
		PlayerInput: input,
	}

	// Execute the template
	var buf bytes.Buffer
	if err := InputClassificationPrompt.Execute(&buf, data); err != nil {
		return "dialog" // Default fallback on template error
	}

	prompt := buf.String()

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   10,
		Temperature: 0.1, // Low temperature for consistent classification
	}

	// Debug output if enabled
	if sm.debug {
		slog.Debug("Scene manager input classification LLM request",
			"component", "scene_manager",
			"prompt", prompt)
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return "dialog" // Default fallback
	}

	classification := strings.ToLower(strings.TrimSpace(resp.Choices[0].Message.Content))

	// Validate the response is one of our expected types
	validTypes := []string{"dialog", "clarification", "action"}
	for _, validType := range validTypes {
		if classification == validType {
			return classification
		}
	}

	return "dialog" // Default if classification is unexpected
}

// handleDialog processes dialog and clarification requests
func (sm *SceneManager) handleDialog(ctx context.Context, input string) {
	// Generate LLM response
	response := sm.generateSceneResponse(ctx, input, "dialog")
	if response != "" {
		sm.ui.DisplayDialog(input, response)
		// Record this exchange in conversation history
		sm.addToConversationHistory(input, response, "dialog")
	} else {
		sm.ui.DisplayDialog(input, "[Unable to generate response - check LLM connection]")
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

	// Generate narrative result with LLM
	narrative := sm.generateActionNarrative(ctx, parsedAction)
	if narrative != "" {
		sm.ui.DisplayNarrative(narrative)
	} else {
		sm.ui.DisplayNarrative("[Unable to generate narrative - check LLM connection]")
	}

	// Apply mechanical effects based on action type and outcome
	sm.applyActionEffects(parsedAction)
}

// generateSceneResponse generates an LLM response for dialog/clarification
func (sm *SceneManager) generateSceneResponse(ctx context.Context, input string, interactionType string) string {
	if sm.engine.llmClient == nil {
		return ""
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
		return "" // Return empty on template error
	}

	prompt := buf.String()

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   300,
		Temperature: 0.7,
	}

	// Debug output if enabled
	if sm.debug {
		slog.Debug("Scene manager scene response LLM request",
			"component", "scene_manager",
			"prompt", prompt)
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return ""
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content)
}

// generateActionNarrative generates narrative text for action results
func (sm *SceneManager) generateActionNarrative(ctx context.Context, parsedAction *action.Action) string {
	if sm.engine.llmClient == nil {
		return ""
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
		return "" // Return empty on template error
	}

	prompt := buf.String()

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   200,
		Temperature: 0.8,
	}

	// Debug output if enabled
	if sm.debug {
		slog.Debug("Scene manager action narrative LLM request",
			"component", "scene_manager",
			"prompt", prompt)
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return ""
	}

	narrative := strings.TrimSpace(resp.Choices[0].Message.Content)

	// Add this to conversation history as well
	actionDescription := fmt.Sprintf("Attempted: %s (Outcome: %s)", parsedAction.Description, parsedAction.Outcome.Type.String())
	sm.addToConversationHistory(actionDescription, narrative, "action")

	return narrative
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
