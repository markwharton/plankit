package setup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeHooks_freshSettings(t *testing.T) {
	settings := NewOrderedObject()
	hooks := buildHooks()

	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}

	raw, _ := settings.Get("hooks")
	var result HooksConfig
	json.Unmarshal(raw, &result)

	if len(result.PreToolUse) != 3 {
		t.Errorf("PreToolUse = %d entries, want 3", len(result.PreToolUse))
	}
	if len(result.PostToolUse) != 1 {
		t.Errorf("PostToolUse = %d entries, want 1", len(result.PostToolUse))
	}
}

func TestMergeHooks_existingUserHooks(t *testing.T) {
	existing := `{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"my-checker","timeout":5}]}]}`
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(existing))
	hooks := buildHooks()

	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}

	raw, _ := settings.Get("hooks")
	var result HooksConfig
	json.Unmarshal(raw, &result)

	// User's Bash hook + plankit's Bash|PowerShell/guard, Edit/protect, Write/protect.
	if len(result.PreToolUse) != 4 {
		t.Errorf("PreToolUse = %d entries, want 4", len(result.PreToolUse))
	}

	// Verify user hook is first (preserved before plankit entries).
	if result.PreToolUse[0].Matcher != "Bash" {
		t.Errorf("first PreToolUse matcher = %q, want Bash", result.PreToolUse[0].Matcher)
	}
	if cmd := HookCommand(result.PreToolUse[0].Hooks[0]); cmd != "my-checker" {
		t.Errorf("first hook command = %q, want my-checker", cmd)
	}
}

func TestMergeHooks_existingPlankitHooks(t *testing.T) {
	// Simulate old plankit hooks (e.g., from a previous pk setup with auto mode).
	existing := `{"PreToolUse":[{"matcher":"Edit","hooks":[{"type":"command","command":"pk protect","timeout":5}]}],"PostToolUse":[{"matcher":"ExitPlanMode","hooks":[{"type":"command","command":"pk preserve","async":true,"timeout":60}]}]}`
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(existing))
	// Re-setup with manual mode — should replace old plankit hooks.
	hooks := buildHooks()

	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}

	raw, _ := settings.Get("hooks")
	var result HooksConfig
	json.Unmarshal(raw, &result)

	// Should have plankit's 3 PreToolUse entries (old one removed, new ones added).
	if len(result.PreToolUse) != 3 {
		t.Errorf("PreToolUse = %d entries, want 3", len(result.PreToolUse))
	}

	// PostToolUse should have the bare preserve hook (mode now lives in .pk.json).
	if len(result.PostToolUse) != 1 {
		t.Fatalf("PostToolUse = %d entries, want 1", len(result.PostToolUse))
	}
	if cmd := HookCommand(result.PostToolUse[0].Hooks[0]); cmd != "pk preserve" {
		t.Errorf("PostToolUse command = %q, want %q", cmd, "pk preserve")
	}
}

func TestMergeHooks_mixedHooks(t *testing.T) {
	// An entry with both a plankit hook and a user hook on the same matcher.
	existing := `{"PreToolUse":[{"matcher":"Edit","hooks":[{"type":"command","command":"pk protect","timeout":5},{"type":"command","command":"my-linter","timeout":10}]}]}`
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(existing))
	hooks := buildHooks()

	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}

	raw, _ := settings.Get("hooks")
	var result HooksConfig
	json.Unmarshal(raw, &result)

	// Should have: user's Edit entry (my-linter only) + plankit's Bash/guard + Edit/protect + Write/protect = 4.
	if len(result.PreToolUse) != 4 {
		t.Errorf("PreToolUse = %d entries, want 4", len(result.PreToolUse))
	}

	// First entry should be the user's surviving hook.
	if result.PreToolUse[0].Matcher != "Edit" {
		t.Errorf("first matcher = %q, want Edit", result.PreToolUse[0].Matcher)
	}
	if len(result.PreToolUse[0].Hooks) != 1 {
		t.Fatalf("first entry hooks = %d, want 1", len(result.PreToolUse[0].Hooks))
	}
	if cmd := HookCommand(result.PreToolUse[0].Hooks[0]); cmd != "my-linter" {
		t.Errorf("first hook command = %q, want my-linter", cmd)
	}
}

