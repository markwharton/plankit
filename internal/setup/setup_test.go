package setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_freshProject(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}); err != nil {
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

	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}); err != nil {
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
	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}); err != nil {
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
	Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true})

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
	Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true})

	final, _ := os.ReadFile(claudeFile)
	if !strings.Contains(string(final), "User's custom content") {
		t.Error("pk setup overwrote user-modified CLAUDE.md")
	}
}

func TestRun_updatesUnmodifiedClaudeMD(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	// First run creates CLAUDE.md.
	Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true})

	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	original, _ := os.ReadFile(claudeFile)

	// Re-run — should update (content unchanged, SHA matches).
	stderr.Reset()
	Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true})

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
	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}); err != nil {
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
	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}); err != nil {
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
	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true}); err != nil {
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
	err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "auto", GuardMode: "block", AllowNonGit: true})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRun_refusesNonGitDir(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block"})
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

	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block"}); err != nil {
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

	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}); err != nil {
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

// --- Baseline tests ---

// newGitRepoDir returns a temp dir with a .git subdirectory so git.IsRepo returns true.
func newGitRepoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("setup: mkdir .git: %v", err)
	}
	return dir
}

// fakeGitExec records calls and returns canned responses.
type fakeGitExec struct {
	calls       []string
	tagList     string // output for `tag --list v* ...`
	revParseErr bool   // make `rev-parse --verify` fail
	pushErr     bool   // make `push` fail
}

func (f *fakeGitExec) exec(dir string, args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	switch {
	case len(args) >= 2 && args[0] == "tag" && args[1] == "--list":
		return f.tagList, nil
	case len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--verify":
		if f.revParseErr {
			return "", fmt.Errorf("bad ref")
		}
		return "abc123", nil
	case len(args) >= 1 && args[0] == "push":
		if f.pushErr {
			return "", fmt.Errorf("push failed")
		}
		return "", nil
	}
	return "", nil
}

func baselineCfg(dir string, stderr *bytes.Buffer, fake *fakeGitExec) Config {
	return Config{
		Stderr:       stderr,
		ProjectDir:   dir,
		PreserveMode: "manual",
		GuardMode:    "block",
		GitExec:      fake.exec,
	}
}

func assertCallMade(t *testing.T, calls []string, want string) {
	t.Helper()
	for _, c := range calls {
		if c == want {
			return
		}
	}
	t.Errorf("missing git call %q; calls=%v", want, calls)
}

func assertCallNotMade(t *testing.T, calls []string, unwanted string) {
	t.Helper()
	for _, c := range calls {
		if c == unwanted {
			t.Errorf("unexpected git call %q; calls=%v", unwanted, calls)
		}
	}
}

func TestRun_baseline_createsV000OnHEAD(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCallMade(t, fake.calls, "tag --list v* --sort=-v:refname")
	assertCallMade(t, fake.calls, "tag v0.0.0 HEAD")
	if !strings.Contains(stderr.String(), "Tagged v0.0.0 on HEAD") {
		t.Errorf("stderr = %q, want 'Tagged v0.0.0 on HEAD'", stderr.String())
	}
}

func TestRun_baseline_noOpWhenTagExists(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{tagList: "v1.2.3\nv1.0.0"}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCallNotMade(t, fake.calls, "tag v0.0.0 HEAD")
	if !strings.Contains(stderr.String(), "Found tag v1.2.3 — already anchored") {
		t.Errorf("stderr = %q, want 'Found tag v1.2.3 — already anchored'", stderr.String())
	}
}

func TestRun_baseline_ignoresNonSemverTag(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{tagList: "v-my-thing\nvnext"}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Lookalike tags must not count as anchored — v0.0.0 should be created.
	assertCallMade(t, fake.calls, "tag v0.0.0 HEAD")
}

func TestRun_baselineAt_usesRef(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.BaselineAt = "deadbeef"

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCallMade(t, fake.calls, "rev-parse --verify deadbeef")
	assertCallMade(t, fake.calls, "tag v0.0.0 deadbeef")
	if !strings.Contains(stderr.String(), "Tagged v0.0.0 on deadbeef") {
		t.Errorf("stderr = %q, want 'Tagged v0.0.0 on deadbeef'", stderr.String())
	}
}

