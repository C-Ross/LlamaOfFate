import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import { GameSidebar } from "@/components/game/GameSidebar"
import type { GameState } from "@/hooks/useGameState"

const emptyState: GameState = {
  player: null,
  situationAspects: [],
  npcs: [],
  fatePoints: 0,
  stressTracks: {},
  consequences: [],
  inConflict: false,
  inChallenge: false,
  challengeTasks: [],
  sceneName: "",
}

describe("GameSidebar", () => {
  it("shows placeholder when no player loaded", () => {
    render(<GameSidebar state={emptyState} />)
    expect(screen.getByText("No character loaded")).toBeInTheDocument()
  })

  it("renders player name as card title", () => {
    const state: GameState = {
      ...emptyState,
      player: {
        name: "Zara",
        highConcept: "Wizard Detective",
        trouble: "Curiosity Kills",
        aspects: ["Well Connected"],
        fatePoints: 3,
        refresh: 3,
        stressTracks: {},
        consequences: [],
      },
      fatePoints: 3,
    }

    render(<GameSidebar state={state} />)
    expect(screen.getByText("Zara")).toBeInTheDocument()
    expect(screen.getByText("Wizard Detective")).toBeInTheDocument()
    expect(screen.getByText("Curiosity Kills")).toBeInTheDocument()
    expect(screen.getByText("Well Connected")).toBeInTheDocument()
  })

  it("renders situation aspects", () => {
    const state: GameState = {
      ...emptyState,
      situationAspects: [
        { name: "Foggy Night", freeInvokes: 1 },
        { name: "Slippery Floor", freeInvokes: 0 },
      ],
    }

    render(<GameSidebar state={state} />)
    expect(screen.getByText("Foggy Night")).toBeInTheDocument()
    expect(screen.getByText("Slippery Floor")).toBeInTheDocument()
  })

  it("shows 'None active' when no situation aspects", () => {
    render(<GameSidebar state={emptyState} />)
    expect(screen.getByText("None active")).toBeInTheDocument()
  })

  it("renders fate points", () => {
    const state: GameState = {
      ...emptyState,
      fatePoints: 5,
      player: {
        name: "Zara",
        highConcept: "",
        trouble: "",
        aspects: [],
        fatePoints: 5,
        refresh: 3,
        stressTracks: {},
        consequences: [],
      },
    }

    render(<GameSidebar state={state} />)
    expect(screen.getByText("5")).toBeInTheDocument()
    expect(screen.getByText("/ 3 refresh")).toBeInTheDocument()
  })

  it("shows 'Combatants' title during conflict", () => {
    const state: GameState = {
      ...emptyState,
      inConflict: true,
      npcs: [{ name: "Grim", highConcept: "", aspects: [], isTakenOut: false }],
    }

    render(<GameSidebar state={state} />)
    expect(screen.getByText("Combatants")).toBeInTheDocument()
  })

  it("shows 'NPCs' title outside conflict", () => {
    const state: GameState = {
      ...emptyState,
      inConflict: false,
      npcs: [{ name: "Grim", highConcept: "", aspects: [], isTakenOut: false }],
    }

    render(<GameSidebar state={state} />)
    expect(screen.getByText("NPCs")).toBeInTheDocument()
  })
})
