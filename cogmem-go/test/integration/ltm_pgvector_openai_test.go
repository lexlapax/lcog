//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/vector/pgvector"
	"github.com/lexlapax/cogmem/pkg/reasoning/adapters/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgVectorWithRealOpenAIEmbeddings(t *testing.T) {
	// Skip this test if not running in integration test mode
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Check for OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test that requires OpenAI API key; set OPENAI_API_KEY to run")
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
	tableName := "test_openai_" + uuid.New().String()[:8]

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
	
	// Initialize OpenAI adapter for generating embeddings
	openaiConfig := openai.Config{
		APIKey:         apiKey,
		ChatModel:      "gpt-4", // Not used for embeddings
		EmbeddingModel: "text-embedding-3-small",
	}
	
	openaiAdapter, err := openai.NewOpenAIAdapter(openaiConfig)
	require.NoError(t, err, "Failed to initialize OpenAI adapter")
	
	// Set up the pgvector adapter
	ctx := context.Background()
	pgConfig := pgvector.PgvectorConfig{
		ConnectionString: pgvectorURL,
		TableName:        tableName,
		DimensionSize:    1536, // OpenAI text-embedding-3-small uses 1536 dimensions
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

	// Test 1: Store and retrieve with real OpenAI embeddings
	t.Run("Store and Retrieve with OpenAI Embeddings", func(t *testing.T) {
		// Create a test entity
		entityID := entity.EntityID("test-entity-openai")
		userID := "test-user-openai"

		// Create sample texts with different topics
		texts := []string{
			"PostgreSQL is an advanced open-source relational database management system.",
			"Machine learning algorithms learn patterns from data without explicit programming.",
			"Quantum computing uses quantum mechanics to process information in new ways.",
			"Vector databases are optimized for storing and retrieving high-dimensional vectors efficiently.",
			"Climate change refers to long-term shifts in temperatures and weather patterns.",
		}

		// Generate OpenAI embeddings for each text
		for i, text := range texts {
			// Generate embedding
			embeddings, err := openaiAdapter.GenerateEmbeddings(ctx, []string{text})
			require.NoError(t, err, "Failed to generate embedding for text %d", i)
			require.Len(t, embeddings, 1, "Should generate one embedding")
			embedding := embeddings[0]
			require.Len(t, embedding, 1536, "OpenAI embedding should have 1536 dimensions")

			// Create and store record
			record := ltm.MemoryRecord{
				ID:          uuid.New().String(),
				EntityID:    entityID,
				UserID:      userID,
				AccessLevel: entity.PrivateToUser,
				Content:     text,
				Metadata: map[string]interface{}{
					"source": "openai_test",
					"index":  i,
				},
				Embedding:  embedding,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			// Store in pgvector
			_, err = adapter.Store(ctx, record)
			require.NoError(t, err, "Failed to store record %d", i)
		}

		// Test semantic search with real embeddings
		searchQueries := []struct {
			query            string
			expectedContains []string
			notExpected      []string
		}{
			{
				query:            "How do database systems work?",
				expectedContains: []string{"PostgreSQL", "database"},
				notExpected:      []string{"quantum", "climate"},
			},
			{
				query:            "Tell me about AI and machine learning",
				expectedContains: []string{"Machine learning", "algorithms", "patterns"},
				notExpected:      []string{"PostgreSQL", "Climate"},
			},
			{
				query:            "What are vector embeddings used for?",
				expectedContains: []string{"Vector", "high-dimensional"},
				notExpected:      []string{"Climate", "Quantum"},
			},
		}

		for i, tc := range searchQueries {
			// Generate embedding for search query
			queryEmbeddings, err := openaiAdapter.GenerateEmbeddings(ctx, []string{tc.query})
			require.NoError(t, err, "Failed to generate embedding for query %d", i)
			queryEmbedding := queryEmbeddings[0]

			// Perform semantic search
			query := ltm.LTMQuery{
				Filters: map[string]interface{}{
					"entity_id": entityID,
				},
				Embedding: queryEmbedding,
				Limit:     2, // Get top 2 results
			}

			results, err := adapter.Retrieve(ctx, query)
			assert.NoError(t, err, "Semantic search should not error for query %d", i)
			require.NotEmpty(t, results, "Should find at least one result for query %d", i)

			// Check if expected content is in the results
			foundExpected := false
			for _, result := range results {
				for _, expected := range tc.expectedContains {
					if contains(result.Content, expected) {
						foundExpected = true
						break
					}
				}

				// Check that unexpected content is not in top results
				for _, notExpected := range tc.notExpected {
					assert.False(t, contains(result.Content, notExpected), 
						"Query %d should not return content with %s but got: %s", 
						i, notExpected, result.Content)
				}
			}

			assert.True(t, foundExpected, "Query %d should find content containing expected terms", i)
		}
	})

	// Test 2: Entity Isolation with Real Embeddings
	t.Run("Entity Isolation with Real Embeddings", func(t *testing.T) {
		// Create two different entities
		entityID1 := entity.EntityID("test-entity-isolation-1")
		entityID2 := entity.EntityID("test-entity-isolation-2")
		userID := "test-user-1"

		// Test texts that are semantically similar but for different entities
		entity1Texts := []string{
			"Entity 1: Information about database systems and SQL.",
			"Entity 1: Details about vector embeddings and similarity search.",
		}

		entity2Texts := []string{
			"Entity 2: Information about database systems and SQL.",
			"Entity 2: Details about vector embeddings and similarity search.",
		}

		// Store records for entity 1
		for _, text := range entity1Texts {
			embeddings, err := openaiAdapter.GenerateEmbeddings(ctx, []string{text})
			require.NoError(t, err)
			embedding := embeddings[0]

			record := ltm.MemoryRecord{
				ID:          uuid.New().String(),
				EntityID:    entityID1,
				UserID:      userID,
				AccessLevel: entity.PrivateToUser,
				Content:     text,
				Metadata:    map[string]interface{}{"source": "openai_test"},
				Embedding:   embedding,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			_, err = adapter.Store(ctx, record)
			require.NoError(t, err)
		}

		// Store records for entity 2
		for _, text := range entity2Texts {
			embeddings, err := openaiAdapter.GenerateEmbeddings(ctx, []string{text})
			require.NoError(t, err)
			embedding := embeddings[0]

			record := ltm.MemoryRecord{
				ID:          uuid.New().String(),
				EntityID:    entityID2,
				UserID:      userID,
				AccessLevel: entity.PrivateToUser,
				Content:     text,
				Metadata:    map[string]interface{}{"source": "openai_test"},
				Embedding:   embedding,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			_, err = adapter.Store(ctx, record)
			require.NoError(t, err)
		}

		// Test queries that should be semantically similar to both entities' content
		searchQueries := []string{
			"Information about SQL databases",
			"Tell me about vector embeddings",
		}

		for _, query := range searchQueries {
			// Generate embedding for query
			queryEmbeddings, err := openaiAdapter.GenerateEmbeddings(ctx, []string{query})
			require.NoError(t, err)
			queryEmbedding := queryEmbeddings[0]

			// Search for entity 1
			entity1Query := ltm.LTMQuery{
				Filters: map[string]interface{}{
					"entity_id": entityID1,
				},
				Embedding: queryEmbedding,
				Limit:     5,
			}

			entity1Results, err := adapter.Retrieve(ctx, entity1Query)
			assert.NoError(t, err)
			require.NotEmpty(t, entity1Results)

			// Verify all results belong to entity 1
			for _, result := range entity1Results {
				assert.Equal(t, entityID1, result.EntityID)
				assert.Contains(t, result.Content, "Entity 1:")
			}

			// Search for entity 2
			entity2Query := ltm.LTMQuery{
				Filters: map[string]interface{}{
					"entity_id": entityID2,
				},
				Embedding: queryEmbedding,
				Limit:     5,
			}

			entity2Results, err := adapter.Retrieve(ctx, entity2Query)
			assert.NoError(t, err)
			require.NotEmpty(t, entity2Results)

			// Verify all results belong to entity 2
			for _, result := range entity2Results {
				assert.Equal(t, entityID2, result.EntityID)
				assert.Contains(t, result.Content, "Entity 2:")
			}
		}
	})
}

// Helper function to check if a string contains another string (case-insensitive)
func contains(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}