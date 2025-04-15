# CogMem Golang Library: Phase 2 Implementation Plan (Test-First)

**Version:** 1.0 (Phase 2 Detail Plan)
**Date:** 2023-10-27 (Placeholder)
**Corresponding Project Structure Version:** 1.0 rev 2
**Based on Implementation Plan Version:** 4.0

## 1. Phase 2 Goal

Add the *capability* to use **chromem-go** as a Vector LTM backend, implement embedding generation via a real Reasoning Engine adapter (e.g., OpenAI), enhance the MMU to handle vector storage/retrieval *when the vector backend is configured*, and introduce the basic Reflection module infrastructure. Deliver a library capable of performing semantic search (RAG) and initiating basic self-reflection cycles, driven by tests. *Note: The MMU in this phase primarily interacts with the single LTM backend specified in the configuration, but gains the logic to handle vector operations if that backend supports them.*

## 2. Overall Approach: Test-First Development (TFD)

For each significant piece of functionality within this phase:
1.  **Write Tests:** Define unit or integration tests that specify the desired behavior and cover primary use cases and edge conditions *before* writing implementation code.
2.  **Implement:** Write the minimum code necessary to make the tests pass.
3.  **Refactor:** Improve the code's structure, clarity, and efficiency while ensuring tests continue to pass.

---

## 3. Detailed Steps for Phase 2

### Step 1: Testing Infrastructure for Vector LTM (`test/`)

*   **1.1. Add Chromem-go Dependency:**
    *   Add `chromem-go` (`go get github.com/philippgille/chromem-go`)
*   **1.2. Setup Docker Compose for Testing:**
    *   Create/Update `test/docker-compose.test.yml` to include a `chromem-go` service - not required, use embedded chromem-go for testing.
    *   *Note:* If `chromem-go` offers a reliable in-memory mode suitable for testing, that can be used as an alternative or addition to Docker.
*   **1.3. Enhance Test Helpers (`test/testutil/`):**
    *   **1.3.1. Implement:** Create helper functions (`test/testutil/chromem.go`) to:
        *   Start/stop the Chromem-go container defined in `docker-compose.test.yml` (using libraries like `testcontainers-go` or custom shell scripts executed from Go tests). - not required
        *   Provide a function `CreateTempChromemGoClient() (*chromemgo.Client, cleanupFunc func())` that connects to the test instance and provides a cleanup function (potentially including clearing test collections).
    *   **1.3.2. Test:** Add a simple test within `test/testutil/` to verify the helper can connect to a running Chromem-go instance.

### Step 2: Chromem-go Vector LTM Adapter (`pkg/mem/ltm/adapters/vector/chromem_go/`)

