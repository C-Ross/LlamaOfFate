package engine

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConflictManager(t *testing.T) {
	cm := newConflictManager(nil, nil, nil, nil)
	require.NotNil(t, cm)
	assert.Nil(t, cm.llmClient)
	assert.Nil(t, cm.characters)
	assert.Nil(t, cm.pendingInvoke)
	assert.Nil(t, cm.pendingMidFlow)
	assert.Nil(t, cm.takenOutChars)
}

func TestConflictManager_SetSceneState(t *testing.T) {
	cm := newConflictManager(nil, nil, nil, nil)
	player := character.NewCharacter("p1", "Hero")
	s := scene.NewScene("s1", "Room", "A room")

	cm.setSceneState(s, player)

	assert.Equal(t, player, cm.player)
	assert.Equal(t, s, cm.currentScene)
}

func TestConflictManager_ResetState(t *testing.T) {
	cm := newConflictManager(nil, nil, nil, nil)
	cm.pendingInvoke = &invokeState{}
	cm.pendingMidFlow = &midFlowState{}
	cm.takenOutChars = []string{"npc1"}

	cm.resetState()

	assert.Nil(t, cm.pendingInvoke)
	assert.Nil(t, cm.pendingMidFlow)
	assert.Nil(t, cm.takenOutChars)
}

func TestSceneManager_ConflictManagerWiring(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	require.NotNil(t, sm.conflict, "ConflictManager should be created by NewSceneManager")

	player := character.NewCharacter("p1", "Hero")
	s := scene.NewScene("s1", "Room", "A room")
	err = sm.StartScene(s, player)
	require.NoError(t, err)

	assert.Equal(t, player, sm.conflict.player, "StartScene should wire player into ConflictManager")
	assert.Equal(t, s, sm.conflict.currentScene, "StartScene should wire scene into ConflictManager")
}

func TestSceneManager_ResetSceneState_ResetsConflictManager(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil)
	sm.conflict.pendingInvoke = &invokeState{}
	sm.conflict.pendingMidFlow = &midFlowState{}
	sm.conflict.takenOutChars = []string{"npc1"}

	sm.resetSceneState()

	assert.Nil(t, sm.conflict.pendingInvoke)
	assert.Nil(t, sm.conflict.pendingMidFlow)
	assert.Nil(t, sm.conflict.takenOutChars)
}

func TestSceneManager_Restore_WiresConflictManager(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil)
	player := character.NewCharacter("p1", "Hero")
	s := scene.NewScene("s1", "Room", "A room")

	state := SceneState{
		CurrentScene: s,
		ScenePurpose: "test purpose",
	}
	sm.Restore(state, player)

	assert.Equal(t, player, sm.conflict.player)
	assert.Equal(t, s, sm.conflict.currentScene)
}
