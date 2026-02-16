import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { NPCAction } from "@/components/game/NPCAction"
import type { NPCAttackEventData } from "@/lib/types"

function makeData(
  overrides: Partial<NPCAttackEventData> = {},
): NPCAttackEventData {
  return {
    AttackerName: "Bandit",
    TargetName: "Jesse",
    AttackSkill: "Fight",
    AttackResult: "Good (+3)",
    DefenseSkill: "Athletics",
    DefenseResult: "Fair (+2)",
    FullDefense: false,
    InitialOutcome: "Success",
    FinalOutcome: "Success",
    Narrative: "",
    ...overrides,
  }
}

describe("NPCAction", () => {
  it("renders attacker and target", () => {
    render(<NPCAction data={makeData()} />)
    expect(screen.getByText("Bandit attacks Jesse")).toBeInTheDocument()
  })

  it("renders attack and defense details", () => {
    render(<NPCAction data={makeData()} />)
    expect(
      screen.getByText(/Attack: Fight Good \(\+3\) vs Defense: Athletics Fair \(\+2\)/),
    ).toBeInTheDocument()
  })

  it("renders outcome badge", () => {
    render(<NPCAction data={makeData({ FinalOutcome: "Success with Style" })} />)
    expect(screen.getByText("Success with Style")).toBeInTheDocument()
  })

  it("renders narrative when present", () => {
    render(
      <NPCAction
        data={makeData({ Narrative: "The bandit lunges forward with his blade." })}
      />,
    )
    expect(
      screen.getByText("The bandit lunges forward with his blade."),
    ).toBeInTheDocument()
  })

  it("does not render narrative when empty", () => {
    render(<NPCAction data={makeData({ Narrative: "" })} />)
    const narrativeBlocks = screen.queryAllByText(
      "The bandit lunges forward with his blade.",
    )
    expect(narrativeBlocks).toHaveLength(0)
  })

  it("has accessible status role", () => {
    render(<NPCAction data={makeData()} />)
    expect(
      screen.getByRole("status", {
        name: "Bandit attacks Jesse — Success",
      }),
    ).toBeInTheDocument()
  })
})
