# Challenges & Contests — Design Document

> Status: **Draft**
> Date: 2026-02-22

## 1. Fate Core Rules Summary

Fate Core defines three structured scene types: **conflicts**, **challenges**, and **contests**.
Conflicts are already implemented. This document designs challenges (primary focus) and
establishes a shared architecture that contests can reuse later.

### 1.1 Challenges ([SRD](https://fate-srd.com/fate-core/challenges))

A challenge is a series of **overcome actions** using **different skills** to resolve a
complex, multi-part situation. Each overcome addresses one facet of the problem.

**Rules:**
- The GM frames the challenge: what's happening, which skills are needed, how many
  overcome actions are required.
- Each overcome action is rolled independently against its own difficulty.
- Results are tallied at the end — partial success is possible (2 of 3 tasks succeed).
- Any participant can attempt any task; tasks can be distributed among characters.
- Standard overcome outcomes apply (fail, tie, succeed, succeed with style).
- Create Advantage actions can be taken instead of overcoming a task, producing aspects
  that help with subsequent tasks.

**Example:** *Escaping a Burning Building*
1. Athletics (Good) — dodge collapsing debris
2. Investigate (Fair) — find the fastest exit
3. Rapport (Average) — calm the panicking crowd

### 1.2 Contests ([SRD](https://fate-srd.com/fate-core/contests))

A contest is a series of **opposed exchanges** where two sides race toward a goal.

**Rules:**
- Each exchange: both sides make overcome rolls against each other.
- Winner of the exchange scores a **victory**.
- Ties produce no victory; both sides get a situation aspect with one free invoke.
- First side to **3 victories** wins.
- Create Advantage actions can be taken instead of overcome, forfeiting a chance at a
  victory that exchange but potentially setting up future exchanges.

---

## 2. Architecture Principles

### 2.1 Follow existing patterns

The conflict system establishes clear conventions:

| Component | Conflict equivalent | Location |
|-----------|-------------------|----------|
| Scene-level state | `ConflictState` | `internal/core/scene/scene.go` |
| Engine manager | `ConflictManager` | `internal/engine/conflict.go` + `conflict_manager.go` |
| LLM prompt template | `conflict_response_prompt.tmpl` | `internal/prompt/templates/` |
| Prompt data struct | `ConflictResponseData` | `internal/prompt/data.go` |
| Render function | `RenderConflictResponse()` | `internal/prompt/templates.go` |
| Marker parsing | `ParseConflictMarker()` | `internal/prompt/markers.go` |
| UI events | `ConflictStartEvent`, etc. | `internal/uicontract/events.go` |
| Type alias re-exports | `engine/ui.go` | `internal/engine/ui.go` |
| Scene-exit integration | `SceneManager.applySceneEnd()` | `internal/engine/scene_manager.go` |

Challenges and contests should mirror this decomposition.

### 2.2 ActionResolver reuse

`ActionResolver` already handles the dice rolling, invoke loops, mid-flow prompts,
narrative generation, and effect application pipeline. Challenges/contests should
delegate individual overcome rolls to `ActionResolver` rather than reimplementing the
dice pipeline.

### 2.3 LLM-driven framing

The LLM currently triggers conflicts via `[CONFLICT:type:id]` markers embedded in
narrative responses. Challenges should follow the same pattern: the LLM decides when a
situation is complex enough to warrant structured resolution, emits a marker, and the
engine takes over the mechanical side.

### 2.4 Persistence compatibility

`GameState.Scene.CurrentScene` already serializes `ConflictState` via YAML. New state
fields (`ChallengeState`, `ContestState`) must be similarly serializable and nullable
so that saves from before these features exist remain loadable.

---

## 3. Challenge Design

### 3.1 Core data model — `scene.ChallengeState`

