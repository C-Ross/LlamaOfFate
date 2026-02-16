import { cn } from "@/lib/utils"
import { FateDie } from "@/components/game/FateDie"
import { parseDiceFaces, type DieFace } from "@/lib/dice"
import type { DefenseRollEventData } from "@/lib/types"

interface DefenseRollProps {
  data: DefenseRollEventData
  className?: string
}

/**
 * Renders a defense roll result with visual dice faces.
 * Prefers structured DiceFaces from the server when available;
 * falls back to parsing dice faces from the Result string for backward
 * compatibility.
 */
export function DefenseRoll({ data, className }: DefenseRollProps) {
  // Prefer structured data from server, fall back to string parsing
  const diceFaces: DieFace[] | null =
    data.DiceFaces && data.DiceFaces.length === 4
      ? (data.DiceFaces as DieFace[])
      : parseDiceFaces(data.Result)

  return (
    <div
      className={cn(
        "rounded-lg border border-border bg-card px-4 py-2 text-sm font-body",
        className,
      )}
      role="status"
      aria-label={`Defense roll: ${data.DefenderName} — ${data.Skill} — ${data.Result}`}
    >
      <span className="font-heading text-xs uppercase tracking-wide text-muted-foreground">
        Defense:{" "}
      </span>
      {data.DefenderName} rolls {data.Skill}:{" "}

      {diceFaces ? (
        <span className="inline-flex items-center gap-1 align-middle ml-1">
          {diceFaces.map((face, i) => (
            <FateDie key={i} face={face} size="sm" />
          ))}
          <span className="ml-1 text-muted-foreground">{data.Result}</span>
        </span>
      ) : (
        <span className="font-bold">{data.Result}</span>
      )}
    </div>
  )
}
