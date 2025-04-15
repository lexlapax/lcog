# CogMem Golang Library

A modular cognitive architecture for LLM agents with tiered memory, dynamic processing, and reflective adaptation.

## Overview

CogMem is a Go library that implements a cognitive architecture for LLM agents, providing:

- Multi-tiered memory management (working memory, long-term memory)
- Entity-based memory isolation for multi-tenant systems
- Pluggable LTM backends (SQL, KV, Vector, Graph databases)
- Lua scripting for customization and extension
- Structured reflection and adaptation capabilities

## Project Status

This project is currently in Phase 1 of development, which is now complete. Phase 1 implemented:

- Core library structure, interfaces, and testing infrastructure
- Entity-based context management for multi-tenant isolation
- Long-term memory with SQLite and BoltDB adapters
- Lua scripting with custom hooks for memory operations
- Memory Management Unit (MMU) for encoding/retrieving memories
- Agent facade for orchestrating component interactions
- Example command-line application

## Getting Started

### Prerequisites

- Go 1.24+ (earlier versions may work but are not tested)
- SQLite3 (for SQL storage option)
- BoltDB (embedded, no separate installation needed)

### Installation

```bash
git clone https://github.com/lexlapax/cogmem.git
cd cogmem/cogmem-go
go mod download
make deps  # Install dependencies
```

### Running the Example Agent

The example agent provides a simple command-line interface to interact with the CogMem system:

```bash
# Build and run the example agent
make run

# Or build and run manually
go build -o bin/example-agent ./cmd/example-agent
./bin/example-agent
```

The example agent supports the following commands:

- `!help` - Show help message
- `!entity <id>` - Set the current entity ID
- `!user <id>` - Set the current user ID
- `!remember <text>` - Store a memory in LTM
- `!lookup <query>` - Retrieve memories matching query
- `!query <question>` - Ask a question using context from memories
- `!config` - Show current configuration
- `!quit` - Exit the application

#### Example Usage

```
$ ./bin/example-agent

=== CogMem Example Agent ===
LTM Store: kv
KV Provider: boltdb
Current Entity: default-entity | Current User: default-user
Type !help for available commands.

cogmem::default-user@default-entity> !remember I like dogs.
Memory stored successfully with ID: 6a8e42f1-9e3d-45a8-af2e-6a0bc76c8e1c

cogmem::default-user@default-entity> !remember Dogs come in many breeds.
Memory stored successfully with ID: d4f89c12-3e6c-42f1-90b2-f108d07a5f23

cogmem::default-user@default-entity> !lookup dog
Found 2 memories matching your query:

Memory 1: Dogs come in many breeds.
  Created: 2025-04-14T23:56:31Z

Memory 2: I like dogs.
  Created: 2025-04-14T23:56:27Z

cogmem::default-user@default-entity> !query What do I like?
I've analyzed the memories and here's what I found: Based on your memories, you like dogs.
```

### Configuration

You can configure the CogMem library by creating a `config.yaml` file in the current directory or in the `configs/` directory. Here's an example configuration file:

```yaml
# LTM (Long-Term Memory) Configuration
ltm:
  # Type can be "mock", "sql", or "kv"
  type: "kv"
  
  # SQL Store Backend Configuration
  sql:
    driver: "sqlite"
    dsn: "./data/cogmem.db"
  
  # KV (Key-Value) Backend Configuration 
  kv:
    provider: "boltdb"
  
# Scripting Configuration
scripting:
  # Paths to directories containing Lua scripts
  paths:
    - "./scripts/mmu"
    - "./scripts/reflection"

# Reasoning Engine Configuration
reasoning:
  # Currently only "mock" is supported in Phase 1
  provider: "mock"
```

### Running Tests

```bash
# Run all unit tests
make test

# Run tests with verbose output
make test-verbose

# Run integration tests
make test-integration

# Run benchmarks
make bench
```

### Building and Development

```bash
# Build all packages
make build

# Format code
make fmt

# Run linter
make lint
```

## Core Components

### Entity Context

CogMem enforces multi-tenant isolation through entity contexts. All operations must be performed with an entity context, which includes:

- `EntityID` - Unique identifier for the entity
- `UserID` - Optional identifier for a user within the entity

### Long-Term Memory (LTM)

The LTM subsystem provides persistent storage for memories with the following features:

- Multiple backend adapters (SQLite, BoltDB)
- Entity-level isolation
- Access control (private to user, shared within entity)
- Metadata support
- Text-based search

### Memory Management Unit (MMU)

The MMU manages the flow of information between components:

- Encoding data into LTM
- Retrieving memories from LTM using different strategies
- Executing Lua hooks for memory operations
- Preparing for memory consolidation (stub for Phase 2)

### Lua Scripting

CogMem includes a sandboxed Lua scripting engine for customization:

- Hook functions for memory operations (before/after retrieve, before/after encode)
- Sandboxed environment for security
- Script directory scanning and loading
- API for interacting with Go code

### Agent Facade

The Agent provides a unified interface to the CogMem system:

- Processing different input types (store, retrieve, query)
- Coordinating between MMU and reasoning components
- Tracking operations for reflection
- Managing entity context propagation

## Project Structure

```
cogmem-go/
├── cmd/
│   └── example-agent/     # Command-line agent application
├── configs/               # Configuration files
├── migrations/            # SQL migration files
├── pkg/                   # Public library code (main library)
│   ├── agent/             # Agent facade & controller
│   ├── config/            # Configuration loading
│   ├── entity/            # Entity IDs and access levels
│   ├── errors/            # Custom error types
│   ├── mem/               # Memory subsystems
│   │   └── ltm/           # Long-Term Memory interfaces
│   │       └── adapters/  # LTM backend implementations
│   │           ├── kv/    # Key-Value adapters (BoltDB)
│   │           ├── mock/  # Mock adapter for testing
│   │           └── sqlstore/ # SQL adapters (SQLite)
│   ├── mmu/               # Memory Management Unit
│   ├── reasoning/         # Reasoning interfaces and adapters
│   └── scripting/         # Lua scripting engine
├── scripts/               # Lua scripts
│   ├── mmu/               # MMU hook functions
│   └── reflection/        # Reflection scripts
└── test/                  # Tests
    └── integration/       # Integration tests
```

## License

[License details]