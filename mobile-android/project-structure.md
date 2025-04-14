Mapping the Feature-Based Slicing and Onion/Clean Architecture principles onto a native Android application using Kotlin, Jetpack libraries (ViewModel, Room, Compose, Hilt/Dagger), and Coroutines/Flow. Gradle modules are highly recommended for enforcing feature boundaries.

**Structure using Gradle Modules (Recommended for Scalability):**

```
mobile-android/
├── app/                        # Main application module: Wiring, Application class, NavHost
│   ├── build.gradle.kts
│   └── src/main/
│       ├── AndroidManifest.xml
│       └── java/com/example/myapp/
│           ├── MyApp.kt        # Application class (Hilt setup)
│           ├── MainActivity.kt # Hosts NavHostController, potentially top-level UI shell
│           └── di/             # App-level DI wiring (if needed beyond Hilt's auto-magic)
│
├── build.gradle.kts            # Root build file
├── settings.gradle.kts         # Declares included modules
│
├── core/                       # Shared code across features (multiple modules)
│   ├── domain/                 # --- Core Domain Module (Innermost) ---
│   │   ├── build.gradle.kts    # (No Android dependencies here!)
│   │   └── src/main/java/com/example/core/domain/
│   │       ├── model/          # Shared data classes/entities (User, Product) - POKOs/Data Classes
│   │       ├── repository/     # Repository *Interfaces* (Ports) (e.g., UserRepository, ProductRepository)
│   │       └── usecase/        # Shared Use Case *Interfaces* or simple base classes (optional)
│   │
│   ├── data/                   # --- Core Data Module (Infrastructure Adapters - Shared) ---
│   │   ├── build.gradle.kts    # (Depends on :core:domain, includes Room/Retrofit libs, Android deps OK)
│   │   └── src/main/java/com/example/core/data/
│   │       ├── local/          # Room DB setup, shared DAOs interfaces/base implementations
│   │       │   └── AppDatabase.kt
│   │       ├── remote/         # Retrofit setup, shared API service interfaces/base implementations
│   │       │   └── ApiClient.kt
│   │       ├── repository/     # *Implementations* of shared repository interfaces from :core:domain
│   │       │   └── UserRepositoryImpl.kt # Implements core:domain:UserRepository
│   │       └── di/             # Hilt module providing shared data sources (DB, ApiClient)
│   │
│   ├── ui/                     # --- Core UI Module (Shared Presentation Elements) ---
│   │   ├── build.gradle.kts    # (Depends on Android UI libs, Compose, potentially :core:common)
│   │   └── src/main/java/com/example/core/ui/
│   │       ├── theme/          # AppTheme, Colors, Typography
│   │       ├── components/     # Reusable Composables (AppButton, LoadingSpinner, ErrorMessage)
│   │       └── util/           # UI-related utilities (e.g., Modifier extensions)
│   │
│   └── common/                 # --- Core Common Utilities Module ---
│       ├── build.gradle.kts    # (Minimal dependencies, maybe Kotlin stdlib, Coroutines)
│       └── src/main/java/com/example/core/common/
│           ├── resource/       # Wrapper class for results (Success, Error, Loading)
│           │   └── Resource.kt
│           └── util/           # Generic utils (logging wrappers, date formatters - non-Android specific)
│
└── feature/                    # Parent directory for feature modules
    ├── profile/                # --- Feature: User Profile ---
    │   ├── build.gradle.kts    # (Depends on relevant :core modules, Android libs)
    │   └── src/main/
    │       ├── java/com/example/feature/profile/
    │       │   ├── di/             # Hilt module for profile-specific dependencies (ViewModel, UseCases, Repositories)
    │       │   │   └── ProfileModule.kt
    │       │   ├── domain/         # (Optional) Feature-specific domain logic/models/usecases (if not in :core:domain)
    │       │   │   └── usecase/
    │       │   │       └── UpdateProfileUseCase.kt # Interface + Impl or just Impl if simple
    │       │   ├── data/           # Feature-specific data logic (if not covered by :core:data)
    │       │   │   ├── repository/ # Implements feature-specific ports or extends core ones
    │       │   │   │   └── ProfileSettingsRepositoryImpl.kt
    │       │   │   └── remote/     # Profile-specific Retrofit service interface
    │       │   │       └── ProfileApiService.kt
    │       │   └── presentation/   # UI Layer for this feature
    │       │       ├── viewmodel/  # ViewModel specific to the profile feature
    │       │       │   └── ProfileViewModel.kt
    │       │       ├── ui/         # Compose Screens and Composables for the profile feature
    │       │       │   └── ProfileScreen.kt
    │       │       │   └── components/
    │       │       │       └── ProfileHeader.kt
    │       │       └── navigation/ # Navigation graph entries specific to this feature
    │       │           └── ProfileNavigation.kt # (e.g., extensions for NavController)
    │       └── res/                # Feature-specific resources (layouts, drawables, strings) - use sparingly, prefer :core:ui
    │
    └── orders/                 # --- Feature: Order History ---
        ├── build.gradle.kts
        └── src/main/
            └── java/com/example/feature/orders/ # (Structure similar to profile)
                ├── di/
                ├── domain/
                ├── data/
                └── presentation/
                    ├── viewmodel/
                    │   └── OrdersViewModel.kt
                    ├── ui/
                    │   └── OrdersScreen.kt
                    └── navigation/

```

**Explanation and Onion Layer Mapping (Android Adaptation):**