```go
// internal/core/scene/scene.go

// ChallengeTask represents one overcome action within a challenge.
type ChallengeTask struct {
    ID          string        `json:"id" yaml:"id"`
    Description string        `json:"description" yaml:"description"`
    Skill       string        `json:"skill" yaml:"skill"`
    Difficulty  int           `json:"difficulty" yaml:"difficulty"` // Ladder value
    Status      TaskStatus    `json:"status" yaml:"status"`         // pending, succeeded, failed, tied
    ActorID     string        `json:"actor_id,omitempty" yaml:"actor_id,omitempty"` // Who attempted it
    Shifts      int           `json:"shifts,omitempty" yaml:"shifts,omitempty"`
}

type TaskStatus string

const (
    TaskPending   TaskStatus = "pending"
    TaskSucceeded TaskStatus = "succeeded"
    TaskFailed    TaskStatus = "failed"
    TaskTied      TaskStatus = "tied"
)

// ChallengeState tracks a multi-task challenge within a scene.
type ChallengeState struct {
    Description string          `json:"description" yaml:"description"` // What the overall challenge is
    Tasks       []ChallengeTask `json:"tasks" yaml:"tasks"`
    Resolved    bool            `json:"resolved" yaml:"resolved"`
}

// PendingTasks returns tasks not yet attempted.
func (cs *ChallengeState) PendingTasks() []ChallengeTask { ... }

// Tally returns (successes, failures, ties).
func (cs *ChallengeState) Tally() (int, int, int) { ... }

// IsComplete returns true when all tasks have been attempted.
func (cs *ChallengeState) IsComplete() bool { ... }

// OverallOutcome returns an overall result label based on the tally.
// "success" if majority succeed, "partial" if mixed, "failure" if majority fail.
func (cs *ChallengeState) OverallOutcome() string { ... }
```

**Scene struct additions:**

```go
type Scene struct {
    // ... existing fields ...
    IsChallenge    bool            `json:"is_challenge" yaml:"is_challenge,omitempty"`
    ChallengeState *ChallengeState `json:"challenge_state,omitempty" yaml:"challenge_state,omitempty"`
}
```

### 3.2 Engine — `ChallengeManager`

Located in `internal/engine/challenge.go` + `internal/engine/challenge_manager.go`,
following the conflict split:

```
challenge_manager.go  — struct definition, constructor, state wiring, accessors
challenge.go          — lifecycle methods (initiate, resolve task, complete)
```

**Struct:**

```go
type ChallengeManager struct {
    // Shared dependencies
    llmClient     llm.LLMClient
    characters    CharacterResolver
    sessionLogger *session.Logger

    // ActionResolver for dice rolling + invoke loops
    actions *ActionResolver

    // Per-scene state
    player       *character.Character
    currentScene *scene.Scene
}
```

**Key methods:**

| Method | Purpose |
|--------|---------|
| `initiateChallenge(desc string, tasks []scene.ChallengeTask)` | Sets `IsChallenge`, creates `ChallengeState` |
| `resolveTask(ctx, taskID string, action *action.Action) []GameEvent` | Delegates dice roll to `ActionResolver`, records outcome |
| `completeChallenge(ctx) []GameEvent` | Tallies results, generates summary narrative, clears state |
| `getTaskInfo() []ChallengeTaskInfo` | Build UI-friendly task list for events |

### 3.3 LLM marker — `[CHALLENGE:json]`

The LLM emits a challenge marker when it identifies a complex multi-task situation.
Unlike conflict markers (which are simple `type:id`), challenge markers carry structured
task data. JSON is appropriate here since it's an LLM structured response.

**Marker format:**

```
[CHALLENGE:{"description":"Escape the burning building","tasks":[{"skill":"Athletics","difficulty":3,"description":"Dodge collapsing debris"},{"skill":"Investigate","difficulty":2,"description":"Find the fastest exit"},{"skill":"Rapport","difficulty":1,"description":"Calm the panicking crowd"}]}]
```

**Parsing** (in `internal/prompt/markers.go`):

```go
type ChallengeTrigger struct {
    Description string
    Tasks       []ChallengeTriggerTask
}

type ChallengeTriggerTask struct {
    Skill       string `json:"skill"`
    Difficulty  int    `json:"difficulty"`
    Description string `json:"description"`
}

var challengeMarkerRegex = regexp.MustCompile(`\[CHALLENGE:(\{[^]]+\})\]`)

func ParseChallengeMarker(response string) (*ChallengeTrigger, string) { ... }
```

**Prompt template instructions** (added to `scene_response_prompt.tmpl` and new
`challenge_response_prompt.tmpl`):

```
CHALLENGE MARKERS:
If the player's action involves a complex situation requiring multiple different skills
to resolve (e.g., escaping a disaster, preparing for a heist, navigating a hazardous
journey), add on its own line at the end:
[CHALLENGE:{"description":"...","tasks":[{"skill":"...","difficulty":N,"description":"..."},...]}}]
- Include 2-5 tasks, each using a DIFFERENT skill
- Difficulty is a Fate ladder value: 0=Mediocre, 1=Average, 2=Fair, 3=Good, 4=Great
- Only trigger when MULTIPLE distinct skills are needed — simple overcome actions don't need this
```

