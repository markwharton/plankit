package repo

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
)

func scratch(t *testing.T, commit bool) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q", "-b", "main")
	mustGit(t, dir, "config", "user.email", "t@t")
	mustGit(t, dir, "config", "user.name", "t")
	if commit {
		if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		mustGit(t, dir, "add", ".")
		mustGit(t, dir, "commit", "-q", "-m", "first")
	}
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, err := git.Exec(dir, args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errw bytes.Buffer
	code := cli.RunIO(append([]string{"pk"}, args...),
		[]*cli.Command{InitCmd, StatusCmd}, nil, &out, &errw)
	return code, out.String(), errw.String()
}

func TestInitThenStatusRoundTrips(t *testing.T) {
	dir := scratch(t, true)
	code, out, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitOK {
		t.Fatalf("init exit %d: %s%s", code, out, errw)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("written config does not load: %v", err)
	}
	if cfg.Release.Branch != "main" || len(cfg.Guard.Branches) != 1 || cfg.Guard.Branches[0] != "main" {
		t.Fatalf("defaults wrong: %+v", cfg)
	}
	if _, err := os.Stat(filepath.Join(dir, PlansDir)); !os.IsNotExist(err) {
		t.Fatal("init must not create docs/plans; preserve creates it on first use")
	}
	if _, err := os.Stat(filepath.Join(dir, config.FileName)); err != nil {
		t.Fatalf("plans dir: %v", err)
	}
	if got := git.LatestTag(dir); got != "v0.0.0" {
		t.Fatalf("baseline tag = %q", got)
	}
	// The conventions note lists the non-hidden types from the config
	// just written; the hidden plan type stays out.
	for _, want := range []string{"Conventional Commits", "feat, fix,", "changelog.types"} {
		if !strings.Contains(out, want) {
			t.Errorf("init output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "plan") && strings.Contains(out, "style, plan") {
		t.Errorf("hidden plan type leaked into the conventions note:\n%s", out)
	}

	code, out, _ = run(t, "status", "--project-dir", dir)
	if code != cli.ExitOK {
		t.Fatalf("status exit %d", code)
	}
	for _, want := range []string{"main (dirty)", "preserve", "manual", "block", "v0.0.0", "0 preserved"} {
		if !strings.Contains(out, want) {
			t.Errorf("status missing %q:\n%s", want, out)
		}
	}
}

func TestInitRefusesSecondRun(t *testing.T) {
	dir := scratch(t, true)
	run(t, "init", "--project-dir", dir)
	code, _, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitState || !strings.Contains(errw, "already configured") {
		t.Fatalf("code=%d stderr=%q", code, errw)
	}
}

func TestInitDryRunTouchesNothing(t *testing.T) {
	dir := scratch(t, true)
	code, out, _ := run(t, "init", "--project-dir", dir, "--dry-run")
	if code != cli.ExitOK || !strings.Contains(out, "would create") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	if _, err := os.Stat(config.Path(dir)); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote .pk.json")
	}
	if git.LatestTag(dir) != "" {
		t.Fatal("dry-run created a tag")
	}
}

func TestInitFlags(t *testing.T) {
	dir := scratch(t, true)
	code, _, _ := run(t, "init", "--project-dir", dir, "--release", "trunk", "--no-baseline")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	cfg, _ := config.Load(dir)
	if cfg.Release.Branch != "trunk" || cfg.Guard.Branches[0] != "trunk" {
		t.Fatalf("release flag not honored: %+v", cfg)
	}
	if git.LatestTag(dir) != "" {
		t.Fatal("--no-baseline still tagged")
	}
}

func TestInitEmptyRepoSkipsBaseline(t *testing.T) {
	dir := scratch(t, false)
	code, out, _ := run(t, "init", "--project-dir", dir)
	if code != cli.ExitOK || !strings.Contains(out, "no baseline tag") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	if git.LatestTag(dir) != "" {
		t.Fatal("tagged an empty repo")
	}
}

func TestStatusUnconfiguredExitsState(t *testing.T) {
	dir := scratch(t, true)
	code, out, errw := run(t, "status", "--project-dir", dir)
	if code != cli.ExitState {
		t.Fatalf("exit %d, want %d", code, cli.ExitState)
	}
	if !strings.Contains(out, "not configured") || !strings.Contains(errw, "pk init") {
		t.Fatalf("out=%q errw=%q", out, errw)
	}
}

func TestStatusNotARepo(t *testing.T) {
	code, _, errw := run(t, "status", "--project-dir", t.TempDir())
	if code != cli.ExitState || !strings.Contains(errw, "not a git repository") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
}

func TestStatusJSON(t *testing.T) {
	dir := scratch(t, true)
	run(t, "init", "--project-dir", dir)
	if err := os.MkdirAll(filepath.Join(dir, PlansDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, PlansDir, "2026-01-01-1-x.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run(t, "status", "--project-dir", dir, "--format", "json")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	var s map[string]any
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("json: %v in %q", err, out)
	}
	if s["configured"] != true || s["plans"] != float64(1) || s["releaseBranch"] != "main" {
		t.Fatalf("state: %v", s)
	}

	// Unconfigured json still emits the report; exit code carries state.
	bare := scratch(t, true)
	code, out, _ = run(t, "status", "--project-dir", bare, "--format", "json")
	if code != cli.ExitState || !strings.Contains(out, `"configured":false`) {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestDefaultProjectDirWalksToRoot(t *testing.T) {
	t.Setenv("PK_PROJECT_DIR", "")
	dir := scratch(t, true)
	run(t, "init", "--project-dir", dir)
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	code, out, _ := run(t, "status")
	if code != cli.ExitOK || !strings.Contains(out, dir) {
		t.Fatalf("from subdir: code=%d out=%q", code, out)
	}
}
