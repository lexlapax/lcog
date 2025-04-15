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

This project is currently in Phase 1 of development, focused on establishing core components:

- Basic library structure and interfaces
- SQL and KV-based long-term memory
- Lua scripting integration
- Multi-entity context handling

## Getting Started

### Prerequisites

- Go 1.24+ (earlier versions may work but are not tested)
- Docker and Docker Compose (for running tests with databases)
- PostgreSQL (for SQL/KV storage option)
- Redis (for KV storage option)
- `sqlc` (for generating database client code)
- `golangci-lint` (for linting)

### Installation

```bash
git clone https://github.com/lexlapax/cogmem.git
cd cogmem/cogmem-go
go mod download
make deps  # Install dependencies
```

### Running Tests

```bash
# Run all unit tests
make test

# Run tests with verbose output
make test-verbose

# Run integration tests (requires database)
make test-integration

# Run benchmarks
make bench
```

### Building and Running

```bash
# Build all packages
make build

# Run the example agent
make run

# Format code
make fmt

# Run linter
make lint

# Generate SQL client code
make sqlc-gen
```

## Project Structure

```
cogmem-go/
├── .github/               # CI/CD workflows, issue templates
├── api/                   # Public API data structures (if needed)
├── cmd/                   # Example applications and CLI tools
│   └── example-agent/     # Simple agent demo
├── configs/               # Configuration files
├── internal/              # Private application code
│   ├── db/                # Internal DB connection helpers
│   └── lua/               # Internal Lua sandbox details
├── migrations/            # SQL migration files
├── pkg/                   # Public library code (main library)
│   ├── agent/             # Agent facade & controller
│   ├── config/            # Configuration loading
│   ├── entity/            # Entity IDs and access levels
│   ├── errors/            # Custom error types
│   ├── mem/               # Memory subsystems
│   │   ├── ltm/           # Long-Term Memory interfaces
│   │   │   └── adapters/  # LTM backend implementations
│   │   └── wm/            # Working Memory
│   ├── mmu/               # Memory Management Unit
│   ├── reflection/        # Reflection module
│   ├── reasoning/         # Reasoning interfaces and LLM adapters
│   ├── perception/        # Perception module interface
│   ├── action/            # Action module interface
│   ├── scripting/         # Lua scripting engine
│   └── ports/             # (Alternative) Core interfaces
├── scripts/               # Lua scripts
│   ├── mmu/
│   └── reflection/
└── test/                  # Integration tests
    └── integration/
```

## Documentation

- [Architecture](../architecture.md)
- [Implementation Plan](../implementation-plan.md)
- [Phase 1 Plan](../impl-01-phase-1-plan.md)

## License

[License details]