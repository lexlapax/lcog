package mock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockStore_Store(t *testing.T) {
	mockStore := NewMockStore()
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
	id, err := mockStore.Store(ctx, record)
	assert.NoError(t, err)
	assert.NotEmpty(t, id, "ID should be generated and not empty")

	// Verify record was stored with correct values
	records, err := mockStore.Retrieve(ctx, ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": id,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, entityID, records[0].EntityID)
	assert.Equal(t, record.Content, records[0].Content)
	assert.Equal(t, record.AccessLevel, records[0].AccessLevel)
	assert.Equal(t, record.Metadata["key1"], records[0].Metadata["key1"])
	assert.Equal(t, record.Metadata["key2"], records[0].Metadata["key2"])
	assert.WithinDuration(t, time.Now(), records[0].CreatedAt, 2*time.Second)
	assert.WithinDuration(t, time.Now(), records[0].UpdatedAt, 2*time.Second)
}

func TestMockStore_Store_MissingEntityContext(t *testing.T) {
	mockStore := NewMockStore()
	
	// Context without entity ID
	ctx := context.Background()
	
	// Create a test record
	record := ltm.MemoryRecord{
		EntityID:    entity.EntityID("test-entity"),
		Content:     "test memory content",
		AccessLevel: entity.SharedWithinEntity,
	}

	// Test storing without entity context
	_, err := mockStore.Store(ctx, record)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrMissingEntityContext))
}

func TestMockStore_Retrieve(t *testing.T) {
	mockStore := NewMockStore()
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
		id, err := mockStore.Store(ctx, record)
		require.NoError(t, err)
		ids = append(ids, id)
	}

	// Test retrieving by content
	t.Run("retrieve by text", func(t *testing.T) {
		results, err := mockStore.Retrieve(ctx, ltm.LTMQuery{
			Text: "searchable",
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0].Content, "searchable")
	})

	// Test retrieving by metadata field
	t.Run("retrieve by metadata", func(t *testing.T) {
		results, err := mockStore.Retrieve(ctx, ltm.LTMQuery{
			Filters: map[string]interface{}{
				"type": "note",
			},
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "note", results[0].Metadata["type"])
	})

	// Test retrieving with exact match
	t.Run("retrieve by exact match", func(t *testing.T) {
		results, err := mockStore.Retrieve(ctx, ltm.LTMQuery{
			ExactMatch: map[string]interface{}{
				"ID": ids[0],
			},
		})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, ids[0], results[0].ID)
	})
}

func TestMockStore_RetrieveIsolation(t *testing.T) {
	mockStore := NewMockStore()
	
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
	idA, err := mockStore.Store(ctxA, recordA)
	require.NoError(t, err)

	// Create and store record for Entity B
	recordB := ltm.MemoryRecord{
		EntityID:    entityB,
		Content:     "entity B record",
		AccessLevel: entity.SharedWithinEntity,
	}
	idB, err := mockStore.Store(ctxB, recordB)
	require.NoError(t, err)

	// Entity A should only see its own records
	resultsA, err := mockStore.Retrieve(ctxA, ltm.LTMQuery{})
	assert.NoError(t, err)
	assert.Len(t, resultsA, 1)
	assert.Equal(t, entityA, resultsA[0].EntityID)
	assert.Equal(t, idA, resultsA[0].ID)

	// Entity B should only see its own records
	resultsB, err := mockStore.Retrieve(ctxB, ltm.LTMQuery{})
	assert.NoError(t, err)
	assert.Len(t, resultsB, 1)
	assert.Equal(t, entityB, resultsB[0].EntityID)
	assert.Equal(t, idB, resultsB[0].ID)

	// Entity A should not be able to access Entity B's record by ID
	resultsA, err = mockStore.Retrieve(ctxA, ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": idB,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, resultsA, 0, "Entity A should not be able to access Entity B's records")
}

func TestMockStore_AccessLevel(t *testing.T) {
	mockStore := NewMockStore()
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
	privateIDa, err := mockStore.Store(ctxUserA, privateRecordA)
	require.NoError(t, err)

	// Create and store a shared record
	sharedRecord := ltm.MemoryRecord{
		EntityID:    entityID,
		AccessLevel: entity.SharedWithinEntity,
		Content:     "shared within entity",
	}
	sharedID, err := mockStore.Store(ctxEntityOnly, sharedRecord)
	require.NoError(t, err)

	// Test that User A can see both their private record and the shared record
	t.Run("user A can see their private and shared records", func(t *testing.T) {
		results, err := mockStore.Retrieve(ctxUserA, ltm.LTMQuery{})
		assert.NoError(t, err)
		assert.Len(t, results, 2)
		
		// Check that both IDs are present
		ids := []string{results[0].ID, results[1].ID}
		assert.Contains(t, ids, privateIDa)
		assert.Contains(t, ids, sharedID)
	})

	// Test that User B can only see the shared record
	t.Run("user B can only see shared records", func(t *testing.T) {
		results, err := mockStore.Retrieve(ctxUserB, ltm.LTMQuery{})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, sharedID, results[0].ID)
	})

	// Test that context without user ID can only see shared records
	t.Run("entity-only context can only see shared records", func(t *testing.T) {
		results, err := mockStore.Retrieve(ctxEntityOnly, ltm.LTMQuery{})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, sharedID, results[0].ID)
	})
}

