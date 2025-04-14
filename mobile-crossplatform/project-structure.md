A recommended structure for a cross-platform mobile app using technologies like **Flutter** or **React Native**. The core principles of Feature Slicing and Onion/Clean Architecture remain highly applicable, promoting maintainable and scalable codebases even with a single shared codebase for UI and logic.

We'll use terminology that bridges both Flutter (Dart) and React Native (JavaScript/TypeScript), specifying framework-specific examples where helpful. Modularity (using Dart packages in Flutter or potentially Yarn/NPM workspaces/strict directory conventions in React Native) is key.

**Recommended Structure (Conceptual):**

```
mobile-crossplatform/
├── lib/  (Flutter)  OR  src/  (React Native)     # Main shared codebase
│   │
│   ├── features/               # Top-level directory for all business features
│   │   ├── profile/            # --- Feature: User Profile ---
│   │   │   ├── domain/         # Innermost: Models, Use Cases, Repository Interfaces (Ports)
│   │   │   │   ├── models/
│   │   │   │   │   └── user_profile.dart | UserProfile.ts
│   │   │   │   ├── usecases/
│   │   │   │   │   └── update_profile_usecase.dart | updateProfileUseCase.ts # Interface + Impl or just Impl
│   │   │   │   └── repositories/
│   │   │   │       └── profile_repository.dart | IProfileRepository.ts # *Interface/Protocol/Abstract Class* (Port)
│   │   │   │
│   │   │   ├── data/           # Infrastructure Adapters: Repository Impls, Data Sources
│   │   │   │   ├── repositories/
│   │   │   │   │   └── profile_repository_impl.dart | ProfileRepository.ts # Concrete Implementation
│   │   │   │   ├── datasources/    # Defines interfaces for specific data sources
│   │   │   │   │   ├── profile_remote_datasource.dart | IProfileRemoteDataSource.ts
│   │   │   │   │   └── profile_local_datasource.dart | IProfileLocalDataSource.ts
│   │   │   │   ├── remote/         # Concrete Remote Datasource (e.g., using http/dio or fetch/axios)
│   │   │   │   │   └── profile_remote_datasource_impl.dart | ProfileRemoteDataSource.ts
│   │   │   │   └── local/          # Concrete Local Datasource (e.g., using shared_preferences/sqflite or AsyncStorage/Realm)
│   │   │   │       └── profile_local_datasource_impl.dart | ProfileLocalDataSource.ts
│   │   │   │
│   │   │   ├── presentation/   # UI Layer: Widgets/Components, State Management (Bloc/ViewModel/Store)
│   │   │   │   ├── state/        # State Management Logic (Bloc, Provider, Riverpod Notifier, Redux Reducer/Actions, Zustand Store)
│   │   │   │   │   └── profile_bloc.dart | profileStore.ts
│   │   │   │   ├── pages/ | screens/ # Feature's main screens/pages
│   │   │   │   │   └── profile_screen.dart | ProfileScreen.tsx
│   │   │   │   └── widgets/ | components/ # UI parts specific to this feature
│   │   │   │       └── profile_header.dart | ProfileHeader.tsx
│   │   │   │
│   │   │   └── profile_feature.dart | index.ts # Barrel file exporting necessary parts
│   │   │
│   │   └── orders/             # --- Feature: Order History ---
│   │       │                   # (Structure similar to profile)
│   │       └── ...
│   │
│   ├── core/                   # Shared code across multiple features (structured by layers)
│   │   ├── domain/             # Innermost Shared: Core Models, Shared Repo Interfaces
│   │   │   ├── models/         # e.g., User.dart | User.ts
│   │   │   └── repositories/   # e.g., auth_repository.dart | IAuthRepository.ts (Interface/Port)
│   │   │
│   │   ├── data/               # Shared Infrastructure: Shared Repo Impls, API Client, DB Setup
│   │   │   ├── repositories/   # e.g., auth_repository_impl.dart | AuthRepository.ts (Implementation)
│   │   │   ├── datasources/    # Interfaces for shared sources
│   │   │   ├── network/        # Shared HTTP client setup (dio instance, axios config)
│   │   │   └── persistence/    # Shared local storage setup (shared_preferences instance, DB helper)
│   │   │
│   │   ├── presentation/       # Shared UI: Reusable Widgets/Components, Theme, Layouts
│   │   │   ├── theme/
│   │   │   ├── widgets/ | components/ # e.g., AppButton, LoadingIndicator
│   │   │   └── layouts/        # e.g., MainAppShell
│   │   │
│   │   ├── common/             # Shared Utilities: Non-layer specific utils, constants, extensions
│   │   │   ├── utils/
│   │   │   ├── constants/
│   │   │   ├── extensions/
│   │   │   └── helpers/        # e.g., Resource class (Success/Error/Loading)
│   │   │
│   │   └── core.dart | index.ts # Barrel file for core exports
│   │
│   ├── navigation/             # Routing/Navigation setup
│   │   └── app_router.dart | AppNavigator.tsx
│   │
│   ├── config/                 # App-level configuration (environment, themes - sometimes part of core/presentation)
│   │   └── environment.dart | config.ts
│   │
│   ├── di/ | injection/        # Dependency Injection setup (e.g., using get_it/injectable, Riverpod providers, or manual/Context)
│   │   └── injector.dart | dependencyContainer.ts
│   │
│   └── main.dart | index.js    # Application entry point
│   └── app.dart | App.tsx      # Root Widget/Component, Provider setup, MaterialApp/NavigationContainer
│
├── android/                    # Native Android project shell/host
├── ios/                        # Native iOS project shell/host
├── test/ | __tests__/          # Unit, widget/component, integration, and potentially E2E tests
│   ├── features/
│   │   └── profile/
│   │       ├── domain/
│   │       ├── data/
│   │       └── presentation/   # Widget/Component tests, Bloc/Store tests
│   ├── core/                   # Tests for shared code
│   └── ...
├── pubspec.yaml | package.json # Project dependencies and metadata
├── README.md
└── ... (Other config files like .eslintrc, .prettierrc, analysis_options.yaml)
```

