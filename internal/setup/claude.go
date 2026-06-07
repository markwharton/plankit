package setup

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/paths"
	"github.com/markwharton/plankit/internal/version"
)

// The /conventions skill also carries a copy of this template for when CLAUDE.md
// is missing. Update both when changing the Critical Rules header.
//
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
	Shell         string `json:"shell,omitempty"`
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

// Hook command constants used by buildHookConfig and InferModesFromCommands.
const (
	GuardBlockCommand     = "pk guard"
	GuardAskCommand       = "pk guard --ask"
	PreserveAutoCommand   = "pk preserve"
	PreserveManualCommand = "pk preserve --notify"
)

// buildHookConfig returns the hook configuration for the given modes.
func buildHookConfig(preserveMode, guardMode string) HooksConfig {
	return buildHookConfigWithPush(preserveMode, guardMode, "off")
}

// buildHookConfigWithPush is buildHookConfig plus a push-guard mode appended to the
// guard command as `--push-guard <mode>` when the mode is not "off".
func buildHookConfigWithPush(preserveMode, guardMode, pushGuardMode string) HooksConfig {
	var preToolUse []HookEntry

	if guardMode != "off" {
		guardCommand := GuardBlockCommand
		if guardMode == "ask" {
			guardCommand = GuardAskCommand
		}
		if pushGuardMode != "" && pushGuardMode != "off" {
			guardCommand += " --push-guard " + pushGuardMode
		}
		preToolUse = append(preToolUse, NewHookEntry("Bash|PowerShell", Hook{Type: "command", Command: guardCommand, Shell: "bash", Timeout: 5}))
	}

	preToolUse = append(preToolUse,
		NewHookEntry("Edit", Hook{Type: "command", Command: "pk protect", Shell: "bash", Timeout: 5}),
		NewHookEntry("Write", Hook{Type: "command", Command: "pk protect", Shell: "bash", Timeout: 5}),
	)

	config := HooksConfig{
		PreToolUse: preToolUse,
	}

	switch preserveMode {
	case "auto":
		config.PostToolUse = preserveHookEntry(PreserveAutoCommand, "Preserving approved plan...", true, 60)
	case "manual":
		config.PostToolUse = preserveHookEntry(PreserveManualCommand, "Checking plan...", false, 10)
	}

	config.SessionStart = []HookEntry{
		NewHookEntry("*", Hook{Type: "command", Command: ".claude/install-pk.sh", Shell: "bash", Timeout: 30}),
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
			Shell:         "bash",
			Timeout:       timeout,
			StatusMessage: statusMessage,
		}),
	}
}

// InferModesFromCommands returns guard and preserve modes from a list of hook
// command strings. Returns ("", "") when no plankit hooks are found (fresh
// project). Returns "off" for a mode when plankit hooks exist but that
// specific mode's command is absent (explicitly disabled).
func InferModesFromCommands(commands []string) (guard, preserve string) {
	hasPlankit := false
	for _, cmd := range commands {
		if IsPlankitHook(cmd) {
			hasPlankit = true
		}
		// Guard command may carry flags (--ask, --push-guard <mode>), so match by
		// prefix and read the branch mode from --ask rather than exact-matching.
		if strings.HasPrefix(cmd, GuardBlockCommand) {
			if strings.Contains(cmd, " --ask") {
				guard = "ask"
			} else {
				guard = "block"
			}
		}
		switch cmd {
		case PreserveManualCommand:
			preserve = "manual"
		case PreserveAutoCommand:
			preserve = "auto"
		}
	}
	if hasPlankit {
		if guard == "" {
			guard = "off"
		}
		if preserve == "" {
			preserve = "off"
		}
	}
	return guard, preserve
}

