# LlamaOfFate - Copilot Instructions

## Project Overview
Text-based RPG implementing Fate Core System with LLM integration. Built in Go with modular architecture.

## Repository Structure
```
cmd/cli/                    - CLI entry point (single hardcoded scene currently)
examples/llm-scene-loop/    - Example scenes (saloon, heist, tower)
internal/
  core/                     - Fate Core mechanics
    action/                 - Action types (Overcome, Create Advantage, Attack, Defend)
    character/              - Character, aspects, stress, consequences
    dice/                   - 4dF dice, ladder, check results
    scene/                  - Scene state, conflicts, situation aspects
  engine/                   - Game loop, LLM integration, prompt templates
  llm/                      - LLM client interface and retry logic
    azure/                  - Azure OpenAI client implementation
  session/                  - Session logging for game transcripts
  ui/terminal/              - Terminal UI implementation
test/
  integration/              - Integration tests
  llmeval/                  - LLM behavior evaluation tests (requires -tags=llmeval)
configs/                    - Configuration files (azure-llm.yaml)
```

## Development Standards

Prefer early returns to reduce nesting.

Don't store llm prompts as raw strings in code; use Go templates instead.

### Testing (REQUIRED)
- Use testify for ALL tests: `assert.Equal(t, expected, actual)`, `require.NotNil(t, object)`
- Maintain high coverage: Unit tests per package + integration tests
- Specify dice rolls: Whenever reasonable, specify the dice roll instead of using a roller.
- Seeded rollers for tests: `dice.NewSeededRoller(12345)` for predictable results if you need a roller.

### Build System
```bash
make build      # Build application
make test       # Run all tests
make run        # Build and run
make clean      # Clean artifacts
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

### Dice & Actions
```go
roller := dice.NewRoller()
action := action.NewAction(id, charID, actionType, skill, description)
action.AddAspectInvoke(aspectInvoke)  // Fate points/free invokes
result := roller.RollWithModifier(skill, action.CalculateBonus())
```

## Fate Core Mechanics
- **Ladder**: Terrible (-2) to Legendary (+8)+
- **4dF Dice**: -1/0/+1 (Minus/Blank/Plus)
- **Stress**: Physical/Mental tracks, configurable boxes
- **Consequences**: Mild(2), Moderate(4), Severe(6), Extreme(8)

When you need to consider the rules use the Fate Core SRD at https://fate-srd.com/fate-core.

## Important Notes
- Character aspects are **flexible** (not fixed 5-aspect model)
- Always use testify assertions in tests
- Follow existing package structure and naming
- Credit the Fate SRD where appropriate.

## Session Logging

Session logs (`session_*.yaml`) capture gameplay for analysis and test extraction.

### Reading Logs
- Logs are YAML with `---` separators between entries
- Each entry has `timestamp`, `type`, and `data`
- Key types: `player_input`, `input_classification`, `action_parse`, `dice_roll`, `narrative`, `dialog`, `taken_out`

### Adding Log Events
When adding features that affect gameplay flow, add logging in `scene_manager.go`:
`sm.sessionLogger.Log("event_type", map[string]any{...})`
Or log existing structs directly: `sm.sessionLogger.Log("action_parse", parsedAction)`
