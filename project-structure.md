# Core Principles Abstract (Backend, Frontend Web, Mobile)

The goal is to create scalable, maintainable, and testable applications across diverse platforms (backend services, web frontends, native mobile apps, cross-platform mobile apps) by combining two key organizational strategies:

## 1. Feature-Based Slicing (Vertical Slices):

*   **Concept:** Instead of organizing code primarily by technical layer (e.g., all controllers/ViewModels together, all database logic together), the primary organization is done by **business feature** or user-facing capability (e.g., User Profile, Order Management, Product Search). Modularity through packages (Dart/Swift), modules (Gradle), or strict directories is strongly encouraged to enforce these boundaries.
*   **Implementation:** Create distinct directories, packages, or modules for each significant feature. Within each feature's boundary, you'll find all the code necessary for that feature to function across its relevant technical layers (e.g., its specific UI screens/widgets/components, state management logic, API interaction logic, domain models, local persistence interactions).
*   **Benefit:** **High Cohesion.** Code related to a single business capability lives together, making it easier to understand, develop, test, and modify that specific functionality without scattering changes across the entire codebase or unrelated modules. Improves team scalability as different teams can own different features.

## 2. Layered Architecture (Onion/Clean/Hexagonal Principles):

*   **Concept:** Apply layers of abstraction *within* or *across* these features to manage dependencies and separate concerns. Dependencies must always flow **inwards**, towards the core business logic, never outwards.
*   **Layers (Conceptual, from Outer to Inner):**
    *   **Infrastructure / Frameworks / Adapters / UI (Outer Layer):** Deals with the "outside world" and specific technology implementations. This includes:
        *   **UI:** Native UI components (Android Views/Compose, iOS UIKit/SwiftUI), Cross-platform UI (Flutter Widgets, React Native Components), Web UI (React Components).
        *   **Device/Platform APIs:** Camera, GPS, Local Storage (SharedPreferences, UserDefaults, AsyncStorage), specific OS integrations.
        *   **External Interactions:** HTTP handlers/controllers (Go/Python backends), database query implementations (SQLAlchemy, Room, CoreData, Go DB drivers), external API clients (Retrofit, Axios, Go http client), push notification handlers, etc.
        *   This layer *adapts* external concerns and framework details to the application's needs, often implementing interfaces defined further in.
    *   **Application / Use Cases / State Management (Middle Layer):** Orchestrates the business logic to fulfill specific application use cases or user actions. This includes:
        *   Application-specific workflows and orchestration.
        *   Mobile/Frontend state management logic (ViewModels, Blocs, Reducers, Stores).
        *   Defining the **interfaces (Ports)** required for external interactions (like data fetching, persistence, device access) – the "what," not the "how."
        *   This layer *does not* know about specific databases, UI frameworks, network libraries, or device APIs directly – it only depends on the *abstractions* (interfaces/ports) it needs.
    *   **Domain / Entities (Inner Core):** Represents the core business concepts, rules, entities, and value objects. This layer is the most stable and has the fewest dependencies (ideally none on other application layers or frameworks). It contains pure business logic and platform-agnostic data structures (POJOs, POKOs, Structs, data classes), independent of how the application is delivered, displayed, or how data is stored/retrieved.
*   **Dependency Rule:** The key rule is that code in an inner layer *cannot* depend on code in an outer layer. The Application layer depends on the Domain layer. The Infrastructure/UI layer depends on the Application layer (often by implementing its interfaces/ports or consuming its state/hooks) and potentially the Domain layer (for data types).
*   **Dependency Inversion:** Outer layers depend on *abstractions* (interfaces/ports/protocols/Abstract Base Classes) defined in the inner/application layers. Concrete implementations (adapters in the Infrastructure/UI/Data layers) are provided at runtime, typically via Dependency Injection. This decouples the core application and domain logic from specific technologies and frameworks.

## Key Elements in Practice (Common Across Stacks):

*   **`features/` Directory/Group:** The primary home for vertical slices, often implemented as distinct modules/packages.
*   **`ports` / Interfaces / Protocols / ABCs:** Explicit contracts defined usually at the edge of the Application or Domain layers, specifying *what* needs to be done (e.g., `saveUser`, `fetchOrders`, `getCurrentLocation`) without detailing *how*.
*   **`adapters` / `infrastructure` / `data` / `ui` / `presentation`:** Directories/layers containing concrete implementations of ports or framework-specific code (e.g., `SqlalchemyUserRepository`, `ProfileViewModel`, `OrdersApiDataSource`, `ProfileScreen`, `UserDao`).
*   **`domain` Directory/Module (within features or shared):** Contains the core business types and pure logic.
*   **`shared` / `core` / `platform` Directory/Module Group:** Houses code genuinely reusable across multiple features, often categorized by layer itself (e.g., `core/ui`, `core/domain`, `platform/network`). Use judiciously to avoid creating a hidden, tightly coupled monolith.
*   **Composition Root (`main.go`, `main.py`, `App.tsx`/`main.tsx`, `MyApp.kt`/`MainActivity.kt`, `YourAppApp.swift`, `main.dart`/`app.dart`):** The application's entry point where concrete dependencies are instantiated and "wired" together (Dependency Injection setup - Hilt, Dagger, GetIt, Riverpod, manual, etc.), configuring and starting the application and its primary UI shell/navigation.

## Overall Benefits:

*   **Testability:** Core Domain and Application/State Management logic can be tested in isolation (often as pure Dart/Kotlin/Swift/Go/TS/JS) by mocking the interfaces/ports defined at their boundaries. UI components and infrastructure adapters can be tested with appropriate framework tools.
*   **Maintainability:** Code is easier to find, understand, and modify due to high cohesion (features) and clear separation of concerns (layers). Boundaries enforced by modules/packages prevent unintended coupling.
*   **Scalability:** Teams can often work on different features concurrently with less friction. The codebase is structured for growth in complexity and team size. Modular builds improve compilation times (especially mobile/frontend).
*   **Flexibility & Replaceability:** Easier to swap external dependencies (e.g., change UI frameworks/details, state management libraries, databases, network clients, persistence solutions) by changing only the adapter implementations in the outer layers, minimizing impact on core business logic. Enables easier adoption of new technologies or refactoring of specific parts.

## project structure
```
this-directory/
├── backend-go/
│   ├── project-structure.md    # project structure with details for go
│   └── readme.md               # file that describes the sub-project  
│   └── go-project folders      # all the subfolders and files as described in structure
├── backend-python/
│   ├── project-structure.md    # project structure with details for python
│   └── readme.md               # file that describes the sub-project 
│   └── python folders          # all the subfolders and files as described in structure
├── frontend-react/
│   ├── project-structure.md    # project structure with details for python
│   └── readme.md               # file that describes the sub-project
│   └── REACT folders           # all the subfolders and files as described in structure
├── mobile-android/
│   ├── project-structure.md    # project structure with details for android native app
│   └── readme.md               # file that describes the sub-project
│   └── android folders           # all the subfolders and files as described in structure
├── mobile-ios/
│   ├── project-structure.md    # project structure with details for iOS native app
│   └── readme.md               # file that describes the sub-project
│   └── ios folders           # all the subfolders and files as described in structure
├── mobile-crossplatform/
│   ├── project-structure.md    # project structure with details for web/mobile app
│   └── readme.md               # file that describes the sub-project
│   └── folder structure        # all the subfolders and files as described in structure
├── architecture.md             # Architecture template with diagrams
├── prd.md                      # product requirements document template
├── project-structure.md        # this file
└── README.md                   # project readme template.
```