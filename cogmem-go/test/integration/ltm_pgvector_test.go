//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/vector/pgvector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgVectorLTMIntegration(t *testing.T) {
	// Skip this test if not running in integration test mode
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Get database connection string from environment variable or use default
	pgvectorURL := os.Getenv("PGVECTOR_TEST_URL")
	if pgvectorURL == "" {
		pgvectorURL = os.Getenv("TEST_DB_URL")
		if pgvectorURL == "" {
			pgvectorURL = "postgres://postgres:postgres@localhost:5432/cogmem_test?sslmode=disable"
		}
	}

	// Create a random table name for tests to avoid conflicts
	tableName := "test_" + uuid.New().String()[:8]

	// Set up a temporary connection to create the pgvector extension
	tempConfig, err := pgxpool.ParseConfig(pgvectorURL)
	require.NoError(t, err, "Failed to parse connection string")
	
	tempCtx, tempCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer tempCancel()
	
	tempPool, err := pgxpool.NewWithConfig(tempCtx, tempConfig)
	require.NoError(t, err, "Failed to create connection pool")
	defer tempPool.Close()
	
	// Enable pgvector extension if not already enabled
	_, err = tempPool.Exec(tempCtx, "CREATE EXTENSION IF NOT EXISTS vector")
	require.NoError(t, err, "Failed to enable vector extension")
	
	// Set up the adapter
	ctx := context.Background()
	pgConfig := pgvector.PgvectorConfig{
		ConnectionString: pgvectorURL,
		TableName:        tableName,
		DimensionSize:    5, // Small dimension for tests
		DistanceMetric:   "cosine",
	}

	adapter, err := pgvector.NewPgvectorAdapter(ctx, pgConfig)
	require.NoError(t, err)
	require.NotNil(t, adapter)

	// Clean up after the test
	defer func() {
		if adapter != nil && adapter.DB() != nil {
			_, err := adapter.DB().Exec(ctx, "DROP TABLE IF EXISTS "+tableName)
			if err != nil {
				t.Logf("Failed to drop test table: %v", err)
			}
			adapter.Close()
		}
	}()

	// Create entity data
	entityID := entity.EntityID("test-entity-" + uuid.New().String())
	userID := "test-user-1"

	// Test storing and retrieving records
	t.Run("Basic Store and Retrieve", func(t *testing.T) {
		// Create a test record
		record := ltm.MemoryRecord{
			ID:          uuid.New().String(),
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.PrivateToUser,
			Content:     "This is a test memory for pgvector",
			Metadata: map[string]interface{}{
				"test_key": "test_value",
				"source":   "integration_test",
			},
			Embedding:  []float32{0.1, 0.2, 0.3, 0.4, 0.5},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Store the record
		id, err := adapter.Store(ctx, record)
		assert.NoError(t, err)
		assert.Equal(t, record.ID, id)

		// Retrieve the record by ID
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"id": id,
			},
		}

		results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		require.Len(t, results, 1)
		
		// Verify the retrieved record
		retrieved := results[0]
		assert.Equal(t, record.ID, retrieved.ID)
		assert.Equal(t, record.EntityID, retrieved.EntityID)
		assert.Equal(t, record.UserID, retrieved.UserID)
		assert.Equal(t, record.AccessLevel, retrieved.AccessLevel)
		assert.Equal(t, record.Content, retrieved.Content)
		assert.Equal(t, "test_value", retrieved.Metadata["test_key"])
		assert.Equal(t, "integration_test", retrieved.Metadata["source"])
	})

	// Test semantic search
	t.Run("Semantic Search", func(t *testing.T) {
		// Create multiple records with different embeddings
		records := []ltm.MemoryRecord{
			{
				ID:          uuid.New().String(),
				EntityID:    entityID,
				UserID:      userID,
				AccessLevel: entity.PrivateToUser,
				Content:     "Apple is a fruit",
				Metadata:    map[string]interface{}{"type": "fruit"},
				Embedding:   []float32{0.1, 0.2, 0.3, 0.4, 0.5},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          uuid.New().String(),
				EntityID:    entityID,
				UserID:      userID,
				AccessLevel: entity.PrivateToUser,
				Content:     "Banana is yellow",
				Metadata:    map[string]interface{}{"type": "fruit"},
				Embedding:   []float32{0.2, 0.3, 0.4, 0.5, 0.6},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				ID:          uuid.New().String(),
				EntityID:    entityID,
				UserID:      userID,
				AccessLevel: entity.PrivateToUser,
				Content:     "Cherry is red",
				Metadata:    map[string]interface{}{"type": "fruit"},
				Embedding:   []float32{0.3, 0.4, 0.5, 0.6, 0.7},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		// Store all records
		for _, record := range records {
			_, err := adapter.Store(ctx, record)
			require.NoError(t, err)
		}

		// Perform semantic search
		query := ltm.LTMQuery{
			Filters: map[string]interface{}{
				"entity_id": entityID,
			},
			Embedding: []float32{0.3, 0.4, 0.5, 0.6, 0.7}, // Closest to "Cherry is red"
			Limit:     2,
		}

		results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		require.LessOrEqual(t, len(results), 2)

		// Verify that "Cherry is red" is in the results (should be the closest match)
		found := false
		for _, result := range results {
			if result.Content == "Cherry is red" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find 'Cherry is red' in semantic search results")
	})

	// Test metadata filtering
	t.Run("Metadata Filtering", func(t *testing.T) {
		// Create records with different metadata
		records := []ltm.MemoryRecord{
			{
				ID:          uuid.New().String(),
				EntityID:    entityID,
				UserID:      userID,
				AccessLevel: entity.PrivateToUser,
				Content:     "Record with tag1",
				Metadata: map[string]interface{}{
					"tag": "tag1",
				},
				Embedding: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          uuid.New().String(),
				EntityID:    entityID,
				UserID:      userID,
				AccessLevel: entity.PrivateToUser,
				Content:     "Record with tag2",
				Metadata: map[string]interface{}{
					"tag": "tag2",
				},
				Embedding: []float32{0.2, 0.3, 0.4, 0.5, 0.6},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		// Store all records
		for _, record := range records {
			_, err := adapter.Store(ctx, record)
			require.NoError(t, err)
		}

		// Query by metadata
		query := ltm.LTMQuery{
			Filters: map[string]interface{}{
				"entity_id": entityID,
				"tag":       "tag1",
			},
		}

		results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		
		// Verify filtering
		for _, result := range results {
			assert.Equal(t, "tag1", result.Metadata["tag"])
			assert.Equal(t, "Record with tag1", result.Content)
		}
	})

	// Test updates
	t.Run("Update Records", func(t *testing.T) {
		// Create a record
		record := ltm.MemoryRecord{
			ID:          uuid.New().String(),
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.PrivateToUser,
			Content:     "Original content",
			Metadata:    map[string]interface{}{},
			Embedding:   []float32{0.1, 0.2, 0.3, 0.4, 0.5},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Store it
		_, err := adapter.Store(ctx, record)
		require.NoError(t, err)

		// Update it
		record.Content = "Updated content"
		record.Embedding = []float32{0.9, 0.8, 0.7, 0.6, 0.5}
		record.Metadata = map[string]interface{}{
			"updated": true,
		}

		err = adapter.Update(ctx, record)
		assert.NoError(t, err)

		// Retrieve and verify the update
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"id": record.ID,
			},
		}

		results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		require.Len(t, results, 1)
		
		updated := results[0]
		assert.Equal(t, "Updated content", updated.Content)
		assert.Equal(t, true, updated.Metadata["updated"])
		assert.InDeltaSlice(t, []float32{0.9, 0.8, 0.7, 0.6, 0.5}, updated.Embedding, 0.01)
	})

	// Test deletion
	t.Run("Delete Records", func(t *testing.T) {
		// Create a record
		record := ltm.MemoryRecord{
			ID:          uuid.New().String(),
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.PrivateToUser,
			Content:     "Content to be deleted",
			Metadata:    map[string]interface{}{},
			Embedding:   []float32{0.1, 0.2, 0.3, 0.4, 0.5},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Store it
		_, err := adapter.Store(ctx, record)
		require.NoError(t, err)

		// Delete it
		err = adapter.Delete(ctx, record.ID)
		assert.NoError(t, err)

		// Verify it's gone
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"id": record.ID,
			},
		}

		results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		assert.Empty(t, results)
	})
}