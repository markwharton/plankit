package setup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withFS(cfg *Config) {
	cfg.ReadFile = os.ReadFile
	cfg.WriteFile = os.WriteFile
	cfg.Stat = os.Stat
	cfg.MkdirAll = os.MkdirAll
	cfg.ReadDir = os.ReadDir
	cfg.Remove = os.Remove
	cfg.Rename = os.Rename
	cfg.LookPath = func(string) (string, error) { return "/usr/local/bin/pk", nil }
}

func TestRun_freshProject(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	settingsFile := filepath.Join(projectDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}

	var settings map[string]json.RawMessage
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify hooks exist.
	if _, ok := settings["hooks"]; !ok {
		t.Fatal("missing hooks key")
	}

	var hooks map[string]interface{}
	json.Unmarshal(settings["hooks"], &hooks)

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok || len(preToolUse) != 3 {
		t.Fatalf("PreToolUse = %v, want 3 entries", hooks["PreToolUse"])
	}

	postToolUse, ok := hooks["PostToolUse"].([]interface{})
	if !ok || len(postToolUse) != 1 {
		t.Fatalf("PostToolUse = %v, want 1 entry", hooks["PostToolUse"])
	}

	// Verify permissions were added.
	if _, ok := settings["permissions"]; !ok {
		t.Fatal("missing permissions key")
	}
	var perms map[string]json.RawMessage
	json.Unmarshal(settings["permissions"], &perms)
	var allowList []string
	json.Unmarshal(perms["allow"], &allowList)
	found := false
	for _, p := range allowList {
		if p == "Bash(pk:*)" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("permissions.allow = %v, want to contain Bash(pk:*)", allowList)
	}

	// Verify skills were created with SHA markers.
	for _, name := range []string{"init", "preserve", "ship"} {
		skillFile := filepath.Join(projectDir, ".claude", "skills", name, "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			t.Fatalf("skill %s not created: %v", name, err)
		}
		content := string(data)
		if !strings.Contains(content, "pk_sha256: ") {
			t.Errorf("skill %s missing pk_sha256 in frontmatter", name)
		}
		if !strings.Contains(content, "name: "+name) {
			t.Errorf("skill %s = %q, want name in frontmatter", name, content)
		}
	}

	// Verify rules were created with SHA markers.
	for _, name := range []string{"model-behavior", "development-standards", "git-discipline", "plankit-tooling"} {
		ruleFile := filepath.Join(projectDir, ".claude", "rules", name+".md")
		data, err := os.ReadFile(ruleFile)
		if err != nil {
			t.Fatalf("rule %s not created: %v", name, err)
		}
		if !strings.Contains(string(data), "pk_sha256: ") {
			t.Errorf("rule %s missing pk_sha256 in frontmatter", name)
		}
	}

	// Verify CLAUDE.md was created.
	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	claudeData, err := os.ReadFile(claudeFile)
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	if !strings.Contains(string(claudeData), "## Critical Rules") {
		t.Error("CLAUDE.md missing Critical Rules section")
	}
	if !strings.Contains(string(claudeData), "<!-- pk:sha256:") {
		t.Error("CLAUDE.md missing SHA marker")
	}

	// Verify stderr output.
	if !strings.Contains(stderr.String(), "guard mode: block, preserve mode: auto") {
		t.Errorf("stderr = %q, want guard and preserve modes mentioned", stderr.String())
	}
}

func TestRun_createsClaudeMD(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	data, err := os.ReadFile(claudeFile)
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}

	content := string(data)
	// Should have SHA marker on first line (HTML comment for non-frontmatter files).
	if !strings.HasPrefix(content, "<!-- pk:sha256:") {
		t.Error("CLAUDE.md should start with SHA comment marker")
	}
	// Content should follow after the marker.
	if !strings.Contains(content, "# CLAUDE.md") {
		t.Error("CLAUDE.md missing heading")
	}
	// Should have critical rules section.
	if !strings.Contains(content, "## Critical Rules") {
		t.Error("CLAUDE.md missing Critical Rules section")
	}
	// Should be lean (no detailed sections — those are in .claude/rules/).
	if strings.Contains(content, "## Model Behavior") {
		t.Error("CLAUDE.md should not contain Model Behavior (moved to rules)")
	}
	if strings.Contains(content, "## Development Standards") {
		t.Error("CLAUDE.md should not contain Development Standards (moved to rules)")
	}
}

