package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/session"
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

// SceneManager handles the main scene loop and player interactions
type SceneManager struct {
	engine                *Engine
	currentScene          *scene.Scene
	player                *character.Character
	reader                *bufio.Reader
	roller                *dice.Roller
	conversationHistory   []ConversationEntry
	ui                    UI
	shouldExit            bool // Set to true when the game should end
	exitOnSceneTransition bool // Set to true to exit the loop on scene transition
	aspectGenerator       *AspectGenerator
	sessionLogger         *session.Logger
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
	TakenOutCharacters  []*character.Character // Characters defeated earlier in this scene
}

// ConflictResponseData holds the data for conflict response template
type ConflictResponseData struct {
	Scene                *scene.Scene
	CharacterContext     string
	AspectsContext       string
	ConversationContext  string
	PlayerInput          string
	OtherCharacters      []*character.Character
	TakenOutCharacters   []*character.Character // Characters defeated earlier in this scene
	CurrentCharacterName string
	ParticipantMap       map[string]*scene.ConflictParticipant
	CharacterMap         map[string]*character.Character
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

// AttackContext holds information about the attack that caused damage
type AttackContext struct {
	Skill       string // The skill used to attack (e.g., "Fight", "Shoot", "Provoke")
	Description string // The narrative description of the attack
	Shifts      int    // The shifts of damage dealt
}

// ConsequenceAspectData holds the data for consequence aspect generation template
type ConsequenceAspectData struct {
	CharacterName string
	AttackerName  string
	Severity      string
	ConflictType  string
	AttackContext
}

// TakenOutData holds the data for taken out narrative template
type TakenOutData struct {
	CharacterName       string
	AttackerName        string
	AttackerHighConcept string
	ConflictType        string
	SceneDescription    string
	AttackContext
}

// NewSceneManager creates a new scene manager
func NewSceneManager(engine *Engine) *SceneManager {
	sm := &SceneManager{
		engine:              engine,
		reader:              bufio.NewReader(os.Stdin),
		roller:              dice.NewRoller(),
		conversationHistory: make([]ConversationEntry, 0),
	}
	// Initialize aspect generator if LLM client is available
	if engine.llmClient != nil {
		sm.aspectGenerator = NewAspectGenerator(engine.llmClient)
	}
	return sm
}

// SetUI sets the UI for the scene manager
func (sm *SceneManager) SetUI(ui UI) {
	sm.ui = ui
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

	for !sm.shouldExit {
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
	// Log player input
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("player_input", map[string]any{"input": input})
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

	// Prepare template data and render
	data := InputClassificationData{
		Scene:       sm.currentScene,
		PlayerInput: input,
	}

	prompt, err := RenderInputClassification(data)
	if err != nil {
		return "", fmt.Errorf("classifyInput: %w: %v", ErrLLMInvalidResponse, err)
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   10,
		Temperature: 0.1, // Low temperature for consistent classification
	})
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

	// Check for conflict markers in the response (both escalation and de-escalation)
	conflictTrigger, cleanedResponse := sm.parseConflictMarker(response)
	conflictResolution, cleanedResponse := sm.parseConflictEndMarker(cleanedResponse)

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

// resolveAction fully resolves a parsed action
func (sm *SceneManager) resolveAction(ctx context.Context, parsedAction *action.Action) {
	// Check if this action should initiate or escalate a conflict
	if parsedAction.Type == action.Attack {
		actionConflictType := core.ConflictTypeForSkill(parsedAction.Skill)

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

	// For attacks against characters, use active defense instead of static difficulty
	var defenseResult *dice.CheckResult
	var targetChar *character.Character
	if parsedAction.Type == action.Attack && parsedAction.Target != "" {
		targetChar = sm.engine.GetCharacter(parsedAction.Target)
		if targetChar != nil {
			defenseResult = sm.rollTargetDefense(targetChar, parsedAction.Skill)
			parsedAction.Difficulty = defenseResult.FinalValue
		}
	}

	// Display initial result
	var resultString string
	if defenseResult != nil && targetChar != nil {
		resultString = fmt.Sprintf("%s (Total: %s vs %s's Defense %s)",
			result.String(), result.FinalValue.String(), targetChar.Name, defenseResult.FinalValue.String())
	} else {
		resultString = fmt.Sprintf("%s (Total: %s vs Difficulty %s)",
			result.String(), result.FinalValue.String(), parsedAction.Difficulty.String())
	}
	initialOutcome := result.CompareAgainst(parsedAction.Difficulty)
	sm.ui.DisplayActionResult(parsedAction.Skill,
		fmt.Sprintf("%s (%+d)", skillLevel.String(), int(skillLevel)),
		parsedAction.CalculateBonus(),
		resultString,
		initialOutcome.Type.String())

	// Log the dice roll
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("dice_roll", map[string]any{
			"skill":       parsedAction.Skill,
			"skill_level": int(skillLevel),
			"bonus":       parsedAction.CalculateBonus(),
			"roll_result": result.String(),
			"final_value": int(result.FinalValue),
			"difficulty":  int(parsedAction.Difficulty),
			"outcome":     initialOutcome.Type.String(),
			"shifts":      initialOutcome.Shifts,
		})
	}

	// Post-roll invoke opportunity
	result = sm.handlePostRollInvokes(result, parsedAction.Difficulty, parsedAction, false)
	parsedAction.CheckResult = result

	// Determine final outcome after invokes
	outcome := result.CompareAgainst(parsedAction.Difficulty)
	parsedAction.Outcome = outcome

	// Display updated outcome if it changed
	if outcome.Type != initialOutcome.Type {
		sm.ui.DisplaySystemMessage(fmt.Sprintf("Final outcome: %s", outcome.Type.String()))
	}

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

	// Log the narrative
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("narrative", map[string]any{
			"text":    narrative,
			"action":  parsedAction.Type.String(),
			"outcome": outcome.Type.String(),
		})
	}

	// Apply mechanical effects based on action type and outcome
	sm.applyActionEffects(parsedAction, targetChar)

	// If we're in a conflict, advance turn and process NPC turns
	if sm.currentScene.IsConflict {
		sm.advanceConflictTurns(ctx)
	}
}

// rollTargetDefense rolls an active defense for a target character
func (sm *SceneManager) rollTargetDefense(target *character.Character, attackSkill string) *dice.CheckResult {
	// Determine defense skill based on attack skill type
	defenseSkill := core.DefenseSkillForAttack(attackSkill)
	defenseLevel := target.GetSkill(defenseSkill)

	// Roll defense
	defenseRoll := sm.roller.RollWithModifier(dice.Mediocre, int(defenseLevel))

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s defends with %s (%s)",
		target.Name, defenseSkill, defenseRoll.FinalValue.String()))

	return defenseRoll
}

