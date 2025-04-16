package pgvector

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDimension = 5

func skipIfNoPgvector(t *testing.T) string {
	// Skip the test if no PGVECTOR_TEST_URL environment variable is provided
	pgvectorURL := os.Getenv("PGVECTOR_TEST_URL")
	if pgvectorURL == "" {
		t.Skip("Skipping pgvector tests: PGVECTOR_TEST_URL environment variable not set")
	}
	return pgvectorURL
}

func createTestRecord(entityID string, userID string, content string) ltm.MemoryRecord {
	return ltm.MemoryRecord{
		ID:          uuid.New().String(),
		EntityID:    entity.EntityID(entityID),
		UserID:      userID,
		AccessLevel: entity.PrivateToUser,
		Content:     content,
		Metadata: map[string]interface{}{
			"test_key": "test_value",
			"source":   "test",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		// For testing, we'll set a fixed embedding
		Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
	}
}

func setupTestAdapter(t *testing.T) (*PgvectorAdapter, context.Context) {
	pgvectorURL := skipIfNoPgvector(t)
	
	// Create a context with a test entity
	entityID := entity.EntityID("test-entity")
	userID := "test-user"
	entityCtx := entity.NewContext(entityID, userID)
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	// Create a random table name for tests to avoid conflicts
	tableName := "test_" + uuid.New().String()[:8]
	
	config := PgvectorConfig{
		ConnectionString: pgvectorURL,
		TableName:        tableName,
		DimensionSize:    testDimension,
		DistanceMetric:   "cosine",
	}
	
	adapter, err := NewPgvectorAdapter(ctx, config)
	require.NoError(t, err)
	require.NotNil(t, adapter)
	
	// Cleanup function to run after the test
	t.Cleanup(func() {
		// Drop the test table
		if adapter != nil && adapter.db != nil {
			_, err := adapter.db.Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			if err != nil {
				t.Logf("Failed to drop test table: %v", err)
			}
			adapter.Close()
		}
	})
	
	return adapter, ctx
}

func TestPgvectorAdapter_Store(t *testing.T) {
	// Skip if no PgVector connection
	skipIfNoPgvector(t)
	
	// Setup
	adapter, ctx := setupTestAdapter(t)
	
	entityID := uuid.New().String()
	userID := "test-user-1"
	
	// Test storing a single record
	record := createTestRecord(entityID, userID, "Test content for single record")
	id, err := adapter.Store(ctx, record)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	
	// Test storing multiple records
	for i := 1; i <= 3; i++ {
		rec := createTestRecord(entityID, userID, "Test content "+string(rune('0'+i)))
		id, err := adapter.Store(ctx, rec)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
	}
}

func TestPgvectorAdapter_Retrieve_Semantic(t *testing.T) {
	// Skip if no PgVector connection
	skipIfNoPgvector(t)
	
	// Setup
	adapter, _ := setupTestAdapter(t)
	
	entityIDStr := uuid.New().String()
	entityID := entity.EntityID(entityIDStr)
	userID := "test-user-1"
	
	// Create context with specific entity
	entityCtx := entity.NewContext(entityID, userID)
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	// Store several records with different contents
	testRecords := []ltm.MemoryRecord{
		createTestRecord(string(entityID), userID, "Apple is a fruit"),
		createTestRecord(string(entityID), userID, "Banana is yellow"),
		createTestRecord(string(entityID), userID, "Cherry is red"),
		createTestRecord(string(entityID), userID, "Orange is citrus"),
		createTestRecord(string(entityID), userID, "Strawberry is sweet"),
	}
	
	// Define custom embeddings that will allow us to test semantic search
	testRecords[0].Embedding = []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	testRecords[1].Embedding = []float32{0.2, 0.3, 0.4, 0.5, 0.6}
	testRecords[2].Embedding = []float32{0.3, 0.4, 0.5, 0.6, 0.7}
	testRecords[3].Embedding = []float32{0.4, 0.5, 0.6, 0.7, 0.8}
	testRecords[4].Embedding = []float32{0.5, 0.6, 0.7, 0.8, 0.9}
	
	for _, rec := range testRecords {
		_, err := adapter.Store(ctx, rec)
		require.NoError(t, err)
	}
	
	// Test semantic search
	queryVector := []float32{0.3, 0.4, 0.5, 0.6, 0.7} // Should match "Cherry is red" best
	
	// Use updated query structure
	query := ltm.LTMQuery{
		Embedding: queryVector,
		Limit:     3,
	}
	
	results, err := adapter.Retrieve(ctx, query)
	assert.NoError(t, err)
	require.NotEmpty(t, results)
	require.LessOrEqual(t, len(results), 3)
	
	// The closest vector should be the one for "Cherry is red"
	found := false
	for _, result := range results {
		if result.Content == "Cherry is red" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find 'Cherry is red' in results")
	
	// Test limit
	query.Limit = 2
	results, err = adapter.Retrieve(ctx, query)
	assert.NoError(t, err)
	require.NotEmpty(t, results)
	require.LessOrEqual(t, len(results), 2)
}

func TestPgvectorAdapter_Retrieve_Filtering(t *testing.T) {
	// Skip if no PgVector connection
	skipIfNoPgvector(t)
	
	// Setup
	adapter, _ := setupTestAdapter(t)
	
	// Create two different entities
	entityID1Str := uuid.New().String()
	entityID2Str := uuid.New().String()
	entityID1 := entity.EntityID(entityID1Str)
	entityID2 := entity.EntityID(entityID2Str)
	userID1 := "test-user-1"
	userID2 := "test-user-2"
	
	// Create entity contexts
	entityCtx1 := entity.NewContext(entityID1, userID1)
	ctx1 := entity.ContextWithEntity(context.Background(), entityCtx1)
	
	entityCtx2 := entity.NewContext(entityID2, userID1)
	ctx2 := entity.ContextWithEntity(context.Background(), entityCtx2)
	
	// Store records for different entities and users
	testRecords := []ltm.MemoryRecord{
		createTestRecord(string(entityID1), userID1, "Entity1 User1 Record1"),
		createTestRecord(string(entityID1), userID1, "Entity1 User1 Record2"),
		createTestRecord(string(entityID1), userID2, "Entity1 User2 Record1"),
		createTestRecord(string(entityID2), userID1, "Entity2 User1 Record1"),
		createTestRecord(string(entityID2), userID2, "Entity2 User2 Record1"),
	}
	
	// Set different access levels
	testRecords[0].AccessLevel = entity.PrivateToUser
	testRecords[1].AccessLevel = entity.SharedWithinEntity
	testRecords[2].AccessLevel = entity.PrivateToUser
	testRecords[3].AccessLevel = entity.PrivateToUser
	testRecords[4].AccessLevel = entity.SharedWithinEntity
	
	// Store records with their appropriate entity contexts
	for i, rec := range testRecords {
		var storeCtx context.Context
		if i < 3 {
			// Entity1 records
			storeCtx = ctx1
		} else {
			// Entity2 records
			storeCtx = ctx2
		}
		_, err := adapter.Store(storeCtx, rec)
		require.NoError(t, err)
	}
	
	// Test retrieving Entity1 records
	query := ltm.LTMQuery{
		Filters: map[string]interface{}{
			"access_level": entity.PrivateToUser,
		},
	}
	
	results, err := adapter.Retrieve(ctx1, query)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
	for _, rec := range results {
		assert.Equal(t, entityID1, rec.EntityID)
		assert.Equal(t, entity.PrivateToUser, rec.AccessLevel)
	}
	
	// Test retrieving Entity1 records with user filter
	query = ltm.LTMQuery{
		Filters: map[string]interface{}{
			"user_id": userID1,
		},
	}
	
	results, err = adapter.Retrieve(ctx1, query)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
	for _, rec := range results {
		assert.Equal(t, entityID1, rec.EntityID)
		assert.Equal(t, userID1, rec.UserID)
	}
	
	// Test filtering by EntityID and AccessLevel
	query = ltm.LTMQuery{
		Filters: map[string]interface{}{
			"access_level": entity.SharedWithinEntity,
		},
	}
	
	results, err = adapter.Retrieve(ctx1, query)
	assert.NoError(t, err)
	if assert.Equal(t, 1, len(results)) && len(results) > 0 {
		assert.Equal(t, entity.SharedWithinEntity, results[0].AccessLevel)
		assert.Equal(t, entityID1, results[0].EntityID)
	}
	
	// Test entity isolation (Entity2 context should only see Entity2 records)
	query = ltm.LTMQuery{
		Filters: map[string]interface{}{
			"user_id": userID1,
		},
	}
	
	results, err = adapter.Retrieve(ctx2, query)
	assert.NoError(t, err)
	if assert.Equal(t, 1, len(results)) && len(results) > 0 {
		assert.Equal(t, entityID2, results[0].EntityID)
		assert.Equal(t, userID1, results[0].UserID)
	}
}

func TestPgvectorAdapter_Retrieve_ByID(t *testing.T) {
	// Skip if no PgVector connection
	skipIfNoPgvector(t)
	
	// Setup
	adapter, _ := setupTestAdapter(t)
	
	entityIDStr := uuid.New().String()
	entityID := entity.EntityID(entityIDStr)
	userID := "test-user-1"
	
	// Create entity context
	entityCtx := entity.NewContext(entityID, userID)
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	// Store a record and remember its ID
	record := createTestRecord(string(entityID), userID, "Test record for ID lookup")
	id, err := adapter.Store(ctx, record)
	require.NoError(t, err)
	require.NotEmpty(t, id)
	
	// Test retrieval by ID
	query := ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"id": id,
		},
	}
	
	results, err := adapter.Retrieve(ctx, query)
	assert.NoError(t, err)
	if assert.Equal(t, 1, len(results)) && len(results) > 0 {
		assert.Equal(t, id, results[0].ID)
		assert.Equal(t, "Test record for ID lookup", results[0].Content)
	}
	
	// Create a different entity context and verify isolation
	otherEntityID := entity.EntityID("other-entity")
	otherEntityCtx := entity.NewContext(otherEntityID, userID)
	otherCtx := entity.ContextWithEntity(context.Background(), otherEntityCtx)
	
	// Attempt to retrieve with different entity context
	results, err = adapter.Retrieve(otherCtx, query)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(results), "Different entity should not see the record")
}

