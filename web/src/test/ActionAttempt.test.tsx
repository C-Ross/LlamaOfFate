import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { ActionAttempt } from "@/components/game/ActionAttempt"

describe("ActionAttempt", () => {
  it("renders action description", () => {
    render(<ActionAttempt data={{ Description: "Jesse draws his six-shooter." }} />)
    expect(screen.getByText("Jesse draws his six-shooter.")).toBeInTheDocument()
  })

  it("renders the Action label", () => {
    render(<ActionAttempt data={{ Description: "Cast a spell" }} />)
    expect(screen.getByText(/Action:/)).toBeInTheDocument()
  })

  it("has accessible status role", () => {
    render(<ActionAttempt data={{ Description: "Sneak past the guard" }} />)
    expect(screen.getByRole("status", {
      name: "Action attempt: Sneak past the guard",
    })).toBeInTheDocument()
  })

  it("merges custom className", () => {
    const { container } = render(
      <ActionAttempt data={{ Description: "test" }} className="custom-class" />,
    )
    expect(container.firstElementChild?.className).toContain("custom-class")
  })
})
