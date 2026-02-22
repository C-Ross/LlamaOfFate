import { useMemo } from "react"
import type {
  GameEvent,
  GameStateSnapshotEventData,
  PlayerSnapshot,
  SituationAspectSnapshot,
  NPCSnapshot,
  StressTrackSnapshot,
  ConsequenceSnapshotEntry,
  AspectCreatedEventData,
  BoostExpiredEventData,
  PlayerStressEventData,
  PlayerConsequenceEventData,
  ConflictStartEventData,
  MilestoneEventData,
  ConcessionEventData,
  InvokeEventData,
  NarrativeEventData,
  RecoveryEventData,
  DamageResolutionEventData,
} from "@/lib/types"

// ---------------------------------------------------------------------------
// Public state shape
// ---------------------------------------------------------------------------

export interface GameState {
  player: PlayerSnapshot | null
  situationAspects: SituationAspectSnapshot[]
  npcs: NPCSnapshot[]
  fatePoints: number
  stressTracks: Record<string, StressTrackSnapshot>
  consequences: ConsequenceSnapshotEntry[]
  inConflict: boolean
  sceneName: string
}

const emptyState: GameState = {
  player: null,
  situationAspects: [],
  npcs: [],
  fatePoints: 0,
  stressTracks: {},
  consequences: [],
  inConflict: false,
  sceneName: "",
}

// ---------------------------------------------------------------------------
// Hook — derives sidebar state from the event stream
// ---------------------------------------------------------------------------

/**
 * Derives sidebar game state from the event stream by scanning events.
 *
 * Instead of using a separate reducer, we derive state from the full event
 * list on each render. The event list is append-only so this is effectively
 * a fold over all events — simple and deterministic.
 */
export function useGameState(events: GameEvent[]): GameState {
  return useMemo(() => deriveState(events), [events])
}

// ---------------------------------------------------------------------------
// Pure derivation function (exported for testing)
// ---------------------------------------------------------------------------

export function deriveState(events: GameEvent[]): GameState {
  let state = { ...emptyState }

  for (const evt of events) {
    switch (evt.event) {
      case "game_state_snapshot": {
        const d = evt.data as GameStateSnapshotEventData
        state = {
          player: d.player,
          situationAspects: d.situationAspects ?? [],
          npcs: d.npcs ?? [],
          fatePoints: d.player?.fatePoints ?? 0,
          stressTracks: d.player?.stressTracks ?? {},
          consequences: d.player?.consequences ?? [],
          inConflict: d.inConflict,
          sceneName: d.sceneName,
        }
        break
      }

      case "narrative": {
        const d = evt.data as NarrativeEventData
        // A narrative event with a SceneName indicates a new scene — reset
        // situation aspects and conflict state (NPCs will be re-sent via a
        // new snapshot if the server emits one, but we clear stale data here).
        if (d.SceneName) {
          state = {
            ...state,
            sceneName: d.SceneName,
            situationAspects: [],
            inConflict: false,
          }
        }
        break
      }

      case "aspect_created": {
        const d = evt.data as AspectCreatedEventData
        state = {
          ...state,
          situationAspects: [
            ...state.situationAspects,
            { name: d.AspectName, freeInvokes: d.FreeInvokes, isBoost: d.IsBoost ?? false },
          ],
        }
        break
      }

      case "boost_expired": {
        const d = evt.data as BoostExpiredEventData
        state = {
          ...state,
          situationAspects: state.situationAspects.filter((a) => a.name !== d.AspectName),
        }
        break
      }

      case "player_stress": {
        const d = evt.data as PlayerStressEventData
        // Parse the TrackState string back to boxes.  The server sends a
        // display string like "[X][X][ ]" — we parse filled vs empty.
        const boxes = parseTrackState(d.TrackState)
        state = {
          ...state,
          stressTracks: {
            ...state.stressTracks,
            [d.StressType]: {
              boxes,
              maxBoxes: boxes.length,
            },
          },
        }
        break
      }

      case "player_consequence": {
        const d = evt.data as PlayerConsequenceEventData
        state = {
          ...state,
          consequences: [
            ...state.consequences,
            { severity: d.Severity, aspect: d.Aspect, recovering: false },
          ],
        }
        break
      }

      case "conflict_start": {
        const d = evt.data as ConflictStartEventData
        // Build NPC list from participants (excluding the player)
        const participantNpcs: NPCSnapshot[] = d.Participants
          .filter((p) => !p.IsPlayer)
          .map((p) => ({
            name: p.CharacterName,
            highConcept: "",
            aspects: [],
            isTakenOut: false,
          }))
        state = {
          ...state,
          inConflict: true,
          // Merge participant NPCs with existing NPCs (keep existing data, add
          // new ones).  Conflict participants may overlap with known NPCs.
          npcs: mergeNpcs(state.npcs, participantNpcs),
        }
        break
      }

      case "conflict_end": {
        state = { ...state, inConflict: false }
        break
      }

      case "milestone": {
        const d = evt.data as MilestoneEventData
        state = { ...state, fatePoints: d.FatePoints }
        break
      }

      case "concession": {
        const d = evt.data as ConcessionEventData
        state = { ...state, fatePoints: d.CurrentFatePoints }
        break
      }

      case "invoke": {
        const d = evt.data as InvokeEventData
        if (!d.IsFree && !d.Failed) {
          state = { ...state, fatePoints: d.FatePointsLeft }
        }
        break
      }

      case "recovery": {
        const d = evt.data as RecoveryEventData
        if (d.Action === "healed") {
          // Remove the healed consequence by matching severity + aspect
          state = {
            ...state,
            consequences: state.consequences.filter(
              (c) => !(c.severity === d.Severity && c.aspect === d.Aspect),
            ),
          }
        } else if (d.Action === "roll" && d.Success) {
          // Mark the consequence as recovering
          state = {
            ...state,
            consequences: state.consequences.map((c) =>
              c.severity === d.Severity && c.aspect === d.Aspect
                ? { ...c, recovering: true }
                : c,
            ),
          }
        }
        break
      }

      case "damage_resolution": {
        const d = evt.data as DamageResolutionEventData
        if (d.TakenOut) {
          // Mark the NPC as taken out
          state = {
            ...state,
            npcs: state.npcs.map((n) =>
              n.name === d.TargetName ? { ...n, isTakenOut: true } : n,
            ),
          }
        }
        break
      }
    }
  }

  return state
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Parse a stress-track display string like "[X][X][ ]" into an array of
 * booleans where `true` = filled.
 */
function parseTrackState(trackState: string): boolean[] {
  const boxes: boolean[] = []
  const re = /\[([^\]]*)\]/g
  let match: RegExpExecArray | null
  while ((match = re.exec(trackState)) !== null) {
    boxes.push(match[1].trim().toUpperCase() === "X")
  }
  return boxes
}

/** Merge new NPCs into the existing list, avoiding duplicates by name. */
function mergeNpcs(existing: NPCSnapshot[], incoming: NPCSnapshot[]): NPCSnapshot[] {
  const byName = new Map<string, NPCSnapshot>()
  for (const n of existing) {
    byName.set(n.name.toLowerCase(), n)
  }
  for (const n of incoming) {
    const key = n.name.toLowerCase()
    if (!byName.has(key)) {
      byName.set(key, n)
    }
  }
  return Array.from(byName.values())
}