// gatherInvokableAspects collects all aspects available for the player to invoke
func (sm *SceneManager) gatherInvokableAspects(usedAspects map[string]bool) []InvokableAspect {
	var aspects []InvokableAspect

	// Character aspects (High Concept, Trouble, Other)
	for _, aspectText := range sm.player.Aspects.GetAll() {
		if aspectText == "" {
			continue
		}
		aspects = append(aspects, InvokableAspect{
			Name:        aspectText,
			Source:      "character",
			SourceID:    sm.player.ID,
			FreeInvokes: 0, // Character aspects don't have free invokes
			AlreadyUsed: usedAspects[aspectText],
		})
	}

	// Player's consequences (can be invoked against self for +2)
	for _, consequence := range sm.player.Consequences {
		if consequence.Aspect == "" {
			continue
		}
		aspects = append(aspects, InvokableAspect{
			Name:        consequence.Aspect,
			Source:      "consequence",
			SourceID:    consequence.ID,
			FreeInvokes: 0,
			AlreadyUsed: usedAspects[consequence.Aspect],
		})
	}

	// Situation aspects
	for _, sitAspect := range sm.currentScene.SituationAspects {
		if sitAspect.Aspect == "" {
			continue
		}
		aspects = append(aspects, InvokableAspect{
			Name:        sitAspect.Aspect,
			Source:      "situation",
			SourceID:    sitAspect.ID,
			FreeInvokes: sitAspect.FreeInvokes,
			AlreadyUsed: usedAspects[sitAspect.Aspect],
		})
	}

	return aspects
}

