package engine

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConflictManager(t *testing.T) {
	cm := newConflictManager(nil, nil, session.NullLogger{})
	require.NotNil(t, cm)
	assert.Nil(t, cm.llmClient)
	assert.Nil(t, cm.characters)
	assert.Nil(t, cm.takenOutChars)
}

func TestConflictManager_SetSceneState(t *testing.T) {
	cm := newConflictManager(nil, nil, session.NullLogger{})
	player := core.NewCharacter("p1", "Hero")
	s := scene.NewScene("s1", "Room", "A room")

	cm.setSceneState(s, player)

	assert.Equal(t, player, cm.player)
	assert.Equal(t, s, cm.currentScene)
}

func TestConflictManager_ResetState(t *testing.T) {
	cm := newConflictManager(nil, nil, session.NullLogger{})
	cm.takenOutChars = []string{"npc1"}

	cm.resetState()

	assert.Nil(t, cm.takenOutChars)
}

func TestSceneManager_ConflictManagerWiring(t *testing.T) {
	engine, err := New(session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	require.NotNil(t, sm.conflict, "ConflictManager should be created by NewSceneManager")

	player := core.NewCharacter("p1", "Hero")
	s := scene.NewScene("s1", "Room", "A room")
	err = sm.StartScene(s, player)
	require.NoError(t, err)

	assert.Equal(t, player, sm.conflict.player, "StartScene should wire player into ConflictManager")
	assert.Equal(t, s, sm.conflict.currentScene, "StartScene should wire scene into ConflictManager")
}

func TestSceneManager_ResetSceneState_ResetsConflictManager(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})
	sm.actions.pendingInvoke = &invokeState{}
	sm.actions.pendingMidFlow = &midFlowState{}
	sm.conflict.takenOutChars = []string{"npc1"}

	sm.resetSceneState()

	assert.Nil(t, sm.actions.pendingInvoke)
	assert.Nil(t, sm.actions.pendingMidFlow)
	assert.Nil(t, sm.conflict.takenOutChars)
}

func TestSceneManager_Restore_WiresConflictManager(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})
	player := core.NewCharacter("p1", "Hero")
	s := scene.NewScene("s1", "Room", "A room")

	state := SceneState{
		CurrentScene: s,
		ScenePurpose: "test purpose",
	}
	sm.Restore(state, player)

	assert.Equal(t, player, sm.conflict.player)
	assert.Equal(t, s, sm.conflict.currentScene)
}

func TestConflictManager_ConflictTypeString_DefaultPhysical(t *testing.T) {
	cm := newConflictManager(nil, nil, session.NullLogger{})
	s := scene.NewScene("s1", "Room", "A room")
	cm.setSceneState(s, core.NewCharacter("p1", "Hero"))

	assert.Equal(t, "physical", cm.conflictTypeString())
}

func TestConflictManager_ConflictTypeString_MentalConflict(t *testing.T) {
	cm := newConflictManager(nil, nil, session.NullLogger{})
	s := scene.NewScene("s1", "Room", "A room")
	s.ConflictState = &scene.ConflictState{Type: scene.MentalConflict}
	cm.setSceneState(s, core.NewCharacter("p1", "Hero"))

	assert.Equal(t, "mental", cm.conflictTypeString())
}

func TestConflictManager_ConflictTypeString_PhysicalConflict(t *testing.T) {
	cm := newConflictManager(nil, nil, session.NullLogger{})
	s := scene.NewScene("s1", "Room", "A room")
	s.ConflictState = &scene.ConflictState{Type: scene.PhysicalConflict}
	cm.setSceneState(s, core.NewCharacter("p1", "Hero"))

	assert.Equal(t, "physical", cm.conflictTypeString())
}
