import { render, screen, act, waitFor } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import App from "../App"
import { Toaster } from "@/components/ui/sonner"

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
    expect(screen.getByText("Stress & Consequences")).toBeInTheDocument()
    expect(screen.getByText("NPCs")).toBeInTheDocument()
  })

  it("renders the input form disabled when not connected", () => {
    render(<App />)
    const input = screen.getByLabelText("Player input")
    expect(input).toBeDisabled()
    const button = screen.getByRole("button", { name: "Send" })
    expect(button).toBeDisabled()
  })

  it("renders a mobile sidebar toggle button", () => {
    render(<App />)
    expect(screen.getByRole("button", { name: "Open game sidebar" })).toBeInTheDocument()
  })

  it("shows a toast when an error_notification event arrives", async () => {
    render(
      <>
        <App />
        <Toaster />
      </>
    )

    // Simulate WebSocket open + error_notification event
    const ws = MockWebSocket.instances[0]
    act(() => {
      ws.readyState = 1
      ws.onopen?.()
    })

    act(() => {
      ws.onmessage?.({
        data: JSON.stringify({
          event: "error_notification",
          data: { Message: "Your saved game could not be loaded and a new game has been started." },
        }),
      })
    })

    await waitFor(() => {
      expect(
        screen.getByText("Your saved game could not be loaded and a new game has been started.")
      ).toBeInTheDocument()
    })
  })
})
