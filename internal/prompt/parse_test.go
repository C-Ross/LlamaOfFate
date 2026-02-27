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

func TestParseGeneratedScene_ValidJSON(t *testing.T) {
	input := `{
		"scene_name": "The Dockside Tavern",
		"description": "A rowdy tavern near the harbor, thick with smoke and sailors.",
		"purpose": "Will the player find the smuggler before the ship departs?",
		"opening_hook": "A fight breaks out near the bar",
		"situation_aspects": ["Crowded Bar", "Smoke-Filled Room"],
		"npcs": [
			{"name": "One-Eye Pete", "high_concept": "Grizzled Dock Worker", "disposition": "neutral"}
		]
	}`

	result, err := ParseGeneratedScene(input)
	require.NoError(t, err)
	assert.Equal(t, "The Dockside Tavern", result.SceneName)
	assert.Equal(t, "A rowdy tavern near the harbor, thick with smoke and sailors.", result.Description)
	assert.Equal(t, "Will the player find the smuggler before the ship departs?", result.Purpose)
	assert.Equal(t, "A fight breaks out near the bar", result.OpeningHook)
	assert.Equal(t, []string{"Crowded Bar", "Smoke-Filled Room"}, result.SituationAspects)
	require.Len(t, result.NPCs, 1)
	assert.Equal(t, "One-Eye Pete", result.NPCs[0].Name)
	assert.Equal(t, "Grizzled Dock Worker", result.NPCs[0].HighConcept)
	assert.Equal(t, "neutral", result.NPCs[0].Disposition)
}

func TestParseGeneratedScene_JSONEmbeddedInText(t *testing.T) {
	input := `Sure, here's the scene:
	{
		"scene_name": "The Alley",
		"description": "A dark narrow alley.",
		"purpose": "Can the hero escape the pursuers?",
		"situation_aspects": [],
		"npcs": []
	}
	End of scene.`

	result, err := ParseGeneratedScene(input)
	require.NoError(t, err)
	assert.Equal(t, "The Alley", result.SceneName)
	assert.Equal(t, "A dark narrow alley.", result.Description)
}

