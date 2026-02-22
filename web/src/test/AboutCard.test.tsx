import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { AboutCard } from "@/components/game/AboutCard"

describe("AboutCard", () => {
  it("renders collapsed by default", () => {
    render(<AboutCard />)
    expect(screen.getByText(/About/)).toBeInTheDocument()
    expect(screen.queryByText(/Evil Hat/)).not.toBeInTheDocument()
  })

  it("expands when clicked to show credit text", () => {
    render(<AboutCard />)
    fireEvent.click(screen.getByRole("button", { name: /about/i }))
    expect(screen.getByText(/Evil Hat Productions/)).toBeInTheDocument()
    expect(screen.getByText(/Llama of Fate v0.1.0/)).toBeInTheDocument()
    expect(
      screen.getByText(/Creative Commons Attribution 3.0 Unported license/),
    ).toBeInTheDocument()
  })

  it("links to the SRD", () => {
    render(<AboutCard />)
    fireEvent.click(screen.getByRole("button", { name: /about/i }))
    const srdLink = screen.getByRole("link", {
      name: /Fate Core System Reference Document/,
    })
    expect(srdLink).toHaveAttribute("href", "https://fate-srd.com/")
  })

  it("links to the CC license", () => {
    render(<AboutCard />)
    fireEvent.click(screen.getByRole("button", { name: /about/i }))
    const licenseLink = screen.getByRole("link", {
      name: /Creative Commons/,
    })
    expect(licenseLink).toHaveAttribute(
      "href",
      "https://creativecommons.org/licenses/by/3.0/",
    )
  })

  it("links to the author GitHub profile", () => {
    render(<AboutCard />)
    fireEvent.click(screen.getByRole("button", { name: /about/i }))
    const authorLink = screen.getByRole("link", {
      name: /C\. Ross Eskridge/,
    })
    expect(authorLink).toHaveAttribute("href", "https://github.com/C-Ross")
  })
})
