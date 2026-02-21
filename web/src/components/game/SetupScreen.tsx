import { useState } from "react"
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import type { ScenarioPreset, CustomSetup } from "@/lib/types"

// ---------------------------------------------------------------------------
// Genre badge color helpers
// ---------------------------------------------------------------------------

const genreColors: Record<string, string> = {
  Western: "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300",
  Cyberpunk: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/40 dark:text-cyan-300",
  Fantasy: "bg-violet-100 text-violet-800 dark:bg-violet-900/40 dark:text-violet-300",
}

function GenreBadge({ genre }: { genre: string }) {
  return (
    <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${genreColors[genre] ?? "bg-muted text-muted-foreground"}`}>
      {genre}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface SetupScreenProps {
  presets: ScenarioPreset[]
  allowCustom: boolean
  generatingMessage: string | null
  onSelectPreset: (presetId: string) => void
  onSelectCustom: (custom: CustomSetup) => void
  /** True when the player has a saved game they can resume. */
  hasSavedGame?: boolean
  /** Called when the player clicks "Continue" to resume their saved game. */
  onContinue?: () => void
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function SetupScreen({
  presets,
  allowCustom,
  generatingMessage,
  onSelectPreset,
  onSelectCustom,
  hasSavedGame = false,
  onContinue,
}: SetupScreenProps) {
  const [mode, setMode] = useState<"pick" | "custom">("pick")

  // Custom form state
  const [name, setName] = useState("")
  const [highConcept, setHighConcept] = useState("")
  const [trouble, setTrouble] = useState("")
  const [genre, setGenre] = useState("")

  // While generating, show a spinner overlay
  if (generatingMessage) {
    return (
      <div className="flex h-full items-center justify-center" data-testid="setup-generating">
        <div className="text-center space-y-4">
          <div className="mx-auto size-10 animate-spin rounded-full border-4 border-muted border-t-primary" />
          <p className="text-muted-foreground font-body">{generatingMessage}</p>
        </div>
      </div>
    )
  }

  if (mode === "custom") {
    const canSubmit = name.trim() && highConcept.trim() && trouble.trim() && genre.trim()

    return (
      <div className="flex h-full items-center justify-center p-6" data-testid="setup-custom">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="font-heading text-xl">Create Your Character</CardTitle>
            <CardDescription>Describe your character and genre. We&apos;ll generate a scenario.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <label className="block space-y-1">
              <span className="text-sm font-medium">Character Name</span>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Ada Lovelace" />
            </label>
            <label className="block space-y-1">
              <span className="text-sm font-medium">High Concept</span>
              <Input value={highConcept} onChange={(e) => setHighConcept(e.target.value)} placeholder="e.g. Rogue AI Whisperer" />
            </label>
            <label className="block space-y-1">
              <span className="text-sm font-medium">Trouble</span>
              <Input value={trouble} onChange={(e) => setTrouble(e.target.value)} placeholder="e.g. Trusts Machines More Than People" />
            </label>
            <label className="block space-y-1">
              <span className="text-sm font-medium">Genre</span>
              <Input value={genre} onChange={(e) => setGenre(e.target.value)} placeholder="e.g. Cyberpunk, Western, Fantasy" />
            </label>
            <div className="flex gap-3 pt-2">
              <Button variant="outline" onClick={() => setMode("pick")}>Back</Button>
              <Button
                disabled={!canSubmit}
                onClick={() => onSelectCustom({ name: name.trim(), highConcept: highConcept.trim(), trouble: trouble.trim(), genre: genre.trim() })}
              >
                Generate Scenario
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  // Default: preset picker
  return (
    <div className="flex h-full items-center justify-center p-6" data-testid="setup-picker">
      <div className="w-full max-w-2xl space-y-6">
        <div className="text-center space-y-2">
          <h1 className="text-3xl font-heading font-bold tracking-widest uppercase">
            <span className="text-accent-foreground/60">Llama</span> of <span className="text-primary">Fate</span>
          </h1>
          <p className="text-muted-foreground font-body">Choose your adventure</p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {hasSavedGame && (
            <Card
              className="cursor-pointer border-primary/50 bg-primary/5 transition-shadow hover:shadow-md"
              onClick={onContinue}
              data-testid="continue-game"
            >
              <CardHeader>
                <CardTitle className="font-heading text-base">Continue Game</CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground font-body">Resume your previous adventure where you left off.</p>
              </CardContent>
            </Card>
          )}
          {presets.map((preset) => (
            <Card
              key={preset.id}
              className="cursor-pointer transition-shadow hover:shadow-md hover:border-primary/50"
              onClick={() => onSelectPreset(preset.id)}
              data-testid={`preset-${preset.id}`}
            >
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="font-heading text-base">{preset.title}</CardTitle>
                </div>
                <GenreBadge genre={preset.genre} />
              </CardHeader>
              <CardContent>
                <p className="text-sm text-muted-foreground font-body">{preset.description}</p>
              </CardContent>
            </Card>
          ))}
        </div>

        {allowCustom && (
          <div className="text-center">
            <Button variant="outline" onClick={() => setMode("custom")} data-testid="custom-button">
              Create Your Own
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}
