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

## Package Structure

```
LlamaOfFate/
├── cmd/
│   └── cli/                    # Command-line interface
│       └── main.go
├── internal/
│   ├── core/                   # Core game mechanics
│   │   ├── character/          # Character management
│   │   │   ├── character.go
│   │   │   ├── aspects.go
│   │   │   ├── skills.go
│   │   │   ├── stunts.go
│   │   │   └── stress.go
│   │   ├── dice/               # Dice rolling and probability
│   │   │   ├── dice.go
│   │   │   └── ladder.go
│   │   ├── scene/              # Scene and conflict management
│   │   │   ├── scene.go
│   │   │   ├── conflict.go
│   │   │   └── aspects.go
│   │   └── action/             # Action resolution system
│   │       ├── action.go
│   │       ├── outcome.go
│   │       └── modifiers.go
│   ├── llm/                    # LLM integration layer
│   │   ├── service.go          # LLM service interface
│   │   ├── parser.go           # Action parsing
│   │   ├── narrator.go         # Narrative generation
│   │   └── ollama/             # Ollama-specific implementation
│   │       └── client.go
│   ├── engine/                 # Game engine coordination
│   │   ├── engine.go           # Main game engine
│   │   ├── state.go            # Game state management
│   │   └── events.go           # Event system
│   ├── storage/                # Data persistence
│   │   ├── interface.go        # Storage interface
│   │   ├── memory/             # In-memory storage
│   │   │   └── memory.go
│   │   └── json/               # JSON file storage
│   │       └── json.go
│   └── ui/                     # User interface components
│       ├── text/               # Text-based interface
│       │   ├── cli.go
│       │   └── commands.go
│       └── web/                # Web interface
│           ├── handlers.go
│           ├── websocket.go
│           └── static/
├── pkg/                        # Public API packages
│   ├── types/                  # Shared type definitions
│   │   ├── character.go
│   │   ├── scene.go
│   │   ├── action.go
│   │   └── game.go
│   └── client/                 # Client library for external integrations
│       └── client.go
├── configs/                    # Configuration files
│   ├── default.yaml
│   ├── skills.yaml             # Default Fate Core skills
│   ├── stunts.yaml             # Default stunts library
│   └── llm.yaml                # LLM service configuration
├── docs/                       # Documentation
│   ├── api.md
│   ├── setup.md
│   └── examples/
├── scripts/                    # Build and development scripts
│   ├── build.sh
│   └── dev.sh
├── test/                       # Integration tests
│   ├── fixtures/               # Test data
│   └── integration/
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
```

### Package Responsibilities

- **`cmd/`**: Entry point for the CLI application
- **`internal/core/`**: Core Fate mechanics implementation, isolated from external dependencies
- **`internal/llm/`**: LLM integration with pluggable backends (Ollama, OpenAI, etc.)
- **`internal/engine/`**: Orchestrates core mechanics and LLM services
- **`internal/storage/`**: Pluggable persistence layer for different storage backends
- **`internal/ui/`**: Text-based user interface implementation
- **`pkg/types/`**: Shared data structures that can be imported by external packages
- **`pkg/client/`**: Go client library for programmatic access to the game engine
- **`configs/`**: YAML configuration files for all system settings (skills, stunts, LLM services, etc.)

## Implementation Notes

- Use Go's strong typing for game state validation
- Implement comprehensive JSON serialization for save/load functionality
- Consider database backends for persistence (PostgreSQL recommended)
- Implement robust error handling for LLM integration failures

## Next Steps

1. Implement core data structures and interfaces
2. Create basic game engine with dice rolling and skill checks
3. Integrate LLM service for action parsing
4. Build CLI interface for game interaction
5. Add persistence layer
6. Implement full conflict resolution system
