// Package rules implements the `pk rules` command. It reports the always-on
// context footprint of the project's .claude/rules/ files plus CLAUDE.md (size,
// estimated tokens, per-rule provenance and craft/conduct kind), and offers an
// opt-in lint pass (--lint, and --lint --strict for house-style checks).
//
// pk rules only reports; it writes no files. Whole-rule-set review (overlap,
// gaps, drift, altitude) is the job of the /review-rules skill, which reads the
// source rules directly.
package rules

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/markwharton/plankit/internal/paths"
	"github.com/markwharton/plankit/internal/setup"
)

// Config holds the dependencies for the rules command.
type Config struct {
	Stderr     io.Writer
	ProjectDir string
	Version    string // pk version (reserved for report context)
	Lint       bool   // run the safety scan instead of the footprint report
	Strict     bool   // with Lint: also run house-style checks
	ReadFile   func(string) ([]byte, error)
	ReadDir    func(string) ([]os.DirEntry, error)
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	return Config{
		Stderr:   os.Stderr,
		ReadFile: os.ReadFile,
		ReadDir:  os.ReadDir,
	}
}

// rule is a single .claude/rules/ file with the metadata the footprint report and
// lint pass need.
type rule struct {
	name        string // filename without .md, e.g. "git-discipline" (sort key)
	displayPath string // path relative to the repo, e.g. ".claude/rules/plankit/git-discipline.md"
	content     string // full file content (what Claude Code loads)
	kind        string // frontmatter kind, or "unclassified"
	provenance  string // "managed", "modified", or "local"
	conditional bool   // has paths: frontmatter, so Claude loads it only on matching files
	bytes       int
	tokens      int
}

// Run reports the always-on footprint (or runs the lint pass) and returns a
// process exit code. It writes no files.
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
		fmt.Fprintf(cfg.Stderr, "pk rules: no rules found in %s\n", filepath.Join(paths.ClaudeDir, paths.RulesDir))
		return 0
	}

	writeFootprint(cfg, rs)
	return 0
}

// collectRules reads and parses every .md file under .claude/rules/, recursing
// into subdirectories (Claude Code discovers rules recursively, so the footprint
// must too). Results are sorted by rule name, then by path for a stable order
// across duplicate stems. A missing rules directory yields an empty slice, not an
// error.
func collectRules(cfg Config) ([]rule, error) {
	root := filepath.Join(cfg.ProjectDir, paths.ClaudeDir, paths.RulesDir)
	var rs []rule
	if err := walkRules(cfg, root, "", &rs); err != nil {
		return nil, err
	}
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].name != rs[j].name {
			return rs[i].name < rs[j].name
		}
		return rs[i].displayPath < rs[j].displayPath
	})
	return rs, nil
}

// walkRules appends a rule for every .md file under dir, descending into
// subdirectories. rel is the slash-separated path of dir relative to .claude/rules
// ("" at the root), used to build each rule's displayPath. A missing directory is
// not an error (yields nothing), matching the previous flat behavior.
func walkRules(cfg Config, dir, rel string, rs *[]rule) error {
	entries, err := cfg.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			sub := name
			if rel != "" {
				sub = rel + "/" + name
			}
			if err := walkRules(cfg, filepath.Join(dir, name), sub, rs); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		full := filepath.Join(dir, name)
		data, err := cfg.ReadFile(full)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", full, err)
		}
		content := NormalizeLF(string(data))
		_, fields := parseFrontmatter(content)

		kind := fields["kind"]
		if kind == "" {
			kind = "unclassified"
		}
		// A paths: key (any value) marks a conditional rule: Claude Code loads it
		// only when a matching file is read, so it is not part of the always-on cost.
		_, conditional := fields["paths"]

		relName := name
		if rel != "" {
			relName = rel + "/" + name
		}

		*rs = append(*rs, rule{
			name:        strings.TrimSuffix(name, ".md"),
			displayPath: ".claude/rules/" + relName,
			content:     content,
			kind:        kind,
			provenance:  provenanceOf(content),
			conditional: conditional,
			bytes:       len(content),
			tokens:      EstimateTokens(content),
		})
	}
	return nil
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

// Token-estimate calibration. There is no tokenizer in the standard library, so
// EstimateTokens approximates token counts from character counts using a single
// calibrated ratio. The ratio is model-specific (tokenization differs per model),
// so it is measured against a named model by evals/calibrate (which calls the
// count_tokens endpoint) and written here by its --write flag. The calibrated
// flag gates the "(estimated, calibrated against <model>)" wording, so the label
// claims calibration only when the ratio came from a real measurement rather than
// the provisional seed.
const (
	// calibrationModel is the model whose tokenizer charsPerToken was measured against.
	calibrationModel = "claude-fable-5"
	// charsPerToken is the calibrated characters-per-token ratio for plankit's
	// shipped markdown. chars/4 runs ~25% low for this content; ~3 is closer.
	// Measured identical (2.93) on claude-opus-4-8 and claude-fable-5.
	charsPerToken = 2.93
	// calibrated reports whether charsPerToken came from a real count_tokens
	// measurement (evals/calibrate --write) rather than the provisional seed.
	// It gates the "calibrated against <model>" wording in the footprint label.
	calibrated = true
)

// EstimateTokens approximates the token count of s using the calibrated
// characters-per-token ratio. Figures are labelled as estimates (see TokenLabel).
// Exported so the maintainer-only evals/footprint tool reports the shipped-rule
// cost with the same ratio the live `pk rules` footprint uses.
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return int(float64(len([]rune(s)))/charsPerToken + 0.5)
}

// TokenLabel returns the parenthetical qualifier for token figures. It claims
// calibration only once charsPerToken has been measured (calibrated == true),
// keeping the wording honest while the constant is still the provisional seed.
func TokenLabel() string {
	if calibrated {
		return "estimated, calibrated against " + calibrationModel
	}
	return "estimated"
}

// NormalizeLF collapses CRLF and lone CR to LF so parsing is newline-consistent.
// Exported so evals/footprint measures byte/token counts on identically-normalized
// content, keeping the maintainer footprint tool and live `pk rules` in agreement.
func NormalizeLF(s string) string {
	if !strings.Contains(s, "\r") {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
