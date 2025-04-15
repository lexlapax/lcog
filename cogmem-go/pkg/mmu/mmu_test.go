package mmu

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/mock"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockVectorCapableLTMStore implements ltm.VectorCapableLTMStore for testing
type mockVectorCapableLTMStore struct {
	*mock.MockStore
	supportsVectors bool
}

func newMockVectorStore(supportsVectors bool) *mockVectorCapableLTMStore {
	return &mockVectorCapableLTMStore{
		MockStore:       mock.NewMockStore(),
		supportsVectors: supportsVectors,
	}
}

func (m *mockVectorCapableLTMStore) SupportsVectorSearch() bool {
	return m.supportsVectors
}

// mockScriptEngine implements a simple mock for the scripting.Engine interface
type mockScriptEngine struct {
	// Map of expected function calls to results
	functionResults map[string]interface{}
	// Record of function calls
	calls []mockCall
}

type mockCall struct {
	FunctionName string
	Args         []interface{}
}

func newMockScriptEngine() *mockScriptEngine {
	return &mockScriptEngine{
		functionResults: make(map[string]interface{}),
	}
}

// mockReasoningEngine implements a simple mock for the reasoning.Engine interface
type mockReasoningEngine struct {
	// Map of expected function calls to results
	embeddingResults map[string][]float32
	processResults   map[string]string
	// Record of function calls
	calls []mockCall
}

func newMockReasoningEngine() *mockReasoningEngine {
	return &mockReasoningEngine{
		embeddingResults: make(map[string][]float32),
		processResults:   make(map[string]string),
	}
}

// GenerateEmbeddings mocks the generation of embeddings.
func (m *mockReasoningEngine) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	m.calls = append(m.calls, mockCall{
		FunctionName: "GenerateEmbeddings",
		Args:         []interface{}{texts},
	})

	// Return a mock embedding for each text
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		// Return a predefined embedding if available
		if emb, ok := m.embeddingResults[text]; ok {
			embeddings[i] = emb
		} else {
			// Generate a default mock embedding - for tests only
			embeddings[i] = []float32{0.1, 0.2, 0.3, 0.4, 0.5}
		}
	}
	
	return embeddings, nil
}

// Process mocks sending a prompt to the reasoning engine.
func (m *mockReasoningEngine) Process(ctx context.Context, prompt string, opts ...reasoning.Option) (string, error) {
	m.calls = append(m.calls, mockCall{
		FunctionName: "Process",
		Args:         []interface{}{prompt},
	})
	
	// Return a predefined response if available
	if response, ok := m.processResults[prompt]; ok {
		return response, nil
	}
	
	// Return a default mock response
	return "Mock response to: " + prompt, nil
}

func (m *mockScriptEngine) LoadScript(name string, content []byte) error {
	return nil
}

func (m *mockScriptEngine) LoadScriptFile(path string) error {
	return nil
}

func (m *mockScriptEngine) LoadScriptDir(dir string) error {
	return nil
}

func (m *mockScriptEngine) ExecuteFunction(ctx context.Context, funcName string, args ...interface{}) (interface{}, error) {
	// Record the call
	m.calls = append(m.calls, mockCall{
		FunctionName: funcName,
		Args:         args,
	})
	
	// Return the mock result if configured
	if result, ok := m.functionResults[funcName]; ok {
		return result, nil
	}
	
	// Default return for tests that don't set up specific results
	return nil, nil
}

func (m *mockScriptEngine) Close() error {
	return nil
}

// Helper function to set up a test environment
func setupTest(t *testing.T, enableLuaHooks bool) (*MMUI, *mock.MockStore, *mockScriptEngine, *mockReasoningEngine, context.Context) {
	// Create mock dependencies
	ltmStore := mock.NewMockStore()
	scriptEngine := newMockScriptEngine()
	reasoningEngine := newMockReasoningEngine()
	
	// Create MMU with configuration
	mmu := NewMMU(ltmStore, reasoningEngine, scriptEngine, Config{
		EnableLuaHooks: enableLuaHooks,
		EnableVectorOperations: false, // Default to disabled for backward compatibility
	})
	
	// Create a context with entity information
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	return mmu, ltmStore, scriptEngine, reasoningEngine, ctx
}

// Helper function to set up a test environment with a vector-capable store
func setupVectorTest(t *testing.T, supportsVectors bool) (*MMUI, *mockVectorCapableLTMStore, *mockScriptEngine, *mockReasoningEngine, context.Context) {
	// Create mock dependencies
	ltmStore := newMockVectorStore(supportsVectors)
	scriptEngine := newMockScriptEngine()
	reasoningEngine := newMockReasoningEngine()
	
	// Create MMU with configuration
	mmu := NewMMU(ltmStore, reasoningEngine, scriptEngine, Config{
		EnableLuaHooks: true,
		EnableVectorOperations: true,
		WorkingMemoryLimit: 5,
	})
	
	// Create a context with entity information
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	return mmu, ltmStore, scriptEngine, reasoningEngine, ctx
}

