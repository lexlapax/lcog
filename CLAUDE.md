# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Structure

The CogMem project is organized as follows:

- `/README.md` - Main project documentation and overview (at the repository root)
- `/docs/` - Detailed documentation on components, features, and implementation 
  - Including `/docs/POSTGRES_TESTING.md` for PostgreSQL adapter testing
- `/cogmem-go/` - The Go implementation of CogMem
  - All code and implementation details are contained here
  - Makefile commands need to be run from this directory

## Build, Lint, and Test Commands

> **IMPORTANT**: All Makefile commands must be run from the `/cogmem-go/` directory, not from the repository root.

### Makefile Commands
- `make build` - Build all packages
- `make test` - Run unit tests
- `make test-verbose` - Run tests with verbose output
- `make test-integration` - Run integration tests (requires database)
- `make test-cmd` - Run command-line tool tests
- `make test-postgres` - Run PostgreSQL-specific tests (see repository root: /docs/POSTGRES_TESTING.md)
- `make bench` - Run benchmarks
- `make run` - Run the example client
- `make fmt` - Format code
- `make lint` - Run linter
- `make sqlc-gen` - Generate SQL client code for adapters
- `make deps` - Install dependencies
- `make dev-db-up` - Start development databases
- `make dev-db-down` - Stop development databases

### Direct Go Commands (Alternative)
- `go test ./pkg/...` - Run all unit tests
- `go test -v ./pkg/specific/package` - Run tests in specific package
- `INTEGRATION_TESTS=true go test ./test/integration/...` - Run integration tests
- `INTEGRATION_TESTS=true go test -tags=integration ./test/cmd/...` - Run command-line tool tests
- `go run cmd/example-client/main.go` - Run the example client
- `go fmt ./...` - Format code
- `sqlc generate` - Generate SQL client code for adapters
- `golangci-lint run` - Run linter
- `go build ./...` - Build all packages

## Code Style Guidelines

- **Formatting**: Follow standard Go formatting (gofmt)
- **Imports**: Group standard library, third-party, and local imports
- **Types**: Use strong typing, prefer interfaces, and proper error handling
- **Naming**: Follow Go conventions (CamelCase for exported, camelCase for private)
- **Error Handling**: Check all errors, use custom error types from pkg/errors
- **Comments**: Document all exported functions, types, and packages
- **Testing**: Follow Test-First Development (TFD) - write tests before implementation
- **Multi-tenancy**: All LTM operations must respect entity context and enforce isolation

## Implementation Principles

- **Project Structure**:
  - Code lives in `/cogmem-go/`
  - Documentation lives in `/docs/`
  - Base configuration in repository root
  - The main `README.md` provides a complete overview of the project

- **Layered Architecture**: Dependencies flow inward (infrastructure → application → domain)
- **Interfaces**: Define interfaces before implementations, use dependency injection
- **LTM Adapters**: Multiple supported backends (SQL, KV, Vector, Graph) behind common interface
- **Lua Integration**: Sandbox all Lua scripts, handle script errors gracefully
- **Entity Context**: Always propagate entity context through the entire call stack
- **Migrations**: Use SQL migrations for database schema changes (in migrations/)
- **Security & Secrets**: 
  - Never hardcode secrets (API keys, passwords, etc.) in committed files
  - Use environment variables or .env files for secrets management
  - Add secrets-containing files to .gitignore
  - Use example files (e.g., .env.example) with placeholders to document required variables

When making changes that affect the implementation, update both the code and its documentation in the corresponding locations.