func TestRun_skipsUnmanagedClaudeMD(t *testing.T) {
	projectDir := t.TempDir()
	// Create a user's own CLAUDE.md without marker.
	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	os.WriteFile(claudeFile, []byte("# My Custom CLAUDE.md\n"), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, _ := os.ReadFile(claudeFile)
	if string(data) != "# My Custom CLAUDE.md\n" {
		t.Error("pk setup overwrote user's unmanaged CLAUDE.md")
	}
}

func TestRun_skipsModifiedClaudeMD(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	// First run creates CLAUDE.md with marker.
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	Run(cfg)

	// User modifies it but keeps the marker line at the top.
	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	data, _ := os.ReadFile(claudeFile)
	content := string(data)
	firstNewline := strings.IndexByte(content, '\n')
	markerLine := content[:firstNewline]
	modified := markerLine + "\n# User's custom content\n"
	os.WriteFile(claudeFile, []byte(modified), 0644)

	// Re-run — should skip.
	stderr.Reset()
	cfg = Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	Run(cfg)

	final, _ := os.ReadFile(claudeFile)
	if !strings.Contains(string(final), "User's custom content") {
		t.Error("pk setup overwrote user-modified CLAUDE.md")
	}
}

func TestRun_updatesUnmodifiedClaudeMD(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	// First run creates CLAUDE.md.
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	Run(cfg)

	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	original, _ := os.ReadFile(claudeFile)

	// Re-run — should update (content unchanged, SHA matches).
	stderr.Reset()
	cfg = Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	Run(cfg)

	updated, _ := os.ReadFile(claudeFile)
	// Content should be identical (same template, same SHA).
	if string(original) != string(updated) {
		t.Error("re-run changed CLAUDE.md content unexpectedly")
	}
}

func TestRun_existingSettings(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(settingsDir, 0755)

	existing := `{"customKey": "preserved"}`
	settingsFile := filepath.Join(settingsDir, "settings.json")
	os.WriteFile(settingsFile, []byte(existing), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, _ := os.ReadFile(settingsFile)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	// Custom key should be preserved.
	var customVal string
	json.Unmarshal(settings["customKey"], &customVal)
	if customVal != "preserved" {
		t.Errorf("customKey = %q, want 'preserved'", customVal)
	}

	// Hooks should be added.
	if _, ok := settings["hooks"]; !ok {
		t.Error("missing hooks key")
	}

	// Backup should exist.
	backupFile := settingsFile + ".bak"
	if _, err := os.Stat(backupFile); err != nil {
		t.Error("backup file not created")
	}
}

func TestRun_existingPermissions(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(settingsDir, 0755)

	existing := `{"permissions": {"allow": ["Bash(make:*)"]}}`
	settingsFile := filepath.Join(settingsDir, "settings.json")
	os.WriteFile(settingsFile, []byte(existing), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, _ := os.ReadFile(settingsFile)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var perms map[string]json.RawMessage
	json.Unmarshal(settings["permissions"], &perms)
	var allowList []string
	json.Unmarshal(perms["allow"], &allowList)

	if len(allowList) != 2 {
		t.Fatalf("allow list = %v, want 2 entries", allowList)
	}
	hasMake := false
	hasPK := false
	for _, p := range allowList {
		if p == "Bash(make:*)" {
			hasMake = true
		}
		if p == "Bash(pk:*)" {
			hasPK = true
		}
	}
	if !hasMake || !hasPK {
		t.Errorf("allow list = %v, want both Bash(make:*) and Bash(pk:*)", allowList)
	}
}

func TestRun_duplicatePermission(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(settingsDir, 0755)

	existing := `{"permissions": {"allow": ["Bash(pk:*)"]}}`
	settingsFile := filepath.Join(settingsDir, "settings.json")
	os.WriteFile(settingsFile, []byte(existing), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, _ := os.ReadFile(settingsFile)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var perms map[string]json.RawMessage
	json.Unmarshal(settings["permissions"], &perms)
	var allowList []string
	json.Unmarshal(perms["allow"], &allowList)

	if len(allowList) != 1 {
		t.Errorf("allow list = %v, want exactly 1 entry (no duplicate)", allowList)
	}
}

func TestRun_invalidJSON(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(settingsDir, 0755)

	settingsFile := filepath.Join(settingsDir, "settings.json")
	os.WriteFile(settingsFile, []byte("not json"), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	err := Run(cfg)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRun_refusesNonGitDir(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block"}
	withFS(&cfg)
	err := Run(cfg)
	if err == nil {
		t.Fatal("expected error for non-git directory without AllowNonGit")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error should mention git repository, got: %v", err)
	}

	// Nothing should have been created.
	if _, statErr := os.Stat(filepath.Join(projectDir, ".claude")); !os.IsNotExist(statErr) {
		t.Error(".claude should not exist after refused setup")
	}
	if _, statErr := os.Stat(filepath.Join(projectDir, "CLAUDE.md")); !os.IsNotExist(statErr) {
		t.Error("CLAUDE.md should not exist after refused setup")
	}
}

func TestRun_gitDirAllowsSetup(t *testing.T) {
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	var stderr bytes.Buffer

	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block"}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() with .git dir should succeed: %v", err)
	}
	// .claude should be created.
	if _, err := os.Stat(filepath.Join(projectDir, ".claude")); err != nil {
		t.Error(".claude should exist after successful setup")
	}
}

func TestRun_manualMode(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	settingsFile := filepath.Join(projectDir, ".claude", "settings.json")
	data, _ := os.ReadFile(settingsFile)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var hooks map[string]interface{}
	json.Unmarshal(settings["hooks"], &hooks)

	postToolUse, ok := hooks["PostToolUse"].([]interface{})
	if !ok || len(postToolUse) != 1 {
		t.Fatalf("PostToolUse = %v, want 1 entry", hooks["PostToolUse"])
	}

	// Verify the command uses --notify and is synchronous (no async field).
	hookData, _ := json.Marshal(postToolUse[0])
	if !strings.Contains(string(hookData), "--notify") {
		t.Errorf("PostToolUse hook = %s, want to contain --notify", string(hookData))
	}
	if strings.Contains(string(hookData), "async") {
		t.Errorf("PostToolUse hook = %s, manual mode should not be async", string(hookData))
	}

	if !strings.Contains(stderr.String(), "guard mode: block, preserve mode: manual") {
		t.Errorf("stderr = %q, want guard and preserve modes mentioned", stderr.String())
	}
}

func TestRun_existingHooks(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(settingsDir, 0755)

	existing := `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"my-checker","timeout":5}]}]}}`
	settingsFile := filepath.Join(settingsDir, "settings.json")
	os.WriteFile(settingsFile, []byte(existing), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, _ := os.ReadFile(settingsFile)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var hooks HooksConfig
	json.Unmarshal(settings["hooks"], &hooks)

	// User's Bash hook should survive.
	found := false
	for _, entry := range hooks.PreToolUse {
		if entry.Matcher == "Bash" {
			for _, h := range entry.Hooks {
				if HookCommand(h) == "my-checker" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Errorf("user's Bash hook lost after pk setup, PreToolUse = %+v", hooks.PreToolUse)
	}

	// Plankit hooks should also be present.
	hasProtect := false
	for _, entry := range hooks.PreToolUse {
		for _, h := range entry.Hooks {
			if HookCommand(h) == "pk protect" {
				hasProtect = true
			}
		}
	}
	if !hasProtect {
		t.Error("plankit protect hook missing after merge")
	}
}

func TestRun_sessionStartHook(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true, Version: "0.7.1"}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	settingsFile := filepath.Join(projectDir, ".claude", "settings.json")
	data, _ := os.ReadFile(settingsFile)
	var settings map[string]json.RawMessage
	json.Unmarshal(data, &settings)

	var hooks map[string]interface{}
	json.Unmarshal(settings["hooks"], &hooks)

	sessionStart, ok := hooks["SessionStart"].([]interface{})
	if !ok || len(sessionStart) != 1 {
		t.Fatalf("SessionStart = %v, want 1 entry", hooks["SessionStart"])
	}

	// Verify install script was created.
	scriptPath := filepath.Join(projectDir, ".claude", "install-pk.sh")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("install-pk.sh not created: %v", err)
	}
	if !strings.Contains(string(data), `PK_VERSION="v0.7.1"`) {
		t.Errorf("install-pk.sh missing correct version, got: %s", string(data))
	}
}

// TestRun_preservesSettingsKeyOrder is the regression guard for the ordering
// bug where pk setup silently reordered user-authored settings.json keys
// (alphabetical, because json.Marshal on a Go map sorts keys). Tools don't
// get to reorder user files for their own convenience.
func TestRun_preservesSettingsKeyOrder(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(settingsDir, 0755)

	// Top-level keys in a non-alphabetical order; inner "hooks" also has
	// categories in a non-alphabetical order (PreToolUse before PostToolUse),
	// plus an unknown category (Stop) that must be preserved in place.
	existing := `{
  "statusLine": {"type": "command", "command": "echo hi"},
  "hooks": {
    "PreToolUse": [{"matcher":"Bash","hooks":[{"type":"command","command":"my-checker","timeout":5}]}],
    "PostToolUse": [{"matcher":"Task","hooks":[{"type":"command","command":"user-hook"}]}],
    "Stop": [{"matcher":"","hooks":[{"type":"command","command":"user-stop"}]}]
  },
  "permissions": {"deny": ["Write(/etc/**)"], "allow": ["Bash(make:*)"]}
}
`
	settingsFile := filepath.Join(settingsDir, "settings.json")
	os.WriteFile(settingsFile, []byte(existing), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true, Version: "v1.0.0"}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}

	// Top-level order: statusLine, hooks, permissions (preserved as input).
	wantTopOrder := []string{`"statusLine"`, `"hooks"`, `"permissions"`}
	if !keysAppearInOrder(string(data), wantTopOrder) {
		t.Errorf("top-level keys reordered; want %v in order.\nGot:\n%s", wantTopOrder, string(data))
	}

	// Inner hooks order: PreToolUse, PostToolUse, Stop (SessionStart appended).
	// SessionStart is added by pk setup because it's a managed category — new
	// keys append to the end. Requires a non-dev Version to include SessionStart.
	wantHooksOrder := []string{`"PreToolUse"`, `"PostToolUse"`, `"Stop"`, `"SessionStart"`}
	if !keysAppearInOrder(string(data), wantHooksOrder) {
		t.Errorf("hooks keys reordered; want %v in order.\nGot:\n%s", wantHooksOrder, string(data))
	}

	// Inner permissions order: deny before allow (as the user wrote it).
	wantPermsOrder := []string{`"deny"`, `"allow"`}
	if !keysAppearInOrder(string(data), wantPermsOrder) {
		t.Errorf("permissions keys reordered; want %v in order.\nGot:\n%s", wantPermsOrder, string(data))
	}
}

// keysAppearInOrder reports whether each marker appears in src, and each one
// starts after the previous one. Used to assert key ordering in serialized JSON.
func keysAppearInOrder(src string, markers []string) bool {
	offset := 0
	for _, m := range markers {
		idx := strings.Index(src[offset:], m)
		if idx < 0 {
			return false
		}
		offset += idx + len(m)
	}
	return true
}

// TestRun_preservesUnknownHookFields is the regression guard for dropping
// user-authored hook fields on round-trip. Fields not declared on plankit's
// Hook struct (whether from another tool, a user customisation, or a future
// Claude Code addition) must pass through byte-for-byte.
func TestRun_preservesUnknownHookFields(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(settingsDir, 0755)

	// A user hook with a field plankit doesn't know about (continueOnError)
	// plus an ordering that plankit's Hook struct wouldn't produce
	// (command before type).
	existing := `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "command": "my-checker",
            "type": "command",
            "timeout": 5,
            "continueOnError": true
          }
        ]
      }
    ]
  }
}
`
	settingsFile := filepath.Join(settingsDir, "settings.json")
	os.WriteFile(settingsFile, []byte(existing), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	out := string(data)

	if !strings.Contains(out, `"continueOnError": true`) {
		t.Errorf("continueOnError dropped on round-trip; settings.json:\n%s", out)
	}
	// Command-before-type order in the user's hook object should survive,
	// since user hooks are carried as raw JSON.
	commandIdx := strings.Index(out, `"command": "my-checker"`)
	typeIdx := strings.Index(out, `"type": "command"`)
	if commandIdx < 0 || typeIdx < 0 || commandIdx > typeIdx {
		t.Errorf("user hook field order reshuffled; want command before type.\n%s", out)
	}
}

func TestRun_commitTip_shownOnChangedRelease(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true, Version: "0.7.1"}
	withFS(&cfg)

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := `chore: update pk-managed files for v0.7.1`
	if !strings.Contains(stderr.String(), want) {
		t.Errorf("stderr = %q, want tip containing %q", stderr.String(), want)
	}
	if !strings.Contains(stderr.String(), "Commit these updates on their own:") {
		t.Errorf("stderr = %q, want tip header", stderr.String())
	}
}

func TestRun_commitTip_hiddenWhenIdempotent(t *testing.T) {
	projectDir := t.TempDir()
	var firstStderr, secondStderr bytes.Buffer

	first := Config{Stderr: &firstStderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true, Version: "0.7.1"}
	withFS(&first)
	if err := Run(first); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	// Sanity: first run should have shown the tip.
	if !strings.Contains(firstStderr.String(), "chore: update pk-managed files") {
		t.Fatalf("first run did not show tip; stderr = %q", firstStderr.String())
	}

	second := Config{Stderr: &secondStderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true, Version: "0.7.1"}
	withFS(&second)
	if err := Run(second); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	if strings.Contains(secondStderr.String(), "chore: update pk-managed files") {
		t.Errorf("idempotent re-run should not show tip; stderr = %q", secondStderr.String())
	}
}

func TestRun_commitTip_hiddenOnDevBuild(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true, Version: "dev"}
	withFS(&cfg)

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if strings.Contains(stderr.String(), "chore: update pk-managed files") {
		t.Errorf("dev build should not show tip; stderr = %q", stderr.String())
	}
}

func TestRun_commitTip_hiddenOnEmptyVersion(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if strings.Contains(stderr.String(), "chore: update pk-managed files") {
		t.Errorf("empty version should not show tip; stderr = %q", stderr.String())
	}
}
