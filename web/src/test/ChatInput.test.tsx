import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { ChatInput } from "@/components/game/ChatInput"

describe("ChatInput", () => {
  it("renders textarea and send button", () => {
    render(<ChatInput onSend={vi.fn()} />)
    expect(screen.getByLabelText("Player input")).toBeInTheDocument()
    expect(screen.getByLabelText("Player input").tagName).toBe("TEXTAREA")
    expect(screen.getByRole("button", { name: "Send" })).toBeInTheDocument()
  })

  it("send button is disabled when input is empty", () => {
    render(<ChatInput onSend={vi.fn()} />)
    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled()
  })

  it("send button enables when text is entered", () => {
    render(<ChatInput onSend={vi.fn()} />)
    const textarea = screen.getByLabelText("Player input")
    fireEvent.change(textarea, { target: { value: "search the room" } })
    expect(screen.getByRole("button", { name: "Send" })).not.toBeDisabled()
  })

  it("calls onSend with trimmed text on submit", () => {
    const onSend = vi.fn()
    render(<ChatInput onSend={onSend} />)
    const textarea = screen.getByLabelText("Player input")
    fireEvent.change(textarea, { target: { value: "  search the room  " } })
    fireEvent.submit(textarea)
    expect(onSend).toHaveBeenCalledWith("search the room")
  })

  it("clears input after sending", () => {
    render(<ChatInput onSend={vi.fn()} />)
    const textarea = screen.getByLabelText("Player input") as HTMLTextAreaElement
    fireEvent.change(textarea, { target: { value: "hello" } })
    fireEvent.submit(textarea)
    expect(textarea.value).toBe("")
  })

  it("does not call onSend when disabled", () => {
    const onSend = vi.fn()
    render(<ChatInput onSend={onSend} disabled />)
    const textarea = screen.getByLabelText("Player input")
    expect(textarea).toBeDisabled()
    // Even if somehow triggered, should not call onSend
    fireEvent.change(textarea, { target: { value: "test" } })
    fireEvent.submit(textarea)
    expect(onSend).not.toHaveBeenCalled()
  })

  it("does not send whitespace-only input", () => {
    const onSend = vi.fn()
    render(<ChatInput onSend={onSend} />)
    const textarea = screen.getByLabelText("Player input")
    fireEvent.change(textarea, { target: { value: "   " } })
    fireEvent.submit(textarea)
    expect(onSend).not.toHaveBeenCalled()
  })

  it("uses custom placeholder", () => {
    render(<ChatInput onSend={vi.fn()} placeholder="Game over" />)
    expect(screen.getByPlaceholderText("Game over")).toBeInTheDocument()
  })

  it("submits on Enter key (without Shift)", () => {
    const onSend = vi.fn()
    render(<ChatInput onSend={onSend} />)
    const textarea = screen.getByLabelText("Player input")
    fireEvent.change(textarea, { target: { value: "I attack the dragon" } })
    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: false })
    expect(onSend).toHaveBeenCalledWith("I attack the dragon")
  })

  it("does not submit on Shift+Enter", () => {
    const onSend = vi.fn()
    render(<ChatInput onSend={onSend} />)
    const textarea = screen.getByLabelText("Player input")
    fireEvent.change(textarea, { target: { value: "line one" } })
    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: true })
    expect(onSend).not.toHaveBeenCalled()
  })

  it("focuses textarea when re-enabled after being disabled", () => {
    const { rerender } = render(<ChatInput onSend={vi.fn()} disabled />)
    const textarea = screen.getByLabelText("Player input")
    expect(textarea).not.toHaveFocus()
    rerender(<ChatInput onSend={vi.fn()} disabled={false} />)
    expect(textarea).toHaveFocus()
  })
})
