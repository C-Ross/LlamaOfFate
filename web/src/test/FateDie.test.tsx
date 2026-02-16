import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { FateDie } from "@/components/game/FateDie"

describe("FateDie", () => {
  it("renders a plus die with + symbol", () => {
    render(<FateDie face={1} />)
    const die = screen.getByRole("img", { name: "Plus die" })
    expect(die).toBeInTheDocument()
    expect(die).toHaveTextContent("+")
  })

  it("renders a minus die with − symbol", () => {
    render(<FateDie face={-1} />)
    const die = screen.getByRole("img", { name: "Minus die" })
    expect(die).toBeInTheDocument()
    expect(die).toHaveTextContent("−")
  })

  it("renders a blank die with no symbol", () => {
    render(<FateDie face={0} />)
    const die = screen.getByRole("img", { name: "Blank die" })
    expect(die).toBeInTheDocument()
    expect(die).toHaveTextContent("")
  })

  it("shows hover tooltip with value for plus die", () => {
    render(<FateDie face={1} />)
    const die = screen.getByRole("img", { name: "Plus die" })
    expect(die).toHaveAttribute("title", "Plus (+1)")
  })

  it("shows hover tooltip with value for minus die", () => {
    render(<FateDie face={-1} />)
    const die = screen.getByRole("img", { name: "Minus die" })
    expect(die).toHaveAttribute("title", "Minus (-1)")
  })

  it("shows hover tooltip with value for blank die", () => {
    render(<FateDie face={0} />)
    const die = screen.getByRole("img", { name: "Blank die" })
    expect(die).toHaveAttribute("title", "Blank (0)")
  })

  it("applies green styling for plus die", () => {
    render(<FateDie face={1} />)
    const die = screen.getByRole("img", { name: "Plus die" })
    expect(die.className).toContain("die-plus")
  })

  it("applies red styling for minus die", () => {
    render(<FateDie face={-1} />)
    const die = screen.getByRole("img", { name: "Minus die" })
    expect(die.className).toContain("die-minus")
  })

  it("applies small size variant", () => {
    render(<FateDie face={1} size="sm" />)
    const die = screen.getByRole("img", { name: "Plus die" })
    expect(die.className).toContain("h-6")
    expect(die.className).toContain("w-6")
  })

  it("applies medium size variant by default", () => {
    render(<FateDie face={1} />)
    const die = screen.getByRole("img", { name: "Plus die" })
    expect(die.className).toContain("h-8")
    expect(die.className).toContain("w-8")
  })

  it("merges custom className", () => {
    render(<FateDie face={0} className="my-custom-class" />)
    const die = screen.getByRole("img", { name: "Blank die" })
    expect(die.className).toContain("my-custom-class")
  })
})
