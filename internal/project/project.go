package project

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Project holds metadata about the detected Android project
type Project struct {
	RootDir         string
	GradlewPath     string
	AppModuleDir    string
	AppModuleName   string
	PackageName     string
	LauncherActivity string
}

// FindProject searches up the directory tree to locate an Android project root
func FindProject(startDir string) (*Project, error) {
	curr, err := filepath.Abs(startDir)
	if err != nil {
		curr = startDir
	}

	var rootDir string
	var fallbackRoot string

	dir := curr
	for {
		// 1. settings.gradle / settings.gradle.kts is the definitive Gradle root
		hasSettings := false
		for _, s := range []string{"settings.gradle", "settings.gradle.kts"} {
			if _, err := os.Stat(filepath.Join(dir, s)); err == nil {
				hasSettings = true
				break
			}
		}

		if hasSettings {
			rootDir = dir
			break
		}

		// 2. gradlew wrapper is also placed at the root
		for _, g := range []string{"gradlew", "gradlew.bat"} {
			if _, err := os.Stat(filepath.Join(dir, g)); err == nil {
				if rootDir == "" {
					rootDir = dir
				}
				break
			}
		}

		// 3. Fallback: single module projects with only build.gradle
		if fallbackRoot == "" {
			for _, b := range []string{"build.gradle", "build.gradle.kts"} {
				if _, err := os.Stat(filepath.Join(dir, b)); err == nil {
					fallbackRoot = dir
					break
				}
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if rootDir == "" {
		rootDir = fallbackRoot
	}

	if rootDir == "" {
		return nil, fmt.Errorf("no Android project found (no settings.gradle or build.gradle detected in %s or its parents)", startDir)
	}

	p := &Project{
		RootDir: rootDir,
	}

	// 1. Locate Gradlew
	gradlewName := "gradlew"
	if runtime.GOOS == "windows" {
		gradlewName = "gradlew.bat"
	}
	gradlew := filepath.Join(rootDir, gradlewName)
	if fi, err := os.Stat(gradlew); err == nil {
		p.GradlewPath = gradlew
		// Ensure it's executable on Unix
		if runtime.GOOS != "windows" && (fi.Mode()&0111 == 0) {
			_ = os.Chmod(gradlew, fi.Mode()|0111)
		}
	} else {
		// Fallback to system gradle
		p.GradlewPath = "gradle"
	}

	// 2. Identify App Module directory
	appModules := []string{"app", "mobile", "."}
	for _, mod := range appModules {
		candDir := filepath.Join(rootDir, mod)
		if _, err := os.Stat(filepath.Join(candDir, "build.gradle")); err == nil {
			p.AppModuleDir = candDir
			p.AppModuleName = mod
			break
		}
		if _, err := os.Stat(filepath.Join(candDir, "build.gradle.kts")); err == nil {
			p.AppModuleDir = candDir
			p.AppModuleName = mod
			break
		}
	}

	if p.AppModuleDir == "" {
		p.AppModuleDir = filepath.Join(rootDir, "app")
		p.AppModuleName = "app"
	}

	// 3. Extract Package Name / Application ID
	p.PackageName = p.detectPackageName()

	// 4. Extract Launcher Activity
	p.LauncherActivity = p.detectLauncherActivity()

	return p, nil
}

func (p *Project) detectPackageName() string {
	// 1. Check build.gradle.kts or build.gradle in app module
	buildFiles := []string{
		filepath.Join(p.AppModuleDir, "build.gradle.kts"),
		filepath.Join(p.AppModuleDir, "build.gradle"),
		filepath.Join(p.RootDir, "build.gradle.kts"),
		filepath.Join(p.RootDir, "build.gradle"),
	}

	reAppID := regexp.MustCompile(`(?i)applicationId\s*=?\s*["']([^"']+)["']`)
	reNamespace := regexp.MustCompile(`(?i)namespace\s*=?\s*["']([^"']+)["']`)

	for _, bf := range buildFiles {
		data, err := os.ReadFile(bf)
		if err != nil {
			continue
		}
		content := string(data)

		// First check applicationId
		if m := reAppID.FindStringSubmatch(content); len(m) > 1 {
			return m[1]
		}
		// Then namespace
		if m := reNamespace.FindStringSubmatch(content); len(m) > 1 {
			return m[1]
		}
	}

	// 2. Check AndroidManifest.xml
	manifestPaths := []string{
		filepath.Join(p.AppModuleDir, "src", "main", "AndroidManifest.xml"),
		filepath.Join(p.RootDir, "src", "main", "AndroidManifest.xml"),
	}

	rePkg := regexp.MustCompile(`package\s*=\s*["']([^"']+)["']`)
	for _, mp := range manifestPaths {
		data, err := os.ReadFile(mp)
		if err != nil {
			continue
		}
		if m := rePkg.FindStringSubmatch(string(data)); len(m) > 1 {
			return m[1]
		}
	}

	return ""
}

type manifestXML struct {
	Package     string `xml:"package,attr"`
	Application struct {
		Activities []struct {
			Name         string `xml:"http://schemas.android.com/apk/res/android name,attr"`
			RawName      string `xml:"name,attr"`
			IntentFilter []struct {
				Action []struct {
					Name    string `xml:"http://schemas.android.com/apk/res/android name,attr"`
					RawName string `xml:"name,attr"`
				} `xml:"action"`
				Category []struct {
					Name    string `xml:"http://schemas.android.com/apk/res/android name,attr"`
					RawName string `xml:"name,attr"`
				} `xml:"category"`
			} `xml:"intent-filter"`
		} `xml:"activity"`
	} `xml:"application"`
}

func (p *Project) detectLauncherActivity() string {
	manifestPaths := []string{
		filepath.Join(p.AppModuleDir, "src", "main", "AndroidManifest.xml"),
		filepath.Join(p.RootDir, "src", "main", "AndroidManifest.xml"),
	}

	for _, mp := range manifestPaths {
		data, err := os.ReadFile(mp)
		if err != nil {
			continue
		}

		var m manifestXML
		if err := xml.Unmarshal(data, &m); err != nil {
			// Fallback regex detection
			reLauncher := regexp.MustCompile(`(?s)<activity[^>]+android:name=["']([^"']+)["'][^>]*>.*?android\.intent\.action\.MAIN.*?android\.intent\.category\.LAUNCHER.*?</activity>`)
			if matches := reLauncher.FindStringSubmatch(string(data)); len(matches) > 1 {
				act := matches[1]
				return p.formatActivity(act)
			}
			continue
		}

		for _, act := range m.Application.Activities {
			actName := act.Name
			if actName == "" {
				actName = act.RawName
			}

			for _, filter := range act.IntentFilter {
				isMain := false
				isLauncher := false

				for _, a := range filter.Action {
					name := a.Name
					if name == "" {
						name = a.RawName
					}
					if strings.Contains(name, "android.intent.action.MAIN") {
						isMain = true
					}
				}

				for _, c := range filter.Category {
					name := c.Name
					if name == "" {
						name = c.RawName
					}
					if strings.Contains(name, "android.intent.category.LAUNCHER") {
						isLauncher = true
					}
				}

				if isMain && isLauncher {
					return p.formatActivity(actName)
				}
			}
		}
	}

	return ""
}

func (p *Project) formatActivity(actName string) string {
	if strings.HasPrefix(actName, ".") && p.PackageName != "" {
		return p.PackageName + actName
	}
	if !strings.Contains(actName, ".") && p.PackageName != "" {
		return p.PackageName + "." + actName
	}
	return actName
}

// FindApk finds the generated APK file for a given variant (e.g. "debug" or "release")
func (p *Project) FindApk(variant string) (string, error) {
	cleanVariant := strings.ToLower(variant)

	// Common output locations
	candidateDirs := []string{
		filepath.Join(p.AppModuleDir, "build", "outputs", "apk", cleanVariant),
		filepath.Join(p.AppModuleDir, "build", "outputs", "apk"),
		filepath.Join(p.RootDir, "build", "outputs", "apk", cleanVariant),
		filepath.Join(p.RootDir, "build", "outputs", "apk"),
	}

	var foundApks []string
	for _, dir := range candidateDirs {
		if _, err := os.Stat(dir); err != nil {
			continue
		}

		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".apk") {
				lower := strings.ToLower(info.Name())
				if strings.Contains(lower, cleanVariant) {
					foundApks = append(foundApks, path)
				}
			}
			return nil
		})
	}

	if len(foundApks) > 0 {
		// Prefer the newest or most specific apk
		var best string
		var bestMod int64
		for _, a := range foundApks {
			if fi, err := os.Stat(a); err == nil {
				if fi.ModTime().Unix() > bestMod {
					bestMod = fi.ModTime().Unix()
					best = a
				}
			}
		}
		return best, nil
	}

	return "", fmt.Errorf("no %s APK found in build outputs. Please run 'adx build %s' first", variant, variant)
}
