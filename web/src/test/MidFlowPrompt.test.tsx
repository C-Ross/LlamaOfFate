import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { MidFlowPrompt } from "@/components/game/MidFlowPrompt"
import type { InputRequestEventData } from "@/lib/types"

function makeChoiceData(
  overrides: Partial<InputRequestEventData> = {},
): InputRequestEventData {
  return {
    Type: "numbered_choice",
    Prompt: "How do you absorb the damage?",
    Options: [
      { Label: "Take a mild consequence" },
      { Label: "Mark stress box 2", Description: "Physical stress" },
    ],
    ...overrides,
  }
}

function makeFreeTextData(
  overrides: Partial<InputRequestEventData> = {},
): InputRequestEventData {
  return {
    Type: "free_text",
    Prompt: "Describe your concession terms:",
    ...overrides,
  }
}

describe("MidFlowPrompt", () => {
  describe("numbered choices", () => {
    it("renders prompt text", () => {
      render(<MidFlowPrompt data={makeChoiceData()} onChoose={vi.fn()} />)
      expect(
        screen.getByText("How do you absorb the damage?"),
      ).toBeInTheDocument()
    })

    it("renders all options", () => {
      render(<MidFlowPrompt data={makeChoiceData()} onChoose={vi.fn()} />)
      expect(
        screen.getByText("Take a mild consequence"),
      ).toBeInTheDocument()
      expect(screen.getByText("Mark stress box 2")).toBeInTheDocument()
    })

    it("renders option descriptions", () => {
      render(<MidFlowPrompt data={makeChoiceData()} onChoose={vi.fn()} />)
      expect(screen.getByText("Physical stress")).toBeInTheDocument()
    })

    it("calls onChoose with index when clicking an option", () => {
      const onChoose = vi.fn()
      render(<MidFlowPrompt data={makeChoiceData()} onChoose={onChoose} />)
      fireEvent.click(screen.getByText("Take a mild consequence"))
      expect(onChoose).toHaveBeenCalledWith(0)
    })

    it("calls onChoose with correct index for second option", () => {
      const onChoose = vi.fn()
      render(<MidFlowPrompt data={makeChoiceData()} onChoose={onChoose} />)
      fireEvent.click(screen.getByText("Mark stress box 2"))
      expect(onChoose).toHaveBeenCalledWith(1)
    })

    it("renders numbered labels", () => {
      render(<MidFlowPrompt data={makeChoiceData()} onChoose={vi.fn()} />)
      expect(screen.getByText("1.")).toBeInTheDocument()
      expect(screen.getByText("2.")).toBeInTheDocument()
    })
  })

  describe("free text", () => {
    it("renders prompt text", () => {
      render(<MidFlowPrompt data={makeFreeTextData()} onChoose={vi.fn()} />)
      expect(
        screen.getByText("Describe your concession terms:"),
      ).toBeInTheDocument()
    })

    it("renders text input", () => {
      render(<MidFlowPrompt data={makeFreeTextData()} onChoose={vi.fn()} />)
      expect(
        screen.getByLabelText("Free text response"),
      ).toBeInTheDocument()
    })

    it("calls onChoose with text on submit", () => {
      const onChoose = vi.fn()
      render(<MidFlowPrompt data={makeFreeTextData()} onChoose={onChoose} />)
      const input = screen.getByLabelText("Free text response")
      fireEvent.change(input, { target: { value: "I surrender peacefully." } })
      fireEvent.click(screen.getByText("Submit"))
      expect(onChoose).toHaveBeenCalledWith(0, "I surrender peacefully.")
    })

    it("calls onChoose on Enter key", () => {
      const onChoose = vi.fn()
      render(<MidFlowPrompt data={makeFreeTextData()} onChoose={onChoose} />)
      const input = screen.getByLabelText("Free text response")
      fireEvent.change(input, { target: { value: "I give up" } })
      fireEvent.keyDown(input, { key: "Enter" })
      expect(onChoose).toHaveBeenCalledWith(0, "I give up")
    })

    it("disables submit when text is empty", () => {
      render(<MidFlowPrompt data={makeFreeTextData()} onChoose={vi.fn()} />)
      expect(screen.getByText("Submit")).toBeDisabled()
    })
  })

  it("has dialog role", () => {
    render(<MidFlowPrompt data={makeChoiceData()} onChoose={vi.fn()} />)
    expect(
      screen.getByRole("dialog", {
        name: "How do you absorb the damage?",
      }),
    ).toBeInTheDocument()
  })

  it("shows resolving spinner after clicking an option", () => {
    render(<MidFlowPrompt data={makeChoiceData()} onChoose={vi.fn()} />)
    fireEvent.click(screen.getByText("Take a mild consequence"))
    expect(screen.getByText("Resolving...")).toBeInTheDocument()
    expect(screen.queryByText("Take a mild consequence")).not.toBeInTheDocument()
  })

  it("shows resolving spinner after free text submit", () => {
    render(<MidFlowPrompt data={makeFreeTextData()} onChoose={vi.fn()} />)
    const input = screen.getByLabelText("Free text response")
    fireEvent.change(input, { target: { value: "I give up" } })
    fireEvent.click(screen.getByText("Submit"))
    expect(screen.getByText("Resolving...")).toBeInTheDocument()
    expect(screen.queryByText("Submit")).not.toBeInTheDocument()
  })
})