func TestParseGeneratedScene_MissingSceneName(t *testing.T) {
	input := `{"description": "A scene.", "purpose": "A purpose.", "situation_aspects": [], "npcs": []}`

	_, err := ParseGeneratedScene(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing scene_name")
}

func TestParseGeneratedScene_MissingDescription(t *testing.T) {
	input := `{"scene_name": "Test", "purpose": "A purpose.", "situation_aspects": [], "npcs": []}`

	_, err := ParseGeneratedScene(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing description")
}

func TestParseGeneratedScene_MissingPurpose(t *testing.T) {
	input := `{"scene_name": "Test", "description": "A description.", "situation_aspects": [], "npcs": []}`

	_, err := ParseGeneratedScene(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing purpose")
}

func TestParseGeneratedScene_InvalidJSON(t *testing.T) {
	input := `not json at all`

	_, err := ParseGeneratedScene(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}

func TestParseGeneratedScene_InvalidJSONEmbedded(t *testing.T) {
	input := `{ invalid json }`

	_, err := ParseGeneratedScene(input)
	require.Error(t, err)
}

func TestParseSceneSummary_ValidJSON(t *testing.T) {
	input := `{
		"scene_description": "A tense standoff at the saloon",
		"key_events": ["Hero disarmed the bandit", "Sheriff arrived"],
		"npcs_encountered": [{"name": "Black Jack", "attitude": "hostile"}],
		"aspects_discovered": ["Wanted Dead or Alive"],
		"unresolved_threads": ["The real mastermind is still out there"],
		"how_ended": "transition",
		"narrative_prose": "The dust settled as the lawman rode into town."
	}`

	result, err := ParseSceneSummary(input)
	require.NoError(t, err)
	assert.Equal(t, "A tense standoff at the saloon", result.SceneDescription)
	assert.Equal(t, []string{"Hero disarmed the bandit", "Sheriff arrived"}, result.KeyEvents)
	assert.Equal(t, "The dust settled as the lawman rode into town.", result.NarrativeProse)
}

func TestParseSceneSummary_JSONEmbeddedInText(t *testing.T) {
	input := `Here is the summary: {"scene_description": "ok", "narrative_prose": "It happened."} Done.`

	result, err := ParseSceneSummary(input)
	require.NoError(t, err)
	assert.Equal(t, "It happened.", result.NarrativeProse)
}

func TestParseSceneSummary_MissingNarrativeProse(t *testing.T) {
	input := `{"scene_description": "A scene.", "narrative_prose": ""}`

	_, err := ParseSceneSummary(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing narrative_prose")
}

func TestParseSceneSummary_InvalidJSON(t *testing.T) {
	input := `this is not json`

	_, err := ParseSceneSummary(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}

func TestParseSceneSummary_InvalidJSONEmbedded(t *testing.T) {
	input := `{ bad json }`

	_, err := ParseSceneSummary(input)
	require.Error(t, err)
}

func TestParseScenarioResolution_ValidJSON(t *testing.T) {
	input := `{
		"is_resolved": true,
		"answered_questions": ["Was the villain stopped?"],
		"reasoning": "The player defeated the final boss."
	}`

	result, err := ParseScenarioResolution(input)
	require.NoError(t, err)
	assert.True(t, result.IsResolved)
	assert.Equal(t, []string{"Was the villain stopped?"}, result.AnsweredQuestions)
	assert.Equal(t, "The player defeated the final boss.", result.Reasoning)
}

func TestParseScenarioResolution_JSONEmbeddedInText(t *testing.T) {
	input := `Analysis: {"is_resolved": false, "answered_questions": [], "reasoning": "Still ongoing."} End.`

	result, err := ParseScenarioResolution(input)
	require.NoError(t, err)
	assert.False(t, result.IsResolved)
	assert.Equal(t, "Still ongoing.", result.Reasoning)
}

func TestParseScenarioResolution_InvalidJSON(t *testing.T) {
	input := `not json`

	_, err := ParseScenarioResolution(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}

func TestParseScenarioResolution_InvalidJSONEmbedded(t *testing.T) {
	input := `{ bad json }`

	_, err := ParseScenarioResolution(input)
	require.Error(t, err)
}

func TestParseScenario_ValidJSON(t *testing.T) {
	input := `{
		"title": "The Lost Crown",
		"problem": "The kingdom's crown has been stolen by a rogue sorcerer.",
		"story_questions": ["Can the hero recover the crown before coronation?", "Will the sorcerer be brought to justice?"],
		"setting": "A medieval kingdom in turmoil",
		"genre": "Fantasy"
	}`

	result, err := ParseScenario(input)
	require.NoError(t, err)
	assert.Equal(t, "The Lost Crown", result.Title)
	assert.Equal(t, "The kingdom's crown has been stolen by a rogue sorcerer.", result.Problem)
	assert.Len(t, result.StoryQuestions, 2)
	assert.Equal(t, "A medieval kingdom in turmoil", result.Setting)
	assert.Equal(t, "Fantasy", result.Genre)
}

func TestParseScenario_JSONEmbeddedInText(t *testing.T) {
	input := `Generated scenario: {"title": "Dark Roads", "problem": "Bandits terrorize the highway."} Done.`

	result, err := ParseScenario(input)
	require.NoError(t, err)
	assert.Equal(t, "Dark Roads", result.Title)
	assert.Equal(t, "Bandits terrorize the highway.", result.Problem)
}

func TestParseScenario_MissingTitle(t *testing.T) {
	input := `{"problem": "A problem exists.", "story_questions": []}`

	_, err := ParseScenario(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing title")
}

func TestParseScenario_MissingProblem(t *testing.T) {
	input := `{"title": "A Title", "story_questions": []}`

	_, err := ParseScenario(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing problem")
}

func TestParseScenario_InvalidJSON(t *testing.T) {
	input := `not json`

	_, err := ParseScenario(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON")
}

func TestParseScenario_InvalidJSONEmbedded(t *testing.T) {
	input := `{ invalid json }`

	_, err := ParseScenario(input)
	require.Error(t, err)
}
