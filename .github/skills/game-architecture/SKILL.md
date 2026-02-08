---
name: game-architecture
description: Guide for modifying game flow, adding features to the game loop, or debugging scene/scenario progression. Use this when asked to work with the engine lifecycle, input pipeline, conflict orchestration, or NPC behavior.
---

# Game Architecture / Engine Lifecycle

This skill covers the orchestration layers in `internal/engine/` that drive the game loop. The engine sits between the `internal/core/` rules and the `internal/prompt/` LLM layer, coordinating player input, LLM calls, dice rolls, and UI feedback.

## Layer Overview

```
GameManager          ← owns scenario lifecycle, milestones, fate point refresh
  └─ ScenarioManager ← owns multi-scene loop, scene generation, summaries, NPC registry, recovery
       └─ SceneManager ← owns single-scene loop, input classification, dialog/action handling, conflict turns
            ├─ ActionParser   ← LLM-driven input → structured action
            ├─ AspectGenerator ← LLM-driven Create Advantage → aspect name
            └─ conflict.go    ← conflict initiation, turn order, damage, invokes, taken-out, concession
```

Each layer creates and configures the layer below it. **Do not skip layers** — e.g., don't call `SceneManager` methods from `GameManager` directly.

## Engine (`engine.go`)

The `Engine` struct is the shared kernel. It holds the LLM client, character registry, and creates the `SceneManager` and `ActionParser`.

```go
type Engine struct {
    actionParser      *ActionParser
    sceneManager      *SceneManager
    llmClient         llm.LLMClient
    characterRegistry map[string]*character.Character
}
```

Key methods:
- `NewWithLLM(client)` — constructor (creates `ActionParser` and `SceneManager`)
- `AddCharacter(char)` / `GetCharacter(id)` / `GetCharacterByName(name)` — character registry
- `ResolveCharacter(target)` — flexible lookup: tries ID, name, then `"Name (ID)"` format
- `GetCharactersByScene(scene)` — returns all characters in a scene
- `GetSceneManager()` — returns the `SceneManager` instance

## GameManager (`game_manager.go`)

Top-level orchestrator. Created by `cmd/cli/main.go`. Configures and runs one `ScenarioManager` per scenario.

**Responsibilities:**
- Wire up Engine + Player + UI + SessionLogger
- Run the scenario via `ScenarioManager.Run(ctx)`
- Handle milestones on scenario completion (fate point refresh, consequence recovery)
- Track `scenarioCount` for consequence recovery timing

```go
gm := engine.NewGameManager(eng)
gm.SetPlayer(player)
gm.SetUI(terminalUI)
gm.SetSessionLogger(logger)
gm.SetScenario(scenario)       // Optional: provide scenario or let ScenarioManager generate one
gm.Run(ctx)                     // Normal flow
gm.RunWithInitialScene(ctx, &engine.InitialSceneConfig{...})  // Demo/test flow
```

**Milestone handling** (on `ScenarioEndResolved`):
1. `player.RefreshFatePoints()` — reset to Refresh value
2. `player.CheckConsequenceRecovery(sceneCount, scenarioCount)` — clear healed consequences
3. Increment `scenarioCount`

## ScenarioManager (`scenario_manager.go`)

Owns the multi-scene loop within a single scenario. Created by `GameManager.Run()`.

**Responsibilities:**
- Generate or accept an initial scene
- Run scene → summarize → check resolution → generate next scene (loop)
- Maintain NPC registry across scenes (name-keyed persistence)
- Track scene summaries (sliding window)
- Handle between-scene consequence recovery
- Extract complications from unresolved threads

### Scenario loop

```
┌─────────────────────────────────────────────────┐
│ getInitialScene() or generateNextScene()        │
│     ↓                                           │
│ SceneManager.StartScene() + RunSceneLoop()      │
│     ↓                                           │
│ Scene ends → generateSceneSummary()            │
│     ↓                                           │
│ checkScenarioResolution() → resolved? → EXIT    │
│     ↓ (not resolved)                            │
│ handleBetweenSceneRecovery()                    │
│     ↓                                           │
│ generateNextScene(transitionHint) → LOOP        │
└─────────────────────────────────────────────────┘
```

**Scene end reasons** (`scene_flow.go`):
| Reason | Constant | Effect |
|--------|----------|--------|
| Player quit | `SceneEndQuit` | Scenario ends with `ScenarioEndQuit` |
| Player taken out | `SceneEndPlayerTakenOut` | Game over, or transition if hint provided |
| Scene transition | `SceneEndTransition` | Generate summary → check resolution → next scene |

