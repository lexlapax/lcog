# CogMem Golang Library: Phase 1 Implementation Plan (Test-First)

**Version:** 1.0 (Phase 1 Detail Plan)
**Date:** 2023-10-27 (Placeholder)
**Corresponding Project Structure Version:** 1.0 rev 2
**Based on Implementation Plan Version:** 3.0

## 1. Phase 1 Goal

Establish the core library structure, interfaces, and testing infrastructure. Implement foundational memory capabilities using **SQLite** (via `sqlc`) and **BoltDB** KV store, integrate Lua scripting (`gopher-lua`) with basic hooks, and ensure robust multi-entity context handling. Deliver a minimally viable library capable of structured data storage/retrieval (focused on embedded/file-based options first) and basic scriptable logic, driven by tests.

## 2. Overall Approach: Test-First Development (TFD)

For each significant piece of functionality within this phase:
1.  **Write Tests:** Define unit or integration tests that specify the desired behavior and cover primary use cases and edge conditions *before* writing implementation code.
2.  **Implement:** Write the minimum code necessary to make the tests pass.
3.  **Refactor:** Improve the code's structure, clarity, and efficiency while ensuring tests continue to pass.

---

## 3. Detailed Steps for Phase 1

### Step 1: Project Initialization & Setup (Foundation)

*   **1.1.** Initialize Git repository (`git init`).
*   **1.2.** Initialize Go module (`go mod init <your-module-path>`).
*   **1.3.** Create top-level directory structure:
    *   `pkg/`, `internal/`, `cmd/`, `configs/`, `scripts/`, `migrations/` (initially might be minimal if focusing on SQLite/BoltDB first), `test/`
    *   Subdirs within `pkg/`: `agent/`, `config/`, `entity/`, `errors/`, `log/`, `mem/`, `mem/ltm/`, `mem/ltm/adapters/`, `mem/ltm/adapters/mock/`, `mem/ltm/adapters/sqlstore/`, `mem/ltm/adapters/sqlstore/sqlite/`, `mem/ltm/adapters/kv/`, `mem/ltm/adapters/kv/boltdb/`, `mmu/`, `reasoning/`, `reasoning/adapters/`, `reasoning/adapters/mock/`, `scripting/`
*   **1.4.** Add basic `.gitignore`, `README.md` (with project description), `LICENSE`.
*   **1.5.** Setup basic CI pipeline (`.github/workflows/go.yml`) to run `go build ./...` and `go test ./...` on pushes/PRs.

### Step 2: Core Types, Interfaces & Errors (`pkg/`)

*   **2.1. Implement `pkg/entity` (TDD):**
    *   **2.1.1. Test:** Write unit tests (`entity_test.go`) for `EntityID`, `AccessLevel`, and `entity.Context` struct creation/validation.
    *   **2.1.2. Implement:** Define types and struct (`entity.go`). Pass tests.
    *   **2.1.3. Test:** Write unit tests (`context_test.go`) for helper functions managing `entity.Context` within Go's standard `context.Context`.
    *   **2.1.4. Implement:** Implement context helper functions (`context.go`). Pass tests.
*   **2.2. Implement `pkg/errors` (TDD):**
    *   **2.2.1. Test:** Write unit tests (`errors_test.go`) defining expected behavior for custom errors (wrapping standard errors, checking via `errors.Is`/`As`). Define initial errors like `ErrNotFound`, `ErrInvalidInput`.
    *   **2.2.2. Implement:** Define error types/variables (`errors.go`). Pass tests.
*   **2.3. Implement `pkg/log` (TDD):**
    *   **2.3.1. Test:** Write unit tests (`log_test.go`) for structured logging using Go's `log/slog` package. Test the creation of loggers with different levels, handlers, and contexts.
    *   **2.3.2. Implement:** Create a logging package (`log.go`) that provides standardized logging throughout the application. Support for different log levels, structured logging with context values, and customizable outputs.
