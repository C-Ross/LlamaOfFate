import { describe, it, expect } from "vitest"
import { deriveState } from "../hooks/useGameState"
import type { GameEvent } from "@/lib/types"

function makeEvent(event: string, data: unknown, id?: string): GameEvent {
  return { id: id ?? `evt-${Math.random()}`, event: event as GameEvent["event"], data }
}

describe("deriveState", () => {
  it("returns empty state for no events", () => {
    const state = deriveState([])
    expect(state.player).toBeNull()
    expect(state.situationAspects).toEqual([])
    expect(state.npcs).toEqual([])
    expect(state.fatePoints).toBe(0)
    expect(state.inConflict).toBe(false)
  })

  it("initialises from game_state_snapshot", () => {
    const snapshot = makeEvent("game_state_snapshot", {
      player: {
        name: "Zara",
        highConcept: "Wizard Detective",
        trouble: "Curiosity Kills",
        aspects: ["Well Connected"],
        fatePoints: 3,
        refresh: 3,
        stressTracks: {
          physical: { boxes: [false, false], maxBoxes: 2 },
          mental: { boxes: [false, false], maxBoxes: 2 },
        },
        consequences: [],
      },
      sceneName: "The Docks",
      situationAspects: [
        { name: "Foggy Night", freeInvokes: 1 },
      ],
      npcs: [
        { name: "Grim", highConcept: "Dock Boss", aspects: [], isTakenOut: false },
      ],
      inConflict: false,
    })

    const state = deriveState([snapshot])
    expect(state.player?.name).toBe("Zara")
    expect(state.player?.highConcept).toBe("Wizard Detective")
    expect(state.fatePoints).toBe(3)
    expect(state.sceneName).toBe("The Docks")
    expect(state.situationAspects).toHaveLength(1)
    expect(state.situationAspects[0].name).toBe("Foggy Night")
    expect(state.npcs).toHaveLength(1)
    expect(state.npcs[0].name).toBe("Grim")
    expect(state.stressTracks.physical.boxes).toEqual([false, false])
  })

  it("adds situation aspects from aspect_created events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("aspect_created", { AspectName: "On Fire", FreeInvokes: 2 }),
    ]

    const state = deriveState(events)
    expect(state.situationAspects).toHaveLength(1)
    expect(state.situationAspects[0]).toEqual({ name: "On Fire", freeInvokes: 2, isBoost: false })
  })

  it("marks boost aspects with isBoost=true when aspect_created has IsBoost", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("aspect_created", { AspectName: "Fleeting Opening", FreeInvokes: 1, IsBoost: true }),
    ]

    const state = deriveState(events)
    expect(state.situationAspects).toHaveLength(1)
    expect(state.situationAspects[0]).toEqual({ name: "Fleeting Opening", freeInvokes: 1, isBoost: true })
  })

  it("removes boost from situation aspects when boost_expired fires", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [{ name: "On Fire", freeInvokes: 0, isBoost: false }],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("aspect_created", { AspectName: "Fleeting Opening", FreeInvokes: 1, IsBoost: true }),
      makeEvent("boost_expired", { AspectName: "Fleeting Opening" }),
    ]

    const state = deriveState(events)
    // Boost should be removed; the regular aspect should remain.
    expect(state.situationAspects).toHaveLength(1)
    expect(state.situationAspects[0].name).toBe("On Fire")
  })

  it("resets situation aspects on new scene narrative", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Scene 1",
        situationAspects: [{ name: "Old Aspect", freeInvokes: 0 }],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("narrative", { Text: "A new scene begins...", SceneName: "Scene 2" }),
    ]

    const state = deriveState(events)
    expect(state.sceneName).toBe("Scene 2")
    expect(state.situationAspects).toEqual([])
  })

  it("does not reset aspects for narrative without SceneName", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Scene 1",
        situationAspects: [{ name: "Old Aspect", freeInvokes: 0 }],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("narrative", { Text: "Something happens..." }),
    ]

    const state = deriveState(events)
    expect(state.situationAspects).toHaveLength(1)
  })

  it("updates stress from player_stress events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: { physical: { boxes: [false, false], maxBoxes: 2 } }, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("player_stress", { Shifts: 2, StressType: "physical", TrackState: "[X][X]" }),
    ]

    const state = deriveState(events)
    expect(state.stressTracks.physical.boxes).toEqual([true, true])
  })

  it("adds consequences from player_consequence events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("player_consequence", { Severity: "mild", Aspect: "Bruised Ribs", Absorbed: 2 }),
    ]

    const state = deriveState(events)
    expect(state.consequences).toHaveLength(1)
    expect(state.consequences[0]).toEqual({ severity: "mild", aspect: "Bruised Ribs", recovering: false })
  })

  it("sets inConflict on conflict_start and clears on conflict_end", () => {
    const events = [
      makeEvent("conflict_start", {
        ConflictType: "physical",
        InitiatorName: "Grim",
        Participants: [
          { CharacterID: "p1", CharacterName: "Zara", Initiative: 3, IsPlayer: true },
          { CharacterID: "n1", CharacterName: "Grim", Initiative: 2, IsPlayer: false },
        ],
      }),
      makeEvent("conflict_end", { Reason: "victory" }),
    ]

    // After conflict_start only
    const mid = deriveState(events.slice(0, 1))
    expect(mid.inConflict).toBe(true)
    expect(mid.npcs).toHaveLength(1)
    expect(mid.npcs[0].name).toBe("Grim")

    // After conflict_end
    const end = deriveState(events)
    expect(end.inConflict).toBe(false)
  })

  it("updates fate points from milestone events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 1, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("milestone", { Type: "scenario_complete", ScenarioTitle: "Test", FatePoints: 3 }),
    ]

    const state = deriveState(events)
    expect(state.fatePoints).toBe(3)
  })

  it("updates fate points from concession events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 1, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("concession", { FatePointsGained: 2, ConsequenceCount: 1, CurrentFatePoints: 3 }),
    ]

    const state = deriveState(events)
    expect(state.fatePoints).toBe(3)
  })

  it("updates fate points from paid invoke events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("invoke", { AspectName: "Wizard Detective", IsFree: false, IsReroll: false, FatePointsLeft: 2, NewTotal: "+5", Failed: false }),
    ]

    const state = deriveState(events)
    expect(state.fatePoints).toBe(2)
  })

  it("does not update fate points from free invoke events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [{ name: "On Fire", freeInvokes: 2, isBoost: false }],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("invoke", { AspectName: "On Fire", IsFree: true, IsReroll: false, FatePointsLeft: 0, NewTotal: "+5", Failed: false }),
    ]

    const state = deriveState(events)
    expect(state.fatePoints).toBe(3)
  })

  it("decrements free invokes on situation aspect from free invoke events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [{ name: "On Fire", freeInvokes: 2, isBoost: false }],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("invoke", { AspectName: "On Fire", IsFree: true, IsReroll: false, FatePointsLeft: 0, NewTotal: "+5", Failed: false }),
    ]

    const state = deriveState(events)
    expect(state.situationAspects).toHaveLength(1)
    expect(state.situationAspects[0].freeInvokes).toBe(1)
  })

  it("decrements free invokes to zero after multiple free invokes", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [{ name: "On Fire", freeInvokes: 2, isBoost: false }],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("invoke", { AspectName: "On Fire", IsFree: true, IsReroll: false, FatePointsLeft: 0, NewTotal: "+5", Failed: false }),
      makeEvent("invoke", { AspectName: "On Fire", IsFree: true, IsReroll: false, FatePointsLeft: 0, NewTotal: "+7", Failed: false }),
    ]

    const state = deriveState(events)
    expect(state.situationAspects).toHaveLength(1)
    expect(state.situationAspects[0].freeInvokes).toBe(0)
  })

  it("does not decrement free invokes on failed invoke", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 0, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [{ name: "On Fire", freeInvokes: 1, isBoost: false }],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("invoke", { AspectName: "On Fire", IsFree: false, IsReroll: false, FatePointsLeft: 0, NewTotal: "", Failed: true }),
    ]

    const state = deriveState(events)
    expect(state.situationAspects[0].freeInvokes).toBe(1)
  })

  it("handles null arrays in snapshot gracefully", () => {
    const snapshot = makeEvent("game_state_snapshot", {
      player: { name: "Zara", highConcept: "", trouble: "", aspects: null, fatePoints: 3, refresh: 3, stressTracks: null, consequences: null },
      sceneName: "Test",
      situationAspects: null,
      npcs: null,
      inConflict: false,
    })

    const state = deriveState([snapshot])
    expect(state.situationAspects).toEqual([])
    expect(state.npcs).toEqual([])
    expect(state.stressTracks).toEqual({})
    expect(state.consequences).toEqual([])
  })

  it("removes healed consequences from recovery events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("player_consequence", { Severity: "mild", Aspect: "Bruised Ribs", Absorbed: 2 }),
      makeEvent("recovery", { Action: "healed", Aspect: "Bruised Ribs", Severity: "mild" }),
    ]

    const state = deriveState(events)
    expect(state.consequences).toEqual([])
  })

  it("marks consequences as recovering on successful recovery roll", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("player_consequence", { Severity: "moderate", Aspect: "Broken Arm", Absorbed: 4 }),
      makeEvent("recovery", { Action: "roll", Aspect: "Broken Arm", Severity: "moderate", Skill: "Will", RollResult: 3, Difficulty: "Great", Success: true }),
    ]

    const state = deriveState(events)
    expect(state.consequences).toHaveLength(1)
    expect(state.consequences[0].recovering).toBe(true)
  })

  it("does not change consequences on failed recovery roll", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [],
        inConflict: false,
      }),
      makeEvent("player_consequence", { Severity: "moderate", Aspect: "Broken Arm", Absorbed: 4 }),
      makeEvent("recovery", { Action: "roll", Aspect: "Broken Arm", Severity: "moderate", Skill: "Will", RollResult: 1, Difficulty: "Great", Success: false }),
    ]

    const state = deriveState(events)
    expect(state.consequences).toHaveLength(1)
    expect(state.consequences[0].recovering).toBe(false)
  })

  it("marks NPCs as taken out from damage_resolution events", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [{ name: "Grim", highConcept: "Dock Boss", aspects: [], isTakenOut: false }],
        inConflict: true,
      }),
      makeEvent("damage_resolution", { TargetName: "Grim", TakenOut: true, VictoryEnd: false }),
    ]

    const state = deriveState(events)
    expect(state.npcs).toHaveLength(1)
    expect(state.npcs[0].isTakenOut).toBe(true)
  })

  it("does not mark NPC as taken out when TakenOut is false", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "", trouble: "", aspects: [], fatePoints: 3, refresh: 3, stressTracks: {}, consequences: [] },
        sceneName: "Test",
        situationAspects: [],
        npcs: [{ name: "Grim", highConcept: "Dock Boss", aspects: [], isTakenOut: false }],
        inConflict: true,
      }),
      makeEvent("damage_resolution", { TargetName: "Grim", TakenOut: false, VictoryEnd: false }),
    ]

    const state = deriveState(events)
    expect(state.npcs[0].isTakenOut).toBe(false)
  })

  it("resets state when a second snapshot arrives (scene transition)", () => {
    const events = [
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "Wizard", trouble: "Curious", aspects: [], fatePoints: 3, refresh: 3, stressTracks: { physical: { boxes: [true, false], maxBoxes: 2 } }, consequences: [{ severity: "mild", aspect: "Bruised", recovering: false }] },
        sceneName: "Scene 1",
        situationAspects: [{ name: "Foggy", freeInvokes: 0 }],
        npcs: [{ name: "Grim", highConcept: "Boss", aspects: [], isTakenOut: false }],
        inConflict: true,
      }),
      // Second snapshot after scene transition — stress cleared, new NPCs, etc.
      makeEvent("game_state_snapshot", {
        player: { name: "Zara", highConcept: "Wizard", trouble: "Curious", aspects: [], fatePoints: 3, refresh: 3, stressTracks: { physical: { boxes: [false, false], maxBoxes: 2 } }, consequences: [] },
        sceneName: "Scene 2",
        situationAspects: [],
        npcs: [{ name: "Luna", highConcept: "Informant", aspects: [], isTakenOut: false }],
        inConflict: false,
      }),
    ]

    const state = deriveState(events)
    expect(state.sceneName).toBe("Scene 2")
    expect(state.stressTracks.physical.boxes).toEqual([false, false])
    expect(state.consequences).toEqual([])
    expect(state.npcs).toHaveLength(1)
    expect(state.npcs[0].name).toBe("Luna")
    expect(state.inConflict).toBe(false)
    expect(state.situationAspects).toEqual([])
  })
})
