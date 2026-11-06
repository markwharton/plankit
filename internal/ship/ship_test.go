package ship

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/changelog"
	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
)

// repo builds a working clone with a bare origin, baseline v0.0.0,
// a pushed develop branch, and committed .pk.json (release.branch
// main). Returns the work dir and the bare origin path.
func repo(t *testing.T) (string, string) {
	t.Helper()
	return repoWith(t, nil)
}

// repoWith is repo with mod applied to the committed .pk.json.
func repoWith(t *testing.T, mod func(*config.PkConfig)) (string, string) {
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
	if mod != nil {
		mod(cfg)
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
	return dir, bare
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

func work(t *testing.T, dir, message string) {
	t.Helper()
	writeF(t, dir, "work.txt", message+"\n")
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", message)
	mustGit(t, dir, "push", "-q", "origin", "develop")
}

func runShip(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var out, errw bytes.Buffer
	argv := append([]string{"pk", "ship", "--project-dir", dir}, args...)
	code := cli.RunIO(argv, []*cli.Command{Cmd}, nil, &out, &errw)
	return code, out.String(), errw.String()
}

func bareRef(t *testing.T, bare, ref string) string {
	t.Helper()
	out, err := git.Exec(bare, "rev-parse", "--verify", "-q", ref)
	if err != nil {
		return ""
	}
	return out
}

func TestShipRunsBothHalves(t *testing.T) {
	dir, bare := repo(t)
	work(t, dir, "feat: the work")
	head, _ := git.Exec(dir, "rev-parse", "HEAD")

	code, out, errw := runShip(t, dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, errw)
	}
	if out != "" {
		t.Fatalf("commit path stdout must stay empty, got %q", out)
	}
	// The release commit sits one past the work commit.
	if got := bareRef(t, bare, "refs/tags/v0.1.0^{commit}"); got == "" || got == head {
		t.Fatalf("origin tag missing or on the wrong commit: %q", got)
	}
	if branch, _ := git.Exec(dir, "branch", "--show-current"); branch != "develop" {
		t.Fatalf("should end back on develop, on %q", branch)
	}
	if !strings.Contains(errw, "Release v0.1.0 complete") {
		t.Fatalf("release banner missing:\n%s", errw)
	}
}

func TestShipResumesFromPendingTrailer(t *testing.T) {
	dir, bare := repo(t)
	work(t, dir, "feat: staged earlier")
	// The changelog half already ran (an interrupted ship, or a manual
	// pk changelog): ship must skip straight to release.
	var out, errw bytes.Buffer
	if code := cli.RunIO([]string{"pk", "changelog", "--project-dir", dir},
		[]*cli.Command{changelog.Cmd}, nil, &out, &errw); code != 0 {
		t.Fatalf("changelog setup: %s", errw.String())
	}

	code, _, shipErr := runShip(t, dir, "--bump", "major")
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, shipErr)
	}
	if !strings.Contains(shipErr, "already pending; skipping changelog") {
		t.Fatalf("resume note missing:\n%s", shipErr)
	}
	if !strings.Contains(shipErr, "--bump and --exclude apply to changelog") {
		t.Fatalf("ignored-flag warning missing:\n%s", shipErr)
	}
	if bareRef(t, bare, "refs/tags/v0.1.0") == "" {
		t.Fatal("release half did not complete")
	}
}

// TestShipOnAReleasedHeadReportsNothingToDo pins what a second ship
// says: a release commit keeps its trailer after it ships, and reading
// the trailer alone called that finished release a pending one.
func TestShipOnAReleasedHeadReportsNothingToDo(t *testing.T) {
	dir, _ := repo(t)
	work(t, dir, "feat: the work")
	if code, _, errw := runShip(t, dir); code != cli.ExitOK {
		t.Fatalf("first ship: exit %d:\n%s", code, errw)
	}

	code, _, errw := runShip(t, dir)
	if code != cli.ExitOK {
		t.Fatalf("second ship: exit %d:\n%s", code, errw)
	}
	if !strings.Contains(errw, "no commits since v0.1.0") {
		t.Fatalf("changelog narration missing:\n%s", errw)
	}
	for _, claim := range []string{"already pending", "nothing to release"} {
		if strings.Contains(errw, claim) {
			t.Fatalf("second ship claims %q:\n%s", claim, errw)
		}
	}
}

