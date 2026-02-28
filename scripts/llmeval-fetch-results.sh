#!/usr/bin/env bash
# Download llmeval-results artifacts from recent CI runs and generate a combined report.
#
# Usage:
#   ./scripts/llmeval-fetch-results.sh          # last 10 runs (default)
#   ./scripts/llmeval-fetch-results.sh 30       # last 30 runs
#   ./scripts/llmeval-fetch-results.sh 10 --flaky  # only show flaky tests
set -euo pipefail

LIMIT="${1:-10}"
shift || true
EXTRA_ARGS=("$@")

RESULTS_DIR="$(mktemp -d)"
COMBINED="$RESULTS_DIR/combined.jsonl"
trap 'rm -rf "$RESULTS_DIR"' EXIT

echo "Fetching up to $LIMIT recent LLM Eval workflow runs..."

# Get run IDs for completed runs that have artifacts
RUN_IDS=$(gh run list \
  --workflow=llm-eval.yml \
  --limit "$LIMIT" \
  --status completed \
  --json databaseId \
  --jq '.[].databaseId')

if [ -z "$RUN_IDS" ]; then
  echo "No completed LLM Eval runs found."
  exit 0
fi

COUNT=0
for RUN_ID in $RUN_IDS; do
  DEST="$RESULTS_DIR/run-$RUN_ID"
  if gh run download "$RUN_ID" --name llmeval-results --dir "$DEST" 2>/dev/null; then
    if [ -f "$DEST/results.jsonl" ]; then
      cat "$DEST/results.jsonl" >> "$COMBINED"
      COUNT=$((COUNT + 1))
    fi
  fi
done

if [ "$COUNT" -eq 0 ]; then
  echo "No llmeval-results artifacts found in the last $LIMIT runs."
  exit 0
fi

echo "Downloaded results from $COUNT runs."
echo ""

LLMEVAL_RESULTS="$COMBINED" go run ./cmd/llmeval-tracker report --last "$COUNT" "${EXTRA_ARGS[@]}"
