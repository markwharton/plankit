// Package git is pk's thin git layer: repository discovery by filesystem
// walk, and an exec wrapper for the handful of commands the kernel needs.
// It stays deliberately small; each layer adds only what its commands use.
package git

import (
	"errors"
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

// ErrDirtyTree reports uncommitted changes where a clean tree is required.
var ErrDirtyTree = errors.New("working tree is not clean; commit or stash changes first")

// CheckCleanTree returns nil for a clean tree, ErrDirtyTree for a dirty
// one, or a wrapped error when git itself fails.
func CheckCleanTree(dir string) error {
	clean, err := Clean(dir)
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}
	if !clean {
		return ErrDirtyTree
	}
	return nil
}

// DefaultBranch discovers origin's default branch via the HEAD symref.
// ok is false when origin advertises none (empty remote, hidden symref);
// err is non-nil only when the command itself fails. Callers treat both
// the same: no default can be established, skip the check.
func DefaultBranch(dir string) (string, bool, error) {
	out, err := Exec(dir, "ls-remote", "--symref", "origin", "HEAD")
	if err != nil {
		return "", false, err
	}
	for _, line := range strings.Split(out, "\n") {
		rest, found := strings.CutPrefix(line, "ref: refs/heads/")
		if !found {
			continue
		}
		name, found := strings.CutSuffix(rest, "\tHEAD")
		if found && name != "" {
			return name, true, nil
		}
	}
	return "", false, nil
}

// HasOtherLocalBranch reports whether any local branch besides the given
// one exists; hints use it to suggest creating a working branch only
// when there is none.
func HasOtherLocalBranch(dir, branch string) bool {
	out, err := Exec(dir, "branch", "--format=%(refname:short)")
	if err != nil {
		return false
	}
	for _, b := range strings.Split(out, "\n") {
		if b = strings.TrimSpace(b); b != "" && b != branch {
			return true
		}
	}
	return false
}

// ParseRepoURL normalizes a remote URL to https form for compare links.
func ParseRepoURL(remoteURL string) string {
	u := strings.TrimSpace(remoteURL)
	if strings.HasPrefix(u, "git@") {
		u = strings.TrimPrefix(u, "git@")
		u = strings.Replace(u, ":", "/", 1)
		u = "https://" + u
	}
	return strings.TrimSuffix(u, ".git")
}
