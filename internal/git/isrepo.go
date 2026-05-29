package git

import (
	"os"
	"path/filepath"
)

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
