import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { AspectBadge } from "@/components/game/AspectBadge"

describe("AspectBadge", () => {
  it("renders aspect name", () => {
    render(<AspectBadge name="Wizard Detective" kind="high-concept" />)
    expect(screen.getByText("Wizard Detective")).toBeInTheDocument()
  })

  it("shows free invokes count when > 0", () => {
    render(<AspectBadge name="On Fire" kind="situation" freeInvokes={2} />)
    expect(screen.getByText("On Fire")).toBeInTheDocument()
    expect(screen.getByText("2")).toBeInTheDocument()
  })

  it("does not show free invokes when 0", () => {
    const { container } = render(<AspectBadge name="Test" kind="general" freeInvokes={0} />)
    // Should not have the invokes sub-span
    const spans = container.querySelectorAll("span span")
    expect(spans).toHaveLength(0)
  })

  it("applies kind-specific styles", () => {
    const { container } = render(<AspectBadge name="Test" kind="trouble" />)
    const badge = container.firstChild as HTMLElement
    expect(badge.className).toContain("aspect-trouble")
  })
})