// handlePostRollInvokes prompts the player to invoke aspects after seeing their roll result.
// Returns the final CheckResult after any invokes/rerolls.
// For attacks: skips if already success with style.
// For defense: skips if attack already fails (defender wins).
// Always skips if no fate points AND no free invokes available.
func (sm *SceneManager) handlePostRollInvokes(result *dice.CheckResult, difficulty dice.Ladder, parsedAction *action.Action, isDefense bool) *dice.CheckResult {
	usedAspects := make(map[string]bool)

	for {
		// Calculate outcome and shifts needed
		outcome := result.CompareAgainst(difficulty)
		shiftsNeeded := 0

		if isDefense {
			// For defense: positive shifts = attack fails (good for player)
			// Skip if attack already fails (shifts >= 0)
			if outcome.Shifts >= 0 {
				break
			}
			// Need enough to at least tie (make shifts = 0)
			shiftsNeeded = -outcome.Shifts
		} else {
			// For attacks/overcome: skip if already success with style
			if outcome.Type == dice.SuccessWithStyle {
				break
			}
			if outcome.Shifts < 0 {
				shiftsNeeded = -outcome.Shifts // Need this many to tie
			} else if outcome.Shifts < 3 {
				shiftsNeeded = 3 - outcome.Shifts // Need this many for success with style
			}
		}

		// Gather available aspects
		available := sm.gatherInvokableAspects(usedAspects)

		// Check if player has any way to invoke
		canInvoke := false
		for _, aspect := range available {
			if aspect.AlreadyUsed {
				continue
			}
			if aspect.FreeInvokes > 0 || sm.player.FatePoints > 0 {
				canInvoke = true
				break
			}
		}
		if !canInvoke {
			break
		}

		// Prompt player
		choice := sm.ui.PromptForInvoke(available, sm.player.FatePoints, result.FinalValue.String(), shiftsNeeded)

		// Player chose to skip
		if choice.Aspect == nil {
			break
		}

		// Spend fate point or use free invoke
		if choice.UseFree {
			// Find and decrement free invoke on situation aspect
			for i := range sm.currentScene.SituationAspects {
				if sm.currentScene.SituationAspects[i].Aspect == choice.Aspect.Name {
					sm.currentScene.SituationAspects[i].UseFreeInvoke()
					break
				}
			}
			sm.ui.DisplaySystemMessage(fmt.Sprintf("Using free invoke on \"%s\"!", choice.Aspect.Name))
		} else {
			if !sm.player.SpendFatePoint() {
				sm.ui.DisplaySystemMessage("Not enough Fate Points!")
				continue
			}
			sm.ui.DisplaySystemMessage(fmt.Sprintf("Invoking \"%s\"! (%d FP remaining)", choice.Aspect.Name, sm.player.FatePoints))
		}

		// Mark aspect as used for this roll
		usedAspects[choice.Aspect.Name] = true

		// Apply the invoke
		if choice.IsReroll {
			result = sm.roller.Reroll(result)
			sm.ui.DisplaySystemMessage(fmt.Sprintf("Rerolled: %s (Total: %s)", result.Roll.String(), result.FinalValue.String()))
		} else {
			result.ApplyInvokeBonus(2)
			sm.ui.DisplaySystemMessage(fmt.Sprintf("+2! New total: %s", result.FinalValue.String()))
		}

		// Track invoke on the action
		parsedAction.AddAspectInvoke(action.AspectInvoke{
			AspectText: choice.Aspect.Name,
			Source:     choice.Aspect.Source,
			SourceID:   choice.Aspect.SourceID,
			IsFree:     choice.UseFree,
			FatePointCost: func() int {
				if choice.UseFree {
					return 0
				}
				return 1
			}(),
			Bonus: func() int {
				if choice.IsReroll {
					return 0
				}
				return 2
			}(),
			IsReroll: choice.IsReroll,
		})
	}

	return result
}

