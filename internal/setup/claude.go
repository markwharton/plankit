package setup

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/msg"
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
		data, _ := MarshalNoHTML(h)
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

// buildHooks returns the static plankit hook set. Commands are bare — the
// guard/preserve modes live in .pk.json, not in the hook command — so the same
// wiring is written for every project regardless of mode. The bare hooks always
// run; each command resolves its own behavior (including "off") from .pk.json at
// runtime. The preserve entry is sync with a 30s timeout: committing one plan
// file is fast, and sync keeps the manual-mode notify message surfacing reliably.
func buildHooks() HooksConfig {
	return HooksConfig{
		PreToolUse: []HookEntry{
			NewHookEntry("Bash|PowerShell", Hook{Type: "command", Command: GuardBlockCommand, Shell: "bash", Timeout: 5}),
			NewHookEntry("Edit", Hook{Type: "command", Command: "pk protect", Shell: "bash", Timeout: 5}),
			NewHookEntry("Write", Hook{Type: "command", Command: "pk protect", Shell: "bash", Timeout: 5}),
		},
		PostToolUse: []HookEntry{
			NewHookEntry("ExitPlanMode", Hook{Type: "command", Command: PreserveAutoCommand, Shell: "bash", Timeout: 30, StatusMessage: "Preserving plan..."}),
		},
		SessionStart: []HookEntry{
			NewHookEntry("*", Hook{Type: "command", Command: paths.InstallScriptRel, Shell: "bash", Timeout: 30}),
		},
	}
}

// Modes is the set of plankit modes inferred from a project's hook commands.
// An empty Guard/Preserve means the mode is not inferable (no plankit hooks);
// "off" means plankit hooks exist but that mode's command is absent. PushGuard
// is only meaningful when Guard is "block" or "ask", and is "" when the guard
// command carries no --push-guard flag.
type Modes struct {
	Guard     string // "block" | "ask" | "off" | ""
	Preserve  string // "auto" | "manual" | "off" | ""
	PushGuard string // "block" | "ask" | ""
}

// InferModesFromCommands returns the modes inferred from a list of hook command
// strings. Returns a zero Modes when no plankit hooks are found (fresh project).
// Returns "off" for guard/preserve when plankit hooks exist but that mode's
// command is absent (explicitly disabled).
func InferModesFromCommands(commands []string) Modes {
	var m Modes
	hasPlankit := false
	for _, cmd := range commands {
		if IsPlankitHook(cmd) {
			hasPlankit = true
		}
		// Guard command may carry flags (--ask, --push-guard <mode>), so match by
		// prefix and read the branch mode from --ask rather than exact-matching.
		// push-guard rides on the same command, so read it here too.
		if strings.HasPrefix(cmd, GuardBlockCommand) {
			if strings.Contains(cmd, " --ask") {
				m.Guard = "ask"
			} else {
				m.Guard = "block"
			}
			if pg := parsePushGuard(cmd); pg != "" {
				m.PushGuard = pg
			}
		}
		switch cmd {
		case PreserveManualCommand:
			m.Preserve = "manual"
		case PreserveAutoCommand:
			m.Preserve = "auto"
		}
	}
	if hasPlankit {
		if m.Guard == "" {
			m.Guard = "off"
		}
		if m.Preserve == "" {
			m.Preserve = "off"
		}
		// Old encoding: a guard hook with no --push-guard flag meant push off.
		// Decode it as "off" (not "") so migration preserves the existing state
		// rather than falling through to the new block default.
		if (m.Guard == "block" || m.Guard == "ask") && m.PushGuard == "" {
			m.PushGuard = "off"
		}
	}
	return m
}