func TestMergeHooks_existingSessionStart(t *testing.T) {
	existing := `{"SessionStart":[{"matcher":"*","hooks":[{"type":"command","command":".claude/install-pk.sh","timeout":30}]}]}`
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(existing))
	hooks := buildHooks()

	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}

	raw, _ := settings.Get("hooks")
	var result HooksConfig
	json.Unmarshal(raw, &result)

	if len(result.SessionStart) != 1 {
		t.Errorf("SessionStart = %d entries, want 1 (no duplicate)", len(result.SessionStart))
	}
}

// TestMergeHooks_preservesUnknownCategories verifies that hook categories pk
// doesn't manage (SessionEnd, Stop, UserPromptSubmit, etc.) pass through
// untouched when pk hooks are merged.
func TestMergeHooks_preservesUnknownCategories(t *testing.T) {
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(`{
			"SessionEnd": [{"matcher":"","hooks":[{"type":"command","command":"entire hooks claude-code session-end"}]}],
			"Stop": [{"matcher":"","hooks":[{"type":"command","command":"entire hooks claude-code stop"}]}],
			"UserPromptSubmit": [{"matcher":"","hooks":[{"type":"command","command":"entire hooks claude-code user-prompt-submit"}]}]
		}`))

	hooks := buildHooks()
	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}

	hooksRaw, _ := settings.Get("hooks")
	merged, err := ParseOrderedObject(hooksRaw)
	if err != nil {
		t.Fatalf("ParseOrderedObject() error = %v", err)
	}

	for _, key := range []string{"SessionEnd", "Stop", "UserPromptSubmit"} {
		raw, ok := merged.Get(key)
		if !ok {
			t.Errorf("category %s was dropped — must be preserved", key)
			continue
		}
		var entries []HookEntry
		if err := json.Unmarshal(raw, &entries); err != nil {
			t.Errorf("category %s malformed after merge: %v", key, err)
			continue
		}
		if len(entries) != 1 {
			t.Errorf("category %s: expected 1 entry, got %d", key, len(entries))
			continue
		}
		cmd := HookCommand(entries[0].Hooks[0])
		if !strings.Contains(cmd, "entire") {
			t.Errorf("category %s: command mangled, got %q", key, cmd)
		}
	}
}

// TestMergeHooks_noTimeoutZero verifies that user hooks without a timeout
// don't get "timeout": 0 stamped on them after merging.
func TestMergeHooks_noTimeoutZero(t *testing.T) {
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(`{
			"PostToolUse": [{"matcher":"Task","hooks":[{"type":"command","command":"user-hook"}]}]
		}`))

	hooks := buildHooks()
	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}

	// The user hook should NOT have "timeout": 0 in the serialized output.
	hooksRaw, _ := settings.Get("hooks")
	hooksJSON := string(hooksRaw)
	if strings.Contains(hooksJSON, `"command":"user-hook","timeout":0`) {
		t.Errorf("timeout: 0 was added to user hook; JSON:\n%s", hooksJSON)
	}
	// Sanity: pk hooks DO have timeouts set, those should remain.
	if !strings.Contains(hooksJSON, `"timeout":5`) {
		t.Errorf("pk hooks lost their timeout; JSON:\n%s", hooksJSON)
	}
}

