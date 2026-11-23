package main

import (
	"fmt"
	"io"
	"strings"
)

// source is one thing Hamilton set down, read from her side: an axiom,
// the theorem derived from them, a rule of the type map, or one of the
// three control structures. Holders are the checks that hold it, or
// "well-formed" for the step that runs before any check. A source with
// no holders is not held.
type source struct {
	Label    string
	Hamilton string
	Holders  []string
	// Joint means the holders hold it together, not each on its own.
	Joint bool
	// Detail says how a holder holds it, where its name does not.
	Detail string
	// Partly says what part is held, when not all of it is.
	Partly string
}

// sources is the one list of Hamilton's sources and what holds each.
// control axioms prints it, and control checks reads it to say which
// sources each check holds.
var sources = []source{
	{Label: "Axiom 1", Hamilton: "A parent invokes its immediate children, and only those",
		Holders: []string{"well-formed"}, Detail: "one root, one parent each"},
	{Label: "Axiom 2", Hamilton: "A parent is responsible for its own output, and only its own",
		Holders: []string{"one-owner"}},
	{Label: "Axiom 3", Hamilton: "A parent decides each child's output access",
		Holders: []string{"one-owner", "replace-checks-owner"}},
	{Label: "Axiom 4", Hamilton: `A parent decides each child's input access, "for reference only"`,
		Holders: []string{"reads-for-reference"}},
	{Label: "Axiom 5", Hamilton: "A parent rejects what is outside its own input",
		Holders: []string{"rejects-declared"}},
	{Label: "Axiom 6", Hamilton: "A parent orders its immediate children",
		Holders: []string{"join-ordered"}},
	{Label: "Theorem", Hamilton: "Single reference, single assignment (from axioms 3 and 4)",
		Holders: []string{"one-owner", "reads-for-reference"}, Joint: true},
	{Label: "Type map", Hamilton: "No two objects in one place",
		Holders: []string{"create-never-replaces", "key-from-content"}},
	{Label: "Type map", Hamilton: "Reject values exist in all types",
		Holders: []string{"reject-values", "rejects-declared"}, Joint: true},
	{Label: "Join", Hamilton: "Each child depends on the ones before",
		Holders: []string{"join-ordered"}},
	{Label: "Include", Hamilton: "The children are independent",
		Holders: []string{"include-independent"}},
	{Label: "Or", Hamilton: "One child is chosen",
		Holders: []string{"or-conditions", "declines-declared"}},
}

// heldBy is a source's "Held by" cell: its holders, joined with "with"
// when they hold it together, then how, then what part, or "not held".
func (s source) heldBy() string {
	if len(s.Holders) == 0 {
		return "not held"
	}
	names := make([]string, len(s.Holders))
	for i, h := range s.Holders {
		names[i] = h
		if h == "well-formed" {
			names[i] = "well-formedness"
		}
	}
	sep := ", "
	if s.Joint {
		sep = " with "
	}
	cell := strings.Join(names, sep)
	if s.Detail != "" {
		cell += ": " + s.Detail
	}
	if s.Partly != "" {
		cell += ", partly: " + s.Partly
	}
	return cell
}

// sourcesOf returns the labels of the sources a check holds, in table
// order, each once.
func sourcesOf(id string) []string {
	var labels []string
	seen := map[string]bool{}
	for _, s := range sources {
		for _, h := range s.Holders {
			if h == id && !seen[s.Label] {
				seen[s.Label] = true
				labels = append(labels, s.Label)
			}
		}
	}
	return labels
}

// printSources writes the table: a header, a rule, and one row per
// source, in columns two spaces apart.
func printSources(w io.Writer) {
	rows := [][3]string{{"Source", "Hamilton", "Held by"}}
	for _, s := range sources {
		rows = append(rows, [3]string{s.Label, s.Hamilton, s.heldBy()})
	}
	width := [2]int{}
	for _, r := range rows {
		for c := 0; c < 2; c++ {
			if n := len([]rune(r[c])); n > width[c] {
				width[c] = n
			}
		}
	}
	line := func(r [3]string) {
		fmt.Fprintf(w, "%s  %s  %s\n", pad(r[0], width[0]), pad(r[1], width[1]), r[2])
	}
	line(rows[0])
	line([3]string{strings.Repeat("-", width[0]), strings.Repeat("-", width[1]), strings.Repeat("-", len("Held by"))})
	for _, r := range rows[1:] {
		line(r)
	}
}

func pad(s string, n int) string {
	return s + strings.Repeat(" ", n-len([]rune(s)))
}
