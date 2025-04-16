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

func TestPgVectorSemanticSearchAndEntityIsolation(t *testing.T) {
	// Skip this test if not running in integration test mode
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Get database connection string from environment variable or use default
	pgvectorURL := os.Getenv("PGVECTOR_TEST_URL")
	if pgvectorURL == "" {
		pgvectorURL = os.Getenv("TEST_DB_URL")
		if pgvectorURL == "" {
			pgvectorURL = os.Getenv("POSTGRES_URL")
			if pgvectorURL == "" {
				pgvectorURL = "postgres://postgres:postgres@localhost:5432/cogmem_test?sslmode=disable"
			}
		}
	}

	t.Logf("Using database URL: %s", pgvectorURL)

	// Create a random table name for tests to avoid conflicts
	tableName := "test_vector_" + uuid.New().String()[:8]

	// Set up a temporary connection to create the pgvector extension
	tempConfig, err := pgxpool.ParseConfig(pgvectorURL)
	require.NoError(t, err, "Failed to parse connection string")
	
	tempCtx, tempCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer tempCancel()
	
	tempPool, err := pgxpool.NewWithConfig(tempCtx, tempConfig)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer tempPool.Close()
	
	// Enable pgvector extension if not already enabled
	_, err = tempPool.Exec(tempCtx, "CREATE EXTENSION IF NOT EXISTS vector")
	require.NoError(t, err, "Failed to enable vector extension")
	
	// Set up the adapter with realistic dimensions for semantic vectors
	ctx := context.Background()
	pgConfig := pgvector.PgvectorConfig{
		ConnectionString: pgvectorURL,
		TableName:        tableName,
		DimensionSize:    1536, // Realistic dimension for embeddings (OpenAI)
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

	// Test 1: Semantic search with realistic embeddings
	t.Run("Semantic Search", func(t *testing.T) {
		// Create a test entity
		entityID := entity.EntityID("test-entity-1")
		userID := "test-user-1"

		// Create test records with realistic embeddings
		// These are 1536-dimensional vectors that should be similar to their respective topics
		databaseRecord := createMemoryRecord(entityID, userID, "PostgreSQL is an open-source relational database management system.", createRandomEmbedding(1536))
		vectorRecord := createMemoryRecord(entityID, userID, "Vector databases are optimized for storing and querying high-dimensional vectors.", createRandomEmbedding(1536))
		embeddingRecord := createMemoryRecord(entityID, userID, "Embeddings convert text or other data into high-dimensional vectors that capture semantic meaning.", createRandomEmbedding(1536))

		// Store all records
		_, err := adapter.Store(ctx, databaseRecord)
		require.NoError(t, err)
		_, err = adapter.Store(ctx, vectorRecord)
		require.NoError(t, err)
		_, err = adapter.Store(ctx, embeddingRecord)
		require.NoError(t, err)

		// Create a "database" related query embedding
		databaseQueryEmbedding := createRandomEmbedding(1536)
		// Make it somewhat similar to the database record embedding
		for i := 0; i < 300; i++ {
			databaseQueryEmbedding[i] = databaseRecord.Embedding[i]
		}

		// Perform semantic search for database-related content
		query := ltm.LTMQuery{
			Filters: map[string]interface{}{
				"entity_id": entityID,
			},
			Embedding: databaseQueryEmbedding,
			Limit:     2,
		}

		results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		assert.NotEmpty(t, results, "Should find at least one result for semantic search")

		// Now create a "vector" related query embedding
		vectorQueryEmbedding := createRandomEmbedding(1536)
		// Make it somewhat similar to the vector record embedding
		for i := 0; i < 300; i++ {
			vectorQueryEmbedding[i] = vectorRecord.Embedding[i]
		}

		// Perform semantic search for vector-related content
		query = ltm.LTMQuery{
			Filters: map[string]interface{}{
				"entity_id": entityID,
			},
			Embedding: vectorQueryEmbedding,
			Limit:     2,
		}

		results, err = adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		assert.NotEmpty(t, results, "Should find at least one result for semantic search")
	})

	// Test 2: Entity Isolation
	t.Run("Entity Isolation", func(t *testing.T) {
		// Create two different entities
		entityID1 := entity.EntityID("test-entity-isolation-1")
		entityID2 := entity.EntityID("test-entity-isolation-2")
		userID := "test-user-1"

		// Create test records for entity 1
		record1 := createMemoryRecord(entityID1, userID, "This memory belongs to entity 1", createRandomEmbedding(1536))
		record2 := createMemoryRecord(entityID1, userID, "This is another memory for entity 1", createRandomEmbedding(1536))

		// Create test record for entity 2
		record3 := createMemoryRecord(entityID2, userID, "This memory belongs to entity 2", createRandomEmbedding(1536))

		// Store all records
		_, err := adapter.Store(ctx, record1)
		require.NoError(t, err)
		_, err = adapter.Store(ctx, record2)
		require.NoError(t, err)
		_, err = adapter.Store(ctx, record3)
		require.NoError(t, err)

		// Query memories for entity 1
		query := ltm.LTMQuery{
			Filters: map[string]interface{}{
				"entity_id": entityID1,
			},
		}

		entity1Results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(entity1Results), "Entity 1 should have exactly 2 memories")

		// Verify content of entity 1 memories
		for _, result := range entity1Results {
			assert.Equal(t, entityID1, result.EntityID)
			assert.Contains(t, result.Content, "entity 1")
		}

		// Query memories for entity 2
		query = ltm.LTMQuery{
			Filters: map[string]interface{}{
				"entity_id": entityID2,
			},
		}

		entity2Results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(entity2Results), "Entity 2 should have exactly 1 memory")

		// Verify content of entity 2 memory
		assert.Equal(t, entityID2, entity2Results[0].EntityID)
		assert.Contains(t, entity2Results[0].Content, "entity 2")
	})

	// Test 3: Entity Isolation with Semantic Search
	t.Run("Entity Isolation with Semantic Search", func(t *testing.T) {
		// Create two different entities
		entityID1 := entity.EntityID("test-semantic-entity-1")
		entityID2 := entity.EntityID("test-semantic-entity-2")
		userID := "test-user-1"

		// Create common embeddings to use across entities
		embedding1 := createRandomEmbedding(1536)
		embedding2 := createRandomEmbedding(1536)

		// Create similar records for both entities with the same embeddings
		record1Entity1 := createMemoryRecord(entityID1, userID, "Entity 1: PostgreSQL database information", embedding1)
		record2Entity1 := createMemoryRecord(entityID1, userID, "Entity 1: Vector embedding information", embedding2)

		record1Entity2 := createMemoryRecord(entityID2, userID, "Entity 2: PostgreSQL database information", embedding1)
		record2Entity2 := createMemoryRecord(entityID2, userID, "Entity 2: Vector embedding information", embedding2)

		// Store all records
		_, err := adapter.Store(ctx, record1Entity1)
		require.NoError(t, err)
		_, err = adapter.Store(ctx, record2Entity1)
		require.NoError(t, err)
		_, err = adapter.Store(ctx, record1Entity2)
		require.NoError(t, err)
		_, err = adapter.Store(ctx, record2Entity2)
		require.NoError(t, err)

		// Create a query embedding similar to the first embedding
		queryEmbedding := make([]float32, 1536)
		copy(queryEmbedding, embedding1)

		// Perform semantic search for entity 1
		query := ltm.LTMQuery{
			Filters: map[string]interface{}{
				"entity_id": entityID1,
			},
			Embedding: queryEmbedding,
			Limit:     5,
		}

		entity1Results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		for _, result := range entity1Results {
			assert.Equal(t, entityID1, result.EntityID, "Semantic search should only return results for entity 1")
			assert.Contains(t, result.Content, "Entity 1:")
		}

		// Perform semantic search for entity 2
		query = ltm.LTMQuery{
			Filters: map[string]interface{}{
				"entity_id": entityID2,
			},
			Embedding: queryEmbedding,
			Limit:     5,
		}

		entity2Results, err := adapter.Retrieve(ctx, query)
		assert.NoError(t, err)
		for _, result := range entity2Results {
			assert.Equal(t, entityID2, result.EntityID, "Semantic search should only return results for entity 2")
			assert.Contains(t, result.Content, "Entity 2:")
		}
	})
}

// Helper functions
func createMemoryRecord(entityID entity.EntityID, userID, content string, embedding []float32) ltm.MemoryRecord {
	return ltm.MemoryRecord{
		ID:          uuid.New().String(),
		EntityID:    entityID,
		UserID:      userID,
		AccessLevel: entity.PrivateToUser,
		Content:     content,
		Metadata: map[string]interface{}{
			"source": "integration_test",
		},
		Embedding:  embedding,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func createRandomEmbedding(size int) []float32 {
	embedding := make([]float32, size)
	for i := 0; i < size; i++ {
		// Generate random values between -1 and 1
		embedding[i] = (2*float32(uuid.New().ID()) / float32(0xFFFFFFFFFFFFFFFF)) - 1
	}
	return embedding
}