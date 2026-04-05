package setup

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_freshProject(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	if err := Run(projectDir, &stderr, "auto", false); err != nil {
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
	if !ok || len(preToolUse) != 2 {
		t.Fatalf("PreToolUse = %v, want 2 entries", hooks["PreToolUse"])
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
	for _, name := range []string{"changelog", "preserve", "release"} {
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

	// Verify CLAUDE.md was created.
	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	claudeData, err := os.ReadFile(claudeFile)
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	if !strings.Contains(string(claudeData), "## Model Behavior") {
		t.Error("CLAUDE.md missing Model Behavior section")
	}
	if !strings.Contains(string(claudeData), "<!-- pk:sha256:") {
		t.Error("CLAUDE.md missing SHA marker")
	}

	// Verify stderr output.
	if !strings.Contains(stderr.String(), "preserve mode: auto") {
		t.Errorf("stderr = %q, want preserve mode mentioned", stderr.String())
	}
}

func TestRun_createsClaudeMD(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	if err := Run(projectDir, &stderr, "manual", false); err != nil {
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
	// Should have key sections.
	for _, section := range []string{"## Model Behavior", "## Development Standards", "### Git Discipline", "### Testing Discipline"} {
		if !strings.Contains(content, section) {
			t.Errorf("CLAUDE.md missing section: %s", section)
		}
	}
	// Should NOT have placeholder text from base.md.
	if strings.Contains(content, "Replace these examples") {
		t.Error("CLAUDE.md contains placeholder text from base.md")
	}
	if strings.Contains(content, "Project Conventions") {
		t.Error("CLAUDE.md should not contain Project Conventions section")
	}
}

func TestRun_skipsUnmanagedClaudeMD(t *testing.T) {
	projectDir := t.TempDir()
	// Create a user's own CLAUDE.md without marker.
	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	os.WriteFile(claudeFile, []byte("# My Custom CLAUDE.md\n"), 0644)

	var stderr bytes.Buffer
	if err := Run(projectDir, &stderr, "manual", false); err != nil {
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
	Run(projectDir, &stderr, "manual", false)

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
	Run(projectDir, &stderr, "manual", false)

	final, _ := os.ReadFile(claudeFile)
	if !strings.Contains(string(final), "User's custom content") {
		t.Error("pk setup overwrote user-modified CLAUDE.md")
	}
}

func TestRun_updatesUnmodifiedClaudeMD(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	// First run creates CLAUDE.md.
	Run(projectDir, &stderr, "manual", false)

	claudeFile := filepath.Join(projectDir, "CLAUDE.md")
	original, _ := os.ReadFile(claudeFile)

	// Re-run — should update (content unchanged, SHA matches).
	stderr.Reset()
	Run(projectDir, &stderr, "manual", false)

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
	if err := Run(projectDir, &stderr, "auto", false); err != nil {
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
	if err := Run(projectDir, &stderr, "auto", false); err != nil {
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
	if err := Run(projectDir, &stderr, "auto", false); err != nil {
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
	err := Run(projectDir, &stderr, "auto", false)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRun_manualMode(t *testing.T) {
	projectDir := t.TempDir()
	var stderr bytes.Buffer

	if err := Run(projectDir, &stderr, "manual", false); err != nil {
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

	if !strings.Contains(stderr.String(), "preserve mode: manual") {
		t.Errorf("stderr = %q, want preserve mode mentioned", stderr.String())
	}
}

func TestContentSHA(t *testing.T) {
	content := "hello world\n"
	sha := contentSHA(content)
	if len(sha) != 64 {
		t.Fatalf("SHA length = %d, want 64", len(sha))
	}
	if sha != contentSHA(content) {
		t.Fatal("SHA is not deterministic")
	}
	if sha == contentSHA("different\n") {
		t.Fatal("different content produced the same SHA")
	}
}

func TestExtractSHA_htmlComment(t *testing.T) {
	sha := "abc123"
	file := "<!-- pk:sha256:abc123 -->\n# CLAUDE.md\nContent.\n"
	got, body, found := extractSHA(file)
	if !found {
		t.Fatal("extractSHA did not find HTML comment marker")
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
	got, body, found := extractSHA(file)
	if !found {
		t.Fatal("extractSHA did not find frontmatter marker")
	}
	if got != "def456" {
		t.Errorf("SHA = %q, want %q", got, "def456")
	}
	if body != "Body content.\n" {
		t.Errorf("body = %q, want %q", body, "Body content.\n")
	}
}

func TestExtractSHA_noMarker(t *testing.T) {
	_, _, found := extractSHA("# Just a file\nNo marker here.\n")
	if found {
		t.Error("extractSHA found a marker in unmarked file")
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
	sha := contentSHA(content)
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

func TestShouldUpdate_pristineFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	body := "Skill body content.\n"
	sha := contentSHA(body)
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
	sha := contentSHA(content)
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
	if err := writeManaged(path, content, &stderr, false); err != nil {
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
	// Round-trip: extractSHA should recover the SHA.
	sha, body, found := extractSHA(written)
	if !found {
		t.Fatal("extractSHA failed on written file")
	}
	if contentSHA(body) != sha {
		t.Error("SHA does not match body content after round-trip")
	}
}

func TestWriteManaged_frontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	var stderr bytes.Buffer

	content := "---\nname: test\ndescription: A test\n---\nBody content.\n"
	if err := writeManaged(path, content, &stderr, false); err != nil {
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
	// Round-trip: extractSHA should recover the SHA.
	sha, body, found := extractSHA(written)
	if !found {
		t.Fatal("extractSHA failed on written file")
	}
	if contentSHA(body) != sha {
		t.Error("SHA does not match body content after round-trip")
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
