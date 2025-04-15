package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test database connection
func setupTestDB(t *testing.T) *pgxpool.Pool {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run.")
	}

	// Get database connection string from environment or use default
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/cogmem_test?sslmode=disable"
	}

	// Create a connection pool
	config, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)

	// Ping to verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err)

	// Clean up any existing test data
	_, err = pool.Exec(ctx, "DELETE FROM memory_records")
	require.NoError(t, err)

	return pool
}

func TestPostgresStore_Store(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	store := NewPostgresStore(pool)
	entityID := entity.EntityID("test-entity")
	ctx := entity.ContextWithEntityID(context.Background(), entityID)

	// Create a test record
	record := ltm.MemoryRecord{
		EntityID:    entityID,
		Content:     "test memory content",
		AccessLevel: entity.SharedWithinEntity,
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	}

	// Test storing a record
	id, err := store.Store(ctx, record)
	assert.NoError(t, err)
	assert.NotEmpty(t, id, "ID should be generated and not empty")

	// Verify record was stored with correct values
	records, err := store.Retrieve(ctx, ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": id,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, entityID, records[0].EntityID)
	assert.Equal(t, record.Content, records[0].Content)
	assert.Equal(t, record.AccessLevel, records[0].AccessLevel)
	assert.Equal(t, "value1", records[0].Metadata["key1"])
	assert.WithinDuration(t, time.Now(), records[0].CreatedAt, 5*time.Second)
	assert.WithinDuration(t, time.Now(), records[0].UpdatedAt, 5*time.Second)
}

func TestPostgresStore_Retrieve(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	store := NewPostgresStore(pool)
	entityID := entity.EntityID("test-entity")
	ctx := entity.ContextWithEntityID(context.Background(), entityID)

	// Create test records
	testRecords := []ltm.MemoryRecord{
		{
			EntityID:    entityID,
			Content:     "first memory",
			AccessLevel: entity.SharedWithinEntity,
			Metadata: map[string]interface{}{
				"type": "note",
				"tags": []string{"important", "work"},
			},
		},
		{
			EntityID:    entityID,
			Content:     "second memory with searchable content",
			AccessLevel: entity.SharedWithinEntity,
			Metadata: map[string]interface{}{
				"type": "document",
				"tags": []string{"reference"},
			},
		},
	}

	// Store test records
	var ids []string
	for _, record := range testRecords {
		id, err := store.Store(ctx, record)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	// Test retrieving by content
	t.Run("retrieve by text", func(t *testing.T) {
		results, err := store.Retrieve(ctx, ltm.LTMQuery{
			Text: "searchable",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0].Content, "searchable")
	})

	// Test retrieving with exact match
	t.Run("retrieve by exact match", func(t *testing.T) {
		results, err := store.Retrieve(ctx, ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": ids[0],
			},
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, ids[0], results[0].ID)
	})
}

func TestPostgresStore_RetrieveIsolation(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	store := NewPostgresStore(pool)
	
	// Create two different entities
	entityA := entity.EntityID("entity-a")
	entityB := entity.EntityID("entity-b")
	
	ctxA := entity.ContextWithEntityID(context.Background(), entityA)
	ctxB := entity.ContextWithEntityID(context.Background(), entityB)

	// Create and store record for Entity A
	recordA := ltm.MemoryRecord{
		EntityID:    entityA,
		Content:     "entity A record",
		AccessLevel: entity.SharedWithinEntity,
	}
	idA, err := store.Store(ctxA, recordA)
	require.NoError(t, err)

	// Create and store record for Entity B
	recordB := ltm.MemoryRecord{
		EntityID:    entityB,
		Content:     "entity B record",
		AccessLevel: entity.SharedWithinEntity,
	}
	idB, err := store.Store(ctxB, recordB)
	require.NoError(t, err)

	// Entity A should only see its own records
	resultsA, err := store.Retrieve(ctxA, ltm.LTMQuery{})
	assert.NoError(t, err)
	assert.Len(t, resultsA, 1)
	assert.Equal(t, entityA, resultsA[0].EntityID)
	assert.Equal(t, idA, resultsA[0].ID)

	// Entity B should only see its own records
	resultsB, err := store.Retrieve(ctxB, ltm.LTMQuery{})
	assert.NoError(t, err)
	assert.Len(t, resultsB, 1)
	assert.Equal(t, entityB, resultsB[0].EntityID)
	assert.Equal(t, idB, resultsB[0].ID)

	// Entity A should not be able to access Entity B's record by ID
	resultsA, err = store.Retrieve(ctxA, ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": idB,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, resultsA, 0, "Entity A should not be able to access Entity B's records")
}

func TestPostgresStore_AccessLevel(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	store := NewPostgresStore(pool)
	entityID := entity.EntityID("test-entity")
	
	// Create contexts for different users within the same entity
	userA := "user-a"
	userB := "user-b"
	
	ctxEntityOnly := entity.ContextWithEntityID(context.Background(), entityID)
	ctxUserA := entity.ContextWithEntity(context.Background(), entity.Context{
		EntityID: entityID,
		UserID:   userA,
	})
	ctxUserB := entity.ContextWithEntity(context.Background(), entity.Context{
		EntityID: entityID,
		UserID:   userB,
	})

	// Create and store a private record for User A
	privateRecordA := ltm.MemoryRecord{
		EntityID:    entityID,
		UserID:      userA,
		AccessLevel: entity.PrivateToUser,
		Content:     "private to user A",
	}
	privateIDa, err := store.Store(ctxUserA, privateRecordA)
	require.NoError(t, err)

	// Create and store a shared record
	sharedRecord := ltm.MemoryRecord{
		EntityID:    entityID,
		AccessLevel: entity.SharedWithinEntity,
		Content:     "shared within entity",
	}
	sharedID, err := store.Store(ctxEntityOnly, sharedRecord)
	require.NoError(t, err)

	// Test that User A can see both their private record and the shared record
	t.Run("user A can see their private and shared records", func(t *testing.T) {
		results, err := store.Retrieve(ctxUserA, ltm.LTMQuery{})
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		
		// Check that both IDs are present
		ids := []string{results[0].ID, results[1].ID}
		assert.Contains(t, ids, privateIDa)
		assert.Contains(t, ids, sharedID)
	})

	// Test that User B can only see the shared record
	t.Run("user B can only see shared records", func(t *testing.T) {
		results, err := store.Retrieve(ctxUserB, ltm.LTMQuery{})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, sharedID, results[0].ID)
	})

	// Test that context without user ID can only see shared records
	t.Run("entity-only context can only see shared records", func(t *testing.T) {
		results, err := store.Retrieve(ctxEntityOnly, ltm.LTMQuery{})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, sharedID, results[0].ID)
	})
}

