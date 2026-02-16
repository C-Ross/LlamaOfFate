import { useState } from "react"
import { GameSidebar } from "@/components/game/GameSidebar"
import { ChatPanel } from "@/components/game/ChatPanel"
import { ChatInput } from "@/components/game/ChatInput"
import { useGameSocket } from "@/hooks/useGameSocket"
import { useGameState } from "@/hooks/useGameState"
import { Badge } from "@/components/ui/badge"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet"

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
    sendInput,
  } = useGameSocket(getWebSocketUrl())

  const gameState = useGameState(events)

  const inputDisabled = !isConnected || isPending || awaitingInvoke || awaitingMidFlow || gameOver

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
          {/* Mobile sidebar toggle */}
          <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
            <SheetTrigger asChild>
              <button
                className="ml-auto rounded-md p-2 text-muted-foreground hover:text-foreground lg:hidden"
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

        {/* Message area */}
        <ChatPanel events={events} isPending={isPending} className="flex-1" />

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
