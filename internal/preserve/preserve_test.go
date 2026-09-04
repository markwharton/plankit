package preserve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/paths"
)

const planBody = "# Ship The Widget\n\nContext: enough substance to clear the minimum plan size threshold.\n"

func fixedNow(t *testing.T) {
	t.Helper()
	old := now
	now = func() time.Time { return time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = old })
}

// scratch builds a configured repo (preserve mode as given) plus a fake
// Claude plans dir holding one approved plan; returns repo and plan path.
func scratch(t *testing.T, mode, content string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@t"}, {"config", "user.name", "t"},
	} {
		if _, err := git.Exec(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644)
	git.Exec(dir, "add", ".")
	git.Exec(dir, "commit", "-q", "-m", "first")
	cfg := config.Default("main")
	cfg.Preserve.Mode = mode
	if err := config.Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	git.Exec(dir, "add", ".pk.json")
	git.Exec(dir, "commit", "-q", "-m", "config")

	claudePlans := filepath.Join(t.TempDir(), ".claude", "plans")
	os.MkdirAll(claudePlans, 0o755)
	planPath := filepath.Join(claudePlans, "widget.md")
	os.WriteFile(planPath, []byte(content), 0o644)
	return dir, planPath
}

func runPreserve(t *testing.T, dir, planPath string, args ...string) (string, string) {
	t.Helper()
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	var stdin io.Reader
	if planPath != "" {
		payload, _ := json.Marshal(map[string]any{
			"cwd":           dir,
			"tool_response": map[string]string{"filePath": planPath},
		})
		stdin = bytes.NewReader(payload)
	}
	var out, errw bytes.Buffer
	argv := append([]string{"pk", "preserve", "--project-dir", dir}, args...)
	code := cli.RunIO(argv, []*cli.Command{Cmd}, stdin, &out, &errw)
	if code != 0 {
		t.Fatalf("hook exit %d (stderr: %s)", code, errw.String())
	}
	return out.String(), errw.String()
}

func commitCount(t *testing.T, dir string) int {
	out, err := git.Exec(dir, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	var n int
	fmt.Sscanf(out, "%d", &n)
	return n
}

func TestAutoModePreservesAndCommits(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	before := commitCount(t, dir)

	out, _ := runPreserve(t, dir, plan)

	dest := filepath.Join(paths.Plans(dir), "2026-09-05-001-ship-the-widget.md")
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("preserved file: %v", err)
	}
	if !bytes.Equal(got, []byte(planBody)) {
		t.Fatal("preserved bytes differ from the approved plan")
	}
	if commitCount(t, dir) != before+1 {
		t.Fatal("expected one new commit")
	}
	subject, _ := git.Exec(dir, "log", "-1", "--format=%s")
	if subject != "plan: Ship The Widget [skip ci]" {
		t.Fatalf("subject = %q", subject)
	}
	if !strings.Contains(out, "Approved plan committed: docs/plans/2026-09-05-001-ship-the-widget.md") {
		t.Fatalf("systemMessage: %s", out)
	}
}

func TestDuplicateContentIsNotRecommitted(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	runPreserve(t, dir, plan)
	before := commitCount(t, dir)

	out, _ := runPreserve(t, dir, plan)
	if commitCount(t, dir) != before {
		t.Fatal("duplicate plan created a commit")
	}
	if !strings.Contains(out, "already preserved") {
		t.Fatalf("out = %s", out)
	}
	entries, _ := os.ReadDir(paths.Plans(dir))
	if len(entries) != 1 {
		t.Fatalf("plans dir has %d entries", len(entries))
	}
}

func TestSequenceIncrementsWithinADay(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	runPreserve(t, dir, plan)

	second := strings.Replace(planBody, "Ship The Widget", "Refine The Widget", 1)
	os.WriteFile(plan, []byte(second), 0o644)
	runPreserve(t, dir, plan)

	if _, err := os.Stat(filepath.Join(paths.Plans(dir), "2026-09-05-002-refine-the-widget.md")); err != nil {
		t.Fatalf("second plan: %v", err)
	}
}

