// Package config provides the unified .pk.json schema and loader.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// Default modes applied when a .pk.json key is absent. These are the single
// source of truth for the fallback values — referenced by both the runtime
// (guard/preserve, when a key is omitted) and pk setup (what it writes
// explicitly). The literal lives exactly once, here.
const (
	DefaultGuardMode    = "block"  // guard.mode
	DefaultGuardPush    = "block"  // guard.push
	DefaultPreserveMode = "manual" // preserve.mode
)

// GuardConfig holds the guard section of .pk.json.
type GuardConfig struct {
	Branches []string `json:"branches,omitempty"`
	Mode     string   `json:"mode,omitempty"` // block | ask | off
	Push     string   `json:"push,omitempty"` // block | ask | off
}

// ResolvedMode returns the branch-guard mode, applying DefaultGuardMode when unset.
func (g GuardConfig) ResolvedMode() string {
	if g.Mode == "" {
		return DefaultGuardMode
	}
	return g.Mode
}

// ResolvedPush returns the push-guard mode, applying DefaultGuardPush when unset.
func (g GuardConfig) ResolvedPush() string {
	if g.Push == "" {
		return DefaultGuardPush
	}
	return g.Push
}

// PreserveConfig holds the preserve section of .pk.json.
type PreserveConfig struct {
	Mode string `json:"mode,omitempty"` // auto | manual | off
}

// ResolvedMode returns the preserve mode, applying DefaultPreserveMode when unset.
func (p PreserveConfig) ResolvedMode() string {
	if p.Mode == "" {
		return DefaultPreserveMode
	}
	return p.Mode
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
	Preserve  PreserveConfig  `json:"preserve,omitempty"`
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
