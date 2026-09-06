package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Release notes are authored, one file per minor or major release, in
// docs/notes/<version>.md with version, date, and title frontmatter.
// The site renders the notes whose tag exists and skips the rest, so a
// note written before its release is shipped never appears early.

type note struct {
	Version, Date, Title string
	Body                 []byte
	major, minor, patch  int
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
	tags := gitTags(root)
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
		if !all && !tags[n.Version] {
			fmt.Fprintf(os.Stderr, "docgen: notes: %s has no tag yet; not published\n", n.Version)
			continue
		}
		notes = append(notes, n)
	}
	sort.Slice(notes, func(i, j int) bool {
		a, b := notes[i], notes[j]
		if a.major != b.major {
			return a.major > b.major
		}
		if a.minor != b.minor {
			return a.minor > b.minor
		}
		return a.patch > b.patch
	})
	return notes, nil
}

// parseNote reads the three required frontmatter keys and the body.
func parseNote(raw []byte) (note, error) {
	text := string(raw)
	if !strings.HasPrefix(text, "---\n") {
		return note{}, fmt.Errorf("missing frontmatter")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return note{}, fmt.Errorf("unterminated frontmatter")
	}
	n := note{Body: []byte(strings.TrimLeft(text[4+end+5:], "\n"))}
	for _, line := range strings.Split(text[4:4+end], "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "version":
			n.Version = v
		case "date":
			n.Date = v
		case "title":
			n.Title = v
		default:
			return note{}, fmt.Errorf("unknown frontmatter key %q (allowed: version, date, title)", k)
		}
	}
	if n.Version == "" || n.Date == "" || n.Title == "" {
		return note{}, fmt.Errorf("frontmatter needs version, date, and title")
	}
	parts := strings.Split(strings.TrimPrefix(n.Version, "v"), ".")
	if len(parts) != 3 || !strings.HasPrefix(n.Version, "v") {
		return note{}, fmt.Errorf("version %q is not vMAJOR.MINOR.PATCH", n.Version)
	}
	nums := make([]int, 3)
	for i, p := range parts {
		x, err := strconv.Atoi(p)
		if err != nil {
			return note{}, fmt.Errorf("version %q is not vMAJOR.MINOR.PATCH", n.Version)
		}
		nums[i] = x
	}
	n.major, n.minor, n.patch = nums[0], nums[1], nums[2]
	return n, nil
}

func gitTags(root string) map[string]bool {
	tags := map[string]bool{}
	out, err := exec.Command("git", "-C", root, "tag", "--list").Output()
	if err != nil {
		return tags
	}
	for _, t := range strings.Fields(string(out)) {
		tags[t] = true
	}
	return tags
}

// notesHTML renders the notes page body, newest first. Each entry is
// anchored at its version, and the version links to the same GitHub
// compare view CHANGELOG.md links, previous tag to this one.
func notesHTML(notes []note, repoURL string, tags map[string]bool) template.HTML {
	var b bytes.Buffer
	b.WriteString("<h1>Release notes</h1>\n")
	for _, n := range notes {
		body, _ := renderHTML(n.Body)
		version := template.HTMLEscapeString(n.Version)
		if prev := previousTag(n, tags); prev != "" && repoURL != "" {
			version = fmt.Sprintf(`<a href="%s/compare/%s...%s">%s</a>`, repoURL, prev, n.Version, version)
		}
		fmt.Fprintf(&b, `<article class="note" id="%s"><h2>%s</h2><p class="muted">%s · %s</p>%s</article>`+"\n",
			template.HTMLEscapeString(n.Version), template.HTMLEscapeString(n.Title), version, template.HTMLEscapeString(n.Date), body)
	}
	return template.HTML(b.String())
}

// previousTag returns the highest semver tag below the note's version,
// or "" when there is none.
func previousTag(n note, tags map[string]bool) string {
	best, bestKey := "", [3]int{-1, -1, -1}
	for t := range tags {
		p, err := parseNote([]byte("---\nversion: " + t + "\ndate: d\ntitle: t\n---\n"))
		if err != nil {
			continue
		}
		key := [3]int{p.major, p.minor, p.patch}
		if key[0] > n.major || (key[0] == n.major && (key[1] > n.minor || (key[1] == n.minor && key[2] >= n.patch))) {
			continue // not below
		}
		if key[0] > bestKey[0] || (key[0] == bestKey[0] && (key[1] > bestKey[1] || (key[1] == bestKey[1] && key[2] > bestKey[2]))) {
			best, bestKey = t, key
		}
	}
	return best
}

// repoWebURL derives https://github.com/<owner>/<repo> from origin, or
// returns "" when origin is absent or not GitHub.
func repoWebURL(root string) string {
	out, err := exec.Command("git", "-C", root, "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return ""
	}
	u := strings.TrimSpace(string(out))
	u = strings.TrimSuffix(u, ".git")
	switch {
	case strings.HasPrefix(u, "git@github.com:"):
		return "https://github.com/" + strings.TrimPrefix(u, "git@github.com:")
	case strings.HasPrefix(u, "https://github.com/"):
		return u
	}
	return ""
}
