package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	update, reason := shouldUpdate(os.ReadFile, path, "content", false)
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

	update, reason := shouldUpdate(os.ReadFile, path, "new content", false)
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

	update, reason := shouldUpdate(os.ReadFile, path, "new content", false)
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
	sha := ContentSHA(body)
	managed := "---\nname: test\npk_sha256: " + sha + "\n---\n" + body
	os.WriteFile(path, []byte(managed), 0644)

	update, reason := shouldUpdate(os.ReadFile, path, "new content", false)
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

	update, reason := shouldUpdate(os.ReadFile, path, "new content", false)
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

	update, reason := shouldUpdate(os.ReadFile, path, "new content", true)
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
	cfg := Config{Stderr: &stderr}
	withFS(&cfg)

	content := "# CLAUDE.md\nContent here.\n"
	if _, err := writeManaged(cfg, path, content, false); err != nil {
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
	cfg := Config{Stderr: &stderr}
	withFS(&cfg)

	content := "---\nname: test\ndescription: A test\n---\nBody content.\n"
	if _, err := writeManaged(cfg, path, content, false); err != nil {
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

func TestWriteManaged_skipsModified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr}
	withFS(&cfg)

	// Write initial managed file.
	writeManaged(cfg, path, "# Original\nContent.\n", false)

	// Simulate user modification: keep marker but change body.
	data, _ := os.ReadFile(path)
	written := string(data)
	firstNewline := strings.IndexByte(written, '\n')
	modified := written[:firstNewline+1] + "# User modified this\n"
	os.WriteFile(path, []byte(modified), 0644)

	// Re-run writeManaged — should skip.
	stderr.Reset()
	writeManaged(cfg, path, "# New content\n", false)

	final, _ := os.ReadFile(path)
	if !strings.Contains(string(final), "User modified this") {
		t.Error("writeManaged overwrote user-modified file")
	}
	if !strings.Contains(stderr.String(), "modified by user") {
		t.Errorf("stderr = %q, want 'modified by user' message", stderr.String())
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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	changed := pruneSkills(fsCfg, skillsDir, map[string]bool{})

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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	changed := pruneSkills(fsCfg, skillsDir, map[string]bool{})

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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	changed := pruneSkills(fsCfg, skillsDir, map[string]bool{})

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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	pruneSkills(fsCfg, skillsDir, map[string]bool{"keeper": true})

	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		t.Errorf("currently-embedded skill should not be touched: %v", err)
	}
}

func TestPruneSkills_missingDir(t *testing.T) {
	var stderr bytes.Buffer
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	if pruneSkills(fsCfg, "/nonexistent/skills/dir", map[string]bool{}) {
		t.Error("pruneSkills should return false when the directory doesn't exist")
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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	pruneSkills(fsCfg, skillsDir, map[string]bool{})

	if _, err := os.Stat(stray); err != nil {
		t.Errorf("non-directory entries should be ignored: %v", err)
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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	changed := pruneRules(fsCfg, rulesDir, map[string]bool{})

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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	pruneRules(fsCfg, rulesDir, map[string]bool{})

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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	pruneRules(fsCfg, rulesDir, map[string]bool{})

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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	pruneRules(fsCfg, rulesDir, map[string]bool{"keeper": true})

	if _, err := os.Stat(ruleFile); err != nil {
		t.Errorf("currently-embedded rule should not be touched: %v", err)
	}
}

func TestPruneRules_missingDir(t *testing.T) {
	var stderr bytes.Buffer
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	if pruneRules(fsCfg, "/nonexistent/rules/dir", map[string]bool{}) {
		t.Error("pruneRules should return false when the directory doesn't exist")
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr should be silent for missing dir, got %q", stderr.String())
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
	fsCfg := Config{Stderr: &stderr}
	withFS(&fsCfg)
	pruneRules(fsCfg, rulesDir, map[string]bool{})

	if _, err := os.Stat(subdir); err != nil {
		t.Errorf("directories under rules/ should be ignored: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("non-.md files under rules/ should be ignored: %v", err)
	}
}

func TestEvaluateRemoval_missingFile(t *testing.T) {
	if evaluateRemoval(os.ReadFile, "/nonexistent/file.md") != "skip" {
		t.Error("evaluateRemoval should return \"skip\" for a missing file")
	}
}
