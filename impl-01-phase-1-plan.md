# CogMem Golang Library: Phase 1 Implementation Plan (Test-First)

**Version:** 1.0 (Phase Plan)
**Date:** 2023-10-27 (Placeholder)
**Corresponding Project Structure Version:** 1.0

## 1. Goals

*   Establish the core library structure, interfaces, and testing infrastructure.
*   Implement foundational memory capabilities using SQL (via `sqlc`) and KV stores (Redis, Postgres HStore).
*   Integrate Lua scripting (`gopher-lua`) with basic MMU hooks.
*   Ensure robust multi-entity context handling from the start.
*   Deliver a minimally viable library capable of structured data storage/retrieval and basic scriptable logic, driven by tests.

## 2. Overall Approach: Test-First Development (TFD)

For each significant piece of functionality:
1.  **Write Tests:** Define unit or integration tests that specify the desired behavior and cover primary use cases and edge conditions.
2.  **Implement:** Write the minimum code necessary to make the tests pass.
3.  **Refactor:** Improve the code's structure, clarity, and efficiency while ensuring tests still pass.

---

## 3. Detailed Steps

### Step 1: Project Initialization & Setup (Foundation)

*   **1.1.** Initialize Git repository (`git init`).
*   **1.2.** Initialize Go module (`go mod init <your-module-path>`).
*   **1.3.** Create top-level directory structure:
    *   `pkg/`, `internal/`, `cmd/`, `configs/`, `scripts/`, `migrations/`, `test/`
    *   Subdirs within `pkg/`: `agent/`, `config/`, `entity/`, `errors/`, `mem/ltm/adapters/`, `mmu/`, `reasoning/adapters/`, `scripting/`
*   **1.4.** Add basic `.gitignore`, `README.md` (with project description), `LICENSE`.
*   **1.5.** Setup basic CI pipeline (`.github/workflows/go.yml`) to run `go build ./...` and `go test ./...` on pushes/PRs.

### Step 2: Core Types, Interfaces & Errors (`pkg/`)

*   **2.1. `pkg/entity`:**
    *   **2.1.1. (TDD) Test `entity_test.go`:** Write unit tests for `EntityID`, `AccessLevel` types (creation, comparison). Define tests for `entity.Context` struct creation and validation.
    *   **2.1.2. Implement `entity.go`:** Define types (`type EntityID string`, `type AccessLevel int const(...)`, `Context struct{...}`). Pass tests.
    *   **2.1.3. (TDD) Test `context_test.go`:** Write unit tests for helper functions to embed/retrieve `entity.Context` data within Go's standard `context.Context`.
    *   **2.1.4. Implement `context.go`:** Implement context helper functions. Pass tests.
*   **2.2. `pkg/errors`:**
    *   **2.2.1. (TDD) Test `errors_test.go`:** Define initial custom error variables/types (e.g., `ErrNotFound`, `ErrInvalidInput`, `ErrPermissionDenied`). Write tests ensuring they can be created, wrap standard errors (`fmt.Errorf("... %w", err)`), and checked using `errors.Is`/`errors.As`.
    *   **2.2.2. Implement `errors.go`:** Define error types/variables. Pass tests.
*   **2.3. Define Core Interfaces:**
    *   **2.3.1. Define `LTMStore` interface in `pkg/mem/ltm/ltm.go`.** Include methods like `Store(ctx context.Context, record MemoryRecord) error`, `Retrieve(ctx context.Context, query LTMQuery) ([]MemoryRecord, error)`, `Update(...)`, `Delete(...)`. Ensure signatures accept `context.Context`.
    *   **2.3.2. Define `MMU` interface in `pkg/mmu/mmu.go`.** Include methods like `EncodeToLTM(ctx context.Context, dataToStore interface{}) error`, `RetrieveFromLTM(ctx context.Context, query LTMQuery) ([]MemoryRecord, error)`, `ConsolidateLTM(...) error`.
    *   **2.3.3. Define `ScriptingEngine` interface in `pkg/scripting/engine.go`.** Include methods like `LoadScript(name string, content []byte) error`, `ExecuteFunction(ctx context.Context, funcName string, args ...interface{}) (interface{}, error)`.
    *   **2.3.4. Define `ReasoningEngine` interface in `pkg/reasoning/engine.go`.** Include `Process(ctx context.Context, prompt string, options ...ReasoningOption) (string, error)`, `GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)`.
    *   *Note: Tests for interfaces are written when implementing adapters/consumers.*

