````skill
---
name: game-architecture
description: Guide for modifying game flow, adding features to the game loop, or debugging scene/scenario progression. Use this when asked to work with the engine lifecycle, input pipeline, conflict orchestration, or NPC behavior.
---

# Game Architecture / Engine Lifecycle

Orchestration layers in `internal/engine/` between `internal/core/` rules and `internal/prompt/` LLM layer.

## Layer Overview

```
GameManager          ← scenario lifecycle, milestones, save/load, fate point refresh
  └─ ScenarioManager ← multi-scene loop, scene generation, summaries, NPC registry, recovery
       └─ SceneManager ← single-scene loop, input classification, dialog/action, conflict turns
            ├─ ActionParser    ← LLM: free-text → structured action
            ├─ AspectGenerator ← LLM: Create Advantage → aspect name
            └─ invoke.go       ← post-roll aspect invocation loop
```

Each layer creates/configures the layer below. **Do not skip layers.**

## Engine (`engine.go`)

Shared kernel: LLM client, character registry, creates `SceneManager` + `ActionParser`.

Key methods:
- `NewWithLLM(client)` — constructor
- `AddCharacter` / `GetCharacter` / `GetCharacterByName` / `ResolveCharacter` — character registry (ID, name, or `"Name (ID)"` format)
- `GetSceneManager()` — returns `SceneManager` instance

## GameManager (`game_manager.go`)

Top-level orchestrator. Creates `ScenarioManager` inside `Start()`.

```go
gm := engine.NewGameManager(eng)
gm.SetPlayer(player)
gm.SetUI(terminalUI)
gm.SetSessionLogger(logger)    // optional
gm.SetScenario(scenario)       // optional: otherwise ScenarioManager generates one
gm.SetSaver(saver)             // optional: defaults to no-op
gm.Run(ctx)                    // terminal blocking loop
gm.RunWithInitialScene(ctx, &engine.InitialSceneConfig{...})  // demo/test

// Event-driven (web) path:
events, _ := gm.Start(ctx)              // opening GameEvents
result, _ := gm.HandleInput(ctx, input)  // InputResult with events
```

`Run()` wires `SceneInfoSetter` on the UI after `Start()` so terminal meta-commands work.

**Milestones** (on `ScenarioEndResolved`): refresh fate points, check consequence recovery, increment `scenarioCount`.

## ScenarioManager (`scenario_manager.go`)

Multi-scene loop within a single scenario. Created by `GameManager.Start()`.

```
getInitialScene() or generateNextScene()
  → SceneManager.StartScene() + HandleInput loop
  → scene ends → generateSceneSummary()
  → checkScenarioResolution() → resolved? → EXIT
  → handleBetweenSceneRecovery()
  → generateNextScene(transitionHint) → LOOP
```

**End reasons** (`scene_flow.go`):

| Scene end | Scenario end |
|-----------|-------------|
| `SceneEndQuit` → `ScenarioEndQuit` | Player quit |
| `SceneEndPlayerTakenOut` → `ScenarioEndPlayerTakenOut` or transition | Player taken out |
| `SceneEndTransition` → summary → resolution check → next scene | Scene transition |

### NPC registry

Named NPCs persist across scenes via `npcRegistry` (keyed by normalized name). Reused when a matching name appears in generated scenes. `npcAttitudes` tracks last-known attitude per NPC.

### LLM pipelines

**Scene generation**: `SceneGenerationData` (transition hint, scenario, player, last 3 summaries, complications, known NPCs) → `prompt.RenderSceneGeneration()` → LLM → `ParseGeneratedScene()` → `scene.NewScene()` + register NPCs

**Scene summary**: `SceneSummaryData` (scene, conversation, NPCs, how ended) → LLM → `SceneSummary` (summary, key events, NPCs, unresolved threads)

**Resolution check**: `ScenarioResolutionData` (scenario, summaries, player) → LLM → `ScenarioResolutionResult` (resolved?, answered questions)

### Between-scene recovery

1. Check if recovering consequences have healed
2. Roll 4dF + best recovery skill vs difficulty (consequence value + 2 for self-treatment)
3. On success → `player.BeginConsequenceRecovery()`, LLM narrative for results

## SceneManager (`scene_manager.go`)

Single scene's interactive loop. Maintains conversation history (last 10, sliding window) and scene end state.

### Input pipeline

Meta-commands (`help`, `character`, `status`, `aspects`, `scene`, `history`) are intercepted by the **UI layer** (`terminal.go handleSpecialCommands()`), not the engine. `HandleInput()` receives only game input:

```
HandleInput(input)
  → pending invoke/mid-flow? → resume
  → isConcedeCommand? (conflict only) → handleConcession()
  → classifyInput() via LLM → "dialog"|"clarification"|"narrative"|"action"
      dialog/clarification/narrative → handleDialog() → generateSceneResponse()
                                        → parse markers ([CONFLICT_START:], [CONFLICT_END:], [SCENE_TRANSITION:])
      action → handleAction() → ActionParser.ParseAction()
                → resolveAction() → roll dice → applyActionEffects()
                → generateActionNarrative()
```

