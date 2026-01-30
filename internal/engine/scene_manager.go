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

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

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

// conflictMarkerRegex matches [CONFLICT:type:character_id] markers for escalation
var conflictMarkerRegex = regexp.MustCompile(`\[CONFLICT:(physical|mental):([^\]]+)\]`)

// conflictEndMarkerRegex matches [CONFLICT:end:reason] markers for de-escalation
var conflictEndMarkerRegex = regexp.MustCompile(`\[CONFLICT:end:(surrender|agreement|retreat|resolved)\]`)

// ConflictTrigger represents a detected conflict initiation
type ConflictTrigger struct {
	Type        scene.ConflictType
	InitiatorID string
}

// ConflictResolution represents a detected conflict de-escalation
type ConflictResolution struct {
	Reason string
}

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

// ConflictResponseData holds the data for conflict response template
type ConflictResponseData struct {
	Scene                *scene.Scene
	CharacterContext     string
	AspectsContext       string
	ConversationContext  string
	PlayerInput          string
	OtherCharacters      []*character.Character
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

// NPCAttackData holds the data for NPC attack narrative template
type NPCAttackData struct {
	ConflictType       string
	Round              int
	SceneName          string
	NPCName            string
	NPCHighConcept     string
	NPCAspects         []string
	Skill              string
	TargetName         string
	TargetHighConcept  string
	SituationAspects   []scene.SituationAspect
	OutcomeDescription string
}

// NPCActionDecisionData holds the data for NPC action decision template
type NPCActionDecisionData struct {
	ConflictType      string
	Round             int
	SceneName         string
	SceneDescription  string
	NPCName           string
	NPCHighConcept    string
	NPCTrouble        string
	NPCAspects        []string
	NPCSkills         map[string]int
	NPCPhysicalStress []bool
	NPCMentalStress   []bool
	Targets           []NPCTargetInfo
	SituationAspects  []scene.SituationAspect
}

// NPCTargetInfo holds information about a potential target for NPC actions
type NPCTargetInfo struct {
	ID             string
	Name           string
	HighConcept    string
	PhysicalStress []bool
	MentalStress   []bool
}

// NPCActionDecision represents the LLM's decision for an NPC action
type NPCActionDecision struct {
	ActionType  string `json:"action_type"`
	Skill       string `json:"skill"`
	TargetID    string `json:"target_id,omitempty"`
	Description string `json:"description"`
}

// ConsequenceAspectData holds the data for consequence aspect generation template
type ConsequenceAspectData struct {
	CharacterName string
	AttackerName  string
	Severity      string
	ConflictType  string
}

// TakenOutData holds the data for taken out narrative template
type TakenOutData struct {
	CharacterName       string
	AttackerName        string
	AttackerHighConcept string
	ConflictType        string
	SceneDescription    string
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

	// Check for conflict markers in the response (both escalation and de-escalation)
	conflictTrigger, cleanedResponse := sm.parseConflictMarker(response)
	conflictResolution, cleanedResponse := sm.parseConflictEndMarker(cleanedResponse)

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
	defenseSkill := sm.getDefenseSkillForAttack(attackSkill)
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

// getDefenseSkillForAttack returns the appropriate defense skill for an attack skill
func (sm *SceneManager) getDefenseSkillForAttack(attackSkill string) string {
	// Physical attack skills -> Athletics defense
	physicalAttacks := map[string]bool{
		"Fight": true, "Shoot": true, "Physique": true,
	}
	if physicalAttacks[attackSkill] {
		return "Athletics"
	}

	// Mental/social attack skills -> Will defense
	mentalAttacks := map[string]bool{
		"Provoke": true, "Deceive": true, "Rapport": true,
	}
	if mentalAttacks[attackSkill] {
		return "Will"
	}

	// Magic/lore attacks could be either - default to Will for supernatural
	if attackSkill == "Lore" {
		return "Will"
	}

	// Default to Athletics for unknown attack types
	return "Athletics"
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

	var buf bytes.Buffer

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
			CurrentCharacterName: currentCharName,
			ParticipantMap:       participantMap,
			CharacterMap:         characterMap,
		}

		if err := ConflictResponsePrompt.Execute(&buf, conflictData); err != nil {
			return "", fmt.Errorf("generateSceneResponse: %w: %v", ErrLLMInvalidResponse, err)
		}
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
		}

		if err := SceneResponsePrompt.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("generateSceneResponse: %w: %v", ErrLLMInvalidResponse, err)
		}
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
func (sm *SceneManager) applyActionEffects(parsedAction *action.Action, target *character.Character) {
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
			stressType := sm.getStressTypeForAttack(parsedAction.Skill)

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

// getStressTypeForAttack determines the stress type based on attack skill
func (sm *SceneManager) getStressTypeForAttack(attackSkill string) character.StressTrackType {
	mentalAttacks := map[string]bool{
		"Provoke": true, "Deceive": true, "Rapport": true, "Lore": true,
	}
	if mentalAttacks[attackSkill] {
		return character.MentalStress
	}
	return character.PhysicalStress
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

	hasMild := false
	hasModerate := false
	hasSevere := false

	for _, c := range target.Consequences {
		switch c.Type {
		case character.MildConsequence:
			hasMild = true
		case character.ModerateConsequence:
			hasModerate = true
		case character.SevereConsequence:
			hasSevere = true
		}
	}

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

// handleTargetTakenOut handles when a target is taken out of the conflict
func (sm *SceneManager) handleTargetTakenOut(target *character.Character) {
	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"\n=== %s is Taken Out! ===", target.Name))
	sm.ui.DisplaySystemMessage("You decide their fate!")

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

// parseConflictEndMarker extracts a conflict resolution from LLM response and returns cleaned text
func (sm *SceneManager) parseConflictEndMarker(response string) (*ConflictResolution, string) {
	matches := conflictEndMarkerRegex.FindStringSubmatch(response)
	if matches == nil {
		return nil, response
	}

	resolution := &ConflictResolution{
		Reason: matches[1],
	}

	// Remove the marker from the response and clean up
	cleanedResponse := conflictEndMarkerRegex.ReplaceAllString(response, "")
	cleanedResponse = strings.Join(strings.Fields(cleanedResponse), " ")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	return resolution, cleanedResponse
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

// resolveConflictPeacefully ends a conflict through non-violent means
func (sm *SceneManager) resolveConflictPeacefully(reason string) {
	if !sm.currentScene.IsConflict {
		return
	}

	// Format reason for display
	reasonMessage := ""
	switch reason {
	case "surrender":
		reasonMessage = "Your opponent surrenders!"
	case "agreement":
		reasonMessage = "You've reached an agreement!"
	case "retreat":
		reasonMessage = "Your opponent retreats!"
	case "resolved":
		reasonMessage = "The conflict has been resolved!"
	default:
		reasonMessage = "The conflict ends!"
	}

	sm.ui.DisplaySystemMessage("\n=== Conflict Resolved ===")
	sm.ui.DisplaySystemMessage(reasonMessage)

	sm.currentScene.EndConflict()

	slog.Info("Conflict resolved peacefully",
		"component", componentSceneManager,
		"reason", reason)
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

// getNPCActionDecision uses the LLM to decide what action an NPC should take
func (sm *SceneManager) getNPCActionDecision(ctx context.Context, npc *character.Character) (*NPCActionDecision, error) {
	if sm.engine.llmClient == nil {
		return nil, fmt.Errorf("LLM client required for NPC decisions")
	}

	// Determine conflict type string
	conflictType := "physical"
	if sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	// Build target list (all active participants except this NPC)
	var targets []NPCTargetInfo
	for _, p := range sm.currentScene.ConflictState.Participants {
		if p.CharacterID == npc.ID || p.Status != scene.StatusActive {
			continue
		}
		char := sm.engine.GetCharacter(p.CharacterID)
		if char == nil {
			continue
		}
		target := NPCTargetInfo{
			ID:          char.ID,
			Name:        char.Name,
			HighConcept: char.Aspects.HighConcept,
		}
		if track, ok := char.StressTracks["physical"]; ok {
			target.PhysicalStress = track.Boxes
		}
		if track, ok := char.StressTracks["mental"]; ok {
			target.MentalStress = track.Boxes
		}
		targets = append(targets, target)
	}

	// Build skill map with integer values
	npcSkills := make(map[string]int)
	for skill, level := range npc.Skills {
		npcSkills[skill] = int(level)
	}

	// Get NPC stress
	var physicalStress, mentalStress []bool
	if track, ok := npc.StressTracks["physical"]; ok {
		physicalStress = track.Boxes
	}
	if track, ok := npc.StressTracks["mental"]; ok {
		mentalStress = track.Boxes
	}

	data := NPCActionDecisionData{
		ConflictType:      conflictType,
		Round:             sm.currentScene.ConflictState.Round,
		SceneName:         sm.currentScene.Name,
		SceneDescription:  sm.currentScene.Description,
		NPCName:           npc.Name,
		NPCHighConcept:    npc.Aspects.HighConcept,
		NPCTrouble:        npc.Aspects.Trouble,
		NPCAspects:        npc.Aspects.GetAll(),
		NPCSkills:         npcSkills,
		NPCPhysicalStress: physicalStress,
		NPCMentalStress:   mentalStress,
		Targets:           targets,
		SituationAspects:  sm.currentScene.SituationAspects,
	}

	var buf bytes.Buffer
	if err := NPCActionDecisionPrompt.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute NPC action decision template: %w", err)
	}

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: buf.String()},
		},
		MaxTokens:   150,
		Temperature: 0.7,
	}

	slog.Debug("NPC action decision LLM request",
		"component", componentSceneManager,
		"npc", npc.Name)

	resp, err := sm.engine.llmClient.ChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Parse JSON response
	content := cleanJSONResponse(resp.Choices[0].Message.Content)
	var decision NPCActionDecision
	if err := json.Unmarshal([]byte(content), &decision); err != nil {
		slog.Warn("Failed to parse NPC action decision",
			"component", componentSceneManager,
			"npc", npc.Name,
			"response", content,
			"error", err)
		return nil, fmt.Errorf("failed to parse decision: %w", err)
	}

	slog.Info("NPC action decision",
		"component", componentSceneManager,
		"npc", npc.Name,
		"action_type", decision.ActionType,
		"skill", decision.Skill,
		"target", decision.TargetID,
		"description", decision.Description)

	return &decision, nil
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

	// Get LLM decision for NPC action
	decision, err := sm.getNPCActionDecision(ctx, npc)
	if err != nil {
		slog.Warn("Failed to get NPC action decision, defaulting to attack",
			"component", componentSceneManager,
			"npc", npc.Name,
			"error", err)
		// Fallback to simple attack
		decision = &NPCActionDecision{
			ActionType:  "ATTACK",
			Skill:       sm.getDefaultAttackSkill(),
			TargetID:    sm.player.ID,
			Description: fmt.Sprintf("%s attacks!", npc.Name),
		}
	}

	// Process the decision based on action type
	switch strings.ToUpper(decision.ActionType) {
	case "DEFEND":
		sm.processNPCDefend(ctx, npc, decision)
	case "CREATE_ADVANTAGE":
		sm.processNPCCreateAdvantage(ctx, npc, decision)
	case "OVERCOME":
		sm.processNPCOvercome(ctx, npc, decision)
	default: // ATTACK or unknown
		sm.processNPCAttack(ctx, npc, decision)
	}
}

