import { useCallback, useMemo } from "react"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  FATE_CORE_SKILLS,
  PYRAMID_SHAPE,
  ladderLabel,
  getDefaultPyramid,
} from "@/lib/skills"
import { pyramidProgress } from "@/lib/validateSkillPyramid"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SkillPyramidFormProps {
  /** Current skill assignments: skill name → ladder level. */
  skills: Record<string, number>
  /** Called whenever a skill is assigned or cleared. */
  onChange: (skills: Record<string, number>) => void
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function SkillPyramidForm({ skills, onChange }: SkillPyramidFormProps) {
  // Set of skills already assigned across all tiers.
  const usedSkills = useMemo(() => new Set(Object.keys(skills)), [skills])

  // Group skills by tier for rendering: level → [skill1, skill2, ...]
  const skillsByTier = useMemo(() => {
    const map = new Map<number, string[]>()
    for (const [skill, level] of Object.entries(skills)) {
      const arr = map.get(level) ?? []
      arr.push(skill)
      map.set(level, arr)
    }
    return map
  }, [skills])

  // For a given tier + slot index, which skill is assigned (if any)?
  const getSkillForSlot = useCallback(
    (level: number, slotIndex: number): string | undefined => {
      const arr = skillsByTier.get(level)
      return arr?.[slotIndex]
    },
    [skillsByTier],
  )

  // When a dropdown changes, update the skills map.
  const handleChange = useCallback(
    (level: number, slotIndex: number, newSkill: string) => {
      const next = { ...skills }

      // Remove old skill from this slot if one was assigned.
      const oldSkill = getSkillForSlot(level, slotIndex)
      if (oldSkill) {
        delete next[oldSkill]
      }

      // Assign new skill (unless "__clear" sentinel).
      if (newSkill !== "__clear") {
        next[newSkill] = level
      }

      onChange(next)
    },
    [skills, onChange, getSkillForSlot],
  )

  const handleUseDefaults = useCallback(() => {
    onChange(getDefaultPyramid())
  }, [onChange])

  const handleClear = useCallback(() => {
    onChange({})
  }, [onChange])

  const progress = pyramidProgress(skills)
  const totalFilled = Object.keys(skills).length

  return (
    <div className="space-y-5" data-testid="skill-pyramid-form">
      {/* Progress summary */}
      <div className="flex items-center justify-between text-sm text-muted-foreground">
        <span>{totalFilled}/10 skills assigned</span>
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleUseDefaults}
            data-testid="use-defaults-button"
          >
            Use Defaults
          </Button>
          {totalFilled > 0 && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleClear}
              data-testid="clear-skills-button"
            >
              Clear
            </Button>
          )}
        </div>
      </div>

      {/* Tier rows */}
      {PYRAMID_SHAPE.map(({ level, count }) => {
        const tierProgress = progress.find((p) => p.level === level)!
        return (
          <div key={level} data-testid={`tier-${level}`}>
            <div className="mb-1.5 flex items-center gap-2 text-sm font-medium">
              <span>{ladderLabel(level)}</span>
              <span className="text-muted-foreground">
                ({tierProgress.filled}/{count})
              </span>
            </div>
            <div className="flex flex-wrap gap-2">
              {Array.from({ length: count }, (_, slotIndex) => {
                const currentSkill = getSkillForSlot(level, slotIndex)
                // Available options: unassigned skills + the currently selected one.
                const options = FATE_CORE_SKILLS.filter(
                  (s) => !usedSkills.has(s) || s === currentSkill,
                )
                return (
                  <Select
                    key={`${level}-${slotIndex}`}
                    value={currentSkill ?? ""}
                    onValueChange={(v) => handleChange(level, slotIndex, v)}
                  >
                    <SelectTrigger
                      className="w-40"
                      data-testid={`skill-select-${level}-${slotIndex}`}
                    >
                      <SelectValue placeholder="Choose skill…" />
                    </SelectTrigger>
                    <SelectContent>
                      {options.map((skill) => (
                        <SelectItem key={skill} value={skill}>
                          {skill}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )
              })}
            </div>
          </div>
        )
      })}
    </div>
  )
}
