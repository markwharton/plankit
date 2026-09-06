package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
)

// The site is the third consumer of the one authored source: the
// skills render as the pk help pages, the README as the front page,
// and the design documents and changelog as themselves, all through
// site/layout.html. Nothing on the site is written by hand; templates
// and the stylesheet are the only inputs that are not already docs.

type topic struct {
	name, desc string
	body       []byte
	command    bool // opening heading is "pk <name>"; documents open with the name alone
}

type navItem struct {
	Href, Label, Desc string
	Current           bool
}

type sitePage struct {
	Path    string // output path relative to the site root
	Title   string
	Body    template.HTML
	Nav     []navItem
	Topics  []navItem
	Mermaid bool
	Home    bool // the front page; the wordmark carries the current marker
}

// buildSite renders every page into out. root is the repository root
// (for README.md, docs/, CHANGELOG.md, and site/ templates).
// linkExt is appended to every internal page link. Deployed, it is
// empty: Cloudflare Pages serves ship.html at /docs/ship and redirects
// the .html form there. For a local preview under a plain static
// server it is ".html".
var linkExt = ""

func buildSite(root, skillsDir, out, pk string, notesAll bool) error {
	layout, err := template.ParseFiles(filepath.Join(root, "site", "layout.html"))
	if err != nil {
		return fmt.Errorf("site layout: %w", err)
	}
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(out, "docs"), 0o755); err != nil {
		return err
	}

	// Topics in pk help order: the overview first, then alphabetical.
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	sort.SliceStable(names, func(i, j int) bool { return names[i] == "plankit" && names[j] != "plankit" })

	var topics []topic
	var topicNav []navItem
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(skillsDir, name, "SKILL.md"))
		if err != nil {
			return err
		}
		fm, body, err := splitFrontmatter(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		topics = append(topics, topic{name, fm["description"], body, bytes.HasPrefix(bytes.TrimSpace(body), []byte("# pk "+name+"\n")) || bytes.Equal(bytes.TrimSpace(body), []byte("# pk "+name))})
		topicNav = append(topicNav, navItem{Href: "/docs/" + name + linkExt, Label: name, Desc: fm["description"]})
	}

	notes, err := readNotes(root, notesAll)
	if err != nil {
		return err
	}
	nav := []navItem{
		{Href: "/architecture" + linkExt, Label: "How it works"},
	}
	if len(notes) > 0 {
		nav = append(nav, navItem{Href: "/notes" + linkExt, Label: "Notes"})
	}
	nav = append(nav,
		navItem{Href: "/changelog" + linkExt, Label: "Changelog"},
		navItem{Href: "https://github.com/markwharton/plankit/blob/main/docs/design.md", Label: "Design (GitHub)"},
	)

	writeHTML := func(path, title string, body template.HTML, mermaid bool) error {
		p := sitePage{Path: path, Title: title, Body: body, Mermaid: mermaid, Home: path == "index.html"}
		clean := "/" + strings.TrimSuffix(path, ".html") + linkExt
		for _, n := range nav {
			n.Current = n.Href == clean
			p.Nav = append(p.Nav, n)
		}
		for _, n := range topicNav {
			n.Current = n.Href == clean
			p.Topics = append(p.Topics, n)
		}
		var buf bytes.Buffer
		if err := layout.Execute(&buf, p); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return os.WriteFile(filepath.Join(out, path), buf.Bytes(), 0o644)
	}
	write := func(path, title string, md []byte) error {
		body, mermaid := renderHTML(md)
		return writeHTML(path, title, body, mermaid)
	}

	for _, t := range topics {
		title := "pk " + t.name
		if !t.command {
			title = t.name
		}
		if err := write("docs/"+t.name+".html", title, t.body); err != nil {
			return err
		}
	}
	front, err := frontPage(root, pk, topics)
	if err != nil {
		return err
	}
	if err := writeHTML("index.html", "plankit", front, false); err != nil {
		return err
	}
	for _, src := range []struct{ file, path, title string }{
		{filepath.Join("docs", "architecture.md"), "architecture.html", "How it works"},
		{"CHANGELOG.md", "changelog.html", "Changelog"},
	} {
		md, err := os.ReadFile(filepath.Join(root, src.file))
		if err != nil {
			return err
		}
		if err := write(src.path, src.title, md); err != nil {
			return err
		}
	}
	if len(notes) > 0 {
		if err := writeHTML("notes.html", "Release notes", notesHTML(notes, repoWebURL(root), gitTags(root)), false); err != nil {
			return err
		}
	}
	// Without a root 404.html, Cloudflare Pages answers every unknown
	// path with the front page.
	if err := writeHTML("404.html", "Not found", template.HTML(`<h1>Not found</h1><p>There is no page at this address. <a href="/">Home</a>.</p>`), false); err != nil {
		return err
	}
	for _, static := range []string{"style.css", "_redirects"} {
		b, err := os.ReadFile(filepath.Join(root, "site", static))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, static), b, 0o644); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "docgen: site with %d topics -> %s\n", len(topics), out)
	return nil
}

