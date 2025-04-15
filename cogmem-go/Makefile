# Makefile for cogmem-go

# Variables
GO = go
GOFLAGS = 
PACKAGES = ./...
BINDIR = ./bin
MAIN_PKG = ./cmd/example-agent

# Tools
GOLINT = golangci-lint
SQLC = sqlc

# Build flags
BUILD_FLAGS = -v

.PHONY: all build clean test test-verbose test-integration lint fmt sqlc-gen help deps

all: build

# Build all packages
build:
	$(GO) build $(GOFLAGS) $(BUILD_FLAGS) $(PACKAGES)

# Build and install example agent
install: 
	mkdir -p $(BINDIR)
	$(GO) build $(GOFLAGS) $(BUILD_FLAGS) -o $(BINDIR)/example-agent $(MAIN_PKG)

# Run the example agent
run:
	$(GO) run $(MAIN_PKG)/main.go

# Run all tests (excluding integration tests by default)
test:
	$(GO) test ./pkg/...

# Run tests with verbose output
test-verbose:
	$(GO) test -v $(PACKAGES)

# Run integration tests
test-integration:
	INTEGRATION_TESTS=true $(GO) test ./test/integration/...

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
	rm -f $(BINDIR)/example-agent
	rm -f $(BINDIR)/*.o $(BINDIR)/*.a
	find . -name "*.test" -delete
	find . -name "*.out" -delete
	$(GO) clean -cache -testcache

# Show help
help:
	@echo "Available targets:"
	@echo "  build           - Build all packages"
	@echo "  install         - Build and install example agent"
	@echo "  run             - Run the example agent"
	@echo "  test            - Run unit tests"
	@echo "  test-verbose    - Run unit tests with verbose output"
	@echo "  test-integration - Run integration tests"
	@echo "  bench           - Run benchmarks"
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  sqlc-gen        - Generate SQL client code"
	@echo "  deps            - Install dependencies"
	@echo "  dev-db-up       - Start development databases"
	@echo "  dev-db-down     - Stop development databases"
	@echo "  clean           - Clean binary artifacts, test files, and Go caches"
	@echo "  help            - Show this help message"