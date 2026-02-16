import { cn } from "@/lib/utils"
import type { TurnAnnouncementEventData } from "@/lib/types"

interface TurnAnnouncementProps {
  data: TurnAnnouncementEventData
  className?: string
}

/**
 * Inline turn change marker for conflicts.
 * Shows turn number and character name with decorative divider lines.
 */
export function TurnAnnouncement({ data, className }: TurnAnnouncementProps) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 py-1 text-xs font-heading uppercase tracking-wide text-muted-foreground",
        className,
      )}
      role="status"
      aria-label={`Turn ${data.TurnNumber}: ${data.CharacterName}${data.IsPlayer ? " (You)" : ""}`}
    >
      <span className="h-px w-4 bg-border" />
      Turn {data.TurnNumber}: {data.CharacterName}
      {data.IsPlayer ? " (You)" : ""}
      <span className="h-px w-4 bg-border" />
    </div>
  )
}
