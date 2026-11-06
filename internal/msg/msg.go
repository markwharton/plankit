// Package msg is pk's message shape: the severity-prefixed helpers every
// command writes through, so an error, a warning, a note, and a hint
// each look the same wherever they come from. The text of a message
// lives beside the condition that raises it, and a message raised from
// more than one place is built by one function.
package msg

import (
	"fmt"
	"io"
)

// Errorf writes "Error: ..." to w.
func Errorf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "Error: "+format+"\n", a...)
}

// Warnf writes "Warning: ..." to w.
func Warnf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "Warning: "+format+"\n", a...)
}

// Notef writes "Note: ..." to w.
func Notef(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "Note: "+format+"\n", a...)
}

// Hintf writes "Hint: ..." to w.
func Hintf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "Hint: "+format+"\n", a...)
}

// Section writes a section header: the title followed by a colon.
func Section(w io.Writer, title string) {
	fmt.Fprintf(w, "%s:\n", title)
}

// Itemf writes a two-space-indented line under the nearest Section.
func Itemf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "  "+format+"\n", a...)
}

// Banner writes the release frame: "=== <s> ===". pk release only.
func Banner(w io.Writer, s string) {
	fmt.Fprintf(w, "=== %s ===\n", s)
}
