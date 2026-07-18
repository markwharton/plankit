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

func TestClassify(t *testing.T) {
	body := "# CLAUDE.md\nContent.\n"
	pristine := "<!-- pk:sha256:" + ContentSHA(body) + " -->\n" + body
	skillBody := "Body content.\n"
	pristineSkill := "---\nname: test\npk_sha256: " + ContentSHA(skillBody) + "\n---\n" + skillBody

	tests := []struct {
		name    string
		content string
		want    Provenance
	}{
		{"no marker", "# Just a file\nNo marker here.\n", NotManaged},
		{"pristine html comment", pristine, Pristine},
		{"pristine frontmatter", pristineSkill, Pristine},
		{"modified body", "<!-- pk:sha256:" + ContentSHA(body) + " -->\n# CLAUDE.md\nEdited.\n", Modified},
		{"pristine crlf", strings.ReplaceAll(pristine, "\n", "\r\n"), Pristine},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.content); got != tt.want {
				t.Errorf("Classify() = %v, want %v", got, tt.want)
			}
		})
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

func TestNormalizeLF(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no CR", "hello\nworld\n", "hello\nworld\n"},
		{"CRLF", "hello\r\nworld\r\n", "hello\nworld\n"},
		{"lone CR", "hello\rworld\r", "hello\nworld\n"},
		{"mixed", "a\r\nb\rc\n", "a\nb\nc\n"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeLF(tt.input)
			if got != tt.want {
				t.Errorf("normalizeLF(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractSHA_htmlComment_CRLF(t *testing.T) {
	file := "<!-- pk:sha256:abc123 -->\r\n# CLAUDE.md\r\nContent.\r\n"
	got, body, found := ExtractSHA(file)
	if !found {
		t.Fatal("ExtractSHA did not find HTML comment marker with CRLF")
	}
	if got != "abc123" {
		t.Errorf("SHA = %q, want %q", got, "abc123")
	}
	if !strings.HasPrefix(body, "# CLAUDE.md") {
		t.Errorf("body = %q, want to start with # CLAUDE.md", body)
	}
	if strings.Contains(body, "\r") {
		t.Error("body should not contain \\r after normalization")
	}
}

func TestExtractSHA_frontmatter_CRLF(t *testing.T) {
	file := "---\r\nname: test\r\ndescription: A test\r\npk_sha256: def456\r\n---\r\nBody content.\r\n"
	got, body, found := ExtractSHA(file)
	if !found {
		t.Fatal("ExtractSHA did not find frontmatter marker with CRLF")
	}
	if got != "def456" {
		t.Errorf("SHA = %q, want %q", got, "def456")
	}
	if body != "Body content.\n" {
		t.Errorf("body = %q, want %q", body, "Body content.\n")
	}
}

func TestShouldUpdate_pristineHTMLComment_CRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	content := "# CLAUDE.md\nContent.\n"
	sha := ContentSHA(content)
	managed := "<!-- pk:sha256:" + sha + " -->\r\n# CLAUDE.md\r\nContent.\r\n"
	os.WriteFile(path, []byte(managed), 0644)

	update, reason := shouldUpdate(os.ReadFile, path, "new content", false)
	if !update {
		t.Fatalf("shouldUpdate for pristine CRLF file = false (%s), want true", reason)
	}
	if reason != "updated" {
		t.Errorf("reason = %q, want %q", reason, "updated")
	}
}

func TestShouldUpdate_pristineFrontmatter_CRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	body := "Skill body content.\n"
	sha := ContentSHA(body)
	managed := "---\r\nname: test\r\npk_sha256: " + sha + "\r\n---\r\nSkill body content.\r\n"
	os.WriteFile(path, []byte(managed), 0644)

	update, reason := shouldUpdate(os.ReadFile, path, "new content", false)
	if !update {
		t.Fatalf("shouldUpdate for pristine CRLF frontmatter file = false (%s), want true", reason)
	}
}

func TestWriteManaged_reportsUnchangedOnRerun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	content := "# CLAUDE.md\nContent here.\n"

	var first bytes.Buffer
	cfg := Config{Stderr: &first}
	withFS(&cfg)
	changed, err := writeManaged(cfg, path, content, false)
	if err != nil {
		t.Fatalf("first writeManaged() error = %v", err)
	}
	if !changed {
		t.Error("first write reported no change")
	}
	if !strings.Contains(first.String(), "created") {
		t.Errorf("first write said %q, want created", strings.TrimSpace(first.String()))
	}

	// Re-run with identical content: pk has not changed the file, so setup
	// should say so rather than claiming an update.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	var second bytes.Buffer
	cfg.Stderr = &second
	changed, err = writeManaged(cfg, path, content, false)
	if err != nil {
		t.Fatalf("second writeManaged() error = %v", err)
	}
	if changed {
		t.Error("re-run reported a change for identical content")
	}
	if !strings.Contains(second.String(), "unchanged") {
		t.Errorf("re-run said %q, want unchanged", strings.TrimSpace(second.String()))
	}
	// Nothing was rewritten, so the file was not touched.
	if after, err := os.Stat(path); err == nil && !after.ModTime().Equal(info.ModTime()) {
		t.Error("re-run rewrote a file whose content had not changed")
	}
}