**Scenario end reasons:**
| Reason | Constant | Trigger |
|--------|----------|---------|
| Resolved | `ScenarioEndResolved` | `checkScenarioResolution()` returns true |
| Quit | `ScenarioEndQuit` | Player chose to quit |
| Player taken out | `ScenarioEndPlayerTakenOut` | No transition hint after taken out |

### NPC registry

Named (non-nameless) NPCs persist across scenes via `npcRegistry map[string]*character.Character` keyed by `normalizeNPCName()` (lowercase, trimmed). When generating a new scene, if an NPC name matches a registry entry, the existing `*character.Character` is reused rather than creating a duplicate.

`npcAttitudes map[string]string` tracks last-known attitude per NPC, updated from scene summaries.

### Scene generation pipeline

```go
SceneGenerationData{
    TransitionHint,         // From previous scene's [SCENE_TRANSITION:hint] marker
    Scenario,               // Problem, story questions, setting, genre
    PlayerName/HighConcept/Trouble/Aspects,
    PreviousSummaries,      // Last 3 scene summaries (sliding window)
    Complications,          // Unresolved threads from summaries
    KnownNPCs,              // NPCSummary{Name, Attitude} from registry
}
→ prompt.RenderSceneGeneration() → LLM → prompt.ParseGeneratedScene()
→ GeneratedScene{SceneName, Description, Purpose, OpeningHook, SituationAspects, NPCs}
→ scene.NewScene() + register NPCs
```

### Scene summary pipeline

After each scene ends:
```go
SceneSummaryData{SceneName, Description, SituationAspects, ConversationHistory, NPCsInScene, TakenOutChars, HowEnded, TransitionHint}
→ prompt.RenderSceneSummary() → LLM → prompt.ParseSceneSummary()
→ SceneSummary{Summary, KeyEvents, NPCsEncountered, UnresolvedThreads}
```

### Scenario resolution check

After each summary, if the scenario has story questions:
```go
ScenarioResolutionData{Scenario, SceneSummaries, LatestSummary, PlayerName, PlayerAspects}
→ prompt.RenderScenarioResolution() → LLM → prompt.ParseScenarioResolution()
→ ScenarioResolutionResult{IsResolved, AnsweredQuestions, Reasoning}
```

### Between-scene recovery

`handleBetweenSceneRecovery()`:
1. Check if already-recovering consequences have healed (`player.CheckConsequenceRecovery`)
2. For non-recovering consequences, roll 4dF + best recovery skill vs difficulty (consequence value + 2 for self-treatment)
3. On success, start recovery (`player.BeginConsequenceRecovery`)
4. Generate LLM narrative for recovery results (may rename consequence aspects)

## SceneManager (`scene_manager.go`)

Owns a single scene's interactive loop. Created fresh by `ScenarioManager` for each scene.

**Responsibilities:**
- Display scene, read input, classify input, dispatch to handler
- Maintain conversation history (last 10 exchanges, sliding window)
- Track scene end state (reason, transition hint, taken-out characters)

### Input pipeline

```
Player types input
    ↓
handleMetaCommand?  →  yes → handle locally (help/scene/character/status)
    ↓ no
isConcedeCommand? (if in conflict)  →  yes → handleConcession()
    ↓ no
classifyInput() via LLM → "dialog" | "clarification" | "narrative" | "action"
    ↓
┌──────────────────────────┬──────────────────────────┐
│ dialog/clarification/    │ action                   │
│ narrative                │                          │
│    ↓                     │    ↓                     │
│ handleDialog()           │ handleAction()           │
│    ↓                     │    ↓                     │
│ generateSceneResponse()  │ ActionParser.ParseAction()│
│    ↓                     │    ↓                     │
│ Parse markers:           │ resolveAction()          │
│ • [CONFLICT_START:]      │    ↓                     │
│ • [CONFLICT_END:]        │ Roll dice, apply effects │
│ • [SCENE_TRANSITION:]    │    ↓                     │
│    ↓                     │ generateActionNarrative() │
│ Display response         │    ↓                     │
│                          │ Display result + narrative│
└──────────────────────────┴──────────────────────────┘
```

