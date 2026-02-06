package engine

import (
	"regexp"
	"strings"
)

// SceneEndReason indicates why a scene ended
type SceneEndReason string

const (
	// SceneEndTransition indicates the player moved to a new location/scene
	SceneEndTransition SceneEndReason = "transition"
	// SceneEndQuit indicates the player chose to quit
	SceneEndQuit SceneEndReason = "quit"
	// SceneEndPlayerTakenOut indicates the player was taken out
	SceneEndPlayerTakenOut SceneEndReason = "player_taken_out"
)

// SceneEndResult contains information about how and why a scene ended
type SceneEndResult struct {
	Reason         SceneEndReason
	TransitionHint string   // From [SCENE_TRANSITION:hint] marker, empty if not a transition
	TakenOutChars  []string // Character IDs taken out during the scene
}

// SceneTransition represents a detected scene exit/transition
type SceneTransition struct {
	Hint string // Where/what comes next (e.g., "streets of Redemption Gulch")
}

// sceneTransitionMarkerRegex matches [SCENE_TRANSITION:hint] markers for scene exits
var sceneTransitionMarkerRegex = regexp.MustCompile(`\[SCENE_TRANSITION:([^\]]+)\]`)

// ParseSceneTransitionMarker extracts a scene transition from LLM response and returns cleaned text
func ParseSceneTransitionMarker(response string) (*SceneTransition, string) {
	matches := sceneTransitionMarkerRegex.FindStringSubmatch(response)
	if matches == nil {
		return nil, response
	}

	transition := &SceneTransition{
		Hint: strings.TrimSpace(matches[1]),
	}

	// Remove the marker from the response and clean up
	cleanedResponse := sceneTransitionMarkerRegex.ReplaceAllString(response, "")
	cleanedResponse = strings.Join(strings.Fields(cleanedResponse), " ")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	return transition, cleanedResponse
}
