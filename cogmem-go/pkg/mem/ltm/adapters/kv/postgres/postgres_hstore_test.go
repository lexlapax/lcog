package postgres

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHstoreStore tests the PostgreSQL Hstore LTM store adapter.
// This is an integration test that requires a PostgreSQL database with hstore extension.
// Set HSTORE_TEST_URL environment variable to run this test, or it will fall back to TEST_DB_URL.
func TestHstoreStore(t *testing.T) {
	// Skip if integration tests are not enabled
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL Hstore integration test. Set INTEGRATION_TESTS=true to run.")
		return
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

	// Create a new HstoreStore
	store := NewHstoreStore(db)
	require.NotNil(t, store, "Failed to create HstoreStore")

	// Create a unique table name for this test run to avoid conflicts
	tableName := fmt.Sprintf("memory_records_%d", time.Now().UnixNano())
	t.Logf("Using table name: %s", tableName)

	// Initialize the store (create tables)
	ctx := context.Background()
	err = store.Initialize(ctx)
	require.NoError(t, err, "Failed to initialize HstoreStore")

	// Clean up after the test
	defer cleanupTable(t, db, "memory_records")

	// Create entity context for testing
	entityID := entity.EntityID("test-entity")
	userID := "test-user"
	entityCtx := entity.NewContext(entityID, userID)
	ctx = entity.ContextWithEntity(ctx, entityCtx)

	// Test Store
	t.Run("Store", func(t *testing.T) {
		// Create a memory record
		record := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
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
		id, err := store.Store(ctx, record)
		require.NoError(t, err, "Failed to store record")
		assert.NotEmpty(t, id, "Record ID should not be empty")

		// Store should work with a record that has empty metadata
		emptyMetadataRecord := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Test content with empty metadata",
		}

		emptyMetadataID, err := store.Store(ctx, emptyMetadataRecord)
		require.NoError(t, err, "Failed to store record with empty metadata")
		assert.NotEmpty(t, emptyMetadataID, "Record ID should not be empty")
	})

	// Test Retrieve by ID
	t.Run("RetrieveByID", func(t *testing.T) {
		// Create and store a record
		record := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Test content for retrieval by ID",
			Metadata: map[string]interface{}{
				"tag": "retrieve-by-id-test",
			},
		}

		id, err := store.Store(ctx, record)
		require.NoError(t, err, "Failed to store record")

		// Retrieve the record by ID
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}

		records, err := store.Retrieve(ctx, query)
		require.NoError(t, err, "Failed to retrieve record by ID")
		require.Len(t, records, 1, "Should retrieve exactly one record")
		assert.Equal(t, id, records[0].ID, "Retrieved record ID should match")
		assert.Equal(t, record.Content, records[0].Content, "Retrieved content should match")
		assert.Equal(t, record.Metadata["tag"], records[0].Metadata["tag"], "Retrieved metadata should match")
	})

	// Test Retrieve by text search
	t.Run("RetrieveByText", func(t *testing.T) {
		// Create and store records with specific text
		record1 := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "This contains the special keyword goldfish",
			Metadata: map[string]interface{}{
				"tag": "text-search-test",
			},
		}

		record2 := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "This does not contain the keyword",
			Metadata: map[string]interface{}{
				"tag": "text-search-test",
			},
		}

		_, err := store.Store(ctx, record1)
		require.NoError(t, err, "Failed to store record1")

		_, err = store.Store(ctx, record2)
		require.NoError(t, err, "Failed to store record2")

		// Retrieve records containing "goldfish"
		query := ltm.LTMQuery{
			Text: "goldfish",
		}

		records, err := store.Retrieve(ctx, query)
		require.NoError(t, err, "Failed to retrieve records by text")
		
		// Verify that only record1 is returned
		foundMatch := false
		for _, record := range records {
			if record.Content == record1.Content {
				foundMatch = true
				break
			}
		}
		assert.True(t, foundMatch, "Should find the record containing 'goldfish'")
	})

	// Test Retrieve by metadata
	t.Run("RetrieveByMetadata", func(t *testing.T) {
		// Create and store records with specific metadata
		record1 := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content with specific metadata",
			Metadata: map[string]interface{}{
				"category": "important",
				"priority": 1,
			},
		}

		record2 := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content with different metadata",
			Metadata: map[string]interface{}{
				"category": "normal",
				"priority": 2,
			},
		}

		_, err := store.Store(ctx, record1)
		require.NoError(t, err, "Failed to store record1")

		_, err = store.Store(ctx, record2)
		require.NoError(t, err, "Failed to store record2")

		// Retrieve records with category=important
		query := ltm.LTMQuery{
			Filters: map[string]interface{}{
				"category": "important",
			},
		}

		records, err := store.Retrieve(ctx, query)
		require.NoError(t, err, "Failed to retrieve records by metadata")
		
		// Verify that only record1 is returned (with category=important)
		foundMatch := false
		for _, record := range records {
			if val, ok := record.Metadata["category"]; ok && val == "important" {
				foundMatch = true
				break
			}
		}
		assert.True(t, foundMatch, "Should find the record with category=important")
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		// Create and store a record
		record := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Initial content",
			Metadata: map[string]interface{}{
				"tag": "update-test",
			},
		}

		id, err := store.Store(ctx, record)
		require.NoError(t, err, "Failed to store record")

		// Update the record
		updatedRecord := ltm.MemoryRecord{
			ID:      id,
			Content: "Updated content",
			Metadata: map[string]interface{}{
				"tag": "updated",
				"new": "value",
			},
		}

		err = store.Update(ctx, updatedRecord)
		require.NoError(t, err, "Failed to update record")

		// Retrieve the updated record
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}

		records, err := store.Retrieve(ctx, query)
		require.NoError(t, err, "Failed to retrieve updated record")
		require.Len(t, records, 1, "Should retrieve exactly one record")
		assert.Equal(t, "Updated content", records[0].Content, "Content should be updated")
		assert.Equal(t, "updated", records[0].Metadata["tag"], "Metadata tag should be updated")
		assert.Equal(t, "value", records[0].Metadata["new"], "New metadata field should be added")
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		// Create and store a record
		record := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Content to be deleted",
			Metadata: map[string]interface{}{
				"tag": "delete-test",
			},
		}

		id, err := store.Store(ctx, record)
		require.NoError(t, err, "Failed to store record")

		// Delete the record
		err = store.Delete(ctx, id)
		require.NoError(t, err, "Failed to delete record")

		// Try to retrieve the deleted record
		query := ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": id,
			},
		}

		records, err := store.Retrieve(ctx, query)
		require.NoError(t, err, "Failed to execute retrieval query")
		assert.Len(t, records, 0, "Deleted record should not be retrievable")
	})

	// Test entity isolation
	t.Run("EntityIsolation", func(t *testing.T) {
		// Create a record for the test entity
		testEntityRecord := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Test entity content",
			Metadata: map[string]interface{}{
				"tag": "entity-isolation-test",
			},
		}

		_, err := store.Store(ctx, testEntityRecord)
		require.NoError(t, err, "Failed to store test entity record")

		// Create a different entity context
		otherEntityID := entity.EntityID("other-entity")
		otherUserID := "other-user"
		otherEntityCtx := entity.NewContext(otherEntityID, otherUserID)
		otherCtx := entity.ContextWithEntity(context.Background(), otherEntityCtx)

		// Create a record for the other entity
		otherEntityRecord := ltm.MemoryRecord{
			EntityID:    otherEntityID,
			UserID:      otherUserID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Other entity content",
			Metadata: map[string]interface{}{
				"tag": "entity-isolation-test",
			},
		}

		_, err = store.Store(otherCtx, otherEntityRecord)
		require.NoError(t, err, "Failed to store other entity record")

		// Retrieve records with the test entity context
		query := ltm.LTMQuery{
			Filters: map[string]interface{}{
				"tag": "entity-isolation-test",
			},
		}

		records, err := store.Retrieve(ctx, query)
		require.NoError(t, err, "Failed to retrieve records for test entity")
		
		// Verify that only the test entity's record is returned
		for _, record := range records {
			assert.Equal(t, entityID, record.EntityID, "Retrieved records should belong to the test entity")
			assert.NotEqual(t, otherEntityID, record.EntityID, "Retrieved records should not belong to the other entity")
		}

		// Retrieve records with the other entity context
		otherRecords, err := store.Retrieve(otherCtx, query)
		require.NoError(t, err, "Failed to retrieve records for other entity")
		
		// Verify that only the other entity's record is returned
		for _, record := range otherRecords {
			assert.Equal(t, otherEntityID, record.EntityID, "Retrieved records should belong to the other entity")
			assert.NotEqual(t, entityID, record.EntityID, "Retrieved records should not belong to the test entity")
		}
	})

	// Test access level filtering
	t.Run("AccessLevelFiltering", func(t *testing.T) {
		// Create a shared record
		sharedRecord := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.SharedWithinEntity,
			Content:     "Shared content",
			Metadata: map[string]interface{}{
				"access": "shared",
			},
		}

		_, err := store.Store(ctx, sharedRecord)
		require.NoError(t, err, "Failed to store shared record")

		// Create a private record
		privateRecord := ltm.MemoryRecord{
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.PrivateToUser,
			Content:     "Private content",
			Metadata: map[string]interface{}{
				"access": "private",
			},
		}

		_, err = store.Store(ctx, privateRecord)
		require.NoError(t, err, "Failed to store private record")

		// Create a different user context for the same entity
		otherUserCtx := entity.NewContext(entityID, "other-user")
		otherCtx := entity.ContextWithEntity(context.Background(), otherUserCtx)

		// Retrieve records with the original user context
		query := ltm.LTMQuery{}
		records, err := store.Retrieve(ctx, query)
		require.NoError(t, err, "Failed to retrieve records")
		
		// The original user should see both shared and private records
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

		// Retrieve records with the other user context
		otherUserRecords, err := store.Retrieve(otherCtx, query)
		require.NoError(t, err, "Failed to retrieve records for other user")
		
		// The other user should only see shared records
		foundShared, foundPrivate = false, false
		for _, record := range otherUserRecords {
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
	})
}

