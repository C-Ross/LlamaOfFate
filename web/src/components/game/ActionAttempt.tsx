import { cn } from "@/lib/utils"
import type { ActionAttemptEventData } from "@/lib/types"

interface ActionAttemptProps {
  data: ActionAttemptEventData
  className?: string
}

/**
 * Renders an action attempt — the player's declared intent before the dice roll.
 * Displayed as a subtle system-style card in the chat stream.
 */
export function ActionAttempt({ data, className }: ActionAttemptProps) {
  return (
    <div
      className={cn(
        "rounded-lg border border-border bg-card px-4 py-2 text-sm font-body text-foreground",
        className,
      )}
      role="status"
      aria-label={`Action attempt: ${data.Description}`}
    >
      <span className="font-heading text-xs uppercase tracking-wide text-muted-foreground">
        Action:{" "}
      </span>
      {data.Description}
    </div>
  )
}
