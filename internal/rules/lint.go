package rules

import (
	"fmt"
	"strings"

	"github.com/markwharton/plankit/internal/safety"
)

// emDash is U+2014, built from its code point so this source stays pure ASCII.
var emDash = string(rune(0x2014))

// finding is one lint result. msg includes the location within the file.
type finding struct {
	file string
	msg  string
	tier string // "safety" or "style"
}

// runLint scans the rules and returns a process exit code: non-zero when any
// finding is reported. The base tier is the objective safety scan; --strict adds
// plankit's house-style checks.
func runLint(cfg Config, rs []rule) int {
	var findings []finding
	for _, r := range rs {
		for _, v := range safety.ScanHidden(r.content) {
			findings = append(findings, finding{file: r.displayPath, msg: v, tier: "safety"})
		}
		if cfg.Strict {
			findings = append(findings, styleFindings(r.displayPath, r.content)...)
		}
	}

	if len(findings) == 0 {
		scope := "safety scan"
		if cfg.Strict {
			scope = "safety + house-style checks"
		}
		fmt.Fprintf(cfg.Stderr, "pk rules --lint: no issues found (%s).\n", scope)
		return 0
	}

	fmt.Fprintf(cfg.Stderr, "pk rules --lint found %d issue(s):\n", len(findings))
	for _, f := range findings {
		fmt.Fprintf(cfg.Stderr, "  %s: %s [%s]\n", f.file, f.msg, f.tier)
	}
	return 1
}

// styleFindings runs plankit's mechanically-checkable house-style checks: em
// dashes, trailing whitespace, and hard-wrapped bullets. Content inside fenced
// code blocks is skipped. These are opt-in (--strict) because they are house
// style, not universal.
func styleFindings(file, content string) []finding {
	var out []finding
	lines := strings.Split(content, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.Contains(line, emDash) {
			out = append(out, finding{file, fmt.Sprintf("line %d: em dash (U+2014)", i+1), "style"})
		}
		if strings.TrimRight(line, " \t") != line {
			out = append(out, finding{file, fmt.Sprintf("line %d: trailing whitespace", i+1), "style"})
		}
		if isBullet(trimmed) && i+1 < len(lines) {
			next := strings.TrimSpace(lines[i+1])
			if next != "" && !isBullet(next) && !strings.HasPrefix(next, "#") && !strings.HasPrefix(next, "```") {
				out = append(out, finding{file, fmt.Sprintf("line %d: hard-wrapped bullet (keep bullets single-line)", i+2), "style"})
			}
		}
	}
	return out
}

// isBullet reports whether a trimmed line begins a markdown list item.
func isBullet(trimmed string) bool {
	for _, p := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	// Ordered list: "<digits>. ".
	dot := strings.IndexByte(trimmed, '.')
	if dot > 0 && dot+1 < len(trimmed) && trimmed[dot+1] == ' ' {
		for _, c := range trimmed[:dot] {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	return false
}
