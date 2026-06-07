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

// writeFootprintLines writes the totals line followed by one aligned row per
// source: CLAUDE.md (when present) and each rule, tagged with its provenance and
// kind. The size and token columns are right-justified to a width computed from
// the values, so they line up in plain text.
func writeFootprintLines(b *strings.Builder, cfg Config, rs []rule) {
	claudeBytes, claudeTokens, hasClaude := claudeStats(cfg)

	type row struct{ label, size, tokens, tags string }
	var rows []row
	if hasClaude {
		rows = append(rows, row{"CLAUDE.md", formatBytes(claudeBytes), humanInt(claudeTokens), ""})
	}
	for _, r := range rs {
		rows = append(rows, row{
			r.displayPath,
			formatBytes(r.bytes),
			humanInt(r.tokens),
			fmt.Sprintf("[%s] %s", provShort(r.provenance), r.kind),
		})
	}

	totalBytes := claudeBytes + sumBytes(rs)
	totalTokens := claudeTokens + sumTokens(rs)
	files := len(rs)
	if hasClaude {
		files++
	}
	fmt.Fprintf(b, "Always-on context: %d files, %s, %s tokens (%s)\n",
		files, formatBytes(totalBytes), humanInt(totalTokens), TokenLabel())

	labelW, sizeW, tokenW := 0, 0, 0
	for _, r := range rows {
		labelW = max(labelW, len(r.label))
		sizeW = max(sizeW, len(r.size))
		tokenW = max(tokenW, len(r.tokens))
	}
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
	content := normalizeLF(string(data))
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

func sumBytes(rs []rule) int {
	var n int
	for _, r := range rs {
		n += r.bytes
	}
	return n
}

func sumTokens(rs []rule) int {
	var n int
	for _, r := range rs {
		n += r.tokens
	}
	return n
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