// generateSceneResponse generates an LLM response for dialog/clarification
func (sm *SceneManager) generateSceneResponse(ctx context.Context, input string, interactionType string) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("generateSceneResponse: %w", ErrLLMUnavailable)
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

	var prompt string
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

		conflictData := ConflictResponseData{
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
		}

		prompt, renderErr = RenderConflictResponse(conflictData)
	} else {
		// Use standard scene response template
		data := SceneResponseData{
			Scene:               sm.currentScene,
			CharacterContext:    sm.buildCharacterContext(),
			AspectsContext:      sm.buildAspectsContext(),
			ConversationContext: sm.buildConversationContext(),
			PlayerInput:         input,
			InteractionType:     interactionType,
			OtherCharacters:     otherCharacters,
			TakenOutCharacters:  takenOutCharacters,
		}

		prompt, renderErr = RenderSceneResponse(data)
	}

	if renderErr != nil {
		return "", fmt.Errorf("generateSceneResponse: %w: %v", ErrLLMInvalidResponse, renderErr)
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   300,
		Temperature: 0.7,
	})
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

	// Prepare template data and render
	data := ActionNarrativeData{
		Scene:               sm.currentScene,
		CharacterContext:    sm.buildCharacterContext(),
		AspectsContext:      sm.buildAspectsContext(),
		ConversationContext: sm.buildConversationContext(),
		Action:              parsedAction,
		OtherCharacters:     otherCharacters,
	}

	prompt, err := RenderActionNarrative(data)
	if err != nil {
		return "", fmt.Errorf("generateActionNarrative: %w: %v", ErrLLMInvalidResponse, err)
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   200,
		Temperature: 0.8,
	})
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
func (sm *SceneManager) applyActionEffects(parsedAction *action.Action, target *character.Character) {
	if parsedAction.Outcome == nil {
		return
	}

	switch parsedAction.Type {
	case action.CreateAdvantage:
		if parsedAction.IsSuccess() {
			aspectName, freeInvokes := sm.generateAspectName(parsedAction)

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

	case action.Attack:
		if target == nil {
			slog.Debug("Attack has no valid target, skipping damage application",
				"component", componentSceneManager)
			return
		}

		if parsedAction.IsSuccess() {
			shifts := parsedAction.Outcome.Shifts
			if shifts < 1 {
				shifts = 1 // Minimum 1 shift on success
			}

			// Determine stress type based on attack skill
			stressType := core.StressTypeForAttack(parsedAction.Skill)

			sm.ui.DisplaySystemMessage(fmt.Sprintf(
				"Your attack deals %d shifts to %s!",
				shifts, target.Name))

			// Apply stress to target
			sm.applyDamageToTarget(target, shifts, stressType)
		} else if parsedAction.Outcome.Type == dice.Tie {
			// On a tie, attacker gets a boost
			sm.ui.DisplaySystemMessage("Tie! You gain a boost against your opponent.")
		}
	}
}

// generateAspectName uses the LLM to generate a creative aspect name for Create an Advantage
// Falls back to a simple description-based name if the LLM is unavailable or fails
func (sm *SceneManager) generateAspectName(parsedAction *action.Action) (string, int) {
	// Determine free invokes based on outcome
	freeInvokes := 1
	if parsedAction.IsSuccessWithStyle() {
		freeInvokes = 2
	}

	// Fallback name if LLM generation fails
	fallbackName := fmt.Sprintf("Advantage from %s", parsedAction.Description)

	// If no aspect generator available, use fallback
	if sm.aspectGenerator == nil {
		slog.Debug("No aspect generator available, using fallback name",
			"component", componentSceneManager)
		return fallbackName, freeInvokes
	}

	// Gather existing aspects for context
	existingAspects := make([]string, 0)
	for _, sa := range sm.currentScene.SituationAspects {
		existingAspects = append(existingAspects, sa.Aspect)
	}

	// Build the request
	req := AspectGenerationRequest{
		Character:       sm.player,
		Action:          parsedAction,
		Outcome:         parsedAction.Outcome,
		Context:         sm.currentScene.Description,
		TargetType:      "situation",
		ExistingAspects: existingAspects,
	}

	// Generate aspect via LLM
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := sm.aspectGenerator.GenerateAspect(ctx, req)
	if err != nil {
		slog.Warn("Failed to generate aspect via LLM, using fallback",
			"component", componentSceneManager,
			"error", err)
		return fallbackName, freeInvokes
	}

	// Use generated aspect text, or fallback if empty
	if response.AspectText == "" {
		return fallbackName, freeInvokes
	}

	// Override free invokes from LLM response if it makes sense
	if response.FreeInvokes > 0 {
		freeInvokes = response.FreeInvokes
	}

	slog.Debug("Generated aspect via LLM",
		"component", componentSceneManager,
		"aspect", response.AspectText,
		"freeInvokes", freeInvokes,
		"reasoning", response.Reasoning)

	return response.AspectText, freeInvokes
}

// applyDamageToTarget applies shifts as stress/consequences to a target
func (sm *SceneManager) applyDamageToTarget(target *character.Character, shifts int, stressType character.StressTrackType) {
	// Try to absorb with stress track
	absorbed := target.TakeStress(stressType, shifts)
	if absorbed {
		sm.ui.DisplaySystemMessage(fmt.Sprintf(
			"%s absorbs the damage with their %s stress track.",
			target.Name, stressType))
		return
	}

	// Target couldn't absorb all stress - check for consequences or taken out
	sm.handleTargetStressOverflow(target, shifts, stressType)
}

// handleTargetStressOverflow handles when a target can't absorb stress
func (sm *SceneManager) handleTargetStressOverflow(target *character.Character, shifts int, stressType character.StressTrackType) {
	// Check if target has available consequences
	availableConseq := sm.getTargetAvailableConsequences(target, shifts)

	if len(availableConseq) == 0 {
		// No way to absorb - target is taken out!
		sm.handleTargetTakenOut(target)
		return
	}

	// NPC takes the most appropriate consequence automatically
	// (In a more sophisticated system, NPC AI could choose strategically)
	bestConseq := availableConseq[0]
	for _, c := range availableConseq {
		if c.Value >= shifts && c.Value < bestConseq.Value {
			bestConseq = c
		}
	}

	// Apply consequence to target
	consequence := character.Consequence{
		ID:        fmt.Sprintf("conseq-%d", time.Now().UnixNano()),
		Type:      bestConseq.Type,
		Aspect:    fmt.Sprintf("Wounded by %s", sm.player.Name),
		Duration:  string(bestConseq.Type),
		CreatedAt: time.Now(),
	}
	target.AddConsequence(consequence)

	absorbed := bestConseq.Value
	remaining := shifts - absorbed

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s takes a %s consequence: \"%s\" (absorbs %d shifts)",
		target.Name, bestConseq.Type, consequence.Aspect, absorbed))

	// If there's remaining damage, try stress again or take out
	if remaining > 0 {
		if target.TakeStress(stressType, remaining) {
			sm.ui.DisplaySystemMessage(fmt.Sprintf(
				"%s absorbs remaining %d shifts with stress.",
				target.Name, remaining))
		} else {
			sm.handleTargetTakenOut(target)
		}
	}
}

