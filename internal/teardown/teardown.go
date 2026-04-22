// Package teardown implements the teardown command that removes plankit
// hooks, skills, rules, and managed files from a project.
package teardown

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/setup"
)

// action describes what teardown will do (or did) with a file or setting.
type action struct {
	label  string // display label (e.g., "changelog/SKILL.md")
	verb   string // "will remove" / "removed" / "will skip" / "skipped"
	reason string // optional reason (e.g., "modified by user")
	path   string // absolute path (for manual removal hint)
}

// Config holds the dependencies for the teardown command.
type Config struct {
	Stderr     io.Writer
	ProjectDir string
	Confirm    bool
	ReadFile   func(string) ([]byte, error)
	WriteFile  func(string, []byte, os.FileMode) error
	Remove     func(string) error
	Stat       func(string) (os.FileInfo, error)
	ReadDir    func(string) ([]os.DirEntry, error)
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stderr:    os.Stderr,
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		Remove:    os.Remove,
		Stat:      os.Stat,
		ReadDir:   os.ReadDir,
	}
}

// Run removes plankit artifacts from a project. By default it previews
// what would be removed. Pass Confirm: true to execute.
func Run(cfg Config) error {
	stderr := cfg.Stderr
	settingsDir := filepath.Join(cfg.ProjectDir, ".claude")
	settingsFile := filepath.Join(settingsDir, "settings.json")

	// Phase 1: Analyze.
	var settingsActions []action
	var fileActions []action
	var dirActions []action
	var skippedPaths []string

	// --- Settings analysis ---
	settingsData, settingsErr := cfg.ReadFile(settingsFile)
	var settings *setup.OrderedObject
	settingsExists := false
	if settingsErr == nil {
		settingsExists = true
		parsed, err := setup.ParseOrderedObject(settingsData)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", settingsFile, err)
		}
		settings = parsed
		settingsActions = analyzeSettings(settings)
	}

	// --- Scan for managed files ---
	fileActions = append(fileActions, scanManagedFiles(cfg, settingsDir, "skills")...)
	fileActions = append(fileActions, scanManagedFiles(cfg, settingsDir, "rules")...)

	// Check CLAUDE.md.
	claudeFile := filepath.Join(cfg.ProjectDir, "CLAUDE.md")
	if a := analyzeManagedFile(cfg, claudeFile, "CLAUDE.md"); a != nil {
		fileActions = append(fileActions, *a)
	}

	// Check install-pk.sh (no SHA check).
	installScript := filepath.Join(settingsDir, "install-pk.sh")
	if _, err := cfg.Stat(installScript); err == nil {
		fileActions = append(fileActions, action{label: "install-pk.sh", path: installScript})
	}

	// Check settings.json.bak.
	backupFile := settingsFile + ".bak"
	if _, err := cfg.Stat(backupFile); err == nil {
		fileActions = append(fileActions, action{label: "settings.json.bak", path: backupFile})
	}

	// Nothing to do?
	if len(settingsActions) == 0 && len(fileActions) == 0 {
		fmt.Fprintln(stderr, "No plankit artifacts found.")
		return nil
	}

	// Collect skipped paths for the hint.
	for i := range fileActions {
		if fileActions[i].reason != "" {
			skippedPaths = append(skippedPaths, fileActions[i].path)
		}
	}

	// Determine directory cleanup candidates after file removal.
	dirActions = analyzeDirs(cfg, settingsDir, fileActions, settingsActions, settingsExists)

	// Phase 2: Preview.
	removeVerb := "will remove"
	skipVerb := "will skip"
	if cfg.Confirm {
		removeVerb = "removed"
		skipVerb = "skipped"
	}

	if len(settingsActions) > 0 {
		fmt.Fprintln(stderr, "Settings (.claude/settings.json):")
		for _, a := range settingsActions {
			fmt.Fprintf(stderr, "  %s ... %s\n", a.label, removeVerb)
		}
	}

	// Group file actions by category.
	printFileGroup(stderr, "Skills", fileActions, removeVerb, skipVerb, func(a action) bool {
		return strings.Contains(a.label, "SKILL.md")
	})
	printFileGroup(stderr, "Rules", fileActions, removeVerb, skipVerb, func(a action) bool {
		return !strings.Contains(a.label, "SKILL.md") && a.label != "CLAUDE.md" &&
			a.label != "install-pk.sh" && a.label != "settings.json.bak"
	})
	printFileGroup(stderr, "Files", fileActions, removeVerb, skipVerb, func(a action) bool {
		return a.label == "CLAUDE.md" || a.label == "install-pk.sh" || a.label == "settings.json.bak"
	})

	if len(dirActions) > 0 {
		fmt.Fprintln(stderr, "Directories:")
		for _, a := range dirActions {
			fmt.Fprintf(stderr, "  %s ... %s\n", a.label, removeVerb)
		}
	}

	if !cfg.Confirm {
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Run with --confirm to apply these changes.")
		if len(skippedPaths) > 0 {
			fmt.Fprintln(stderr, "")
			fmt.Fprintln(stderr, "Skipped files were modified after setup. To remove manually:")
			for _, p := range skippedPaths {
				fmt.Fprintf(stderr, "  rm %s\n", p)
			}
		}
		return nil
	}

	// Phase 3: Execute.
	//
	// Order matters: settings.json first, then files, then directories.
	// If a later step fails (e.g., a file removal hits a permissions error),
	// the settings are already consistent — hooks no longer reference removed
	// paths. The user can re-run teardown to clean up remaining artifacts.

	// Edit or remove settings.json first.
	if settingsExists && len(settingsActions) > 0 {
		removeHooks(settings)
		removePermission(settings, "Bash(pk:*)")

		if settings.Len() == 0 {
			if err := cfg.Remove(settingsFile); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove %s: %w", settingsFile, err)
			}
		} else {
			output, err := json.MarshalIndent(settings, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal settings: %w", err)
			}
			output = append(output, '\n')
			if err := cfg.WriteFile(settingsFile, output, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", settingsFile, err)
			}
		}
	}

	// Remove managed files (skip those with reasons — they're user-modified).
	for _, a := range fileActions {
		if a.reason != "" {
			continue
		}
		if err := cfg.Remove(a.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %w", a.path, err)
		}
	}

	// Remove empty directories (leaf-first order from analyzeDirs).
	for _, a := range dirActions {
		_ = cfg.Remove(a.path) // ignore errors — dir may not be empty
	}

	if len(skippedPaths) > 0 {
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Skipped files were modified after setup. To remove manually:")
		for _, p := range skippedPaths {
			fmt.Fprintf(stderr, "  rm %s\n", p)
		}
	}

	fmt.Fprintln(stderr, "Restart Claude Code to apply changes.")
	return nil
}

