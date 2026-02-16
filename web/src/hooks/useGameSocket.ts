import { useCallback, useEffect, useReducer, useRef } from "react"
import type {
  ClientMessage,
  GameEvent,
  GameEventType,
  ResultMeta,
  ServerMessage,
} from "@/lib/types"

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

export interface GameSocketState {
  events: GameEvent[]
  isConnected: boolean
  isPending: boolean
  awaitingInvoke: boolean
  awaitingMidFlow: boolean
  gameOver: boolean
  sceneEnded: boolean
}

const initialState: GameSocketState = {
  events: [],
  isConnected: false,
  isPending: false,
  awaitingInvoke: false,
  awaitingMidFlow: false,
  gameOver: false,
  sceneEnded: false,
}

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

type Action =
  | { type: "connected" }
  | { type: "disconnected" }
  | { type: "event"; event: GameEvent }
  | { type: "result_meta"; meta: ResultMeta }
  | { type: "send_pending" }

let nextEventId = 0

function reducer(state: GameSocketState, action: Action): GameSocketState {
  switch (action.type) {
    case "connected":
      return { ...state, isConnected: true }
    case "disconnected":
      return { ...initialState }
    case "event":
      return { ...state, events: [...state.events, action.event] }
    case "result_meta":
      return {
        ...state,
        isPending: false,
        awaitingInvoke: action.meta.awaitingInvoke,
        awaitingMidFlow: action.meta.awaitingMidFlow,
        gameOver: action.meta.gameOver,
        sceneEnded: action.meta.sceneEnded,
      }
    case "send_pending":
      return { ...state, isPending: true }
    default:
      return state
  }
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export interface UseGameSocketReturn extends GameSocketState {
  sendInput: (text: string) => void
  sendInvokeResponse: (aspectIndex: number, isReroll: boolean) => void
  sendMidFlowResponse: (choiceIndex: number, freeText?: string) => void
}

export function useGameSocket(url: string): UseGameSocketReturn {
  const [state, dispatch] = useReducer(reducer, initialState)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Stable send helper
  const send = useCallback((msg: ClientMessage) => {
    const ws = wsRef.current
    if (ws && ws.readyState === WebSocket.OPEN) {
      dispatch({ type: "send_pending" })
      ws.send(JSON.stringify(msg))
    }
  }, [])

  const sendInput = useCallback(
    (text: string) => {
      // Add the player's message to the event stream immediately so it
      // appears in chat before the server responds.
      const playerEvent: GameEvent = {
        id: `evt-${nextEventId++}`,
        event: "player_input",
        data: { text },
      }
      dispatch({ type: "event", event: playerEvent })
      send({ type: "input", text })
    },
    [send],
  )

  const sendInvokeResponse = useCallback(
    (aspectIndex: number, isReroll: boolean) =>
      send({ type: "invoke_response", aspectIndex, isReroll }),
    [send],
  )

  const sendMidFlowResponse = useCallback(
    (choiceIndex: number, freeText?: string) =>
      send({ type: "mid_flow_response", choiceIndex, freeText }),
    [send],
  )

  useEffect(() => {
    function connect() {
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        dispatch({ type: "connected" })
      }

      ws.onclose = () => {
        dispatch({ type: "disconnected" })
        wsRef.current = null
        // Auto-reconnect after 2 seconds (unless gameOver)
        reconnectTimer.current = setTimeout(() => {
          connect()
        }, 2000)
      }

      ws.onerror = () => {
        // onclose will fire after onerror
      }

      ws.onmessage = (e: MessageEvent) => {
        try {
          const msg = JSON.parse(e.data as string) as ServerMessage

          if (msg.event === "result_meta") {
            dispatch({ type: "result_meta", meta: msg.data as ResultMeta })
            return
          }

          const gameEvent: GameEvent = {
            id: `evt-${nextEventId++}`,
            event: msg.event as GameEventType,
            data: msg.data,
          }
          dispatch({ type: "event", event: gameEvent })
        } catch {
          // Ignore malformed messages
        }
      }
    }

    connect()

    return () => {
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current)
      }
      const ws = wsRef.current
      if (ws) {
        ws.onclose = null // prevent reconnect on intentional close
        ws.close()
        wsRef.current = null
      }
    }
  }, [url])

  return {
    ...state,
    sendInput,
    sendInvokeResponse,
    sendMidFlowResponse,
  }
}