// getTargetAvailableConsequences returns available consequence slots for a target
func (sm *SceneManager) getTargetAvailableConsequences(target *character.Character, shifts int) []ConsequenceOption {
	available := []ConsequenceOption{}

	// Use the character's CanTakeConsequence method which respects NPC type
	if target.CanTakeConsequence(character.MildConsequence) {
		available = append(available, ConsequenceOption{
			Type:  character.MildConsequence,
			Value: character.MildConsequence.Value(),
		})
	}
	if target.CanTakeConsequence(character.ModerateConsequence) {
		available = append(available, ConsequenceOption{
			Type:  character.ModerateConsequence,
			Value: character.ModerateConsequence.Value(),
		})
	}
	if target.CanTakeConsequence(character.SevereConsequence) {
		available = append(available, ConsequenceOption{
			Type:  character.SevereConsequence,
			Value: character.SevereConsequence.Value(),
		})
	}

	return available
}

// handleTargetTakenOut handles when a target is taken out of the conflict
func (sm *SceneManager) handleTargetTakenOut(target *character.Character) {
	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"\n=== %s is Taken Out! ===", target.Name))
	sm.ui.DisplaySystemMessage("You decide their fate!")

	// Log the taken out event
	if sm.sessionLogger != nil {
		sm.sessionLogger.Log("taken_out", map[string]any{
			"character_id":   target.ID,
			"character_name": target.Name,
			"by_player":      sm.player.ID,
		})
	}

	// Mark the target as taken out for the duration of this scene
	// This prevents them from rejoining conflicts until a new scene begins
	sm.currentScene.MarkCharacterTakenOut(target.ID)

	// Mark the target as taken out in the conflict
	if sm.currentScene.IsConflict && sm.currentScene.ConflictState != nil {
		sm.currentScene.SetParticipantStatus(target.ID, scene.StatusTakenOut)

		// Check if conflict should end (all opponents taken out)
		activeOpponents := 0
		for _, p := range sm.currentScene.ConflictState.Participants {
			if p.CharacterID != sm.player.ID && p.Status == scene.StatusActive {
				activeOpponents++
			}
		}

		if activeOpponents == 0 {
			sm.ui.DisplaySystemMessage("\n=== Victory! All opponents defeated! ===")
			sm.currentScene.EndConflict()
		}
	}

	slog.Info("Target taken out",
		"component", componentSceneManager,
		"target", target.ID,
		"target_name", target.Name)
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

