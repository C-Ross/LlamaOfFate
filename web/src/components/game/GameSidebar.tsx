import { SidebarCard } from "@/components/SidebarCard"
import { AboutCard } from "@/components/game/AboutCard"
import { AspectBadge } from "@/components/game/AspectBadge"
import { StressTrack } from "@/components/game/StressTrack"
import { FatePointTracker } from "@/components/game/FatePointTracker"
import { NpcPanel } from "@/components/game/NpcPanel"
import { cn } from "@/lib/utils"
import type { GameState } from "@/hooks/useGameState"

interface GameSidebarProps {
  state: GameState
  className?: string
}

export function GameSidebar({ state, className }: GameSidebarProps) {
  const { player, situationAspects, npcs, fatePoints, stressTracks, consequences, inConflict } = state

  return (
    <div className={cn("space-y-4", className)}>
      {/* Character card */}
      <SidebarCard title={player?.name ?? "Character"}>
        {player ? (
          <div className="space-y-2">
            <div className="flex flex-wrap gap-1">
              {player.highConcept && (
                <AspectBadge name={player.highConcept} kind="high-concept" />
              )}
              {player.trouble && (
                <AspectBadge name={player.trouble} kind="trouble" />
              )}
              {player.aspects?.map((a) => (
                <AspectBadge key={a} name={a} kind="general" />
              ))}
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground font-body">
            No character loaded
          </p>
        )}
      </SidebarCard>

      {/* Situation aspects */}
      <SidebarCard title="Situation Aspects">
        {situationAspects.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {situationAspects.map((a, i) => (
              <AspectBadge
                key={`${a.name}-${i}`}
                name={a.name}
                kind={a.isBoost ? "boost" : "situation"}
                freeInvokes={a.freeInvokes}
              />
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground font-body">None active</p>
        )}
      </SidebarCard>

      {/* Fate points */}
      <SidebarCard title="Fate Points">
        <FatePointTracker current={fatePoints} refresh={player?.refresh} />
      </SidebarCard>

      {/* Stress & consequences */}
      <SidebarCard title="Stress & Consequences">
        <StressTrack stressTracks={stressTracks} consequences={consequences} />
      </SidebarCard>

      {/* NPCs */}
      <SidebarCard title={inConflict ? "Combatants" : "NPCs"}>
        <NpcPanel npcs={npcs} />
      </SidebarCard>

      {/* About / Credits */}
      <AboutCard />
    </div>
  )
}
