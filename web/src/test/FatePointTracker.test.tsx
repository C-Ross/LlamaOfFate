import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { FatePointTracker } from "@/components/game/FatePointTracker"

describe("FatePointTracker", () => {
  it("renders current fate points", () => {
    render(<FatePointTracker current={3} />)
    expect(screen.getByText("3")).toBeInTheDocument()
  })

  it("shows refresh value when provided", () => {
    render(<FatePointTracker current={2} refresh={3} />)
    expect(screen.getByText("2")).toBeInTheDocument()
    expect(screen.getByText("/ 3 refresh")).toBeInTheDocument()
  })

  it("omits refresh when not provided", () => {
    render(<FatePointTracker current={5} />)
    expect(screen.queryByText(/refresh/)).not.toBeInTheDocument()
  })
})
