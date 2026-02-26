package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/C-Ross/LlamaOfFate/internal/config"
	"github.com/C-Ross/LlamaOfFate/internal/core/character"
	"github.com/C-Ross/LlamaOfFate/internal/core/dice"
	"github.com/C-Ross/LlamaOfFate/internal/engine"
	"github.com/C-Ross/LlamaOfFate/internal/llm"
	"github.com/C-Ross/LlamaOfFate/internal/uicontract"
)

// GameServer wraps the game engine as an MCP tool server.
type GameServer struct {
	llmClient llm.LLMClient
	presets   map[string]*config.LoadedScenario
	mcpServer *server.MCPServer

	mu      sync.Mutex
	session engine.GameSessionManager // current active game session
	gm      *engine.GameManager       // underlying GameManager (for state inspection)
}

// New creates a GameServer. If llmClient is nil, only preset games can be
// started (no LLM-generated scenarios). If configRoot is non-empty, presets
// are loaded from it (e.g. "configs").
func New(llmClient llm.LLMClient, configRoot string) (*GameServer, error) {
	gs := &GameServer{
		llmClient: llmClient,
		presets:   make(map[string]*config.LoadedScenario),
	}

	if configRoot != "" {
		loaded, err := config.LoadAll(configRoot)
		if err != nil {
			slog.Warn("failed to load presets", "error", err)
		} else {
			gs.presets = loaded
		}
	}

	gs.mcpServer = server.NewMCPServer(
		"LlamaOfFate",
		"0.1.0",
		server.WithInstructions(mcpInstructions),
	)

	gs.registerTools()
	return gs, nil
}

// MCPServer returns the underlying MCP server for use with stdio or HTTP transports.
func (gs *GameServer) MCPServer() *server.MCPServer {
	return gs.mcpServer
}

const mcpInstructions = `LlamaOfFate is a text-based RPG implementing the Fate Core System.

Workflow:
1. Call list_presets to see available scenarios.
2. Call start_game with a preset_id to begin a game.
3. Call handle_input with the player's action text to play.
4. When the response has awaiting_invoke=true, call provide_invoke_response.
5. When the response has awaiting_midflow=true, call provide_midflow_response.
6. Call get_game_state at any time to inspect character stats, aspects, stress, etc.
7. Call save_game to persist progress.

Events in responses are typed JSON objects with "type" and "data" fields.
Key event types: narrative, dialog, action_result, conflict_start, invoke_prompt, input_request, game_state_snapshot, game_over.`

// registerTools registers all MCP tools on the server.
func (gs *GameServer) registerTools() {
	gs.mcpServer.AddTool(listPresetsTool, gs.handleListPresets)
	gs.mcpServer.AddTool(startGameTool, gs.handleStartGame)
	gs.mcpServer.AddTool(handleInputTool, gs.handleHandleInput)
	gs.mcpServer.AddTool(invokeResponseTool, gs.handleInvokeResponse)
	gs.mcpServer.AddTool(midflowResponseTool, gs.handleMidflowResponse)
	gs.mcpServer.AddTool(getGameStateTool, gs.handleGetGameState)
	gs.mcpServer.AddTool(saveGameTool, gs.handleSaveGame)
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

var listPresetsTool = mcp.NewTool("list_presets",
	mcp.WithDescription("List available scenario presets that can be used with start_game."),
	mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(true),
		DestructiveHint: mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}),
)

var startGameTool = mcp.NewTool("start_game",
	mcp.WithDescription("Start a new game session with the given preset scenario and optional player overrides."),
	mcp.WithToolAnnotation(mcp.ToolAnnotation{
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}),
	mcp.WithString("preset_id",
		mcp.Required(),
		mcp.Description("Scenario preset ID from list_presets (e.g. 'saloon', 'heist', 'tower')."),
	),
	mcp.WithString("player_name",
		mcp.Description("Override the default player name."),
	),
	mcp.WithString("player_high_concept",
		mcp.Description("Override the default player high concept aspect."),
	),
	mcp.WithString("player_trouble",
		mcp.Description("Override the default player trouble aspect."),
	),
)

var handleInputTool = mcp.NewTool("handle_input",
	mcp.WithDescription("Send player input (action, dialog, or command) to the running game and receive events."),
	mcp.WithToolAnnotation(mcp.ToolAnnotation{
		DestructiveHint: mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}),
	mcp.WithString("text",
		mcp.Required(),
		mcp.Description("The player's input text (e.g. 'I search the room', 'I attack the guard', dialog text)."),
	),
)

