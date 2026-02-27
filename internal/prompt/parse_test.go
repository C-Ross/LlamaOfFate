package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClassification(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Clean responses
		{"simple word", "dialog", "dialog"},
		{"uppercase", "DIALOG", "dialog"},
		{"mixed case", "Narrative", "narrative"},

		// Whitespace
		{"leading/trailing spaces", "  clarification  ", "clarification"},
		{"leading/trailing newlines", "\naction\n", "action"},

		// Trailing explanations
		{"trailing explanation", "dialog - the player is speaking", "dialog"},
		{"newline after type", "action\nbecause there is opposition", "action"},
		{"tab after type", "narrative\tthis is mundane", "narrative"},

		// Markdown formatting
		{"markdown heading", "## narrative", "narrative"},
		{"markdown bold", "**action**", "action"},
		{"markdown heading with explanation", "## dialog because they are speaking", "dialog"},
		{"backtick wrapped", "`clarification`", "clarification"},
		{"double backtick", "``narrative``", "narrative"},

		// Quotes
		{"double quotes", "\"narrative\"", "narrative"},
		{"single quotes", "'action'", "action"},

		// Edge cases
		{"empty string", "", ""},
		{"only punctuation", "##**", ""},
		{"only spaces", "   ", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseClassification(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseFateNarration_ValidJSON(t *testing.T) {
	input := `{
		"fates": [
			{"id": "npc-1", "name": "Goblin", "description": "Killed", "permanent": true},
			{"id": "npc-2", "name": "Orc", "description": "Fled", "permanent": false}
		],
		"narrative": "The goblin fell lifeless. The orc scrambled away into the darkness."
	}`

	result, err := ParseFateNarration(input)
	require.NoError(t, err)
	require.Len(t, result.Fates, 2)

	assert.Equal(t, "npc-1", result.Fates[0].ID)
	assert.Equal(t, "Goblin", result.Fates[0].Name)
	assert.Equal(t, "Killed", result.Fates[0].Description)
	assert.True(t, result.Fates[0].Permanent)

	assert.Equal(t, "npc-2", result.Fates[1].ID)
	assert.Equal(t, "Orc", result.Fates[1].Name)
	assert.Equal(t, "Fled", result.Fates[1].Description)
	assert.False(t, result.Fates[1].Permanent)

	assert.Contains(t, result.Narrative, "goblin fell lifeless")
}

func TestParseFateNarration_JSONEmbeddedInText(t *testing.T) {
	input := `Here is the result:
	{
		"fates": [{"id": "npc-guard", "name": "Guard", "description": "Captured", "permanent": false}],
		"narrative": "You tie up the guard."
	}
	That's the output.`

	result, err := ParseFateNarration(input)
	require.NoError(t, err)
	require.Len(t, result.Fates, 1)
	assert.Equal(t, "npc-guard", result.Fates[0].ID)
	assert.Equal(t, "Guard", result.Fates[0].Name)
}

func TestParseFateNarration_MissingFates(t *testing.T) {
	input := `{"fates": [], "narrative": "Nothing happened."}`

	_, err := ParseFateNarration(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing fates")
}

func TestParseFateNarration_MissingNarrative(t *testing.T) {
	input := `{"fates": [{"id": "npc-1", "name": "Goblin", "description": "Dead", "permanent": true}], "narrative": ""}`

	_, err := ParseFateNarration(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing narrative")
}

func TestParseFateNarration_InvalidJSON(t *testing.T) {
	input := `not json at all`

	_, err := ParseFateNarration(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}

func TestParseGeneratedScene_Valid(t *testing.T) {
	input := `{
		"scene_name": "The Abandoned Warehouse",
		"description": "A dark, decrepit warehouse by the docks",
		"purpose": "Find the stolen artifact",
		"opening_hook": "A shadow moves near the crates",
		"situation_aspects": ["Darkness", "Broken Glass"],
		"npcs": [{"name": "Guard Dog", "high_concept": "Vicious Watchdog", "disposition": "hostile"}]
	}`

	result, err := ParseGeneratedScene(input)
	require.NoError(t, err)
	assert.Equal(t, "The Abandoned Warehouse", result.SceneName)
	assert.Equal(t, "A dark, decrepit warehouse by the docks", result.Description)
	assert.Equal(t, "Find the stolen artifact", result.Purpose)
	assert.Equal(t, "A shadow moves near the crates", result.OpeningHook)
	require.Len(t, result.SituationAspects, 2)
	require.Len(t, result.NPCs, 1)
	assert.Equal(t, "Guard Dog", result.NPCs[0].Name)
}

func TestParseGeneratedScene_JSONEmbeddedInText(t *testing.T) {
	input := `Here is the scene:
	{
		"scene_name": "The Tavern",
		"description": "A rowdy inn",
		"purpose": "Gather information",
		"situation_aspects": [],
		"npcs": []
	}
	End of response.`

	result, err := ParseGeneratedScene(input)
	require.NoError(t, err)
	assert.Equal(t, "The Tavern", result.SceneName)
}

func TestParseGeneratedScene_MissingSceneName(t *testing.T) {
	input := `{"description": "A place", "purpose": "A goal", "situation_aspects": [], "npcs": []}`

	_, err := ParseGeneratedScene(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing scene_name")
}

func TestParseGeneratedScene_MissingDescription(t *testing.T) {
	input := `{"scene_name": "Place", "purpose": "A goal", "situation_aspects": [], "npcs": []}`

	_, err := ParseGeneratedScene(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing description")
}

func TestParseGeneratedScene_MissingPurpose(t *testing.T) {
	input := `{"scene_name": "Place", "description": "A place", "situation_aspects": [], "npcs": []}`

	_, err := ParseGeneratedScene(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing purpose")
}

func TestParseGeneratedScene_InvalidJSON(t *testing.T) {
	input := `not valid json`

	_, err := ParseGeneratedScene(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}

func TestParseSceneSummary_Valid(t *testing.T) {
	input := `{
		"scene_description": "A tense standoff at the docks",
		"key_events": ["Player disarmed the guard", "Secret passage discovered"],
		"npcs_encountered": [{"name": "Dockmaster", "attitude": "neutral"}],
		"aspects_discovered": ["Hidden Contraband"],
		"unresolved_threads": ["Who hired the guards?"],
		"how_ended": "transition",
		"narrative_prose": "The player slipped past the guards and discovered the hidden passage."
	}`

	result, err := ParseSceneSummary(input)
	require.NoError(t, err)
	assert.Equal(t, "A tense standoff at the docks", result.SceneDescription)
	assert.Equal(t, "The player slipped past the guards and discovered the hidden passage.", result.NarrativeProse)
	require.Len(t, result.KeyEvents, 2)
	require.Len(t, result.NPCsEncountered, 1)
	assert.Equal(t, "Dockmaster", result.NPCsEncountered[0].Name)
}

func TestParseSceneSummary_MissingNarrativeProse(t *testing.T) {
	input := `{"scene_description": "A place", "key_events": [], "narrative_prose": ""}`

	_, err := ParseSceneSummary(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing narrative_prose")
}

func TestParseSceneSummary_InvalidJSON(t *testing.T) {
	input := `not valid json`

	_, err := ParseSceneSummary(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}

func TestParseSceneSummary_JSONEmbeddedInText(t *testing.T) {
	input := `Here is the summary:
	{
		"scene_description": "Docks",
		"key_events": [],
		"narrative_prose": "The adventure continued."
	}
	Done.`

	result, err := ParseSceneSummary(input)
	require.NoError(t, err)
	assert.Equal(t, "The adventure continued.", result.NarrativeProse)
}

func TestParseScenarioResolution_Valid(t *testing.T) {
	input := `{
		"is_resolved": true,
		"answered_questions": ["Who stole the artifact?", "Can the city be saved?"],
		"reasoning": "The player recovered the artifact and exposed the villain."
	}`

	result, err := ParseScenarioResolution(input)
	require.NoError(t, err)
	assert.True(t, result.IsResolved)
	require.Len(t, result.AnsweredQuestions, 2)
	assert.Equal(t, "The player recovered the artifact and exposed the villain.", result.Reasoning)
}

func TestParseScenarioResolution_NotResolved(t *testing.T) {
	input := `{"is_resolved": false, "answered_questions": [], "reasoning": "The main villain is still at large."}`

	result, err := ParseScenarioResolution(input)
	require.NoError(t, err)
	assert.False(t, result.IsResolved)
}

func TestParseScenarioResolution_InvalidJSON(t *testing.T) {
	input := `not valid json`

	_, err := ParseScenarioResolution(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}

func TestParseScenarioResolution_JSONEmbeddedInText(t *testing.T) {
	input := `Analysis: {"is_resolved": false, "answered_questions": [], "reasoning": "Incomplete."} End.`

	result, err := ParseScenarioResolution(input)
	require.NoError(t, err)
	assert.False(t, result.IsResolved)
	assert.Equal(t, "Incomplete.", result.Reasoning)
}

func TestParseScenario_Valid(t *testing.T) {
	input := `{
		"title": "The Missing Artifact",
		"problem": "A powerful relic has been stolen from the museum",
		"dramatic_questions": ["Who is behind the theft?", "Can the artifact be recovered?"],
		"setting": "A steampunk city in the 1890s",
		"opening_scene_hint": "The museum director contacts you"
	}`

	result, err := ParseScenario(input)
	require.NoError(t, err)
	assert.Equal(t, "The Missing Artifact", result.Title)
	assert.Equal(t, "A powerful relic has been stolen from the museum", result.Problem)
}

func TestParseScenario_MissingTitle(t *testing.T) {
	input := `{"problem": "Something bad happened", "dramatic_questions": []}`

	_, err := ParseScenario(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing title")
}

func TestParseScenario_MissingProblem(t *testing.T) {
	input := `{"title": "A Scenario", "dramatic_questions": []}`

	_, err := ParseScenario(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing problem")
}

func TestParseScenario_InvalidJSON(t *testing.T) {
	input := `not valid json`

	_, err := ParseScenario(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}

func TestParseScenario_JSONEmbeddedInText(t *testing.T) {
	input := `Generated scenario:
	{
		"title": "Dark Conspiracy",
		"problem": "The city council is controlled by a shadow organization",
		"dramatic_questions": ["Can the player expose the truth?"]
	}
	That's the scenario.`

	result, err := ParseScenario(input)
	require.NoError(t, err)
	assert.Equal(t, "Dark Conspiracy", result.Title)
	assert.Equal(t, "The city council is controlled by a shadow organization", result.Problem)
}
