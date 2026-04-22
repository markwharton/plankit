// Package setup implements the setup command that configures a project's
// .claude/settings.json to use plankit for plan preservation and protection.
package setup

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/version"
)

//go:embed skills/*/SKILL.md
var skillsFS embed.FS

//go:embed rules/*.md
var rulesFS embed.FS

//go:embed template/CLAUDE.md
var templateFS embed.FS

//go:embed template/install-pk.sh
var installScriptTemplate string

// Hook represents a single hook command entry.
// Field order determines JSON output order.
type Hook struct {
	Type          string `json:"type"`
	Command       string `json:"command"`
	Async         bool   `json:"async,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

// HookEntry pairs a matcher pattern with its hook commands. Hooks are
// carried as []json.RawMessage so user-authored hook objects pass through
// pk setup byte-for-byte — including fields that plankit doesn't recognise
// (e.g., continueOnError, a future Claude Code field, or a field from
// another tool). Plankit-owned hooks are built via NewHookEntry, which
// marshals the typed Hook struct into raw JSON at construction time.
type HookEntry struct {
	Matcher string            `json:"matcher"`
	Hooks   []json.RawMessage `json:"hooks"`
}

// NewHookEntry builds a HookEntry from the typed Hook values that plankit
// owns. Plankit hooks get their canonical field layout (type, command, async,
// timeout, statusMessage in struct-declaration order); user hooks are never
// round-tripped through this constructor — they stay as raw JSON.
func NewHookEntry(matcher string, hooks ...Hook) HookEntry {
	raw := make([]json.RawMessage, len(hooks))
	for i, h := range hooks {
		data, _ := json.Marshal(h)
		raw[i] = data
	}
	return HookEntry{Matcher: matcher, Hooks: raw}
}

// HookCommand extracts the command field from a raw hook object. Returns ""
// when the object is malformed or has no command. Used to identify plankit-
// owned hooks during merge and teardown without parsing the whole object.
func HookCommand(raw json.RawMessage) string {
	var x struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(raw, &x)
	return x.Command
}

// HooksConfig defines the hook arrays for each Claude Code event.
// Field order determines JSON output order.
type HooksConfig struct {
	PreToolUse   []HookEntry `json:"PreToolUse"`
	PostToolUse  []HookEntry `json:"PostToolUse,omitempty"`
	SessionStart []HookEntry `json:"SessionStart,omitempty"`
}

// KnownHookCategories lists the Claude Code hook categories plankit manages.
// Both mergeHooks (setup) and removeHooks (teardown) iterate this list.
// Adding a new category means: add its name here, add a matching field to
// HooksConfig, and add a case to HooksConfig.categoryEntries.
var KnownHookCategories = []string{"PreToolUse", "PostToolUse", "SessionStart"}

// categoryEntries returns the HookEntries in h for the given category name.
// Returns nil when name is not a known category.
func (h HooksConfig) categoryEntries(name string) []HookEntry {
	switch name {
	case "PreToolUse":
		return h.PreToolUse
	case "PostToolUse":
		return h.PostToolUse
	case "SessionStart":
		return h.SessionStart
	}
	return nil
}

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
		oo.keys = append(oo.keys, key)
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

// buildHookConfig returns the hook configuration for the given modes.
func buildHookConfig(preserveMode, guardMode string) HooksConfig {
	guardCommand := "pk guard"
	if guardMode == "ask" {
		guardCommand = "pk guard --ask"
	}

	config := HooksConfig{
		PreToolUse: []HookEntry{
			NewHookEntry("Bash", Hook{Type: "command", Command: guardCommand, Timeout: 5}),
			NewHookEntry("Edit", Hook{Type: "command", Command: "pk protect", Timeout: 5}),
			NewHookEntry("Write", Hook{Type: "command", Command: "pk protect", Timeout: 5}),
		},
	}

	switch preserveMode {
	case "auto":
		config.PostToolUse = preserveHookEntry("pk preserve", "Preserving approved plan...", true, 60)
	case "manual":
		config.PostToolUse = preserveHookEntry("pk preserve --notify", "Checking plan...", false, 10)
	}

	config.SessionStart = []HookEntry{
		NewHookEntry("*", Hook{Type: "command", Command: ".claude/install-pk.sh", Timeout: 30}),
	}

	return config
}

// preserveHookEntry builds a PostToolUse entry for the given preserve command.
func preserveHookEntry(command, statusMessage string, async bool, timeout int) []HookEntry {
	return []HookEntry{
		NewHookEntry("ExitPlanMode", Hook{
			Type:          "command",
			Command:       command,
			Async:         async,
			Timeout:       timeout,
			StatusMessage: statusMessage,
		}),
	}
}

// Skill represents a skill file to install.
type Skill struct {
	Name    string
	Content string
}

// skills returns the skills to install from the embedded filesystem.
func skills() ([]Skill, error) {
	entries, err := fs.ReadDir(skillsFS, "skills")
	if err != nil {
		return nil, err
	}

	var result []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		content, err := fs.ReadFile(skillsFS, "skills/"+entry.Name()+"/SKILL.md")
		if err != nil {
			return nil, err
		}
		result = append(result, Skill{
			Name:    entry.Name(),
			Content: string(content),
		})
	}
	return result, nil
}

// Rule represents a rules file to install.
type Rule struct {
	Name    string
	Content string
}

// rules returns the rules to install from the embedded filesystem.
func rules() ([]Rule, error) {
	entries, err := fs.ReadDir(rulesFS, "rules")
	if err != nil {
		return nil, err
	}

	var result []Rule
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		content, err := fs.ReadFile(rulesFS, "rules/"+entry.Name())
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		result = append(result, Rule{
			Name:    name,
			Content: string(content),
		})
	}
	return result, nil
}

const commentPrefix = "<!-- pk:sha256:"
const commentSuffix = " -->"
const frontmatterKey = "pk_sha256: "

// ContentSHA computes the SHA256 hash of content.
func ContentSHA(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// ExtractSHA extracts a pk SHA and the hashed content from a file.
// Supports two formats:
//   - HTML comment on first line: <!-- pk:sha256:... --> (for CLAUDE.md)
//   - YAML frontmatter field: pk_sha256: ... (for skills with frontmatter)
//
// Returns (sha, hashedContent, found).
func ExtractSHA(fileContent string) (string, string, bool) {
	// Try HTML comment on first line.
	firstNewline := strings.IndexByte(fileContent, '\n')
	if firstNewline > 0 {
		firstLine := fileContent[:firstNewline]
		if strings.HasPrefix(firstLine, commentPrefix) && strings.HasSuffix(firstLine, commentSuffix) {
			sha := firstLine[len(commentPrefix) : len(firstLine)-len(commentSuffix)]
			content := fileContent[firstNewline+1:]
			return sha, content, true
		}
	}

	// Try frontmatter pk_sha256 field.
	if strings.HasPrefix(fileContent, "---\n") {
		closeIdx := strings.Index(fileContent[4:], "\n---\n")
		if closeIdx >= 0 {
			frontmatter := fileContent[4 : 4+closeIdx]
			body := fileContent[4+closeIdx+5:] // skip past \n---\n
			for _, line := range strings.Split(frontmatter, "\n") {
				if strings.HasPrefix(line, frontmatterKey) {
					sha := strings.TrimSpace(line[len(frontmatterKey):])
					return sha, body, true
				}
			}
		}
	}

	return "", "", false
}

// embedSHA embeds a SHA into content using the appropriate format.
// Skills (content starting with ---) use a frontmatter field.
// Other files use an HTML comment on the first line.
func embedSHA(content string, sha string) string {
	if strings.HasPrefix(content, "---\n") {
		// Insert pk_sha256 field into existing frontmatter.
		closeIdx := strings.Index(content[4:], "\n---\n")
		if closeIdx >= 0 {
			frontmatter := content[4 : 4+closeIdx]
			body := content[4+closeIdx+5:]
			return "---\n" + frontmatter + "\n" + frontmatterKey + sha + "\n---\n" + body
		}
	}
	// HTML comment on first line.
	return commentPrefix + sha + commentSuffix + "\n" + content
}

// shouldUpdate checks whether a managed file should be updated.
// Returns (true, reason) if the file should be written, (false, reason) if it should be skipped.
func shouldUpdate(path string, newContent string, force bool) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, "created"
		}
		return false, "skipped (unreadable)"
	}

	if force {
		return true, "updated (forced)"
	}

	storedSHA, hashedContent, found := ExtractSHA(string(data))
	if !found {
		return false, "skipped (not managed by pk)"
	}

	if ContentSHA(hashedContent) != storedSHA {
		return false, "skipped (modified by user)"
	}

	return true, "updated"
}

// writeManaged writes content to path with a SHA marker embedded in the file.
// Skills with YAML frontmatter get a pk_sha256 field; other files get an HTML comment on line 1.
// If the file exists and has been modified by the user, it is skipped unless force is true.
func writeManaged(path string, content string, stderr io.Writer, force bool) error {
	update, reason := shouldUpdate(path, content, force)
	if !update {
		fmt.Fprintf(stderr, "  %s: %s\n", filepath.Base(path), reason)
		return nil
	}

	// Compute SHA over the body that will be hashed (content after frontmatter for skills,
	// content after the comment line for CLAUDE.md). Since embedSHA splits at the same
	// boundaries as ExtractSHA, we hash the original content which becomes the body.
	var sha string
	if strings.HasPrefix(content, "---\n") {
		// For skills: SHA covers the body after frontmatter.
		closeIdx := strings.Index(content[4:], "\n---\n")
		if closeIdx >= 0 {
			body := content[4+closeIdx+5:]
			sha = ContentSHA(body)
		} else {
			sha = ContentSHA(content)
		}
	} else {
		// For non-frontmatter files: SHA covers the full content.
		sha = ContentSHA(content)
	}

	managed := embedSHA(content, sha)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(managed), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	fmt.Fprintf(stderr, "  %s: %s\n", displayName(path), reason)
	return nil
}

// displayName returns a short display name for a managed file path.
// Uses parent/file for skills (e.g., "init/SKILL.md") and just the filename otherwise.
func displayName(path string) string {
	dir := filepath.Base(filepath.Dir(path))
	base := filepath.Base(path)
	if base == "SKILL.md" {
		return dir + "/" + base
	}
	return base
}

// Config holds the dependencies for the setup command.
type Config struct {
	Stderr       io.Writer
	ProjectDir   string
	PreserveMode string
	GuardMode    string
	Force        bool
	AllowNonGit  bool
	Version      string
	Baseline     bool
	BaselineAt   string
	Push         bool
	GitExec      func(projectDir string, args ...string) (string, error)
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stderr:  os.Stderr,
		GitExec: git.Exec,
	}
}

// ScriptVersion reads the pinned version from a file.
// Returns the version string and true if found, or ("", false) if the file
// does not exist or has no VERSION pin.
func ScriptVersion(filePath string) (string, bool) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if _, ok := versionPinName(line); ok {
			idx := strings.Index(line, `="`)
			if idx >= 0 && strings.HasSuffix(line, `"`) {
				return line[idx+2 : len(line)-1], true
			}
		}
	}
	return "", false
}

