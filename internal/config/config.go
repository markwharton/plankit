// Package config is the .pk.json schema: plankit's committed repo policy.
//
// The schema is adopted from v1 unchanged, so existing repositories parse
// as they are. Two behaviors are contractual: an absent file means
// plankit is off (ErrNotConfigured, and every hook exits immediately),
// and a present file is policy, so unknown keys and invalid modes fail
// loudly instead of being ignored.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the policy file, at the repository root.
const FileName = ".pk.json"

// ErrNotConfigured reports that no .pk.json exists: plankit is off here.
var ErrNotConfigured = errors.New("not configured (no " + FileName + ")")

// Default modes applied when a key is absent. The literal lives exactly
// once, here; init writes them explicitly so the policy file reads whole.
// PlanType is the conventional-commit type pk preserve writes when it
// commits a plan. It is pk's protocol vocabulary, like the Release-Tag
// trailer, not user policy: the .pk.json entry for it controls only
// its changelog presentation (hidden by default).
const PlanType = "plan"

const (
	DefaultGuardMode     = "block"  // guard.mode
	DefaultGuardPush     = "block"  // guard.push
	DefaultGuardBreaking = "ask"    // guard.breaking
	DefaultPreserveMode  = "manual" // preserve.mode
)

// GuardConfig holds the guard section.
type GuardConfig struct {
	Branches []string `json:"branches,omitempty"`
	Mode     string   `json:"mode,omitempty"` // block | ask | off
	Push     string   `json:"push,omitempty"` // block | ask | off
	// Breaking governs commits whose message carries a breaking-change
	// marker (! or a BREAKING CHANGE footer). Markers are user-approved
	// claims, not agent judgment, so guard asks before one is written.
	Breaking string `json:"breaking,omitempty"` // ask | off
}

// ResolvedMode returns guard.mode with the default applied.
func (g GuardConfig) ResolvedMode() string {
	if g.Mode == "" {
		return DefaultGuardMode
	}
	return g.Mode
}

// ResolvedPush returns guard.push with the default applied.
func (g GuardConfig) ResolvedPush() string {
	if g.Push == "" {
		return DefaultGuardPush
	}
	return g.Push
}

// ResolvedBreaking returns guard.breaking with the default applied.
func (g GuardConfig) ResolvedBreaking() string {
	if g.Breaking == "" {
		return DefaultGuardBreaking
	}
	return g.Breaking
}

// PreserveConfig holds the preserve section.
type PreserveConfig struct {
	Mode string `json:"mode,omitempty"` // auto | manual | off
}

// ResolvedMode returns preserve.mode with the default applied.
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

// VersionFile describes a file carrying a version string to update.
type VersionFile struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// ChangelogHooks holds lifecycle hooks for the changelog process.
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

// ResolvedTypes returns changelog.types, or the default table when the
// file leaves it empty. Every reader of the table resolves it here so
// "empty means defaults" is a property of the config, not a habit of
// its readers.
func (c ChangelogConfig) ResolvedTypes() []TypeConfig {
	if len(c.Types) == 0 {
		return Default("").Changelog.Types
	}
	return c.Types
}

// ReleaseHooks holds lifecycle hooks for the release process. preRelease
// runs before the tag is created; prePush runs after tagging, before the
// push, when the tag ref exists.
type ReleaseHooks struct {
	PreRelease string `json:"preRelease,omitempty"`
	PrePush    string `json:"prePush,omitempty"`
}

// ReleaseSection holds the release section.
type ReleaseSection struct {
	Branch string       `json:"branch,omitempty"`
	Hooks  ReleaseHooks `json:"hooks,omitempty"`
}

// PkConfig is the .pk.json schema. Each top-level key maps to a pk
// command.
type PkConfig struct {
	Changelog ChangelogConfig `json:"changelog,omitempty"`
	Guard     GuardConfig     `json:"guard,omitempty"`
	Preserve  PreserveConfig  `json:"preserve,omitempty"`
	Release   ReleaseSection  `json:"release,omitempty"`
}

// Path returns the policy file path for a repository root.
func Path(root string) string { return filepath.Join(root, FileName) }

// Load reads root/.pk.json. An absent file returns ErrNotConfigured; a
// present file is decoded strictly and validated.
func Load(root string) (*PkConfig, error) {
	data, err := os.ReadFile(Path(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotConfigured
		}
		return nil, fmt.Errorf("reading %s: %w", FileName, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var cfg PkConfig
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", FileName, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", FileName, err)
	}
	return &cfg, nil
}

// Validate checks mode enums and the changelog type table. Policy typos
// should fail here, at load, not silently change behavior at hook time.
func (c *PkConfig) Validate() error {
	if err := oneOf("guard.mode", c.Guard.Mode, "block", "ask", "off"); err != nil {
		return err
	}
	if err := oneOf("guard.push", c.Guard.Push, "block", "ask", "off"); err != nil {
		return err
	}
	if err := oneOf("guard.breaking", c.Guard.Breaking, "ask", "off"); err != nil {
		return err
	}
	if err := oneOf("preserve.mode", c.Preserve.Mode, "auto", "manual", "off"); err != nil {
		return err
	}
	for i, t := range c.Changelog.Types {
		if t.Type == "" {
			return fmt.Errorf("changelog.types[%d]: type is required", i)
		}
		if t.Section == "" && !t.Hidden {
			return fmt.Errorf("changelog.types[%d] (%s): section is required unless hidden", i, t.Type)
		}
	}
	return nil
}

func oneOf(key, value string, allowed ...string) error {
	if value == "" {
		return nil // absent: the resolved default applies
	}
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fmt.Errorf("%s: %q is not one of %v", key, value, allowed)
}

// Default returns the canonical policy that pk init writes: the full
// conventional-commit type table, guard and preserve modes stated
// explicitly, and the given release branch guarded.
func Default(releaseBranch string) *PkConfig {
	return &PkConfig{
		Changelog: ChangelogConfig{
			Types: []TypeConfig{
				{Type: "feat", Section: "Added"},
				{Type: "fix", Section: "Fixed"},
				{Type: "deprecate", Section: "Deprecated"},
				{Type: "revert", Section: "Removed"},
				{Type: "security", Section: "Security"},
				{Type: "refactor", Section: "Changed"},
				{Type: "perf", Section: "Changed"},
				{Type: "docs", Section: "Documentation"},
				{Type: "chore", Section: "Maintenance"},
				{Type: "test", Section: "Maintenance"},
				{Type: "build", Section: "Maintenance"},
				{Type: "ci", Section: "Maintenance"},
				{Type: "style", Section: "Maintenance"},
				{Type: PlanType, Section: "Plans", Hidden: true},
			},
		},
		Guard: GuardConfig{
			Branches: []string{releaseBranch},
			Mode:     DefaultGuardMode,
			Breaking: DefaultGuardBreaking,
			Push:     DefaultGuardPush,
		},
		Preserve: PreserveConfig{Mode: DefaultPreserveMode},
		Release:  ReleaseSection{Branch: releaseBranch},
	}
}

// Write serializes cfg to root/.pk.json, indented, trailing newline.
func Write(root string, cfg *PkConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(root), append(data, '\n'), 0o644)
}
