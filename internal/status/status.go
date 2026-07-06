// Package status implements the status command that reports the plankit
// configuration state of a project.
package status

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/paths"
	"github.com/markwharton/plankit/internal/readiness"
	"github.com/markwharton/plankit/internal/setup"
)

// Config holds the dependencies for the status command.
type Config struct {
	Stderr     io.Writer
	ProjectDir string
	Brief      bool
	ReadFile   func(string) ([]byte, error)
	Stat       func(string) (os.FileInfo, error)
	ReadDir    func(string) ([]os.DirEntry, error)
	GitExec    func(dir string, args ...string) (string, error)
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stderr:   os.Stderr,
		ReadFile: os.ReadFile,
		Stat:     os.Stat,
		ReadDir:  os.ReadDir,
		GitExec:  git.Exec,
	}
}

// managedFile describes a scanned pk-managed file.
type managedFile struct {
	label    string // display label (e.g., "development-standards.md")
	path     string // absolute path
	modified bool   // true if SHA doesn't match (user modified)
}

// hookSummary describes a category of pk hooks.
type hookSummary struct {
	category string   // e.g., "PreToolUse"
	commands []string // e.g., ["pk guard", "pk protect"]
}

// Run inspects the project and reports plankit configuration.
// Returns (configured, error). If plankit is not configured, configured is false
// and error is nil — callers can use this to decide exit code behavior.
func Run(cfg Config) (bool, error) {
	stderr := cfg.Stderr
	settingsDir := filepath.Join(cfg.ProjectDir, paths.ClaudeDir)
	settingsFile := filepath.Join(settingsDir, paths.SettingsFile)

	// Detect git repository.
	isGit := isGitRepo(cfg, cfg.ProjectDir)

	// Read settings.json.
	var settings map[string]json.RawMessage
	settingsExists := false
	if data, err := cfg.ReadFile(settingsFile); err == nil {
		settingsExists = true
		if err := json.Unmarshal(data, &settings); err != nil {
			return false, fmt.Errorf("failed to parse %s: %w", settingsFile, err)
		}
	}

	// Analyze hooks and permissions.
	hooks := analyzeHooks(settings)
	hasPermission := hasPKPermission(settings)

	// Scan managed files.
	rules := scanManaged(cfg, filepath.Join(settingsDir, "rules"), false)
	skills := scanManaged(cfg, filepath.Join(settingsDir, "skills"), true)

	// Check CLAUDE.md.
	claudeFile := filepath.Join(cfg.ProjectDir, "CLAUDE.md")
	claudeMD := checkSingleFile(cfg, claudeFile, "CLAUDE.md")

	// Check install-pk.sh.
	installScript := filepath.Join(settingsDir, "install-pk.sh")
	_, installErr := cfg.Stat(installScript)
	hasInstallScript := installErr == nil

	// Load .pk.json if present.
	pkConf, hasPKConfig, err := loadPKConfig(cfg)
	if err != nil {
		return false, err
	}

	// Determine if configured.
	configured := len(hooks) > 0 || hasPermission || claudeMD != nil ||
		len(rules) > 0 || len(skills) > 0 || hasInstallScript

	// Readiness: release-readiness checks, evaluated when plankit hooks are
	// installed in a git repository (offline; local refs only).
	var checks []readiness.Check
	if isGit && len(hooks) > 0 && cfg.GitExec != nil {
		checks = readiness.Evaluate(cfg.GitExec, cfg.ProjectDir, pkConf)
	}

	// Brief mode: one-line summary.
	if cfg.Brief {
		return runBrief(cfg, configured, pkConf, len(hooks) > 0, isGit, checks)
	}

	if !configured {
		fmt.Fprintln(stderr, "plankit is not configured in this project.")
		msg.Hintf(stderr, "To install: pk setup")
		if !isGit {
			fmt.Fprintln(stderr, "")
			msg.Notef(stderr, "this is not a git repository; pk requires git for most commands")
		}
		return false, nil
	}

	// Configured — print detailed status.
	// Non-git is a misconfigured state; lead with an error and recovery path.
	if !isGit {
		fmt.Fprintln(stderr, "plankit is configured, but this is not a git repository.")
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Most pk commands will not work here.")
		msg.Hintf(stderr, "To make this a git repository: git init")
		msg.Hintf(stderr, "To remove plankit instead: pk teardown --confirm")
		fmt.Fprintln(stderr, "")
	} else {
		fmt.Fprintln(stderr, "plankit is configured in this project.")
		fmt.Fprintln(stderr, "")
	}

	// Modes (from .pk.json, with defaults resolved). Shown when plankit hooks
	// are installed. push is surfaced only when guard is active (block/ask).
	if len(hooks) > 0 {
		guardMode := pkConf.Guard.ResolvedMode()
		msg.Section(stderr, "Modes")
		fmt.Fprintf(stderr, "  guard:    %s\n", guardMode)
		if guardMode == "block" || guardMode == "ask" {
			fmt.Fprintf(stderr, "  push:     %s\n", pkConf.Guard.ResolvedPush())
		}
		fmt.Fprintf(stderr, "  preserve: %s\n", pkConf.Preserve.ResolvedMode())
		fmt.Fprintln(stderr, "")
	}

	// Hooks.
	if len(hooks) > 0 {
		msg.Section(stderr, "Hooks")
		for _, h := range hooks {
			fmt.Fprintf(stderr, "  %-13s %s\n", h.category+":", strings.Join(h.commands, ", "))
		}
		fmt.Fprintln(stderr, "")
	} else if settingsExists {
		fmt.Fprintln(stderr, "Hooks: none configured")
		fmt.Fprintln(stderr, "")
	}

	// Managed files.
	msg.Section(stderr, "Managed files")
	if claudeMD != nil {
		printFileStatus(stderr, "CLAUDE.md", claudeMD.modified)
	}
	printDirectoryStatus(stderr, ".claude/rules/", rules)
	printDirectoryStatus(stderr, ".claude/skills/", skills)
	if hasInstallScript {
		fmt.Fprintln(stderr, "  .claude/install-pk.sh   present")
	}
	fmt.Fprintln(stderr, "")

	// Permission.
	if hasPermission {
		msg.Section(stderr, "Permission")
		fmt.Fprintln(stderr, "  Bash(pk:*)             allowed")
		fmt.Fprintln(stderr, "")
	}

	// Config (.pk.json) — show key fields.
	if hasPKConfig {
		msg.Section(stderr, "Config (.pk.json)")
		if n := len(pkConf.Changelog.Types); n > 0 {
			fmt.Fprintf(stderr, "  changelog.types:  %d configured\n", n)
		}
		if pkConf.Changelog.Hooks.PreCommit != "" {
			fmt.Fprintf(stderr, "  changelog.hooks:  preCommit set\n")
		}
		if pkConf.Changelog.Hooks.PostVersion != "" {
			fmt.Fprintf(stderr, "  changelog.hooks:  postVersion set\n")
		}
		if pkConf.Release.Branch != "" {
			fmt.Fprintf(stderr, "  release.branch:   %s\n", pkConf.Release.Branch)
		}
		if pkConf.Release.Hooks.PreRelease != "" {
			fmt.Fprintf(stderr, "  release.hooks:    preRelease set\n")
		}
		if len(pkConf.Guard.Branches) > 0 {
			fmt.Fprintf(stderr, "  guard.branches:   %s\n", strings.Join(pkConf.Guard.Branches, ", "))
		}
	}

	printReadiness(stderr, hasPKConfig, checks)

	return true, nil
}

