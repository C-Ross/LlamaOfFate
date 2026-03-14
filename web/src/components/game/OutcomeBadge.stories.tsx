import type { Meta, StoryObj } from "@storybook/react-vite"
import { OutcomeBadge } from "./OutcomeBadge"

const meta = {
  title: "Dice/OutcomeBadge",
  component: OutcomeBadge,
  argTypes: {
    outcome: {
      control: { type: "select" },
      options: ["Success with Style", "Success", "Tie", "Failure"],
    },
    size: {
      control: { type: "inline-radio" },
      options: ["sm", "md"],
    },
  },
  args: { outcome: "Success", size: "md" },
} satisfies Meta<typeof OutcomeBadge>

export default meta
type Story = StoryObj<typeof meta>

export const SuccessWithStyle: Story = { args: { outcome: "Success with Style" } }
export const Success: Story = { args: { outcome: "Success" } }
export const Tie: Story = { args: { outcome: "Tie" } }
export const Failure: Story = { args: { outcome: "Failure" } }

export const AllOutcomes: Story = {
  render: () => (
    <div className="flex items-center gap-3 flex-wrap">
      <OutcomeBadge outcome="Success with Style" />
      <OutcomeBadge outcome="Success" />
      <OutcomeBadge outcome="Tie" />
      <OutcomeBadge outcome="Failure" />
    </div>
  ),
}

export const SmallVariant: Story = {
  render: () => (
    <div className="flex items-center gap-3 flex-wrap">
      <OutcomeBadge outcome="Success with Style" size="sm" />
      <OutcomeBadge outcome="Success" size="sm" />
      <OutcomeBadge outcome="Tie" size="sm" />
      <OutcomeBadge outcome="Failure" size="sm" />
    </div>
  ),
}