// InferModes reads hook commands from a parsed settings object and returns the
// current guard and preserve modes. Returns ("", "") when modes cannot be
// inferred (no hooks, no pk commands, or malformed JSON).
func InferModes(settings *OrderedObject) (guard, preserve string) {
	hooksRaw, ok := settings.Get("hooks")
	if !ok {
		return "", ""
	}
	var hooks HooksConfig
	if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
		return "", ""
	}
	var commands []string
	for _, entries := range [][]HookEntry{hooks.PreToolUse, hooks.PostToolUse, hooks.SessionStart} {
		for _, entry := range entries {
			for _, h := range entry.Hooks {
				commands = append(commands, HookCommand(h))
			}
		}
	}
	return InferModesFromCommands(commands)
}

// InferModesFromSettings reads .claude/settings.json under dir and returns the
// guard and preserve modes inferred from its hook commands. Returns ("", "")
// when the file is missing, unreadable, malformed, or has no inferable pk hooks.
func InferModesFromSettings(readFile func(string) ([]byte, error), dir string) (guard, preserve string) {
	data, err := readFile(filepath.Join(dir, paths.ClaudeDir, paths.SettingsFile))
	if err != nil {
		return "", ""
	}
	parsed, err := ParseOrderedObject(data)
	if err != nil {
		return "", ""
	}
	return InferModes(parsed)
}

// InferPushGuardFromCommands returns the push-guard mode ("block"/"ask") parsed from a
// guard hook command's `--push-guard <mode>` flag, or "" if absent.
func InferPushGuardFromCommands(commands []string) string {
	for _, cmd := range commands {
		if !strings.HasPrefix(cmd, GuardBlockCommand) {
			continue
		}
		const flag = "--push-guard "
		if i := strings.Index(cmd, flag); i >= 0 {
			if fields := strings.Fields(cmd[i+len(flag):]); len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

// InferPushGuardFromSettings reads .claude/settings.json under dir and returns the
// push-guard mode from its guard hook command, or "" when absent/unreadable.
func InferPushGuardFromSettings(readFile func(string) ([]byte, error), dir string) string {
	data, err := readFile(filepath.Join(dir, paths.ClaudeDir, paths.SettingsFile))
	if err != nil {
		return ""
	}
	parsed, err := ParseOrderedObject(data)
	if err != nil {
		return ""
	}
	hooksRaw, ok := parsed.Get("hooks")
	if !ok {
		return ""
	}
	var hooks HooksConfig
	if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
		return ""
	}
	var commands []string
	for _, entry := range hooks.PreToolUse {
		for _, h := range entry.Hooks {
			commands = append(commands, HookCommand(h))
		}
	}
	return InferPushGuardFromCommands(commands)
}

// writeInstallScript writes the cloud-sandbox bootstrap script to .claude/install-pk.sh.
// The script is template-substituted with the running pk version and written with 0755
// permissions. For development builds (version "dev"), the script is skipped.
// Returns (changed, error). changed is true only when the bytes actually written differ from what was on disk.
func writeInstallScript(cfg Config, projectDir string, pkVersion string) (bool, error) {
	if version.IsDevBuild(pkVersion) {
		fmt.Fprintln(cfg.Stderr, "  install-pk.sh: skipped (development build)")
		return false, nil
	}
	if !strings.HasPrefix(pkVersion, "v") {
		pkVersion = "v" + pkVersion
	}
	content := strings.Replace(installScriptTemplate, "{{VERSION}}", pkVersion, 1)
	scriptPath := filepath.Join(projectDir, paths.ClaudeDir, paths.InstallScript)

	existing, _ := cfg.ReadFile(scriptPath)

	if err := cfg.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		return false, fmt.Errorf("failed to create directory for %s: %w", scriptPath, err)
	}
	// WriteFile only applies the mode on creation, not when overwriting.
	cfg.Remove(scriptPath)
	if err := cfg.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", scriptPath, err)
	}
	fmt.Fprintf(cfg.Stderr, "  install-pk.sh: updated (pinned %s)\n", pkVersion)
	return string(existing) != content, nil
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
