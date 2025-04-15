package integration

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrations verifies that migrations can be applied and rolled back successfully.
func TestMigrations(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run.")
	}
	
	// Skip pgvector-dependent migrations test if pgvector is not available
	if os.Getenv("SKIP_PGVECTOR_TESTS") == "true" {
		t.Skip("Skipping pgvector migration tests; set SKIP_PGVECTOR_TESTS=false to run")
	}

	// Get database connection string from environment or use default
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5433/cogmem_test?sslmode=disable"
	}

	// Connect to the database
	db, err := sql.Open("postgres", dbURL)
	require.NoError(t, err, "Failed to connect to database")
	defer db.Close()

	err = db.Ping()
	require.NoError(t, err, "Failed to ping database")

	// Clean up existing tables
	cleanupDatabase(t, db)

	// Set up migration
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	require.NoError(t, err, "Failed to create migration driver")

	migrator, err := migrate.NewWithDatabaseInstance(
		"file://../../migrations",
		"postgres", driver,
	)
	require.NoError(t, err, "Failed to create migrator")
	
	// Apply migrations
	err = migrator.Up()
	if err != nil && err != migrate.ErrNoChange {
		// If pgvector migration fails due to missing extension, just skip this test
		if strings.Contains(err.Error(), "vector.control") {
			t.Skip("Skipping migration test due to missing pgvector extension in test database")
			return
		}
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Verify the memory_records table was created
	assertTableExists(t, db, "memory_records", true)

	// Verify the hstore extension was enabled
	var hasHstore bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pg_extension WHERE extname = 'hstore'
		)
	`).Scan(&hasHstore)
	require.NoError(t, err)
	assert.True(t, hasHstore, "hstore extension should be enabled")

	// Roll back migrations
	err = migrator.Down()
	if err != nil && err != migrate.ErrNoChange {
		t.Logf("Warning: Failed to roll back migrations: %v", err)
		// Don't fail the test here, as we want to continue cleaning up
	}

	// Verify memory_records table was dropped
	assertTableExists(t, db, "memory_records", false)
}

// cleanupDatabase drops all tables in the test database to start with a clean slate
func cleanupDatabase(t *testing.T, db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create the schema_migrations table if it doesn't exist yet
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint NOT NULL,
			dirty boolean NOT NULL,
			PRIMARY KEY (version)
		)
	`)
	require.NoError(t, err, "Failed to create schema_migrations table")

	// Truncate the schema_migrations table
	_, err = db.ExecContext(ctx, "TRUNCATE schema_migrations")
	require.NoError(t, err, "Failed to truncate schema_migrations table")

	// Drop the memory_records table if it exists
	_, err = db.ExecContext(ctx, "DROP TABLE IF EXISTS memory_records")
	require.NoError(t, err, "Failed to drop memory_records table")
}

// assertTableExists checks if a table exists or not and fails the test if the result doesn't match expected
func assertTableExists(t *testing.T, db *sql.DB, tableName string, shouldExist bool) {
	var exists bool
	query := `SELECT EXISTS (
		SELECT FROM information_schema.tables WHERE table_name = $1
	)`
	err := db.QueryRow(query, tableName).Scan(&exists)
	require.NoError(t, err, "Failed to check if table exists")
	
	if shouldExist {
		assert.True(t, exists, "Table %s should exist", tableName)
	} else {
		assert.False(t, exists, "Table %s should not exist", tableName)
	}
}