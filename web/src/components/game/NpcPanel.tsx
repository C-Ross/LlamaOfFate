import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { AspectBadge } from "@/components/game/AspectBadge"
import { cn } from "@/lib/utils"
import type { NPCSnapshot } from "@/lib/types"

interface NpcPanelProps {
  npcs: NPCSnapshot[]
  className?: string
}

export function NpcPanel({ npcs, className }: NpcPanelProps) {
  if (npcs.length === 0) {
    return (
      <p className={cn("text-sm text-muted-foreground font-body", className)}>
        No NPCs in scene
      </p>
    )
  }

  return (
    <div className={cn("space-y-1", className)}>
      {npcs.map((npc) => (
        <NpcEntry key={npc.name} npc={npc} />
      ))}
    </div>
  )
}

function NpcEntry({ npc }: { npc: NPCSnapshot }) {
  const hasDetails = npc.highConcept || (npc.aspects && npc.aspects.length > 0)

  if (!hasDetails) {
    return (
      <div className="flex items-center gap-2 py-1">
        <span className={cn("text-sm font-heading", npc.isTakenOut && "line-through text-muted-foreground")}>
          {npc.name}
        </span>
        {npc.isTakenOut && (
          <span className="text-xs text-destructive">(taken out)</span>
        )}
      </div>
    )
  }

  return (
    <Collapsible>
      <CollapsibleTrigger className="flex w-full items-center gap-2 py-1 text-left hover:bg-accent/50 rounded px-1 -mx-1 transition-colors">
        <svg
          className="h-3 w-3 text-muted-foreground transition-transform [[data-state=open]>svg&]:rotate-90"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="m9 5 7 7-7 7" />
        </svg>
        <span className={cn("text-sm font-heading", npc.isTakenOut && "line-through text-muted-foreground")}>
          {npc.name}
        </span>
        {npc.isTakenOut && (
          <span className="text-xs text-destructive">(taken out)</span>
        )}
      </CollapsibleTrigger>
      <CollapsibleContent className="pl-5 pb-1">
        <div className="flex flex-wrap gap-1">
          {npc.highConcept && (
            <AspectBadge name={npc.highConcept} kind="high-concept" />
          )}
          {npc.aspects?.map((a) => (
            <AspectBadge key={a} name={a} kind="general" />
          ))}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
