A structure for a native iOS application using Swift, SwiftUI (as the primary UI framework), Combine or Async/Await for concurrency, and Swift Package Manager (SPM) for modularity, following the Feature-Based Slicing and Onion/Clean Architecture principles.

Using SPM packages is the most idiomatic way in modern Swift development to enforce boundaries similar to Gradle modules in Android.

**Project Structure using Swift Packages:**

```
mobile-ios.xcworkspace/  # Xcode Workspace containing App + Packages
├── YourApp/                   # Main Application Target (.app)
│   ├── YourAppApp.swift       # @main App struct (SwiftUI App Lifecycle)
│   ├── Assets.xcassets
│   ├── Info.plist
│   ├── Preview Content/
│   └── ... (Other App target files)
│
└── Packages/                  # Directory managed by Xcode, containing local SPM packages
    ├── Features/              # Grouping folder (optional, for organization)
    │   ├── ProfileFeature/    # --- Feature: User Profile (SPM Package) ---
    │   │   ├── Package.swift  # Declares dependencies (e.g., on Core packages)
    │   │   ├── Sources/
    │   │   │   └── ProfileFeature/
    │   │   │       ├── Domain/         # Innermost: Feature-specific models, use cases, repository protocols
    │   │   │       │   ├── Model/
    │   │   │       │   │   └── UserProfile.swift
    │   │   │       │   ├── UseCase/
    │   │   │       │   │   └── UpdateProfileUseCase.swift # Protocol + Implementation (or just Impl)
    │   │   │       │   └── Repository/
    │   │   │       │       └── ProfileSettingsRepository.swift # *Protocol* (Port)
    │   │   │       ├── Data/           # Infrastructure: Concrete implementations of protocols
    │   │   │       │   └── Repository/
    │   │   │       │       └── ConcreteProfileSettingsRepository.swift # Implements protocol from Domain
    │   │   │       ├── Presentation/   # UI Layer: SwiftUI Views, ViewModels
    │   │   │       │   ├── ViewModel/
    │   │   │       │   │   └── ProfileViewModel.swift # ObservableObject
    │   │   │       │   └── View/
    │   │   │       │       └── ProfileScreen.swift    # SwiftUI View (Entry point for this feature)
    │   │   │       └── DI/             # (Optional) Internal wiring/factories for this feature
    │   │   └── Tests/
    │   │       └── ProfileFeatureTests/ # Unit tests for this feature's components
    │   │
    │   └── OrdersFeature/     # --- Feature: Order History (SPM Package) ---
    │       ├── Package.swift
    │       ├── Sources/
    │       │   └── OrdersFeature/       # (Structure similar to ProfileFeature)
    │       │       ├── Domain/
    │       │       ├── Data/
    │       │       └── Presentation/
    │       └── Tests/
    │           └── OrdersFeatureTests/
    │
    └── Core/                  # Grouping folder for shared Core packages
        ├── CoreDomain/        # --- Core Domain (SPM Package - Innermost) ---
        │   ├── Package.swift  # (No UI framework dependencies)
        │   ├── Sources/
        │   │   └── CoreDomain/
        │   │       ├── Model/          # Shared data models (structs, enums - e.g., User, OrderStatus)
        │   │       └── Repository/     # Shared repository *Protocols* (Ports) (e.g., AuthRepository)
        │   └── Tests/
        │       └── CoreDomainTests/
        │
        ├── CoreData/          # --- Core Data/Infrastructure (SPM Package - Adapters) ---
        │   ├── Package.swift  # (Depends on CoreDomain, Foundation, potentially CoreData/Realm etc.)
        │   ├── Sources/
        │   │   └── CoreData/
        │   │       ├── Network/        # Networking setup (e.g., URLSession wrapper, API client)
        │   │       │   └── ApiClient.swift
        │   │       ├── Persistence/    # Persistence setup (e.g., CoreData stack, Realm config)
        │   │       │   └── PersistenceController.swift
        │   │       └── Repository/     # *Implementations* of shared protocols from CoreDomain
        │   │           └── ConcreteAuthRepository.swift # Implements AuthRepository protocol
        │   └── Tests/
        │       └── CoreDataTests/
        │
        ├── CoreUI/            # --- Core UI (SPM Package - Shared Presentation) ---
        │   ├── Package.swift  # (Depends on SwiftUI)
        │   ├── Sources/
        │   │   └── CoreUI/
        │   │       ├── Theme/          # AppTheme definition (colors, fonts)
        │   │       ├── Components/     # Reusable SwiftUI Views (e.g., LoadingView, ErrorBanner, CustomButton)
        │   │       └── Extensions/     # SwiftUI View extensions, modifiers
        │   └── Tests/
        │       └── CoreUITests/
        │
        └── CoreCommon/        # --- Core Common Utilities (SPM Package) ---
            ├── Package.swift  # (Minimal dependencies, e.g., Foundation, Combine)
            ├── Sources/
            │   └── CoreCommon/
            │       ├── Util/           # Generic utilities (logging, date formatters)
            │       └── Resource/       # Result/Resource wrapper enum (e.g., .loading, .success(T), .failure(Error))
            └── Tests/
                └── CoreCommonTests/

```

**Explanation and Onion Layer Mapping:**

1.  **`YourApp` Target (Outer Layer - Composition Root & Framework Host):**
    *   The main executable application.
    *   Contains the `@main App` struct (`YourAppApp.swift`), which is the entry point for the SwiftUI application lifecycle.
    *   Sets up the root view hierarchy (e.g., `TabView`, `NavigationView`).
    *   Handles top-level navigation, potentially using a coordinator pattern or SwiftUI's built-in navigation, linking different feature entry points.
    *   Performs initial setup and **Dependency Injection**. This might involve:
        *   Instantiating concrete implementations from `CoreData` or Feature `Data` layers.
        *   Passing dependencies down via SwiftUI's `EnvironmentObject`, initializers, or a dedicated DI container (less common in pure SwiftUI but possible).
    *   Depends on all necessary `Feature` packages and `Core` packages.

