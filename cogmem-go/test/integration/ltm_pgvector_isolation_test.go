package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/vector/pgvector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPgVectorEntityIsolation tests that entity isolation works properly
func TestPgVectorEntityIsolation(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TESTS=true to run.")
	}

	connectionString := os.Getenv("POSTGRES_URL")
	if connectionString == "" {
		t.Skip("Skipping pgvector test. Set POSTGRES_URL environment variable to run.")
	}

	// Create a unique table name for this test
	tableName := "test_isolation_" + time.Now().Format("20060102150405")

	// Create pgvector config
	pgvectorConfig := pgvector.PgvectorConfig{
		ConnectionString: connectionString,
		TableName:        tableName,
		DimensionSize:    4, // Small dimension for testing
		DistanceMetric:   "cosine",
	}

	// Initialize pgvector adapter
	ctx := context.Background()
	adapter, err := pgvector.NewPgvectorAdapter(ctx, pgvectorConfig)
	require.NoError(t, err, "Failed to create pgvector adapter")
	defer adapter.Close()

	// Create test data for two different entities
	entityID1 := entity.EntityID("test-entity-1")
	entityID2 := entity.EntityID("test-entity-2")
	userID := "test-user"

	// Create entity contexts
	entityCtx1 := entity.NewContext(entityID1, userID)
	entityCtx2 := entity.NewContext(entityID2, userID)

	// Create contexts with entity information
	ctx1 := entity.ContextWithEntity(ctx, entityCtx1)
	ctx2 := entity.ContextWithEntity(ctx, entityCtx2)

	// Create memory records
	record1 := ltm.MemoryRecord{
		EntityID:    entityID1,
		UserID:      userID,
		AccessLevel: entity.SharedWithinEntity,
		Content:     "This is a test memory for entity 1",
		Embedding:   []float32{0.1, 0.2, 0.3, 0.4},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]interface{}{"test": "value"},
	}

	record2 := ltm.MemoryRecord{
		EntityID:    entityID2,
		UserID:      userID,
		AccessLevel: entity.SharedWithinEntity,
		Content:     "This is a test memory for entity 2",
		Embedding:   []float32{0.5, 0.6, 0.7, 0.8},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]interface{}{"test": "value"},
	}

	// Store the records
	record1ID, err := adapter.Store(ctx1, record1)
	require.NoError(t, err, "Failed to store record for entity 1")
	record1.ID = record1ID

	record2ID, err := adapter.Store(ctx2, record2)
	require.NoError(t, err, "Failed to store record for entity 2")
	record2.ID = record2ID

	// Test 1: Test keyword-based lookup with entity isolation
	t.Run("Keyword Lookup with Entity Isolation", func(t *testing.T) {
		// Create a query that would match both records
		query := ltm.LTMQuery{
			Text:  "test memory",
			Limit: 10,
		}

		// Perform search with entity context 1
		resultsEntity1, err := adapter.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve records for entity 1")

		// Perform search with entity context 2
		resultsEntity2, err := adapter.Retrieve(ctx2, query)
		require.NoError(t, err, "Failed to retrieve records for entity 2")

		// Verify that only records for the correct entity are returned
		assert.Equal(t, 1, len(resultsEntity1), "Entity 1 should only see its own records")
		assert.Equal(t, 1, len(resultsEntity2), "Entity 2 should only see its own records")

		if len(resultsEntity1) > 0 {
			assert.Equal(t, entityID1, resultsEntity1[0].EntityID, "Record should be for entity 1")
			assert.Equal(t, "This is a test memory for entity 1", resultsEntity1[0].Content)
		}

		if len(resultsEntity2) > 0 {
			assert.Equal(t, entityID2, resultsEntity2[0].EntityID, "Record should be for entity 2")
			assert.Equal(t, "This is a test memory for entity 2", resultsEntity2[0].Content)
		}
	})

	// Test 2: Test vector-based lookup with entity isolation
	t.Run("Vector Lookup with Entity Isolation", func(t *testing.T) {
		// Create a vector query that would match both records
		query := ltm.LTMQuery{
			Embedding: []float32{0.3, 0.3, 0.3, 0.3}, // Somewhat in the middle
			Limit:     10,
		}

		// Perform search with entity context 1
		resultsEntity1, err := adapter.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve records for entity 1")

		// Perform search with entity context 2
		resultsEntity2, err := adapter.Retrieve(ctx2, query)
		require.NoError(t, err, "Failed to retrieve records for entity 2")

		// Verify that only records for the correct entity are returned
		assert.Equal(t, 1, len(resultsEntity1), "Entity 1 should only see its own records")
		assert.Equal(t, 1, len(resultsEntity2), "Entity 2 should only see its own records")

		if len(resultsEntity1) > 0 {
			assert.Equal(t, entityID1, resultsEntity1[0].EntityID, "Record should be for entity 1")
		}

		if len(resultsEntity2) > 0 {
			assert.Equal(t, entityID2, resultsEntity2[0].EntityID, "Record should be for entity 2")
		}
	})

	// Test 3: Test ID-based lookup with entity isolation
	t.Run("ID Lookup with Entity Isolation", func(t *testing.T) {
		// Create queries to look up records by ID
		query1 := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"id": record1ID,
			},
		}

		query2 := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"id": record2ID,
			},
		}

		// Entity 1 looks up its own record - should succeed
		resultsOwn, err := adapter.Retrieve(ctx1, query1)
		require.NoError(t, err, "Failed to retrieve own record")
		assert.Equal(t, 1, len(resultsOwn), "Entity should see its own record")

		// Entity 1 looks up entity 2's record - should return empty
		resultsOther, err := adapter.Retrieve(ctx1, query2)
		require.NoError(t, err, "No error expected for cross-entity lookup")
		assert.Equal(t, 0, len(resultsOther), "Entity should not see other entity's records")

		// Entity 2 looks up entity 1's record - should return empty
		resultsOther, err = adapter.Retrieve(ctx2, query1)
		require.NoError(t, err, "No error expected for cross-entity lookup")
		assert.Equal(t, 0, len(resultsOther), "Entity should not see other entity's records")
	})
}