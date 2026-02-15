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

  it("resets state on disconnect", () => {
    const { result } = renderHook(() => useGameSocket("ws://localhost/ws"))
    act(() => getLastWs().simulateOpen())
    act(() => {
      getLastWs().simulateMessage({ event: "narrative", data: { Text: "Hello" } })
    })
    expect(result.current.events).toHaveLength(1)

    act(() => getLastWs().simulateClose())
    expect(result.current.isConnected).toBe(false)
    expect(result.current.events).toHaveLength(0)
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
})
