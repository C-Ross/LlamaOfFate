# LlamaOfFate

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

**Credits:** This work is based on Fate Core System, a product of Evil Hat Productions, LLC, developed, authored, and edited by Leonard Balsera, Brian Engard, Jeremy Keller, Ryan Macklin, Mike Olson, Clark Valentine, Amanda Valentine, Fred Hicks, and Rob Donoghue, and licensed for our use under the [Creative Commons Attribution 3.0 Unported license](https://creativecommons.org/licenses/by/3.0/).

Fate™ is a trademark of Evil Hat Productions, LLC.

## Key Features

### Pre-Game Setup
- **Preset Scenarios**: Choose from curated scenarios (Saloon, Heist, Tower) with pre-built characters
- **Custom Character Creation**: Create your own character with custom name, high concept, trouble, and skills
- **LLM-Generated Scenarios**: Provide a genre description and let the LLM generate a unique scenario
- **Continue Game**: Resume your last saved game from the setup screen

### Natural Language Processing
- **Action Parsing**: Convert free-form text like "I sneak past the guards using the shadows" into structured game actions
- **Context Awareness**: LLM maintains awareness of current scene, character capabilities, and recent events
- **Fluid descriptions** The LLM narrates the outcome in fluid prose, incorporating aspects and outcomes.

### Fate Core Mechanics
- **Aspect System**: Full support for character, situation, and consequence aspects
- **Complete Skill System**: All 18 default Fate Core skills with proper action mappings
- **Stress and Consequences**: Proper implementation of physical and mental damage
- **Fate Point Economy**: Track and manage fate point spending and gaining

### Scene Management
- **Dynamic Scenes**: Create and modify scenes with situation aspects
- **Conflict System**: Handle conflicts with initiative, zones, and positioning
- **Narrative Continuity**: Maintain story context across scenes and sessions

## Configuration

### Preset Scenarios and Characters

LlamaOfFate includes preset scenarios and characters stored as YAML files in the `configs/` directory:

- **Scenarios**: `configs/scenarios/*.yaml` - Pre-built scenarios (saloon, heist, tower)
- **Characters**: `configs/characters/*.yaml` - Pre-built player characters (jesse-calhoun, zero, lyra-moonwhisper)

These YAML files define characters with their aspects, skills, stress tracks, and scenario details including initial scenes and NPCs. You can create your own by following the structure in the existing files.

### Azure ML Setup

LlamaOfFate uses Azure ML for LLM integration. Configuration is stored in `configs/azure-llm.yaml`.

**Recommended Setup (Environment Variables):**

```bash
# Set your Azure ML credentials via environment variables
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
- **`just run`** - Build and run the CLI
- **`just go-test`** - Run Go tests
- **`just go-lint`** - Run golangci-lint
- **`just go-validate`** - vet + fmtcheck + lint + test + build
- **`just test-llm`** - Run LLM evaluation tests (requires Azure credentials)
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

LlamaOfFate automatically saves your game progress at key points during play.

- **Auto-save**: The game saves automatically at scene transitions and key moments
- **Continue Game**: On the setup screen, select "Continue Game" to resume from your last save
- **Save validation**: Corrupted or incompatible saves are detected and you'll be notified with an error message
- **Save location**: Game saves are stored as YAML files (configurable via `GameSaver` interface)

## Package Structure

```
LlamaOfFate/
├── cmd/
│   ├── cli/                    # Command-line interface
│   └── server/                 # WebSocket server entry point
├── internal/
│   ├── core/                   # Core game mechanics
│   │   ├── action/             # Action resolution system
│   │   ├── character/          # Character management
│   │   ├── dice/               # Dice rolling and probability
│   │   └── scene/              # Scene and conflict management
│   ├── engine/                 # Game engine (scene/scenario managers, action parsing, conflict resolution)
│   ├── syncdriver/             # Synchronous blocking game loop (wraps async engine API)
│   ├── config/                 # Configuration loader (YAML scenarios and characters)
│   ├── llm/                    # LLM integration layer
│   │   └── azure/              # Azure OpenAI implementation
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
├── configs/                    # Configuration files
│   ├── azure-llm.yaml          # LLM configuration
│   ├── characters/*.yaml       # Preset player characters
│   └── scenarios/*.yaml        # Preset scenarios
├── docs/                       # Documentation
│   └── architecture.md         # Architecture documentation
├── test/                       # Tests
│   ├── integration/            # Integration tests
│   └── llmeval/                # LLM evaluation tests
└── [standard Go project files]
```

### Package Responsibilities

- **`cmd/cli/`**: Entry point for the command-line application
- **`cmd/server/`**: Entry point for the WebSocket server
- **`internal/core/`**: Core Fate mechanics implementation (character, dice, scene, action, skills)
- **`internal/engine/`**: Purely async/event-driven game engine (GameSessionManager interface: Start/HandleInput/ProvideInvokeResponse/ProvideMidFlowResponse/Save); emits GameEvents for UI rendering
- **`internal/syncdriver/`**: Synchronous blocking game loop that wraps the engine's async API for terminal-style UIs (Run function drives: ReadInput → HandleInput → Emit events → drive prompts → repeat)
- **`internal/config/`**: Configuration loader for YAML-based scenarios and characters (LoadAll, LoadCharacter, LoadScenario)
- **`internal/llm/`**: LLM integration with Azure OpenAI backend, including retry logic and response handling
- **`internal/prompt/`**: LLM prompt template rendering and response parsing (template data types, render functions, marker extraction)
- **`internal/session/`**: Session logging for game transcripts
- **`internal/storage/`**: Game state persistence with YAML-based save/load and validation
- **`internal/uicontract/`**: UI interface contracts (UI, SceneInfo, GameEvent types, etc.) for decoupling engine from UI implementations
- **`internal/ui/terminal/`**: Terminal UI implementation; handles meta-commands and renders GameEvents to console
- **`internal/ui/web/`**: WebSocket UI implementation; bridges engine events to WebSocket clients
- **`web/`**: React frontend — Vite 7, React 19, TypeScript, Tailwind CSS v4, shadcn/ui, Vitest; includes setup screen with preset picker and custom character creation
- **`examples/`**: Example programs demonstrating LLM scene loops, scenario generation, and walkthroughs
- **`configs/`**: YAML configuration files (azure-llm.yaml, characters/*.yaml, scenarios/*.yaml)
- **`test/integration/`**: Integration tests for the game system
- **`test/llmeval/`**: LLM evaluation tests for prompt behavior

## Implementation Status

### Completed Features
- ✅ Core data structures (character, aspects, stress, consequences)
- ✅ Complete Fate Core dice system (4dF) and skill ladder
- ✅ All 18 default Fate Core skills with action mappings
- ✅ Game engine with scene and scenario management
- ✅ LLM integration with Azure OpenAI
- ✅ Action parsing from natural language input
- ✅ Conflict resolution system with stress and consequences
- ✅ CLI interface for game interaction
- ✅ Session logging for game transcripts
- ✅ Game state persistence (save/load functionality via YAML with validation)
- ✅ Event-driven UI architecture with async invoke/input support
- ✅ Integration tests and LLM evaluation tests
- ✅ Web UI with full gameplay support (React + Vite + Tailwind + shadcn/ui)
- ✅ WebSocket server backend
- ✅ Pre-game setup flow with preset scenarios and custom character creation
- ✅ LLM-powered scenario generation from genre descriptions
- ✅ YAML-based configuration system for scenarios and characters

### Planned Features
- 📋 Additional LLM backends (Ollama, OpenAI direct)
- 📋 Public API packages for external integrations
- 📋 Database backends for long-term storage
- 📋 Character sheet editor in web UI
