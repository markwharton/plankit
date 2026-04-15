package status

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/setup"
)

// setupProject creates a fully-configured pk project in a temp dir.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(filepath.Join(claudeDir, "skills", "changelog"), 0755)
	os.MkdirAll(filepath.Join(claudeDir, "skills", "init"), 0755)
	os.MkdirAll(filepath.Join(claudeDir, "rules"), 0755)

	for _, name := range []string{"changelog", "init"} {
		body := "# " + name + " skill\n"
		sha := setup.ContentSHA(body)
		content := "---\nname: " + name + "\npk_sha256: " + sha + "\n---\n" + body
		os.WriteFile(filepath.Join(claudeDir, "skills", name, "SKILL.md"), []byte(content), 0644)
	}

	for _, name := range []string{"development-standards", "git-discipline"} {
		body := "# " + name + "\n"
		sha := setup.ContentSHA(body)
		content := "---\ndescription: " + name + "\npk_sha256: " + sha + "\n---\n" + body
		os.WriteFile(filepath.Join(claudeDir, "rules", name+".md"), []byte(content), 0644)
	}

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

	os.WriteFile(filepath.Join(claudeDir, "install-pk.sh"), []byte("#!/bin/bash\n"), 0755)

	claudeBody := "# CLAUDE.md\n"
	claudeSHA := setup.ContentSHA(claudeBody)
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"),
		[]byte("<!-- pk:sha256:"+claudeSHA+" -->\n"+claudeBody), 0644)

	return dir
}

func testConfig(dir string) (Config, *bytes.Buffer) {
	var stderr bytes.Buffer
	cfg := DefaultConfig()
	cfg.Stderr = &stderr
	cfg.ProjectDir = dir
	return cfg, &stderr
}

func TestRun_notConfigured(t *testing.T) {
	dir := t.TempDir()
	cfg, stderr := testConfig(dir)

	configured, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if configured {
		t.Error("expected configured=false for empty project")
	}
	output := stderr.String()
	if !strings.Contains(output, "plankit is not configured") {
		t.Errorf("missing 'not configured' message, got: %s", output)
	}
	if !strings.Contains(output, "pk setup") {
		t.Error("missing setup hint in not-configured output")
	}
}

func TestRun_fullyConfigured(t *testing.T) {
	dir := setupProject(t)
	cfg, stderr := testConfig(dir)

	configured, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !configured {
		t.Error("expected configured=true for full setup")
	}

	output := stderr.String()
	expected := []string{
		"plankit is configured",
		"Modes:",
		"guard:",
		"block",
		"preserve:",
		"manual",
		"Hooks:",
		"PreToolUse:",
		"pk guard",
		"pk protect",
		"PostToolUse:",
		"pk preserve --notify",
		"SessionStart:",
		".claude/install-pk.sh",
		"Managed files:",
		"CLAUDE.md",
		"pristine",
		".claude/rules/",
		".claude/skills/",
		"Permission:",
		"Bash(pk:*)",
		"allowed",
	}
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nFull output:\n%s", want, output)
		}
	}
}

func TestRun_modifiedFiles(t *testing.T) {
	dir := setupProject(t)

	// Modify a rule.
	rulePath := filepath.Join(dir, ".claude", "rules", "development-standards.md")
	data, _ := os.ReadFile(rulePath)
	os.WriteFile(rulePath, []byte(string(data)+"\n# User edits\n"), 0644)

	cfg, stderr := testConfig(dir)
	configured, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !configured {
		t.Error("expected configured=true")
	}

	output := stderr.String()
	if !strings.Contains(output, "1 modified") {
		t.Errorf("expected modified count, got: %s", output)
	}
	if !strings.Contains(output, "development-standards.md (modified by user)") {
		t.Errorf("expected modified file listed, got: %s", output)
	}
}

func TestRun_askGuardMode(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": "pk guard --ask", "timeout": 5}},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	cfg, stderr := testConfig(dir)
	if _, err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "guard:") || !strings.Contains(output, "ask") {
		t.Errorf("expected guard: ask mode, got: %s", output)
	}
}

func TestRun_autoPreserveMode(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PostToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "ExitPlanMode",
					"hooks":   []interface{}{map[string]interface{}{"type": "command", "command": "pk preserve", "timeout": 60}},
				},
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	cfg, stderr := testConfig(dir)
	if _, err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "preserve:") || !strings.Contains(output, "auto") {
		t.Errorf("expected preserve: auto mode, got: %s", output)
	}
}

