# Engine Test Patterns

This document describes the test patterns used in the `engine` package.
Copilot agents should follow these patterns when adding new tests.

## Test Layers

The engine has two distinct test layers for action resolution:

### 1. Unit Tests (`applyActionEffects`)

Inject a pre-built `*action.Action` with `Outcome` already set.
**Bypass**: dice rolling, invoke loop, narrative generation.
**Purpose**: Verify that a given outcome produces correct mechanical effects (aspect creation, damage, boosts).

```go
testAction.Outcome = &dice.Outcome{Type: dice.Tie, Shifts: 0}
events := sm.actions.applyActionEffects(ctx, testAction, target)
```

These tests live in `conflict_test.go` (attack/CaA/overcome effects) and `scene_manager_test.go`.

### 2. Pipeline Tests (`resolveAction`)

Exercise the **full** `resolveAction` pipeline: dice roll → invoke loop → narrative → effects → scene mutations.
**Purpose**: Verify that outcomes propagate correctly through every layer, including the invoke loop.

These tests use a shared setup helper per action type and are differentiated by filename: `{action_type}_test.go`.

#### Pattern

Each action type gets its own file (`create_advantage_test.go`, `attack_test.go`, `overcome_test.go`) with:

1. **A `setup*SM` helper** that creates a `SceneManager` with `MockLLMClient`, a player character, and an active scene.
2. **One test per outcome tier** (Failure, Tie, Success, SWS) using `PlannedRoller` to force the exact dice total.
3. **Invoke-path tests** where the player has fate points and the invoke loop fires.

```go
func setupFooSM(t *testing.T, fatePoints int) *SceneManager {
    t.Helper()
    mockClient := &MockLLMClient{response: `{"aspect_text":"X","description":"Y","reasoning":"Z"}`}
    engine, err := NewWithLLM(mockClient)
    require.NoError(t, err)

    sm := engine.GetSceneManager()
    player := character.NewCharacter("player-1", "Hero")
    player.FatePoints = fatePoints
    player.SetSkill("Notice", dice.Fair)   // Set the skill under test
    engine.AddCharacter(player)

    testScene := scene.NewScene("test-scene", "Test Room", "A room for testing.")
    testScene.AddCharacter(player.ID)
    err = sm.StartScene(testScene, player)
    require.NoError(t, err)
    return sm
}
```

Each test then:
1. Swaps the roller: `sm.actions.roller = dice.NewPlannedRoller([]int{total})`
2. Creates an action with a known `Difficulty`
3. Calls `sm.actions.resolveAction(ctx, testAction)`
4. Asserts on: `testAction.Outcome.Type`, emitted `GameEvent` types, and scene state mutations

#### Dice Total Cheat Sheet

With skill `Fair (+2)` vs difficulty `Fair (+2)`:

| Desired Outcome | Dice Total | Final Value | Shifts |
|---|---|---|---|
| Failure | -1 | Average (+1) | -1 |
| Tie | 0 | Fair (+2) | 0 |
| Success | 1 | Good (+3) | +1 |
| Success with Style | 3 | Superb (+5) | +3 |

#### Attack Pipeline Notes

Attack tests need a target NPC in the scene. `resolveAction` auto-initiates a conflict
for attacks. After the player's turn, `advanceConflictTurns` processes the NPC's
counter-attack turn, which consumes **two additional** `PlannedRoller` entries
(NPC attack roll + player defense roll).

Total rolls per attack test: **4 minimum**.

```
[0] player attack dice
[1] NPC defense dice
[2] NPC counter-attack dice
[3] player defense dice (against NPC)
```

**Important:** Use `[-1, 0]` for the NPC counter-attack rolls (positions 2-3).
This makes the NPC fail by 1 shift (`Fight(+2)+(-1)=+1` vs `Athletics(+2)+0=+2`),
which avoids creating spurious boosts (ties create boosts) or stress (success deals damage)
that would interfere with assertions about the player's original attack.

The `MockLLMClient` response for attack setups should be valid JSON matching the
NPC action decision schema so the NPC falls through to the expected attack behavior:

```json
{"action":"attack","skill":"Fight","target":"player-1","description":"counter-attack","reasoning":"test"}
```

#### Event Assertion Helpers

Use the generic helpers from `event_recorder_test.go`:

- `RequireFirstFrom[T](t, events)` — asserts exactly one event of type `T` exists and returns it
- `SliceOfType[T](events)` — returns all events of type `T`
- `AssertNoEventIn[T](t, events)` — asserts no event of type `T` exists

## Existing Files

| File | Layer | Scope |
|---|---|---|
| `conflict_test.go` | Unit | `applyActionEffects` for Attack, CaA, Overcome; conflict management |
| `scene_manager_test.go` | Unit | `applyActionEffects` for CaA; `HandleInput` routing; active opposition |
| `create_advantage_test.go` | Pipeline | CaA: Tie, Success, SWS, Failure, Tie+invoke-skip |
| `attack_test.go` | Pipeline | Attack: Tie (boost), Success (damage), SWS (damage), Failure, Failure-defend-SWS |
| `overcome_test.go` | Pipeline | Overcome: Tie (succeed at cost), Success, SWS (boost), Failure |
| `invoke_test.go` | Unit | `buildInvokePrompt`, `ProvideInvokeResponse` mechanics |