// PinVersion updates a version pin in a script file. It finds the first line
// matching SOMETHING_VERSION="vX.Y.Z" (any uppercase variable ending in VERSION)
// and replaces the version. Returns (true, nil) if updated, (false, nil) if the
// file does not exist (no-op), or (false, error) on failure.
func PinVersion(filePath string, ver string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, nil
	}
	if !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if name, ok := versionPinName(line); ok {
			lines[i] = fmt.Sprintf(`%s="v%s"`, name, strings.TrimPrefix(ver, "v"))
			found = true
			break
		}
	}
	if !found {
		return false, fmt.Errorf("%s has no VERSION pin", filepath.Base(filePath))
	}
	if err := os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0755); err != nil {
		return false, err
	}
	return true, nil
}

// versionPinName checks if a line matches the pattern SOMETHING_VERSION="v..."
// and returns the variable name. Returns ("", false) if no match.
func versionPinName(line string) (string, bool) {
	idx := strings.Index(line, `VERSION="v`)
	if idx < 0 {
		return "", false
	}
	name := line[:idx+len("VERSION")]
	for _, c := range name {
		if !((c >= 'A' && c <= 'Z') || c == '_') {
			return "", false
		}
	}
	if !strings.HasSuffix(line, `"`) {
		return "", false
	}
	return name, true
}

