package main

import (
	"strings"
	"testing"
)

// tableRows returns the cells of the first markdown table after the line
// that starts with marker: the header row first, then each body row. The
// separator row (|---|) is left out. Cells are trimmed, and a cell that
// is one backquoted word keeps its backquotes. ok is false when no line
// starts with marker, or no table follows it.
func tableRows(text, marker string) (rows [][]string, ok bool) {
	lines := strings.Split(text, "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, marker) {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, false
	}
	inTable := false
	for _, l := range lines[start+1:] {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "|") {
			if inTable {
				break
			}
			continue
		}
		inTable = true
		cells := strings.Split(strings.Trim(t, "|"), "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if isSeparator(cells) {
			continue
		}
		rows = append(rows, cells)
	}
	return rows, len(rows) > 0
}

func isSeparator(cells []string) bool {
	for _, c := range cells {
		if strings.Trim(c, "-: ") != "" {
			return false
		}
	}
	return true
}

func TestTableRowsReadsOneTable(t *testing.T) {
	text := "intro\n\n## Here\n\nprose\n\n| A | B |\n|---|:-:|\n| `x` | 1 |\n| y | 2 |\n\nafter\n\n| Z | Z |\n"
	rows, ok := tableRows(text, "## Here")
	want := [][]string{{"A", "B"}, {"`x`", "1"}, {"y", "2"}}
	if !ok || len(rows) != len(want) {
		t.Fatalf("tableRows: got %q, ok %v", rows, ok)
	}
	for i := range want {
		if strings.Join(rows[i], "|") != strings.Join(want[i], "|") {
			t.Errorf("row %d: got %q, want %q", i, rows[i], want[i])
		}
	}
	if _, ok := tableRows(text, "## Missing"); ok {
		t.Error("tableRows: a missing marker must report not ok")
	}
}