func TestRun_baselineAt_invalidRef(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{revParseErr: true}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.BaselineAt = "not-a-ref"

	err := Run(cfg)
	if err == nil {
		t.Fatal("Run() expected error for invalid ref, got nil")
	}
	if !strings.Contains(err.Error(), "does not resolve") {
		t.Errorf("error = %v, want 'does not resolve'", err)
	}
	assertCallNotMade(t, fake.calls, "tag v0.0.0 not-a-ref")
}

func TestRun_baselinePush_callsGitPushWithHEAD(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.Push = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCallMade(t, fake.calls, "push origin HEAD v0.0.0")
	if !strings.Contains(stderr.String(), "Pushed HEAD and v0.0.0 to origin") {
		t.Errorf("stderr = %q, want 'Pushed HEAD and v0.0.0 to origin'", stderr.String())
	}
}

func TestRun_baselineAtPush_pushesTagOnly(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.BaselineAt = "deadbeef"
	cfg.Push = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// With --at, HEAD is NOT pushed — only the tag.
	assertCallMade(t, fake.calls, "push origin v0.0.0")
	assertCallNotMade(t, fake.calls, "push origin HEAD v0.0.0")
	if !strings.Contains(stderr.String(), "Pushed v0.0.0 to origin") {
		t.Errorf("stderr = %q, want 'Pushed v0.0.0 to origin'", stderr.String())
	}
	if strings.Contains(stderr.String(), "HEAD and v0.0.0") {
		t.Errorf("stderr = %q, should not mention HEAD when --at is used", stderr.String())
	}
}

func TestRun_baseline_printsPushHintWhenNotPushed(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(stderr.String(), "git push origin v0.0.0") {
		t.Errorf("stderr = %q, want push hint", stderr.String())
	}
	assertCallNotMade(t, fake.calls, "push origin v0.0.0")
}

func TestRun_baseline_requiresGitRepo(t *testing.T) {
	// Temp dir without .git — git.IsRepo returns false.
	dir := t.TempDir()
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.AllowNonGit = true // let setup proceed past the IsRepo guard at the top

	err := Run(cfg)
	if err == nil || !strings.Contains(err.Error(), "--baseline requires a git repository") {
		t.Errorf("Run() error = %v, want '--baseline requires a git repository'", err)
	}
}

func TestRun_noTagsTip_shownWhenNoTags(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	// Baseline not set.

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(stderr.String(), "No version tags found") {
		t.Errorf("stderr = %q, want 'No version tags found' tip", stderr.String())
	}
}

func TestRun_noTagsTip_hiddenWhenTagsExist(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{tagList: "v0.1.0"}
	cfg := baselineCfg(dir, &stderr, fake)

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if strings.Contains(stderr.String(), "No version tags found") {
		t.Errorf("stderr = %q, tip should be hidden when tags exist", stderr.String())
	}
}

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

