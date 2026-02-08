package prompt

import (
	"bytes"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputClassificationTemplate(t *testing.T) {
	// Create test data
	testScene := scene.NewScene("test-scene", "Test Scene", "A test scene description")
	data := InputClassificationData{
		Scene:       testScene,
		PlayerInput: "What do I see?",
	}

	// Execute the template
	var buf bytes.Buffer
	err := InputClassificationPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail")

	result := buf.String()

	// Verify the template was populated correctly
	assert.Contains(t, result, "Test Scene", "Scene name should be included")
	assert.Contains(t, result, "A test scene description", "Scene description should be included")
	assert.Contains(t, result, "What do I see?", "Player input should be included")
	assert.Contains(t, result, "Fate Core principle", "Template should contain Fate Core guidance")
}

func TestSceneResponseTemplate(t *testing.T) {
	// Create test data
	testScene := scene.NewScene("test-scene", "Test Scene", "A test scene description")
	data := SceneResponseData{
		Scene:               testScene,
		CharacterContext:    "Test Character Context",
		AspectsContext:      "Test Aspects Context",
		ConversationContext: "Test Conversation Context",
		PlayerInput:         "Look around",
		InteractionType:     "clarification",
	}

	// Execute the template
	var buf bytes.Buffer
	err := SceneResponsePrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail")

	result := buf.String()

	// Verify the template was populated correctly
	assert.Contains(t, result, "Test Scene", "Scene name should be included")
	assert.Contains(t, result, "Test Character Context", "Character context should be included")
	assert.Contains(t, result, "Look around", "Player input should be included")
	assert.Contains(t, result, "clarification", "Interaction type should be included")
	assert.Contains(t, result, "FATE CORE GM PRINCIPLES", "Template should contain GM guidance")
}

func TestConsequenceAspectTemplate(t *testing.T) {
	// Create test data
	data := ConsequenceAspectData{
		CharacterName: "Hero",
		AttackerName:  "Dark Knight",
		Severity:      "moderate",
		ConflictType:  "physical",
		AttackContext: AttackContext{
			Skill:       "Fight",
			Description: "The Dark Knight's sword crashes down on your shield",
			Shifts:      4,
		},
	}

	// Execute the template
	var buf bytes.Buffer
	err := ConsequenceAspectPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail")

	result := buf.String()

	// Verify the template was populated correctly
	assert.Contains(t, result, "Hero", "Character name should be included")
	assert.Contains(t, result, "Dark Knight", "Attacker name should be included")
	assert.Contains(t, result, "moderate", "Severity should be included")
	assert.Contains(t, result, "physical", "Conflict type should be included")
	assert.Contains(t, result, "Fight", "Attack skill should be included")
	assert.Contains(t, result, "The Dark Knight's sword crashes down on your shield", "Attack description should be included")
	assert.Contains(t, result, "4", "Attack shifts should be included")
}

func TestConsequenceAspectTemplateWithoutAttackContext(t *testing.T) {
	// Create test data without attack context (optional fields)
	data := ConsequenceAspectData{
		CharacterName: "Hero",
		AttackerName:  "Dark Knight",
		Severity:      "mild",
		ConflictType:  "mental",
	}

	// Execute the template
	var buf bytes.Buffer
	err := ConsequenceAspectPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail even without attack context")

	result := buf.String()

	// Verify the template was populated with basic fields
	assert.Contains(t, result, "Hero", "Character name should be included")
	assert.Contains(t, result, "Dark Knight", "Attacker name should be included")
	assert.Contains(t, result, "mild", "Severity should be included")
	assert.Contains(t, result, "mental", "Conflict type should be included")
}

func TestTakenOutTemplate(t *testing.T) {
	// Create test data
	data := TakenOutData{
		CharacterName:       "Hero",
		AttackerName:        "Dark Knight",
		AttackerHighConcept: "Corrupted Champion of Darkness",
		ConflictType:        "physical",
		SceneDescription:    "A dark throne room with shadowy pillars",
		AttackContext: AttackContext{
			Skill:       "Fight",
			Description: "The Dark Knight's final blow strikes true",
			Shifts:      6,
		},
	}

	// Execute the template
	var buf bytes.Buffer
	err := TakenOutPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail")

	result := buf.String()

	// Verify the template was populated correctly
	assert.Contains(t, result, "Hero", "Character name should be included")
	assert.Contains(t, result, "Dark Knight", "Attacker name should be included")
	assert.Contains(t, result, "Corrupted Champion of Darkness", "Attacker high concept should be included")
	assert.Contains(t, result, "physical", "Conflict type should be included")
	assert.Contains(t, result, "A dark throne room with shadowy pillars", "Scene description should be included")
	assert.Contains(t, result, "Fight", "Attack skill should be included")
	assert.Contains(t, result, "The Dark Knight's final blow strikes true", "Attack description should be included")
	assert.Contains(t, result, "6", "Attack shifts should be included")
}

func TestTakenOutTemplateWithoutAttackContext(t *testing.T) {
	// Create test data without attack context (optional fields)
	data := TakenOutData{
		CharacterName:       "Hero",
		AttackerName:        "Dark Knight",
		AttackerHighConcept: "Corrupted Champion of Darkness",
		ConflictType:        "mental",
		SceneDescription:    "A dark throne room",
	}

	// Execute the template
	var buf bytes.Buffer
	err := TakenOutPrompt.Execute(&buf, data)
	require.NoError(t, err, "Template execution should not fail even without attack context")

	result := buf.String()

	// Verify the template was populated with basic fields
	assert.Contains(t, result, "Hero", "Character name should be included")
	assert.Contains(t, result, "Dark Knight", "Attacker name should be included")
	assert.Contains(t, result, "Corrupted Champion of Darkness", "Attacker high concept should be included")
	assert.Contains(t, result, "mental", "Conflict type should be included")
	assert.Contains(t, result, "A dark throne room", "Scene description should be included")
}

func TestRecoveryNarrativeTemplate(t *testing.T) {
	data := RecoveryNarrativeData{
		CharacterName: "Simon Falcon",
		SceneSetting:  "The crew rests after escaping the orbital station",
		Consequences: []RecoveryAttempt{
			{
				Aspect:     "Bruised Ribs",
				Severity:   "mild",
				Skill:      "Physique",
				RollResult: 3,
				Difficulty: "2",
				Outcome:    "success",
			},
			{
				Aspect:     "Shattered Confidence",
				Severity:   "moderate",
				Skill:      "Will",
				RollResult: 1,
				Difficulty: "4",
				Outcome:    "failure",
			},
		},
	}

	rendered, err := RenderRecoveryNarrative(data)
	require.NoError(t, err)

	assert.Contains(t, rendered, "Simon Falcon")
	assert.Contains(t, rendered, "Bruised Ribs")
	assert.Contains(t, rendered, "Shattered Confidence")
	assert.Contains(t, rendered, "success")
	assert.Contains(t, rendered, "failure")
}
