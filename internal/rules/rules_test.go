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
		path := filepath.Join(rulesDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
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

func TestFootprintReport(t *testing.T) {
	cfg, buf := setupProject(t, "# Project\n\nsome claude rules\n", map[string]string{
		"git-discipline.md":        managedRule("Git stuff", "craft", "# Git Discipline\n\n- commit with purpose\n"),
		"development-standards.md": managedRule("Dev stuff", "craft", "# Development Standards\n\n- fail fast\n"),
		"local-extra.md":           localRule("My own rule", "", "# Local Extra\n\n- my preference\n"),
	})

	if code := Run(cfg); code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, buf.String())
	}
	out := buf.String()

	// pk rules writes no file.
	if _, err := os.Stat(filepath.Join(cfg.ProjectDir, "RULES.md")); !os.IsNotExist(err) {
		t.Error("pk rules must not write any file")
	}
	// Totals line: CLAUDE.md + 3 rules = 4 files.
	if !strings.Contains(out, "Always-on context: 4 files") {
		t.Errorf("footprint totals wrong:\n%s", out)
	}
	// CLAUDE.md is counted.
	if !strings.Contains(out, "CLAUDE.md") {
		t.Errorf("CLAUDE.md not in report:\n%s", out)
	}
	// Per-rule rows carry provenance + kind tags.
	if !strings.Contains(out, ".claude/rules/git-discipline.md") || !strings.Contains(out, "[managed] craft") {
		t.Errorf("missing per-rule row with provenance/kind:\n%s", out)
	}
	if !strings.Contains(out, "[local] unclassified") {
		t.Errorf("local rule without kind should show [local] unclassified:\n%s", out)
	}
	// Provenance tally: 2 managed, 0 modified, 1 user-authored.
	if !strings.Contains(out, "Provenance: 2 managed (pristine), 0 modified, 1 user-authored.") {
		t.Errorf("wrong provenance tally:\n%s", out)
	}
	// Rows sorted by filename: development-standards < git-discipline < local-extra.
	iDev := strings.Index(out, "development-standards.md")
	iGit := strings.Index(out, "git-discipline.md")
	iLocal := strings.Index(out, "local-extra.md")
	if !(iDev < iGit && iGit < iLocal) {
		t.Errorf("rows not sorted by filename: dev=%d git=%d local=%d\n%s", iDev, iGit, iLocal, out)
	}
}

func TestRecursiveDiscovery(t *testing.T) {
	cfg, _ := setupProject(t, "", map[string]string{
		"top.md":                    localRule("top", "", "# Top\n\n- t\n"),
		"plankit/git-discipline.md": managedRule("git", "craft", "# Git Discipline\n\n- c\n"),
	})
	rs, err := collectRules(cfg)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, r := range rs {
		got[r.displayPath] = true
	}
	for _, want := range []string{".claude/rules/top.md", ".claude/rules/plankit/git-discipline.md"} {
		if !got[want] {
			t.Errorf("subdir rule not discovered: missing %q in %v", want, got)
		}
	}
	if len(rs) != 2 {
		t.Errorf("got %d rules, want 2: %v", len(rs), got)
	}
}

func TestConditionalSplit(t *testing.T) {
	conditional := "---\ndescription: scoped\npaths:\n  - \"**/*.go\"\n---\n# Scoped\n\n- only on go files\n"
	cfg, buf := setupProject(t, "# Project\n\nclaude\n", map[string]string{
		"always.md": localRule("always", "conduct", "# Always\n\n- every session\n"),
		"scoped.md": conditional,
	})
	if code := Run(cfg); code != 0 {
		t.Fatalf("Run returned %d, want 0; stderr=%s", code, buf.String())
	}
	out := buf.String()

	// Always-on counts CLAUDE.md + the non-paths rule = 2 files; the paths: rule is excluded.
	if !strings.Contains(out, "Always-on context: 2 files") {
		t.Errorf("conditional rule must not count toward always-on:\n%s", out)
	}
	// Conditional group is present and lists the scoped rule.
	ci := strings.Index(out, "Conditional (loads on matching files): 1 files")
	if ci < 0 {
		t.Fatalf("missing conditional group header:\n%s", out)
	}
	si := strings.Index(out, ".claude/rules/scoped.md")
	if si < ci {
		t.Errorf("scoped rule should appear under the conditional header (header=%d row=%d):\n%s", ci, si, out)
	}
	// The always-on rule should appear before the conditional header.
	if ai := strings.Index(out, ".claude/rules/always.md"); ai < 0 || ai > ci {
		t.Errorf("always-on rule should appear in the always-on group (row=%d header=%d):\n%s", ai, ci, out)
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
	// Expected values track the calibrated charsPerToken ratio: round(runes / ratio).
	if got := EstimateTokens(""); got != 0 {
		t.Errorf("EstimateTokens(empty) = %d, want 0", got)
	}
	for _, runes := range []int{4, 5, 100} {
		want := int(float64(runes)/charsPerToken + 0.5)
		if got := EstimateTokens(strings.Repeat("x", runes)); got != want {
			t.Errorf("EstimateTokens(%d chars) = %d, want %d", runes, got, want)
		}
	}
}

func TestTokenLabel(t *testing.T) {
	got := TokenLabel()
	if calibrated {
		if want := "estimated, calibrated against " + calibrationModel; got != want {
			t.Errorf("tokenLabel() = %q, want %q", got, want)
		}
	} else if got != "estimated" {
		t.Errorf("tokenLabel() = %q, want %q (uncalibrated)", got, "estimated")
	}
}

func TestHumanInt(t *testing.T) {
	cases := map[int]string{0: "0", 42: "42", 999: "999", 1000: "1,000", 5800: "5,800", 1234567: "1,234,567"}
	for n, want := range cases {
		if got := HumanInt(n); got != want {
			t.Errorf("HumanInt(%d) = %q, want %q", n, got, want)
		}
	}
}
