/**
 * Demo page for visually testing challenge events and the ChallengeBanner.
 * Access at /challenge-demo.html when running `just web-dev`.
 *
 * Walk through a complete challenge lifecycle:
 *   1. Narrative setup → challenge triggered
 *   2. ChallengeStartEvent with task list
 *   3. Player attempts each task → task results appear one by one
 *   4. ChallengeCompleteEvent with final tally
 *
 * Use the toolbar buttons to step through stages or jump to a specific phase.
 */
import { useState, useCallback } from "react"
import { ChatPanel } from "@/components/game/ChatPanel"
import { ChatInput } from "@/components/game/ChatInput"
import { ChallengeBanner } from "@/components/game/ChallengeBanner"
import type {
  GameEvent,
  ChallengeStartEventData,
  ChallengeTaskResultEventData,
  ChallengeCompleteEventData,
  ChallengeTaskInfo,
} from "@/lib/types"

// ---------------------------------------------------------------------------
// Mock challenge data
// ---------------------------------------------------------------------------

let nextId = 1
function id() {
  return String(nextId++)
}

const challengeTasks: ChallengeTaskInfo[] = [
  { ID: "task-1", Description: "Scale the crumbling walls before they collapse", Skill: "Athletics", Difficulty: "Good (+3)", Status: "pending" },
  { ID: "task-2", Description: "Spot the structural weak points to find the safest path", Skill: "Notice", Difficulty: "Fair (+2)", Status: "pending" },
  { ID: "task-3", Description: "Disable the arcane ward blocking the exit", Skill: "Lore", Difficulty: "Great (+4)", Status: "pending" },
]

const prologueEvents: GameEvent[] = [
  {
    id: id(),
    event: "narrative",
    data: {
      SceneName: "The Crumbling Tower",
      Text: "Ancient stones groan around you. Dust cascades from the ceiling as cracks spider-web across the walls. The tower is collapsing — you need to get out, and fast.",
    },
  },
  {
    id: id(),
    event: "player_input",
    data: { text: "I try to find a way out before the whole place comes down!" },
  },
  {
    id: id(),
    event: "dialog",
    data: {
      PlayerInput: "I try to find a way out before the whole place comes down!",
      GMResponse: "The staircase behind you crumbles into rubble. There's a sealed door on the far wall, but you'll need to climb, navigate, and break through magical defenses to reach it. This won't be simple.",
    },
  },
]

const challengeStartEvent: GameEvent = {
  id: id(),
  event: "challenge_start",
  data: {
    Description: "Escape the collapsing tower before you're buried alive",
    Tasks: challengeTasks,
  } satisfies ChallengeStartEventData,
}

// Task 1: Athletics — Success with Style
const task1Events: GameEvent[] = [
  {
    id: id(),
    event: "player_input",
    data: { text: "I leap from ledge to ledge, using my Athletics to scale the walls." },
  },
  {
    id: id(),
    event: "action_attempt",
    data: { Description: "You attempt to Overcome with Athletics against Good (+3) difficulty." },
  },
  {
    id: id(),
    event: "action_result",
    data: {
      Skill: "Athletics",
      SkillRank: "Great",
      SkillBonus: 4,
      Bonuses: 0,
      Result: "Fantastic (+6)",
      Outcome: "Succeed with Style",
      DiceFaces: [1, 1, 0, 0],
      Total: 6,
      TotalRank: "Fantastic",
      Difficulty: 3,
      DiffRank: "Good",
    },
  },
  {
    id: id(),
    event: "dialog",
    data: {
      PlayerInput: "I leap from ledge to ledge, using my Athletics to scale the walls.",
      GMResponse: "You vault effortlessly between crumbling ledges, each jump perfectly timed as stone collapses behind you. A masterful display of agility — you reach the upper level with breath to spare.",
    },
  },
  {
    id: id(),
    event: "challenge_task_result",
    data: {
      TaskID: "task-1",
      Description: "Scale the crumbling walls before they collapse",
      Skill: "Athletics",
      Outcome: "succeeded_with_style",
      Shifts: 3,
    } satisfies ChallengeTaskResultEventData,
  },
]

