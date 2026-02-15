import { cn } from "@/lib/utils"

export type AspectKind =
  | "high-concept"
  | "trouble"
  | "general"
  | "situation"
  | "boost"

const kindStyles: Record<AspectKind, string> = {
  "high-concept": "bg-aspect-high-concept/15 text-aspect-high-concept border-aspect-high-concept/30",
  trouble: "bg-aspect-trouble/15 text-aspect-trouble border-aspect-trouble/30",
  general: "bg-aspect-general/15 text-aspect-general border-aspect-general/30",
  situation: "bg-aspect-situation/15 text-aspect-situation border-aspect-situation/30",
  boost: "bg-boost/15 text-boost border-boost/30",
}

interface AspectBadgeProps {
  name: string
  kind: AspectKind
  freeInvokes?: number
  className?: string
}

export function AspectBadge({ name, kind, freeInvokes, className }: AspectBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-xs font-heading font-medium",
        kindStyles[kind],
        className,
      )}
    >
      {name}
      {freeInvokes != null && freeInvokes > 0 && (
        <span className="ml-1 rounded-full bg-current/20 px-1.5 text-[10px]">
          {freeInvokes}
        </span>
      )}
    </span>
  )
}