// parsePushGuard returns the push-guard mode from a guard command's
// `--push-guard <mode>` flag, or "" if absent.
func parsePushGuard(cmd string) string {
	const flag = "--push-guard "
	if i := strings.Index(cmd, flag); i >= 0 {
		if fields := strings.Fields(cmd[i+len(flag):]); len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

// InferModes reads hook commands from a parsed settings object and returns the
// inferred modes. Returns a zero Modes when modes cannot be inferred (no hooks,
// no pk commands, or malformed JSON).
func InferModes(settings *OrderedObject) Modes {
	hooksRaw, ok := settings.Get("hooks")
	if !ok {
		return Modes{}
	}
	var hooks HooksConfig
	if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
		return Modes{}
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
// modes inferred from its hook commands. Returns a zero Modes when the file is
// missing, unreadable, malformed, or has no inferable pk hooks.
func InferModesFromSettings(readFile func(string) ([]byte, error), dir string) Modes {
	data, err := readFile(filepath.Join(dir, paths.ClaudeDir, paths.SettingsFile))
	if err != nil {
		return Modes{}
	}
	parsed, err := ParseOrderedObject(data)
	if err != nil {
		return Modes{}
	}
	return InferModes(parsed)
}

// writeInstallScript writes the cloud-sandbox bootstrap script to .claude/install-pk.sh.
// The script is template-substituted with the running pk version and written with 0755
// permissions. For development builds (version "dev"), the script is skipped.
// Returns (changed, error). changed is true only when the bytes actually written differ from what was on disk.
func writeInstallScript(cfg Config, projectDir string, pkVersion string) (bool, error) {
	if version.IsDevBuild(pkVersion) {
		msg.Itemf(cfg.Stderr, "install-pk.sh: skipped (development build)")
		return false, nil
	}
	if !strings.HasPrefix(pkVersion, "v") {
		pkVersion = "v" + pkVersion
	}
	content := strings.Replace(installScriptTemplate, "{{VERSION}}", pkVersion, 1)
	scriptPath := filepath.Join(projectDir, paths.ClaudeDir, paths.InstallScript)

	// Already pinned to this version: say so and leave the file alone, matching
	// writeManaged. Rewriting identical bytes would only churn the mtime.
	//
	// The executable bit is part of "already correct": the SessionStart hook
	// runs this script, so a copy that lost the bit (a Windows checkout, a
	// core.fileMode=false clone, an unzip) must still be repaired by the
	// rewrite below rather than passed over as unchanged.
	if isUpToDate(cfg.ReadFile, scriptPath, []byte(content)) && isExecutable(cfg.Stat, scriptPath) {
		msg.Itemf(cfg.Stderr, "install-pk.sh: unchanged (pinned %s)", pkVersion)
		return false, nil
	}

	if err := cfg.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		return false, fmt.Errorf("failed to create directory for %s: %w", scriptPath, err)
	}
	// WriteFile only applies the mode on creation, not when overwriting.
	cfg.Remove(scriptPath)
	if err := cfg.WriteFile(scriptPath, []byte(content), 0755); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", scriptPath, err)
	}
	msg.Itemf(cfg.Stderr, "install-pk.sh: updated (pinned %s)", pkVersion)
	return true, nil
}

// addPermission adds a permission string to the settings "permissions.allow" list
// if it is not already present. Preserves existing key order in the permissions
// object (allow, deny, ask, and any future keys).
// PkPermission is the settings.json permission entry that allows pk commands
// to run without prompting. setup adds it; status and teardown detect it.
const PkPermission = "Bash(pk:*)"

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
	allowJSON, err := MarshalNoHTML(allowList)
	if err != nil {
		return err
	}
	perms.Set("allow", json.RawMessage(allowJSON))

	permsJSON, err := MarshalNoHTML(perms)
	if err != nil {
		return err
	}
	settings.Set("permissions", json.RawMessage(permsJSON))

	return nil
}

