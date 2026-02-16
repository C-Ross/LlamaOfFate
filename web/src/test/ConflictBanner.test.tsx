import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { ConflictBanner } from "@/components/game/ConflictBanner"

describe("ConflictBanner", () => {
  it("renders when active", () => {
    render(<ConflictBanner active={true} />)
    expect(screen.getByText("Conflict In Progress")).toBeInTheDocument()
  })

  it("renders nothing when inactive", () => {
    const { container } = render(<ConflictBanner active={false} />)
    expect(container.innerHTML).toBe("")
  })

  it("has alert role when active", () => {
    render(<ConflictBanner active={true} />)
    expect(
      screen.getByRole("alert", { name: "Conflict in progress" }),
    ).toBeInTheDocument()
  })

  it("merges custom className", () => {
    const { container } = render(
      <ConflictBanner active={true} className="custom-class" />,
    )
    expect(container.firstElementChild?.className).toContain("custom-class")
  })
})
