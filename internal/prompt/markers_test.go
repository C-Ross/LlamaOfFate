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
	response := `You step out of the tavern into the cold night. [SCENE_TRANSITION:streets of Redemption Gulch]`

	transition, cleaned := ParseSceneTransitionMarker(response)

	require.NotNil(t, transition)
	assert.Equal(t, "streets of Redemption Gulch", transition.Hint)
	assert.Equal(t, "You step out of the tavern into the cold night.", cleaned)
}

func TestParseSceneTransitionMarker_NoMarker(t *testing.T) {
	response := "Regular narrative with no transition marker."

	transition, cleaned := ParseSceneTransitionMarker(response)

	assert.Nil(t, transition)
	assert.Equal(t, response, cleaned)
}

func TestParseSceneTransitionMarker_MarkerAtStart(t *testing.T) {
	response := `[SCENE_TRANSITION:the abandoned mine] You follow the trail north.`

	transition, cleaned := ParseSceneTransitionMarker(response)

	require.NotNil(t, transition)
	assert.Equal(t, "the abandoned mine", transition.Hint)
	assert.Equal(t, "You follow the trail north.", cleaned)
}

func TestParseSceneTransitionMarker_MarkerInMiddle(t *testing.T) {
	response := `Before. [SCENE_TRANSITION:city docks] After.`

	transition, cleaned := ParseSceneTransitionMarker(response)

	require.NotNil(t, transition)
	assert.Equal(t, "city docks", transition.Hint)
	assert.Equal(t, "Before. After.", cleaned)
}

func TestParseSceneTransitionMarker_WhitespaceInHint(t *testing.T) {
	response := `Narrative. [SCENE_TRANSITION:  the old library  ]`

	transition, cleaned := ParseSceneTransitionMarker(response)

	require.NotNil(t, transition)
	assert.Equal(t, "the old library", transition.Hint)
	assert.Equal(t, "Narrative.", cleaned)
}

func TestParseConflictMarker_PhysicalConflict(t *testing.T) {
	response := `The thug draws his knife. [CONFLICT:physical:thug-01]`

	trigger, cleaned := ParseConflictMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, scene.PhysicalConflict, trigger.Type)
	assert.Equal(t, "thug-01", trigger.InitiatorID)
	assert.Equal(t, "The thug draws his knife.", cleaned)
}

func TestParseConflictMarker_MentalConflict(t *testing.T) {
	response := `The inquisitor's eyes bore into yours. [CONFLICT:mental:inquisitor-1]`

	trigger, cleaned := ParseConflictMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, scene.MentalConflict, trigger.Type)
	assert.Equal(t, "inquisitor-1", trigger.InitiatorID)
	assert.Equal(t, "The inquisitor's eyes bore into yours.", cleaned)
}

func TestParseConflictMarker_NoMarker(t *testing.T) {
	response := "No conflict here, just a peaceful scene."

	trigger, cleaned := ParseConflictMarker(response)

	assert.Nil(t, trigger)
	assert.Equal(t, response, cleaned)
}

func TestParseConflictMarker_MarkerAtStart(t *testing.T) {
	response := `[CONFLICT:physical:guard-01] The guard charges you.`

	trigger, cleaned := ParseConflictMarker(response)

	require.NotNil(t, trigger)
	assert.Equal(t, scene.PhysicalConflict, trigger.Type)
	assert.Equal(t, "guard-01", trigger.InitiatorID)
	assert.Equal(t, "The guard charges you.", cleaned)
}

func TestParseConflictEndMarker_Surrender(t *testing.T) {
	response := `The bandit drops his weapon. [CONFLICT:end:surrender]`

	resolution, cleaned := ParseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "surrender", resolution.Reason)
	assert.Equal(t, "The bandit drops his weapon.", cleaned)
}

func TestParseConflictEndMarker_Agreement(t *testing.T) {
	response := `They reach a deal. [CONFLICT:end:agreement]`

	resolution, _ := ParseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "agreement", resolution.Reason)
}

func TestParseConflictEndMarker_Retreat(t *testing.T) {
	response := `The enemy flees. [CONFLICT:end:retreat]`

	resolution, _ := ParseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "retreat", resolution.Reason)
}

func TestParseConflictEndMarker_Resolved(t *testing.T) {
	response := `The situation calms. [CONFLICT:end:resolved]`

	resolution, _ := ParseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "resolved", resolution.Reason)
}

func TestParseConflictEndMarker_NoMarker(t *testing.T) {
	response := "Just narrative, no conflict end."

	resolution, cleaned := ParseConflictEndMarker(response)

	assert.Nil(t, resolution)
	assert.Equal(t, response, cleaned)
}

func TestParseConflictEndMarker_InvalidReason(t *testing.T) {
	response := `Text. [CONFLICT:end:unknown_reason]`

	resolution, cleaned := ParseConflictEndMarker(response)

	assert.Nil(t, resolution, "should not match invalid reason")
	assert.Equal(t, response, cleaned)
}

func TestParseConflictEndMarker_MarkerCleanedFromText(t *testing.T) {
	response := `Before. [CONFLICT:end:surrender] After.`

	resolution, cleaned := ParseConflictEndMarker(response)

	require.NotNil(t, resolution)
	assert.Equal(t, "Before. After.", cleaned)
}
