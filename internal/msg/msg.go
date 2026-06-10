// Package msg renders pk's standard stderr message forms.
//
// All human-readable output goes to stderr through these helpers; stdout is
// reserved for hook protocol JSON. The forms:
//
//   - Severity prefixes: "Error: " (cannot proceed), "Warning: " (proceeded
//     but degraded), "Note: " (informational aside). Text after a prefix
//     starts lowercase and single clauses take no trailing period.
//   - Hook attribution: hook commands (guard, preserve, protect) prefix
//     diagnostics with "pk <cmd>: " because their stderr interleaves into the
//     session log; user-invoked commands never self-attribute.
//   - Section headers: a capitalized noun phrase ending in a colon, flush
//     left. The "=== ... ===" banner frames pk release only.
//   - Items and hints: two-space indent under the nearest header or
//     triggering message. Hints name the exact, runnable next command; the
//     "or: <git commands>" line offers the git equivalent when pk is a thin
//     wrapper. No backticks or markdown; output renders in a terminal.
//
// Helpers return nothing: callers ignore stderr write errors, and hook
// commands must never fail on them. Plain fmt.Fprintf remains appropriate
// for free-form body text (document dumps, aligned field blocks).
package msg

import (
	"fmt"
	"io"
)

// Errorf writes "Error: " followed by the formatted message.
func Errorf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "Error: "+format+"\n", args...)
}

// Warnf writes "Warning: " followed by the formatted message.
func Warnf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "Warning: "+format+"\n", args...)
}

// Notef writes "Note: " followed by the formatted message.
func Notef(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "Note: "+format+"\n", args...)
}

// Hookf writes a hook-command diagnostic: "pk <cmd>: " followed by the
// formatted message.
func Hookf(w io.Writer, cmd, format string, args ...any) {
	fmt.Fprintf(w, "pk "+cmd+": "+format+"\n", args...)
}

// Section writes a section header: the title followed by a colon.
func Section(w io.Writer, title string) {
	fmt.Fprintf(w, "%s:\n", title)
}

// Itemf writes a two-space-indented line under the nearest section header.
func Itemf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "  "+format+"\n", args...)
}

// Hintf writes a two-space-indented next-step line under the triggering
// message. Same form as Itemf; the separate name keeps call sites readable.
func Hintf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "  "+format+"\n", args...)
}

// Or writes the git-equivalent escape hatch under a pk-command hint.
func Or(w io.Writer, gitCmd string) {
	fmt.Fprintf(w, "  or: %s\n", gitCmd)
}

// Banner writes the release frame: "=== <s> ===". pk release only.
func Banner(w io.Writer, s string) {
	fmt.Fprintf(w, "=== %s ===\n", s)
}