func TestManualModeWritesPointerThenExplicitRunCommits(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "manual", planBody)
	before := commitCount(t, dir)

	out, _ := runPreserve(t, dir, plan)
	if commitCount(t, dir) != before {
		t.Fatal("manual mode must not commit")
	}
	if !strings.Contains(out, "/plankit:preserve") || !strings.Contains(out, "additionalContext") {
		t.Fatalf("manual response: %s", out)
	}
	ptr, err := os.ReadFile(filepath.Join(dir, ".git", "pk-pending-plan"))
	if err != nil || strings.TrimSpace(string(ptr)) != plan {
		t.Fatalf("pointer: %q err=%v", ptr, err)
	}

	// Explicit invocation: no stdin payload, consumes the pointer.
	out, _ = runPreserve(t, dir, "")
	if commitCount(t, dir) != before+1 {
		t.Fatal("explicit run should commit")
	}
	if !strings.Contains(out, "Approved plan committed") {
		t.Fatalf("out = %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "pk-pending-plan")); !os.IsNotExist(err) {
		t.Fatal("pointer should be consumed")
	}
}

func TestStalePointerIsRemoved(t *testing.T) {
	dir, _ := scratch(t, "manual", planBody)
	ptr := filepath.Join(dir, ".git", "pk-pending-plan")
	os.WriteFile(ptr, []byte("/nowhere/gone.md\n"), 0o644)
	runPreserve(t, dir, "")
	if _, err := os.Stat(ptr); !os.IsNotExist(err) {
		t.Fatal("stale pointer should be deleted")
	}
}

func TestOffModeDoesNothing(t *testing.T) {
	dir, plan := scratch(t, "off", planBody)
	before := commitCount(t, dir)
	out, _ := runPreserve(t, dir, plan)
	if out != "" || commitCount(t, dir) != before {
		t.Fatalf("off mode acted: %s", out)
	}
}

func TestUnconfiguredRepoIsSilent(t *testing.T) {
	dir, plan := scratch(t, "auto", planBody)
	os.Remove(config.Path(dir))
	out, _ := runPreserve(t, dir, plan)
	if out != "" {
		t.Fatalf("unconfigured must no-op: %s", out)
	}
}

func TestShortPlanIsSkipped(t *testing.T) {
	dir, plan := scratch(t, "auto", "# Tiny\n")
	out, _ := runPreserve(t, dir, plan)
	if out != "" {
		t.Fatalf("short plan preserved: %s", out)
	}
}

func TestCRLFPlanKeepsCleanTitle(t *testing.T) {
	fixedNow(t)
	crlf := "# Windows Plan\r\n\r\nEnough body content to clear the minimum size threshold easily.\r\n"
	dir, plan := scratch(t, "auto", crlf)
	runPreserve(t, dir, plan)
	if _, err := os.Stat(filepath.Join(paths.Plans(dir), "2026-09-05-001-windows-plan.md")); err != nil {
		t.Fatalf("CRLF slug: %v", err)
	}
	subject, _ := git.Exec(dir, "log", "-1", "--format=%s")
	if subject != "plan: Windows Plan [skip ci]" {
		t.Fatalf("subject carries CR? %q", subject)
	}
}

func TestDryRunPreviewsOnly(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	before := commitCount(t, dir)
	_, errw := runPreserve(t, dir, plan, "--dry-run")
	if !strings.Contains(errw, "2026-09-05-001-ship-the-widget.md") {
		t.Fatalf("preview: %s", errw)
	}
	if commitCount(t, dir) != before {
		t.Fatal("dry-run committed")
	}
	if _, err := os.Stat(paths.Plans(dir)); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote the plans dir")
	}
}

func TestExtractPlanPathHandlesWindowsEscapes(t *testing.T) {
	raw := json.RawMessage(`{"filePath":"C:\\Users\\m\\.claude\\plans\\p.md"}`)
	home := func() (string, error) { return "C:\\Users\\m", nil }
	if got := extractPlanPath(raw, home); got != "C:/Users/m/.claude/plans/p.md" {
		t.Fatalf("got %q", got)
	}
	legacy := json.RawMessage(`"Plan saved to /home/m/.claude/plans/x.md"`)
	if got := extractPlanPath(legacy, os.UserHomeDir); got != "/home/m/.claude/plans/x.md" {
		t.Fatalf("legacy form: %q", got)
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Ship The Widget", "ship-the-widget"},
		{"Fix: CRLF & paths (v2)", "fix-crlf-paths-v2"},
		{"  --- ", ""},
		{"Ünïcode Tïtle", "ünïcode-tïtle"},
	}
	for _, tc := range cases {
		if got := slugify(tc.in, 60); got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
