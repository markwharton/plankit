// Package msg is pk's message contract: severity-prefixed output helpers
// and the catalog that the errors reference is generated from.
//
// Commands emit through this package rather than fmt so every message has
// a stable shape, and register their messages in the catalog so the help
// system can render the complete reference without executing anything.
package msg

import (
	"fmt"
	"io"
	"sort"
)

// ID names a catalogued message. Convention: "<command>.<slug>".
type ID string

// Def is a catalogued message: the text template and the hint that names
// the fix. Hints are part of the contract: an error without a next step
// is half a message.
type Def struct {
	Text string
	Hint string
}

var catalog = map[ID]Def{}

// Register adds a message to the catalog. Duplicate registration is a
// programmer error and panics; the catalog is compile-time data.
func Register(id ID, d Def) {
	if _, dup := catalog[id]; dup {
		panic("msg: duplicate message id " + string(id))
	}
	catalog[id] = d
}

// IDs returns the catalogued ids, sorted, for reference generation.
func IDs() []ID {
	ids := make([]ID, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// Lookup returns a catalogued message.
func Lookup(id ID) (Def, bool) {
	d, ok := catalog[id]
	return d, ok
}

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
