package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// scratch creates a git repo with one commit and returns its path.
func scratch(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q", "-b", "main")
	mustGit(t, dir, "config", "user.email", "t@t")
	mustGit(t, dir, "config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "first")
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, err := Exec(dir, args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

func TestFindRootWalksUp(t *testing.T) {
	dir := scratch(t)
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	root, ok := FindRoot(sub)
	if !ok || root != dir {
		t.Fatalf("root=%q ok=%v want %q", root, ok, dir)
	}
	if _, ok := FindRoot(t.TempDir()); ok {
		t.Fatal("found a root in a plain temp dir")
	}
}

func TestBranchCleanCommitsTags(t *testing.T) {
	dir := scratch(t)
	if b, err := CurrentBranch(dir); err != nil || b != "main" {
		t.Fatalf("branch=%q err=%v", b, err)
	}
	if clean, _ := Clean(dir); !clean {
		t.Fatal("fresh commit should be clean")
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if clean, _ := Clean(dir); clean {
		t.Fatal("modified tree should be dirty")
	}
	if !HasCommits(dir) {
		t.Fatal("HasCommits should be true")
	}
	if tags, err := Tags(dir); err != nil || len(tags) != 0 {
		t.Fatalf("Tags=%v err=%v want none", tags, err)
	}
	if err := CreateTag(dir, "v0.0.0"); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "tag", "v0.10.0")
	mustGit(t, dir, "tag", "notes")
	if tags, err := Tags(dir); err != nil || strings.Join(tags, " ") != "notes v0.0.0 v0.10.0" {
		t.Fatalf("Tags=%v err=%v want every tag name", tags, err)
	}
}

func TestEmptyRepoReportsUnbornBranch(t *testing.T) {
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q", "-b", "main")
	if HasCommits(dir) {
		t.Fatal("empty repo should have no commits")
	}
	if b, err := CurrentBranch(dir); err != nil || b != "main" {
		t.Fatalf("branch=%q err=%v", b, err)
	}
}

func TestRemoteBranchExists(t *testing.T) {
	dir := scratch(t)
	if _, err := RemoteBranchExists(dir, "main"); err == nil {
		t.Fatal("no origin should be an error, not an absent branch")
	}
	bare := filepath.Join(t.TempDir(), "origin.git")
	mustGit(t, dir, "init", "-q", "--bare", bare)
	mustGit(t, dir, "remote", "add", "origin", bare)
	mustGit(t, dir, "push", "-q", "origin", "main:feature/main")
	if ok, err := RemoteBranchExists(dir, "main"); err != nil || ok {
		t.Fatalf("feature/main must not count as main: ok=%v err=%v", ok, err)
	}
	mustGit(t, dir, "push", "-q", "origin", "main")
	if ok, err := RemoteBranchExists(dir, "main"); err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestIsAncestor(t *testing.T) {
	dir := scratch(t)
	first, _ := Exec(dir, "rev-parse", "HEAD")
	mustGit(t, dir, "switch", "-q", "-c", "side")
	mustGit(t, dir, "commit", "-q", "--allow-empty", "-m", "side")
	side, _ := Exec(dir, "rev-parse", "HEAD")
	mustGit(t, dir, "switch", "-q", "main")
	mustGit(t, dir, "commit", "-q", "--allow-empty", "-m", "second")
	if ok, err := IsAncestor(dir, first, "HEAD"); !ok || err != nil {
		t.Fatalf("root must be an ancestor: %v %v", ok, err)
	}
	if ok, err := IsAncestor(dir, side, "HEAD"); ok || err != nil {
		t.Fatalf("a side commit is not: %v %v", ok, err)
	}
	if _, err := IsAncestor(dir, "no-such-rev", "HEAD"); err == nil {
		t.Fatal("an unknown revision must be an error, not false")
	}
}

// The web address of a remote never carries what the remote's URL
// used to sign in.
func TestParseRepoURL(t *testing.T) {
	cases := map[string]string{
		"https://github.com/o/r.git":                           "https://github.com/o/r",
		"https://github.com/o/r":                               "https://github.com/o/r",
		"  https://github.com/o/r.git\n":                       "https://github.com/o/r",
		"https://x-access-token:ghs_SECRET@github.com/o/r.git": "https://github.com/o/r",
		"https://user@github.com/o/r.git":                      "https://github.com/o/r",
		"http://user:pw@git.example:8080/o/r.git":              "http://git.example:8080/o/r",
		"ssh://git@github.com/o/r.git":                         "https://github.com/o/r",
		"ssh://git@github.com:22/o/r.git":                      "https://github.com/o/r",
		"git@github.com:o/r.git":                               "https://github.com/o/r",
		"deploy@git.example:group/r.git":                       "https://git.example/group/r",
		"/srv/git/r.git":                                       "/srv/git/r",
		"https://user:pw@[bad/o/r.git":                         "",
	}
	for in, want := range cases {
		if got := ParseRepoURL(in); got != want {
			t.Errorf("ParseRepoURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// A detached HEAD is on no branch, and the answer says so with an
// empty name, never a word that could be taken for a branch.
func TestCurrentBranchOnADetachedHead(t *testing.T) {
	dir := scratch(t)
	mustGit(t, dir, "checkout", "-q", "--detach")
	if b, err := CurrentBranch(dir); err != nil || b != "" {
		t.Fatalf("branch=%q err=%v, want no branch", b, err)
	}
}