func TestMMU_EncodeToLTM_String(t *testing.T) {
	// Setup
	mmu, ltmStore, _, _, ctx := setupTest(t, false)
	
	// Test encoding a simple string
	memoryID, err := mmu.EncodeToLTM(ctx, "This is a test memory")
	require.NoError(t, err)
	require.NotEmpty(t, memoryID)
	
	// Verify it was stored correctly
	query := ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": memoryID,
		},
	}
	records, err := ltmStore.Retrieve(ctx, query)
	require.NoError(t, err)
	require.Len(t, records, 1)
	
	// Verify record contents
	record := records[0]
	assert.Equal(t, "This is a test memory", record.Content)
	assert.Equal(t, entity.EntityID("test-entity"), record.EntityID)
	assert.Equal(t, "test-user", record.UserID)
	assert.Equal(t, entity.SharedWithinEntity, record.AccessLevel)
	assert.NotNil(t, record.Metadata)
	assert.Contains(t, record.Metadata, "encoded_at")
}

func TestMMU_EncodeToLTM_Map(t *testing.T) {
	// Setup
	mmu, ltmStore, _, _, ctx := setupTest(t, false)
	
	// Test encoding a map with content and metadata
	memoryData := map[string]interface{}{
		"content": "Map content test",
		"metadata": map[string]interface{}{
			"type": "note",
			"tags": []string{"test", "memory"},
		},
		"access_level": int(entity.PrivateToUser),
	}
	
	memoryID, err := mmu.EncodeToLTM(ctx, memoryData)
	require.NoError(t, err)
	
	// Verify it was stored correctly
	query := ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": memoryID,
		},
	}
	records, err := ltmStore.Retrieve(ctx, query)
	require.NoError(t, err)
	require.Len(t, records, 1)
	
	// Verify record contents
	record := records[0]
	assert.Equal(t, "Map content test", record.Content)
	assert.Equal(t, entity.PrivateToUser, record.AccessLevel)
	assert.Equal(t, "note", record.Metadata["type"])
}

func TestMMU_EncodeToLTM_NoEntityContext(t *testing.T) {
	// Setup without entity context
	mmu, _, _, _, _ := setupTest(t, false)
	
	// Should fail without entity context
	_, err := mmu.EncodeToLTM(context.Background(), "This should fail")
	assert.Error(t, err)
	assert.Equal(t, entity.ErrMissingEntityContext, err)
}

func TestMMU_RetrieveFromLTM_StringQuery(t *testing.T) {
	// Setup
	mmu, ltmStore, _, _, ctx := setupTest(t, false)
	
	// First store some test data
	testRecords := []ltm.MemoryRecord{
		{
			EntityID:    "test-entity",
			Content:     "This is the first test memory",
			AccessLevel: entity.SharedWithinEntity,
			Metadata:    map[string]interface{}{"type": "note"},
		},
		{
			EntityID:    "test-entity",
			Content:     "This is the second test memory",
			AccessLevel: entity.SharedWithinEntity,
			Metadata:    map[string]interface{}{"type": "note"},
		},
	}
	
	for _, record := range testRecords {
		_, err := ltmStore.Store(ctx, record)
		require.NoError(t, err)
	}
	
	// Test retrieval with string query
	results, err := mmu.RetrieveFromLTM(ctx, "first", DefaultRetrievalOptions())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "first")
	
	// Test retrieval with no results
	results, err = mmu.RetrieveFromLTM(ctx, "nonexistent", DefaultRetrievalOptions())
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestMMU_RetrieveFromLTM_MapQuery(t *testing.T) {
	// Setup
	mmu, ltmStore, _, _, ctx := setupTest(t, false)
	
	// First store some test data
	now := time.Now()
	testRecords := []ltm.MemoryRecord{
		{
			EntityID:    "test-entity",
			Content:     "Test memory with tag1",
			AccessLevel: entity.SharedWithinEntity,
			Metadata:    map[string]interface{}{"tag": "tag1", "importance": 5},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			EntityID:    "test-entity",
			Content:     "Test memory with tag2",
			AccessLevel: entity.SharedWithinEntity,
			Metadata:    map[string]interface{}{"tag": "tag2", "importance": 3},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	
	for _, record := range testRecords {
		_, err := ltmStore.Store(ctx, record)
		require.NoError(t, err)
	}
	
	// Test retrieval with map query using text
	mapQuery := map[string]interface{}{
		"text": "tag1",
	}
	results, err := mmu.RetrieveFromLTM(ctx, mapQuery, DefaultRetrievalOptions())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "tag1")
	
	// Test retrieval with map query using filters
	mapQuery = map[string]interface{}{
		"filters": map[string]interface{}{
			"tag": "tag2",
		},
	}
	results, err = mmu.RetrieveFromLTM(ctx, mapQuery, DefaultRetrievalOptions())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "tag2")
	
	// Test with metadata filtering disabled
	options := DefaultRetrievalOptions()
	options.IncludeMetadata = false
	results, err = mmu.RetrieveFromLTM(ctx, "tag1", options)
	require.NoError(t, err)
	if len(results) > 0 {
		assert.Nil(t, results[0].Metadata)
	}
}

