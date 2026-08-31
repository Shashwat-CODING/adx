<p align="center">
  <img src="adx.png" alt="ADX Banner" width="100%">
</p>

<p align="center">
  <h1 align="center">⚡ ADX — Android Developer Experience CLI</h1>
  <p align="center">
    <strong>Making Android development effortless — eliminate the pain of messy Gradle and ADB commands.</strong>
  </p>
  <p align="center">
    Developed with ❤️ by <a href="https://github.com/Shashwat-CODING"><strong>Shashwat</strong></a>
  </p>
  <p align="center">
    <a href="https://github.com/Shashwat-CODING/adx/releases"><img src="https://img.shields.io/github/v/release/Shashwat-CODING/adx?color=blue&style=flat-square" alt="Latest Release"></a>
    <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go" alt="Go Version"></a>
    <a href="https://github.com/Shashwat-CODING/adx/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License"></a>
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey?style=flat-square" alt="Platform Support">
  </p>
</p>

---

```
           _       
  __ _  __| |_  __ 
 / _  |/ _  | \ \/ /
| (_| | (_| |  ><  
 \__,_|\__,_|/_/\_\
 android developer experience cli
```

---

## 💡 Why ADX?

Tired of typing long `./gradlew assembleDebug` commands, memorizing esoteric `adb` flags, wrestling with hung daemons, or manually hunting library coordinates on Maven Central?

**ADX** is a unified developer experience CLI designed to eliminate the friction, verbosity, and maintenance overhead of Gradle and ADB in Android projects. Whether you build in **Kotlin** or **Java**, ADX handles the heavy lifting behind the scenes so you can focus on shipping features.

### The Problem vs. The Solution

| Developer Task | Traditional Android Workflow | With ADX |
|---|---|---|
| **Build & Deploy** | `./gradlew assembleDebug` + `adb install -r -t ...` + `adb shell am start ...` | `adx run` |
| **Multiple Devices** | `adb devices` ➜ copy serial ➜ `adb -s <serial> install ...` | Interactive selection menu or deploy to all devices |
| **Add Dependency** | Search browser ➜ find version ➜ edit `libs.versions.toml` ➜ sync | `adx add retrofit` (auto-resolves & syncs) |
| **Update Libraries** | Check dependencies manually one by one | `adx update` |
| **Corrupted Caches** | Build breaks after git branch switch; manual `rm -rf .gradle build` | `adx nuke` |
| **Hung Daemons / ADB** | Find PIDs, kill processes, restart adb server | `adx kill` |
| **App Logs** | `adb logcat` (drowning in OS noise) | `adx logs` (filtered to app package & active PID) |
| **Launch Emulator** | Open Android Studio Device Manager or type long emulator path | `adx run emulator` |
| **Firebase / API Keys** | Run lengthy `keytool -list -v` commands | `adx sha` |
| **Local API Testing** | `adb reverse tcp:8080 tcp:8080` | `adx reverse 8080` |
| **Take Screenshot** | Android Studio screenshot button or `adb exec-out screencap` | `adx screenshot` (with clickable link) |

---

## 🚀 Installation

### Option 1: One-Line Installers

#### macOS & Linux (Terminal)
```bash
curl -fsSL https://raw.githubusercontent.com/Shashwat-CODING/adx/main/install.sh | bash
```

#### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/Shashwat-CODING/adx/main/install.ps1 | iex
```

---

### Option 2: Using Go

```bash
go install github.com/Shashwat-CODING/adx@latest
```
*Ensure `$GOPATH/bin` or `~/go/bin` is added to your shell's `PATH`:*
```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

---

### Option 3: Prebuilt Standalone Binaries

