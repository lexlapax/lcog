//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/sqlstore/postgres"
)

// TestPostgresSQLStoreLTMOperations tests the SQLStore PostgreSQL adapter functionality.
func TestPostgresSQLStoreLTMOperations(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL SQLStore integration test. Set INTEGRATION_TESTS=true to run.")
	}

	// Get database connection string from environment variable or use default
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/cogmem_test?sslmode=disable"
	}

	// Create a connection pool
	config, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err, "Failed to parse connection string")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err, "Failed to create connection pool")
	defer pool.Close()

	// Ping to verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "Failed to ping database")

	// Create memory_records table if it doesn't exist
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS memory_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_id TEXT NOT NULL,
			user_id TEXT,
			access_level INTEGER NOT NULL,
			content TEXT NOT NULL,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			CONSTRAINT memory_records_entity_id_idx UNIQUE (id, entity_id)
		);
		
		-- Add index on entity_id for performance
		CREATE INDEX IF NOT EXISTS idx_memory_records_entity_id ON memory_records(entity_id);
		
		-- Add index on metadata for JSON querying
		CREATE INDEX IF NOT EXISTS idx_memory_records_metadata ON memory_records USING GIN (metadata);
	`)
	require.NoError(t, err, "Failed to create memory_records table")
	
	// Clean up any existing test data
	_, err = pool.Exec(ctx, "DELETE FROM memory_records")
	require.NoError(t, err, "Failed to clean up test data")

	// Create the adapter
	store := postgres.NewPostgresStore(pool)
	require.NotNil(t, store, "Failed to create PostgreSQL store")

	// Create test entities and contexts
	entity1ID := entity.EntityID("test-entity-1")
	entity2ID := entity.EntityID("test-entity-2")
	user1ID := "test-user-1"
	user2ID := "test-user-2"
	
	ctx1 := entity.ContextWithEntity(context.Background(), entity.NewContext(entity1ID, user1ID))
	ctx2 := entity.ContextWithEntity(context.Background(), entity.NewContext(entity2ID, user2ID))

	t.Run("Store and Retrieve Basic Record", func(t *testing.T) {
		// Create a test record
		record := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Test content for SQLStore",
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
				"nested": map[string]interface{}{
					"inner": "value",
				},
			},
		}

		// Store the record
		id, err := store.Store(ctx1, record)
		require.NoError(t, err, "Failed to store record")
		require.NotEmpty(t, id, "Record ID should not be empty")

		// Retrieve the record by ID
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}

		records, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve record")
		require.Len(t, records, 1, "Should retrieve exactly one record")
		
		// Verify retrieved record matches original
		retrieved := records[0]
		assert.Equal(t, id, retrieved.ID, "ID should match")
		assert.Equal(t, entity1ID, retrieved.EntityID, "EntityID should match")
		assert.Equal(t, user1ID, retrieved.UserID, "UserID should match")
		assert.Equal(t, entity.SharedWithinEntity, retrieved.AccessLevel, "AccessLevel should match")
		assert.Equal(t, "Test content for SQLStore", retrieved.Content, "Content should match")
		assert.Equal(t, "value1", retrieved.Metadata["key1"], "Metadata key1 should match")
		assert.Equal(t, float64(123), retrieved.Metadata["key2"], "Metadata key2 should match")
		assert.Equal(t, true, retrieved.Metadata["key3"], "Metadata key3 should match")
		
		// Verify nested metadata was properly stored and retrieved
		nestedMap, ok := retrieved.Metadata["nested"].(map[string]interface{})
		require.True(t, ok, "Nested metadata should be a map")
		assert.Equal(t, "value", nestedMap["inner"], "Nested metadata value should match")
	})

	t.Run("Entity Isolation", func(t *testing.T) {
		// Create records for different entities
		record1 := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content for entity 1",
			Metadata: map[string]interface{}{
				"entity": "1",
			},
		}

		record2 := ltm.MemoryRecord{
			EntityID:    entity2ID,
			UserID:      user2ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content for entity 2",
			Metadata: map[string]interface{}{
				"entity": "2",
			},
		}

		// Store both records
		id1, err := store.Store(ctx1, record1)
		require.NoError(t, err, "Failed to store record for entity 1")

		// Store entity 2's record
		id2, err := store.Store(ctx2, record2)
		require.NoError(t, err, "Failed to store record for entity 2")

		// Verify entity 2 can see its own record
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id2,
			},
		}
		results, err := store.Retrieve(ctx2, query)
		require.NoError(t, err, "Failed to retrieve entity 2's own record")
		require.Len(t, results, 1, "Entity 2 should see its own record")
		assert.Equal(t, "Content for entity 2", results[0].Content, "Content should match")

		// Try to retrieve entity1's record from entity2's context
		query = ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id1,
			},
		}
		results, err = store.Retrieve(ctx2, query)
		require.NoError(t, err, "Query execution should not fail")
		assert.Empty(t, results, "Entity 1's record should not be visible to entity 2")

		// Retrieve all records for entity 1
		results, err = store.Retrieve(ctx1, ltm.LTMQuery{})
		require.NoError(t, err, "Failed to retrieve records for entity 1")
		for _, r := range results {
			assert.Equal(t, entity1ID, r.EntityID, "Retrieved records should belong to entity 1")
		}

		// Retrieve all records for entity 2
		results, err = store.Retrieve(ctx2, ltm.LTMQuery{})
		require.NoError(t, err, "Failed to retrieve records for entity 2")
		for _, r := range results {
			assert.Equal(t, entity2ID, r.EntityID, "Retrieved records should belong to entity 2")
		}
	})

	t.Run("Access Level Control", func(t *testing.T) {
		// Create shared and private records
		sharedRecord := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Shared content within entity",
			Metadata: map[string]interface{}{
				"access": "shared",
			},
		}

		privateRecord := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.PrivateToUser,
			Content:     "Private content for user only",
			Metadata: map[string]interface{}{
				"access": "private",
			},
		}

		// Store both records
		sharedID, err := store.Store(ctx1, sharedRecord)
		require.NoError(t, err, "Failed to store shared record")

		privateID, err := store.Store(ctx1, privateRecord)
		require.NoError(t, err, "Failed to store private record")

		// Create context for different user in same entity
		otherUserCtx := entity.ContextWithEntity(context.Background(), entity.NewContext(entity1ID, "other-user"))

		// Test that original user can see both records
		query := ltm.LTMQuery{
			Filters: map[string]interface{}{
				"access": "shared",
			},
		}
		results, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve shared records")
		assert.NotEmpty(t, results, "Original user should see shared records")

		query = ltm.LTMQuery{
			Filters: map[string]interface{}{
				"access": "private",
			},
		}
		results, err = store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve private records")
		assert.NotEmpty(t, results, "Original user should see private records")

		// Test that other user can only see shared records
		query = ltm.LTMQuery{
			Filters: map[string]interface{}{
				"access": "shared",
			},
		}
		results, err = store.Retrieve(otherUserCtx, query)
		require.NoError(t, err, "Failed to retrieve shared records for other user")
		assert.NotEmpty(t, results, "Other user should see shared records")

		query = ltm.LTMQuery{
			Filters: map[string]interface{}{
				"access": "private",
			},
		}
		results, err = store.Retrieve(otherUserCtx, query)
		require.NoError(t, err, "Failed to retrieve private records for other user")
		assert.Empty(t, results, "Other user should not see private records")

		// Test direct access by ID
		query = ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": privateID,
			},
		}
		results, err = store.Retrieve(otherUserCtx, query)
		require.NoError(t, err, "Failed to query private record by ID")
		assert.Empty(t, results, "Other user should not see private record by ID")

		query = ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": sharedID,
			},
		}
		results, err = store.Retrieve(otherUserCtx, query)
		require.NoError(t, err, "Failed to query shared record by ID")
		assert.NotEmpty(t, results, "Other user should see shared record by ID")
	})

	t.Run("Text Search", func(t *testing.T) {
		// Create records with specific text content
		records := []ltm.MemoryRecord{
			{
				EntityID:    entity1ID,
				UserID:      user1ID,
				AccessLevel: entity.SharedWithinEntity,
				Content:     "Apple pie recipe with cinnamon",
				Metadata: map[string]interface{}{
					"type": "recipe",
				},
			},
			{
				EntityID:    entity1ID,
				UserID:      user1ID,
				AccessLevel: entity.SharedWithinEntity,
				Content:     "Banana bread with walnuts",
				Metadata: map[string]interface{}{
					"type": "recipe",
				},
			},
			{
				EntityID:    entity1ID,
				UserID:      user1ID,
				AccessLevel: entity.SharedWithinEntity,
				Content:     "Cherry chocolate cake",
				Metadata: map[string]interface{}{
					"type": "recipe",
				},
			},
		}

		// Store all records
		for _, record := range records {
			_, err := store.Store(ctx1, record)
			require.NoError(t, err, "Failed to store record")
		}

		// Search for "recipe" in text - should find records
		query := ltm.LTMQuery{
			Text: "recipe",
		}
		results, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to search by text")
		assert.NotEmpty(t, results, "Should find records containing 'recipe'")

		// Search for "banana" in text - should find only one
		query = ltm.LTMQuery{
			Text: "banana",
		}
		results, err = store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to search by text")
		assert.NotEmpty(t, results, "Should find records containing 'banana'")

		foundBanana := false
		for _, result := range results {
			if result.Content == "Banana bread with walnuts" {
				foundBanana = true
				break
			}
		}
		assert.True(t, foundBanana, "Should find the banana record")

		// Combine text search with metadata filter
		query = ltm.LTMQuery{
			Text: "cake",
			Filters: map[string]interface{}{
				"type": "recipe",
			},
		}
		results, err = store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to search by text with metadata filter")
		
		foundCherry := false
		for _, result := range results {
			if result.Content == "Cherry chocolate cake" {
				foundCherry = true
				break
			}
		}
		assert.True(t, foundCherry, "Should find the cherry cake record")
	})

	t.Run("Update Record", func(t *testing.T) {
		// Create and store a record
		record := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Original content",
			Metadata: map[string]interface{}{
				"version": 1,
			},
		}

		// Store the record
		id, err := store.Store(ctx1, record)
		require.NoError(t, err, "Failed to store record")

		// Get original record to capture timestamp
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}
		originalResults, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve original record")
		require.Len(t, originalResults, 1, "Should find original record")
		originalTimestamp := originalResults[0].UpdatedAt

		// Wait to ensure timestamp change is detectable
		time.Sleep(1 * time.Millisecond)

		// Update the record
		updatedRecord := ltm.MemoryRecord{
			ID:          id,
			EntityID:    entity1ID,
			Content:     "Updated content",
			Metadata: map[string]interface{}{
				"version": 2,
				"updated": true,
			},
		}

		err = store.Update(ctx1, updatedRecord)
		require.NoError(t, err, "Failed to update record")

		// Retrieve updated record
		updatedResults, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve updated record")
		require.Len(t, updatedResults, 1, "Should find updated record")
		
		// Verify updates were applied
		updated := updatedResults[0]
		assert.Equal(t, "Updated content", updated.Content, "Content should be updated")
		assert.Equal(t, float64(2), updated.Metadata["version"], "Version should be updated")
		assert.Equal(t, true, updated.Metadata["updated"], "Updated field should be added")
		
		// Verify timestamp was updated
		assert.True(t, updated.UpdatedAt.After(originalTimestamp), "UpdatedAt should be later than original")
	})

	t.Run("Delete Record", func(t *testing.T) {
		// Create and store a record
		record := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content to be deleted",
		}

		// Store the record
		id, err := store.Store(ctx1, record)
		require.NoError(t, err, "Failed to store record")

		// Verify record exists
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}
		results, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to verify record exists")
		require.Len(t, results, 1, "Record should exist before deletion")

		// Delete the record
		err = store.Delete(ctx1, id)
		require.NoError(t, err, "Failed to delete record")

		// Verify record no longer exists
		results, err = store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to check if record was deleted")
		assert.Empty(t, results, "Record should be deleted")

		// Try to delete again - should return error
		err = store.Delete(ctx1, id)
		assert.Error(t, err, "Deleting non-existent record should return error")
	})

	t.Run("Cross-Entity Operations", func(t *testing.T) {
		// Create a record for entity1
		record := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Record for cross-entity test",
		}

		// Store the record
		id, err := store.Store(ctx1, record)
		require.NoError(t, err, "Failed to store record")

		// Try to update from entity2's context
		updatedRecord := ltm.MemoryRecord{
			ID:      id,
			Content: "Attempted update from wrong entity",
		}
		err = store.Update(ctx2, updatedRecord)
		assert.Error(t, err, "Update from wrong entity should fail")

		// Try to delete from entity2's context
		err = store.Delete(ctx2, id)
		assert.Error(t, err, "Delete from wrong entity should fail")

		// Verify record still exists and is unchanged
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}
		results, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve record after failed cross-entity operations")
		require.Len(t, results, 1, "Record should still exist")
		assert.Equal(t, "Record for cross-entity test", results[0].Content, "Content should be unchanged")
	})

	t.Run("Pagination and Limits", func(t *testing.T) {
		// Create multiple records
		for i := 0; i < 10; i++ {
			record := ltm.MemoryRecord{
				EntityID:    entity1ID,
				UserID:      user1ID,
				AccessLevel: entity.SharedWithinEntity,
				Content:     "Pagination test record",
				Metadata: map[string]interface{}{
					"index": i,
				},
			}
			_, err := store.Store(ctx1, record)
			require.NoError(t, err, "Failed to store record")
		}

		// Retrieve with limit
		query := ltm.LTMQuery{
			Limit: 5,
		}
		results, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve with limit")
		assert.LessOrEqual(t, len(results), 5, "Number of results should respect limit")
	})
}