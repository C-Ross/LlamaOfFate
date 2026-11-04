import type { Meta, StoryObj } from "@storybook/react-vite"
import { ChallengeBanner } from "./ChallengeBanner"
import type { ChallengeTaskInfo } from "@/lib/types"

const tasks: ChallengeTaskInfo[] = [
  { ID: "task-1", Description: "Scale the crumbling walls before they collapse", Skill: "Athletics", Difficulty: "Good (+3)", Status: "pending" },
  { ID: "task-2", Description: "Spot the structural weak points to find the safest path", Skill: "Notice", Difficulty: "Fair (+2)", Status: "pending" },
  { ID: "task-3", Description: "Disable the arcane ward blocking the exit", Skill: "Lore", Difficulty: "Great (+4)", Status: "pending" },
]

const meta = {
  title: "Game/ChallengeBanner",
  component: ChallengeBanner,
  args: { active: true },
  argTypes: {
    active: { control: "boolean" },
  },
} satisfies Meta<typeof ChallengeBanner>

export default meta
type Story = StoryObj<typeof meta>

export const AllPending: Story = {
  args: { active: true, tasks },
}

export const MidProgress: Story = {
  args: {
    active: true,
    tasks: [
      { ...tasks[0], Status: "succeeded_with_style" },
      { ...tasks[1], Status: "failed" },
      { ...tasks[2], Status: "pending" },
    ],
  },
}

export const AllResolved: Story = {
  args: {
    active: true,
    tasks: [
      { ...tasks[0], Status: "succeeded_with_style" },
      { ...tasks[1], Status: "failed" },
      { ...tasks[2], Status: "tied" },
    ],
  },
}

export const Inactive: Story = {
  args: { active: false, tasks },
}
