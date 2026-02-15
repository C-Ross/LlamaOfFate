````skill
---
name: playwright-testing
description: Guide for using Playwright MCP tools to interact with and test the LlamaOfFate web UI in a headless browser. Use this when asked to test the web UI, verify visual behavior, or puppet the browser.
---

# Playwright Testing

This skill covers using the Playwright MCP tools (`browser_navigate`, `browser_snapshot`, `browser_click`, `browser_type`, `browser_press_key`, `browser_take_screenshot`, `browser_wait_for`, `browser_resize`, etc.) to test the LlamaOfFate web UI in a headless Chrome browser.

## Prerequisites

### Starting the Services

The web UI requires **two services** running:

1. **Go WebSocket server** (port 8080):
   ```bash
   go build -o ./bin/server ./cmd/server
   ./bin/server --port 8080 &
   ```
   Or use `just serve` if available.

2. **Vite dev server** (port 5173):
   ```bash
   cd web && npx vite --port 5173
   ```
   Or use `just web-dev`.

Run both as background processes before navigating with Playwright.

In dev mode the React app connects directly to `ws://localhost:8080/ws` (bypasses Vite proxy to avoid EPIPE on disconnect).

### Installing Playwright

If Playwright is not installed:
```bash
cd web && npm install -D @playwright/test
npx playwright install chrome
npx playwright install-deps chromium
```

## Tool Reference

### Navigation

```
browser_navigate(url="http://localhost:5173")
```

Always navigate to the Vite dev server URL, not the Go server.

### Snapshots (Accessibility Tree)

```
browser_snapshot()
```

Returns the page's accessibility tree as YAML. Each interactive element has a `ref` attribute for targeting. Use snapshots to:
- Verify page structure and content
- Get `ref` values for click/type targets
- Check element states (disabled, active, etc.)

### Taking Screenshots

```
browser_take_screenshot(type="png", fullPage=true)
```

- `type` is **required** (use `"png"`)
- `fullPage: true` captures the entire page; `false` captures the viewport only
- Returns a path to the saved image

### Clicking Elements

```
browser_click(element="descriptive label", ref="e16")
```

Use `ref` from the snapshot. The `element` parameter is a human description.

### Typing Text

```
browser_type(element="Player input textbox", ref="e16", text="I look around")
```

This replaces existing text (uses `fill()`). Use for the chat input.

### Pressing Keys

```
browser_press_key(key="Enter")
```

Use after typing to submit input. Common keys: `Enter`, `Escape`, `Tab`.

### Waiting for Content

```
browser_wait_for(textGone="Thinking...")
```

Use `textGone` to wait for an element to disappear (e.g., the "Thinking..." indicator after the LLM responds). Default timeout is 5 seconds — for LLM responses this may be too short; the server processes LLM calls which can take 10-30 seconds.

### Resizing the Viewport

```
browser_resize(width=1280, height=800)
```

Default viewport is phone-sized (~780px). Resize to 1280+ to see the desktop layout with the sidebar visible. The sidebar is hidden below `lg` breakpoint (1024px).

## Page Structure

### Layout (Desktop — width ≥ 1024px)

```
┌─────────────────────────────────────┬──────────────────┐
│ LLAMA OF FATE  [Connected]          │ Jesse Calhoun    │
│                                     │ [aspects]        │
│ ── SCENE NAME ──                    │                  │
│ [narrative blocks]                  │ Situation Aspects│
│ [player input → GM response]        │ [badges]         │
│ [action results, dice, combat]      │                  │
│                                     │ Fate Points      │
│                                     │ 3 / 3 refresh    │
│                                     │                  │
│                                     │ Stress & Conseq  │
│                                     │ Mental  [1][2]   │
│                                     │ Physical [1][2]  │
│                                     │                  │
│                                     │ NPCs             │
│                                     │ [collapsible]    │
├─────────────────────────────────────┤                  │
│ [What do you do?          ] [Send]  │                  │
└─────────────────────────────────────┴──────────────────┘
```

