// Command footprint computes the always-on context cost of the rule set plankit
// ships (internal/setup/template/CLAUDE.md + internal/setup/rules/*.md) and
// refreshes the marker-delimited line in README.md. Skills (internal/setup/skills)
// are reported separately as on-demand.
//
// It is maintainer-only, plankit-repo-only tooling: it reads internal/setup from
// source on disk, so it only makes sense inside this repo. plankit wires it into
// the changelog preCommit hook via `go run ./evals/footprint`, which reads the
// repo's own source (no dependence on the possibly-stale pk on PATH) and stays
// out of the shipped pk binary entirely. It is zero-dep and never touches the
// network, so it is safe in the release path — unlike evals/calibrate, which
// needs the API. Both share the one calibrated estimator in internal/rules.
//
// Usage:
//
//	go run ./evals/footprint            # report + refresh README at the repo root
//	go run ./evals/footprint --repo .   # explicit repo root
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/markwharton/plankit/internal/rules"
)

// Markers delimiting the always-on footprint line in README.md. The region
// between them is regenerated; everything else is left byte-for-byte intact.
const (
	markerStart = "<!-- shipped-footprint:start -->"
	markerEnd   = "<!-- shipped-footprint:end -->"
)

// file is one shipped source file with its size and estimated token cost.
type file struct {
	label  string
	bytes  int
	tokens int
}

func main() {
	repo := flag.String("repo", ".", "Path to the plankit repository root")
	flag.Parse()

	if err := run(*repo); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(repo string) error {
	always, skills, err := collect(repo)
	if err != nil {
		return err
	}
	report(os.Stdout, always, skills)

	_, alwaysTokens := totals(always)
	readmePath := filepath.Join(repo, "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "No README.md at repo root; footprint reported only.")
			return nil
		}
		return fmt.Errorf("read README.md: %w", err)
	}
	updated, found, err := rewriteRegion(string(data), alwaysTokens)
	if err != nil {
		return err
	}
	switch {
	case !found:
		fmt.Fprintf(os.Stderr, "README.md has no %s region; left unchanged.\n", markerStart)
	case updated == string(data):
		fmt.Fprintln(os.Stderr, "README.md footprint line already current.")
	default:
		if err := os.WriteFile(readmePath, []byte(updated), 0644); err != nil {
			return fmt.Errorf("write README.md: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Updated README.md shipped-footprint line.")
	}
	return nil
}

// collect reads the shipped source set: the always-on files (template CLAUDE.md +
// rules) and the on-demand skills, each newline-normalized and measured with the
// shared calibrated estimator. A missing CLAUDE.md means this isn't the plankit
// repo, which is a clear error — this tool is plankit-repo-only.
func collect(repo string) (always, skills []file, err error) {
	setupDir := filepath.Join(repo, "internal", "setup")
	claudePath := filepath.Join(setupDir, "template", "CLAUDE.md")

	claude, err := os.ReadFile(claudePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w (run this only inside the plankit repo)", claudePath, err)
	}
	always = append(always, stat("template/CLAUDE.md", claude))

	rulesFiles, err := mdFiles(filepath.Join(setupDir, "rules"), "rules/")
	if err != nil {
		return nil, nil, err
	}
	always = append(always, rulesFiles...)

	skills, err = skillFiles(filepath.Join(setupDir, "skills"))
	if err != nil {
		return nil, nil, err
	}
	return always, skills, nil
}

// mdFiles reads every *.md directly under dir, sorted, labelled prefix+name.
func mdFiles(dir, prefix string) ([]file, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	var fs []file
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filepath.Join(dir, e.Name()), err)
		}
		fs = append(fs, stat(prefix+e.Name(), data))
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].label < fs[j].label })
	return fs, nil
}

// skillFiles reads each skills/<name>/SKILL.md, sorted. A missing skills
// directory yields none, not an error.
func skillFiles(skillsDir string) ([]file, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", skillsDir, err)
	}
	var fs []file
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		fs = append(fs, stat("skills/"+e.Name()+"/SKILL.md", data))
	}
	sort.Slice(fs, func(i, j int) bool { return fs[i].label < fs[j].label })
	return fs, nil
}

func stat(label string, data []byte) file {
	content := normalizeLF(string(data))
	return file{label: label, bytes: len(content), tokens: rules.EstimateTokens(content)}
}

func totals(fs []file) (bytes, tokens int) {
	for _, f := range fs {
		bytes += f.bytes
		tokens += f.tokens
	}
	return
}

// footprintLine is the single sentence written into the README marker region.
func footprintLine(tokens int) string {
	return fmt.Sprintf(
		"Always-on rules footprint: ≈%s tokens (%s) for the rules and CLAUDE.md `pk setup` installs, loaded every session. Your edits and added rules change it; run `pk rules` for your own estimate.",
		humanInt(tokens), rules.TokenLabel(),
	)
}

// rewriteRegion replaces the text between the markers with a freshly generated
// line. found is false (content unchanged) when the start marker is absent; a
// start marker with no matching end marker after it is an error.
func rewriteRegion(content string, tokens int) (result string, found bool, err error) {
	si := strings.Index(content, markerStart)
	if si < 0 {
		return content, false, nil
	}
	ei := strings.Index(content, markerEnd)
	if ei < 0 || ei < si {
		return content, false, fmt.Errorf("README.md has %s but no matching %s after it", markerStart, markerEnd)
	}
	before := content[:si+len(markerStart)]
	after := content[ei:]
	return before + "\n" + footprintLine(tokens) + "\n" + after, true, nil
}

func report(w *os.File, always, skills []file) {
	var b strings.Builder
	b.WriteString("Shipped footprint (the files `pk setup` installs), read from source:\n")
	writeBlock(&b, "Always-on (rules + CLAUDE.md)", always)
	if len(skills) > 0 {
		writeBlock(&b, "On-demand (skills, not always-on)", skills)
	}
	fmt.Fprint(w, b.String())
}

// writeBlock writes a totals line plus one aligned row per file, right-justifying
// the size and token columns so they line up in plain text.
func writeBlock(b *strings.Builder, heading string, fs []file) {
	tb, tt := totals(fs)
	fmt.Fprintf(b, "%s: %d files, %s, %s tokens (%s)\n",
		heading, len(fs), formatBytes(tb), humanInt(tt), rules.TokenLabel())

	labelW, sizeW, tokenW := 0, 0, 0
	for _, f := range fs {
		labelW = max(labelW, len(f.label))
		sizeW = max(sizeW, len(formatBytes(f.bytes)))
		tokenW = max(tokenW, len(humanInt(f.tokens)))
	}
	for _, f := range fs {
		fmt.Fprintf(b, "  %-*s  %*s  %*s tokens\n",
			labelW, f.label, sizeW, formatBytes(f.bytes), tokenW, humanInt(f.tokens))
	}
}

// formatBytes renders a byte count as a compact, approximate size.
func formatBytes(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("~%d KB", (n+512)/1024)
}

// humanInt formats an integer with thousands separators (e.g. 5800 -> "5,800").
func humanInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// normalizeLF collapses CRLF and lone CR to LF so counts match internal/rules.
func normalizeLF(s string) string {
	if !strings.Contains(s, "\r") {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
