# Mobile Application: [Your Project Name] - Native Android Component

**This README provides details specific to the Native Android application.**

For overall project goals, architecture, and structure philosophy, please refer to the main project documentation:

*   **Root README:** [../README.md](../README.md)
*   **Architecture Document:** [../architecture.md](../architecture.md)
*   **Project Structure Philosophy:** [../project-structure.md](../project-structure.md)
*   **Android Structure Details:** [./project-structure.md](./project-structure.md) *(Link to the detailed structure doc within this directory)*

---

## 1. Overview

This directory contains the source code for the native Android application built with Kotlin (`[Specify version, e.g., v1.7+]`) and Jetpack libraries, primarily Jetpack Compose (`[Confirm if Compose is used]`). Its main responsibilities include:

*   `[Responsibility 1, e.g., Providing the native user interface for Android users]`
*   `[Responsibility 2, e.g., Consuming the backend API to display and manage data]`
*   `[Responsibility 3, e.g., Handling user interactions and local state management via ViewModels/StateFlow]`
*   `[Responsibility 4, e.g., Utilizing device features like Camera, Location, etc.]`
*   `[Responsibility 5, e.g., Persisting data locally using Room]`

---

## 2. Getting Started

### 2.1. Prerequisites

*   **Android Studio:** `[Specify recommended version, e.g., Dolphin | Electric Eel]` or newer. ([Download](https://developer.android.com/studio))
*   **Android SDK:** Target SDK `[Specify API level, e.g., 33]` and Minimum SDK `[Specify API level, e.g., 24]`. Ensure required SDK Platforms and Build-Tools are installed via the SDK Manager in Android Studio.
*   **Kotlin Plugin:** Included with recent Android Studio versions.
*   **Emulator or Physical Device:** Running Android API `[Minimum SDK Level]` or higher.
*   **(Optional) JDK:** `[Specify version, e.g., JDK 11 or 17]` (Often bundled with Android Studio).

### 2.2. Installation & Setup

1.  **Clone the main repository** (if not already done):
    ```bash
    git clone [your-repo-url]
    cd [your-project-name]/mobile-android
    ```

2.  **Open Project in Android Studio:**
    *   Launch Android Studio.
    *   Select "Open" or "Open an Existing Project".
    *   Navigate to and select the `mobile-android` directory within the cloned repository.
    *   Android Studio will automatically sync the Gradle project. This might take a few minutes the first time.

3.  **Environment Variables / API Keys:**
    *   Sensitive information like API keys should **not** be hardcoded. This project uses `[Specify method, e.g., values stored in local.properties and accessed via BuildConfig, or secrets-gradle-plugin]`.
    *   Create/configure the necessary files:
        ```bash
        # Example if using local.properties
        cp local.properties.example local.properties
        # Edit local.properties with required keys (e.g., API_KEY="YOUR_KEY")
        ```
    *   Refer to `[Link to config access logic or build.gradle file]` for required keys. Remember to add `local.properties` to your `.gitignore` file (it usually is by default).

### 2.3. Running the Application

1.  **Select Target Device:** In Android Studio's toolbar, choose an available emulator or connected physical device.
2.  **Select Build Variant:** Ensure the correct build variant (e.g., `debug`) is selected in the "Build Variants" tool window (usually View > Tool Windows > Build Variants).
3.  **Run:** Click the "Run 'app'" button (green play icon ▶️) in the toolbar or use the menu option `Run > Run 'app'`.

Android Studio will build the application, install it on the selected device/emulator, and launch it. Logs can be viewed in the "Logcat" tool window.

---

## 3. Project Structure

A detailed explanation of the directory structure, Gradle modules, and architectural layers for this Android application can be found in [./project-structure.md](./project-structure.md).

Key modules/directories:

*   `app/`: Main application module, entry point, navigation host.
*   `core/`: Shared modules (domain, data, ui, common).
*   `feature/`: Modules for specific application features (e.g., `profile`, `orders`). Each typically contains `domain`, `data`, `presentation` layers.

---

## 4. Running Tests

Android Studio provides integration for running tests.

*   **Unit Tests:** Located typically in `module/src/test/java/...`
    *   Right-click on a test file, class, method, or directory in the Project view and select "Run 'Tests in ...'".
    *   Run all unit tests for a module via Gradle:
        ```bash
        ./gradlew :[moduleName]:testDebugUnitTest
        # Example: ./gradlew :feature:profile:testDebugUnitTest
        ```
*   **Instrumentation Tests (UI/Integration):** Located typically in `module/src/androidTest/java/...`
    *   Require an emulator or physical device.
    *   Right-click on a test file, class, method, or directory in the Project view and select "Run 'Tests in ...'".
    *   Run all instrumentation tests for a module via Gradle:
        ```bash
        ./gradlew :[moduleName]:connectedDebugAndroidTest
        # Example: ./gradlew :feature:profile:connectedDebugAndroidTest
        ```
*   **Run all tests project-wide (use with caution, can be slow):**
    ```bash
    ./gradlew test connectedAndroidTest
    ```

---

## 5. Building the Application (APK/AAB)

*   **Generate Signed Bundle/APK:** Use Android Studio's build menu: `Build > Generate Signed Bundle / APK...`
    *   Follow the wizard to create or use an existing upload key/keystore.
    *   Choose **Android App Bundle (.aab)** for distribution via Google Play (recommended).
    *   Choose **APK** for direct installation or other distribution methods.
    *   Select the appropriate build variant (usually `release`).
*   **Build via Gradle:**
    ```bash
    # Build debug APK
    ./gradlew assembleDebug

    # Build release AAB (requires signing configuration in build.gradle)
    ./gradlew bundleRelease
    ```
*   Build outputs are typically found in `app/build/outputs/apk/` or `app/build/outputs/bundle/`.

---

## 6. Linting and Formatting

This project uses `[Specify tools, e.g., Android Lint, ktlint]` to maintain code quality.

*   **Run Lint:**
    ```bash
    ./gradlew lintDebug # Or lintRelease
    ```
    *(Lint checks often run automatically during builds)*
*   **Run ktlint (if configured):**
    ```bash
    ./gradlew ktlintCheck # Or ktlintFormat
    ```
*   Android Studio typically provides real-time linting and formatting based on project settings (File > Settings > Editor > Code Style > Kotlin).

---

## 7. Deployment

*   Refer to the main **[Architecture Document](../architecture.md#section-8-infrastructure--deployment)** for the overall deployment strategy.
*   Deployment to Google Play involves uploading the signed **Android App Bundle (.aab)** generated in the build step via the [Google Play Console](https://play.google.com/console/).
*   Internal testing tracks (Closed, Open) are recommended before production releases.

---

## 8. Contributing

Refer to the main project's contributing guidelines `CONTRIBUTING.md` in the root directory. Follow standard Kotlin coding conventions and Android development best practices. Adhere to the configured `ktlint` rules.

---