# LlamaOfFate - Copilot Instructions

## Project Overview
Text-based RPG implementing Fate Core System with LLM integration. Built in Go with modular architecture.

## Architecture
```
cmd/cli/           - CLI interface and entry point
internal/core/     - Core Fate mechanics (dice, character, action, scene)
internal/engine/   - Game engine coordination
internal/ui/text/  - Text-based UI
test/integration/  - Integration tests
```

## Development Standards

### Testing (REQUIRED)
- **Use testify for ALL tests**: `assert.Equal(t, expected, actual)`, `require.NotNil(t, object)`
- **Maintain high coverage**: Unit tests per package + integration tests
- **Seeded rollers for tests**: `dice.NewSeededRoller(12345)` for predictable results

### Build System
```bash
make build      # Build application
make test       # Run all tests
make run        # Build and run
make clean      # Clean artifacts
```

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
