import type { Meta, StoryObj } from "@storybook/react-vite"
import { InvokePrompt } from "./InvokePrompt"
import type { InvokePromptEventData } from "@/lib/types"

const withAspects: InvokePromptEventData = {
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

const meta = {
  title: "Game/InvokePrompt",
  component: InvokePrompt,
  args: {
    onInvoke: () => {},
    onDecline: () => {},
  },
} satisfies Meta<typeof InvokePrompt>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: { data: withAspects },
}

export const FreeInvokeAvailable: Story = {
  args: {
    data: {
      ...withAspects,
      Available: [
        { Name: "Detective's Eye", Source: "character", FreeInvokes: 2, AlreadyUsed: false },
        { Name: "Crime Scene", Source: "scene", FreeInvokes: 1, AlreadyUsed: false },
      ],
      CurrentResult: "Tie",
      ShiftsNeeded: 1,
    },
  },
}

export const NoFatePoints: Story = {
  args: {
    data: {
      ...withAspects,
      FatePoints: 0,
    },
  },
}