// writePkModes writes guard.mode / guard.push / preserve.mode into .pk.json,
// field-merging so existing keys (guard.branches, release, changelog) are
// preserved. Top-level keys are sorted alphabetically to match the conventions
// skill. .pk.json is user-owned, so no SHA marker is embedded. Returns whether
// the file's bytes changed.
func writePkModes(cfg Config, projectDir, guardMode, guardPush, preserveMode string) (bool, error) {
	path := filepath.Join(projectDir, paths.PkConfig)
	existing, readErr := cfg.ReadFile(path)
	pk := NewOrderedObject()
	if readErr == nil {
		parsed, err := ParseOrderedObject(existing)
		if err != nil {
			return false, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		pk = parsed
	}

	if err := setNested(pk, "guard", "mode", guardMode); err != nil {
		return false, err
	}
	if err := setNested(pk, "guard", "push", guardPush); err != nil {
		return false, err
	}
	if err := setNested(pk, "preserve", "mode", preserveMode); err != nil {
		return false, err
	}
	pk.SortKeys()

	output, err := MarshalIndentNoHTML(pk)
	if err != nil {
		return false, err
	}
	if isUpToDate(cfg.ReadFile, path, output) {
		return false, nil
	}
	if err := cfg.WriteFile(path, output, 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", path, err)
	}
	return true, nil
}

// setNested sets obj[field] = value (a JSON string) inside the nested object at
// pk[objKey], creating the nested object if absent and preserving its other
// fields and their order.
func setNested(pk *OrderedObject, objKey, field, value string) error {
	v, err := MarshalNoHTML(value)
	if err != nil {
		return err
	}
	return setNestedRaw(pk, objKey, field, json.RawMessage(v))
}

// hasNested reports whether pk[objKey][field] is present, so a caller can seed
// a value only when the project has not set one.
func hasNested(pk *OrderedObject, objKey, field string) bool {
	raw, ok := pk.Get(objKey)
	if !ok {
		return false
	}
	obj, err := ParseOrderedObject(raw)
	if err != nil {
		return false
	}
	return obj.Has(field)
}

// setNestedRaw is setNested for a value that is already JSON — an array or
// object, where setNested's string marshalling would be wrong.
func setNestedRaw(pk *OrderedObject, objKey, field string, value json.RawMessage) error {
	obj := NewOrderedObject()
	if raw, ok := pk.Get(objKey); ok {
		parsed, err := ParseOrderedObject(raw)
		if err != nil {
			return err
		}
		obj = parsed
	}
	obj.Set(field, value)
	objJSON, err := MarshalNoHTML(obj)
	if err != nil {
		return err
	}
	pk.Set(objKey, json.RawMessage(objJSON))
	return nil
}

// WriteTopology writes the branch topology into .pk.json: guard.branches
// naming the release branch, and release.branch naming it again. It
// field-merges like writePkModes, so an existing .pk.json keeps its other
// keys, and it is idempotent — an unchanged file is not rewritten.
//
// pk setup deliberately never writes these keys: it reconciles managed files
// and modes, and the topology is a project decision. pk init owns that
// decision, which is why the writer lives here but the caller does not.
// Returns whether the file's bytes changed.
func WriteTopology(cfg Config, projectDir, releaseBranch string) (bool, error) {
	path := filepath.Join(projectDir, paths.PkConfig)
	existing, readErr := cfg.ReadFile(path)
	pk := NewOrderedObject()
	if readErr == nil {
		parsed, err := ParseOrderedObject(existing)
		if err != nil {
			return false, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		pk = parsed
	}

	// Seed guard.branches only when the project has not set it. A project may
	// legitimately guard more than the release branch (["main", "production"]),
	// and replacing that list with a single entry would silently unguard the
	// rest. pk init fills in what is missing; it does not restate decisions the
	// project has already made.
	if !hasNested(pk, "guard", "branches") {
		branches, err := MarshalNoHTML([]string{releaseBranch})
		if err != nil {
			return false, err
		}
		if err := setNestedRaw(pk, "guard", "branches", json.RawMessage(branches)); err != nil {
			return false, err
		}
	}
	if err := setNested(pk, "release", "branch", releaseBranch); err != nil {
		return false, err
	}
	pk.SortKeys()

	output, err := MarshalIndentNoHTML(pk)
	if err != nil {
		return false, err
	}
	if isUpToDate(cfg.ReadFile, path, output) {
		return false, nil
	}
	if err := cfg.WriteFile(path, output, 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", path, err)
	}
	return true, nil
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
	hooksJSON, err := MarshalNoHTML(existing)
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
	mergedJSON, err := MarshalNoHTML(merged)
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
	return strings.HasPrefix(command, "pk ") || command == paths.InstallScriptRel
}
