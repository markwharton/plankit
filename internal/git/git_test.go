package git

import (
	"os"
	"path/filepath"
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
	if got := LatestTag(dir); got != "" {
		t.Fatalf("LatestTag=%q want empty", got)
	}
	if err := CreateTag(dir, "v0.0.0"); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "tag", "v0.10.0")
	mustGit(t, dir, "tag", "v0.2.0")
	if got := LatestTag(dir); got != "v0.10.0" {
		t.Fatalf("LatestTag=%q want v0.10.0 (version sort)", got)
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
