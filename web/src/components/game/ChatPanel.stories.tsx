import type { Meta, StoryObj } from "@storybook/react-vite"
import { ChatPanel } from "./ChatPanel"
import { InvokePrompt } from "./InvokePrompt"
import { MidFlowPrompt } from "./MidFlowPrompt"
import type { GameEvent, InvokePromptEventData, InputRequestEventData } from "@/lib/types"

let nextId = 1
function id() { return String(nextId++) }

const saloonEvents: GameEvent[] = [
  {
    id: id(),
    event: "narrative",
    data: { Text: "The Grand Saloon is thick with cigar smoke and the clink of whiskey glasses. A poker game is underway at the centre table, and all eyes turn to you as you push through the swinging doors." },
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
]

const invokeData: InvokePromptEventData = {
  Available: [
    { Name: "Smooth-Talking Grifter", Source: "character", FreeInvokes: 0, AlreadyUsed: false },
    { Name: "I Never Lose Twice", Source: "character", FreeInvokes: 1, AlreadyUsed: false },
    { Name: "Sharp-Eyed and Cunning", Source: "situation", FreeInvokes: 0, AlreadyUsed: false },
    { Name: "Smoke-Filled Room", Source: "scene", FreeInvokes: 0, AlreadyUsed: true },
  ],
  FatePoints: 3,
  CurrentResult: "Fail",
  ShiftsNeeded: 2,
}

const midFlowData: InputRequestEventData = {
  Type: "numbered_choice",
  Prompt: "Dealer Graves hits you with a cutting remark. You take 2 stress. Choose a consequence or absorb with stress:",
  Options: [
    { Label: "Absorb 2 stress", Description: "Mark the 2-stress box" },
    { Label: "Mild consequence", Description: "Take a mild consequence to absorb 2 shifts" },
  ],
}

const meta = {
  title: "Game/ChatPanel",
  component: ChatPanel,
  decorators: [
    (Story) => (
      <div className="h-[600px] flex flex-col bg-background">
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof ChatPanel>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {
  args: { events: [], isPending: false },
}

export const WithEvents: Story = {
  args: { events: saloonEvents, isPending: false, className: "flex-1 min-h-0" },
}

export const Pending: Story = {
  args: { events: saloonEvents, isPending: true, className: "flex-1 min-h-0" },
}

export const WithInvokePrompt: Story = {
  args: {
    events: saloonEvents,
    isPending: false,
    className: "flex-1 min-h-0",
    invokeSlot: (
      <InvokePrompt
        data={invokeData}
        onInvoke={() => {}}
        onDecline={() => {}}
      />
    ),
  },
}

export const WithMidFlowPrompt: Story = {
  args: {
    events: saloonEvents,
    isPending: false,
    className: "flex-1 min-h-0",
    midFlowSlot: (
      <MidFlowPrompt
        data={midFlowData}
        onChoose={() => {}}
      />
    ),
  },
}
