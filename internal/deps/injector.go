package deps

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// UpdateResult holds details of an available or applied dependency update
type UpdateResult struct {
	Coordinate string
	OldVersion string
	NewVersion string
	File       string
}

// AddDependency adds the given dependency into the project build files
func AddDependency(rootDir string, appModuleDir string, dep *Dependency, config string) (string, error) {
	if config == "" {
		config = "implementation"
	}

	tomlPath := filepath.Join(rootDir, "gradle", "libs.versions.toml")
	if _, err := os.Stat(tomlPath); err == nil {
		return addCatalogDependency(tomlPath, appModuleDir, dep, config)
	}

	// Direct build.gradle.kts
	ktsPath := filepath.Join(appModuleDir, "build.gradle.kts")
	if _, err := os.Stat(ktsPath); err == nil {
		return addKtsDependency(ktsPath, dep, config)
	}

	// Direct build.gradle
	groovyPath := filepath.Join(appModuleDir, "build.gradle")
	if _, err := os.Stat(groovyPath); err == nil {
		return addGroovyDependency(groovyPath, dep, config)
	}

	return "", fmt.Errorf("no build.gradle, build.gradle.kts, or libs.versions.toml found in %s", appModuleDir)
}

func sanitizeAlias(artifact string) string {
	clean := strings.ReplaceAll(artifact, "-", ".")
	clean = strings.ReplaceAll(clean, "_", ".")
	return clean
}

func addCatalogDependency(tomlPath string, appModuleDir string, dep *Dependency, config string) (string, error) {
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return "", err
	}
	content := string(data)

	alias := strings.ToLower(dep.Artifact)
	alias = strings.ReplaceAll(alias, "_", "-")

	// Check if already in toml
	coord := fmt.Sprintf("%s:%s", dep.Group, dep.Artifact)
	if strings.Contains(content, coord) {
		return "", fmt.Errorf("dependency %s is already declared in %s", coord, tomlPath)
	}

	// Add entry to [libraries] section
	entry := fmt.Sprintf("%s = { module = \"%s:%s\", version = \"%s\" }\n", alias, dep.Group, dep.Artifact, dep.Version)

	var newContent string
	if strings.Contains(content, "[libraries]") {
		parts := strings.SplitN(content, "[libraries]", 2)
		newContent = parts[0] + "[libraries]\n" + entry + parts[1]
	} else {
		newContent = content + "\n[libraries]\n" + entry
	}

	if err := os.WriteFile(tomlPath, []byte(newContent), 0644); err != nil {
		return "", err
	}

	// Now also add to app/build.gradle.kts or app/build.gradle
	ktsPath := filepath.Join(appModuleDir, "build.gradle.kts")
	if _, err := os.Stat(ktsPath); err == nil {
		ref := strings.ReplaceAll(alias, "-", ".")
		depLine := fmt.Sprintf("    %s(libs.%s)\n", config, ref)
		_ = injectLineIntoDependencies(ktsPath, depLine)
		return fmt.Sprintf("Added to %s as 'libs.%s' and injected into %s", filepath.Base(tomlPath), ref, filepath.Base(ktsPath)), nil
	}

	groovyPath := filepath.Join(appModuleDir, "build.gradle")
	if _, err := os.Stat(groovyPath); err == nil {
		ref := strings.ReplaceAll(alias, "-", ".")
		depLine := fmt.Sprintf("    %s libs.%s\n", config, ref)
		_ = injectLineIntoDependencies(groovyPath, depLine)
		return fmt.Sprintf("Added to %s as 'libs.%s' and injected into %s", filepath.Base(tomlPath), ref, filepath.Base(groovyPath)), nil
	}

	return fmt.Sprintf("Added to %s as '%s'", tomlPath, alias), nil
}

func addKtsDependency(ktsPath string, dep *Dependency, config string) (string, error) {
	depLine := fmt.Sprintf("    %s(\"%s:%s:%s\")\n", config, dep.Group, dep.Artifact, dep.Version)
	if err := injectLineIntoDependencies(ktsPath, depLine); err != nil {
		return "", err
	}
	return fmt.Sprintf("Injected %s into %s", dep.Coordinate(), filepath.Base(ktsPath)), nil
}

func addGroovyDependency(groovyPath string, dep *Dependency, config string) (string, error) {
	depLine := fmt.Sprintf("    %s '%s:%s:%s'\n", config, dep.Group, dep.Artifact, dep.Version)
	if err := injectLineIntoDependencies(groovyPath, depLine); err != nil {
		return "", err
	}
	return fmt.Sprintf("Injected %s into %s", dep.Coordinate(), filepath.Base(groovyPath)), nil
}