var invokeResponseTool = mcp.NewTool("provide_invoke_response",
	mcp.WithDescription("Respond to an invoke prompt (awaiting_invoke=true). Choose an aspect to invoke or skip."),
	mcp.WithToolAnnotation(mcp.ToolAnnotation{
		DestructiveHint: mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}),
	mcp.WithNumber("aspect_index",
		mcp.Required(),
		mcp.Description("Index into the available aspects array from the invoke_prompt event. Use -1 to skip."),
	),
	mcp.WithBoolean("is_reroll",
		mcp.Description("True to reroll dice, false (default) for +2 bonus."),
	),
)

var midflowResponseTool = mcp.NewTool("provide_midflow_response",
	mcp.WithDescription("Respond to a mid-flow prompt (awaiting_midflow=true). Provide a choice index or free text."),
	mcp.WithToolAnnotation(mcp.ToolAnnotation{
		DestructiveHint: mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}),
	mcp.WithNumber("choice_index",
		mcp.Description("0-based index for numbered_choice prompts. Ignored for free_text prompts."),
	),
	mcp.WithString("text",
		mcp.Description("Free-form text for free_text prompts. Ignored for numbered_choice prompts."),
	),
)

var getGameStateTool = mcp.NewTool("get_game_state",
	mcp.WithDescription("Get the current game state snapshot: player stats, scene, aspects, NPCs, stress tracks."),
	mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(true),
		DestructiveHint: mcp.ToBoolPtr(false),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}),
)

var saveGameTool = mcp.NewTool("save_game",
	mcp.WithDescription("Save the current game state to disk."),
	mcp.WithToolAnnotation(mcp.ToolAnnotation{
		DestructiveHint: mcp.ToBoolPtr(false),
		IdempotentHint:  mcp.ToBoolPtr(true),
		OpenWorldHint:   mcp.ToBoolPtr(false),
	}),
)

// ---------------------------------------------------------------------------
// inputResultJSON is the structured response for handle_input, invoke, midflow.
// ---------------------------------------------------------------------------

type inputResultJSON struct {
	Events          json.RawMessage `json:"events"`
	AwaitingInvoke  bool            `json:"awaiting_invoke"`
	AwaitingMidFlow bool            `json:"awaiting_midflow"`
	SceneEnded      bool            `json:"scene_ended"`
	GameOver        bool            `json:"game_over"`
}

func formatInputResult(result *engine.InputResult) (*mcp.CallToolResult, error) {
	eventsJSON, err := marshalEvents(result.Events)
	if err != nil {
		return nil, fmt.Errorf("marshal events: %w", err)
	}

	out := inputResultJSON{
		Events:          json.RawMessage(eventsJSON),
		AwaitingInvoke:  result.AwaitingInvoke,
		AwaitingMidFlow: result.AwaitingMidFlow,
		SceneEnded:      result.SceneEnded,
		GameOver:        result.GameOver,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(data)), nil
}

// ---------------------------------------------------------------------------
// Tool handlers
// ---------------------------------------------------------------------------

