# CogMem Golang Library: Implementation Plan (Test-First)

**Version:** 3.0 (Phase Plan based on Project Structure v1.0 rev 2)
**Date:** 2023-10-27 (Placeholder)
**Corresponding Project Structure Version:** 1.0 rev 2

## 1. Introduction

This document outlines a detailed 3-phase implementation plan for the CogMem Golang Library, designed to align with the `project-structure.md` (v1.0 rev 2). This plan adopts a **Test-First Development (TFD)** approach throughout.
*   **Phase 1:** Focuses on establishing the foundational library structure, core interfaces, testing infrastructure, implementing basic memory storage using **SQLite** and **BoltDB**, integrating Lua scripting, and ensuring multi-entity context handling.
*   **Phase 2:** Centers on implementing vector-based long-term memory using **Milvus** for Retrieval-Augmented Generation (RAG) capabilities and introducing the basic reflection module for adaptation.
*   **Phase 3:** Concentrates on adding graph-based LTM (prioritizing **Postgres/Apache Age**), enabling multi-agent collaboration via shared memory, maturing the reflection loop, enhancing Lua scripting, and finalizing documentation.

## 2. Overall Approach: Test-First Development (TFD)

For each significant piece of functionality:
1.  **Write Tests:** Define unit or integration tests that specify the desired behavior and cover primary use cases and edge conditions *before* writing implementation code.
2.  **Implement:** Write the minimum code necessary to make the tests pass.
3.  **Refactor:** Improve the code's structure, clarity, and efficiency while ensuring tests continue to pass.

---

## 3. Phase 1: Foundation, SQLite/BoltDB LTM & Lua Scripting

**Goal:** Establish the core library structure, interfaces, and testing infrastructure. Implement foundational memory capabilities using SQLite (via `sqlc`) and BoltDB KV store, integrate Lua scripting (`gopher-lua`) with basic hooks, and ensure robust multi-entity context handling. Deliver a minimally viable library capable of structured data storage/retrieval (focused on embedded/file-based options first) and basic scriptable logic, driven by tests.

**Detailed Steps:**

1.  **Project Initialization & Setup:**
    *   Initialize the Git repository.
    *   Initialize the Go module using `go mod init`.
    *   Create the top-level directory structure including `pkg/`, `internal/`, `cmd/`, `configs/`, `scripts/`, `migrations/`, `test/`, and necessary subdirectories within `pkg/` (e.g., `agent`, `entity`, `mem/ltm/adapters/sqlstore/sqlite`, `mem/ltm/adapters/kv/boltdb`, `mmu`, `scripting`).
    *   Add essential project files like `.gitignore`, a basic `README.md`, and `LICENSE`.
    *   Configure a basic CI pipeline (e.g., GitHub Actions) to automatically build and run tests (`go build ./...`, `go test ./...`).

2.  **Core Types, Interfaces & Errors:**
    *   **Implement `pkg/entity` (TDD):**
        *   Write unit tests (`entity_test.go`) for `EntityID`, `AccessLevel`, and `entity.Context` struct creation/validation.
        *   Implement the types and struct (`entity.go`).
        *   Write unit tests (`context_test.go`) for helper functions managing `entity.Context` within Go's `context.Context`.
        *   Implement the context helper functions (`context.go`).
    *   **Implement `pkg/errors` (TDD):**
        *   Write unit tests (`errors_test.go`) defining expected behavior for custom errors (wrapping standard errors, checking via `errors.Is`/`As`).
        *   Implement initial custom error types/variables (`errors.go`).
    *   **Define Core Interfaces:**
        *   Define the `LTMStore` interface in `pkg/mem/ltm/ltm.go` with methods for CRUD operations, ensuring signatures accept `context.Context`.
        *   Define the `MMU` interface in `pkg/mmu/mmu.go` with methods for encoding, retrieving, and consolidating LTM data.
        *   Define the `ScriptingEngine` interface in `pkg/scripting/engine.go` with methods for loading and executing Lua scripts/functions.
        *   Define the `ReasoningEngine` interface in `pkg/reasoning/engine.go` with methods for processing prompts and generating embeddings.

