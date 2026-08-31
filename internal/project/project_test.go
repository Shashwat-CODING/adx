package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectAndPackage(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup mock android project structure
	settingsFile := filepath.Join(tmpDir, "settings.gradle.kts")
	if err := os.WriteFile(settingsFile, []byte(`rootProject.name = "TestApp"`), 0644); err != nil {
		t.Fatal(err)
	}

	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	buildFile := filepath.Join(appDir, "build.gradle.kts")
	gradleContent := `
plugins {
    alias(libs.plugins.android.application)
}

android {
    namespace = "com.example.testapp"
    defaultConfig {
        applicationId = "com.example.testapp"
        versionCode = 1
        versionName = "1.0"
    }
}
`
	if err := os.WriteFile(buildFile, []byte(gradleContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create subfolder deep inside
	subDir := filepath.Join(appDir, "src", "main", "java", "com", "example")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	manifestDir := filepath.Join(appDir, "src", "main")
	manifestContent := `<?xml version="1.0" encoding="utf-8"?>
<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <application>
        <activity
            android:name=".MainActivity"
            android:exported="true">
            <intent-filter>
                <action android:name="android.intent.action.MAIN" />
                <category android:name="android.intent.category.LAUNCHER" />
            </intent-filter>
        </activity>
    </application>
</manifest>`
	if err := os.WriteFile(filepath.Join(manifestDir, "AndroidManifest.xml"), []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test discovery from subfolder
	p, err := FindProject(subDir)
	if err != nil {
		t.Fatalf("expected to find project, got error: %v", err)
	}

	if p.RootDir != tmpDir {
		t.Errorf("expected rootDir %s, got %s", tmpDir, p.RootDir)
	}

	if p.PackageName != "com.example.testapp" {
		t.Errorf("expected packageName 'com.example.testapp', got '%s'", p.PackageName)
	}

	if p.LauncherActivity != "com.example.testapp.MainActivity" {
		t.Errorf("expected launcherActivity 'com.example.testapp.MainActivity', got '%s'", p.LauncherActivity)
	}
}

func TestFindApk(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "app")
	apkDir := filepath.Join(appDir, "build", "outputs", "apk", "debug")
	if err := os.MkdirAll(apkDir, 0755); err != nil {
		t.Fatal(err)
	}

	dummyApk := filepath.Join(apkDir, "app-debug.apk")
	if err := os.WriteFile(dummyApk, []byte("fake-apk-content"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Project{
		RootDir:      tmpDir,
		AppModuleDir: appDir,
	}

	apk, err := p.FindApk("debug")
	if err != nil {
		t.Fatalf("expected to find debug apk: %v", err)
	}

	if apk != dummyApk {
		t.Errorf("expected apk path %s, got %s", dummyApk, apk)
	}
}
