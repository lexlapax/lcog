# Mobile Application: [Your Project Name] - Cross-Platform Component

**This README provides details specific to the Cross-Platform mobile application built with [Specify Framework: Flutter OR React Native].**

For overall project goals, architecture, and structure philosophy, please refer to the main project documentation:

*   **Root README:** [../README.md](../README.md)
*   **Architecture Document:** [../architecture.md](../architecture.md)
*   **Project Structure Philosophy:** [../project-structure.md](../project-structure.md)
*   **Cross-Platform Structure Details:** [./project-structure.md](./project-structure.md) *(Link to the detailed structure doc within this directory)*

---

## 1. Overview

This directory contains the shared source code for the mobile application targeting both Android and iOS using `[Specify Framework: Flutter / React Native]` and `[Specify Language: Dart / TypeScript]`. Its main responsibilities include:

*   `[Responsibility 1, e.g., Providing a consistent user interface across Android and iOS]`
*   `[Responsibility 2, e.g., Consuming the backend API to display and manage data]`
*   `[Responsibility 3, e.g., Handling user interactions and shared state management using [State management library, e.g., Riverpod/Bloc/Redux/Zustand]]`
*   `[Responsibility 4, e.g., Implementing features X, Y, and Z using shared code]`
*   `[Responsibility 5, e.g., Interacting with native device features via plugins/packages]`

---

## 2. Getting Started

### 2.1. Prerequisites