*   **2.4. Define Core Interfaces:**
    *   **2.4.1. Define `LTMStore` interface in `pkg/mem/ltm/ltm.go`.** Include methods like `Store(ctx context.Context, record MemoryRecord) error`, `Retrieve(ctx context.Context, query LTMQuery) ([]MemoryRecord, error)`, `Update(...)`, `Delete(...)`. Ensure signatures accept `context.Context`.
    *   **2.4.2. Define `MMU` interface in `pkg/mmu/mmu.go`.** Include methods like `EncodeToLTM(ctx context.Context, dataToStore interface{}) error`, `RetrieveFromLTM(ctx context.Context, query LTMQuery) ([]MemoryRecord, error)`, `ConsolidateLTM(...) error` (placeholder for Phase 2).
    *   **2.4.3. Define `ScriptingEngine` interface in `pkg/scripting/engine.go`.** Include methods like `LoadScript(name string, content []byte) error`, `ExecuteFunction(ctx context.Context, funcName string, args ...interface{}) (interface{}, error)`.
    *   **2.4.4. Define `ReasoningEngine` interface in `pkg/reasoning/engine.go`.** Include `Process(ctx context.Context, prompt string, options ...ReasoningOption) (string, error)`, `GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)` (implementation deferred).

### Step 3: Configuration (`pkg/config`)

*   **3.1. Implement (TDD):**
    *   **3.1.1. Test:** Write unit tests (`config_test.go`) for loading YAML config. Define `Config` struct in the test first (include fields for `LTM.Type` (string), `LTM.SQLite.Path` (string), `LTM.BoltDB.Path` (string), `Lua.ScriptsPath` (string), `Log.Level` (string), `Log.Format` (string)). Test defaults, required fields, environment variable overrides.
    *   **3.1.2. Implement:** Define the `Config` struct. Implement loading logic (`config.go`, `load.go`) using `viper` or similar. Pass tests.
*   **3.2. Create Example:** Add `configs/config.example.yaml` matching the defined struct, showing how to configure SQLite and BoltDB paths, logging level and format options.

### Step 4: Testing Infrastructure (`test/`)

*   **4.1. Implement Mocking:** Add `testify/mock` dependency (`go get github.com/stretchr/testify/mock`).
*   **4.2. Implement Test Helpers:** Create utilities in `test/testutil/` to:
    *   Create and manage temporary SQLite database files (`CreateTempSQLiteDB() (db *sql.DB, path string, cleanupFunc func())`).
    *   Create and manage temporary BoltDB database files (`CreateTempBoltDB() (db *bbolt.DB, path string, cleanupFunc func())`).
    *   Ensure `cleanupFunc` removes the temporary files.

### Step 5: LTM - Mock Adapter (`pkg/mem/ltm/adapters/mock/`)

*   **5.1. Define `MemoryRecord`:** Finalize the `MemoryRecord` struct definition in `pkg/mem/ltm/ltm.go`. Include essential fields: `ID` (string), `EntityID` (entity.EntityID), `UserID` (string, optional), `AccessLevel` (entity.AccessLevel), `Content` (string or []byte), `Metadata` (map[string]interface{}), `Embedding` ([]float32, initially nil), `CreatedAt` (time.Time), `UpdatedAt` (time.Time).
*   **5.2. Implement (TDD):**
    *   **5.2.1. Test:** Write unit tests (`mock_test.go`) covering the `LTMStore` interface for the mock adapter. Test CRUD operations, `EntityID` filtering, `AccessLevel` filtering logic (e.g., retrieving private only if UserID matches context, retrieving shared if EntityID matches), and verify isolation between entities using table-driven tests.
    *   **5.2.2. Implement:** Create the mock adapter (`mock.go`) using in-memory maps (e.g., `map[entity.EntityID]map[string]MemoryRecord`) protected by mutexes. Ensure all tests pass.

### Step 6: LTM - SQLite Adapter (`pkg/mem/ltm/adapters/sqlstore/sqlite/`)

