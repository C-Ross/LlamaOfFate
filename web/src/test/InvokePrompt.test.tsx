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

/** Helper: expand the collapsed prompt to reveal aspect rows. */
function expandPrompt() {
  const expandBtn = screen.getByRole("button", { expanded: false })
  fireEvent.click(expandBtn)
}

describe("InvokePrompt", () => {
  it("renders compact header with invoke label", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(screen.getByText("Invoke?")).toBeInTheDocument()
  })

  it("shows current result and shifts in collapsed summary", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(
      screen.getByText(/Fair \(\+2\) · 2 shifts needed · 3 FP/),
    ).toBeInTheDocument()
  })

  it("starts collapsed — aspects not visible", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(screen.queryByText("Quick Draw")).not.toBeInTheDocument()
  })

  it("shows aspects after expanding", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expandPrompt()
    expect(screen.getByText("Quick Draw")).toBeInTheDocument()
    expect(screen.getByText("Dark Alley")).toBeInTheDocument()
  })

  it("shows free invokes on aspects that have them", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expandPrompt()
    expect(screen.getByText(/★1 free/)).toBeInTheDocument()
  })

  it("calls onInvoke with +2 when clicking +2", () => {
    const onInvoke = vi.fn()
    render(
      <InvokePrompt data={makeData()} onInvoke={onInvoke} onDecline={vi.fn()} />,
    )
    expandPrompt()
    const plus2Buttons = screen.getAllByText("+2")
    fireEvent.click(plus2Buttons[0])
    expect(onInvoke).toHaveBeenCalledWith(0, false)
  })

  it("calls onInvoke with reroll when clicking Reroll", () => {
    const onInvoke = vi.fn()
    render(
      <InvokePrompt data={makeData()} onInvoke={onInvoke} onDecline={vi.fn()} />,
    )
    expandPrompt()
    const rerollButtons = screen.getAllByText("Reroll")
    fireEvent.click(rerollButtons[1])
    expect(onInvoke).toHaveBeenCalledWith(1, true)
  })

  it("calls onDecline when clicking Skip in collapsed state", () => {
    const onDecline = vi.fn()
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={onDecline} />,
    )
    fireEvent.click(screen.getByText("Skip"))
    expect(onDecline).toHaveBeenCalledOnce()
  })

  it("calls onDecline when clicking Decline in expanded state", () => {
    const onDecline = vi.fn()
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={onDecline} />,
    )
    expandPrompt()
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
    expandPrompt()
    expect(screen.getByText("Used")).toBeInTheDocument()
  })

  it("shows no-FP message when no fate points and no free invokes", () => {
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
    expandPrompt()
    expect(screen.getByText("No FP")).toBeInTheDocument()
  })

  it("has region role", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(
      screen.getByRole("region", { name: "Invoke an aspect" }),
    ).toBeInTheDocument()
  })

  it("shows resolving spinner after clicking +2", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expandPrompt()
    fireEvent.click(screen.getAllByText("+2")[0])
    expect(screen.getByText("Resolving...")).toBeInTheDocument()
    expect(screen.queryByText("+2")).not.toBeInTheDocument()
  })

  it("shows resolving spinner after clicking Skip", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    fireEvent.click(screen.getByText("Skip"))
    expect(screen.getByText("Resolving...")).toBeInTheDocument()
    expect(screen.queryByText("Skip")).not.toBeInTheDocument()
  })

  it("shows aspect count on expand button", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expect(screen.getByText(/2 Aspects/)).toBeInTheDocument()
  })

  it("collapses when clicking Collapse", () => {
    render(
      <InvokePrompt data={makeData()} onInvoke={vi.fn()} onDecline={vi.fn()} />,
    )
    expandPrompt()
    expect(screen.getByText("Quick Draw")).toBeInTheDocument()
    fireEvent.click(screen.getByText("Collapse"))
    expect(screen.queryByText("Quick Draw")).not.toBeInTheDocument()
  })
})
