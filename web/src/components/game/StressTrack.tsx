import { cn } from "@/lib/utils"
import type { StressTrackSnapshot, ConsequenceSnapshotEntry } from "@/lib/types"

// ---------------------------------------------------------------------------
// Stress boxes
// ---------------------------------------------------------------------------

interface StressBoxesProps {
  label: string
  track: StressTrackSnapshot
}

function StressBoxes({ label, track }: StressBoxesProps) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs font-heading text-muted-foreground w-16 capitalize">
        {label}
      </span>
      <div className="flex gap-1">
        {track.boxes.map((filled, i) => (
          <div
            key={i}
            className={cn(
              "h-5 w-5 rounded border text-center text-[10px] leading-5 font-heading",
              filled
                ? "border-destructive bg-destructive/20 text-destructive"
                : "border-border bg-muted/50 text-muted-foreground",
            )}
          >
            {filled ? "X" : i + 1}
          </div>
        ))}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Consequence slot
// ---------------------------------------------------------------------------

const severityColor: Record<string, string> = {
  mild: "text-consequence-mild",
  moderate: "text-consequence-moderate",
  severe: "text-consequence-severe",
  extreme: "text-destructive",
}

const severityShifts: Record<string, string> = {
  mild: "2",
  moderate: "4",
  severe: "6",
  extreme: "8",
}

interface ConsequenceSlotProps {
  entry: ConsequenceSnapshotEntry
}

function ConsequenceSlot({ entry }: ConsequenceSlotProps) {
  const color = severityColor[entry.severity] ?? "text-muted-foreground"
  return (
    <div className="flex items-baseline gap-2 text-sm">
      <span className={cn("font-heading text-xs uppercase", color)}>
        {entry.severity} ({severityShifts[entry.severity] ?? "?"})
      </span>
      {entry.aspect ? (
        <span className="font-body italic">{entry.aspect}</span>
      ) : (
        <span className="text-muted-foreground font-body">—</span>
      )}
      {entry.recovering && (
        <span className="text-xs text-boost">(recovering)</span>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Combined component
// ---------------------------------------------------------------------------

interface StressTrackProps {
  stressTracks: Record<string, StressTrackSnapshot>
  consequences: ConsequenceSnapshotEntry[]
}

export function StressTrack({ stressTracks, consequences }: StressTrackProps) {
  const trackEntries = Object.entries(stressTracks)

  return (
    <div className="space-y-3">
      {/* Stress boxes */}
      {trackEntries.length > 0 ? (
        trackEntries.map(([name, track]) => (
          <StressBoxes key={name} label={name} track={track} />
        ))
      ) : (
        <p className="text-xs text-muted-foreground font-body">No stress tracks</p>
      )}

      {/* Consequences */}
      {consequences.length > 0 && (
        <div className="space-y-1 pt-1 border-t border-border">
          <span className="text-xs font-heading text-muted-foreground uppercase">
            Consequences
          </span>
          {consequences.map((c, i) => (
            <ConsequenceSlot key={i} entry={c} />
          ))}
        </div>
      )}
    </div>
  )
}