// TestShipWithNothingToReleaseStopsAfterChangelog: no commits since the
// baseline tag means the changelog half stages nothing, so the release
// half must not run and report a pending release.
func TestShipWithNothingToReleaseStopsAfterChangelog(t *testing.T) {
	dir, _ := repo(t)

	code, _, errw := runShip(t, dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, errw)
	}
	if !strings.Contains(errw, "no commits since v0.0.0") {
		t.Fatalf("changelog narration missing:\n%s", errw)
	}
	if claim := "no Release-Tag trailer on HEAD"; strings.Contains(errw, claim) {
		t.Fatalf("ship claims %q with nothing to release:\n%s", claim, errw)
	}
}

// assertUndone checks that a failed ship left the repository as it
// found it: develop at the work commit, no CHANGELOG.md, no tag
// anywhere, and origin's main where it was.
func assertUndone(t *testing.T, dir, bare, workHead, mainBefore, errw string) {
	t.Helper()
	if !strings.Contains(errw, "Undid release commit v0.1.0") {
		t.Errorf("the undo did not report:\n%s", errw)
	}
	if h, _ := git.Exec(dir, "rev-parse", "HEAD"); h != workHead {
		t.Fatal("the changelog commit was not undone")
	}
	if branch, _ := git.Exec(dir, "branch", "--show-current"); branch != "develop" {
		t.Fatalf("should be back on develop, on %q", branch)
	}
	if _, err := os.Stat(filepath.Join(dir, "CHANGELOG.md")); !os.IsNotExist(err) {
		t.Fatalf("CHANGELOG.md should be gone, stat err=%v", err)
	}
	if tags, _ := git.Exec(dir, "tag", "--list", "v0.1.0"); tags != "" {
		t.Fatal("local tag survived")
	}
	if bareRef(t, bare, "refs/tags/v0.1.0") != "" {
		t.Fatal("tag reached origin")
	}
	if bareRef(t, bare, "refs/heads/main") != mainBefore {
		t.Fatal("origin main moved")
	}
}

