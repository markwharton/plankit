// Package changelog implements the pk changelog command.
// It generates CHANGELOG.md from conventional commits, commits, and tags.
package changelog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Config holds injectable dependencies for testing.
type Config struct {
	Stderr    io.Writer
	GitExec   func(args ...string) (string, error)
	ReadFile  func(name string) ([]byte, error)
	WriteFile func(name string, data []byte, perm os.FileMode) error
	RunScript func(command string) error
	Now       func() time.Time

	// Bump overrides auto-detected version bump: "major", "minor", or "patch".
	Bump string
	// DryRun previews the changelog section without writing, committing, or tagging.
	DryRun bool
}

// DefaultConfig returns a Config wired to real implementations.
func DefaultConfig() Config {
	return Config{
		Stderr: os.Stderr,
		GitExec: func(args ...string) (string, error) {
			out, err := exec.Command("git", args...).CombinedOutput()
			return strings.TrimRight(string(out), "\n"), err
		},
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		RunScript: func(command string) error {
			cmd := exec.Command("sh", "-c", command)
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		Now: time.Now,
	}
}

// Version holds semver components.
type Version struct {
	Major, Minor, Patch int
}

// Commit represents a parsed conventional commit.
type Commit struct {
	Hash     string
	Type     string
	Scope    string
	Message  string
	Breaking bool
}

// CommitGroup holds commits grouped by changelog section heading.
type CommitGroup struct {
	Heading string
	Items   []Commit
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
	Type string `json:"type"` // "json"
}

// Hooks holds lifecycle hook commands.
type Hooks struct {
	PostVersion string `json:"postVersion,omitempty"`
	PreCommit   string `json:"preCommit,omitempty"`
}

// ChangelogConfig is the .changelog.json schema.
type ChangelogConfig struct {
	Types        []TypeConfig  `json:"types,omitempty"`
	VersionFiles []VersionFile `json:"versionFiles,omitempty"`
	Hooks        Hooks         `json:"hooks,omitempty"`
}

// Bump type constants.
const (
	BumpPatch = iota + 1
	BumpMinor
	BumpMajor
)

var defaultTypes = []TypeConfig{
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
}

// commitRegex parses conventional commit subjects: type(scope)!: message
var commitRegex = regexp.MustCompile(`^(\w+)(?:\(([^)]*)\))?(!)?\s*:\s*(.+)$`)

const changelogHeader = `# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).
`

// Run executes the changelog command. Returns the process exit code.
func Run(cfg Config) int {
	// 1. Load config.
	config := loadConfig(cfg.ReadFile)

	// 2. Get latest tag.
	tagOutput, err := cfg.GitExec("tag", "--list", "v*", "--sort=-v:refname")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: failed to list tags: %v\n", err)
		return 1
	}
	latestTag := firstLine(tagOutput)
	if latestTag == "" {
		fmt.Fprintln(cfg.Stderr, "Error: no version tags found")
		fmt.Fprintln(cfg.Stderr, "  To start from scratch:     git tag v0.0.0")
		fmt.Fprintln(cfg.Stderr, "  To match existing version: git tag v1.2.3")
		return 1
	}
	baseVersion, ok := parseVersion(latestTag)
	if !ok {
		fmt.Fprintf(cfg.Stderr, "Error: invalid version tag %q\n", latestTag)
		return 1
	}

	// 3. Get commits since tag.
	logOutput, err := cfg.GitExec("log", "--format=%h%x00%s%x00%b%x00", latestTag+"..HEAD", "--reverse")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: failed to read git log: %v\n", err)
		return 1
	}

	// 4. Parse conventional commits.
	commits := parseLog(logOutput)
	if len(commits) == 0 {
		fmt.Fprintln(cfg.Stderr, "No new conventional commits found.")
		return 0
	}
	fmt.Fprintf(cfg.Stderr, "Found %d conventional commit(s)\n", len(commits))

	// 5. Determine bump.
	bump, err := resolveBump(cfg.Bump, commits)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: %v\n", err)
		return 1
	}

	// 6. Compute next version.
	next := bumpVersion(baseVersion, bump)
	nextTag := formatVersion(next)
	fmt.Fprintf(cfg.Stderr, "Generating %s\n", nextTag)

	// 7. Generate section.
	groups := groupCommits(commits, config.Types)
	date := cfg.Now().Format("2006-01-02")
	section := formatSection(nextTag, date, groups)

	// 8. Dry run.
	if cfg.DryRun {
		fmt.Fprintln(cfg.Stderr, "")
		fmt.Fprint(cfg.Stderr, section)
		return 0
	}

	// 9. Version without v prefix for files and hooks.
	ver := strings.TrimPrefix(nextTag, "v")

	// 10. Update version files.
	for _, vf := range config.VersionFiles {
		if err := updateVersionFile(cfg.ReadFile, cfg.WriteFile, vf.Path, ver); err != nil {
			fmt.Fprintf(cfg.Stderr, "Error: failed to update %s: %v\n", vf.Path, err)
			return 1
		}
		fmt.Fprintf(cfg.Stderr, "Updated %s\n", vf.Path)
	}

	// 11. Run postVersion hook.
	if config.Hooks.PostVersion != "" {
		fmt.Fprintf(cfg.Stderr, "Running postVersion hook...\n")
		if err := cfg.RunScript("VERSION=" + ver + " " + config.Hooks.PostVersion); err != nil {
			fmt.Fprintf(cfg.Stderr, "Error: postVersion hook failed: %v\n", err)
			return 1
		}
	}

	// 12. Get repo URL for comparison links.
	repoURL := ""
	if remoteURL, err := cfg.GitExec("remote", "get-url", "origin"); err == nil {
		repoURL = parseRepoURL(remoteURL)
	}

	// 13. Read existing CHANGELOG.md.
	existing, _ := cfg.ReadFile("CHANGELOG.md")

	// 14. Insert section and comparison link.
	updated := insertSection(string(existing), section)
	if repoURL != "" {
		refLink := fmt.Sprintf("[%s]: %s/compare/%s...%s", nextTag, repoURL, latestTag, nextTag)
		updated = appendRefLink(updated, refLink)
	}

	// 15. Write CHANGELOG.md.
	if err := cfg.WriteFile("CHANGELOG.md", []byte(updated), 0644); err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: failed to write CHANGELOG.md: %v\n", err)
		return 1
	}

	// 16. Run preCommit hook.
	if config.Hooks.PreCommit != "" {
		fmt.Fprintf(cfg.Stderr, "Running preCommit hook...\n")
		if err := cfg.RunScript("VERSION=" + ver + " " + config.Hooks.PreCommit); err != nil {
			fmt.Fprintf(cfg.Stderr, "Error: preCommit hook failed: %v\n", err)
			return 1
		}
	}

	// 17. Git add, commit, tag.
	// Explicitly add CHANGELOG.md (may be new/untracked) and version files.
	addFiles := []string{"add", "CHANGELOG.md"}
	for _, vf := range config.VersionFiles {
		addFiles = append(addFiles, vf.Path)
	}
	if _, err := cfg.GitExec(addFiles...); err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git add failed: %v\n", err)
		return 1
	}
	// Also stage any tracked files modified by hooks.
	if _, err := cfg.GitExec("add", "-u"); err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git add failed: %v\n", err)
		return 1
	}
	commitMsg := fmt.Sprintf("chore: release %s", nextTag)
	if _, err := cfg.GitExec("commit", "-m", commitMsg); err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git commit failed: %v\n", err)
		return 1
	}
	if _, err := cfg.GitExec("tag", nextTag); err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git tag failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(cfg.Stderr, "Released %s\n", nextTag)
	return 0
}

