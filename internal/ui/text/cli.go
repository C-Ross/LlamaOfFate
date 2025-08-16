package text

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/C-Ross/LlamaOfFate/internal/engine"
)

// CLI represents the command-line interface
type CLI struct {
	engine *engine.Engine
	reader *bufio.Reader
}

// NewCLI creates a new CLI instance
func NewCLI(engine *engine.Engine) (*CLI, error) {
	if engine == nil {
		return nil, fmt.Errorf("engine cannot be nil")
	}

	return &CLI{
		engine: engine,
		reader: bufio.NewReader(os.Stdin),
	}, nil
}

// Run starts the interactive CLI loop
func (c *CLI) Run() error {
	for {
		fmt.Print("> ")

		input, err := c.reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Clean up the input
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Process the command
		if shouldExit := c.processCommand(input); shouldExit {
			break
		}
	}

	return nil
}

// processCommand handles a single command and returns true if the CLI should exit
func (c *CLI) processCommand(input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "quit", "exit", "q":
		fmt.Println("Goodbye!")
		return true
	case "help", "h":
		c.showHelp()
	case "version":
		fmt.Printf("Engine version: %s\n", c.engine.GetVersion())
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Type 'help' for available commands.")
	}

	return false
}

// showHelp displays available commands
func (c *CLI) showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help, h        - Show this help message")
	fmt.Println("  version        - Show engine version")
	fmt.Println("  quit, exit, q  - Exit the application")
	fmt.Println()
	fmt.Println("More commands will be added as the system is developed.")
}