3.  **Configuration (`pkg/config`):**
    *   **Implement (TDD):**
        *   Write unit tests (`config_test.go`) for loading YAML configuration, defining the `Config` struct within the test first (include fields for `LTM.Type`, `LTM.SQLite.Path`, `LTM.BoltDB.Path`). Test defaults, required fields, env var overrides.
        *   Implement the `Config` struct and loading logic (`config.go`, `load.go`) using a library like `viper`.
    *   **Create Example:** Add `configs/config.example.yaml` matching the defined struct.

4.  **Testing Infrastructure (`test/`):**
    *   **Implement Mocking:** Add `testify/mock` dependency for creating mock objects in unit tests.
    *   **Implement Test Helpers:** Create utilities in `test/testutil/` to manage temporary SQLite database files and BoltDB database files for integration tests, ensuring clean state for each test run.

5.  **LTM - Mock Adapter (`pkg/mem/ltm/adapters/mock/`):**
    *   **Define `MemoryRecord`:** Specify the `MemoryRecord` struct in `pkg/mem/ltm/ltm.go` with necessary fields (ID, EntityID, UserID, AccessLevel, Content, Metadata, Embedding, Timestamps).
    *   **Implement (TDD):**
        *   Write unit tests (`mock_test.go`) covering the `LTMStore` interface for the mock adapter (CRUD, EntityID filtering, AccessLevel filtering, isolation).
        *   Implement the mock adapter (`mock.go`) using in-memory storage (e.g., nested maps) with mutexes for basic concurrency safety.

6.  **LTM - SQLite Adapter (`pkg/mem/ltm/adapters/sqlstore/sqlite/`):**
    *   **Implement (TDD):**
        *   Write integration tests (`sqlite3_test.go`) using temporary DB files via test helpers. Tests must verify CRUD, strict `EntityID` isolation, `AccessLevel` filtering, and correct data type mapping for SQLite.
        *   Configure `sqlc` for SQLite (`sqlc.yaml`). Write SQLite-compatible SQL queries (`query.sql`). Generate Go code using `sqlc generate`. *(Note: Schema creation might happen directly in test setup or adapter initialization for SQLite)*.
        *   Implement the adapter (`sqlite3.go`) using the generated code and `database/sql` driver to pass integration tests.

7.  **LTM - BoltDB Adapter (`pkg/mem/ltm/adapters/kv/boltdb/`):**
    *   **Implement (TDD):**
        *   Write integration tests (`boltdb_test.go`) using temporary DB files. Verify CRUD operations using BoltDB buckets (e.g., `entityID -> recordID -> marshaled Record`), ensure entity isolation via buckets, and test basic access level filtering (likely within the marshaled data).
        *   Implement the adapter (`boltdb.go`) using the `go.etcd.io/bbolt` library, employing BoltDB transactions and bucket operations to pass tests.

8.  **Lua Scripting Engine (`pkg/scripting/`):**
    *   **Implement (TDD):**
        *   Write unit tests (`engine_test.go`) for the `ScriptingEngine` implementation covering script loading, function execution, error handling, and sandboxing (verifying disabled modules like `os`, `io`).
        *   Implement the engine (`engine.go`) using `gopher-lua`, ensuring proper state initialization and selective opening of safe Lua libraries.
        *   Write unit tests (`api_test.go`) for Go functions intended to be callable from Lua. Test argument passing (Go->Lua, Lua->Go), return values, and complex type marshalling (structs/maps/slices to Lua tables).
        *   Implement the Go API functions (`api.go`) and expose them to the Lua state during initialization (`sandbox.go`).
    *   **Create Examples:** Add basic placeholder Lua scripts in `scripts/` (e.g., `scripts/mmu/retrieval_filter.lua`).

9.  **Basic Memory Management Unit (MMU) (`pkg/mmu/`):**
    *   **Implement (TDD):**
        *   Write unit tests (`mmu_test.go`) for the `MMU` implementation, mocking `LTMStore` and `ScriptingEngine`. Test core `EncodeToLTM` and `RetrieveFromLTM` logic, ensuring correct calls to dependencies with proper `entity.Context`. Test invocation of basic Lua hooks (e.g., `before_retrieve`, `after_retrieve`) and handling of their potential errors.
        *   Implement the basic MMU logic (`mmu.go`) and integrate calls to the scripting engine at defined hook points (`lua_hooks.go`).