// analyzeSettings identifies pk hooks and permissions to remove.
// Uses OrderedObject so preview and execute paths see the same key order.
func analyzeSettings(settings *setup.OrderedObject) []action {
	var actions []action

	if raw, ok := settings.Get("hooks"); ok {
		hooks, err := setup.ParseOrderedObject(raw)
		if err == nil {
			for _, category := range setup.KnownHookCategories {
				actions = append(actions, findPKHooksInCategory(hooks, category)...)
			}
		}
	}

	// Check for pk permission.
	if permRaw, ok := settings.Get("permissions"); ok {
		perms, err := setup.ParseOrderedObject(permRaw)
		if err == nil {
			if allowRaw, ok := perms.Get("allow"); ok {
				var allowList []string
				if json.Unmarshal(allowRaw, &allowList) == nil {
					for _, p := range allowList {
						if p == "Bash(pk:*)" {
							actions = append(actions, action{label: "permissions.allow: Bash(pk:*)"})
						}
					}
				}
			}
		}
	}

	return actions
}

// findPKHooksInCategory returns removal actions for pk hooks in a single
// hook category. Returns nil if the category is missing or has no pk hooks.
func findPKHooksInCategory(hooks *setup.OrderedObject, category string) []action {
	raw, ok := hooks.Get(category)
	if !ok {
		return nil
	}
	var entries []setup.HookEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil
	}
	var actions []action
	for _, entry := range entries {
		for _, h := range entry.Hooks {
			if setup.IsPlankitHook(h.Command) {
				actions = append(actions, action{
					label: fmt.Sprintf("%s[%s]: %s", category, entry.Matcher, h.Command),
				})
			}
		}
	}
	return actions
}

// scanManagedFiles scans a directory for files with pk_sha256 markers.
// category is "skills" or "rules".
func scanManagedFiles(cfg Config, settingsDir, category string) []action {
	var actions []action
	dir := filepath.Join(settingsDir, category)

	switch category {
	case "skills":
		// Skills are in subdirectories: skills/{name}/SKILL.md
		entries, err := cfg.ReadDir(dir)
		if err != nil {
			return nil // directory doesn't exist
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
			label := entry.Name() + "/SKILL.md"
			if a := analyzeManagedFile(cfg, skillFile, label); a != nil {
				actions = append(actions, *a)
			}
		}
	case "rules":
		// Rules are flat: rules/{name}.md
		entries, err := cfg.ReadDir(dir)
		if err != nil {
			return nil
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			ruleFile := filepath.Join(dir, entry.Name())
			if a := analyzeManagedFile(cfg, ruleFile, entry.Name()); a != nil {
				actions = append(actions, *a)
			}
		}
	}

	return actions
}

