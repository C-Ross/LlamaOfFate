import { useState, useMemo, useEffect, useRef } from "react"
import { toast } from "sonner"
import { GameSidebar } from "@/components/game/GameSidebar"
import { ChatPanel } from "@/components/game/ChatPanel"
import { ChatInput } from "@/components/game/ChatInput"
import { ConflictBanner } from "@/components/game/ConflictBanner"
import { InvokePrompt } from "@/components/game/InvokePrompt"
import { MidFlowPrompt } from "@/components/game/MidFlowPrompt"
import { SetupScreen } from "@/components/game/SetupScreen"
import { useGameSocket } from "@/hooks/useGameSocket"
import { useGameState } from "@/hooks/useGameState"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"
import type {
  InvokePromptEventData,
  InputRequestEventData,
  SetupRequestEventData,
  ErrorNotificationEventData,
} from "@/lib/types"

function getWebSocketUrl(): string {
  if (typeof window === "undefined") return "ws://localhost:8080/ws"
  // In dev mode, connect directly to the Go backend to avoid Vite proxy
  // EPIPE errors on WebSocket disconnect (tab close, page reload).
  // TODO: once Go serves static files in production, the else branch will
  // route through the same host automatically.
  if (import.meta.env.DEV) return "ws://localhost:8080/ws"
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:"
  return `${proto}//${window.location.host}/ws`
}

function App() {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const {
    events,
    isConnected,
    isPending,
    awaitingInvoke,
    awaitingMidFlow,
    gameOver,
    setupRequest,
    setupGeneratingMessage,
    sendInput,
    sendInvokeResponse,
    sendMidFlowResponse,
    sendSetupPreset,
    sendSetupCustom,
    newGame,
    continueGame,
    hasSavedGame,
  } = useGameSocket(getWebSocketUrl())

  const gameState = useGameState(events)

  // Find the most recent invoke prompt data when awaiting an invoke response
  const invokePromptData = useMemo(() => {
    if (!awaitingInvoke) return null
    for (let i = events.length - 1; i >= 0; i--) {
      if (events[i].event === "invoke_prompt") {
        return events[i].data as InvokePromptEventData
      }
    }
    return null
  }, [events, awaitingInvoke])

  // Track the event ID of the latest invoke prompt so the InvokePrompt
  // component remounts (resetting its local "submitted" state) when a new
  // prompt arrives after an invoke that leaves more invokes available.
  const invokePromptKey = useMemo(() => {
    for (let i = events.length - 1; i >= 0; i--) {
      if (events[i].event === "invoke_prompt") return events[i].id
    }
    return "none"
  }, [events])

  // Find the most recent input request data when awaiting a mid-flow response
  const midFlowPromptData = useMemo(() => {
    if (!awaitingMidFlow) return null
    for (let i = events.length - 1; i >= 0; i--) {
      if (events[i].event === "input_request") {
        return events[i].data as InputRequestEventData
      }
    }
    return null
  }, [events, awaitingMidFlow])

  const inputDisabled = !isConnected || isPending || awaitingInvoke || awaitingMidFlow || gameOver

  // Show a toast for error_notification events (e.g. save load failures).
  const lastToastedIndex = useRef(-1)
  useEffect(() => {
    for (let i = lastToastedIndex.current + 1; i < events.length; i++) {
      if (events[i].event === "error_notification") {
        const data = events[i].data as ErrorNotificationEventData
        toast.error(data.Message)
      }
    }
    lastToastedIndex.current = events.length - 1
  }, [events])

  // If the server sent a setup_request, show the setup screen instead of the game.
  if (setupRequest || setupGeneratingMessage) {
    return (
      <SetupScreen
        presets={(setupRequest as SetupRequestEventData | null)?.presets ?? []}
        allowCustom={(setupRequest as SetupRequestEventData | null)?.allowCustom ?? false}
        generatingMessage={setupGeneratingMessage}
        onSelectPreset={sendSetupPreset}
        onSelectCustom={sendSetupCustom}
        hasSavedGame={hasSavedGame}
        onContinue={continueGame}
      />
    )
  }

  return (
    <div className="flex h-screen w-screen overflow-hidden">
      {/* Chat Panel — left side */}
      <div className="flex min-h-0 flex-1 flex-col">
        {/* Header */}
        <header className="flex items-center gap-3 border-b border-border px-6 py-4">
          <h1 className="text-2xl font-heading font-bold tracking-widest uppercase text-foreground">
            <span className="text-accent-foreground/60">Llama</span> of <span className="text-primary">Fate</span>
          </h1>
          <Badge
            variant="outline"
            className={isConnected ? "text-boost border-boost/50" : "text-muted-foreground"}
          >
            {isConnected ? "Connected" : "Not Connected"}
          </Badge>
          {isPending && (
            <span className="text-xs text-muted-foreground animate-pulse">Thinking...</span>
          )}
          {/* New Game button — visible once game is in progress or finished */}
          <Button
            variant="ghost"
            size="sm"
            className="ml-auto text-xs text-muted-foreground hover:text-foreground"
            onClick={newGame}
            data-testid="new-game-button"
          >
            New Game
          </Button>
          {/* Mobile sidebar toggle */}
          <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
            <SheetTrigger asChild>
              <button
                className="rounded-md p-2 text-muted-foreground hover:text-foreground lg:hidden"
                aria-label="Open game sidebar"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <rect width="18" height="18" x="3" y="3" rx="2" />
                  <path d="M15 3v18" />
                </svg>
              </button>
            </SheetTrigger>
            <SheetContent side="right" className="w-80 overflow-y-auto p-4">
              <SheetHeader>
                <SheetTitle className="text-sm font-heading uppercase tracking-wider">Game Info</SheetTitle>
              </SheetHeader>
              <GameSidebar state={gameState} />
            </SheetContent>
          </Sheet>
        </header>

        {/* Conflict banner */}
        <ConflictBanner active={gameState.inConflict} />

        {/* Message area — relative container for overlaid prompts */}
        <div className="relative flex-1 min-h-0">
          <ChatPanel
            events={events}
            isPending={isPending}
            className="h-full"
            invokeSlot={
              awaitingInvoke && invokePromptData ? (
                <InvokePrompt
                  key={invokePromptKey}
                  data={invokePromptData}
                  onInvoke={sendInvokeResponse}
                  onDecline={() => sendInvokeResponse(-1, false)}
                />
              ) : undefined
            }
            midFlowSlot={
              awaitingMidFlow && midFlowPromptData ? (
                <MidFlowPrompt
                  data={midFlowPromptData}
                  onChoose={sendMidFlowResponse}
                />
              ) : undefined
            }
          />
        </div>

        {/* Input area */}
        <ChatInput
          onSend={sendInput}
          disabled={inputDisabled}
          placeholder={
            gameOver
              ? "Game over"
              : awaitingInvoke
                ? "Awaiting invoke response..."
                : awaitingMidFlow
                  ? "Awaiting your choice..."
                  : "What do you do?"
          }
        />
      </div>

      {/* Sidebar — right side */}
      <aside className="hidden w-80 min-h-0 flex-col overflow-y-auto border-l border-border p-4 lg:flex"> 
        <GameSidebar state={gameState} />
      </aside>
    </div>
  )
}

export default App
