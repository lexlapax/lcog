This combines Feature-Based (Vertical Slice) approach with the Onion Architecture (or Clean/Hexagonal Architecture principles) for an idiomatic Go backend service structure.

The core idea is:
1.  **Organize by Feature:** Group code related to a specific business capability together.
2.  **Apply Onion Layers *within* or *across* Features:** Ensure dependencies flow inwards towards the core domain logic.

a recommended structure:

```
backend-go/
├── cmd/
│   └── server/
│       └── main.go             # Entry point, DI wire-up, server start
├── internal/
│   ├── featureA/               # --- Feature: User Management ---
│   │   ├── domain.go           # Feature-specific domain models/entities (e.g., User struct, validation rules)
│   │   ├── service.go          # Application logic/use cases (e.g., CreateUser, GetUserByID) - Defines required ports (interfaces)
│   │   ├── ports.go            # (Optional but good practice) Explicit interface definitions (e.g., UserRepository, TokenGenerator) needed by service.go
│   │   ├── handler_http.go     # Adapters: HTTP handlers calling the service (maps HTTP req/res to service calls)
│   │   ├── storage_postgres.go # Adapters: Postgres implementation of UserRepository
│   │   └── storage_redis.go    # Adapters: Redis implementation for maybe caching user sessions (another port implementation)
│   │
│   ├── featureB/               # --- Feature: Order Processing ---
│   │   ├── domain.go           # (e.g., Order struct, OrderStatus enum)
│   │   ├── service.go          # (e.g., PlaceOrder, GetOrder) - Defines required ports (e.g., OrderRepository, ProductFinder, PaymentGateway)
│   │   ├── ports.go            # (Optional) Interface definitions
│   │   ├── handler_grpc.go     # Adapters: gRPC handlers
│   │   ├── storage_mongo.go    # Adapters: MongoDB implementation of OrderRepository
│   │   └── client_productsvc.go # Adapters: Client to call an external Product Service (implements ProductFinder port)
│   │
│   ├── domain/                 # --- Shared Core Domain --- (Only if needed)
│   │   ├── shared_entity.go    # Truly core entities used by MULTIPLE features (e.g., Money struct, TenantID)
│   │   └── errors.go           # Common domain error types (e.g., ErrNotFound, ErrValidation)
│   │
│   ├── platform/               # --- Shared Infrastructure/Platform Code ---
│   │   ├── config/             # Configuration loading logic
│   │   │   └── config.go
│   │   ├── database/           # DB connection setup, transaction management helpers
│   │   │   └── postgres.go
│   │   │   └── mongo.go
│   │   ├── server/             # HTTP/gRPC server setup, middleware wiring
│   │   │   └── http.go
│   │   │   └── grpc.go
│   │   │   └── middleware/
│   │   │       └── auth.go
│   │   ├── logging/            # Shared logging setup
│   │   │   └── logger.go
│   │   └── messaging/          # Shared message queue producers/consumers setup (e.g. Kafka, RabbitMQ)
│   │       └── kafka.go
│   │
│   └── auth/                   # --- Cross-Cutting Concern: Authentication (example) ---
│       ├── domain.go           # Auth-specific concepts (e.g., Session, Token)
│       ├── service.go          # Auth logic (e.g., Login, ValidateToken) - Defines required ports (e.g., UserCredentialsFinder, TokenStore)
│       ├── ports.go            # Interface definitions
│       ├── handler_http.go     # Login HTTP endpoint
│       ├── token_jwt.go        # JWT implementation of TokenGenerator/TokenValidator
│       └── store_redis.go      # Redis implementation for TokenStore (e.g., session store)
│
├── api/                        # API definitions (e.g., OpenAPI specs, Protobuf files)
│   └── openapi/
│   │   └── spec.yaml
│   └── proto/
│       └── featureB.proto
│
├── configs/                    # Configuration files (e.g., config.yaml)
│   └── config.dev.yaml
│
├── migrations/                 # Database migration files
│   └── postgres/
│       └── 001_create_users_table.sql
│
├── scripts/                    # Build/deploy/helper scripts
│   └── run-dev.sh
│
├── go.mod
├── go.sum
├── Makefile                    # Optional: for build/test automation
└── README.md
```

**Explanation and Onion Layer Mapping:**