func TestMMU_RetrieveFromLTM_WithLuaHooks(t *testing.T) {
	// Setup with Lua hooks enabled
	mmu, ltmStore, scriptEngine, _, ctx := setupTest(t, true)
	
	// Store a test record
	record := ltm.MemoryRecord{
		EntityID:    "test-entity",
		Content:     "This is a test memory for Lua hooks",
		AccessLevel: entity.SharedWithinEntity,
	}
	id, err := ltmStore.Store(ctx, record)
	require.NoError(t, err)
	
	// Configure the mock script engine to modify the query
	modifiedQuery := map[string]interface{}{
		"text": "modified by Lua",
		"limit": float64(5),
	}
	scriptEngine.functionResults["before_retrieve"] = modifiedQuery
	
	// Reset the calls tracking
	scriptEngine.calls = nil
	
	// Test retrieval to verify the hook was called
	_, err = mmu.RetrieveFromLTM(ctx, "original query", DefaultRetrievalOptions())
	require.NoError(t, err)
	
	// Verify the before_retrieve hook was called
	require.GreaterOrEqual(t, len(scriptEngine.calls), 1)
	assert.Equal(t, "before_retrieve", scriptEngine.calls[0].FunctionName)
	
	// Configure the mock script engine to modify the results
	scriptEngine.functionResults["after_retrieve"] = []interface{}{
		map[string]interface{}{
			"id": id,
			"content": "Modified by Lua",
		},
	}
	
	// Reset the calls tracking again
	scriptEngine.calls = nil
	
	// Test retrieval again
	_, err = mmu.RetrieveFromLTM(ctx, "test", DefaultRetrievalOptions())
	require.NoError(t, err)
	
	// Verify the after_retrieve hook was called
	found := false
	t.Logf("Calls made: %d", len(scriptEngine.calls))
	for i, call := range scriptEngine.calls {
		t.Logf("Call %d: %s", i, call.FunctionName)
		if call.FunctionName == "after_retrieve" {
			found = true
			break
		}
	}
	
	// Create a test record directly (without hooks) to get at least one result
	// This is needed because the after_retrieve hook is only called if there are results
	testRecord := ltm.MemoryRecord{
		ID:          "test-direct",
		EntityID:    "test-entity",
		Content:     "Direct test for after_retrieve",
		AccessLevel: entity.SharedWithinEntity,
	}
	_, err = ltmStore.Store(ctx, testRecord)
	require.NoError(t, err)
	
	// Reset the calls again
	scriptEngine.calls = nil
	
	// This should definitely trigger the after_retrieve hook since we know there are results
	_, err = mmu.RetrieveFromLTM(ctx, "Direct", DefaultRetrievalOptions())
	require.NoError(t, err)
	
	// Check again
	found = false
	t.Logf("Calls made after direct test: %d", len(scriptEngine.calls))
	for i, call := range scriptEngine.calls {
		t.Logf("Call %d: %s", i, call.FunctionName)
		if call.FunctionName == "after_retrieve" {
			found = true
			break
		}
	}
	
	assert.True(t, found, "after_retrieve hook should have been called")
}

func TestMMU_ConsolidateLTM(t *testing.T) {
	// Setup
	mmu, _, _, _, ctx := setupTest(t, false)
	
	// Test the placeholder implementation
	err := mmu.ConsolidateLTM(ctx, "test insight")
	assert.NoError(t, err)
}