// loadConfig reads .changelog.json or returns defaults.
func loadConfig(readFile func(string) ([]byte, error)) ChangelogConfig {
	data, err := readFile(".changelog.json")
	if err != nil {
		return ChangelogConfig{Types: defaultTypes}
	}
	var config ChangelogConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ChangelogConfig{Types: defaultTypes}
	}
	if len(config.Types) == 0 {
		config.Types = defaultTypes
	}
	return config
}

// parseVersion parses "vX.Y.Z" into a Version.
func parseVersion(tag string) (Version, bool) {
	s := strings.TrimPrefix(tag, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return Version{}, false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return Version{}, false
	}
	return Version{Major: major, Minor: minor, Patch: patch}, true
}

// formatVersion produces "vX.Y.Z" from a Version.
func formatVersion(v Version) string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// bumpVersion increments a Version by the given bump type.
func bumpVersion(v Version, bump int) Version {
	switch bump {
	case BumpMajor:
		return Version{Major: v.Major + 1}
	case BumpMinor:
		return Version{Major: v.Major, Minor: v.Minor + 1}
	case BumpPatch:
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	default:
		return v
	}
}

// parseCommit parses a conventional commit from hash, subject, and body.
func parseCommit(hash, subject, body string) (Commit, bool) {
	m := commitRegex.FindStringSubmatch(subject)
	if m == nil {
		return Commit{}, false
	}
	c := Commit{
		Hash:     hash,
		Type:     m[1],
		Scope:    m[2],
		Breaking: m[3] == "!",
		Message:  m[4],
	}
	if !c.Breaking {
		c.Breaking = hasBreakingTrailer(body)
	}
	return c, true
}