1.  **`cmd/server/main.go` (Outer Layer - Frameworks/Drivers):**
    *   Reads configuration (`internal/platform/config`).
    *   Sets up infrastructure connections (DBs, loggers - using `internal/platform/...`).
    *   **Dependency Injection:** Instantiates concrete adapters (`storage_postgres`, `handler_http`, `client_productsvc`, etc.).
    *   Instantiates services (`featureA.NewService`, `featureB.NewService`), injecting the adapter implementations that satisfy the required ports (interfaces).
    *   Instantiates handlers (`featureA.NewHTTPHandler`, `featureB.NewGRPCHandler`), injecting the services.
    *   Sets up the HTTP/gRPC server (`internal/platform/server`) and registers the handlers.
    *   Starts the server.

2.  **`internal/` (The Application Core & Adapters):**
    *   **`internal/featureX/` (Vertical Slices):**
        *   **`domain.go` (Innermost Layer - Domain Entities):** Contains the core data structures and business rules specific *to that feature*. Minimal dependencies. May depend on `internal/domain` for truly shared types.
        *   **`service.go` / `ports.go` (Application Layer):** Orchestrates use cases. Defines the *what* (interfaces/ports like `UserRepository`) but not the *how*. Depends *only* on its own `domain.go` and potentially `internal/domain`. **Crucially, it defines the interfaces (ports) it needs for external interactions (DB, external APIs, etc.).**
        *   **`handler_*.go`, `storage_*.go`, `client_*.go` (Outer Layers - Adapters/Interfaces):** Implement the ports defined by the `service.go`. They adapt external technologies (HTTP, gRPC, Postgres, Mongo, external HTTP clients) to the application layer's needs. They depend *inward* on the `service.go` (to call its methods) or implement interfaces defined in `service.go` or `ports.go`. They also depend on `internal/platform` for shared infrastructure setup (like DB connections).

    *   **`internal/domain/` (Innermost Layer - Shared Domain):** *Use sparingly*. Only for entities or value objects genuinely fundamental and shared across *multiple* features without causing unwanted coupling. Depends on nothing else within `internal`.

    *   **`internal/platform/` (Outer Layer - Infrastructure/Frameworks):** Shared, reusable infrastructure code. Doesn't contain business logic. It provides tools and setup for the adapters (e.g., a function to get a configured `*sql.DB` instance). Depends on external libraries (DB drivers, web frameworks, logging libraries). Feature adapters depend on this.

    *   **`internal/auth/` (Cross-Cutting Feature):** Some concerns like auth span multiple features. Structure them similarly to a feature, applying onion layers within. Its handlers might be registered as middleware via `internal/platform/server`.

3.  **`api/` (External):** Contract definitions. Not Go code, but essential for defining how clients interact.
4.  **`configs/` (External):** Runtime configuration.
5.  **`migrations/` (External):** Database schema management.

**Dependency Flow (Onion Rule Enforcement):**

*   `cmd` depends on `internal/*` (to wire everything up).
*   `internal/featureX/handler_*` depends on `internal/featureX/service` (and potentially `domain`).
*   `internal/featureX/storage_*` implements interfaces from `internal/featureX/service|ports` and depends on `internal/featureX/domain` (for types) and `internal/platform/database` (for connections).
*   `internal/featureX/service` depends *only* on `internal/featureX/domain` and `internal/domain`. **It does NOT depend on handlers, storage, or platform.**
*   `internal/featureX/domain` depends *only* on `internal/domain` (if it exists) or nothing else in `internal`.
*   `internal/platform/*` depends mostly on external libraries. Feature adapters depend on it.
*   `internal/domain` depends on nothing else in `internal`.

**Benefits:**

*   **High Cohesion:** Code related to a feature is located together.
*   **Low Coupling:** Features are relatively independent. Changes in one feature are less likely to break others. Dependencies are explicit through interfaces (ports).
*   **Testability:** Core domain and application logic (`domain.go`, `service.go`) can be tested independently of frameworks and infrastructure by mocking the interfaces (ports). Adapters can be tested in integration.
*   **Maintainability & Scalability:** Easier to navigate, understand, and assign features to different developers or teams. Follows SOLID principles, especially Dependency Inversion.
*   **Flexibility:** Easier to swap infrastructure components (e.g., change DB from Postgres to Mongo for a specific feature) by just changing the adapter implementation.

This structure provides a robust foundation for building maintainable, testable, and scalable backend services in Go, respecting both feature-centric organization and clean architectural principles. Remember to adapt it based on the specific complexity and needs of your project – don't over-engineer if your service is very simple.