func TestWriteInstallScript_releaseVersion(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	wsCfg := Config{Stderr: &stderr}
	withFS(&wsCfg)
	if _, err := writeInstallScript(wsCfg, projectDir, "0.7.1"); err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}

	scriptPath := filepath.Join(projectDir, ".claude", "install-pk.sh")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("install-pk.sh not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `PK_VERSION="v0.7.1"`) {
		t.Errorf("script should contain PK_VERSION=\"v0.7.1\", got: %s", content)
	}
	if strings.Contains(content, "{{VERSION}}") {
		t.Error("script still contains template placeholder")
	}

	// Verify executable permissions.
	info, _ := os.Stat(scriptPath)
	if info.Mode().Perm()&0111 == 0 {
		t.Error("install-pk.sh should be executable")
	}

	if !strings.Contains(stderr.String(), "pinned v0.7.1") {
		t.Errorf("stderr = %q, want pinned version mentioned", stderr.String())
	}
}

func TestWriteInstallScript_vPrefixedVersion(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	wsCfg := Config{Stderr: &stderr}
	withFS(&wsCfg)
	if _, err := writeInstallScript(wsCfg, projectDir, "v0.8.0"); err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}

	scriptPath := filepath.Join(projectDir, ".claude", "install-pk.sh")
	data, _ := os.ReadFile(scriptPath)
	if !strings.Contains(string(data), `PK_VERSION="v0.8.0"`) {
		t.Errorf("script should not double-prefix v, got: %s", string(data))
	}
}

func TestWriteInstallScript_devBuild(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	wsCfg := Config{Stderr: &stderr}
	withFS(&wsCfg)
	if _, err := writeInstallScript(wsCfg, projectDir, "dev"); err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}

	scriptPath := filepath.Join(projectDir, ".claude", "install-pk.sh")
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Error("install-pk.sh should not be created for dev builds")
	}

	if !strings.Contains(stderr.String(), "skipped") {
		t.Errorf("stderr = %q, want 'skipped' message for dev build", stderr.String())
	}
}

func TestWriteInstallScript_emptyVersion(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	wsCfg := Config{Stderr: &stderr}
	withFS(&wsCfg)
	if _, err := writeInstallScript(wsCfg, projectDir, ""); err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}

	scriptPath := filepath.Join(projectDir, ".claude", "install-pk.sh")
	if _, err := os.Stat(scriptPath); !os.IsNotExist(err) {
		t.Error("install-pk.sh should not be created for empty version")
	}
}

func TestWriteInstallScript_idempotent(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	wsCfg := Config{Stderr: &stderr}
	withFS(&wsCfg)
	writeInstallScript(wsCfg, projectDir, "0.7.0")
	stderr.Reset()
	writeInstallScript(wsCfg, projectDir, "0.7.1")

	scriptPath := filepath.Join(projectDir, ".claude", "install-pk.sh")
	data, _ := os.ReadFile(scriptPath)
	if !strings.Contains(string(data), `PK_VERSION="v0.7.1"`) {
		t.Error("re-run should update pinned version to v0.7.1")
	}
}

func TestWriteInstallScript_versionIsolatedPath(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	wsCfg := Config{Stderr: &stderr}
	withFS(&wsCfg)
	if _, err := writeInstallScript(wsCfg, projectDir, "0.14.1"); err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}

	scriptPath := filepath.Join(projectDir, ".claude", "install-pk.sh")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("install-pk.sh not created: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `install_dir="$HOME/.local/share/pk/$PK_VERSION"`) {
		t.Errorf("script should install to per-version path under $HOME/.local/share/pk, got: %s", content)
	}
	// Cloud sandbox base images plant a stale pk at the legacy $HOME/.local/bin/pk
	// path on every restart. The script must clear it before the presence gate so
	// the per-version cache can take over.
	if !strings.Contains(content, `rm -f "$HOME/.local/bin/pk"`) {
		t.Errorf("script must clear legacy $HOME/.local/bin/pk on each run, got: %s", content)
	}
	// The new template must never WRITE pk into the legacy directory — guard
	// against the old install_dir assignment and any redirect target sneaking
	// back in.
	if strings.Contains(content, `install_dir="$HOME/.local/bin"`) {
		t.Errorf("script must not assign install_dir to $HOME/.local/bin (legacy shared path), got: %s", content)
	}
	if strings.Contains(content, `>> "$HOME/.local/bin`) || strings.Contains(content, `> "$HOME/.local/bin`) {
		t.Errorf("script must not redirect into $HOME/.local/bin, got: %s", content)
	}
}

