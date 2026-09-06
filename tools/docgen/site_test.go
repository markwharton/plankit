package main

import (
	"os"
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
	for _, f := range []string{"layout.html", "style.css", "_redirects"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "site", f))
		must(err)
		must(os.MkdirAll(filepath.Join(root, "site"), 0o755))
		must(os.WriteFile(filepath.Join(root, "site", f), b, 0o644))
	}
	skills := filepath.Join(root, "skills")
	for _, s := range []struct{ name, desc string }{{"zeta", "Last"}, {"plankit", "Overview"}, {"alpha", "First"}} {
		must(os.MkdirAll(filepath.Join(skills, s.name), 0o755))
		must(os.WriteFile(filepath.Join(skills, s.name, "SKILL.md"),
			[]byte("---\nname: "+s.name+"\ndescription: "+s.desc+"\n---\n\n# "+s.name+"\n\nBody of "+s.name+".\n"), 0o644))
	}
	must(os.MkdirAll(filepath.Join(root, "docs"), 0o755))
	must(os.WriteFile(filepath.Join(root, "README.md"), []byte("# plankit\n\nFront page intro.\n\n## Install\n\n```\n/plugin install plankit\n```\n\n## Other\n\nNot on the front page.\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "docs", "design.md"), []byte("# Design\n\n```mermaid\nflowchart LR\n  A --> B\n```\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "docs", "architecture.md"), []byte("# Architecture\n\nMap.\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, "CHANGELOG.md"), []byte("# Changelog\n"), 0o644))

	out := filepath.Join(root, "dist")
	must(buildSite(root, skills, out, ""))

	for _, p := range []string{"index.html", "design.html", "architecture.html", "changelog.html",
		"docs/plankit.html", "docs/alpha.html", "docs/zeta.html", "style.css", "_redirects"} {
		if _, err := os.Stat(filepath.Join(out, p)); err != nil {
			t.Errorf("missing %s", p)
		}
	}
	index, _ := os.ReadFile(filepath.Join(out, "index.html"))
	s := string(index)
	if strings.Index(s, "/docs/plankit.html") > strings.Index(s, "/docs/alpha.html") {
		t.Error("plankit must lead the topic navigation")
	}
	if strings.Index(s, "/docs/alpha.html") > strings.Index(s, "/docs/zeta.html") {
		t.Error("topics after the overview must be alphabetical")
	}
	if !strings.Contains(s, "Front page intro.") || !strings.Contains(s, "/plugin install plankit") || strings.Contains(s, "Not on the front page") {
		t.Error("front page must carry the README intro and Install section and nothing further")
	}
	if !strings.Contains(s, `href="/docs/alpha.html"`) || strings.Contains(s, `class="card" href="/docs/plankit.html"`) {
		t.Error("command grid must list commands and skip the overview")
	}
	if strings.Contains(s, "mermaid.esm.min.mjs") {
		t.Error("a page without diagrams must not load mermaid")
	}
	design, _ := os.ReadFile(filepath.Join(out, "design.html"))
	if !strings.Contains(string(design), "mermaid.esm.min.mjs") || !strings.Contains(string(design), "language-mermaid") {
		t.Error("design page must carry the mermaid script and the fence for it to draw")
	}
	if !strings.Contains(string(design), `aria-current="page"`) {
		t.Error("current page must be marked in the navigation")
	}
}
