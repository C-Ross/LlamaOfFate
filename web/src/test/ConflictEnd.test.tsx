import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { ConflictEnd } from "@/components/game/ConflictEnd"

describe("ConflictEnd", () => {
  it("renders conflict end reason", () => {
    render(<ConflictEnd data={{ Reason: "All enemies defeated" }} />)
    expect(
      screen.getByText("Conflict Ended: All enemies defeated"),
    ).toBeInTheDocument()
  })

  it("has accessible status role", () => {
    render(<ConflictEnd data={{ Reason: "Concession accepted" }} />)
    expect(
      screen.getByRole("status", {
        name: "Conflict ended: Concession accepted",
      }),
    ).toBeInTheDocument()
  })

  it("merges custom className", () => {
    const { container } = render(
      <ConflictEnd
        data={{ Reason: "Victory" }}
        className="custom-class"
      />,
    )
    expect(container.firstElementChild?.className).toContain("custom-class")
  })
})