// writeInstallScript writes the cloud-sandbox bootstrap script to .claude/install-pk.sh.
// The script is template-substituted with the running pk version and written with 0755
// permissions. For development builds (version "dev"), the script is skipped.
func writeInstallScript(projectDir string, pkVersion string, stderr io.Writer) error {
	if pkVersion == "" || pkVersion == "dev" {
		fmt.Fprintln(stderr, "  install-pk.sh: skipped (development build)")
		return nil
	}
	if !strings.HasPrefix(pkVersion, "v") {
		pkVersion = "v" + pkVersion
	}
	content := strings.Replace(installScriptTemplate, "{{VERSION}}", pkVersion, 1)
	scriptPath := filepath.Join(projectDir, ".claude", "install-pk.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", scriptPath, err)
	}
	if err := os.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write %s: %w", scriptPath, err)
	}
	fmt.Fprintf(stderr, "  install-pk.sh: updated (pinned %s)\n", pkVersion)
	return nil
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
	settingsDir := filepath.Join(projectDir, ".claude")
	settingsFile := filepath.Join(settingsDir, "settings.json")

	// Refuse to install outside a git working tree unless explicitly allowed.
	// pk requires git for most commands (guard, changelog, release, preserve),
	// though rules, skills, and protect still work without it.
	// IsRepo walks up parents, so monorepo subdirectories are correctly detected.
	if !git.IsRepo(os.Stat, projectDir) {
		if !cfg.AllowNonGit {
			return fmt.Errorf("this is not a git repository. pk requires git for most commands.\n\nRun `git init` first, or pass --allow-non-git to proceed anyway")
		}
		fmt.Fprintln(stderr, "Warning: this is not a git repository. Proceeding because --allow-non-git was set. Some commands (changelog, release) will not work until git is initialized.")
	}

	// Read existing settings or start fresh. OrderedObject preserves the
	// user's existing key order — pk setup must not reorder settings.json.
	settings := NewOrderedObject()
	data, err := os.ReadFile(settingsFile)
	if err == nil {
		parsed, err := ParseOrderedObject(data)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", settingsFile, err)
		}
		settings = parsed
	}

	// Merge plankit hooks with any existing user hooks.
	guardMode := cfg.GuardMode
	hookConfig := buildHookConfig(preserveMode, guardMode)
	if err := mergeHooks(settings, hookConfig); err != nil {
		return fmt.Errorf("failed to merge hooks: %w", err)
	}

	// Add pk permission for skill execution.
	if err := addPermission(settings, "Bash(pk:*)"); err != nil {
		return fmt.Errorf("failed to add permission: %w", err)
	}

	// Write with backup.
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Backup existing file.
	if _, err := os.Stat(settingsFile); err == nil {
		backupFile := settingsFile + ".bak"
		if err := os.Rename(settingsFile, backupFile); err != nil {
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

	if err := os.WriteFile(settingsFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", settingsFile, err)
	}

	fmt.Fprintf(stderr, "Configured plankit in %s (guard mode: %s, preserve mode: %s)\n", settingsFile, guardMode, preserveMode)

	// Install CLAUDE.md if none exists or if pristine (never forced — CLAUDE.md is user-owned once customized).
	claudeTemplate, err := fs.ReadFile(templateFS, "template/CLAUDE.md")
	if err != nil {
		return fmt.Errorf("failed to read embedded CLAUDE.md template: %w", err)
	}
	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	if err := writeManaged(claudeFile, string(claudeTemplate), stderr, false); err != nil {
		return err
	}

	// Install skills.
	skillsList, err := skills()
	if err != nil {
		return fmt.Errorf("failed to load embedded skills: %w", err)
	}
	fmt.Fprintln(stderr, "Skills:")
	for _, skill := range skillsList {
		skillFile := filepath.Join(settingsDir, "skills", skill.Name, "SKILL.md")
		if err := writeManaged(skillFile, skill.Content, stderr, force); err != nil {
			return err
		}
	}

	// Install rules.
	rulesList, err := rules()
	if err != nil {
		return fmt.Errorf("failed to load embedded rules: %w", err)
	}
	fmt.Fprintln(stderr, "Rules:")
	for _, rule := range rulesList {
		ruleFile := filepath.Join(settingsDir, "rules", rule.Name+".md")
		if err := writeManaged(ruleFile, rule.Content, stderr, force); err != nil {
			return err
		}
	}

	// Install bootstrap script for cloud sandboxes.
	fmt.Fprintln(stderr, "Bootstrap:")
	if err := writeInstallScript(projectDir, cfg.Version, stderr); err != nil {
		return err
	}

	// Check if pk is in PATH.
	if _, err := exec.LookPath("pk"); err != nil {
		fmt.Fprintln(stderr, "Warning: pk is not in your PATH. Hooks will silently skip until it is installed.")
	}

	// Baseline tag or discoverability tip.
	inGitRepo := git.IsRepo(os.Stat, projectDir)
	if cfg.Baseline {
		if !inGitRepo {
			return fmt.Errorf("--baseline requires a git repository")
		}
		if err := runBaseline(cfg, projectDir); err != nil {
			return err
		}
	} else if inGitRepo {
		if _, ok := hasValidSemverTag(cfg, projectDir); !ok {
			fmt.Fprintln(stderr, "No version tags found. If you plan to use pk changelog / pk release, anchor with:")
			fmt.Fprintln(stderr, "  pk setup --baseline --push")
			fmt.Fprintln(stderr, "  or: git tag v0.0.0 && git push origin v0.0.0")
		}
	}

	fmt.Fprintln(stderr, "Restart Claude Code to apply changes.")
	return nil
}

// hasValidSemverTag returns the first tag matching "v*" that parses as a valid
// semver (per pk changelog's acceptance rule), or "", false if none exists.
func hasValidSemverTag(cfg Config, projectDir string) (string, bool) {
	output, err := cfg.GitExec(projectDir, "tag", "--list", "v*", "--sort=-v:refname")
	if err != nil || output == "" {
		return "", false
	}
	for _, line := range strings.Split(output, "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}
		if _, ok := version.ParseSemver(tag); ok {
			return tag, true
		}
	}
	return "", false
}

// runBaseline creates a v0.0.0 baseline tag if no valid semver tag exists.
// If cfg.BaselineAt is set, tags that ref; otherwise tags HEAD.
// If cfg.Push is set, also pushes the tag to origin.
func runBaseline(cfg Config, projectDir string) error {
	if existing, ok := hasValidSemverTag(cfg, projectDir); ok {
		fmt.Fprintf(cfg.Stderr, "Found tag %s — already anchored\n", existing)
		return nil
	}
	target := "HEAD"
	if cfg.BaselineAt != "" {
		if _, err := cfg.GitExec(projectDir, "rev-parse", "--verify", cfg.BaselineAt); err != nil {
			return fmt.Errorf("--at ref %q does not resolve", cfg.BaselineAt)
		}
		target = cfg.BaselineAt
	}
	if _, err := cfg.GitExec(projectDir, "tag", "v0.0.0", target); err != nil {
		return fmt.Errorf("failed to create tag v0.0.0: %w", err)
	}
	fmt.Fprintf(cfg.Stderr, "Tagged v0.0.0 on %s\n", target)
	if cfg.Push {
		// When tagging HEAD (default), also push the current branch so the tagged
		// commit is reachable from a branch on origin. When --at names a specific
		// ref, push only the tag — the user chose the ref explicitly, pk doesn't
		// assume which branch goes with it.
		pushArgs := []string{"push", "origin"}
		if cfg.BaselineAt == "" {
			pushArgs = append(pushArgs, "HEAD")
		}
		pushArgs = append(pushArgs, "v0.0.0")
		if _, err := cfg.GitExec(projectDir, pushArgs...); err != nil {
			return fmt.Errorf("failed to push baseline: %w", err)
		}
		if cfg.BaselineAt == "" {
			fmt.Fprintln(cfg.Stderr, "Pushed HEAD and v0.0.0 to origin")
		} else {
			fmt.Fprintln(cfg.Stderr, "Pushed v0.0.0 to origin")
		}
	} else {
		fmt.Fprintln(cfg.Stderr, "Run 'pk setup --baseline --push' to publish, or 'git push origin v0.0.0'")
	}
	return nil
}

// addPermission adds a permission string to the settings "permissions.allow" list
// if it is not already present. Preserves existing key order in the permissions
// object (allow, deny, ask, and any future keys).
func addPermission(settings *OrderedObject, perm string) error {
	perms := NewOrderedObject()
	if raw, ok := settings.Get("permissions"); ok {
		parsed, err := ParseOrderedObject(raw)
		if err != nil {
			return err
		}
		perms = parsed
	}

	var allowList []string
	if raw, ok := perms.Get("allow"); ok {
		if err := json.Unmarshal(raw, &allowList); err != nil {
			return err
		}
	}

	for _, p := range allowList {
		if p == perm {
			return nil
		}
	}

	allowList = append(allowList, perm)
	allowJSON, err := json.Marshal(allowList)
	if err != nil {
		return err
	}
	perms.Set("allow", json.RawMessage(allowJSON))

	permsJSON, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	settings.Set("permissions", json.RawMessage(permsJSON))

	return nil
}

// mergeHooks merges plankit hooks into existing settings, preserving user hooks
// and any unknown hook categories (e.g., SessionEnd, Stop, UserPromptSubmit).
// Existing hooks with commands starting with "pk " are replaced; all others are
// kept. Key order is preserved across the merge — both in the outer settings
// object and the inner hooks object.
func mergeHooks(settings *OrderedObject, newHooks HooksConfig) error {
	existing := NewOrderedObject()
	if raw, ok := settings.Get("hooks"); ok {
		parsed, err := ParseOrderedObject(raw)
		if err != nil {
			return err
		}
		existing = parsed
	}

	// Iterate KnownHookCategories so adding a new category is a one-liner.
	for _, cat := range KnownHookCategories {
		if err := mergeCategory(existing, cat, newHooks.categoryEntries(cat)); err != nil {
			return err
		}
	}

	if existing.Len() == 0 {
		settings.Delete("hooks")
		return nil
	}
	hooksJSON, err := json.Marshal(existing)
	if err != nil {
		return err
	}
	settings.Set("hooks", json.RawMessage(hooksJSON))
	return nil
}

// mergeCategory merges plankit hooks into a single category, preserving user
// hooks and the category's existing position in the hooks object. Empty
// categories after merging are removed.
func mergeCategory(existing *OrderedObject, key string, newEntries []HookEntry) error {
	var existingEntries []HookEntry
	if raw, ok := existing.Get(key); ok {
		if err := json.Unmarshal(raw, &existingEntries); err != nil {
			return err
		}
	}
	merged := mergeHookCategory(existingEntries, newEntries)
	if len(merged) == 0 {
		existing.Delete(key)
		return nil
	}
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	existing.Set(key, json.RawMessage(mergedJSON))
	return nil
}

// mergeHookCategory removes plankit hooks from existing entries and appends new plankit entries.
func mergeHookCategory(existing, plankit []HookEntry) []HookEntry {
	var result []HookEntry
	for _, entry := range existing {
		filtered := filterNonPlankitHooks(entry.Hooks)
		if len(filtered) > 0 {
			entry.Hooks = filtered
			result = append(result, entry)
		}
	}
	return append(result, plankit...)
}

// filterNonPlankitHooks returns hooks whose command is not managed by plankit.
// Operates on raw JSON so unknown fields on user hooks survive unchanged.
func filterNonPlankitHooks(hooks []json.RawMessage) []json.RawMessage {
	var result []json.RawMessage
	for _, h := range hooks {
		if !IsPlankitHook(HookCommand(h)) {
			result = append(result, h)
		}
	}
	return result
}

// IsPlankitHook reports whether a hook command is managed by plankit.
func IsPlankitHook(command string) bool {
	return strings.HasPrefix(command, "pk ") || command == ".claude/install-pk.sh"
}
