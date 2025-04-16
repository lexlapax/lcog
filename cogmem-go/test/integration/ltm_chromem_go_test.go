//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/vector/chromem_go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChromemGoLTMIntegration(t *testing.T) {
	// Skip this test if not running in integration test mode
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Create a temporary storage path for the test
	tempDir := t.TempDir()
	collectionName := "integration-test-" + uuid.New().String()[:8]

	// Set up the adapter
	ctx := context.Background()
	// Use dimension size that matches our test embeddings (5 dimensions)
	dimensions := 5
	config := chromem_go.ChromemGoConfig{
		Collection:  collectionName,
		StoragePath: tempDir,
		Dimensions:  dimensions,
	}

	adapter, err := chromem_go.NewChromemGoAdapterWithConfig(&config)
	require.NoError(t, err)
	require.NotNil(t, adapter)

	// Create entity data
	entityID := entity.EntityID("test-entity-" + uuid.New().String())
	userID := "test-user-1"

	// Test basic store and retrieve operations
	t.Run("Basic Store and Retrieve", func(t *testing.T) {
		// Create a test record
		record := ltm.MemoryRecord{
			ID:          uuid.New().String(),
			EntityID:    entityID,
			UserID:      userID,
			AccessLevel: entity.PrivateToUser,
			Content:     "This is a test memory for ChromemGo",
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

	// Test update operation
	t.Run("Update Record", func(t *testing.T) {
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
		assert.NotNil(t, updated.Metadata["updated"])
		assert.NotEmpty(t, updated.Embedding)
	})

	// Verify vector search capability
	t.Run("Vector Search", func(t *testing.T) {
		// Check if the adapter supports vector search
		assert.True(t, adapter.SupportsVectorSearch())
	})
}