func TestRun_existingHooks(t *testing.T) {
	projectDir := t.TempDir()
	settingsDir := filepath.Join(projectDir, ".claude")
	os.MkdirAll(settingsDir, 0755)

	existing := `{"hooks":{"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"my-checker","timeout":5}]}]}}`
	settingsFile := filepath.Join(settingsDir, "settings.json")
	os.WriteFile(settingsFile, []byte(existing), 0644)

	var stderr bytes.Buffer
	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}); err != nil {
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

func TestContentSHA(t *testing.T) {
	content := "hello world\n"
	sha := ContentSHA(content)
	if len(sha) != 64 {
		t.Fatalf("SHA length = %d, want 64", len(sha))
	}
	if sha != ContentSHA(content) {
		t.Fatal("SHA is not deterministic")
	}
	if sha == ContentSHA("different\n") {
		t.Fatal("different content produced the same SHA")
	}
}

func TestExtractSHA_htmlComment(t *testing.T) {
	sha := "abc123"
	file := "<!-- pk:sha256:abc123 -->\n# CLAUDE.md\nContent.\n"
	got, body, found := ExtractSHA(file)
	if !found {
		t.Fatal("ExtractSHA did not find HTML comment marker")
	}
	if got != sha {
		t.Errorf("SHA = %q, want %q", got, sha)
	}
	if !strings.HasPrefix(body, "# CLAUDE.md") {
		t.Errorf("body = %q, want to start with # CLAUDE.md", body)
	}
}

func TestExtractSHA_frontmatter(t *testing.T) {
	file := "---\nname: test\ndescription: A test\npk_sha256: def456\n---\nBody content.\n"
	got, body, found := ExtractSHA(file)
	if !found {
		t.Fatal("ExtractSHA did not find frontmatter marker")
	}
	if got != "def456" {
		t.Errorf("SHA = %q, want %q", got, "def456")
	}
	if body != "Body content.\n" {
		t.Errorf("body = %q, want %q", body, "Body content.\n")
	}
}

func TestExtractSHA_noMarker(t *testing.T) {
	_, _, found := ExtractSHA("# Just a file\nNo marker here.\n")
	if found {
		t.Error("ExtractSHA found a marker in unmarked file")
	}
}

func TestEmbedSHA_htmlComment(t *testing.T) {
	content := "# CLAUDE.md\nContent.\n"
	result := embedSHA(content, "abc123")
	if !strings.HasPrefix(result, "<!-- pk:sha256:abc123 -->") {
		t.Errorf("embedSHA for non-frontmatter should start with HTML comment, got: %q", result[:40])
	}
	if !strings.Contains(result, content) {
		t.Error("embedSHA lost original content")
	}
}

func TestEmbedSHA_frontmatter(t *testing.T) {
	content := "---\nname: test\ndescription: A test\n---\nBody content.\n"
	result := embedSHA(content, "def456")
	if !strings.HasPrefix(result, "---\n") {
		t.Error("embedSHA for frontmatter should start with ---")
	}
	if !strings.Contains(result, "pk_sha256: def456") {
		t.Error("embedSHA should contain pk_sha256 field in frontmatter")
	}
	if !strings.Contains(result, "name: test") {
		t.Error("embedSHA lost original frontmatter fields")
	}
	if !strings.Contains(result, "Body content.") {
		t.Error("embedSHA lost body content")
	}
}

func TestShouldUpdate_newFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.md")
	update, reason := shouldUpdate(path, "content", false)
	if !update {
		t.Fatalf("shouldUpdate for new file = false (%s), want true", reason)
	}
	if reason != "created" {
		t.Errorf("reason = %q, want %q", reason, "created")
	}
}

func TestShouldUpdate_unmanagedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.md")
	os.WriteFile(path, []byte("# My custom file\nContent here.\n"), 0644)

	update, reason := shouldUpdate(path, "new content", false)
	if update {
		t.Fatal("shouldUpdate for unmanaged file = true, want false")
	}
	if !strings.Contains(reason, "not managed") {
		t.Errorf("reason = %q, want to contain 'not managed'", reason)
	}
}

func TestShouldUpdate_pristineHTMLComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	content := "# CLAUDE.md\nContent.\n"
	sha := ContentSHA(content)
	managed := "<!-- pk:sha256:" + sha + " -->\n" + content
	os.WriteFile(path, []byte(managed), 0644)

	update, reason := shouldUpdate(path, "new content", false)
	if !update {
		t.Fatalf("shouldUpdate for pristine file = false (%s), want true", reason)
	}
	if reason != "updated" {
		t.Errorf("reason = %q, want %q", reason, "updated")
	}
}

func TestPruneSkills_removesUnmodifiedDeprecated(t *testing.T) {
	skillsDir := t.TempDir()
	body := "Skill body for the deprecated entry.\n"
	sha := ContentSHA(body)
	managed := "---\nname: gone\npk_sha256: " + sha + "\n---\n" + body
	skillDir := filepath.Join(skillsDir, "gone")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(managed), 0644)

	var stderr bytes.Buffer
	changed := pruneSkills(skillsDir, map[string]bool{}, &stderr)

	if !changed {
		t.Error("pruneSkills returned false; expected true after removal")
	}
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); !os.IsNotExist(err) {
		t.Error("deprecated SKILL.md should have been removed")
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("empty skill directory should have been removed")
	}
	if !strings.Contains(stderr.String(), "gone/SKILL.md: removed") {
		t.Errorf("stderr = %q, want removal notice", stderr.String())
	}
}