### Dialog flow

`generateSceneResponse()`: builds context → chooses template (`RenderConflictResponse` if in conflict, else `RenderSceneResponse`) → LLM → parse markers → handle triggered markers.

### Action flow

`handleAction()`: `ActionParser.ParseAction()` → `resolveAction()` (auto-initiates conflict for attacks) → `applyActionEffects()` → `generateActionNarrative()`.

## Conflict System (`conflict.go`)

Methods on `SceneManager`. Triggered by `[CONFLICT_START:]` markers or player Attack actions.

### Lifecycle

```
initiateConflict(type, initiator) → build participants, calculate initiative
  → player acts → advanceConflictTurns() → NPC turns via processNPCTurn() → back to player
End: all opponents taken out | player concedes | [CONFLICT_END:] marker | player taken out
```

### Action resolution (`resolveAction`)

1. Attack without conflict → auto-initiate; type mismatch → escalate
2. Roll 4dF + skill; for attacks: roll target defense (+2 if full defense)
3. `handlePostRollInvokes()` → invoke prompt loop
4. `applyActionEffects()`:
   - **Attack success** → `applyDamageToTarget()` → stress → consequences → taken out
   - **Create Advantage** → `AspectGenerator` → situation aspect (tie=boost, success=1 free invoke, style=2)
   - **Overcome/Defend** → narrative only

### Invocation (`invoke.go`)

All invoke logic in `invoke.go` (called from `conflict.go` and `npc.go`):
1. `gatherInvokableAspects()` → character, situation (with free invokes), consequence aspects
2. `beginInvokeLoop()` → emits `InvokePromptEvent`, sets `sm.pendingInvoke`
3. Terminal: `GameManager.driveBlockingPrompts()` type-asserts UI to `InvokePrompter`
4. Apply +2 or reroll; spend fate point or free invoke; loop until declined
5. NPC defense invokes: `resumeTurns` flag triggers `maybeResumeConflictTurns`

### Damage

`applyDamageToTarget(target, shifts, stressType)`: try `TakeStress()` → `fillTargetStressOverflow()` (consequence slots) → `applyTargetTakenOut()`. Player taken-out uses LLM to classify outcome (continue/transition/game over).

### Concession

`handleConcession()`: must be before a roll. Awards 1 + consequences-taken fate points, generates LLM narrative, ends conflict.

### Escalation

`handleConflictEscalation(newType)` — changes conflict type, recalculates initiative.

## NPC Turns (`npc.go`)

```
processNPCTurn(npcID)
  → getNPCActionDecision() via LLM (temp=0.7)
  → ATTACK | CREATE_ADVANTAGE | OVERCOME | DEFEND
  → processNPC<Type>()
Fallback: Attack with DefaultAttackSkillForConflict targeting player.
```

## ActionParser (`action_parser.go`)

`ActionParseRequest` → LLM (temp=0.3) → `ActionParseResponse` → `action.NewAction()`. Uses `parseActionType()` to handle LLM mistakes (e.g., skill name instead of action type).

## UI Interface (`ui.go`, `uicontract/`)

- `ReadInput() (string, bool, error)` — input + exit flag
- `Emit(event GameEvent)` — single output channel

Optional interfaces:
- `InvokePrompter` — synchronous invoke prompts (terminal)
- `MidFlowPrompter` — synchronous mid-flow input (e.g., consequence selection)
- `SceneInfoSetter` — wired by `GameManager.Run()` for meta-command access to scene/player data

## Where to Add New Features

| Feature | File(s) |
|---------|---------|
| Meta-command | `internal/ui/terminal/terminal.go` `handleSpecialCommands()` |
| Input classification type | `scene_manager.go` constants + `HandleInput()` switch |
| Action outcome effect | `conflict.go` `applyActionEffects()` |
| Conflict mechanic | `conflict.go` (method on `*SceneManager`) |
| LLM prompt/response | `internal/prompt/` (template + data struct + parser) |
| NPC action type | `npc.go` (`processNPC<Type>()` + switch) |
| Scene generation | `scenario_manager.go` `generateNextScene()` |
| Scenario lifecycle | `scenario_manager.go` Start/HandleInput loop |
| Milestone/recovery | `game_manager.go` or `scenario_manager.go` |
| UI display | `ui.go` interface + `internal/ui/terminal/` |

## Session Logging

```go
sm.sessionLogger.Log("event_type", map[string]any{...})   // SceneManager
sm.sessionLogger.Log("action_parse", parsedAction)          // structs directly
m.sessionLogger.Log("scene_generated", map[string]any{...}) // ScenarioManager
```

All events written to YAML transcript files for replay and evaluation.
````
