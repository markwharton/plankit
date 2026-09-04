package cli

import (
	"bytes"
	"strings"
	"testing"
)

func testCmd(name string, hook bool, flags []FlagSpec, run func(*Context) error) *Command {
	if run == nil {
		run = func(*Context) error { return nil }
	}
	return &Command{Name: name, Summary: "Test " + name, Hook: hook, Flags: flags, Run: run}
}

func run(t *testing.T, argv []string, cmds ...*Command) (code int, stdout, stderr string) {
	t.Helper()
	var out, err bytes.Buffer
	code = RunIO(append([]string{"pk"}, argv...), cmds, nil, &out, &err)
	return code, out.String(), err.String()
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	code, _, stderr := run(t, []string{"bogus"}, testCmd("version", false, nil, nil))
	if code != ExitUsage {
		t.Fatalf("exit = %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(stderr, `unknown command "bogus"`) {
		t.Fatalf("stderr = %q, want unknown command message", stderr)
	}
	if !strings.Contains(stderr, "Commands:") {
		t.Fatalf("stderr should include usage, got %q", stderr)
	}
}

func TestNoArgsPrintsUsage(t *testing.T) {
	code, _, stderr := run(t, nil, testCmd("version", false, nil, nil))
	if code != ExitUsage || !strings.Contains(stderr, "Usage: pk <command>") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestDeclaredFlagsReachContext(t *testing.T) {
	var got string
	var flagged bool
	cmd := testCmd("demo", false, []FlagSpec{
		{Name: "file", Type: StringFlag, Usage: "A file"},
		{Name: "force", Type: BoolFlag, Usage: "Force"},
	}, func(c *Context) error {
		got = c.String("file")
		flagged = c.Bool("force")
		return nil
	})
	code, _, _ := run(t, []string{"demo", "--file", "x.txt", "--force", "rest"}, cmd)
	if code != ExitOK || got != "x.txt" || !flagged {
		t.Fatalf("code=%d file=%q force=%v", code, got, flagged)
	}
}

func TestPositionalArgs(t *testing.T) {
	var args []string
	cmd := testCmd("demo", false, nil, func(c *Context) error { args = c.Args(); return nil })
	run(t, []string{"demo", "a", "b"}, cmd)
	if len(args) != 2 || args[0] != "a" || args[1] != "b" {
		t.Fatalf("args = %v", args)
	}
}

func TestExitCodeTaxonomy(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, ExitOK},
		{Usagef("bad flag"), ExitUsage},
		{Statef("not configured"), ExitState},
		{errAny, ExitInternal},
	}
	for _, tc := range cases {
		cmd := testCmd("demo", false, nil, func(*Context) error { return tc.err })
		code, _, _ := run(t, []string{"demo"}, cmd)
		if code != tc.want {
			t.Errorf("err=%v: exit=%d want %d", tc.err, code, tc.want)
		}
	}
}

var errAny = &anyError{}

type anyError struct{}

func (*anyError) Error() string { return "boom" }

func TestHintPrintsUnlessQuiet(t *testing.T) {
	cmd := testCmd("demo", false, nil, func(*Context) error {
		return WithHint(Statef("no .pk.json"), "run pk init")
	})
	_, _, stderr := run(t, []string{"demo"}, cmd)
	if !strings.Contains(stderr, "Hint: run pk init") {
		t.Fatalf("stderr = %q, want hint", stderr)
	}
	_, _, stderr = run(t, []string{"demo", "--quiet"}, cmd)
	if strings.Contains(stderr, "Hint:") {
		t.Fatalf("stderr = %q, want no hint under --quiet", stderr)
	}
}

func TestBadFlagShowsKebabUsage(t *testing.T) {
	cmd := testCmd("demo", false, []FlagSpec{{Name: "file", Type: StringFlag, Usage: "A file"}}, nil)
	code, _, stderr := run(t, []string{"demo", "--nope"}, cmd)
	if code != ExitUsage {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"--file <value>", "--project-dir", "Universal flags:"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q:\n%s", want, stderr)
		}
	}
}

func TestHelpFlagExitsZero(t *testing.T) {
	cmd := testCmd("demo", false, nil, func(*Context) error { return errAny })
	code, stdout, _ := run(t, []string{"demo", "--help"}, cmd)
	if code != ExitOK || !strings.Contains(stdout, "Usage: pk demo") {
		t.Fatalf("code=%d stdout=%q", code, stdout)
	}
}

func TestFormatValidation(t *testing.T) {
	cmd := testCmd("demo", false, nil, nil)
	code, _, stderr := run(t, []string{"demo", "--format", "yaml"}, cmd)
	if code != ExitUsage || !strings.Contains(stderr, "invalid --format") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestProjectDirResolution(t *testing.T) {
	var got string
	cmd := testCmd("demo", false, nil, func(c *Context) error { got = c.ProjectDir; return nil })

	t.Setenv("PK_PROJECT_DIR", "")
	run(t, []string{"demo", "--project-dir", "/tmp"}, cmd)
	if got != "/tmp" {
		t.Fatalf("flag: got %q", got)
	}

	t.Setenv("PK_PROJECT_DIR", "/var")
	run(t, []string{"demo"}, cmd)
	if got != "/var" {
		t.Fatalf("env: got %q", got)
	}

	run(t, []string{"demo", "--project-dir", "/tmp"}, cmd)
	if got != "/tmp" {
		t.Fatalf("flag over env: got %q", got)
	}
}

func TestPresentationOnBuffers(t *testing.T) {
	// Buffers are not TTYs: no style, no wrapping, unless forced.
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "")
	var seen Style
	var width int
	cmd := testCmd("demo", false, nil, func(c *Context) error { seen, width = c.Style, c.Width; return nil })

	run(t, []string{"demo"}, cmd)
	if seen != StyleNone || width != 0 {
		t.Fatalf("default buffer: style=%v width=%d", seen, width)
	}

	t.Setenv("CLICOLOR_FORCE", "1")
	run(t, []string{"demo"}, cmd)
	if seen != StyleANSI {
		t.Fatalf("CLICOLOR_FORCE: style=%v", seen)
	}

	t.Setenv("NO_COLOR", "1")
	run(t, []string{"demo"}, cmd)
	if seen != StyleNone {
		t.Fatalf("NO_COLOR wins: style=%v", seen)
	}

	t.Setenv("NO_COLOR", "")
	run(t, []string{"demo", "--plain"}, cmd)
	if seen != StyleNone || width != 0 {
		t.Fatalf("--plain: style=%v width=%d", seen, width)
	}
}

func TestHookCommandUsage(t *testing.T) {
	cmd := testCmd("guard", true, nil, nil)
	code, stdout, _ := run(t, []string{"guard", "--help"}, cmd)
	if code != ExitOK || !strings.Contains(stdout, "reads JSON on stdin") {
		t.Fatalf("code=%d stdout=%q", code, stdout)
	}
}
