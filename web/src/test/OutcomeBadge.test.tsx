import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { OutcomeBadge } from "@/components/game/OutcomeBadge"

describe("OutcomeBadge", () => {
  it("renders Success", () => {
    render(<OutcomeBadge outcome="Success" />)
    const badge = screen.getByRole("status", { name: "Outcome: Success" })
    expect(badge).toBeInTheDocument()
    expect(badge).toHaveTextContent("Success")
  })

  it("renders Failure", () => {
    render(<OutcomeBadge outcome="Failure" />)
    const badge = screen.getByRole("status", { name: "Outcome: Failure" })
    expect(badge).toBeInTheDocument()
    expect(badge.className).toContain("outcome-failure")
  })

  it("renders Tie", () => {
    render(<OutcomeBadge outcome="Tie" />)
    const badge = screen.getByRole("status", { name: "Outcome: Tie" })
    expect(badge).toBeInTheDocument()
    expect(badge.className).toContain("outcome-tie")
  })

  it("renders Success with Style with glow effect", () => {
    render(<OutcomeBadge outcome="Success with Style" />)
    const badge = screen.getByRole("status", { name: "Outcome: Success with Style" })
    expect(badge).toBeInTheDocument()
    expect(badge.className).toContain("outcome-style")
    expect(badge.className).toContain("shadow")
  })

  it("renders Success styling (not style) for plain Success", () => {
    render(<OutcomeBadge outcome="Success" />)
    const badge = screen.getByRole("status")
    expect(badge.className).toContain("outcome-success")
    expect(badge.className).not.toContain("shadow")
  })

  it("applies small size variant", () => {
    render(<OutcomeBadge outcome="Tie" size="sm" />)
    const badge = screen.getByRole("status")
    expect(badge.className).toContain("px-2")
  })

  it("applies medium size variant by default", () => {
    render(<OutcomeBadge outcome="Tie" />)
    const badge = screen.getByRole("status")
    expect(badge.className).toContain("px-3")
  })

  it("shows hover tooltip with outcome text", () => {
    render(<OutcomeBadge outcome="Failure" />)
    const badge = screen.getByRole("status")
    expect(badge).toHaveAttribute("title", "Failure")
  })

  it("merges custom className", () => {
    render(<OutcomeBadge outcome="Tie" className="my-class" />)
    const badge = screen.getByRole("status")
    expect(badge.className).toContain("my-class")
  })
})
