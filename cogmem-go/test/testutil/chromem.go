// test/testutil/chromem.go
package testutil

import (
	"testing"

	chromem "github.com/philippgille/chromem-go"
	"github.com/stretchr/testify/require"
)

// CreateTempChromemGoClient creates a new, in-memory chromem-go instance
// suitable for isolated testing.
// It returns the client and a no-op cleanup function, as the instance
// is expected to be garbage collected after the test.
func CreateTempChromemGoClient(t *testing.T) (*chromem.DB, func()) {
	// In version 0.7.0, chromem-go doesn't have Options or persistent storage
	// It's always in-memory for now
	client := chromem.NewDB()

	// Define the cleanup function
	cleanupFunc := func() {
		// No explicit close method in chromem-go yet, 
		// relying on GC for cleanup
	}

	return client, cleanupFunc
}

// CreateTempChromemGoClientOnDisk creates a ChromemGo instance with on-disk storage for testing.
// It creates a temporary directory for the database files and returns the client
// and a cleanup function that will remove the temporary directory.
func CreateTempChromemGoClientOnDisk(t *testing.T) (*chromem.DB, func()) {
	// Create a temporary directory for the database
	tempDir := t.TempDir() // Go's testing package creates a temp dir that's cleaned up after test
	
	// Create a persistent DB using the temporary directory
	// Second parameter is allowConcurrentAccess
	client, err := chromem.NewPersistentDB(tempDir, false)
	require.NoError(t, err, "Failed to create persistent chromem-go client")
	
	// Define cleanup function
	cleanupFunc := func() {
		// t.TempDir() automatically cleans up the directory after test
	}
	
	return client, cleanupFunc
}

// --- Deprecated/Removed ---
// The StartChromaDBContainer function and ChromaDBContainer struct
// are removed as they were based on the incorrect Docker approach.