// applyAttackDamageToPlayer applies attack damage to the player
func (sm *SceneManager) applyAttackDamageToPlayer(ctx context.Context, outcome *dice.Outcome, attacker *character.Character, attackCtx AttackContext) {
	// Apply stress if the attack hit
	switch outcome.Type {
	case dice.Success, dice.SuccessWithStyle:
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
			sm.handleStressOverflow(ctx, shifts, stressType, attacker, attackCtx)
		}
	case dice.Tie:
		sm.ui.DisplaySystemMessage("The attack is deflected, but grants a boost!")
	default:
		sm.ui.DisplaySystemMessage("You successfully defend!")
	}
}

// handleStressOverflow handles when the player cannot absorb stress with their stress track
func (sm *SceneManager) handleStressOverflow(ctx context.Context, shifts int, stressType character.StressTrackType, attacker *character.Character, attackCtx AttackContext) {
	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"You cannot absorb %d shifts with your stress track!",
		shifts,
	))

	// Determine available consequences
	availableConsequences := sm.getAvailableConsequences(shifts)

	if len(availableConsequences) == 0 {
		// No consequences available - taken out
		sm.ui.DisplaySystemMessage("You have no available consequences! You are taken out!")
		sm.handleTakenOut(ctx, attacker, attackCtx)
		return
	}

	// Display options (concession is not available here - per Fate Core rules,
	// you must concede BEFORE the roll, not after taking shifts)
	sm.ui.DisplaySystemMessage("\nYou must choose how to handle this:")

	for i, conseq := range availableConsequences {
		sm.ui.DisplaySystemMessage(fmt.Sprintf(
			"  %d. Take a %s consequence (absorbs %d shifts)",
			i+1, conseq.Type, conseq.Value,
		))
	}
	sm.ui.DisplaySystemMessage(fmt.Sprintf("  %d. Be Taken Out - Your opponent decides your fate", len(availableConsequences)+1))

	// Read player choice
	sm.ui.DisplaySystemMessage("\nEnter your choice (number):")
	input, _, err := sm.ui.ReadInput()
	if err != nil {
		slog.Error("Failed to read consequence choice", "error", err)
		sm.handleTakenOut(ctx, attacker, attackCtx)
		return
	}

	choice := strings.TrimSpace(input)

	// Check for consequence choices
	for i, conseq := range availableConsequences {
		if choice == fmt.Sprintf("%d", i+1) {
			sm.applyConsequence(ctx, conseq.Type, shifts, attacker, attackCtx)
			return
		}
	}

	// Check for taken out choice
	if choice == fmt.Sprintf("%d", len(availableConsequences)+1) {
		sm.handleTakenOut(ctx, attacker, attackCtx)
		return
	}

	// Invalid choice - default to taken out
	sm.ui.DisplaySystemMessage("Invalid choice. You are taken out!")
	sm.handleTakenOut(ctx, attacker, attackCtx)
}

