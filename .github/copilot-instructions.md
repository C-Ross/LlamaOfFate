# LlamaOfFate - Copilot Instructions

## Project Overview
Text-based RPG implementing Fate Core System with LLM integration. Built in Go with modular architecture.

Core premise is to leverage LLMs for narrative generation and player input parsing while maintaining a robust implementation of Fate mechanics.

## Repository Structure
```
cmd/cli/                    - CLI entry point
examples/                   - Evaluation tools (llm-scene-loop, scenario-generator, scenario-walkthrough, scene-generator)
internal/
  core/                     - Fate Core mechanics (action, character, dice, scene)
  engine/                   - Game loop, LLM orchestration (game_manager → scene_flow → scene_manager)
  prompt/                   - LLM prompt system (Go templates, data structs, marker parsing)
  llm/                      - LLM client interface and retry logic (azure/ implementation)
  logging/                  - Structured logging (slog)
  session/                  - Session logging for game transcripts (YAML)
  ui/terminal/              - Terminal UI implementation
test/
  integration/              - Integration tests
  llmeval/                  - LLM behavior evaluation tests (requires -tags=llmeval)
configs/                    - Configuration files (azure-llm.yaml)
```

## Development Standards

Prefer early returns to reduce nesting.

Don't store llm prompts as raw strings in code; use Go templates instead.

Avoid duplication by creating resuable functions and stucts.  If you think you must duplicate, ask the user for clarification on how to refactor instead.

Ensure `just validate` passes before committing: all tests and linters must succeed.

### Testing (REQUIRED)
- Use testify for ALL tests: `assert.Equal(t, expected, actual)`, `require.NotNil(t, object)`
- Maintain high coverage: Unit tests per package + integration tests
- Specify dice rolls: Whenever reasonable, specify the dice roll instead of using a roller.
- Seeded rollers for tests: `dice.NewSeededRoller(12345)` for predictable results if you need a roller.

### Build System
```bash
just build      # Build application
just test       # Run all tests
just run        # Build and run
just clean      # Clean artifacts
just lint       # Run linters
just validate   # Run tests and linters
just test-llm   # Run llm evaluate tests - may consume resources
```

### Format
- Format according to `go fmt`
- Use Go template for all prompts, do not in line prompt text generation

### Data Formats
- **Prefer YAML over JSON** for configuration, data files, and logs (readability for long text)
- **Exception: LLM structured responses** use JSON (industry standard, better parsing reliability)
- Config files use YAML (e.g., `azure-llm.yaml`)

## Key Patterns

### Dynamic Character Aspects (NOT traditional 5-aspect)
```go
char.Aspects.HighConcept = "Wizard Detective"
char.Aspects.Trouble = "The Lure of Ancient Mysteries"
char.Aspects.AddAspect("Well Connected")  // Unlimited additional aspects
```

When you need to consider the rules use the Fate Core SRD at https://fate-srd.com/fate-core.

## Important Notes
- Character aspects are **flexible** (not fixed 5-aspect model)
- Always use testify assertions in tests
- Follow existing package structure and naming
- Credit the Fate SRD where appropriate.

## Session Logging

When adding features that affect gameplay flow, add logging in `scene_manager.go`:
`sm.sessionLogger.Log("event_type", map[string]any{...})`
Or log existing structs directly: `sm.sessionLogger.Log("action_parse", parsedAction)`
