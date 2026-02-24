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
// When SceneName is set, the UI renders a scene header before the text.
type NarrativeEvent struct {
	Text      string
	SceneName string // Optional: scene name for header display
	Purpose   string // Optional: scene purpose/dramatic question for this scene
}

func (NarrativeEvent) gameEvent() {}

// DialogEvent corresponds to UI.DisplayDialog.
type DialogEvent struct {
	PlayerInput string
	GMResponse  string
	IsRecap     bool   // True when replaying prior conversation on resume
	RecapType   string // "dialog", "action", "conflict" — set when IsRecap is true
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
	Skill        string
	SkillRank    string // Fate ladder name, e.g. "Average"
	SkillBonus   int    // Numeric skill level, e.g. 1
	Bonuses      int
	Result       string // Legacy display string (kept for terminal UI)
	Outcome      string
	DiceFaces    []int  `json:"DiceFaces,omitempty"` // Individual die values (-1, 0, +1)
	Total        int    // Final roll value (dice + skill + bonuses)
	TotalRank    string // Fate ladder name of Total, e.g. "Fair"
	Difficulty   int    // Opposition difficulty or defense value
	DiffRank     string // Fate ladder name of Difficulty, e.g. "Fair"
	DefenderName string // Non-empty when rolling against a character's active defense
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
	DiceFaces    []int  `json:"DiceFaces,omitempty"` // Individual die values (-1, 0, +1)
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
	Severity string // "mild", "moderate", "severe"
	Aspect   string // The consequence aspect text
	Absorbed int    // Shifts absorbed by this consequence
}

