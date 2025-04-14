This combines Feature-Based (Vertical Slice) approach with the Onion Architecture (or Clean/Hexagonal Architecture principles) for an idiomatic Go backend service structure.

The core idea remains the same: organize by business capability (feature) and ensure dependencies flow inwards from UI/Frameworks towards core application logic and domain types.

Here's a potential idiomatic structure:

```md
frontend-react/
├── public/                     # Static assets (index.html, favicons, etc.)
│   └── index.html
├── src/
│   ├── features/               # Top-level directory for all business features
│   │   ├── __tests__/          # Feature-specific tests (optional placement)
│   │   │
│   │   ├── profile/            # --- Feature: User Profile ---
│   │   │   ├── __tests__/      # Unit/integration tests for this feature
│   │   │   │   ├── domain.test.ts
│   │   │   │   ├── application.test.ts # Mocking ports/services
│   │   │   │   └── ui.test.tsx         # Component tests (React Testing Library)
│   │   │   │
│   │   │   ├── domain/         # Innermost: Feature-specific types, constants, simple validation logic
│   │   │   │   └── types.ts    # (e.g., UserProfile interface)
│   │   │   │   └── validators.ts # (e.g., simple pure functions for validation)
│   │   │   │
│   │   │   ├── application/    # Application Logic: State management, use cases, ports
│   │   │   │   ├── profileStore.ts # (e.g., Zustand/Redux slice/Context state) - Defines state & actions/reducers
│   │   │   │   ├── useProfileLoader.ts # Custom hook orchestrating data fetching & state updates (a "use case" hook)
│   │   │   │   └── ports.ts    # Interfaces defining required external interactions (e.g., `IProfileApi`)
│   │   │   │
│   │   │   ├── ui/             # UI Layer: React components, presentation hooks, styles
│   │   │   │   ├── components/ # Presentational or smart components specific to the profile feature
│   │   │   │   │   └── ProfileForm.tsx
│   │   │   │   │   └── ProfileDisplay.tsx
│   │   │   │   ├── hooks/      # Custom hooks purely for UI logic/presentation within this feature
│   │   │   │   │   └── useProfileFormState.ts
│   │   │   │   ├── styles/     # Feature-specific styles (CSS Modules, styled-components)
│   │   │   │   │   └── ProfileForm.module.css
│   │   │   │   └── ProfilePage.tsx # The main component/page for this feature, assembling others
│   │   │   │
│   │   │   ├── infrastructure/ # Adapters: Concrete implementations of ports, specific external interactions
│   │   │   │   └── profileApi.ts # Implements `ports.IProfileApi` using fetch/axios, interacts with `shared/infrastructure/apiClient`
│   │   │   │
│   │   │   └── index.ts        # Barrel file: Exports the main component (`ProfilePage`) or other necessary parts for routing
│   │   │
│   │   └── orders/             # --- Feature: Order History ---
│   │       ├── domain/
│   │       │   └── types.ts    # (e.g., Order interface, OrderStatus enum)
│   │       ├── application/
│   │       │   ├── ordersStore.ts
│   │       │   ├── useOrdersList.ts
│   │       │   └── ports.ts    # (e.g., IOrdersApi)
│   │       ├── ui/
│   │       │   ├── components/
│   │       │   │   └── OrderListItem.tsx
│   │       │   ├── hooks/
│   │       │   │   └── useOrderFiltering.ts
│   │       │   ├── styles/
│   │       │   └── OrdersPage.tsx
│   │       ├── infrastructure/
│   │       │   └── ordersApi.ts # Implements `ports.IOrdersApi`
│   │       └── index.ts
│   │
│   ├── shared/                 # Code shared across multiple features
│   │   ├── __tests__/
│   │   │
│   │   ├── domain/             # Truly shared core types/constants (use sparingly)
│   │   │   └── types.ts        # (e.g., UserID, CurrencyCode, BaseEntity)
│   │   │
│   │   ├── application/        # Shared application logic/state (e.g., Auth state)
│   │   │   └── authStore.ts
│   │   │   └── useAuth.ts      # Hook providing auth state/actions
│   │   │   └── ports.ts        # Interfaces for shared ports (e.g., IAuthApi)
│   │   │
│   │   ├── ui/                 # Shared UI components, hooks, layouts, themes
│   │   │   ├── components/     # Generic, reusable presentational components (Button, Modal, InputField)
│   │   │   │   └── Button.tsx
│   │   │   │   └── Spinner.tsx
│   │   │   ├── hooks/          # Generic UI utility hooks (useDebounce, useWindowSize)
│   │   │   │   └── useDebounce.ts
│   │   │   ├── layouts/        # Page layout components (AppLayout, Sidebar)
│   │   │   │   └── AppLayout.tsx
│   │   │   └── theme/          # Theming configuration (CSS vars, styled-components theme)
│   │   │       └── index.ts
│   │   │
│   │   └── infrastructure/     # Shared infrastructure: API client setup, storage wrappers
│   │       ├── apiClient.ts    # Configured Axios/fetch instance, base URLs, interceptors
│   │       ├── storage.ts      # Wrapper around localStorage/sessionStorage
│   │       └── authApi.ts      # Implementation for `shared/application/ports.IAuthApi`
│   │
│   ├── config/                 # Application configuration (environment variables, constants)
│   │   └── index.ts
│   │
│   ├── lib/                    # Generic utility functions (non-domain, non-React)
│   │   └── formatters.ts       # (e.g., date formatting, currency formatting)
│   │   └── utils.ts
│   │
│   ├── routing/                # Routing configuration and setup
│   │   ├── index.tsx           # Defines routes, potentially lazy-loading feature components
│   │   └── protectedRoute.tsx  # Example: Component for handling authenticated routes
│   │
│   ├── styles/                 # Global styles, resets, base CSS
│   │   └── global.css
│   │
│   ├── App.tsx                 # Root component: Sets up providers, layout, routing
│   ├── main.tsx                # Application entry point (renders App into the DOM)
│   └── react-app-env.d.ts      # TypeScript definitions for environment variables
│
├── .eslintrc.js                # ESLint configuration
├── .prettierrc.js              # Prettier configuration
├── tsconfig.json               # TypeScript compiler options
├── package.json
├── README.md
└── vite.config.ts | craco.config.js # Build tool configuration (Vite/CRA)
```

