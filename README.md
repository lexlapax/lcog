# CogMem

A Modular Cognitive Architecture for LLM Agents with Tiered Memory, Dynamic Processing, and Reflective Adaptation

## Overview

CogMem is a Go library that implements a cognitive architecture for LLM agents, providing:

- Multi-tiered memory management (working memory, long-term memory)
- Entity-based memory isolation for multi-tenant systems
- Pluggable LTM backends (SQL, KV, Vector, Graph databases)
- Lua scripting for customization and extension
- Structured reflection and adaptation capabilities
- Semantic search with vector embeddings (RAG)

## Features

- **Multiple Storage Backends**: Support for various storage technologies
  - Key-Value stores: BoltDB, Redis, PostgreSQL HStore
  - SQL stores: SQLite, PostgreSQL
  - Vector stores: Chromem-go, PostgreSQL pgvector
  - Future: Graph stores

- **Entity Isolation**: Multi-tenant design with strong isolation between entities

- **Access Control**: Private and shared memory support within entities

- **Lua Scripting**: Extensible with Lua for custom memory processing

- **Reasoning Engine**: Integrate with LLMs and other reasoning systems

## Project Status

This project is currently in Phase 2 of development, which is now complete. Phase 2 implemented:

- Vector-based LTM storage using Chromem-go and PostgreSQL pgvector
- OpenAI reasoning engine for embedding generation and processing
- RAG (Retrieval-Augmented Generation) capabilities in the MMU
- Basic reflection module for insight generation
- Enhanced MMU with vector operations and working memory management
- Updated CogMemClient with reflection triggering

## Table of Contents