// TestFailedReleaseUndoesTheChangelogCommit: a ship either completes
// or leaves the repository as it found it. Each case fails the release
// half at a different point; the exit code stays the release half's.
func TestFailedReleaseUndoesTheChangelogCommit(t *testing.T) {
	cases := []struct {
		name    string
		mod     func(*config.PkConfig)
		reject  bool // origin rejects the push
		failure string
	}{
		{"pre-release hook", func(c *config.PkConfig) { c.Release.Hooks.PreRelease = "exit 3" }, false, "preRelease hook failed"},
		{"pre-push hook, with the tag made", func(c *config.PkConfig) { c.Release.Hooks.PrePush = "exit 4" }, false, "prePush hook failed"},
		{"rejected push", nil, true, "git push failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, bare := repoWith(t, tc.mod)
			work(t, dir, "feat: the work")
			workHead, _ := git.Exec(dir, "rev-parse", "HEAD")
			mainBefore := bareRef(t, bare, "refs/heads/main")
			if tc.reject {
				hook := filepath.Join(bare, "hooks", "pre-receive")
				if err := os.WriteFile(hook, []byte("#!/bin/sh\necho rejected >&2\nexit 1\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			}

			code, _, errw := runShip(t, dir)
			if code != cli.ExitState || !strings.Contains(errw, tc.failure) {
				t.Fatalf("code=%d errw=%q", code, errw)
			}
			assertUndone(t, dir, bare, workHead, mainBefore, errw)
		})
	}
}

// TestTrunkFlowFailedReleaseUndoesTheChangelogCommit: trunk flow has
// no merge to roll back; the commit is undone on the default branch.
func TestTrunkFlowFailedReleaseUndoesTheChangelogCommit(t *testing.T) {
	dir, bare := repoWith(t, func(c *config.PkConfig) {
		c.Release.Branch = ""
		c.Guard.Branches = nil
		c.Release.Hooks.PreRelease = "exit 3"
	})
	mustGit(t, dir, "switch", "-q", "main")
	writeF(t, dir, "work.txt", "work\n")
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "feat: the work")
	mustGit(t, dir, "push", "-q", "origin", "main")
	workHead, _ := git.Exec(dir, "rev-parse", "HEAD")

	code, _, errw := runShip(t, dir)
	if code != cli.ExitState || !strings.Contains(errw, "preRelease hook failed") || !strings.Contains(errw, "Undid release commit v0.1.0") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if h, _ := git.Exec(dir, "rev-parse", "HEAD"); h != workHead {
		t.Fatal("the changelog commit was not undone")
	}
	if bareRef(t, bare, "refs/tags/v0.1.0") != "" || bareRef(t, bare, "refs/heads/main") != workHead {
		t.Fatal("something reached origin")
	}
}

// TestUndoneShipRunsAgainAsTheSameCommand: after a failed ship nothing
// is left of it, so the same command, options included, starts again
// from the commits and releases the same version.
func TestUndoneShipRunsAgainAsTheSameCommand(t *testing.T) {
	fixed := filepath.Join(t.TempDir(), "fixed")
	dir, bare := repoWith(t, func(c *config.PkConfig) {
		c.Release.Hooks.PreRelease = "test -f " + fixed
	})
	work(t, dir, "feat: the work")
	workHead, _ := git.Exec(dir, "rev-parse", "HEAD")

	if code, _, errw := runShip(t, dir, "--bump", "major"); code != cli.ExitState || !strings.Contains(errw, "Undid release commit v1.0.0") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if h, _ := git.Exec(dir, "rev-parse", "HEAD"); h != workHead {
		t.Fatal("the changelog commit was not undone")
	}

	writeF(t, filepath.Dir(fixed), "fixed", "")
	if code, _, errw := runShip(t, dir, "--bump", "major"); code != cli.ExitOK {
		t.Fatalf("second run: exit %d:\n%s", code, errw)
	}
	if bareRef(t, bare, "refs/tags/v1.0.0") == "" {
		t.Fatal("the second run did not release v1.0.0")
	}
}

// TestFailedReleaseLeavesAnEarlierChangelogCommit: a changelog commit
// that was pending before ship ran was not made by this run, so ship
// does not try to undo it.
func TestFailedReleaseLeavesAnEarlierChangelogCommit(t *testing.T) {
	dir, _ := repoWith(t, func(c *config.PkConfig) {
		c.Release.Hooks.PreRelease = "exit 3"
	})
	work(t, dir, "feat: staged earlier")
	var out, errw bytes.Buffer
	if code := cli.RunIO([]string{"pk", "changelog", "--project-dir", dir},
		[]*cli.Command{changelog.Cmd}, nil, &out, &errw); code != 0 {
		t.Fatalf("changelog setup: %s", errw.String())
	}
	staged, _ := git.Exec(dir, "rev-parse", "HEAD")

	code, _, shipErr := runShip(t, dir)
	if code != cli.ExitState || !strings.Contains(shipErr, "preRelease hook failed") {
		t.Fatalf("code=%d errw=%q", code, shipErr)
	}
	if strings.Contains(shipErr, "Undid release commit") || strings.Contains(shipErr, "cannot undo") {
		t.Fatalf("ship must not try to undo a commit it did not make:\n%s", shipErr)
	}
	if h, _ := git.Exec(dir, "rev-parse", "HEAD"); h != staged {
		t.Fatal("the earlier changelog commit was undone")
	}
}

// TestRefusedUndoKeepsTheChangelogCommit: whether the commit is safe
// to remove is undo's call. When it refuses, here over a tree the
// release hook left dirty, the commit stays, undo says why, and the
// exit code is still the release half's.
func TestRefusedUndoKeepsTheChangelogCommit(t *testing.T) {
	dir, _ := repoWith(t, func(c *config.PkConfig) {
		c.Release.Hooks.PreRelease = "touch junk.txt; exit 3"
	})
	work(t, dir, "feat: the work")

	code, _, errw := runShip(t, dir)
	if code != cli.ExitState {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	for _, want := range []string{"preRelease hook failed", "cannot undo: working tree is not clean"} {
		if !strings.Contains(errw, want) {
			t.Errorf("missing %q in:\n%s", want, errw)
		}
	}
	if subj, _ := git.Exec(dir, "log", "-1", "--format=%s"); subj != "chore: release v0.1.0" {
		t.Fatalf("the changelog commit should stand, HEAD is %q", subj)
	}
}

func TestChangelogRefusalStopsShip(t *testing.T) {
	dir, bare := repo(t)
	work(t, dir, "feat: x")
	writeF(t, dir, "dirty.txt", "dirty\n")

	code, _, errw := runShip(t, dir)
	if code != cli.ExitState || !strings.Contains(errw, "not clean") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if tags, _ := git.Exec(dir, "tag", "--list", "v0.1.0"); tags != "" {
		t.Fatal("release half must not run after a changelog refusal")
	}
	if bareRef(t, bare, "refs/tags/v0.1.0") != "" {
		t.Fatal("nothing may reach origin")
	}
}

func TestDryRunWithoutPendingPreviewsSection(t *testing.T) {
	dir, bare := repo(t)
	work(t, dir, "feat: preview me")

	code, out, errw := runShip(t, dir, "--dry-run")
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, errw)
	}
	if !strings.Contains(out, "## [v0.1.0]") || !strings.Contains(out, "- preview me (") {
		t.Fatalf("section preview missing:\n%s", out)
	}
	if !strings.Contains(errw, "release rehearses against the changelog commit") {
		t.Fatalf("rehearsal note missing:\n%s", errw)
	}
	if subj, _ := git.Exec(dir, "log", "-1", "--format=%s"); subj != "feat: preview me" {
		t.Fatal("dry-run committed")
	}
	if bareRef(t, bare, "refs/tags/v0.1.0") != "" {
		t.Fatal("dry-run reached origin")
	}
}

