package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// hasTable parses markdown with the site's table extension and reports
// whether the tree holds a table. It reads the structure, not the text,
// so a pipe in prose or in a code block is not mistaken for a table.
func hasTable(src []byte) bool {
	doc := goldmark.New(goldmark.WithExtensions(extension.Table)).Parser().Parse(text.NewReader(src))
	found := false
	gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering && n.Kind() == extast.KindTable {
			found = true
			return gast.WalkStop, nil
		}
		return gast.WalkContinue, nil
	})
	return found
}

// Tables render on the site's docs pages, but pk help reads skills
// through the IR, which has no table, so a table in a skill would show
// in the terminal as raw pipes. No skill may hold one.
func TestNoSkillHoldsATable(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "skills", "*", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("found no skills under ../../skills; this test reads skills/<topic>/SKILL.md")
	}
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if hasTable(src) {
			t.Errorf("%s holds a table, which pk help cannot render; use a list, or move the table to a docs/ page", p)
		}
	}
}

// The check's instrument is hasTable: it must find a table, and must
// not find one in a code block or in prose that uses a pipe.
func TestHasTableReadsTheStructure(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want bool
	}{
		{"a table", "| a | b |\n|---|---|\n| 1 | 2 |\n", true},
		{"a pipe in prose", "Use `a | b` to pipe, or a || b.\n", false},
		{"a table in a code block", "```\n| a | b |\n|---|---|\n```\n", false},
		{"plain text", "# Title\n\nNo table here.\n", false},
	} {
		if got := hasTable([]byte(tc.src)); got != tc.want {
			t.Errorf("%s: hasTable = %v, want %v", tc.name, got, tc.want)
		}
	}
}
