package mmu

import (
	"context"
	"testing"
	"time"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
func setupTest(t *testing.T, enableLuaHooks bool) (*MMUI, *mock.MockStore, *mockScriptEngine, context.Context) {
	// Create mock dependencies
	ltmStore := mock.NewMockStore()
	scriptEngine := newMockScriptEngine()
	
	// Create MMU with configuration
	mmu := NewMMU(ltmStore, scriptEngine, Config{
		EnableLuaHooks: enableLuaHooks,
	})
	
	// Create a context with entity information
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	return mmu, ltmStore, scriptEngine, ctx
}

func TestMMU_EncodeToLTM_String(t *testing.T) {
	// Setup
	mmu, ltmStore, _, ctx := setupTest(t, false)
	
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
	mmu, ltmStore, _, ctx := setupTest(t, false)
	
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
	mmu, _, _, _ := setupTest(t, false)
	
	// Should fail without entity context
	_, err := mmu.EncodeToLTM(context.Background(), "This should fail")
	assert.Error(t, err)
	assert.Equal(t, entity.ErrMissingEntityContext, err)
}

func TestMMU_RetrieveFromLTM_StringQuery(t *testing.T) {
	// Setup
	mmu, ltmStore, _, ctx := setupTest(t, false)
	
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
	mmu, ltmStore, _, ctx := setupTest(t, false)
	
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
	mmu, ltmStore, scriptEngine, ctx := setupTest(t, true)
	
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
	mmu, _, _, ctx := setupTest(t, false)
	
	// Test the placeholder implementation
	err := mmu.ConsolidateLTM(ctx, "test insight")
	assert.NoError(t, err)
}