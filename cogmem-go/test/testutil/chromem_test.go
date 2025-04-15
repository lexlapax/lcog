// test/testutil/chromem_test.go
package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTempChromemGoClient(t *testing.T) {
	client, cleanup := CreateTempChromemGoClient(t)
	require.NotNil(t, client, "Client should not be nil")
	require.NotNil(t, cleanup, "Cleanup function should not be nil")

	// Defer the cleanup function (even if it's a no-op, it's good practice)
	defer cleanup()

	// Verify basic collection operations
	collectionName := "test-collection-basic"
	metadata := map[string]string{"test": "value"}

	// Define a simple embedding function for testing
	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}

	// Create Collection
	coll, err := client.CreateCollection(collectionName, metadata, embeddingFunc)
	assert.NoError(t, err, "Should be able to create a collection")
	require.NotNil(t, coll, "Created collection should not be nil")
	assert.Equal(t, collectionName, coll.Name, "Collection name should match")

	// List Collections
	collections := client.ListCollections()
	assert.NotNil(t, collections, "Should be able to list collections")
	_, found := collections[collectionName]
	assert.True(t, found, "Newly created collection should be in the list")

	// Get Collection
	fetchedColl := client.GetCollection(collectionName, embeddingFunc)
	require.NotNil(t, fetchedColl, "Fetched collection should not be nil")
	assert.Equal(t, collectionName, fetchedColl.Name, "Fetched collection name should match")

	// Delete Collection
	err = client.DeleteCollection(collectionName)
	assert.NoError(t, err, "Should be able to delete the collection")

	// Verify Deletion
	collections = client.ListCollections()
	_, found = collections[collectionName]
	assert.False(t, found, "Deleted collection should not be in the list")

	t.Log("Successfully tested basic operations on in-memory chromem-go client")
}