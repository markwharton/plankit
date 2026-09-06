package main

import (
	"github.com/markwharton/plankit/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildSite renders a miniature repository through the real site
// templates and checks the pages, navigation order, and the mermaid
// hook are all produced.
func TestBuildSite(t *testing.T) {
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"layout.html", "style.css", "_redirects", "favicon.svg", "favicon.ico", "apple-touch-icon.png"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "site", f))
		must(err)
		must(os.MkdirAll(filepath.Join(root, "site"), 0o755))
		must(os.WriteFile(filepath.Join(root, "site", f), b, 0o644))
	}
	skills := filepath.Join(root, "skills")
	for _, s := range []struct{ name, desc, title string }{{"zeta", "Last", "pk zeta"}, {"overview", "Overview", "overview"}, {"alpha", "First", "pk alpha"}, {"craft", "Standards", "craft"}} {
		must(os.MkdirAll(filepath.Join(skills, s.name), 0o755))
		must(os.WriteFile(filepath.Join(skills, s.name, "SKILL.md"),
			[]byte("---\nname: "+s.name+"\ndescription: "+s.desc+"\n---\n\n# "+s.title+"\n\nBody of "+s.name+".\n"), 0o644))
	}
	must(os.MkdirAll(filepath.Join(root, "docs"), 0o755))
	must(os.WriteFile(filepath.Join(root, "README.md"), []byte("# plankit\n\nFront page intro.\n\n## Install\n\n```\n/plugin install plankit\n```\n\n## Other\n\nNot on the front page.\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "docs", "architecture.md"), []byte("# How it works\n\n```mermaid\nflowchart LR\n  A --> B\n```\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte("# Changelog\n"), 0o644))

	out := filepath.Join(root, "dist")
	must(buildSite(root, skills, out, "", false))

	for _, p := range []string{"index.html", "architecture.html", "changelog.html", "favicon.svg", "favicon.ico", "apple-touch-icon.png",
		"help/overview.html", "help/alpha.html", "help/zeta.html", "style.css", "_redirects"} {
		if _, err := os.Stat(filepath.Join(out, p)); err != nil {
			t.Errorf("missing %s", p)
		}
	}
	index, _ := os.ReadFile(filepath.Join(out, "index.html"))
	s := string(index)
	if strings.Index(s, `"/help/overview"`) > strings.Index(s, `"/help/craft"`) || strings.Index(s, `"/help/craft"`) > strings.Index(s, `"/help/alpha"`) {
		t.Error("the overview leads, then documents, then commands")
	}
	if strings.Index(s, `"/help/alpha"`) > strings.Index(s, `"/help/zeta"`) {
		t.Error("commands follow, alphabetically")
	}
	if !strings.Contains(s, `<hr>`) {
		t.Error("a rule separates documents from commands in the sidebar")
	}
	if !strings.Contains(s, "Front page intro.") || !strings.Contains(s, "/plugin install plankit") || strings.Contains(s, "Not on the front page") {
		t.Error("front page must carry the README intro and Install section and nothing further")
	}
	if strings.Contains(s, `>Home<`) || strings.Contains(s, "is the plugin;") {
		t.Error("no Home item and no generator-authored tagline: the wordmark is the home link and the README is the hero")
	}
	if !strings.Contains(s, `class="brand" href="/" aria-current="page"`) {
		t.Error("the wordmark carries the current marker on the front page")
	}
	if !strings.Contains(s, `href="/`+config.SchemaFile+`"`) {
		t.Error("the footer links the policy file's schema")
	}
	if !strings.Contains(s, `href="/style.css?v=`) {
		t.Error("the stylesheet link carries its content hash")
	}
	if !strings.Contains(s, `class="card" href="/help/alpha"`) || strings.Contains(s, `class="card" href="/help/overview"`) || strings.Contains(s, `class="card" href="/help/craft"`) {
		t.Error("command grid must list command pages and skip documents")
	}
	if !strings.Contains(s, `href="/help/craft"`) {
		t.Error("documents still appear in the topic navigation")
	}
	if strings.Contains(s, `href="/help/alpha.html"`) || !strings.Contains(s, `href="/architecture"`) {
		t.Error("deployed links are extension-less")
	}
	if _, err := os.Stat(filepath.Join(out, "404.html")); err != nil {
		t.Error("a root 404.html must exist or Pages falls back to the front page")
	}
	// The preview form appends .html so a plain static server resolves it.
	linkExt = ".html"
	defer func() { linkExt = "" }()
	must(buildSite(root, skills, out, "", false))
	index, _ = os.ReadFile(filepath.Join(out, "index.html"))
	if !strings.Contains(string(index), `href="/help/alpha.html"`) || !strings.Contains(string(index), `href="/architecture.html"`) {
		t.Error("preview links must end in .html")
	}
	if strings.Contains(s, "mermaid.esm.min.mjs") {
		t.Error("a page without diagrams must not load mermaid")
	}
	design, _ := os.ReadFile(filepath.Join(out, "architecture.html"))
	if !strings.Contains(string(design), "mermaid.esm.min.mjs") || !strings.Contains(string(design), "language-mermaid") {
		t.Error("how-it-works page must carry the mermaid script and the fence for it to draw")
	}
	if _, err := os.Stat(filepath.Join(out, "design.html")); err == nil {
		t.Error("design.md is a contributor document and must not be published")
	}
	if !strings.Contains(string(design), `aria-current="page"`) {
		t.Error("current page must be marked in the navigation")
	}
}

// TestNotesRenderOnlyReleasedVersions builds a repository with one tag
// and two notes, and checks the released note is published, the
// unreleased one is not, and the navigation gains Notes only when a
// note exists.
func TestNotesRenderOnlyReleasedVersions(t *testing.T) {
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"config", "user.email", "t@t"}, {"config", "user.name", "t"},
		{"commit", "-q", "--allow-empty", "-m", "x"}, {"tag", "v0.0.0"}, {"tag", "v0.1.0"},
		{"remote", "add", "origin", "git@github.com:acme/widgets.git"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		must(cmd.Run())
	}
	for _, f := range []string{"layout.html", "style.css", "_redirects", "favicon.svg", "favicon.ico", "apple-touch-icon.png"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "site", f))
		must(err)
		must(os.MkdirAll(filepath.Join(root, "site"), 0o755))
		must(os.WriteFile(filepath.Join(root, "site", f), b, 0o644))
	}
	skills := filepath.Join(root, "skills")
	must(os.MkdirAll(filepath.Join(skills, "overview"), 0o755))
	must(os.WriteFile(filepath.Join(skills, "overview", "SKILL.md"), []byte("---\nname: overview\ndescription: d\n---\n\n# overview\n\nBody.\n"), 0o644))
	must(os.MkdirAll(filepath.Join(root, "docs", "notes"), 0o755))
	must(os.WriteFile(filepath.Join(root, "README.md"), []byte("# plankit\n\nIntro.\n\n## Install\n\nx\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "docs", "architecture.md"), []byte("# How it works\n\nMap.\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte("# Changelog\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "docs", "notes", "v0.1.0.md"), []byte("---\nversion: v0.1.0\ndate: 2026-01-01\ntitle: First released\n---\n\nShipped.\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "docs", "notes", "v0.2.0.md"), []byte("---\nversion: v0.2.0\ndate: 2026-02-01\ntitle: Not yet\n---\n\nPending.\n"), 0o644))

	out := filepath.Join(root, "dist")
	must(buildSite(root, skills, out, "", false))
	page, err := os.ReadFile(filepath.Join(out, "notes.html"))
	must(err)
	if !strings.Contains(string(page), "First released") || strings.Contains(string(page), "Not yet") {
		t.Fatalf("notes page must carry only released notes:\n%s", page)
	}
	if !strings.Contains(string(page), `id="v0.1.0"`) ||
		!strings.Contains(string(page), `href="https://github.com/acme/widgets/compare/v0.0.0...v0.1.0"`) {
		t.Fatalf("entry must be anchored at its version and link the compare view:\n%s", page)
	}
	index, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if !strings.Contains(string(index), `href="/notes"`) {
		t.Fatal("navigation must gain Notes when a released note exists")
	}

	// A mismatched filename is refused.
	must(os.WriteFile(filepath.Join(root, "docs", "notes", "v0.3.0.md"), []byte("---\nversion: v0.9.9\ndate: d\ntitle: t\n---\n\nx\n"), 0o644))
	if err := buildSite(root, skills, out, "", false); err == nil || !strings.Contains(err.Error(), "does not match the filename") {
		t.Fatalf("mismatched version must fail the build, got %v", err)
	}
}

// TestCodeSpanAcrossLinesCompilesToOneLine pins that a code span
// broken across source lines carries a space, not a newline.
func TestCodeSpanAcrossLinesCompilesToOneLine(t *testing.T) {
	d, err := compile("x", []byte("---\nname: x\ndescription: d\n---\n\n# x\n\nUse `--bump\nmajor|minor` here.\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range d.Blocks {
		for _, s := range b.Inlines {
			if s.Type == "code" && s.Text != "--bump major|minor" {
				t.Fatalf("code span text %q, want %q", s.Text, "--bump major|minor")
			}
		}
	}
}

// TestFrontPageRequiresTheReadmeSections: a README without an Install
// section, or without intro prose, fails the build rather than
// building a front page with a hole in it.
func TestFrontPageRequiresTheReadmeSections(t *testing.T) {
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"layout.html", "style.css", "_redirects", "favicon.svg", "favicon.ico", "apple-touch-icon.png"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "site", f))
		must(err)
		must(os.MkdirAll(filepath.Join(root, "site"), 0o755))
		must(os.WriteFile(filepath.Join(root, "site", f), b, 0o644))
	}
	skills := filepath.Join(root, "skills")
	must(os.MkdirAll(filepath.Join(skills, "overview"), 0o755))
	must(os.WriteFile(filepath.Join(skills, "overview", "SKILL.md"), []byte("---\nname: overview\ndescription: d\n---\n\n# overview\n\nBody.\n"), 0o644))
	must(os.MkdirAll(filepath.Join(root, "docs"), 0o755))
	must(os.WriteFile(filepath.Join(root, "docs", "architecture.md"), []byte("# How it works\n\nMap.\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte("# Changelog\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "README.md"), []byte("# plankit\n\nIntro.\n\n## Installing\n\nrenamed\n"), 0o644))
	err := buildSite(root, skills, filepath.Join(root, "dist"), "", false)
	if err == nil || !strings.Contains(err.Error(), "Install") {
		t.Fatalf("a README without an Install section must fail the build, got %v", err)
	}
}