10. **Reasoning Engine - Mock (`pkg/reasoning/`):**
    *   **Implement (TDD):**
        *   Write unit tests (`adapters/mock/mock_test.go`) for the mock `ReasoningEngine` adapter, covering setting canned responses/embeddings and verifying method calls.
        *   Implement the mock adapter (`adapters/mock/mock.go`).

11. **Agent Facade & Loop (`pkg/agent/`):**
    *   **Implement (TDD):**
        *   Write unit tests (`agent_test.go`) for the `Agent` facade, mocking all dependencies. Test the primary control flow orchestration and correct `entity.Context` propagation. Test basic error handling from dependencies.
        *   Implement the `Agent` struct (accepting dependencies via constructor) and the basic controller logic (`agent.go`, `controller.go`).

12. **Example Application (`cmd/example-agent/`):**
    *   **Implement:** Create `cmd/example-agent/main.go`. This application should load configuration, instantiate the chosen LTM adapter (SQLite or BoltDB), scripting engine, mock reasoning engine, and the `Agent`. Run a simple command-line interaction loop demonstrating storing and retrieving information within a specific `entity.Context`.
    *   **Manual Test:** Execute the example, configuring it for both SQLite and BoltDB separately. Verify basic storage, retrieval, and entity isolation work as expected. Check for expected log output from Lua hooks if implemented.

13. **Phase 1 Review & Refactor:**
    *   Conduct a thorough code review of all implemented components.
    *   Analyze test coverage reports and add tests for any identified gaps.
    *   Refactor code for clarity, efficiency, and adherence to best practices based on reviews and analysis.
    *   Verify all tests pass reliably in the CI pipeline.
    *   Update `README.md` and add godoc comments explaining Phase 1 features, configuration (SQLite/BoltDB), basic Lua usage, and how to run the example application.

**Testing Focus (Phase 1):** Comprehensive unit testing for all components. Integration tests validating **SQLite** and **BoltDB** adapters against actual file-based databases, confirming storage, retrieval, and multi-entity isolation. Tests verifying basic Lua script execution and hook invocation.

**Outcome / Deliverable (Phase 1):** A functional core CogMem library with established testing practices. It supports persistent, structured memory storage using embedded/file-based **SQLite and BoltDB**, with enforced multi-entity isolation. Basic Lua scripting is integrated for customizing simple MMU logic points. A minimal agent control loop is operational.

---

## 4. Phase 2: Vector LTM (Milvus) & Basic RAG / Reflection

**Goal:** Implement vector-based long-term memory using **Milvus** for semantic search (RAG), introduce the foundational reflection module, and refine the agent loop to support these capabilities.

**Detailed Steps:**

1.  **Testing Infrastructure:**
    *   **Setup Milvus:** Define a Milvus service within `test/docker-compose.test.yml`.
    *   **Update Helpers:** Enhance test helpers in `test/testutil/` to manage the Milvus Docker container lifecycle (start/stop/connect) for integration tests.

2.  **Milvus Vector LTM Adapter (`pkg/mem/ltm/adapters/vector/milvus/`):**
    *   **Implement (TDD):**
        *   Write integration tests (`milvus_test.go`) for the `LTMStore` interface targeting Milvus. Tests must cover storing records with vector embeddings, performing semantic retrieval using query vectors, ensuring correct `EntityID` filtering (using Milvus partitions or expressions), and handling metadata filtering alongside vector search.
        *   Implement the adapter (`milvus.go`) using the official Milvus Go SDK. Ensure the implementation handles connection management, collection setup, data insertion/search operations, and filtering logic to pass the integration tests.

3.  **MMU Enhancements (Embeddings & Vector Retrieval):**
    *   **Implement (TDD):**
        *   Write unit and integration tests for MMU modifications. Test that `encode_to_ltm` correctly calls the `ReasoningEngine` (mocked or real) to obtain embeddings before storing data using the `LTMStore` (specifically testing with the Milvus adapter in integration). Test that `retrieve_from_ltm` invokes the semantic search capability of the vector LTM adapter when the appropriate strategy is selected. Test basic WM overflow heuristics. Test any new Lua hooks related to vector processing (e.g., result ranking).
        *   Enhance the `pkg/mmu` implementation to include embedding generation logic, add the semantic retrieval strategy, implement basic WM overflow management, and integrate any new Lua hooks.

