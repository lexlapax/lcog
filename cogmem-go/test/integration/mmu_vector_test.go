//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning/adapters/openai"
	"github.com/lexlapax/cogmem/test/testutil"
)

func init() {
	// Try multiple locations for .env file
	if err := godotenv.Load(); err != nil {
		// Try project root
		_ = godotenv.Load("../../.env")
	}
}

func getOpenAIAPIKey() string {
	return os.Getenv("OPENAI_API_KEY")
}

// TestMmuWithOpenAI tests the MMU vector operations with just the OpenAI adapter
func TestMmuWithOpenAI(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Get OpenAI API key
	apiKey := getOpenAIAPIKey()
	if apiKey == "" {
		t.Skip("Skipping test because OPENAI_API_KEY is not set")
	}

	// Create a mock LTM store with vector capability
	ltmStore := testutil.NewMockVectorStore()

	// Create the OpenAI adapter
	reasoningEngine, err := openai.NewOpenAIAdapter(openai.Config{
		APIKey: apiKey,
		EmbeddingModel: "text-embedding-3-small", // Use a smaller model for testing
		ChatModel: "gpt-3.5-turbo", // Use a smaller model for testing
	})
	require.NoError(t, err)

	// Create the MMU
	mmuInstance := mmu.NewMMU(
		ltmStore,
		reasoningEngine,
		nil, // No script engine for this test
		mmu.Config{
			EnableLuaHooks: false,
			EnableVectorOperations: true,
			WorkingMemoryLimit: 10,
		},
	)

	// Create a context with entity
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)

	// Test storing content that should generate embeddings
	testContent := "Artificial intelligence systems process information similar to humans."
	
	// Store content
	memoryID, err := mmuInstance.EncodeToLTM(ctx, testContent)
	require.NoError(t, err)
	require.NotEmpty(t, memoryID)
	
	// Verify embedding was generated
	storedRecord := ltmStore.GetRecord(memoryID)
	require.NotNil(t, storedRecord, "Record should be stored")
	assert.Equal(t, testContent, storedRecord.Content, "Content should match")
	assert.NotEmpty(t, storedRecord.Embedding, "Embedding should be generated")
	assert.Greater(t, len(storedRecord.Embedding), 100, "Embedding should have significant dimension")
	
	// Test semantic search
	queryText := "neural networks and machine learning"
	options := mmu.RetrievalOptions{
		MaxResults: 5,
		Strategy:   "semantic",
	}
	
	// Test that semantic search triggers embedding generation for the query
	_, err = mmuInstance.RetrieveFromLTM(ctx, queryText, options)
	require.NoError(t, err)
	
	// Verify the query was embedded by checking the lastQueryEmbedding in our mock
	queryEmbedding := ltmStore.GetLastQueryEmbedding()
	assert.NotEmpty(t, queryEmbedding, "Query embedding should be generated")
	assert.Greater(t, len(queryEmbedding), 100, "Query embedding should have significant dimension")
}