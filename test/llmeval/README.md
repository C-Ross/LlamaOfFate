# LLM Evaluation Tests

This package contains evaluation tests for LLM-powered components. These tests call real LLM APIs to validate classification accuracy and behavior.

## Purpose

These tests evaluate how well the LLM performs specific tasks:

- **Action Parser Tests** (`action_parser_llm_eval_test.go`): Verify that player inputs are correctly classified as Overcome, Attack, or Create Advantage actions with appropriate skills and difficulty ratings.
- **Input Classification Tests** (`input_classification_llm_eval_test.go`): Verify that player inputs are correctly categorized as dialog, clarification requests, or game actions.
- **Aspect Generator Tests** (`aspect_generator_llm_eval_test.go`): Verify that Create Advantage outcomes produce valid aspects with correct free invokes, appropriate length (2-10 words), and no duplicates of existing aspects.

Use these tests to:
- Validate LLM configuration changes
- Detect regressions after prompt template modifications
- Diagnose classification biases (e.g., over-selecting Create Advantage vs Overcome)
- Evaluate different LLM models or parameters

## Requirements

### Connectivity

These tests require a live connection to the Azure OpenAI API. You must set the following environment variables:

```bash
export AZURE_API_ENDPOINT="https://your-resource.openai.azure.com/"
export AZURE_API_KEY="your-api-key"
```

Tests will be skipped if these credentials are not configured.

### Build Tag

Tests are gated behind the `llmeval` build tag to prevent them from running during normal test cycles (they are slow and require API access).

Run with:
```bash
make test-llm
```

Or directly:
```bash
go test -v -tags=llmeval ./test/llmeval/...
```

### Verbose Output

By default, tests only output a summary. To see per-test details (useful for debugging failures), set:

```bash
VERBOSE_TESTS=1 make test-llm
```

This enables detailed logging showing:
- Each test case name and input
- Expected vs actual classifications
- Skill and difficulty comparisons

## Test Structure

Each test file follows a consistent pattern:

1. **Test cases** are defined as structs with inputs and expected outputs
2. **Acceptable alternatives** allow flexibility (e.g., multiple valid skills)
3. **Difficulty tolerance** of ±1 accounts for reasonable LLM variance
4. **Summary reports** show pass/fail rates by category

## Tracking Flakiness

LLM tests are non-deterministic. Use the tracker to record results over multiple runs and identify which tests are reliably passing vs. flaky.

```bash
# Run all LLM eval tests and record results
just test-llm-track

# Run a specific test 5 times to check stability
just test-llm-track-n TestSceneTransition 5

# View stability report (last 10 runs)
just test-llm-report

# Show only flaky tests
just test-llm-flaky
```

Results accumulate in `test/llmeval/results.jsonl` (gitignored). Each run appends one record with per-subtest pass/fail status and the git commit SHA.

You can also query results directly with `jq`:

```bash
# Which tests failed in the most recent run?
tail -1 test/llmeval/results.jsonl | jq '[.tests | to_entries[] | select(.value=="fail") | .key]'

# Count pass/fail for a specific test across all runs
jq -r '.tests["TestSceneTransition_LLMEvaluation/ShouldExit/Step_outside"] // empty' \
  test/llmeval/results.jsonl | sort | uniq -c
```

## Known Issues

- Intimidation inputs can be ambiguously classified as Attack or Create Advantage depending on context. See [Issue #9](https://github.com/C-Ross/LlamaOfFate/issues/9).

## Adding New Tests

When adding test cases:

1. Group by expected classification type
2. Include clear, unambiguous inputs when possible
3. Document any known ambiguities
4. Use `AcceptableSkills` for inputs where multiple skills are valid
5. Set `SkipDifficultyCheck: true` for attacks (difficulty comes from active defense)
