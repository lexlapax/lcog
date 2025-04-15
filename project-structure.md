# Project Structure: CogMem Golang Library

**Version:** 1.0
**Date:** 2023-10-27 (Placeholder)
**Authors:** AI Language Model based on provided context

## 1. Introduction

This document outlines the proposed directory structure for the **CogMem Golang Library**. The structure aims to be clean, scalable, and maintainable, reflecting the modular architecture defined in `architecture.md` and adhering to Go community best practices alongside the principles of Layered Architecture and Module/Package-based organization.

The goal is to create a structure that:
*   Clearly separates public API (`pkg/`) from internal implementation details (`internal/`).
*   Organizes code by architectural component/capability.
*   Facilitates testing at different levels.
*   Makes navigation and contribution straightforward.

## 2. Core Principles Recap

*   **Standard Go Layout:** Adheres to common conventions (`pkg/`, `internal/`, `cmd/`).
*   **Modularity:** Core cognitive components (MMU, LTM, Reflection, etc.) reside in distinct packages within `pkg/`, exposing interfaces and structs for public use.
*   **Layered Architecture:** Dependencies flow inwards. Infrastructure implementations (DB adapters, LLM clients) are typically in sub-packages (`adapters/`) or `internal/`, implementing interfaces defined closer to the core application logic.
*   **Package-Based Cohesion:** Code related to a specific component or functionality lives within its package.
*   **Testability:** Unit tests (`*_test.go`) reside alongside the code they test. Integration tests may live in dedicated `test/` directories or within packages where appropriate.

## 3. Top-Level Directory Structure

cogmem-go/
├── .github/              # CI/CD workflows, issue templates, etc.
├── api/                  # (Optional) Public API data structures (if needed separate from pkg)
├── cmd/                  # Example applications, CLIs, or tools using the library
│   └── example-agent/    # Example: A simple agent demonstrating library usage
│       └── main.go
├── configs/              # Example configuration files (YAML, JSON)
│   └── config.example.yaml
├── internal/             # Private application and library code (not importable by others)
│   └── db/               # Example: Internal DB connection helpers
│   └── lua/              # Example: Internal Lua sandbox setup details
├── migrations/        # Database migration files (SQL)
│   ├── embed.go       # Go file to embed migrations using //go:embed
│   ├── 0001_init_schema.up.sql
│   ├── 0001_init_schema.down.sql
│   ├── 0002_create_users_table.up.sql
│   ├── 0002_create_users_table.down.sql
│   └── ...
├── pkg/                  # Public library code (importable by others) - THE CORE LIBRARY
│   ├── agent/            # Main agent facade & Executive Controller (or pkg/cogmem/)
│   ├── config/           # Configuration loading structs and logic
│   ├── entity/           # EntityID, AccessLevel, context definitions
│   ├── errors/           # Custom error types
│   ├── mem/              # Memory subsystems
│   │   ├── ltm/          # Long-Term Memory interface and adapters
│   │   └── wm/           # Working Memory management helpers/interfaces
│   ├── mmu/              # Memory Management Unit implementation
│   ├── reflection/       # Reflection module implementation
│   ├── reasoning/        # Reasoning engine interface and adapters
│   ├── perception/       # Perception module interface (basic impl)
│   ├── action/           # Action module interface (basic impl)
│   ├── scripting/        # Lua scripting engine integration (gopher-lua)
│   └── ports/            # (Alternative) Central location for core interfaces
├── scripts/              # Default/example Lua scripts
│   ├── mmu/
│   │   └── retrieval_filter.lua
│   └── reflection/
│       └── analysis.lua
├── test/                 # Integration tests, shared test utilities
│   └── integration/
│       └── mmu_ltm_test.go
├── .gitignore
├── go.mod                # Go module definition
├── go.sum                # Go module checksums
├── LICENSE
└── README.md             # Project README

## 4. Detailed `pkg/` Structure Breakdown

This is the heart of the library, containing code intended for public consumption.

*   **`pkg/agent/` (or `pkg/cogmem/`)**
    *   `agent.go`: Defines the main `Agent` struct and its methods. Acts as the primary facade and Executive Controller.
    *   `interfaces.go`: (Optional) Defines interfaces implemented *by* the agent or expected *by* the agent if needed.
    *   `controller.go`: Implementation of the executive control loop logic.
    *   `agent_test.go`: Unit tests for the agent facade and controller logic.

