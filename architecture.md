# Software Architecture Document: CogMem Library (Golang)

**Version:** 1.0
**Date:** 2023-10-27 (Placeholder)
**Authors:** AI Language Model based on provided context

## 1. Introduction

This document outlines the software architecture for the **CogMem Library**, a Golang framework designed to implement the CogMem cognitive architecture as described in the whitepaper and Product Requirements Document (PRD). The library aims to provide developers with a modular, performant, multi-tenant, and scriptable foundation for building LLM agents with advanced memory and adaptation capabilities. The primary implementation language is Go, leveraging its concurrency features and strong typing, with embedded Lua (`gopher-lua`) for flexible customization.

**Scope:** This document focuses on the internal structure, components, interactions, patterns, and implementation strategies for the Golang library itself. It does not detail specific application implementations built *using* the library.

## 2. Guiding Principles & Goals

The architecture adheres to the following principles:

*   **Modularity:** Components are distinct Go packages with clear responsibilities and interfaces (inspired by Cognitive Architectures).
*   **Layered Architecture (Onion/Clean):** Dependencies flow inwards (Infrastructure/Adapters -> Application Logic -> Domain Core). Strict separation of concerns.
*   **Dependency Inversion:** Inner layers define interfaces (ports); outer layers implement them (adapters). Promotes testability and flexibility.
*   **Multi-Tenancy Native:** Entity context (`entity_id`, access levels) is a first-class citizen, enforced throughout relevant modules (MMU, LTM).
*   **Extensibility:** Designed for extension through interface implementations (e.g., LTM backends) and Lua scripting.
*   **Performance & Concurrency:** Leverage Go's capabilities for efficient execution, especially around I/O.
*   **Testability:** Emphasis on unit and integration testing facilitated by modularity and interfaces.
*   **Scriptability:** Provide controlled flexibility via embedded Lua (`gopher-lua`) at key points.
*   **Security:** Prioritize secure design, especially regarding Lua sandboxing and data isolation.

The architectural goals directly reflect the PRD goals: deliver a robust, flexible, multi-tenant Go library implementing the CogMem concepts.

## 3. High-Level Architecture Diagram (Conceptual)

*(Mermaid Diagram Description - visualize this)*

```mermaid
graph TD
    subgraph Application Layer / Use Cases
        CogMemClient(CogMem Client Facade / Executive Controller)
        MMU(Memory Management Unit)
        Reflection(Reflection Module)
        Reasoning(Reasoning Engine Interface)
        Perception(Perception Interface)
        Action(Action Interface)
    end

    subgraph Domain Layer
        CoreTypes(Core Domain Types: EntityID, AccessLevel, MemoryRecord Base)
    end

    subgraph Infrastructure Layer / Adapters
        LTM_Interface[LTM Store Interface]
        subgraph LTM Backends
            VectorDB(Vector DB Adapter)
            GraphDB(Graph DB Adapter)
            KVStore(KV Store Adapter)
            MockDB(Mock DB Adapter)
        end
        LLM_Client(LLM Client Adapter)
        LuaEngine(Lua Scripting Engine)
        ExternalTools(External Tool/API Adapters)
    end

    subgraph External Systems
        UserApp(User Application / CogMem Client Host)
        LLM_Service(LLM Service API)
        VectorDB_Service(Vector Database)
        GraphDB_Service(Graph Database)
        KVStore_Service(Key/Value Store)
        LuaScripts(Lua Scripts .lua)
    end

    %% Interactions
    UserApp --> CogMemClient;
    CogMemClient --> Perception;
    CogMemClient --> MMU;
    CogMemClient --> Reasoning;
    CogMemClient --> Reflection;
    CogMemClient --> Action;

    MMU --> LTM_Interface;
    MMU --> LuaEngine;
    MMU --> Reasoning; %% For summarization/structuring before LTM write

    Reflection --> MMU; %% To consolidate LTM
    Reflection --> LuaEngine;
    Reflection --> Reasoning; %% For analysis

    Reasoning --> LLM_Client;
    Reasoning --> MMU; %% Request LTM retrieval

    Action --> ExternalTools;

    LTM_Interface --> CoreTypes;
    MMU --> CoreTypes;
    CogMemClient --> CoreTypes;

    VectorDB -- Implements --> LTM_Interface;
    GraphDB -- Implements --> LTM_Interface;
    KVStore -- Implements --> LTM_Interface;
    MockDB -- Implements --> LTM_Interface;

    VectorDB --> VectorDB_Service;
    GraphDB --> GraphDB_Service;
    KVStore --> KVStore_Service;

    LuaEngine --> LuaScripts;

```

