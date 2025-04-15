## implementation-plan.md

# CogMem Golang Library: Implementation Plan (Test-First)

**Version:** 4.0 (Hybrid LTM Focus)
**Date:** 2023-10-27 (Placeholder)
**Corresponding Project Structure Version:** 1.0 rev 2

## 1. Introduction

This document outlines a detailed 3-phase implementation plan for the CogMem Golang Library, designed to align with the `project-structure.md` (v1.0 rev 2). The plan adopts a **Test-First Development (TFD)** approach and explicitly builds towards the architectural goal of supporting **hybrid Long-Term Memory (LTM)** configurations, where an agent can leverage multiple storage backends (e.g., SQL, KV, Vector, Graph) simultaneously.

*   **Phase 1:** Focuses on establishing the foundational library structure, core interfaces, testing infrastructure, implementing basic **single-instance** LTM capabilities using **SQLite** and **BoltDB**, integrating Lua scripting, and ensuring robust multi-entity context handling. This lays the groundwork for individual storage types.
*   **Phase 2:** Centers on adding **Vector LTM** capabilities (initially **Chromem-go**) for Retrieval-Augmented Generation (RAG), implementing embedding generation, and introducing the basic reflection module. The MMU will learn to handle vector-specific operations, still likely operating primarily against one *type* of backend at a time or with very basic routing.
*   **Phase 3:** Concentrates on adding **Graph LTM** capabilities (prioritizing **Postgres/Apache Age**), **implementing the core MMU orchestration logic** to manage and query *multiple configured LTM backends concurrently* for true hybrid storage, enabling multi-agent collaboration via shared memory, maturing the reflection loop, and finalizing documentation.

## 2. Overall Approach: Test-First Development (TFD)

For each significant piece of functionality:
1.  **Write Tests:** Define unit or integration tests that specify the desired behavior and cover primary use cases and edge conditions *before* writing implementation code.
2.  **Implement:** Write the minimum code necessary to make the tests pass.
3.  **Refactor:** Improve the code's structure, clarity, and efficiency while ensuring tests continue to pass.

---

## 3. Phase 1: Foundation, SQLite/BoltDB LTM Capability & Lua Scripting

**Goal:** Establish the core library structure, interfaces, and testing infrastructure. Implement the *capability* to use **SQLite** (via `sqlc`) or **BoltDB** KV store as a *single, configured* LTM backend, integrate Lua scripting (`gopher-lua`) with basic hooks, and ensure robust multi-entity context handling. This phase delivers adapters for basic structured/KV storage and the core application framework, laying the groundwork for future hybrid LTM orchestration.

**Detailed Steps:**

1.  **Project Initialization & Setup:**
    *   Initialize Git repository.
    *   Initialize Go module.
    *   Create the standard top-level directory structure (`pkg/`, `internal/`, `cmd/`, `configs/`, `scripts/`, `migrations/`, `test/`) and required subdirectories within `pkg/` based on `project-structure.md` (including adapters for mock, sqlite, boltdb).
    *   Add `.gitignore`, `README.md`, `LICENSE`.
    *   Setup basic CI pipeline (e.g., GitHub Actions) for builds and tests.

2.  **Core Types, Interfaces & Errors:**
    *   **Implement `pkg/entity` (TDD):** Test and implement `EntityID`, `AccessLevel`, `entity.Context` struct, and context helper functions.
    *   **Implement `pkg/errors` (TDD):** Test and implement initial custom error types/variables.
    *   **Implement `slog` logging (TDD):** Test and implement slog logging, info, warn debug etc.
    *   **Define Core Interfaces:**
        *   Define `LTMStore` interface (`pkg/mem/ltm/ltm.go`) supporting CRUD operations, accepting `context.Context`. Define `MemoryRecord` struct here.
        *   Define `MMU` interface (`pkg/mmu/mmu.go`) with initial methods for `EncodeToLTM`, `RetrieveFromLTM`, and a placeholder `ConsolidateLTM`.
        *   Define `ScriptingEngine` interface (`pkg/scripting/engine.go`) for loading/executing Lua.
        *   Define `ReasoningEngine` interface (`pkg/reasoning/engine.go`) for processing/embeddings.

3.  **Configuration (`pkg/config`):**
    *   **Implement (TDD):** Test and implement configuration loading (`viper`) and the `Config` struct, including fields to specify *which* LTM backend type to use (`LTM.Type: "sqlite"` or `"boltdb"`) and paths/connection details (`LTM.SQLite.Path`, `LTM.BoltDB.Path`), plus Lua script paths.
    *   **Create Example:** Add `configs/config.example.yaml`.

