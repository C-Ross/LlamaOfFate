import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { ChatMessage } from "@/components/game/ChatMessage"
import type { GameEvent } from "@/lib/types"

function makeEvent(event: string, data: unknown): GameEvent {
  return { id: `test-${event}`, event: event as GameEvent["event"], data }
}

describe("ChatMessage", () => {
  it("renders narrative text", () => {
    const event = makeEvent("narrative", { Text: "The saloon doors swing open." })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("The saloon doors swing open.")).toBeInTheDocument()
  })

  it("renders narrative with scene name header", () => {
    const event = makeEvent("narrative", {
      Text: "A dusty road stretches ahead.",
      SceneName: "The Frontier",
    })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("The Frontier")).toBeInTheDocument()
    expect(screen.getByText("A dusty road stretches ahead.")).toBeInTheDocument()
  })

  it("renders dialog with player input and GM response", () => {
    const event = makeEvent("dialog", {
      PlayerInput: "I search the room.",
      GMResponse: "You find a hidden lever.",
    })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("I search the room.")).toBeInTheDocument()
    expect(screen.getByText("You find a hidden lever.")).toBeInTheDocument()
  })

  it("renders dialog with only GM response", () => {
    const event = makeEvent("dialog", {
      PlayerInput: "",
      GMResponse: "A voice echoes from the shadows.",
    })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("A voice echoes from the shadows.")).toBeInTheDocument()
  })

  it("renders system message", () => {
    const event = makeEvent("system_message", { Message: "You have 3 fate points." })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("You have 3 fate points.")).toBeInTheDocument()
  })

  it("renders action attempt", () => {
    const event = makeEvent("action_attempt", { Description: "Jesse draws his six-shooter." })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("Jesse draws his six-shooter.")).toBeInTheDocument()
  })

  it("renders action result with skill and outcome", () => {
    const event = makeEvent("action_result", {
      Skill: "Shoot",
      SkillLevel: "Great (+4)",
      Bonuses: 0,
      Result: "Superb (+5)",
      Outcome: "Success with Style",
    })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("Shoot (Great (+4))")).toBeInTheDocument()
    expect(screen.getByText("Success with Style")).toBeInTheDocument()
  })

  it("renders game over", () => {
    const event = makeEvent("game_over", { Reason: "The story has ended." })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("Game Over")).toBeInTheDocument()
    expect(screen.getByText("The story has ended.")).toBeInTheDocument()
  })

  it("renders conflict start", () => {
    const event = makeEvent("conflict_start", {
      ConflictType: "physical",
      InitiatorName: "Bandit",
      Participants: [
        { CharacterID: "1", CharacterName: "Jesse", Initiative: 4, IsPlayer: true },
        { CharacterID: "2", CharacterName: "Bandit", Initiative: 2, IsPlayer: false },
      ],
    })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("Conflict! (physical)")).toBeInTheDocument()
  })

  it("renders turn announcement", () => {
    const event = makeEvent("turn_announcement", {
      CharacterName: "Jesse",
      TurnNumber: 1,
      IsPlayer: true,
    })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("Turn 1: Jesse (You)")).toBeInTheDocument()
  })

  it("renders invoke prompt with aspects", () => {
    const event = makeEvent("invoke_prompt", {
      Available: [
        { Name: "Quick Draw", Source: "character", FreeInvokes: 0 },
        { Name: "Dark Alley", Source: "situation", FreeInvokes: 1 },
      ],
      FatePoints: 3,
      CurrentResult: "Fair (+2)",
      ShiftsNeeded: 2,
    })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("Invoke an Aspect?")).toBeInTheDocument()
    expect(screen.getByText("Quick Draw")).toBeInTheDocument()
    expect(screen.getByText("Dark Alley")).toBeInTheDocument()
  })

  it("renders milestone", () => {
    const event = makeEvent("milestone", {
      Type: "scenario_complete",
      ScenarioTitle: "Trouble in Redemption Gulch",
      FatePoints: 5,
    })
    render(<ChatMessage event={event} />)
    expect(screen.getByText("Milestone: Trouble in Redemption Gulch")).toBeInTheDocument()
  })

  it("renders nothing for unknown event type", () => {
    const event = makeEvent("unknown_event" as string, {})
    const { container } = render(<ChatMessage event={event} />)
    // Should render the wrapper div but no inner content
    expect(container.querySelector("div")).toBeInTheDocument()
  })
})