4.  **Reasoning Engine (Embeddings):**
    *   **Implement (TDD):**
        *   Write tests for a real `ReasoningEngine` adapter (e.g., `openai/openai.go`) focusing specifically on its ability to generate embeddings (mock underlying HTTP calls).
        *   Implement or verify the chosen adapter (`openai/openai.go` or similar) effectively generates embeddings required by the MMU.

5.  **Reflection Module Basics (`pkg/reflection/`):**
    *   **Implement (TDD):**
        *   Write unit tests (`reflection_test.go`) for the `ReflectionModule` interface, mocking `MMU` and `ReasoningEngine`. Test basic analysis triggering, the flow of calling the reasoning engine for analysis, parsing insights, triggering `MMU.ConsolidateLTM` (mocked) with the correct context/insight, and invocation of basic Lua hooks.
        *   Implement the basic reflection module structure (`reflection.go`, `analyzer.go`, `insight.go`). Define the `ConsolidateLTM` method signature in the MMU interface and provide a minimal MMU implementation for it (e.g., logging or storing the insight via `encode_to_ltm`). Integrate basic Lua hooks (`lua_hooks.go`).

6.  **Agent Loop Integration (RAG & Reflection):**
    *   **Implement (TDD):**
        *   Update `Agent` unit tests (`agent_test.go`) to verify that retrieved vector results (RAG context) are correctly incorporated before calling the reasoning engine and that the reflection module is triggered appropriately (e.g., after an action).
        *   Refine the `pkg/agent` control loop implementation to handle the RAG flow and integrate reflection triggering.

7.  **Examples & Configuration:**
    *   **Update Example:** Modify `cmd/example-agent/main.go` to demonstrate the RAG workflow using the configured Milvus LTM.
    *   **Update Config:** Add Milvus connection parameters and basic reflection settings to `configs/config.example.yaml`.

8.  **(Optional Stretch) Other Vector Adapters:** If development time allows, implement adapters for other vector stores like Postgres/pgvector or Weaviate, following the same TDD process (define tests, implement adapter, ensure tests pass).

9.  **Phase 2 Review & Refactor:**
    *   Conduct code reviews for Phase 2 additions.
    *   Check test coverage, particularly for Milvus interactions and reflection logic.
    *   Refactor implementations for clarity and efficiency.
    *   Ensure all tests, including Milvus integration tests, pass reliably in CI.
    *   Update documentation (`README.md`, godoc) to cover Milvus configuration, RAG usage, and the basic reflection mechanism.

**Testing Focus (Phase 2):** Integration tests for the **Milvus** adapter are paramount. Unit and integration tests verifying the MMU's handling of embeddings and vector retrieval. Unit tests covering the basic flow of the reflection module.

**Outcome / Deliverable (Phase 2):** The library now supports semantic search capabilities (RAG) via **Milvus** vector LTM. A foundational reflection mechanism is implemented for basic analysis and insight storage. The reasoning engine is capable of generating embeddings.

---

## 5. Phase 3: Graph LTM, Collaboration & Advanced Features

**Goal:** Implement graph-based LTM for relationship-aware memory (prioritizing **Postgres/Apache Age** if feasible, then others), enable multi-agent collaboration via shared memory with concurrency control, mature the reflection loop, enhance Lua scripting capabilities, and finalize documentation.

**Detailed Steps:**

1.  **Testing Infrastructure:**
    *   **Setup Graph DBs:** Add services for required graph databases (e.g., PostgreSQL with Apache Age enabled, potentially Neo4j) to `test/docker-compose.test.yml`.
    *   **Update Helpers:** Enhance `test/testutil/` helpers to manage the lifecycle and provide connections for the selected graph databases during integration tests.

2.  **Graph LTM Adapters (`pkg/mem/ltm/adapters/graph/`):**
    *   **Implement (TDD - Prioritize Postgres/Apache Age):**
        *   Write integration tests targeting the chosen graph database(s). Tests must verify storing nodes and relationships, retrieval via graph traversal, enforcement of `EntityID` scoping (e.g., using graph partitioning or labeling), and potentially temporal queries if the adapter design supports them.
        *   Implement the graph adapter(s) (`postgres/postgres_apacheage.go`, `neo4j/neo4j.go`, etc.). Ensure the implementation correctly interacts with the graph database to pass the integration tests.

