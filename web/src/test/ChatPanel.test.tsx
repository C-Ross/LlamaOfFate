import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { ChatPanel } from "@/components/game/ChatPanel"
import type { GameEvent } from "@/lib/types"

function makeEvent(id: string, event: string, data: unknown): GameEvent {
  return { id, event: event as GameEvent["event"], data }
}

describe("ChatPanel", () => {
  it("renders empty state when no events", () => {
    render(<ChatPanel events={[]} />)
    expect(screen.getByText("Waiting for the story to begin...")).toBeInTheDocument()
  })

  it("renders narrative events", () => {
    const events: GameEvent[] = [
      makeEvent("1", "narrative", { Text: "The wind howls." }),
      makeEvent("2", "narrative", { Text: "Dust fills the air." }),
    ]
    render(<ChatPanel events={events} />)
    expect(screen.getByText("The wind howls.")).toBeInTheDocument()
    expect(screen.getByText("Dust fills the air.")).toBeInTheDocument()
  })

  it("filters out result_meta events", () => {
    const events: GameEvent[] = [
      makeEvent("1", "narrative", { Text: "Story text." }),
      makeEvent("2", "result_meta", { awaitingInvoke: false }),
    ]
    render(<ChatPanel events={events} />)
    expect(screen.getByText("Story text.")).toBeInTheDocument()
    // result_meta should not render anything visible
  })

  it("shows empty state when only non-displayable events exist", () => {
    const events: GameEvent[] = [
      makeEvent("1", "result_meta", { awaitingInvoke: false }),
    ]
    render(<ChatPanel events={events} />)
    expect(screen.getByText("Waiting for the story to begin...")).toBeInTheDocument()
  })

  it("renders mixed event types in order", () => {
    const events: GameEvent[] = [
      makeEvent("1", "narrative", { Text: "Scene opens.", SceneName: "The Saloon" }),
      makeEvent("2", "dialog", { PlayerInput: "Hello", GMResponse: "Howdy partner." }),
      makeEvent("3", "system_message", { Message: "Fate point spent." }),
    ]
    render(<ChatPanel events={events} />)
    expect(screen.getByText("The Saloon")).toBeInTheDocument()
    expect(screen.getByText("Scene opens.")).toBeInTheDocument()
    expect(screen.getByText("Hello")).toBeInTheDocument()
    expect(screen.getByText("Howdy partner.")).toBeInTheDocument()
    expect(screen.getByText("Fate point spent.")).toBeInTheDocument()
  })
})
