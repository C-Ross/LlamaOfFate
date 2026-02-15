import { render, screen } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import App from "../App"

// Mock WebSocket — App creates a WebSocket on mount via useGameSocket
class MockWebSocket {
  static instances: MockWebSocket[] = []
  readyState = 0
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null

  constructor(_url: string) { // eslint-disable-line @typescript-eslint/no-unused-vars
    MockWebSocket.instances.push(this)
  }

  send = vi.fn()
  close = vi.fn()
}

beforeEach(() => {
  MockWebSocket.instances = []
  vi.stubGlobal("WebSocket", MockWebSocket)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("App", () => {
  it("renders the title", () => {
    render(<App />)
    expect(screen.getByText("Llama")).toBeInTheDocument()
    expect(screen.getByText("Fate")).toBeInTheDocument()
  })

  it("renders the connection badge as Not Connected initially", () => {
    render(<App />)
    expect(screen.getByText("Not Connected")).toBeInTheDocument()
  })

  it("renders sidebar cards", () => {
    render(<App />)
    expect(screen.getByText("Character")).toBeInTheDocument()
    expect(screen.getByText("Situation Aspects")).toBeInTheDocument()
    expect(screen.getByText("Fate Points")).toBeInTheDocument()
  })

  it("renders the input form disabled when not connected", () => {
    render(<App />)
    const input = screen.getByLabelText("Player input")
    expect(input).toBeDisabled()
    const button = screen.getByRole("button", { name: "Send" })
    expect(button).toBeDisabled()
  })
})
