# LlamaOfFate

[![Validate](https://github.com/C-Ross/LlamaOfFate/actions/workflows/validate.yml/badge.svg)](https://github.com/C-Ross/LlamaOfFate/actions/workflows/validate.yml)

A text-based RPG system implementing the Fate Core rules with LLM-powered generation and action interpretation.

## Overview

LlamaOfFate is a text-based RPG that brings the flexibility and narrative focus of Fate Core to a digital medium. The system leverages Large Language Models (LLMs) to:

- Parse freeform text input into game actions
- Generate dynamic descriptions and narrative responses
- Assist with scene management and story progression
- Provide contextual suggestions for aspects and consequences

## Core Design Philosophy

- **Narrative First**: All game mechanics serve the story
- **Player Agency**: Natural language input allows for creative problem-solving
- **LLM Integration**: AI assists but doesn't replace human creativity and decision-making
- **Fate Core**: Faithful implementation of official Fate Core rules
- **Event-Driven UI**: Engine emits structured events; UI implementations control presentation

## Fate Core System

This system implements the [Fate Core System Reference Document](https://fate-srd.com/fate-core), which is available under the Creative Commons Attribution 3.0 Unported license.

**Credits:** This work is based on Fate Core System, a product of Evil Hat Productions, LLC, developed, authored, and edited by Leonard Balsera, Brian Engard, Jeremy Keller, Ryan Macklin, Mike Olson, Clark Valentine, Amanda Valentine, Fred Hicks, and Rob Donoghue, and licensed for our use under the [Creative Commons Attribution 3.0 Unported license](https://creativecommons.org/licenses/by/3.0/). This software is an original implementation of the Fate Core rules; it is not the Fate Core SRD text itself.

Fate™ is a trademark of Evil Hat Productions, LLC.

## Key Features

### Natural Language Processing
- **Action Parsing**: Convert free-form text like "I sneak past the guards using the shadows" into structured game actions
- **Context Awareness**: LLM maintains awareness of current scene, character capabilities, and recent events
- **Fluid descriptions** The LLM narrates the outcome in fluid prose, incorporating aspects and outcomes.

### Fate Core Mechanics
- **Aspect System**: Full support for character, situation, and consequence aspects
- **Complete Skill System**: All 18 default Fate Core skills with proper action mappings
- **Stress and Consequences**: Proper implementation of physical and mental damage
- **Fate Point Economy**: Track and manage fate point spending and gaining
- **Challenge System**: Multi-task challenges with skill-based overcome actions and outcome tallying

### Scene Management
- **Dynamic Scenes**: Create and modify scenes with situation aspects
- **Conflict System**: Handle conflicts with initiative, zones, and positioning
- **Challenge System**: Multi-step challenges with task tracking and partial success
- **Narrative Continuity**: Maintain story context across scenes and sessions

## Configuration

LlamaOfFate supports multiple LLM backends for maximum flexibility. Choose the one that works best for your needs.

### Cloud LLM Setup (OpenAI-compatible)

LlamaOfFate uses any OpenAI-compatible endpoint for LLM integration. Configuration is stored in `configs/azure-llm.yaml`.

**Recommended Setup (Environment Variables):**

```bash
# Set your LLM credentials via environment variables
export AZURE_API_ENDPOINT="https://your-resource.cognitiveservices.azure.com/openai/deployments/your-deployment/chat/completions?api-version=2024-05-01-preview"
export AZURE_API_KEY="your-api-key-here"
```

Environment variables take precedence over values in the configuration file, making it safe to commit your config to version control.

**Configuration File:**

Edit `configs/azure-llm.yaml` to set your preferred model and timeout:

```yaml
# api_endpoint and api_key can be left empty if using environment variables
api_endpoint: ""
api_key: ""

# Choose your Llama model
model_name: "Llama-4-Maverick-17B-128E-Instruct-FP8"

# Request timeout in seconds
timeout: 300
```

### Ollama Setup (Local)

For a completely local experience without cloud dependencies, use [Ollama](https://ollama.ai/):

**Installation:**

```bash
# Install Ollama (see https://ollama.ai/ for your platform)
# macOS/Linux:
curl -fsSL https://ollama.ai/install.sh | sh

# Pull a model
./scripts/ollama-pull.sh
# Or manually: ollama pull llama3.2:3b
```

**Configuration:**

Edit `configs/ollama-llm.yaml` (default configuration provided):

```yaml
# Ollama's OpenAI-compatible endpoint
api_endpoint: "http://localhost:11434/v1/chat/completions"
api_key: "ollama"  # Ollama doesn't require a real key

# Must match a pulled model
model_name: "llama3.2:3b"

timeout: 300
```

**Running with Ollama:**

```bash
# Ensure Ollama is running
ollama serve  # or ollama is running as a service

# Use the Ollama config
export LLM_CONFIG=configs/ollama-llm.yaml
just run
```

## Development Automation

LlamaOfFate uses **GitHub Agentic Workflows** ([gh-aw](https://github.com/github/gh-aw)) for automated development tasks. The repository includes several AI-powered workflows:

- **readme-updater**: Automatically updates README.md when significant changes are pushed to main
- **skills-updater**: Updates `.github/skills/` documentation when relevant code changes
- **coverage-improver**: Analyzes test coverage and suggests improvements

These workflows [run automatically](https://github.github.com/gh-aw/introduction/overview/).

**Installing gh-aw CLI:**
```bash
gh extension install github/gh-aw
```

## Building and Running

LlamaOfFate uses [`just`](https://github.com/casey/just) as a command runner for common development tasks.

### Installing Just

**macOS:**
```bash
brew install just
```

**Linux:**
```bash
# Using cargo (Rust package manager)
cargo install just

# Or download pre-built binaries from:
# https://github.com/casey/just/releases
```

**Windows:**
```powershell
# Using cargo (Rust package manager)
cargo install just

# Or using Chocolatey
choco install just

# Or using Scoop
scoop install just
```

### Available Commands

Run `just` without arguments to see all available commands. Common commands include:

**Unified:**
- **`just validate`** - Run all validation checks (Go + Web)
- **`just clean`** - Clean all build artifacts

**Go:**
- **`just build`** - Build the CLI application
- **`just build-server`** - Build the WebSocket server
- **`just build-mcpserver`** - Build the MCP server
- **`just run`** - Build and run the CLI
- **`just serve`** - Build and run the WebSocket server
- **`just go-test`** - Run Go tests
- **`just go-lint`** - Run golangci-lint
- **`just go-validate`** - vet + fmtcheck + lint + test + build
- **`just test-llm`** - Run LLM evaluation tests (requires LLM credentials or LLM_PROVIDER=ollama)
- **`just test-llm-track`** - Run LLM tests and track results for flakiness analysis
- **`just test-llm-report`** - Show LLM test stability report
- **`just test-llm-flaky`** - Show only flaky LLM tests
- **`just test-llm-fetch`** - Fetch LLM test results from CI
- **`just go-fmt`** - Format Go code with gofmt

**Web:**
- **`just web-dev`** - Start Vite dev server
- **`just web-test`** - Run Vitest
- **`just web-lint`** - Run ESLint
- **`just web-build`** - Production build
- **`just web-validate`** - lint + test + build
- **`just web-install`** - Install npm dependencies

### Quick Start

```bash
# Install dependencies
just go-deps
just web-install

# Build the application
just build

# Run all validations (recommended before committing)
just validate

# Run the CLI application
just run

# Start the web UI dev server
just web-dev
```

## Saving and Loading Games

LlamaOfFate automatically saves your game progress at key points during play. When you restart the application, it will automatically resume from your last save.

- **Auto-save**: The game saves automatically at scene transitions and key moments
- **Auto-resume**: On startup, the game resumes from the last unfinished scenario
- **Save location**: Game saves are stored as YAML files (configurable via `GameSaver` interface)

## Package Structure

```
LlamaOfFate/
├── cmd/
│   ├── cli/                    # Command-line interface
│   ├── server/                 # WebSocket server entry point
│   ├── mcpserver/              # MCP (Model Context Protocol) server
│   └── llmeval-tracker/        # LLM test flakiness tracking tool
├── internal/
│   ├── core/                   # Core game mechanics (character, aspects, stress, consequences, skills)
│   │   ├── action/             # Action resolution system
│   │   ├── dice/               # Dice rolling and probability
│   │   └── scene/              # Scene and conflict management
│   ├── engine/                 # Game engine (scene/scenario managers, action parsing, conflict resolution)
│   ├── syncdriver/             # Synchronous blocking game loop (wraps async engine API)
│   ├── llm/                    # LLM integration layer
│   │   └── openai/             # OpenAI-compatible LLM implementation
│   ├── prompt/                 # LLM prompt templates and response parsing
│   ├── session/                # Session logging for game transcripts
│   ├── storage/                # Game state persistence (YAML save/load)
│   ├── logging/                # Application logging
│   └── ui/
│       ├── terminal/           # Terminal-based interface
│       └── web/                # WebSocket UI implementation
├── web/                        # React frontend (Vite, Tailwind v4, shadcn/ui)
│   └── src/                    # Components, theme, tests
├── examples/                   # Example programs and scenarios
│   ├── llm-scene-loop/         # Interactive scene loop example
│   ├── scenario-generator/     # Scenario generation example
│   ├── scenario-walkthrough/   # Scenario walkthrough example
│   └── scene-generator/        # Scene generation example
├── scripts/                    # Utility scripts
│   └── llmeval-fetch-results.sh # Fetch LLM test results from CI
├── configs/                    # Configuration files (azure-llm.yaml)
├── docs/                       # Documentation
│   └── architecture.md         # Architecture documentation
├── test/                       # Tests
│   └── llmeval/                # LLM evaluation tests
└── [standard Go project files]
```

### Package Responsibilities

- **`cmd/cli/`**: Entry point for the command-line application
- **`cmd/server/`**: Entry point for the WebSocket server
- **`cmd/mcpserver/`**: Entry point for the MCP server (programmatic game interaction via Model Context Protocol)
- **`cmd/llmeval-tracker/`**: Tool for tracking LLM test flakiness and generating stability reports
- **`internal/core/`**: Core Fate mechanics implementation (character, dice, scene, action, skills, challenges)
- **`internal/engine/`**: Purely async/event-driven game engine (GameSessionManager interface: Start/HandleInput/ProvideInvokeResponse/ProvideMidFlowResponse/Save); emits GameEvents for UI rendering; includes conflict and challenge managers
- **`internal/syncdriver/`**: Synchronous blocking game loop that wraps the engine's async API for terminal-style UIs (Run function drives: ReadInput → HandleInput → Emit events → drive prompts → repeat)
- **`internal/llm/`**: LLM integration with OpenAI-compatible backends, including retry logic and response handling
- **`internal/prompt/`**: LLM prompt template rendering and response parsing (template data types, render functions, marker extraction)
- **`internal/session/`**: Session logging for game transcripts
- **`internal/storage/`**: Game state persistence with YAML-based save/load
- **`internal/uicontract/`**: UI interface contracts (UI, SceneInfo, GameEvent types, etc.) for decoupling engine from UI implementations
- **`internal/ui/terminal/`**: Terminal UI implementation; handles meta-commands and renders GameEvents to console
- **`internal/ui/web/`**: WebSocket UI implementation; bridges engine events to WebSocket clients
- **`internal/mcpserver/`**: MCP server implementation; exposes game tools for programmatic interaction (start game, send input, inspect state)
- **`web/`**: React frontend — Vite 7, React 19, TypeScript, Tailwind CSS v4, shadcn/ui, Vitest
- **`examples/`**: Example programs demonstrating LLM scene loops, scenario generation, and walkthroughs
- **`scripts/`**: Utility scripts for development and testing workflows
- **`configs/`**: YAML configuration files (azure-llm.yaml)
- **`test/llmeval/`**: LLM evaluation tests for prompt behavior

## Implementation Status

### Completed Features
- ✅ Core data structures (character, aspects, stress, consequences)
- ✅ Complete Fate Core dice system (4dF) and skill ladder
- ✅ All 18 default Fate Core skills with action mappings
- ✅ Game engine with scene and scenario management
- ✅ LLM integration with OpenAI-compatible endpoints (Azure, Ollama, OpenAI)
- ✅ Action parsing from natural language input
- ✅ Conflict resolution system with stress and consequences
- ✅ Challenge system with multi-task tracking and outcome tallying
- ✅ CLI interface for game interaction
- ✅ Session logging for game transcripts
- ✅ Game state persistence (save/load functionality via YAML)
- ✅ Event-driven UI architecture with async invoke/input support
- ✅ Integration tests and LLM evaluation tests
- ✅ Web UI scaffold (React + Vite + Tailwind + shadcn/ui)
- ✅ WebSocket server backend
- ✅ MCP (Model Context Protocol) server for programmatic game interaction
- ✅ Ollama support for local LLM backend (no cloud dependencies required)

### Planned Features
- 📋 Contest system (competitive opposed exchanges)
- 📋 Additional LLM backends (OpenAI direct, other providers)
- 📋 WebSocket integration connecting web UI to game engine
- 📋 Public API packages for external integrations
- 📋 Database backends for long-term storage