func TestPruneSkills_preservesUserModified(t *testing.T) {
	skillsDir := t.TempDir()
	body := "Original body.\n"
	sha := ContentSHA(body)
	// User edited the body — body no longer hashes to sha.
	managed := "---\nname: tweaked\npk_sha256: " + sha + "\n---\nUser changed this.\n"
	skillDir := filepath.Join(skillsDir, "tweaked")
	os.MkdirAll(skillDir, 0755)
	skillFile := filepath.Join(skillDir, "SKILL.md")
	os.WriteFile(skillFile, []byte(managed), 0644)

	var stderr bytes.Buffer
	changed := pruneSkills(skillsDir, map[string]bool{}, &stderr)

	if changed {
		t.Error("pruneSkills returned true; expected false when nothing was removed")
	}
	if _, err := os.Stat(skillFile); err != nil {
		t.Errorf("user-modified SKILL.md should have been preserved: %v", err)
	}
	if !strings.Contains(stderr.String(), "preserved") {
		t.Errorf("stderr = %q, want preservation warning", stderr.String())
	}
}

func TestPruneSkills_ignoresUserCreated(t *testing.T) {
	skillsDir := t.TempDir()
	// No pk_sha256 frontmatter — pk has never managed this.
	skillDir := filepath.Join(skillsDir, "mine")
	os.MkdirAll(skillDir, 0755)
	skillFile := filepath.Join(skillDir, "SKILL.md")
	os.WriteFile(skillFile, []byte("---\nname: mine\n---\nMy own skill.\n"), 0644)

	var stderr bytes.Buffer
	changed := pruneSkills(skillsDir, map[string]bool{}, &stderr)

	if changed {
		t.Error("pruneSkills returned true; expected false when nothing pk-managed was found")
	}
	if _, err := os.Stat(skillFile); err != nil {
		t.Errorf("user-created SKILL.md should have been left alone: %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be silent for user-created skills, got %q", stderr.String())
	}
}

func TestPruneSkills_keepsCurrentlyEmbedded(t *testing.T) {
	skillsDir := t.TempDir()
	body := "Body content.\n"
	sha := ContentSHA(body)
	managed := "---\nname: keeper\npk_sha256: " + sha + "\n---\n" + body
	skillDir := filepath.Join(skillsDir, "keeper")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(managed), 0644)

	var stderr bytes.Buffer
	pruneSkills(skillsDir, map[string]bool{"keeper": true}, &stderr)

	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Errorf("currently-embedded skill should not be touched: %v", err)
	}
}

func TestPruneRules_removesUnmodifiedDeprecated(t *testing.T) {
	rulesDir := t.TempDir()
	body := "Rule body content.\n"
	sha := ContentSHA(body)
	managed := "---\ndescription: gone rule\npk_sha256: " + sha + "\n---\n" + body
	ruleFile := filepath.Join(rulesDir, "gone.md")
	os.WriteFile(ruleFile, []byte(managed), 0644)

	var stderr bytes.Buffer
	changed := pruneRules(rulesDir, map[string]bool{}, &stderr)

	if !changed {
		t.Error("pruneRules returned false; expected true after removal")
	}
	if _, err := os.Stat(ruleFile); !os.IsNotExist(err) {
		t.Error("deprecated rule file should have been removed")
	}
	if !strings.Contains(stderr.String(), "gone.md: removed") {
		t.Errorf("stderr = %q, want removal notice", stderr.String())
	}
}

func TestPruneRules_preservesUserModified(t *testing.T) {
	rulesDir := t.TempDir()
	body := "Original rule body.\n"
	sha := ContentSHA(body)
	managed := "---\ndescription: tweaked\npk_sha256: " + sha + "\n---\nUser edited this.\n"
	ruleFile := filepath.Join(rulesDir, "tweaked.md")
	os.WriteFile(ruleFile, []byte(managed), 0644)

	var stderr bytes.Buffer
	pruneRules(rulesDir, map[string]bool{}, &stderr)

	if _, err := os.Stat(ruleFile); err != nil {
		t.Errorf("user-modified rule should have been preserved: %v", err)
	}
	if !strings.Contains(stderr.String(), "preserved") {
		t.Errorf("stderr = %q, want preservation warning", stderr.String())
	}
}

