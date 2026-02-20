import { useCallback, useEffect, useReducer, useRef } from "react"
import type {
  ClientMessage,
  CustomSetup,
  GameEvent,
  GameEventType,
  ResultMeta,
  SetupRequestEventData,
  SessionInitEventData,
  ServerMessage,
} from "@/lib/types"

const GAME_ID_STORAGE_KEY = "llamaoffate_game_id"

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
  gameId: string | null
  /** Non-null when the server has sent setup_request — player must pick a scenario. */
  setupRequest: SetupRequestEventData | null
  /** Non-null while LLM scenario generation is in progress. */
  setupGeneratingMessage: string | null
}

const initialState: GameSocketState = {
  events: [],
  isConnected: false,
  isPending: false,
  awaitingInvoke: false,
  awaitingMidFlow: false,
  gameOver: false,
  sceneEnded: false,
  gameId: null,
  setupRequest: null,
  setupGeneratingMessage: null,
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
  | { type: "session_init"; gameId: string }
  | { type: "setup_request"; data: SetupRequestEventData }
  | { type: "setup_generating"; message: string }
  | { type: "setup_complete" }
  | { type: "new_game" }

let nextEventId = 0

function reducer(state: GameSocketState, action: Action): GameSocketState {
  switch (action.type) {
    case "connected":
      return { ...state, isConnected: true }
    case "disconnected":
      // Preserve events and gameId across disconnects so the chat history
      // stays visible while the client reconnects to the same game.
      return {
        ...state,
        isConnected: false,
        isPending: false,
        awaitingInvoke: false,
        awaitingMidFlow: false,
      }
    case "session_init":
      return { ...state, gameId: action.gameId }
    case "setup_request":
      return { ...state, setupRequest: action.data, setupGeneratingMessage: null }
    case "setup_generating":
      return { ...state, setupGeneratingMessage: action.message }
    case "setup_complete":
      return { ...state, setupRequest: null, setupGeneratingMessage: null }
    case "new_game":
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
  sendSetupPreset: (presetId: string) => void
  sendSetupCustom: (custom: CustomSetup) => void
  /** Disconnect, clear stored game ID, and reconnect for a fresh setup flow. */
  newGame: () => void
}

export function useGameSocket(url: string): UseGameSocketReturn {
  const [state, dispatch] = useReducer(reducer, initialState)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  /** Track whether we're in setup mode inside the WS onmessage closure. */
  const inSetupRef = useRef(false)

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

  const sendSetupPreset = useCallback(
    (presetId: string) => send({ type: "setup", presetId }),
    [send],
  )

  const sendSetupCustom = useCallback(
    (custom: CustomSetup) => send({ type: "setup", custom }),
    [send],
  )

  // Store connect function so newGame can trigger a reconnect.
  const connectRef = useRef<(() => void) | null>(null)

  const newGame = useCallback(() => {
    // Clear stored game ID so the server creates a fresh session.
    localStorage.removeItem(GAME_ID_STORAGE_KEY)
    dispatch({ type: "new_game" })

    // Tear down the current WebSocket and reconnect without a game_id.
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current)
      reconnectTimer.current = null
    }
    const ws = wsRef.current
    if (ws) {
      ws.onclose = null // prevent auto-reconnect on this close
      ws.close()
      wsRef.current = null
    }
    inSetupRef.current = false
    connectRef.current?.()
  }, [])

  useEffect(() => {
    function connect() {
      // Append the game ID (from localStorage) so the server can resume the game.
      const storedGameId = localStorage.getItem(GAME_ID_STORAGE_KEY)
      let connectUrl = url
      if (storedGameId) {
        const sep = url.includes("?") ? "&" : "?"
        connectUrl = `${url}${sep}game_id=${encodeURIComponent(storedGameId)}`
      }

      const ws = new WebSocket(connectUrl)
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

          // Handle session_init: store the game ID so we can reconnect later.
          if (msg.event === "session_init") {
            const initData = msg.data as SessionInitEventData
            if (initData.gameId) {
              localStorage.setItem(GAME_ID_STORAGE_KEY, initData.gameId)
              dispatch({ type: "session_init", gameId: initData.gameId })
            }
            return
          }

          // Handle setup_request: server wants the player to pick a scenario.
          if (msg.event === "setup_request") {
            inSetupRef.current = true
            dispatch({ type: "setup_request", data: msg.data as SetupRequestEventData })
            return
          }

          // Handle setup_generating: LLM is building a custom scenario.
          if (msg.event === "setup_generating") {
            dispatch({ type: "setup_generating", message: (msg.data as { message: string }).message })
            return
          }

          // Once we receive a real game event, setup is done.
          if (inSetupRef.current) {
            inSetupRef.current = false
            dispatch({ type: "setup_complete" })
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
    connectRef.current = connect

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
    sendSetupPreset,
    sendSetupCustom,
    newGame,
  }
}
