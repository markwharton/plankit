package rules

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/setup"
)

// managedRule builds a pk-managed rule file (valid pk_sha256 over its body).
func managedRule(desc, kind, body string) string {
	fm := "---\ndescription: " + desc + "\n"
	if kind != "" {
		fm += "kind: " + kind + "\n"
	}
	fm += "pk_sha256: " + setup.ContentSHA(body) + "\n---\n"
	return fm + body
}

// localRule builds a user-authored rule file (no pk_sha256).
func localRule(desc, kind, body string) string {
	fm := "---\ndescription: " + desc + "\n"
	if kind != "" {
		fm += "kind: " + kind + "\n"
	}
	fm += "---\n"
	return fm + body
}

// writeRules lays out a project dir with the given rule files and an optional
// CLAUDE.md, returning a Config pointed at it with stderr captured in buf.
func setupProject(t *testing.T, claudeMD string, files map[string]string) (Config, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(rulesDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if claudeMD != "" {
		if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(claudeMD), 0644); err != nil {
			t.Fatal(err)
		}
	}
	buf := &bytes.Buffer{}
	cfg := DefaultConfig()
	cfg.ProjectDir = dir
	cfg.Version = "v1.2.3"
	cfg.Stderr = buf
	return cfg, buf
}

func TestGenerateDocument(t *testing.T) {
	cfg, buf := setupProject(t, "# Project\n\nsome claude rules\n", map[string]string{
		"git-discipline.md":        managedRule("Git stuff", "craft", "# Git Discipline\n\n- commit with purpose\n"),
		"development-standards.md": managedRule("Dev stuff", "craft", "# Development Standards\n\n- fail fast\n"),
		"local-extra.md":           localRule("My own rule", "", "# Local Extra\n\n- my preference\n"),
	})

	if code := Run(cfg); code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, buf.String())
	}

	data, err := os.ReadFile(filepath.Join(cfg.ProjectDir, "RULES.md"))
	if err != nil {
		t.Fatalf("RULES.md not written: %v", err)
	}
	doc := string(data)

	// Version header.
	if !strings.Contains(doc, "<!-- plankit v1.2.3 -->") {
		t.Error("missing version header")
	}
	// Provenance: 2 managed, 0 modified, 1 local.
	if !strings.Contains(doc, "Provenance: 2 managed (pristine), 0 modified, 1 user-authored.") {
		t.Errorf("wrong provenance tally:\n%s", doc)
	}
	// Alphabetical order by filename: development-standards before git-discipline before local-extra.
	iDev := strings.Index(doc, "## Development Standards")
	iGit := strings.Index(doc, "## Git Discipline")
	iLocal := strings.Index(doc, "## Local Extra")
	if !(iDev < iGit && iGit < iLocal) {
		t.Errorf("sections not alphabetical by filename: dev=%d git=%d local=%d", iDev, iGit, iLocal)
	}
	// Provenance + kind tags in the TOC.
	if !strings.Contains(doc, "[managed] (kind: craft") {
		t.Error("missing managed/craft tag in TOC")
	}
	if !strings.Contains(doc, "[local] (kind: unclassified") {
		t.Error("local rule without kind should show unclassified")
	}
	// Body H1 stripped (no leftover "# Git Discipline" under the "## Git Discipline" heading).
	if strings.Contains(doc, "\n# Git Discipline\n") {
		t.Error("body H1 should be stripped")
	}

	// Footprint to stderr names the file count and a tally.
	out := buf.String()
	if !strings.Contains(out, "Wrote RULES.md") {
		t.Error("missing wrote confirmation")
	}
	if !strings.Contains(out, "4 files") { // CLAUDE.md + 3 rules
		t.Errorf("footprint file count wrong:\n%s", out)
	}
}

func TestMissingDescriptionShown(t *testing.T) {
	cfg, _ := setupProject(t, "", map[string]string{
		"no-desc.md": "---\nkind: craft\n---\n# No Desc\n\n- body\n",
	})
	if code := Run(cfg); code != 0 {
		t.Fatalf("Run returned %d", code)
	}
	data, _ := os.ReadFile(filepath.Join(cfg.ProjectDir, "RULES.md"))
	if !strings.Contains(string(data), "(no description)") {
		t.Errorf("missing-description placeholder absent:\n%s", data)
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	cfg, buf := setupProject(t, "", map[string]string{
		"a.md": managedRule("A", "craft", "# A\n\n- x\n"),
	})
	cfg.DryRun = true
	if code := Run(cfg); code != 0 {
		t.Fatalf("Run returned %d", code)
	}
	if _, err := os.Stat(filepath.Join(cfg.ProjectDir, "RULES.md")); !os.IsNotExist(err) {
		t.Error("dry-run must not write RULES.md")
	}
	if !strings.Contains(buf.String(), "(dry-run) RULES.md not written") {
		t.Errorf("missing dry-run notice:\n%s", buf.String())
	}
}

func TestNoRulesDirectory(t *testing.T) {
	dir := t.TempDir() // no .claude/rules
	buf := &bytes.Buffer{}
	cfg := DefaultConfig()
	cfg.ProjectDir = dir
	cfg.Stderr = buf
	if code := Run(cfg); code != 0 {
		t.Fatalf("Run returned %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "no rules found") {
		t.Errorf("expected no-rules message, got: %s", buf.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "RULES.md")); !os.IsNotExist(err) {
		t.Error("should not write RULES.md when there are no rules")
	}
}

func TestProvenanceClassification(t *testing.T) {
	body := "# Title\n\n- rule\n"
	modified := "---\ndescription: M\npk_sha256: " + setup.ContentSHA("different body") + "\n---\n" + body
	cfg, _ := setupProject(t, "", map[string]string{
		"managed.md":  managedRule("M", "craft", body),
		"modified.md": modified,
		"local.md":    localRule("L", "conduct", body),
	})
	rs, err := collectRules(cfg)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range rs {
		got[r.name] = r.provenance
	}
	want := map[string]string{"managed": "managed", "modified": "modified", "local": "local"}
	for name, prov := range want {
		if got[name] != prov {
			t.Errorf("%s: provenance = %q, want %q", name, got[name], prov)
		}
	}
}

func TestParseFrontmatter(t *testing.T) {
	body, fields := parseFrontmatter("---\ndescription: hi\nkind: craft\n---\nthe body\n")
	if body != "the body\n" {
		t.Errorf("body = %q", body)
	}
	if fields["description"] != "hi" || fields["kind"] != "craft" {
		t.Errorf("fields = %v", fields)
	}

	// No frontmatter: whole content is the body.
	body, fields = parseFrontmatter("just text\n")
	if body != "just text\n" || len(fields) != 0 {
		t.Errorf("no-frontmatter parse wrong: body=%q fields=%v", body, fields)
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := estimateTokens(""); got != 0 {
		t.Errorf("empty = %d", got)
	}
	if got := estimateTokens("abcd"); got != 1 {
		t.Errorf("4 chars = %d, want 1", got)
	}
	if got := estimateTokens("abcde"); got != 2 {
		t.Errorf("5 chars = %d, want 2", got)
	}
}

func TestHumanInt(t *testing.T) {
	cases := map[int]string{0: "0", 42: "42", 999: "999", 1000: "1,000", 5800: "5,800", 1234567: "1,234,567"}
	for n, want := range cases {
		if got := humanInt(n); got != want {
			t.Errorf("humanInt(%d) = %q, want %q", n, got, want)
		}
	}
}