func TestWriteInstallScript_fixesPermissions(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	claudeDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	scriptPath := filepath.Join(claudeDir, "install-pk.sh")
	os.WriteFile(scriptPath, []byte("#!/bin/sh\n# stale"), 0644)

	wsCfg := Config{Stderr: &stderr}
	withFS(&wsCfg)
	if _, err := writeInstallScript(wsCfg, projectDir, "0.9.0"); err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}

	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Error("install-pk.sh should be executable after overwriting a 0644 file")
	}
}

func TestInferModesFromCommands_blockAndManual(t *testing.T) {
	m := InferModesFromCommands([]string{
		GuardBlockCommand, "pk protect", PreserveManualCommand,
	})
	if m.Guard != "block" {
		t.Errorf("guard = %q, want %q", m.Guard, "block")
	}
	if m.Preserve != "manual" {
		t.Errorf("preserve = %q, want %q", m.Preserve, "manual")
	}
}

func TestInferModesFromCommands_askAndAuto(t *testing.T) {
	m := InferModesFromCommands([]string{
		GuardAskCommand, "pk protect", PreserveAutoCommand,
	})
	if m.Guard != "ask" {
		t.Errorf("guard = %q, want %q", m.Guard, "ask")
	}
	if m.Preserve != "auto" {
		t.Errorf("preserve = %q, want %q", m.Preserve, "auto")
	}
}

func TestInferModesFromCommands_empty(t *testing.T) {
	m := InferModesFromCommands(nil)
	if m.Guard != "" {
		t.Errorf("guard = %q, want empty", m.Guard)
	}
	if m.Preserve != "" {
		t.Errorf("preserve = %q, want empty", m.Preserve)
	}
}

// oldStyleSettings builds a settings.json body in the PRE-migration format,
// where guard/preserve modes were encoded in the hook commands (off = the hook
// or flag is omitted). Used to test InferModes* as the migration reader.
func oldStyleSettings(t *testing.T, preserveMode, guardMode, pushGuard string) []byte {
	t.Helper()
	entry := func(matcher, command string) map[string]any {
		return map[string]any{
			"matcher": matcher,
			"hooks":   []any{map[string]any{"type": "command", "command": command}},
		}
	}
	var pre []any
	if guardMode != "off" {
		cmd := "pk guard"
		if guardMode == "ask" {
			cmd = "pk guard --ask"
		}
		if pushGuard != "" && pushGuard != "off" {
			cmd += " --push-guard " + pushGuard
		}
		pre = append(pre, entry("Bash|PowerShell", cmd))
	}
	pre = append(pre, entry("Edit", "pk protect"))
	var post []any
	switch preserveMode {
	case "auto":
		post = append(post, entry("ExitPlanMode", "pk preserve"))
	case "manual":
		post = append(post, entry("ExitPlanMode", "pk preserve --notify"))
	}
	settings := map[string]any{"hooks": map[string]any{"PreToolUse": pre, "PostToolUse": post}}
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal old-style settings: %v", err)
	}
	return data
}