// analyzeManagedFile checks a single file for a pk SHA marker.
// Returns nil if the file doesn't exist or has no pk marker (user-created).
func analyzeManagedFile(cfg Config, path, label string) *action {
	data, err := cfg.ReadFile(path)
	if err != nil {
		return nil // doesn't exist
	}

	storedSHA, body, found := setup.ExtractSHA(string(data))
	if !found {
		return nil // no pk marker — user-created, ignore
	}

	if setup.ContentSHA(body) != storedSHA {
		return &action{label: label, reason: "modified by user", path: path}
	}

	return &action{label: label, path: path}
}

// analyzeDirs determines which directories can be removed after file removal.
// Returns directories in leaf-first order.
func analyzeDirs(cfg Config, settingsDir string, fileActions, settingsActions []action, settingsExists bool) []action {
	var dirs []action

	// Count how many files will remain in each skill subdirectory.
	skillsDir := filepath.Join(settingsDir, "skills")
	if entries, err := cfg.ReadDir(skillsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			subdir := filepath.Join(skillsDir, entry.Name())
			// Check if the SKILL.md in this subdir is being removed.
			removing := false
			for _, a := range fileActions {
				if a.path == filepath.Join(subdir, "SKILL.md") && a.reason == "" {
					removing = true
					break
				}
			}
			if removing {
				// Check if directory will be empty after removal.
				subEntries, err := cfg.ReadDir(subdir)
				if err == nil && len(subEntries) == 1 {
					dirs = append(dirs, action{label: ".claude/skills/" + entry.Name() + "/", path: subdir})
				}
			}
		}
	}

	// Check if skills/ itself will be empty.
	if willBeEmpty(cfg, skillsDir, dirs) {
		dirs = append(dirs, action{label: ".claude/skills/", path: skillsDir})
	}

	// Check if rules/ will be empty.
	rulesDir := filepath.Join(settingsDir, "rules")
	rulesRemoving := 0
	for _, a := range fileActions {
		if strings.HasPrefix(a.path, rulesDir) && a.reason == "" {
			rulesRemoving++
		}
	}
	if entries, err := cfg.ReadDir(rulesDir); err == nil && len(entries) == rulesRemoving && rulesRemoving > 0 {
		dirs = append(dirs, action{label: ".claude/rules/", path: rulesDir})
	}

	// Check if .claude/ itself will be empty.
	// It's empty if: skills/ removed, rules/ removed, settings.json removed or will be removed,
	// install-pk.sh removed, settings.json.bak removed, and no other files.
	if willBeEmptyAfterTeardown(cfg, settingsDir, dirs, fileActions, settingsActions, settingsExists) {
		dirs = append(dirs, action{label: ".claude/", path: settingsDir})
	}

	return dirs
}

// willBeEmpty checks if a directory will be empty after removing subdirs in dirs.
func willBeEmpty(cfg Config, dir string, removedDirs []action) bool {
	entries, err := cfg.ReadDir(dir)
	if err != nil {
		return false
	}
	remaining := len(entries)
	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		for _, d := range removedDirs {
			if d.path == entryPath {
				remaining--
				break
			}
		}
	}
	return remaining == 0
}

// willBeEmptyAfterTeardown checks if .claude/ will be completely empty after teardown.
func willBeEmptyAfterTeardown(cfg Config, settingsDir string, removedDirs []action, fileActions, settingsActions []action, settingsExists bool) bool {
	entries, err := cfg.ReadDir(settingsDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		entryPath := filepath.Join(settingsDir, entry.Name())

		// Check if it's a directory being removed.
		dirRemoved := false
		for _, d := range removedDirs {
			if d.path == entryPath {
				dirRemoved = true
				break
			}
		}
		if dirRemoved {
			continue
		}

		// Check if it's a file being removed.
		fileRemoved := false
		for _, a := range fileActions {
			if a.path == entryPath && a.reason == "" {
				fileRemoved = true
				break
			}
		}
		if fileRemoved {
			continue
		}

		// Check if it's settings.json that will be removed (empty after edits).
		if entry.Name() == "settings.json" && settingsExists && len(settingsActions) > 0 {
			if settingsWillBeEmpty(cfg, entryPath) {
				continue
			}
			return false
		}

		// Something will remain.
		return false
	}

	return true
}

