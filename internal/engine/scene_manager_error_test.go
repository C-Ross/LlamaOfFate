package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core"
	"github.com/C-Ross/LlamaOfFate/internal/core/action"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/core/scene"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/session"
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
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

	_, err = sm.classifyInput(context.Background(), "test input")

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestClassifyInput_EmptyResponse(t *testing.T) {
	mockClient := &MockFailingLLMClient{err: nil} // Returns empty choices
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

	_, err = sm.classifyInput(context.Background(), "test input")

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestClassifyInput_UnexpectedClassification(t *testing.T) {
	mockClient := newTestLLMClient("invalid_type")
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

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
		{"unreasonable type", "unreasonable", inputTypeUnreasonable},
		{"uppercase dialog", "DIALOG", inputTypeDialog},
		{"mixed case action", "AcTiOn", inputTypeAction},
		{"mixed case narrative", "Narrative", inputTypeNarrative},
		{"mixed case unreasonable", "Unreasonable", inputTypeUnreasonable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newTestLLMClient(tt.response)
			engine, err := NewWithLLM(mockClient, session.NullLogger{})
			require.NoError(t, err)

			sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
			testScene := scene.NewScene("test", "Test", "Test scene")
			sm.currentScene = testScene
			sm.conflict.currentScene = testScene
			sm.actions.currentScene = testScene

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
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	player := core.NewCharacter("p1", "Player")
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	engine.AddCharacter(player)

	_, err = sm.generateSceneResponse(context.Background(), "hello", inputTypeDialog)

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestGenerateSceneResponse_EmptyResponse(t *testing.T) {
	mockClient := &MockFailingLLMClient{err: nil}
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	player := core.NewCharacter("p1", "Player")
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
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

	_, err := sm.GenerateActionNarrative(context.Background(), testAction)

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestGenerateActionNarrative_LLMError(t *testing.T) {
	mockClient := &MockFailingLLMClient{err: errors.New("connection refused")}
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	player := core.NewCharacter("p1", "Player")
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	engine.AddCharacter(player)

	testAction := action.NewAction("a1", "p1", action.Overcome, "Athletics", "Jump")
	testAction.Outcome = &dice.Outcome{Type: dice.Success}

	_, err = sm.GenerateActionNarrative(context.Background(), testAction)

	require.Error(t, err)
	assert.True(t, errors.Is(err, llm.ErrUnavailable))
}

func TestBuildMechanicalNarrative(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})

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

			narrative := sm.BuildMechanicalNarrative(testAction)

			assert.Contains(t, narrative, tt.description)
			assert.Contains(t, narrative, tt.expectSubstr)
		})
	}
}

func TestBuildMechanicalNarrative_NilOutcome(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})

	testAction := action.NewAction("a1", "p1", action.Overcome, "Athletics", "jump the gap")
	testAction.Outcome = nil

	narrative := sm.BuildMechanicalNarrative(testAction)

	assert.Contains(t, narrative, "jump the gap")
	assert.Contains(t, narrative, "attempt")
}

func TestBuildMechanicalNarrative_DefaultOutcome(t *testing.T) {
	sm := NewSceneManager(nil, nil, nil, session.NullLogger{})

	testAction := action.NewAction("a1", "p1", action.Overcome, "Athletics", "cross the bridge")
	testAction.Outcome = &dice.Outcome{Type: dice.OutcomeType(99)}

	narrative := sm.BuildMechanicalNarrative(testAction)

	assert.Contains(t, narrative, "cross the bridge")
	assert.Contains(t, narrative, "completes")
}

func TestProcessInput_ClassificationFallback(t *testing.T) {
	// When classification fails, it should default to dialog
	mockClient := &MockFailingLLMClient{err: errors.New("network error")}
	engine, err := NewWithLLM(mockClient, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	player := core.NewCharacter("p1", "Player")
	testScene := scene.NewScene("test", "Test", "Test scene")
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	engine.AddCharacter(player)

	// This should fail classification but still handle as dialog
	result, err := sm.HandleInput(context.Background(), "hello")
	require.NoError(t, err)

	// HandleInput should return a DialogEvent (fallback to dialog on classification failure)
	assert.NotNil(t, result)
}

func TestHandleInput_UnreasonableRoutesToHandleUnreasonable(t *testing.T) {
	// When classification returns "unreasonable", HandleInput should route to handleUnreasonable
	// and produce a DialogEvent with the GM's redirection response.
	client := newTestLLMClient(
		"unreasonable",
		"You reach out with your mind, but nothing happens. Perhaps you should try something more... practical.",
	)

	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})
	player := core.NewCharacter("player-1", "Jesse")
	player.Aspects.HighConcept = "Grizzled Rancher"
	engine.AddCharacter(player)

	testScene := scene.NewScene("saloon", "Dusty Saloon", "A Western saloon")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene
	sm.player = player
	sm.conflict.player = player
	sm.actions.player = player

	result, err := sm.HandleInput(context.Background(), "I use telekinesis")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should get a DialogEvent (not an ActionAttemptEvent)
	AssertHasEventIn[DialogEvent](t, result.Events)
	AssertNoEventIn[ActionAttemptEvent](t, result.Events)

	// Should NOT trigger invoke or mid-flow
	assert.False(t, result.AwaitingInvoke)
	assert.False(t, result.AwaitingMidFlow)

	// Verify conversation history recorded the exchange
	history := sm.GetConversationHistory()
	require.Len(t, history, 1)
	assert.Equal(t, "I use telekinesis", history[0].PlayerInput)
	assert.Equal(t, inputTypeUnreasonable, history[0].Type)
}

func TestSetGenre(t *testing.T) {
	sm := &SceneManager{}
	assert.Empty(t, sm.genre)

	sm.SetGenre("Western")
	assert.Equal(t, "Western", sm.genre)
}

func TestClassifyInput_IncludesCharacterAspectsAndGenre(t *testing.T) {
	// Verify that classifyInput passes character aspects and genre to the template.
	client := newTestLLMClient("action")

	engine, err := NewWithLLM(client, session.NullLogger{})
	require.NoError(t, err)

	sm := NewSceneManager(engine, engine.llmClient, engine.actionParser, session.NullLogger{})

	player := core.NewCharacter("p1", "Hero")
	player.Aspects.HighConcept = "Grizzled Rancher"
	player.Aspects.Trouble = "Too Old For This"
	player.Aspects.AddAspect("Quick Draw")
	sm.player = player

	sm.genre = "Western"

	testScene := scene.NewScene("test", "Test", "A dusty town")
	sm.currentScene = testScene
	sm.conflict.currentScene = testScene
	sm.actions.currentScene = testScene

	_, err = sm.classifyInput(context.Background(), "I draw my gun")
	require.NoError(t, err)

	// Check rendered prompt contains character aspects and genre
	require.NotEmpty(t, client.capturedPrompts)
	promptText := client.capturedPrompts[0]
	assert.Contains(t, promptText, "Grizzled Rancher")
	assert.Contains(t, promptText, "Quick Draw")
	assert.Contains(t, promptText, "Western")
}
