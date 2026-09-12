package help

import (
	"bytes"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/cli"
)

func runHelp(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errw bytes.Buffer
	code := cli.RunIO(append([]string{"pk", "help"}, args...), []*cli.Command{Cmd}, nil, &out, &errw)
	return code, out.String(), errw.String()
}

func TestNonTTYGetsRawAuthoredBytes(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")
	_, raw, ok := Topic("overview")
	if !ok {
		t.Fatal("plankit topic not embedded")
	}
	code, out, _ := runHelp(t, "overview")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if out != string(raw) {
		t.Fatalf("pipe output is not the authored bytes\ngot:  %q\nwant: %q", out[:40], string(raw)[:40])
	}
	if !strings.HasPrefix(out, "---\nname: overview\n") {
		t.Fatalf("raw output should include frontmatter, got %q", out[:30])
	}
}

func TestForcedColorRendersIR(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	code, out, _ := runHelp(t, "overview")
	if code != cli.ExitOK || !strings.Contains(out, "\x1b[1;4m") {
		t.Fatalf("code=%d, want rendered ANSI, got %q", code, out[:40])
	}
	if strings.Contains(out, "# plankit") {
		t.Fatal("rendered output should not contain markdown syntax")
	}
}

func TestTOCListsTopicsOverviewFirst(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "")
	code, out, _ := runHelp(t)
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"overview", "craft", "help", "version"} {
		if !strings.Contains(out, want) {
			t.Errorf("TOC missing %q", want)
		}
	}
	if strings.Index(out, "overview") > strings.Index(out, "craft") || strings.Index(out, "craft") > strings.Index(out, "brief") {
		t.Fatalf("documents lead, then commands:\n%s", out)
	}
	if !strings.Contains(out, "prompt\n\n  brief") {
		t.Fatalf("a blank line separates documents from commands:\n%s", out)
	}
	if strings.Index(out, "overview") > strings.Index(out, "version") {
		t.Fatal("overview should be pinned first")
	}
}

func TestUnknownTopicIsUsageErrorWithHint(t *testing.T) {
	code, _, errw := runHelp(t, "bogus")
	if code != cli.ExitUsage {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(errw, `unknown help topic "bogus"`) || !strings.Contains(errw, "available topics:") {
		t.Fatalf("stderr = %q", errw)
	}
}

func TestDocumentIsEveryPageInOrder(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "")
	code, out, _ := runHelp(t, Document)
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	metas, err := Topics()
	if err != nil {
		t.Fatal(err)
	}
	at := -1
	// The first page opens the document, so there is no newline before
	// its heading; search a copy that has one.
	hay := "\n" + out
	for _, m := range metas {
		i := strings.Index(hay, "\n# "+heading(t, m.Name))
		if i < 0 {
			t.Fatalf("document is missing %s", m.Name)
		}
		if i < at {
			t.Fatalf("%s is out of index order", m.Name)
		}
		at = i
	}
	if strings.Contains(out, "\nname: ") {
		t.Fatal("frontmatter reached the document")
	}
	if got, want := strings.Count(out, "\n---\n"), len(metas)-1; got != want {
		t.Fatalf("%d rules between %d pages, want %d", got, len(metas), want)
	}
}

// heading is the page's own H1 text: "overview" for a document, "pk
// ship" for a command page.
func heading(t *testing.T, name string) string {
	t.Helper()
	d, _, ok := Topic(name)
	if !ok || len(d.Blocks) == 0 {
		t.Fatalf("topic %s has no blocks", name)
	}
	var sb strings.Builder
	for _, s := range d.Blocks[0].Inlines {
		sb.WriteString(s.Text)
	}
	return sb.String()
}

// The document takes the same fork as a topic: the terminal probe
// decides rendered against source, and a flag would outrank both.
func TestDocumentRendersForATerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	code, out, _ := runHelp(t, Document)
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "\x1b[1;4m") {
		t.Fatalf("forced color should render the pages: %q", out[:60])
	}
	if strings.Contains(out, "\n# overview") {
		t.Fatal("rendered output should not carry markdown syntax")
	}
}

// The reserved name must shadow no page: TestCommandsAndSkillsAreOneToOne
// pairs commands with skills and would not notice.
func TestDocumentShadowsNoTopic(t *testing.T) {
	if _, _, ok := Topic(Document); ok {
		t.Fatalf("a topic named %q would be unreachable", Document)
	}
}

func TestManFormatRendersRoff(t *testing.T) {
	code, out, _ := runHelp(t, "ship", "--format", "man")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(out, ".TH PK-SHIP 1 ") {
		t.Fatalf("not a man page: %q", out[:40])
	}
}

func TestUnknownFormatNamesTheSet(t *testing.T) {
	code, _, errw := runHelp(t, "ship", "--format", "yaml")
	if code != cli.ExitUsage {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(errw, "invalid --format") || !strings.Contains(errw, "text or man") {
		t.Fatalf("stderr = %q", errw)
	}
}

func TestEveryEmbeddedTopicRenders(t *testing.T) {
	metas, err := Topics()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range metas {
		d, raw, ok := Topic(m.Name)
		if !ok || len(raw) == 0 {
			t.Fatalf("topic %s incomplete", m.Name)
		}
		if out := Render(d, cli.StyleANSI, 80); len(out) == 0 {
			t.Fatalf("topic %s rendered empty", m.Name)
		}
	}
}
