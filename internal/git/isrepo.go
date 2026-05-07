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

// IsRepo reports whether dir is inside a git working tree. It walks up
// parent directories looking for a .git entry (directory or file — git
// submodules and worktrees use a file). Returns false if no .git is
// found up to the filesystem root.
//
// Accepts an injected stat function for testability; pass os.Stat for
// real filesystem access.
func IsRepo(stat func(string) (os.FileInfo, error), dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	for {
		if _, err := stat(filepath.Join(abs, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			// Reached filesystem root.
			return false
		}
		abs = parent
	}
}
