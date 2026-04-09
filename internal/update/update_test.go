package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"v1.1.0", "v1.0.0", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.0.1", "v1.0.0", true},
		{"v1.10.0", "v1.9.0", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.1.0", false},
		{"v0.9.0", "v1.0.0", false},
		{"invalid", "v1.0.0", false},
		{"v1.0.0", "invalid", false},
		{"", "v1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.latest, tt.current), func(t *testing.T) {
			got := isNewer(tt.latest, tt.current)
			if got != tt.want {
				t.Errorf("isNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

func TestCheck_devVersion(t *testing.T) {
	cfg := Config{CurrentVersion: "dev"}
	latest, available := Check(cfg)
	if available || latest != "" {
		t.Errorf("Check(dev) = (%q, %v), want (\"\", false)", latest, available)
	}
}

func TestCheck_cachedResult(t *testing.T) {
	cacheDir := t.TempDir()
	cacheFile := filepath.Join(cacheDir, "plankit", "version-check.json")
	os.MkdirAll(filepath.Dir(cacheFile), 0755)

	entry := CacheEntry{
		CheckedAt: time.Now().Add(-1 * time.Hour),
		Latest:    "v2.0.0",
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(cacheFile, data, 0644)

	httpCalled := false
	cfg := Config{
		CurrentVersion: "v1.0.0",
		CacheDir:       func() (string, error) { return cacheDir, nil },
		HTTPGet: func(url string) (*http.Response, error) {
			httpCalled = true
			return nil, fmt.Errorf("should not be called")
		},
		Now: time.Now,
	}

	latest, available := Check(cfg)
	if httpCalled {
		t.Error("HTTP was called despite valid cache")
	}
	if latest != "v2.0.0" || !available {
		t.Errorf("Check() = (%q, %v), want (\"v2.0.0\", true)", latest, available)
	}
}

func TestCheck_expiredCache(t *testing.T) {
	cacheDir := t.TempDir()
	cacheFile := filepath.Join(cacheDir, "plankit", "version-check.json")
	os.MkdirAll(filepath.Dir(cacheFile), 0755)

	entry := CacheEntry{
		CheckedAt: time.Now().Add(-25 * time.Hour),
		Latest:    "v1.5.0",
	}
	data, _ := json.Marshal(entry)
	os.WriteFile(cacheFile, data, 0644)

	cfg := Config{
		CurrentVersion: "v1.0.0",
		CacheDir:       func() (string, error) { return cacheDir, nil },
		HTTPGet: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"tag_name": "v2.0.0"}`)),
			}, nil
		},
		Now: time.Now,
	}

	latest, available := Check(cfg)
	if latest != "v2.0.0" || !available {
		t.Errorf("Check() = (%q, %v), want (\"v2.0.0\", true)", latest, available)
	}

	newData, _ := os.ReadFile(cacheFile)
	var newEntry CacheEntry
	json.Unmarshal(newData, &newEntry)
	if newEntry.Latest != "v2.0.0" {
		t.Errorf("cache latest = %q, want \"v2.0.0\"", newEntry.Latest)
	}
}

func TestCheck_noUpdate(t *testing.T) {
	cfg := Config{
		CurrentVersion: "v1.0.0",
		CacheDir:       func() (string, error) { return t.TempDir(), nil },
		HTTPGet: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"tag_name": "v1.0.0"}`)),
			}, nil
		},
		Now: time.Now,
	}

	latest, available := Check(cfg)
	if latest != "v1.0.0" || available {
		t.Errorf("Check() = (%q, %v), want (\"v1.0.0\", false)", latest, available)
	}
}

func TestCheck_httpError(t *testing.T) {
	cfg := Config{
		CurrentVersion: "v1.0.0",
		CacheDir:       func() (string, error) { return t.TempDir(), nil },
		HTTPGet: func(url string) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
		Now: time.Now,
	}

	latest, available := Check(cfg)
	if latest != "" || available {
		t.Errorf("Check() = (%q, %v), want (\"\", false)", latest, available)
	}
}

func TestCheck_cacheBoundary(t *testing.T) {
	cacheDir := t.TempDir()
	cacheFile := filepath.Join(cacheDir, "plankit", "version-check.json")
	os.MkdirAll(filepath.Dir(cacheFile), 0755)

	fixedNow := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)

	t.Run("just under 24h uses cache", func(t *testing.T) {
		entry := CacheEntry{
			CheckedAt: fixedNow.Add(-24*time.Hour + time.Minute),
			Latest:    "v2.0.0",
		}
		data, _ := json.Marshal(entry)
		os.WriteFile(cacheFile, data, 0644)

		httpCalled := false
		cfg := Config{
			CurrentVersion: "v1.0.0",
			CacheDir:       func() (string, error) { return cacheDir, nil },
			HTTPGet: func(url string) (*http.Response, error) {
				httpCalled = true
				return nil, fmt.Errorf("should not be called")
			},
			Now: func() time.Time { return fixedNow },
		}

		latest, available := Check(cfg)
		if httpCalled {
			t.Error("HTTP was called despite cache being under 24h old")
		}
		if latest != "v2.0.0" || !available {
			t.Errorf("Check() = (%q, %v), want (\"v2.0.0\", true)", latest, available)
		}
	})

	t.Run("exactly 24h refetches", func(t *testing.T) {
		entry := CacheEntry{
			CheckedAt: fixedNow.Add(-24 * time.Hour),
			Latest:    "v1.5.0",
		}
		data, _ := json.Marshal(entry)
		os.WriteFile(cacheFile, data, 0644)

		httpCalled := false
		cfg := Config{
			CurrentVersion: "v1.0.0",
			CacheDir:       func() (string, error) { return cacheDir, nil },
			HTTPGet: func(url string) (*http.Response, error) {
				httpCalled = true
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"tag_name": "v3.0.0"}`)),
				}, nil
			},
			Now: func() time.Time { return fixedNow },
		}

		latest, _ := Check(cfg)
		if !httpCalled {
			t.Error("HTTP was not called despite cache being exactly 24h old")
		}
		if latest != "v3.0.0" {
			t.Errorf("latest = %q, want \"v3.0.0\"", latest)
		}
	})
}

func TestCheck_httpNon200(t *testing.T) {
	cfg := Config{
		CurrentVersion: "v1.0.0",
		CacheDir:       func() (string, error) { return t.TempDir(), nil },
		HTTPGet: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: 403,
				Body:       io.NopCloser(strings.NewReader(`{"message": "rate limited"}`)),
			}, nil
		},
		Now: time.Now,
	}

	latest, available := Check(cfg)
	if latest != "" || available {
		t.Errorf("Check() = (%q, %v), want (\"\", false) on 403", latest, available)
	}
}

func TestFormatNotice(t *testing.T) {
	notice := FormatNotice("v2.0.0", "v1.0.0")
	if !strings.Contains(notice, "v1.0.0") || !strings.Contains(notice, "v2.0.0") {
		t.Errorf("FormatNotice() = %q, want to contain both versions", notice)
	}
	if !strings.Contains(notice, "go install") {
		t.Errorf("FormatNotice() = %q, want to contain install command", notice)
	}
}
