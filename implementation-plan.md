## implementation-plan.md

# Implementation Plan: CogMem Golang Library (Test-First)

**Version:** 2.0
**Date:** 2023-10-27 (Placeholder)
**Authors:** AI Language Model based on provided context

## 1. Introduction

This document outlines a revised 3-phase implementation plan for the CogMem Golang Library, designed to align with the updated `project-structure.md` (v1.0). This plan prioritizes a **Test-First Development (TFD)** approach, starting with foundational components, basic structured memory (SQL/KV), and Lua scripting integration in Phase 1, moving to vector-based memory in Phase 2, and concluding with graph-based memory, advanced features, and collaboration support in Phase 3.

The Test-First approach means that for each significant piece of functionality, tests (unit or integration) will be written *before* the implementation code, driving the design and verifying correctness from the outset.

## 2. Phase 1: Foundation, SQL/KV LTM & Lua Scripting

**Goal:** Establish the core library structure, interfaces, and testing infrastructure. Implement foundational memory capabilities using SQL (via `sqlc`) and KV stores, integrate Lua scripting with basic hooks, and ensure robust multi-entity context handling. Deliver a minimally viable library capable of structured data storage/retrieval and basic scriptable logic.

**Key Features / Modules / Tasks (Test-First Approach):**

1.  **Setup & Core Structure:**
    *   Initialize Go module, Git repository, CI basics (.github/).
    *   Create the top-level directory structure (`pkg/`, `internal/`, `cmd/`, `configs/`, `scripts/`, `migrations/`, `test/`).
    *   Define core interfaces in `pkg/` (start with `LTMStore`, `MMU`, `ReasoningEngine`, `ScriptingEngine`). Write basic interface satisfaction tests (can they be instantiated, do methods exist?).
    *   Implement `pkg/entity`: Define types (`EntityID`, `AccessLevel`, `Context`), write unit tests for context creation/management helpers.
    *   Implement `pkg/errors`: Define initial custom error types, write tests ensuring they wrap standard errors correctly.
    *   Implement `pkg/config`: Define config structs, write tests for loading/validation using `configs/config.example.yaml`.

2.  **Testing Infrastructure:**
    *   Set up Docker Compose files or test helpers in `test/` for spinning up PostgreSQL (for SQL, KV, later Vector/Graph extensions) and Redis instances for integration testing.
    *   Implement basic mocking utilities (or decide on a library like `testify/mock`).

3.  **LTM Foundation & SQL/KV Adapters:**
    *   **Tests:** Write integration tests for the `LTMStore` interface targeting SQL (Postgres, potentially SQLite) and KV (Redis, Postgres HStore) storage. These tests *must* cover:
        *   CRUD operations (`Store`, `Retrieve`, `Update`, `Delete`) for `MemoryRecord`.
        *   Strict `EntityID` filtering (data for entity A is not visible to entity B).
        *   Basic `AccessLevel` filtering (e.g., retrieving only `private` for a specific user context).
        *   Handling of `MemoryRecord` serialization/deserialization.
    *   **Schema & Migrations:** Define initial SQL schema in `migrations/` (e.g., `0001_init_schema.up.sql`, `0002_create_memory_table.up.sql`). Implement `migrations/embed.go`. Write tests to ensure migrations can be applied and rolled back against a test DB using a library like `golang-migrate`.
    *   **SQL Adapter (`pkg/mem/ltm/adapters/sqlstore/`):** Set up `sqlc` configuration. Write SQL queries (`query.sql`). Run `sqlc generate`. Implement the adapter (`postgres/postgres.go`, `sqlite/sqlite3.go`) using the generated code to satisfy the `LTMStore` interface and pass the previously written integration tests. Use `internal/dbutil` for connection pooling if needed.
    *   **KV Adapter (`pkg/mem/ltm/adapters/kv/`):** Implement the adapters (`redis/redis.go`, `postgres/postgres_hstore.go`) to satisfy the `LTMStore` interface and pass the KV integration tests.