**Key Flows:**
1.  **Request Handling:** User Application interacts with the `CogMemClient Facade`. The Facade (acting as Executive Controller) receives the input and `entity_context`.
2.  **Memory Operation:** The Controller invokes the `MMU`. The MMU, using the `entity_context`, interacts with the configured `LTM Store Interface` implementation (e.g., `VectorDB Adapter`) to retrieve or store data. LTM adapters handle communication with external databases, ensuring filtering by `entity_id` and `access_level`. Lua scripts might be invoked by the MMU for custom logic.
3.  **Reasoning:** The Controller provides context (including retrieved LTM data) to the `Reasoning Engine Interface`. Its implementation (e.g., `LLM Client Adapter`) interacts with an external LLM.
4.  **Reflection:** Periodically or triggered, the `Reflection Module` analyzes history (potentially via MMU reads), uses the `Reasoning Engine` for analysis, potentially invokes Lua scripts, and generates insights that trigger `MMU` consolidation actions.

## 4. Component Breakdown (Go Packages)

The library will be organized into packages within the `pkg/` directory (intended for public use) and potentially `internal/` (for implementation details).

*   **`pkg/cogmem` **:
    *   **Responsibilities:** Provides the main entry point/facade (`CogMemClient` struct) for users of the library. Implements the Executive Controller logic. Manages the overall agent lifecycle, orchestrates module interactions, handles `entity_context` propagation.
    *   **Key Interfaces:** Defines the primary `CogMemClient` interface/struct.
    *   **Dependencies:** `config`, `entity`, `mmu`, `reflection`, `reasoning`, `perception`, `action`.

*   **`pkg/entity`**:
    *   **Responsibilities:** Defines core types related to multi-tenancy like `EntityID`, `AccessLevel`. May contain helper functions for context management.
    *   **Key Interfaces/Types:** `EntityID`, `AccessLevel`.
    *   **Dependencies:** None (or only standard Go libraries).

*   **`pkg/mem`**:
    *   **Responsibilities:** Parent package for memory-related components.
    *   **Sub-packages:**
        *   **`pkg/mem/ltm`**:
            *   **Responsibilities:** Defines the core LTM abstractions and provides adapters.
            *   **Key Interfaces/Types:** `LTMStore` interface (defining CRUD operations like `Store`, `Retrieve`, `Update`, `Delete`, `Consolidate`, all requiring `entity.Context` and handling access levels), `MemoryRecord` struct (with fields for `EntityID`, `AccessLevel`, `Content`, `Metadata`, `Embedding`, etc.).
            *   **Dependencies:** `entity`, `errors`, standard Go libraries.
            *   **Sub-packages (`pkg/mem/ltm/adapters/`):** Contains concrete implementations of `LTMStore` for different backends (e.g., `vector/weaviate`, `graph/zep`, `kv/redis`, `mock`). These adapters handle DB-specific logic and API calls. Depend on `pkg/mem/ltm` and specific DB client libraries.
        *   **`pkg/mem/wm`**:
            *   **Responsibilities:** Defines interfaces and logic related to managing the conceptual Working Memory (LLM context). This might involve helper functions for formatting context strings, tracking token counts, but the actual context lives outside the library (passed to the LLM).
            *   **Key Interfaces/Types:** `WorkingMemoryManager` interface (optional, could be handled directly by MMU/Agent).
            *   **Dependencies:** `entity`, `mmu` (potentially).

*   **`pkg/mmu`**:
    *   **Responsibilities:** Implements the Memory Management Unit logic. Orchestrates reads/writes between WM conceptual space and LTM Store. Implements retrieval strategies, consolidation logic, overflow management. Integrates with Lua scripting for customization.
    *   **Key Interfaces/Types:** `MMU` interface/struct. Defines logic for `encode_to_ltm`, `retrieve_from_ltm`, etc.
    *   **Dependencies:** `entity`, `mem/ltm`, `mem/wm`, `scripting`, `reasoning` (for summarization/structuring), `errors`, `config`.

*   **`pkg/reflection`**:
    *   **Responsibilities:** Implements the self-reflection loop. Analyzes CogMemClient history/performance within an `entity_context`. Generates insights. Triggers LTM consolidation via MMU. Integrates with Lua scripting.
    *   **Key Interfaces/Types:** `ReflectionModule` interface/struct.
    *   **Dependencies:** `entity`, `mmu`, `reasoning`, `scripting`, `errors`, `config`.

*   **`pkg/reasoning`**:
    *   **Responsibilities:** Defines the interface for the reasoning engine and provides adapters.
    *   **Key Interfaces/Types:** `ReasoningEngine` interface (e.g., `Process(context, prompt) (result, error)`).
    *   **Dependencies:** `entity`, `errors`, `config`.
    *   **Sub-packages (`pkg/reasoning/adapters/`):** Concrete implementations, e.g., `openai`, `anthropic`, `local_llm`. Depend on specific LLM client libraries.

