import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import App from "../App"

describe("App", () => {
  it("renders the title", () => {
    render(<App />)
    expect(screen.getByText("Llama")).toBeInTheDocument()
    expect(screen.getByText("Fate")).toBeInTheDocument()
  })

  it("renders the connection badge", () => {
    render(<App />)
    expect(screen.getByText("Not Connected")).toBeInTheDocument()
  })

  it("renders sidebar cards", () => {
    render(<App />)
    expect(screen.getByText("Character")).toBeInTheDocument()
    expect(screen.getByText("Situation Aspects")).toBeInTheDocument()
    expect(screen.getByText("Fate Points")).toBeInTheDocument()
  })

  it("renders the input form disabled", () => {
    render(<App />)
    const input = screen.getByPlaceholderText("What do you do?")
    expect(input).toBeDisabled()
    const button = screen.getByRole("button", { name: "Send" })
    expect(button).toBeDisabled()
  })
})
