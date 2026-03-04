#!/usr/bin/env bash
set -euo pipefail

MODEL="${1:-llama3.2:3b}"

echo "==> Pulling Ollama model ($MODEL)..."
ollama serve &
OLLAMA_PID=$!
sleep 2
ollama pull "$MODEL"
kill $OLLAMA_PID 2>/dev/null || true

echo "==> Model $MODEL ready."
