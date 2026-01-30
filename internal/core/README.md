# Core Package

Implements the fundamental Fate Core game mechanics.

## Packages

- **`dice/`** - Fate ladder system and 4dF rolling
- **`character/`** - Characters with aspects, skills, stress, consequences  
- **`action/`** - Four action types and resolution
- **`scene/`** - Scene management and conflict system
- **`skills.go`** - Skill classification (attack/defense mappings, stress types, initiative)

## Usage

```go
// Create a character
char := character.NewCharacter("hero", "Zara the Bold")
char.Aspects.HighConcept = "Daring Sky Pirate"
char.SetSkill("Athletics", dice.Good)

// Create a scene
scene := scene.NewScene("deck", "Burning Airship", "...")
scene.AddCharacter(char.ID)

// Create and resolve an action
action := action.NewAction("leap", char.ID, action.Overcome, "Athletics", "Jump the gap")
action.Difficulty = dice.Great

// Roll and resolve
roller := dice.NewRoller()
result := roller.RollWithModifier(char.GetSkill("Athletics"), 0)
action.Outcome = result.CompareAgainst(action.Difficulty)
```

## Test Coverage

- **67 unit tests** with 79-100% coverage per package
- **4 integration tests** demonstrating gameplay scenarios
- Run tests: `go test ./internal/core/... -v`
