package help

import (
	"strings"
	"testing"
	"time"
)

func fixedDate(t *testing.T) {
	t.Helper()
	old := now
	now = func() time.Time { return time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = old })
}

func doc(blocks ...Block) *Doc {
	return &Doc{Schema: 1, Meta: Meta{Name: "ship", Description: "Cut the pending work"}, Blocks: blocks}
}

func TestManPageHasHeaderAndName(t *testing.T) {
	fixedDate(t)
	out, err := RenderMan(doc(para(txt("body"))))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, `.TH PK-SHIP 1 "2026-09-12"`) {
		t.Fatalf("TH line: %q", firstLine(out))
	}
	if !strings.Contains(out, ".SH NAME\nship \\- Cut the pending work\n") {
		t.Fatalf("NAME section missing:\n%s", out)
	}
}

// The leading H1 repeats the NAME section on a single page, so it is
// dropped and H2 becomes the section heading.
func TestSinglePageDropsH1AndPromotesH2(t *testing.T) {
	fixedDate(t)
	out, err := RenderMan(doc(
		Block{Type: "heading", Level: 1, ID: "a", Inlines: []Span{txt("pk ship")}},
		Block{Type: "heading", Level: 2, ID: "b", Inlines: []Span{txt("Usage")}},
	))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "PK SHIP\n") {
		t.Fatalf("H1 should not become a section:\n%s", out)
	}
	if !strings.Contains(out, ".SH USAGE\n") {
		t.Fatalf("H2 should be the section:\n%s", out)
	}
}

func TestEscaping(t *testing.T) {
	fixedDate(t)
	out, err := RenderMan(doc(
		para(txt(`a-b`)),
		para(txt(`back\slash`)),
		para(txt(`.leading dot`)),
		para(txt(`'leading quote`)),
	))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`a\-b`, `back\eslash`, `\&.leading`, `\&'leading`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestSpansAndCode(t *testing.T) {
	fixedDate(t)
	out, err := RenderMan(doc(para(
		Span{Type: "text", Text: "bold", Strong: true},
		txt(" "),
		Span{Type: "text", Text: "it", Em: true},
		txt(" "),
		Span{Type: "code", Text: "pk ship"},
	)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`\fBbold\fR`, `\fIit\fR`, `\fBpk ship\fR`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestCodeblockIsNoFill(t *testing.T) {
	fixedDate(t)
	out, err := RenderMan(doc(Block{Type: "codeblock", Text: "pk ship\npk status\n"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, ".nf\npk ship\npk status\n.fi\n") {
		t.Fatalf("code block:\n%s", out)
	}
}

func TestListBullets(t *testing.T) {
	fixedDate(t)
	out, err := RenderMan(doc(Block{Type: "list", Items: [][]Block{
		{para(txt("first"))},
		{para(txt("second"))},
	}}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(out, `.IP "\(bu" 4`) != 2 {
		t.Fatalf("bullets:\n%s", out)
	}
	if strings.Contains(out, ".IP \"\\(bu\" 4\n.PP\n") {
		t.Fatalf("an item's first paragraph must not reopen with .PP:\n%s", out)
	}
}

// An IR node this writer does not know must fail loudly: that is what
// makes the roff writer a check on the schema rather than a third place
// to forget.
func TestUnknownNodeTypesAreErrors(t *testing.T) {
	fixedDate(t)
	if _, err := RenderMan(doc(Block{Type: "table"})); err == nil {
		t.Fatal("unknown block type rendered without error")
	}
	if _, err := RenderMan(doc(para(Span{Type: "footnote", Text: "x"}))); err == nil {
		t.Fatal("unknown span type rendered without error")
	}
}

// mandoc reports "skipping paragraph macro: PP after SH": a section
// heading has already opened a paragraph.
func TestNoParagraphMacroAfterAHeading(t *testing.T) {
	fixedDate(t)
	metas, err := Topics()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range metas {
		d, _, _ := Topic(m.Name)
		out, err := RenderMan(d)
		if err != nil {
			t.Fatalf("topic %s: %v", m.Name, err)
		}
		lines := strings.Split(out, "\n")
		for i, line := range lines[:len(lines)-1] {
			if !strings.HasPrefix(line, ".SH ") && !strings.HasPrefix(line, ".SS ") {
				continue
			}
			if lines[i+1] == ".PP" {
				t.Fatalf("topic %s: .PP after %q", m.Name, line)
			}
		}
	}
}

// mandoc reports a source line over 80 bytes as a style problem, and a
// man page is read as source as well as rendered.
func TestSourceLinesFitEightyBytes(t *testing.T) {
	fixedDate(t)
	metas, err := Topics()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range metas {
		d, _, _ := Topic(m.Name)
		out, err := RenderMan(d)
		if err != nil {
			t.Fatalf("topic %s: %v", m.Name, err)
		}
		noFill := false
		for _, line := range strings.Split(out, "\n") {
			switch line {
			case ".nf":
				noFill = true
				continue
			case ".fi":
				noFill = false
				continue
			}
			// A code block is copied as authored, and its width is the
			// author's business: mandoc does not apply the check inside
			// .nf either. The .TH line is one field per quoted argument
			// and cannot wrap.
			if noFill || strings.HasPrefix(line, ".TH ") {
				continue
			}
			if len(line) > 80 {
				t.Errorf("topic %s: %d-byte line: %.60s...", m.Name, len(line), line)
			}
		}
	}
}

// The H1 goes into NAME, so the prose that followed it needs a section
// of its own rather than being left under NAME.
func TestLeadProseBecomesDescription(t *testing.T) {
	fixedDate(t)
	out, err := RenderMan(doc(
		Block{Type: "heading", Level: 1, ID: "a", Inlines: []Span{txt("pk ship")}},
		para(txt("lead prose")),
		Block{Type: "heading", Level: 2, ID: "b", Inlines: []Span{txt("Usage")}},
	))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, ".SH DESCRIPTION\nlead prose\n") {
		t.Fatalf("lead prose is not under DESCRIPTION:\n%s", out)
	}
	// A page whose first block is a heading owes no DESCRIPTION.
	out, err = RenderMan(doc(Block{Type: "heading", Level: 2, ID: "b", Inlines: []Span{txt("Usage")}}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "DESCRIPTION") {
		t.Fatalf("empty DESCRIPTION emitted:\n%s", out)
	}
}

func TestEveryTopicRendersAsMan(t *testing.T) {
	fixedDate(t)
	metas, err := Topics()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range metas {
		d, _, ok := Topic(m.Name)
		if !ok {
			t.Fatalf("topic %s not embedded", m.Name)
		}
		out, err := RenderMan(d)
		if err != nil {
			t.Fatalf("topic %s: %v", m.Name, err)
		}
		if !strings.Contains(out, ".SH NAME") {
			t.Fatalf("topic %s has no NAME section", m.Name)
		}
	}
}

func TestDocumentManHasOneHeader(t *testing.T) {
	fixedDate(t)
	metas, err := Topics()
	if err != nil {
		t.Fatal(err)
	}
	docs := make([]*Doc, 0, len(metas))
	for _, m := range metas {
		d, _, _ := Topic(m.Name)
		docs = append(docs, d)
	}
	out, err := RenderManDocument(docs)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(out, ".TH ") != 1 {
		t.Fatalf("one .TH for the whole document, got %d", strings.Count(out, ".TH "))
	}
	if !strings.Contains(out, ".SH OVERVIEW\n") {
		t.Fatalf("each page should be a section:\n%s", out[:400])
	}
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}
