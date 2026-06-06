// Package rules implements the `pk rules` command. It aggregates the project's
// .claude/rules/ files into a single, predictable-format RULES.md document with
// a context-footprint summary, and offers an opt-in lint pass (--lint, and
// --lint --strict for house-style checks).
//
// RULES.md is built to be pasted into a Claude session for review of the rule
// set as a whole: it carries per-rule provenance (pristine plankit-managed,
// modified, or user-authored), the pk version that shipped the managed rules,
// an estimated context cost, and an optional craft/conduct classification.
package rules

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/markwharton/plankit/internal/setup"
)

// Config holds the dependencies for the rules command.
type Config struct {
	Stderr     io.Writer
	ProjectDir string
	Version    string // pk version, recorded in the RULES.md header
	Lint       bool   // run the safety scan instead of generating RULES.md
	Strict     bool   // with Lint: also run house-style checks
	DryRun     bool   // print the footprint summary without writing RULES.md
	ReadFile   func(string) ([]byte, error)
	WriteFile  func(string, []byte, os.FileMode) error
	ReadDir    func(string) ([]os.DirEntry, error)
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stderr:    os.Stderr,
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		ReadDir:   os.ReadDir,
	}
}

// rule is a single .claude/rules/ file with its parsed metadata.
type rule struct {
	name        string // filename without .md, e.g. "git-discipline"
	displayPath string // ".claude/rules/git-discipline.md"
	title       string // H1 from the body, e.g. "Git Discipline"
	body        string // content after the frontmatter block
	content     string // full file content (what Claude Code loads)
	description string // frontmatter description, or ""
	kind        string // frontmatter kind, or "unclassified"
	provenance  string // "managed", "modified", or "local"
	bytes       int
	tokens      int
}

// Run generates RULES.md (or runs the lint pass) and returns a process exit code.
func Run(cfg Config) int {
	rs, err := collectRules(cfg)
	if err != nil {
		fmt.Fprintln(cfg.Stderr, "Error:", err)
		return 1
	}

	if cfg.Lint {
		return runLint(cfg, rs)
	}

	if len(rs) == 0 {
		fmt.Fprintf(cfg.Stderr, "pk rules: no rules found in %s\n", filepath.Join(".claude", "rules"))
		return 0
	}

	doc := buildDocument(cfg, rs)

	if cfg.DryRun {
		writeFootprint(cfg, rs, false)
		return 0
	}

	out := filepath.Join(cfg.ProjectDir, "RULES.md")
	if err := cfg.WriteFile(out, []byte(doc), 0644); err != nil {
		fmt.Fprintln(cfg.Stderr, "Error:", fmt.Errorf("failed to write RULES.md: %w", err))
		return 1
	}
	writeFootprint(cfg, rs, true)
	return 0
}

// collectRules reads and parses every .md file under .claude/rules/, sorted by
// filename. A missing rules directory yields an empty slice, not an error.
func collectRules(cfg Config) ([]rule, error) {
	dir := filepath.Join(cfg.ProjectDir, ".claude", "rules")
	entries, err := cfg.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var rs []rule
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := cfg.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		content := normalizeLF(string(data))
		body, fields := parseFrontmatter(content)
		name := strings.TrimSuffix(e.Name(), ".md")

		kind := fields["kind"]
		if kind == "" {
			kind = "unclassified"
		}

		rs = append(rs, rule{
			name:        name,
			displayPath: ".claude/rules/" + e.Name(),
			title:       titleFromBody(body, name),
			body:        body,
			content:     content,
			description: fields["description"],
			kind:        kind,
			provenance:  provenanceOf(content),
			bytes:       len(content),
			tokens:      estimateTokens(content),
		})
	}

	sort.Slice(rs, func(i, j int) bool { return rs[i].name < rs[j].name })
	return rs, nil
}

// provenanceOf classifies a rule file the same way pk status does: pk-managed and
// pristine, pk-managed but modified, or user-authored (no pk marker).
func provenanceOf(content string) string {
	storedSHA, body, found := setup.ExtractSHA(content)
	if !found {
		return "local"
	}
	if setup.ContentSHA(body) == storedSHA {
		return "managed"
	}
	return "modified"
}

// parseFrontmatter splits leading YAML frontmatter (--- ... ---) from the body
// and returns the body plus a key/value map of the frontmatter lines. Files
// without frontmatter return the whole content as the body and an empty map.
// Unlike setup.ExtractSHA, this works whether or not a pk_sha256 key is present,
// so user-authored rules parse too.
func parseFrontmatter(content string) (body string, fields map[string]string) {
	fields = map[string]string{}
	if !strings.HasPrefix(content, "---\n") {
		return content, fields
	}
	closeIdx := strings.Index(content[4:], "\n---\n")
	if closeIdx < 0 {
		return content, fields
	}
	frontmatter := content[4 : 4+closeIdx]
	body = content[4+closeIdx+5:] // skip past "\n---\n"
	for _, line := range strings.Split(frontmatter, "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return body, fields
}

// titleFromBody returns the first markdown H1 ("# Title") in body, falling back
// to a title-cased form of the filename.
func titleFromBody(body, name string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return strings.ReplaceAll(name, "-", " ")
}

// estimateTokens is a documented rough heuristic (~4 chars per token). There is
// no tokenizer in the standard library; figures are labelled as estimates.
func estimateTokens(s string) int {
	return (len([]rune(s)) + 3) / 4
}

// normalizeLF collapses CRLF and lone CR to LF so parsing is newline-consistent.
func normalizeLF(s string) string {
	if !strings.Contains(s, "\r") {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