func TestPostgresStore_Update(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	store := NewPostgresStore(pool)
	entityID := entity.EntityID("test-entity")
	ctx := entity.ContextWithEntityID(context.Background(), entityID)

	// Create and store a record
	originalRecord := ltm.MemoryRecord{
		EntityID:    entityID,
		Content:     "original content",
		AccessLevel: entity.SharedWithinEntity,
		Metadata: map[string]interface{}{
			"original": true,
		},
	}
	id, err := store.Store(ctx, originalRecord)
	require.NoError(t, err)

	// Update the record
	updatedRecord := ltm.MemoryRecord{
		ID:          id,
		EntityID:    entityID,
		Content:     "updated content",
		AccessLevel: entity.SharedWithinEntity,
		Metadata: map[string]interface{}{
			"updated": true,
		},
	}
	err = store.Update(ctx, updatedRecord)
	assert.NoError(t, err)

	// Retrieve and verify update
	results, err := store.Retrieve(ctx, ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": id,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "updated content", results[0].Content)
	assert.Equal(t, true, results[0].Metadata["updated"])
	assert.NotContains(t, results[0].Metadata, "original")
}

func TestPostgresStore_Delete(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	store := NewPostgresStore(pool)
	entityID := entity.EntityID("test-entity")
	ctx := entity.ContextWithEntityID(context.Background(), entityID)

	// Create and store a record
	record := ltm.MemoryRecord{
		EntityID:    entityID,
		Content:     "record to delete",
		AccessLevel: entity.SharedWithinEntity,
	}
	id, err := store.Store(ctx, record)
	require.NoError(t, err)

	// Delete the record
	err = store.Delete(ctx, id)
	assert.NoError(t, err)

	// Verify record was deleted
	results, err := store.Retrieve(ctx, ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": id,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, results, 0, "Record should be deleted")
}
