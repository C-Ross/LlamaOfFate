import { GameSidebar } from "@/components/game/GameSidebar"
import { ChatPanel } from "@/components/game/ChatPanel"
import { ChatInput } from "@/components/game/ChatInput"
import { useGameSocket } from "@/hooks/useGameSocket"
import { useGameState } from "@/hooks/useGameState"
import { Badge } from "@/components/ui/badge"

function getWebSocketUrl(): string {
  if (typeof window === "undefined") return "ws://localhost:8080/ws"
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:"
  return `${proto}//${window.location.host}/ws`
}

function App() {
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
        </header>

        {/* Message area */}
        <ChatPanel events={events} className="flex-1" />

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
