import { describe, expect, it } from "vitest"
import {
  DEFAULT_SKILL_PRIORITY,
  FATE_CORE_SKILLS,
  FATE_CORE_SKILL_SET,
  Ladder,
  ladderLabel,
  PYRAMID_SHAPE,
  PYRAMID_TOTAL_SKILLS,
  getDefaultPyramid,
} from "@/lib/skills"

describe("FATE_CORE_SKILLS", () => {
  it("contains exactly 18 skills", () => {
    expect(FATE_CORE_SKILLS).toHaveLength(18)
  })

  it("is sorted alphabetically", () => {
    const sorted = [...FATE_CORE_SKILLS].sort()
    expect(FATE_CORE_SKILLS).toEqual(sorted)
  })

  it("has no duplicates", () => {
    expect(FATE_CORE_SKILL_SET.size).toBe(FATE_CORE_SKILLS.length)
  })
})

describe("Ladder", () => {
  it("has correct numeric values", () => {
    expect(Ladder.Average).toBe(1)
    expect(Ladder.Fair).toBe(2)
    expect(Ladder.Good).toBe(3)
    expect(Ladder.Great).toBe(4)
    expect(Ladder.Superb).toBe(5)
  })
})

describe("ladderLabel", () => {
  it("returns human-readable labels", () => {
    expect(ladderLabel(Ladder.Average)).toBe("Average (+1)")
    expect(ladderLabel(Ladder.Great)).toBe("Great (+4)")
  })
})

describe("PYRAMID_SHAPE", () => {
  it("sums to PYRAMID_TOTAL_SKILLS", () => {
    const total = PYRAMID_SHAPE.reduce((sum, tier) => sum + tier.count, 0)
    expect(total).toBe(PYRAMID_TOTAL_SKILLS)
  })

  it("follows 1-2-3-4 distribution", () => {
    expect(PYRAMID_SHAPE.map((t) => t.count)).toEqual([1, 2, 3, 4])
  })
})

describe("DEFAULT_SKILL_PRIORITY", () => {
  it("has exactly 10 skills", () => {
    expect(DEFAULT_SKILL_PRIORITY).toHaveLength(PYRAMID_TOTAL_SKILLS)
  })

  it("uses only valid Fate Core skills", () => {
    for (const skill of DEFAULT_SKILL_PRIORITY) {
      expect(FATE_CORE_SKILL_SET.has(skill)).toBe(true)
    }
  })

  it("has no duplicates", () => {
    const unique = new Set(DEFAULT_SKILL_PRIORITY)
    expect(unique.size).toBe(DEFAULT_SKILL_PRIORITY.length)
  })
})

describe("getDefaultPyramid", () => {
  it("returns a valid pyramid with correct tier counts", () => {
    const pyramid = getDefaultPyramid()
    expect(Object.keys(pyramid)).toHaveLength(PYRAMID_TOTAL_SKILLS)
    const counts = new Map<number, number>()
    for (const level of Object.values(pyramid)) {
      counts.set(level, (counts.get(level) ?? 0) + 1)
    }
    for (const { level, count } of PYRAMID_SHAPE) {
      expect(counts.get(level)).toBe(count)
    }
  })

  it("assigns priority skills in order (first = Great)", () => {
    const pyramid = getDefaultPyramid()
    expect(pyramid[DEFAULT_SKILL_PRIORITY[0]]).toBe(Ladder.Great)
  })

  it("returns a fresh copy each time", () => {
    const a = getDefaultPyramid()
    const b = getDefaultPyramid()
    expect(a).toEqual(b)
    a.Notice = 99
    expect(b.Notice).toBe(Ladder.Great)
  })
})
