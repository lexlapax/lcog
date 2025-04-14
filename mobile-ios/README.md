# Mobile Application: [Your Project Name] - Native iOS Component

**This README provides details specific to the Native iOS application.**

For overall project goals, architecture, and structure philosophy, please refer to the main project documentation:

*   **Root README:** [../README.md](../README.md)
*   **Architecture Document:** [../architecture.md](../architecture.md)
*   **Project Structure Philosophy:** [../project-structure.md](../project-structure.md)
*   **iOS Structure Details:** [./project-structure.md](./project-structure.md) *(Link to the detailed structure doc within this directory)*

---

## 1. Overview

This directory contains the source code for the native iOS application built with Swift (`[Specify version, e.g., v5.7+]`) and SwiftUI (`[Confirm if SwiftUI is primary UI]`). Its main responsibilities include:

*   `[Responsibility 1, e.g., Providing the native user interface for iOS users]`
*   `[Responsibility 2, e.g., Consuming the backend API to display and manage data]`
*   `[Responsibility 3, e.g., Handling user interactions and local state management using Combine/AsyncAwait and ObservableObject]`
*   `[Responsibility 4, e.g., Utilizing device features like Camera, Location, etc.]`
*   `[Responsibility 5, e.g., Persisting data locally using CoreData/Realm]`

---

## 2. Getting Started

### 2.1. Prerequisites

*   **macOS:** `[Specify minimum version, e.g., Monterey 12.x]` or newer.
*   **Xcode:** `[Specify minimum version, e.g., 14.x]` or newer. ([Download from Mac App Store](https://apps.apple.com/us/app/xcode/id497799835))
*   **Swift:** Included with Xcode. Version `[Specify version, e.g., 5.7+]`.
*   **Simulator or Physical Device:** Running iOS `[Specify minimum target iOS version, e.g., 15.0]` or higher.
*   **(Optional) Swift Package Manager (SPM):** Used for managing dependencies (integrated into Xcode).
*   **(Optional) CocoaPods / Carthage:** If used for dependency management (less common with SPM focus).

### 2.2. Installation & Setup

1.  **Clone the main repository** (if not already done):
    ```bash
    git clone [your-repo-url]
    cd [your-project-name]/mobile-ios
    ```

2.  **Open Project in Xcode:**
    *   Locate the `[YourAppWorkspace.xcworkspace]` file (if using local SPM packages or CocoaPods) or the `[YourApp.xcodeproj]` file within the `mobile-ios` directory.
    *   Double-click the `.xcworkspace` or `.xcodeproj` file to open it in Xcode.
    *   Xcode will automatically resolve Swift Package dependencies if defined in `Package.swift` files or the project settings. This might take a moment.
    *   If using CocoaPods: Run `pod install` in the `mobile-ios` directory first, then open the `.xcworkspace`.

3.  **Configuration / Secrets:**
    *   Sensitive information like API keys should **not** be hardcoded. This project uses `[Specify method, e.g., values stored in configuration files (like .xcconfig) and accessed via Info.plist, or loaded from a secure properties file]`.
    *   Create/configure the necessary files:
        ```bash
        # Example if using configuration files not checked into git
        # cp Configuration/Secrets.example.xcconfig Configuration/Secrets.xcconfig
        # Edit Configuration/Secrets.xcconfig with required keys (e.g., API_KEY = YOUR_KEY)
        ```
    *   Ensure these configuration files are included in `.gitignore`. Refer to `[Link to config access logic or documentation]` for required keys/setup.

### 2.3. Running the Application

1.  **Select Target Scheme & Device:** In Xcode's toolbar (top-left), select the main application scheme (e.g., `YourApp`). Then, select a target Simulator or connected physical device from the adjacent dropdown list.
2.  **Run:** Click the "Run" button (triangle icon ▶️) in the toolbar or use the keyboard shortcut `Cmd + R` (⌘R).

Xcode will build the application, install it on the selected simulator/device, and launch it. Debug logs appear in Xcode's console output pane.

---

## 3. Project Structure

A detailed explanation of the directory structure, Swift Packages (if used), and architectural layers for this iOS application can be found in [./project-structure.md](./project-structure.md).

Key directories/packages:

*   `YourApp/`: Main application target, App Delegate/Scene Delegate or `@main` App struct, Info.plist, Assets.
*   `Packages/`: Local Swift Packages for modularization.
    *   `Core/`: Shared packages (CoreDomain, CoreData, CoreUI, CoreCommon).
    *   `Features/`: Packages for specific application features (e.g., `ProfileFeature`, `OrdersFeature`). Each typically contains `Domain`, `Data`, `Presentation` layers.

---

## 4. Running Tests

Xcode provides built-in support for running tests defined within the project or packages.

*   **Run All Tests:** Use the keyboard shortcut `Cmd + U` (⌘U) or go to `Product > Test`.
*   **Run Specific Test Target/Suite/Case:**
    *   Open the Test Navigator (diamond icon ◊ in the left sidebar).
    *   Click the play button next to the desired test target, suite, or individual test case.
*   **Run Tests via Command Line (xcodebuild):**
    ```bash
    # Example (adjust scheme, destination)
    xcodebuild test \
      -workspace YourAppWorkspace.xcworkspace \
      -scheme YourApp \
      -destination 'platform=iOS Simulator,name=iPhone 14 Pro'
    ```

Tests are typically located within a `Tests` directory inside each Swift Package or within the main app target's test target.

---

## 5. Building for Distribution (Archiving)

*   **Archive:**
    1.  Select `Product > Archive` from the Xcode menu. (Ensure a physical device or "Any iOS Device (arm64)" is selected as the run destination, not a simulator).
    2.  Xcode will build and archive the application. The Organizer window will appear upon completion.
*   **Distribute:**
    1.  In the Organizer window, select the archive you just created.
    2.  Click "Distribute App".
    3.  Follow the wizard to choose a distribution method (App Store Connect, Ad Hoc, Development, Enterprise).
    4.  Xcode will handle code signing and package the app accordingly (usually as an `.ipa` file).

---

## 6. Linting and Formatting

This project uses `[Specify tools, e.g., SwiftLint, Xcode's built-in formatter]` to maintain code quality.

*   **SwiftLint (if configured):**
    *   Linting might run automatically as part of the build process if integrated via Build Phases.
    *   Run manually (if installed):
        ```bash
        swiftlint lint # Run in the project root or specific directories
        swiftlint autocorrect # To fix violations automatically
        ```
*   **Xcode Formatting:** Use `Ctrl + I` (⌃I) to re-indent code or configure formatting settings in Xcode Preferences (Text Editing > Indentation).

---

## 7. Deployment

*   Refer to the main **[Architecture Document](../architecture.md#section-8-infrastructure--deployment)** for the overall deployment strategy.
*   Deployment to the Apple App Store involves uploading the archived build (`.ipa`) generated in the distribution step via [App Store Connect](https://appstoreconnect.apple.com/) or using Xcode's Organizer window directly.
*   Using TestFlight for beta testing via App Store Connect is highly recommended before production releases.

---

## 8. Contributing

Refer to the main project's contributing guidelines `CONTRIBUTING.md` in the root directory. Follow standard Swift coding conventions ([Swift API Design Guidelines](https://www.swift.org/documentation/api-design-guidelines/)) and iOS development best practices. Adhere to the configured `SwiftLint` rules.

---