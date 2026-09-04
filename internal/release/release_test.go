package release

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

// repo builds a working clone with a bare origin, baseline v0.0.0, a
// develop branch, committed .pk.json (release.branch main), and one
// pending release: a feat commit plus the pk changelog commit carrying
// Release-Tag: v0.1.0. Returns the work dir and the bare origin path.
func repo(t *testing.T, mutate func(*config.PkConfig)) (string, string) {
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
	return dir, bare
}

// pending runs pk changelog to put a Release-Tag commit on HEAD.
func pending(t *testing.T, dir string) {
	t.Helper()
	writeF(t, dir, "work.txt", "w\n")
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "feat: the work")
	mustGit(t, dir, "push", "-q", "origin", "develop")
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "changelog", "--project-dir", dir},
		[]*cli.Command{changelog.Cmd}, nil, &out, &errw)
	if code != 0 {
		t.Fatalf("changelog setup failed: %s", errw.String())
	}
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

func runRel(t *testing.T, dir string, args ...string) (int, string) {
	t.Helper()
	var out, errw bytes.Buffer
	argv := append([]string{"pk", "release", "--project-dir", dir}, args...)
	code := cli.RunIO(argv, []*cli.Command{Cmd}, nil, &out, &errw)
	if out.Len() != 0 {
		t.Fatalf("release stdout should stay empty, got %q", out.String())
	}
	return code, errw.String()
}

func bareRef(t *testing.T, bare, ref string) string {
	t.Helper()
	out, err := git.Exec(bare, "rev-parse", "--verify", "-q", ref)
	if err != nil {
		return ""
	}
	return out
}

func TestMergeFlowReleases(t *testing.T) {
	dir, bare := repo(t, nil)
	pending(t, dir)
	head, _ := git.Exec(dir, "rev-parse", "HEAD")

	code, errw := runRel(t, dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, errw)
	}
	if got := bareRef(t, bare, "refs/tags/v0.1.0^{commit}"); got != head {
		t.Fatalf("origin tag at %q, want %q", got, head)
	}
	if got := bareRef(t, bare, "refs/heads/main"); got != head {
		t.Fatalf("origin main at %q, want %q", got, head)
	}
	if got := bareRef(t, bare, "refs/heads/develop"); got != head {
		t.Fatalf("origin develop at %q, want %q", got, head)
	}
	if branch, _ := git.Exec(dir, "branch", "--show-current"); branch != "develop" {
		t.Fatalf("should end back on develop, on %q", branch)
	}
	if !strings.Contains(errw, "=== Release v0.1.0 complete ===") {
		t.Fatalf("missing banner:\n%s", errw)
	}
}

func TestDryRunTouchesNothing(t *testing.T) {
	dir, bare := repo(t, nil)
	pending(t, dir)

	code, errw := runRel(t, dir, "--dry-run")
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, errw)
	}
	if !strings.Contains(errw, "Would create tag v0.1.0") || !strings.Contains(errw, "Would push main and v0.1.0") {
		t.Fatalf("rehearsal narration:\n%s", errw)
	}
	if tags, _ := git.Exec(dir, "tag", "--list", "v0.1.0"); tags != "" {
		t.Fatal("dry-run created a local tag")
	}
	if bareRef(t, bare, "refs/tags/v0.1.0") != "" {
		t.Fatal("dry-run pushed a tag")
	}
	if branch, _ := git.Exec(dir, "branch", "--show-current"); branch != "develop" {
		t.Fatalf("dry-run switched branches: %q", branch)
	}
}

func TestRefusals(t *testing.T) {
	// No trailer.
	dir, _ := repo(t, nil)
	if code, errw := runRel(t, dir); code != cli.ExitState || !strings.Contains(errw, "run pk changelog first") {
		t.Fatalf("no trailer: code=%d errw=%q", code, errw)
	}

	// Tag already exists.
	dir, _ = repo(t, nil)
	pending(t, dir)
	mustGit(t, dir, "tag", "v0.1.0")
	if code, errw := runRel(t, dir); code != cli.ExitState || !strings.Contains(errw, "already exists locally") {
		t.Fatalf("tag exists: code=%d errw=%q", code, errw)
	}

	// On the release branch.
	dir, _ = repo(t, nil)
	mustGit(t, dir, "switch", "-q", "main")
	if code, errw := runRel(t, dir); code != cli.ExitState || !strings.Contains(errw, "you're on the release branch") {
		t.Fatalf("on release branch: code=%d errw=%q", code, errw)
	}

	// Behind origin: someone pushed to develop after our fetch of it.
	dir, bare := repo(t, nil)
	pending(t, dir)
	other := t.TempDir()
	mustGit(t, other, "clone", "-q", bare, "clone2")
	clone2 := filepath.Join(other, "clone2")
	mustGit(t, clone2, "config", "user.email", "o@o")
	mustGit(t, clone2, "config", "user.name", "o")
	mustGit(t, clone2, "switch", "-q", "develop")
	writeF(t, clone2, "b.txt", "b\n")
	mustGit(t, clone2, "add", ".")
	mustGit(t, clone2, "commit", "-q", "-m", "fix: from elsewhere")
	mustGit(t, clone2, "push", "-q", "origin", "develop")
	if code, errw := runRel(t, dir); code != cli.ExitState || !strings.Contains(errw, "behind origin/develop") {
		t.Fatalf("behind: code=%d errw=%q", code, errw)
	}
}

