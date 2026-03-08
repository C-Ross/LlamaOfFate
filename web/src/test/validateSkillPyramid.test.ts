import { describe, expect, it } from "vitest"
import {
  validateSkillPyramid,
  pyramidProgress,
} from "@/lib/validateSkillPyramid"
import { Ladder } from "@/lib/skills"

// A valid standard pyramid: 1×Great, 2×Good, 3×Fair, 4×Average.
const validPyramid: Record<string, number> = {
  Notice: Ladder.Great,
  Athletics: Ladder.Good,
  Will: Ladder.Good,
  Investigate: Ladder.Fair,
  Rapport: Ladder.Fair,
  Fight: Ladder.Fair,
  Stealth: Ladder.Average,
  Physique: Ladder.Average,
  Empathy: Ladder.Average,
  Shoot: Ladder.Average,
}

describe("validateSkillPyramid", () => {
  it("accepts a valid standard pyramid", () => {
    const result = validateSkillPyramid(validPyramid)
    expect(result.valid).toBe(true)
    expect(result.errors).toHaveLength(0)
  })

  it("rejects unknown skill names", () => {
    const skills = { ...validPyramid }
    delete skills.Notice
    skills["Hacking"] = Ladder.Great
    const result = validateSkillPyramid(skills)
    expect(result.valid).toBe(false)
    expect(result.errors).toContainEqual("Unknown skill: Hacking")
  })

  it("rejects wrong number of skills", () => {
    const tooFew = { ...validPyramid }
    delete tooFew.Shoot
    const result = validateSkillPyramid(tooFew)
    expect(result.valid).toBe(false)
    expect(result.errors.some((e) => e.includes("Expected 10 skills, got 9"))).toBe(true)
  })

  it("rejects wrong tier distribution", () => {
    // Two skills at Great instead of one.
    const skills = { ...validPyramid, Athletics: Ladder.Great }
    const result = validateSkillPyramid(skills)
    expect(result.valid).toBe(false)
    expect(result.errors.some((e) => e.includes("at +4"))).toBe(true)
    expect(result.errors.some((e) => e.includes("at +3"))).toBe(true)
  })

  it("rejects an empty map", () => {
    const result = validateSkillPyramid({})
    expect(result.valid).toBe(false)
    expect(result.errors.some((e) => e.includes("Expected 10 skills, got 0"))).toBe(true)
  })

  it("collects multiple errors", () => {
    const result = validateSkillPyramid({ Hacking: 99 })
    expect(result.valid).toBe(false)
    expect(result.errors.length).toBeGreaterThanOrEqual(2)
  })
})

describe("pyramidProgress", () => {
  it("returns full progress for a complete pyramid", () => {
    const progress = pyramidProgress(validPyramid)
    for (const tier of progress) {
      expect(tier.filled).toBe(tier.required)
    }
  })

  it("returns zero filled for an empty map", () => {
    const progress = pyramidProgress({})
    for (const tier of progress) {
      expect(tier.filled).toBe(0)
    }
  })

  it("tracks partial progress correctly", () => {
    const partial: Record<string, number> = {
      Notice: Ladder.Great,
      Athletics: Ladder.Good,
      // Missing: 1 Good, 3 Fair, 4 Average
    }
    const progress = pyramidProgress(partial)
    const great = progress.find((t) => t.level === Ladder.Great)!
    const good = progress.find((t) => t.level === Ladder.Good)!
    const fair = progress.find((t) => t.level === Ladder.Fair)!
    expect(great.filled).toBe(1)
    expect(great.required).toBe(1)
    expect(good.filled).toBe(1)
    expect(good.required).toBe(2)
    expect(fair.filled).toBe(0)
    expect(fair.required).toBe(3)
  })

  it("caps filled at required even if over-filled", () => {
    const overfilled: Record<string, number> = {
      Notice: Ladder.Great,
      Athletics: Ladder.Great,
      Will: Ladder.Great,
    }
    const progress = pyramidProgress(overfilled)
    const great = progress.find((t) => t.level === Ladder.Great)!
    expect(great.filled).toBe(1) // capped at required=1
  })
})