func TestMockStore_Update(t *testing.T) {
	mockStore := NewMockStore()
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
	id, err := mockStore.Store(ctx, originalRecord)
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
	err = mockStore.Update(ctx, updatedRecord)
	assert.NoError(t, err)

	// Retrieve and verify update
	results, err := mockStore.Retrieve(ctx, ltm.LTMQuery{
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

func TestMockStore_Update_EntityIsolation(t *testing.T) {
	mockStore := NewMockStore()
	
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
	idA, err := mockStore.Store(ctxA, recordA)
	require.NoError(t, err)

	// Try to update Entity A's record from Entity B's context
	updatedRecord := ltm.MemoryRecord{
		ID:          idA,
		EntityID:    entityA, // Still has Entity A's ID
		Content:     "attempted update from entity B",
		AccessLevel: entity.SharedWithinEntity,
	}
	err = mockStore.Update(ctxB, updatedRecord)
	assert.Error(t, err, "Should error when Entity B tries to update Entity A's record")

	// Verify record was not changed
	results, err := mockStore.Retrieve(ctxA, ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": idA,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "entity A record", results[0].Content)
}

func TestMockStore_Delete(t *testing.T) {
	mockStore := NewMockStore()
	entityID := entity.EntityID("test-entity")
	ctx := entity.ContextWithEntityID(context.Background(), entityID)

	// Create and store a record
	record := ltm.MemoryRecord{
		EntityID:    entityID,
		Content:     "record to delete",
		AccessLevel: entity.SharedWithinEntity,
	}
	id, err := mockStore.Store(ctx, record)
	require.NoError(t, err)

	// Delete the record
	err = mockStore.Delete(ctx, id)
	assert.NoError(t, err)

	// Verify record was deleted
	results, err := mockStore.Retrieve(ctx, ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": id,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, results, 0, "Record should be deleted")
}

func TestMockStore_Delete_EntityIsolation(t *testing.T) {
	mockStore := NewMockStore()
	
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
	idA, err := mockStore.Store(ctxA, recordA)
	require.NoError(t, err)

	// Try to delete Entity A's record from Entity B's context
	err = mockStore.Delete(ctxB, idA)
	assert.Error(t, err, "Should error when Entity B tries to delete Entity A's record")

	// Verify record was not deleted
	results, err := mockStore.Retrieve(ctxA, ltm.LTMQuery{
		ExactMatch: map[string]interface{}{
			"ID": idA,
		},
	})
	assert.NoError(t, err)
	assert.Len(t, results, 1, "Record should still exist")
}
