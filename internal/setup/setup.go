// Package setup implements the setup command that configures a project's
// .claude/settings.json to use plankit for plan preservation and protection.
package setup

import (
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
)

//go:embed skills/*/SKILL.md
var skillsFS embed.FS

//go:embed rules/*.md
var rulesFS embed.FS

//go:embed template/CLAUDE.md
var templateFS embed.FS

// Hook represents a single hook command entry.
// Field order determines JSON output order.
type Hook struct {
	Type          string `json:"type"`
	Command       string `json:"command"`
	Async         bool   `json:"async,omitempty"`
	Timeout       int    `json:"timeout"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

// HookEntry pairs a matcher pattern with its hook commands.
type HookEntry struct {
	Matcher string `json:"matcher"`
	Hooks   []Hook `json:"hooks"`
}

// HooksConfig defines the PreToolUse and PostToolUse hook arrays.
// Field order determines JSON output order.
type HooksConfig struct {
	PreToolUse  []HookEntry `json:"PreToolUse"`
	PostToolUse []HookEntry `json:"PostToolUse,omitempty"`
}

// buildHookConfig returns the hook configuration for the given preserve mode.
func buildHookConfig(preserveMode string) HooksConfig {
	config := HooksConfig{
		PreToolUse: []HookEntry{
			{
				Matcher: "Bash",
				Hooks:   []Hook{{Type: "command", Command: "pk guard", Timeout: 5}},
			},
			{
				Matcher: "Edit",
				Hooks:   []Hook{{Type: "command", Command: "pk protect", Timeout: 5}},
			},
			{
				Matcher: "Write",
				Hooks:   []Hook{{Type: "command", Command: "pk protect", Timeout: 5}},
			},
		},
	}

	switch preserveMode {
	case "auto":
		config.PostToolUse = preserveHookEntry("pk preserve", "Preserving approved plan...", true, 60)
	case "manual":
		config.PostToolUse = preserveHookEntry("pk preserve --notify", "Checking plan...", false, 10)
	}

	return config
}

// preserveHookEntry builds a PostToolUse entry for the given preserve command.
func preserveHookEntry(command, statusMessage string, async bool, timeout int) []HookEntry {
	return []HookEntry{
		{
			Matcher: "ExitPlanMode",
			Hooks: []Hook{{
				Type:          "command",
				Command:       command,
				Async:         async,
				Timeout:       timeout,
				StatusMessage: statusMessage,
			}},
		},
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

// contentSHA computes the SHA256 hash of content.
func contentSHA(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// extractSHA extracts a pk SHA and the hashed content from a file.
// Supports two formats:
//   - HTML comment on first line: <!-- pk:sha256:... --> (for CLAUDE.md)
//   - YAML frontmatter field: pk_sha256: ... (for skills with frontmatter)
//
// Returns (sha, hashedContent, found).
func extractSHA(fileContent string) (string, string, bool) {
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

	storedSHA, hashedContent, found := extractSHA(string(data))
	if !found {
		return false, "skipped (not managed by pk)"
	}

	if contentSHA(hashedContent) != storedSHA {
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
	// boundaries as extractSHA, we hash the original content which becomes the body.
	var sha string
	if strings.HasPrefix(content, "---\n") {
		// For skills: SHA covers the body after frontmatter.
		closeIdx := strings.Index(content[4:], "\n---\n")
		if closeIdx >= 0 {
			body := content[4+closeIdx+5:]
			sha = contentSHA(body)
		} else {
			sha = contentSHA(content)
		}
	} else {
		// For non-frontmatter files: SHA covers the full content.
		sha = contentSHA(content)
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
	Force        bool
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stderr: os.Stderr,
	}
}

// Run configures the project's .claude/settings.json to use plankit.
func Run(cfg Config) error {
	projectDir := cfg.ProjectDir
	stderr := cfg.Stderr
	preserveMode := cfg.PreserveMode
	force := cfg.Force
	settingsDir := filepath.Join(projectDir, ".claude")
	settingsFile := filepath.Join(settingsDir, "settings.json")

	// Read existing settings or start fresh.
	var settings map[string]json.RawMessage
	data, err := os.ReadFile(settingsFile)
	if err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse %s: %w", settingsFile, err)
		}
	} else {
		settings = make(map[string]json.RawMessage)
	}

	// Merge plankit hooks with any existing user hooks.
	hookConfig := buildHookConfig(preserveMode)
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

	fmt.Fprintf(stderr, "Configured plankit in %s (preserve mode: %s)\n", settingsFile, preserveMode)

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

	// Check if pk is in PATH.
	if _, err := exec.LookPath("pk"); err != nil {
		fmt.Fprintln(stderr, "Warning: pk is not in your PATH. Hooks will silently skip until it is installed.")
	}

	fmt.Fprintln(stderr, "Restart Claude Code to apply changes.")
	return nil
}

// addPermission adds a permission string to the settings "permissions.allow" list
// if it is not already present.
func addPermission(settings map[string]json.RawMessage, perm string) error {
	// Parse existing permissions.
	var perms map[string]json.RawMessage
	if raw, ok := settings["permissions"]; ok {
		if err := json.Unmarshal(raw, &perms); err != nil {
			return err
		}
	} else {
		perms = make(map[string]json.RawMessage)
	}

	// Parse existing allow list.
	var allowList []string
	if raw, ok := perms["allow"]; ok {
		if err := json.Unmarshal(raw, &allowList); err != nil {
			return err
		}
	}

	// Check if permission already exists.
	for _, p := range allowList {
		if p == perm {
			return nil
		}
	}

	// Add permission.
	allowList = append(allowList, perm)
	allowJSON, err := json.Marshal(allowList)
	if err != nil {
		return err
	}
	perms["allow"] = json.RawMessage(allowJSON)

	permsJSON, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	settings["permissions"] = json.RawMessage(permsJSON)

	return nil
}

// mergeHooks merges plankit hooks into existing settings, preserving user hooks.
// Existing hooks with commands starting with "pk " are replaced; all others are kept.
func mergeHooks(settings map[string]json.RawMessage, newHooks HooksConfig) error {
	var existing HooksConfig
	if raw, ok := settings["hooks"]; ok {
		if err := json.Unmarshal(raw, &existing); err != nil {
			return err
		}
	}

	merged := HooksConfig{
		PreToolUse:  mergeHookCategory(existing.PreToolUse, newHooks.PreToolUse),
		PostToolUse: mergeHookCategory(existing.PostToolUse, newHooks.PostToolUse),
	}

	hooksJSON, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	settings["hooks"] = json.RawMessage(hooksJSON)
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

// filterNonPlankitHooks returns hooks whose command does not start with "pk ".
func filterNonPlankitHooks(hooks []Hook) []Hook {
	var result []Hook
	for _, h := range hooks {
		if !strings.HasPrefix(h.Command, "pk ") {
			result = append(result, h)
		}
	}
	return result
}
