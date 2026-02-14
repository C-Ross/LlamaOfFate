package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// compile-time check
var _ uicontract.UI = (*TerminalUI)(nil)

// TerminalUI implements the UI interface for terminal-based interaction
type TerminalUI struct {
	reader    *bufio.Reader
	sceneInfo uicontract.SceneInfo
	shownHint bool // true after the initial help hint has been displayed
}

// NewTerminalUI creates a new terminal UI instance
func NewTerminalUI() *TerminalUI {
	return &TerminalUI{
		reader: bufio.NewReader(os.Stdin),
	}
}

// SetSceneInfo sets the scene information for display methods such as DisplayCharacter.
func (ui *TerminalUI) SetSceneInfo(sceneInfo uicontract.SceneInfo) {
	ui.sceneInfo = sceneInfo
}

// Emit renders a structured GameEvent to the terminal.
func (ui *TerminalUI) Emit(event uicontract.GameEvent) {
	switch e := event.(type) {
	case uicontract.NarrativeEvent:
		ui.displayNarrative(e.Text)
	case uicontract.DialogEvent:
		ui.displayDialog(e.PlayerInput, e.GMResponse)
	case uicontract.SystemMessageEvent:
		ui.displaySystemMessage(e.Message)
	case uicontract.ActionAttemptEvent:
		ui.displayActionAttempt(e.Description)
	case uicontract.ActionResultEvent:
		ui.displayActionResult(e.Skill, e.SkillLevel, e.Bonuses, e.Result, e.Outcome)
	case uicontract.SceneTransitionEvent:
		ui.displaySceneTransition(e.Narrative, e.NewSceneHint)
	case uicontract.GameOverEvent:
		ui.displayGameOver(e.Reason)
	case uicontract.ConflictStartEvent:
		ui.displayConflictStart(e.ConflictType, e.InitiatorName, e.Participants)
	case uicontract.ConflictEscalationEvent:
		ui.displayConflictEscalation(e.FromType, e.ToType, e.TriggerCharName)
	case uicontract.TurnAnnouncementEvent:
		ui.displayTurnAnnouncement(e.CharacterName, e.TurnNumber, e.IsPlayer)
	case uicontract.ConflictEndEvent:
		ui.displayConflictEnd(e.Reason)
	case uicontract.CharacterDisplayEvent:
		ui.displayCharacter()

	// Composite mechanical events
	case uicontract.DefenseRollEvent:
		ui.displaySystemMessage(fmt.Sprintf("%s defends with %s (%s)", e.DefenderName, e.Skill, e.Result))
	case uicontract.DamageResolutionEvent:
		ui.displayDamageResolution(e)
	case uicontract.PlayerAttackResultEvent:
		ui.displayPlayerAttackResult(e)
	case uicontract.AspectCreatedEvent:
		ui.displaySystemMessage(fmt.Sprintf("Created situation aspect: '%s' with %d free invoke(s)", e.AspectName, e.FreeInvokes))
	case uicontract.NPCAttackEvent:
		ui.displayNPCAttack(e)
	case uicontract.PlayerStressEvent:
		ui.displaySystemMessage(fmt.Sprintf("You take %d %s stress! (%s)", e.Shifts, e.StressType, e.TrackState))
	case uicontract.PlayerDefendedEvent:
		if e.IsTie {
			ui.displaySystemMessage("The attack is deflected, but grants a boost!")
		} else {
			ui.displaySystemMessage("You successfully defend!")
		}
	case uicontract.PlayerConsequenceEvent:
		ui.displayPlayerConsequence(e)
	case uicontract.PlayerTakenOutEvent:
		ui.displayPlayerTakenOut(e)
	case uicontract.ConcessionEvent:
		ui.displayConcession(e)
	case uicontract.OutcomeChangedEvent:
		ui.displaySystemMessage(fmt.Sprintf("Final outcome: %s", e.FinalOutcome))
	}
}

