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
	for _, name := range []string{"conventions", "preserve", "ship"} {
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

	// Verify rules were created with SHA markers under the plankit/ subdirectory.
	for _, name := range []string{"model-behavior", "development-standards", "git-discipline", "plankit-tooling"} {
		ruleFile := filepath.Join(projectDir, ".claude", "rules", "plankit", name+".md")
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
	if !strings.Contains(stderr.String(), "guard: block, push: block, preserve: auto") {
		t.Errorf("stderr = %q, want resolved modes mentioned", stderr.String())
	}
}

// TestRun_migratesModesToPkJSON verifies a re-run lifts modes out of old-style
// (flag-bearing) hooks into .pk.json, preserves the existing guard.branches and
// release keys (field-merge), and rewrites the hooks bare.
func TestRun_migratesModesToPkJSON(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	oldSettings := `{"hooks":{"PreToolUse":[{"matcher":"Bash|PowerShell","hooks":[{"type":"command","command":"pk guard --ask --push-guard block"}]}],"PostToolUse":[{"matcher":"ExitPlanMode","hooks":[{"type":"command","command":"pk preserve --notify"}]}]}}`
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(oldSettings), 0644)
	os.WriteFile(filepath.Join(projectDir, ".pk.json"), []byte(`{"guard":{"branches":["main"]},"release":{"branch":"main"}}`), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	pk, _ := os.ReadFile(filepath.Join(projectDir, ".pk.json"))
	for _, want := range []string{`"mode": "ask"`, `"push": "block"`, `"branches"`, `"manual"`, `"release"`} {
		if !strings.Contains(string(pk), want) {
			t.Errorf(".pk.json missing %q\n%s", want, string(pk))
		}
	}

	settingsData, _ := os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	for _, flag := range []string{"--ask", "--push-guard", "--notify"} {
		if strings.Contains(string(settingsData), flag) {
			t.Errorf("settings.json still carries %q after migration:\n%s", flag, string(settingsData))
		}
	}
}

// TestRun_bareGuardMigratesPushOff verifies an existing bare `pk guard` (push
// off in the old encoding) migrates to push:"off" — preserved, not flipped to
// the new block default.
func TestRun_bareGuardMigratesPushOff(t *testing.T) {
	projectDir := t.TempDir()
	claudeDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(claudeDir, "settings.json"),
		[]byte(`{"hooks":{"PreToolUse":[{"matcher":"Bash|PowerShell","hooks":[{"type":"command","command":"pk guard"}]}]}}`), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	pk, _ := os.ReadFile(filepath.Join(projectDir, ".pk.json"))
	if !strings.Contains(string(pk), `"push": "off"`) {
		t.Errorf("bare pk guard should migrate to push off, got:\n%s", string(pk))
	}
}

func TestResolveMode_precedence(t *testing.T) {
	if got := resolveMode("flag", "existing", "migrated", "default"); got != "flag" {
		t.Errorf("got %q, want flag (first non-empty)", got)
	}
	if got := resolveMode("", "existing", "migrated", "default"); got != "existing" {
		t.Errorf("got %q, want existing", got)
	}
	if got := resolveMode("", "", "migrated", "default"); got != "migrated" {
		t.Errorf("got %q, want migrated", got)
	}
	if got := resolveMode("", "", "", "default"); got != "default" {
		t.Errorf("got %q, want default", got)
	}
	if got := resolveMode("", "", ""); got != "" {
		t.Errorf("got %q, want empty (all empty)", got)
	}
}

func TestWritePkModes_errorPaths(t *testing.T) {
	// Malformed existing .pk.json -> top-level parse error.
	d1 := t.TempDir()
	os.WriteFile(filepath.Join(d1, ".pk.json"), []byte("{not json"), 0644)
	if _, err := writePkModes(Config{ReadFile: os.ReadFile, WriteFile: os.WriteFile}, d1, "block", "block", "manual"); err == nil {
		t.Error("expected parse error for malformed .pk.json")
	}

	// A malformed nested guard value -> setNested parse error.
	d2 := t.TempDir()
	os.WriteFile(filepath.Join(d2, ".pk.json"), []byte(`{"guard":"not-an-object"}`), 0644)
	if _, err := writePkModes(Config{ReadFile: os.ReadFile, WriteFile: os.WriteFile}, d2, "block", "block", "manual"); err == nil {
		t.Error("expected setNested parse error for malformed guard object")
	}

	// WriteFile failure -> write error.
	d3 := t.TempDir()
	failWrite := func(string, []byte, os.FileMode) error { return os.ErrPermission }
	if _, err := writePkModes(Config{ReadFile: os.ReadFile, WriteFile: failWrite}, d3, "block", "block", "manual"); err == nil {
		t.Error("expected write error")
	}

	// Idempotent: an identical second write reports no change (the bytes.Equal path).
	d4 := t.TempDir()
	cfg := Config{ReadFile: os.ReadFile, WriteFile: os.WriteFile}
	if _, err := writePkModes(cfg, d4, "block", "block", "manual"); err != nil {
		t.Fatalf("first write: %v", err)
	}
	changed, err := writePkModes(cfg, d4, "block", "block", "manual")
	if err != nil {
		t.Fatalf("second write: %v", err)
	}
	if changed {
		t.Error("expected no change on identical re-write")
	}
}

// TestMarshalNoHTML verifies the helpers keep & < > literal (no HTML escaping)
// while JSON-mandated escapes like \" remain, and that only the indent variant
// carries a trailing newline.
func TestMarshalNoHTML(t *testing.T) {
	out, err := MarshalNoHTML("a && b < c > d \"quoted\"")
	if err != nil {
		t.Fatalf("MarshalNoHTML() error = %v", err)
	}
	want := `"a && b < c > d \"quoted\""`
	if string(out) != want {
		t.Errorf("MarshalNoHTML() = %s, want %s", out, want)
	}

	indented, err := MarshalIndentNoHTML(map[string]string{"k": "a && b"})
	if err != nil {
		t.Fatalf("MarshalIndentNoHTML() error = %v", err)
	}
	if !bytes.Contains(indented, []byte("a && b")) {
		t.Errorf("MarshalIndentNoHTML() = %s, want literal &&", indented)
	}
	if !bytes.HasSuffix(indented, []byte("\"\n}\n")) {
		t.Errorf("MarshalIndentNoHTML() = %q, want single trailing newline after }", indented)
	}
}

// TestWritePkModes_keepsAmpersandsLiteral is a regression test: a user-written
// hook command containing && must round-trip through the field-merge rewrite
// without being HTML-escaped to \u0026\u0026.
func TestWritePkModes_keepsAmpersandsLiteral(t *testing.T) {
	dir := t.TempDir()
	existing := `{
  "changelog": {
    "hooks": {
      "preCommit": "pk pin --file .claude/install-pk.sh $VERSION && go run ./evals/footprint"
    }
  }
}
`
	os.WriteFile(filepath.Join(dir, ".pk.json"), []byte(existing), 0644)
	if _, err := writePkModes(Config{ReadFile: os.ReadFile, WriteFile: os.WriteFile}, dir, "block", "block", "manual"); err != nil {
		t.Fatalf("writePkModes() error = %v", err)
	}
	out, err := os.ReadFile(filepath.Join(dir, ".pk.json"))
	if err != nil {
		t.Fatalf("read .pk.json: %v", err)
	}
	if !bytes.Contains(out, []byte("$VERSION && go run")) {
		t.Errorf(".pk.json = %s, want literal &&", out)
	}
	if bytes.Contains(out, []byte("\\u0026")) {
		t.Errorf(".pk.json = %s, want no \\u0026 escapes", out)
	}
}

// TestRun_keepsUserHookAmpersandsLiteral verifies a user hook command in
// settings.json containing && survives the merge + rewrite unescaped.
func TestRun_keepsUserHookAmpersandsLiteral(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(settingsDir, 0755)
	existing := `{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write",
        "hooks": [
          {"type": "command", "command": "make lint && make test"}
        ]
      }
    ]
  }
}
`
	os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte(existing), 0644)

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	out, err := os.ReadFile(filepath.Join(settingsDir, "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}
	if !bytes.Contains(out, []byte("make lint && make test")) {
		t.Errorf("settings.json = %s, want user hook with literal &&", out)
	}
	if bytes.Contains(out, []byte("\\u0026")) {
		t.Errorf("settings.json = %s, want no \\u0026 escapes", out)
	}
}

// TestRun_migratesFlatRulesToSubdir verifies that setup moves rules into
// .claude/rules/plankit/ and cleans up pre-subdir installs: a pristine pk rule at
// the old flat location is removed, while a user-modified one and the user's own
// rule are preserved.
func TestRun_migratesFlatRulesToSubdir(t *testing.T) {
	projectDir := t.TempDir()
	rulesDir := filepath.Join(projectDir, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// An old pristine pk-managed rule at the flat top-level (marker matches body).
	pristineBody := "# Old Pristine\n\n- body\n"
	pristine := embedSHA("---\ndescription: old\n---\n"+pristineBody, ContentSHA(pristineBody))
	if err := os.WriteFile(filepath.Join(rulesDir, "git-discipline.md"), []byte(pristine), 0644); err != nil {
		t.Fatal(err)
	}
	// A user-modified pk-managed rule at the top-level (marker no longer matches body).
	modifiedBody := "# Modified\n\n- changed by user\n"
	modified := embedSHA("---\ndescription: mod\n---\n"+modifiedBody, ContentSHA("a different body"))
	if err := os.WriteFile(filepath.Join(rulesDir, "model-behavior.md"), []byte(modified), 0644); err != nil {
		t.Fatal(err)
	}
	// A user's own rule (no pk marker) at the top-level.
	if err := os.WriteFile(filepath.Join(rulesDir, "my-own.md"), []byte("# Mine\n\n- keep me\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// New subdir copies exist.
	for _, name := range []string{"git-discipline", "model-behavior", "development-standards", "plankit-tooling"} {
		if _, err := os.Stat(filepath.Join(rulesDir, "plankit", name+".md")); err != nil {
			t.Errorf("subdir rule %s not installed: %v", name, err)
		}
	}
	// Old pristine top-level rule removed by the migration sweep.
	if _, err := os.Stat(filepath.Join(rulesDir, "git-discipline.md")); !os.IsNotExist(err) {
		t.Errorf("old pristine top-level git-discipline.md should be removed (stat err=%v)", err)
	}
	// User-modified top-level rule preserved (with a warning); user's own rule preserved.
	if _, err := os.Stat(filepath.Join(rulesDir, "model-behavior.md")); err != nil {
		t.Errorf("user-modified top-level rule should be preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(rulesDir, "my-own.md")); err != nil {
		t.Errorf("user's own rule should be preserved: %v", err)
	}
	if !strings.Contains(stderr.String(), "preserved") {
		t.Errorf("expected a preserved warning for the modified rule; stderr=%s", stderr.String())
	}

	// Idempotent: a second run leaves the subdir copy byte-identical.
	subFile := filepath.Join(rulesDir, "plankit", "git-discipline.md")
	before, _ := os.ReadFile(subFile)
	stderr.Reset()
	if err := Run(cfg); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}
	after, _ := os.ReadFile(subFile)
	if string(before) != string(after) {
		t.Error("second setup changed the subdir rule; expected idempotent")
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

	// The hook is bare (mode lives in .pk.json) and synchronous (no async field).
	hookData, _ := json.Marshal(postToolUse[0])
	if !strings.Contains(string(hookData), `"pk preserve"`) {
		t.Errorf("PostToolUse hook = %s, want bare \"pk preserve\"", string(hookData))
	}
	if strings.Contains(string(hookData), "--notify") {
		t.Errorf("PostToolUse hook = %s, modes must not be encoded in the command", string(hookData))
	}
	if strings.Contains(string(hookData), "async") {
		t.Errorf("PostToolUse hook = %s, preserve hook should not be async", string(hookData))
	}

	// Manual mode is recorded in .pk.json, not the hook command.
	pkData, err := os.ReadFile(filepath.Join(projectDir, ".pk.json"))
	if err != nil {
		t.Fatalf(".pk.json not written: %v", err)
	}
	if !strings.Contains(string(pkData), `"mode": "manual"`) {
		t.Errorf(".pk.json = %s, want preserve mode manual", string(pkData))
	}

	if !strings.Contains(stderr.String(), "guard: block, push: block, preserve: manual") {
		t.Errorf("stderr = %q, want resolved modes mentioned", stderr.String())
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

func TestRun_conventionsReminder_shownWhenNoPkJSON(t *testing.T) {
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block"}
	withFS(&cfg)

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(stderr.String(), "Run /conventions") {
		t.Errorf("stderr = %q, want /conventions reminder when no .pk.json", stderr.String())
	}
}

func TestRun_conventionsReminder_hiddenWhenPkJSONPresent(t *testing.T) {
	projectDir := t.TempDir()
	os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
	if err := os.WriteFile(filepath.Join(projectDir, ".pk.json"), []byte(`{"release":{"branch":"main"}}`), 0644); err != nil {
		t.Fatalf("write .pk.json: %v", err)
	}
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block"}
	withFS(&cfg)

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if strings.Contains(stderr.String(), "Run /conventions") {
		t.Errorf("stderr = %q, should not show /conventions reminder when .pk.json present", stderr.String())
	}
}

func TestRun_noBackupOrRewriteWhenSettingsUnchanged(t *testing.T) {
	projectDir := t.TempDir()
	settingsFile := filepath.Join(projectDir, ".claude", "settings.json")
	backupFile := settingsFile + ".bak"

	run := func() {
		t.Helper()
		var stderr bytes.Buffer
		cfg := Config{Stderr: &stderr, ProjectDir: projectDir, AllowNonGit: true}
		withFS(&cfg)
		if err := Run(cfg); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	}

	run()
	// A fresh project has no prior settings, so there is nothing to back up.
	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		t.Error("backup created on a fresh project with no existing settings")
	}
	before, err := os.Stat(settingsFile)
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}

	// Re-run resolves to identical settings: nothing to write, nothing to back up.
	run()
	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		t.Error("re-run created a backup even though settings did not change")
	}
	after, err := os.Stat(settingsFile)
	if err != nil {
		t.Fatalf("stat settings.json: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("re-run rewrote settings.json even though it did not change")
	}
}

func TestRun_backupHoldsWhatPkFound(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	settingsFile := filepath.Join(settingsDir, "settings.json")
	// Settings with no plankit hooks: setup has real work to do.
	original := []byte("{\n  \"model\": \"opus\"\n}\n")
	if err := os.WriteFile(settingsFile, original, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, AllowNonGit: true}
	withFS(&cfg)
	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	backup, err := os.ReadFile(settingsFile + ".bak")
	if err != nil {
		t.Fatalf("backup not created when settings changed: %v", err)
	}
	// The backup must be the settings pk found, so it can actually restore them.
	if !bytes.Equal(backup, original) {
		t.Errorf("backup = %q, want the original settings %q", backup, original)
	}
	updated, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	if !strings.Contains(string(updated), "hooks") {
		t.Error("hooks were not written")
	}
	if !strings.Contains(string(updated), `"opus"`) {
		t.Error("existing user key was lost")
	}
	if !strings.Contains(stderr.String(), "Backed up existing settings") {
		t.Error("backup was not reported")
	}
}
