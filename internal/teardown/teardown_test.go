package teardown

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/setup"
)

// setupProject creates a minimal pk setup in a temp directory.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(filepath.Join(claudeDir, "skills", "changelog"), 0755)
	os.MkdirAll(filepath.Join(claudeDir, "skills", "init"), 0755)
	os.MkdirAll(filepath.Join(claudeDir, "skills", "preserve"), 0755)
	os.MkdirAll(filepath.Join(claudeDir, "skills", "release"), 0755)
	os.MkdirAll(filepath.Join(claudeDir, "rules"), 0755)

	// Write managed skills with pk_sha256 markers.
	for _, name := range []string{"changelog", "init", "preserve", "release"} {
		body := "# " + name + " skill\n"
		sha := setup.ContentSHA(body)
		content := "---\nname: " + name + "\npk_sha256: " + sha + "\n---\n" + body
		os.WriteFile(filepath.Join(claudeDir, "skills", name, "SKILL.md"), []byte(content), 0644)
	}

	// Write managed rules with pk_sha256 markers.
	for _, name := range []string{"development-standards", "git-discipline", "model-behavior", "plankit-tooling"} {
		body := "# " + name + "\n"
		sha := setup.ContentSHA(body)
		content := "---\ndescription: " + name + "\npk_sha256: " + sha + "\n---\n" + body
		os.WriteFile(filepath.Join(claudeDir, "rules", name+".md"), []byte(content), 0644)
	}

	// Write settings.json with hooks and permission.
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": "pk guard", "timeout": 5}},
				},
				map[string]interface{}{
					"matcher": "Edit",
					"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": "pk protect", "timeout": 5}},
				},
			},
			"PostToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "ExitPlanMode",
					"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": "pk preserve --notify", "timeout": 10}},
				},
			},
			"SessionStart": []interface{}{
				map[string]interface{}{
					"matcher": "*",
					"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": ".claude/install-pk.sh", "timeout": 30}},
				},
			},
		},
		"permissions": map[string]interface{}{
			"allow": []string{"Bash(pk:*)"},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	// Write install-pk.sh.
	os.WriteFile(filepath.Join(claudeDir, "install-pk.sh"), []byte("#!/bin/bash\n"), 0755)

	// Write settings.json.bak.
	os.WriteFile(filepath.Join(claudeDir, "settings.json.bak"), []byte("{}"), 0644)

	// Write CLAUDE.md with pk marker.
	claudeBody := "# CLAUDE.md\n"
	claudeSHA := setup.ContentSHA(claudeBody)
	claudeContent := "<!-- pk:sha256:" + claudeSHA + " -->\n" + claudeBody
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(claudeContent), 0644)

	return dir
}

func testConfig(dir string) (Config, *bytes.Buffer) {
	var stderr bytes.Buffer
	cfg := DefaultConfig()
	cfg.Stderr = &stderr
	cfg.ProjectDir = dir
	return cfg, &stderr
}

func TestRun_fullCycle(t *testing.T) {
	dir := setupProject(t)
	cfg, stderr := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	output := stderr.String()

	// Verify all pk files are gone.
	for _, name := range []string{"changelog", "init", "preserve", "release"} {
		path := filepath.Join(dir, ".claude", "skills", name, "SKILL.md")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("skill %s still exists", name)
		}
	}
	for _, name := range []string{"development-standards", "git-discipline", "model-behavior", "plankit-tooling"} {
		path := filepath.Join(dir, ".claude", "rules", name+".md")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("rule %s still exists", name)
		}
	}

	// CLAUDE.md should be removed (pristine).
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Error("CLAUDE.md still exists")
	}

	// install-pk.sh should be removed.
	if _, err := os.Stat(filepath.Join(dir, ".claude", "install-pk.sh")); !os.IsNotExist(err) {
		t.Error("install-pk.sh still exists")
	}

	// settings.json.bak should be removed.
	if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json.bak")); !os.IsNotExist(err) {
		t.Error("settings.json.bak still exists")
	}

	// settings.json should be removed (was all pk content).
	if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json")); !os.IsNotExist(err) {
		t.Error("settings.json still exists")
	}

	// .claude/ directory should be removed.
	if _, err := os.Stat(filepath.Join(dir, ".claude")); !os.IsNotExist(err) {
		t.Error(".claude directory still exists")
	}

	if !strings.Contains(output, "Restart Claude Code") {
		t.Error("missing restart message")
	}
}

