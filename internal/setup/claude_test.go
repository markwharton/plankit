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
	hooks := buildHookConfig("manual", "block")

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
	hooks := buildHookConfig("manual", "block")

	if err := mergeHooks(settings, hooks); err != nil {
		t.Fatalf("mergeHooks() error = %v", err)
	}

	raw, _ := settings.Get("hooks")
	var result HooksConfig
	json.Unmarshal(raw, &result)

	// User's Bash hook + plankit's Bash/guard, Edit/protect, Write/protect.
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
	hooks := buildHookConfig("manual", "block")

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

	// PostToolUse should have the manual mode hook (--notify), not the old auto one.
	if len(result.PostToolUse) != 1 {
		t.Fatalf("PostToolUse = %d entries, want 1", len(result.PostToolUse))
	}
	if cmd := HookCommand(result.PostToolUse[0].Hooks[0]); !strings.Contains(cmd, "--notify") {
		t.Errorf("PostToolUse command = %q, want --notify", cmd)
	}
}

func TestMergeHooks_mixedHooks(t *testing.T) {
	// An entry with both a plankit hook and a user hook on the same matcher.
	existing := `{"PreToolUse":[{"matcher":"Edit","hooks":[{"type":"command","command":"pk protect","timeout":5},{"type":"command","command":"my-linter","timeout":10}]}]}`
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(existing))
	hooks := buildHookConfig("manual", "block")

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
	hooks := buildHookConfig("manual", "block")

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

	hooks := buildHookConfig("manual", "block")
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

	hooks := buildHookConfig("manual", "block")
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

func TestInferModesFromCommands_blockAndManual(t *testing.T) {
	guard, preserve := InferModesFromCommands([]string{
		GuardBlockCommand, "pk protect", PreserveManualCommand,
	})
	if guard != "block" {
		t.Errorf("guard = %q, want %q", guard, "block")
	}
	if preserve != "manual" {
		t.Errorf("preserve = %q, want %q", preserve, "manual")
	}
}

func TestInferModesFromCommands_askAndAuto(t *testing.T) {
	guard, preserve := InferModesFromCommands([]string{
		GuardAskCommand, "pk protect", PreserveAutoCommand,
	})
	if guard != "ask" {
		t.Errorf("guard = %q, want %q", guard, "ask")
	}
	if preserve != "auto" {
		t.Errorf("preserve = %q, want %q", preserve, "auto")
	}
}

func TestInferModesFromCommands_empty(t *testing.T) {
	guard, preserve := InferModesFromCommands(nil)
	if guard != "" {
		t.Errorf("guard = %q, want empty", guard)
	}
	if preserve != "" {
		t.Errorf("preserve = %q, want empty", preserve)
	}
}

func TestInferModes_roundTrip(t *testing.T) {
	tests := []struct {
		name         string
		preserveMode string
		guardMode    string
		wantGuard    string
		wantPreserve string
	}{
		{"block and manual", "manual", "block", "block", "manual"},
		{"ask and auto", "auto", "ask", "ask", "auto"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := NewOrderedObject()
			hooks := buildHookConfig(tt.preserveMode, tt.guardMode)
			if err := mergeHooks(settings, hooks); err != nil {
				t.Fatalf("mergeHooks() error = %v", err)
			}
			guard, preserve := InferModes(settings)
			if guard != tt.wantGuard {
				t.Errorf("guard = %q, want %q", guard, tt.wantGuard)
			}
			if preserve != tt.wantPreserve {
				t.Errorf("preserve = %q, want %q", preserve, tt.wantPreserve)
			}
		})
	}
}

func TestInferModes_noHooks(t *testing.T) {
	settings := NewOrderedObject()
	guard, preserve := InferModes(settings)
	if guard != "" || preserve != "" {
		t.Errorf("expected empty for no hooks, got guard=%q preserve=%q", guard, preserve)
	}
}

func TestInferModes_corruptHooks(t *testing.T) {
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(`{invalid`))
	guard, preserve := InferModes(settings)
	if guard != "" || preserve != "" {
		t.Errorf("expected empty for corrupt hooks, got guard=%q preserve=%q", guard, preserve)
	}
}

func TestInferModes_userHooksOnly(t *testing.T) {
	settings := NewOrderedObject()
	settings.Set("hooks", json.RawMessage(`{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"my-custom-hook"}]}]}`))
	guard, preserve := InferModes(settings)
	if guard != "" || preserve != "" {
		t.Errorf("expected empty for user-only hooks, got guard=%q preserve=%q", guard, preserve)
	}
}
