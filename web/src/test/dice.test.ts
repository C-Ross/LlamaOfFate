import { describe, it, expect } from "vitest"
import {
  parseDiceFaces,
  parseResultDetail,
  dieFaceLabel,
  sumDice,
  type DieFace,
} from "@/lib/dice"

describe("parseDiceFaces", () => {
  it("parses four dice faces from a standard result string", () => {
    const result = "[+][-][ ][+] (Total: Good (+3) vs Difficulty Fair (+2))"
    expect(parseDiceFaces(result)).toEqual([1, -1, 0, 1])
  })

  it("parses all-plus roll", () => {
    expect(parseDiceFaces("[+][+][+][+] (Total: Legendary (+8))")).toEqual([1, 1, 1, 1])
  })

  it("parses all-minus roll", () => {
    expect(parseDiceFaces("[-][-][-][-] (Total: Terrible (-2))")).toEqual([-1, -1, -1, -1])
  })

  it("parses all-blank roll", () => {
    expect(parseDiceFaces("[ ][ ][ ][ ] (Total: Mediocre (+0))")).toEqual([0, 0, 0, 0])
  })

  it("returns null for a string without dice faces", () => {
    expect(parseDiceFaces("Good (+3)")).toBeNull()
  })

  it("returns null for fewer than 4 dice", () => {
    expect(parseDiceFaces("[+][-][ ]")).toBeNull()
  })

  it("returns only the first 4 dice if more are present", () => {
    expect(parseDiceFaces("[+][-][ ][+][-]")).toEqual([1, -1, 0, 1])
  })

  it("returns null for empty string", () => {
    expect(parseDiceFaces("")).toBeNull()
  })
})

describe("parseResultDetail", () => {
  it("extracts the parenthesized detail from a result string", () => {
    const result = "[+][-][ ][+] (Total: Good (+3) vs Difficulty Fair (+2))"
    expect(parseResultDetail(result)).toBe("Total: Good (+3) vs Difficulty Fair (+2)")
  })

  it("handles defense comparison format", () => {
    const result = "[+][ ][-][+] (Total: Great (+4) vs Bandit's Defense Good (+3))"
    expect(parseResultDetail(result)).toBe("Total: Great (+4) vs Bandit's Defense Good (+3)")
  })

  it("returns the full string if no parenthesized portion found", () => {
    expect(parseResultDetail("Good (+3)")).toBe("Good (+3)")
  })
})

describe("dieFaceLabel", () => {
  it("returns Plus for +1", () => {
    expect(dieFaceLabel(1)).toBe("Plus")
  })

  it("returns Minus for -1", () => {
    expect(dieFaceLabel(-1)).toBe("Minus")
  })

  it("returns Blank for 0", () => {
    expect(dieFaceLabel(0)).toBe("Blank")
  })
})

describe("sumDice", () => {
  it("sums positive dice", () => {
    expect(sumDice([1, 1, 1, 1])).toBe(4)
  })

  it("sums negative dice", () => {
    expect(sumDice([-1, -1, -1, -1])).toBe(-4)
  })

  it("sums mixed dice", () => {
    const faces: DieFace[] = [1, -1, 0, 1]
    expect(sumDice(faces)).toBe(1)
  })

  it("sums all-blank dice to zero", () => {
    expect(sumDice([0, 0, 0, 0])).toBe(0)
  })
})