func TestRun_userCreatedSkillIgnored(t *testing.T) {
	dir := setupProject(t)

	// Add a user-created skill without pk_sha256.
	userSkillDir := filepath.Join(dir, ".claude", "skills", "my-custom")
	os.MkdirAll(userSkillDir, 0755)
	os.WriteFile(filepath.Join(userSkillDir, "SKILL.md"), []byte("---\nname: my-custom\n---\n"), 0644)

	cfg, stderr := testConfig(dir)
	if _, err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stderr.String()
	if strings.Contains(output, "my-custom") {
		t.Error("user-created skill should not appear in status output")
	}
	// Should still show 2 skills (pristine).
	if !strings.Contains(output, "2 file(s)") {
		t.Errorf("expected 2 skills counted, got: %s", output)
	}
}

func TestRun_pkConfigDetails(t *testing.T) {
	dir := setupProject(t)
	pkJSON := `{
		"changelog": {
			"types": [{"type": "feat", "section": "Added"}, {"type": "fix", "section": "Fixed"}],
			"hooks": {"preCommit": "pk pin --file script.sh $VERSION"}
		},
		"guard": {"branches": ["main", "release"]},
		"release": {"branch": "main", "hooks": {"preRelease": "make test"}}
	}`
	os.WriteFile(filepath.Join(dir, ".pk.json"), []byte(pkJSON), 0644)

	cfg, stderr := testConfig(dir)
	if _, err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stderr.String()
	expected := []string{
		"Config (.pk.json):",
		"changelog.types:",
		"2 configured",
		"changelog.hooks:",
		"preCommit set",
		"release.branch:",
		"main",
		"release.hooks:",
		"preRelease set",
		"guard.branches:",
	}
	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nFull output:\n%s", want, output)
		}
	}
}

func TestRun_notGitRepo(t *testing.T) {
	dir := setupProject(t)
	// setupProject doesn't create .git — so this is a non-git dir with pk set up.

	cfg, stderr := testConfig(dir)
	if _, err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "not a git repository") {
		t.Errorf("expected git warning in configured output, got: %s", output)
	}
}

func TestRun_gitRepoDetected(t *testing.T) {
	dir := setupProject(t)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cfg, stderr := testConfig(dir)
	if _, err := Run(cfg); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := stderr.String()
	if strings.Contains(output, "not a git repository") {
		t.Errorf("unexpected git warning for git repo, got: %s", output)
	}
}

func TestRun_notConfiguredNotGit(t *testing.T) {
	dir := t.TempDir()
	// No .git, no pk setup.

	cfg, stderr := testConfig(dir)
	configured, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if configured {
		t.Error("expected configured=false")
	}

	output := stderr.String()
	if !strings.Contains(output, "not a git repository") {
		t.Errorf("expected git note, got: %s", output)
	}
}

func TestRun_brief_configured(t *testing.T) {
	dir := setupProject(t)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cfg, stderr := testConfig(dir)
	cfg.Brief = true
	configured, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !configured {
		t.Error("expected configured=true")
	}

	output := stderr.String()
	if !strings.Contains(output, "plankit: configured") {
		t.Errorf("expected brief configured output, got: %s", output)
	}
	if !strings.Contains(output, "guard=block") {
		t.Errorf("expected guard mode in brief, got: %s", output)
	}
	if !strings.Contains(output, "preserve=manual") {
		t.Errorf("expected preserve mode in brief, got: %s", output)
	}
	// Only one line.
	lines := strings.Count(strings.TrimSpace(output), "\n")
	if lines != 0 {
		t.Errorf("expected one-line brief output, got %d newlines:\n%s", lines, output)
	}
}

func TestRun_brief_notConfigured(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	cfg, stderr := testConfig(dir)
	cfg.Brief = true
	configured, _ := Run(cfg)
	if configured {
		t.Error("expected configured=false")
	}

	output := strings.TrimSpace(stderr.String())
	if output != "plankit: not configured" {
		t.Errorf("expected exact brief output, got: %q", output)
	}
}

func TestRun_brief_notConfiguredNotGit(t *testing.T) {
	dir := t.TempDir()

	cfg, stderr := testConfig(dir)
	cfg.Brief = true
	Run(cfg)

	output := strings.TrimSpace(stderr.String())
	if !strings.Contains(output, "not a git repository") {
		t.Errorf("expected git note in brief, got: %q", output)
	}
}

func TestRun_corruptSettingsJSON(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	os.MkdirAll(claudeDir, 0755)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte("{invalid"), 0644)

	cfg, _ := testConfig(dir)
	_, err := Run(cfg)
	if err == nil {
		t.Fatal("expected error for corrupt settings.json")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_partialSetup(t *testing.T) {
	// Settings.json has pk hooks but no managed files exist.
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
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0644)

	cfg, stderr := testConfig(dir)
	configured, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !configured {
		t.Error("expected configured=true (hooks present)")
	}

	output := stderr.String()
	if !strings.Contains(output, "Hooks:") {
		t.Error("expected hooks section")
	}
	if !strings.Contains(output, "pk guard") {
		t.Error("expected pk guard listed")
	}
}
