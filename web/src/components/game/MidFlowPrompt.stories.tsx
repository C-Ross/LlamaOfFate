import type { Meta, StoryObj } from "@storybook/react-vite"
import { MidFlowPrompt } from "./MidFlowPrompt"
import type { InputRequestEventData } from "@/lib/types"

const numberedChoice: InputRequestEventData = {
  Type: "numbered_choice",
  Prompt: "Dealer Graves hits you with a cutting remark. You take 2 stress. Choose a consequence or absorb with stress:",
  Options: [
    { Label: "Absorb 2 stress", Description: "Mark the 2-stress box" },
    { Label: "Mild consequence", Description: "Take a mild consequence to absorb 2 shifts" },
  ],
}

const freeText: InputRequestEventData = {
  Type: "free_text",
  Prompt: "The guards demand to know your business. What do you tell them?",
}

const manyOptions: InputRequestEventData = {
  Type: "numbered_choice",
  Prompt: "You have been taken out. What happens to your character?",
  Options: [
    { Label: "Captured", Description: "Enemies take you prisoner" },
    { Label: "Wounded and fleeing", Description: "Escape but mark a severe consequence" },
    { Label: "Knocked out", Description: "Wake up later at a safe location" },
    { Label: "Something worse…", Description: "Negotiate with the GM for a dramatic exit" },
  ],
}

const meta = {
  title: "Game/MidFlowPrompt",
  component: MidFlowPrompt,
  args: { onChoose: () => {} },
} satisfies Meta<typeof MidFlowPrompt>

export default meta
type Story = StoryObj<typeof meta>

export const NumberedChoice: Story = {
  args: { data: numberedChoice },
}

export const FreeText: Story = {
  args: { data: freeText },
}

export const ManyOptions: Story = {
  args: { data: manyOptions },
}
