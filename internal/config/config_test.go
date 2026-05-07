package config

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestLoad_missingFile(t *testing.T) {
	cfg, err := Load(func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	}, ".pk.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Guard.Branches) != 0 {
		t.Errorf("guard.branches = %v, want empty", cfg.Guard.Branches)
	}
	if len(cfg.Changelog.Types) != 0 {
		t.Errorf("changelog.types = %v, want empty", cfg.Changelog.Types)
	}
	if cfg.Release.Branch != "" {
		t.Errorf("release.branch = %q, want empty", cfg.Release.Branch)
	}
}

func TestLoad_validJSON(t *testing.T) {
	data := []byte(`{
		"guard": {"branches": ["main", "production"]},
		"changelog": {"showScope": true},
		"release": {"branch": "main", "hooks": {"preRelease": "echo hi"}}
	}`)
	cfg, err := Load(func(string) ([]byte, error) {
		return data, nil
	}, ".pk.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Guard.Branches) != 2 || cfg.Guard.Branches[0] != "main" {
		t.Errorf("guard.branches = %v, want [main production]", cfg.Guard.Branches)
	}
	if !cfg.Changelog.ShowScope {
		t.Error("changelog.showScope = false, want true")
	}
	if cfg.Release.Branch != "main" {
		t.Errorf("release.branch = %q, want main", cfg.Release.Branch)
	}
	if cfg.Release.Hooks.PreRelease != "echo hi" {
		t.Errorf("release.hooks.preRelease = %q, want 'echo hi'", cfg.Release.Hooks.PreRelease)
	}
}

func TestLoad_partialJSON(t *testing.T) {
	data := []byte(`{"guard": {"branches": ["main"]}}`)
	cfg, err := Load(func(string) ([]byte, error) {
		return data, nil
	}, ".pk.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Guard.Branches) != 1 {
		t.Errorf("guard.branches = %v, want [main]", cfg.Guard.Branches)
	}
	if cfg.Release.Branch != "" {
		t.Errorf("release.branch = %q, want empty", cfg.Release.Branch)
	}
}

func TestLoad_readError(t *testing.T) {
	_, err := Load(func(string) ([]byte, error) {
		return nil, errors.New("permission denied")
	}, ".pk.json")
	if err == nil {
		t.Fatal("expected error for non-ErrNotExist read failure")
	}
	if !strings.Contains(err.Error(), "failed to read .pk.json") {
		t.Errorf("error = %q, want wrapped message", err)
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %q, want original cause", err)
	}
}

func TestLoad_malformedJSON(t *testing.T) {
	_, err := Load(func(string) ([]byte, error) {
		return []byte(`{invalid`), nil
	}, ".pk.json")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "failed to parse .pk.json") {
		t.Errorf("error = %q, want 'failed to parse .pk.json' prefix", err)
	}
}