// ReadInput reads input from the terminal and returns cleaned input, exit status, and error.
// Meta-commands (help, scene, etc.) are handled here so the engine never sees them.
func (ui *TerminalUI) ReadInput() (input string, isExit bool, err error) {
	if !ui.shownHint {
		fmt.Println("Type 'help' for commands, 'exit' to end.")
		ui.shownHint = true
	}
	fmt.Print("\n> ")

	rawInput, err := ui.reader.ReadString('\n')
	if err != nil {
		return "", false, fmt.Errorf("failed to read input: %w", err)
	}

	// Clean the input
	input = strings.TrimSpace(rawInput)
	if input == "" {
		return "", false, nil
	}

	// Check for exit commands first
	if ui.isExitCommand(input) {
		return input, true, nil
	}

	// Handle meta-commands locally; return empty string so the engine skips them
	if ui.sceneInfo != nil && ui.handleSpecialCommands(input) {
		return "", false, nil
	}

	return input, false, nil
}

// displayActionAttempt displays that the player is attempting an action
func (ui *TerminalUI) displayActionAttempt(description string) {
	fmt.Printf("\nYou attempt to: %s\n", description)
}

// displayActionResult displays the mechanical result of an action (dice roll, bonuses, outcome)
func (ui *TerminalUI) displayActionResult(skill string, skillLevel string, bonuses int, result string, outcome string) {
	fmt.Printf("Skill (%s): %s\n", skill, skillLevel)
	if bonuses != 0 {
		fmt.Printf("Bonuses: %+d\n", bonuses)
	}
	fmt.Printf("Rolled: %s\n", result)
	fmt.Printf("Outcome: %s\n", outcome)
}

// displayNarrative displays narrative text from the GM
func (ui *TerminalUI) displayNarrative(narrative string) {
	fmt.Printf("\n%s\n", narrative)
}

// displayDialog displays a dialog exchange between player and GM
func (ui *TerminalUI) displayDialog(playerInput, gmResponse string) {
	fmt.Printf("\nYou: %s\n", playerInput)
	fmt.Printf("\nGM: %s\n", gmResponse)
}

// displaySystemMessage displays system messages to the player
func (ui *TerminalUI) displaySystemMessage(message string) {
	fmt.Printf("\n%s\n", message)
}

// PromptForInvoke prompts the player to invoke an aspect after a roll
func (ui *TerminalUI) PromptForInvoke(available []uicontract.InvokableAspect, fatePoints int, currentResult string, shiftsNeeded int) *uicontract.InvokeChoice {
	// Filter to only show usable aspects (has free invokes OR player has FP)
	var usable []uicontract.InvokableAspect
	for _, aspect := range available {
		if aspect.AlreadyUsed {
			continue
		}
		if aspect.FreeInvokes > 0 || fatePoints > 0 {
			usable = append(usable, aspect)
		}
	}

	if len(usable) == 0 {
		return &uicontract.InvokeChoice{Aspect: nil}
	}

	// Build prompt showing available aspects with numbers
	fmt.Printf("\nInvoke? [%d FP]", fatePoints)
	if shiftsNeeded > 0 {
		fmt.Printf(" (need +%d)", shiftsNeeded)
	}
	fmt.Println()

	for i, aspect := range usable {
		marker := ""
		if aspect.FreeInvokes > 0 {
			marker = "★"
		}
		fmt.Printf("  [%d] %s%s\n", i+1, aspect.Name, marker)
	}
	fmt.Print("  [Enter] skip: ")

	// Read choice
	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return &uicontract.InvokeChoice{Aspect: nil}
	}
	input = strings.TrimSpace(input)

	if input == "" {
		return &uicontract.InvokeChoice{Aspect: nil}
	}

	// Parse number
	var choice int
	_, err = fmt.Sscanf(input, "%d", &choice)
	if err != nil || choice < 1 || choice > len(usable) {
		fmt.Println("Invalid choice.")
		return &uicontract.InvokeChoice{Aspect: nil}
	}

	selectedAspect := &usable[choice-1]

	// Determine if using free invoke or fate point
	useFree := selectedAspect.FreeInvokes > 0

	// Ask +2 or reroll
	fmt.Print("+2 or Reroll? (b/r): ")
	input, err = ui.reader.ReadString('\n')
	if err != nil {
		return &uicontract.InvokeChoice{Aspect: nil}
	}
	input = strings.TrimSpace(strings.ToLower(input))

	isReroll := input == "r" || input == "reroll"

	return &uicontract.InvokeChoice{
		Aspect:   selectedAspect,
		UseFree:  useFree,
		IsReroll: isReroll,
	}
}