*   **[Framework] Setup:** Follow the official installation guide for `[Flutter OR React Native]` for your operating system:
    *   Flutter: [https://docs.flutter.dev/get-started/install](https://docs.flutter.dev/get-started/install)
    *   React Native: [https://reactnative.dev/docs/environment-setup](https://reactnative.dev/docs/environment-setup) (Choose "React Native CLI Quickstart")
*   **Flutter Specific:**
    *   Flutter SDK: `[Specify version, e.g., 3.x]`
    *   Dart SDK: Bundled with Flutter.
*   **React Native Specific:**
    *   Node.js: `[Specify version range, e.g., ^16 || ^18]`
    *   Package Manager: `[Specify npm OR yarn]`
    *   Watchman: Recommended for macOS.
    *   JDK: `[Specify version, e.g., 11+]`
    *   Android Studio: For Android development/emulator.
    *   Xcode: For iOS development/simulator (Requires macOS).
*   **Emulator/Simulator or Physical Device:** An Android emulator/device and/or iOS simulator/device configured according to the framework setup guide.
*   **Code Editor:** VS Code (recommended with relevant framework extensions) or Android Studio/IntelliJ (Flutter).

*   **Verify Setup:** Run the framework's diagnostic tool:
    ```bash
    # Flutter
    flutter doctor

    # React Native (npx comes with Node.js)
    npx react-native doctor
    ```
    Address any issues reported by the doctor command.

### 2.2. Installation & Setup

1.  **Clone the main repository** (if not already done):
    ```bash
    git clone [your-repo-url]
    cd [your-project-name]/mobile-crossplatform
    ```

2.  **Install Dependencies:**
    *   **Flutter:**
        ```bash
        flutter pub get
        ```
    *   **React Native:**
        ```bash
        npm install # or yarn install
        # Additional step for iOS pods
        cd ios && pod install && cd ..
        ```

3.  **Environment Variables / Configuration:**
    *   This project uses `[Specify method, e.g., .env files with flutter_dotenv/react-native-dotenv, compile-time variables]` for configuration (like API base URLs, keys).
    *   Copy the example environment file:
        ```bash
        cp .env.example .env
        ```
    *   Modify the `.env` file with your local development settings.
    *   Refer to `[Link to config loading logic or documentation]` for required variables and setup (e.g., adding `.env` to assets in `pubspec.yaml` for Flutter). Ensure `.env` is in your `.gitignore`.

### 2.3. Running the Application

1.  **Ensure an emulator is running or a physical device is connected** and recognized by the framework (`flutter devices` or `npx react-native run-android --list-devices` / check Xcode).
2.  **Start the application:**
    *   **Flutter:**
        ```bash
        flutter run
        # To choose a specific device if multiple are connected:
        # flutter run -d [deviceId]
        ```
    *   **React Native:**
        *   **Start Metro Bundler (in a separate terminal):**
            ```bash
            npm start # or yarn start
            ```
        *   **Run on Android:**
            ```bash
            npm run android # or yarn android
            ```
        *   **Run on iOS:**
            ```bash
            npm run ios # or yarn ios
            ```

The framework will build the app, install it on the target device/emulator, and connect to the debugger/bundler. Changes in the code should trigger a hot reload/refresh.

---

## 3. Project Structure

A detailed explanation of the directory structure (potentially using packages for Flutter) and architectural layers for this cross-platform application can be found in [./project-structure.md](./project-structure.md).

Key directories (`lib/` for Flutter, `src/` or root for RN):

*   `features/`: Contains code organized by business feature (UI, state, domain, data).
*   `core/`: Shared code (reusable widgets/components, domain types, API clients, utilities).
*   `navigation/`: Routing configuration.
*   `config/`: Configuration access.
*   `di/` | `injection/`: Dependency injection setup.
*   `main.dart` | `index.js`: Application entry point.
*   `app.dart` | `App.tsx`: Root application widget/component.
*   `android/` & `ios/`: Native project host directories. Modifications here are sometimes necessary for specific native integrations or build configurations.

---

## 4. Running Tests

*   **Flutter:**
    *   **Unit Tests:** (`test/**_test.dart`)
        ```bash
        flutter test
        ```
    *   **Widget Tests:** (`test/**_test.dart` using `flutter_test`)
        ```bash
        flutter test
        ```
    *   **Integration Tests:** (`integration_test/**_test.dart`) Require a device/emulator.
        ```bash
        flutter test integration_test
        ```
    *   **Run tests with coverage:**
        ```bash
        flutter test --coverage
        # Generates coverage/lcov.info
        ```
*   **React Native:** (Uses Jest by default)
    *   **Run all tests:**
        ```bash
        npm test # or yarn test
        ```
    *   **Run tests in watch mode:**
        ```bash
        npm test -- --watch
        ```
    *   **Run tests with coverage:**
        ```bash
        npm test -- --coverage
        ```
    *   *(End-to-end tests might use separate tools like Detox or Appium)*

---

## 5. Building for Release

*   **Flutter:**
    *   **Android App Bundle (for Play Store):**
        ```bash
        flutter build appbundle --release
        # Output: build/app/outputs/bundle/release/app-release.aab
        ```
    *   **Android APK:**
        ```bash
        flutter build apk --release
        # Output: build/app/outputs/flutter-apk/app-release.apk
        ```
    *   **iOS Archive (for App Store):** Open the `ios/Runner.xcworkspace` in Xcode and follow the native iOS archiving process (see `mobile-ios/README.md`), ensuring the build configuration uses the Flutter release mode. Alternatively, use `flutter build ipa --release`.
*   **React Native:**
    *   **Android:** Follow the official guide for [generating a signed APK/AAB](https://reactnative.dev/docs/signed-apk-android). Usually involves running Gradle commands from the `android` directory:
        ```bash
        cd android && ./gradlew assembleRelease # For APK
        # or
        cd android && ./gradlew bundleRelease   # For AAB
        cd ..
        ```
    *   **iOS:** Open the `ios/[YourProjectName].xcworkspace` in Xcode and follow the native iOS archiving process (see `mobile-ios/README.md`), ensuring the build configuration is set to `Release`.

---

## 6. Linting and Formatting

*   **Flutter (Dart):**
    *   **Format:**
        ```bash
        dart format .
        ```
    *   **Analyze/Lint:** (Uses rules from `analysis_options.yaml`)
        ```bash
        flutter analyze
        # or
        dart analyze
        ```
*   **React Native (TypeScript/JavaScript):**
    *   Uses `[Specify tools, e.g., ESLint, Prettier]`. Configuration files (`.eslintrc.js`, `.prettierrc.js`) are typically in the root.
    *   **Run linter:**
        ```bash
        npm run lint # or yarn lint
        ```
    *   **Run formatter:**
        ```bash
        npm run format # or yarn format
        ```

---

## 7. Deployment

*   Refer to the main **[Architecture Document](../architecture.md#section-8-infrastructure--deployment)** for the overall deployment strategy.
*   Deployment involves building the release artifacts for each platform (Android AAB, iOS IPA) as described in the "Building for Release" section.
*   Upload the generated artifacts to the respective app stores:
    *   **Android:** Google Play Console.
    *   **iOS:** App Store Connect (often via Xcode Organizer or Transporter app).
*   Consider using CI/CD services specialized for mobile (like Codemagic, Bitrise, App Center, or GitHub Actions with appropriate mobile build actions) to automate building and deployment.

---

## 8. Contributing

Refer to the main project's contributing guidelines `CONTRIBUTING.md` in the root directory. Follow idiomatic coding conventions for `[Dart/TypeScript]` and `[Flutter/React Native]`. Adhere to the configured linter/formatter rules.

---