### Step 3: Configuration (`pkg/config`)

*   **3.1. (TDD) Test `config_test.go`:** Write unit tests for loading config from YAML. Define `Config` struct in the test first (include fields for `LTM.Type`, `LTM.SQL.DSN`, `LTM.KV.Redis.Addr`, etc.). Test defaults, required fields, environment variable overrides (using `viper` or similar).
*   **3.2. Implement `config.go`, `load.go`:** Define the `Config` struct. Implement loading logic using `viper`. Pass tests.
*   **3.3. Create `configs/config.example.yaml`:** Provide a sample configuration file matching the struct.

### Step 4: Testing Infrastructure (`test/`)

*   **4.1. Create `test/docker-compose.test.yml`:** Define services for PostgreSQL (ensure `postgres-contrib` is included for HStore) and Redis.
*   **4.2. Implement `test/testutil/db.go` (or similar):** Write Go helper functions using libraries like `ory/dockertest` to programmatically start/stop these Docker containers before/after integration test suites and provide database connections (`*sql.DB`, `redis.Client`) to tests.
*   **4.3. Setup Mocking:** Add `testify/mock` dependency (`go get github.com/stretchr/testify/mock`).

### Step 5: LTM - Mock Adapter (`pkg/mem/ltm/`)

*   **5.1. Define `MemoryRecord` struct in `pkg/mem/ltm/ltm.go`:** Include fields like `ID`, `EntityID`, `UserID` (optional, for private), `AccessLevel`, `Content`, `Metadata` (map[string]interface{}), `Embedding` ([]float32), `CreatedAt`, `UpdatedAt`.
*   **5.2. (TDD) Test `adapters/mock/mock_test.go`:** Write unit tests specifically for the `LTMStore` interface targeting the *mock* adapter. Use table-driven tests covering:
    *   `Store`: Verify record is stored.
    *   `Retrieve`: Verify filtering by `EntityID`, exact key match, simple metadata match.
    *   `Retrieve` Isolation: Test retrieving for Entity A doesn't return Entity B's data.
    *   `Retrieve` Access Level: Test basic `private_to_user` vs `shared_within_entity` retrieval logic (requires passing appropriate user info in `entity.Context`).
    *   `Update`/`Delete`: Verify correct record is affected based on ID and `EntityID`.
*   **5.3. Implement `adapters/mock/mock.go`:** Create an in-memory map (`map[EntityID]map[RecordID]MemoryRecord` or similar) and implement the `LTMStore` interface methods to pass the unit tests. Use mutexes for basic concurrency safety within the mock.

### Step 6: LTM - SQL Adapters & Migrations (`pkg/mem/ltm/adapters/sqlstore/`, `migrations/`)

*   **6.1. Schema & Migrations:**
    *   **6.1.1. Write `migrations/0001_init_schema.up.sql`, `...down.sql`:** Initial setup (e.g., enable extensions like HStore if needed globally).
    *   **6.1.2. Write `migrations/0002_create_memory_records.up.sql`, `...down.sql`:** Define the `memory_records` table matching the `MemoryRecord` struct (map Go types to SQL types). Add `entity_id`, `access_level`, `user_id` columns. Define appropriate indexes (on `entity_id`, potentially GIN index for metadata).
    *   **6.1.3. Implement `migrations/embed.go`:** Use `//go:embed *.sql`.
    *   **6.1.4. (TDD) Test `test/integration/migration_test.go`:** Write integration tests using `golang-migrate/migrate` and the test DB helper (`testutil`) to apply all UP migrations, verify table existence/structure, and then apply all DOWN migrations successfully.
