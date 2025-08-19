package engine

import "github.com/C-Ross/LlamaOfFate/internal/llm"

// Engine represents the core game engine
type Engine struct {
	actionParser *ActionParser
	llmClient    llm.LLMClient
}

// New creates a new game engine instance
func New() (*Engine, error) {
	return &Engine{}, nil
}

// NewWithLLM creates a new game engine instance with an LLM client
func NewWithLLM(llmClient llm.LLMClient) (*Engine, error) {
	engine := &Engine{
		llmClient:    llmClient,
		actionParser: NewActionParser(llmClient),
	}
	return engine, nil
}

// Start initializes the game engine
func (e *Engine) Start() error {
	return nil
}

// Stop shuts down the game engine
func (e *Engine) Stop() error {
	return nil
}

// GetVersion returns the engine version
func (e *Engine) GetVersion() string {
	return "0.1.0"
}

// GetActionParser returns the action parser instance
func (e *Engine) GetActionParser() *ActionParser {
	return e.actionParser
}
