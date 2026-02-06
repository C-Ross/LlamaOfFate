package engine

import (
	"context"
	"testing"
	"time"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLLMClientForScenario provides predictable responses for scenario manager tests
type MockLLMClientForScenario struct {
	responses []string
	callIndex int
}

func (m *MockLLMClientForScenario) ChatCompletion(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	response := `{"scene_name": "Test Scene", "description": "A test scene.", "situation_aspects": ["Aspect 1"], "npcs": []}`
	if m.callIndex < len(m.responses) {
		response = m.responses[m.callIndex]
		m.callIndex++
	}
	return &llm.CompletionResponse{
		ID:      "test-response",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "test-model",
		Choices: []llm.CompletionResponseChoice{
			{
				Index: 0,
				Message: llm.Message{
					Role:    "assistant",
					Content: response,
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (m *MockLLMClientForScenario) ChatCompletionStream(ctx context.Context, req llm.CompletionRequest, handler llm.StreamHandler) error {
	return nil
}

func (m *MockLLMClientForScenario) GetModelInfo() llm.ModelInfo {
	return llm.ModelInfo{Name: "test-model"}
}

func TestNewScenarioManager(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	assert.NotNil(t, sm)
	assert.Equal(t, engine, sm.engine)
	assert.Equal(t, player, sm.player)
}

func TestScenarioManager_Run_RequiresEngine(t *testing.T) {
	sm := &ScenarioManager{}
	err := sm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "engine is required")
}

func TestScenarioManager_Run_RequiresLLM(t *testing.T) {
	engine, err := New() // No LLM
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	err = sm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM client is required")
}

func TestScenarioManager_Run_RequiresUI(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	err = sm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UI is required")
}

func TestScenarioManager_Run_RequiresPlayer(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	sm := &ScenarioManager{
		engine: engine,
		ui:     &MockUI{},
	}

	err = sm.Run(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "player character is required")
}

func TestScenarioManager_SetSettings(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	settings := ScenarioSettings{
		Genre:          "Western",
		SettingContext: "The Old West",
	}
	sm.SetSettings(settings)

	assert.Equal(t, "Western", sm.settings.Genre)
	assert.Equal(t, "The Old West", sm.settings.SettingContext)
}

func TestScenarioManager_SetInitialScene(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	testScene := scene.NewScene("scene1", "Test Scene", "A test scene")
	npc := character.NewCharacter("npc1", "Test NPC")

	sm.SetInitialScene(testScene, []*character.Character{npc})

	assert.Equal(t, testScene, sm.initialScene)
	assert.Len(t, sm.initialNPCs, 1)
	assert.Equal(t, npc, sm.initialNPCs[0])
}

func TestScenarioManager_parseGeneratedScene_Valid(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	jsonResponse := `{
		"scene_name": "The Dusty Trail",
		"description": "A winding path through the desert, heat waves shimmer in the distance.",
		"situation_aspects": ["Blazing Sun", "Rocky Terrain"],
		"npcs": [
			{"name": "Old Prospector", "high_concept": "Grizzled Desert Wanderer", "disposition": "friendly"}
		]
	}`

	generated, err := sm.parseGeneratedScene(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "The Dusty Trail", generated.SceneName)
	assert.Contains(t, generated.Description, "winding path")
	assert.Len(t, generated.SituationAspects, 2)
	assert.Equal(t, "Blazing Sun", generated.SituationAspects[0])
	assert.Len(t, generated.NPCs, 1)
	assert.Equal(t, "Old Prospector", generated.NPCs[0].Name)
	assert.Equal(t, "friendly", generated.NPCs[0].Disposition)
}

func TestScenarioManager_parseGeneratedScene_WithCodeBlock(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	// LLMs sometimes wrap JSON in markdown code blocks
	jsonResponse := "```json\n{\"scene_name\": \"Test\", \"description\": \"A test.\", \"situation_aspects\": [], \"npcs\": []}\n```"

	generated, err := sm.parseGeneratedScene(jsonResponse)
	require.NoError(t, err)

	assert.Equal(t, "Test", generated.SceneName)
}

func TestScenarioManager_parseGeneratedScene_MissingFields(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	// Missing scene_name
	jsonResponse := `{"description": "A test scene."}`

	_, err = sm.parseGeneratedScene(jsonResponse)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing scene_name")
}

func TestScenarioManager_parseGeneratedScene_InvalidJSON(t *testing.T) {
	engine, err := NewWithLLM(&MockLLMClientForScenario{})
	require.NoError(t, err)

	player := character.NewCharacter("player1", "Test Hero")
	sm := NewScenarioManager(engine, player)

	// Invalid JSON
	jsonResponse := "This is not JSON at all"

	_, err = sm.parseGeneratedScene(jsonResponse)
	assert.Error(t, err)
}

func TestScenarioSettings_Defaults(t *testing.T) {
	settings := ScenarioSettings{}
	assert.Equal(t, "", settings.Genre)
	assert.Equal(t, "", settings.SettingContext)
}

func TestGeneratedScene_EmptyNPCs(t *testing.T) {
	generated := GeneratedScene{
		SceneName:        "Empty Scene",
		Description:      "A scene with no NPCs",
		SituationAspects: []string{"Lonely"},
		NPCs:             []GeneratedNPC{},
	}

	assert.Equal(t, "Empty Scene", generated.SceneName)
	assert.Len(t, generated.NPCs, 0)
}