// settingsWillBeEmpty checks if settings.json will be empty after removing pk content.
func settingsWillBeEmpty(cfg Config, path string) bool {
	data, err := cfg.ReadFile(path)
	if err != nil {
		return false
	}
	settings, err := setup.ParseOrderedObject(data)
	if err != nil {
		return false
	}

	// Simulate removal.
	removeHooks(settings)
	removePermission(settings, "Bash(pk:*)")

	return settings.Len() == 0
}

// removeHooks removes all plankit hooks from the settings hooks config.
// Strips plankit hooks from every managed category (see setup.KnownHookCategories),
// preserving user hooks and any unknown categories (e.g., SessionEnd, Stop,
// UserPromptSubmit). Key order is preserved across the edit. If all managed
// categories become empty and no other categories exist, the hooks key is removed.
func removeHooks(settings *setup.OrderedObject) {
	raw, ok := settings.Get("hooks")
	if !ok {
		return
	}

	hooks, err := setup.ParseOrderedObject(raw)
	if err != nil {
		return
	}

	for _, key := range setup.KnownHookCategories {
		filterCategoryKey(hooks, key)
	}

	if hooks.Len() == 0 {
		settings.Delete("hooks")
		return
	}

	hooksJSON, err := json.Marshal(hooks)
	if err != nil {
		return
	}
	settings.Set("hooks", json.RawMessage(hooksJSON))
}

// filterCategoryKey removes plankit hooks from a single hook category,
// deleting the key if the category becomes empty. Preserves its position.
func filterCategoryKey(hooks *setup.OrderedObject, key string) {
	raw, ok := hooks.Get(key)
	if !ok {
		return
	}
	var entries []setup.HookEntry
	if json.Unmarshal(raw, &entries) != nil {
		return
	}
	filtered := filterCategory(entries)
	if len(filtered) == 0 {
		hooks.Delete(key)
		return
	}
	filteredJSON, err := json.Marshal(filtered)
	if err != nil {
		return
	}
	hooks.Set(key, json.RawMessage(filteredJSON))
}

// filterCategory removes plankit hooks from a hook category.
func filterCategory(entries []setup.HookEntry) []setup.HookEntry {
	var result []setup.HookEntry
	for _, entry := range entries {
		var filtered []setup.Hook
		for _, h := range entry.Hooks {
			if !setup.IsPlankitHook(h.Command) {
				filtered = append(filtered, h)
			}
		}
		if len(filtered) > 0 {
			entry.Hooks = filtered
			result = append(result, entry)
		}
	}
	return result
}

// removePermission removes a permission string from settings. Preserves the
// existing key order of the permissions object.
func removePermission(settings *setup.OrderedObject, perm string) {
	permRaw, ok := settings.Get("permissions")
	if !ok {
		return
	}

	perms, err := setup.ParseOrderedObject(permRaw)
	if err != nil {
		return
	}

	allowRaw, ok := perms.Get("allow")
	if !ok {
		return
	}

	var allowList []string
	if json.Unmarshal(allowRaw, &allowList) != nil {
		return
	}

	var filtered []string
	for _, p := range allowList {
		if p != perm {
			filtered = append(filtered, p)
		}
	}

	if len(filtered) == 0 {
		perms.Delete("allow")
	} else {
		allowJSON, _ := json.Marshal(filtered)
		perms.Set("allow", json.RawMessage(allowJSON))
	}

	if perms.Len() == 0 {
		settings.Delete("permissions")
	} else {
		permsJSON, _ := json.Marshal(perms)
		settings.Set("permissions", json.RawMessage(permsJSON))
	}
}

// printFileGroup prints a group of file actions filtered by a predicate.
func printFileGroup(w io.Writer, header string, actions []action, removeVerb, skipVerb string, match func(action) bool) {
	var matched []action
	for _, a := range actions {
		if match(a) {
			matched = append(matched, a)
		}
	}
	if len(matched) == 0 {
		return
	}
	fmt.Fprintln(w, header+":")
	for _, a := range matched {
		if a.reason != "" {
			fmt.Fprintf(w, "  %s ... %s (%s)\n", a.label, skipVerb, a.reason)
		} else {
			fmt.Fprintf(w, "  %s ... %s\n", a.label, removeVerb)
		}
	}
}