*   **6.1. Implement (TDD):**
    *   **6.1.1. Test:** Write integration tests (`sqlite3_test.go`). Use the test helper (`testutil.CreateTempSQLiteDB`) for setup/teardown. Adapt mock tests to verify behavior against a real SQLite database: CRUD, strict `EntityID` filtering (using `WHERE entity_id = ?`), `AccessLevel`/`UserID` filtering, correct mapping of Go types (time.Time, map[string]interface{} likely via JSON marshal/unmarshal to TEXT column).
    *   **6.1.2. Configure SQLC:** Create `pkg/mem/ltm/adapters/sqlstore/sqlc.yaml` specifying the SQLite driver.
    *   **6.1.3. Write Queries:** Create `pkg/mem/ltm/adapters/sqlstore/query.sql`. Define necessary SQL statements (CREATE TABLE if not exists - potentially run on init, INSERT, SELECT with filtering, UPDATE, DELETE) using `sqlc` syntax.
    *   **6.1.4. Generate Code:** Run `sqlc generate`.
    *   **6.1.5. Implement Adapter:** Implement `sqlite3.go`. Create a struct holding the `*sql.DB`. Implement `LTMStore` methods using the `sqlc`-generated functions. Handle potential JSON marshalling/unmarshalling for metadata. Ensure all integration tests pass.

### Step 7: LTM - BoltDB Adapter (`pkg/mem/ltm/adapters/kv/boltdb/`)

*   **7.1. Implement (TDD):**
    *   **7.1.1. Test:** Write integration tests (`boltdb_test.go`). Use `testutil.CreateTempBoltDB` for setup/teardown. Verify `LTMStore` interface behavior:
        *   Use BoltDB buckets, potentially named by `EntityID`.
        *   Store `MemoryRecord` by marshalling (e.g., JSON, Gob) as the value associated with the record ID key within the entity's bucket.
        *   Verify CRUD operations within transactions (`db.Update`, `db.View`).
        *   Verify `EntityID` isolation by ensuring operations only affect the correct bucket.
        *   Verify `AccessLevel`/`UserID` filtering (likely requires iterating keys in a bucket, unmarshalling, and checking fields within the view transaction).
    *   **7.1.2. Implement Adapter:** Implement `boltdb.go`. Use the `go.etcd.io/bbolt` library. Implement `LTMStore` methods, managing buckets and transactions correctly. Handle marshalling/unmarshalling of `MemoryRecord`. Ensure all integration tests pass.

### Step 8: Lua Scripting Engine (`pkg/scripting/`)

*   **8.1. Implement (TDD):**
    *   **8.1.1. Test:** Write unit tests (`engine_test.go`) for the `ScriptingEngine` implementation: Test loading scripts (valid/invalid), executing functions (existing/non-existing), passing basic arguments (strings, numbers, bools), receiving return values, handling Lua runtime errors. Test sandboxing by asserting that `os`, `io`, etc., are nil or error out.
    *   **8.1.2. Implement Engine:** Implement `engine.go` using `gopher-lua`. Initialize Lua state (`lua.NewState`) with options to `SkipOpenLibs` and selectively open safe base libraries (`base`, `table`, `string`, `math`). Use the structured logger from `pkg/log` for all logging.
    *   **8.1.3. Test API:** Write unit tests (`api_test.go`) for Go functions exposed to Lua. Define simple Go functions (e.g., `Log(level string, msg string)`). Write tests executing Lua code that calls these functions, verifying calls (using spies/mocks if complex) and data marshalling (Go struct/map/slice <-> Lua table).
    *   **8.1.4. Implement API & Sandbox:** Implement the Go API functions (`api.go`) and register them with the Lua state during initialization. Finalize sandbox setup (`sandbox.go`). Ensure all scripting tests pass.
*   **8.2. Create Examples:** Add basic placeholder Lua scripts like `scripts/mmu/retrieval_filter.lua` containing empty or logging functions (e.g., `function before_retrieve(ctx, query) print("Lua: before_retrieve called") return query end`).

### Step 9: Basic Memory Management Unit (MMU) (`pkg/mmu/`)