func (gs *GameServer) handleListPresets(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	type presetInfo struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Genre       string `json:"genre"`
		Description string `json:"description"`
	}

	presets := make([]presetInfo, 0, len(gs.presets))
	for id, ls := range gs.presets {
		presets = append(presets, presetInfo{
			ID:          id,
			Title:       ls.Raw.Title,
			Genre:       ls.Raw.Genre,
			Description: ls.Raw.Description,
		})
	}

	data, err := json.MarshalIndent(presets, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("failed to marshal presets"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func (gs *GameServer) handleStartGame(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	presetID := req.GetString("preset_id", "")
	if presetID == "" {
		return mcp.NewToolResultError("preset_id is required"), nil
	}

	ls, ok := gs.presets[presetID]
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("unknown preset: %q — call list_presets to see available options", presetID)), nil
	}

	gs.mu.Lock()
	defer gs.mu.Unlock()

	// Build the engine
	gameEngine, err := gs.newEngine()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("engine creation failed: %v", err)), nil
	}

	// Build player (allow overrides)
	player := clonePlayer(ls.Player)
	if name := req.GetString("player_name", ""); name != "" {
		player.Name = name
	}
	if hc := req.GetString("player_high_concept", ""); hc != "" {
		player.Aspects.HighConcept = hc
	}
	if trouble := req.GetString("player_trouble", ""); trouble != "" {
		player.Aspects.Trouble = trouble
	}

	gm := engine.NewGameManager(gameEngine)
	gm.SetPlayer(player)
	gm.SetScenario(ls.Scenario)

	// If the preset has an initial scene, configure it
	if ls.Scene != nil {
		gm.SetInitialScene(&engine.InitialSceneConfig{
			Scene: ls.Scene,
			NPCs:  ls.NPCs,
		})
	}

	events, err := gm.Start(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("start failed: %v", err)), nil
	}

	gs.session = gm
	gs.gm = gm

	eventsJSON, err := marshalEvents(events)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal events: %v", err)), nil
	}

	type startResult struct {
		Status string          `json:"status"`
		Events json.RawMessage `json:"events"`
	}
	out := startResult{
		Status: "started",
		Events: json.RawMessage(eventsJSON),
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal failed"), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func (gs *GameServer) handleHandleInput(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.session == nil {
		return mcp.NewToolResultError("no active game session — call start_game first"), nil
	}

	text := req.GetString("text", "")
	if text == "" {
		return mcp.NewToolResultError("text is required"), nil
	}

	result, err := gs.session.HandleInput(ctx, text)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("handle_input failed: %v", err)), nil
	}

	return formatInputResult(result)
}

func (gs *GameServer) handleInvokeResponse(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.session == nil {
		return mcp.NewToolResultError("no active game session — call start_game first"), nil
	}

	aspectIndex := req.GetInt("aspect_index", uicontract.InvokeSkip)
	isReroll := req.GetBool("is_reroll", false)

	resp := uicontract.InvokeResponse{
		AspectIndex: aspectIndex,
		IsReroll:    isReroll,
	}

	result, err := gs.session.ProvideInvokeResponse(ctx, resp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invoke_response failed: %v", err)), nil
	}

	return formatInputResult(result)
}

func (gs *GameServer) handleMidflowResponse(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.session == nil {
		return mcp.NewToolResultError("no active game session — call start_game first"), nil
	}

	resp := uicontract.MidFlowResponse{
		ChoiceIndex: req.GetInt("choice_index", 0),
		Text:        req.GetString("text", ""),
	}

	result, err := gs.session.ProvideMidFlowResponse(ctx, resp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("midflow_response failed: %v", err)), nil
	}

	return formatInputResult(result)
}

func (gs *GameServer) handleGetGameState(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.gm == nil {
		return mcp.NewToolResultError("no active game session — call start_game first"), nil
	}

	snapshot := gs.gm.BuildStateSnapshot()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal state: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func (gs *GameServer) handleSaveGame(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if gs.session == nil {
		return mcp.NewToolResultError("no active game session — call start_game first"), nil
	}

	if err := gs.session.Save(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("save failed: %v", err)), nil
	}

	return mcp.NewToolResultText(`{"status": "saved"}`), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (gs *GameServer) newEngine() (*engine.Engine, error) {
	if gs.llmClient != nil {
		return engine.NewWithLLM(gs.llmClient)
	}
	return engine.New()
}

// clonePlayer creates a shallow copy of the player character so that
// modifications (name/aspect overrides) don't mutate the preset.
func clonePlayer(src *character.Character) *character.Character {
	if src == nil {
		p := character.NewCharacter("player1", "Player")
		return p
	}
	cp := *src
	cp.Aspects = character.Aspects{
		HighConcept:  src.Aspects.HighConcept,
		Trouble:      src.Aspects.Trouble,
		OtherAspects: append([]string(nil), src.Aspects.OtherAspects...),
	}
	// Deep copy stress tracks
	cp.StressTracks = make(map[string]*character.StressTrack, len(src.StressTracks))
	for k, v := range src.StressTracks {
		track := *v
		track.Boxes = append([]bool(nil), v.Boxes...)
		cp.StressTracks[k] = &track
	}
	// Deep copy skills
	cp.Skills = make(map[string]dice.Ladder, len(src.Skills))
	for k, v := range src.Skills {
		cp.Skills[k] = v
	}
	return &cp
}