### 3.4 UI events

```go
// internal/uicontract/events.go

// ChallengeStartEvent announces a new challenge.
type ChallengeStartEvent struct {
    Description string
    Tasks       []ChallengeTaskInfo
}

// ChallengeTaskInfo describes one task in a challenge (for UI display).
type ChallengeTaskInfo struct {
    ID          string
    Description string
    Skill       string
    Difficulty  string // Ladder name, e.g. "Good (+3)"
    Status      string // "pending", "succeeded", "failed", "tied"
}

// ChallengeTaskResultEvent announces the outcome of a single task.
type ChallengeTaskResultEvent struct {
    TaskID      string
    Description string
    Skill       string
    Outcome     string // "success", "failure", "tie", "success_with_style"
    Shifts      int
}

// ChallengeCompleteEvent announces the overall challenge resolution.
type ChallengeCompleteEvent struct {
    Successes int
    Failures  int
    Ties      int
    Overall   string // "success", "partial", "failure"
    Narrative string // LLM-generated summary
}
```

### 3.5 UI presentation

Challenges need visual treatment in both the terminal and web UIs. The conflict system
establishes the pattern: a **persistent banner** + **inline chat event cards** +
**game state snapshot fields**.

#### 3.5.1 Terminal UI

**Challenge start** — boxed banner (same frame as conflict, different label) followed
by a task checklist:

```
╔══════════════════════════════════════════╗
║           CHALLENGE BEGINS!              ║
║  Escape the Burning Building             ║
╚══════════════════════════════════════════╝

Tasks:
  [ ] 1. Dodge collapsing debris (Athletics vs Good)
  [ ] 2. Find the fastest exit (Investigate vs Fair)
  [ ] 3. Calm the panicking crowd (Rapport vs Average)
```

**Task result** — inline status line after each overcome roll resolves:

```
  [✓] 1. Dodge collapsing debris — Success (2 shifts)
  [✗] 2. Find the fastest exit — Failed
  [~] 3. Calm the panicking crowd — Tie (boost gained)
```

**Challenge complete** — tally banner + LLM narrative:

```
╔══════════════════════════════════════════╗
║         CHALLENGE COMPLETE               ║
║  2 of 3 tasks succeeded (partial)        ║
╚══════════════════════════════════════════╝

<LLM-generated narrative summarizing the partial success>
```

**Terminal methods** (following the `displayConflictStart` / `displayConflictEnd`
pattern):

| Method | Renders |
|--------|---------|
| `displayChallengeStart(description string, tasks []ChallengeTaskInfo)` | Banner + task list |
| `displayChallengeTaskResult(task ChallengeTaskResultEvent)` | Single task status line |
| `displayChallengeComplete(event ChallengeCompleteEvent)` | Tally banner |

#### 3.5.2 Web UI

**Persistent banner** — `ChallengeBanner` component, equivalent to `ConflictBanner`.
Uses amber/warning color scheme (vs. destructive red for conflicts) to visually
distinguish "structured tension" from "active combat":

```tsx
// components/game/ChallengeBanner.tsx
<div
  className="flex items-center justify-center gap-2 bg-warning/10
             border-b border-warning/30 px-4 py-2 text-xs font-heading
             uppercase tracking-widest text-warning"
  role="alert"
  aria-label="Challenge in progress"
>
  <span className="inline-block h-2 w-2 rounded-full bg-warning animate-pulse" />
  Challenge In Progress
  <span className="inline-block h-2 w-2 rounded-full bg-warning animate-pulse" />
</div>
```

Mounted in `App.tsx` alongside `ConflictBanner`:
```tsx
<ConflictBanner active={gameState.inConflict} />
<ChallengeBanner active={gameState.inChallenge} />
```

**Chat event cards** — three new renderers in `ChatMessage.tsx`:

1. **`ChallengeStartMessage`** — amber-bordered card with description + task checklist:

```tsx
<div className="my-3 rounded-lg border border-warning/50 bg-warning/5 px-4 py-3 space-y-2">
  <div className="font-heading text-sm font-bold uppercase tracking-wide text-warning">
    Challenge: {data.Description}
  </div>
  {data.Tasks.map(task => (
    <div key={task.ID} className="text-xs font-body text-muted-foreground">
      ○ {task.Description} ({task.Skill} vs {task.Difficulty})
    </div>
  ))}
</div>
```