*   **9.1. Implement (TDD):**
    *   **9.1.1. Test:** Write unit tests (`mmu_test.go`) for the `MMU` implementation. Mock `LTMStore` and `ScriptingEngine` interfaces.
        *   Test `EncodeToLTM`: Verify it correctly transforms input data into a `MemoryRecord` and calls `LTMStore.Store` with the record and the correct `entity.Context`.
        *   Test `RetrieveFromLTM`: Verify it constructs an appropriate `LTMQuery` based on input and context, calls `LTMStore.Retrieve`, and returns the results.
        *   Test Lua Hook Invocation: Verify `ScriptingEngine.ExecuteFunction("before_retrieve", ...)` is called before `LTMStore.Retrieve` with appropriate arguments (context, query). Verify `ScriptingEngine.ExecuteFunction("after_retrieve", ...)` is called after retrieval with results. Test correct handling if Lua functions return errors or modify data (e.g., the query).
    *   **9.1.2. Implement:** Implement the basic MMU logic in `mmu.go`. Implement the logic to call Lua hooks via the `ScriptingEngine` in `lua_hooks.go`. Use structured logging from `pkg/log` instead of the standard library's `log` package. Ensure unit tests pass.

### Step 10: Reasoning Engine - Mock (`pkg/reasoning/`)

*   **10.1. Implement (TDD):**
    *   **10.1.1. Test:** Write unit tests (`adapters/mock/mock_test.go`) for the mock `ReasoningEngine`. Test setting canned responses for `Process` and canned embeddings (e.g., `[][]float32{{0.1, 0.2}}`) for `GenerateEmbeddings`. Verify methods are called as expected.
    *   **10.1.2. Implement:** Implement the mock adapter (`adapters/mock/mock.go`) to satisfy the interface and pass tests.

### Step 11: Agent Facade & Loop (`pkg/agent/`)

*   **11.1. Implement (TDD):**
    *   **11.1.1. Test:** Write unit tests (`agent_test.go`) for the `Agent` facade. Mock all dependencies (`MMU`, `ReasoningEngine`, etc.). Test the basic controller flow: Input -> `MMU.Retrieve` (mocked results) -> `Reasoning.Process` (mocked response) -> Action/Output (verify args if action module mocked). Verify `entity.Context` propagation to all dependency calls. Test error handling scenarios.
    *   **11.1.2. Implement:** Implement the `Agent` struct (`agent.go`) accepting dependencies via a constructor (for DI). Implement the basic orchestration logic in `controller.go`. Pass unit tests.

### Step 12: Example Application (`cmd/example-agent/`)

*   **12.1. Implement (TDD):** Create `cmd/example-agent/main.go`. The application should:
    *   Load configuration (`pkg/config`).
    *   Instantiate the selected `LTMStore` (SQLite or BoltDB based on config).
    *   Instantiate the `ScriptingEngine` and load scripts from the configured path.
    *   Instantiate the mock `ReasoningEngine`.
    *   Instantiate the `pkg/agent.Agent` by injecting dependencies.
    *   Implement command line editing via `liner` package and test
    *   Run a simple command-line loop:
        *   Read user input.
        *   Prompt for an Entity ID.
        *   Create `entity.Context`.
        *   Call `Agent.Process(...)` with the input and context.
        *   Print the result.
        *   Include specific commands like `!remember <text>` (calls `MMU.EncodeToLTM`) and `!lookup <query>` (calls `MMU.RetrieveFromLTM`).
    
*   **12.2. Manual Test:** Run the example application against both SQLite and BoltDB configurations.
    *   Verify `!remember` stores data for the specified entity.
    *   Verify querying retrieves data only for the current entity.
    *   Verify storing for a different entity doesn't affect the first.
    *   Check console output for logs from Lua hooks (e.g., `Lua: before_retrieve called`).

### Step 13: Phase 1 Review & Refactor

*   **13.1. Code Review:** Conduct peer reviews of all code developed in Phase 1, focusing on correctness, clarity, adherence to Go conventions, test quality, and potential improvements.
*   **13.2. Test Coverage:** Use `go test -cover` tools to assess test coverage. Identify and address significant gaps in unit or integration tests.
*   **13.3. Refactor:** Implement improvements based on code reviews and test coverage analysis. Refine code structure, naming, error handling, and documentation. Ensure all tests continue to pass after refactoring.
*   **13.4. CI Verification:** Confirm that the full test suite passes reliably in the automated CI environment.
*   **13.5. Documentation:** Update `README.md` to accurately reflect Phase 1 capabilities (SQLite/BoltDB support, Lua hooks, basic agent loop). Add comprehensive godoc comments to all public types, functions, and interfaces within the `pkg/` directory. Explain basic configuration and how to run the example.