// ConsequenceOption represents an available consequence the player can take
type ConsequenceOption struct {
	Type  character.ConsequenceType
	Value int
}

// getAvailableConsequences returns consequences that can absorb the given shifts
func (sm *SceneManager) getAvailableConsequences(shifts int) []ConsequenceOption {
	available := []ConsequenceOption{}

	// Use the character's CanTakeConsequence method which respects NPC type
	if sm.player.CanTakeConsequence(character.MildConsequence) {
		available = append(available, ConsequenceOption{
			Type:  character.MildConsequence,
			Value: character.MildConsequence.Value(),
		})
	}
	if sm.player.CanTakeConsequence(character.ModerateConsequence) {
		available = append(available, ConsequenceOption{
			Type:  character.ModerateConsequence,
			Value: character.ModerateConsequence.Value(),
		})
	}
	if sm.player.CanTakeConsequence(character.SevereConsequence) {
		available = append(available, ConsequenceOption{
			Type:  character.SevereConsequence,
			Value: character.SevereConsequence.Value(),
		})
	}

	return available
}

// applyConsequence applies a consequence to the player character
func (sm *SceneManager) applyConsequence(ctx context.Context, conseqType character.ConsequenceType, shifts int, attacker *character.Character, attackCtx AttackContext) {
	// Generate a consequence aspect via LLM
	aspectName, err := sm.generateConsequenceAspect(ctx, conseqType, attacker, attackCtx)
	if err != nil {
		slog.Error("Failed to generate consequence aspect", "error", err)
		caser := cases.Title(language.English)
		aspectName = fmt.Sprintf("%s Wound", caser.String(string(conseqType)))
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
			sm.handleStressOverflow(ctx, remaining, stressType, attacker, attackCtx)
		}
	}
}

