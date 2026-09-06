package help

import (
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/cli"
)

func para(spans ...Span) Block { return Block{Type: "para", Inlines: spans} }
func txt(s string) Span        { return Span{Type: "text", Text: s} }

func TestWrapPlain(t *testing.T) {
	d := &Doc{Schema: 1, Meta: Meta{Name: "t", Description: "t"}, Blocks: []Block{
		para(txt("alpha beta gamma delta epsilon")),
	}}
	got := Render(d, cli.StyleNone, 12)
	want := "alpha beta\ngamma delta\nepsilon\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWidthZeroIsOneLine(t *testing.T) {
	d := &Doc{Blocks: []Block{para(txt("alpha beta gamma"))}}
	if got := Render(d, cli.StyleNone, 0); got != "alpha beta gamma\n" {
		t.Fatalf("got %q", got)
	}
}

func TestHeadingAndInlineStyles(t *testing.T) {
	d := &Doc{Blocks: []Block{
		{Type: "heading", Level: 1, ID: "x", Inlines: []Span{txt("Title")}},
		para(Span{Type: "text", Text: "bold", Strong: true}, Span{Type: "code", Text: "pk"}),
	}}
	got := Render(d, cli.StyleANSI, 0)
	for _, want := range []string{"\x1b[1;4mTitle\x1b[0m", "\x1b[1mbold\x1b[0m", "\x1b[36mpk\x1b[0m"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	plain := Render(d, cli.StyleNone, 0)
	if strings.Contains(plain, "\x1b[") {
		t.Fatalf("plain render leaked escapes: %q", plain)
	}
}

func TestCodeblockIndent(t *testing.T) {
	d := &Doc{Blocks: []Block{{Type: "codeblock", Lang: "bash", Text: "pk init\npk status\n"}}}
	want := "    pk init\n    pk status\n"
	if got := Render(d, cli.StyleNone, 80); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestListBulletsAndHangingIndent(t *testing.T) {
	d := &Doc{Blocks: []Block{{
		Type: "list",
		Items: [][]Block{
			{para(txt("first item wraps onto more"))},
			{para(txt("second"))},
		},
	}}}
	got := Render(d, cli.StyleNone, 18)
	want := "  - first item\n    wraps onto\n    more\n  - second\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestOrderedListNumbers(t *testing.T) {
	d := &Doc{Blocks: []Block{{
		Type:    "list",
		Ordered: true,
		Items:   [][]Block{{para(txt("one"))}, {para(txt("two"))}},
	}}}
	got := Render(d, cli.StyleNone, 40)
	if !strings.Contains(got, "  1. one") || !strings.Contains(got, "  2. two") {
		t.Fatalf("got %q", got)
	}
}

func TestLinkAppendsURL(t *testing.T) {
	d := &Doc{Blocks: []Block{para(Span{Type: "text", Text: "the docs", URL: "https://x.test/d"})}}
	got := Render(d, cli.StyleNone, 0)
	if got != "the docs (https://x.test/d)\n" {
		t.Fatalf("got %q", got)
	}
	// Autolink-style spans, text == url, must not double up.
	d = &Doc{Blocks: []Block{para(Span{Type: "text", Text: "https://x.test", URL: "https://x.test"})}}
	if got := Render(d, cli.StyleNone, 0); got != "https://x.test\n" {
		t.Fatalf("got %q", got)
	}
}

func TestHighlightRuns(t *testing.T) {
	d := &Doc{Blocks: []Block{{Type: "codeblock", Runs: []Run{
		{Kind: "kw", Text: "func"}, {Text: " main() {\n"}, {Kind: "str", Text: "\"hi\""}, {Text: "\n}"},
	}}}}
	got := Render(d, cli.StyleANSI, 80)
	for _, want := range []string{"\x1b[1mfunc\x1b[0m", "\x1b[32m\"hi\"\x1b[0m"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if lines := strings.Count(got, "\n"); lines != 3 {
		t.Fatalf("want 3 lines, got %d in %q", lines, got)
	}
}

// TestInlineCodeKeepsItsNeighbors pins that a code span glues to the
// punctuation and letters around it exactly as the source has them,
// and gets a space only where the source had one.
func TestInlineCodeKeepsItsNeighbors(t *testing.T) {
	d := &Doc{Blocks: []Block{{Type: "para", Inlines: []Span{
		{Type: "text", Text: "A breaking change ("},
		{Type: "code", Text: "type!:"},
		{Type: "text", Text: " or a "},
		{Type: "code", Text: "BREAKING"},
		{Type: "text", Text: " footer) is "},
		{Type: "code", Text: "pk release"},
		{Type: "text", Text: "'s moment."},
	}}}}
	want := "A breaking change (type!: or a BREAKING footer) is pk release's moment.\n"
	if got := Render(d, cli.StyleNone, 0); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	// Two code spans separated only by whitespace still get the space.
	d = &Doc{Blocks: []Block{{Type: "para", Inlines: []Span{
		{Type: "code", Text: "a"}, {Type: "text", Text: " "}, {Type: "code", Text: "b"},
	}}}}
	if got := Render(d, cli.StyleNone, 0); got != "a b\n" {
		t.Fatalf("got %q, want %q", got, "a b\n")
	}
}
