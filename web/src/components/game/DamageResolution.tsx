import { cn } from "@/lib/utils"
import type { DamageResolutionEventData } from "@/lib/types"

interface DamageResolutionProps {
  data: DamageResolutionEventData
  className?: string
}

/**
 * Inline damage/stress/consequence display for conflict damage resolution.
 * Shows stress absorbed, consequences taken, and taken-out status.
 */
export function DamageResolution({ data, className }: DamageResolutionProps) {
  return (
    <div
      className={cn(
        "rounded-lg border border-border bg-card px-4 py-2 text-sm font-body space-y-1",
        className,
      )}
      role="status"
      aria-label={`Damage to ${data.TargetName}${data.TakenOut ? " — Taken Out" : ""}`}
    >
      <div className="font-heading text-xs uppercase tracking-wide text-muted-foreground">
        Damage to {data.TargetName}
      </div>
      {data.Absorbed && (
        <div>
          Stress: {data.Absorbed.Shifts} shift
          {data.Absorbed.Shifts !== 1 ? "s" : ""} ({data.Absorbed.TrackType}){" "}
          {data.Absorbed.TrackState}
        </div>
      )}
      {data.Consequence && (
        <div className={consequenceColor(data.Consequence.Severity)}>
          {data.Consequence.Severity} consequence: {data.Consequence.Aspect}
        </div>
      )}
      {data.RemainingAbsorbed && (
        <div>
          Remaining stress: {data.RemainingAbsorbed.Shifts} shift
          {data.RemainingAbsorbed.Shifts !== 1 ? "s" : ""} ({data.RemainingAbsorbed.TrackType}){" "}
          {data.RemainingAbsorbed.TrackState}
        </div>
      )}
      {data.TakenOut && (
        <div className="font-bold text-destructive">Taken Out!</div>
      )}
      {data.VictoryEnd && (
        <div className="font-heading text-xs uppercase text-primary">
          Victory — conflict ends!
        </div>
      )}
    </div>
  )
}

function consequenceColor(severity: string): string {
  switch (severity.toLowerCase()) {
    case "mild":
      return "text-consequence-mild"
    case "moderate":
      return "text-consequence-moderate"
    case "severe":
      return "text-consequence-severe"
    default:
      return "text-foreground"
  }
}
