// Fate Core skill constants and default skill pyramid.
// Mirrors Go constants in internal/core/skills_list.go.

// ---------------------------------------------------------------------------
// Ladder levels (matches internal/core/dice/ladder.go)
// ---------------------------------------------------------------------------

export const Ladder = {
  Average: 1,
  Fair: 2,
  Good: 3,
  Great: 4,
  Superb: 5,
} as const

export type LadderLevel = (typeof Ladder)[keyof typeof Ladder]

/** Human-readable label for a ladder value. */
export const ladderLabel = (level: LadderLevel): string => {
  const labels: Record<LadderLevel, string> = {
    [Ladder.Average]: "Average (+1)",
    [Ladder.Fair]: "Fair (+2)",
    [Ladder.Good]: "Good (+3)",
    [Ladder.Great]: "Great (+4)",
    [Ladder.Superb]: "Superb (+5)",
  }
  return labels[level]
}

// ---------------------------------------------------------------------------
// The 18 canonical Fate Core skills (alphabetical)
// ---------------------------------------------------------------------------

export const FATE_CORE_SKILLS = [
  "Athletics",
  "Burglary",
  "Contacts",
  "Crafts",
  "Deceive",
  "Drive",
  "Empathy",
  "Fight",
  "Investigate",
  "Lore",
  "Notice",
  "Physique",
  "Provoke",
  "Rapport",
  "Resources",
  "Shoot",
  "Stealth",
  "Will",
] as const

export type FateCoreSkill = (typeof FATE_CORE_SKILLS)[number]

/** Set for O(1) validation of skill names. */
export const FATE_CORE_SKILL_SET: ReadonlySet<string> = new Set(FATE_CORE_SKILLS)

// ---------------------------------------------------------------------------
// Standard pyramid shape: 1 Great, 2 Good, 3 Fair, 4 Average (10 skills)
// ---------------------------------------------------------------------------

export const PYRAMID_SHAPE: ReadonlyArray<{ level: LadderLevel; count: number }> = [
  { level: Ladder.Great, count: 1 },
  { level: Ladder.Good, count: 2 },
  { level: Ladder.Fair, count: 3 },
  { level: Ladder.Average, count: 4 },
]

export const PYRAMID_TOTAL_SKILLS = 10

// ---------------------------------------------------------------------------
// Default skill priority — the 10 most commonly useful Fate Core skills,
// ordered by general versatility. Used by "Use Defaults" to fill a pyramid.
// ---------------------------------------------------------------------------

export const DEFAULT_SKILL_PRIORITY: readonly FateCoreSkill[] = [
  "Notice",
  "Athletics",
  "Will",
  "Investigate",
  "Rapport",
  "Fight",
  "Stealth",
  "Physique",
  "Empathy",
  "Shoot",
] as const

/**
 * Returns a default pyramid by assigning the priority skills to tiers
 * in order: first skill → Great, next 2 → Good, next 3 → Fair, last 4 → Average.
 * Result is a fresh Record so callers can mutate it freely.
 */
export function getDefaultPyramid(): Record<string, number> {
  const result: Record<string, number> = {}
  let i = 0
  for (const { level, count } of PYRAMID_SHAPE) {
    for (let j = 0; j < count; j++) {
      result[DEFAULT_SKILL_PRIORITY[i]] = level
      i++
    }
  }
  return result
}