*   **`pkg/entity/`**
    *   `entity.go`: Defines `EntityID` (e.g., `type EntityID string`), `AccessLevel` (e.g., `type AccessLevel int`), `Context` struct.
    *   `context.go`: Helper functions for creating and managing entity context within `context.Context`.
    *   `entity_test.go`: Unit tests for entity types and helpers.

*   **`pkg/mem/`**
    *   **`ltm/`**
        *   `ltm.go`: Defines the core `LTMStore` interface. Defines `MemoryRecord` struct and other shared LTM types.
        *   `errors.go`: LTM-specific errors.
        *   `ltm_test.go`: Tests for shared LTM types/helpers.
        *   **`adapters/`**: Concrete implementations of `LTMStore`.
            *   **`mock/`**:
                *   `mock.go`: In-memory mock implementation for testing.
            *   **`sqlstore/`**:
                *   `sqlstore.go`: (Optional) Common SQL DB `sqlc` logic or sub-interfaces.
                *   `postgres/postgres.go`: sqlc/pgx adapter implementing `LTMStore`.
                *   `sqlite/sqlite3.go`: sqlc/sqlite3 adapter implementing `LTMStore`.
                *   `postgres/postgres_test.go`: Tests for postgres adapter (likely integration tests requiring a postgres instance instance).
            *   **`vector/`**:
                *   `vector.go`: (Optional) Common vector DB logic or sub-interfaces.
                *   `postgres/postgres_pgvector.go`: postgres `pgvector` adapter implementing `LTMStore`.
                *   `weaviate/weaviate.go`: Weaviate adapter implementing `LTMStore`.
                *   `milvus/milvus.go`: Milvus adapter implementing `LTMStore`.
                *   `postgres/postgres_pgvector.go`: Tests for pgvector adapter (likely integration tests requiring a postgres instance).
            *   **`graph/`**:
                *   `postgres/postgres_apacheage.go`: Adapter for interacting `apache_age`graph extension in postges.
                *   `neo4j/neo4j.go`: Adapter for interacting with a neo4j server instance.
                *   `dgraph/dgraph.go`: Dgraph adapter.
                *   `postgres/postgres_apacheage_test.go`: Tests for interacting `apache_age` postgres instance
            *   **`kv/`**:
                *   `postgres/postgres_hstore.go`: postgres kv store leveraging `hstore` extension.
                *   `redis/redis.go`: Redis adapter.
                *   `postgres/postgres_hstore_test.go`: Tests for interacting with `hstore` postgres
    *   **`wm/`**
        *   `manager.go`: Defines `WorkingMemoryManager` interface (if used) or helper functions for managing context size / formatting.
        *   `manager_test.go`: Tests for WM helpers.

*   **`pkg/mmu/`**
    *   `mmu.go`: Defines the `MMU` interface and default implementation struct. Contains core logic for encode/retrieve/consolidate/overflow.
    *   `retrieval.go`: Implementation of different retrieval strategies.
    *   `consolidation.go`: Implementation of consolidation logic.
    *   `lua_hooks.go`: Code specifically handling calls to the scripting engine for MMU hooks.
    *   `mmu_test.go`: Unit tests for MMU logic (mocking LTMStore, ScriptingEngine).

*   **`pkg/reflection/`**
    *   `reflection.go`: Defines `ReflectionModule` interface and implementation. Orchestrates analysis and feedback loop.
    *   `analyzer.go`: Logic for analyzing history/traces.
    *   `insight.go`: Logic for generating insights.
    *   `lua_hooks.go`: Code handling calls to Lua for reflection hooks.
    *   `reflection_test.go`: Unit tests (mocking MMU, Reasoning, Scripting).

*   **`pkg/reasoning/`**
    *   `engine.go`: Defines the `ReasoningEngine` interface.
    *   **`adapters/`**: Concrete implementations.
        *   `openai/openai.go`: Adapter for OpenAI API.
        *   `anthropic/anthropic.go`: Adapter for Anthropic API.
        *   `mock/mock.go`: Mock implementation for testing.

*   **`pkg/perception/` / `pkg/action/`**
    *   `interface.go`: Defines the `PerceptionModule` / `ActionModule` interface.
    *   `basic/basic.go`: (Optional) Simple text-based default implementation.