*   **`pkg/perception` / `pkg/action`**:
    *   **Responsibilities:** Define interfaces for how the agent perceives input and executes actions. Default implementations might be simple pass-throughs or basic text handling. Users typically provide their own adapters based on the application context.
    *   **Key Interfaces/Types:** `PerceptionModule` interface, `ActionModule` interface.
    *   **Dependencies:** `entity`, `errors`.

*   **`pkg/scripting`**:
    *   **Responsibilities:** Manages the embedded Lua (`gopher-lua`) interpreter. Provides functions for loading, executing, and sandboxing Lua scripts. Defines the API exposed from Go to Lua.
    *   **Key Interfaces/Types:** `Engine` interface/struct.
    *   **Dependencies:** `gopher-lua`, standard Go libraries.

*   **`pkg/config`**:
    *   **Responsibilities:** Defines structs for library configuration (LLM keys, DB connections, Lua paths, feature flags). Provides functions for loading configuration (e.g., from YAML/JSON using libraries like Viper).
    *   **Dependencies:** Standard Go libraries (e.g., `os`, `encoding`), potentially `spf13/viper`.

*   **`pkg/errors`**:
    *   **Responsibilities:** Defines custom error types used throughout the library for clearer error handling and propagation.
    *   **Dependencies:** Standard Go `errors`.

*   **`internal/`**: Contains implementation details not meant for public consumption, e.g., specific database connection pool logic, complex query builders, internal Lua helper functions. Code here is hidden behind `pkg/` interfaces.

## 5. Key Architectural Patterns & Concepts

*   **Layered Architecture:** As described above, separating Infrastructure (LTM Adapters, LLM Clients, Lua Engine), Application Logic (MMU, Reflection, Agent Controller), and Domain (Core Types like EntityID).
*   **Dependency Injection (DI):** The `CogMemClient` (or user application) will act as the Composition Root. It will be responsible for instantiating concrete implementations (like specific LTM adapters, LLM clients) and injecting them into the modules that depend on their interfaces (e.g., injecting an `LTMStore` implementation into the `MMU`). DI frameworks are not strictly required but can be used.
*   **Interface-Based Design (Ports & Adapters):** Core logic depends on interfaces (`LTMStore`, `ReasoningEngine`). Concrete implementations (`adapters`) are provided externally. This enhances testability (mocking) and replaceability.
*   **Context Propagation:** Go's `context.Context` can be used alongside custom `entity.Context` wrappers to propagate cancellation signals, deadlines, and crucially, the `EntityID` and potentially user-specific information throughout request handling.
*   **Strategy Pattern:** Retrieval methods within the MMU, consolidation rules, or reflection analysis logic can be implemented using the Strategy pattern, potentially allowing selection via configuration or even Lua scripts.

## 6. Data Flow Example (Simplified Retrieval Request)

1.  **UserApp -> `CogMemClient.Process(input, entityCtx)`:** Application calls the main CogMemClient facade.
2.  **CogMemClient:** Stores `entityCtx`. Potentially calls `Perception` module.
3.  **CogMemClient -> `MMU.Retrieve(query, entityCtx, options)`:** CogMemClient determines info is needed from LTM.
4.  **MMU:**
    *   Applies `entityCtx` filter parameters.
    *   Selects retrieval strategy based on `options` or config.
    *   *Optionally:* Invokes Lua script via `ScriptingEngine` for pre-query modification or filtering logic based on `entityCtx`.
    *   Calls `LTMStore.Retrieve(refinedQuery, entityCtxFilter, options)` on the injected LTM adapter.
5.  **LTM Adapter (e.g., VectorDB Adapter):**
    *   Constructs DB-specific query (e.g., Weaviate GraphQL query) including `WHERE` clauses for `entity_id` and potentially `access_level`.
    *   Calls external DB service.
    *   Receives results, parses them into `[]MemoryRecord`.
    *   Returns results to MMU.
6.  **MMU:**
    *   Receives `[]MemoryRecord`.
    *   *Optionally:* Invokes Lua script via `ScriptingEngine` for post-retrieval ranking or filtering based on `entityCtx`.
    *   Formats results for WM/Reasoning.
    *   Returns results to CogMemClient.
7.  **CogMemClient:** Adds retrieved info to the context provided to the `ReasoningEngine`.
8.  **CogMemClient -> `ReasoningEngine.Process(contextWithLTM, prompt)`:** Calls the LLM.
9.  ... subsequent steps (Action, Response).

## 7. Multi-Tenancy Implementation Details

