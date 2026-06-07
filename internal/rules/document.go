package rules

import (
	"fmt"
	"path/filepath"
	"strings"
)

// writeFootprint prints the always-on context report to stderr: a totals line, one
// aligned row per source (CLAUDE.md and each rule, with its provenance and kind),
// and a provenance tally. pk rules reports; it writes no files.
func writeFootprint(cfg Config, rs []rule) {
	var b strings.Builder
	writeFootprintLines(&b, cfg, rs)
	managed, modified, local := tally(rs)
	fmt.Fprintf(&b, "Provenance: %d managed (pristine), %d modified, %d user-authored.\n", managed, modified, local)
	fmt.Fprint(cfg.Stderr, b.String())
}

// footrow is one rendered line of the footprint report.
type footrow struct{ label, size, tokens, tags string }

// writeFootprintLines partitions sources into always-on and conditional groups and
// writes each as a totals header plus one aligned row per source: CLAUDE.md (when
// present) and each rule, tagged with its provenance and kind. A rule with paths:
// frontmatter loads only when Claude reads a matching file, so it is reported under
// "Conditional", never folded into the always-on total. The size and token columns
// are right-justified to a width computed across both groups, so they line up.
func writeFootprintLines(b *strings.Builder, cfg Config, rs []rule) {
	claudeBytes, claudeTokens, hasClaude := claudeStats(cfg)

	var always, conditional []footrow
	alwaysBytes, alwaysTokens := 0, 0
	condBytes, condTokens := 0, 0
	if hasClaude {
		always = append(always, footrow{"CLAUDE.md", FormatBytes(claudeBytes), HumanInt(claudeTokens), ""})
		alwaysBytes += claudeBytes
		alwaysTokens += claudeTokens
	}
	for _, r := range rs {
		fr := footrow{
			r.displayPath,
			FormatBytes(r.bytes),
			HumanInt(r.tokens),
			fmt.Sprintf("[%s] %s", provShort(r.provenance), r.kind),
		}
		if r.conditional {
			conditional = append(conditional, fr)
			condBytes += r.bytes
			condTokens += r.tokens
		} else {
			always = append(always, fr)
			alwaysBytes += r.bytes
			alwaysTokens += r.tokens
		}
	}

	// Width across every row so the two groups share aligned columns.
	labelW, sizeW, tokenW := 0, 0, 0
	for _, r := range append(append([]footrow{}, always...), conditional...) {
		labelW = max(labelW, len(r.label))
		sizeW = max(sizeW, len(r.size))
		tokenW = max(tokenW, len(r.tokens))
	}

	writeGroup(b, fmt.Sprintf("Always-on context: %d files, %s, %s tokens (%s)",
		len(always), FormatBytes(alwaysBytes), HumanInt(alwaysTokens), TokenLabel()),
		always, labelW, sizeW, tokenW)
	if len(conditional) > 0 {
		writeGroup(b, fmt.Sprintf("Conditional (loads on matching files): %d files, %s, %s tokens (%s)",
			len(conditional), FormatBytes(condBytes), HumanInt(condTokens), TokenLabel()),
			conditional, labelW, sizeW, tokenW)
	}
}

// writeGroup writes a header line followed by its aligned rows.
func writeGroup(b *strings.Builder, header string, rows []footrow, labelW, sizeW, tokenW int) {
	fmt.Fprintf(b, "%s\n", header)
	for _, r := range rows {
		line := fmt.Sprintf("  %-*s  %*s  %*s tokens", labelW, r.label, sizeW, r.size, tokenW, r.tokens)
		if r.tags != "" {
			line += "  " + r.tags
		}
		fmt.Fprintf(b, "%s\n", line)
	}
}

// claudeStats reports the size and token estimate of CLAUDE.md, which Claude Code
// also loads every session, so the footprint total is honest.
func claudeStats(cfg Config) (bytes, tokens int, exists bool) {
	data, err := cfg.ReadFile(filepath.Join(cfg.ProjectDir, "CLAUDE.md"))
	if err != nil {
		return 0, 0, false
	}
	content := NormalizeLF(string(data))
	return len(content), EstimateTokens(content), true
}

func tally(rs []rule) (managed, modified, local int) {
	for _, r := range rs {
		switch r.provenance {
		case "managed":
			managed++
		case "modified":
			modified++
		default:
			local++
		}
	}
	return
}

func provShort(p string) string {
	switch p {
	case "managed":
		return "managed"
	case "modified":
		return "modified"
	default:
		return "local"
	}
}

// FormatBytes renders a byte count as a compact, approximate size. Exported so
// evals/footprint renders sizes identically to the live `pk rules` footprint.
func FormatBytes(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("~%d KB", (n+512)/1024)
}

// HumanInt formats an integer with thousands separators (e.g. 5800 -> "5,800").
// Exported so evals/footprint formats token counts identically to `pk rules`.
func HumanInt(n int) string {
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