// generateConsequenceAspect uses LLM to generate a consequence aspect
func (sm *SceneManager) generateConsequenceAspect(ctx context.Context, conseqType character.ConsequenceType, attacker *character.Character, attackCtx AttackContext) (string, error) {
	if sm.engine.llmClient == nil {
		return "", fmt.Errorf("LLM client required")
	}

	conflictType := "physical"
	if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	data := ConsequenceAspectData{
		CharacterName: sm.player.Name,
		AttackerName:  attacker.Name,
		Severity:      string(conseqType),
		ConflictType:  conflictType,
		AttackContext: attackCtx,
	}

	prompt, err := RenderConsequenceAspect(data)
	if err != nil {
		return "", fmt.Errorf("failed to render consequence aspect template: %w", err)
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   20,
		Temperature: 0.8,
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// isConcedeCommand checks if the input is a concession command
// Per Fate Core rules, concession must happen before a roll is made
func (sm *SceneManager) isConcedeCommand(input string) bool {
	normalized := strings.ToLower(strings.TrimSpace(input))
	concedeCommands := []string{"concede", "i concede", "concession", "i give up", "give up"}
	for _, cmd := range concedeCommands {
		if normalized == cmd {
			return true
		}
	}
	return false
}

// handleConcession handles when the player concedes the conflict
func (sm *SceneManager) handleConcession(ctx context.Context) {
	sm.ui.DisplaySystemMessage("\n=== You Concede! ===")
	sm.ui.DisplaySystemMessage("You choose to lose the conflict on your own terms.")
	sm.ui.DisplaySystemMessage("You get to narrate how you exit the scene and avoid the worst consequences.")

	// Award fate points: 1 for conceding + 1 for each consequence taken in this conflict
	// Per Fate Core: "you get a fate point for choosing to concede.
	// On top of that, if you've sustained any consequences in this conflict,
	// you get an additional fate point for each consequence."
	fatePointsGained := 1 // Base for conceding
	consequenceCount := len(sm.player.Consequences)
	fatePointsGained += consequenceCount

	for i := 0; i < fatePointsGained; i++ {
		sm.player.GainFatePoint()
	}

	if consequenceCount > 0 {
		sm.ui.DisplaySystemMessage(fmt.Sprintf(
			"You gain %d Fate Points (1 for conceding + %d for consequences)! (Now: %d)",
			fatePointsGained, consequenceCount, sm.player.FatePoints))
	} else {
		sm.ui.DisplaySystemMessage(fmt.Sprintf("You gain a Fate Point for conceding! (Now: %d)", sm.player.FatePoints))
	}

	// Mark player as conceded and end the conflict
	if sm.currentScene.ConflictState != nil {
		sm.currentScene.SetParticipantStatus(sm.player.ID, scene.StatusConceded)
		sm.currentScene.EndConflict()
		sm.ui.DisplayConflictEnd("You have conceded the conflict.")
	}

	// Read and display the concession narrative (no roll required - this is pure narration)
	sm.ui.DisplaySystemMessage("\nDescribe how you concede and exit the conflict:")
	input, _, err := sm.ui.ReadInput()
	if err != nil {
		slog.Error("Failed to read concession description", "error", err)
		return
	}

	if input != "" {
		// Display the player's narration and acknowledge it without any mechanical resolution
		sm.ui.DisplayNarrative(fmt.Sprintf("%s %s", sm.player.Name, input))
		sm.addToConversationHistory(input, "You exit the conflict on your own terms.", inputTypeDialog)
	}
}

// TakenOutResult represents the outcome classification of being taken out
type TakenOutResult int

const (
	TakenOutContinue   TakenOutResult = iota // Continue in same scene (knocked down, stunned, etc.)
	TakenOutTransition                       // Transition to new scene (captured, driven out, etc.)
	TakenOutGameOver                         // Game ending (death, permanent incapacitation)
)

// handleTakenOut handles when the player is taken out
func (sm *SceneManager) handleTakenOut(ctx context.Context, attacker *character.Character, attackCtx AttackContext) {
	sm.ui.DisplaySystemMessage("\n=== You Are Taken Out! ===")
	sm.ui.DisplaySystemMessage(fmt.Sprintf("%s decides your fate.", attacker.Name))

	// Generate narrative and outcome classification for being taken out
	narrative, outcome, newSceneHint, err := sm.generateTakenOutNarrativeAndOutcome(ctx, attacker, attackCtx)
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

// generateTakenOutNarrativeAndOutcome generates narrative and classifies the outcome
func (sm *SceneManager) generateTakenOutNarrativeAndOutcome(ctx context.Context, attacker *character.Character, attackCtx AttackContext) (narrative string, outcome TakenOutResult, newSceneHint string, err error) {
	if sm.engine.llmClient == nil {
		return "", TakenOutTransition, "", fmt.Errorf("LLM client required")
	}

	conflictType := "physical"
	if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	data := TakenOutData{
		CharacterName:       sm.player.Name,
		AttackerName:        attacker.Name,
		AttackerHighConcept: attacker.Aspects.HighConcept,
		ConflictType:        conflictType,
		SceneDescription:    sm.currentScene.Description,
		AttackContext:       attackCtx,
	}

	prompt, err := RenderTakenOut(data)
	if err != nil {
		return "", TakenOutTransition, "", fmt.Errorf("failed to render taken out template: %w", err)
	}

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, llm.CompletionRequest{
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		MaxTokens:   200,
		Temperature: 0.7,
	})
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
