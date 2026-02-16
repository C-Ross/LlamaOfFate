import { cn } from "@/lib/utils"
import { OutcomeBadge } from "@/components/game/OutcomeBadge"
import type { NPCAttackEventData } from "@/lib/types"

interface NPCActionProps {
  data: NPCAttackEventData
  className?: string
}

/**
 * Renders an NPC attack result with attack/defense skills, outcome badge,
 * and optional narrative text.
 */
export function NPCAction({ data, className }: NPCActionProps) {
  return (
    <div className={cn("space-y-2", className)}>
      <div
        className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm font-body space-y-1"
        role="status"
        aria-label={`${data.AttackerName} attacks ${data.TargetName} — ${data.FinalOutcome}`}
      >
        <div className="font-heading text-xs uppercase tracking-wide text-destructive">
          {data.AttackerName} attacks {data.TargetName}
        </div>
        <div>
          Attack: {data.AttackSkill} {data.AttackResult} vs Defense:{" "}
          {data.DefenseSkill} {data.DefenseResult}
        </div>
        <div>
          <OutcomeBadge outcome={data.FinalOutcome} size="sm" />
        </div>
      </div>
      {data.Narrative && (
        <div className="rounded-lg bg-secondary/50 px-4 py-3 text-sm font-body text-foreground leading-relaxed whitespace-pre-wrap">
          {data.Narrative}
        </div>
      )}
    </div>
  )
}