func TestDryRunWithPendingRehearsesRelease(t *testing.T) {
	dir, _ := repo(t)
	work(t, dir, "feat: staged")
	var out, errw bytes.Buffer
	if code := cli.RunIO([]string{"pk", "changelog", "--project-dir", dir},
		[]*cli.Command{changelog.Cmd}, nil, &out, &errw); code != 0 {
		t.Fatalf("changelog setup: %s", errw.String())
	}

	code, _, shipErr := runShip(t, dir, "--dry-run")
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, shipErr)
	}
	if !strings.Contains(shipErr, "Would create tag v0.1.0") {
		t.Fatalf("release rehearsal missing:\n%s", shipErr)
	}
	if tags, _ := git.Exec(dir, "tag", "--list", "v0.1.0"); tags != "" {
		t.Fatal("dry-run created a tag")
	}
}

// TestFormatIsNotShipFlag: --format is declared only by the commands
// with a structured output, so ship refuses it as an unknown flag.
func TestFormatIsNotShipFlag(t *testing.T) {
	dir, _ := repo(t)
	if code, _, errw := runShip(t, dir, "--format", "json"); code != cli.ExitUsage || !strings.Contains(errw, "format") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
}

// TestStrayArgumentRefusedBeforeAnythingRuns pins the incident that
// motivated frame-wide arity checking: pk ship help must be a usage
// error, not a release.
func TestStrayArgumentRefusedBeforeAnythingRuns(t *testing.T) {
	dir, _ := repo(t)
	work(t, dir, "feat: x")
	head, _ := git.Exec(dir, "rev-parse", "HEAD")

	code, _, errw := runShip(t, dir, "help")
	if code != cli.ExitUsage || !strings.Contains(errw, "unexpected argument") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if h, _ := git.Exec(dir, "rev-parse", "HEAD"); h != head {
		t.Fatal("something committed")
	}
}
