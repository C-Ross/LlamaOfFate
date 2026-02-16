import { cn } from "@/lib/utils"
import { dieFaceLabel, type DieFace } from "@/lib/dice"

interface FateDieProps {
  /** Die face value: -1 (minus), 0 (blank), or +1 (plus). */
  face: DieFace
  /** Optional size variant. */
  size?: "sm" | "md"
  className?: string
}

/**
 * Renders a single Fate die face with the appropriate symbol and color.
 *
 * - Plus (+1): green with "+" symbol
 * - Minus (-1): red with "−" symbol
 * - Blank (0): neutral with empty face
 */
export function FateDie({ face, size = "md", className }: FateDieProps) {
  const label = dieFaceLabel(face)

  return (
    <span
      role="img"
      aria-label={`${label} die`}
      title={`${label} (${face > 0 ? "+" : ""}${face})`}
      className={cn(
        "inline-flex items-center justify-center rounded font-heading font-bold select-none border",
        size === "md" && "h-8 w-8 text-base",
        size === "sm" && "h-6 w-6 text-sm",
        faceStyles(face),
        className,
      )}
    >
      {faceSymbol(face)}
    </span>
  )
}

function faceSymbol(face: DieFace): string {
  switch (face) {
    case 1:
      return "+"
    case -1:
      return "\u2212" // minus sign (−)
    case 0:
      return ""
  }
}

function faceStyles(face: DieFace): string {
  switch (face) {
    case 1:
      return "bg-die-plus/15 text-die-plus border-die-plus/40"
    case -1:
      return "bg-die-minus/15 text-die-minus border-die-minus/40"
    case 0:
      return "bg-muted text-muted-foreground border-border"
  }
}
