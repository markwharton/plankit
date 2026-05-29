package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepoRoot_returnsRoot(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".git"), 0755)
	sub := filepath.Join(root, "packages", "foo")
	os.MkdirAll(sub, 0755)

	got, ok := RepoRoot(os.Stat, sub)
	if !ok {
		t.Fatal("expected ok=true for subdirectory of a git repo")
	}
	if got != root {
		t.Errorf("RepoRoot = %q, want %q", got, root)
	}
}

func TestRepoRoot_atRoot(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".git"), 0755)

	got, ok := RepoRoot(os.Stat, root)
	if !ok {
		t.Fatal("expected ok=true for directory with .git")
	}
	if got != root {
		t.Errorf("RepoRoot = %q, want %q", got, root)
	}
}

func TestRepoRoot_notARepo(t *testing.T) {
	dir := t.TempDir()

	_, ok := RepoRoot(os.Stat, dir)
	if ok {
		t.Error("expected ok=false for directory without .git")
	}
}

func TestIsRepo_directGitDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)

	if !IsRepo(os.Stat, dir) {
		t.Error("expected true for directory with .git")
	}
}

func TestIsRepo_gitFile(t *testing.T) {
	// Git submodules and worktrees use a .git file, not a directory.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: /path/to/actual/.git\n"), 0644)

	if !IsRepo(os.Stat, dir) {
		t.Error("expected true for directory with .git as a file")
	}
}

func TestIsRepo_parentHasGit(t *testing.T) {
	// Monorepo subdirectory case.
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".git"), 0755)
	sub := filepath.Join(root, "packages", "foo")
	os.MkdirAll(sub, 0755)

	if !IsRepo(os.Stat, sub) {
		t.Error("expected true for subdirectory of a git repo")
	}
}

func TestIsRepo_deeplyNested(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".git"), 0755)
	sub := filepath.Join(root, "a", "b", "c", "d", "e")
	os.MkdirAll(sub, 0755)

	if !IsRepo(os.Stat, sub) {
		t.Error("expected true for deeply nested subdirectory")
	}
}

func TestIsRepo_notAGitRepo(t *testing.T) {
	dir := t.TempDir()
	// No .git anywhere.

	if IsRepo(os.Stat, dir) {
		t.Error("expected false for directory without .git")
	}
}

func TestIsRepo_stopsAtFilesystemRoot(t *testing.T) {
	// A bogus path that doesn't exist — should return false without looping forever.
	if IsRepo(os.Stat, "/nonexistent/path/that/does/not/exist") {
		t.Error("expected false for nonexistent path")
	}
}