// hasBreakingTrailer checks for BREAKING CHANGE or BREAKING-CHANGE in the body.
func hasBreakingTrailer(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "BREAKING CHANGE:") || strings.HasPrefix(trimmed, "BREAKING-CHANGE:") {
			return true
		}
	}
	return false
}

// parseLog splits NUL-delimited git log output into Commits.
// Format: %h%x00%s%x00%b%x00 (hash NUL subject NUL body NUL) repeated.
func parseLog(output string) []Commit {
	if strings.TrimSpace(output) == "" {
		return nil
	}
	// Split on NUL. Each commit produces 3 fields + possible trailing empty.
	fields := strings.Split(output, "\x00")
	var commits []Commit
	for i := 0; i+2 < len(fields); i += 3 {
		hash := strings.TrimSpace(fields[i])
		subject := strings.TrimSpace(fields[i+1])
		body := strings.TrimSpace(fields[i+2])
		if hash == "" || subject == "" {
			continue
		}
		if c, ok := parseCommit(hash, subject, body); ok {
			commits = append(commits, c)
		}
	}
	return commits
}

// detectBump returns the highest bump type from commits.
func detectBump(commits []Commit) int {
	bump := BumpPatch
	for _, c := range commits {
		if c.Breaking {
			return BumpMajor
		}
		if c.Type == "feat" && bump < BumpMinor {
			bump = BumpMinor
		}
	}
	return bump
}

// resolveBump resolves the bump from flag or auto-detection.
func resolveBump(flag string, commits []Commit) (int, error) {
	if flag == "" {
		return detectBump(commits), nil
	}
	switch flag {
	case "major":
		return BumpMajor, nil
	case "minor":
		return BumpMinor, nil
	case "patch":
		return BumpPatch, nil
	default:
		return 0, fmt.Errorf("invalid --bump value %q (must be major, minor, or patch)", flag)
	}
}

// groupCommits groups commits by changelog section using the type config.
// Hidden types are excluded. Section ordering follows the config order.
func groupCommits(commits []Commit, types []TypeConfig) []CommitGroup {
	// Build type→section lookup and track which types are hidden.
	typeSection := make(map[string]string)
	hidden := make(map[string]bool)
	for _, tc := range types {
		if tc.Hidden {
			hidden[tc.Type] = true
		} else {
			typeSection[tc.Type] = tc.Section
		}
	}

	// Group commits by section, preserving config order.
	sectionCommits := make(map[string][]Commit)
	for _, c := range commits {
		if hidden[c.Type] {
			continue
		}
		section, ok := typeSection[c.Type]
		if !ok {
			continue // unknown type, skip
		}
		sectionCommits[section] = append(sectionCommits[section], c)
	}

	// Build groups in config order (first appearance of each section).
	var groups []CommitGroup
	seen := make(map[string]bool)
	for _, tc := range types {
		if tc.Hidden || seen[tc.Section] {
			continue
		}
		seen[tc.Section] = true
		if items, ok := sectionCommits[tc.Section]; ok {
			groups = append(groups, CommitGroup{Heading: tc.Section, Items: items})
		}
	}
	return groups
}

