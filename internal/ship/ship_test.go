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
	bare := filepath.Join(t.TempDir(), "origin.git")
	if _, err := git.Exec(t.TempDir(), "init", "-q", "--bare", "-b", "main", bare); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q", "-b", "main")
	mustGit(t, dir, "config", "user.email", "t@t")
	mustGit(t, dir, "config", "user.name", "t")
	mustGit(t, dir, "remote", "add", "origin", bare)
	if err := config.Write(dir, config.Default("main")); err != nil {
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