// getDefaultAttackSkill returns the default attack skill based on conflict type
func (sm *SceneManager) getDefaultAttackSkill() string {
	if sm.currentScene.ConflictState.Type == scene.MentalConflict {
		return "Provoke"
	}
	return "Fight"
}

// processNPCDefend handles an NPC choosing full defense
func (sm *SceneManager) processNPCDefend(ctx context.Context, npc *character.Character, decision *NPCActionDecision) {
	// Set full defense flag
	sm.currentScene.SetFullDefense(npc.ID)

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s takes a defensive stance! (+2 to all defense rolls this exchange)",
		npc.Name,
	))

	// Generate narrative
	narrative := fmt.Sprintf("%s braces for incoming attacks, focusing entirely on defense.", npc.Name)
	if decision.Description != "" {
		narrative = decision.Description
	}
	sm.ui.DisplayNarrative(narrative)
}

// processNPCCreateAdvantage handles an NPC creating an advantage
func (sm *SceneManager) processNPCCreateAdvantage(ctx context.Context, npc *character.Character, decision *NPCActionDecision) {
	skill := decision.Skill
	if skill == "" {
		skill = "Notice" // Default
	}

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(skill)

	// Roll against Fair (+2) difficulty for creating aspects
	npcRoll := sm.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))
	difficulty := dice.Fair
	outcome := npcRoll.CompareAgainst(difficulty)

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s attempts to Create an Advantage with %s (%s vs Fair)",
		npc.Name,
		skill,
		npcRoll.FinalValue.String(),
	))

	switch outcome.Type {
	case dice.Success, dice.SuccessWithStyle:
		// Create situation aspect
		aspectName := fmt.Sprintf("%s's Advantage", npc.Name)
		if decision.Description != "" {
			// Try to extract a short aspect name from the description
			aspectName = decision.Description
			if len(aspectName) > 40 {
				aspectName = aspectName[:40]
			}
		}

		freeInvokes := 1
		if outcome.Type == dice.SuccessWithStyle {
			freeInvokes = 2
		}

		aspectID := fmt.Sprintf("npc-advantage-%d", time.Now().UnixNano())
		situationAspect := scene.NewSituationAspect(aspectID, aspectName, npc.ID, freeInvokes)
		sm.currentScene.AddSituationAspect(situationAspect)
		sm.ui.DisplaySystemMessage(fmt.Sprintf(
			"Created aspect: \"%s\" with %d free invoke(s)!",
			aspectName,
			freeInvokes,
		))
		sm.ui.DisplayNarrative(fmt.Sprintf("%s gains a tactical advantage!", npc.Name))
	case dice.Tie:
		sm.ui.DisplaySystemMessage("The attempt succeeds but grants a boost to opponents!")
		sm.ui.DisplayNarrative(fmt.Sprintf("%s's maneuver is partially successful.", npc.Name))
	default:
		sm.ui.DisplaySystemMessage("The attempt fails!")
		sm.ui.DisplayNarrative(fmt.Sprintf("%s's gambit doesn't pay off.", npc.Name))
	}
}

