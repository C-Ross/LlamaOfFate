import { renderHook, act } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { useGameSocket } from "@/hooks/useGameSocket"

// Minimal WebSocket mock
class MockWebSocket {
  static instances: MockWebSocket[] = []
  static OPEN = 1

  readyState = 0
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  onmessage: ((e: { data: string }) => void) | null = null

  url: string

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }

  send = vi.fn()
  close = vi.fn()

  // Test helpers
  simulateOpen() {
    this.readyState = MockWebSocket.OPEN
    this.onopen?.()
  }

  simulateMessage(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) })
  }

  simulateClose() {
    this.readyState = 3
    this.onclose?.()
  }
}

beforeEach(() => {
  MockWebSocket.instances = []
  vi.stubGlobal("WebSocket", MockWebSocket)
  vi.useFakeTimers()
  localStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.useRealTimers()
})

function getLastWs(): MockWebSocket {
  return MockWebSocket.instances[MockWebSocket.instances.length - 1]
}

describe("useGameSocket", () => {
  it("starts disconnected", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    expect(result.current.isConnected).toBe(false)
    expect(result.current.events).toEqual([])
  })

  it("connects to the provided URL", () => {
    renderHook(() => useGameSocket("ws://test:8080/ws"))
    expect(getLastWs().url).toBe("ws://test:8080/ws")
  })

  it("sets isConnected on open", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())
    expect(result.current.isConnected).toBe(true)
  })

  it("accumulates game events", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => {
      getLastWs().simulateMessage({ event: "narrative", data: { Text: "Hello" } })
    })

    expect(result.current.events).toHaveLength(1)
    expect(result.current.events[0].event).toBe("narrative")
  })

  it("updates flags from result_meta", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => {
      getLastWs().simulateMessage({
        event: "result_meta",
        data: { awaitingInvoke: true, awaitingMidFlow: false, gameOver: false, sceneEnded: false },
      })
    })

    expect(result.current.awaitingInvoke).toBe(true)
    expect(result.current.awaitingMidFlow).toBe(false)
    expect(result.current.gameOver).toBe(false)
  })

  it("does not add result_meta to events array", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => {
      getLastWs().simulateMessage({
        event: "result_meta",
        data: { awaitingInvoke: false, awaitingMidFlow: false, gameOver: false, sceneEnded: false },
      })
    })

    expect(result.current.events).toHaveLength(0)
  })

  it("sendInput sends correct JSON", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => result.current.sendInput("I search the room"))

    expect(getLastWs().send).toHaveBeenCalledWith(
      JSON.stringify({ type: "input", text: "I search the room" }),
    )
  })

  it("sendInvokeResponse sends correct JSON", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => result.current.sendInvokeResponse(1, true))

    expect(getLastWs().send).toHaveBeenCalledWith(
      JSON.stringify({ type: "invoke_response", aspectIndex: 1, isReroll: true }),
    )
  })

  it("sendMidFlowResponse sends correct JSON", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => result.current.sendMidFlowResponse(0, "I surrender"))

    expect(getLastWs().send).toHaveBeenCalledWith(
      JSON.stringify({ type: "mid_flow_response", choiceIndex: 0, freeText: "I surrender" }),
    )
  })

  it("sets isPending after sending", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => result.current.sendInput("test"))
    expect(result.current.isPending).toBe(true)

    // Pending clears on result_meta
    act(() => {
      getLastWs().simulateMessage({
        event: "result_meta",
        data: { awaitingInvoke: false, awaitingMidFlow: false, gameOver: false, sceneEnded: false },
      })
    })
    expect(result.current.isPending).toBe(false)
  })

  it("preserves events on disconnect", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())
    act(() => {
      getLastWs().simulateMessage({ event: "narrative", data: { Text: "Hello" } })
    })
    expect(result.current.events).toHaveLength(1)

    act(() => getLastWs().simulateClose())
    expect(result.current.isConnected).toBe(false)
    // Events are preserved across disconnects so chat history stays visible
    expect(result.current.events).toHaveLength(1)
  })

  it("stores gameId from session_init in localStorage", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => {
      getLastWs().simulateMessage({
        event: "session_init",
        data: { gameId: "abc123" },
      })
    })

    expect(localStorage.getItem("llamaoffate_game_id")).toBe("abc123")
    expect(result.current.gameId).toBe("abc123")
  })

  it("does not add session_init to events array", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => {
      getLastWs().simulateMessage({
        event: "session_init",
        data: { gameId: "abc123" },
      })
    })

    expect(result.current.events).toHaveLength(0)
  })

  it("appends game_id to URL on reconnect when stored", () => {
    renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    // Simulate receiving a game ID
    act(() => {
      getLastWs().simulateMessage({
        event: "session_init",
        data: { gameId: "reconnect-id" },
      })
    })

    // Disconnect and reconnect
    act(() => getLastWs().simulateClose())
    act(() => vi.advanceTimersByTime(2500))

    const reconnectedWs = getLastWs()
    expect(reconnectedWs.url).toBe("ws://localhost/ws?game_id=reconnect-id")
  })

  it("attempts reconnection after disconnect", () => {
    renderHook(() => useGameSocket("ws://localhost/ws"))
    expect(MockWebSocket.instances).toHaveLength(1)

    act(() => getLastWs().simulateClose())

    // Advance timer past reconnect delay
    act(() => vi.advanceTimersByTime(2500))

    expect(MockWebSocket.instances).toHaveLength(2)
  })

  it("does not send when WebSocket is not open", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    // Don't call simulateOpen — readyState is 0

    act(() => result.current.sendInput("test"))
    expect(getLastWs().send).not.toHaveBeenCalled()
  })

  it("dispatches setup_request from server", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => {
      getLastWs().simulateMessage({
        event: "setup_request",
        data: {
          presets: [{ id: "saloon", title: "Trouble in Redemption Gulch", genre: "Western", description: "Outlaws." }],
          allowCustom: true,
        },
      })
    })

    expect(result.current.setupRequest).not.toBeNull()
    expect(result.current.setupRequest?.presets).toHaveLength(1)
    expect(result.current.setupRequest?.allowCustom).toBe(true)
    // setup_request should NOT be added to events
    expect(result.current.events).toHaveLength(0)
  })

  it("dispatches setup_generating from server", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => {
      getLastWs().simulateMessage({
        event: "setup_generating",
        data: { message: "Building world..." },
      })
    })

    expect(result.current.setupGeneratingMessage).toBe("Building world...")
  })

  it("clears setup state when first game event arrives", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    // Enter setup
    act(() => {
      getLastWs().simulateMessage({
        event: "setup_request",
        data: { presets: [], allowCustom: false },
      })
    })
    expect(result.current.setupRequest).not.toBeNull()

    // Receive a game event — setup should clear
    act(() => {
      getLastWs().simulateMessage({ event: "narrative", data: { Text: "Adventure!" } })
    })
    expect(result.current.setupRequest).toBeNull()
    expect(result.current.setupGeneratingMessage).toBeNull()
    expect(result.current.events).toHaveLength(1)
  })

  it("sendSetupPreset sends correct JSON", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => result.current.sendSetupPreset("heist"))

    expect(getLastWs().send).toHaveBeenCalledWith(
      JSON.stringify({ type: "setup", presetId: "heist" }),
    )
  })

  it("sendSetupCustom sends correct JSON", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    act(() => result.current.sendSetupCustom({
      name: "Ada", highConcept: "Hacker", trouble: "Paranoid", genre: "Cyberpunk",
    }))

    expect(getLastWs().send).toHaveBeenCalledWith(
      JSON.stringify({
        type: "setup",
        custom: { name: "Ada", highConcept: "Hacker", trouble: "Paranoid", genre: "Cyberpunk" },
      }),
    )
  })

  it("newGame clears state and reconnects without game_id", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    // Store a game_id
    act(() => {
      getLastWs().simulateMessage({ event: "session_init", data: { gameId: "old-game" } })
    })
    expect(localStorage.getItem("llamaoffate_game_id")).toBe("old-game")

    // Start new game
    act(() => result.current.newGame())

    expect(localStorage.getItem("llamaoffate_game_id")).toBeNull()
    expect(localStorage.getItem("llamaoffate_saved_game_id")).toBe("old-game")
    expect(result.current.events).toHaveLength(0)
    expect(result.current.gameId).toBeNull()
    expect(result.current.hasSavedGame).toBe(true)

    // A new WebSocket should have been created without game_id
    const newWs = getLastWs()
    expect(newWs.url).toBe("ws://localhost/ws")
  })

  it("continueGame reconnects with saved game_id", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    // Simulate having a saved game and starting new game
    act(() => {
      getLastWs().simulateMessage({ event: "session_init", data: { gameId: "saved-game" } })
    })
    act(() => result.current.newGame())

    expect(result.current.hasSavedGame).toBe(true)

    // Continue the saved game
    act(() => result.current.continueGame())

    expect(localStorage.getItem("llamaoffate_game_id")).toBe("saved-game")
    expect(localStorage.getItem("llamaoffate_saved_game_id")).toBeNull()
    expect(result.current.hasSavedGame).toBe(false)

    // A new WebSocket should have been created with the saved game_id
    const newWs = getLastWs()
    expect(newWs.url).toBe("ws://localhost/ws?game_id=saved-game")
  })

  it("continueGame is a no-op when no saved game exists", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())

    const wsBefore = getLastWs()
    act(() => result.current.continueGame())

    // No new WebSocket created — same instance
    expect(getLastWs()).toBe(wsBefore)
  })
})
