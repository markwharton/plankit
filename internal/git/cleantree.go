package git

import (
	"errors"
	"fmt"
	"strings"
)

// ErrDirtyTree is returned by CheckCleanTree when the working tree has
// uncommitted changes.
var ErrDirtyTree = errors.New("working tree is not clean — commit or stash changes first")

// CheckCleanTree runs `git status --porcelain` via gitExec and returns
// nil if the working tree is clean, ErrDirtyTree if it's dirty, or a
// wrapped error if the git command itself fails.
//
// The dir argument is passed to gitExec; pass "" for the current directory.
func CheckCleanTree(gitExec func(dir string, args ...string) (string, error), dir string) error {
	status, err := gitExec(dir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return ErrDirtyTree
	}
	return nil
}
