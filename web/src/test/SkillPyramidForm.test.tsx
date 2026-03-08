import { render, screen, fireEvent } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { SkillPyramidForm } from "@/components/game/SkillPyramidForm"
import { Ladder } from "@/lib/skills"

describe("SkillPyramidForm", () => {
  it("renders all four tier sections", () => {
    render(<SkillPyramidForm skills={{}} onChange={() => {}} />)
    expect(screen.getByTestId("tier-4")).toBeInTheDocument()
    expect(screen.getByTestId("tier-3")).toBeInTheDocument()
    expect(screen.getByTestId("tier-2")).toBeInTheDocument()
    expect(screen.getByTestId("tier-1")).toBeInTheDocument()
  })

  it("renders 10 skill select triggers total", () => {
    render(<SkillPyramidForm skills={{}} onChange={() => {}} />)
    // 1 + 2 + 3 + 4 = 10
    const triggers = screen.getAllByRole("combobox")
    expect(triggers).toHaveLength(10)
  })

  it("shows 0/10 skills assigned when empty", () => {
    render(<SkillPyramidForm skills={{}} onChange={() => {}} />)
    expect(screen.getByText("0/10 skills assigned")).toBeInTheDocument()
  })

  it("shows correct count when skills are provided", () => {
    const skills = {
      Notice: Ladder.Great,
      Athletics: Ladder.Good,
      Will: Ladder.Good,
    }
    render(<SkillPyramidForm skills={skills} onChange={() => {}} />)
    expect(screen.getByText("3/10 skills assigned")).toBeInTheDocument()
  })

  it("calls onChange with defaults when Use Defaults is clicked", () => {
    const onChange = vi.fn()
    render(<SkillPyramidForm skills={{}} onChange={onChange} />)
    fireEvent.click(screen.getByTestId("use-defaults-button"))
    expect(onChange).toHaveBeenCalledTimes(1)
    const result = onChange.mock.calls[0][0] as Record<string, number>
    expect(Object.keys(result)).toHaveLength(10)
  })

  it("calls onChange with empty object when Clear is clicked", () => {
    const onChange = vi.fn()
    const skills = { Notice: Ladder.Great }
    render(<SkillPyramidForm skills={skills} onChange={onChange} />)
    fireEvent.click(screen.getByTestId("clear-skills-button"))
    expect(onChange).toHaveBeenCalledWith({})
  })

  it("does not show Clear button when no skills assigned", () => {
    render(<SkillPyramidForm skills={{}} onChange={() => {}} />)
    expect(screen.queryByTestId("clear-skills-button")).not.toBeInTheDocument()
  })

  it("shows tier progress counts", () => {
    const skills = {
      Notice: Ladder.Great,
      Athletics: Ladder.Good,
    }
    render(<SkillPyramidForm skills={skills} onChange={() => {}} />)
    // Great tier: 1/1
    expect(screen.getByTestId("tier-4")).toHaveTextContent("(1/1)")
    // Good tier: 1/2
    expect(screen.getByTestId("tier-3")).toHaveTextContent("(1/2)")
    // Fair tier: 0/3
    expect(screen.getByTestId("tier-2")).toHaveTextContent("(0/3)")
  })

  it("renders the form container with correct test id", () => {
    render(<SkillPyramidForm skills={{}} onChange={() => {}} />)
    expect(screen.getByTestId("skill-pyramid-form")).toBeInTheDocument()
  })
})
