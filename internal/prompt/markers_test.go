package prompt

import (
	"testing"

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
