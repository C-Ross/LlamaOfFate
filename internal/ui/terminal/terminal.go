package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
)

// TerminalUI implements the UI interface for terminal-based interaction
type TerminalUI struct {
	reader    *bufio.Reader
	sceneInfo engine.SceneInfo
}

// NewTerminalUI creates a new terminal UI instance
func NewTerminalUI() *TerminalUI {
	return &TerminalUI{
		reader: bufio.NewReader(os.Stdin),
	}
}

// SetSceneInfo sets the scene information for command handling
func (ui *TerminalUI) SetSceneInfo(sceneInfo engine.SceneInfo) {
	ui.sceneInfo = sceneInfo
}

// ReadInput reads input from the terminal and returns cleaned input, exit status, and error
func (ui *TerminalUI) ReadInput() (input string, isExit bool, err error) {
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

	// Handle special commands internally if scene info is available
	if ui.sceneInfo != nil && ui.handleSpecialCommands(input) {
		// Command was handled, return empty input to indicate no further processing needed
		return "", false, nil
	}

	return input, false, nil
}

// DisplayActionAttempt displays that the player is attempting an action
func (ui *TerminalUI) DisplayActionAttempt(description string) {
	fmt.Printf("\nYou attempt to: %s\n", description)
}

// DisplayActionResult displays the mechanical result of an action (dice roll, bonuses, outcome)
func (ui *TerminalUI) DisplayActionResult(skill string, skillLevel string, bonuses int, result string, outcome string) {
	fmt.Printf("Skill (%s): %s\n", skill, skillLevel)
	if bonuses != 0 {
		fmt.Printf("Bonuses: %+d\n", bonuses)
	}
	fmt.Printf("Rolled: %s\n", result)
	fmt.Printf("Outcome: %s\n", outcome)
}

// DisplayNarrative displays narrative text from the GM
func (ui *TerminalUI) DisplayNarrative(narrative string) {
	fmt.Printf("\n%s\n", narrative)
}

// DisplayDialog displays a dialog exchange between player and GM
func (ui *TerminalUI) DisplayDialog(playerInput, gmResponse string) {
	fmt.Printf("\nYou: %s\n", playerInput)
	fmt.Printf("\nGM: %s\n", gmResponse)
}

// DisplaySystemMessage displays system messages to the player
func (ui *TerminalUI) DisplaySystemMessage(message string) {
	fmt.Printf("\n%s\n", message)
}

// handleSpecialCommands processes special scene commands and returns true if handled
func (ui *TerminalUI) handleSpecialCommands(input string) bool {
	parts := strings.Fields(strings.ToLower(input))
	if len(parts) == 0 {
		return false
	}

	command := parts[0]

	switch command {
	case "help":
		ui.showHelp()
		return true
	case "scene":
		ui.displayScene()
		return true
	case "character", "char":
		ui.displayCharacter()
		return true
	case "status":
		ui.displayStatus()
		return true
	case "aspects":
		ui.displayAspects()
		return true
	case "history", "conversation":
		ui.displayConversationHistory()
		return true
	}

	return false
}

// Display methods
func (ui *TerminalUI) displayScene() {
	if ui.sceneInfo == nil {
		fmt.Println("No scene information available.")
		return
	}

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

func (ui *TerminalUI) displayStatus() {
	if ui.sceneInfo == nil {
		fmt.Println("No scene information available.")
		return
	}

	player := ui.sceneInfo.GetPlayer()
	if player == nil {
		return
	}

	fmt.Println("\n=== Status ===")

	// Show stress tracks
	for trackType, track := range player.StressTracks {
		fmt.Printf("%s: %s\n", strings.ToUpper(trackType[:1])+trackType[1:], track.String())
	}

	// Show consequences
	if len(player.Consequences) > 0 {
		fmt.Println("\nConsequences:")
		for _, consequence := range player.Consequences {
			fmt.Printf("  %s: %s\n", consequence.Type, consequence.Aspect)
		}
	}
}

func (ui *TerminalUI) displayAspects() {
	if ui.sceneInfo == nil {
		fmt.Println("No scene information available.")
		return
	}

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

// DisplayConflictStart displays the start of a conflict with initiative order
func (ui *TerminalUI) DisplayConflictStart(conflictType string, initiatorName string, participants []engine.ConflictParticipantInfo) {
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

// DisplayConflictEscalation displays when a conflict escalates to a different type
func (ui *TerminalUI) DisplayConflictEscalation(fromType, toType, triggerCharName string) {
	fmt.Println("\n┌──────────────────────────────────────────┐")
	fmt.Printf("│  CONFLICT ESCALATES!                     │\n")
	fmt.Printf("│  %s → %s\n", fromType, toType)
	fmt.Printf("│  Triggered by: %s\n", triggerCharName)
	fmt.Println("└──────────────────────────────────────────┘")
	fmt.Println("\nInitiative is being recalculated...")
}

// DisplayTurnAnnouncement displays whose turn it is in the conflict
func (ui *TerminalUI) DisplayTurnAnnouncement(characterName string, turnNumber int, isPlayer bool) {
	if isPlayer {
		fmt.Printf("\n=== Turn %d: Your turn, %s! ===\n", turnNumber, characterName)
	} else {
		fmt.Printf("\n=== Turn %d: %s's turn ===\n", turnNumber, characterName)
	}
}

// DisplayConflictEnd displays the end of a conflict
func (ui *TerminalUI) DisplayConflictEnd(reason string) {
	fmt.Println("\n╔══════════════════════════════════════════╗")
	fmt.Printf("║         CONFLICT ENDS                    ║\n")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("\n%s\n", reason)
}

// DisplayGameOver displays the game over screen
func (ui *TerminalUI) DisplayGameOver(reason string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║              GAME OVER                   ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Printf("\n%s\n", reason)
	fmt.Println("\nYour adventure has come to an end.")
	fmt.Println("Thank you for playing!")
}

// DisplaySceneTransition displays a transition to a new scene
func (ui *TerminalUI) DisplaySceneTransition(narrative string, newSceneHint string) {
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

func (ui *TerminalUI) displayConversationHistory() {
	if ui.sceneInfo == nil {
		fmt.Println("No scene information available.")
		return
	}

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

// Ensure TerminalUI implements the UI interface
var _ engine.UI = (*TerminalUI)(nil)