func TestWriteManaged_reportsUpdatedWhenContentChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr}
	withFS(&cfg)

	if _, err := writeManaged(cfg, path, "# CLAUDE.md\nOld.\n", false); err != nil {
		t.Fatalf("writeManaged() error = %v", err)
	}
	stderr.Reset()

	// New embedded content, pristine file on disk: a real update.
	changed, err := writeManaged(cfg, path, "# CLAUDE.md\nNew.\n", false)
	if err != nil {
		t.Fatalf("writeManaged() error = %v", err)
	}
	if !changed {
		t.Error("changed content reported as no change")
	}
	if !strings.Contains(stderr.String(), "updated") {
		t.Errorf("said %q, want updated", strings.TrimSpace(stderr.String()))
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "New.") {
		t.Error("new content was not written")
	}
}

func TestWriteManaged_forceOnIdenticalContentIsUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "SKILL.md")
	content := "---\nname: demo\n---\n\nBody.\n"
	var stderr bytes.Buffer
	cfg := Config{Stderr: &stderr}
	withFS(&cfg)

	if _, err := writeManaged(cfg, path, content, false); err != nil {
		t.Fatalf("writeManaged() error = %v", err)
	}
	stderr.Reset()

	// --force overrides the user-modification check, not physics: identical
	// bytes are still identical.
	changed, err := writeManaged(cfg, path, content, true)
	if err != nil {
		t.Fatalf("writeManaged() error = %v", err)
	}
	if changed {
		t.Error("forced re-write of identical content reported a change")
	}
	if !strings.Contains(stderr.String(), "unchanged") {
		t.Errorf("said %q, want unchanged", strings.TrimSpace(stderr.String()))
	}
}

func TestIsUpToDate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	tests := []struct {
		name string
		path string
		want []byte
		ok   bool
	}{
		{"identical", path, []byte("hello\n"), true},
		{"different content", path, []byte("goodbye\n"), false},
		{"trailing byte differs", path, []byte("hello"), false},
		{"missing file", filepath.Join(dir, "absent.txt"), []byte("hello\n"), false},
		{"empty want against real file", path, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUpToDate(os.ReadFile, tt.path, tt.want); got != tt.ok {
				t.Errorf("isUpToDate() = %v, want %v", got, tt.ok)
			}
		})
	}
}

func TestIsUpToDate_unreadableIsNotUpToDate(t *testing.T) {
	// A read failure must never be mistaken for "already correct", or setup
	// would skip repairing a file it cannot inspect.
	failing := func(string) ([]byte, error) { return nil, os.ErrPermission }
	if isUpToDate(failing, "anything", nil) {
		t.Error("an unreadable file reported as up to date")
	}
}

func TestIsExecutable(t *testing.T) {
	dir := t.TempDir()
	exec := filepath.Join(dir, "run.sh")
	plain := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(exec, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(plain, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if !isExecutable(os.Stat, exec) {
		t.Error("0755 file reported as not executable")
	}
	if isExecutable(os.Stat, plain) {
		t.Error("0644 file reported as executable")
	}
	if isExecutable(os.Stat, filepath.Join(dir, "absent")) {
		t.Error("missing file reported as executable")
	}
}
