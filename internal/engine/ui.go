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
