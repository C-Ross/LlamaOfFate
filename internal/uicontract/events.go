// Package uicontract — GameEvent types for event-driven scene processing.
//
// Each event corresponds to a Display* method on the UI interface. The engine
// produces events; the caller (terminal loop, web handler, etc.) decides how
// to render them.
package uicontract

// GameEvent is the interface implemented by all event types returned from
// HandleInput. Each concrete event maps 1:1 to a UI.Display* method.
type GameEvent interface {
	gameEvent() // marker method — prevents external implementations
}

// NarrativeEvent corresponds to UI.DisplayNarrative.
type NarrativeEvent struct {
	Text string
}

func (NarrativeEvent) gameEvent() {}

// DialogEvent corresponds to UI.DisplayDialog.
type DialogEvent struct {
	PlayerInput string
	GMResponse  string
}

func (DialogEvent) gameEvent() {}

// SystemMessageEvent corresponds to UI.DisplaySystemMessage.
type SystemMessageEvent struct {
	Message string
}

func (SystemMessageEvent) gameEvent() {}

// ActionAttemptEvent corresponds to UI.DisplayActionAttempt.
type ActionAttemptEvent struct {
	Description string
}

func (ActionAttemptEvent) gameEvent() {}

// ActionResultEvent corresponds to UI.DisplayActionResult.
type ActionResultEvent struct {
	Skill      string
	SkillLevel string
	Bonuses    int
	Result     string
	Outcome    string
}

func (ActionResultEvent) gameEvent() {}

// SceneTransitionEvent corresponds to UI.DisplaySceneTransition.
type SceneTransitionEvent struct {
	Narrative    string
	NewSceneHint string
}

func (SceneTransitionEvent) gameEvent() {}

// GameOverEvent corresponds to UI.DisplayGameOver.
type GameOverEvent struct {
	Reason string
}

func (GameOverEvent) gameEvent() {}

// ConflictStartEvent corresponds to UI.DisplayConflictStart.
type ConflictStartEvent struct {
	ConflictType  string
	InitiatorName string
	Participants  []ConflictParticipantInfo
}

func (ConflictStartEvent) gameEvent() {}

// ConflictEscalationEvent corresponds to UI.DisplayConflictEscalation.
type ConflictEscalationEvent struct {
	FromType        string
	ToType          string
	TriggerCharName string
}

func (ConflictEscalationEvent) gameEvent() {}

// TurnAnnouncementEvent corresponds to UI.DisplayTurnAnnouncement.
type TurnAnnouncementEvent struct {
	CharacterName string
	TurnNumber    int
	IsPlayer      bool
}

func (TurnAnnouncementEvent) gameEvent() {}

// ConflictEndEvent corresponds to UI.DisplayConflictEnd.
type ConflictEndEvent struct {
	Reason string
}

func (ConflictEndEvent) gameEvent() {}

// CharacterDisplayEvent corresponds to UI.DisplayCharacter.
type CharacterDisplayEvent struct{}

func (CharacterDisplayEvent) gameEvent() {}

// InvokePromptEvent is emitted when the engine needs the player to decide
// whether to invoke an aspect after a roll. The UI renders this and collects
// an InvokeResponse.
type InvokePromptEvent struct {
	Available     []InvokableAspect // Aspects the player may invoke
	FatePoints    int               // Player's current fate points
	CurrentResult string            // Current roll result (e.g. "Good (+3)")
	ShiftsNeeded  int               // Shifts needed to improve outcome
}

func (InvokePromptEvent) gameEvent() {}

// InputRequestEvent is emitted when the engine needs mid-flow input from the
// player (e.g. consequence choice after stress overflow, concession narration).
// The UI renders the appropriate control based on Type and collects a
// MidFlowResponse.
type InputRequestEvent struct {
	Type    InputRequestType // "numbered_choice" or "free_text"
	Prompt  string           // Human-readable prompt text
	Options []InputOption    // For numbered_choice only; empty for free_text
	Context map[string]any   // Additional context (NPC names, consequence types, etc.)
}

func (InputRequestEvent) gameEvent() {}

// ---------------------------------------------------------------------------
// Composite mechanical events — structured game-state events that replace
// ad-hoc SystemMessageEvent calls in the conflict resolution pipeline.
// ---------------------------------------------------------------------------

// DefenseRollEvent is emitted when a target rolls an active defense.
type DefenseRollEvent struct {
	DefenderName string // Name of the defending character
	Skill        string // Defense skill used
	Result       string // Roll result (e.g. "Good (+3)")
}

func (DefenseRollEvent) gameEvent() {}

// StressAbsorptionDetail describes how stress was absorbed by a track.
type StressAbsorptionDetail struct {
	TrackType  string // "physical" or "mental"
	Shifts     int    // Shifts absorbed
	TrackState string // String representation of the stress track after absorption
}

