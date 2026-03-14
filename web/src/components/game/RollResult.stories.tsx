import type { Meta, StoryObj } from "@storybook/react-vite"
import { RollResult } from "./RollResult"
import type { ActionResultEventData } from "@/lib/types"

const successData: ActionResultEventData = {
  Skill: "Fight",
  SkillRank: "Good",
  SkillBonus: 3,
  Bonuses: 0,
  Result: "[+][-][ ][+] (Total: Great (+4) vs Difficulty Fair (+2))",
  Outcome: "Success",
  DiceFaces: [1, -1, 0, 1],
  Total: 4,
  TotalRank: "Great",
  Difficulty: 2,
  DiffRank: "Fair",
}

const styleData: ActionResultEventData = {
  Skill: "Shoot",
  SkillRank: "Great",
  SkillBonus: 4,
  Bonuses: 2,
  Result: "[+][+][+][ ] (Total: Legendary (+8) vs Difficulty Good (+3))",
  Outcome: "Success with Style",
  DiceFaces: [1, 1, 1, 0],
  Total: 8,
  TotalRank: "Legendary",
  Difficulty: 3,
  DiffRank: "Good",
}

const failureData: ActionResultEventData = {
  Skill: "Athletics",
  SkillRank: "Average",
  SkillBonus: 1,
  Bonuses: 0,
  Result: "[-][-][ ][-] (Total: Poor (-1) vs Difficulty Fair (+2))",
  Outcome: "Failure",
  DiceFaces: [-1, -1, 0, -1],
  Total: -1,
  TotalRank: "Poor",
  Difficulty: 2,
  DiffRank: "Fair",
}

const tieData: ActionResultEventData = {
  Skill: "Investigate",
  SkillRank: "Fair",
  SkillBonus: 2,
  Bonuses: 0,
  Result: "[ ][ ][ ][ ] (Total: Fair (+2) vs Difficulty Fair (+2))",
  Outcome: "Tie",
  DiceFaces: [0, 0, 0, 0],
  Total: 2,
  TotalRank: "Fair",
  Difficulty: 2,
  DiffRank: "Fair",
}

const vsDefenderData: ActionResultEventData = {
  Skill: "Provoke",
  SkillRank: "Average",
  SkillBonus: 1,
  Bonuses: 0,
  Result: "[+][ ][ ][+] (Total: Good (+3) vs Shadow Bandit's Defense Fair (+2))",
  Outcome: "Success",
  DiceFaces: [1, 0, 0, 1],
  Total: 3,
  TotalRank: "Good",
  Difficulty: 2,
  DiffRank: "Fair",
  DefenderName: "Shadow Bandit",
}

const meta = {
  title: "Dice/RollResult",
  component: RollResult,
} satisfies Meta<typeof RollResult>

export default meta
type Story = StoryObj<typeof meta>

export const Success: Story = { args: { data: successData } }
export const SuccessWithStyle: Story = { args: { data: styleData } }
export const Failure: Story = { args: { data: failureData } }
export const Tie: Story = { args: { data: tieData } }
export const VsDefender: Story = { args: { data: vsDefenderData } }

export const WithBonuses: Story = {
  args: {
    data: { ...styleData, Bonuses: 2 },
  },
}

export const AllOutcomes: Story = {
  render: () => (
    <div className="space-y-3 max-w-lg">
      <RollResult data={styleData} />
      <RollResult data={successData} />
      <RollResult data={tieData} />
      <RollResult data={failureData} />
    </div>
  ),
}
