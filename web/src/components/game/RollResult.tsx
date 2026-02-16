import { cn } from "@/lib/utils"
import { FateDie } from "@/components/game/FateDie"
import { OutcomeBadge } from "@/components/game/OutcomeBadge"
import { parseDiceFaces, sumDice, type DieFace } from "@/lib/dice"
import type { ActionResultEventData } from "@/lib/types"

interface RollResultProps {
  data: ActionResultEventData
  className?: string
}

/** Format a signed number, e.g. 3 → "+3", -1 → "-1" */
function signed(n: number): string {
  return n >= 0 ? `+${n}` : `${n}`
}

/**
 * Displays a complete action result with visual Fate dice, skill info, and
 * outcome badge. Uses structured fields from the server for clean formatting;
 * falls back to parsing dice faces from the Result string for backward
 * compatibility.
 */
export function RollResult({ data, className }: RollResultProps) {
  // Prefer structured data from server, fall back to string parsing
  const diceFaces: DieFace[] | null =
    data.DiceFaces && data.DiceFaces.length === 4
      ? (data.DiceFaces as DieFace[])
      : parseDiceFaces(data.Result)

  // Build the "Total vs Difficulty" detail string from structured fields
  const hasStructured = data.TotalRank && data.DiffRank
  const detailText = hasStructured
    ? data.DefenderName
      ? `Total: ${data.TotalRank} ${signed(data.Total)} vs ${data.DefenderName}'s ${data.DiffRank} ${signed(data.Difficulty)}`
      : `Total: ${data.TotalRank} ${signed(data.Total)} vs ${data.DiffRank} ${signed(data.Difficulty)}`
    : null

  return (
    <div
      className={cn(
        "rounded-lg border border-primary/30 bg-card px-4 py-3 text-sm font-body space-y-2",
        className,
      )}
      role="region"
      aria-label={`Roll result: ${data.Skill} — ${data.Outcome}`}
    >
      {/* Header: skill name · ladder rank + bonus, then outcome badge */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <span className="font-heading text-xs uppercase tracking-wide text-muted-foreground">
          {data.Skill}
          {data.SkillRank && <> · {data.SkillRank} {signed(data.SkillBonus)}</>}
        </span>
        <OutcomeBadge outcome={data.Outcome} />
      </div>

      {/* Dice row + total */}
      {diceFaces ? (
        <div className="flex items-center gap-3 flex-wrap">
          {/* 4 Fate dice */}
          <div className="flex items-center gap-1" role="group" aria-label="Dice roll">
            {diceFaces.map((face, i) => (
              <FateDie key={i} face={face} />
            ))}
          </div>

          {/* Dice sum */}
          <span
            className="font-heading text-sm text-muted-foreground"
            title="Dice total"
          >
            = {signed(sumDice(diceFaces))}
          </span>

          {/* Detailed breakdown */}
          {detailText && (
            <span
              className="text-xs text-muted-foreground"
              title="Full result breakdown"
            >
              {detailText}
            </span>
          )}
        </div>
      ) : (
        /* Fallback: plain text result if dice can't be parsed */
        <div className="text-foreground">
          Roll: {data.Result}
        </div>
      )}

      {/* Bonus line */}
      {data.Bonuses > 0 && (
        <div className="text-xs text-primary">
          +{data.Bonuses} bonus from invokes
        </div>
      )}
    </div>
  )
}