// TestInferModes_roundTrip checks the migration reader recovers the modes from
// old-style (flag-bearing) hooks, including the absent-push-flag → "off" decode.
func TestInferModes_roundTrip(t *testing.T) {
	tests := []struct {
		name                               string
		preserveMode, guardMode, pushGuard string
		wantGuard, wantPush, wantPreserve  string
	}{
		{"block, manual, no push flag", "manual", "block", "", "block", "off", "manual"},
		{"ask, auto, push block", "auto", "ask", "block", "ask", "block", "auto"},
		{"guard off", "manual", "off", "", "off", "", "manual"},
		{"preserve off", "off", "block", "", "block", "off", "off"},
		{"both off", "off", "off", "", "off", "", "off"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings, err := ParseOrderedObject(oldStyleSettings(t, tt.preserveMode, tt.guardMode, tt.pushGuard))
			if err != nil {
				t.Fatalf("ParseOrderedObject() error = %v", err)
			}
			m := InferModes(settings)
			if m.Guard != tt.wantGuard {
				t.Errorf("guard = %q, want %q", m.Guard, tt.wantGuard)
			}
			if m.PushGuard != tt.wantPush {
				t.Errorf("push = %q, want %q", m.PushGuard, tt.wantPush)
			}
			if m.Preserve != tt.wantPreserve {
				t.Errorf("preserve = %q, want %q", m.Preserve, tt.wantPreserve)
			}
		})
	}
}

func TestInferModes_noHooks(t *testing.T) {
	settings := NewOrderedObject()
	m := InferModes(settings)
	if m.Guard != "" || m.Preserve != "" {
		t.Errorf("expected empty for no hooks, got guard=%q preserve=%q", m.Guard, m.Preserve)
	}
}

func TestInferModes_corruptHooks(t *testing.T) {
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(`{invalid`))
	m := InferModes(settings)
	if m.Guard != "" || m.Preserve != "" {
		t.Errorf("expected empty for corrupt hooks, got guard=%q preserve=%q", m.Guard, m.Preserve)
	}
}

func TestInferModes_userHooksOnly(t *testing.T) {
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(`{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"my-custom-hook"}]}]}`))
	m := InferModes(settings)
	if m.Guard != "" || m.Preserve != "" {
		t.Errorf("expected empty for user-only hooks, got guard=%q preserve=%q", m.Guard, m.Preserve)
	}
}

func TestInferModesFromSettings(t *testing.T) {
	t.Run("missing file returns empty", func(t *testing.T) {
		readFile := func(string) ([]byte, error) { return nil, os.ErrNotExist }
		m := InferModesFromSettings(readFile, "proj")
		if m.Guard != "" || m.Preserve != "" {
			t.Errorf("got guard=%q preserve=%q, want empty", m.Guard, m.Preserve)
		}
	})

	t.Run("malformed JSON returns empty", func(t *testing.T) {
		readFile := func(string) ([]byte, error) { return []byte("{not json"), nil }
		m := InferModesFromSettings(readFile, "proj")
		if m.Guard != "" || m.Preserve != "" {
			t.Errorf("got guard=%q preserve=%q, want empty", m.Guard, m.Preserve)
		}
	})

	t.Run("reads block/manual (push off) from settings path", func(t *testing.T) {
		data := oldStyleSettings(t, "manual", "block", "")
		wantPath := filepath.Join("proj", ".claude", "settings.json")
		readFile := func(path string) ([]byte, error) {
			if path != wantPath {
				t.Errorf("read path = %q, want %q", path, wantPath)
			}
			return data, nil
		}
		m := InferModesFromSettings(readFile, "proj")
		if m.Guard != "block" || m.PushGuard != "off" || m.Preserve != "manual" {
			t.Errorf("got guard=%q push=%q preserve=%q, want block/off/manual", m.Guard, m.PushGuard, m.Preserve)
		}
	})

	t.Run("guard command absent infers off", func(t *testing.T) {
		data := oldStyleSettings(t, "manual", "off", "")
		readFile := func(string) ([]byte, error) { return data, nil }
		m := InferModesFromSettings(readFile, "proj")
		if m.Guard != "off" || m.Preserve != "manual" {
			t.Errorf("got guard=%q preserve=%q, want off/manual", m.Guard, m.Preserve)
		}
	})
}

