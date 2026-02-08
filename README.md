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

## Fate Core System

This system implements the [Fate Core System Reference Document](https://fate-srd.com/fate-core), which is available under the Creative Commons Attribution 3.0 Unported license.

**Credits:** This work is based on Fate Core System, a product of Evil Hat Productions, LLC, developed, authored, and edited by Leonard Balsera, Brian Engard, Jeremy Keller, Ryan Macklin, Mike Olson, Clark Valentine, Amanda Valentine, Fred Hicks, and Rob Donoghue, and licensed for our use under the [Creative Commons Attribution 3.0 Unported license](https://creativecommons.org/licenses/by/3.0/).

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

### Scene Management
- **Dynamic Scenes**: Create and modify scenes with situation aspects
- **Conflict System**: Handle conflicts with initiative, zones, and positioning
- **Narrative Continuity**: Maintain story context across scenes and sessions

## Configuration

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

## Package Structure

```
LlamaOfFate/
├── cmd/
│   └── cli/                    # Command-line interface
├── internal/
│   ├── core/                   # Core game mechanics
│   │   ├── action/             # Action resolution system
│   │   ├── character/          # Character management
│   │   ├── dice/               # Dice rolling and probability
│   │   └── scene/              # Scene and conflict management
│   ├── engine/                 # Game engine (scene/scenario managers, action parsing, conflict resolution)
│   ├── llm/                    # LLM integration layer
│   │   └── azure/              # Azure OpenAI implementation
│   ├── session/                # Session logging for game transcripts
│   ├── logging/                # Application logging
│   └── ui/
│       └── terminal/           # Terminal-based interface
├── examples/                   # Example programs and scenarios
│   ├── llm-scene-loop/         # Interactive scene loop example
│   ├── scenario-generator/     # Scenario generation example
│   ├── scenario-walkthrough/   # Scenario walkthrough example
│   └── scene-generator/        # Scene generation example
├── configs/                    # Configuration files (azure-llm.yaml)
├── docs/                       # Documentation
│   └── architecture.md         # Architecture documentation
├── test/                       # Tests
│   ├── integration/            # Integration tests
│   └── llmeval/                # LLM evaluation tests
└── [standard Go project files]
```

### Package Responsibilities

- **`cmd/cli/`**: Entry point for the command-line application
- **`internal/core/`**: Core Fate mechanics implementation (character, dice, scene, action, skills)
- **`internal/engine/`**: Orchestrates core mechanics and LLM services (scene managers, action parsing, conflict resolution)
- **`internal/llm/`**: LLM integration with Azure OpenAI backend, including retry logic and response handling
- **`internal/session/`**: Session logging for game transcripts
- **`internal/ui/terminal/`**: Terminal-based user interface implementation
- **`examples/`**: Example programs demonstrating LLM scene loops, scenario generation, and walkthroughs
- **`configs/`**: YAML configuration files (azure-llm.yaml)
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
- ✅ Integration tests and LLM evaluation tests

### Planned Features
- 📋 Additional LLM backends (Ollama, OpenAI direct)
- 📋 Persistence layer (save/load functionality)
- 📋 Web-based user interface
- 📋 Public API packages for external integrations
- 📋 Database backends for long-term storage