func TestPgvectorAdapter_Update(t *testing.T) {
	// Skip if no PgVector connection
	skipIfNoPgvector(t)
	
	// Setup
	adapter, _ := setupTestAdapter(t)
	
	entityIDStr := uuid.New().String()
	entityID := entity.EntityID(entityIDStr)
	userID := "test-user-1"
	
	// Create entity context
	entityCtx := entity.NewContext(entityID, userID)
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	// Store a record
	record := createTestRecord(string(entityID), userID, "Original content")
	id, err := adapter.Store(ctx, record)
	require.NoError(t, err)
	
	// Get the record to update it
	query := ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"id": id,
		},
	}
	
	results, err := adapter.Retrieve(ctx, query)
	assert.NoError(t, err)
	require.Equal(t, 1, len(results))
	
	recordToUpdate := results[0]
	recordToUpdate.Content = "Updated content"
	recordToUpdate.Metadata["updated"] = true
	recordToUpdate.Embedding = []float32{0.9, 0.8, 0.7, 0.6, 0.5}
	
	err = adapter.Update(ctx, recordToUpdate)
	assert.NoError(t, err)
	
	// Retrieve and verify the update
	updatedResults, err := adapter.Retrieve(ctx, query)
	assert.NoError(t, err)
	require.Equal(t, 1, len(updatedResults))
	assert.Equal(t, "Updated content", updatedResults[0].Content)
	assert.Equal(t, true, updatedResults[0].Metadata["updated"])
	assert.InDeltaSlice(t, []float32{0.9, 0.8, 0.7, 0.6, 0.5}, updatedResults[0].Embedding, 0.01)
}

