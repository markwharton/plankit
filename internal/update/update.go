// Package update checks for newer versions of pk via the GitHub Releases API.
// Results are cached daily to avoid repeated HTTP calls.
package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// CacheEntry stores the result of a version check.
type CacheEntry struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

const updateTimeout = 2 * time.Second

var updateClient = &http.Client{Timeout: updateTimeout}

// Config holds injectable dependencies for testing.
type Config struct {
	CurrentVersion string
	CacheDir       func() (string, error)
	HTTPGet        func(url string) (*http.Response, error)
	Now            func() time.Time
}

// DefaultConfig returns a Config wired to real implementations.
func DefaultConfig(currentVersion string) Config {
	return Config{
		CurrentVersion: currentVersion,
		CacheDir:       os.UserCacheDir,
		HTTPGet:        updateClient.Get,
		Now:            time.Now,
	}
}

// Check returns the latest version and whether an update is available.
// Uses a daily cache to avoid repeated HTTP calls.
func Check(cfg Config) (latest string, available bool) {
	if cfg.CurrentVersion == "dev" || cfg.CurrentVersion == "" {
		return "", false
	}

	cacheFile := cacheFilePath(cfg.CacheDir)

	if cacheFile != "" {
		if data, err := os.ReadFile(cacheFile); err == nil {
			var entry CacheEntry
			if json.Unmarshal(data, &entry) == nil {
				if cfg.Now().Sub(entry.CheckedAt) < 24*time.Hour {
					return entry.Latest, isNewer(entry.Latest, cfg.CurrentVersion)
				}
			}
		}
	}

	latest = fetchLatest(cfg.HTTPGet)
	if latest == "" {
		return "", false
	}

	if cacheFile != "" {
		os.MkdirAll(filepath.Dir(cacheFile), 0755)
		entry := CacheEntry{CheckedAt: cfg.Now(), Latest: latest}
		if data, err := json.Marshal(entry); err == nil {
			os.WriteFile(cacheFile, data, 0644)
		}
	}

	return latest, isNewer(latest, cfg.CurrentVersion)
}

// FormatNotice returns a human-readable update notice.
func FormatNotice(latest, current string) string {
	return fmt.Sprintf("Update available: pk %s → %s. Run: go install github.com/markwharton/plankit/cmd/pk@latest", current, latest)
}

func cacheFilePath(cacheDir func() (string, error)) string {
	dir, err := cacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "plankit", "version-check.json")
}

const releaseURL = "https://api.github.com/repos/markwharton/plankit/releases/latest"

func fetchLatest(httpGet func(string) (*http.Response, error)) string {
	resp, err := httpGet(releaseURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ""
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if json.Unmarshal(body, &release) != nil {
		return ""
	}

	return release.TagName
}

// isNewer returns true if latest is a newer semver than current.
func isNewer(latest, current string) bool {
	latestParts := parseVersion(latest)
	currentParts := parseVersion(current)
	if latestParts == nil || currentParts == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}
	return false
}

// parseVersion parses "vX.Y.Z" or "X.Y.Z" into [major, minor, patch].
func parseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	result := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		result[i] = n
	}
	return result
}
