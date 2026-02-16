import { cn } from "@/lib/utils"
import { FateDie } from "@/components/game/FateDie"
import { OutcomeBadge } from "@/components/game/OutcomeBadge"
import { parseDiceFaces, parseResultDetail, sumDice, type DieFace } from "@/lib/dice"
import type { ActionResultEventData } from "@/lib/types"

interface RollResultProps {
  data: ActionResultEventData
  className?: string
}

/**
 * Displays a complete action result with visual Fate dice, skill info, and
 * outcome badge. Prefers structured DiceFaces from the server when available;
 * falls back to parsing dice faces from the Result string for backward
 * compatibility.
 */
export function RollResult({ data, className }: RollResultProps) {
  // Prefer structured data from server, fall back to string parsing
  const diceFaces: DieFace[] | null =
    data.DiceFaces && data.DiceFaces.length === 4
      ? (data.DiceFaces as DieFace[])
      : parseDiceFaces(data.Result)
  const detail = parseResultDetail(data.Result)

  return (
    <div
      className={cn(
        "rounded-lg border border-primary/30 bg-card px-4 py-3 text-sm font-body space-y-2",
        className,
      )}
      role="region"
      aria-label={`Roll result: ${data.Skill} — ${data.Outcome}`}
    >
      {/* Header: skill + outcome badge */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <span className="font-heading text-xs uppercase tracking-wide text-muted-foreground">
          {data.Skill} ({data.SkillLevel})
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
            = {sumDice(diceFaces) >= 0 ? "+" : ""}{sumDice(diceFaces)}
          </span>

          {/* Detailed breakdown (from parenthesized portion of Result) */}
          <span
            className="text-xs text-muted-foreground"
            title="Full result breakdown"
          >
            {detail}
          </span>
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
