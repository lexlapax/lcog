//go:build integration
// +build integration

package openai_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lexlapax/cogmem/pkg/reasoning/adapters/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getAPIKey() string {
	// Try loading from environment variable first
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		return apiKey
	}
	
	// Use the key from the config file
	return apiKey
}

func TestIntegration_GenerateEmbeddings(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Get API key
	apiKey := getAPIKey()
	require.NotEmpty(t, apiKey, "OpenAI API key is empty")

	// Create adapter
	config := openai.Config{
		APIKey:         apiKey,
		EmbeddingModel: "text-embedding-3-small", // Using a smaller model for testing
	}
	adapter, err := openai.NewOpenAIAdapter(config)
	require.NoError(t, err)

	// Test with simple texts
	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning models can process and generate human language",
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Generate embeddings
	embeddings, err := adapter.GenerateEmbeddings(ctx, texts)
	require.NoError(t, err, "Failed to generate embeddings")
	require.Len(t, embeddings, 2, "Should return 2 embeddings")

	// Verify embedding dimensions
	expectedDimension := 1536 // text-embedding-3-small has 1536-dimensional vectors
	assert.Len(t, embeddings[0], expectedDimension, "First embedding should have correct dimension")
	assert.Len(t, embeddings[1], expectedDimension, "Second embedding should have correct dimension")

	// Verify embeddings are different
	equal := true
	for i := 0; i < expectedDimension; i++ {
		if embeddings[0][i] != embeddings[1][i] {
			equal = false
			break
		}
	}
	assert.False(t, equal, "Embeddings for different texts should be different")
}

func TestIntegration_Process(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Get API key
	apiKey := getAPIKey()
	require.NotEmpty(t, apiKey, "OpenAI API key is empty")

	// Create adapter
	config := openai.Config{
		APIKey:    apiKey,
		ChatModel: "gpt-3.5-turbo", // Using a smaller model for testing
	}
	adapter, err := openai.NewOpenAIAdapter(config)
	require.NoError(t, err)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test simple prompt
	prompt := "What is the capital of France? Respond with just one word."
	response, err := adapter.Process(ctx, prompt)
	require.NoError(t, err, "Failed to get response")
	
	// Verify we get a reasonable response (should contain "Paris" in some form)
	t.Logf("Response to 'capital of France' query: %s", response)
	assert.Contains(t, response, "Paris", "Response should mention Paris")
}
