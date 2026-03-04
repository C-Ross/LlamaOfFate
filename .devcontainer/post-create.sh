#!/usr/bin/env bash
set -euo pipefail

echo "==> Downloading Go modules..."
go mod download

echo "==> Installing web dependencies..."
cd web && npm install && cd ..

echo "==> Installing gh-aw..."
curl -sL https://raw.githubusercontent.com/github/gh-aw/main/install-gh-aw.sh | bash

echo "==> Installing Ollama..."
sudo apt-get update -qq && sudo apt-get install -y -qq zstd >/dev/null
curl -fsSL https://ollama.com/install.sh | sh

echo "==> Post-create setup complete."