4.  **Testing Infrastructure (`test/`):**
    *   **Implement Mocking:** Add `testify/mock` dependency.
    *   **Implement Test Helpers:** Create utilities (`test/testutil/`) to manage temporary SQLite and BoltDB database files for integration testing.

5.  **LTM - Mock Adapter (`pkg/mem/ltm/adapters/mock/`):**
    *   **Implement (TDD):** Test and implement the mock `LTMStore` using in-memory maps, covering CRUD and basic entity/access filtering.

6.  **LTM - SQLite Adapter Capability (`pkg/mem/ltm/adapters/sqlstore/sqlite/`):**
    *   **Implement (TDD):**
        *   Write integration tests (`sqlite3_test.go`) using temporary DB files, verifying `LTMStore` interface compliance (CRUD, `EntityID` isolation, `AccessLevel` filtering, type mapping).
        *   Configure `sqlc` for SQLite; write `query.sql`; generate code. Implement the adapter (`sqlite3.go`) using `sqlc` and `database/sql`.

7.  **LTM - BoltDB Adapter Capability (`pkg/mem/ltm/adapters/kv/boltdb/`):**
    *   **Implement (TDD):**
        *   Write integration tests (`boltdb_test.go`) using temporary DB files. Verify `LTMStore` compliance using BoltDB buckets for entity isolation and marshalling for storage. Test CRUD and filtering.
        *   Implement the adapter (`boltdb.go`) using `go.etcd.io/bbolt`.

8.  **Lua Scripting Engine (`pkg/scripting/`):**
    *   **Implement (TDD):** Test and implement the `ScriptingEngine` wrapper for `gopher-lua`, including secure sandboxing and a basic Go<->Lua API. Test loading scripts and executing simple functions.
    *   **Create Examples:** Add basic placeholder Lua scripts in `scripts/`.

9.  **Basic Memory Management Unit (MMU) (`pkg/mmu/`):**
    *   **Implement (TDD):**
        *   Write unit tests (`mmu_test.go`) mocking `LTMStore` and `ScriptingEngine`. Test that the MMU correctly uses the *single configured* `LTMStore` (passed during initialization) for `EncodeToLTM` and `RetrieveFromLTM`. Test basic Lua hook invocation (`before_retrieve`, `after_retrieve`).
        *   Implement the basic MMU (`mmu.go`, `lua_hooks.go`). It should accept *one* `LTMStore` instance during setup.

10. **Reasoning Engine - Mock (`pkg/reasoning/`):**
    *   **Implement (TDD):** Test and implement the mock `ReasoningEngine` adapter (`adapters/mock/`).

11. **CogMemClient Facade & Loop (`pkg/cogmem/`):**
    *   **Implement (TDD):** Test and implement the `CogMemClient` facade and basic controller loop, mocking dependencies. Ensure it initializes and uses the MMU (which in turn uses the single configured LTM). Test `entity.Context` propagation.

12. **Example Application (`cmd/example-client/`):**
    *   **Implement & Test:** Create `cmd/example-client/main.go`. It should load config, instantiate the *configured* LTM adapter (SQLite *or* BoltDB), scripting engine, mock reasoning engine, and the CogMemClient. Run a simple CLI loop demonstrating storing/retrieving within an entity context using the selected backend. Manually test switching the backend via configuration.

13. **Phase 1 Review & Refactor:**
    *   Conduct code reviews, check test coverage, refactor for clarity and robustness. Ensure all tests pass in CI.
    *   Update documentation (`README.md`, godoc) clarifying Phase 1 setup (choosing SQLite *or* BoltDB via config), Lua hooks, and example usage. State that hybrid LTM is the goal, but this phase focuses on single-backend operation.

**Testing Focus (Phase 1):** Unit tests for core logic. Integration tests validating **SQLite** and **BoltDB** adapters individually. Tests verifying basic Lua integration.

**Outcome / Deliverable (Phase 1):** A functional core library with established testing practices. Supports using **either SQLite or BoltDB as a single, configured persistent LTM backend**, with enforced multi-entity isolation. Basic Lua scripting is integrated. A minimal agent loop operates using the configured LTMStore. The foundation for adding more LTM types and future hybrid orchestration is laid.

---

## 4. Phase 2: Vector LTM (Chromem-go) Capability & Basic Reflection

