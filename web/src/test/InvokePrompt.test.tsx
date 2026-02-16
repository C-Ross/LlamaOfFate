import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { InvokePrompt } from "@/components/game/InvokePrompt"
import type { InvokePromptEventData } from "@/lib/types"

function makeData(
  overrides: Partial<InvokePromptEventData> = {},
): InvokePromptEventData {
  return {
    Available: [
      { Name: "Quick Draw", Source: "character", FreeInvokes: 0 },
      { Name: "Dark Alley", Source: "situation", FreeInvokes: 1 },
    ],
    FatePoints: 3,
    CurrentResult: "Fair (+2)",
    ShiftsNeeded: 2,
    ...overrides,
  }
}

describe("InvokePrompt", () => {
  it("renders header text", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(screen.getByText("Invoke an Aspect?")).toBeInTheDocument()
  })

  it("renders current result and shifts needed", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(
      screen.getByText(/Current: Fair \(\+2\) · 2 shifts needed · 3 fate points/),
    ).toBeInTheDocument()
  })

  it("renders available aspects", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(screen.getByText("Quick Draw")).toBeInTheDocument()
    expect(screen.getByText("Dark Alley")).toBeInTheDocument()
  })

  it("shows free invokes on aspects that have them", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(screen.getByText(/1 free/)).toBeInTheDocument()
  })

  it("calls onInvoke with +2 when clicking +2 Bonus", () => {
    const onInvoke = vi.fn()
    render(
      <InvokePrompt data={makeData()} onInvoke={onInvoke} onDecline={vi.fn()} />,
    )
    const plus2Buttons = screen.getAllByText("+2 Bonus")
    fireEvent.click(plus2Buttons[0])
    expect(onInvoke).toHaveBeenCalledWith(0, false)
  })

  it("calls onInvoke with reroll when clicking Reroll", () => {
    const onInvoke = vi.fn()
    render(
      <InvokePrompt data={makeData()} onInvoke={onInvoke} onDecline={vi.fn()} />,
    )
    const rerollButtons = screen.getAllByText("Reroll")
    fireEvent.click(rerollButtons[1])
    expect(onInvoke).toHaveBeenCalledWith(1, true)
  })

  it("calls onDecline when clicking decline", () => {
    const onDecline = vi.fn()
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={onDecline} />,
    )
    fireEvent.click(screen.getByText(/Decline/))
    expect(onDecline).toHaveBeenCalledOnce()
  })

  it("shows already-used label for used aspects", () => {
    render(
      <InvokePrompt
        data={makeData({
          Available: [
            {
              Name: "Quick Draw",
              Source: "character",
              FreeInvokes: 0,
              AlreadyUsed: true,
            },
          ],
        })}
        onInvoke={vi.fn()}
        onDecline={vi.fn()}
      />,
    )
    expect(screen.getByText("Already used")).toBeInTheDocument()
  })

  it("shows not-enough-FP message when no fate points and no free invokes", () => {
    render(
      <InvokePrompt
        data={makeData({
          FatePoints: 0,
          Available: [
            { Name: "Quick Draw", Source: "character", FreeInvokes: 0 },
          ],
        })}
        onInvoke={vi.fn()}
        onDecline={vi.fn()}
      />,
    )
    expect(screen.getByText("Not enough fate points")).toBeInTheDocument()
  })

  it("has dialog role", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(
      screen.getByRole("dialog", { name: "Invoke an aspect" }),
    ).toBeInTheDocument()
  })

  it("shows resolving spinner after clicking +2 Bonus", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    fireEvent.click(screen.getAllByText("+2 Bonus")[0])
    expect(screen.getByText("Resolving...")).toBeInTheDocument()
    expect(screen.queryByText("+2 Bonus")).not.toBeInTheDocument()
  })

  it("shows resolving spinner after clicking Decline", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    fireEvent.click(screen.getByText(/Decline/))
    expect(screen.getByText("Resolving...")).toBeInTheDocument()
    expect(screen.queryByText(/Decline/)).not.toBeInTheDocument()
  })
})
