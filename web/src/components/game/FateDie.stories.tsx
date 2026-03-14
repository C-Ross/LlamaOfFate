import type { Meta, StoryObj } from "@storybook/react-vite"
import { FateDie } from "./FateDie"

const meta = {
  title: "Dice/FateDie",
  component: FateDie,
  argTypes: {
    face: {
      control: { type: "inline-radio" },
      options: [-1, 0, 1],
    },
    size: {
      control: { type: "inline-radio" },
      options: ["sm", "md"],
    },
  },
  args: { face: 1, size: "md" },
} satisfies Meta<typeof FateDie>

export default meta
type Story = StoryObj<typeof meta>

export const Plus: Story = { args: { face: 1 } }
export const Blank: Story = { args: { face: 0 } }
export const Minus: Story = { args: { face: -1 } }

export const Small: Story = { args: { face: 1, size: "sm" } }

export const AllFaces: Story = {
  render: () => (
    <div className="flex items-center gap-3">
      <FateDie face={1} />
      <FateDie face={0} />
      <FateDie face={-1} />
    </div>
  ),
}
