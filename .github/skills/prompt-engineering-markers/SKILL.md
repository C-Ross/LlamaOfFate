---
name: prompt-engineering-markers
description: Guide for modifying LLM prompt templates and adding marker parsing for structured LLM responses. Use this when asked to add markers, modify prompts, or parse LLM output.
---

# Prompt Engineering & Marker Parsing

This skill covers modifying LLM prompt templates and implementing marker parsing for structured responses.

## Prompt Templates

Located in `internal/engine/templates/*.tmpl`. Each template has a corresponding data struct and render function.

## Data Structures

Defined in `internal/engine/scene_manager.go`:

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

In `internal/engine/templates.go`:

```go
engine.RenderSceneResponse(data SceneResponseData) (string, error)
engine.RenderInputClassification(data InputClassificationData) (string, error)
engine.RenderActionNarrative(data ActionNarrativeData) (string, error)
engine.RenderConflictResponse(data ConflictResponseData) (string, error)
```

## Adding New Markers

Follow these 7 steps to add a new marker:

### Step 1: Define regex in `internal/engine/conflict.go`

```go
// myMarkerRegex matches [MY_MARKER:value] markers
var myMarkerRegex = regexp.MustCompile(`\[MY_MARKER:([^\]]+)\]`)
```

### Step 2: Add struct for parsed data

```go
// MyMarker represents a detected my marker
type MyMarker struct {
    Value string
}
```

### Step 3: Add parser function

```go
// parseMyMarker extracts a my marker from LLM response and returns cleaned text
func (sm *SceneManager) parseMyMarker(response string) (*MyMarker, string) {
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

### Step 4: Update template with marker instructions

In the template file, add to the SCENE MARKERS section:

```
MY MARKER - Description of when to use:
If [condition], add at the end:
[MY_MARKER:value description]
Examples:
- Condition A → [MY_MARKER:example_value_a]
- Condition B → [MY_MARKER:example_value_b]
Only add this when [specific conditions].
```

### Step 5: Handle in `handleDialog()` in `scene_manager.go`

```go
// In handleDialog(), after other marker parsing:
myMarker, cleanedResponse := sm.parseMyMarker(cleanedResponse)

// Later, handle the marker:
if myMarker != nil {
    sm.handleMyMarker(myMarker)
}
```

### Step 6: Add unit tests in `scene_manager_test.go`

```go
func TestSceneManager_ParseMyMarker_WithValue(t *testing.T) {
    engine, err := New()
    require.NoError(t, err)
    sm := NewSceneManager(engine)

    response := "Some narrative text. [MY_MARKER:test_value]"
    marker, cleanedResponse := sm.parseMyMarker(response)

    require.NotNil(t, marker)
    assert.Equal(t, "test_value", marker.Value)
    assert.Equal(t, "Some narrative text.", cleanedResponse)
}

func TestSceneManager_ParseMyMarker_NoMarker(t *testing.T) {
    engine, err := New()
    require.NoError(t, err)
    sm := NewSceneManager(engine)

    response := "Just regular text without markers."
    marker, cleanedResponse := sm.parseMyMarker(response)

    assert.Nil(t, marker)
    assert.Equal(t, "Just regular text without markers.", cleanedResponse)
}
```

### Step 7: Add LLM eval tests

Create tests in `test/llmeval/` to verify the LLM correctly uses the marker. See the `llm-eval-tests` skill for structure.

## Existing Markers

Search `conflict.go` for existing marker regexes (e.g., `conflictMarkerRegex`, `sceneTransitionMarkerRegex`) to see current patterns.

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
