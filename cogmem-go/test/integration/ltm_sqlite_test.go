//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/sqlstore/sqlite"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	// Try multiple locations for .env file
	if err := godotenv.Load(); err != nil {
		// Try project root
		_ = godotenv.Load("../../.env")
	}
}

// TestSQLiteLTMOperations tests the core SQLite adapter functionality.
func TestSQLiteLTMOperations(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Create a temporary SQLite database
	tmpFile, err := os.CreateTemp("", "cogmem-test-*.db")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Connect to the temporary database
	db, err := sql.Open("sqlite3", tmpFile.Name())
	require.NoError(t, err)
	defer db.Close()

	// Create the memory_records table
	_, err = db.Exec(`
		CREATE TABLE memory_records (
			id TEXT PRIMARY KEY,
			entity_id TEXT NOT NULL,
			user_id TEXT,
			access_level INTEGER NOT NULL,
			content TEXT NOT NULL,
			metadata TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`)
	require.NoError(t, err)

	// Create the SQLite adapter
	store := sqlite.NewSQLiteStore(db)

	// Create test contexts
	entity1Ctx := entity.NewContext("entity1", "user1")
	entity2Ctx := entity.NewContext("entity2", "user2")
	ctx1 := entity.ContextWithEntity(context.Background(), entity1Ctx)
	ctx2 := entity.ContextWithEntity(context.Background(), entity2Ctx)

	t.Run("Store and Retrieve Basic Record", func(t *testing.T) {
		// Create a test record
		record := ltm.MemoryRecord{
			EntityID:    entity.EntityID("entity1"),
			UserID:      "user1",
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Test content for entity1",
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		}

		// Store the record
		id, err := store.Store(ctx1, record)
		require.NoError(t, err)
		require.NotEmpty(t, id)

		// Retrieve the record by ID
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}
		results, err := store.Retrieve(ctx1, query)
		require.NoError(t, err)
		require.Len(t, results, 1)

		// Verify retrieved record
		retrieved := results[0]
		assert.Equal(t, id, retrieved.ID)
		assert.Equal(t, entity.EntityID("entity1"), retrieved.EntityID)
		assert.Equal(t, "user1", retrieved.UserID)
		assert.Equal(t, entity.SharedWithinEntity, retrieved.AccessLevel)
		assert.Equal(t, "Test content for entity1", retrieved.Content)
		assert.Equal(t, "value1", retrieved.Metadata["key1"])
		assert.Equal(t, float64(42), retrieved.Metadata["key2"])
		assert.False(t, retrieved.CreatedAt.IsZero())
		assert.False(t, retrieved.UpdatedAt.IsZero())
	})

	t.Run("Entity Isolation", func(t *testing.T) {
		// Create records for two different entities
		record1 := ltm.MemoryRecord{
			EntityID:    entity.EntityID("entity1"),
			UserID:      "user1",
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content for entity1 only",
		}

		record2 := ltm.MemoryRecord{
			EntityID:    entity.EntityID("entity2"),
			UserID:      "user2",
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content for entity2 only",
		}

		// Store both records
		id1, err := store.Store(ctx1, record1)
		require.NoError(t, err)

		id2, err := store.Store(ctx2, record2)
		require.NoError(t, err)

		// Attempt to retrieve entity1's record from entity2's context
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id1,
			},
		}
		results, err := store.Retrieve(ctx2, query)
		require.NoError(t, err)
		assert.Empty(t, results, "Entity1's record should not be visible to entity2")

		// Retrieve entity2's record from entity2's context
		query = ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id2,
			},
		}
		results, err = store.Retrieve(ctx2, query)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "Content for entity2 only", results[0].Content)
	})

	t.Run("Access Level Control", func(t *testing.T) {
		// Create a shared record
		sharedRecord := ltm.MemoryRecord{
			EntityID:    entity.EntityID("entity1"),
			UserID:      "user1",
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Shared content within entity1",
		}

		// Create a private record
		privateRecord := ltm.MemoryRecord{
			EntityID:    entity.EntityID("entity1"),
			UserID:      "user1",
			AccessLevel: entity.PrivateToUser,
			Content:     "Private content for user1 only",
		}

		// Store both records
		sharedID, err := store.Store(ctx1, sharedRecord)
		require.NoError(t, err)

		privateID, err := store.Store(ctx1, privateRecord)
		require.NoError(t, err)

		// Create context for a different user in the same entity
		otherUserCtx := entity.NewContext("entity1", "user2")
		ctx1OtherUser := entity.ContextWithEntity(context.Background(), otherUserCtx)

		// Retrieve shared record with different user - should succeed
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": sharedID,
			},
		}
		results, err := store.Retrieve(ctx1OtherUser, query)
		require.NoError(t, err)
		require.Len(t, results, 1, "Shared record should be visible to other users in same entity")

		// Retrieve private record with different user - should not be visible
		query = ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": privateID,
			},
		}
		results, err = store.Retrieve(ctx1OtherUser, query)
		require.NoError(t, err)
		assert.Empty(t, results, "Private record should not be visible to other users")

		// Original user should still see their private record
		results, err = store.Retrieve(ctx1, query)
		require.NoError(t, err)
		require.Len(t, results, 1, "Original user should see their private record")
	})

	t.Run("Text Search", func(t *testing.T) {
		// Create records with different text content
		records := []ltm.MemoryRecord{
			{
				EntityID:    entity.EntityID("entity1"),
				UserID:      "user1",
				AccessLevel: entity.SharedWithinEntity,
				Content:     "Apple pie recipe",
			},
			{
				EntityID:    entity.EntityID("entity1"),
				UserID:      "user1",
				AccessLevel: entity.SharedWithinEntity,
				Content:     "Banana bread recipe",
			},
			{
				EntityID:    entity.EntityID("entity1"),
				UserID:      "user1",
				AccessLevel: entity.SharedWithinEntity,
				Content:     "Cherry cake recipe",
			},
		}

		// Store all records
		for _, record := range records {
			_, err := store.Store(ctx1, record)
			require.NoError(t, err)
		}

		// Search for "recipe" - should find all 3
		query := ltm.LTMQuery{
			Text: "recipe",
		}
		results, err := store.Retrieve(ctx1, query)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 3, "Should find at least 3 records with 'recipe'")

		// Search for "banana" - should find only one
		query = ltm.LTMQuery{
			Text: "banana",
		}
		results, err = store.Retrieve(ctx1, query)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1, "Should find at least one record with 'banana'")
		// Verify it's the right record
		foundBanana := false
		for _, result := range results {
			if result.Content == "Banana bread recipe" {
				foundBanana = true
				break
			}
		}
		assert.True(t, foundBanana, "Should find the banana bread record")
	})

	t.Run("Update Record", func(t *testing.T) {
		// Create a record
		record := ltm.MemoryRecord{
			EntityID:    entity.EntityID("entity1"),
			UserID:      "user1",
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Original content",
			Metadata: map[string]interface{}{
				"version": 1,
			},
		}

		// Store the record
		id, err := store.Store(ctx1, record)
		require.NoError(t, err)

		// Update the record
		updatedRecord := ltm.MemoryRecord{
			ID:          id,
			EntityID:    entity.EntityID("entity1"), // Must match original
			Content:     "Updated content",
			Metadata: map[string]interface{}{
				"version": 2,
			},
		}
		
		// Small delay to ensure updated timestamp is different
		time.Sleep(1 * time.Millisecond)
		
		err = store.Update(ctx1, updatedRecord)
		require.NoError(t, err)

		// Retrieve the updated record
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}
		results, err := store.Retrieve(ctx1, query)
		require.NoError(t, err)
		require.Len(t, results, 1)

		// Verify updated content and metadata
		updated := results[0]
		assert.Equal(t, "Updated content", updated.Content)
		assert.Equal(t, float64(2), updated.Metadata["version"])
		
		// Verify timestamps
		assert.False(t, updated.UpdatedAt.Equal(updated.CreatedAt), 
			"Updated time should be different from created time")
	})

	t.Run("Delete Record", func(t *testing.T) {
		// Create a record
		record := ltm.MemoryRecord{
			EntityID:    entity.EntityID("entity1"),
			UserID:      "user1",
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content to be deleted",
		}

		// Store the record
		id, err := store.Store(ctx1, record)
		require.NoError(t, err)

		// Verify record exists
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}
		results, err := store.Retrieve(ctx1, query)
		require.NoError(t, err)
		require.Len(t, results, 1)

		// Delete the record
		err = store.Delete(ctx1, id)
		require.NoError(t, err)

		// Verify record no longer exists
		results, err = store.Retrieve(ctx1, query)
		require.NoError(t, err)
		assert.Empty(t, results, "Record should be deleted")

		// Attempt to delete again - should error
		err = store.Delete(ctx1, id)
		assert.Error(t, err, "Deleting non-existent record should return error")
	})

	t.Run("Cross-Entity Operations", func(t *testing.T) {
		// Try to store a record with mismatched entity IDs
		record := ltm.MemoryRecord{
			EntityID:    entity.EntityID("entity2"), // Different from context
			UserID:      "user1",
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content with mismatched entity",
		}

		// This should fail
		_, err := store.Store(ctx1, record)
		assert.Error(t, err, "Storing with mismatched entity IDs should fail")

		// Create a valid record for entity1
		validRecord := ltm.MemoryRecord{
			EntityID:    entity.EntityID("entity1"),
			UserID:      "user1",
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Valid content",
		}

		// Store valid record
		id, err := store.Store(ctx1, validRecord)
		require.NoError(t, err)

		// Try to update from entity2's context - should fail
		updateRecord := ltm.MemoryRecord{
			ID:      id,
			Content: "Updated by wrong entity",
		}
		err = store.Update(ctx2, updateRecord)
		assert.Error(t, err, "Updating from wrong entity context should fail")

		// Try to delete from entity2's context - should fail
		err = store.Delete(ctx2, id)
		assert.Error(t, err, "Deleting from wrong entity context should fail")
	})
}