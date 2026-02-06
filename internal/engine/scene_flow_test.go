package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSceneEndReason_Constants(t *testing.T) {
	// Verify the constants have expected values
	assert.Equal(t, SceneEndReason("transition"), SceneEndTransition)
	assert.Equal(t, SceneEndReason("quit"), SceneEndQuit)
	assert.Equal(t, SceneEndReason("player_taken_out"), SceneEndPlayerTakenOut)
}

func TestSceneEndResult_Fields(t *testing.T) {
	result := SceneEndResult{
		Reason:         SceneEndTransition,
		TransitionHint: "the dusty streets",
		TakenOutChars:  []string{"npc_1", "npc_2"},
	}

	assert.Equal(t, SceneEndTransition, result.Reason)
	assert.Equal(t, "the dusty streets", result.TransitionHint)
	assert.Len(t, result.TakenOutChars, 2)
}

func TestParseSceneTransitionMarker_WithLocation(t *testing.T) {
	response := "The swinging doors creak as you step out into the afternoon sun. [SCENE_TRANSITION:the dusty streets of Redemption Gulch]"
	transition, cleanedResponse := ParseSceneTransitionMarker(response)

	assert.NotNil(t, transition)
	assert.Equal(t, "the dusty streets of Redemption Gulch", transition.Hint)
	assert.Equal(t, "The swinging doors creak as you step out into the afternoon sun.", cleanedResponse)
}

func TestParseSceneTransitionMarker_ShortHint(t *testing.T) {
	response := "You mount your horse and ride off. [SCENE_TRANSITION:the road ahead]"
	transition, cleanedResponse := ParseSceneTransitionMarker(response)

	assert.NotNil(t, transition)
	assert.Equal(t, "the road ahead", transition.Hint)
	assert.Equal(t, "You mount your horse and ride off.", cleanedResponse)
}

func TestParseSceneTransitionMarker_NoMarker(t *testing.T) {
	response := "You walk over to the window and look outside."
	transition, cleanedResponse := ParseSceneTransitionMarker(response)

	assert.Nil(t, transition)
	assert.Equal(t, "You walk over to the window and look outside.", cleanedResponse)
}

func TestParseSceneTransitionMarker_MarkerInMiddle(t *testing.T) {
	response := "With a tip of your hat, you exit the saloon. [SCENE_TRANSITION:outside] The bright sun blinds you momentarily."
	transition, cleanedResponse := ParseSceneTransitionMarker(response)

	assert.NotNil(t, transition)
	assert.Equal(t, "outside", transition.Hint)
	assert.Equal(t, "With a tip of your hat, you exit the saloon. The bright sun blinds you momentarily.", cleanedResponse)
}