*   **Entity Context:** A dedicated struct (e.g., `entity.Context`) containing `EntityID` (mandatory) and potentially `UserID`, `SessionID`, access roles/tokens will be passed down the call stack, possibly embedded within Go's standard `context.Context`.
*   **LTM Interface Enforcement:** The `LTMStore` interface methods *must* accept the `entity.Context` and implementations *must* use the `EntityID` to filter data (e.g., adding `WHERE entity_id = ?` to SQL, using tenant features in vector DBs, filtering graph traversals). Access level checks (`private_to_user` vs `shared_within_entity`) must also be implemented here, using `UserID` if available in the context.
*   **Shared Memory:** Data marked `shared_within_entity` is retrieved if the request's `entity_context.EntityID` matches the record's `EntityID`. Writes might require specific permissions checked via the context. Concurrency control (see below) is critical here.

## 8. Lua Integration Strategy

*   **`pkg/scripting`:** This package encapsulates `gopher-lua`.
*   **Engine Initialization:** A Lua state (`lua.LState`) pool might be used for concurrency. Each state needs careful initialization and sandboxing.
*   **Sandboxing:** Disable dangerous Lua modules (`os`, `io` limited access, potentially `debug`). Use `gopher-lua` options to restrict execution time and memory usage.
*   **Go<->Lua API:** Expose specific Go functions to Lua (e.g., `getCogMemContext()`, `logInfo()`, maybe restricted LTM read helpers). Data passed between Go and Lua needs careful type mapping (`glua.LValue`).
*   **Invocation:** Modules like MMU and Reflection will call the `scripting.Engine` to execute specific, pre-defined Lua functions loaded from user-configured scripts, passing necessary Go data (like current context, candidate records) converted to Lua types.

## 9. Concurrency Model

*   **Request-Level Concurrency:** The user application hosting the CogMem library is expected to handle incoming requests concurrently (e.g., one goroutine per HTTP request).
*   **Intra-Request Concurrency:** Within a single CogMemClient request processing flow:
    *   I/O operations (LTMStore calls, LLM API calls) should be performed asynchronously using goroutines and channels or async patterns to avoid blocking the main processing thread.
    *   Reflection might run in a separate background goroutine, potentially triggered by events/timers.
    *   **Shared LTM Access:** Concurrent reads to shared memory are generally safe if adapters are stateless. Concurrent writes *require* careful handling:
        *   **Optimistic Locking:** Use version numbers on records.
        *   **Pessimistic Locking:** Database-level row/document locking (can cause contention).
        *   **Conflict Resolution Logic:** Define how to handle write conflicts (e.g., last-write-wins, merge logic - potentially Lua-scriptable). The choice depends on the LTM backend and specific use case.

## 10. Configuration Management

*   **`pkg/config`:** Defines Go structs mirroring configuration file structure.
*   **Loading:** Use libraries like Viper to load from YAML/JSON/env vars.
*   **Structure:** Configuration should allow specifying:
    *   LLM provider, model name, API keys.
    *   LTM backend choice (e.g., "weaviate", "redis", "mock").
    *   Connection details for chosen LTM backend(s).
    *   Paths to Lua scripts for different modules/hooks.
    *   Reflection settings (trigger frequency, analysis depth).
    *   Retrieval strategy defaults and parameters.
    *   Logging levels.

## 11. Error Handling

*   **Custom Errors (`pkg/errors`):** Define specific error types (e.g., `ErrLTMUnavailable`, `ErrEntityNotFound`, `ErrAccessDenied`, `ErrLuaExecutionFailed`) for better programmatic handling.
*   **Propagation:** Errors should be propagated clearly up the call stack. Avoid swallowing errors.
*   **Context:** Errors should include relevant context where possible (e.g., which entity ID failed, which script caused an error).

## 12. Testing Strategy

*   **Unit Tests:** Each package should have extensive unit tests (`_test.go` files). Dependencies on interfaces should be mocked (using GoMock, testify/mock, or manual mocks). Test individual functions and logic branches. Lua interaction points should be tested by mocking the Lua engine or testing script logic separately if possible.
*   **Integration Tests:** Test interactions between modules (e.g., MMU interacting with a mock LTMStore). Test LTM adapters against real (but containerized/local) databases. Test Lua script execution with the actual `gopher-lua` engine but potentially mock external calls *from* Lua.
*   **Multi-Tenancy Tests:** Specific integration tests verifying data isolation between different entity IDs and correct functioning of shared memory access (including concurrent access simulation).
*   **End-to-End (E2E) Tests (in `cmd/` examples):** Simple example applications in `cmd/` can serve as E2E tests for basic library functionality.

## 13. Deployment Considerations (as a Library)

*   **Import:** Users import `pkg/cogmem` and other necessary public packages into their Go applications.
*   **Instantiation:** The user application instantiates the `cogmem.CogMemClient`, providing configuration and concrete implementations for required interfaces (especially LTMStore, ReasoningEngine, potentially Action/Perception).
*   **Versioning:** Use Go modules and semantic versioning.
*   **Dependencies:** Keep external dependencies minimal and well-justified.

This architecture provides a solid foundation for building the CogMem library in Go, balancing features, performance, multi-tenancy, flexibility, and maintainability.
