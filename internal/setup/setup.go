// Package setup implements the setup command that configures a project's
// .claude/settings.json to use plankit for plan preservation and protection.
package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/paths"
	"github.com/markwharton/plankit/internal/readiness"
	"github.com/markwharton/plankit/internal/version"
)

// OrderedObject is a JSON object that preserves its key insertion order
// across unmarshal/marshal cycles. Go's standard map[string]json.RawMessage
// marshals keys alphabetically, which would silently reorder user-authored
// files like .claude/settings.json on every pk setup. Tools don't get to
// reorder user files for their own convenience — key order is a user choice.
type OrderedObject struct {
	keys   []string
	values map[string]json.RawMessage
}

// NewOrderedObject returns an empty object.
func NewOrderedObject() *OrderedObject {
	return &OrderedObject{values: make(map[string]json.RawMessage)}
}

// ParseOrderedObject parses a JSON object, preserving key order as it appears
// in the input. Returns an empty object for null or empty input.
func ParseOrderedObject(raw json.RawMessage) (*OrderedObject, error) {
	oo := NewOrderedObject()
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return oo, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected JSON object, got %v", tok)
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %v", tok)
		}
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
		if _, exists := oo.values[key]; !exists {
			oo.keys = append(oo.keys, key)
		}
		oo.values[key] = val
	}
	return oo, nil
}

// Get returns the raw JSON value for key and whether the key is present.
func (oo *OrderedObject) Get(key string) (json.RawMessage, bool) {
	v, ok := oo.values[key]
	return v, ok
}

// Has reports whether key is present.
func (oo *OrderedObject) Has(key string) bool {
	_, ok := oo.values[key]
	return ok
}

// Set updates the value for key. New keys are appended to the end;
// existing keys keep their position.
func (oo *OrderedObject) Set(key string, val json.RawMessage) {
	if _, exists := oo.values[key]; !exists {
		oo.keys = append(oo.keys, key)
	}
	oo.values[key] = val
}

// Delete removes key. No-op when key is absent.
func (oo *OrderedObject) Delete(key string) {
	if _, exists := oo.values[key]; !exists {
		return
	}
	delete(oo.values, key)
	for i, k := range oo.keys {
		if k == key {
			oo.keys = append(oo.keys[:i], oo.keys[i+1:]...)
			return
		}
	}
}

// Len returns the number of keys.
func (oo *OrderedObject) Len() int {
	return len(oo.keys)
}

// SortKeys sorts the object's top-level keys alphabetically. Used when writing
// .pk.json so pk setup and the conventions skill agree on key order.
func (oo *OrderedObject) SortKeys() {
	sort.Strings(oo.keys)
}

