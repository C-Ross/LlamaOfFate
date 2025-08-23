package engine

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
