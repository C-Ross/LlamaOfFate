// Dice parsing utilities for Fate dice visualization.

/** A single Fate die face value: -1 (minus), 0 (blank), or +1 (plus). */
export type DieFace = -1 | 0 | 1

/**
 * Parse Fate dice faces from a result string.
 *
 * The Go backend formats roll results with dice faces as a prefix:
 *   "[+][-][ ][+] (Total: Good (+3) vs Difficulty Fair (+2))"
 *
 * Returns an array of 4 DieFace values, or null if the string doesn't
 * contain parseable dice faces.
 */
export function parseDiceFaces(result: string): DieFace[] | null {
  const pattern = /\[([+\- ])\]/g
  const faces: DieFace[] = []

  let match: RegExpExecArray | null
  while ((match = pattern.exec(result)) !== null) {
    const symbol = match[1]
    if (symbol === "+") faces.push(1)
    else if (symbol === "-") faces.push(-1)
    else faces.push(0)
  }

  return faces.length >= 4 ? faces.slice(0, 4) : null
}

/**
 * Extract the total and comparison from a result string.
 *
 * Input: "[+][-][ ][+] (Total: Good (+3) vs Difficulty Fair (+2))"
 * Returns: "Total: Good (+3) vs Difficulty Fair (+2)"
 *
 * Falls back to the full string if no parenthesized total is found.
 */
export function parseResultDetail(result: string): string {
  // Match the outermost parenthesized group that starts with "Total:"
  const match = result.match(/\(Total:.+\)\s*$/)
  return match ? match[0].slice(1, -1) : result
}

/** Label for a die face, used in aria-label and tooltips. */
export function dieFaceLabel(face: DieFace): string {
  switch (face) {
    case 1:
      return "Plus"
    case -1:
      return "Minus"
    case 0:
      return "Blank"
  }
}

/** Sum an array of die faces. */
export function sumDice(faces: DieFace[]): number {
  return faces.reduce<number>((sum, f) => sum + f, 0)
}