4.  **Lua Scripting Integration (`pkg/scripting/`):**
    *   **Tests:** Write unit tests for the `ScriptingEngine` interface:
        *   Loading scripts from files/strings.
        *   Executing simple Lua functions.
        *   Basic sandboxing (verifying dangerous modules are disabled).
        *   Go<->Lua data marshaling (passing basic Go structs/maps/slices to Lua and back).
        *   Testing the defined Go API exposed to Lua (e.g., calling a mock Go function from Lua).
    *   **Implementation:** Implement the `ScriptingEngine` using `gopher-lua`. Implement sandboxing (`sandbox.go`) and the Go API (`api.go`). Ensure tests pass.

5.  **Basic MMU (`pkg/mmu/`):**
    *   **Tests:** Write unit tests for the `MMU` interface (mocking `LTMStore` and `ScriptingEngine`):
        *   `encode_to_ltm`: Verifying it calls `LTMStore.Store` with correct `MemoryRecord` and `entity.Context`.
        *   `retrieve_from_ltm`: Verifying it calls `LTMStore.Retrieve` with correct filters based on `entity.Context` and options.
        *   (Minimal) Lua Hook Invocation: Verify that specific MMU actions (e.g., before/after retrieve) attempt to call corresponding functions in the (mocked) `ScriptingEngine`.
    *   **Implementation:** Implement the basic MMU logic to pass the unit tests. Integrate calls to the `ScriptingEngine` at basic hook points (`lua_hooks.go`).

6.  **Basic Reasoning Engine (`pkg/reasoning/`):**
    *   **Tests:** Write unit tests for the `ReasoningEngine` interface using a mock adapter.
    *   **Implementation:** Implement the interface definition and a `mock/mock.go` adapter. *(Optional: Implement a basic real adapter like `openai/openai.go` if needed for MMU testing, mocking the HTTP client)*.

7.  **Agent Facade & Loop (`pkg/agent/`):**
    *   **Tests:** Write unit tests for the `Agent` facade, mocking all its dependencies (MMU, Reasoning, etc.). Test the basic control flow orchestration and `entity.Context` propagation.
    *   **Implementation:** Implement the `Agent` struct and the basic controller loop to pass tests.

8.  **Examples (`cmd/example-agent/`):**
    *   Create a minimal working example demonstrating library setup, configuration loading, agent instantiation (using SQL/KV LTM and mock reasoning), and processing a simple interaction loop. This serves as an end-to-end integration check.

**Testing Focus:** Heavy emphasis on unit tests for isolated logic. Crucial integration tests for DB adapters (SQL/KV) verifying storage, retrieval, and multi-entity isolation against real (containerized) databases. Migration system testing. Basic Lua execution and hook integration tests.

**Outcome / Deliverable:** A functional core library with established testing practices. Supports persistent, structured memory storage (SQL/KV) with enforced multi-entity isolation. Basic Lua scripting is integrated for customizing simple MMU logic points. A minimal agent loop is functional.

## 3. Phase 2: Vector LTM & Basic RAG / Reflection

**Goal:** Implement vector-based long-term memory for semantic search (RAG), introduce the foundational reflection module, and refine the agent loop to support these capabilities.

**Key Features / Modules / Tasks (Test-First Approach):**

1.  **Vector LTM Adapters (`pkg/mem/ltm/adapters/vector/`):**
    *   **Tests:** Write integration tests for the `LTMStore` interface targeting vector databases (`postgres/postgres_pgvector.go`, `weaviate/weaviate.go`, `milvus/milvus.go`). These tests *must* cover:
        *   Storing records with vector embeddings.
        *   Semantic retrieval based on a query vector.
        *   Correct `EntityID` filtering combined with vector search.
        *   Handling metadata filtering alongside vector search (if supported by DB).
    *   **Implementation:** Implement the vector adapters, ensuring they pass the integration tests. This involves integrating with vector DB clients and handling embedding data. PostgreSQL adapter requires managing the `pgvector` extension (likely via migrations from Phase 1).

