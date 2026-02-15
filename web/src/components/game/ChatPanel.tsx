import { useEffect, useRef } from "react"
import { ScrollArea } from "@/components/ui/scroll-area"
import { ChatMessage } from "@/components/game/ChatMessage"
import { CHAT_DISPLAYABLE_EVENTS, type GameEvent } from "@/lib/types"

interface ChatPanelProps {
  events: GameEvent[]
  className?: string
}

export function ChatPanel({ events, className }: ChatPanelProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  const displayable = events.filter((e) => CHAT_DISPLAYABLE_EVENTS.has(e.event))

  // Auto-scroll to bottom when new events arrive
  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: "smooth" })
  }, [displayable.length])

  return (
    <ScrollArea className={className}>
      <div className="mx-auto max-w-2xl space-y-4 px-6 py-4">
        {displayable.length === 0 && (
          <div className="rounded-lg bg-secondary/50 px-4 py-3 text-sm text-muted-foreground italic font-body">
            Waiting for the story to begin...
          </div>
        )}
        {displayable.map((event) => (
          <ChatMessage key={event.id} event={event} />
        ))}
        <div ref={bottomRef} />
      </div>
    </ScrollArea>
  )
}
