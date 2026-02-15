import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { NpcPanel } from "@/components/game/NpcPanel"

describe("NpcPanel", () => {
  it("shows empty state when no NPCs", () => {
    render(<NpcPanel npcs={[]} />)
    expect(screen.getByText("No NPCs in scene")).toBeInTheDocument()
  })

  it("renders NPC names", () => {
    render(
      <NpcPanel
        npcs={[
          { name: "Grim", highConcept: "Dock Boss", aspects: [], isTakenOut: false },
          { name: "Luna", highConcept: "", aspects: [], isTakenOut: false },
        ]}
      />,
    )

    expect(screen.getByText("Grim")).toBeInTheDocument()
    expect(screen.getByText("Luna")).toBeInTheDocument()
  })

  it("shows taken-out indicator", () => {
    render(
      <NpcPanel
        npcs={[
          { name: "Grim", highConcept: "", aspects: [], isTakenOut: true },
        ]}
      />,
    )

    expect(screen.getByText("Grim")).toBeInTheDocument()
    expect(screen.getByText("(taken out)")).toBeInTheDocument()
  })

  it("renders NPC without details as simple entry", () => {
    const { container } = render(
      <NpcPanel
        npcs={[
          { name: "Guard", highConcept: "", aspects: [], isTakenOut: false },
        ]}
      />,
    )

    // Should not have a collapsible trigger (no details to expand)
    expect(container.querySelector("button")).toBeNull()
    expect(screen.getByText("Guard")).toBeInTheDocument()
  })
})
