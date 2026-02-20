import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { SetupScreen } from "@/components/game/SetupScreen"
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

  it("switches to custom form and submits", () => {
    const onSelectCustom = vi.fn()
    render(
      <SetupScreen
        presets={presets}
        allowCustom={true}
        generatingMessage={null}
        onSelectPreset={vi.fn()}
        onSelectCustom={onSelectCustom}
      />,
    )

    // Click to go to custom form
    fireEvent.click(screen.getByTestId("custom-button"))
    expect(screen.getByTestId("setup-custom")).toBeInTheDocument()

    // Fill the form
    fireEvent.change(screen.getByPlaceholderText("e.g. Ada Lovelace"), { target: { value: "Ada" } })
    fireEvent.change(screen.getByPlaceholderText("e.g. Rogue AI Whisperer"), { target: { value: "Hacker Supreme" } })
    fireEvent.change(screen.getByPlaceholderText("e.g. Trusts Machines More Than People"), { target: { value: "Too Curious" } })
    fireEvent.change(screen.getByPlaceholderText("e.g. Cyberpunk, Western, Fantasy"), { target: { value: "Cyberpunk" } })

    fireEvent.click(screen.getByText("Generate Scenario"))
    expect(onSelectCustom).toHaveBeenCalledWith({
      name: "Ada",
      highConcept: "Hacker Supreme",
      trouble: "Too Curious",
      genre: "Cyberpunk",
    })
  })

  it("disables submit button when form is incomplete", () => {
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

    const submitButton = screen.getByText("Generate Scenario")
    expect(submitButton).toBeDisabled()
  })

  it("shows Back button in custom form", () => {
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

    // Clicking Back returns to picker
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
})
