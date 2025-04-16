# Makefile for cogmem-go

# Variables
GO = go
GOFLAGS = 
PACKAGES = ./...
BINDIR = ./bin
MAIN_PKG = ./cmd/example-client

# Tools
GOLINT = golangci-lint
SQLC = sqlc

# Build flags
BUILD_FLAGS = -v

.PHONY: all build clean test test-verbose test-integration test-cmd test-cmd-postgres test-cmd-script test-cmd-script-mock test-cmd-script-boltdb test-cmd-script-sqlite test-cmd-script-postgres test-cmd-script-chromemgo test-cmd-script-all test-postgres lint fmt sqlc-gen help deps create-test-db drop-test-db test-db-setup

all: build

# Build all packages
build:
	$(GO) build $(GOFLAGS) $(BUILD_FLAGS) $(PACKAGES)

# Build and install example client
install: 
	mkdir -p $(BINDIR)
	$(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(BINDIR)/example-client $(MAIN_PKG)

# Run the example client
run:
	$(GO) run $(MAIN_PKG)/main.go

# Run all tests (excluding integration tests by default)
test:
	$(GO) test ./pkg/...

# Run tests with verbose output
test-verbose:
	$(GO) test -v $(PACKAGES)

# Create test database for integration tests
create-test-db:
	docker exec -it cogmem_postgres psql -U postgres -c "CREATE DATABASE cogmem_test;"

# Drop test database
drop-test-db:
	docker exec -it cogmem_postgres psql -U postgres -c "DROP DATABASE cogmem_test;"

# Set up test database (start postgres and create test db)
test-db-setup: dev-db-up create-test-db

# Run integration tests
test-integration:
	INTEGRATION_TESTS=true $(GO) test -v -count=1 -tags=integration ./test/integration/...

# Run command-line tool tests
test-cmd:
	INTEGRATION_TESTS=true $(GO) test -v -count=1 -tags=integration ./test/cmd/...

# Run command-line tool tests with postgres
test-cmd-postgres:
	INTEGRATION_TESTS=true TEST_POSTGRES=true $(GO) test -v -count=1 -tags=integration ./test/cmd/example_client_test.go -run TestExampleClientWithPostgresConfig

# Run script-based tests for example-client with various configurations
test-cmd-script: install
	cd ./test/scripts && ./run_tests.sh mock

# Run script-based tests with mock backend
test-cmd-script-mock: install
	cd ./test/scripts && ./run_tests.sh mock

# Run script-based tests with BoltDB backend
test-cmd-script-boltdb: install
	cd ./test/scripts && ./run_tests.sh boltdb

# Run script-based tests with SQLite backend
test-cmd-script-sqlite: install
	cd ./test/scripts && ./run_tests.sh sqlite

# Run script-based tests with PostgreSQL backend (requires POSTGRES_URL and OPENAI_API_KEY)
test-cmd-script-postgres: install
	cd ./test/scripts && ./run_tests.sh postgres

# Run script-based tests with ChromemGo backend (requires OPENAI_API_KEY)
test-cmd-script-chromemgo: install
	cd ./test/scripts && ./run_tests.sh chromemgo

# Run all script-based tests
test-cmd-script-all: install
	@echo "Running all script-based tests (this may take a while)..."
	@cd ./test/scripts && ./run_tests.sh all || echo "Some tests may have failed, but continuing with others..."

# Run PostgreSQL-specific integration tests (HStore, SQLStore, and PgVector adapters)
# See ../docs/POSTGRES_TESTING.md for details
test-postgres:
	INTEGRATION_TESTS=true $(GO) test -v -count=1 -tags=integration ./test/integration/ltm_postgres*.go ./test/integration/ltm_pgvector_test.go

# Run benchmarks
bench:
	$(GO) test -bench=. $(PACKAGES)

# Run linter
lint:
	$(GOLINT) run

# Format code
fmt:
	$(GO) fmt $(PACKAGES)

# Generate code from SQL
sqlc-gen:
	$(SQLC) generate

# Install dependencies
deps:
	$(GO) get -v -t $(PACKAGES)

# Start development databases using docker-compose
dev-db-up:
	docker-compose -f docker-compose.dev.yml up -d

# Stop development databases
dev-db-down:
	docker-compose -f docker-compose.dev.yml down

# Clean build artifacts
clean:
	rm -f $(BINDIR)/example-client
	rm -f $(BINDIR)/*.o $(BINDIR)/*.a
	find . -name "*.test" -delete
	find . -name "*.out" -delete
	$(GO) clean -cache -testcache

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build all packages"
	@echo "  install         - Build and install example client"
	@echo "  run             - Run the example client"
	@echo "  test            - Run unit tests"
	@echo "  test-verbose    - Run unit tests with verbose output"
	@echo "  create-test-db  - Create test database for integration tests"
	@echo "  drop-test-db    - Drop test database"
	@echo "  test-db-setup   - Start database and create test database"
	@echo "  test-integration - Run integration tests"
	@echo "  test-cmd        - Run command-line tool tests"
	@echo "  test-cmd-postgres - Run command-line tool tests with PostgreSQL"
	@echo "  test-cmd-script - Run script-based tests for example-client (default: mock)"
	@echo "  test-cmd-script-mock - Run script-based tests with mock backend"
	@echo "  test-cmd-script-boltdb - Run script-based tests with BoltDB backend"
	@echo "  test-cmd-script-sqlite - Run script-based tests with SQLite backend"
	@echo "  test-cmd-script-postgres - Run script-based tests with PostgreSQL backend"
	@echo "  test-cmd-script-chromemgo - Run script-based tests with ChromemGo backend"
	@echo "  test-cmd-script-all - Run all script-based tests"
	@echo "  test-postgres   - Run PostgreSQL-specific integration tests (see ../docs/POSTGRES_TESTING.md)"
	@echo "  bench           - Run benchmarks"
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  sqlc-gen        - Generate SQL client code"
	@echo "  deps            - Install dependencies"
	@echo "  dev-db-up       - Start development databases"
	@echo "  dev-db-down     - Stop development databases"
	@echo "  clean           - Clean binary artifacts, test files, and Go caches"
	@echo "  help            - Show this help message"