**Input classification types:**
| Type | Meaning | Handler |
|------|---------|---------|
| `dialog` | Speaking to NPCs, asking questions | `handleDialog()` |
| `clarification` | Asking about the scene/rules | `handleDialog()` |
| `narrative` | Describing character actions narratively | `handleDialog()` |
| `action` | Attempting a Fate Core mechanical action | `handleAction()` |

### Dialog flow

`handleDialog()` → `generateSceneResponse()` which:
1. Builds character/aspects/conversation context strings
2. Chooses template: `prompt.RenderConflictResponse()` (if in conflict) or `prompt.RenderSceneResponse()` (normal)
3. Sends to LLM, gets narrative response
4. Parses markers from response: `[CONFLICT_START:type|initiator]`, `[CONFLICT_END:reason]`, `[SCENE_TRANSITION:hint]`
5. Displays cleaned response, then handles any triggered markers

### Action flow

`handleAction()`:
1. `ActionParser.ParseAction()` — LLM interprets raw input as structured `action.Action` (type, skill, target, difficulty)
2. `resolveAction()` — auto-initiates conflict if Attack and not in conflict; rolls dice; handles invocations
3. `applyActionEffects()` — applies stress, consequences, aspects based on outcome
4. `generateActionNarrative()` — LLM generates narrative for the mechanical result

## ActionParser (`action_parser.go`)

LLM-driven parser that converts free-text player input into a structured `action.Action`.

```go
ActionParseRequest{Character, RawInput, Context, Scene, OtherCharacters}
→ system prompt (template) + user prompt (template) → LLM (temp=0.3)
→ ActionParseResponse{ActionType, Skill, Description, Target, Difficulty, Reasoning, Confidence}
→ action.NewAction() or action.NewActionWithTarget()
```

The parser uses `parseActionType()` which handles common LLM mistakes (e.g., returning a skill name instead of an action type).

## Conflict System (`conflict.go`)

All conflict logic is methods on `SceneManager`. Conflicts are triggered by LLM markers (`[CONFLICT_START:]`) or by player Attack actions.

### Conflict lifecycle

```
Trigger: [CONFLICT_START:physical|npc-id] marker or player Attack action
    ↓
initiateConflict(type, initiator)
    ↓
  - Build participants from scene characters (skip taken-out)
  - Calculate initiative (core.CalculateInitiative)
  - scene.StartConflictWithInitiator() — initiator goes first
    ↓
Main loop: player acts → advanceConflictTurns() processes NPC turns → back to player
    ↓
End conditions:
  - All opponents taken out → [CONFLICT_END:] marker from LLM
  - Player concedes → handleConcession()
  - Peaceful resolution → [CONFLICT_END:reason] marker
  - Player taken out → handleTargetTakenOut() → may end scene
```

### Turn processing

`advanceConflictTurns()`:
- After player acts, advance past player's turn slot
- Process each NPC turn via `processNPCTurn()` until back to player
- `NextTurn()` advances the initiative order, incrementing round when wrapping

### Player action resolution (`resolveAction`)

1. If Attack and no conflict → auto-initiate; if type mismatch → escalate
2. Roll 4dF + skill + invoke bonuses
3. For attacks: roll target defense (with full defense +2 bonus if applicable)
4. `handlePostRollInvokes()` — prompt player to invoke aspects for +2/reroll after seeing initial result
5. Compare final values → `Outcome`
6. `applyActionEffects()` — based on action type and outcome:
   - **Attack success**: `applyDamageToTarget()` → stress → consequences → taken out
   - **Create Advantage success**: `generateAspectName()` via `AspectGenerator` → add situation aspect
   - **Overcome/Defend**: narrative only (for now)

### Post-roll invocation (`handlePostRollInvokes`)

After rolling, the player can invoke aspects to improve their result:
1. `gatherInvokableAspects()` — collects character aspects, situation aspects (with free invokes), consequence aspects
2. `ui.PromptForInvoke()` — shows available aspects, fate points, current result, shifts needed
3. Apply +2 bonus or reroll; spend fate point or use free invoke
4. Loop until player declines or no aspects/points remain

### Damage resolution

`applyDamageToTarget(target, shifts, stressType)`:
1. Try `target.TakeStress(type, shifts)` — checks box ≥ shifts
2. If stress absorbed → done
3. `handleTargetStressOverflow()` — offer consequence slots that can absorb remaining shifts
4. If no absorption possible → `handleTargetTakenOut(target)`

`handleTargetTakenOut(target)`:
- If NPC: mark taken out, set participant status, check if conflict should end
- If player: use LLM to classify outcome (`TakenOutContinue`, `TakenOutTransition`, `TakenOutGameOver`), set scene end reason

