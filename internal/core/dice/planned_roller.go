package dice

import "time"

// PlannedRoller is a test double that returns pre-configured 4dF dice totals
// in sequence. Each call to RollWithModifier pops the next planned total,
// combines it with the supplied skill + modifier, and returns a fully formed
// CheckResult. Reroll works the same way — it pops the next planned total.
//
// Usage:
//
//	roller := dice.NewPlannedRoller([]int{2, -1, 0})
//	// First RollWithModifier will use dice total 2
//	// Second will use -1
//	// Third will use 0
//	// Further calls panic (out of planned rolls)
type PlannedRoller struct {
	totals []int
	index  int
}

// NewPlannedRoller creates a PlannedRoller from a sequence of 4dF dice totals.
// Each total should be in the range [-4, +4] (the possible sums of 4 Fate dice),
// though no enforcement is applied so out-of-range values can be used for edge-case tests.
func NewPlannedRoller(totals []int) *PlannedRoller {
	return &PlannedRoller{totals: totals}
}

func (p *PlannedRoller) next() int {
	if p.index >= len(p.totals) {
		panic("PlannedRoller: no more planned rolls — add more totals")
	}
	total := p.totals[p.index]
	p.index++
	return total
}

// rollFromTotal builds a Roll with the given total. Individual dice faces are
// synthetic (all blank except the first die which carries the full total as a
// single value clamped to [-1,+1] per die). For tests that only inspect the
// total/final value this is fine; for tests that inspect individual dice faces,
// use NewSeededRoller instead.
func rollFromTotal(total int) *Roll {
	roll := &Roll{RolledAt: time.Now(), Total: total}
	remaining := total
	for i := 0; i < 4; i++ {
		switch {
		case remaining > 0:
			roll.Dice[i] = Plus
			remaining--
		case remaining < 0:
			roll.Dice[i] = Minus
			remaining++
		default:
			roll.Dice[i] = Blank
		}
	}
	return roll
}

// RollWithModifier pops the next planned dice total, combines it with skill +
// modifier, and returns a CheckResult.
func (p *PlannedRoller) RollWithModifier(skill Ladder, modifier int) *CheckResult {
	total := p.next()
	roll := rollFromTotal(total)
	finalValue := Ladder(int(skill) + total + modifier)

	return &CheckResult{
		Roll:         roll,
		BaseSkill:    skill,
		Modifier:     modifier,
		FinalValue:   finalValue,
		CalculatedAt: time.Now(),
	}
}

// Reroll pops the next planned dice total and builds a new CheckResult,
// preserving the original's skill and modifier.
func (p *PlannedRoller) Reroll(original *CheckResult) *CheckResult {
	total := p.next()
	roll := rollFromTotal(total)
	finalValue := Ladder(int(original.BaseSkill) + total + original.Modifier)

	return &CheckResult{
		Roll:         roll,
		BaseSkill:    original.BaseSkill,
		Modifier:     original.Modifier,
		FinalValue:   finalValue,
		CalculatedAt: time.Now(),
		Rerolled:     true,
		OriginalRoll: original.Roll,
	}
}

// Remaining returns how many planned rolls have not yet been consumed.
func (p *PlannedRoller) Remaining() int {
	return len(p.totals) - p.index
}