// printReadiness renders the release-readiness checks. All-pass collapses to
// a single line; failed checks carry their next-step hint (and git
// equivalent) indented beneath them.
func printReadiness(stderr io.Writer, afterConfig bool, checks []readiness.Check) {
	if len(checks) == 0 {
		return
	}
	if afterConfig {
		fmt.Fprintln(stderr, "")
	}
	if readiness.Ready(checks) {
		fmt.Fprintln(stderr, "Readiness: ready for pk changelog / pk release")
		return
	}
	width := 0
	for _, c := range checks {
		if len(c.Label) > width {
			width = len(c.Label)
		}
	}
	msg.Section(stderr, "Readiness")
	for _, c := range checks {
		msg.Itemf(stderr, "%-*s   %s", width, c.Label, c.Value)
		if c.Hint != "" {
			msg.Itemf(stderr, "  %s", c.Hint)
		}
		if c.Or != "" {
			msg.Itemf(stderr, "  or: %s", c.Or)
		}
	}
}

// runBrief prints a one-line status summary. Useful for scripting.
// Returns (configured, error). configured mirrors the input so Run can return runBrief's tuple directly.
func runBrief(cfg Config, configured bool, pkConf config.PkConfig, hasHooks bool, isGit bool, checks []readiness.Check) (bool, error) {
	stderr := cfg.Stderr
	if !configured {
		note := ""
		if !isGit {
			note = " (not a git repository)"
		}
		fmt.Fprintf(stderr, "plankit: not configured%s\n", note)
		return false, nil
	}
	parts := []string{"configured"}
	if hasHooks {
		guardMode := pkConf.Guard.ResolvedMode()
		parts = append(parts, "guard="+guardMode)
		if guardMode == "block" || guardMode == "ask" {
			parts = append(parts, "push="+pkConf.Guard.ResolvedPush())
		}
		parts = append(parts, "preserve="+pkConf.Preserve.ResolvedMode())
	}
	if !isGit {
		parts = append(parts, "not-a-git-repo")
	}
	if len(checks) > 0 {
		if readiness.Ready(checks) {
			parts = append(parts, "ready")
		} else {
			parts = append(parts, "not-ready")
		}
	}
	fmt.Fprintf(stderr, "plankit: %s\n", strings.Join(parts, ", "))
	return true, nil
}

// isGitRepo reports whether dir is inside a git working tree. It walks up
// parent directories, so monorepo subdirectories are correctly detected.
func isGitRepo(cfg Config, dir string) bool {
	return git.IsRepo(cfg.Stat, dir)
}