// Test helper functions

// jsonToHstoreStr
func TestJsonToHstoreStr(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty object",
			input:    "{}",
			expected: "",
		},
		{
			name:     "Simple key-value",
			input:    `{"key": "value"}`,
			expected: `"key"=>"value"`,
		},
		{
			name:     "Multiple key-values",
			input:    `{"key1": "value1", "key2": "value2"}`,
			expected: `"key1"=>"value1", "key2"=>"value2"`,
		},
		{
			name:     "Numeric value",
			input:    `{"num": 123}`,
			expected: `"num"=>"123"`,
		},
		{
			name:     "Boolean value",
			input:    `{"bool": true}`,
			expected: `"bool"=>"true"`,
		},
		{
			name:     "Null value",
			input:    `{"null": null}`,
			expected: `"null"=>""`,
		},
		{
			name:     "Nested object",
			input:    `{"obj": {"nested": "value"}}`,
			expected: `"obj"=>"{\"nested\":\"value\"}"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := jsonToHstoreStr([]byte(tc.input))
			require.NoError(t, err, "jsonToHstoreStr should not return error")
			
			// The order of keys in the result might be different, so we need to compare differently
			if tc.expected == "" {
				assert.Equal(t, "", result, "Empty object should result in empty string")
			} else {
				parts := strings.Split(tc.expected, ", ")
				for _, part := range parts {
					assert.Contains(t, result, part, "Result should contain expected key-value pair")
				}
			}
		})
	}
}

// hstoreStrToMap
func TestHstoreStrToMap(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: map[string]interface{}{},
		},
		{
			name:     "Simple key-value",
			input:    `"key"=>"value"`,
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "Multiple key-values",
			input:    `"key1"=>"value1", "key2"=>"value2"`,
			expected: map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "Numeric value",
			input:    `"num"=>"123"`,
			expected: map[string]interface{}{"num": float64(123)},
		},
		{
			name:     "Boolean value",
			input:    `"bool"=>"true"`,
			expected: map[string]interface{}{"bool": true},
		},
		{
			name:     "Nested object",
			input:    `"obj"=>"{\"nested\":\"value\"}"`,
			expected: map[string]interface{}{"obj": map[string]interface{}{"nested": "value"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := hstoreStrToMap(tc.input)
			require.NoError(t, err, "hstoreStrToMap should not return error")
			assert.Equal(t, tc.expected, result, "Result map should match expected")
		})
	}
}

// Helper function to clean up test tables
func cleanupTable(t *testing.T, db *sqlx.DB, tableName string) {
	_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", tableName))
	if err != nil {
		t.Logf("Failed to truncate table %s: %v", tableName, err)
	}
}