2.  **MMU Enhancements:**
    *   **Tests:** Write unit/integration tests for:
        *   Embedding Generation: Verify `encode_to_ltm` calls the `ReasoningEngine` (mocked) to get embeddings before calling `LTMStore.Store` (mocked or real vector adapter).
        *   Semantic Retrieval Strategy: Verify `retrieve_from_ltm` correctly invokes the semantic search capabilities of the underlying vector `LTMStore`.
        *   WM Overflow: Test basic heuristics for selecting memory to move from WM to LTM (mocking WM state).
        *   Lua Hooks: Test new Lua hooks related to embedding generation or vector result ranking.
    *   **Implementation:** Enhance `pkg/mmu` to include embedding generation logic (calling `ReasoningEngine`), implement the semantic retrieval strategy, add basic WM overflow logic, and integrate any new Lua hooks.

3.  **Reasoning Engine (Embeddings):**
    *   **Tests:** If not done in Phase 1, add tests for a real `ReasoningEngine` adapter (e.g., OpenAI) specifically for its embedding capability (mocking the HTTP call).
    *   **Implementation:** Ensure the chosen reasoning adapter can reliably generate embeddings.

4.  **Reflection Module Basics (`pkg/reflection/`):**
    *   **Tests:** Write unit tests for the `ReflectionModule` interface (mocking `MMU` and `ReasoningEngine`):
        *   Basic Analysis Triggering (e.g., on specific error types, or manual invocation).
        *   Analysis Logic: Verify it formats history/context and calls `ReasoningEngine.Process` for analysis.
        *   Insight Generation: Verify it parses LLM analysis response into a structured insight.
        *   Consolidation Trigger: Verify it calls `MMU.ConsolidateLTM` (mocked) with the generated insight and correct `entity.Context`.
        *   Lua Hooks: Test basic hooks for triggering or formatting insights.
    *   **Implementation:** Implement the basic reflection module structure (`reflection.go`, `analyzer.go`, `insight.go`) and integrate basic Lua hooks (`lua_hooks.go`). Implement the `ConsolidateLTM` *method signature* in the MMU interface and a basic implementation (e.g., just logging or storing the insight as a new memory record via `encode_to_ltm`).

5.  **Agent Loop Integration:**
    *   **Tests:** Update `Agent` unit tests to verify integration of RAG results into the reasoning context and basic triggering of the reflection module.
    *   **Implementation:** Refine `pkg/agent` control loop to handle the flow: Retrieve (Vector) -> Augment Context -> Reason. Integrate reflection triggering (e.g., post-action).

6.  **Examples & Config:** Update `cmd/example-agent` to demonstrate RAG. Update `configs/config.example.yaml` with vector DB and basic reflection settings.

**Testing Focus:** Integration tests for vector DB adapters are critical. Unit and integration tests for MMU embedding/retrieval logic. Unit tests for the reflection module's core flow (trigger -> analyze -> insight -> consolidate trigger).

**Outcome / Deliverable:** The library now supports semantic search (RAG) via vector LTMs alongside structured storage. A foundational reflection mechanism is in place, enabling basic analysis and insight storage within entity contexts.

## 4. Phase 3: Graph LTM, Collaboration & Advanced Features

**Goal:** Implement graph-based LTM for relationship-aware memory, enable multi-agent collaboration via shared memory with concurrency control, mature the reflection loop, enhance Lua scripting capabilities, and finalize documentation.

**Key Features / Modules / Tasks (Test-First Approach):**

