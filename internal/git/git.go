// Package git is pk's thin git layer: repository discovery by filesystem
// walk, and an exec wrapper for the handful of commands the kernel needs.
// It stays deliberately small; each layer adds only what its commands use.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindRoot walks upward from start looking for a .git entry (a directory
// in normal checkouts, a file in worktrees and submodules). It needs no
// git binary, so resolution stays cheap on every invocation.
func FindRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// Exec runs git in dir and returns trimmed stdout. Failures carry git's
// stderr, which is usually the message the person needs.
func Exec(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(errb.String())
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", args[0], detail)
	}
	return strings.TrimSpace(out.String()), nil
}

// CurrentBranch returns the checked-out branch name. An empty repository
// (no commits) still reports its unborn branch via symbolic-ref.
func CurrentBranch(dir string) (string, error) {
	out, err := Exec(dir, "symbolic-ref", "--short", "-q", "HEAD")
	if err == nil && out != "" {
		return out, nil
	}
	return Exec(dir, "rev-parse", "--abbrev-ref", "HEAD")
}

// Clean reports whether the working tree has no uncommitted changes.
func Clean(dir string) (bool, error) {
	out, err := Exec(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out == "", nil
}

// HasCommits reports whether HEAD resolves to a commit.
func HasCommits(dir string) bool {
	_, err := Exec(dir, "rev-parse", "--verify", "-q", "HEAD")
	return err == nil
}

// LatestTag returns the highest v-prefixed tag by version sort, or ""
// when none exist.
func LatestTag(dir string) string {
	out, err := Exec(dir, "tag", "-l", "v*", "--sort=-v:refname")
	if err != nil || out == "" {
		return ""
	}
	return strings.SplitN(out, "\n", 2)[0]
}

// CreateTag creates a lightweight tag at HEAD.
func CreateTag(dir, name string) error {
	_, err := Exec(dir, "tag", name)
	return err
}