**Goal:** Add the *capability* to use **chromem-go** as a Vector LTM backend, implement necessary MMU enhancements for handling embeddings and semantic search *when chromem-go is configured*, introduce the basic reflection module, and implement a real Reasoning Engine adapter for embeddings.

**Detailed Steps:**

1.  **Testing Infrastructure:**
    *   **Setup Chromem-go:** Add `chromem-go` to vector store .
    *   **Update Helpers:** Enhance `test/testutil/` to manage the Chromem-go container for integration tests.

2.  **Chromem-go Vector LTM Adapter Capability (`pkg/mem/ltm/adapters/vector/chromem_go/`):**
    *   **Implement (TDD):**
        *   Write integration tests (`chromem_go/chromem_go_test.go`) verifying `LTMStore` compliance for chromem-go. Test storing records with embeddings, semantic retrieval, `EntityID` filtering (via partitions/expressions), and metadata filtering.
        *   Implement the Chromem-go adapter (`chromem_go/chromem_go.go`) using the chromem-go Go SDK, passing integration tests.

3.  **MMU Enhancements (Vector Handling):**
    *   **Implement (TDD):**
        *   Write unit/integration tests for MMU modifications. Test embedding generation: verify `EncodeToLTM` calls `ReasoningEngine.GenerateEmbeddings` *if* the configured `LTMStore` is identified as a vector store (or has vector capabilities). Test semantic retrieval: verify `RetrieveFromLTM` uses a "semantic" strategy invoking vector search on the LTMStore *if* appropriate options are passed and the store supports it. Test basic WM overflow logic. Test any new vector-related Lua hooks.
        *   Enhance `pkg/mmu`: Add logic to call `GenerateEmbeddings`. Add a semantic retrieval strategy. Implement basic WM overflow. Integrate relevant Lua hooks. *Note: The MMU likely still operates primarily against the single LTMStore instance it was initialized with, but now understands vector operations if that instance supports them.*

4.  **Reasoning Engine (Embeddings):**
    *   **Implement (TDD):**
        *   Test and implement a real `ReasoningEngine` adapter (e.g., `openai/openai.go`) capable of generating embeddings (mocking HTTP).

5.  **Reflection Module Basics (`pkg/reflection/`):**
    *   **Implement (TDD):**
        *   Test and implement the basic `ReflectionModule` structure and logic (`reflection_test.go`, `reflection.go`, `analyzer.go`, `insight.go`), mocking dependencies (`MMU`, `ReasoningEngine`). Test basic triggering, analysis flow, insight generation, and the call to `MMU.ConsolidateLTM` (still likely a basic implementation in MMU). Test basic reflection-related Lua hooks.

6.  **CogMemClient Loop Integration (RAG & Reflection):**
    *   **Implement (TDD):** Update `CogMemClient` tests and implementation to integrate RAG results (when semantic retrieval is used) into the reasoning context and add basic reflection module triggering.

7.  **Examples & Configuration:**
    *   **Update Example:** Enhance `cmd/example-client` to demonstrate RAG workflow *when configured to use Chromem-go*.
    *   **Update Config:** Add Chromem-go connection parameters to `configs/config.example.yaml` and the option to select `"Chromem-go"` as the `LTM.Type`. Add basic reflection config options.

8.  **(Optional Stretch) Other Vector Adapters:** Implement adapters for Postgres/pgvector or Weaviate if time allows, following TDD.

9.  **Phase 2 Review & Refactor:**
    *   Conduct code reviews, check test coverage (especially Chromem-go and MMU vector logic). Refactor. Ensure CI passes.
    *   Update documentation explaining how to configure and use Chromem-go for vector storage/RAG, and the basic reflection mechanism. Reiterate that only one LTM backend is active at runtime in this phase.

**Testing Focus (Phase 2):** Integration tests for the **Chromem-go** adapter. Unit/Integration tests for MMU's handling of embeddings and semantic retrieval strategy. Unit tests for the reflection module's core flow.

**Outcome / Deliverable (Phase 2):** The library now has the *capability* to use **Chromem-go** for vector storage and semantic search (RAG) when configured. A basic reflection mechanism is implemented. The MMU understands vector operations but likely still interacts with only the single configured LTM backend.

---

## 5. Phase 3: Graph LTM Capability, Hybrid Orchestration & Collaboration

**Goal:** Add the *capability* to use **Graph LTM** backends (prioritizing **cayley**), **implement the core MMU orchestration logic** enabling true **hybrid LTM** operation (using multiple backends concurrently), enable multi-agent collaboration via shared memory with concurrency control, mature the reflection loop, and finalize documentation.

