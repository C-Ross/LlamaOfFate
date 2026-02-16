import { FateDie } from "@/components/game/FateDie"
import { OutcomeBadge } from "@/components/game/OutcomeBadge"
import { RollResult } from "@/components/game/RollResult"
import { ActionAttempt } from "@/components/game/ActionAttempt"
import { DefenseRoll } from "@/components/game/DefenseRoll"
import type { ActionResultEventData, DefenseRollEventData } from "@/lib/types"

const successResult: ActionResultEventData = {
  Skill: "Fight",
  SkillLevel: "Good (+3)",
  Bonuses: 0,
  Result: "[+][-][ ][+] (Total: Great (+4) vs Difficulty Fair (+2))",
  Outcome: "Success",
}

const styleResult: ActionResultEventData = {
  Skill: "Shoot",
  SkillLevel: "Great (+4)",
  Bonuses: 2,
  Result: "[+][+][+][ ] (Total: Legendary (+8) vs Difficulty Good (+3))",
  Outcome: "Success with Style",
}

const failureResult: ActionResultEventData = {
  Skill: "Athletics",
  SkillLevel: "Average (+1)",
  Bonuses: 0,
  Result: "[-][-][ ][-] (Total: Poor (-1) vs Difficulty Fair (+2))",
  Outcome: "Failure",
}

const tieResult: ActionResultEventData = {
  Skill: "Investigate",
  SkillLevel: "Fair (+2)",
  Bonuses: 0,
  Result: "[ ][ ][ ][ ] (Total: Fair (+2) vs Difficulty Fair (+2))",
  Outcome: "Tie",
}

const defenseData: DefenseRollEventData = {
  DefenderName: "Shadow Bandit",
  Skill: "Athletics",
  Result: "Good (+3)",
}

export function DiceDemo() {
  return (
    <div className="min-h-screen bg-background text-foreground p-8 space-y-8 max-w-2xl mx-auto">
      <h1 className="font-heading text-2xl font-bold">Dice Visualization Demo</h1>

      {/* Individual dice faces */}
      <section className="space-y-3">
        <h2 className="font-heading text-lg font-semibold">Fate Dice Faces</h2>
        <div className="flex items-center gap-3">
          <FateDie face={1} />
          <FateDie face={0} />
          <FateDie face={-1} />
          <span className="text-muted-foreground text-sm ml-2">Medium (default)</span>
        </div>
        <div className="flex items-center gap-3">
          <FateDie face={1} size="sm" />
          <FateDie face={0} size="sm" />
          <FateDie face={-1} size="sm" />
          <span className="text-muted-foreground text-sm ml-2">Small</span>
        </div>
      </section>

      {/* Outcome badges */}
      <section className="space-y-3">
        <h2 className="font-heading text-lg font-semibold">Outcome Badges</h2>
        <div className="flex items-center gap-3 flex-wrap">
          <OutcomeBadge outcome="Success with Style" />
          <OutcomeBadge outcome="Success" />
          <OutcomeBadge outcome="Tie" />
          <OutcomeBadge outcome="Failure" />
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <OutcomeBadge outcome="Success with Style" size="sm" />
          <OutcomeBadge outcome="Success" size="sm" />
          <OutcomeBadge outcome="Tie" size="sm" />
          <OutcomeBadge outcome="Failure" size="sm" />
        </div>
      </section>

      {/* Action attempt */}
      <section className="space-y-3">
        <h2 className="font-heading text-lg font-semibold">Action Attempt</h2>
        <ActionAttempt data={{ Description: "Jesse draws his six-shooter and fires at the bandit." }} />
      </section>

      {/* Roll results for each outcome */}
      <section className="space-y-3">
        <h2 className="font-heading text-lg font-semibold">Roll Results</h2>
        <RollResult data={styleResult} />
        <RollResult data={successResult} />
        <RollResult data={tieResult} />
        <RollResult data={failureResult} />
      </section>

      {/* Defense roll */}
      <section className="space-y-3">
        <h2 className="font-heading text-lg font-semibold">Defense Roll</h2>
        <DefenseRoll data={defenseData} />
      </section>
    </div>
  )
}
