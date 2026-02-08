---
name: llm-eval-tests
description: Guide for creating and running LLM evaluation tests that verify prompt behavior. Use this when asked to create LLM tests, evaluate prompts, or test LLM behavior.
---

# LLM Eval Tests

LLM eval tests verify that prompts produce expected LLM behavior. Located in `test/llmeval/`.

## File Structure

```go
//go:build llmeval

package llmeval_test

import (
    "context"
    "os"
    "testing"

    "github.com/C-Ross/LlamaOfFate/internal/core/character"
    "github.com/C-Ross/LlamaOfFate/internal/core/scene"
    "github.com/C-Ross/LlamaOfFate/internal/llm"
    "github.com/C-Ross/LlamaOfFate/internal/llm/azure"
    promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestCase struct with: Name, PlayerInput, SceneContext, Expected*, Description
type MyTestCase struct {
    Name             string
    PlayerInput      string
    SceneName        string
    SceneDescription string
    Expected*        string/bool  // Whatever you're testing
    Description      string       // Why this should pass/fail
}

// getPositiveTestCases() / getNegativeTestCases() functions
func getPositiveTestCases() []MyTestCase { ... }
func getNegativeTestCases() []MyTestCase { ... }

// evaluate*() function: renders template, calls LLM, checks result
func evaluateMyTest(ctx context.Context, client llm.LLMClient, tc MyTestCase) MyResult { ... }

// Main test with summary statistics at end
func TestMyFeature_LLMEvaluation(t *testing.T) { ... }
```

## Key Patterns

### Build Tag (REQUIRED)
First line must be the build tag, followed by blank line:
```go
//go:build llmeval

package llmeval_test
```

### Config and Client Setup
```go
if os.Getenv("AZURE_API_ENDPOINT") == "" || os.Getenv("AZURE_API_KEY") == "" {
    t.Skip("Skipping LLM evaluation test: AZURE_API_ENDPOINT and AZURE_API_KEY must be set")
}

config, err := azure.LoadConfig("../../configs/azure-llm.yaml")
require.NoError(t, err, "Failed to load Azure config")

client := azure.NewClient(*config)
ctx := context.Background()
```

### Template Rendering
Use `promptpkg.Render*()` functions (imported as `promptpkg "github.com/C-Ross/LlamaOfFate/internal/prompt"`):
```go
prompt, err := promptpkg.RenderSceneResponse(data)
prompt, err := promptpkg.RenderInputClassification(data)
prompt, err := promptpkg.RenderActionNarrative(data)
prompt, err := promptpkg.RenderActionParse(data)
prompt, err := promptpkg.RenderSceneGeneration(data)
prompt, err := promptpkg.RenderScenarioGeneration(data)
```

### Scene Aspects
The scene struct uses `SituationAspects`, not `Aspects`:
```go
for _, aspect := range s.SituationAspects {
    // aspect.Aspect is the text
}
```

### LLM Call
Use low temperature (0.1-0.3) for consistent classification:
```go
resp, err := client.ChatCompletion(ctx, llm.CompletionRequest{
    Messages:    []llm.Message{{Role: "user", Content: prompt}},
    MaxTokens:   500,
    Temperature: 0.3,
})
```

### Summary Statistics
Always print summary at end:
```go
t.Log("\n========== TEST SUMMARY ==========")
t.Logf("Positive cases: %d/%d (%.1f%%)", correct, total, float64(correct)*100/float64(total))
t.Log("\n--- Failed Cases ---")
for _, r := range results {
    if !r.Matches {
        t.Logf("FAIL: '%s'", r.TestCase.PlayerInput)
    }
}
```

## Running Tests

All commands tee output to a temp file for review and tail the last lines to
keep terminal output manageable:

```bash
# Run specific test
go test -v -tags=llmeval -run TestName ./test/llmeval/ -timeout 5m \
  2>&1 | tee /tmp/llmeval_results.txt | tail -3

# Run with verbose output
VERBOSE=1 go test -tags=llmeval -run TestName ./test/llmeval/ \
  2>&1 | tee /tmp/llmeval_results.txt | tail -3

# Run all llmeval tests
go test -v -tags=llmeval ./test/llmeval/ -timeout 10m \
  2>&1 | tee /tmp/llmeval_results.txt | tail -3
```

### Reviewing Results

```bash
# Review full output
cat /tmp/llmeval_results.txt

# Check accuracy across runs
grep "Accuracy:" /tmp/llmeval_results.txt

# Check which tests failed
grep "FAIL:" /tmp/llmeval_results.txt
```

### Durability Testing

LLM responses are non-deterministic. Use `-count=N` to run tests multiple times
and validate that results are stable:

```bash
# Run 5 iterations, save full output, show only final result
go test -v -tags=llmeval -count=5 -run TestName ./test/llmeval/ \
  2>&1 | tee /tmp/llmeval_results.txt | tail -3
```

## Example: Scene Transition Test

See `test/llmeval/scene_transition_llm_eval_test.go` for a complete example with:
- Positive cases (should trigger marker)
- Negative cases (should NOT trigger marker)
- Regex matching for markers
- Summary statistics