// processNPCOvercome handles an NPC attempting to overcome an obstacle
func (sm *SceneManager) processNPCOvercome(ctx context.Context, npc *character.Character, decision *NPCActionDecision) {
	skill := decision.Skill
	if skill == "" {
		skill = "Athletics" // Default
	}

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(skill)

	// Roll against Fair (+2) difficulty
	npcRoll := sm.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))
	difficulty := dice.Fair
	outcome := npcRoll.CompareAgainst(difficulty)

	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s attempts to Overcome with %s (%s vs Fair)",
		npc.Name,
		skill,
		npcRoll.FinalValue.String(),
	))

	switch outcome.Type {
	case dice.Success, dice.SuccessWithStyle:
		sm.ui.DisplaySystemMessage("The obstacle is overcome!")
		narrative := decision.Description
		if narrative == "" {
			narrative = fmt.Sprintf("%s successfully overcomes the challenge.", npc.Name)
		}
		sm.ui.DisplayNarrative(narrative)
	case dice.Tie:
		sm.ui.DisplaySystemMessage("Success, but at a minor cost.")
		sm.ui.DisplayNarrative(fmt.Sprintf("%s manages to push through, but not without difficulty.", npc.Name))
	default:
		sm.ui.DisplaySystemMessage("The attempt fails!")
		sm.ui.DisplayNarrative(fmt.Sprintf("%s is unable to overcome the obstacle.", npc.Name))
	}
}