// Task 2: Notice — Failure
const task2Events: GameEvent[] = [
  {
    id: id(),
    event: "player_input",
    data: { text: "I scan the structure for weak spots — where's the safest path forward?" },
  },
  {
    id: id(),
    event: "action_attempt",
    data: { Description: "You attempt to Overcome with Notice against Fair (+2) difficulty." },
  },
  {
    id: id(),
    event: "action_result",
    data: {
      Skill: "Notice",
      SkillRank: "Average",
      SkillBonus: 1,
      Bonuses: 0,
      Result: "Terrible (-2)",
      Outcome: "Fail",
      DiceFaces: [-1, -1, -1, 0],
      Total: -2,
      TotalRank: "Terrible",
      Difficulty: 2,
      DiffRank: "Fair",
    },
  },
  {
    id: id(),
    event: "dialog",
    data: {
      PlayerInput: "I scan the structure for weak spots — where's the safest path forward?",
      GMResponse: "Dust clouds your vision and a falling beam barely misses you. You misjudge the path completely, stumbling into a dead-end alcove as more debris rains down around you.",
    },
  },
  {
    id: id(),
    event: "challenge_task_result",
    data: {
      TaskID: "task-2",
      Description: "Spot the structural weak points to find the safest path",
      Skill: "Notice",
      Outcome: "failed",
      Shifts: -4,
    } satisfies ChallengeTaskResultEventData,
  },
]

// Task 3: Lore — Tie
const task3Events: GameEvent[] = [
  {
    id: id(),
    event: "player_input",
    data: { text: "I examine the arcane ward and try to unravel the enchantment with my knowledge of the old tongues." },
  },
  {
    id: id(),
    event: "action_attempt",
    data: { Description: "You attempt to Overcome with Lore against Great (+4) difficulty." },
  },
  {
    id: id(),
    event: "action_result",
    data: {
      Skill: "Lore",
      SkillRank: "Great",
      SkillBonus: 4,
      Bonuses: 0,
      Result: "Great (+4)",
      Outcome: "Tie",
      DiceFaces: [0, 1, -1, 0],
      Total: 4,
      TotalRank: "Great",
      Difficulty: 4,
      DiffRank: "Great",
    },
  },
  {
    id: id(),
    event: "dialog",
    data: {
      PlayerInput: "I examine the arcane ward and try to unravel the enchantment.",
      GMResponse: "The ward resists your incantation, pushing back with equal force. You manage to weaken it enough to squeeze through the door — but the magical feedback leaves a ringing in your ears that won't fade.",
    },
  },
  {
    id: id(),
    event: "challenge_task_result",
    data: {
      TaskID: "task-3",
      Description: "Disable the arcane ward blocking the exit",
      Skill: "Lore",
      Outcome: "tied",
      Shifts: 0,
    } satisfies ChallengeTaskResultEventData,
  },
]

const challengeCompleteEvent: GameEvent = {
  id: id(),
  event: "challenge_complete",
  data: {
    Successes: 1,
    Failures: 1,
    Ties: 1,
    Overall: "partial",
    Narrative: "The tower shudders as the ward partially unravels — enough to weaken the barrier, but not enough to break it cleanly.",
  } satisfies ChallengeCompleteEventData,
}

const epilogueEvent: GameEvent = {
  id: id(),
  event: "dialog",
  data: {
    PlayerInput: "",
    GMResponse: "You tumble out of the tower just as the entire structure groans and collapses into a mountain of rubble and dust. You're alive — but battered, disoriented, and the ward's magical feedback has temporarily dulled your senses. A partial victory at best.",
  },
}

// ---------------------------------------------------------------------------
// Stage definitions
// ---------------------------------------------------------------------------

type StageKey = "prologue" | "start" | "task1" | "task2" | "task3" | "complete" | "epilogue"

interface Stage {
  label: string
  events: GameEvent[]
}

const stages: Record<StageKey, Stage> = {
  prologue: { label: "Prologue", events: prologueEvents },
  start: { label: "Challenge Start", events: [challengeStartEvent] },
  task1: { label: "Task 1 (Success)", events: task1Events },
  task2: { label: "Task 2 (Fail)", events: task2Events },
  task3: { label: "Task 3 (Tie)", events: task3Events },
  complete: { label: "Complete", events: [challengeCompleteEvent] },
  epilogue: { label: "Epilogue", events: [epilogueEvent] },
}

