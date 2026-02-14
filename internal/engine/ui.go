package engine

import (
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// Type aliases re-export the UI contract types so that code within the engine
// package (and external consumers of engine) can continue using the short names.
// The canonical definitions live in the uicontract package.

type UI = uicontract.UI
type SceneInfo = uicontract.SceneInfo
type SceneInfoSetter = uicontract.SceneInfoSetter
type ConflictParticipantInfo = uicontract.ConflictParticipantInfo
type InvokableAspect = uicontract.InvokableAspect
type InvokeChoice = uicontract.InvokeChoice
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
type CharacterDisplayEvent = uicontract.CharacterDisplayEvent
type InvokePromptEvent = uicontract.InvokePromptEvent
type InvokeResponse = uicontract.InvokeResponse
type InputRequestEvent = uicontract.InputRequestEvent
type InputRequestType = uicontract.InputRequestType
type InputOption = uicontract.InputOption
type MidFlowResponse = uicontract.MidFlowResponse