// processNPCAttack handles an NPC attacking a target
func (sm *SceneManager) processNPCAttack(ctx context.Context, npc *character.Character, decision *NPCActionDecision) {
	// Determine target
	target := sm.player // Default to player
	targetID := decision.TargetID
	if targetID != "" && targetID != sm.player.ID {
		if t := sm.engine.GetCharacter(targetID); t != nil {
			target = t
		}
	}

	// Use the skill from the decision, or default based on conflict type
	attackSkill := decision.Skill
	if attackSkill == "" {
		attackSkill = sm.getDefaultAttackSkill()
	}

	// Determine defense skill based on attack skill
	defenseSkill := sm.getDefenseSkillForAttack(attackSkill)

	// Get NPC's skill level
	npcSkillLevel := npc.GetSkill(attackSkill)

	// Roll NPC's attack
	npcRoll := sm.roller.RollWithModifier(dice.Mediocre, int(npcSkillLevel))

	// Get target's defense (check for full defense bonus)
	targetDefenseLevel := target.GetSkill(defenseSkill)
	defenseBonus := 0
	if participant := sm.currentScene.GetParticipant(target.ID); participant != nil && participant.FullDefense {
		defenseBonus = 2
	}
	targetDefense := sm.roller.RollWithModifier(dice.Mediocre, int(targetDefenseLevel)+defenseBonus)

	// Display the mechanical result
	defenseDisplay := defenseSkill
	if defenseBonus > 0 {
		defenseDisplay = fmt.Sprintf("%s+2 (Full Defense)", defenseSkill)
	}
	sm.ui.DisplaySystemMessage(fmt.Sprintf(
		"%s attacks %s with %s (%s) vs %s (%s)",
		npc.Name,
		target.Name,
		attackSkill,
		npcRoll.FinalValue.String(),
		defenseDisplay,
		targetDefense.FinalValue.String(),
	))

	// Initial outcome (before player invokes)
	initialOutcome := npcRoll.CompareAgainst(targetDefense.FinalValue)
	sm.ui.DisplaySystemMessage(fmt.Sprintf("Initial outcome: %s", initialOutcome.Type.String()))

	// If target is the player, allow them to invoke aspects to improve defense
	if target.ID == sm.player.ID {
		// Create a temporary action to track invokes for defense
		defenseAction := action.NewAction("defense-invoke", sm.player.ID, action.Defend, defenseSkill, "Defending against attack")

		// Player can invoke to improve their defense
		// isDefense=true means skip prompt if attack already fails
		targetDefense = sm.handlePostRollInvokes(targetDefense, npcRoll.FinalValue, defenseAction, true)

		// Recalculate outcome with potentially improved defense
		// Note: For defense, we compare attacker vs defender, so we still use npcRoll.CompareAgainst
		// but the targetDefense.FinalValue may have increased
	}

	// Compare results (final)
	outcome := npcRoll.CompareAgainst(targetDefense.FinalValue)

	// Display updated outcome if it changed
	if outcome.Type != initialOutcome.Type {
		sm.ui.DisplaySystemMessage(fmt.Sprintf("Final outcome: %s", outcome.Type.String()))
	}

	// Generate narrative for the attack
	npcNarrative, err := sm.generateNPCAttackNarrative(ctx, npc, attackSkill, outcome)
	if err != nil {
		slog.Error("Failed to generate NPC attack narrative",
			"component", componentSceneManager,
			"npc_id", npc.ID,
			"error", err)
		// Fallback narrative
		if outcome.Type == dice.Success || outcome.Type == dice.SuccessWithStyle {
			npcNarrative = fmt.Sprintf("%s's attack hits!", npc.Name)
		} else {
			npcNarrative = fmt.Sprintf("%s's attack misses.", npc.Name)
		}
	}
	sm.ui.DisplayNarrative(npcNarrative)

	// Only apply damage if target is the player (for now, NPC vs NPC damage not fully implemented)
	if target.ID == sm.player.ID {
		sm.applyAttackDamageToPlayer(ctx, outcome, npc)
	} else {
		// For NPC targets, just show the result
		if outcome.Type == dice.Success || outcome.Type == dice.SuccessWithStyle {
			sm.ui.DisplaySystemMessage(fmt.Sprintf("%s takes %d shifts of stress!", target.Name, outcome.Shifts))
		}
	}
}

