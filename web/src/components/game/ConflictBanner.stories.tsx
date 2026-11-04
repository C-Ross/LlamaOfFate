import type { Meta, StoryObj } from "@storybook/react-vite"
import { ConflictBanner } from "./ConflictBanner"

const meta = {
  title: "Game/ConflictBanner",
  component: ConflictBanner,
  args: { active: true },
  argTypes: {
    active: { control: "boolean" },
  },
} satisfies Meta<typeof ConflictBanner>

export default meta
type Story = StoryObj<typeof meta>

export const Active: Story = { args: { active: true } }
export const Inactive: Story = { args: { active: false } }
