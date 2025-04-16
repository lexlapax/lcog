//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/kv/postgres"
)

// TestPostgresHstoreLTMOperations tests the PostgreSQL Hstore adapter functionality.
func TestPostgresHstoreLTMOperations(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL Hstore integration test. Set INTEGRATION_TESTS=true to run.")
	}

	// Get database connection string from environment variable or use default
	dbURL := os.Getenv("HSTORE_TEST_URL")
	if dbURL == "" {
		dbURL = os.Getenv("TEST_DB_URL")
		if dbURL == "" {
			dbURL = "postgres://postgres:postgres@localhost:5432/cogmem_test?sslmode=disable"
		}
	}

	// Connect to the database
	db, err := sqlx.Connect("postgres", dbURL)
	require.NoError(t, err, "Failed to connect to PostgreSQL")
	defer db.Close()
	
	// Enable hstore extension if not already enabled
	_, err = db.Exec("CREATE EXTENSION IF NOT EXISTS hstore")
	require.NoError(t, err, "Failed to enable hstore extension")
	
	// Drop the test table if it exists to ensure a clean state
	_, err = db.Exec("DROP TABLE IF EXISTS memory_records_hstore")
	require.NoError(t, err, "Failed to drop test table")
	
	// Create a specific table for Hstore tests
	_, err = db.Exec(`
		CREATE TABLE memory_records_hstore (
			id TEXT PRIMARY KEY,
			entity_id TEXT NOT NULL,
			user_id TEXT,
			access_level INTEGER NOT NULL,
			content TEXT NOT NULL,
			metadata hstore,
			created_at TIMESTAMP WITH TIME ZONE,
			updated_at TIMESTAMP WITH TIME ZONE
		)
	`)
	require.NoError(t, err, "Failed to create memory_records_hstore table")
	
	// Create index on entity_id for faster lookups
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS memory_records_hstore_entity_id_idx ON memory_records_hstore (entity_id)
	`)
	require.NoError(t, err, "Failed to create entity_id index")

	// Create a new HstoreStore with custom table name
	store := postgres.NewHstoreStore(db)
	store.WithTableName("memory_records_hstore")
	require.NotNil(t, store, "Failed to create HstoreStore")

	// Initialize the store (create tables)
	ctx := context.Background()
	err = store.Initialize(ctx)
	require.NoError(t, err, "Failed to initialize HstoreStore")
	
	// Clean up after the test
	defer func() {
		_, err := db.Exec("DROP TABLE IF EXISTS memory_records_hstore")
		if err != nil {
			t.Logf("Failed to drop test table: %v", err)
		}
	}()

	// Additional cleanup to ensure we don't leave any data behind
	defer func() {
		_, err := db.Exec("TRUNCATE TABLE memory_records_hstore CASCADE")
		if err != nil {
			t.Logf("Failed to truncate table: %v", err)
		}
	}()

	// Create test contexts
	entity1ID := entity.EntityID("test-entity-1")
	entity2ID := entity.EntityID("test-entity-2")
	user1ID := "test-user-1"
	user2ID := "test-user-2"
	
	entity1Ctx := entity.NewContext(entity1ID, user1ID)
	entity2Ctx := entity.NewContext(entity2ID, user2ID)
	ctx1 := entity.ContextWithEntity(ctx, entity1Ctx)
	ctx2 := entity.ContextWithEntity(ctx, entity2Ctx)

	t.Run("Store and Retrieve Basic Record", func(t *testing.T) {
		// Create a test record
		record := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Test content for PostgreSQL Hstore",
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
				"key4": map[string]string{
					"nested": "value",
				},
			},
		}

		// Store the record
		id, err := store.Store(ctx1, record)
		require.NoError(t, err, "Failed to store record")
		assert.NotEmpty(t, id, "Record ID should not be empty")

		// Retrieve the record by ID
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}

		records, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve record by ID")
		require.Len(t, records, 1, "Should retrieve exactly one record")
		assert.Equal(t, id, records[0].ID, "Retrieved record ID should match")
		assert.Equal(t, record.Content, records[0].Content, "Retrieved content should match")
		assert.Equal(t, "value1", records[0].Metadata["key1"], "Retrieved metadata should match")
		assert.Equal(t, float64(123), records[0].Metadata["key2"], "Retrieved numeric metadata should match")
		assert.Equal(t, true, records[0].Metadata["key3"], "Retrieved boolean metadata should match")
		
		// Verify nested object was properly serialized and deserialized
		nestedObj, ok := records[0].Metadata["key4"].(map[string]interface{})
		require.True(t, ok, "Nested object should be a map")
		assert.Equal(t, "value", nestedObj["nested"], "Nested value should match")
	})

	t.Run("Entity Isolation", func(t *testing.T) {
		// Create a record for the first entity
		record1 := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content for entity 1",
			Metadata: map[string]interface{}{
				"entity": "1",
			},
		}

		// Create a record for the second entity
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

		id2, err := store.Store(ctx2, record2)
		require.NoError(t, err, "Failed to store record for entity 2")

		// Entity 1 should only see its own records
		records1, err := store.Retrieve(ctx1, ltm.LTMQuery{})
		require.NoError(t, err, "Failed to retrieve records for entity 1")
		
		// Verify all records belong to entity 1
		for _, record := range records1 {
			assert.Equal(t, entity1ID, record.EntityID, "Records for entity 1 should have entity1ID")
		}

		// Entity 2 should only see its own records
		records2, err := store.Retrieve(ctx2, ltm.LTMQuery{})
		require.NoError(t, err, "Failed to retrieve records for entity 2")
		
		// Verify all records belong to entity 2
		for _, record := range records2 {
			assert.Equal(t, entity2ID, record.EntityID, "Records for entity 2 should have entity2ID")
		}

		// Entity 1 should not be able to access entity 2's record by ID
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id2,
			},
		}
		records, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to execute query")
		assert.Empty(t, records, "Entity 1 should not see entity 2's records")

		// Entity 2 should not be able to access entity 1's record by ID
		query = ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id1,
			},
		}
		records, err = store.Retrieve(ctx2, query)
		require.NoError(t, err, "Failed to execute query")
		assert.Empty(t, records, "Entity 2 should not see entity 1's records")
	})

	t.Run("Access Level Control", func(t *testing.T) {
		// Create a shared record
		sharedRecord := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Shared content within entity",
			Metadata: map[string]interface{}{
				"access": "shared",
			},
		}

		// Create a private record
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

		// Create a different user context for the same entity
		otherUserCtx := entity.NewContext(entity1ID, "other-user")
		otherCtx := entity.ContextWithEntity(context.Background(), otherUserCtx)

		// Original user should see both shared and private records
		query := ltm.LTMQuery{}
		records, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve records for original user")
		
		// Find both shared and private records
		var foundShared, foundPrivate bool
		for _, record := range records {
			if access, ok := record.Metadata["access"]; ok {
				if access == "shared" {
					foundShared = true
				}
				if access == "private" {
					foundPrivate = true
				}
			}
		}
		assert.True(t, foundShared, "Original user should see shared records")
		assert.True(t, foundPrivate, "Original user should see private records")

		// Other user should only see shared records
		records, err = store.Retrieve(otherCtx, query)
		require.NoError(t, err, "Failed to retrieve records for other user")
		
		// Reset flags
		foundShared, foundPrivate = false, false
		for _, record := range records {
			if access, ok := record.Metadata["access"]; ok {
				if access == "shared" {
					foundShared = true
				}
				if access == "private" {
					foundPrivate = true
				}
			}
		}
		assert.True(t, foundShared, "Other user should see shared records")
		assert.False(t, foundPrivate, "Other user should not see private records")

		// Direct ID access should respect access levels
		query = ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": privateID,
			},
		}
		records, err = store.Retrieve(otherCtx, query)
		require.NoError(t, err, "Failed to retrieve private record by ID for other user")
		assert.Empty(t, records, "Other user should not be able to access private record by ID")

		query = ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": sharedID,
			},
		}
		records, err = store.Retrieve(otherCtx, query)
		require.NoError(t, err, "Failed to retrieve shared record by ID for other user")
		assert.NotEmpty(t, records, "Other user should be able to access shared record by ID")
	})

	t.Run("Text Search", func(t *testing.T) {
		// Create records with specific text
		record1 := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "This contains the special keyword goldfish",
			Metadata: map[string]interface{}{
				"tag": "text-search-test",
			},
		}

		record2 := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "This does not contain the keyword",
			Metadata: map[string]interface{}{
				"tag": "text-search-test",
			},
		}

		// Store the records
		_, err := store.Store(ctx1, record1)
		require.NoError(t, err, "Failed to store record1")

		_, err = store.Store(ctx1, record2)
		require.NoError(t, err, "Failed to store record2")

		// Search for records containing "goldfish"
		query := ltm.LTMQuery{
			Text: "goldfish",
		}

		records, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve records by text")
		
		// Verify only record1 is returned
		foundMatch := false
		for _, record := range records {
			if record.Content == record1.Content {
				foundMatch = true
				break
			}
		}
		assert.True(t, foundMatch, "Should find the record containing 'goldfish'")
	})

	t.Run("Metadata Filtering", func(t *testing.T) {
		// Create records with specific metadata
		record1 := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content with specific metadata",
			Metadata: map[string]interface{}{
				"category": "important",
				"priority": 1,
			},
		}

		record2 := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content with different metadata",
			Metadata: map[string]interface{}{
				"category": "normal",
				"priority": 2,
			},
		}

		// Store the records
		_, err := store.Store(ctx1, record1)
		require.NoError(t, err, "Failed to store record1")

		_, err = store.Store(ctx1, record2)
		require.NoError(t, err, "Failed to store record2")

		// Retrieve records with category=important
		query := ltm.LTMQuery{
			Filters: map[string]interface{}{
				"category": "important",
			},
		}

		records, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve records by metadata")
		
		// Verify that only record1 is returned
		foundMatch := false
		for _, record := range records {
			if val, ok := record.Metadata["category"]; ok && val == "important" {
				foundMatch = true
				break
			}
		}
		assert.True(t, foundMatch, "Should find the record with category=important")
	})

	t.Run("Update Record", func(t *testing.T) {
		// Create and store a record
		record := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Initial content",
			Metadata: map[string]interface{}{
				"tag": "update-test",
			},
		}

		id, err := store.Store(ctx1, record)
		require.NoError(t, err, "Failed to store record")

		// Get the original timestamp
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}
		originalRecords, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve original record")
		require.Len(t, originalRecords, 1, "Should retrieve exactly one record")
		originalTimestamp := originalRecords[0].UpdatedAt

		// Small delay to ensure timestamps will be different
		time.Sleep(1 * time.Millisecond)

		// Update the record
		updatedRecord := ltm.MemoryRecord{
			ID:      id,
			Content: "Updated content",
			Metadata: map[string]interface{}{
				"tag": "updated",
				"new": "value",
			},
		}

		err = store.Update(ctx1, updatedRecord)
		require.NoError(t, err, "Failed to update record")

		// Retrieve the updated record
		updatedRecords, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve updated record")
		require.Len(t, updatedRecords, 1, "Should retrieve exactly one record")
		assert.Equal(t, "Updated content", updatedRecords[0].Content, "Content should be updated")
		assert.Equal(t, "updated", updatedRecords[0].Metadata["tag"], "Metadata tag should be updated")
		assert.Equal(t, "value", updatedRecords[0].Metadata["new"], "New metadata field should be added")
		
		// Check that timestamp was updated
		assert.True(t, updatedRecords[0].UpdatedAt.After(originalTimestamp), "Updated timestamp should be after original timestamp")
	})

	t.Run("Delete Record", func(t *testing.T) {
		// Create and store a record
		record := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content to be deleted",
			Metadata: map[string]interface{}{
				"tag": "delete-test",
			},
		}

		id, err := store.Store(ctx1, record)
		require.NoError(t, err, "Failed to store record")

		// Delete the record
		err = store.Delete(ctx1, id)
		require.NoError(t, err, "Failed to delete record")

		// Try to retrieve the deleted record
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}

		records, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to execute retrieval query")
		assert.Len(t, records, 0, "Deleted record should not be retrievable")
		
		// Try to delete again - should fail
		err = store.Delete(ctx1, id)
		assert.Error(t, err, "Deleting non-existent record should fail")
	})

	t.Run("Cross-Entity Operations", func(t *testing.T) {
		// Create a record for entity1
		record := ltm.MemoryRecord{
			EntityID:    entity1ID,
			UserID:      user1ID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content for cross-entity test",
		}

		// Store the record
		id, err := store.Store(ctx1, record)
		require.NoError(t, err, "Failed to store record")

		// Try to update the record from entity2's context - should fail
		updatedRecord := ltm.MemoryRecord{
			ID:      id,
			Content: "Updated from wrong entity",
		}

		err = store.Update(ctx2, updatedRecord)
		assert.Error(t, err, "Updating record from wrong entity context should fail")

		// Try to delete the record from entity2's context - should fail
		err = store.Delete(ctx2, id)
		assert.Error(t, err, "Deleting record from wrong entity context should fail")
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
			Text:  "Pagination",
			Limit: 5,
		}
		records, err := store.Retrieve(ctx1, query)
		require.NoError(t, err, "Failed to retrieve records with limit")
		assert.LessOrEqual(t, len(records), 5, "Number of records should respect limit")
	})
}