2. **`ChallengeTaskResultMessage`** — color-coded inline card per task outcome:
   - ✓ Success / Success with Style → green border (`border-boost/30`)
   - ✗ Failure → red border (`border-destructive/30`)
   - ~ Tie → muted border (boost gained)

3. **`ChallengeCompleteMessage`** — full-width centered card (like `MilestoneMessage`):
   - **Success** (all/most pass): green border, `text-boost`
   - **Partial** (mixed): amber border, `text-warning`
   - **Failure** (all/most fail): red border, `text-destructive`
   - Includes LLM-generated summary narrative

**Wire protocol event names:**

| Event | Wire name | Go type |
|-------|-----------|---------|
| Challenge begins | `challenge_start` | `ChallengeStartEvent` |
| Single task result | `challenge_task_result` | `ChallengeTaskResultEvent` |
| Challenge complete | `challenge_complete` | `ChallengeCompleteEvent` |

**TypeScript types** (in `lib/types.ts`):

```typescript
export interface ChallengeStartEventData {
  Description: string
  Tasks: ChallengeTaskInfo[]
}

export interface ChallengeTaskInfo {
  ID: string
  Description: string
  Skill: string
  Difficulty: string
  Status: string
}

export interface ChallengeTaskResultEventData {
  TaskID: string
  Description: string
  Skill: string
  Outcome: string
  Shifts: number
}

export interface ChallengeCompleteEventData {
  Successes: number
  Failures: number
  Ties: number
  Overall: string
  Narrative: string
}
```

**Game state snapshot** — `GameStateSnapshotEventData` gains:

```typescript
export interface GameStateSnapshotEventData {
  // ... existing fields ...
  inConflict: boolean
  inChallenge: boolean
  challengeTasks?: ChallengeTaskInfo[]
}
```

#### 3.5.3 Key UI decisions

- **No turn structure** — Unlike conflicts, challenges have no initiative or turns.
  The player acts naturally and the engine matches their skill to a pending task.
  No `TurnAnnouncementEvent` equivalent is needed.
- **Banner dismissal** — `ChallengeBanner` disappears when `inChallenge` goes false
  (same lifecycle as `ConflictBanner`). The `ChallengeCompleteMessage` card persists
  in chat history.
- **Task identification** — When the player acts during a challenge, the engine
  matches the parsed action's skill to a pending task. If ambiguous (e.g., two tasks
  could match), the engine emits an `InputRequestEvent` via the existing mid-flow
  prompt infrastructure rather than inventing a new interaction pattern.

### 3.6 SceneManager integration

**HandleInput flow changes:**

```
HandleInput(input)
  ├─ classifyInput → "action"
  │   └─ handleAction
  │       ├─ if IsChallenge → challenge.handleChallengeAction(ctx, input)
  │       │   ├─ identify which task the player is attempting
  │       │   ├─ delegate dice roll to ActionResolver
  │       │   ├─ record result on ChallengeState
  │       │   ├─ if all tasks done → completeChallenge()
  │       │   └─ return events
  │       ├─ if IsConflict → existing conflict path
  │       └─ else → normal action path (may trigger challenge via marker)
  │
  ├─ classifyInput → "dialog"/"narrative"
  │   └─ handleDialog
  │       ├─ parse markers (conflict, challenge, scene transition)
  │       ├─ if challengeTrigger → initiateChallenge()
  │       └─ if IsChallenge → use challenge_response_prompt.tmpl
```

The main branching point in `handleAction` mirrors the existing `IsConflict` check.
Similarly, `generateSceneResponse` should use a challenge-specific prompt template when
`IsChallenge` is true, just as it already does for conflicts.

### 3.7 Persistence

`SceneState.CurrentScene` already serializes `*scene.Scene` to YAML. Adding nullable
`ChallengeState *ChallengeState` to `Scene` is automatically handled — old saves will
deserialize with `nil` state, and `InitDefaults()` doesn't need to allocate it.

`GameState.Validate()` needs no changes since challenges are optional scene state.

### 3.8 Session logging

Following the existing pattern in `scene_manager.go`:

