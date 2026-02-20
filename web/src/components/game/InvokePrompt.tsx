import { useCallback, useState } from "react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import type { InvokePromptEventData, InvokableAspect } from "@/lib/types"

interface InvokePromptProps {
  data: InvokePromptEventData
  onInvoke: (aspectIndex: number, isReroll: boolean) => void
  onDecline: () => void
  className?: string
}

/**
 * Inline invoke prompt that lives in the chat flow rather than as a modal overlay.
 *
 * Starts collapsed showing a compact summary with an expand toggle. When expanded,
 * each aspect shows a single row with compact +2/Reroll buttons. This keeps the
 * chat scrollable so the player can see what they rolled.
 */
export function InvokePrompt({
  data,
  onInvoke,
  onDecline,
  className,
}: InvokePromptProps) {
  const [submitted, setSubmitted] = useState(false)
  const [expanded, setExpanded] = useState(false)

  const handleInvoke = useCallback(
    (aspectIndex: number, isReroll: boolean) => {
      setSubmitted(true)
      onInvoke(aspectIndex, isReroll)
    },
    [onInvoke],
  )

  const handleDecline = useCallback(() => {
    setSubmitted(true)
    onDecline()
  }, [onDecline])

  if (submitted) {
    return (
      <div
        className={cn(
          "rounded-lg border border-fate-point/30 bg-card px-4 py-3",
          className,
        )}
      >
        <div className="flex items-center justify-center gap-2 py-1 text-sm text-muted-foreground" aria-live="polite">
          <span className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
          Resolving...
        </div>
      </div>
    )
  }

  return (
    <div
      className={cn(
        "rounded-lg border-2 border-fate-point/50 bg-card px-4 py-3 shadow-sm",
        className,
      )}
      role="region"
      aria-label="Invoke an aspect"
    >
      {/* Collapsed summary — always visible */}
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-heading text-sm font-bold text-fate-point whitespace-nowrap">
            Invoke?
          </span>
          <span className="text-xs font-body text-muted-foreground truncate">
            {data.CurrentResult} · {data.ShiftsNeeded} shift
            {data.ShiftsNeeded !== 1 ? "s" : ""} needed · {data.FatePoints} FP
          </span>
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          {!expanded && (
            <Button
              variant="outline"
              size="sm"
              className="font-heading text-xs h-7 px-2"
              onClick={handleDecline}
            >
              Skip
            </Button>
          )}
          <Button
            variant={expanded ? "ghost" : "default"}
            size="sm"
            className="font-heading text-xs h-7 px-2"
            onClick={() => setExpanded((prev) => !prev)}
            aria-expanded={expanded}
            aria-controls="invoke-options"
          >
            {expanded ? "Collapse" : `${data.Available?.filter(a => !a.AlreadyUsed).length ?? 0} Aspects ▾`}
          </Button>
        </div>
      </div>

      {/* Expanded aspect list */}
      {expanded && (
        <div id="invoke-options" className="mt-3 space-y-1.5">
          {data.Available?.map((aspect, index) => (
            <AspectRow
              key={index}
              aspect={aspect}
              index={index}
              fatePoints={data.FatePoints}
              onInvoke={handleInvoke}
            />
          ))}
          <Button
            variant="outline"
            size="sm"
            className="w-full font-heading text-xs h-7 mt-1"
            onClick={handleDecline}
          >
            Decline — Keep Current Result
          </Button>
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Aspect row sub-component — compact single-line layout
// ---------------------------------------------------------------------------

interface AspectRowProps {
  aspect: InvokableAspect
  index: number
  fatePoints: number
  onInvoke: (aspectIndex: number, isReroll: boolean) => void
}

function AspectRow({ aspect, index, fatePoints, onInvoke }: AspectRowProps) {
  const isFree = aspect.FreeInvokes > 0
  const canAfford = isFree || fatePoints > 0
  const alreadyUsed = aspect.AlreadyUsed ?? false

  const handlePlus2 = useCallback(() => {
    if (canAfford && !alreadyUsed) onInvoke(index, false)
  }, [canAfford, alreadyUsed, index, onInvoke])

  const handleReroll = useCallback(() => {
    if (canAfford && !alreadyUsed) onInvoke(index, true)
  }, [canAfford, alreadyUsed, index, onInvoke])

  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-md border bg-secondary/50 px-3 py-1.5",
        (alreadyUsed || !canAfford) && "opacity-50",
      )}
    >
      {/* Aspect info */}
      <div
        className="flex-1 min-w-0"
        title={`${aspect.Source} aspect${isFree ? ` — ${aspect.FreeInvokes} free invoke${aspect.FreeInvokes !== 1 ? "s" : ""}` : " — costs 1 fate point"}`}
      >
        <span className="font-bold text-sm">{aspect.Name}</span>
        <span className="text-xs text-muted-foreground ml-1.5">
          {isFree ? `★${aspect.FreeInvokes} free` : "1 FP"}
        </span>
      </div>

      {/* Action buttons — compact inline */}
      {!alreadyUsed && canAfford && (
        <div className="flex gap-1 shrink-0">
          <Button
            variant="default"
            size="sm"
            className="font-heading text-xs h-6 px-2"
            onClick={handlePlus2}
          >
            +2
          </Button>
          <Button
            variant="secondary"
            size="sm"
            className="font-heading text-xs h-6 px-2"
            onClick={handleReroll}
          >
            Reroll
          </Button>
        </div>
      )}
      {alreadyUsed && (
        <span className="text-xs text-muted-foreground italic shrink-0">Used</span>
      )}
      {!canAfford && !alreadyUsed && (
        <span className="text-xs text-destructive italic shrink-0">No FP</span>
      )}
    </div>
  )
}
