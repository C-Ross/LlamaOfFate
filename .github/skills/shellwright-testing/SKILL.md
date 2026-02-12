---
name: shellwright-testing
description: Guide for using shellwright MCP tools to launch, interact with, and test LlamaOfFate programs in a PTY session. Use this when asked to run examples, test the CLI, or do interactive testing via shellwright.
---

# Shellwright Testing

This skill covers using the shellwright MCP tools (`shell_start`, `shell_send`, `shell_read`, `shell_stop`, `shell_screenshot`) to launch and test LlamaOfFate programs interactively.

## Tool Behavior

### `read` vs `send`

- **`shell_read`** returns the current terminal buffer instantly. Use it to poll for output. No delay needed.
- **`shell_send`** sends input and returns `bufferBefore`/`bufferAfter`. Use it only when sending actual input (commands, text). Do NOT send empty strings to poll — use `read` instead.
- The `delay_ms` on `send` controls how long to wait after sending before capturing `bufferAfter`. If `bufferAfter` doesn't show the expected output, the app may still be processing — follow up with `read`.

### Delay Guidelines for `send`

| Operation | `delay_ms` |
|---|---|
| Built-in commands (`help`, `scene`, `character`, `aspects`, `status`) | 200 |
| App startup (first read after `shell_start`) | 500 |
| LLM-powered actions (natural language input) | 200, then poll with `read` |

For LLM calls, send with a short delay (200ms) then poll with `read` until the input prompt reappears. Do not use large delays on `send` — the tool does not return early.

### Completion Detection

Check for the app's input prompt at the end of the `read` buffer to know when the app is ready for the next command:

| Program | Ready prompt |
|---|---|
| `llamaoffate` | `\n> ` |
| `llm-scene-loop` | `\n> ` |
| `scenario-walkthrough` | `\n> ` |
| Batch programs | Process exits; buffer contains full output |

### Terminal Sizing

Use `cols: 120, rows: 50` to minimize scrolling artifacts in the buffer.

## Launching Programs

Launch directly via `shell_start` with the binary as `command` and flags as `args` — no bash wrapper needed. This keeps the buffer clean (no shell prompt noise).

```
shell_start(command="./bin/llamaoffate", args=[], cols=120, rows=50)
shell_start(command="./bin/llm-scene-loop", args=["-scene", "saloon", "-log", ""], cols=120, rows=50)
shell_start(command="./bin/scenario-generator", args=["-name", "Test", "-concept", "Warrior"], cols=120, rows=50)
```

CWD defaults to the workspace root, so relative paths to `./bin/` and `configs/` work.

Use `-log ""` on programs that support it to disable session log file creation during testing.

## Building

No justfile targets exist for examples. Build manually:

```bash
go build -o ./bin/llamaoffate ./cmd/cli
go build -o ./bin/llm-scene-loop ./examples/llm-scene-loop
go build -o ./bin/scenario-generator ./examples/scenario-generator
go build -o ./bin/scenario-walkthrough ./examples/scenario-walkthrough
go build -o ./bin/scene-generator ./examples/scene-generator
```

All programs require `configs/azure-llm.yaml` for LLM access.

## Program Inventory

| Program | Source | Mode | Input | Exit |
|---|---|---|---|---|
| llamaoffate | `cmd/cli` | Interactive | Free-text at `> ` | `exit`, `quit`, `end`, `leave`, `resolve` |
| llm-scene-loop | `examples/llm-scene-loop` | Interactive | Free-text at `> ` | `exit`, `quit` |
| scenario-generator | `examples/scenario-generator` | Batch | None | Process exits |
| scenario-walkthrough | `examples/scenario-walkthrough` | Interactive | Line input at `> ` | `quit`, `q`, or max-scenes reached |
| scene-generator | `examples/scene-generator` | Batch | None | Process exits |

## Testing Patterns

### Batch Programs (scenario-generator, scene-generator)

```
1. shell_start(command="./bin/scenario-generator", args=["-name", "Test", "-concept", "Warrior", "-log", ""])
2. read — output is already in the buffer (LLM call may take time; poll with read until complete)
3. Verify expected content in output
4. shell_stop
```

### Interactive Programs (llamaoffate, llm-scene-loop, scenario-walkthrough)

```
1. shell_start(command="./bin/llm-scene-loop", args=["-scene", "saloon", "-log", ""])
2. read — confirm startup, look for > prompt
3. send("help\r", delay_ms=200) — test built-in commands, check bufferAfter
4. send("I look around\r", delay_ms=200) — for LLM input, then poll with read until > reappears
5. send("exit\r", delay_ms=200) — end the session
6. shell_stop
```

### Screenshots

Use `shell_screenshot` to capture terminal state as PNG when visual verification is needed. Primarily useful for debugging buffer issues or documenting behavior.