func TestMMU_EncodeToLTM_WithEmbedding(t *testing.T) {
	// Setup with a vector-capable store
	mmu, ltmStore, _, reasoningEngine, ctx := setupVectorTest(t, true)
	
	// Set up a custom embedding result
	customEmbedding := []float32{0.5, 0.5, 0.5, 0.5, 0.5}
	reasoningEngine.embeddingResults["Test content for embedding"] = customEmbedding
	
	// Test encoding a string that should generate an embedding
	memoryID, err := mmu.EncodeToLTM(ctx, "Test content for embedding")
	require.NoError(t, err)
	
	// Verify the embedding was generated and stored
	query := ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": memoryID,
		},
	}
	records, err := ltmStore.Retrieve(ctx, query)
	require.NoError(t, err)
	require.Len(t, records, 1)
	
	// Verify record contents
	record := records[0]
	assert.Equal(t, "Test content for embedding", record.Content)
	assert.NotNil(t, record.Embedding)
	assert.Equal(t, customEmbedding, record.Embedding)
	
	// Verify the reasoning engine was called
	foundEmbeddingCall := false
	for _, call := range reasoningEngine.calls {
		if call.FunctionName == "GenerateEmbeddings" {
			foundEmbeddingCall = true
			if args, ok := call.Args[0].([]string); ok {
				assert.Contains(t, args, "Test content for embedding")
			}
		}
	}
	assert.True(t, foundEmbeddingCall, "GenerateEmbeddings should have been called")
}

func TestMMU_EncodeToLTM_NoEmbeddingForNonVectorStore(t *testing.T) {
	// Setup with a non-vector-capable store
	mmu, ltmStore, _, reasoningEngine, ctx := setupVectorTest(t, false)
	
	// Test encoding a string that should NOT generate an embedding
	memoryID, err := mmu.EncodeToLTM(ctx, "Test content without embedding")
	require.NoError(t, err)
	
	// Verify the record was stored without an embedding
	query := ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": memoryID,
		},
	}
	records, err := ltmStore.Retrieve(ctx, query)
	require.NoError(t, err)
	require.Len(t, records, 1)
	
	// Verify record contents
	record := records[0]
	assert.Equal(t, "Test content without embedding", record.Content)
	assert.Empty(t, record.Embedding, "Embedding should not be generated for non-vector stores")
	
	// Verify the reasoning engine was NOT called
	foundEmbeddingCall := false
	for _, call := range reasoningEngine.calls {
		if call.FunctionName == "GenerateEmbeddings" {
			foundEmbeddingCall = true
		}
	}
	assert.False(t, foundEmbeddingCall, "GenerateEmbeddings should not have been called")
}

func TestMMU_RetrieveFromLTM_SemanticSearch(t *testing.T) {
	// Setup with a vector-capable store
	mmu, ltmStore, _, reasoningEngine, ctx := setupVectorTest(t, true)
	
	// Store a test record with an embedding
	testRecord := ltm.MemoryRecord{
		ID:          "test-semantic",
		EntityID:    "test-entity",
		Content:     "Semantic search test record",
		AccessLevel: entity.SharedWithinEntity,
		Embedding:   []float32{0.1, 0.2, 0.3, 0.4, 0.5},
	}
	_, err := ltmStore.Store(ctx, testRecord)
	require.NoError(t, err)
	
	// Set up a custom embedding result for the query
	customQueryEmbedding := []float32{0.15, 0.25, 0.35, 0.45, 0.55}
	reasoningEngine.embeddingResults["semantic search"] = customQueryEmbedding
	
	// Test semantic retrieval
	options := DefaultRetrievalOptions()
	options.Strategy = "semantic"
	results, err := mmu.RetrieveFromLTM(ctx, "semantic search", options)
	require.NoError(t, err)
	
	// Verify we got results
	assert.NotEmpty(t, results)
	
	// Verify query embedding was generated
	foundEmbeddingCall := false
	for _, call := range reasoningEngine.calls {
		if call.FunctionName == "GenerateEmbeddings" {
			foundEmbeddingCall = true
			if args, ok := call.Args[0].([]string); ok && len(args) > 0 {
				assert.Equal(t, "semantic search", args[0])
			}
		}
	}
	assert.True(t, foundEmbeddingCall, "GenerateEmbeddings should have been called")
}

func TestMMU_WorkingMemoryOverflow(t *testing.T) {
	// Setup with a low working memory limit
	mmu, _, _, _, ctx := setupVectorTest(t, true)
	
	// Initially, working memory should be empty
	assert.Empty(t, mmu.workingMemory)
	
	// Add records to working memory directly to test overflow
	for i := 0; i < 6; i++ {
		record := ltm.MemoryRecord{
			ID:       fmt.Sprintf("test-%d", i),
			EntityID: "test-entity",
			Content:  fmt.Sprintf("Working memory record %d", i),
		}
		mmu.workingMemory = append(mmu.workingMemory, record)
	}
	
	// Now trigger the overflow management
	mmu.ManageWorkingMemoryOverflow(ctx)
	
	// Verify overflow was managed (records were evicted)
	assert.Less(t, len(mmu.workingMemory), 6, "Working memory should have evicted some records")
	assert.Equal(t, 3, len(mmu.workingMemory), "Working memory should have half the records after LRU eviction")
}