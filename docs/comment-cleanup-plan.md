# Engine Refactoring Plan

**Date:** 2026-03-01  
**Scope:** `internal/engine/` — 45,165 Go lines

## Completed

| Commit | Refactoring | Net lines |
|--------|-------------|-----------|
| `8d08761` | Consolidate 4 duplicate mock LLM clients into `testLLMClient` | −185 |
| `ec9d3de` | Replace manual nameless NPC checks with `IsNameless()` | −4 |
| `6aa045e` | Extract `processResult()` from 3 copy-pasted ScenarioManager methods | −19 |
| `952c4b0` | Consolidate 5 SM setup helpers into shared `setupTestSM` | +14 |
| | **Total** | **−194** |

## Remaining

### 1. Split ScenarioManager (1061 lines)

`scenario_manager.go` is the largest file. Candidates for extraction:
- **NPC registry** — NPC creation, lookup, and nameless-NPC management
- **Scene recovery** — retry/fallback logic when scene generation fails
- **Scenario serialization** — save/load state for persistence

### 2. Split ConflictManager (929 lines)

`conflict.go` is the second largest. Candidates:
- **Initiative tracker** — initiative calculation, sorting, turn order
- **Damage resolver** — stress/consequence application, overflow handling

## Comments

Comment cleanup (separator banners, redundant godoc) would save ~80–90 lines (0.2% of codebase). Not worth a dedicated pass — clean up opportunistically when touching files for the above refactorings.