* [Getting Started](#getting-started)
    * [Prerequisites](#prerequisites)
    * [Installation](#installation)
    * [Running the Example Client](#running-the-example-client)
    * [Configuration](#configuration)
* [Key Documentation](#key-documentation)
* [Core Components](#core-components)
* [Project Structure](#project-structure)
* [Running Tests](#running-tests)
* [Contributing](#contributing)
* [License](#license)

## Getting Started

### Prerequisites

- Go 1.24+ (earlier versions may work but are not tested)
- SQLite3 (for SQL storage option)
- BoltDB (embedded, no separate installation needed)
- OpenAI API key (for embedding generation and LLM reasoning)
- Docker (optional, for running PostgreSQL with pgvector)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/lexlapax/cogmem.git
   cd cogmem
   ```

2. Install dependencies:
   ```bash
   cd cogmem-go
   go mod download
   make deps  # Install dependencies
   ```

3. Set up environment variables:
   - Create a `.env` file with your OpenAI API key and other configuration

### Running the Example Client

The example client provides a simple command-line interface to interact with the CogMem system:

```bash
# Build and run the example client
cd cogmem-go
make run

# Or build and run manually
go build -o bin/example-client ./cmd/example-client
./bin/example-client
```

The example client supports the following commands:

- `!help` - Show help message
- `!entity <id>` - Set the current entity ID
- `!user <id>` - Set the current user ID
- `!remember <text>` - Store a memory in LTM
- `!lookup <query>` - Retrieve memories matching query
- `!search <query>` - Semantic search for memories (RAG)
- `!query <question>` - Ask a question using context from memories
- `!reflect` - Trigger reflection process manually
- `!config` - Show current configuration
- `!quit` - Exit the application

#### Example Usage

```
$ ./bin/example-client

=== CogMem Client ===
LTM Store: chromemgo
Reasoning Engine: openai
Current Entity: default-entity | Current User: default-user
Type !help for available commands.

cogmem::default-user@default-entity> !remember I like dogs especially Golden Retrievers.
Memory stored successfully with ID: 6a8e42f1-9e3d-45a8-af2e-6a0bc76c8e1c

cogmem::default-user@default-entity> !remember Cats are independent and make good pets for busy people.
Memory stored successfully with ID: d4f89c12-3e6c-42f1-90b2-f108d07a5f23

cogmem::default-user@default-entity> !search pets for apartment living
Found 1 memory matching your semantic search:

Memory 1: Cats are independent and make good pets for busy people.
  Created: 2025-04-14T23:56:31Z

cogmem::default-user@default-entity> !query What kind of pets do I like?
I've analyzed your memories and found that you like dogs, especially Golden Retrievers. You also mentioned that cats make good pets for busy people because they're independent.
```

### Configuration

You can configure the CogMem library by creating a `config.yaml` file in the current directory or in the `configs/` directory. Example configurations are available in the `configs/` directory:

- `config.example.yaml` - General example with all options
- `chromemgo.yaml` - Configuration for using Chromem-go vector database
- `pgvector.yaml` - Configuration for using PostgreSQL with pgvector extension

#### Vector LTM Configuration

```yaml
# LTM with Chromem-go vector database
ltm:
  type: "chromemgo"
  chromemgo:
    collection_name: "memories"
    dimensions: 1536
    distance_metric: "cosine"
```

```yaml
# LTM with PostgreSQL pgvector
ltm:
  type: "pgvector"
  pgvector:
    connection_string: "${PGVECTOR_URL}"
    table_name: "memory_vectors"
    dimensions: 1536
    distance_metric: "cosine"
```

#### OpenAI Reasoning Engine Configuration

```yaml
reasoning:
  engine: "openai"
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-3.5-turbo"
    embedding_model: "text-embedding-ada-002"
    max_tokens: 1024
    temperature: 0.7
```

#### Reflection Module Configuration

```yaml
reflection:
  enabled: true
  scripts_path: "scripts/reflection"
  analysis_frequency: 100
  analysis_model: "gpt-3.5-turbo"
  analysis_temperature: 0.3
```

### Database Setup

For PostgreSQL with pgvector:

```bash
# Start development databases
cd cogmem-go
make dev-db-up

# Create test database for integration tests
docker exec -it cogmem_postgres psql -U postgres -c "CREATE DATABASE cogmem_test;"

# Drop test database when needed
docker exec -it cogmem_postgres psql -U postgres -c "DROP DATABASE cogmem_test;"

# Stop development databases
make dev-db-down
```

## Key Documentation

Understand the project goals, design, and plan:

* **Product Requirements:** [./docs/prd.md](./docs/prd.md)
* **Architecture:** [./docs/architecture.md](./docs/architecture.md)
* **Implementation Plan:** [./docs/implementation-plan.md](./docs/implementation-plan.md)
* **Structure Philosophy:** [./project-structure.md](./project-structure.md)

Feature Documentation:

* **RAG (Retrieval-Augmented Generation):** [./docs/rag.md](./docs/rag.md)
* **Reflection Module:** [./docs/reflection.md](./docs/reflection.md)

Development & Testing:

* **PostgreSQL Testing:** [./docs/POSTGRES_TESTING.md](./docs/POSTGRES_TESTING.md)

## Core Components

### Entity Context

CogMem enforces multi-tenant isolation through entity contexts. All operations must be performed with an entity context, which includes:

- `EntityID` - Unique identifier for the entity
- `UserID` - Optional identifier for a user within the entity

### Long-Term Memory (LTM)

The LTM subsystem provides persistent storage for memories with the following features:

- Multiple backend adapters (SQLite, BoltDB, Chromem-go, PostgreSQL pgvector)
- Entity-level isolation
- Access control (private to user, shared within entity)
- Metadata support
- Text-based search
- Vector-based semantic search

### Memory Management Unit (MMU)

The MMU manages the flow of information between components:

- Encoding data into LTM with automatic embedding generation
- Retrieving memories from LTM using different strategies
- Executing Lua hooks for memory operations
- Working memory overflow management
- Semantic search capabilities with vector embeddings

### Reasoning Engine

The Reasoning Engine provides:

- LLM processing with OpenAI models
- Embedding generation for semantic search
- Context-aware processing for RAG use cases

### Reflection Module

The Reflection module enables self-improvement:

- Analyzing memory patterns to derive insights
- Generating structured insights about agent behavior
- Storing insights back into LTM
- Customizable with Lua scripts

### Lua Scripting

CogMem includes a sandboxed Lua scripting engine for customization:

- Hook functions for memory operations (before/after retrieve, before/after encode)
- Reflection hooks for analysis customization
- Sandboxed environment for security
- Script directory scanning and loading
- API for interacting with Go code

### CogMemClient Facade

The CogMemClient provides a unified interface to the CogMem system:

- Processing different input types (store, retrieve, query)
- Coordinating between MMU, reasoning, and reflection components
- Tracking operations for reflection
- Managing entity context propagation
- Triggering reflection cycles

Note: The previous "Agent" interface has been renamed to "CogMemClient" for clarity, with a backward compatibility layer provided.

## Project Structure

```
cogmem/                      # Root project directory
├── CLAUDE.md                # Instructions for Claude AI assistant
├── CONTRIBUTING.md          # Contribution guidelines
├── README.md                # Project README
├── project-structure.md     # Project structure guidelines
├── cogmem-go/               # Golang implementation
│   ├── cmd/
│   │   └── example-client/  # Command-line client application
│   ├── configs/             # Configuration files
│   ├── migrations/          # SQL migration files
│   ├── pkg/                 # Public library code (main library)
│   │   ├── cogmem/          # CogMemClient facade & controller
│   │   ├── config/          # Configuration loading
│   │   ├── entity/          # Entity IDs and access levels
│   │   ├── errors/          # Custom error types
│   │   ├── mem/             # Memory subsystems
│   │   │   └── ltm/         # Long-Term Memory interfaces
│   │   │       └── adapters/ # LTM backend implementations
│   │   │           ├── kv/   # Key-Value adapters (BoltDB)
│   │   │           ├── mock/ # Mock adapter for testing
│   │   │           ├── sqlstore/ # SQL adapters (SQLite, Postgres)
│   │   │           └── vector/ # Vector adapters (Chromem-go, pgvector)
│   │   ├── mmu/             # Memory Management Unit
│   │   ├── reasoning/       # Reasoning interfaces and adapters
│   │   │   └── adapters/    # Reasoning engine adapters (OpenAI, Mock)
│   │   ├── reflection/      # Reflection module
│   │   └── scripting/       # Lua scripting engine
│   ├── scripts/             # Lua scripts
│   │   ├── mmu/             # MMU hook functions
│   │   └── reflection/      # Reflection scripts
│   └── test/                # Tests
│       ├── integration/     # Integration tests
│       └── testutil/        # Test helpers
└── docs/                    # Documentation
    ├── architecture.md      # Architecture documentation
    ├── cogmem-whitepaper-draft.md # Whitepaper draft
    ├── impl-01-phase-1-plan.md # Phase 1 implementation plan
    ├── impl-02-phase-2-plan.md # Phase 2 implementation plan
    ├── implementation-plan.md # Overall implementation plan
    ├── prd.md               # Product requirements document
    ├── project-structure-template.md # Project structure template
    ├── rag.md               # RAG capabilities documentation
    └── reflection.md        # Reflection module documentation
```

## Running Tests

```bash
cd cogmem-go

# Run all unit tests
make test

# Run tests with verbose output
make test-verbose

# Prepare for integration tests
make dev-db-up
docker exec -it cogmem_postgres psql -U postgres -c "CREATE DATABASE cogmem_test;"

# Run integration tests
make test-integration

# Run PostgreSQL-specific tests (HStore, SQLStore, and PgVector)
make test-postgres

# Clean up after integration tests
docker exec -it cogmem_postgres psql -U postgres -c "DROP DATABASE cogmem_test;"
make dev-db-down

# Run benchmarks
make bench
```

## Building and Development

```bash
cd cogmem-go

# Build all packages
make build

# Format code
make fmt

# Run linter
make lint
```

## Contributing

Please read [CONTRIBUTING.md](./CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the LICENSE file for details.