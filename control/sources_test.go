package main

import (
	"os"
	"strings"
	"testing"
)

// The owner's table of Hamilton's sources is a reference file,
// control/axioms.md, beside the code it describes. control axioms prints
// it row for row: the same sources, in the same order, in the same
// words. A change to the table is the owner's; a plan may say why, but
// no test reads a plan.
const axiomsMarker = "## Hamilton's sources"

func TestAxiomsPrintsTheOwnersTable(t *testing.T) {
	data, err := os.ReadFile("axioms.md")
	if err != nil {
		t.Fatal(err)
	}
	plan, text := "axioms.md", string(data)
	table, ok := tableRows(text, axiomsMarker)
	if !ok || len(table) < 2 || strings.Join(table[0], "|") != "Source|Hamilton|Held by" {
		t.Fatalf("%s: the reference has no table with the columns Source, Hamilton, Held by", plan)
	}
	want := table[1:]
	code, out, _ := runCLI("axioms")
	if code != 0 {
		t.Fatalf("axioms: exit %d", code)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("axioms: want a header, a rule, and %d rows, got:\n%s", len(want), out)
	}
	rows := lines[2:]
	if len(rows) != len(want) {
		t.Fatalf("axioms: want the %d rows of %s, got %d:\n%s", len(want), plan, len(rows), out)
	}
	for i, w := range want {
		if len(w) != 3 {
			t.Fatalf("%s: reference row %d has %d cells, want 3", plan, i+1, len(w))
		}
		if got := splitRow(rows[i]); got != [3]string{w[0], w[1], w[2]} {
			t.Errorf("row %d, against %s:\n got  %q\n want %q", i+1, plan, got, w)
		}
	}
}

// splitRow reads one printed row back into its three cells. Columns are
// separated by two or more spaces, which no cell contains.
func splitRow(line string) [3]string {
	var cells []string
	for _, part := range strings.Split(line, "  ") {
		if part = strings.TrimSpace(part); part != "" {
			cells = append(cells, part)
		}
	}
	var row [3]string
	copy(row[:], cells)
	if len(cells) != 3 {
		row[2] = "(" + strings.Join(cells, " | ") + ")"
	}
	return row
}

// Every holder a row names is a check, or the well-formedness step, and
// every check holds at least one row: a check with no source, or a
// source naming a check that does not exist, fails here.
func TestEveryHolderIsACheckAndEveryCheckHoldsASource(t *testing.T) {
	ids := map[string]bool{"well-formed": true}
	for _, id := range soundChecks {
		ids[id] = true
	}
	held := map[string]bool{}
	for _, s := range sources {
		for _, h := range s.Holders {
			if !ids[h] {
				t.Errorf("%s names %s, which is not a check", s.Label, h)
			}
			held[h] = true
		}
	}
	for _, id := range soundChecks {
		if !held[id] {
			t.Errorf("check %s holds no Hamilton source", id)
		}
	}
}