// renderHTML converts markdown to HTML with goldmark's own renderer,
// the same parser the IR compiler validates with. Mermaid fences
// arrive as <pre><code class="language-mermaid"> and the layout's
// script draws them client-side.
func renderHTML(md []byte) (template.HTML, bool) {
	var buf bytes.Buffer
	if err := goldmark.New().Convert(md, &buf); err != nil {
		return template.HTML("<p>render error: " + template.HTMLEscapeString(err.Error()) + "</p>"), false
	}
	return template.HTML(buf.String()), strings.Contains(string(md), "```mermaid")
}

// frontPage composes the landing page from the repository: the
// README's opening paragraphs and install section, the live demo, the
// command grid from the skills' own descriptions, and the latest tag.
// It is the one page the README is not, and every byte is derived.
func frontPage(root, pk string, topics []topic) (template.HTML, error) {
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		return "", err
	}
	intro, install := readmeSections(string(readme))
	introHTML, _ := renderHTML([]byte(intro))
	installHTML, _ := renderHTML([]byte(install))

	steps, err := runDemo(pk)
	if err != nil {
		return "", fmt.Errorf("demo: %w", err)
	}

	var b strings.Builder
	// The hero is the README's own opening; the generator holds no
	// sentence of its own.
	b.WriteString(`<section class="hero">`)
	b.WriteString(string(introHTML))
	b.WriteString(`</section>`)
	b.WriteString(`<section class="install"><h2>Install</h2>` + string(installHTML) + `</section>`)
	b.WriteString(string(demoHTML(steps)))
	b.WriteString(`<section class="commands"><h2>Commands</h2><p class="muted">One page per command, the same page in the terminal as <code>pk help &lt;name&gt;</code>.</p><div class="grid">`)
	for _, t := range topics {
		if !t.command {
			continue // documents have pages but no command card
		}
		b.WriteString(`<a class="card" href="/docs/` + t.name + linkExt + `"><code>pk ` + template.HTMLEscapeString(t.name) + `</code><span>` + template.HTMLEscapeString(t.desc) + `</span></a>`)
	}
	b.WriteString(`</div></section>`)
	if tag := latestTag(root); tag != "" {
		b.WriteString(`<p class="muted release">Latest release: <a href="https://github.com/markwharton/plankit/releases/latest">` + template.HTMLEscapeString(tag) + `</a> · six platforms · <a href="/marketplace.json">marketplace.json</a></p>`)
	}
	return template.HTML(b.String()), nil
}

// readmeSections returns the README's opening prose (everything after
// the H1 up to the first H2) and the body of its Install section.
func readmeSections(readme string) (intro, install string) {
	parts := strings.Split(readme, "\n## ")
	if len(parts) > 0 {
		intro = strings.TrimSpace(strings.TrimPrefix(parts[0], "# plankit"))
	}
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, "Install\n") {
			install = strings.TrimSpace(strings.TrimPrefix(p, "Install\n"))
		}
	}
	return intro, install
}

func latestTag(root string) string {
	out, err := exec.Command("git", "-C", root, "describe", "--tags", "--abbrev=0").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
