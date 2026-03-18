import type { Meta, StoryObj } from "@storybook/react-vite"
import ChallengeDemo from "@/ChallengeDemo"

const meta = {
  title: "Walkthroughs/Challenge",
  component: ChallengeDemo,
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof ChallengeDemo>

export default meta
type Story = StoryObj<typeof meta>

/** Step through a complete challenge lifecycle using the toolbar buttons or Enter. */
export const Default: Story = {}
