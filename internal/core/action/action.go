package action

import (
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
)

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
