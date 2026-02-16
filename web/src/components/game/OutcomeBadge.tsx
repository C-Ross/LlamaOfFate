import { cn } from "@/lib/utils"

interface OutcomeBadgeProps {
  /** Outcome string from the engine: "Failure", "Tie", "Success", "Success with Style". */
  outcome: string
  /** Optional size variant. */
  size?: "sm" | "md"
  className?: string
}

/**
 * Color-coded outcome badge for Fate action results.
 *
 * - Failure: red
 * - Tie: amber/yellow
 * - Success: green
 * - Success with Style: gold with subtle glow
 */
export function OutcomeBadge({ outcome, size = "md", className }: OutcomeBadgeProps) {
  const normalized = outcome.toLowerCase()

  return (
    <span
      role="status"
      aria-label={`Outcome: ${outcome}`}
      title={outcome}
      className={cn(
        "inline-flex items-center rounded-full font-heading font-bold uppercase tracking-wide",
        size === "md" && "px-3 py-0.5 text-xs",
        size === "sm" && "px-2 py-0.5 text-[10px]",
        badgeStyles(normalized),
        className,
      )}
    >
      {outcome}
    </span>
  )
}

function badgeStyles(outcome: string): string {
  if (outcome.includes("success with style")) {
    return "bg-outcome-style/20 text-outcome-style border border-outcome-style/40 shadow-[0_0_6px_0] shadow-outcome-style/25"
  }
  if (outcome.includes("success")) {
    return "bg-outcome-success/15 text-outcome-success border border-outcome-success/30"
  }
  if (outcome.includes("tie")) {
    return "bg-outcome-tie/15 text-outcome-tie border border-outcome-tie/30"
  }
  // failure
  return "bg-outcome-failure/15 text-outcome-failure border border-outcome-failure/30"
}
