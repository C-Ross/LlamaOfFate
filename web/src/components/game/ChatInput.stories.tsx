import type { Meta, StoryObj } from "@storybook/react-vite"
import { ChatInput } from "./ChatInput"

const meta = {
  title: "Game/ChatInput",
  component: ChatInput,
  args: {
    onSend: () => {},
  },
} satisfies Meta<typeof ChatInput>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}

export const Disabled: Story = {
  args: {
    disabled: true,
    placeholder: "Waiting for response…",
  },
}

export const WaitingForInvoke: Story = {
  args: {
    disabled: true,
    placeholder: "Choose whether to invoke an aspect first…",
  },
}
