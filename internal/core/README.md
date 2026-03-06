# Core Package

Implements the fundamental Fate Core game mechanics.

## Sub-packages

- **`dice/`** - Fate ladder system and 4dF rolling
- **`action/`** - Four action types and resolution
- **`scene/`** - Scene management and conflict system

## Files in this package

- **`character.go`** - Characters with aspects, skills, stress, consequences
- **`character_type.go`** - NPC threat levels (nameless, supporting, main NPC, PC)
- **`skills.go`** - Skill classification (attack/defense mappings, stress types, initiative)
- **`skills_list.go`** - Named constants for the 18 default Fate Core skills

## Usage

```go
// Create a character
char := core.NewCharacter("hero", "Zara the Bold")
char.Aspects.HighConcept = "Daring Sky Pirate"
char.SetSkill(core.SkillAthletics, dice.Good)

// Create a scene
sc := scene.NewScene("deck", "Burning Airship", "...")
sc.AddCharacter(char.ID)

// Create and resolve an action
act := action.NewAction("leap", char.ID, action.Overcome, core.SkillAthletics, "Jump the gap")
act.Difficulty = dice.Great

// Roll and resolve
roller := dice.NewRoller()
result := roller.RollWithModifier(char.GetSkill(core.SkillAthletics), 0)
act.Outcome = result.CompareAgainst(act.Difficulty)
```

## Test Coverage

- **67 unit tests** with 79-100% coverage per package
- Run tests: `go test ./internal/core/... -v`