func TestPruneRules_ignoresUserCreated(t *testing.T) {
	rulesDir := t.TempDir()
	ruleFile := filepath.Join(rulesDir, "mine.md")
	os.WriteFile(ruleFile, []byte("# My rule\n\nNo pk marker.\n"), 0644)

	var stderr bytes.Buffer
	pruneRules(rulesDir, map[string]bool{}, &stderr)

	if _, err := os.Stat(ruleFile); err != nil {
		t.Errorf("user-created rule should have been left alone: %v", err)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be silent for user-created rules, got %q", stderr.String())
	}
}

func TestPruneRules_keepsCurrentlyEmbedded(t *testing.T) {
	rulesDir := t.TempDir()
	body := "Rule body.\n"
	sha := ContentSHA(body)
	managed := "---\ndescription: keeper\npk_sha256: " + sha + "\n---\n" + body
	ruleFile := filepath.Join(rulesDir, "keeper.md")
	os.WriteFile(ruleFile, []byte(managed), 0644)

	var stderr bytes.Buffer
	pruneRules(rulesDir, map[string]bool{"keeper": true}, &stderr)

	if _, err := os.Stat(ruleFile); err != nil {
		t.Errorf("currently-embedded rule should not be touched: %v", err)
	}
}

func TestPruneSkills_missingDir(t *testing.T) {
	var stderr bytes.Buffer
	if pruneSkills("/nonexistent/skills/dir", map[string]bool{}, &stderr) {
		t.Error("pruneSkills should return false when the directory doesn't exist")
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be silent for missing dir, got %q", stderr.String())
	}
}

func TestPruneRules_missingDir(t *testing.T) {
	var stderr bytes.Buffer
	if pruneRules("/nonexistent/rules/dir", map[string]bool{}, &stderr) {
		t.Error("pruneRules should return false when the directory doesn't exist")
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be silent for missing dir, got %q", stderr.String())
	}
}

func TestPruneSkills_skipsNonDirEntries(t *testing.T) {
	skillsDir := t.TempDir()
	// A stray file at the skills/ root level (not inside a subdirectory).
	stray := filepath.Join(skillsDir, "README.md")
	os.WriteFile(stray, []byte("# Notes about my skills\n"), 0644)

	var stderr bytes.Buffer
	pruneSkills(skillsDir, map[string]bool{}, &stderr)

	if _, err := os.Stat(stray); err != nil {
		t.Errorf("non-directory entries should be ignored: %v", err)
	}
}

func TestPruneRules_skipsDirectoriesAndNonMd(t *testing.T) {
	rulesDir := t.TempDir()
	// A subdirectory at the rules/ root level.
	subdir := filepath.Join(rulesDir, "drafts")
	os.MkdirAll(subdir, 0755)
	// A non-.md file at the rules/ root level.
	other := filepath.Join(rulesDir, "scratch.txt")
	os.WriteFile(other, []byte("scratch notes\n"), 0644)

	var stderr bytes.Buffer
	pruneRules(rulesDir, map[string]bool{}, &stderr)

	if _, err := os.Stat(subdir); err != nil {
		t.Errorf("directories under rules/ should be ignored: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("non-.md files under rules/ should be ignored: %v", err)
	}
}

func TestEvaluateRemoval_missingFile(t *testing.T) {
	if evaluateRemoval("/nonexistent/file.md") != "skip" {
		t.Error("evaluateRemoval should return \"skip\" for a missing file")
	}
}

func TestShouldUpdate_pristineFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	body := "Skill body content.\n"
	sha := ContentSHA(body)
	managed := "---\nname: test\npk_sha256: " + sha + "\n---\n" + body
	os.WriteFile(path, []byte(managed), 0644)

	update, reason := shouldUpdate(path, "new content", false)
	if !update {
		t.Fatalf("shouldUpdate for pristine frontmatter file = false (%s), want true", reason)
	}
}