```go
sm.sessionLogger.Log("challenge_start", map[string]any{
    "description": challenge.Description,
    "tasks":       challenge.Tasks,
})

sm.sessionLogger.Log("challenge_task_result", map[string]any{
    "task_id":   task.ID,
    "skill":     task.Skill,
    "outcome":   task.Status,
    "shifts":    task.Shifts,
})

sm.sessionLogger.Log("challenge_complete", map[string]any{
    "successes": successes,
    "failures":  failures,
    "ties":      ties,
    "overall":   overall,
})
```

---

## 4. Contest Design (Future — structural considerations)

Contests follow a different pattern than challenges but share the same architectural
slots. Documenting the design now ensures we don't paint ourselves into a corner.

### 4.1 Data model — `scene.ContestState`

```go
type ContestState struct {
    Description    string          `json:"description" yaml:"description"`
    PlayerGoal     string          `json:"player_goal" yaml:"player_goal"`
    OpponentGoal   string          `json:"opponent_goal" yaml:"opponent_goal"`
    OpponentID     string          `json:"opponent_id" yaml:"opponent_id"`
    Exchanges      []ContestExchange `json:"exchanges" yaml:"exchanges"`
    PlayerVictories   int          `json:"player_victories" yaml:"player_victories"`
    OpponentVictories int          `json:"opponent_victories" yaml:"opponent_victories"`
    VictoriesNeeded   int          `json:"victories_needed" yaml:"victories_needed"` // Default 3
    Resolved       bool            `json:"resolved" yaml:"resolved"`
}

type ContestExchange struct {
    Round         int    `json:"round" yaml:"round"`
    PlayerSkill   string `json:"player_skill" yaml:"player_skill"`
    OpponentSkill string `json:"opponent_skill" yaml:"opponent_skill"`
    Winner        string `json:"winner" yaml:"winner"` // "player", "opponent", "tie"
}
```

**Scene struct:**

```go
IsContest    bool          `json:"is_contest" yaml:"is_contest,omitempty"`
ContestState *ContestState `json:"contest_state,omitempty" yaml:"contest_state,omitempty"`
```

### 4.2 Key differences from challenges

| Aspect | Challenge | Contest |
|--------|-----------|---------|
| Opposition | Passive difficulties | Active opposed rolls (NPC rolls too) |
| Structure | Fixed task list, any order | Sequential exchanges, first to N wins |
| NPC involvement | None (tasks are vs environment) | NPC rolls each exchange |
| Victory condition | Tally at end | Running score, stops at threshold |
| Action types | Overcome only | Overcome (may substitute Create Advantage) |

### 4.3 Shared infrastructure

Both challenges and contests need:
- A marker parser in `markers.go`
- A data struct in `data.go` + render function in `templates.go` + `.tmpl` file
- A manager struct in `engine/` wired to `ActionResolver` and `SceneManager`
- UI events in `uicontract/events.go` + aliases in `engine/ui.go`
- Session logging calls

The patterns are identical; only the game logic differs.

### 4.4 Mutual exclusion

Per Fate Core, a scene uses exactly *one* structured type at a time. The `Scene` struct
should enforce this:

```go
func (s *Scene) ActiveStructuredType() string {
    if s.IsConflict { return "conflict" }
    if s.IsChallenge { return "challenge" }
    if s.IsContest { return "contest" }
    return "none"
}

func (s *Scene) StartChallenge(...) error {
    if s.ActiveStructuredType() != "none" {
        return fmt.Errorf("cannot start challenge: scene already has active %s", s.ActiveStructuredType())
    }
    // ...
}
```

This also means `handleAction` branches are mutually exclusive — a natural if/else-if
chain rather than complex state combinations.

---

## 5. Test Strategy

### 5.1 Unit tests

| Package | Tests |
|---------|-------|
| `scene` | `ChallengeState` methods: `PendingTasks`, `Tally`, `IsComplete`, `OverallOutcome`, `StartChallenge` mutual exclusion |
| `prompt` | `ParseChallengeMarker` with valid/invalid/missing JSON, round-trip rendering of `ChallengeResponseData` |
| `engine` | `ChallengeManager.initiateChallenge`, `resolveTask` (seeded roller), `completeChallenge`, integration with `ActionResolver` invoke loop |
| `uicontract` | Event type assertions (compile-time via `gameEvent()` marker method) |

### 5.2 Integration tests