// resolveMode returns the first non-empty value, applying the precedence the
// caller passes (explicit flag > existing .pk.json > migrated > default).
func resolveMode(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// MarshalJSON emits the object with keys in their preserved order. The output
// is compact; json.MarshalIndent at the top level re-indents the whole tree.
func (oo *OrderedObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range oo.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kJSON, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(kJSON)
		buf.WriteByte(':')
		buf.Write(oo.values[k])
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// Config holds the dependencies for the setup command.
type Config struct {
	Stderr        io.Writer
	ProjectDir    string
	PreserveMode  string
	GuardMode     string
	PushGuardMode string
	Force         bool
	AllowNonGit   bool
	Version       string
	Baseline      bool
	BaselineAt    string
	Push          bool
	GitExec       func(projectDir string, args ...string) (string, error)
	ReadFile      func(string) ([]byte, error)
	WriteFile     func(string, []byte, os.FileMode) error
	Stat          func(string) (os.FileInfo, error)
	MkdirAll      func(string, os.FileMode) error
	ReadDir       func(string) ([]os.DirEntry, error)
	Remove        func(string) error
	Rename        func(string, string) error
	LookPath      func(string) (string, error)
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stderr:    os.Stderr,
		GitExec:   git.Exec,
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		Stat:      os.Stat,
		MkdirAll:  os.MkdirAll,
		ReadDir:   os.ReadDir,
		Remove:    os.Remove,
		Rename:    os.Rename,
		LookPath:  exec.LookPath,
	}
}

// Run configures the project's .claude/settings.json to use plankit.
func Run(cfg Config) error {
	if cfg.GitExec == nil {
		cfg.GitExec = git.Exec
	}
	projectDir := cfg.ProjectDir
	stderr := cfg.Stderr
	preserveMode := cfg.PreserveMode
	force := cfg.Force
	settingsDir := filepath.Join(projectDir, paths.ClaudeDir)
	settingsFile := filepath.Join(settingsDir, paths.SettingsFile)

	// Refuse to install outside a git working tree unless explicitly allowed.
	// pk requires git for most commands (guard, changelog, release, preserve),
	// though rules, skills, and protect still work without it.
	// IsRepo walks up parents, so monorepo subdirectories are correctly detected.
	if !git.IsRepo(cfg.Stat, projectDir) {
		if !cfg.AllowNonGit {
			return fmt.Errorf("this is not a git repository. pk requires git for most commands.\n\nRun `git init` first, or pass --allow-non-git to proceed anyway")
		}
		msg.Warnf(stderr, "this is not a git repository. Proceeding because --allow-non-git was set. Some commands (changelog, release) will not work until git is initialized.")
	}

	// Read existing settings or start fresh. OrderedObject preserves the
	// user's existing key order — pk setup must not reorder settings.json.
	settings := NewOrderedObject()
	data, err := cfg.ReadFile(settingsFile)
	if err == nil {
		parsed, err := ParseOrderedObject(data)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", settingsFile, err)
		}
		settings = parsed
	}

	// Capture the project's current modes from the existing (possibly old-style,
	// flag-bearing) hooks before we overwrite them — this is how a re-run migrates
	// modes into .pk.json. Resolve each final mode by precedence: explicit setup
	// flag > existing .pk.json value > migrated from old hook > default const.
	oldModes := InferModes(settings)
	pkConf, err := config.Load(cfg.ReadFile, filepath.Join(projectDir, paths.PkConfig))
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", paths.PkConfig, err)
	}
	guardMode := resolveMode(cfg.GuardMode, pkConf.Guard.Mode, oldModes.Guard, config.DefaultGuardMode)
	guardPush := resolveMode(cfg.PushGuardMode, pkConf.Guard.Push, oldModes.PushGuard, config.DefaultGuardPush)
	preserveMode = resolveMode(cfg.PreserveMode, pkConf.Preserve.Mode, oldModes.Preserve, config.DefaultPreserveMode)

	// Merge the static, bare plankit hooks (modes now live in .pk.json).
	hookConfig := buildHooks()
	if version.IsDevBuild(cfg.Version) {
		hookConfig.SessionStart = nil
	}
	if err := mergeHooks(settings, hookConfig); err != nil {
		return fmt.Errorf("failed to merge hooks: %w", err)
	}

	// Add pk permission for skill execution.
	if err := addPermission(settings, "Bash(pk:*)"); err != nil {
		return fmt.Errorf("failed to add permission: %w", err)
	}

	// Write with backup.
	if err := cfg.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Backup existing file.
	if _, err := cfg.Stat(settingsFile); err == nil {
		backupFile := settingsFile + ".bak"
		if err := cfg.Rename(settingsFile, backupFile); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		fmt.Fprintf(stderr, "Backed up existing settings to %s\n", filepath.Base(backupFile))
	}

	// Write new settings.
	output, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	output = append(output, '\n')

	if err := cfg.WriteFile(settingsFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", settingsFile, err)
	}

	// Track whether anything actually changed on disk, so the commit-message tip
	// is only printed when there's something for the user to commit.
	anyChanged := !bytes.Equal(data, output)

	fmt.Fprintf(stderr, "Configured plankit in %s (guard: %s, push: %s, preserve: %s)\n", settingsFile, guardMode, guardPush, preserveMode)

	// Write the resolved modes into .pk.json (field-merge; user-owned, no SHA).
	pkChanged, err := writePkModes(cfg, projectDir, guardMode, guardPush, preserveMode)
	if err != nil {
		return err
	}
	if pkChanged {
		fmt.Fprintf(stderr, "Wrote guard/preserve modes to %s\n", paths.PkConfig)
	}
	anyChanged = anyChanged || pkChanged

	// Install CLAUDE.md if none exists or if pristine (never forced — CLAUDE.md is user-owned once customized).
	claudeTemplate, err := fs.ReadFile(templateFS, "template/CLAUDE.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded CLAUDE.md template: %w", err)
	}
	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	changed, err := writeManaged(cfg, claudeFile, string(claudeTemplate), false)
	if err != nil {
		return err
	}
	anyChanged = anyChanged || changed

	// Install skills.
	skillsList, err := skills()
	if err != nil {
		return fmt.Errorf("failed to load embedded skills: %w", err)
	}
	msg.Section(stderr, "Skills")
	keptSkills := map[string]bool{}
	for _, skill := range skillsList {
		skillFile := filepath.Join(settingsDir, "skills", skill.Name, "SKILL.md")
		changed, err := writeManaged(cfg, skillFile, skill.Content, force)
		if err != nil {
			return err
		}
		anyChanged = anyChanged || changed
		keptSkills[skill.Name] = true
	}
	if pruneSkills(cfg, filepath.Join(settingsDir, "skills"), keptSkills) {
		anyChanged = true
	}

	// Install rules into a plankit/ subdirectory. A flat .claude/rules/ is a shared
	// namespace, so a same-named adopter rule would silently shadow ours; the subdir
	// makes collisions impossible. Claude Code discovers .claude/rules/ recursively,
	// so the rules still load every session.
	rulesList, err := rules()
	if err != nil {
		return fmt.Errorf("failed to load embedded rules: %w", err)
	}
	msg.Section(stderr, "Rules")
	keptRules := map[string]bool{}
	rulesSubdir := filepath.Join(settingsDir, "rules", "plankit")
	for _, rule := range rulesList {
		ruleFile := filepath.Join(rulesSubdir, rule.Name+".md")
		changed, err := writeManaged(cfg, ruleFile, rule.Content, force)
		if err != nil {
			return err
		}
		anyChanged = anyChanged || changed
		keptRules[rule.Name] = true
	}
	if pruneRules(cfg, rulesSubdir, keptRules) {
		anyChanged = true
	}
	// Migration: remove pristine pk-managed rules left at the old flat top-level
	// location by pre-subdir installs. The empty kept-set means every pristine pk
	// rule there is pruned; user-modified rules are preserved (with a warning) and
	// non-pk files (plankit-development.md, the adopter's own rules) are skipped.
	// pruneRules skips subdirectories, so the plankit/ subdir is untouched here.
	if pruneRules(cfg, filepath.Join(settingsDir, "rules"), map[string]bool{}) {
		anyChanged = true
	}

	// Install bootstrap script for cloud sandboxes.
	msg.Section(stderr, "Bootstrap")
	changed, err = writeInstallScript(cfg, projectDir, cfg.Version)
	if err != nil {
		return err
	}
	anyChanged = anyChanged || changed

	// Check if pk is in PATH.
	if _, err := cfg.LookPath("pk"); err != nil {
		msg.Warnf(stderr, "pk is not in your PATH. Hooks will silently skip until it is installed.")
	}

	// Baseline tag or discoverability tip.
	inGitRepo := git.IsRepo(cfg.Stat, projectDir)
	if cfg.Baseline {
		if !inGitRepo {
			return fmt.Errorf("--baseline requires a git repository")
		}
		if err := runBaseline(cfg, projectDir); err != nil {
			return err
		}
	} else if inGitRepo {
		if _, ok := readiness.ValidSemverTag(cfg.GitExec, projectDir); !ok {
			fmt.Fprintln(stderr, "No version tags found. If you plan to use pk changelog / pk release, anchor with:")
			msg.Hintf(stderr, "pk setup --baseline --push")
			msg.Or(stderr, "git tag v0.0.0 && git push origin v0.0.0")
		}
	}

	// Commit-message tip: shown only when something actually changed on disk
	// and pk is a real release build (dev builds have no meaningful version to pin).
	if anyChanged && !version.IsDevBuild(cfg.Version) {
		tipVersion := cfg.Version
		if !strings.HasPrefix(tipVersion, "v") {
			tipVersion = "v" + tipVersion
		}
		fmt.Fprintln(stderr, "Commit these updates on their own:")
		msg.Hintf(stderr, "git commit -m %q", "chore: update pk-managed files for "+tipVersion)
	}

	// Conventions reminder: shown only when no .pk.json exists, so a configured
	// project re-running setup on an upgrade is not nagged. Without release.branch,
	// pk release silently falls back to trunk flow.
	if inGitRepo && pkConf.Release.Branch == "" {
		fmt.Fprintln(stderr, "No release branch in .pk.json. Run /conventions in Claude Code to set guard.branches and release.branch.")
		msg.Itemf(stderr, "Without release.branch, pk release uses trunk flow (tags the current branch, no merge to a release branch).")
	}

	fmt.Fprintln(stderr, "Restart Claude Code to apply changes.")
	return nil
}