// applyAttackDamageToPlayer applies attack damage to the player
func (sm *SceneManager) applyAttackDamageToPlayer(ctx context.Context, outcome *dice.Outcome, attacker *character.Character) {
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
			sm.handleStressOverflow(ctx, shifts, stressType, attacker)
		}
	case dice.Tie:
		sm.ui.DisplaySystemMessage("The attack is deflected, but grants a boost!")
	default:
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
		// No consequences available - taken out
		sm.ui.DisplaySystemMessage("You have no available consequences! You are taken out!")
		sm.handleTakenOut(ctx, attacker)
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
		sm.handleTakenOut(ctx, attacker)
		return
	}

	choice := strings.TrimSpace(input)

	// Check for consequence choices
	for i, conseq := range availableConsequences {
		if choice == fmt.Sprintf("%d", i+1) {
			sm.applyConsequence(ctx, conseq.Type, shifts, attacker)
			return
		}
	}

	// Check for taken out choice
	if choice == fmt.Sprintf("%d", len(availableConsequences)+1) {
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

	data := ConsequenceAspectData{
		CharacterName: sm.player.Name,
		AttackerName:  attacker.Name,
		Severity:      string(conseqType),
		ConflictType:  conflictType,
	}

	var buf bytes.Buffer
	if err := ConsequenceAspectPrompt.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute consequence aspect template: %w", err)
	}

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: buf.String()},
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

