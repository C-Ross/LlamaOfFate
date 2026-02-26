---
name: prompt-engineering-markers
description: Guide for modifying LLM prompt templates and adding marker parsing for structured LLM responses. Use this when asked to add markers, modify prompts, or parse LLM output.
---

# Prompt Engineering & Marker Parsing

This skill covers modifying LLM prompt templates and implementing marker parsing for structured responses.

## Prompt Templates

Located in `internal/prompt/templates/*.tmpl`. Each template has a corresponding data struct in `internal/prompt/data.go` and render function in `internal/prompt/templates.go`.

## Data Structures

Defined in `internal/prompt/data.go`:

```go
// For scene_response_prompt.tmpl
type SceneResponseData struct {
    Scene               *scene.Scene
    CharacterContext    string
    AspectsContext      string
    ConversationContext string
    PlayerInput         string
    InteractionType     string
    OtherCharacters     []*character.Character
    TakenOutCharacters  []*character.Character
}

// For input_classification_prompt.tmpl
type InputClassificationData struct {
    Scene       *scene.Scene
    PlayerInput string
}
```

## Render Functions

In `internal/prompt/templates.go`:

```go
prompt.RenderSceneResponse(data SceneResponseData) (string, error)
prompt.RenderInputClassification(data InputClassificationData) (string, error)
prompt.RenderActionNarrative(data ActionNarrativeData) (string, error)
prompt.RenderConflictResponse(data ConflictResponseData) (string, error)
prompt.RenderConsequenceAspect(data ConsequenceAspectData) (string, error)
prompt.RenderTakenOut(data TakenOutData) (string, error)
prompt.RenderNPCActionDecision(data NPCActionDecisionData) (string, error)
prompt.RenderNPCAttack(data NPCAttackData) (string, error)
prompt.RenderActionParse(data ActionParseTemplateData) (string, error)
prompt.RenderActionParseSystem() (string, error)
prompt.RenderAspectGeneration(data AspectGenerationRequest) (string, error)
prompt.RenderAspectGenerationSystem() (string, error)
prompt.RenderSceneGeneration(data SceneGenerationData) (string, error)
prompt.RenderSceneSummary(data SceneSummaryData) (string, error)
prompt.RenderScenarioGeneration(data ScenarioGenerationData) (string, error)
prompt.RenderScenarioResolution(data ScenarioResolutionData) (string, error)
prompt.RenderRecoveryNarrative(data RecoveryNarrativeData) (string, error)
```

## Adding New Markers

Follow these 6 steps to add a new marker:

### Step 1: Define regex, struct, and parser in `internal/prompt/markers.go`

All marker definitions live in `internal/prompt/markers.go` as exported package-level functions:

```go
// MyMarker represents a detected my marker
type MyMarker struct {
    Value string
}

// myMarkerRegex matches [MY_MARKER:value] markers
var myMarkerRegex = regexp.MustCompile(`\[MY_MARKER:([^\]]+)\]`)

// ParseMyMarker extracts a my marker from LLM response and returns cleaned text
func ParseMyMarker(response string) (*MyMarker, string) {
    matches := myMarkerRegex.FindStringSubmatch(response)
    if matches == nil {
        return nil, response
    }

    marker := &MyMarker{
        Value: strings.TrimSpace(matches[1]),
    }

    // Remove the marker from the response and clean up
    cleanedResponse := myMarkerRegex.ReplaceAllString(response, "")
    cleanedResponse = strings.Join(strings.Fields(cleanedResponse), " ")
    cleanedResponse = strings.TrimSpace(cleanedResponse)

    return marker, cleanedResponse
}
```

### Step 2: (Optional) Add convenience wrapper on SceneManager in `internal/engine/`

If the marker is conflict-related, add a thin wrapper in `internal/engine/conflict.go`:

```go
func (sm *SceneManager) parseMyMarker(response string) (*prompt.MyMarker, string) {
    return prompt.ParseMyMarker(response)
}
```

Otherwise, call `prompt.ParseMyMarker()` directly from the handler.

### Step 3: Update template with marker instructions

In the relevant template file under `internal/prompt/templates/`, add to the SCENE MARKERS section:

```
MY MARKER - Description of when to use:
If [condition], add at the end:
[MY_MARKER:value description]
Examples:
- Condition A → [MY_MARKER:example_value_a]
- Condition B → [MY_MARKER:example_value_b]
Only add this when [specific conditions].
```

### Step 4: Handle in `handleDialog()` in `internal/engine/scene_manager.go`

```go
// In handleDialog(), after other marker parsing:
myMarker, cleanedResponse := prompt.ParseMyMarker(cleanedResponse)

// Later, handle the marker:
if myMarker != nil {
    sm.handleMyMarker(myMarker)
}
```

### Step 5: Add unit tests

Add unit tests alongside the parser. If the parser is in `internal/prompt/markers.go`, add tests in `internal/prompt/` (or `internal/engine/conflict_test.go` if you added a wrapper):

```go
func TestParseMyMarker_WithValue(t *testing.T) {
    response := "Some narrative text. [MY_MARKER:test_value]"
    marker, cleanedResponse := prompt.ParseMyMarker(response)

    require.NotNil(t, marker)
    assert.Equal(t, "test_value", marker.Value)
    assert.Equal(t, "Some narrative text.", cleanedResponse)
}

func TestParseMyMarker_NoMarker(t *testing.T) {
    response := "Just regular text without markers."
    marker, cleanedResponse := prompt.ParseMyMarker(response)

    assert.Nil(t, marker)
    assert.Equal(t, "Just regular text without markers.", cleanedResponse)
}
```

### Step 6: Add LLM eval tests

Create tests in `test/llmeval/` to verify the LLM correctly uses the marker. See the `llm-eval-tests` skill for structure.

## Existing Markers

All marker regexes and parsers are in `internal/prompt/markers.go`:
- `sceneTransitionMarkerRegex` / `ParseSceneTransitionMarker()` — scene exits
- `conflictMarkerRegex` / `ParseConflictMarker()` — conflict escalation
- `conflictEndMarkerRegex` / `ParseConflictEndMarker()` — conflict de-escalation
- `challengeMarkerRegex` / `ParseChallengeMarker()` — challenge initiation

The engine layer in `internal/engine/conflict.go` has thin `SceneManager` method wrappers that delegate to these.

## Prompt Best Practices

1. **Hide markers from players**: Include "internal system use only - never explain to player"

2. **Provide clear examples**: Show when to use AND when NOT to use

3. **Keep format simple**: Use `[TYPE:value]` or `[TYPE:subtype:value]`

4. **Be specific about conditions**: "Only add this when the player is PHYSICALLY LEAVING"

5. **Test both cases**: Always test positive (should trigger) and negative (should NOT trigger)

6. **Critical instruction**: End marker section with:
   ```
   CRITICAL: Do NOT explain your reasoning about markers to the player.
   ```

## Template Variables

Access scene data in templates:

```gotmpl
{{.Scene.Name}}
{{.Scene.Description}}
{{.Scene.IsConflict}}
{{.PlayerInput}}
{{.CharacterContext}}
{{range $char := .OtherCharacters}}
  {{$char.Name}} ({{$char.ID}})
{{end}}
```