Download standalone precompiled binaries directly from [GitHub Releases](https://github.com/Shashwat-CODING/adx/releases):

| OS | Architecture | Download |
|---|---|---|
| **macOS** | Apple Silicon (M1/M2/M3/M4) | [Download tar.gz](https://github.com/Shashwat-CODING/adx/releases) |
| **macOS** | Intel (x86_64) | [Download tar.gz](https://github.com/Shashwat-CODING/adx/releases) |
| **Linux** | x86_64 | [Download tar.gz](https://github.com/Shashwat-CODING/adx/releases) |
| **Linux** | ARM64 | [Download tar.gz](https://github.com/Shashwat-CODING/adx/releases) |
| **Windows** | x86_64 (`adx.exe`) | [Download zip](https://github.com/Shashwat-CODING/adx/releases) |

#### Quick Manual Setup:
```bash
# Example for macOS Apple Silicon
tar -xzf adx_*_darwin_arm64.tar.gz
sudo mv adx /usr/local/bin/
```

---

### Option 4: Homebrew (macOS & Linux)

```bash
brew install Shashwat-CODING/tap/adx
```

---

### Option 5: Build From Source

```bash
git clone https://github.com/Shashwat-CODING/adx.git
cd adx
go build -o adx .
sudo mv adx /usr/local/bin/
```

---

## 📖 Command Reference

### 🛠️ Build & Run Workflows

#### Build APK
Builds debug or release APKs with a clean animated task spinner and clickable output links.
```bash
adx build                # Build debug APK
adx build release        # Build release APK
adx build -v             # Build with full verbose streaming logs
adx build debug --clean  # Clean before building
```

#### Run App (Build + Deploy + Launch)
Builds the APK, lets you select from connected physical devices or emulators, installs the APK, and launches the app.
```bash
adx run                  # Build debug, select target, install, and open app
adx run release          # Build release, install, and open app
adx run --no-build       # Skip rebuild, install and open existing APK
adx run -d emulator-5554 # Deploy to a specific device serial
```

#### Launch Emulator
Lists available Android Virtual Devices (AVDs). If only 1 exists, launches it immediately; if multiple exist, displays an interactive selector.
```bash
adx emulator             # Launch AVD
adx run emulator         # Alternative alias
adx emulator Pixel_8     # Launch specific AVD
```

---

### 📦 Dependency Management

#### Add Dependency
Searches Maven Central, finds the latest stable version, injects coordinates into your Version Catalog (`libs.versions.toml`) or `build.gradle[.kts]`, and downloads it with Gradle.
```bash
adx add retrofit
adx add coil-compose
adx add androidx.room:room-runtime
adx add hilt-compiler --config ksp
adx add junit --config testImplementation
```

#### Update Dependencies
Scans declared project dependencies, queries Maven Central for newer stable releases, and updates your project files.
```bash
adx update --check       # Preview available updates without editing files
adx update               # Update all dependencies and sync with Gradle
```

---

### 🧹 Cache & Daemon Troubleshooting

#### Nuke Corrupted Caches
Fixes corrupted Kotlin, KSP, and KAPT caches that break builds after switching branches.
```bash
adx nuke                 # Deletes .gradle/, all build/ folders, and stops daemons
```

#### Kill Rogue Daemons & Reset ADB
Terminates stuck Gradle/Kotlin daemons consuming 100% CPU/RAM and restarts the ADB server.
```bash
adx kill
```

#### Clean Build Outputs
Standard `./gradlew clean`.
```bash
adx clean
```

---

### 📱 Device & Media Utilities

#### Live App Logs
Streams real-time logcat output filtered strictly to your app's package ID and active process ID.
```bash
adx logs                 # Stream logs for current application
adx logs --clear         # Clear logcat buffer before streaming
adx logs -p com.pkg      # Stream logs for explicit package name
```

#### Screen Capture & Recording
Captures media directly from the device with IDE-clickable `file:///...` links.
```bash
adx screenshot           # Take screenshot (saves screenshot-<timestamp>.png)
adx screenshot bug.png   # Save to specific file
adx record demo.mp4      # Record screen video (press Ctrl+C to stop)
```

#### Deep Link Testing
Dispatches an `android.intent.action.VIEW` intent directly to the connected device.
```bash
adx open "myapp://details?id=42"
adx open "https://example.com/checkout"
```

#### Keystore Fingerprints (Firebase & Google APIs)
Extracts and displays MD5, SHA-1, and SHA-256 fingerprints from `~/.android/debug.keystore` (or custom keystores) in clean, copy-pasteable format.
```bash
adx sha
adx sha --keystore /path/to/release.jks --alias mykey --password secret
```

#### App & Network Utilities
```bash
adx reverse 8080         # Forward localhost:8080 to Android device
adx clear                # Clear app data & cache on device (Room, SharedPreferences)
adx uninstall            # Uninstall app from connected device(s)
adx stop                 # Force stop app process
adx devices              # List all attached devices and emulators
adx doctor               # Validate JDK, Android SDK, ADB, and Gradle wrapper
adx info                 # Inspect project root, module structure, and package ID
adx bundle release       # Build Google Play App Bundle (.aab)
adx test                 # Run unit tests with clickable HTML report
adx lint                 # Run Android Lint analysis with clickable report
```

---

## ☕ Kotlin & Java Compatibility

`adx` is 100% compatible with both **Kotlin** and **Java** Android projects:
- **Kotlin DSL** (`build.gradle.kts`) and **Groovy DSL** (`build.gradle`) are both natively supported.
- **Gradle Version Catalogs** (`gradle/libs.versions.toml`) are detected and updated automatically.
- Multi-module and single-module project architectures are auto-detected without manual configuration.

---

## 🤝 Contributing

Contributions, issues, and feature requests are welcome!

1. Fork the Project (`https://github.com/Shashwat-CODING/adx`)
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'feat: add some amazing feature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 👤 Author

Developed by **Shashwat**
- **GitHub**: [@Shashwat-CODING](https://github.com/Shashwat-CODING)
- **Repository**: [https://github.com/Shashwat-CODING/adx](https://github.com/Shashwat-CODING/adx)

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