// generateTakenOutNarrativeAndOutcome generates narrative and classifies the outcome
func (sm *SceneManager) generateTakenOutNarrativeAndOutcome(ctx context.Context, attacker *character.Character) (narrative string, outcome TakenOutResult, newSceneHint string, err error) {
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
	}

	var buf bytes.Buffer
	if err := TakenOutPrompt.Execute(&buf, data); err != nil {
		return "", TakenOutTransition, "", fmt.Errorf("failed to execute taken out template: %w", err)
	}

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: buf.String()},
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

	// Build outcome description based on result
	outcomeDesc := "misses completely"
	switch outcome.Type {
	case dice.SuccessWithStyle:
		outcomeDesc = "lands a devastating blow"
	case dice.Success:
		outcomeDesc = "connects solidly"
	case dice.Tie:
		outcomeDesc = "is barely deflected, but creates an opening"
	case dice.Failure:
		outcomeDesc = "misses completely"
	}

	// Determine conflict type string
	conflictType := "physical"
	if sm.currentScene.ConflictState != nil && sm.currentScene.ConflictState.Type == scene.MentalConflict {
		conflictType = "mental"
	}

	// Get round number
	round := 1
	if sm.currentScene.ConflictState != nil {
		round = sm.currentScene.ConflictState.Round
	}

	// Build template data with full context
	data := NPCAttackData{
		ConflictType:       conflictType,
		Round:              round,
		SceneName:          sm.currentScene.Name,
		NPCName:            npc.Name,
		NPCHighConcept:     npc.Aspects.HighConcept,
		NPCAspects:         npc.Aspects.GetAll(),
		Skill:              skill,
		TargetName:         sm.player.Name,
		TargetHighConcept:  sm.player.Aspects.HighConcept,
		SituationAspects:   sm.currentScene.SituationAspects,
		OutcomeDescription: outcomeDesc,
	}

	// Execute template
	var buf bytes.Buffer
	if err := NPCAttackPrompt.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute NPC attack template: %w", err)
	}

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: buf.String()},
		},
		MaxTokens:   100,
		Temperature: 0.8,
	}

	slog.Debug("NPC attack narrative LLM request",
		"component", componentSceneManager,
		"npc", npc.Name,
		"skill", skill,
		"outcome", outcome.Type.String())

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
