import type { Meta, StoryObj } from "@storybook/react-vite"
import { DefenseRoll } from "./DefenseRoll"

const meta = {
  title: "Dice/DefenseRoll",
  component: DefenseRoll,
} satisfies Meta<typeof DefenseRoll>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    data: {
      DefenderName: "Shadow Bandit",
      Skill: "Athletics",
      Result: "Good (+3)",
    },
  },
}

export const WithDiceFaces: Story = {
  args: {
    data: {
      DefenderName: "Dealer Graves",
      Skill: "Deceive",
      Result: "[+][ ][-][+] Fair (+2)",
      DiceFaces: [1, 0, -1, 1],
    },
  },
}
