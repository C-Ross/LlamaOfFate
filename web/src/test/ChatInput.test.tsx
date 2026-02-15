import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { ChatInput } from "@/components/game/ChatInput"

describe("ChatInput", () => {
  it("renders input and send button", () => {
    render(<ChatInput onSend={vi.fn()} />)
    expect(screen.getByLabelText("Player input")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Send" })).toBeInTheDocument()
  })

  it("send button is disabled when input is empty", () => {
    render(<ChatInput onSend={vi.fn()} />)
    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled()
  })

  it("send button enables when text is entered", () => {
    render(<ChatInput onSend={vi.fn()} />)
    const input = screen.getByLabelText("Player input")
    fireEvent.change(input, { target: { value: "search the room" } })
    expect(screen.getByRole("button", { name: "Send" })).not.toBeDisabled()
  })

  it("calls onSend with trimmed text on submit", () => {
    const onSend = vi.fn()
    render(<ChatInput onSend={onSend} />)
    const input = screen.getByLabelText("Player input")
    fireEvent.change(input, { target: { value: "  search the room  " } })
    fireEvent.submit(input)
    expect(onSend).toHaveBeenCalledWith("search the room")
  })

  it("clears input after sending", () => {
    render(<ChatInput onSend={vi.fn()} />)
    const input = screen.getByLabelText("Player input") as HTMLInputElement
    fireEvent.change(input, { target: { value: "hello" } })
    fireEvent.submit(input)
    expect(input.value).toBe("")
  })

  it("does not call onSend when disabled", () => {
    const onSend = vi.fn()
    render(<ChatInput onSend={onSend} disabled />)
    const input = screen.getByLabelText("Player input")
    expect(input).toBeDisabled()
    // Even if somehow triggered, should not call onSend
    fireEvent.change(input, { target: { value: "test" } })
    fireEvent.submit(input)
    expect(onSend).not.toHaveBeenCalled()
  })

  it("does not send whitespace-only input", () => {
    const onSend = vi.fn()
    render(<ChatInput onSend={onSend} />)
    const input = screen.getByLabelText("Player input")
    fireEvent.change(input, { target: { value: "   " } })
    fireEvent.submit(input)
    expect(onSend).not.toHaveBeenCalled()
  })

  it("uses custom placeholder", () => {
    render(<ChatInput onSend={vi.fn()} placeholder="Game over" />)
    expect(screen.getByPlaceholderText("Game over")).toBeInTheDocument()
  })
})
