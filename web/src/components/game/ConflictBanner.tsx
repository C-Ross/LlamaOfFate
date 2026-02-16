import { cn } from "@/lib/utils"

interface ConflictBannerProps {
  /** Whether a conflict is currently active. */
  active: boolean
  className?: string
}

/**
 * Persistent banner displayed at the top of the chat area during an active
 * conflict. Provides a clear visual indicator that the game is in conflict
 * mode and mechanical actions apply.
 *
 * Renders nothing when `active` is false.
 */
export function ConflictBanner({ active, className }: ConflictBannerProps) {
  if (!active) return null

  return (
    <div
      className={cn(
        "flex items-center justify-center gap-2 bg-destructive/10 border-b border-destructive/30 px-4 py-2 text-xs font-heading uppercase tracking-widest text-destructive",
        className,
      )}
      role="alert"
      aria-label="Conflict in progress"
    >
      <span className="inline-block h-2 w-2 rounded-full bg-destructive animate-pulse" />
      Conflict In Progress
      <span className="inline-block h-2 w-2 rounded-full bg-destructive animate-pulse" />
    </div>
  )
}