*   **2.1. Implement (TDD):**
    *   **2.1.1. Test (`chromem_go_test.go` - Integration):** Write integration tests *first*, using the `testutil.CreateTempChromemGoClient` helper:
        *   **Test Setup:** Ensure tests create unique Chromem-go collections for isolation and clean them up.
        *   **Test Store:** Verify `Store` correctly adds records, including non-nil `Embedding` vectors, `Metadata`, and `EntityID` (likely stored as metadata for filtering). Test storing multiple records.
        *   **Test Retrieve (Semantic):** Verify `Retrieve` with a query vector correctly returns the most similar records based on cosine similarity (or the default distance metric). Test `k` limit.
        *   **Test Retrieve (Filtering):** Verify `Retrieve` correctly filters results based on `EntityID` provided in the `LTMQuery` (using Chromem-go's `where` clause on metadata). Verify filtering by `AccessLevel` and `UserID` (also via metadata). Test combination of semantic search *and* metadata filtering.
        *   **Test Retrieve (ID Lookup):** Verify retrieval by record ID (if supported by the interface/adapter, potentially via metadata filtering).
        *   **Test Update:** Verify updating a record's content, metadata, and embedding.
        *   **Test Delete:** Verify deleting a record.
        *   **Test Edge Cases:** Test storing/retrieving records with empty metadata, handling errors from the Chromem-go client (e.g., connection issues - may require mocking the client interface for *some* unit tests if integration testing is too slow/complex).
    *   **2.1.2. Implement Adapter (`chromem_go.go`):** Implement the `LTMStore` interface using the `chromem-go` SDK (`chromemgo` package).
        *   Implement logic to map `MemoryRecord` fields (including `EntityID`, `UserID`, `AccessLevel`) to Chromem-go's document structure (content, embeddings, metadata). Pay attention to filtering requirements.
        *   Implement the different retrieval modes (semantic, filtered, ID-based).
        *   Use structured logging (`pkg/log`).
        *   Ensure all integration tests pass.

### Step 3: Reasoning Engine - Real Embedding Generation (`pkg/reasoning/adapters/openai/`)

*   **3.1. Implement (TDD):**
    *   **3.1.1. Test (`openai_test.go` - Unit/Integration):**
        *   **Test `GenerateEmbeddings` (Unit - Mocked HTTP):** Write unit tests mocking the HTTP client used by the OpenAI Go library (`go-openai`). Verify that `GenerateEmbeddings` constructs the correct API request (model, input texts) and correctly parses the embedding vectors from the mocked response. Test handling of API errors (rate limits, auth errors) from the mocked response.
        *   **Test `GenerateEmbeddings` (Integration - Optional/CI-only):** Write integration tests (potentially skipped locally, run only in CI with secrets) that call the actual OpenAI API (using a test key/budget). Verify it returns embeddings of the expected dimension for a given model. *Caution: This incurs cost and external dependency.*
        *   **Test `Process` (Unit - Mocked HTTP):** Write basic unit tests for the `Process` method (if implementing chat/completion here too), mocking the HTTP client similarly.
    *   **3.1.2. Implement Adapter (`openai.go`):**
        *   Add `go-openai` dependency (`go get github.com/sashabaranov/go-openai`).
        *   Implement the `ReasoningEngine` interface using the `go-openai` client.
        *   Focus primarily on implementing `GenerateEmbeddings`. Implement `Process` if needed for reflection later.
        *   Handle configuration (API key, model selection) passed during adapter initialization.
        *   Use structured logging (`pkg/log`).
        *   Ensure unit tests pass.

### Step 4: MMU Enhancements for Vector Operations (`pkg/mmu/`)

*   **4.1. Implement (TDD):**
    *   **4.1.1. Test (`mmu_test.go` - Unit):** Extend MMU unit tests, mocking `LTMStore` and `ReasoningEngine`:
        *   **Test Conditional Embedding:** Mock the configured `LTMStore` to indicate vector support (e.g., via a new interface method `SupportsVectorSearch() bool` or type assertion). Verify `EncodeToLTM` *only* calls `ReasoningEngine.GenerateEmbeddings` if the LTM store supports vectors and the input data requires embedding. Verify the generated embedding is added to the `MemoryRecord` before `LTMStore.Store` is called. Test cases where embedding fails.
        *   **Test Semantic Retrieval:** Verify `RetrieveFromLTM` uses a specific "semantic" retrieval strategy when requested in `LTMQuery` options *and* the configured LTM store supports it. Verify it constructs the query vector (potentially by embedding the query text using `ReasoningEngine.GenerateEmbeddings`) and passes it to `LTMStore.Retrieve`.
        *   **Test Retrieval Strategy Selection:** Test how the MMU selects between keyword/ID lookup and semantic retrieval based on `LTMQuery` options or configuration.
        *   **Test WM Overflow (Basic):** Add tests for the `manage_wm_overflow` logic (if implementing). Mock context size calculation. Verify it selects records to evict (e.g., based on simple heuristics like LRU initially) and calls `EncodeToLTM` for them.
        *   **Test New Lua Hooks:** Test any new Lua hooks related to embedding generation (e.g., `before_embedding`) or semantic retrieval results (e.g., `rank_semantic_results`).
    *   **4.1.2. Implement (`mmu.go`, `retrieval.go`, `lua_hooks.go`, `overflow.go`):**
        *   Modify the `MMU` struct and constructor if needed (e.g., to accept the `ReasoningEngine` for embedding queries).
        *   Enhance `EncodeToLTM` logic to conditionally call `GenerateEmbeddings`. Consider adding a way for `LTMStore` implementations to signal vector support.
        *   Implement the semantic retrieval strategy in `retrieval.go`.
        *   Implement basic WM overflow logic in `overflow.go`.
        *   Integrate new Lua hook calls in `lua_hooks.go`.
        *   Use structured logging (`pkg/log`).
        *   Ensure all unit tests pass.

### Step 5: Reflection Module - Basic Structure (`pkg/reflection/`)

*   **5.1. Implement (TDD):**
    *   **5.1.1. Test (`reflection_test.go` - Unit):** Write unit tests for the basic reflection flow, mocking `MMU` and `ReasoningEngine`:
        *   **Test Module Creation:** Test creating the `ReflectionModule` instance with its dependencies.
        *   **Test Triggering:** Test triggering the reflection process (e.g., via a `TriggerReflection(ctx)` method).
        *   **Test Analysis Flow:** Verify the core analysis steps:
            *   Retrieving relevant history/data via `MMU.RetrieveFromLTM` (mocked). Define what "relevant" means initially (e.g., last N interactions).
            *   Calling `ReasoningEngine.Process` with a prompt designed for analysis/insight generation (mocked response).
            *   Parsing the LLM response into structured `Insight` objects.
        *   **Test Consolidation Trigger:** Verify the module calls `MMU.ConsolidateLTM` with the generated insights (mocked). `ConsolidateLTM` in the MMU might still be a placeholder or have very basic logic in this phase.
        *   **Test Error Handling:** Test how errors from dependencies (MMU, Reasoning) are handled.
        *   **Test Lua Hooks:** Test basic Lua hooks (`before_reflection_analysis`, `after_insight_generation`).
    *   **5.1.2. Implement (`reflection.go`, `analyzer.go`, `insight.go`, `lua_hooks.go`):**
        *   Define the `ReflectionModule` interface and struct.
        *   Define the `Insight` struct.
        *   Implement the basic `TriggerReflection` method orchestrating the analysis flow.
        *   Implement `analyzer.go` containing logic to fetch history and call the LLM for analysis.
        *   Implement `insight.go` for parsing LLM responses into insights.
        *   Implement `lua_hooks.go` for calling reflection-related Lua scripts.
        *   Use structured logging (`pkg/log`).
        *   Ensure all unit tests pass.

### Step 6: CogMemClient Facade Integration (`pkg/cogmem/`)

*   **6.1. Implement (TDD):**
    *   **6.1.1. Test (`cogmem_test.go` - Unit):** Update CogMemClient unit tests:
        *   **Test RAG Context:** Verify that when semantic retrieval results are returned from `MMU.RetrieveFromLTM`, they are formatted and included appropriately in the context passed to `ReasoningEngine.Process`.
        *   **Test Reflection Triggering:** Mock the `ReflectionModule`. Verify the `AgeCogMemClientnt`'s main processing loop calls `ReflectionModule.TriggerReflection` based on configured conditions (e.g., after every N interactions, or on specific error types).
    *   **6.1.2. Implement (`cogmem.go`, `controller.go`):**
        *   Modify the `CogMemClient` struct/constructor to accept the `ReflectionModule`.
        *   Update the controller logic to handle RAG results from the MMU and include them in the reasoning context.
        *   Add logic to trigger the `ReflectionModule` based on simple conditions.
        *   Use structured logging (`pkg/log`).
        *   Ensure unit tests pass.

### Step 7: Configuration and Example Application Updates

*   **7.1. Update Configuration (`pkg/config`, `configs/`):**
    *   **7.1.1. Implement (TDD):** Modify `config.go` tests and struct to:
        *   Include configuration sections for `LTM.ChromemGo` (URL, collection name, etc.).
        *   Allow `LTM.Type: "chromemgo"`.
        *   Include basic `Reflection` settings (e.g., `TriggerFrequency: <int>`, `AnalysisModel: <string>`).
        *   Include `Reasoning.OpenAI` settings (API key, embedding model, chat model).
    *   **7.1.2. Update Example Config:** Update `configs/config.example.yaml` with the new sections.
*   **7.2. Enhance Example Application (`cmd/example-client/`):**
    *   **7.2.1. Implement:** Modify `cmd/example-client/main.go`:
        *   Add logic to instantiate the `ChromemGoAdapter` if configured.
        *   Add logic to instantiate the `OpenAIAdapter` (reading key from config/env).
        *   Add logic to instantiate the `ReflectionModule`.
        *   Pass the real `ReasoningEngine` and `ReflectionModule` to the `CogMemClient`.
        *   Modify the CLI loop or add commands to demonstrate RAG: e.g., a `!search <query>` command that explicitly triggers semantic retrieval and displays results, or modify the default processing to use RAG.
        *   Add logging output indicating when reflection is triggered.
    *   **7.2.2. Manual Test:** Run the example application configured with `ChromemGoAdapter` and `OpenAIAdapter`.
        *   Verify storing data results in vector embeddings being generated (check logs/debug).
        *   Verify semantic search retrieves relevant results.
        *   Verify basic reflection cycle triggers and logs activity.

### Step 8: (Optional Stretch) Other Vector LTM Adapters

*   **8.1.** If time permits, implement adapters for other vector stores (e.g., `pkg/mem/ltm/adapters/vector/postgres_pgvector/`, `pkg/mem/ltm/adapters/vector/weaviate/`) following the same TDD pattern used for Chromem-go (Steps 1 & 2, adapting test helpers and implementation details).

### Step 9: Phase 2 Review & Refactor

*   **9.1. Code Review:** Conduct peer reviews focusing on Chromem-go integration, MMU vector logic, Reflection module structure, Reasoning engine implementation, and CogMemClient loop changes.
*   **9.2. Test Coverage:** Check coverage, especially for new modules/adapters and modified MMU logic. Add tests for gaps.
*   **9.3. Refactor:** Implement improvements based on reviews. Ensure clarity, robustness, and adherence to conventions.
*   **9.4. CI Verification:** Confirm the full test suite (including Chromem-go integration tests if feasible in CI) passes reliably.
*   **9.5. Documentation:** Update `README.md`, godoc comments, and potentially add a new `docs/rag.md` or `docs/reflection.md`. Explain how to configure Chromem-go, OpenAI, and the basic reflection settings. Clarify that the library now *supports* vector LTM when configured, setting the stage for hybrid LTM in Phase 3.

---

**Outcome / Deliverable (Phase 2):** A library that can be configured to use **Chromem-go** for vector LTM, enabling semantic search (RAG). It includes a real **OpenAI adapter** for generating embeddings. The **MMU** understands vector operations for the configured backend. A basic **Reflection module** is implemented and integrated into the agent loop. The library is tested, documented for Phase 2 capabilities, and ready for Phase 3 (Hybrid LTM Orchestration and Graph LTM).