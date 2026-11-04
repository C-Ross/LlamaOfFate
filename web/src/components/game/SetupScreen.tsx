import { useState } from "react"
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { SkillPyramidForm } from "@/components/game/SkillPyramidForm"
import { validateSkillPyramid } from "@/lib/validateSkillPyramid"
import { getDefaultPyramid, ladderLabel, PYRAMID_SHAPE } from "@/lib/skills"
import type { LadderLevel } from "@/lib/skills"
import type { ScenarioPreset, CustomSetup } from "@/lib/types"

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const aspectPlaceholders = [
  "e.g. Well Connected in the Underworld",
  "e.g. My Father's Sword",
  "e.g. Never Leave a Friend Behind",
]

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
  const [wizardStep, setWizardStep] = useState<"identity" | "skills" | "review">("identity")

  // Custom form state
  const [name, setName] = useState("")
  const [highConcept, setHighConcept] = useState("")
  const [trouble, setTrouble] = useState("")
  const [genre, setGenre] = useState("")
  const [aspects, setAspects] = useState<string[]>(["", "", ""])
  const [skills, setSkills] = useState<Record<string, number>>(getDefaultPyramid)

  const resetCustomForm = () => {
    setMode("pick")
    setWizardStep("identity")
  }

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

  // ---------------------------------------------------------------------------
  // Custom wizard — Step 1: Identity & Aspects
  // ---------------------------------------------------------------------------
  if (mode === "custom" && wizardStep === "identity") {
    const canProceed = name.trim() && highConcept.trim() && trouble.trim() && genre.trim()

    return (
      <div className="flex h-full items-center justify-center p-6" data-testid="setup-custom" data-step="identity">
        <Card className="w-full max-w-md">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="font-heading text-xl">Create Your Character</CardTitle>
              <span className="text-xs text-muted-foreground">Step 1 of 3</span>
            </div>
            <CardDescription>Define who your character is.</CardDescription>
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

            {/* Optional additional aspects */}
            <div className="space-y-2 pt-2">
              <span className="text-sm font-medium text-muted-foreground">Additional Aspects (optional)</span>
              {aspects.map((a, i) => (
                <Input
                  key={i}
                  value={a}
                  onChange={(e) => {
                    const next = [...aspects]
                    next[i] = e.target.value
                    setAspects(next)
                  }}
                  placeholder={aspectPlaceholders[i]}
                  data-testid={`aspect-input-${i}`}
                />
              ))}
            </div>

            <div className="flex gap-3 pt-2">
              <Button variant="outline" onClick={resetCustomForm}>Back</Button>
              <Button disabled={!canProceed} onClick={() => setWizardStep("skills")} data-testid="next-to-skills">
                Next: Skills
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  // ---------------------------------------------------------------------------
  // Custom wizard — Step 2: Skills
  // ---------------------------------------------------------------------------
  if (mode === "custom" && wizardStep === "skills") {
    const pyramidValid = validateSkillPyramid(skills).valid

    return (
      <div className="flex h-full items-center justify-center p-6" data-testid="setup-custom" data-step="skills">
        <Card className="w-full max-w-lg">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="font-heading text-xl">Skill Pyramid</CardTitle>
              <span className="text-xs text-muted-foreground">Step 2 of 3</span>
            </div>
            <CardDescription>
              Assign 10 skills in a pyramid: 1 Great, 2 Good, 3 Fair, 4 Average.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <SkillPyramidForm skills={skills} onChange={setSkills} />

            <div className="flex gap-3 pt-2">
              <Button variant="outline" onClick={() => setWizardStep("identity")} data-testid="back-to-identity">
                Back
              </Button>
              <Button
                disabled={!pyramidValid}
                onClick={() => setWizardStep("review")}
                data-testid="next-to-review"
              >
                Next: Review
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  // ---------------------------------------------------------------------------
  // Custom wizard — Step 3: Review
  // ---------------------------------------------------------------------------
  if (mode === "custom" && wizardStep === "review") {
    const filteredAspects = aspects.map((a) => a.trim()).filter(Boolean)
    const submission: CustomSetup = {
      name: name.trim(),
      highConcept: highConcept.trim(),
      trouble: trouble.trim(),
      genre: genre.trim(),
      ...(filteredAspects.length > 0 ? { aspects: filteredAspects } : {}),
      skills,
    }

    // Group skills by tier for display.
    const skillsByTier = new Map<number, string[]>()
    for (const [skill, level] of Object.entries(skills)) {
      const arr = skillsByTier.get(level) ?? []
      arr.push(skill)
      skillsByTier.set(level, arr)
    }

    return (
      <div className="flex h-full items-center justify-center p-6" data-testid="setup-custom" data-step="review">
        <Card className="w-full max-w-md">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="font-heading text-xl">Review Character</CardTitle>
              <span className="text-xs text-muted-foreground">Step 3 of 3</span>
            </div>
            <CardDescription>Confirm your character before starting.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Identity */}
            <div className="space-y-1">
              <p className="text-sm"><span className="font-medium">Name:</span> {submission.name}</p>
              <p className="text-sm"><span className="font-medium">High Concept:</span> {submission.highConcept}</p>
              <p className="text-sm"><span className="font-medium">Trouble:</span> {submission.trouble}</p>
              <p className="text-sm"><span className="font-medium">Genre:</span> {submission.genre}</p>
            </div>

            {/* Aspects */}
            {filteredAspects.length > 0 && (
              <div className="space-y-1" data-testid="review-aspects">
                <p className="text-sm font-medium">Additional Aspects</p>
                <ul className="list-disc list-inside text-sm text-muted-foreground">
                  {filteredAspects.map((a, i) => (
                    <li key={i}>{a}</li>
                  ))}
                </ul>
              </div>
            )}

            {/* Skills */}
            <div className="space-y-1" data-testid="review-skills">
              <p className="text-sm font-medium">Skills</p>
              <div className="space-y-1 text-sm text-muted-foreground">
                {PYRAMID_SHAPE.map(({ level }) => {
                  const tierSkills = skillsByTier.get(level)
                  if (!tierSkills?.length) return null
                  return (
                    <p key={level}>
                      <span className="font-medium">{ladderLabel(level as LadderLevel)}:</span>{" "}
                      {tierSkills.join(", ")}
                    </p>
                  )
                })}
              </div>
            </div>

            {/* Fate Core defaults */}
            <div className="space-y-1 text-sm text-muted-foreground">
              <p><span className="font-medium">Refresh:</span> 3</p>
              <p><span className="font-medium">Stress:</span> 2 Physical, 2 Mental</p>
            </div>

            <div className="flex gap-3 pt-2">
              <Button variant="outline" onClick={() => setWizardStep("skills")} data-testid="back-to-skills">
                Back
              </Button>
              <Button onClick={() => onSelectCustom(submission)} data-testid="start-adventure">
                Start Adventure
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
