import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { SidebarCard } from "../components/SidebarCard"

describe("SidebarCard", () => {
  it("renders the title", () => {
    render(<SidebarCard title="Test Title">Content</SidebarCard>)
    expect(screen.getByText("Test Title")).toBeInTheDocument()
  })

  it("renders children", () => {
    render(
      <SidebarCard title="Card">
        <span>Child content</span>
      </SidebarCard>
    )
    expect(screen.getByText("Child content")).toBeInTheDocument()
  })
})