func TestRun_previewOnly(t *testing.T) {
	dir := setupProject(t)
	cfg, stderr := testConfig(dir)
	// Confirm is false by default.

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown preview failed: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "Run with --confirm to apply") {
		t.Error("missing --confirm hint in preview output")
	}

	// Nothing should be changed.
	if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json")); err != nil {
		t.Error("settings.json was removed during preview")
	}
	for _, name := range []string{"changelog", "init", "preserve", "release"} {
		path := filepath.Join(dir, ".claude", "skills", name, "SKILL.md")
		if _, err := os.Stat(path); err != nil {
			t.Errorf("skill %s was removed during preview", name)
		}
	}
}

func TestRun_mixedHooks(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	// Settings with both user and pk hooks on same matcher.
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{"type": "command", "command": "pk guard", "timeout": 5},
						map[string]interface{}{"type": "command", "command": "my-linter", "timeout": 10},
					},
				},
			},
		},
		"permissions": map[string]interface{}{
			"allow": []string{"Bash(pk:*)", "Bash(make:*)"},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	cfg, _ := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	// Read back settings.json.
	result, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatal("settings.json was removed, expected it to remain with user hooks")
	}

	var parsed map[string]json.RawMessage
	json.Unmarshal(result, &parsed)

	// Hooks should remain with only user hook.
	var hooks setup.HooksConfig
	json.Unmarshal(parsed["hooks"], &hooks)
	if len(hooks.PreToolUse) != 1 {
		t.Fatalf("expected 1 PreToolUse entry, got %d", len(hooks.PreToolUse))
	}
	if len(hooks.PreToolUse[0].Hooks) != 1 {
		t.Fatalf("expected 1 hook in entry, got %d", len(hooks.PreToolUse[0].Hooks))
	}
	if hooks.PreToolUse[0].Hooks[0].Command != "my-linter" {
		t.Errorf("expected user hook my-linter, got %s", hooks.PreToolUse[0].Hooks[0].Command)
	}

	// Permission should keep Bash(make:*) only.
	var perms map[string]json.RawMessage
	json.Unmarshal(parsed["permissions"], &perms)
	var allowList []string
	json.Unmarshal(perms["allow"], &allowList)
	if len(allowList) != 1 || allowList[0] != "Bash(make:*)" {
		t.Errorf("expected [Bash(make:*)], got %v", allowList)
	}
}

func TestRun_modifiedSkill(t *testing.T) {
	dir := setupProject(t)

	// Modify a skill (break the SHA).
	skillFile := filepath.Join(dir, ".claude", "skills", "changelog", "SKILL.md")
	data, _ := os.ReadFile(skillFile)
	modified := string(data) + "\n# User customization\n"
	os.WriteFile(skillFile, []byte(modified), 0644)

	cfg, stderr := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	// Modified skill should still exist.
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		t.Error("modified skill was removed")
	}

	// Other skills should be removed.
	for _, name := range []string{"init", "preserve", "release"} {
		path := filepath.Join(dir, ".claude", "skills", name, "SKILL.md")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("pristine skill %s still exists", name)
		}
	}

	output := stderr.String()
	if !strings.Contains(output, "skipped (modified by user)") {
		t.Error("missing skip message for modified skill")
	}
	if !strings.Contains(output, "To remove manually") {
		t.Error("missing manual removal hint")
	}
}

func TestRun_userCreatedSkill(t *testing.T) {
	dir := setupProject(t)

	// Add a user-created skill (no pk_sha256 marker).
	userSkillDir := filepath.Join(dir, ".claude", "skills", "my-custom")
	os.MkdirAll(userSkillDir, 0755)
	os.WriteFile(filepath.Join(userSkillDir, "SKILL.md"), []byte("---\nname: my-custom\n---\n# My skill\n"), 0644)

	cfg, stderr := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	// User skill should still exist.
	if _, err := os.Stat(filepath.Join(userSkillDir, "SKILL.md")); os.IsNotExist(err) {
		t.Error("user-created skill was removed")
	}

	// skills/ directory should remain because user skill exists.
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills")); os.IsNotExist(err) {
		t.Error(".claude/skills/ was removed despite user skill existing")
	}

	output := stderr.String()
	if strings.Contains(output, "my-custom") {
		t.Error("user-created skill should not appear in output")
	}
}

