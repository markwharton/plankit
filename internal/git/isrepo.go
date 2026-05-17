package git

import (
	"os"
	"path/filepath"
)

// IsInsideWorkTree verifies the directory is inside a git work tree by
// invoking git rev-parse. Returns nil on success, the exec error otherwise.
func IsInsideWorkTree(gitExec func(string, ...string) (string, error), dir string) error {
	_, err := gitExec(dir, "rev-parse", "--is-inside-work-tree")
	return err
}

// TopLevel returns the absolute path of the repository root by invoking
// git rev-parse --show-toplevel. Returns an error if not inside a work tree.
func TopLevel(gitExec func(string, ...string) (string, error), dir string) (string, error) {
	return gitExec(dir, "rev-parse", "--show-toplevel")
}

// RepoRoot reports the root of the git repository containing dir. It walks up
// parent directories looking for a .git entry (directory or file — git
// submodules and worktrees use a file). Returns the absolute path and true,
// or ("", false) if no repository is found.
//
// Accepts an injected stat function for testability; pass os.Stat for
// real filesystem access.
func RepoRoot(stat func(string) (os.FileInfo, error), dir string) (string, bool) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	for {
		if _, err := stat(filepath.Join(abs, ".git")); err == nil {
			return abs, true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", false
		}
		abs = parent
	}
}

// IsRepo reports whether dir is inside a git working tree.
func IsRepo(stat func(string) (os.FileInfo, error), dir string) bool {
	_, ok := RepoRoot(stat, dir)
	return ok
}
