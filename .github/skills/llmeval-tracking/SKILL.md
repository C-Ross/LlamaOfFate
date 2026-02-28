---
name: llmeval-tracking
description: Guide for fetching, analyzing, and interpreting LLM eval test stability results from CI. Use this when asked to check LLM eval flakiness, fetch CI results, analyze test stability, or run the llmeval-tracker report.
---

# LLM Eval Tracking

Track pass rates and flakiness of LLM eval tests across CI runs. Results are
stored as JSONL, one line per run. The `llmeval-tracker` CLI generates stability
reports with per-test trend indicators.

## Quick Reference

```bash
# Fetch results from recent CI runs and show a combined report
./scripts/llmeval-fetch-results.sh          # last 10 runs
./scripts/llmeval-fetch-results.sh 30       # last 30 runs
./scripts/llmeval-fetch-results.sh 10 --flaky  # only flaky tests

# Or use justfile targets
just test-llm-fetch             # same as ./scripts/llmeval-fetch-results.sh
just test-llm-report            # report from local results.jsonl
just test-llm-flaky             # local report filtered to flaky tests
```

## Architecture

### Data Flow

1. `.github/workflows/llm-eval.yml` runs daily at 2 AM UTC (schedule only).
2. `go test -v -json -tags=llmeval` pipes output to `llmeval-tracker record`.
3. `record` parses `go test -json`, prunes parent tests, writes one JSONL line to
   `test/llmeval/results.jsonl`.
4. The workflow uploads `results.jsonl` as the `llmeval-results` artifact (90-day retention).
5. `scripts/llmeval-fetch-results.sh` downloads artifacts from multiple runs via
   `gh run download`, concatenates them, and runs the report.

### JSONL Format

Each line in `results.jsonl` is a `RunRecord`:

```json
{
  "timestamp": "2026-02-28T04:12:00Z",
  "commit": "581a439...",
  "duration_secs": 0,
  "tests": {
    "TestActionParser_LLMEval/Attack_the_bandit_with_my_sword": "pass",
    "TestActionParser_LLMEval/Persuade_the_guard_to_let_us_through": "fail"
  }
}
```

- `tests` maps leaf subtest names to `"pass"`, `"fail"`, or `"skip"`.
- Parent tests are pruned — only leaves are stored.

### Tracker CLI (`cmd/llmeval-tracker`)

```bash
# Record a run (reads go test -json from stdin)
go test -v -json -tags=llmeval ./test/llmeval/... | go run ./cmd/llmeval-tracker record

# Show report (last 10 runs by default)
go run ./cmd/llmeval-tracker report
go run ./cmd/llmeval-tracker report --last 20
go run ./cmd/llmeval-tracker report --flaky

# Override results file location
LLMEVAL_RESULTS=/tmp/combined.jsonl go run ./cmd/llmeval-tracker report
```

## Reading the Report

The report has two sections:

### Run Summary

```
Runs:
  #1   2026-02-28 04:12  581a439  258 pass  11 fail   0 skip  (96%)
  #2   2026-03-01 04:08  a1b2c3d  265 pass   4 fail   0 skip  (99%)
```

Each row is one CI run: date, short commit SHA, pass/fail/skip counts, and overall
pass rate.

### Per-test Results

```
Per-test results:
  TestActionParser_LLMEval/Complex_case       2/3  ( 66.7%)  ✓✗✓
  TestOpposition_LLMEval/Guard_scenario        3/3  (100.0%)  ✓✓✓
```

- Sorted flakiest-first (lowest pass rate at top).
- Trend column: `✓` = pass, `✗` = fail, `-` = skip, `·` = not present in that run.
- With `--flaky`: only tests below 100% pass rate are shown.

## Interpreting Results

### Healthy test suite
- Overall pass rate ≥ 95%.
- No individual test below 80%.

### Flaky tests (60-95%)
- Prompt may be ambiguous — check the test case input and expected output.
- Temperature may be too high for classification prompts (prefer 0.1-0.3).
- Look at the trend: `✓✓✗✓✓` (occasional miss) vs `✗✓✗✓✗` (coin flip).

### Consistently failing (< 60%)
- Likely a prompt regression or a test case that no longer matches behavior.
- Check recent prompt template changes in `internal/prompt/templates/`.
- Run the specific test locally with `VERBOSE=1` to see LLM output.

### New test not present in older runs
- Shows `·` for runs before it existed — this is expected.

## Workflow Details

The CI workflow (`.github/workflows/llm-eval.yml`):
- Runs on schedule only (daily 2 AM UTC), no manual dispatch.
- Skips if no commits in the last 24 hours (check-changes job).
- Requires `AZURE_API_ENDPOINT` and `AZURE_API_KEY` secrets.
- Uploads `llmeval-results` artifact with 90-day retention.

## Local Tracking

To record results from local test runs (stored in `test/llmeval/results.jsonl`):

```bash
just test-llm-track                     # run all + record
just test-llm-track-n TestName 5        # run specific test 5 times + record each
```

Local results.jsonl is gitignored. CI and local tracking are independent.

## Ad-hoc Analysis with jq

```bash
# Count runs
wc -l test/llmeval/results.jsonl

# List all test names from the latest run
tail -1 test/llmeval/results.jsonl | jq -r '.tests | keys[]'

# Show failures from the latest run
tail -1 test/llmeval/results.jsonl | jq -r '.tests | to_entries[] | select(.value=="fail") | .key'

# Pass rate for a specific test across all runs
jq -r '.tests["TestActionParser_LLMEval/Attack_the_bandit"]' test/llmeval/results.jsonl
```
