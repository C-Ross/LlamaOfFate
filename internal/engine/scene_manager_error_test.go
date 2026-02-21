package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockFailingLLMClient simulates LLM failures
type MockFailingLLMClient struct {
	err error
}

func (m *MockFailingLLMClient) ChatCompletion(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llm.CompletionResponse{Choices: []llm.CompletionResponseChoice{}}, nil
}

func (m *MockFailingLLMClient) ChatCompletionStream(ctx context.Context, req llm.CompletionRequest, handler llm.StreamHandler) error {
	return m.err
}

func (m *MockFailingLLMClient) GetModelInfo() llm.ModelInfo {
	return llm.ModelInfo{Name: "mock-failing", Provider: "test"}
}

func TestClassifyInput_LLMUnavailable(t *testing.T) {
	sm := &SceneManager{
		llmClient: nil,
	}

	_, err := sm.classifyInput(context.Background(), "hello")

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestClassifyInput_LLMError(t *testing.T) {
	mockClient := &MockFailingLLMClient{err: errors.New("network error")}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene

	_, err = sm.classifyInput(context.Background(), "test input")

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestClassifyInput_EmptyResponse(t *testing.T) {
	mockClient := &MockFailingLLMClient{err: nil} // Returns empty choices
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene

	_, err = sm.classifyInput(context.Background(), "test input")

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestClassifyInput_UnexpectedClassification(t *testing.T) {
	mockClient := &MockLLMClient{response: "invalid_type"}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene

	_, err = sm.classifyInput(context.Background(), "test input")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected classification")
}

func TestClassifyInput_ValidTypes(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected string
	}{
		{"dialog type", "dialog", inputTypeDialog},
		{"clarification type", "clarification", inputTypeClarification},
		{"action type", "action", inputTypeAction},
		{"narrative type", "narrative", inputTypeNarrative},
		{"uppercase dialog", "DIALOG", inputTypeDialog},
		{"mixed case action", "AcTiOn", inputTypeAction},
		{"mixed case narrative", "Narrative", inputTypeNarrative},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{response: tt.response}
			engine, err := NewWithLLM(mockClient)
			require.NoError(t, err)

			sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
			testScene := scene.NewScene("test", "Test", "Test scene")
			sm.currentScene = testScene
			sm.conflict.currentScene = testScene

			result, err := sm.classifyInput(context.Background(), "test input")

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateSceneResponse_LLMUnavailable(t *testing.T) {
	sm := &SceneManager{
		llmClient: nil,
	}

	_, err := sm.generateSceneResponse(context.Background(), "hello", inputTypeDialog)

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestGenerateSceneResponse_LLMError(t *testing.T) {
	mockClient := &MockFailingLLMClient{err: errors.New("timeout")}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	player := character.NewCharacter("p1", "Player")
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.player = player
	sm.conflict.player = player
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	engine.AddCharacter(player)

	_, err = sm.generateSceneResponse(context.Background(), "hello", inputTypeDialog)

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestGenerateSceneResponse_EmptyResponse(t *testing.T) {
	mockClient := &MockFailingLLMClient{err: nil}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	player := character.NewCharacter("p1", "Player")
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.player = player
	sm.conflict.player = player
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	engine.AddCharacter(player)

	_, err = sm.generateSceneResponse(context.Background(), "hello", inputTypeDialog)

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestGenerateActionNarrative_LLMUnavailable(t *testing.T) {
	sm := &SceneManager{
		llmClient: nil,
	}

	testAction := action.NewAction("a1", "p1", action.Overcome, "Athletics", "Jump")
	testAction.Outcome = &dice.Outcome{Type: dice.Success}

	_, err := sm.generateActionNarrative(context.Background(), testAction)

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestGenerateActionNarrative_LLMError(t *testing.T) {
	mockClient := &MockFailingLLMClient{err: errors.New("connection refused")}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	player := character.NewCharacter("p1", "Player")
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.player = player
	sm.conflict.player = player
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	engine.AddCharacter(player)

	testAction := action.NewAction("a1", "p1", action.Overcome, "Athletics", "Jump")
	testAction.Outcome = &dice.Outcome{Type: dice.Success}

	_, err = sm.generateActionNarrative(context.Background(), testAction)

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestBuildMechanicalNarrative(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil)

	tests := []struct {
		name         string
		outcomeType  dice.OutcomeType
		description  string
		expectSubstr string
	}{
		{
			name:         "success with style",
			outcomeType:  dice.SuccessWithStyle,
			description:  "dodge the attack",
			expectSubstr: "succeeds brilliantly",
		},
		{
			name:         "success",
			outcomeType:  dice.Success,
			description:  "climb the wall",
			expectSubstr: "succeeds",
		},
		{
			name:         "tie",
			outcomeType:  dice.Tie,
			description:  "pick the lock",
			expectSubstr: "partially succeeds",
		},
		{
			name:         "failure",
			outcomeType:  dice.Failure,
			description:  "swim across",
			expectSubstr: "fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testAction := action.NewAction("a1", "p1", action.Overcome, "Athletics", tt.description)
			testAction.Outcome = &dice.Outcome{Type: tt.outcomeType}

			narrative := sm.buildMechanicalNarrative(testAction)

			assert.Contains(t, narrative, tt.description)
			assert.Contains(t, narrative, tt.expectSubstr)
		})
	}
}

func TestBuildMechanicalNarrative_NilOutcome(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil)

	testAction := action.NewAction("a1", "p1", action.Overcome, "Athletics", "jump the gap")
	testAction.Outcome = nil

	narrative := sm.buildMechanicalNarrative(testAction)

	assert.Contains(t, narrative, "jump the gap")
	assert.Contains(t, narrative, "attempt")
}

func TestBuildMechanicalNarrative_DefaultOutcome(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil)

	testAction := action.NewAction("a1", "p1", action.Overcome, "Athletics", "cross the bridge")
	testAction.Outcome = &dice.Outcome{Type: dice.OutcomeType(99)}

	narrative := sm.buildMechanicalNarrative(testAction)

	assert.Contains(t, narrative, "cross the bridge")
	assert.Contains(t, narrative, "completes")
}

func TestProcessInput_ClassificationFallback(t *testing.T) {
	// When classification fails, it should default to dialog
	mockClient := &MockFailingLLMClient{err: errors.New("network error")}
	engine, err := NewWithLLM(mockClient)
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser)
	player := character.NewCharacter("p1", "Player")
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.player = player
	sm.conflict.player = player
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	engine.AddCharacter(player)

	// This should fail classification but still handle as dialog
	result, err := sm.HandleInput(context.Background(), "hello")
	require.NoError(t, err)

	// HandleInput should return a DialogEvent (fallback to dialog on classification failure)
	assert.NotNil(t, result)
}