**Explanation and Onion Layer Mapping (Cross-Platform Mobile Adaptation):**

1.  **`main.dart`/`index.js` & `app.dart`/`App.tsx` (Outer Layer - Composition Root & Framework Host):**
    *   The application entry point.
    *   Initializes essential services, logging, environment configuration.
    *   Sets up the **Dependency Injection** container (`di/`) or Provider scopes (Riverpod `ProviderScope`, `MultiProvider`, React Context Providers). This is where concrete implementations (`data` layer) are bound to interfaces/abstractions (`domain` layer).
    *   Sets up the root UI widget/component (`MaterialApp`, `CupertinoApp`, `NavigationContainer`).
    *   Configures routing/navigation (`navigation/`).
    *   Wires together the core application structure.

2.  **`core/` (Shared Code - Structured by Layers):**
    *   **`core/domain` (Innermost Layer - Shared Domain):** Pure Dart/TypeScript code. Shared business models (structs/classes), core business logic/rules, and **interfaces/abstract classes (Ports)** for shared repositories (e.g., `IAuthRepository`). No Flutter/React Native framework dependencies.
    *   **`core/common` (Shared Utilities):** Framework-agnostic utilities (logging, date formatting, `Resource` wrapper). Minimal dependencies.
    *   **`core/data` (Outer Layer - Shared Infrastructure/Adapters):** Concrete implementations for shared repository interfaces from `core/domain`. Includes shared setup for networking (`http`/`dio`/`axios` instances, interceptors), persistence (`shared_preferences`/`AsyncStorage` wrappers, database connections), etc. Depends on `core/domain`, `core/common`.
    *   **`core/presentation` (Outer Layer - Shared UI/Presentation):** Reusable UI Widgets/Components (`AppButton`, `LoadingIndicator`), application theme definitions, shared layouts. Depends on the UI framework (Flutter/React Native) and potentially `core/common`.

3.  **`features/` (Vertical Slices with Internal Layers):**
    *   Each subdirectory represents a distinct user-facing feature. Modularity can be enforced using Dart packages (Flutter) or strict directory conventions/workspaces (React Native).
    *   **`domain/` (Innermost - Feature Domain):** Feature-specific models, business logic, Use Case definitions (often classes/functions orchestrating repository calls), and feature-specific repository **interfaces (Ports)**. Depends on `core/domain`, `core/common`. No UI/Infrastructure framework dependencies.
    *   **`data/` (Outer Layer - Feature Infrastructure/Adapters):** Concrete implementations of the feature's repository interfaces (from its own `domain/`). Defines and implements specific data sources (`remote`/`local`) required by the repository implementation. May use shared clients/DB setup from `core/data`. Depends on its own `domain/`, `core/domain`, `core/common`, `core/data`.
    *   **`presentation/` (Outer Layer - Feature UI/Presentation):**
        *   `state/`: State management logic (Blocs, Cubits, ViewModels, Stores, Reducers). Fetches data via Use Cases or Repository interfaces (injected), manages UI state, handles user interactions. Depends on its `domain/`, `core/domain`, `core/common`.
        *   `pages/`/`screens/` & `widgets/`/`components/`: UI code (Flutter Widgets or React Native Components). Consumes state from the `state/` layer, renders UI, dispatches events to the state management layer. Depends on its `state/`, `core/presentation`, potentially `domain/` models (for display).

**Dependency Flow (Onion Rule Enforcement):**

*   Dependencies flow inwards: `Presentation` -> `Domain` (via Use Cases/Repo Interfaces), `Data` -> `Domain` (implements Repo Interfaces).
*   State Management (`presentation/state`) depends on `Domain` (Use Cases/Repo Interfaces), *not* directly on `Data`.
*   UI (`presentation/pages`, `presentation/widgets`) depends on `presentation/state` and `core/presentation`.
*   `Data` layer (Repositories, DataSources) implements `Domain` interfaces and uses `core/data` infrastructure.
*   `Domain` layer depends only on `core/domain` and `core/common` (or nothing).
*   `core` layers follow the same inward dependency flow.
*   Features depend on `core` layers but generally *not* on other features directly (navigation handles transitions).
*   DI/Service Location (`di/` or `main`/`app`) wires concrete `Data` implementations to `Domain` interfaces used by the `Presentation` (State) layer.

**Benefits:**

*   **Testability:** Domain logic is pure Dart/TS. State Management logic can be tested by mocking Use Cases/Repositories. Data layer can be tested by mocking data sources or network calls. Widgets/Components can be unit/integration tested.
*   **Maintainability:** Code for a feature is co-located. Clear boundaries (especially with Flutter packages) reduce coupling. Easier navigation and understanding.
*   **Scalability:** Structure supports growing complexity and larger teams. Modular builds (Flutter packages) improve compilation times.
*   **Code Sharing:** Maximizes code reuse across iOS and Android via the shared `lib`/`src` directory.
*   **Flexibility:** Easier to swap data sources, state management solutions, or even UI components within defined boundaries.

This structure provides a robust, scalable, and testable foundation for cross-platform mobile development using Flutter or React Native, applying proven architectural patterns to manage complexity. Remember to choose the modularity approach (Flutter packages vs. RN directories/workspaces) that best suits your team and project size.