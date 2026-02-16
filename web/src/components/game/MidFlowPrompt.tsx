import { useState, useCallback } from "react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import type { InputRequestEventData } from "@/lib/types"

interface MidFlowPromptProps {
  data: InputRequestEventData
  onChoose: (choiceIndex: number, freeText?: string) => void
  className?: string
}

/**
 * Interactive mid-flow prompt for consequence selection, taken-out decisions,
 * and other mid-action choices.
 *
 * For numbered choices: shows clickable option buttons.
 * For free text: shows a text input with submit.
 */
export function MidFlowPrompt({
  data,
  onChoose,
  className,
}: MidFlowPromptProps) {
  const [freeText, setFreeText] = useState("")
  const [submitted, setSubmitted] = useState(false)
  const isFreeText = data.Type === "free_text"

  const handleOptionClick = useCallback(
    (index: number) => {
      setSubmitted(true)
      onChoose(index)
    },
    [onChoose],
  )

  const handleFreeTextSubmit = useCallback(() => {
    const trimmed = freeText.trim()
    if (trimmed) {
      setSubmitted(true)
      onChoose(0, trimmed)
      setFreeText("")
    }
  }, [freeText, onChoose])

  return (
    <div
      className={cn(
        "rounded-lg border-2 border-primary/50 bg-card px-4 py-4 space-y-3 shadow-lg shadow-primary/10",
        className,
      )}
      role="dialog"
      aria-label={data.Prompt}
    >
      {/* Prompt text */}
      <div className="text-sm font-body text-foreground font-medium">
        {data.Prompt}
      </div>

      {/* Submitted spinner */}
      {submitted ? (
        <div className="flex items-center justify-center gap-2 py-3 text-sm text-muted-foreground" aria-live="polite">
          <span className="inline-block h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
          Resolving...
        </div>
      ) : (
        <>
          {/* Numbered choices */}
          {!isFreeText && data.Options && data.Options.length > 0 && (
            <div className="space-y-2">
              {data.Options.map((opt, i) => (
                <Button
                  key={i}
                  variant="outline"
                  className="w-full justify-start text-left font-body text-sm h-auto py-2 px-3"
                  onClick={() => handleOptionClick(i)}
                >
                  <span className="font-heading font-bold text-primary mr-2">
                    {i + 1}.
                  </span>
                  <span className="flex-1">
                    {opt.Label}
                    {opt.Description && (
                      <span className="block text-xs text-muted-foreground mt-0.5">
                        {opt.Description}
                      </span>
                    )}
                  </span>
                </Button>
              ))}
            </div>
          )}

          {/* Free text input */}
          {isFreeText && (
            <div className="flex gap-2">
              <input
                type="text"
                value={freeText}
                onChange={(e) => setFreeText(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault()
                    handleFreeTextSubmit()
                  }
                }}
                className="flex-1 rounded-md border border-input bg-input px-3 py-2 text-sm font-body ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                placeholder="Enter your response..."
                aria-label="Free text response"
                autoFocus
              />
              <Button
                onClick={handleFreeTextSubmit}
                disabled={!freeText.trim()}
                className="font-heading"
              >
                Submit
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