const stageOrder: StageKey[] = ["prologue", "start", "task1", "task2", "task3", "complete", "epilogue"]

// ---------------------------------------------------------------------------
// Banner state derivation — mirrors useGameState logic for the demo
// ---------------------------------------------------------------------------

function deriveBannerState(events: GameEvent[]): { active: boolean; tasks: ChallengeTaskInfo[] } {
  let active = false
  let tasks: ChallengeTaskInfo[] = []

  for (const evt of events) {
    if (evt.event === "challenge_start") {
      const d = evt.data as ChallengeStartEventData
      active = true
      tasks = d.Tasks?.map((t) => ({ ...t })) ?? []
    } else if (evt.event === "challenge_task_result") {
      const d = evt.data as ChallengeTaskResultEventData
      tasks = tasks.map((t) => (t.ID === d.TaskID ? { ...t, Status: d.Outcome } : t))
    } else if (evt.event === "challenge_complete") {
      active = false
      tasks = []
    }
  }

  return { active, tasks }
}

// ---------------------------------------------------------------------------
// Demo component
// ---------------------------------------------------------------------------

export default function ChallengeDemo() {
  const [currentStageIdx, setCurrentStageIdx] = useState(0)

  // Accumulate events up to the current stage
  const visibleEvents = stageOrder
    .slice(0, currentStageIdx + 1)
    .flatMap((key) => stages[key].events)

  const bannerState = deriveBannerState(visibleEvents)

  const advance = useCallback(() => {
    setCurrentStageIdx((prev) => Math.min(prev + 1, stageOrder.length - 1))
  }, [])

  const jumpTo = useCallback((idx: number) => {
    setCurrentStageIdx(idx)
  }, [])

  const reset = useCallback(() => {
    setCurrentStageIdx(0)
  }, [])

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden bg-background text-foreground">
      {/* Toolbar */}
      <header className="flex items-center gap-3 border-b border-border px-6 py-3">
        <h1 className="text-lg font-heading font-bold tracking-widest uppercase">
          <span className="text-accent-foreground/60">Demo</span> — Challenge
        </h1>
        <div className="flex gap-1.5 ml-auto flex-wrap">
          {stageOrder.map((key, idx) => (
            <button
              key={key}
              className={`rounded-md px-3 py-1 text-xs font-heading uppercase tracking-wide border transition-colors ${
                idx === currentStageIdx
                  ? "bg-primary text-primary-foreground border-primary"
                  : idx < currentStageIdx
                    ? "bg-primary/20 text-primary border-primary/30"
                    : "bg-secondary text-secondary-foreground border-border hover:bg-secondary/80"
              }`}
              onClick={() => jumpTo(idx)}
            >
              {stages[key].label}
            </button>
          ))}
          <button
            className="rounded-md px-3 py-1 text-xs font-heading uppercase tracking-wide border border-destructive/50 text-destructive hover:bg-destructive/10"
            onClick={reset}
          >
            Reset
          </button>
        </div>
      </header>

      {/* Challenge banner */}
      <ChallengeBanner active={bannerState.active} tasks={bannerState.tasks} />

      {/* Chat area */}
      <div className="relative flex-1 min-h-0">
        <ChatPanel events={visibleEvents} isPending={false} className="h-full" />
      </div>

      {/* Input — advance on send */}
      <ChatInput
        onSend={advance}
        disabled={currentStageIdx >= stageOrder.length - 1}
        placeholder={
          currentStageIdx >= stageOrder.length - 1
            ? "Challenge complete — use toolbar to reset"
            : "Press Enter or click Send to advance…"
        }
      />

      {/* Stage indicator */}
      <div className="border-t border-border bg-card px-6 py-2 text-xs font-mono text-muted-foreground flex items-center gap-4">
        <span>
          Stage {currentStageIdx + 1}/{stageOrder.length}: {stages[stageOrder[currentStageIdx]].label}
        </span>
        <span className="text-muted-foreground/50">|</span>
        <span>{visibleEvents.length} events rendered</span>
        {bannerState.active && (
          <>
            <span className="text-muted-foreground/50">|</span>
            <span className="text-primary">
              Challenge active — {bannerState.tasks.filter((t) => t.Status === "pending").length} tasks remaining
            </span>
          </>
        )}
      </div>
    </div>
  )
}