1.  **`:app` Module (Outer Layer - Composition Root & Framework):**
    *   Initializes the Application (`MyApp.kt`), setting up Dependency Injection (Hilt).
    *   Hosts the main Activity (`MainActivity.kt`) which typically sets up the `NavHost` using Jetpack Navigation.
    *   Knows about all `:feature:*` modules to orchestrate navigation between them.
    *   Acts as the final wiring point where dependencies provided by Hilt are injected into Activities/Fragments/ViewModels.

2.  **`:core:*` Modules (Shared Layers):**
    *   **`:core:domain` (Innermost Layer - Domain):**
        *   Contains pure Kotlin code: data classes for business entities, interfaces defining *what* data operations are needed (`UserRepository` - the Port), and potentially core business logic rules or use case interfaces.
        *   **Crucially, NO Android SDK dependencies.** This makes it platform-agnostic and easily testable.
    *   **`:core:common` (Utility Layer - Variable):**
        *   Generic utilities (wrappers like `Resource`, maybe logging abstractions). Should ideally have minimal/no Android dependencies. Sits alongside or just outside Domain.
    *   **`:core:data` (Outer Layer - Infrastructure / Adapters):**
        *   Provides concrete implementations for the repository interfaces defined in `:core:domain`.
        *   Contains Android/JVM-specific code: Room database setup/DAOs, Retrofit API service definitions/client setup, SharedPreferences wrappers.
        *   Depends *inward* on `:core:domain` (for models and interfaces it implements).
    *   **`:core:ui` (Outer Layer - Presentation / UI):**
        *   Shared UI elements: Themes, reusable Compose components (`AppButton`), potentially base ViewModels or UI state classes if needed (though often better in `:core:common` if non-Android specific).
        *   Depends on Android UI libraries. May depend on `:core:common`. Does *not* depend directly on `:core:data` or `:core:domain`.

3.  **`:feature:*` Modules (Vertical Slices with Internal Layers):**
    *   Each module encapsulates a distinct user-facing feature.
    *   **`domain/` (Innermost - Feature Specific):** Optional. If a feature has complex business logic or models *not* shared, they live here. Depends on `:core:domain`. No Android deps. Often includes Use Case classes/interfaces specific to the feature (e.g., `UpdateProfileUseCase`). Use cases orchestrate calls to repositories.
    *   **`data/` (Outer Layer - Infrastructure Adapters - Feature Specific):** Implements feature-specific repository interfaces (from its own `domain/` or `:core:domain`) or defines feature-specific DAOs/API endpoints. Depends on `:core:data` (for shared DB/API clients) and its own/core `domain`. Android deps OK.
    *   **`presentation/` (Outer Layer - UI / Presentation):**
        *   `viewmodel/`: Android ViewModels using Jetpack ViewModel. They fetch data (by calling Use Cases or Repositories directly), manage UI-related state (often using `StateFlow` or `LiveData`), handle user input logic, and expose state to the UI. They depend on Use Cases or Repository interfaces (injected via Hilt). *Should minimize direct Android Framework dependencies beyond ViewModel lifecycle*.
        *   `ui/`: Compose Screens and Composables (or Fragments/Activities/XML if not using Compose). Observes state from the `ViewModel`, renders the UI, and delegates user events back to the `ViewModel`. Depends on `ViewModel`, `:core:ui`, and potentially domain models (for display). Contains Android UI dependencies.
    *   **`di/` (Outer Layer - DI Setup):** Hilt modules define how to provide dependencies *within this feature* (ViewModels, Use Cases, Repository implementations).

**Dependency Flow (Onion Rule Enforcement via Gradle Modules):**

*   `:app` depends on `:feature:*` and `:core:*`.
*   `:feature:*` depends on `:core:*` modules as needed (e.g., `domain`, `data` via interfaces, `ui`, `common`). Features should *not* depend directly on other features; navigation handles transitions.
*   `:core:data` depends on `:core:domain`, `:core:common`.
*   `:core:ui` depends on `:core:common` (potentially).
*   `:core:common` depends on `:core:domain` (potentially).
*   `:core:domain` depends on nothing else within the project.
*   Within a feature module, dependencies flow inwards: `presentation` -> `domain` (use cases/interfaces), `data` -> `domain`. ViewModels typically depend on repository *interfaces* or Use Cases, not concrete data implementations. Hilt connects the interfaces to implementations.

**Benefits:**

*   **Testability:** Domain layer is pure Kotlin. Use Cases/ViewModels can be tested by providing mock repository implementations. UI can be tested using Compose testing APIs or Espresso, potentially with mock ViewModels. Data layer can be integration tested.
*   **Maintainability:** Code for a feature is co-located. Gradle modules enforce boundaries, preventing tight coupling. Clear separation of concerns via layers.
*   **Scalability:** Excellent for large teams and complex apps. Modules improve build times (incremental builds). Features can be developed more independently.
*   **Reusability:** Core modules (`:core:domain`, `:core:common`, `:core:ui`) promote reuse across features. `:core:domain` could even be reused in a KMM project.
*   **Replaceability:** Easier to swap data sources (e.g., different networking library, different local cache) by changing implementations in the `data` layer without affecting Use Cases or UI, thanks to dependency inversion via repository interfaces.

This modular structure, combining feature slicing with clean architecture layers, provides a robust and scalable foundation for modern Android development. For smaller projects, you *could* implement this using packages within a single `:app` module, but you lose the strong boundary enforcement and build time benefits of Gradle modules.