func TestRun_modifiedClaudeMD(t *testing.T) {
	dir := setupProject(t)

	// Modify CLAUDE.md (break the SHA but keep the marker).
	claudeFile := filepath.Join(dir, "CLAUDE.md")
	data, _ := os.ReadFile(claudeFile)
	modified := string(data) + "\n## My project rules\n"
	os.WriteFile(claudeFile, []byte(modified), 0644)

	cfg, stderr := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	// CLAUDE.md should still exist.
	if _, err := os.Stat(claudeFile); os.IsNotExist(err) {
		t.Error("modified CLAUDE.md was removed")
	}

	output := stderr.String()
	if !strings.Contains(output, "CLAUDE.md") || !strings.Contains(output, "modified by user") {
		t.Error("missing skip message for modified CLAUDE.md")
	}
}

func TestRun_unmanagedClaudeMD(t *testing.T) {
	dir := setupProject(t)

	// Replace CLAUDE.md with one that has no pk marker.
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# My CLAUDE.md\n"), 0644)

	cfg, _ := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	// CLAUDE.md should still exist (no marker = not pk-managed).
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); os.IsNotExist(err) {
		t.Error("unmanaged CLAUDE.md was removed")
	}
}

func TestRun_noSettingsFile(t *testing.T) {
	dir := setupProject(t)
	os.Remove(filepath.Join(dir, ".claude", "settings.json"))

	cfg, _ := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	// Should still clean up other files.
	for _, name := range []string{"changelog", "init", "preserve", "release"} {
		path := filepath.Join(dir, ".claude", "skills", name, "SKILL.md")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("skill %s still exists", name)
		}
	}
}

func TestRun_corruptSettingsJSON(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{invalid json"), 0644)

	cfg, _ := testConfig(dir)
	cfg.Confirm = true

	err := Run(cfg)
	if err == nil {
		t.Fatal("expected error for corrupt settings.json")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_emptyProject(t *testing.T) {
	dir := t.TempDir()

	cfg, stderr := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	if !strings.Contains(stderr.String(), "No plankit artifacts found") {
		t.Error("expected 'No plankit artifacts found' message")
	}
}

func TestRun_permissionCleanup(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	// Only pk permission, no hooks.
	settings := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []string{"Bash(pk:*)"},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	cfg, _ := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	// settings.json should be removed (empty after removing permission).
	if _, err := os.Stat(filepath.Join(claudeDir, "settings.json")); !os.IsNotExist(err) {
		t.Error("settings.json still exists after removing only permission")
	}
}

func TestRun_settingsOtherKeys(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": "pk guard", "timeout": 5}},
				},
			},
		},
		"permissions": map[string]interface{}{
			"allow": []string{"Bash(pk:*)"},
		},
		"customKey": "preserve me",
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	cfg, _ := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	// settings.json should remain with customKey.
	result, err := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	if err != nil {
		t.Fatal("settings.json was removed despite having custom keys")
	}

	var parsed map[string]json.RawMessage
	json.Unmarshal(result, &parsed)
	if _, ok := parsed["customKey"]; !ok {
		t.Error("customKey was removed")
	}
	if _, ok := parsed["hooks"]; ok {
		t.Error("hooks key still present after teardown")
	}
	if _, ok := parsed["permissions"]; ok {
		t.Error("permissions key still present after teardown")
	}
}

func TestRun_idempotent(t *testing.T) {
	dir := setupProject(t)
	cfg, _ := testConfig(dir)
	cfg.Confirm = true

	// First run.
	if err := Run(cfg); err != nil {
		t.Fatalf("first teardown failed: %v", err)
	}

	// Second run — should succeed with "nothing to do".
	var stderr2 bytes.Buffer
	cfg.Stderr = &stderr2
	if err := Run(cfg); err != nil {
		t.Fatalf("second teardown failed: %v", err)
	}

	if !strings.Contains(stderr2.String(), "No plankit artifacts found") {
		t.Error("expected 'No plankit artifacts found' on second run")
	}
}

