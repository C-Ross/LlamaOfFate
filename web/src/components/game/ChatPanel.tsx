import { useEffect, useRef } from "react"
import { ChatMessage } from "@/components/game/ChatMessage"
import { CHAT_DISPLAYABLE_EVENTS, type GameEvent } from "@/lib/types"
import { cn } from "@/lib/utils"

interface ChatPanelProps {
  events: GameEvent[]
  isPending?: boolean
  className?: string
}

export function ChatPanel({ events, isPending = false, className }: ChatPanelProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  const displayable = events.filter((e) => CHAT_DISPLAYABLE_EVENTS.has(e.event))

  // Auto-scroll to bottom when new events arrive or pending state changes
  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: "smooth" })
  }, [displayable.length, isPending])

  return (
    <div className={cn("overflow-y-auto", className)}>
      <div className="mx-auto max-w-2xl space-y-4 px-6 py-4">
        {displayable.length === 0 && (
          <div className="rounded-lg bg-secondary/50 px-4 py-3 text-sm text-muted-foreground italic font-body">
            Waiting for the story to begin...
          </div>
        )}
        {displayable.map((event) => (
          <ChatMessage key={event.id} event={event} />
        ))}
        {isPending && (
          <div className="flex items-center gap-1.5 px-4 py-3 animate-in fade-in-0 duration-300">
            <span className="h-2 w-2 rounded-full bg-muted-foreground/60 animate-bounce [animation-delay:0ms]" />
            <span className="h-2 w-2 rounded-full bg-muted-foreground/60 animate-bounce [animation-delay:150ms]" />
            <span className="h-2 w-2 rounded-full bg-muted-foreground/60 animate-bounce [animation-delay:300ms]" />
          </div>
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