*   **6.2. SQLC Setup:**
    *   **6.2.1. Create `pkg/mem/ltm/adapters/sqlstore/sqlc.yaml`:** Configure `sqlc` for Postgres and SQLite drivers.
    *   **6.2.2. Write `pkg/mem/ltm/adapters/sqlstore/query.sql`:** Define SQL queries for `Store` (INSERT), `Retrieve` (SELECT with mandatory `WHERE entity_id = ?` and optional `access_level = ?`, `user_id = ?`), `Update`, `Delete`. Use `sqlc` syntax (`-- name: CreateMemoryRecord :one`).
    *   **6.2.3. Run `sqlc generate`:** Generate Go code for queries.
*   **6.3. Adapter Implementation:**
    *   **6.3.1. (TDD) Test `postgres/postgres_test.go`, `sqlite/sqlite3_test.go`:** Write integration tests using the *real* databases via test helpers. Reuse/adapt mock test cases but verify against actual DB state:
        *   CRUD operations succeed.
        *   `EntityID` isolation is strictly enforced by queries.
        *   `AccessLevel` filtering works as expected.
        *   Data types are correctly mapped (timestamps, metadata maps/JSONB).
    *   **6.3.2. Implement `postgres/postgres.go`:** Create a struct implementing `LTMStore`. Use `sqlc`-generated functions, `pgx` connection pool. Pass Postgres integration tests.
    *   **6.3.3. Implement `sqlite/sqlite3.go`:** Create a struct implementing `LTMStore`. Use `sqlc`-generated functions, `database/sql` with SQLite driver. Pass SQLite integration tests.
    *   **6.3.4. (Refactor) `internal/dbutil`:** If common connection handling/retry logic emerges, refactor it here.

### Step 7: LTM - KV Adapters (`pkg/mem/ltm/adapters/kv/`)

*   **7.1. (TDD) Test `redis/redis_test.go`, `postgres/postgres_hstore_test.go`:** Write integration tests targeting Redis and Postgres HStore via test helpers. Verify:
    *   CRUD operations work using KV patterns (e.g., storing records as JSON/protobuf blobs, using HSET in Redis, HStore column in Postgres).
    *   Entity Isolation: Use key prefixing (e.g., `entity:<entity_id>:record:<record_id>`) or similar strategies and verify retrieval only gets data for the correct entity.
    *   Access Level filtering (might be harder in pure KV, potentially store access level within the value blob and filter client-side, or use secondary indexes if available).
*   **7.2. Implement `redis/redis.go`:** Use a Redis client library (`go-redis/redis`). Implement `LTMStore` methods using appropriate Redis commands (SET/GET, HSET/HGETALL, DEL, SCAN for retrieval by entity). Pass Redis tests.
*   **7.3. Implement `postgres/postgres_hstore.go`:** Use `pgx`. Requires queries that interact with the HStore column (ensure migration added it). Pass HStore tests.

### Step 8: Lua Scripting Engine (`pkg/scripting/`)

*   **8.1. (TDD) Test `engine_test.go`:** Write unit tests for the `ScriptingEngine` interface implementation:
    *   `LoadScript`: Test loading valid and invalid Lua code.
    *   `ExecuteFunction`: Test calling existing/non-existing Lua functions, passing basic args (numbers, strings), receiving basic return values. Test error handling for Lua runtime errors.
    *   Sandboxing: Explicitly test that `os.execute`, `io.open` (etc.) are `nil` or error out within the executed Lua environment. Use `lua.LState.DoString("return os == nil")`.
*   **8.2. Implement `engine.go`:** Implement the engine using `gopher-lua`. Use `lua.NewState(lua.Options{SkipOpenLibs: true})` and selectively open safe libraries (`base`, `string`, `table`, `math`).
*   **8.3. (TDD) Test `api_test.go`:** Write unit tests for the Go functions intended to be callable from Lua.
    *   Define simple Go functions (e.g., `logMessage(level, msg string)`, `getContextValue(key string)`).
    *   Write tests that execute Lua code calling these Go functions (using the `ScriptingEngine`), verifying the Go functions were called with correct args (using mocks or spies if needed) and that data returned from Go is usable in Lua. Test passing complex Go structs/maps/slices and verify they become Lua tables.