**Explanation and Onion Layer Mapping (Frontend Adaptation):**

1.  **`main.tsx` / `App.tsx` (Outer Layer - Frameworks & Composition Root):**
    *   Initializes the React application.
    *   Sets up global providers (State Management Store Provider, React Router Provider, Theme Provider, Query Client Provider if using React Query/TanStack Query).
    *   Wires up routing (`src/routing`).
    *   Renders the main application layout (`src/shared/ui/layouts`).
    *   This is where dependencies are implicitly "injected" via Context providers or store setup.

2.  **`src/` (Application Core & Adapters):**
    *   **`src/features/feature_x/` (Vertical Slices):**
        *   **`domain/` (Innermost Layer - Domain):** Contains feature-specific TypeScript types/interfaces, enums, constants, and potentially simple, pure validation functions related to the feature's core concepts. Depends *only* on `src/shared/domain` (if necessary) and potentially `src/lib` for ultra-generic utils. **No React, no state management, no data fetching.**
        *   **`application/` (Application Layer):** Manages the feature's state and orchestrates actions/use cases.
            *   `*Store.ts`: State management logic (e.g., Zustand store, Redux slice, Context + useReducer). Defines the shape of the state and the actions/reducers to modify it.
            *   `use*UseCaseHook.ts` (Optional): Custom hooks that encapsulate specific application logic, often involving fetching data (by calling methods defined in `ports.ts`) and updating the store.
            *   `ports.ts`: Defines TypeScript interfaces for necessary external operations (primarily data fetching/persistence like `IProfileApi`). The application layer depends on these *abstractions*, not concrete implementations. Depends *only* on `domain/`.
        *   **`ui/` (UI Layer / Presentation Adapters):** Contains React components and hooks responsible for rendering the UI and handling user interactions *for this specific feature*.
            *   `components/`: React components, ranging from simple presentational components to "smart" components that connect to the application layer's state (`*Store.ts`) or use case hooks (`use*UseCaseHook.ts`).
            *   `hooks/`: Custom hooks focused solely on UI logic or presentation concerns within the feature (e.g., managing complex form state *before* submitting to an application use case).
            *   `*Page.tsx`: Often the main entry component for a feature, assembling other UI components.
            *   Depends *inward* on `application/` (to get state, dispatch actions, call use case hooks) and `domain/` (for types). May use components/hooks from `src/shared/ui`.
        *   **`infrastructure/` (Infrastructure Adapters):** Concrete implementations of the interfaces defined in `application/ports.ts`. This is where actual data fetching happens.
            *   `*Api.ts`: Implements the port (e.g., `IProfileApi`) using a specific library (like `axios` or `fetch`) and interacts with shared clients (`src/shared/infrastructure/apiClient`).
            *   Depends on `domain/` (for types), `application/ports.ts` (the interface it implements), and `src/shared/infrastructure` (for configured clients, storage wrappers).

    *   **`src/shared/` (Shared Code Across Layers):**
        *   `domain/`: Truly core types shared by multiple features (e.g., `UserID`). Innermost layer.
        *   `application/`: Shared state or logic (e.g., authentication state and actions). Application Layer. Defines its own ports if needed.
        *   `ui/`: Reusable UI components, hooks, layouts, themes. UI Layer. Depends potentially on `shared/application` (e.g., an auth-aware button).
        *   `infrastructure/`: Shared setup for external interactions (configured API client, localStorage wrapper). Infrastructure Layer. Implements ports from `shared/application` (like `IAuthApi`).

    *   **`src/lib/` (Frameworks/Drivers - Utilities):** Generic, framework-agnostic utility functions (e.g., date formatting). Doesn't depend on React or application state.
    *   **`src/config/` (External Configuration):** Accessing environment variables, etc.
    *   **`src/routing/` (Frameworks/Drivers):** Uses the React Router library to define application routes, often lazy-loading feature entry points (`features/profile/index.ts`).

