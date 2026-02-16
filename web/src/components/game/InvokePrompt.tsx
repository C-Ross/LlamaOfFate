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
 * Interactive invoke prompt overlay for aspect invocation during conflicts.
 *
 * Shows available aspects with clickable buttons. The player can choose to
 * invoke an aspect for a +2 bonus or a reroll, or decline the invocation.
 * Each aspect shows its source and any remaining free invokes.
 */
export function InvokePrompt({
  data,
  onInvoke,
  onDecline,
  className,
}: InvokePromptProps) {
  const [submitted, setSubmitted] = useState(false)

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

  return (
    <div
      className={cn(
        "rounded-lg border-2 border-fate-point/50 bg-card px-4 py-4 space-y-3 shadow-lg shadow-fate-point/10",
        className,
      )}
      role="dialog"
      aria-label="Invoke an aspect"
    >
      {/* Header */}
      <div className="space-y-1">
        <div className="font-heading text-sm font-bold text-fate-point">
          Invoke an Aspect?
        </div>
        <div className="text-xs font-body text-muted-foreground">
          Current: {data.CurrentResult} · {data.ShiftsNeeded} shift
          {data.ShiftsNeeded !== 1 ? "s" : ""} needed · {data.FatePoints} fate
          point{data.FatePoints !== 1 ? "s" : ""}
        </div>
      </div>

      {/* Submitted spinner */}
      {submitted ? (
        <div className="flex items-center justify-center gap-2 py-3 text-sm text-muted-foreground" aria-live="polite">
          <span className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
          Resolving...
        </div>
      ) : (
        <>
          {/* Aspect list */}
          <div className="space-y-2">
            {data.Available?.map((aspect, index) => (
              <AspectRow
                key={index}
                aspect={aspect}
                index={index}
                fatePoints={data.FatePoints}
                onInvoke={handleInvoke}
              />
            ))}
          </div>

          {/* Decline button */}
          <Button
            variant="outline"
            size="sm"
            className="w-full font-heading text-xs"
            onClick={handleDecline}
          >
            Decline — Keep Current Result
          </Button>
        </>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Aspect row sub-component
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
        "rounded-md border bg-secondary/50 px-3 py-2 space-y-1",
        alreadyUsed && "opacity-50",
        !canAfford && "opacity-50",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <div>
          <span className="font-bold text-sm">{aspect.Name}</span>
          <span className="text-xs text-muted-foreground ml-2">
            ({aspect.Source}
            {isFree ? `, ${aspect.FreeInvokes} free` : ""}
            {!isFree ? ", 1 FP" : ""})
          </span>
        </div>
      </div>
      {!alreadyUsed && canAfford && (
        <div className="flex gap-2">
          <Button
            variant="default"
            size="sm"
            className="flex-1 font-heading text-xs"
            onClick={handlePlus2}
          >
            +2 Bonus
          </Button>
          <Button
            variant="secondary"
            size="sm"
            className="flex-1 font-heading text-xs"
            onClick={handleReroll}
          >
            Reroll
          </Button>
        </div>
      )}
      {alreadyUsed && (
        <div className="text-xs text-muted-foreground italic">Already used</div>
      )}
      {!canAfford && !alreadyUsed && (
        <div className="text-xs text-destructive italic">Not enough fate points</div>
      )}
    </div>
  )
}