*   **`pkg/scripting/`**
    *   `engine.go`: Defines the `Engine` interface and implementation wrapping `gopher-lua`. Handles loading, execution, sandboxing.
    *   `sandbox.go`: Specific code for setting up the Lua sandbox environment.
    *   `api.go`: Defines and exposes the Go functions callable from Lua scripts.
    *   `engine_test.go`: Tests for Lua engine wrapper (potentially executing trivial Lua scripts).

*   **`pkg/config/`**
    *   `config.go`: Defines Go structs representing the library's configuration schema.
    *   `load.go`: Functions to load configuration from files/env vars (e.g., using Viper).
    *   `config_test.go`: Tests for config loading and validation.

*   **`pkg/errors/`**
    *   `errors.go`: Defines common custom error types used across the library.

*   **`pkg/ports/` (Alternative Structure)**
    *   If preferred, core interfaces (`LTMStore`, `ReasoningEngine`, `ScriptingEngine`, etc.) could be centralized here instead of living within the primary implementing package (`ltm`, `reasoning`, `scripting`). This can sometimes help break import cycles but can also reduce cohesion. The primary structure proposed above keeps interfaces closer to their main implementations/consumers.

## 5. `internal/` Usage

This directory contains code that is essential for the library to function but is not intended to be part of the public API. Go enforces this; code in `internal/` cannot be imported by projects outside the current Go module (`cogmem-go`).

*   **`internal/dbutil/`**: Might contain helper functions for database connection pooling, query building helpers specific to an adapter, etc., used by `pkg/mem/ltm/adapters/*`.
*   **`internal/luautil/`**: Might contain complex helper functions for Go<->Lua data conversion, detailed sandbox setup logic used by `pkg/scripting`.
*   **`internal/httpclient/`**: Common HTTP client setup used by various adapters (LLM, Zep, etc.).

## 6. `migrations/`
    *   Contains raw SQL migration files (`.up.sql`, `.down.sql`). This is generally preferred over Go-based migrations for libraries as it's more explicit and portable.
    *   **`embed.go`:** Uses `//go:embed` directive to embed the migration files directly into the compiled library binary. This makes distribution easier – the consumer doesn't need separate migration files.

    ```go
    // migrations/embed.go
    package migrations

    import "embed"

    //go:embed *.sql
    var FS embed.FS
    ```

## 7. `cmd/` Usage

This directory holds example programs or command-line tools that *use* the CogMem library. They demonstrate how to import and wire together the library components.

*   **`cmd/example-agent/main.go`**: A simple application that:
    *   Loads configuration (`pkg/config`).
    *   Instantiates LTM adapter, Reasoning adapter, Scripting engine.
    *   Instantiates the `pkg/agent.Agent` facade, injecting the dependencies.
    *   Runs a simple interaction loop, processing input with a specific `entity.Context`.

## 8. `scripts/` Directory

Contains default or example Lua scripts corresponding to the extension points defined in the architecture.

*   **`scripts/mmu/retrieval_filter.lua`**: Example Lua script for filtering/ranking retrieved LTM records.
*   **`scripts/reflection/analysis.lua`**: Example Lua script for custom reflection analysis logic.
*   Users will typically copy and modify these or provide paths to their own scripts in the configuration.

## 9. `configs/` Directory

Contains example configuration files.

*   **`configs/config.example.yaml`**: Shows the structure and available options for configuring the library (LLM keys, LTM connection strings, script paths, etc.).

## 10. `test/` Directory

Contains tests that span multiple packages, typically integration tests.

*   **`test/integration/mmu_ltm_test.go`**: An integration test verifying the interaction between the MMU (`pkg/mmu`) and a real (or containerized) LTM backend via its adapter (`pkg/mem/ltm/adapters/...`).
*   **`test/multitenancy/isolation_test.go`**: Tests specifically designed to verify entity data isolation across multiple packages.


## 11. Dependency Management

*   **`go.mod` / `go.sum`**: Standard Go module files manage project dependencies. Dependencies should be kept minimal and justifiable.

This structure provides a clear separation of concerns, promotes modularity, aligns with Go conventions, and directly supports the development and maintenance of the CogMem library as described in the architecture document.
