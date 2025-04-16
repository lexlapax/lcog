# PostgreSQL Adapter Testing Guide

This guide outlines the test coverage for PostgreSQL adapters implemented in the CogMem project. The project now has comprehensive test coverage for all PostgreSQL implementations across different storage types: Key-Value (KV), SQL, and Vector.

## PostgreSQL Adapters

CogMem supports three PostgreSQL-based storage adapters:

1. **PostgreSQL Hstore KV Adapter**
   - Implementation: `/cogmem-go/pkg/mem/ltm/adapters/kv/postgres/postgres_hstore.go`
   - Unit Tests: `/cogmem-go/pkg/mem/ltm/adapters/kv/postgres/postgres_hstore_test.go`
   - Integration Tests: `/cogmem-go/test/integration/ltm_postgres_hstore_test.go`

2. **PostgreSQL SQL Adapter**
   - Implementation: `/cogmem-go/pkg/mem/ltm/adapters/sqlstore/postgres/postgres.go`
   - Unit Tests: `/cogmem-go/pkg/mem/ltm/adapters/sqlstore/postgres/postgres_test.go`
   - Integration Tests: `/cogmem-go/test/integration/ltm_postgres_sqlstore_test.go`

3. **PostgreSQL PgVector Adapter**
   - Implementation: `/cogmem-go/pkg/mem/ltm/adapters/vector/pgvector/pgvector.go`
   - Unit Tests: `/cogmem-go/pkg/mem/ltm/adapters/vector/pgvector/pgvector_test.go`
   - Integration Tests: `/cogmem-go/test/integration/ltm_pgvector_test.go`

## Test Coverage

All PostgreSQL adapters now have consistent test coverage for:

- Basic CRUD operations (Create, Read, Update, Delete)
- Entity isolation (ensuring data from one entity is not accessible to another)
- Access level control (private vs. shared records)
- Text search functionality
- Metadata filtering
- Error handling for non-existent records
- Cross-entity operation rejection
- JSON/metadata handling

## Running PostgreSQL Tests

### Prerequisites

1. A running PostgreSQL instance with:
   - The `pgvector` extension installed
   - The `hstore` extension installed
   - A `cogmem_test` database created

### Environment Variables

Set the following environment variables:

```bash
# Enable integration tests
export INTEGRATION_TESTS=true

# PostgreSQL connection strings
export TEST_DB_URL=postgres://postgres:postgres@localhost:5432/cogmem_test?sslmode=disable
export PGVECTOR_TEST_URL=postgres://postgres:postgres@localhost:5432/cogmem_test?sslmode=disable
export HSTORE_TEST_URL=postgres://postgres:postgres@localhost:5432/cogmem_test?sslmode=disable
```

### Using Makefile Commands

We've added a new Makefile target to specifically run PostgreSQL-related tests:

```bash
# Start PostgreSQL and create test database
make test-db-setup

# Run all PostgreSQL-specific tests (HStore, SQLStore, and PgVector adapters)
make test-postgres

# Run all integration tests (including PostgreSQL)
make test-integration

# Clean up
make drop-test-db
make dev-db-down
```

## Sample Test Database Setup

```bash
# Start the database container
docker-compose -f docker-compose.dev.yml up -d

# Create test database
docker exec -it cogmem_postgres psql -U postgres -c "CREATE DATABASE cogmem_test;"

# Enable required extensions
docker exec -it cogmem_postgres psql -U postgres -d cogmem_test -c "CREATE EXTENSION IF NOT EXISTS hstore;"
docker exec -it cogmem_postgres psql -U postgres -d cogmem_test -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

## Troubleshooting

If tests fail, check:

1. Database connectivity - verify connection strings are correct
2. Extensions - ensure `pgvector` and `hstore` are enabled in the test database
3. Database permissions - ensure the user has necessary permissions
4. Tables - verify no lingering test tables are causing conflicts

## Future Improvements

1. Add performance benchmarks comparing different PostgreSQL adapters
2. Enhance error handling for database connection issues
3. Add more complex query tests using advanced PostgreSQL features