// formatSection renders a version's changelog section as markdown.
func formatSection(ver, date string, groups []CommitGroup) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## [%s] - %s\n", ver, date)
	for _, g := range groups {
		fmt.Fprintf(&b, "\n### %s\n\n", g.Heading)
		for _, c := range g.Items {
			prefix := ""
			if c.Breaking {
				prefix = "**BREAKING:** "
			}
			fmt.Fprintf(&b, "- %s%s (%s)\n", prefix, c.Message, c.Hash)
		}
	}
	return b.String()
}

// insertSection inserts a new version section into existing changelog content.
// If existing is empty, generates a new file with the standard header.
func insertSection(existing, section string) string {
	if strings.TrimSpace(existing) == "" {
		return changelogHeader + "\n" + section
	}
	// Find the first "## [" line and insert before it.
	idx := strings.Index(existing, "\n## [")
	if idx >= 0 {
		return existing[:idx+1] + section + "\n" + existing[idx+1:]
	}
	// No version sections found, append after content.
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	return existing + "\n" + section
}

// updateVersionFile updates the root-level "version" field in a JSON file.
// Uses json.Decoder to locate the field and splices in the new value,
// preserving all formatting, key order, and indentation.
func updateVersionFile(readFile func(string) ([]byte, error), writeFile func(string, []byte, os.FileMode) error, path, ver string) error {
	content, err := readFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	updated, err := spliceJSONVersion(content, ver)
	if err != nil {
		return fmt.Errorf("update %s: %w", path, err)
	}

	return writeFile(path, updated, 0644)
}

// spliceJSONVersion locates the root-level "version" key in JSON content
// and replaces its value with the new version string.
func spliceJSONVersion(content []byte, newVersion string) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(content))

	// Expect opening brace.
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("expected JSON object: %w", err)
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected JSON object, got %v", tok)
	}

	for dec.More() {
		// Read key.
		keyTok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyTok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %T", keyTok)
		}

		// Record offset before reading value.
		beforeValue := dec.InputOffset()

		// Read value.
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}

		if key == "version" {
			afterValue := dec.InputOffset()

			// Find where the raw value starts within content[beforeValue:afterValue].
			segment := content[beforeValue:afterValue]
			rawIdx := bytes.Index(segment, raw)
			if rawIdx < 0 {
				return nil, fmt.Errorf("could not locate version value in source")
			}

			absStart := int(beforeValue) + rawIdx
			absEnd := absStart + len(raw)

			newValue, _ := json.Marshal(newVersion)

			result := make([]byte, 0, len(content)-len(raw)+len(newValue))
			result = append(result, content[:absStart]...)
			result = append(result, newValue...)
			result = append(result, content[absEnd:]...)
			return result, nil
		}
	}

	return nil, fmt.Errorf("no version field found at root level")
}

// parseRepoURL converts a git remote URL to an HTTPS base URL.
// Handles SSH (git@github.com:owner/repo.git) and HTTPS formats.
func parseRepoURL(remoteURL string) string {
	u := strings.TrimSpace(remoteURL)
	// SSH format: git@github.com:owner/repo.git
	if strings.HasPrefix(u, "git@") {
		u = strings.TrimPrefix(u, "git@")
		u = strings.Replace(u, ":", "/", 1)
		u = "https://" + u
	}
	u = strings.TrimSuffix(u, ".git")
	return u
}

// appendRefLink appends a markdown reference link definition to the content.
// Ensures proper newline separation.
func appendRefLink(content, refLink string) string {
	s := strings.TrimRight(content, "\n")
	return s + "\n\n" + refLink + "\n"
}

// firstLine returns the first non-empty line from s.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
