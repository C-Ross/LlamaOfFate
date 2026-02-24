import { cn } from "@/lib/utils"
import type { ChallengeTaskInfo } from "@/lib/types"

interface ChallengeBannerProps {
  /** Whether a challenge is currently active. */
  active: boolean
  /** The tasks in the active challenge. */
  tasks?: ChallengeTaskInfo[]
  className?: string
}

/**
 * Persistent banner displayed at the top of the chat area during an active
 * challenge. Shows the remaining tasks and their status.
 *
 * Renders nothing when `active` is false.
 */
export function ChallengeBanner({ active, tasks = [], className }: ChallengeBannerProps) {
  if (!active) return null

  const pending = tasks.filter((t) => t.Status === "pending").length
  const total = tasks.length

  return (
    <div
      className={cn(
        "bg-primary/10 border-b border-primary/30 px-4 py-2",
        className,
      )}
      role="alert"
      aria-label="Challenge in progress"
    >
      <div className="flex items-center justify-center gap-2 text-xs font-heading uppercase tracking-widest text-primary">
        <span className="inline-block h-2 w-2 rounded-full bg-primary animate-pulse" />
        Challenge In Progress ({total - pending}/{total})
        <span className="inline-block h-2 w-2 rounded-full bg-primary animate-pulse" />
      </div>
      {tasks.length > 0 && (
        <div className="mt-1 flex flex-wrap justify-center gap-2 text-xs font-body">
          {tasks.map((task) => (
            <span
              key={task.ID}
              className={cn(
                "rounded px-2 py-0.5",
                task.Status === "pending" && "bg-secondary text-secondary-foreground",
                (task.Status === "succeeded" || task.Status === "succeeded_with_style") &&
                  "bg-green-500/20 text-green-700 dark:text-green-400",
                task.Status === "failed" && "bg-destructive/20 text-destructive",
                task.Status === "tied" && "bg-yellow-500/20 text-yellow-700 dark:text-yellow-400",
              )}
            >
              {task.Skill}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
