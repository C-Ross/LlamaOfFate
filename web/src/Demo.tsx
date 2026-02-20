/**
 * Demo page for visually testing inline invoke & mid-flow prompts.
 * Access at /demo.html when running `just web-dev`.
 */
import { useState } from "react"
import { ChatPanel } from "@/components/game/ChatPanel"
import { ChatInput } from "@/components/game/ChatInput"
import { InvokePrompt } from "@/components/game/InvokePrompt"
import { MidFlowPrompt } from "@/components/game/MidFlowPrompt"
import { ConflictBanner } from "@/components/game/ConflictBanner"
import type { GameEvent, InvokePromptEventData, InputRequestEventData } from "@/lib/types"

// ---------------------------------------------------------------------------
// Mock data
// ---------------------------------------------------------------------------

let nextId = 1
function id() {
  return String(nextId++)
}

const mockEvents: GameEvent[] = [
  {
    id: id(),
    event: "narrative",
    data: {
      Text: "The Grand Saloon is thick with cigar smoke and the clink of whiskey glasses. A poker game is underway at the centre table, and all eyes turn to you as you push through the swinging doors.",
    },
  },
  {
    id: id(),
    event: "system_message",
    data: { Message: "Scene started: The Grand Saloon" },
  },
  {
    id: id(),
    event: "player_input",
    data: { text: "I stride up to the poker table and challenge the dealer to a game of wits." },
  },
  {
    id: id(),
    event: "dialog",
    data: {
      PlayerInput: "I stride up to the poker table and challenge the dealer to a game of wits.",
      GMResponse: "The dealer grins beneath his wide-brimmed hat, shuffling cards with practiced ease. \"You sure about that, stranger? Last fella who sat in that chair left with empty pockets and a bruised ego.\"",
    },
  },
  {
    id: id(),
    event: "conflict_start",
    data: {
      ConflictType: "mental",
      InitiatorName: "Dealer Graves",
      Participants: [
        { CharacterID: "player", CharacterName: "Jack Hartley", Initiative: 3, IsPlayer: true },
        { CharacterID: "npc-dealer", CharacterName: "Dealer Graves", Initiative: 2, IsPlayer: false },
      ],
    },
  },
  {
    id: id(),
    event: "turn_announcement",
    data: { CharacterName: "Jack Hartley", TurnNumber: 1, IsPlayer: true },
  },
  {
    id: id(),
    event: "player_input",
    data: { text: "I call his bluff with a confident smirk, using Deceive." },
  },
  {
    id: id(),
    event: "action_attempt",
    data: { Description: "Jack Hartley attempts to Overcome with Deceive against Dealer Graves." },
  },
  {
    id: id(),
    event: "action_result",
    data: {
      Skill: "Deceive",
      SkillRank: "Good",
      SkillBonus: 3,
      Bonuses: 0,
      Result: "Mediocre (+0)",
      Outcome: "Fail",
      DiceFaces: [-1, 0, -1, 1],
      Total: 2,
      TotalRank: "Fair",
      Difficulty: 4,
      DiffRank: "Great",
      DefenderName: "Dealer Graves",
    },
  },
  {
    id: id(),
    event: "invoke_prompt",
    data: {
      Available: [
        { Name: "Smooth-Talking Grifter", Source: "character", FreeInvokes: 0, AlreadyUsed: false },
        { Name: "I Never Lose Twice", Source: "character", FreeInvokes: 1, AlreadyUsed: false },
        { Name: "Sharp-Eyed and Cunning", Source: "situation", FreeInvokes: 0, AlreadyUsed: false },
        { Name: "Smoke-Filled Room", Source: "scene", FreeInvokes: 0, AlreadyUsed: true },
      ],
      FatePoints: 3,
      CurrentResult: "Fail",
      ShiftsNeeded: 2,
    } satisfies InvokePromptEventData,
  },
]

const mockMidFlowData: InputRequestEventData = {
  Type: "numbered_choice",
  Prompt: "Dealer Graves hits you with a cutting remark. You take 2 stress. Choose a consequence or absorb with stress:",
  Options: [
    { Label: "Absorb 2 stress", Description: "Mark the 2-stress box" },
    { Label: "Mild consequence", Description: "Take a mild consequence to absorb 2 shifts" },
  ],
}

// ---------------------------------------------------------------------------
// Demo shell
// ---------------------------------------------------------------------------

type DemoMode = "invoke" | "midflow" | "both" | "none"

export default function Demo() {
  const [mode, setMode] = useState<DemoMode>("invoke")
  const [invokeLog, setInvokeLog] = useState<string[]>([])

  const handleInvoke = (aspectIndex: number, isReroll: boolean) => {
    setInvokeLog((prev) => [...prev, `Invoked aspect ${aspectIndex}, reroll=${isReroll}`])
  }
  const handleDecline = () => {
    setInvokeLog((prev) => [...prev, "Declined invoke"])
  }
  const handleMidFlow = (choiceIndex: number, freeText?: string) => {
    setInvokeLog((prev) => [...prev, `Mid-flow choice ${choiceIndex}${freeText ? `: ${freeText}` : ""}`])
  }

  const invokeData = mockEvents.find((e) => e.event === "invoke_prompt")!.data as InvokePromptEventData

  const showInvoke = mode === "invoke" || mode === "both"
  const showMidFlow = mode === "midflow" || mode === "both"

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden bg-background text-foreground">
      {/* Demo toolbar */}
      <header className="flex items-center gap-3 border-b border-border px-6 py-3">
        <h1 className="text-lg font-heading font-bold tracking-widest uppercase">
          <span className="text-accent-foreground/60">Demo</span> — Inline Prompts
        </h1>
        <div className="flex gap-1.5 ml-auto">
          {(["invoke", "midflow", "both", "none"] as DemoMode[]).map((m) => (
            <button
              key={m}
              className={`rounded-md px-3 py-1 text-xs font-heading uppercase tracking-wide border ${
                mode === m
                  ? "bg-primary text-primary-foreground border-primary"
                  : "bg-secondary text-secondary-foreground border-border hover:bg-secondary/80"
              }`}
              onClick={() => setMode(m)}
            >
              {m}
            </button>
          ))}
        </div>
      </header>

      <ConflictBanner active />

      {/* Chat area */}
      <div className="relative flex-1 min-h-0">
        <ChatPanel
          events={mockEvents}
          isPending={false}
          className="h-full"
          invokeSlot={
            showInvoke ? (
              <InvokePrompt
                data={invokeData}
                onInvoke={handleInvoke}
                onDecline={handleDecline}
              />
            ) : undefined
          }
          midFlowSlot={
            showMidFlow ? (
              <MidFlowPrompt
                data={mockMidFlowData}
                onChoose={handleMidFlow}
              />
            ) : undefined
          }
        />
      </div>

      {/* Disabled input to show the disabled state */}
      <ChatInput
        onSend={() => {}}
        disabled={showInvoke || showMidFlow}
        placeholder={
          showInvoke
            ? "Awaiting invoke response..."
            : showMidFlow
              ? "Awaiting your choice..."
              : "What do you do?"
        }
      />

      {/* Action log */}
      {invokeLog.length > 0 && (
        <div className="border-t border-border bg-card px-6 py-2 text-xs font-mono text-muted-foreground max-h-24 overflow-y-auto">
          {invokeLog.map((entry, i) => (
            <div key={i}>→ {entry}</div>
          ))}
        </div>
      )}
    </div>
  )
}