func TestPushGuardWiring(t *testing.T) {
	t.Run("guard mode still inferred when push flag present", func(t *testing.T) {
		if m := InferModesFromCommands([]string{"pk guard --ask --push-guard block", "pk protect"}); m.Guard != "ask" {
			t.Errorf("guard = %q, want ask (must parse despite --push-guard)", m.Guard)
		}
		if m := InferModesFromCommands([]string{"pk guard --push-guard block"}); m.Guard != "block" {
			t.Errorf("guard = %q, want block", m.Guard)
		}
	})

	t.Run("push-guard parsed, absent decodes to off", func(t *testing.T) {
		if m := InferModesFromCommands([]string{"pk guard --ask --push-guard block"}); m.PushGuard != "block" {
			t.Errorf("PushGuard = %q, want block", m.PushGuard)
		}
		// Guard present, no --push-guard flag: old encoding meant push off.
		if m := InferModesFromCommands([]string{"pk guard --ask"}); m.PushGuard != "off" {
			t.Errorf("PushGuard = %q, want off (absent flag decodes to off)", m.PushGuard)
		}
		// No guard command at all: push is moot, stays empty.
		if m := InferModesFromCommands([]string{"pk protect"}); m.PushGuard != "" {
			t.Errorf("PushGuard = %q, want empty (guard off, push moot)", m.PushGuard)
		}
	})

	t.Run("push-guard round-trips through old-style settings", func(t *testing.T) {
		readFile := func(string) ([]byte, error) { return oldStyleSettings(t, "manual", "ask", "block"), nil }
		if m := InferModesFromSettings(readFile, "proj"); m.PushGuard != "block" {
			t.Errorf("PushGuard = %q, want block", m.PushGuard)
		}
	})
}

// TestBuildHooks_static verifies the hook set is fixed and bare regardless of
// mode — guard, protect (Edit+Write), preserve, and the session bootstrap are
// always present, with no mode flags on any command (modes live in .pk.json).
func TestBuildHooks_static(t *testing.T) {
	hooks := buildHooks()
	if len(hooks.PreToolUse) != 3 {
		t.Fatalf("PreToolUse = %d entries, want 3 (guard + Edit/protect + Write/protect)", len(hooks.PreToolUse))
	}
	if len(hooks.PostToolUse) != 1 {
		t.Errorf("PostToolUse = %d entries, want 1 (preserve)", len(hooks.PostToolUse))
	}
	if len(hooks.SessionStart) != 1 {
		t.Errorf("SessionStart = %d entries, want 1 (install-pk.sh)", len(hooks.SessionStart))
	}
	wantCmds := map[string]string{
		"Bash|PowerShell": "pk guard",
		"ExitPlanMode":    "pk preserve",
	}
	for _, e := range append(append([]HookEntry{}, hooks.PreToolUse...), hooks.PostToolUse...) {
		cmd := HookCommand(e.Hooks[0])
		if strings.Contains(cmd, "--") {
			t.Errorf("hook command %q carries a flag; modes must live in .pk.json", cmd)
		}
		if want, ok := wantCmds[e.Matcher]; ok && cmd != want {
			t.Errorf("matcher %q command = %q, want %q", e.Matcher, cmd, want)
		}
	}
}

func TestInferModesFromCommands_protectOnly(t *testing.T) {
	m := InferModesFromCommands([]string{"pk protect"})
	if m.Guard != "off" {
		t.Errorf("guard = %q, want %q", m.Guard, "off")
	}
	if m.Preserve != "off" {
		t.Errorf("preserve = %q, want %q", m.Preserve, "off")
	}
}

func TestBuildHookConfig_shellBash(t *testing.T) {
	hooks := buildHooks()
	settings := NewOrderedObject()
	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}
	raw, _ := settings.Get("hooks")
	hooksJSON := string(raw)
	if !strings.Contains(hooksJSON, `"shell":"bash"`) {
		t.Errorf("pk hooks missing shell:bash; JSON:\n%s", hooksJSON)
	}
}

