# adx

`adx` is a blazing-fast, zero-config CLI tool for Android developers working with Gradle, Kotlin, and Java. It eliminates cumbersome Gradle wrapper (`./gradlew`) and ADB commands by automating project detection, multi-device targeting, APK installation, live logcat streaming, clean builds, emulator launching, and dependency management.

---

## Features

- 🚀 **Zero Configuration**: Automatically discovers your Android project root, Gradle wrapper, `applicationId`, launcher activities, and APK build outputs.
- 📱 **Smart Device & Emulator Management**:
  - `adx run emulator` / `adx emulator`: Lists AVDs, auto-launches if only 1 exists, or displays an interactive selection menu.
  - Automatically targets connected devices or emulators.
  - If multiple devices are connected, offers an interactive selector to choose a single device or deploy to **All connected devices**.
- 🛠️ **Simplified Builds & Runs**:
  - `adx build` / `adx build release`: Animated spinner tracking active Gradle tasks (`:app:compileDebugKotlin`), high-contrast red error reporting, and clickable output paths.
  - `adx run` / `adx run release`: Builds, selects device, installs, and launches app (`--open`).
- 📦 **Automated Dependency Management**:
  - `adx add <name>`: Searches Maven Central (or aliases like `retrofit`, `coil-compose`, `room`), injects into `libs.versions.toml` or `build.gradle[.kts]`, and downloads with Gradle.
  - `adx update`: Checks for newer stable releases and upgrades project versions.
- 🧹 **Deep Clean / Nuke**:
  - `adx nuke`: Fixes corrupted Kotlin, KSP, and KAPT caches that plague developers after git branch switching.
  - `adx kill`: Kills rogue daemons and restarts ADB server.
- 📸 **Media & Testing Utilities**:
  - `adx screenshot [file.png]`: Captures device screen with clickable file link.
  - `adx record [file.mp4]`: Records device video until Ctrl+C.
  - `adx open <url>`: Tests deep links and app links.
  - `adx sha`: Extracts SHA-1 and SHA-256 fingerprints for Firebase and Google APIs.
  - `adx uninstall`: Cleans up app packages from device(s).
  - `adx reverse 8080`: Reverse port forwards localhost to Android device.
  - `adx clear`: Clears app data & cache without reinstalling.
  - `adx test`: Fast unit test execution with clickable HTML report.
  - `adx lint`: Android Lint analysis with clickable report.
  - `adx bundle`: Builds Google Play `.aab` bundles.

---

## Installation

```bash
cd /Users/shashwat/Desktop/andcli
go install .
```

`adx` is installed at:
- `/opt/homebrew/bin/adx`
- `/Users/shashwat/.local/bin/adx`
- `~/go/bin/adx`

---

## Command Reference

### 1. Emulator
```bash
# Auto-launch if 1 AVD exists, or show picker if multiple
adx emulator
adx run emulator

# Launch specific AVD directly
adx emulator Pixel_8_API_34
```

### 2. Build APK
```bash
# Build debug APK with animated spinner
adx build

# Build release APK
adx build release

# Verbose live output
adx build -v
```

### 3. Run App (Build + Deploy + Launch)
```bash
# Build debug, install on connected device, and launch app
adx run

# Build release, install and launch
adx run release

# Install existing APK without rebuilding
adx run --no-build
```

### 4. Dependency Management
```bash
# Add dependency from Maven Central and download with Gradle
adx add retrofit
adx add coil-compose
adx add androidx.room:room-runtime
adx add hilt-compiler --config ksp

# Check for newer stable versions of project dependencies
adx update --check

# Upgrade project dependencies and sync
adx update
```

### 5. Media Capture & Deep Links
```bash
# Take screenshot
adx screenshot
adx screenshot bug-report.png

# Record screen video (press Ctrl+C to stop)
adx record demo.mp4

# Test deep link or app link
adx open "myapp://details?id=42"
```

### 6. Keystore Fingerprints (Firebase & Google APIs)
```bash
# Print debug keystore SHA-1 and SHA-256
adx sha

# Print custom release keystore fingerprints
adx sha --keystore /path/to/release.jks --alias mykey --password secret
```

### 7. Cache Fixes & Cleanup
```bash
# Deep clean: deletes .gradle/, build/ folders, and stops daemons
adx nuke

# Kill stuck Gradle daemons and restart ADB
adx kill

# Clear app data on device (resets Room DB / SharedPreferences)
adx clear

# Uninstall app from device
adx uninstall
```

### 8. Network & Quality Tools
```bash
# Reverse port forward localhost:8080 to device
adx reverse 8080

# Run unit tests with clickable HTML report
adx test

# Run Android Lint
adx lint

# Build App Bundle (.aab for Google Play)
adx bundle

# Stream real-time logs filtered to app package
adx logs --clear

# Validate toolchain environment
adx doctor
```