3.  **Multi-Agent Shared Memory & Concurrency:**
    *   **Implement (TDD):**
        *   Write specific integration tests verifying that memory marked `shared_within_entity` can be correctly written and read by multiple agents (simulated via goroutines) operating under the same `EntityID`, using adapters implemented so far (SQLite, BoltDB, Milvus, selected Graph DB). Re-verify isolation for different `EntityID`s.
        *   Write integration tests simulating concurrent writes to the same shared memory item, verifying that data corruption is prevented.
        *   Implement full support for the `shared_within_entity` access level within the `LTMStore` interface and relevant adapter implementations.
        *   Implement concurrency control mechanisms (e.g., optimistic locking using version fields/properties, or leveraging database transaction isolation levels where applicable) within the adapters and verify through concurrency tests.

4.  **MMU Advanced Features:**
    *   **Implement (TDD):**
        *   Write unit and integration tests for new MMU retrieval strategies: graph traversal (using mock/real graph LTM) and iterative retrieval (mocking LTM responses over multiple calls).
        *   Write unit and integration tests for enhanced `ConsolidateLTM` logic that performs meaningful updates based on reflection insights (e.g., merging graph nodes, updating structured facts). Test Lua hooks for defining custom consolidation rules.
        *   Implement the advanced retrieval strategies and the sophisticated `ConsolidateLTM` logic within `pkg/mmu`.

5.  **Reflection Enhancements:**
    *   **Implement (TDD):**
        *   Write unit and integration tests for advanced reflection capabilities: more complex triggering mechanisms (e.g., surprise detection), analysis leveraging graph context (if available), reflection insights directly driving LTM modifications via `MMU.ConsolidateLTM`, and Lua-driven analysis/insight generation logic.
        *   Enhance the `pkg/reflection` implementation with these advanced features, ensuring tests pass.

6.  **Lua Scripting Enhancements:**
    *   **Implement (TDD):**
        *   Write unit tests for any new Go functions exposed to the Lua API needed for advanced MMU/Reflection hooks. Test more complex interactions between Go and Lua scripts.
        *   Expand the Go<->Lua API (`pkg/scripting/api.go`) as required and update the integration points in MMU/Reflection.

7.  **(Optional Stretch) Other Adapters:** If desired and time permits, implement any remaining adapters from the project structure (e.g., Postgres SQL, Redis KV, Weaviate) following the TDD process.

8.  **Documentation & Polish:**
    *   **Write Comprehensive Docs:** Create final user documentation (README, godoc, potentially external site) covering the full architecture, all features, configuration options for every supported adapter, detailed Lua scripting guide, multi-tenancy/collaboration specifics, and practical examples.
    *   **Refine:** Polish the public API, improve error handling and logging across the library based on accumulated experience.
    *   **Create Examples:** Develop more diverse and complex examples in `cmd/` showcasing various features (RAG, reflection, graph memory, scripting).

9.  **(Optional) Benchmarking:**
    *   Develop a benchmark suite (`test/benchmark/`) to measure the performance of key operations (e.g., retrieval latency/throughput for different LTM types, reflection cycle time). Execute benchmarks and document results.

10. **Phase 3 Review & Refactor:**
    *   Conduct a final, comprehensive code review of the entire library.
    *   Perform final test coverage analysis and ensure high coverage.
    *   Complete final refactoring for consistency, performance, and maintainability.
    *   Ensure all tests pass reliably across all supported configurations in CI.
    *   Finalize all documentation content.

**Testing Focus (Phase 3):** Integration tests for graph database adapters. Critical integration tests verifying shared memory access and concurrency control mechanisms. End-to-end tests demonstrating advanced reflection loops that modify LTM state. Tests covering complex Lua script interactions and the expanded Go<->Lua API.

**Outcome / Deliverable (Phase 3):** A feature-rich CogMem library supporting diverse memory backends (including graph databases), robust multi-entity isolation, secure multi-agent collaboration via shared memory, advanced adaptive capabilities through a mature reflection loop, and extensive customization via Lua scripting. The library is thoroughly tested, well-documented, and suitable for building sophisticated LLM agents.