func TestDivergedReleaseBranchRefused(t *testing.T) {
	dir, bare := repo(t, nil)
	pending(t, dir)

	// A commit lands directly on origin/main: the release push would be
	// rejected, so pre-flight must catch it before tagging.
	other := t.TempDir()
	mustGit(t, other, "clone", "-q", bare, "clone2")
	clone2 := filepath.Join(other, "clone2")
	mustGit(t, clone2, "config", "user.email", "o@o")
	mustGit(t, clone2, "config", "user.name", "o")
	writeF(t, clone2, "hotfix.txt", "h\n")
	mustGit(t, clone2, "add", ".")
	mustGit(t, clone2, "commit", "-q", "-m", "fix: hotfix on main")
	mustGit(t, clone2, "push", "-q", "origin", "main")

	code, errw := runRel(t, dir)
	if code != cli.ExitState || !strings.Contains(errw, "has diverged from develop") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if tags, _ := git.Exec(dir, "tag", "--list", "v0.1.0"); tags != "" {
		t.Fatal("refusal must not leave a tag behind")
	}
}

func TestRejectedPushRollsEverythingBack(t *testing.T) {
	dir, bare := repo(t, nil)
	pending(t, dir)
	mainBefore := bareRef(t, bare, "refs/heads/main")

	// Origin rejects all pushes: the one failure pre-flight cannot see.
	hook := filepath.Join(bare, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho rejected >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	code, errw := runRel(t, dir)
	if code != cli.ExitState || !strings.Contains(errw, "git push failed") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	for _, want := range []string{"Cleaned up local tag v0.1.0", "Rolled back merge on main"} {
		if !strings.Contains(errw, want) {
			t.Errorf("missing %q in:\n%s", want, errw)
		}
	}
	if tags, _ := git.Exec(dir, "tag", "--list", "v0.1.0"); tags != "" {
		t.Fatal("local tag survived the rollback")
	}
	if branch, _ := git.Exec(dir, "branch", "--show-current"); branch != "develop" {
		t.Fatalf("should be back on develop, on %q", branch)
	}
	localMain, _ := git.Exec(dir, "rev-parse", "refs/heads/main")
	if localMain != mainBefore {
		t.Fatal("merge on main was not rolled back")
	}
}

func TestHooks(t *testing.T) {
	dir, _ := repo(t, func(c *config.PkConfig) {
		c.Release.Hooks.PreRelease = "printf %s:%s \"$VERSION\" \"$TAG\" > prerel.txt"
		c.Release.Hooks.PrePush = "git tag --points-at HEAD > prepush.txt"
	})
	pending(t, dir)
	if code, errw := runRel(t, dir); code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, errw)
	}
	// preRelease ran before the tag existed, with both env forms.
	got, _ := os.ReadFile(filepath.Join(dir, "prerel.txt"))
	if string(got) != "0.1.0:v0.1.0" {
		t.Fatalf("preRelease env = %q", got)
	}
	// prePush ran with the tag ref in existence.
	got, _ = os.ReadFile(filepath.Join(dir, "prepush.txt"))
	if !strings.Contains(string(got), "v0.1.0") {
		t.Fatalf("prePush should see the tag, got %q", got)
	}
}

func TestFailingPrePushCleansUp(t *testing.T) {
	dir, bare := repo(t, func(c *config.PkConfig) {
		c.Release.Hooks.PrePush = "exit 3"
	})
	pending(t, dir)
	code, errw := runRel(t, dir)
	if code != cli.ExitState || !strings.Contains(errw, "pre-push hook failed") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if tags, _ := git.Exec(dir, "tag", "--list", "v0.1.0"); tags != "" {
		t.Fatal("tag survived the failed prePush")
	}
	if bareRef(t, bare, "refs/tags/v0.1.0") != "" {
		t.Fatal("nothing should have been pushed")
	}
}

func TestTrunkFlowReleasesFromDefaultBranch(t *testing.T) {
	dir, bare := repo(t, func(c *config.PkConfig) {
		c.Release.Branch = ""
		c.Guard.Branches = nil
	})
	mustGit(t, dir, "switch", "-q", "main")
	mustGit(t, dir, "merge", "-q", "--ff-only", "develop")
	pendingOnMain := func() {
		writeF(t, dir, "work.txt", "w\n")
		mustGit(t, dir, "add", ".")
		mustGit(t, dir, "commit", "-q", "-m", "feat: trunk work")
		mustGit(t, dir, "push", "-q", "origin", "main")
		var out, errw bytes.Buffer
		if code := cli.RunIO([]string{"pk", "changelog", "--project-dir", dir},
			[]*cli.Command{changelog.Cmd}, nil, &out, &errw); code != 0 {
			t.Fatalf("changelog: %s", errw.String())
		}
	}
	pendingOnMain()
	head, _ := git.Exec(dir, "rev-parse", "HEAD")

	code, errw := runRel(t, dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d:\n%s", code, errw)
	}
	if !strings.Contains(errw, "Trunk flow") || !strings.Contains(errw, "On main (default branch on origin)") {
		t.Fatalf("trunk narration:\n%s", errw)
	}
	if got := bareRef(t, bare, "refs/tags/v0.1.0^{commit}"); got != head {
		t.Fatalf("origin tag at %q, want %q", got, head)
	}
}
