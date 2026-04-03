// Package setup implements the setup command that configures a project's
// .claude/settings.json to use plankit for plan preservation and protection.
package setup

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed skills/*/SKILL.md
var skillsFS embed.FS

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
		config.PostToolUse = preserveHookEntry("pk preserve", "Preserving approved plan...", 60)
	case "manual":
		config.PostToolUse = preserveHookEntry("pk preserve --notify", "Checking plan...", 10)
	}

	return config
}

// preserveHookEntry builds a PostToolUse entry for the given preserve command.
func preserveHookEntry(command, statusMessage string, timeout int) []HookEntry {
	return []HookEntry{
		{
			Matcher: "ExitPlanMode",
			Hooks: []Hook{{
				Type:          "command",
				Command:       command,
				Async:         true,
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

// Run configures the project's .claude/settings.json to use plankit.
func Run(projectDir string, stderr io.Writer, preserveMode string) error {
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

	// Marshal the hooks config.
	hookConfig := buildHookConfig(preserveMode)
	hooksJSON, err := json.Marshal(hookConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal hooks config: %w", err)
	}
	settings["hooks"] = json.RawMessage(hooksJSON)

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

	// Install skills.
	skillsList, err := skills()
	if err != nil {
		return fmt.Errorf("failed to load embedded skills: %w", err)
	}
	for _, skill := range skillsList {
		skillDir := filepath.Join(settingsDir, "skills", skill.Name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return fmt.Errorf("failed to create skills directory: %w", err)
		}
		skillFile := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillFile, []byte(skill.Content), 0644); err != nil {
			return fmt.Errorf("failed to write skill %s: %w", skill.Name, err)
		}
		fmt.Fprintf(stderr, "Installed skill: /%s\n", skill.Name)
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
