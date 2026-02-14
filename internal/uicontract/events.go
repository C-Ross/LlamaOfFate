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
