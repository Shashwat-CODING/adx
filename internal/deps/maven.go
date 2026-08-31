package deps

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Dependency holds Maven coordinates
type Dependency struct {
	Group    string
	Artifact string
	Version  string
}

// Coordinate returns group:artifact:version
func (d *Dependency) Coordinate() string {
	if d.Version != "" {
		return fmt.Sprintf("%s:%s:%s", d.Group, d.Artifact, d.Version)
	}
	return fmt.Sprintf("%s:%s", d.Group, d.Artifact)
}

// CommonAliases provides quick coordinate mappings for popular Android libraries
var CommonAliases = map[string]string{
	"retrofit":           "com.squareup.retrofit2:retrofit",
	"retrofit-gson":      "com.squareup.retrofit2:converter-gson",
	"retrofit-moshi":     "com.squareup.retrofit2:converter-moshi",
	"okhttp":             "com.squareup.okhttp3:okhttp",
	"okhttp-logging":     "com.squareup.okhttp3:logging-interceptor",
	"gson":               "com.google.code.gson:gson",
	"moshi":              "com.squareup.moshi:moshi",
	"coil":               "io.coil-kt:coil",
	"coil-compose":       "io.coil-kt.coil3:coil-compose",
	"glide":              "com.github.bumptech.glide:glide",
	"room":               "androidx.room:room-runtime",
	"room-ktx":           "androidx.room:room-ktx",
	"room-compiler":      "androidx.room:room-compiler",
	"hilt":               "com.google.dagger:hilt-android",
	"hilt-compiler":      "com.google.dagger:hilt-android-compiler",
	"koin":               "io.insert-koin:koin-android",
	"coroutines":         "org.jetbrains.kotlinx:kotlinx-coroutines-android",
	"timber":             "com.jakewharton.timber:timber",
	"datastore":          "androidx.datastore:datastore-preferences",
	"lifecycle-viewmodel":"androidx.lifecycle:lifecycle-viewmodel-ktx",
	"activity-compose":   "androidx.activity:activity-compose",
	"navigation-compose": "androidx.navigation:navigation-compose",
	"lottie":             "com.airbnb.android:lottie",
	"lottie-compose":     "com.airbnb.android:lottie-compose",
}

type mavenResponse struct {
	Response struct {
		Docs []struct {
			ID            string `json:"id"`
			G             string `json:"g"`
			A             string `json:"a"`
			LatestVersion string `json:"latestVersion"`
		} `json:"docs"`
	} `json:"response"`
}

// ResolveDependency resolves a user input query into complete group:artifact:version coordinates
func ResolveDependency(query string) (*Dependency, error) {
	trimmed := strings.TrimSpace(query)

	// 1. Direct 3-part coordinate: group:artifact:version
	parts := strings.Split(trimmed, ":")
	if len(parts) == 3 {
		return &Dependency{
			Group:    parts[0],
			Artifact: parts[1],
			Version:  parts[2],
		}, nil
	}

	// 2. Check alias
	lower := strings.ToLower(trimmed)
	if mapped, ok := CommonAliases[lower]; ok {
		trimmed = mapped
		parts = strings.Split(trimmed, ":")
	}

	// 3. 2-part coordinate: group:artifact -> resolve latest version
	if len(parts) == 2 {
		group := parts[0]
		artifact := parts[1]
		version, err := QueryLatestVersion(group, artifact)
		if err != nil {
			return nil, fmt.Errorf("could not find latest version for %s:%s on Maven Central: %w", group, artifact, err)
		}
		return &Dependency{
			Group:    group,
			Artifact: artifact,
			Version:  version,
		}, nil
	}

	// 4. Free text search on Maven Central
	dep, err := SearchMavenCentral(trimmed)
	if err != nil {
		return nil, fmt.Errorf("library '%s' not found on Maven Central: %w", query, err)
	}
	return dep, nil
}

// QueryLatestVersion gets the latest version of an exact group and artifact from Maven Central
func QueryLatestVersion(group, artifact string) (string, error) {
	apiURL := fmt.Sprintf("https://search.maven.org/solrsearch/select?q=g:%%22%s%%22+AND+a:%%22%s%%22&rows=1&wt=json",
		url.QueryEscape(group), url.QueryEscape(artifact))

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("maven API returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var mResp mavenResponse
	if err := json.Unmarshal(body, &mResp); err != nil {
		return "", err
	}

	if len(mResp.Response.Docs) == 0 {
		return "", fmt.Errorf("no artifacts found for %s:%s", group, artifact)
	}

	return mResp.Response.Docs[0].LatestVersion, nil
}

// SearchMavenCentral performs a general search on Maven Central
func SearchMavenCentral(query string) (*Dependency, error) {
	apiURL := fmt.Sprintf("https://search.maven.org/solrsearch/select?q=%s&rows=1&wt=json", url.QueryEscape(query))

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("maven API returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var mResp mavenResponse
	if err := json.Unmarshal(body, &mResp); err != nil {
		return nil, err
	}

	if len(mResp.Response.Docs) == 0 {
		return nil, fmt.Errorf("no search results for '%s'", query)
	}

	doc := mResp.Response.Docs[0]
	return &Dependency{
		Group:    doc.G,
		Artifact: doc.A,
		Version:  doc.LatestVersion,
	}, nil
}