2.  **`Core` Packages (Shared Layers):**
    *   **`CoreDomain` (Innermost Layer - Domain):**
        *   Pure Swift code. Contains shared data models (structs, enums) and **protocols** defining contracts for data access (`AuthRepository`) or core business logic.
        *   **NO UI framework (SwiftUI/UIKit) or infrastructure (URLSession/CoreData) dependencies.** Platform-agnostic. Highly testable.
    *   **`CoreCommon` (Utility Layer):**
        *   Generic Swift utilities, result wrappers (`Resource`), logging abstractions. Minimal dependencies, potentially only Foundation/Combine. Sits alongside or just outside `CoreDomain`.
    *   **`CoreData` (Outer Layer - Infrastructure / Adapters):**
        *   Provides concrete **implementations** for the repository protocols defined in `CoreDomain`.
        *   Contains platform/framework-specific code: `URLSession` wrappers, `CoreData` stack/entities, `UserDefaults` access, specific API service implementations (e.g., using Moya/Alamofire if preferred).
        *   Depends *inward* on `CoreDomain` (for models and protocols it implements) and `CoreCommon`.
    *   **`CoreUI` (Outer Layer - Presentation / UI):**
        *   Contains shared, reusable SwiftUI views (`CustomButton`), modifiers, theme definitions (colors, fonts).
        *   Depends on SwiftUI and potentially `CoreCommon`. Does *not* depend directly on `CoreData` or `CoreDomain`.

3.  **`Feature` Packages (Vertical Slices with Internal Layers):**
    *   Each package encapsulates a distinct user-facing feature (Profile, Orders).
    *   **`Domain/` (Innermost - Feature Specific):**
        *   Feature-specific models, Use Case definitions (often as protocols and concrete classes/structs), and feature-specific repository **protocols** (Ports).
        *   Depends on `CoreDomain` and `CoreCommon`. No UI/Infrastructure deps.
    *   **`Data/` (Outer Layer - Infrastructure Adapters - Feature Specific):**
        *   Concrete **implementations** of repository protocols defined in its own `Domain/` or `CoreDomain`.
        *   May define feature-specific network requests or local persistence logic.
        *   Depends on its own `Domain/`, `CoreDomain`, `CoreCommon`, and `CoreData` (to reuse shared `ApiClient` or persistence setup).
    *   **`Presentation/` (Outer Layer - UI / Presentation):**
        *   `ViewModel/`: `ObservableObject` classes responsible for managing the UI state (`@Published` properties), handling user interactions, and orchestrating data flow by calling Use Cases or Repository protocols. Uses Combine or Async/Await for asynchronous operations. Depends on its feature's `Domain/`, `CoreDomain`, `CoreCommon`, and potentially `CoreUI`. *Minimizes direct framework dependencies beyond ObservableObject/Combine/SwiftUI property wrappers.*
        *   `View/`: SwiftUI Views that observe the `ViewModel`, render the UI based on state, and forward user actions to the `ViewModel`. Depends on its `ViewModel`, `CoreUI`, and potentially `CoreDomain` models (for display).
    *   **Public Interface:** Each feature package needs to publicly expose its entry point View (e.g., `public struct ProfileScreen: View {...}`) so the main App target can navigate to it.

**Dependency Flow (Onion Rule Enforcement via SPM):**

*   `YourApp` target depends on Feature packages and relevant Core packages.
*   Feature packages depend on necessary Core packages (`CoreDomain`, `CoreData` via protocols, `CoreUI`, `CoreCommon`). Features should generally *not* depend directly on other Feature packages (use navigation/coordination in the App target or a dedicated Navigation package).
*   `CoreData` depends on `CoreDomain`, `CoreCommon`.
*   `CoreUI` depends on `CoreCommon` (potentially).
*   `CoreCommon` depends on `CoreDomain` (potentially).
*   `CoreDomain` depends on nothing else within the project's packages (only Swift standard library, Foundation basics).
*   Within a feature package: `Presentation` depends on `Domain`. `Data` depends on `Domain`. ViewModels depend on Use Case/Repository *protocols*, not concrete implementations. Dependencies are wired via initializers or EnvironmentObject in the App target or feature's internal DI.

**Benefits:**

*   **Testability:** `CoreDomain` and feature `Domain` are pure Swift. ViewModels/Use Cases are testable by mocking repository/use case protocols. `CoreData` implementations can be integration tested. SwiftUI views can be tested using snapshotting or UI testing frameworks (though often less emphasis than ViewModel testing).
*   **Maintainability:** Feature code is co-located. SPM enforces strict boundaries, preventing spaghetti code. Clear separation of concerns.
*   **Scalability:** Excellent for large teams. SPM significantly improves build times for large projects compared to a monolithic target. Features can be developed, tested, and potentially even shipped independently (though usually integrated via the App target).
*   **Reusability:** Core packages are highly reusable across features, and potentially even across different apps.
*   **Replaceability:** Infrastructure details (networking library, persistence framework) are confined to the `Data` layers. They can be swapped by changing the concrete implementation classes without affecting the `Domain` or `Presentation` layers, thanks to dependency inversion via protocols.

This modular SPM-based structure provides a robust, testable, and scalable foundation for building modern iOS applications using Swift and SwiftUI, adhering to clean architecture principles within a feature-sliced organization.