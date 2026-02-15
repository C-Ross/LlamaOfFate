package engine

import (
	"context"

	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// GameSessionManager is the async API surface for driving a game session.
// GameManager implements this interface. Consumers (e.g. the web package)
// depend on this interface rather than on *GameManager directly.
type GameSessionManager interface {
	Start(ctx context.Context) ([]GameEvent, error)
	HandleInput(ctx context.Context, input string) (*InputResult, error)
	ProvideInvokeResponse(ctx context.Context, resp InvokeResponse) (*InputResult, error)
	ProvideMidFlowResponse(ctx context.Context, resp MidFlowResponse) (*InputResult, error)
}

// Compile-time check: *GameManager implements GameSessionManager.
var _ GameSessionManager = (*GameManager)(nil)

// Type aliases re-export the UI contract types so that code within the engine
// package (and external consumers of engine) can continue using the short names.
// The canonical definitions live in the uicontract package.

type SceneInfo = uicontract.SceneInfo
type ConflictParticipantInfo = uicontract.ConflictParticipantInfo
type InvokableAspect = uicontract.InvokableAspect
type GameEvent = uicontract.GameEvent
type NarrativeEvent = uicontract.NarrativeEvent
type DialogEvent = uicontract.DialogEvent
type SystemMessageEvent = uicontract.SystemMessageEvent
type ActionAttemptEvent = uicontract.ActionAttemptEvent
type ActionResultEvent = uicontract.ActionResultEvent
type SceneTransitionEvent = uicontract.SceneTransitionEvent
type GameOverEvent = uicontract.GameOverEvent
type ConflictStartEvent = uicontract.ConflictStartEvent
type ConflictEscalationEvent = uicontract.ConflictEscalationEvent
type TurnAnnouncementEvent = uicontract.TurnAnnouncementEvent
type ConflictEndEvent = uicontract.ConflictEndEvent
type InvokePromptEvent = uicontract.InvokePromptEvent
type InvokeResponse = uicontract.InvokeResponse
type InputRequestEvent = uicontract.InputRequestEvent
type InputOption = uicontract.InputOption
type MidFlowResponse = uicontract.MidFlowResponse

// Composite mechanical event aliases
type DefenseRollEvent = uicontract.DefenseRollEvent
type StressAbsorptionDetail = uicontract.StressAbsorptionDetail
type ConsequenceDetail = uicontract.ConsequenceDetail
type DamageResolutionEvent = uicontract.DamageResolutionEvent
type PlayerAttackResultEvent = uicontract.PlayerAttackResultEvent
type AspectCreatedEvent = uicontract.AspectCreatedEvent
type NPCAttackEvent = uicontract.NPCAttackEvent
type PlayerStressEvent = uicontract.PlayerStressEvent
type PlayerDefendedEvent = uicontract.PlayerDefendedEvent
type PlayerConsequenceEvent = uicontract.PlayerConsequenceEvent
type PlayerTakenOutEvent = uicontract.PlayerTakenOutEvent
type ConcessionEvent = uicontract.ConcessionEvent
type OutcomeChangedEvent = uicontract.OutcomeChangedEvent
type InvokeEvent = uicontract.InvokeEvent
type NPCActionResultEvent = uicontract.NPCActionResultEvent
type RecoveryEvent = uicontract.RecoveryEvent
type StressOverflowEvent = uicontract.StressOverflowEvent
type MilestoneEvent = uicontract.MilestoneEvent
type GameResumedEvent = uicontract.GameResumedEvent
