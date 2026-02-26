package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- clonePlayer tests ---

func TestClonePlayer_NilSource(t *testing.T) {
	cp := clonePlayer(nil)
	require.NotNil(t, cp)
	assert.Equal(t, "Player", cp.Name)
}

func TestClonePlayer_DeepCopies(t *testing.T) {
	src := character.NewCharacter("p1", "Hero")
	src.Aspects.HighConcept = "Brave Warrior"
	src.Aspects.Trouble = "Quick to Anger"
	src.Aspects.OtherAspects = []string{"Well Connected"}
	src.Skills["Athletics"] = dice.Good

	cp := clonePlayer(src)

	// Values match
	assert.Equal(t, "Hero", cp.Name)
	assert.Equal(t, "Brave Warrior", cp.Aspects.HighConcept)
	assert.Equal(t, dice.Good, cp.Skills["Athletics"])

	// Mutations don't leak
	cp.Aspects.HighConcept = "Sneaky Thief"
	cp.Skills["Athletics"] = dice.Superb
	cp.Aspects.OtherAspects = append(cp.Aspects.OtherAspects, "Street Smart")

	assert.Equal(t, "Brave Warrior", src.Aspects.HighConcept)
	assert.Equal(t, dice.Good, src.Skills["Athletics"])
	assert.Len(t, src.Aspects.OtherAspects, 1)
}

// --- Tool handler tests (no active session) ---

func newTestServer(t *testing.T) *GameServer {
	t.Helper()
	gs, err := New(nil, "") // no LLM, no config root
	require.NoError(t, err)
	return gs
}

func callTool(t *testing.T, gs *GameServer, req mcp.CallToolRequest) *mcp.CallToolResult {
	t.Helper()
	handler := findHandler(gs, req.Params.Name)
	require.NotNil(t, handler, "handler not found for tool %q", req.Params.Name)
	result, err := handler(context.Background(), req)
	require.NoError(t, err)
	return result
}

// findHandler looks up a tool handler by name. mcp-go doesn't expose a direct
// lookup, so we use the MCPServer's internal state via AddTool registration.
// For testing, we re-implement the dispatch on GameServer.
func findHandler(gs *GameServer, name string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	switch name {
	case "list_presets":
		return gs.handleListPresets
	case "start_game":
		return gs.handleStartGame
	case "handle_input":
		return gs.handleHandleInput
	case "provide_invoke_response":
		return gs.handleInvokeResponse
	case "provide_midflow_response":
		return gs.handleMidflowResponse
	case "get_game_state":
		return gs.handleGetGameState
	case "save_game":
		return gs.handleSaveGame
	default:
		return nil
	}
}

func makeRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

func TestListPresets_Empty(t *testing.T) {
	gs := newTestServer(t)

	result := callTool(t, gs, makeRequest("list_presets", nil))

	require.NotNil(t, result)
	assert.False(t, result.IsError)

	// Should be an empty JSON array
	text := extractText(t, result)
	assert.Equal(t, "[]", text)
}

func TestHandleInput_NoSession(t *testing.T) {
	gs := newTestServer(t)

	result := callTool(t, gs, makeRequest("handle_input", map[string]any{"text": "hello"}))

	require.NotNil(t, result)
	assert.True(t, result.IsError)
	text := extractText(t, result)
	assert.Contains(t, text, "no active game session")
}

func TestGetGameState_NoSession(t *testing.T) {
	gs := newTestServer(t)

	result := callTool(t, gs, makeRequest("get_game_state", nil))

	require.NotNil(t, result)
	assert.True(t, result.IsError)
	text := extractText(t, result)
	assert.Contains(t, text, "no active game session")
}

func TestSaveGame_NoSession(t *testing.T) {
	gs := newTestServer(t)

	result := callTool(t, gs, makeRequest("save_game", nil))

	require.NotNil(t, result)
	assert.True(t, result.IsError)
	text := extractText(t, result)
	assert.Contains(t, text, "no active game session")
}

func TestStartGame_UnknownPreset(t *testing.T) {
	gs := newTestServer(t)

	result := callTool(t, gs, makeRequest("start_game", map[string]any{"preset_id": "nonexistent"}))

	require.NotNil(t, result)
	assert.True(t, result.IsError)
	text := extractText(t, result)
	assert.Contains(t, text, "unknown preset")
}

func TestStartGame_MissingPresetID(t *testing.T) {
	gs := newTestServer(t)

	result := callTool(t, gs, makeRequest("start_game", nil))

	require.NotNil(t, result)
	assert.True(t, result.IsError)
	text := extractText(t, result)
	assert.Contains(t, text, "preset_id is required")
}

func TestInvokeResponse_NoSession(t *testing.T) {
	gs := newTestServer(t)

	result := callTool(t, gs, makeRequest("provide_invoke_response", map[string]any{"aspect_index": float64(-1)}))

	require.NotNil(t, result)
	assert.True(t, result.IsError)
	text := extractText(t, result)
	assert.Contains(t, text, "no active game session")
}

func TestMidflowResponse_NoSession(t *testing.T) {
	gs := newTestServer(t)

	result := callTool(t, gs, makeRequest("provide_midflow_response", map[string]any{"choice_index": float64(0)}))

	require.NotNil(t, result)
	assert.True(t, result.IsError)
	text := extractText(t, result)
	assert.Contains(t, text, "no active game session")
}

// --- FormatInputResult test ---

func TestFormatInputResult_Structure(t *testing.T) {
	ir := &engine.InputResult{
		Events: []uicontract.GameEvent{
			uicontract.NarrativeEvent{Text: "The door opens."},
			uicontract.DialogEvent{PlayerInput: "I enter", GMResponse: "Welcome"},
		},
		AwaitingInvoke:  false,
		AwaitingMidFlow: true,
		SceneEnded:      false,
		GameOver:        false,
	}

	result, err := formatInputResult(ir)
	require.NoError(t, err)
	require.NotNil(t, result)

	text := extractText(t, result)
	var out inputResultJSON
	err = json.Unmarshal([]byte(text), &out)
	require.NoError(t, err)

	assert.False(t, out.AwaitingInvoke)
	assert.True(t, out.AwaitingMidFlow)
	assert.False(t, out.SceneEnded)
	assert.False(t, out.GameOver)

	var events []eventEnvelope
	err = json.Unmarshal(out.Events, &events)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, "narrative", events[0].Type)
	assert.Equal(t, "dialog", events[1].Type)
}

// extractText pulls the text content from a CallToolResult.
func extractText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, result.Content)
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatal("no text content in result")
	return ""
}
