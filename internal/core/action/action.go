package action

import (
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
)

// AttackSideEffect describes the side effect of an attack outcome
// beyond raw damage (boosts granted by ties or defend-with-style).
type AttackSideEffect int

const (
	// NoSideEffect means no boost is generated.
	NoSideEffect AttackSideEffect = iota
	// AttackerBoost means the attacker receives a boost (Fate Core tie on attack).
	AttackerBoost
	// AttackerBoostOption means the attacker may optionally reduce shifts by 1
	// to gain a boost (Fate Core success-with-style on attack).
	AttackerBoostOption
	// DefenderBoost means the defender receives a boost (Fate Core defend with style, failure by 3+).
	DefenderBoost
)

// ResolveAttackOutcome applies Fate Core attack rules to an outcome:
//   - On success: shifts are clamped to a minimum of 1, no side effect.
//   - On success-with-style: shifts are clamped to a minimum of 1,
//     attacker may optionally reduce by 1 for a boost (AttackerBoostOption).
//   - On tie: shifts are 0, attacker gets a boost.
//   - On failure by 3+: shifts are 0, defender gets a boost.
//   - On other failure: shifts are 0, no side effect.
//
// Returns the effective damage shifts and any side effect.
func ResolveAttackOutcome(outcome *dice.Outcome) (shifts int, side AttackSideEffect) {
	switch outcome.Type {
	case dice.SuccessWithStyle:
		shifts = outcome.Shifts
		if shifts < 1 {
			shifts = 1
		}
		return shifts, AttackerBoostOption
	case dice.Success:
		shifts = outcome.Shifts
		if shifts < 1 {
			shifts = 1 // Minimum 1 shift on success — Fate Core SRD Attack
		}
		return shifts, NoSideEffect
	case dice.Tie:
		return 0, AttackerBoost
	default: // Failure
		if outcome.Shifts <= -3 {
			return 0, DefenderBoost
		}
		return 0, NoSideEffect
	}
}

// ActionType represents the four basic actions in Fate Core
type ActionType int

const (
	Overcome ActionType = iota
	CreateAdvantage
	Attack
	Defend
)

// String returns the name of the action type
func (at ActionType) String() string {
	switch at {
	case Overcome:
		return "Overcome"
	case CreateAdvantage:
		return "Create an Advantage"
	case Attack:
		return "Attack"
	case Defend:
		return "Defend"
	default:
		return "Unknown"
	}
}

// Action represents a character's attempted action
type Action struct {
	ID          string     `json:"id"`
	CharacterID string     `json:"character_id"`
	Type        ActionType `json:"type"`
	Skill       string     `json:"skill"`
	Description string     `json:"description"`
	RawInput    string     `json:"raw_input"`
	Target      string     `json:"target,omitempty"` // The target of the action (character ID, object, or description)

	// Active NPC opposition (Fate Core: active vs passive opposition).
	// When set, an NPC rolls their skill as opposition instead of using a
	// flat difficulty. Applies to Overcome and Create Advantage outside conflict.
	OpposingNPCID string `json:"opposing_npc_id,omitempty"` // NPC providing active opposition
	OpposingSkill string `json:"opposing_skill,omitempty"`  // Skill the NPC uses to oppose

	// Action Modifiers
	Difficulty dice.Ladder    `json:"difficulty"`
	Aspects    []AspectInvoke `json:"aspects"`
	Stunts     []string       `json:"stunt_ids"`

	// Results
	CheckResult *dice.CheckResult `json:"check_result,omitempty"`
	Outcome     *dice.Outcome     `json:"outcome,omitempty"`
	Effects     []Effect          `json:"effects"`

	// Narrative
	ResultText string `json:"result_text"`

	Timestamp time.Time `json:"timestamp"`
}

// AspectInvoke represents using an aspect in an action
type AspectInvoke struct {
	AspectText    string `json:"aspect_text"`
	Source        string `json:"source"` // "character", "situation", "consequence"
	SourceID      string `json:"source_id"`
	IsFree        bool   `json:"is_free"`
	FatePointCost int    `json:"fate_point_cost"`
	Bonus         int    `json:"bonus"`     // Usually +2
	IsReroll      bool   `json:"is_reroll"` // true = used for reroll instead of +2
}

// Effect represents a mechanical effect of an action
type Effect struct {
	Type        string      `json:"type"`   // "stress", "consequence", "aspect", "advantage", "boost"
	Target      string      `json:"target"` // Character or scene ID
	Value       interface{} `json:"value"`  // Type-specific data
	Description string      `json:"description"`
}

// NewAction creates a new action
func NewAction(id, characterID string, actionType ActionType, skill, description string) *Action {
	return &Action{
		ID:          id,
		CharacterID: characterID,
		Type:        actionType,
		Skill:       skill,
		Description: description,
		Difficulty:  dice.Mediocre, // Default difficulty
		Aspects:     make([]AspectInvoke, 0),
		Stunts:      make([]string, 0),
		Effects:     make([]Effect, 0),
		Timestamp:   time.Now(),
	}
}

// NewActionWithTarget creates a new action with a target
func NewActionWithTarget(id, characterID string, actionType ActionType, skill, description, target string) *Action {
	action := NewAction(id, characterID, actionType, skill, description)
	action.Target = target
	return action
}

// AddAspectInvoke adds an aspect invocation to the action
func (a *Action) AddAspectInvoke(invoke AspectInvoke) {
	a.Aspects = append(a.Aspects, invoke)
}

// AddEffect adds an effect to the action
func (a *Action) AddEffect(effect Effect) {
	a.Effects = append(a.Effects, effect)
}

// CalculateBonus calculates the total bonus from aspects and stunts
func (a *Action) CalculateBonus() int {
	bonus := 0

	// Add bonuses from aspect invocations
	for _, aspect := range a.Aspects {
		bonus += aspect.Bonus
	}

	// TODO: Add stunt bonuses when stunt system is implemented

	return bonus
}

// IsSuccess returns true if the action was successful
func (a *Action) IsSuccess() bool {
	if a.Outcome == nil {
		return false
	}
	return a.Outcome.Type == dice.Success || a.Outcome.Type == dice.SuccessWithStyle
}

// IsSuccessWithStyle returns true if the action succeeded with style
func (a *Action) IsSuccessWithStyle() bool {
	if a.Outcome == nil {
		return false
	}
	return a.Outcome.Type == dice.SuccessWithStyle
}

// FreeInvokesForOutcome returns the number of free invokes and whether the
// result is a boost for a Create an Advantage action, based on the outcome.
//
// Per Fate Core SRD (Create an Advantage):
//   - Success with Style → 2 free invokes, not a boost
//   - Success           → 1 free invoke, not a boost
//   - Tie               → 1 free invoke on a boost aspect
//   - Failure           → 0 free invokes
func FreeInvokesForOutcome(outcome dice.OutcomeType) (freeInvokes int, isBoost bool) {
	switch outcome {
	case dice.SuccessWithStyle:
		return 2, false
	case dice.Success:
		return 1, false
	case dice.Tie:
		return 1, true
	default: // dice.Failure
		return 0, false
	}
}
