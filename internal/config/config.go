// Package config provides the unified .pk.json schema and loader.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// GuardConfig holds the guard section of .pk.json.
type GuardConfig struct {
	Branches []string `json:"branches,omitempty"`
}

// TypeConfig maps a conventional commit type to a changelog section.
type TypeConfig struct {
	Type    string `json:"type"`
	Section string `json:"section,omitempty"`
	Hidden  bool   `json:"hidden,omitempty"`
}

// VersionFile describes a file containing a version string to update.
type VersionFile struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// ChangelogHooks holds lifecycle hook commands for the changelog process.
type ChangelogHooks struct {
	PostVersion string `json:"postVersion,omitempty"`
	PreCommit   string `json:"preCommit,omitempty"`
}

// ChangelogConfig holds configuration for pk changelog.
type ChangelogConfig struct {
	Types        []TypeConfig   `json:"types,omitempty"`
	VersionFiles []VersionFile  `json:"versionFiles,omitempty"`
	ShowScope    bool           `json:"showScope,omitempty"`
	Hooks        ChangelogHooks `json:"hooks,omitempty"`
}

// ReleaseHooks holds lifecycle hook commands for the release process.
type ReleaseHooks struct {
	PreRelease string `json:"preRelease,omitempty"`
}

// ReleaseSection holds the release config from .pk.json.
type ReleaseSection struct {
	Branch string       `json:"branch,omitempty"`
	Hooks  ReleaseHooks `json:"hooks,omitempty"`
}

// PkConfig is the unified .pk.json schema. Each top-level key maps to a
// pk subcommand.
type PkConfig struct {
	Changelog ChangelogConfig `json:"changelog,omitempty"`
	Guard     GuardConfig     `json:"guard,omitempty"`
	Release   ReleaseSection  `json:"release,omitempty"`
}

// Load reads and parses .pk.json from the given path. Returns a zero-value
// PkConfig (not an error) when the file does not exist. Returns an error
// only when the file exists but contains malformed JSON.
func Load(readFile func(string) ([]byte, error), path string) (PkConfig, error) {
	data, err := readFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PkConfig{}, nil
		}
		return PkConfig{}, fmt.Errorf("failed to read .pk.json: %w", err)
	}
	var cfg PkConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return PkConfig{}, fmt.Errorf("failed to parse .pk.json: %w", err)
	}
	return cfg, nil
}
