package prompt

import (
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseChallengeMarker_Valid(t *testing.T) {
	response := `The building shakes as flames engulf the corridor. [CHALLENGE:Escape the burning building]`

	trigger, cleaned := ParseChallengeMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, "Escape the burning building", trigger.Description)
	assert.Equal(t, "The building shakes as flames engulf the corridor.", cleaned)
}

func TestParseChallengeMarker_NoMarker(t *testing.T) {
	response := "Just regular narrative without any challenge markers."

	trigger, cleaned := ParseChallengeMarker(response)

	assert.Nil(t, trigger)
	assert.Equal(t, response, cleaned)
}

func TestParseChallengeMarker_EmptyDescription(t *testing.T) {
	response := `Some text. [CHALLENGE: ]`

	trigger, cleaned := ParseChallengeMarker(response)

	assert.Nil(t, trigger, "should reject challenge with empty description")
	assert.Equal(t, response, cleaned)
}

func TestParseChallengeMarker_MarkerAtStart(t *testing.T) {
	response := `[CHALLENGE:Infiltrate the compound] And then more text.`

	trigger, cleaned := ParseChallengeMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, "Infiltrate the compound", trigger.Description)
	assert.Equal(t, "And then more text.", cleaned)
}

func TestParseChallengeMarker_MarkerInMiddle(t *testing.T) {
	response := `Before text. [CHALLENGE:Navigate the storm] After text.`

	trigger, cleaned := ParseChallengeMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, "Navigate the storm", trigger.Description)
	assert.Equal(t, "Before text. After text.", cleaned)
}

func TestParseChallengeMarker_WhitespaceInDescription(t *testing.T) {
	response := `Narrative. [CHALLENGE:  Prepare defenses before the siege  ]`

	trigger, cleaned := ParseChallengeMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, "Prepare defenses before the siege", trigger.Description)
	assert.Equal(t, "Narrative.", cleaned)
}

func TestParseSceneTransitionMarker_Valid(t *testing.T) {
	response := `You step outside into the rain. [SCENE_TRANSITION:The streets of Redemption Gulch] The city awaits.`

	transition, cleaned := ParseSceneTransitionMarker(response)

	require.NotNil(t, transition)
	assert.Equal(t, "The streets of Redemption Gulch", transition.Hint)
	assert.Equal(t, "You step outside into the rain. The city awaits.", cleaned)
}

func TestParseSceneTransitionMarker_NoMarker(t *testing.T) {
	response := "Regular narrative without any transition markers."

	transition, cleaned := ParseSceneTransitionMarker(response)

	assert.Nil(t, transition)
	assert.Equal(t, response, cleaned)
}

func TestParseSceneTransitionMarker_MarkerAtEnd(t *testing.T) {
	response := `The scene concludes. [SCENE_TRANSITION:The tavern]`

	transition, cleaned := ParseSceneTransitionMarker(response)

	require.NotNil(t, transition)
	assert.Equal(t, "The tavern", transition.Hint)
	assert.Equal(t, "The scene concludes.", cleaned)
}

func TestParseSceneTransitionMarker_TrimsWhitespace(t *testing.T) {
	response := `[SCENE_TRANSITION:  The dark forest  ]`

	transition, cleaned := ParseSceneTransitionMarker(response)

	require.NotNil(t, transition)
	assert.Equal(t, "The dark forest", transition.Hint)
	assert.Equal(t, "", cleaned)
}

func TestParseConflictMarker_Physical(t *testing.T) {
	response := `The thug lunges forward. [CONFLICT:physical:npc-thug-1] He means business.`

	trigger, cleaned := ParseConflictMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, scene.PhysicalConflict, trigger.Type)
	assert.Equal(t, "npc-thug-1", trigger.InitiatorID)
	assert.Equal(t, "The thug lunges forward. He means business.", cleaned)
}

func TestParseConflictMarker_Mental(t *testing.T) {
	response := `The manipulator smiles. [CONFLICT:mental:npc-villain]`

	trigger, cleaned := ParseConflictMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, scene.MentalConflict, trigger.Type)
	assert.Equal(t, "npc-villain", trigger.InitiatorID)
	assert.Equal(t, "The manipulator smiles.", cleaned)
}

func TestParseConflictMarker_NoMarker(t *testing.T) {
	response := "The guard watches you suspiciously but does nothing."

	trigger, cleaned := ParseConflictMarker(response)

	assert.Nil(t, trigger)
	assert.Equal(t, response, cleaned)
}

func TestParseConflictMarker_InvalidType(t *testing.T) {
	response := `Some text [CONFLICT:emotional:npc-1]`

	trigger, cleaned := ParseConflictMarker(response)

	assert.Nil(t, trigger)
	assert.Equal(t, response, cleaned)
}

func TestParseConflictEndMarker_Surrender(t *testing.T) {
	response := `The enemy throws down their weapon. [CONFLICT:end:surrender] The fight is over.`

	resolution, cleaned := ParseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "surrender", resolution.Reason)
	assert.Equal(t, "The enemy throws down their weapon. The fight is over.", cleaned)
}

func TestParseConflictEndMarker_Agreement(t *testing.T) {
	response := `They reach a truce. [CONFLICT:end:agreement]`

	resolution, cleaned := ParseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "agreement", resolution.Reason)
	assert.Equal(t, "They reach a truce.", cleaned)
}

func TestParseConflictEndMarker_Retreat(t *testing.T) {
	response := `The bandits flee into the night. [CONFLICT:end:retreat]`

	resolution, cleaned := ParseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "retreat", resolution.Reason)
	assert.Equal(t, "The bandits flee into the night.", cleaned)
}

func TestParseConflictEndMarker_Resolved(t *testing.T) {
	response := `[CONFLICT:end:resolved]`

	resolution, cleaned := ParseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "resolved", resolution.Reason)
	assert.Equal(t, "", cleaned)
}

func TestParseConflictEndMarker_NoMarker(t *testing.T) {
	response := "The conflict continues with no end in sight."

	resolution, cleaned := ParseConflictEndMarker(response)

	assert.Nil(t, resolution)
	assert.Equal(t, response, cleaned)
}

func TestParseConflictEndMarker_InvalidReason(t *testing.T) {
	response := `Battle ends. [CONFLICT:end:victory]`

	resolution, cleaned := ParseConflictEndMarker(response)

	assert.Nil(t, resolution)
	assert.Equal(t, response, cleaned)
}