func injectLineIntoDependencies(filePath string, lineToInject string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(data)

	if strings.Contains(content, strings.TrimSpace(lineToInject)) {
		return nil // already exists
	}

	idx := strings.Index(content, "dependencies {")
	if idx != -1 {
		insertPos := idx + len("dependencies {") + 1
		newContent := content[:insertPos] + "\n" + lineToInject + content[insertPos:]
		return os.WriteFile(filePath, []byte(newContent), 0644)
	}

	// Fallback append dependencies block
	newContent := content + "\n\ndependencies {\n" + lineToInject + "}\n"
	return os.WriteFile(filePath, []byte(newContent), 0644)
}

// UpdateDependencies checks and optionally updates dependencies in the project
func UpdateDependencies(rootDir string, appModuleDir string, checkOnly bool) ([]UpdateResult, error) {
	var results []UpdateResult

	// 1. Check version catalog
	tomlPath := filepath.Join(rootDir, "gradle", "libs.versions.toml")
	if _, err := os.Stat(tomlPath); err == nil {
		tomlResults, err := updateVersionCatalog(tomlPath, checkOnly)
		if err == nil {
			results = append(results, tomlResults...)
		}
	}

	// 2. Check build.gradle.kts
	ktsPath := filepath.Join(appModuleDir, "build.gradle.kts")
	if _, err := os.Stat(ktsPath); err == nil {
		ktsResults, err := updateBuildFile(ktsPath, checkOnly)
		if err == nil {
			results = append(results, ktsResults...)
		}
	}

	return results, nil
}

func updateVersionCatalog(tomlPath string, checkOnly bool) ([]UpdateResult, error) {
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return nil, err
	}
	content := string(data)
	var results []UpdateResult

	// Match: module = "group:artifact", version = "version"
	reModule := regexp.MustCompile(`module\s*=\s*["']([^:"']+):([^"']+)["'],?\s*version\s*=\s*["']([^"']+)["']`)
	matches := reModule.FindAllStringSubmatch(content, -1)

	updatedContent := content
	for _, m := range matches {
		group := m[1]
		artifact := m[2]
		currentVer := m[3]

		latest, err := QueryLatestVersion(group, artifact)
		if err == nil && latest != "" && latest != currentVer {
			res := UpdateResult{
				Coordinate: fmt.Sprintf("%s:%s", group, artifact),
				OldVersion: currentVer,
				NewVersion: latest,
				File:       filepath.Base(tomlPath),
			}
			results = append(results, res)

			if !checkOnly {
				oldPattern := fmt.Sprintf(`"%s:%s", version = "%s"`, group, artifact, currentVer)
				newPattern := fmt.Sprintf(`"%s:%s", version = "%s"`, group, artifact, latest)
				updatedContent = strings.Replace(updatedContent, oldPattern, newPattern, 1)
			}
		}
	}

	if !checkOnly && len(results) > 0 {
		_ = os.WriteFile(tomlPath, []byte(updatedContent), 0644)
	}

	return results, nil
}

func updateBuildFile(filePath string, checkOnly bool) ([]UpdateResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	content := string(data)
	var results []UpdateResult

	reDep := regexp.MustCompile(`["']([^:"'\s]+):([^:"'\s]+):([^"'\s]+)["']`)
	matches := reDep.FindAllStringSubmatch(content, -1)

	updatedContent := content
	for _, m := range matches {
		group := m[1]
		artifact := m[2]
		currentVer := m[3]

		latest, err := QueryLatestVersion(group, artifact)
		if err == nil && latest != "" && latest != currentVer {
			res := UpdateResult{
				Coordinate: fmt.Sprintf("%s:%s", group, artifact),
				OldVersion: currentVer,
				NewVersion: latest,
				File:       filepath.Base(filePath),
			}
			results = append(results, res)

			if !checkOnly {
				oldStr := fmt.Sprintf(`"%s:%s:%s"`, group, artifact, currentVer)
				newStr := fmt.Sprintf(`"%s:%s:%s"`, group, artifact, latest)
				updatedContent = strings.Replace(updatedContent, oldStr, newStr, 1)
			}
		}
	}

	if !checkOnly && len(results) > 0 {
		_ = os.WriteFile(filePath, []byte(updatedContent), 0644)
	}

	return results, nil
}