func TestPgvectorAdapter_Delete(t *testing.T) {
	// Skip if no PgVector connection
	skipIfNoPgvector(t)
	
	// Setup
	adapter, _ := setupTestAdapter(t)
	
	entityIDStr := uuid.New().String()
	entityID := entity.EntityID(entityIDStr)
	userID := "test-user-1"
	
	// Create entity context
	entityCtx := entity.NewContext(entityID, userID)
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	// Store a record
	record := createTestRecord(string(entityID), userID, "Content to be deleted")
	id, err := adapter.Store(ctx, record)
	require.NoError(t, err)
	
	// Confirm it's stored
	query := ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"id": id,
		},
	}
	
	results, err := adapter.Retrieve(ctx, query)
	assert.NoError(t, err)
	require.Equal(t, 1, len(results))
	
	// Delete the record
	err = adapter.Delete(ctx, id)
	assert.NoError(t, err)
	
	// Confirm it's deleted
	results, err = adapter.Retrieve(ctx, query)
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestPgvectorAdapter_EdgeCases(t *testing.T) {
	// Skip if no PgVector connection
	skipIfNoPgvector(t)
	
	// Setup
	adapter, _ := setupTestAdapter(t)
	
	entityIDStr := uuid.New().String()
	entityID := entity.EntityID(entityIDStr)
	userID := "test-user-1"
	
	// Create entity context
	entityCtx := entity.NewContext(entityID, userID)
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	// Test dimension mismatch
	record := createTestRecord(string(entityID), userID, "Record with wrong embedding dimension")
	record.Embedding = []float32{0.1, 0.2, 0.3} // Wrong dimension
	_, err := adapter.Store(ctx, record)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedding dimension mismatch")
	
	// Test empty embedding
	record = createTestRecord(string(entityID), userID, "Record with empty embedding")
	record.Embedding = []float32{} // Empty embedding
	_, err = adapter.Store(ctx, record)
	assert.Error(t, err)
	
	// Test nil embedding
	record = createTestRecord(string(entityID), userID, "Record with nil embedding")
	record.Embedding = nil // Nil embedding
	_, err = adapter.Store(ctx, record)
	assert.Error(t, err)
	
	// Test with correct embedding
	record = createTestRecord(string(entityID), userID, "Record with correct embedding")
	id, err := adapter.Store(ctx, record)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	
	// Test deleting non-existent record
	err = adapter.Delete(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Equal(t, ErrRecordNotFound, err)
	
	// Test updating non-existent record
	record.ID = "non-existent-id"
	err = adapter.Update(ctx, record)
	assert.Error(t, err)
	assert.Equal(t, ErrRecordNotFound, err)
	
	// Test retrieving with missing entity context
	missingCtx := context.Background() // No entity context
	query := ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"id": id,
		},
	}
	_, err = adapter.Retrieve(missingCtx, query)
	assert.Error(t, err)
	assert.Equal(t, entity.ErrMissingEntityContext, err)
}

func TestPgvectorAdapter_SupportsVectorSearch(t *testing.T) {
	// Skip if no PgVector connection
	skipIfNoPgvector(t)
	
	// Setup
	adapter, _ := setupTestAdapter(t)
	defer adapter.Close()
	
	assert.True(t, adapter.SupportsVectorSearch())
}