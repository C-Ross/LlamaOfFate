package dice

import (
	"math/rand"
	"time"
)

// FateDie represents a single Fate die value (-1, 0, +1)
type FateDie int

const (
	Minus FateDie = -1
	Blank FateDie = 0
	Plus  FateDie = 1
)

// String returns the symbol representation of the die
func (d FateDie) String() string {
	switch d {
	case Minus:
		return "[-]"
	case Blank:
		return "[ ]"
	case Plus:
		return "[+]"
	default:
		return "[?]"
	}
}

// Roll represents a 4dF dice roll
type Roll struct {
	Dice   [4]FateDie `json:"dice"`
	Total  int        `json:"total"`
	RolledAt time.Time `json:"rolled_at"`
}

// String returns a formatted representation of the roll
func (r *Roll) String() string {
	return r.Dice[0].String() + r.Dice[1].String() + r.Dice[2].String() + r.Dice[3].String()
}

// Roller handles dice rolling with optional seeding for reproducible results
type Roller struct {
	rng *rand.Rand
}

// NewRoller creates a new dice roller
func NewRoller() *Roller {
	return &Roller{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewSeededRoller creates a new dice roller with a specific seed (for testing)
func NewSeededRoller(seed int64) *Roller {
	return &Roller{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// Roll4dF performs a 4dF roll
func (r *Roller) Roll4dF() *Roll {
	roll := &Roll{
		RolledAt: time.Now(),
	}
	
	total := 0
	for i := 0; i < 4; i++ {
		// Each die has 1/3 chance each of -1, 0, +1
		value := r.rng.Intn(3) - 1
		roll.Dice[i] = FateDie(value)
		total += value
	}
	
	roll.Total = total
	return roll
}

// RollWithModifier performs a skill check with a base skill and modifiers
func (r *Roller) RollWithModifier(skill Ladder, modifier int) *CheckResult {
	roll := r.Roll4dF()
	finalValue := Ladder(int(skill) + roll.Total + modifier)
	
	return &CheckResult{
		Roll:        roll,
		BaseSkill:   skill,
		Modifier:    modifier,
		FinalValue:  finalValue,
		CalculatedAt: time.Now(),
	}
}

// CheckResult represents the result of a skill check
type CheckResult struct {
	Roll         *Roll     `json:"roll"`
	BaseSkill    Ladder    `json:"base_skill"`
	Modifier     int       `json:"modifier"`
	FinalValue   Ladder    `json:"final_value"`
	CalculatedAt time.Time `json:"calculated_at"`
}

// String returns a formatted representation of the check result
func (cr *CheckResult) String() string {
	return cr.Roll.String()
}

// CompareAgainst compares this result against a difficulty and returns the outcome
func (cr *CheckResult) CompareAgainst(difficulty Ladder) *Outcome {
	shifts := cr.FinalValue.Compare(difficulty)
	
	var result OutcomeType
	switch {
	case shifts < 0:
		result = Failure
	case shifts == 0:
		result = Tie
	case shifts >= 1 && shifts <= 2:
		result = Success
	default: // shifts >= 3
		result = SuccessWithStyle
	}
	
	return &Outcome{
		Type:       result,
		Shifts:     shifts,
		Result:     cr,
		Difficulty: difficulty,
	}
}

// OutcomeType represents the type of outcome for an action
type OutcomeType int

const (
	Failure OutcomeType = iota
	Tie
	Success
	SuccessWithStyle
)

// String returns the name of the outcome type
func (o OutcomeType) String() string {
	switch o {
	case Failure:
		return "Failure"
	case Tie:
		return "Tie"
	case Success:
		return "Success"
	case SuccessWithStyle:
		return "Success with Style"
	default:
		return "Unknown"
	}
}

// Outcome represents the result of comparing a check against a difficulty
type Outcome struct {
	Type       OutcomeType   `json:"type"`
	Shifts     int           `json:"shifts"`
	Result     *CheckResult  `json:"result"`
	Difficulty Ladder        `json:"difficulty"`
}
