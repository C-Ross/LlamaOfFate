# Web UI Implementation Plan

**Epic:** [#5 — Add web interface for browser-based gameplay](https://github.com/C-Ross/LlamaOfFate/issues/5)
**Branch:** `feature/web-ui`

## Codebase Assessment

The codebase is **fully ready** for a web UI. Key findings:

| Area | Status |
|------|--------|
| Event-driven engine API | Complete — `Start()` + `HandleInput()` return `*InputResult` with `[]GameEvent` |
| UI coupling | Zero — engine never imports `ui/terminal`; all output via `uicontract.Emit()` |
| Invoke/MidFlow protocol | Async-ready — `AwaitingInvoke`/`AwaitingMidFlow` flags + dedicated response methods |
| Direct `renderEvents` calls | Eliminated — only used in the blocking `GameManager.Run()` terminal path |
| Persistence | Working — `GameStateSaver` with YAML implementation; web UI inherits it for free |
| Dependencies | Minimal — no HTTP/WS libraries yet; clean surface for adding `nhooyr.io/websocket` |

No engine changes are required. The async `Start(ctx)` / `HandleInput(ctx, input)` API was explicitly designed for this.

## Sub-Issues & Sequencing

```
#73 Go WebSocket backend          ──┐
                                    ├── #75 WebSocket hook + ChatPanel ──┐
#74 React + Vite scaffold ─────────┘                                    ├── #76 Game sidebar
                                                                        ├── #77 Dice visualization
                                                                        └──── #78 Conflict UI (depends on 76, 77)
```

**Critical path:** #73 → #74 → #75 → #78

## Phase 1: Go WebSocket Backend (#73)

**Goal:** Expose the existing engine API over WebSocket so the game is playable via `websocat`.

### Tasks

1. **Add `nhooyr.io/websocket` dependency**
   ```bash
   go get nhooyr.io/websocket
   ```

2. **Create `internal/ui/web/messages.go`**
   - `MarshalEvent(GameEvent) ([]byte, error)` — type-switch on every `GameEvent` concrete type, emit JSON `{"event": "<snake_case_type>", "data": {...}}`
   - `ResultMeta` struct with `AwaitingInvoke`, `AwaitingMidFlow`, `GameOver`, `SceneEnded` — sent after each `InputResult`
   - `ClientMessage` + `ParseClientMessage([]byte)` — parse the 3 client message types: `input`, `invoke_response`, `mid_flow_response`
   - Unit tests for round-trip marshaling/parsing

3. **Create `internal/ui/web/web.go`**
   - `WebUI` struct implementing `uicontract.UI`
   - `Emit(GameEvent)` pushes events into a channel (buffered)
   - `ReadInput()` blocks on a separate input channel — web handler writes into it
   - No `InvokePrompter`/`MidFlowPrompter` (uses async event path)
   - Compile-time interface check: `var _ uicontract.UI = (*WebUI)(nil)`

4. **Create `internal/ui/web/session.go`**
   - `Session` manages one WebSocket ↔ one `GameManager`
   - On connect: creates `WebUI`, wires `GameManager`, calls `Start(ctx)`, streams initial events
   - Read loop: parse `ClientMessage` → dispatch to `HandleInput` / `ProvideInvokeResponse` / `ProvideMidFlowResponse`
   - Write loop: drain event channel → `MarshalEvent` → WebSocket write
   - Sends `result_meta` after each `InputResult` response
   - Graceful shutdown on disconnect

5. **Create `internal/ui/web/handlers.go`**
   - `NewHandler(engineFactory)` — returns `http.Handler`
   - `GET /ws` — WebSocket upgrade → `Session`
   - `GET /` — serves static files (placeholder for now)

6. **Create `cmd/server/main.go`**
   - Parse flags: `--port`, `--llm-config`
   - Create LLM client, engine, handler
   - `http.ListenAndServe` with graceful shutdown via `signal.NotifyContext`

7. **Justfile: add `just serve`**

8. **Unit tests:** Marshaling round-trip, client message parsing, session lifecycle (mock WebSocket)

### Key Design Decisions
- `WebUI.Emit()` writes to a buffered channel; the write goroutine drains it. This decouples engine speed from WebSocket write backpressure.
- `ReadInput()` blocks on a channel — the session read goroutine pushes parsed input messages into it. This satisfies the `UI` interface contract without polling.
- One `GameManager` per WebSocket connection. No shared state between sessions.

## Phase 2: React + Vite + Tailwind Scaffold (#74)

**Goal:** Working frontend shell with dark fantasy theme, shadcn/ui components, zero game logic.

### Tasks

1. **Scaffold Vite project** in `web/`
   ```bash
   npm create vite@latest web -- --template react-ts
   ```

2. **Install dependencies:** Tailwind CSS v4, shadcn/ui, framer-motion, Cinzel + Crimson Pro fonts

3. **Configure theme:** Dark fantasy palette (near-black purple bg, amber/gold primary, burning orange accent)

4. **Initialize shadcn/ui components:** Button, Card, ScrollArea, Input, Badge

5. **Create placeholder `App.tsx`** with two-panel layout (chat left, sidebar right)

6. **Wire Vite proxy** to Go backend (`/ws` → `localhost:8080`)

7. **Justfile targets:** `web-install`, `web-dev`, `web-build`

8. **`.gitignore`** — add `web/node_modules/`, `web/dist/`

## Phase 3: WebSocket Hook + ChatPanel (#75)

**Goal:** Functional game loop in the browser — connect, send input, see narrative/dialog.

### Tasks

1. **`lib/types.ts`** — TypeScript types mirroring all `GameEvent` structs

2. **`hooks/useGameSocket.ts`** — custom hook:
   - Manages WebSocket lifecycle (connect, disconnect, reconnect)
   - Event accumulation via `useReducer`
   - `sendInput()`, `sendInvokeResponse()`, `sendMidFlowResponse()`
   - Tracks `result_meta` state (awaiting flags, game over)

3. **`components/game/ChatPanel.tsx`** — renders event stream as chat bubbles
   - GM messages left, player messages right
   - Auto-scroll on new events
   - Filter to displayable event types

4. **`components/game/ChatMessage.tsx`** — single event rendered by type
   - Narrative → styled prose block
   - Dialog → GM/player bubble pair
   - System → muted info line
   - Scene transition → separator with hint

5. **`components/game/ChatInput.tsx`** — text input + send
   - Disabled when awaiting invoke/mid-flow or pending
   - Enter to send

## Phase 4: Game Sidebar (#76)

**Goal:** Character panel, situation aspects, stress/consequences, NPCs.

### Tasks

1. **State extraction reducer** — derive game state from event stream
2. **`GameSidebar.tsx`** — collapsible card sections
3. **`AspectBadge.tsx`** — color-coded by type
4. **`StressTrack.tsx`** — physical + mental boxes, consequence slots
5. **`FatePointTracker.tsx`** — current count
6. **`NpcPanel.tsx`** — collapsible NPC list

**Open question:** May need a `GameStateSnapshot` event from backend on WebSocket connect to initialize sidebar state. Evaluate during Phase 3.

## Phase 5: Dice Visualization (#77)

**Goal:** Animated Fate dice with proper +/-/blank faces, skill totals, outcome badges.

### Tasks

1. **`FateDie.tsx`** — single die face with framer-motion animation
2. **`RollResult.tsx`** — 4 dice + skill + modifiers + total + outcome badge
3. **`ActionAttempt.tsx`** — pre-roll display
4. **Integrate into ChatPanel** as inline chat elements

## Phase 6: Conflict System (#78)

**Goal:** Full conflict UI — invoke prompts, damage resolution, NPC turns, concession.

### Tasks

1. **`ConflictBanner.tsx`** — persistent banner during conflicts
2. **`TurnAnnouncement.tsx`** — inline turn change markers
3. **`InvokePrompt.tsx`** — modal dialog for aspect invocation
4. **`MidFlowPrompt.tsx`** — modal for consequence selection / taken-out decisions
5. **`DamageResolution.tsx`** — inline damage/stress/consequence display
6. **`NPCAction.tsx`** — NPC attack results
7. **`ConflictEnd.tsx`** — conflict resolution summary

## Getting Started

Phase 1 (#73) is the immediate next step. It has zero frontend dependencies and produces a testable artifact (playable via `websocat`). Start with `messages.go` (pure serialization logic, easy to unit test) and build outward.

**Recommended first PR structure:**
1. `messages.go` + tests
2. `web.go` + `session.go` + tests
3. `handlers.go` + `cmd/server/main.go` + justfile
