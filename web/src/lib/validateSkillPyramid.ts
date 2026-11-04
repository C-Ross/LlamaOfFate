// Client-side skill pyramid validation.
// Mirrors the server-side ValidateStandardSkillPyramid in Go but returns
// structured errors suitable for real-time UI feedback.

import {
  FATE_CORE_SKILL_SET,
  PYRAMID_SHAPE,
  PYRAMID_TOTAL_SKILLS,
  type LadderLevel,
} from "./skills"

export interface PyramidValidationResult {
  valid: boolean
  errors: string[]
}

/**
 * Validates that a skill map forms a legal standard Fate Core pyramid.
 *
 * Rules:
 * - Exactly 10 skills total
 * - 1 at Great (+4), 2 at Good (+3), 3 at Fair (+2), 4 at Average (+1)
 * - All skill names must be from the canonical 18 Fate Core skills
 * - No duplicate skill names (enforced by Record keys)
 *
 * Returns { valid: true, errors: [] } when the pyramid is correct,
 * or { valid: false, errors: [...] } with human-readable reasons.
 */
export function validateSkillPyramid(
  skills: Record<string, number>,
): PyramidValidationResult {
  const errors: string[] = []

  // Check for unknown skill names.
  for (const name of Object.keys(skills)) {
    if (!FATE_CORE_SKILL_SET.has(name)) {
      errors.push(`Unknown skill: ${name}`)
    }
  }

  const count = Object.keys(skills).length
  if (count !== PYRAMID_TOTAL_SKILLS) {
    errors.push(`Expected ${PYRAMID_TOTAL_SKILLS} skills, got ${count}`)
  }

  // Count skills at each tier and compare to the required shape.
  const tierCounts = new Map<number, number>()
  for (const level of Object.values(skills)) {
    tierCounts.set(level, (tierCounts.get(level) ?? 0) + 1)
  }

  for (const { level, count: required } of PYRAMID_SHAPE) {
    const actual = tierCounts.get(level) ?? 0
    if (actual !== required) {
      errors.push(`Need ${required} skill(s) at +${level}, got ${actual}`)
    }
  }

  return { valid: errors.length === 0, errors }
}

/**
 * Returns the number of remaining skill slots to fill, broken down by tier.
 * Useful for showing progress in the UI (e.g., "2 more Fair skills needed").
 */
export function pyramidProgress(
  skills: Record<string, number>,
): { level: LadderLevel; label: string; required: number; filled: number }[] {
  const tierCounts = new Map<number, number>()
  for (const level of Object.values(skills)) {
    tierCounts.set(level, (tierCounts.get(level) ?? 0) + 1)
  }

  const labels: Record<number, string> = {
    4: "Great (+4)",
    3: "Good (+3)",
    2: "Fair (+2)",
    1: "Average (+1)",
  }

  return PYRAMID_SHAPE.map(({ level, count }) => ({
    level,
    label: labels[level] ?? `+${level}`,
    required: count,
    filled: Math.min(tierCounts.get(level) ?? 0, count),
  }))
}
