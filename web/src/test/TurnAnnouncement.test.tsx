import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { TurnAnnouncement } from "@/components/game/TurnAnnouncement"

describe("TurnAnnouncement", () => {
  it("renders turn number and character name", () => {
    render(
      <TurnAnnouncement
        data={{ CharacterName: "Jesse", TurnNumber: 2, IsPlayer: false }}
      />,
    )
    expect(screen.getByText("Turn 2: Jesse")).toBeInTheDocument()
  })

  it("shows (You) for player turns", () => {
    render(
      <TurnAnnouncement
        data={{ CharacterName: "Jesse", TurnNumber: 1, IsPlayer: true }}
      />,
    )
    expect(screen.getByText("Turn 1: Jesse (You)")).toBeInTheDocument()
  })

  it("has accessible status role", () => {
    render(
      <TurnAnnouncement
        data={{ CharacterName: "Bandit", TurnNumber: 3, IsPlayer: false }}
      />,
    )
    expect(
      screen.getByRole("status", { name: "Turn 3: Bandit" }),
    ).toBeInTheDocument()
  })

  it("merges custom className", () => {
    const { container } = render(
      <TurnAnnouncement
        data={{ CharacterName: "Jesse", TurnNumber: 1, IsPlayer: true }}
        className="custom-class"
      />,
    )
    expect(container.firstElementChild?.className).toContain("custom-class")
  })
})
