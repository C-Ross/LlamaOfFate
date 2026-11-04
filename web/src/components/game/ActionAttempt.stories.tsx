import type { Meta, StoryObj } from "@storybook/react-vite"
import { ActionAttempt } from "./ActionAttempt"

const meta = {
  title: "Dice/ActionAttempt",
  component: ActionAttempt,
} satisfies Meta<typeof ActionAttempt>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    data: { Description: "Jesse draws his six-shooter and fires at the bandit." },
  },
}

export const Overcome: Story = {
  args: {
    data: { Description: "You attempt to Overcome with Athletics against Good (+3) difficulty." },
  },
}