func TestRun_directoryCleanup(t *testing.T) {
	dir := setupProject(t)

	// Add a user-created skill so skills/ shouldn't be removed.
	userDir := filepath.Join(dir, ".claude", "skills", "my-custom")
	os.MkdirAll(userDir, 0755)
	os.WriteFile(filepath.Join(userDir, "SKILL.md"), []byte("---\nname: custom\n---\n"), 0644)

	cfg, _ := testConfig(dir)
	cfg.Confirm = true

	if err := Run(cfg); err != nil {
		t.Fatalf("teardown failed: %v", err)
	}

	// .claude/skills/ should remain (user skill exists).
	if _, err := os.Stat(filepath.Join(dir, ".claude", "skills")); os.IsNotExist(err) {
		t.Error(".claude/skills/ was removed despite user skill")
	}

	// .claude/skills/my-custom/ should remain.
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		t.Error("user skill directory was removed")
	}

	// pk skill dirs should be removed.
	for _, name := range []string{"changelog", "init", "preserve", "release"} {
		skillDir := filepath.Join(dir, ".claude", "skills", name)
		if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
			t.Errorf(".claude/skills/%s/ still exists", name)
		}
	}

	// .claude/rules/ should be removed (all pk rules gone).
	if _, err := os.Stat(filepath.Join(dir, ".claude", "rules")); !os.IsNotExist(err) {
		t.Error(".claude/rules/ still exists")
	}
}

// TestRemoveHooks_preservesUnknownCategories verifies teardown doesn't drop
// hook categories outside the three pk manages (PreToolUse/PostToolUse/SessionStart).
func TestRemoveHooks_preservesUnknownCategories(t *testing.T) {
	settings := setup.NewOrderedObject()
	settings.Set("hooks", json.RawMessage(`{
			"PreToolUse": [{"matcher":"Bash","hooks":[{"type":"command","command":"pk guard","timeout":5}]}],
			"SessionEnd": [{"matcher":"","hooks":[{"type":"command","command":"entire hooks claude-code session-end"}]}],
			"Stop": [{"matcher":"","hooks":[{"type":"command","command":"entire hooks claude-code stop"}]}]
		}`))

	removeHooks(settings)

	raw, ok := settings.Get("hooks")
	if !ok {
		t.Fatal("hooks key removed but SessionEnd and Stop should have kept it alive")
	}

	hooks, err := setup.ParseOrderedObject(raw)
	if err != nil {
		t.Fatalf("ParseOrderedObject() error = %v", err)
	}

	if hooks.Has("PreToolUse") {
		t.Error("PreToolUse should be removed (only had a pk hook)")
	}
	if !hooks.Has("SessionEnd") {
		t.Error("SessionEnd was dropped by teardown — must be preserved")
	}
	if !hooks.Has("Stop") {
		t.Error("Stop was dropped by teardown — must be preserved")
	}
}

func TestRemoveHooks_emptyAfterRemoval(t *testing.T) {
	settings := setup.NewOrderedObject()
	settings.Set("hooks", json.RawMessage(`{
			"PreToolUse": [{"matcher":"Bash","hooks":[{"type":"command","command":"pk guard","timeout":5}]}]
		}`))

	removeHooks(settings)

	if settings.Has("hooks") {
		t.Error("hooks key should be removed when all hooks are pk-owned")
	}
}

func TestRemovePermission_keepOthers(t *testing.T) {
	settings := setup.NewOrderedObject()
	settings.Set("permissions", json.RawMessage(`{"allow":["Bash(pk:*)","Bash(make:*)"]}`))

	removePermission(settings, "Bash(pk:*)")

	permsRaw, _ := settings.Get("permissions")
	perms, err := setup.ParseOrderedObject(permsRaw)
	if err != nil {
		t.Fatalf("ParseOrderedObject() error = %v", err)
	}
	allowRaw, _ := perms.Get("allow")
	var allowList []string
	json.Unmarshal(allowRaw, &allowList)

	if len(allowList) != 1 || allowList[0] != "Bash(make:*)" {
		t.Errorf("expected [Bash(make:*)], got %v", allowList)
	}
}

func TestRemovePermission_emptyAfterRemoval(t *testing.T) {
	settings := setup.NewOrderedObject()
	settings.Set("permissions", json.RawMessage(`{"allow":["Bash(pk:*)"]}`))

	removePermission(settings, "Bash(pk:*)")

	if settings.Has("permissions") {
		t.Error("permissions key should be removed when empty")
	}
}
