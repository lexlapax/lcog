// test/testutil/chromem.go
package testutil

import (
	"testing"

	chromem "github.com/philippgille/chromem-go"
)

// CreateTempChromemGoClient creates a new, in-memory chromem-go instance
// suitable for isolated testing.
// It returns the client and a no-op cleanup function, as the instance
// is expected to be garbage collected after the test.
// For more complex scenarios requiring cleanup between test stages within
// a single test function, manual collection cleanup might be needed.
func CreateTempChromemGoClient(t *testing.T) (*chromem.DB, func()) {
	// Create a new in-memory instance for each test.
	// The 'NewDB' function in chromem-go sets up the in-memory state.
	// As of chromem-go v0.7.0, NewDB() doesn't return an error.
	client := chromem.NewDB()

	// Define the cleanup function. For a purely in-memory instance that's
	// test-scoped, garbage collection is often sufficient.
	// If tests require explicit cleanup (e.g., deleting collections),
	// this function could be enhanced, or cleanup done manually in the test.
	cleanupFunc := func() {
		// No-op for now, relying on GC for the in-memory instance.
		// If chromem-go introduces explicit Close() or Reset() methods later,
		// they could be called here.
		// t.Log("Chromem-go test instance cleanup (no-op)")
	}

	return client, cleanupFunc
}

// --- Deprecated/Removed ---
// The StartChromaDBContainer function and ChromaDBContainer struct
// are removed as they were based on the incorrect Docker approach.