*   **8.4. Implement `api.go`, `sandbox.go`:** Implement the Go API functions and expose them to the Lua state during initialization. Finalize sandbox setup. Pass all scripting tests.
*   **8.5. Create `scripts/mmu/retrieval_filter.lua` (basic):** Add placeholder Lua functions like `function before_retrieve(ctx, query) -- Log or modify query; return query end`.

### Step 9: Basic Memory Management Unit (MMU) (`pkg/mmu/`)

*   **9.1. (TDD) Test `mmu_test.go`:** Write unit tests for the `MMU` implementation. Use `testify/mock` to mock `LTMStore` and `ScriptingEngine`.
    *   Test `EncodeToLTM` calls `LTMStore.Store` with the correct `MemoryRecord` (transformed from input data) and `entity.Context`.
    *   Test `RetrieveFromLTM` calls `LTMStore.Retrieve` with correct query parameters derived from input and `entity.Context`.
    *   Test Lua Hooks:
        *   Verify `ScriptingEngine.ExecuteFunction("before_retrieve", ...)` is called before `LTMStore.Retrieve`.
        *   Verify `ScriptingEngine.ExecuteFunction("after_retrieve", ...)` is called after `LTMStore.Retrieve` with the results.
        *   Test handling of errors returned by Lua functions.
*   **9.2. Implement `mmu.go`, `lua_hooks.go`:** Implement the basic MMU logic, including calls to the `ScriptingEngine` at the tested hook points. Pass unit tests.

### Step 10: Reasoning Engine - Mock (`pkg/reasoning/`)

*   **10.1. (TDD) Test `adapters/mock/mock_test.go`:** Write unit tests for the mock `ReasoningEngine` adapter. Test setting canned responses/embeddings and verifying calls to `Process` and `GenerateEmbeddings`.
*   **10.2. Implement `adapters/mock/mock.go`:** Implement the mock adapter to satisfy the `ReasoningEngine` interface and pass tests.

### Step 11: Agent Facade & Loop (`pkg/agent/`)

*   **11.1. (TDD) Test `agent_test.go`:** Write unit tests for the `Agent` struct and its main processing method(s). Mock all dependencies (`MMU`, `ReasoningEngine`, `ScriptingEngine`, etc.).
    *   Test the basic control flow: Input -> `MMU.Retrieve` -> `Reasoning.Process` -> Output/Action (mocked).
    *   Verify `entity.Context` is correctly extracted and passed down to dependencies.
    *   Test basic error handling if dependencies return errors.
*   **11.2. Implement `agent.go`, `controller.go`:** Implement the `Agent` struct accepting dependencies via constructor (Dependency Injection). Implement the core orchestration logic. Pass tests.

### Step 12: Example Application (`cmd/example-agent/`)

*   **12.1. Implement `cmd/example-agent/main.go`:**
    *   Load configuration from `configs/config.example.yaml`.
    *   Instantiate `LTMStore` (e.g., Postgres adapter using config DSN).
    *   Instantiate `ScriptingEngine`, load example scripts.
    *   Instantiate mock `ReasoningEngine`.
    *   Instantiate the `Agent` with the configured dependencies.
    *   Run a simple command-line loop: Read input -> Create `entity.Context` (e.g., hardcoded entity ID) -> Call `Agent.Process(...)` -> Print output. Include commands to explicitly store ("remember") and retrieve information.
*   **12.2. Manual Test:** Run the example application. Verify storing data for entity "A" and retrieving it works. Verify storing for entity "B" doesn't interfere. Verify basic Lua script logs appear if hooks print messages.

### Step 13: Phase 1 Review & Refactor

*   **13.1. Code Review:** Review all implemented code for clarity, adherence to Go best practices, consistency, potential bugs.
*   **13.2. Test Coverage:** Check test coverage; add tests for any significant gaps identified.
*   **13.3. Refactor:** Address review comments, improve code structure, remove duplication.
*   **13.4. CI Verification:** Ensure all tests (unit, integration, migration) pass reliably in the CI pipeline.
*   **13.5. Documentation:** Update `README.md` explaining Phase 1 features, how to configure LTM (SQL/KV), basic Lua hooks, and how to run the example. Add godoc comments to public interfaces and types.