func TestShouldUpdate_modifiedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	content := "original content\n"
	sha := ContentSHA(content)
	managed := "<!-- pk:sha256:" + sha + " -->\nuser modified this\n"
	os.WriteFile(path, []byte(managed), 0644)

	update, reason := shouldUpdate(path, "new content", false)
	if update {
		t.Fatal("shouldUpdate for modified file = true, want false")
	}
	if !strings.Contains(reason, "modified by user") {
		t.Errorf("reason = %q, want to contain 'modified by user'", reason)
	}
}

func TestShouldUpdate_forceOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "managed.md")
	os.WriteFile(path, []byte("# custom\n"), 0644)

	update, reason := shouldUpdate(path, "new content", true)
	if !update {
		t.Fatalf("shouldUpdate with force = false (%s), want true", reason)
	}
	if !strings.Contains(reason, "forced") {
		t.Errorf("reason = %q, want to contain 'forced'", reason)
	}
}

func TestWriteManaged_htmlComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	var stderr bytes.Buffer

	content := "# CLAUDE.md\nContent here.\n"
	if _, err := writeManaged(path, content, &stderr, false); err != nil {
		t.Fatalf("writeManaged() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	written := string(data)
	// Should start with HTML comment marker.
	if !strings.HasPrefix(written, "<!-- pk:sha256:") {
		t.Errorf("non-frontmatter file should start with HTML comment: %q", written[:40])
	}
	// Should contain the content.
	if !strings.Contains(written, "# CLAUDE.md") {
		t.Error("file does not contain original content")
	}
	// Round-trip: ExtractSHA should recover the SHA.
	sha, body, found := ExtractSHA(written)
	if !found {
		t.Fatal("ExtractSHA failed on written file")
	}
	if ContentSHA(body) != sha {
		t.Error("SHA does not match body content after round-trip")
	}
}

func TestWriteManaged_frontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	var stderr bytes.Buffer

	content := "---\nname: test\ndescription: A test\n---\nBody content.\n"
	if _, err := writeManaged(path, content, &stderr, false); err != nil {
		t.Fatalf("writeManaged() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	written := string(data)
	// Should start with frontmatter.
	if !strings.HasPrefix(written, "---\n") {
		t.Error("frontmatter file should start with ---")
	}
	// Should have pk_sha256 in frontmatter.
	if !strings.Contains(written, "pk_sha256: ") {
		t.Error("file missing pk_sha256 in frontmatter")
	}
	// Should preserve original fields.
	if !strings.Contains(written, "name: test") {
		t.Error("file lost original frontmatter fields")
	}
	// Round-trip: ExtractSHA should recover the SHA.
	sha, body, found := ExtractSHA(written)
	if !found {
		t.Fatal("ExtractSHA failed on written file")
	}
	if ContentSHA(body) != sha {
		t.Error("SHA does not match body content after round-trip")
	}
}

func TestScriptVersion_found(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-pk.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nPK_VERSION=\"v0.8.0\"\n"), 0755)

	ver, found := ScriptVersion(path)
	if !found {
		t.Fatal("ScriptVersion did not find PK_VERSION")
	}
	if ver != "v0.8.0" {
		t.Errorf("ScriptVersion = %q, want %q", ver, "v0.8.0")
	}
}

func TestScriptVersion_customName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nMY_APP_VERSION=\"v1.2.3\"\n"), 0755)

	ver, found := ScriptVersion(path)
	if !found {
		t.Fatal("ScriptVersion did not find MY_APP_VERSION")
	}
	if ver != "v1.2.3" {
		t.Errorf("ScriptVersion = %q, want %q", ver, "v1.2.3")
	}
}

func TestScriptVersion_notFound(t *testing.T) {
	_, found := ScriptVersion(filepath.Join(t.TempDir(), "missing.sh"))
	if found {
		t.Error("ScriptVersion should return false when file does not exist")
	}
}

func TestScriptVersion_noVersionLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\necho hello\n"), 0755)

	_, found := ScriptVersion(path)
	if found {
		t.Error("ScriptVersion should return false when no VERSION line")
	}
}

func TestPinVersion_updatesVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-pk.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nPK_VERSION=\"v0.8.0\"\ninstall_dir=\"$HOME/.local/bin\"\n"), 0755)

	updated, err := PinVersion(path, "0.8.1")
	if err != nil {
		t.Fatalf("PinVersion() error = %v", err)
	}
	if !updated {
		t.Fatal("PinVersion should return updated=true")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `PK_VERSION="v0.8.1"`) {
		t.Errorf("script should contain v0.8.1, got: %s", string(data))
	}
}