- Full `HandleInput` → challenge initiation via marker → task resolution → completion
- Challenge during save/load round-trip (persistence)
- Mutual exclusion: attempt to start challenge during conflict → error

### 5.3 LLM evaluation tests (future)

Tag: `-tags=llmeval`

- LLM correctly emits `[CHALLENGE:...]` marker for multi-skill situations
- LLM does *not* emit challenge marker for simple single-skill overcome actions
- Challenge task descriptions are coherent and match the narrative context

---

## 6. Implementation Plan

Ordered by dependency; each step should pass `just validate` independently.

| Step | Description | Files touched |
|------|-------------|---------------|
| 1 | Add `ChallengeState`, `ChallengeTask`, `TaskStatus` to scene package with methods + tests | `scene/scene.go`, `scene/challenge.go` (new), `scene/challenge_test.go` (new) |
| 2 | Add mutual exclusion helper `ActiveStructuredType()` + guard in `StartChallenge` | `scene/scene.go`, `scene/scene_test.go` |
| 3 | Add `ParseChallengeMarker()` + tests | `prompt/markers.go`, `prompt/markers_test.go` |
| 4 | Add `ChallengeResponseData`, render function, prompt template | `prompt/data.go`, `prompt/templates.go`, `prompt/templates/challenge_response_prompt.tmpl` (new) |
| 5 | Add challenge UI events + aliases | `uicontract/events.go`, `engine/ui.go` |
| 6 | Add `ChallengeManager` struct + lifecycle methods | `engine/challenge_manager.go` (new), `engine/challenge.go` (new) |
| 7 | Wire `ChallengeManager` into `SceneManager` (constructor, `StartScene`, `resetSceneState`, `handleDialog`, `handleAction`, `generateSceneResponse`) | `engine/scene_manager.go` |
| 8 | Add challenge marker instructions to `scene_response_prompt.tmpl` | `prompt/templates/scene_response_prompt.tmpl` |
| 9 | Session logging for challenge events | `engine/challenge.go` |
| 10 | Terminal UI: `displayChallengeStart`, `displayChallengeTaskResult`, `displayChallengeComplete` + emit cases | `ui/terminal/terminal.go` |
| 11 | Web UI: `ChallengeBanner` component, `ChatMessage` renderers, TS types, wire protocol, game state snapshot | `web/src/components/game/ChallengeBanner.tsx` (new), `web/src/components/game/ChatMessage.tsx`, `web/src/lib/types.ts`, `web/src/App.tsx`, `ui/web/messages.go` |
| 12 | Integration tests (HandleInput → challenge flow) | `engine/scene_manager_test.go` or `engine/challenge_test.go` |
| 13 | Persistence round-trip test | `engine/persistence_test.go` |

---

## 7. Open Questions

1. **Task ordering**: Should the player pick which task to attempt, or should they
   describe an action and the engine match it to the most appropriate pending task?
   *Decision:* Skill-match with mid-flow disambiguation. The engine matches the
   action parser's identified skill to a pending task. If ambiguous, use a mid-flow
   prompt — but prefer to avoid breaking the flow when possible (e.g., if only one
   task uses the matched skill, resolve it silently).

2. **Create Advantage during challenges**: Fate Core allows CA as an alternative to
   overcome. Should the player be able to Create Advantage during a challenge to
   add situation aspects that help later tasks?
   *Decision:* Yes — the action parser already distinguishes CA from Overcome.
   If the player does CA during a challenge, it doesn't resolve a task but creates
   an aspect normally. This keeps the system honest to the rules.

3. **NPC tasks**: In some challenges, an NPC ally might handle one task while the
   player handles others. For now, the player handles all tasks. NPC participation
   can be added later with the NPC action system.
   *Decision:* Player only for now. Defer NPC task participation to the NPC action
   system.

4. **Challenge failure consequences**: Fate Core leaves the consequences of challenge
   failure to GM narration. The LLM should generate appropriate narrative for partial
   and full failure, potentially triggering scene transitions or conflict escalation.
   *Decision:* LLM-narrated with optional escalation markers. The challenge completion
   prompt describes the outcome and the LLM generates narrative. For failures, the LLM
   may include `[SCENE_TRANSITION:...]` or `[CONFLICT:...]` markers if appropriate.

---

*This work is based on Fate Core System by Evil Hat Productions, LLC, licensed under
[CC BY 3.0](https://creativecommons.org/licenses/by/3.0/).*
