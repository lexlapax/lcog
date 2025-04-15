package integration

import (
	"database/sql"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// TestMigrations verifies that migrations can be applied and rolled back successfully.
func TestMigrations(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run.")
	}

	// Get database connection string from environment or use default
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/cogmem_test?sslmode=disable"
	}

	// Connect to the database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Create migrate instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Fatalf("Failed to create migration driver: %v", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		"file://../../migrations",
		"postgres", driver,
	)
	if err != nil {
		t.Fatalf("Failed to create migrator: %v", err)
	}

	// Drop all tables to start clean
	if err := migrator.Drop(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to drop database: %v", err)
	}

	// Apply migrations
	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Verify memory_records table exists
	var tableExists bool
	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'memory_records')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if table exists: %v", err)
	}

	if !tableExists {
		t.Fatal("memory_records table was not created by migrations")
	}

	// Roll back migrations
	if err := migrator.Down(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Failed to roll back migrations: %v", err)
	}

	// Verify memory_records table was dropped
	err = db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'memory_records')").Scan(&tableExists)
	if err != nil {
		t.Fatalf("Failed to check if table exists: %v", err)
	}

	if tableExists {
		t.Fatal("memory_records table was not dropped by down migration")
	}
}