### Key Elements

| Element | Role / Selector | Notes |
|---|---|---|
| Header | `banner` with `heading "Llama of Fate"` | Contains connection badge, "Thinking..." indicator |
| Connection badge | Text "Connected" or "Not Connected" | Green when connected |
| Chat area | Main content area with event cards | Auto-scrolls to bottom |
| Input | `textbox "Player input"` | Placeholder changes contextually |
| Send button | `button "Send"` | Disabled when input is empty or pending |
| Sidebar (desktop) | `complementary` role | Only visible at lg+ width (≥1024px) |
| Sidebar (mobile) | `button "Open game sidebar"` → Sheet | Hamburger icon, opens overlay |

### Connection & State Indicators

| Indicator | Meaning |
|---|---|
| "Connected" (green badge) | WebSocket connected |
| "Not Connected" (muted badge) | WebSocket disconnected |
| "Thinking..." | LLM request in progress; input disabled |
| Input disabled, placeholder "Awaiting invoke response..." | `awaitingInvoke` flag set |
| Input disabled, placeholder "Awaiting your choice..." | `awaitingMidFlow` flag set |
| Input disabled, placeholder "Game over" | Game has ended |

## Testing Patterns

### Basic Interaction Flow

```
1. browser_navigate(url="http://localhost:5173")
2. browser_snapshot()                            — verify "Connected", scene loaded
3. browser_resize(width=1280, height=800)        — show sidebar
4. browser_click(ref=<input_ref>)                — focus input
5. browser_type(ref=<input_ref>, text="I look around")
6. browser_press_key(key="Enter")                — submit
7. browser_snapshot()                            — see "Thinking..."
8. browser_wait_for(textGone="Thinking...")       — wait for LLM
9. browser_snapshot()                            — verify GM response
10. browser_take_screenshot(type="png", fullPage=true)
```

### Verifying Combat

Send an aggressive action (e.g., "I punch Gus in the face") to trigger the Fate Core conflict system. After the LLM responds, the snapshot will include:

- `Conflict! (physical)` — conflict start card
- `Defense:` — defense roll message
- Skill name, outcome (Success/Failure/Tie/Success with Style)
- `HIT: X shifts` — attack result
- `Damage to [NPC]` with Taken Out / consequence info
- `Turn N: [Character]` — turn announcements
- NPC action results
- Sidebar changes: "NPCs" → "Combatants", taken-out NPCs marked

### Verifying the Sidebar

Resize to desktop width first:
```
browser_resize(width=1280, height=800)
browser_snapshot()
```

The `complementary` role element contains:
- Player name + character aspects (high concept, trouble)
- Situation aspects as badges
- Fate points (current / refresh)
- Stress tracks (mental, physical) with numbered boxes
- NPC list (collapsible with aspects)

### Mobile Sidebar

At default viewport (< 1024px), the sidebar is hidden. Test via:
```
browser_click(element="Open game sidebar", ref=<button_ref>)
browser_snapshot()  — Sheet overlay with sidebar content
```

### Waiting for LLM Responses

LLM calls can take 10-30 seconds. The pattern is:

1. Send input → see "Thinking..." appear
2. Use `browser_wait_for(textGone="Thinking...")` — waits for it to disappear
3. If it times out (default 5s), retry or take a snapshot to check current state
4. The input re-enables and placeholder returns to "What do you do?" when ready

### Console Errors

Expect these benign entries:
- **Warning**: `WebSocket connection to 'ws://localhost:...'` — appears briefly during initial connect
- **Error**: `Failed to load resource: favicon.ico 404` — no favicon configured yet

## Snapshot Tips

- Snapshots return `[ref=eNN]` on interactive elements — use these for click/type
- Text content appears inline in the YAML
- `[disabled]` and `[active]` states are shown
- `<changed>` markers appear when comparing consecutive snapshots
- For long pages, the snapshot includes all rendered content (not just viewport)

````