**Detailed Steps:**

1.  **Testing Infrastructure:**
    *   **Setup Graph DBs:** Add services for required graph databases (e.g., cayley Postgres+Age, Neo4j) 
    *   **Update Helpers:** Enhance `test/testutil/` for graph database management.

2.  **Graph LTM Adapter Capability (`pkg/mem/ltm/adapters/graph/`):**
    *   **Implement (TDD - Prioritize cayley):**
        *   Write integration tests verifying `LTMStore` compliance for graph databases. Test node/relationship storage, graph traversal retrieval, `EntityID` scoping, and potentially temporal queries.
        *   Implement the graph adapter(s) (`cayley/cayley.go`, etc.).

3.  **MMU Hybrid LTM Orchestration:**
    *   **Design:** Define strategies within the MMU for handling *multiple, concurrently configured* LTM backends:
        *   How is `EncodeToLTM` routed? (Based on `MemoryRecord` type/metadata? Store everywhere? Configurable rules?)
        *   How is `RetrieveFromLTM` routed? (Based on query type - semantic to Vector, structured ID lookup to SQL/KV, relationship query to Graph? Query multiple stores?)
        *   How are results aggregated if a query hits multiple stores?
    *   **Implement (TDD):**
        *   Write unit tests for the MMU's new orchestration logic, mocking *multiple different types* of `LTMStore` interfaces (e.g., one mock SQL store, one mock Vector store). Test routing logic for `EncodeToLTM` and `RetrieveFromLTM` based on designed strategies. Test result aggregation.
        *   Refactor the MMU (`mmu.go`) to accept a *map or slice* of configured `LTMStore` instances during initialization. Implement the designed routing and aggregation logic. Ensure tests pass.
    *   **Update Config:** Modify `pkg/config` to allow configuring *multiple* LTM backends simultaneously (e.g., a list/map of LTM configurations instead of a single `LTM.Type`). Update `config.example.yaml`.

4.  **Multi-Agent Shared Memory & Concurrency:**
    *   **Implement (TDD):**
        *   Write integration tests verifying `shared_within_entity` access level works correctly across applicable adapter types (SQL, KV, Graph). Test concurrent writes from simulated agents to shared memory, ensuring data integrity via locking/conflict resolution.
        *   Implement full support for `shared_within_entity` in the `LTMStore` interface and adapters. Implement concurrency control mechanisms (e.g., optimistic locking) within adapters.

5.  **Reflection Enhancements:**
    *   **Implement (TDD):** Test and implement advanced reflection capabilities: sophisticated triggers, analysis possibly leveraging graph context, insights driving meaningful LTM updates via `MMU.ConsolidateLTM` (which now might affect multiple LTM stores), and enhanced Lua scripting for reflection logic.

6.  **Lua Scripting Enhancements:**
    *   **Implement (TDD):** Test and implement any new Go<->Lua API functions needed for hybrid LTM orchestration or advanced reflection. Test more complex script interactions.

7.  **(Optional Stretch) Other Adapters:** Implement any remaining desired adapters (Postgres SQL, Redis KV, Weaviate, etc.).

8.  **Documentation & Polish:**
    *   **Write Comprehensive Docs:** Create final documentation covering the full architecture, *how to configure and use hybrid LTM setups*, all features, Lua scripting, multi-agent collaboration, and diverse examples.
    *   **Refine:** Polish APIs, error handling, logging.
    *   **Create Examples:** Develop examples demonstrating hybrid LTM usage, shared memory, and advanced reflection.

9.  **(Optional) Benchmarking:** Develop and run benchmarks, potentially comparing single vs. hybrid LTM performance for different query types.

10. **Phase 3 Review & Refactor:** Final code reviews, test coverage checks, refactoring. Ensure all tests, including hybrid LTM orchestration and concurrency tests, pass reliably in CI. Finalize all documentation.

**Testing Focus (Phase 3):** Integration tests for graph adapters. **Crucial unit tests for MMU multi-backend orchestration logic.** Integration tests verifying shared memory access and concurrency control. End-to-end tests for advanced reflection loops potentially modifying multiple LTM stores.

**Outcome / Deliverable (Phase 3):** A feature-rich CogMem library supporting **hybrid LTM configurations** (allowing concurrent use of configured SQL, KV, Vector, Graph backends), robust multi-entity isolation, secure multi-agent collaboration via shared memory, advanced adaptation via reflection, and extensive customization via Lua scripting. The MMU intelligently routes operations across configured LTM stores. The library is thoroughly tested and documented.
