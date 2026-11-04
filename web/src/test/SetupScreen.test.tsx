import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { SetupScreen } from "@/components/game/SetupScreen"
import { getDefaultPyramid } from "@/lib/skills"
import type { ScenarioPreset } from "@/lib/types"

const presets: ScenarioPreset[] = [
  { id: "saloon", title: "Trouble in Redemption Gulch", genre: "Western", description: "Outlaws threaten a frontier town." },
  { id: "heist", title: "The Prometheus Job", genre: "Cyberpunk", description: "Extract a data core." },
  { id: "tower", title: "The Wizard's Tower", genre: "Fantasy", description: "Investigate a magical disturbance." },
]

describe("SetupScreen", () => {
  it("renders preset cards", () => {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={false}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
      />,
    )

    expect(screen.getByText("Trouble in Redemption Gulch")).toBeInTheDocument()
    expect(screen.getByText("The Prometheus Job")).toBeInTheDocument()
    expect(screen.getByText("The Wizard's Tower")).toBeInTheDocument()
  })

  it("shows genre badges", () => {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={false}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
      />,
    )

    expect(screen.getByText("Western")).toBeInTheDocument()
    expect(screen.getByText("Cyberpunk")).toBeInTheDocument()
    expect(screen.getByText("Fantasy")).toBeInTheDocument()
  })

  it("calls onSelectPreset when clicking a card", () => {
    const onSelectPreset = vi.fn()
    render(
      <SetupScreen
        presets={presets}
        allowCustom={false}
        generatingMessage={null}
        onSelectPreset={onSelectPreset}
        onSelectCustom={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByTestId("preset-heist"))
    expect(onSelectPreset).toHaveBeenCalledWith("heist")
  })

  it("shows Create Your Own button when allowCustom is true", () => {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={true}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
      />,
    )

    expect(screen.getByTestId("custom-button")).toBeInTheDocument()
  })

  it("does not show Create Your Own button when allowCustom is false", () => {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={false}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
      />,
    )

    expect(screen.queryByTestId("custom-button")).not.toBeInTheDocument()
  })

  it("switches to custom form and shows identity step", () => {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={true}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByTestId("custom-button"))
    expect(screen.getByTestId("setup-custom")).toBeInTheDocument()
    expect(screen.getByTestId("setup-custom")).toHaveAttribute("data-step", "identity")
    expect(screen.getByText("Step 1 of 3")).toBeInTheDocument()
  })

  it("disables Next button when identity form is incomplete", () => {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={true}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByTestId("custom-button"))
    expect(screen.getByTestId("next-to-skills")).toBeDisabled()
  })

  it("Back button in identity step returns to picker", () => {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={true}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByTestId("custom-button"))
    expect(screen.getByText("Back")).toBeInTheDocument()

    fireEvent.click(screen.getByText("Back"))
    expect(screen.getByTestId("setup-picker")).toBeInTheDocument()
  })

  it("shows generating spinner when generatingMessage is set", () => {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={true}
        generatingMessage="Generating your scenario..."
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
      />,
    )

    expect(screen.getByTestId("setup-generating")).toBeInTheDocument()
    expect(screen.getByText("Generating your scenario...")).toBeInTheDocument()
    // Preset cards should NOT be visible
    expect(screen.queryByTestId("setup-picker")).not.toBeInTheDocument()
  })

  it("shows Continue card when hasSavedGame is true", () => {
    const onContinue = vi.fn()
    render(
      <SetupScreen
        presets={presets}
        allowCustom={false}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
        hasSavedGame={true}
        onContinue={onContinue}
      />,
    )

    expect(screen.getByTestId("continue-game")).toBeInTheDocument()
    expect(screen.getByText("Continue Game")).toBeInTheDocument()
    fireEvent.click(screen.getByTestId("continue-game"))
    expect(onContinue).toHaveBeenCalled()
  })

  it("does not show Continue card when hasSavedGame is false", () => {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={false}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={vi.fn()}
        hasSavedGame={false}
      />,
    )

    expect(screen.queryByTestId("continue-game")).not.toBeInTheDocument()
  })

  // ---------------------------------------------------------------------------
  // Wizard navigation tests
  // ---------------------------------------------------------------------------

  function fillIdentity() {
    fireEvent.change(screen.getByPlaceholderText("e.g. Ada Lovelace"), { target: { value: "Ada" } })
    fireEvent.change(screen.getByPlaceholderText("e.g. Rogue AI Whisperer"), { target: { value: "Hacker Supreme" } })
    fireEvent.change(screen.getByPlaceholderText("e.g. Trusts Machines More Than People"), { target: { value: "Too Curious" } })
    fireEvent.change(screen.getByPlaceholderText("e.g. Cyberpunk, Western, Fantasy"), { target: { value: "Cyberpunk" } })
  }

  function renderCustomWizard(onSelectCustom = vi.fn()) {
    render(
      <SetupScreen
        presets={presets}
        allowCustom={true}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={onSelectCustom}
      />,
    )
    fireEvent.click(screen.getByTestId("custom-button"))
  }

  it("navigates from identity to skills step", () => {
    renderCustomWizard()
    fillIdentity()

    fireEvent.click(screen.getByTestId("next-to-skills"))
    expect(screen.getByTestId("setup-custom")).toHaveAttribute("data-step", "skills")
    expect(screen.getByText("Step 2 of 3")).toBeInTheDocument()
    expect(screen.getByText("Skill Pyramid")).toBeInTheDocument()
  })

  it("shows additional aspect inputs on identity step", () => {
    renderCustomWizard()

    expect(screen.getByTestId("aspect-input-0")).toBeInTheDocument()
    expect(screen.getByTestId("aspect-input-1")).toBeInTheDocument()
    expect(screen.getByTestId("aspect-input-2")).toBeInTheDocument()
  })

  it("navigates back from skills to identity step", () => {
    renderCustomWizard()
    fillIdentity()
    fireEvent.click(screen.getByTestId("next-to-skills"))

    fireEvent.click(screen.getByTestId("back-to-identity"))
    expect(screen.getByTestId("setup-custom")).toHaveAttribute("data-step", "identity")
  })

  it("skills step starts with default pyramid pre-populated", () => {
    renderCustomWizard()
    fillIdentity()
    fireEvent.click(screen.getByTestId("next-to-skills"))

    // Next: Review should be enabled because defaults form a valid pyramid
    expect(screen.getByTestId("next-to-review")).not.toBeDisabled()
  })

  it("review step shows identity fields", () => {
    renderCustomWizard()
    fillIdentity()
    fireEvent.click(screen.getByTestId("next-to-skills"))
    fireEvent.click(screen.getByTestId("next-to-review"))

    expect(screen.getByText("Step 3 of 3")).toBeInTheDocument()
    expect(screen.getByText("Ada")).toBeInTheDocument()
    expect(screen.getByText("Hacker Supreme")).toBeInTheDocument()
    expect(screen.getByText("Too Curious")).toBeInTheDocument()
    expect(screen.getByText("Cyberpunk")).toBeInTheDocument()
  })

  it("review step shows default skills", () => {
    renderCustomWizard()
    fillIdentity()
    fireEvent.click(screen.getByTestId("next-to-skills"))
    fireEvent.click(screen.getByTestId("next-to-review"))

    expect(screen.getByText("Skills")).toBeInTheDocument()
    expect(screen.getByText(/Notice/)).toBeInTheDocument()
  })

  it("review step shows additional aspects when provided", () => {
    renderCustomWizard()
    fillIdentity()
    fireEvent.change(screen.getByTestId("aspect-input-0"), { target: { value: "Well Connected" } })
    fireEvent.click(screen.getByTestId("next-to-skills"))
    fireEvent.click(screen.getByTestId("next-to-review"))

    expect(screen.getByTestId("review-aspects")).toBeInTheDocument()
    expect(screen.getByText("Well Connected")).toBeInTheDocument()
  })

  it("navigates back from review to skills", () => {
    renderCustomWizard()
    fillIdentity()
    fireEvent.click(screen.getByTestId("next-to-skills"))
    fireEvent.click(screen.getByTestId("next-to-review"))

    fireEvent.click(screen.getByTestId("back-to-skills"))
    expect(screen.getByTestId("setup-custom")).toHaveAttribute("data-step", "skills")
  })

  it("submits from review with default skills", () => {
    const onSelectCustom = vi.fn()
    renderCustomWizard(onSelectCustom)
    fillIdentity()
    fireEvent.click(screen.getByTestId("next-to-skills"))
    fireEvent.click(screen.getByTestId("next-to-review"))

    fireEvent.click(screen.getByTestId("start-adventure"))
    expect(onSelectCustom).toHaveBeenCalledWith({
      name: "Ada",
      highConcept: "Hacker Supreme",
      trouble: "Too Curious",
      genre: "Cyberpunk",
      skills: getDefaultPyramid(),
    })
  })

  it("submits from review with aspects and default skills", () => {
    const onSelectCustom = vi.fn()
    renderCustomWizard(onSelectCustom)
    fillIdentity()
    fireEvent.change(screen.getByTestId("aspect-input-0"), { target: { value: "Well Connected" } })
    fireEvent.change(screen.getByTestId("aspect-input-2"), { target: { value: "Never Backs Down" } })
    fireEvent.click(screen.getByTestId("next-to-skills"))
    fireEvent.click(screen.getByTestId("next-to-review"))

    fireEvent.click(screen.getByTestId("start-adventure"))
    expect(onSelectCustom).toHaveBeenCalledWith({
      name: "Ada",
      highConcept: "Hacker Supreme",
      trouble: "Too Curious",
      genre: "Cyberpunk",
      aspects: ["Well Connected", "Never Backs Down"],
      skills: getDefaultPyramid(),
    })
  })
})
