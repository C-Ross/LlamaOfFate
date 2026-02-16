import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { StressTrack } from "@/components/game/StressTrack"

describe("StressTrack", () => {
  it("renders stress boxes for each track", () => {
    render(
      <StressTrack
        stressTracks={{
          physical: { boxes: [true, false], maxBoxes: 2 },
          mental: { boxes: [false, false], maxBoxes: 2 },
        }}
        consequences={[]}
      />,
    )

    expect(screen.getByText("physical")).toBeInTheDocument()
    expect(screen.getByText("mental")).toBeInTheDocument()
  })

  it("renders filled and empty boxes", () => {
    const { container } = render(
      <StressTrack
        stressTracks={{
          physical: { boxes: [true, false, false], maxBoxes: 3 },
        }}
        consequences={[]}
      />,
    )

    // The first box should show X (filled), others show numbers
    const boxes = container.querySelectorAll(".h-5.w-5")
    expect(boxes).toHaveLength(3)
    expect(boxes[0].textContent).toBe("X")
    expect(boxes[1].textContent).toBe("2")
    expect(boxes[2].textContent).toBe("3")
  })

  it("shows empty state when no tracks", () => {
    render(<StressTrack stressTracks={{}} consequences={[]} />)
    expect(screen.getByText("No stress tracks")).toBeInTheDocument()
  })

  it("renders consequences", () => {
    render(
      <StressTrack
        stressTracks={{}}
        consequences={[
          { severity: "mild", aspect: "Bruised Ribs", recovering: false },
          { severity: "moderate", aspect: "Broken Arm", recovering: true },
        ]}
      />,
    )

    expect(screen.getByText("Bruised Ribs")).toBeInTheDocument()
    expect(screen.getByText("Broken Arm")).toBeInTheDocument()
    expect(screen.getByText("(recovering)")).toBeInTheDocument()
  })

  it("shows consequence severity labels", () => {
    render(
      <StressTrack
        stressTracks={{}}
        consequences={[
          { severity: "mild", aspect: "Scratch", recovering: false },
        ]}
      />,
    )

    expect(screen.getByText("mild (2)")).toBeInTheDocument()
  })
})
