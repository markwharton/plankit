package protect

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/paths"
)

func scratch(t *testing.T, configured bool) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.Exec(dir, "init", "-q", "-b", "main"); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(paths.Plans(dir), 0o755)
	if configured {
		if err := config.Write(dir, config.Default("main")); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func runProtect(t *testing.T, dir, filePath string) string {
	t.Helper()
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	payload, _ := json.Marshal(map[string]any{
		"cwd":        dir,
		"tool_input": map[string]string{"file_path": filePath},
	})
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "protect"}, []*cli.Command{Cmd}, bytes.NewReader(payload), &out, &errw)
	if code != 0 {
		t.Fatalf("hook exit %d", code)
	}
	return out.String()
}

func TestDeniesWritesUnderPlans(t *testing.T) {
	dir := scratch(t, true)
	for _, p := range []string{
		filepath.Join(dir, "docs", "plans", "2026-01-01-001-x.md"),
		"docs/plans/2026-01-01-001-x.md",
		"docs/plans/sub/y.md",
	} {
		out := runProtect(t, dir, p)
		if !strings.Contains(out, `"permissionDecision":"deny"`) || !strings.Contains(out, "immutable") {
			t.Errorf("path %q should be denied, got %q", p, out)
		}
	}
}

func TestAllowsEverythingElse(t *testing.T) {
	dir := scratch(t, true)
	for _, p := range []string{
		filepath.Join(dir, "docs", "notes.md"),
		"docs/plans.md",
		"docs/plansdir/x.md",
		"README.md",
	} {
		if out := runProtect(t, dir, p); out != "" {
			t.Errorf("path %q should be allowed, got %q", p, out)
		}
	}
}

func TestUnconfiguredRepoIsSilent(t *testing.T) {
	dir := scratch(t, false)
	out := runProtect(t, dir, "docs/plans/x.md")
	if out != "" {
		t.Fatalf("unconfigured must no-op: %s", out)
	}
}

func TestNoFilePathIsSilent(t *testing.T) {
	dir := scratch(t, true)
	var out bytes.Buffer
	payload := []byte(`{"cwd":"` + dir + `","tool_input":{"command":"ls"}}`)
	cli.RunIO([]string{"pk", "protect"}, []*cli.Command{Cmd}, bytes.NewReader(payload), &out, &out)
	if out.Len() != 0 {
		t.Fatalf("out = %s", out.String())
	}
}

func TestSymlinkIntoPlansIsDenied(t *testing.T) {
	dir := scratch(t, true)
	target := filepath.Join(paths.Plans(dir), "real.md")
	os.WriteFile(target, []byte("# x\n"), 0o644)
	link := filepath.Join(dir, "alias.md")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if out := runProtect(t, dir, link); !strings.Contains(out, "deny") {
		t.Fatalf("symlink bypass: %q", out)
	}
}

// Regression for the macOS failure: /var is a symlink to /private/var,
// so the payload cwd arrives in symlinked form while the (nonexistent)
// write target cannot be symlink-resolved. Reproduced portably with a
// symlinked repo root: the plans dir resolves to the real path, the
// missing target must resolve with it, or the prefixes never match.
func TestSymlinkedRootStillDenied(t *testing.T) {
	real := scratch(t, true)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	payload, _ := json.Marshal(map[string]any{
		"cwd":        link,
		"tool_input": map[string]string{"file_path": "docs/plans/2026-01-01-001-new.md"},
	})
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	var out, errw bytes.Buffer
	cli.RunIO([]string{"pk", "protect"}, []*cli.Command{Cmd}, bytes.NewReader(payload), &out, &errw)
	if !strings.Contains(out.String(), `"permissionDecision":"deny"`) {
		t.Fatalf("symlinked root bypassed protect: %q", out.String())
	}
}
