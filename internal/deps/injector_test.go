package deps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddKtsDependency(t *testing.T) {
	tmpDir := t.TempDir()
	ktsPath := filepath.Join(tmpDir, "build.gradle.kts")
	initialContent := `
plugins {
    id("com.android.application")
}

dependencies {
    implementation("androidx.core:core-ktx:1.12.0")
}
`
	if err := os.WriteFile(ktsPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	dep := &Dependency{
		Group:    "com.squareup.retrofit2",
		Artifact: "retrofit",
		Version:  "2.11.0",
	}

	msg, err := addKtsDependency(ktsPath, dep, "implementation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(msg, "Injected") {
		t.Errorf("expected Injected in message, got %s", msg)
	}

	updated, err := os.ReadFile(ktsPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(updated), `implementation("com.squareup.retrofit2:retrofit:2.11.0")`) {
		t.Errorf("expected dependency in file, got:\n%s", string(updated))
	}
}

func TestAddCatalogDependency(t *testing.T) {
	tmpDir := t.TempDir()
	gradleDir := filepath.Join(tmpDir, "gradle")
	if err := os.MkdirAll(gradleDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlPath := filepath.Join(gradleDir, "libs.versions.toml")
	tomlContent := `[versions]
coreKtx = "1.12.0"

[libraries]
androidx-core-ktx = { group = "androidx.core", name = "core-ktx", version.ref = "coreKtx" }
`
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	appDir := filepath.Join(tmpDir, "app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	appKts := filepath.Join(appDir, "build.gradle.kts")
	if err := os.WriteFile(appKts, []byte("dependencies {\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	dep := &Dependency{
		Group:    "com.google.code.gson",
		Artifact: "gson",
		Version:  "2.10.1",
	}

	_, err := addCatalogDependency(tomlPath, appDir, dep, "implementation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updatedToml, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(updatedToml), `gson = { module = "com.google.code.gson:gson", version = "2.10.1" }`) {
		t.Errorf("expected gson entry in toml, got:\n%s", string(updatedToml))
	}

	updatedAppKts, err := os.ReadFile(appKts)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(updatedAppKts), `implementation(libs.gson)`) {
		t.Errorf("expected implementation(libs.gson) in app build.gradle.kts, got:\n%s", string(updatedAppKts))
	}
}
