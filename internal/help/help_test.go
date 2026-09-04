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
	code := cli.RunIO(append([]string{"pk", "help"}, args...), []*cli.Command{Cmd}, &out, &errw)
	return code, out.String(), errw.String()
}

func TestNonTTYGetsRawAuthoredBytes(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")
	_, raw, ok := Topic("plankit")
	if !ok {
		t.Fatal("plankit topic not embedded")
	}
	code, out, _ := runHelp(t, "plankit")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if out != string(raw) {
		t.Fatalf("pipe output is not the authored bytes\ngot:  %q\nwant: %q", out[:40], string(raw)[:40])
	}
	if !strings.HasPrefix(out, "---\nname: plankit\n") {
		t.Fatalf("raw output should include frontmatter, got %q", out[:30])
	}
}

func TestForcedColorRendersIR(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	code, out, _ := runHelp(t, "plankit")
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
	for _, want := range []string{"plankit", "help", "version"} {
		if !strings.Contains(out, want) {
			t.Errorf("TOC missing %q", want)
		}
	}
	if strings.Index(out, "plankit") > strings.Index(out, "version") {
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
