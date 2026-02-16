import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { RollResult } from "@/components/game/RollResult"
import type { ActionResultEventData } from "@/lib/types"

function makeData(overrides: Partial<ActionResultEventData> = {}): ActionResultEventData {
  return {
    Skill: "Fight",
    SkillRank: "Good",
    SkillBonus: 3,
    Bonuses: 0,
    Result: "[+][-][ ][+] (Total: Great (+4) vs Difficulty Fair (+2))",
    Outcome: "Success",
    Total: 4,
    TotalRank: "Great",
    Difficulty: 2,
    DiffRank: "Fair",
    ...overrides,
  }
}

describe("RollResult", () => {
  it("renders skill name and rank", () => {
    render(<RollResult data={makeData()} />)
    expect(screen.getByText(/Fight/)).toBeInTheDocument()
    expect(screen.getByText(/Good \+3/)).toBeInTheDocument()
  })

  it("renders outcome badge", () => {
    render(<RollResult data={makeData({ Outcome: "Success with Style" })} />)
    expect(screen.getByRole("status", { name: "Outcome: Success with Style" })).toBeInTheDocument()
  })

  it("renders four dice faces when parseable", () => {
    render(<RollResult data={makeData()} />)
    const diceGroup = screen.getByRole("group", { name: "Dice roll" })
    expect(diceGroup).toBeInTheDocument()
    const dice = diceGroup.querySelectorAll("[role='img']")
    expect(dice).toHaveLength(4)
  })

  it("renders correct dice face types", () => {
    render(<RollResult data={makeData()} />)
    // [+][-][ ][+] => 2 plus, 1 minus, 1 blank
    expect(screen.getAllByRole("img", { name: "Plus die" })).toHaveLength(2)
    expect(screen.getByRole("img", { name: "Minus die" })).toBeInTheDocument()
    expect(screen.getByRole("img", { name: "Blank die" })).toBeInTheDocument()
  })

  it("shows dice sum", () => {
    render(<RollResult data={makeData()} />)
    // [+][-][ ][+] = +1
    expect(screen.getByText("= +1")).toBeInTheDocument()
  })

  it("shows structured total vs difficulty with ladder names", () => {
    render(<RollResult data={makeData()} />)
    expect(screen.getByText("Total: Great +4 vs Fair +2")).toBeInTheDocument()
  })

  it("shows defender name when rolling against a character", () => {
    render(<RollResult data={makeData({
      DefenderName: "Bandit",
      Difficulty: 3,
      DiffRank: "Good",
    })} />)
    expect(screen.getByText("Total: Great +4 vs Bandit's Good +3")).toBeInTheDocument()
  })

  it("falls back to plain text when dice can't be parsed", () => {
    render(<RollResult data={makeData({ Result: "Good (+3)" })} />)
    expect(screen.getByText("Roll: Good (+3)")).toBeInTheDocument()
    expect(screen.queryByRole("group", { name: "Dice roll" })).not.toBeInTheDocument()
  })

  it("shows bonus line when bonuses > 0", () => {
    render(<RollResult data={makeData({ Bonuses: 2 })} />)
    expect(screen.getByText("+2 bonus from invokes")).toBeInTheDocument()
  })

  it("hides bonus line when bonuses are 0", () => {
    render(<RollResult data={makeData({ Bonuses: 0 })} />)
    expect(screen.queryByText(/bonus/)).not.toBeInTheDocument()
  })

  it("has accessible region label", () => {
    render(<RollResult data={makeData({ Outcome: "Failure" })} />)
    expect(screen.getByRole("region", { name: "Roll result: Fight — Failure" })).toBeInTheDocument()
  })

  it("renders all-negative roll correctly", () => {
    render(<RollResult data={makeData({
      Result: "[-][-][-][-] (Total: Terrible (-2) vs Difficulty Fair (+2))",
      Total: -2,
      TotalRank: "Terrible",
    })} />)
    const dice = screen.getAllByRole("img", { name: "Minus die" })
    expect(dice).toHaveLength(4)
    expect(screen.getByText("= -4")).toBeInTheDocument()
  })

  it("prefers structured DiceFaces over string parsing", () => {
    // Result string has [+][+][+][+] but DiceFaces says all minus
    render(<RollResult data={makeData({
      Result: "[+][+][+][+] (Total: Great (+4) vs Difficulty Fair (+2))",
      DiceFaces: [-1, -1, -1, -1],
    })} />)
    // Should render minus dice from DiceFaces, not plus from string
    const dice = screen.getAllByRole("img", { name: "Minus die" })
    expect(dice).toHaveLength(4)
    expect(screen.queryByRole("img", { name: "Plus die" })).not.toBeInTheDocument()
  })

  it("falls back to string parsing when DiceFaces is missing", () => {
    render(<RollResult data={makeData({
      Result: "[+][-][ ][+] (Total: Great (+4) vs Difficulty Fair (+2))",
      DiceFaces: undefined,
    })} />)
    expect(screen.getAllByRole("img", { name: "Plus die" })).toHaveLength(2)
    expect(screen.getByRole("img", { name: "Minus die" })).toBeInTheDocument()
    expect(screen.getByRole("img", { name: "Blank die" })).toBeInTheDocument()
  })
})
