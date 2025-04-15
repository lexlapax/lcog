# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Lint, and Test Commands

- Go: `go test ./...` - Run all tests
- Go: `go test -v ./pkg/specific/package` - Run tests in specific package
- Go: `go test ./test/integration/...` - Run integration tests
- Go: `go run cmd/example-agent/main.go` - Run the example agent
- Go: `go fmt ./...` - Format code
- Go: `sqlc generate` - Generate SQL client code for adapters
- Go: `golangci-lint run` - Run linter
- Go: `go build ./...` - Build all packages

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

- **Layered Architecture**: Dependencies flow inward (infrastructure → application → domain)
- **Interfaces**: Define interfaces before implementations, use dependency injection
- **LTM Adapters**: Multiple supported backends (SQL, KV, Vector, Graph) behind common interface
- **Lua Integration**: Sandbox all Lua scripts, handle script errors gracefully
- **Entity Context**: Always propagate entity context through the entire call stack
- **Migrations**: Use SQL migrations for database schema changes (in migrations/)