import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { DamageResolution } from "@/components/game/DamageResolution"
import type { DamageResolutionEventData } from "@/lib/types"

function makeData(
  overrides: Partial<DamageResolutionEventData> = {},
): DamageResolutionEventData {
  return {
    TargetName: "Bandit",
    TakenOut: false,
    VictoryEnd: false,
    ...overrides,
  }
}

describe("DamageResolution", () => {
  it("renders target name", () => {
    render(<DamageResolution data={makeData()} />)
    expect(screen.getByText("Damage to Bandit")).toBeInTheDocument()
  })

  it("renders stress absorption", () => {
    render(
      <DamageResolution
        data={makeData({
          Absorbed: { TrackType: "physical", Shifts: 2, TrackState: "[X][X][ ]" },
        })}
      />,
    )
    expect(screen.getByText(/2 shifts \(physical\)/)).toBeInTheDocument()
  })

  it("renders singular shift", () => {
    render(
      <DamageResolution
        data={makeData({
          Absorbed: { TrackType: "mental", Shifts: 1, TrackState: "[X][ ]" },
        })}
      />,
    )
    expect(screen.getByText(/1 shift \(mental\)/)).toBeInTheDocument()
  })

  it("renders consequence", () => {
    render(
      <DamageResolution
        data={makeData({
          Consequence: { Severity: "Mild", Aspect: "Bruised Ribs", Absorbed: 2 },
        })}
      />,
    )
    expect(screen.getByText(/Mild consequence: Bruised Ribs/)).toBeInTheDocument()
  })

  it("renders taken out", () => {
    render(<DamageResolution data={makeData({ TakenOut: true })} />)
    expect(screen.getByText("Taken Out!")).toBeInTheDocument()
  })

  it("renders victory end", () => {
    render(
      <DamageResolution data={makeData({ TakenOut: true, VictoryEnd: true })} />,
    )
    expect(screen.getByText("Taken Out!")).toBeInTheDocument()
    expect(screen.getByText(/Victory — conflict ends!/)).toBeInTheDocument()
  })

  it("renders remaining stress absorption", () => {
    render(
      <DamageResolution
        data={makeData({
          RemainingAbsorbed: { TrackType: "physical", Shifts: 1, TrackState: "[X][ ]" },
        })}
      />,
    )
    expect(screen.getByText(/Remaining stress: 1 shift/)).toBeInTheDocument()
  })

  it("has accessible status role", () => {
    render(<DamageResolution data={makeData({ TakenOut: true })} />)
    expect(
      screen.getByRole("status", {
        name: "Damage to Bandit — Taken Out",
      }),
    ).toBeInTheDocument()
  })
})