### Concession

`handleConcession()`:
1. Must be called **before** a roll (checked by input pipeline position)
2. Award fate points: 1 + number of consequences taken in conflict
3. Set participant status to `StatusConceded`
4. Generate LLM narrative for the concession
5. End conflict, clear stress

### Conflict escalation

`handleConflictEscalation(newType)` — changes conflict type (e.g., mental → physical) and recalculates initiative order using new type's skills.

## AspectGenerator (`aspect_generator.go`)

Used during Create Advantage actions to generate thematic aspect names via LLM.

```go
AspectGenerationRequest{Action, Outcome, SceneContext, OtherCharacters}
→ LLM → AspectGenerationResponse{AspectText, Description, Duration, FreeInvokes, IsBoost}
```

- On Tie → generates a **boost** (one free invoke, disappears after use)
- On Success → full aspect with 1 free invoke
- On Success with Style → full aspect with 2 free invokes

## NPC Turn Pipeline (`npc.go`)

During conflict, each NPC turn follows this pipeline:

```
processNPCTurn(npcID)
    ↓
getNPCActionDecision() via LLM
    ↓
NPCActionDecisionData{ConflictType, Round, Scene, NPC stats/aspects/stress, Targets, SituationAspects}
→ prompt.RenderNPCActionDecision() → LLM (temp=0.7)
→ NPCActionDecision{ActionType, Skill, TargetID, Description}
    ↓
Switch on ActionType:
  ATTACK           → processNPCAttack()    → roll attack vs defense, invoke, damage, LLM narrative
  CREATE_ADVANTAGE → processNPCCreateAdvantage() → roll vs Fair(+2), create situation aspect
  OVERCOME         → processNPCOvercome()  → roll vs Fair(+2), narrative
  DEFEND           → processNPCDefend()    → set full defense (+2 to all defense rolls)
```

Fallback: if LLM decision fails, NPC defaults to Attack with `DefaultAttackSkillForConflict(type)` targeting the player.

## UI Interface (`ui.go`)

The `UI` interface decouples engine from presentation. The terminal implementation lives in `internal/ui/terminal/`.

Key interface groups:
- **Input**: `ReadInput() (string, bool, error)` — returns input and exit flag
- **Display**: `DisplayNarrative()`, `DisplayDialog()`, `DisplaySystemMessage()`, `DisplayActionResult()`
- **Conflict**: `DisplayConflictStart()`, `DisplayTurnAnnouncement()`, `DisplayConflictEnd()`
- **Invocation**: `PromptForInvoke(available, fatePoints, currentResult, shiftsNeeded) *InvokeChoice`
- **Flow**: `DisplayGameOver()`, `DisplaySceneTransition()`, `DisplayCharacter()`

Optional: `SceneInfoSetter` — if the UI implements this, `SceneManager.SetUI()` calls `SetSceneInfo(sm)` to provide access to scene/player/conversation data.

## Where to Add New Features

| Feature type | Layer | File(s) |
|-------------|-------|---------|
| New meta-command (help, status) | SceneManager | `scene_manager.go` `handleMetaCommand()` |
| New input classification type | SceneManager | `scene_manager.go` constants + `processInput()` switch |
| New action outcome effect | SceneManager | `conflict.go` `applyActionEffects()` |
| New conflict mechanic | SceneManager | `conflict.go` (add method on `*SceneManager`) |
| New LLM prompt/response | prompt package | `internal/prompt/` (template + data struct + parser) |
| New NPC action type | SceneManager | `npc.go` (add `processNPC<Type>()` + update switch) |
| Scene generation changes | ScenarioManager | `scenario_manager.go` `generateNextScene()` |
| New scenario lifecycle event | ScenarioManager | `scenario_manager.go` Run() loop |
| New milestone/recovery logic | GameManager or ScenarioManager | `game_manager.go` or `scenario_manager.go` |
| New UI display method | UI interface + terminal | `ui.go` interface + `internal/ui/terminal/` |

## Session Logging

Add logging at the appropriate layer:

```go
sm.sessionLogger.Log("event_type", map[string]any{...})   // SceneManager
sm.sessionLogger.Log("action_parse", parsedAction)          // Log structs directly
m.sessionLogger.Log("scene_generated", map[string]any{...}) // ScenarioManager
```

All session events are written to YAML transcript files for replay and evaluation.
