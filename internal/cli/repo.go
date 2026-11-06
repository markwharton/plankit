package cli

import (
	"errors"

	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
)

// Root returns the git repository that holds dir, or the one refusal
// every command gives for a directory that is not in one. A command
// that knows a better next step adds its own hint to the same error.
func Root(dir string) (string, error) {
	root, ok := git.FindRoot(dir)
	if !ok {
		return "", Statef("not a git repository: %s", dir)
	}
	return root, nil
}

// InitHint is the one way pk names the fix for a repository that has
// no policy file.
const InitHint = "run pk init to write " + config.FileName

// Policy loads root's policy file for a command a person typed. A
// repository without one is the one refusal, with InitHint; a file that
// does not load is its own error. Hooks call config.Load instead: they
// stay silent in an unconfigured repository, and have to tell that
// case from a file that fails.
func Policy(root string) (*config.PkConfig, error) {
	cfg, err := config.Load(root)
	if errors.Is(err, config.ErrNotConfigured) {
		return nil, WithHint(Statef("%v", err), "%s", InitHint)
	}
	if err != nil {
		return nil, Statef("%v", err)
	}
	return cfg, nil
}
