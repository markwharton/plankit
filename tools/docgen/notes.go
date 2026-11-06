package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/help"
	"github.com/markwharton/plankit/internal/version"
)

// Release notes are authored, one file per minor or major release, in
// docs/notes/<version>.md with version, date, and title frontmatter.
// The site renders the notes whose tag exists and skips the rest, so a
// note written before its release is shipped never appears early.

type note struct {
	Version, Date, Title string
	Body                 []byte
	ver                  version.Semver
}

func readNotes(root string, all bool) ([]note, error) {
	dir := filepath.Join(root, "docs", "notes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	tags, err := git.Tags(root)
	if err != nil {
		return nil, err
	}
	released := map[string]bool{}
	for _, t := range tags {
		released[t] = true
	}
	var notes []note
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "v") || !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		n, err := parseNote(raw)
		if err != nil {
			return nil, fmt.Errorf("docs/notes/%s: %w", name, err)
		}
		if want := strings.TrimSuffix(name, ".md"); n.Version != want {
			return nil, fmt.Errorf("docs/notes/%s: frontmatter version %q does not match the filename", name, n.Version)
		}
		if !all && !released[n.Version] {
			fmt.Fprintf(os.Stderr, "docgen: notes: %s has no tag yet; not published\n", n.Version)
			continue
		}
		notes = append(notes, n)
	}
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].ver.Compare(notes[j].ver) > 0
	})
	return notes, nil
}

// parseNote reads the three required frontmatter keys and the body,
// through the one frontmatter reader.
func parseNote(raw []byte) (note, error) {
	fm, body, err := help.SplitFrontmatter(raw)
	if err != nil {
		return note{}, err
	}
	for k := range fm {
		if k != "version" && k != "date" && k != "title" {
			return note{}, fmt.Errorf("unknown frontmatter key %q (allowed: version, date, title)", k)
		}
	}
	n := note{Version: fm["version"], Date: fm["date"], Title: fm["title"], Body: bytes.TrimLeft(body, "\n")}
	if n.Version == "" || n.Date == "" || n.Title == "" {
		return note{}, fmt.Errorf("frontmatter needs version, date, and title")
	}
	// A note is for a release: its version is a release tag, the form
	// pk writes, with nothing after the patch number.
	sv, ok := version.ParseSemver(n.Version)
	if !ok || sv.String() != n.Version || sv.PreRelease != "" || sv.Build != "" {
		return note{}, fmt.Errorf("version %q is not vMAJOR.MINOR.PATCH", n.Version)
	}
	n.ver = sv
	return n, nil
}

// notesHTML renders the notes page body, newest first. Each entry is
// anchored at its version, and the version links to the same GitHub
// compare view CHANGELOG.md links, previous tag to this one.
func notesHTML(notes []note, repoURL string, tags []string) template.HTML {
	var b bytes.Buffer
	b.WriteString("<h1>Release notes</h1>\n")
	for _, n := range notes {
		body, _ := renderHTML(n.Body)
		version := template.HTMLEscapeString(n.Version)
		// A compare needs two ends; the first release links its commits.
		if prev := previousTag(n, tags); prev != "" && repoURL != "" {
			version = fmt.Sprintf(`<a href="%s/compare/%s...%s">%s</a>`, repoURL, prev, n.Version, version)
		} else if repoURL != "" {
			version = fmt.Sprintf(`<a href="%s/commits/%s">%s</a>`, repoURL, n.Version, version)
		}
		fmt.Fprintf(&b, `<article class="note" id="%s"><h2>%s</h2><p class="muted">%s · %s</p>%s</article>`+"\n",
			template.HTMLEscapeString(n.Version), template.HTMLEscapeString(n.Title), version, template.HTMLEscapeString(n.Date), body)
	}
	return template.HTML(b.String())
}

// previousTag returns the latest release tag below the note's version,
// or "" when there is none.
func previousTag(n note, tags []string) string {
	var below []string
	for _, t := range tags {
		if sv, ok := version.ParseSemver(t); ok && sv.Compare(n.ver) < 0 {
			below = append(below, t)
		}
	}
	prev, _, _ := version.Latest(below)
	return prev
}

// repoLink is the repository's web address for the notes page's links:
// origin's, as the changelog's compare links use it, and "" when origin
// is absent or is not a web address (a local path).
func repoLink(root string) string {
	u := git.RepoURL(root)
	if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
		return ""
	}
	return u
}