1.  **Graph LTM Adapters (`pkg/mem/ltm/adapters/graph/`):**
    *   **Tests:** Write integration tests for the `LTMStore` interface targeting graph databases (`postgres/postgres_apacheage.go`, `neo4j/neo4j.go`, `dgraph/dgraph.go`). Tests must cover:
        *   Storing nodes and relationships representing memories/entities.
        *   Retrieval based on graph traversal (e.g., finding related memories).
        *   `EntityID` scoping within the graph structure (e.g., partitioning, labeling).
        *   Temporal queries (if supported by DB/adapter design, e.g., mimicking Zep).
    *   **Implementation:** Implement the graph adapters. PostgreSQL adapter requires managing the `apache_age` extension.

2.  **Multi-Agent Shared Memory & Concurrency:**
    *   **Tests:** Write specific integration tests (likely using SQL or Graph adapters):
        *   Shared Write/Read: Verify agents with the same `EntityID` can write to and read memory marked `shared_within_entity`.
        *   Isolation: Re-verify agents with different `EntityID`s *cannot* access shared data.
        *   Concurrency: Simulate concurrent writes to the same shared memory record/node from different "agents" (goroutines) and verify that locking/conflict resolution prevents data corruption (requires careful test setup).
    *   **Implementation:**
        *   Refine `LTMStore` interface/adapters to fully support the `shared_within_entity` `AccessLevel`.
        *   Implement concurrency control mechanisms (e.g., optimistic locking using version fields in SQL/KV/Nodes, or potentially leveraging DB transaction isolation) within relevant adapters (`sqlstore`, `kv`, `graph`).

3.  **MMU Advanced Features:**
    *   **Tests:** Write unit/integration tests for:
        *   Graph Retrieval Strategy: Verify `retrieve_from_ltm` utilizes graph traversal capabilities.
        *   Iterative Retrieval: Test logic that refines queries over multiple LTM calls (mocking LTM responses initially).
        *   Advanced Consolidation: Test `ConsolidateLTM` logic that performs meaningful updates (e.g., merging nodes in graph LTM, updating facts in SQL LTM based on reflection insights). Test Lua hooks for consolidation rules.
    *   **Implementation:** Implement advanced retrieval strategies (graph, iterative) and more sophisticated `ConsolidateLTM` logic in `pkg/mmu`.

4.  **Reflection Enhancements:**
    *   **Tests:** Write unit/integration tests for:
        *   Advanced Triggers (e.g., based on detected surprise, specific task failures).
        *   Sophisticated Analysis using LLM + potentially graph context.
        *   Reflection directly modifying LTM state via `MMU.ConsolidateLTM`.
        *   Lua-driven analysis or insight generation.
    *   **Implementation:** Enhance `pkg/reflection` with more advanced triggers and analysis capabilities, potentially leveraging graph context. Ensure insights drive meaningful consolidation actions.

5.  **Lua Scripting Enhancements:**
    *   **Tests:** Add tests for any new Go<->Lua API functions or more complex script interactions needed for advanced MMU/Reflection hooks.
    *   **Implementation:** Expand Lua API (`pkg/scripting/api.go`) and hook integrations as needed based on MMU/Reflection requirements.

6.  **Documentation & Polish:**
    *   Write comprehensive README, godoc comments, and potentially a separate documentation site. Cover architecture, usage, configuration, Lua scripting, multi-tenancy, examples.
    *   Refine APIs, error handling, and logging based on experience from previous phases.
    *   Create more diverse examples in `cmd/`.

7.  **(Optional) Benchmarking:** Develop and run benchmarks for key operations (retrieval latency/throughput for different LTM types, reflection loop time).

**Testing Focus:** Integration tests for graph adapters and graph-based retrieval. Critical integration tests for shared memory access and concurrency control. End-to-end tests for advanced reflection loops modifying LTM state. Testing complex Lua script interactions.

**Outcome / Deliverable:** A feature-rich CogMem library supporting diverse memory backends (SQL, KV, Vector, Graph), robust multi-entity isolation, secure multi-agent collaboration via shared memory, advanced adaptation via reflection, and extensive customization via Lua scripting. The library is well-tested, documented, and suitable for building sophisticated LLM agents.