func TestBuildHookConfig_guardMatcherIncludesPowerShell(t *testing.T) {
	hooks := buildHooks()
	if hooks.PreToolUse[0].Matcher != "Bash|PowerShell" {
		t.Errorf("guard matcher = %q, want %q", hooks.PreToolUse[0].Matcher, "Bash|PowerShell")
	}
}

func TestMergeHooks_noShellOnUserHooks(t *testing.T) {
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(`{
			"PostToolUse": [{"matcher":"Task","hooks":[{"type":"command","command":"user-hook"}]}]
		}`))
	hooks := buildHooks()
	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}
	hooksRaw, _ := settings.Get("hooks")
	hooksJSON := string(hooksRaw)
	if strings.Contains(hooksJSON, `"command":"user-hook","shell"`) {
		t.Errorf("shell was added to user hook; JSON:\n%s", hooksJSON)
	}
}

func TestWriteInstallScript_reportsUnchangedOnRerun(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr}
	withFS(&cfg)

	changed, err := writeInstallScript(cfg, dir, "v1.2.3")
	if err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}
	if !changed {
		t.Error("first write reported no change")
	}
	if !strings.Contains(stderr.String(), "updated (pinned v1.2.3)") {
		t.Errorf("first write said %q", strings.TrimSpace(stderr.String()))
	}

	stderr.Reset()
	changed, err = writeInstallScript(cfg, dir, "v1.2.3")
	if err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}
	if changed {
		t.Error("re-run at the same version reported a change")
	}
	if !strings.Contains(stderr.String(), "unchanged (pinned v1.2.3)") {
		t.Errorf("re-run said %q, want unchanged", strings.TrimSpace(stderr.String()))
	}
}

func TestWriteInstallScript_reportsUpdatedOnVersionBump(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr}
	withFS(&cfg)

	if _, err := writeInstallScript(cfg, dir, "v1.2.3"); err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}
	stderr.Reset()

	changed, err := writeInstallScript(cfg, dir, "v1.3.0")
	if err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}
	if !changed {
		t.Error("version bump reported no change")
	}
	if !strings.Contains(stderr.String(), "updated (pinned v1.3.0)") {
		t.Errorf("said %q, want updated", strings.TrimSpace(stderr.String()))
	}
	data, err := os.ReadFile(filepath.Join(dir, ".claude", "install-pk.sh"))
	if err != nil {
		t.Fatalf("read script: %v", err)
	}
	if !strings.Contains(string(data), "v1.3.0") {
		t.Error("new version was not pinned into the script")
	}
	// The 0755 mode must survive the rewrite, or the SessionStart hook cannot run it.
	info, _ := os.Stat(filepath.Join(dir, ".claude", "install-pk.sh"))
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("script is not executable after rewrite: %v", info.Mode().Perm())
	}
}

func TestWriteInstallScript_repairsPermissionsOnIdenticalContent(t *testing.T) {
	// A clone or unzip can strip the executable bit while leaving the bytes
	// intact. The SessionStart hook runs this script, so setup must repair it
	// rather than reporting it as unchanged.
	dir := t.TempDir()
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr}
	withFS(&cfg)

	if _, err := writeInstallScript(cfg, dir, "v1.2.3"); err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}
	scriptPath := filepath.Join(dir, ".claude", "install-pk.sh")
	if err := os.Chmod(scriptPath, 0644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	stderr.Reset()

	changed, err := writeInstallScript(cfg, dir, "v1.2.3")
	if err != nil {
		t.Fatalf("writeInstallScript() error = %v", err)
	}
	if !changed {
		t.Error("permission repair reported no change")
	}
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("executable bit not repaired: %v", info.Mode().Perm())
	}
	if strings.Contains(stderr.String(), "unchanged") {
		t.Errorf("said %q, but the file needed repair", strings.TrimSpace(stderr.String()))
	}
}
