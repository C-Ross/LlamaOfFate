package engine

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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
}

// ConversationEntry represents a single exchange in the scene
type ConversationEntry struct {
	PlayerInput string    `json:"player_input"`
	GMResponse  string    `json:"gm_response"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"` // "dialog", "action", "clarification"
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
}

// ActionNarrativeData holds the data for action narrative template
type ActionNarrativeData struct {
	Scene               *scene.Scene
	CharacterContext    string
	AspectsContext      string
	ConversationContext string
	Action              *action.Action
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

// StartScene begins a new scene with the given description
func (sm *SceneManager) StartScene(id, name, description string, player *character.Character) error {
	sm.currentScene = scene.NewScene(id, name, description)
	sm.player = player
	sm.currentScene.AddCharacter(player.ID)
	sm.currentScene.ActiveCharacter = player.ID

	// Display scene description
	sm.displayScene()

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

	fmt.Println("\n--- Scene Loop Started ---")
	fmt.Println("Type 'help' for commands, 'exit' to end the scene, or describe what you want to do.")

	for {
		fmt.Print("\n> ")

		input, err := sm.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Check for scene exit
		if shouldExit := sm.processInput(ctx, input); shouldExit {
			break
		}
	}

	return nil
}

// processInput handles player input and returns true if the scene should end
func (sm *SceneManager) processInput(ctx context.Context, input string) bool {
	// Handle special commands first
	if sm.handleSpecialCommands(input) {
		return false
	}

	// Check for exit commands
	if sm.isExitCommand(input) {
		fmt.Println("\n--- Scene Ended ---")
		return true
	}

	// Use LLM to determine the type of input
	inputType := sm.classifyInput(ctx, input)

	switch inputType {
	case "dialog", "clarification":
		sm.handleDialog(ctx, input)
	case "action":
		sm.handleAction(ctx, input)
	default:
		// Default to dialog if classification is unclear
		sm.handleDialog(ctx, input)
	}

	return false
}

// handleSpecialCommands processes special scene commands
func (sm *SceneManager) handleSpecialCommands(input string) bool {
	parts := strings.Fields(strings.ToLower(input))
	if len(parts) == 0 {
		return false
	}

	command := parts[0]

	switch command {
	case "help":
		sm.showHelp()
		return true
	case "scene":
		sm.displayScene()
		return true
	case "character", "char":
		sm.displayCharacter()
		return true
	case "status":
		sm.displayStatus()
		return true
	case "aspects":
		sm.displayAspects()
		return true
	case "history", "conversation":
		sm.displayConversationHistory()
		return true
	}

	return false
}

// isExitCommand checks if the input is an exit command
func (sm *SceneManager) isExitCommand(input string) bool {
	exitCommands := []string{"exit", "quit", "end", "leave", "resolve"}
	lowerInput := strings.ToLower(strings.TrimSpace(input))

	for _, cmd := range exitCommands {
		if lowerInput == cmd {
			return true
		}
	}

	return false
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
	fmt.Printf("\nYou: %s\n", input)

	// Generate LLM response
	response := sm.generateSceneResponse(ctx, input, "dialog")
	if response != "" {
		fmt.Printf("\nGM: %s\n", response)
		// Record this exchange in conversation history
		sm.addToConversationHistory(input, response, "dialog")
	} else {
		fmt.Printf("\nGM: [Unable to generate response - check LLM connection]\n")
	}
}

// handleAction processes player actions
func (sm *SceneManager) handleAction(ctx context.Context, input string) {
	fmt.Printf("\nYou attempt to: %s\n", input)

	// Parse the action using the action parser
	if sm.engine.actionParser != nil {
		action, err := sm.engine.actionParser.ParseAction(ctx, ActionParseRequest{
			Character: sm.player,
			RawInput:  input,
			Context:   sm.currentScene.Description,
		})

		if err != nil {
			fmt.Printf("Failed to parse action: %v\n", err)
			return
		}

		sm.resolveAction(ctx, action)
	} else {
		fmt.Println("Action parser not available - LLM client required for action processing.")
	}
}

// resolveAction fully resolves a parsed action
func (sm *SceneManager) resolveAction(ctx context.Context, parsedAction *action.Action) {
	fmt.Printf("Action: %s using %s\n", parsedAction.Type.String(), parsedAction.Skill)

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

	// Display result
	fmt.Printf("Skill (%s): %s (%+d)\n", parsedAction.Skill, skillLevel.String(), int(skillLevel))
	if parsedAction.CalculateBonus() != 0 {
		fmt.Printf("Bonuses: %+d\n", parsedAction.CalculateBonus())
	}
	fmt.Printf("Rolled: %s (Total: %s vs Difficulty %s)\n",
		result.String(), result.FinalValue.String(), parsedAction.Difficulty.String())
	fmt.Printf("Outcome: %s\n", outcome.Type.String())

	// Generate narrative result with LLM
	narrative := sm.generateActionNarrative(ctx, parsedAction)
	if narrative != "" {
		fmt.Printf("\n%s\n", narrative)
	} else {
		fmt.Printf("\n[Unable to generate narrative - check LLM connection]\n")
	}

	// Apply mechanical effects based on action type and outcome
	sm.applyActionEffects(parsedAction)
}

// generateSceneResponse generates an LLM response for dialog/clarification
func (sm *SceneManager) generateSceneResponse(ctx context.Context, input string, interactionType string) string {
	if sm.engine.llmClient == nil {
		return ""
	}

	// Prepare template data
	data := SceneResponseData{
		Scene:               sm.currentScene,
		CharacterContext:    sm.buildCharacterContext(),
		AspectsContext:      sm.buildAspectsContext(),
		ConversationContext: sm.buildConversationContext(),
		PlayerInput:         input,
		InteractionType:     interactionType,
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

	// Prepare template data
	data := ActionNarrativeData{
		Scene:               sm.currentScene,
		CharacterContext:    sm.buildCharacterContext(),
		AspectsContext:      sm.buildAspectsContext(),
		ConversationContext: sm.buildConversationContext(),
		Action:              parsedAction,
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
			fmt.Printf("Created situation aspect: '%s' with %d free invoke(s)\n",
				aspectName, freeInvokes)
		}
	}

	// TODO: Add more action type effects (Attack, Defend, etc.)
}

// Display methods
func (sm *SceneManager) displayScene() {
	fmt.Printf("\n=== %s ===\n", sm.currentScene.Name)
	fmt.Printf("%s\n", sm.currentScene.Description)

	if len(sm.currentScene.SituationAspects) > 0 {
		fmt.Println("\nSituation Aspects:")
		for _, aspect := range sm.currentScene.SituationAspects {
			invokes := ""
			if aspect.FreeInvokes > 0 {
				invokes = fmt.Sprintf(" (%d free invoke(s))", aspect.FreeInvokes)
			}
			fmt.Printf("  - %s%s\n", aspect.Aspect, invokes)
		}
	}
}

func (sm *SceneManager) displayCharacter() {
	if sm.player == nil {
		fmt.Println("No active character.")
		return
	}

	fmt.Printf("\n=== %s ===\n", sm.player.Name)
	fmt.Printf("High Concept: %s\n", sm.player.Aspects.HighConcept)
	fmt.Printf("Trouble: %s\n", sm.player.Aspects.Trouble)

	if len(sm.player.Aspects.OtherAspects) > 0 {
		fmt.Println("Other Aspects:")
		for _, aspect := range sm.player.Aspects.OtherAspects {
			if aspect != "" {
				fmt.Printf("  - %s\n", aspect)
			}
		}
	}

	fmt.Printf("Fate Points: %d\n", sm.player.FatePoints)
}

func (sm *SceneManager) displayStatus() {
	if sm.player == nil {
		return
	}

	fmt.Println("\n=== Status ===")

	// Show stress tracks
	for trackType, track := range sm.player.StressTracks {
		fmt.Printf("%s: %s\n", strings.ToUpper(trackType[:1])+trackType[1:], track.String())
	}

	// Show consequences
	if len(sm.player.Consequences) > 0 {
		fmt.Println("\nConsequences:")
		for _, consequence := range sm.player.Consequences {
			fmt.Printf("  %s: %s\n", consequence.Type, consequence.Aspect)
		}
	}
}

func (sm *SceneManager) displayAspects() {
	fmt.Println("\n=== Available Aspects ===")

	if sm.player != nil {
		fmt.Println("Character Aspects:")
		for _, aspect := range sm.player.Aspects.GetAll() {
			fmt.Printf("  - %s\n", aspect)
		}
	}

	if len(sm.currentScene.SituationAspects) > 0 {
		fmt.Println("\nSituation Aspects:")
		for _, aspect := range sm.currentScene.SituationAspects {
			invokes := ""
			if aspect.FreeInvokes > 0 {
				invokes = fmt.Sprintf(" (%d free)", aspect.FreeInvokes)
			}
			fmt.Printf("  - %s%s\n", aspect.Aspect, invokes)
		}
	}
}

func (sm *SceneManager) showHelp() {
	fmt.Println("\n=== Scene Commands ===")
	fmt.Println("  help           - Show this help")
	fmt.Println("  scene          - Show scene description")
	fmt.Println("  character      - Show character details")
	fmt.Println("  status         - Show character status (stress, consequences)")
	fmt.Println("  aspects        - Show all available aspects")
	fmt.Println("  history        - Show recent conversation history")
	fmt.Println("  exit/quit      - End the scene")
	fmt.Println("\n=== Natural Language Input ===")
	fmt.Println("The system uses AI to understand your intent. You can:")
	fmt.Println("")
	fmt.Println("Dialog & Questions:")
	fmt.Println("  \"What do I see?\" \"Look around\" \"Examine the door\"")
	fmt.Println("  \"I say 'Hello there'\" \"Ask about the treasure\"")
	fmt.Println("")
	fmt.Println("Actions (requiring dice rolls):")
	fmt.Println("  \"Attack the goblin\" \"Sneak past the guard\"")
	fmt.Println("  \"Create an advantage by analyzing the situation\"")
	fmt.Println("  \"Overcome the obstacle by climbing\"")
	fmt.Println("")
	fmt.Println("The AI will determine whether you're asking questions,")
	fmt.Println("taking actions, or having conversations automatically!")
}

func (sm *SceneManager) displayConversationHistory() {
	fmt.Println("\n=== Recent Conversation ===")

	if len(sm.conversationHistory) == 0 {
		fmt.Println("No conversation history yet.")
		return
	}

	// Show last 5 exchanges
	start := len(sm.conversationHistory) - 5
	if start < 0 {
		start = 0
	}

	for i := start; i < len(sm.conversationHistory); i++ {
		entry := sm.conversationHistory[i]
		fmt.Printf("\n[%s] You: %s\n", entry.Type, entry.PlayerInput)
		fmt.Printf("GM: %s\n", entry.GMResponse)
	}
}

// GetCurrentScene returns the current scene
func (sm *SceneManager) GetCurrentScene() *scene.Scene {
	return sm.currentScene
}

// GetPlayer returns the current player character
func (sm *SceneManager) GetPlayer() *character.Character {
	return sm.player
}

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
