import { cn } from "@/lib/utils"

interface FatePointTrackerProps {
  current: number
  refresh?: number
  className?: string
}

export function FatePointTracker({ current, refresh, className }: FatePointTrackerProps) {
  return (
    <div className={cn("flex items-center gap-3", className)}>
      <span className="text-3xl font-bold text-fate-point font-heading">{current}</span>
      {refresh != null && (
        <span className="text-xs text-muted-foreground font-body">
          / {refresh} refresh
        </span>
      )}
    </div>
  )
}