// DamageResolutionEvent is emitted when damage is applied to a target (NPC).
// It rolls up stress absorption, consequences, and taken-out into one event.
type DamageResolutionEvent struct {
	TargetName        string                  // The defender/target's name
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

// AspectCreatedEvent is emitted when Create an Advantage succeeds or a boost is granted.
type AspectCreatedEvent struct {
	AspectName  string // The new situation aspect
	FreeInvokes int    // Number of free invokes granted
	IsBoost     bool   // True if this is a boost (temporary, auto-removed after free invoke is used)
}

func (AspectCreatedEvent) gameEvent() {}

// BoostExpiredEvent is emitted when a boost's single free invoke has been consumed
// and the boost is automatically removed from the scene per Fate Core rules
// ("A boost vanishes as soon as it's used for the first time.").
type BoostExpiredEvent struct {
	AspectName string // Name of the boost that was consumed and removed
}

func (BoostExpiredEvent) gameEvent() {}

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
	Severity       string                  // "mild", "moderate", "severe"
	Aspect         string                  // The consequence aspect text
	Absorbed       int                     // Shifts absorbed by the consequence
	StressAbsorbed *StressAbsorptionDetail // Non-nil if remaining shifts went to stress
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

// InvokeEvent is emitted when a player invokes an aspect (free or paid).
type InvokeEvent struct {
	AspectName     string // Name of the aspect invoked
	IsFree         bool   // True if using a free invoke
	IsReroll       bool   // True if rerolling (false = +2)
	FatePointsLeft int    // Remaining fate points after invocation (0 if free)
	NewRoll        string // New roll string (reroll only, empty for +2)
	NewTotal       string // New total after invoke
	Failed         bool   // True if the invoke failed (not enough FP)
}

func (InvokeEvent) gameEvent() {}

// NPCActionResultEvent is emitted when an NPC performs a non-attack action
// (defend, create advantage, overcome) during a conflict.
type NPCActionResultEvent struct {
	NPCName    string // Name of the acting NPC
	ActionType string // "defend", "create_advantage", "overcome"
	Skill      string // Skill used (empty for defend)
	RollResult string // Roll result (e.g. "Good (+3)"), empty for defend
	Difficulty string // Difficulty faced (e.g. "Fair"), empty for defend
	Outcome    string // "success", "success_with_style", "tie", "failure", empty for defend

	// Create Advantage specific
	AspectCreated string // Name of aspect created (success only)
	FreeInvokes   int    // Free invokes granted (success only)
}

func (NPCActionResultEvent) gameEvent() {}

// RecoveryEvent is emitted for between-scene consequence recovery.
type RecoveryEvent struct {
	Action     string // "healed" (consequence fully healed) or "roll" (recovery attempt)
	Aspect     string // The consequence aspect text
	Severity   string // "mild", "moderate", "severe"
	Skill      string // Skill used for recovery roll (roll only)
	RollResult int    // Roll result value (roll only)
	Difficulty string // Difficulty faced (roll only)
	Success    bool   // Whether the recovery roll succeeded (roll only)
}

func (RecoveryEvent) gameEvent() {}

// StressOverflowEvent is emitted when a character cannot absorb stress
// with their stress track.
type StressOverflowEvent struct {
	Shifts            int  // Shifts that couldn't be absorbed
	NoConsequences    bool // True if no consequences are available (taken out)
	RemainingOverflow bool // True if remaining shifts after consequence can't be absorbed
}

func (StressOverflowEvent) gameEvent() {}

// MilestoneEvent is emitted when the player reaches a scenario milestone
// (scenario complete). Contains mechanical results for the UI to present.
type MilestoneEvent struct {
	Type          string // "scenario_complete" (future: "session", "major")
	ScenarioTitle string // Title of the completed scenario
	FatePoints    int    // Player's fate points after refresh
}

func (MilestoneEvent) gameEvent() {}

// GameResumedEvent is emitted when a saved game is loaded and resumed.
type GameResumedEvent struct {
	ScenarioTitle string // Title of the scenario being resumed
	SceneName     string // Name of the scene being resumed
}

func (GameResumedEvent) gameEvent() {}

// ErrorNotificationEvent is emitted when a non-fatal error occurs that the
// player should be informed about (e.g. a save file was incompatible and the
// game started fresh instead of resuming). UIs should display this as a
// prominent, transient notification (e.g. a toast).
type ErrorNotificationEvent struct {
	Message string
}

func (ErrorNotificationEvent) gameEvent() {}

// ---------------------------------------------------------------------------
// Full-state snapshot — sent once after Start() so the UI can initialise.
// ---------------------------------------------------------------------------

// GameStateSnapshotEvent is emitted after Start() completes so that the web
// UI's sidebar can initialise character info, stress tracks, aspects, etc.
type GameStateSnapshotEvent struct {
	Player           PlayerSnapshot            `json:"player"`
	SceneName        string                    `json:"sceneName"`
	SituationAspects []SituationAspectSnapshot `json:"situationAspects"`
	NPCs             []NPCSnapshot             `json:"npcs"`
	InConflict       bool                      `json:"inConflict"`
	InChallenge      bool                      `json:"inChallenge"`
	ChallengeTasks   []ChallengeTaskInfo       `json:"challengeTasks,omitempty"`
}

func (GameStateSnapshotEvent) gameEvent() {}

// PlayerSnapshot is a serialisable view of the player character.
type PlayerSnapshot struct {
	Name         string                         `json:"name"`
	HighConcept  string                         `json:"highConcept"`
	Trouble      string                         `json:"trouble"`
	Aspects      []string                       `json:"aspects"`
	FatePoints   int                            `json:"fatePoints"`
	Refresh      int                            `json:"refresh"`
	StressTracks map[string]StressTrackSnapshot `json:"stressTracks"`
	Consequences []ConsequenceSnapshotEntry     `json:"consequences"`
}

// StressTrackSnapshot is a serialisable view of a stress track.
type StressTrackSnapshot struct {
	Boxes    []bool `json:"boxes"`
	MaxBoxes int    `json:"maxBoxes"`
}

// ConsequenceSnapshotEntry is a serialisable view of a consequence slot.
type ConsequenceSnapshotEntry struct {
	Severity   string `json:"severity"`
	Aspect     string `json:"aspect"`
	Recovering bool   `json:"recovering"`
}

// SituationAspectSnapshot is a serialisable view of a situation aspect.
type SituationAspectSnapshot struct {
	Name        string `json:"name"`
	FreeInvokes int    `json:"freeInvokes"`
	IsBoost     bool   `json:"isBoost,omitempty"`
}

// NPCSnapshot is a serialisable view of an NPC visible in the current scene.
type NPCSnapshot struct {
	Name        string   `json:"name"`
	HighConcept string   `json:"highConcept"`
	Aspects     []string `json:"aspects"`
	IsTakenOut  bool     `json:"isTakenOut"`
}

// ---------------------------------------------------------------------------
// Challenge events
// ---------------------------------------------------------------------------

// ChallengeStartEvent announces a new challenge.
type ChallengeStartEvent struct {
	Description string
	Tasks       []ChallengeTaskInfo
}

func (ChallengeStartEvent) gameEvent() {}

// ChallengeTaskInfo describes one task in a challenge (for UI display).
type ChallengeTaskInfo struct {
	ID          string
	Description string
	Skill       string
	Difficulty  string // Ladder name, e.g. "Good (+3)"
	Status      string // "pending", "succeeded", "failed", "tied", "succeeded_with_style"
}

// ChallengeTaskResultEvent announces the outcome of a single task.
type ChallengeTaskResultEvent struct {
	TaskID      string
	Description string
	Skill       string
	Outcome     string // "success", "failure", "tie", "success_with_style"
	Shifts      int
}

func (ChallengeTaskResultEvent) gameEvent() {}

// ChallengeCompleteEvent announces the overall challenge resolution.
type ChallengeCompleteEvent struct {
	Successes int
	Failures  int
	Ties      int
	Overall   string // "success", "partial", "failure"
	Narrative string // LLM-generated summary
}

func (ChallengeCompleteEvent) gameEvent() {}
