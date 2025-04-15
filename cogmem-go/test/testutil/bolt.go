package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

// CreateTempBoltDB creates a temporary BoltDB database for testing purposes.
// It returns the database connection, the file path, and a cleanup function.
func CreateTempBoltDB(t *testing.T) (*bolt.DB, string, func()) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "cogmem_boltdb_test")
	require.NoError(t, err)

	// Create database path
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open the database
	db, err := bolt.Open(dbPath, 0600, nil)
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, dbPath, cleanup
}