// displayCharacter displays the player character sheet.
// The engine calls this via Emit(CharacterDisplayEvent{}); the data comes from the
// SceneInfo provider injected by SetSceneInfo.
func (ui *TerminalUI) displayCharacter() {
	if ui.sceneInfo == nil {
		fmt.Println("No scene information available.")
		return
	}

	player := ui.sceneInfo.GetPlayer()
	if player == nil {
		fmt.Println("No active character.")
		return
	}

	fmt.Printf("\n=== %s ===\n", player.Name)
	fmt.Printf("High Concept: %s\n", player.Aspects.HighConcept)
	fmt.Printf("Trouble: %s\n", player.Aspects.Trouble)

	if len(player.Aspects.OtherAspects) > 0 {
		fmt.Println("Other Aspects:")
		for _, aspect := range player.Aspects.OtherAspects {
			if aspect != "" {
				fmt.Printf("  - %s\n", aspect)
			}
		}
	}

	fmt.Printf("Fate Points: %d\n", player.FatePoints)
}

// handleSpecialCommands processes meta-commands and returns true if handled.
func (ui *TerminalUI) handleSpecialCommands(input string) bool {
	parts := strings.Fields(strings.ToLower(input))
	if len(parts) == 0 {
		return false
	}

	switch parts[0] {
	case "help", "?":
		ui.showHelp()
	case "scene":
		ui.displayScene()
	case "character", "char", "me":
		ui.displayCharacter()
	case "status":
		ui.displayStatus()
	case "aspects":
		ui.displayAspects()
	case "history", "conversation":
		ui.displayConversationHistory()
	default:
		return false
	}
	return true
}

