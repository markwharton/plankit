package changelog

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
)

func fixedNow(t *testing.T) {
	t.Helper()
	old := now
	now = func() time.Time { return time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = old })
}

// repo builds a working clone with a bare origin: baseline v0.0.0 on
// main (pushed), a develop branch (pushed), .pk.json committed with
// guard on main and release.branch main.
func repo(t *testing.T, mutate func(*config.PkConfig)) string {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "origin.git")
	if _, err := git.Exec(t.TempDir(), "init", "-q", "--bare", "-b", "main", bare); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q", "-b", "main")
	mustGit(t, dir, "config", "user.email", "t@t")
	mustGit(t, dir, "config", "user.name", "t")
	mustGit(t, dir, "remote", "add", "origin", bare)

	cfg := config.Default("main")
	if mutate != nil {
		mutate(cfg)
	}
	if err := config.Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	writeF(t, dir, "a.txt", "a\n")
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "chore: scaffold")
	mustGit(t, dir, "tag", "v0.0.0")
	mustGit(t, dir, "push", "-q", "-u", "origin", "main", "--tags")
	mustGit(t, dir, "switch", "-q", "-c", "develop")
	mustGit(t, dir, "push", "-q", "-u", "origin", "develop")
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, err := git.Exec(dir, args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

func writeF(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// commit makes a conventional commit touching a throwaway file.
func commit(t *testing.T, dir, message string) {
	t.Helper()
	writeF(t, dir, "work.txt", message+"\n")
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", message)
}

func runCL(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var out, errw bytes.Buffer
	argv := append([]string{"pk", "changelog", "--project-dir", dir}, args...)
	code := cli.RunIO(argv, []*cli.Command{Cmd}, nil, &out, &errw)
	return code, out.String(), errw.String()
}

func headSubject(t *testing.T, dir string) string {
	t.Helper()
	s, err := git.Exec(dir, "log", "-1", "--format=%s")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestGeneratesSectionCommitsWithTrailer(t *testing.T) {
	fixedNow(t)
	dir := repo(t, nil)
	commit(t, dir, "feat: add the widget")
	commit(t, dir, "fix: stop the leak")
	commit(t, dir, "plan: hidden entry [skip ci]")
	commit(t, dir, "not a conventional commit")

	code, out, errw := runCL(t, dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d\nstderr: %s", code, errw)
	}
	if out != "" {
		t.Fatalf("stdout should be empty on commit path, got %q", out)
	}

	cl, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(cl)
	for _, want := range []string{
		"# Changelog",
		"## [v0.1.0] - 2026-09-05",
		"### Added",
		"- add the widget (",
		"### Fixed",
		"- stop the leak (",
		"/compare/v0.0.0...v0.1.0",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("CHANGELOG missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "hidden entry") {
		t.Error("hidden plan type leaked into the changelog")
	}

	if got := headSubject(t, dir); got != "chore: release v0.1.0" {
		t.Fatalf("subject = %q", got)
	}
	if _, tag, err := ReadReleaseTagTrailer(dir); err != nil || tag != "v0.1.0" {
		t.Fatalf("trailer = %q err=%v", tag, err)
	}
}

func TestBumpInference(t *testing.T) {
	cases := []struct {
		subject string
		body    string
		want    string
	}{
		{"fix: patch it", "", "v0.0.1"},
		{"feat: minor it", "", "v0.1.0"},
		{"feat!: break it", "", "v1.0.0"},
		{"refactor: quiet change", "BREAKING CHANGE: the api moved", "v1.0.0"},
	}
	for _, tc := range cases {
		fixedNow(t)
		dir := repo(t, nil)
		writeF(t, dir, "work.txt", tc.subject)
		mustGit(t, dir, "add", ".")
		args := []string{"commit", "-q", "-m", tc.subject}
		if tc.body != "" {
			args = append(args, "-m", tc.body)
		}
		mustGit(t, dir, args...)
		code, _, errw := runCL(t, dir)
		if code != cli.ExitOK {
			t.Fatalf("%s: exit %d: %s", tc.subject, code, errw)
		}
		if _, tag, _ := ReadReleaseTagTrailer(dir); tag != tc.want {
			t.Errorf("%s: tag = %q, want %q", tc.subject, tag, tc.want)
		}
	}
}

func TestBumpOverrideAndValidation(t *testing.T) {
	dir := repo(t, nil)
	commit(t, dir, "feat: would be minor")
	code, _, _ := runCL(t, dir, "--bump", "major")
	if code != cli.ExitOK {
		t.Fatal("override failed")
	}
	if _, tag, _ := ReadReleaseTagTrailer(dir); tag != "v1.0.0" {
		t.Fatalf("tag = %q", tag)
	}

	dir2 := repo(t, nil)
	commit(t, dir2, "fix: x")
	code, _, errw := runCL(t, dir2, "--bump", "huge")
	if code != cli.ExitUsage || !strings.Contains(errw, "invalid --bump") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
}

func TestDryRunPrintsSectionToStdout(t *testing.T) {
	fixedNow(t)
	dir := repo(t, nil)
	commit(t, dir, "feat: preview me")
	writeF(t, dir, "dirty.txt", "dirty is fine in dry-run\n")

	code, out, _ := runCL(t, dir, "--dry-run")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(out, "## [v0.1.0] - 2026-09-05") || !strings.Contains(out, "- preview me (") {
		t.Fatalf("stdout = %q", out)
	}
	if headSubject(t, dir) != "feat: preview me" {
		t.Fatal("dry-run committed")
	}
}

func TestRefusals(t *testing.T) {
	// Dirty tree.
	dir := repo(t, nil)
	commit(t, dir, "feat: x")
	writeF(t, dir, "dirty.txt", "x\n")
	if code, _, errw := runCL(t, dir); code != cli.ExitState || !strings.Contains(errw, "not clean") {
		t.Fatalf("dirty: code=%d errw=%q", code, errw)
	}

	// Protected branch.
	dir = repo(t, nil)
	mustGit(t, dir, "switch", "-q", "main")
	if code, _, errw := runCL(t, dir); code != cli.ExitState || !strings.Contains(errw, "protected branch") {
		t.Fatalf("protected: code=%d errw=%q", code, errw)
	}

	// Branch not on origin.
	dir = repo(t, nil)
	mustGit(t, dir, "switch", "-q", "-c", "local-only")
	commit(t, dir, "feat: x")
	if code, _, errw := runCL(t, dir); code != cli.ExitState || !strings.Contains(errw, "does not exist on origin") {
		t.Fatalf("no-origin: code=%d errw=%q", code, errw)
	}

	// Unconfigured.
	dir = repo(t, nil)
	os.Remove(config.Path(dir))
	mustGit(t, dir, "add", "-u")
	mustGit(t, dir, "commit", "-q", "-m", "chore: drop config")
	if code, _, errw := runCL(t, dir); code != cli.ExitState || !strings.Contains(errw, "pk init") {
		t.Fatalf("unconfigured: code=%d errw=%q", code, errw)
	}

	// Pending trailer refuses a second run while HEAD is the release
	// commit (the guard is HEAD-only by design: committing past it means
	// you have moved on, and the stale trailer is release's problem).
	dir = repo(t, nil)
	commit(t, dir, "feat: x")
	runCL(t, dir)
	if code, _, errw := runCL(t, dir); code != cli.ExitState || !strings.Contains(errw, "already pending") {
		t.Fatalf("pending: code=%d errw=%q", code, errw)
	}
}

func TestExclude(t *testing.T) {
	dir := repo(t, nil)
	commit(t, dir, "feat: keep me")
	commit(t, dir, "feat: drop me")
	drop, _ := git.Exec(dir, "log", "-1", "--format=%h")

	code, _, errw := runCL(t, dir, "--exclude", drop+",deadbeef")
	if code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	if !strings.Contains(errw, "deadbeef did not match") {
		t.Fatalf("unmatched exclude should warn: %s", errw)
	}
	cl, _ := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if strings.Contains(string(cl), "drop me") || !strings.Contains(string(cl), "keep me") {
		t.Fatalf("exclude not applied:\n%s", cl)
	}
}

func TestExcludeAffectsBump(t *testing.T) {
	dir := repo(t, nil)
	commit(t, dir, "fix: small")
	commit(t, dir, "feat: big")
	feat, _ := git.Exec(dir, "log", "-1", "--format=%h")
	runCL(t, dir, "--exclude", feat)
	if _, tag, _ := ReadReleaseTagTrailer(dir); tag != "v0.0.1" {
		t.Fatalf("bump should fall to patch after excluding the feat, got %q", tag)
	}
}

func TestVersionFileSplicePreservesFormatting(t *testing.T) {
	dir := repo(t, func(c *config.PkConfig) {
		c.Changelog.VersionFiles = []config.VersionFile{{Path: "pkg.json", Type: "json"}}
	})
	original := "{\n    \"name\":   \"demo\",\n    \"version\": \"0.0.0\",\n    \"keep\":  [1, 2]\n}\n"
	writeF(t, dir, "pkg.json", original)
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "chore: add pkg.json")
	commit(t, dir, "feat: bump me")

	if code, _, errw := runCL(t, dir); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "pkg.json"))
	want := strings.Replace(original, `"0.0.0"`, `"0.1.0"`, 1)
	if string(got) != want {
		t.Fatalf("splice broke formatting:\n%q\nwant\n%q", got, want)
	}
}

func TestSubjectEscaping(t *testing.T) {
	dir := repo(t, nil)
	commit(t, dir, "feat: use <T> & `a<b` safely")
	runCL(t, dir)
	cl, _ := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if !strings.Contains(string(cl), "use &lt;T> &amp; `a<b` safely") {
		t.Fatalf("escaping wrong:\n%s", cl)
	}
}

func TestSecondReleaseInsertsAboveFirst(t *testing.T) {
	dir := repo(t, nil)
	commit(t, dir, "feat: one")
	runCL(t, dir)
	// Simulate the first release completing so the pending guard clears
	// and the next range starts at the release commit.
	mustGit(t, dir, "tag", "v0.1.0")
	commit(t, dir, "fix: two")
	if code, _, errw := runCL(t, dir); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	cl, _ := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	s := string(cl)
	if strings.Index(s, "## [v0.1.1]") > strings.Index(s, "## [v0.1.0]") {
		t.Fatalf("new section should sit above the old:\n%s", s)
	}
	if !strings.Contains(s, "/compare/v0.1.0...v0.1.1") {
		t.Fatalf("compare link should span the previous tag:\n%s", s)
	}
}

func TestHooksReceiveVersion(t *testing.T) {
	dir := repo(t, func(c *config.PkConfig) {
		c.Changelog.Hooks.PostVersion = "printf %s \"$VERSION\" > post.txt"
		c.Changelog.Hooks.PreCommit = "printf %s \"$VERSION\" > pre.txt && git add pre.txt post.txt"
	})
	commit(t, dir, "feat: hooked")
	if code, _, errw := runCL(t, dir); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	for _, f := range []string{"post.txt", "pre.txt"} {
		got, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil || string(got) != "0.1.0" {
			t.Fatalf("%s = %q err=%v", f, got, err)
		}
	}
	if clean, _ := git.Clean(dir); !clean {
		t.Fatal("hook outputs should be committed")
	}
}

func TestFailingHookAborts(t *testing.T) {
	dir := repo(t, func(c *config.PkConfig) {
		c.Changelog.Hooks.PostVersion = "exit 7"
	})
	commit(t, dir, "feat: x")
	code, _, errw := runCL(t, dir)
	if code != cli.ExitState || !strings.Contains(errw, "postVersion hook failed") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if headSubject(t, dir) != "feat: x" {
		t.Fatal("failed hook must not commit")
	}
	if status, _ := git.Exec(dir, "status", "--porcelain"); status != "" {
		t.Fatalf("a failed hook must leave the tree clean, got:\n%s", status)
	}
	if !strings.Contains(errw, "the tree is clean") {
		t.Fatalf("the error must say the tree is clean: %q", errw)
	}

	// The same after CHANGELOG.md is written: a failing preCommit hook
	// restores it too.
	dir = repo(t, func(c *config.PkConfig) {
		c.Changelog.Hooks.PreCommit = "exit 3"
	})
	commit(t, dir, "feat: y")
	if code, _, errw := runCL(t, dir); code != cli.ExitState || !strings.Contains(errw, "preCommit hook failed") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if status, _ := git.Exec(dir, "status", "--porcelain"); status != "" {
		t.Fatalf("preCommit failure must leave the tree clean, got:\n%s", status)
	}
}

func TestUndo(t *testing.T) {
	dir := repo(t, nil)
	commit(t, dir, "feat: x")
	runCL(t, dir)
	if headSubject(t, dir) != "chore: release v0.1.0" {
		t.Fatal("setup failed")
	}

	code, _, errw := runCL(t, dir, "--undo")
	if code != cli.ExitOK {
		t.Fatalf("undo exit %d: %s", code, errw)
	}
	if headSubject(t, dir) != "feat: x" {
		t.Fatal("undo did not reset")
	}
	if _, err := os.Stat(filepath.Join(dir, "CHANGELOG.md")); !os.IsNotExist(err) {
		t.Fatal("undo should restore the tree (CHANGELOG.md gone)")
	}

	// Nothing to undo now.
	if code, _, errw := runCL(t, dir, "--undo"); code != cli.ExitState || !strings.Contains(errw, "no Release-Tag trailer") {
		t.Fatalf("second undo: code=%d errw=%q", code, errw)
	}

	// A pushed release commit refuses to undo.
	runCL(t, dir)
	mustGit(t, dir, "push", "-q", "origin", "develop")
	if code, _, errw := runCL(t, dir, "--undo"); code != cli.ExitState || !strings.Contains(errw, "already on the remote") {
		t.Fatalf("pushed undo: code=%d errw=%q", code, errw)
	}
}

func TestTrunkFlowRefusesNonDefaultBranch(t *testing.T) {
	dir := repo(t, func(c *config.PkConfig) {
		c.Release.Branch = ""
		c.Guard.Branches = nil
	})
	commit(t, dir, "feat: x")
	code, _, errw := runCL(t, dir) // on develop; origin default is main
	if code != cli.ExitState || !strings.Contains(errw, "trunk flow releases from the default branch") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}

	mustGit(t, dir, "switch", "-q", "main")
	mustGit(t, dir, "merge", "-q", "--ff-only", "develop")
	if code, _, errw := runCL(t, dir); code != cli.ExitOK {
		t.Fatalf("trunk flow on default branch: exit %d: %s", code, errw)
	}
}