**Dependency Flow (Onion Rule Enforcement):**

*   `features/**/ui/` depends on `features/**/application/`, `features/**/domain/`, `shared/ui`, `shared/application`, `shared/domain`.
*   `features/**/infrastructure/` depends on `features/**/application/ports.ts`, `features/**/domain/`, `shared/infrastructure`, `shared/domain`.
*   `features/**/application/` depends on `features/**/domain/`, `features/**/application/ports.ts`, `shared/application`, `shared/domain`. **Crucially, it does NOT depend on `ui/` or `infrastructure/`.**
*   `features/**/domain/` depends only on `shared/domain` (if needed).
*   `shared/*` follows similar rules within its scope. `shared/ui` depends on `shared/application`/`shared/domain`, `shared/infrastructure` depends on `shared/application`/`shared/domain`, `shared/application` depends on `shared/domain`.
*   Outer layers (`routing`, `App.tsx`, `main.tsx`, `lib`, `config`) depend inwards or on external libraries.

**Benefits:**

*   **High Cohesion:** All code related to a feature (types, state, UI, API calls) is grouped.
*   **Low Coupling:** Features are relatively isolated. Changes in `profile` are unlikely to break `orders`. Shared code is explicit.
*   **Testability:**
    *   Domain logic (`domain/`) is pure and easily unit-tested.
    *   Application logic (`application/`) can be tested by mocking ports (`ports.ts`).
    *   UI components (`ui/`) can be tested (e.g., with React Testing Library) by providing mock state/handlers or mocking the application layer hooks/stores.
    *   Infrastructure adapters (`infrastructure/`) can be integration-tested or unit-tested by mocking the underlying HTTP client.
*   **Maintainability & Scalability:** Easier to navigate, understand feature boundaries, and onboard new developers. Refactoring within a feature is contained.
*   **Replaceability:** Easier to swap UI libraries (less likely, but possible), state management, or data fetching implementations by changing adapters/implementations without rewriting core logic.

This structure provides a solid foundation for complex React applications, promoting modularity and testability through feature slicing and layered architecture principles. Adjust the strictness and specific folder names based on your team's preferences and project complexity.