func (ui *TerminalUI) showHelp() {
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

func (ui *TerminalUI) displayScene() {
	scene := ui.sceneInfo.GetCurrentScene()
	if scene == nil {
		fmt.Println("No active scene.")
		return
	}

	fmt.Printf("\n=== %s ===\n", scene.Name)
	fmt.Printf("%s\n", scene.Description)

	if len(scene.SituationAspects) > 0 {
		fmt.Println("\nSituation Aspects:")
		for _, aspect := range scene.SituationAspects {
			invokes := ""
			if aspect.FreeInvokes > 0 {
				invokes = fmt.Sprintf(" (%d free invoke(s))", aspect.FreeInvokes)
			}
			fmt.Printf("  - %s%s\n", aspect.Aspect, invokes)
		}
	}
}

func (ui *TerminalUI) displayStatus() {
	player := ui.sceneInfo.GetPlayer()
	if player == nil {
		fmt.Println("No active character.")
		return
	}

	fmt.Println("\n=== Status ===")

	// Stress tracks
	for _, track := range player.StressTracks {
		fmt.Println(track.String())
	}

	// Consequences
	if len(player.Consequences) > 0 {
		fmt.Println("\nConsequences:")
		for _, c := range player.Consequences {
			fmt.Printf("  %s: %s\n", c.Type, c.Aspect)
		}
	}

	fmt.Printf("Fate Points: %d\n", player.FatePoints)
}

func (ui *TerminalUI) displayAspects() {
	fmt.Println("\n=== Available Aspects ===")

	player := ui.sceneInfo.GetPlayer()
	if player != nil {
		fmt.Println("Character Aspects:")
		for _, aspect := range player.Aspects.GetAll() {
			fmt.Printf("  - %s\n", aspect)
		}
	}

	scene := ui.sceneInfo.GetCurrentScene()
	if scene != nil && len(scene.SituationAspects) > 0 {
		fmt.Println("\nSituation Aspects:")
		for _, aspect := range scene.SituationAspects {
			invokes := ""
			if aspect.FreeInvokes > 0 {
				invokes = fmt.Sprintf(" (%d free)", aspect.FreeInvokes)
			}
			fmt.Printf("  - %s%s\n", aspect.Aspect, invokes)
		}
	}
}

func (ui *TerminalUI) displayConversationHistory() {
	fmt.Println("\n=== Recent Conversation ===")

	history := ui.sceneInfo.GetConversationHistory()
	if len(history) == 0 {
		fmt.Println("No conversation history yet.")
		return
	}

	// Show last 5 exchanges
	start := len(history) - 5
	if start < 0 {
		start = 0
	}

	for i := start; i < len(history); i++ {
		entry := history[i]
		fmt.Printf("\n[%s] You: %s\n", entry.Type, entry.PlayerInput)
		fmt.Printf("GM: %s\n", entry.GMResponse)
	}
}

// displayConflictStart displays the start of a conflict with initiative order
func (ui *TerminalUI) displayConflictStart(conflictType string, initiatorName string, participants []uicontract.ConflictParticipantInfo) {
	fmt.Println("\n╔══════════════════════════════════════════╗")
	fmt.Printf("║         CONFLICT BEGINS!                 ║\n")
	fmt.Printf("║         Type: %-25s  ║\n", conflictType)
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("\n%s initiates hostilities!\n", initiatorName)

	fmt.Println("\n--- Initiative Order ---")
	for i, p := range participants {
		marker := "  "
		if p.IsPlayer {
			marker = "► "
		}
		fmt.Printf("%s%d. %s (Initiative: %d)\n", marker, i+1, p.CharacterName, p.Initiative)
	}
	fmt.Println("------------------------")
}

// displayConflictEscalation displays when a conflict escalates to a different type
func (ui *TerminalUI) displayConflictEscalation(fromType, toType, triggerCharName string) {
	fmt.Println("\n┌──────────────────────────────────────────┐")
	fmt.Printf("│  CONFLICT ESCALATES!                     │\n")
	fmt.Printf("│  %s → %s\n", fromType, toType)
	fmt.Printf("│  Triggered by: %s\n", triggerCharName)
	fmt.Println("└──────────────────────────────────────────┘")
	fmt.Println("\nInitiative is being recalculated...")
}

// displayTurnAnnouncement displays whose turn it is in the conflict
func (ui *TerminalUI) displayTurnAnnouncement(characterName string, turnNumber int, isPlayer bool) {
	if isPlayer {
		fmt.Printf("\n=== Turn %d: Your turn, %s! ===\n", turnNumber, characterName)
	} else {
		fmt.Printf("\n=== Turn %d: %s's turn ===\n", turnNumber, characterName)
	}
}

// displayConflictEnd displays the end of a conflict
func (ui *TerminalUI) displayConflictEnd(reason string) {
	fmt.Println("\n╔══════════════════════════════════════════╗")
	fmt.Printf("║         CONFLICT ENDS                    ║\n")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("\n%s\n", reason)
}

// displayGameOver displays the game over screen
func (ui *TerminalUI) displayGameOver(reason string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║              GAME OVER                   ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("\n%s\n", reason)
	fmt.Println("\nYour adventure has come to an end.")
	fmt.Println("Thank you for playing!")
}

// displaySceneTransition displays a transition to a new scene
func (ui *TerminalUI) displaySceneTransition(narrative string, newSceneHint string) {
	fmt.Println()
	fmt.Println("════════════════════════════════════════════")
	fmt.Println("              Scene Transition              ")
	fmt.Println("════════════════════════════════════════════")
	fmt.Printf("\n%s\n", narrative)
	if newSceneHint != "" {
		fmt.Printf("\n[%s]\n", newSceneHint)
	}
	fmt.Println()
	fmt.Println("════════════════════════════════════════════")
}

// displayDamageResolution renders a DamageResolutionEvent (NPC taking damage).
func (ui *TerminalUI) displayDamageResolution(e uicontract.DamageResolutionEvent) {
	if e.Absorbed != nil {
		fmt.Printf("\n%s absorbs the damage with their %s stress track.\n", e.TargetName, e.Absorbed.TrackType)
		return
	}
	if e.Consequence != nil {
		fmt.Printf("\n%s takes a %s consequence: \"%s\" (absorbs %d shifts)\n",
			e.Consequence.TargetName, e.Consequence.Severity, e.Consequence.Aspect, e.Consequence.Absorbed)
	}
	if e.RemainingAbsorbed != nil {
		fmt.Printf("\n%s absorbs remaining %d shifts with stress.\n",
			e.TargetName, e.RemainingAbsorbed.Shifts)
	}
	if e.TakenOut {
		fmt.Printf("\n=== %s is Taken Out! ===\n", e.TargetName)
	}
	if e.VictoryEnd {
		fmt.Printf("\n=== Victory! All opponents defeated! ===\n")
	}
}

// displayPlayerAttackResult renders a PlayerAttackResultEvent.
func (ui *TerminalUI) displayPlayerAttackResult(e uicontract.PlayerAttackResultEvent) {
	if e.TargetMissing {
		fmt.Printf("\nCould not find target '%s' — attack has no effect.\n", e.TargetHint)
		return
	}
	if e.IsTie {
		fmt.Printf("\nTie! You gain a boost against your opponent.\n")
		return
	}
	if e.Shifts > 0 {
		fmt.Printf("\nYour attack deals %d shifts to %s!\n", e.Shifts, e.TargetName)
	}
}

// displayNPCAttack renders an NPCAttackEvent.
func (ui *TerminalUI) displayNPCAttack(e uicontract.NPCAttackEvent) {
	fmt.Printf("\n%s attacks %s with %s (%s) vs %s (%s)\n",
		e.AttackerName, e.TargetName, e.AttackSkill, e.AttackResult, e.DefenseSkill, e.DefenseResult)
	if e.FinalOutcome != e.InitialOutcome {
		fmt.Printf("\nFinal outcome: %s\n", e.FinalOutcome)
	}
	if e.Narrative != "" {
		fmt.Printf("\n%s\n", e.Narrative)
	}
}

// displayPlayerConsequence renders a PlayerConsequenceEvent.
func (ui *TerminalUI) displayPlayerConsequence(e uicontract.PlayerConsequenceEvent) {
	fmt.Printf("\nYou take a %s consequence: \"%s\"\n", e.Severity, e.Aspect)
	fmt.Printf("\nThe consequence absorbs %d shifts.\n", e.Absorbed)
	if e.StressAbsorbed != nil {
		fmt.Printf("\nYou absorb the remaining %d shifts as stress. (%s)\n",
			e.StressAbsorbed.Shifts, e.StressAbsorbed.TrackState)
	}
}

// displayPlayerTakenOut renders a PlayerTakenOutEvent.
func (ui *TerminalUI) displayPlayerTakenOut(e uicontract.PlayerTakenOutEvent) {
	fmt.Printf("\n=== You Are Taken Out! ===\n")
	fmt.Printf("\n%s decides your fate.\n", e.AttackerName)

	switch e.Outcome {
	case "game_over":
		fmt.Printf("\n%s\n", e.Narrative)
		ui.displayGameOver(fmt.Sprintf("%s has met their end.", e.AttackerName))
	case "transition":
		ui.displaySceneTransition(e.Narrative, e.NewSceneHint)
		fmt.Printf("\nThe scene shifts around you...\n")
	default: // "continue"
		fmt.Printf("\n%s\n", e.Narrative)
		ui.displayConflictEnd(fmt.Sprintf("%s has won the conflict.", e.AttackerName))
	}
}

// displayConcession renders a ConcessionEvent.
func (ui *TerminalUI) displayConcession(e uicontract.ConcessionEvent) {
	fmt.Printf("\n=== You Concede! ===\n")
	fmt.Printf("\nYou choose to lose the conflict on your own terms.\n")
	fmt.Printf("\nYou get to narrate how you exit the scene and avoid the worst consequences.\n")
	if e.ConsequenceCount > 0 {
		fmt.Printf("\nYou gain %d Fate Points (1 for conceding + %d for consequences)! (Now: %d)\n",
			e.FatePointsGained, e.ConsequenceCount, e.CurrentFatePoints)
	} else {
		fmt.Printf("\nYou gain a Fate Point for conceding! (Now: %d)\n", e.CurrentFatePoints)
	}
}

// isExitCommand checks if the input is an exit command
func (ui *TerminalUI) isExitCommand(input string) bool {
	exitCommands := []string{"exit", "quit", "end", "leave", "resolve"}
	lowerInput := strings.ToLower(strings.TrimSpace(input))

	for _, cmd := range exitCommands {
		if lowerInput == cmd {
			return true
		}
	}

	return false
}