// loadPKConfig parses .pk.json using the shared config loader.
// Returns (config, exists, error). If the file doesn't exist, returns zero
// values with exists=false. Parse errors propagate.
func loadPKConfig(cfg Config) (config.PkConfig, bool, error) {
	path := filepath.Join(cfg.ProjectDir, paths.PkConfig)
	// Check existence: config.Load treats any readFile error as "missing".
	if _, err := cfg.ReadFile(path); err != nil {
		return config.PkConfig{}, false, nil
	}
	parsed, err := config.Load(cfg.ReadFile, path)
	if err != nil {
		return config.PkConfig{}, true, err
	}
	return parsed, true, nil
}

// analyzeHooks parses settings and returns pk hooks grouped by category.
func analyzeHooks(settings map[string]json.RawMessage) []hookSummary {
	raw, ok := settings["hooks"]
	if !ok {
		return nil
	}

	var hooks setup.HooksConfig
	if err := json.Unmarshal(raw, &hooks); err != nil {
		return nil
	}

	var result []hookSummary
	if cmds := extractPKCommands(hooks.PreToolUse); len(cmds) > 0 {
		result = append(result, hookSummary{category: "PreToolUse", commands: cmds})
	}
	if cmds := extractPKCommands(hooks.PostToolUse); len(cmds) > 0 {
		result = append(result, hookSummary{category: "PostToolUse", commands: cmds})
	}
	if cmds := extractPKCommands(hooks.SessionStart); len(cmds) > 0 {
		result = append(result, hookSummary{category: "SessionStart", commands: cmds})
	}
	return result
}

// extractPKCommands returns the deduplicated pk-managed commands from a hook category.
func extractPKCommands(entries []setup.HookEntry) []string {
	seen := make(map[string]bool)
	var cmds []string
	for _, entry := range entries {
		for _, h := range entry.Hooks {
			cmd := setup.HookCommand(h)
			if setup.IsPlankitHook(cmd) && !seen[cmd] {
				seen[cmd] = true
				cmds = append(cmds, cmd)
			}
		}
	}
	return cmds
}

// hasPKPermission checks if Bash(pk:*) is in settings.permissions.allow.
func hasPKPermission(settings map[string]json.RawMessage) bool {
	permRaw, ok := settings["permissions"]
	if !ok {
		return false
	}
	var perms map[string]json.RawMessage
	if err := json.Unmarshal(permRaw, &perms); err != nil {
		return false
	}
	allowRaw, ok := perms["allow"]
	if !ok {
		return false
	}
	var allowList []string
	if err := json.Unmarshal(allowRaw, &allowList); err != nil {
		return false
	}
	for _, p := range allowList {
		if p == "Bash(pk:*)" {
			return true
		}
	}
	return false
}

// scanManaged scans a directory for pk-managed files and returns them sorted.
// If nested is true, looks for <dir>/<subdir>/SKILL.md (skills layout); otherwise
// scans every <name>.md under dir recursively (rules layout), so pk-managed rules
// installed under .claude/rules/plankit/ are found. Subdir rules carry their path
// relative to dir as the label (e.g. "plankit/git-discipline.md").
func scanManaged(cfg Config, dir string, nested bool) []managedFile {
	var files []managedFile
	visit := func(path, rel string) error {
		if mf := checkSingleFile(cfg, path, rel); mf != nil {
			files = append(files, *mf)
		}
		return nil
	}
	// Walk errors mean an unreadable directory; it contributes nothing, as before.
	if nested {
		_ = setup.WalkSkillFiles(cfg.ReadDir, dir, visit)
	} else {
		_ = setup.WalkRuleFiles(cfg.ReadDir, dir, visit)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].label < files[j].label })
	return files
}

// checkSingleFile returns a managedFile if the file has a pk_sha256 marker,
// or nil if the file doesn't exist or isn't pk-managed.
func checkSingleFile(cfg Config, path, label string) *managedFile {
	data, err := cfg.ReadFile(path)
	if err != nil {
		return nil
	}
	prov := setup.Classify(string(data))
	if prov == setup.NotManaged {
		return nil
	}
	return &managedFile{
		label:    label,
		path:     path,
		modified: prov == setup.Modified,
	}
}

// printFileStatus prints a single file with its pristine/modified status.
func printFileStatus(w io.Writer, label string, modified bool) {
	status := "pristine"
	if modified {
		status = "modified by user"
	}
	fmt.Fprintf(w, "  %-23s %s\n", label, status)
}

// printDirectoryStatus prints a directory summary with counts of files and modifications.
func printDirectoryStatus(w io.Writer, dirLabel string, files []managedFile) {
	if len(files) == 0 {
		return
	}
	modifiedCount := 0
	for _, f := range files {
		if f.modified {
			modifiedCount++
		}
	}
	var summary string
	if modifiedCount == 0 {
		summary = fmt.Sprintf("%d file(s), all pristine", len(files))
	} else {
		summary = fmt.Sprintf("%d file(s), %d modified", len(files), modifiedCount)
	}
	fmt.Fprintf(w, "  %-23s %s\n", dirLabel, summary)
	for _, f := range files {
		if f.modified {
			fmt.Fprintf(w, "    - %s (modified by user)\n", f.label)
		}
	}
}
