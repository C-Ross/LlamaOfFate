import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { DefenseRoll } from "@/components/game/DefenseRoll"
import type { DefenseRollEventData } from "@/lib/types"

function makeData(overrides: Partial<DefenseRollEventData> = {}): DefenseRollEventData {
  return {
    DefenderName: "Bandit",
    Skill: "Athletics",
    Result: "Good (+3)",
    ...overrides,
  }
}

describe("DefenseRoll", () => {
  it("renders defender name and skill", () => {
    render(<DefenseRoll data={makeData()} />)
    expect(screen.getByText(/Bandit/)).toBeInTheDocument()
    expect(screen.getByText(/Athletics/)).toBeInTheDocument()
  })

  it("renders result as bold text when no dice faces present", () => {
    render(<DefenseRoll data={makeData({ Result: "Fair (+2)" })} />)
    expect(screen.getByText("Fair (+2)")).toBeInTheDocument()
  })

  it("renders dice faces when result contains them", () => {
    render(<DefenseRoll data={makeData({
      Result: "[+][ ][-][+] Good (+3)",
    })} />)
    expect(screen.getAllByRole("img", { name: "Plus die" })).toHaveLength(2)
    expect(screen.getByRole("img", { name: "Blank die" })).toBeInTheDocument()
    expect(screen.getByRole("img", { name: "Minus die" })).toBeInTheDocument()
  })

  it("has accessible status label", () => {
    render(<DefenseRoll data={makeData()} />)
    expect(screen.getByRole("status", {
      name: "Defense roll: Bandit — Athletics — Good (+3)",
    })).toBeInTheDocument()
  })

  it("renders the Defense label", () => {
    render(<DefenseRoll data={makeData()} />)
    expect(screen.getByText(/Defense:/)).toBeInTheDocument()
  })

  it("renders dice faces from structured DiceFaces field", () => {
    render(<DefenseRoll data={makeData({
      Result: "Good (+3)",
      DiceFaces: [1, 0, -1, 1],
    })} />)
    expect(screen.getAllByRole("img", { name: "Plus die" })).toHaveLength(2)
    expect(screen.getByRole("img", { name: "Blank die" })).toBeInTheDocument()
    expect(screen.getByRole("img", { name: "Minus die" })).toBeInTheDocument()
  })

  it("prefers structured DiceFaces over string parsing", () => {
    // Result has [+][+][+][+] but DiceFaces says all blank
    render(<DefenseRoll data={makeData({
      Result: "[+][+][+][+] Good (+3)",
      DiceFaces: [0, 0, 0, 0],
    })} />)
    const dice = screen.getAllByRole("img", { name: "Blank die" })
    expect(dice).toHaveLength(4)
    expect(screen.queryByRole("img", { name: "Plus die" })).not.toBeInTheDocument()
  })
})