// ConsequenceDetail describes a consequence taken during damage resolution.
type ConsequenceDetail struct {
	TargetName string // Character who took the consequence
	Severity   string // "mild", "moderate", "severe"
	Aspect     string // The consequence aspect text
	Absorbed   int    // Shifts absorbed by this consequence
}

// DamageResolutionEvent is emitted when damage is applied to a target (NPC).
// It rolls up stress absorption, consequences, and taken-out into one event.
type DamageResolutionEvent struct {
	TargetName        string                  // The defender/target's name
	TotalShifts       int                     // Shifts of stress dealt
	StressType        string                  // "physical" or "mental"
	Absorbed          *StressAbsorptionDetail // Non-nil if stress track absorbed damage
	Consequence       *ConsequenceDetail      // Non-nil if a consequence was taken
	RemainingAbsorbed *StressAbsorptionDetail // Non-nil if remaining shifts were absorbed after consequence
	TakenOut          bool                    // True if the target was taken out
	VictoryEnd        bool                    // True if this caused the conflict to end via victory
}

func (DamageResolutionEvent) gameEvent() {}

// PlayerAttackResultEvent is emitted when the player's attack resolves.
// Rolls up the shifts dealt, tie boost, or failure message into one event.
type PlayerAttackResultEvent struct {
	TargetName    string // Name of the target hit
	Shifts        int    // Shifts dealt (0 on tie/failure)
	IsTie         bool   // Attack resulted in a tie (boost granted)
	TargetMissing bool   // True if the target could not be found
	TargetHint    string // Name hint for missing target
}

func (PlayerAttackResultEvent) gameEvent() {}

// AspectCreatedEvent is emitted when Create an Advantage succeeds.
type AspectCreatedEvent struct {
	AspectName  string // The new situation aspect
	FreeInvokes int    // Number of free invokes granted
}

func (AspectCreatedEvent) gameEvent() {}

// NPCAttackEvent is emitted after an NPC's full attack sequence resolves
// (attack roll, defense roll, initial outcome, optional narrative).
type NPCAttackEvent struct {
	AttackerName   string // NPC name
	TargetName     string // Target name (usually the player)
	AttackSkill    string // Skill used for the attack
	AttackResult   string // Attack roll result
	DefenseSkill   string // Skill used for defense
	DefenseResult  string // Defense roll result
	FullDefense    bool   // Whether full defense bonus was applied
	InitialOutcome string // Outcome before invokes (e.g. "Success")
	FinalOutcome   string // Outcome after invokes (may differ from initial)
	Narrative      string // LLM-generated narrative for the attack
}

func (NPCAttackEvent) gameEvent() {}

// PlayerStressEvent is emitted when the player absorbs stress from an attack.
type PlayerStressEvent struct {
	Shifts     int    // Shifts of stress absorbed
	StressType string // "physical" or "mental"
	TrackState string // Stress track display after absorption
}

func (PlayerStressEvent) gameEvent() {}

// PlayerDefendedEvent is emitted when the player successfully defends (failure
// or tie result on the attacker's roll).
type PlayerDefendedEvent struct {
	IsTie bool // True if attacker tied (grants a boost)
}

func (PlayerDefendedEvent) gameEvent() {}

// PlayerConsequenceEvent is emitted when the player takes a consequence.
type PlayerConsequenceEvent struct {
	Severity        string                  // "mild", "moderate", "severe"
	Aspect          string                  // The consequence aspect text
	Absorbed        int                     // Shifts absorbed by the consequence
	RemainingShifts int                     // Remaining shifts after absorption (0 if fully absorbed)
	StressAbsorbed  *StressAbsorptionDetail // Non-nil if remaining shifts went to stress
}

func (PlayerConsequenceEvent) gameEvent() {}

// PlayerTakenOutEvent is emitted when the player is taken out of a conflict.
type PlayerTakenOutEvent struct {
	AttackerName string // Who took the player out
	Narrative    string // LLM-generated narrative for being taken out
	Outcome      string // "game_over", "transition", or "continue"
	NewSceneHint string // Hint for scene transition (when outcome is "transition")
}

func (PlayerTakenOutEvent) gameEvent() {}

// ConcessionEvent is emitted when the player concedes a conflict.
type ConcessionEvent struct {
	FatePointsGained  int // Total fate points gained
	ConsequenceCount  int // Number of consequences that contributed extra FP
	CurrentFatePoints int // Player's fate points after gain
}

func (ConcessionEvent) gameEvent() {}

// OutcomeChangedEvent is emitted when invokes change the final outcome of a roll.
type OutcomeChangedEvent struct {
	FinalOutcome string // The new outcome after invokes (e.g. "Success")
}

func (OutcomeChangedEvent) gameEvent() {}
