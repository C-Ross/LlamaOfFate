import { cn } from "@/lib/utils"
import type { ConflictEndEventData } from "@/lib/types"

interface ConflictEndProps {
  data: ConflictEndEventData
  className?: string
}

/**
 * Conflict resolution summary — displayed as a divider-style message
 * marking the end of a conflict encounter.
 */
export function ConflictEnd({ data, className }: ConflictEndProps) {
  return (
    <div
      className={cn(
        "my-3 flex items-center gap-2 text-xs font-heading uppercase tracking-widest text-muted-foreground",
        className,
      )}
      role="status"
      aria-label={`Conflict ended: ${data.Reason}`}
    >
      <span className="h-px flex-1 bg-border" />
      Conflict Ended: {data.Reason}
      <span className="h-px flex-1 bg-border" />
    </div>
  )
}