func TestPinVersion_customName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nMY_APP_VERSION=\"v1.0.0\"\n"), 0755)

	updated, err := PinVersion(path, "1.1.0")
	if err != nil {
		t.Fatalf("PinVersion() error = %v", err)
	}
	if !updated {
		t.Fatal("PinVersion should return updated=true")
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `MY_APP_VERSION="v1.1.0"`) {
		t.Errorf("script should contain v1.1.0, got: %s", string(data))
	}
}

func TestPinVersion_noFile(t *testing.T) {
	updated, err := PinVersion(filepath.Join(t.TempDir(), "missing.sh"), "0.8.1")
	if err != nil {
		t.Fatalf("PinVersion should not error when file doesn't exist, got: %v", err)
	}
	if updated {
		t.Error("PinVersion should return updated=false when file doesn't exist")
	}
}

func TestPinVersion_vPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "install-pk.sh")
	os.WriteFile(path, []byte("#!/usr/bin/env bash\nPK_VERSION=\"v0.8.0\"\n"), 0755)

	PinVersion(path, "v0.8.1")
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), `"vv0.8.1"`) {
		t.Error("PinVersion should not double-prefix v")
	}
}

func TestWriteInstallScript_releaseVersion(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	if _, err := writeInstallScript(projectDir, "0.7.1", &stderr); err != nil {
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

	if _, err := writeInstallScript(projectDir, "v0.8.0", &stderr); err != nil {
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

	if _, err := writeInstallScript(projectDir, "dev", &stderr); err != nil {
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

	if _, err := writeInstallScript(projectDir, "", &stderr); err != nil {
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

	writeInstallScript(projectDir, "0.7.0", &stderr)
	stderr.Reset()
	writeInstallScript(projectDir, "0.7.1", &stderr)

	scriptPath := filepath.Join(projectDir, ".claude", "install-pk.sh")
	data, _ := os.ReadFile(scriptPath)
	if !strings.Contains(string(data), `PK_VERSION="v0.7.1"`) {
		t.Error("re-run should update pinned version to v0.7.1")
	}
}

func TestWriteInstallScript_versionIsolatedPath(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	if _, err := writeInstallScript(projectDir, "0.14.1", &stderr); err != nil {
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

func TestRun_sessionStartHook(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true, Version: "0.7.1"}); err != nil {
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

func TestWriteManaged_skipsModified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	var stderr bytes.Buffer

	// Write initial managed file.
	writeManaged(path, "# Original\nContent.\n", &stderr, false)

	// Simulate user modification: keep marker but change body.
	data, _ := os.ReadFile(path)
	written := string(data)
	firstNewline := strings.IndexByte(written, '\n')
	modified := written[:firstNewline+1] + "# User modified this\n"
	os.WriteFile(path, []byte(modified), 0644)

	// Re-run writeManaged — should skip.
	stderr.Reset()
	writeManaged(path, "# New content\n", &stderr, false)

	final, _ := os.ReadFile(path)
	if !strings.Contains(string(final), "User modified this") {
		t.Error("writeManaged overwrote user-modified file")
	}
	if !strings.Contains(stderr.String(), "modified by user") {
		t.Errorf("stderr = %q, want 'modified by user' message", stderr.String())
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
	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}); err != nil {
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
	// keys append to the end.
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
	if err := Run(Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true}); err != nil {
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

func TestRun_commitTip_shownOnChangedRelease(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true, Version: "0.7.1"}

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
	if err := Run(first); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	// Sanity: first run should have shown the tip.
	if !strings.Contains(firstStderr.String(), "chore: update pk-managed files") {
		t.Fatalf("first run did not show tip; stderr = %q", firstStderr.String())
	}

	second := Config{Stderr: &secondStderr, ProjectDir: projectDir, PreserveMode: "manual", GuardMode: "block", AllowNonGit: true, Version: "0.7.1"}
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

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if strings.Contains(stderr.String(), "chore: update pk-managed files") {
		t.Errorf("empty version should not show tip; stderr = %q", stderr.String())
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
