# Contributing to LlamaOfFate

Thank you for your interest in contributing! LlamaOfFate is a text-based RPG implementing the Fate Core System with LLM integration, and we welcome contributions of all kinds.

## Getting Started

### Prerequisites

- **Go 1.24+**
- **Node.js 20+** (for the web frontend)
- **[just](https://github.com/casey/just)** command runner
- **An LLM backend** — either [Ollama](https://ollama.ai/) (free, local) or an OpenAI-compatible API

### Setup

```bash
git clone https://github.com/C-Ross/LlamaOfFate.git
cd LlamaOfFate

# Install Go and web dependencies
just web-install

# Run all validation checks
just validate
```

### Running Locally

**CLI (terminal UI):**
```bash
export LLM_CONFIG=configs/ollama-llm.yaml
just run
```

**Web UI:**
```bash
export LLM_CONFIG=configs/ollama-llm.yaml
just serve
# Open http://localhost:8080
```

## Making Changes

### Before You Start

- Check [existing issues](https://github.com/C-Ross/LlamaOfFate/issues) to see if someone is already working on it
- For larger changes, open an issue first to discuss the approach
- This project is configured for GitHub Copilot-assisted development workflows.
- AI-assisted changes are welcome. Please review generated code carefully and ensure you understand the final changes you submit.

### Development Workflow

1. Create a branch from `main`
2. Make your changes
3. Run `just validate` — **all checks must pass** before submitting
4. Playtest your change when it affects gameplay flow, prompts, scene behavior, or user interaction
5. Commit with a clear message describing what and why
6. Open a pull request

### What `just validate` Checks

- `go vet` — static analysis
- `gofmt` — formatting (code must be formatted with `go fmt`)
- `golangci-lint` — linting
- `go test` — unit tests
- `go build` — compilation of all binaries
- `npm run lint` — ESLint for the web frontend
- `vitest` — web frontend tests
- `vite build` — web frontend builds

### Code Style

- Follow standard Go conventions (`go fmt`)
- Prefer early returns to reduce nesting
- Use Go templates for all LLM prompts — never inline prompt text in Go code
- Prefer YAML over JSON for configuration and data files
- Use [testify](https://github.com/stretchr/testify) for all test assertions

### Testing

- Write tests for new functionality using testify assertions
- For deterministic dice tests, specify the roll directly or use `dice.NewSeededRoller(12345)`
- LLM evaluation tests live in `test/llmeval/` and require the `llmeval` build tag plus one of:
  - **Ollama (local):** `export LLM_PROVIDER=ollama` (uses `configs/ollama-llm.yaml`)
  - **Azure/OpenAI:** `export AZURE_API_ENDPOINT=<url>` and `export AZURE_API_KEY=<key>` (uses `configs/azure-llm.yaml`)
  ```bash
  just test-llm
  ```

## Fate Core System

This project implements the [Fate Core SRD](https://fate-srd.com/fate-core). When working on game mechanics:

- Reference the SRD for rules questions
- Credit the Fate SRD in any documentation that references rules

Fate Core SRD content is licensed under [CC BY 3.0 Unported](https://creativecommons.org/licenses/by/3.0/) by Evil Hat Productions.

## Project Structure

- `internal/core/` — Fate Core mechanics (dice, actions, characters, scenes)
- `internal/engine/` — Game loop and LLM orchestration
- `internal/prompt/` — LLM prompt templates (Go templates in `templates/`)
- `internal/llm/` — LLM client interface
- `internal/ui/` — UI implementations (terminal, web)
- `web/` — React frontend (Vite, Tailwind v4, shadcn/ui)
- `test/llmeval/` — LLM evaluation tests

## Community Standards

We are committed to providing a welcoming and inclusive community. Please review our [Code of Conduct](CODE_OF_CONDUCT.md) to understand our expectations for respectful collaboration.

## Getting Help

- Check [docs/architecture.md](docs/architecture.md) for system design details
- Open an issue for bugs or feature discussions
