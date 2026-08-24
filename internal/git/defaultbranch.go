package git

import "strings"

// DefaultBranch discovers the default branch on the origin remote by running
// `git ls-remote --symref origin HEAD` via gitExec. It returns the branch name
// and true when origin advertises a HEAD symref, or "" and false when it does
// not (an empty remote, or a server that hides the symref). The error is
// non-nil only when the git command itself fails; callers treat false and a
// command error the same way — no default can be established, skip the check.
//
// The dir argument is passed to gitExec; pass "" for the current directory.
func DefaultBranch(gitExec func(dir string, args ...string) (string, error), dir string) (string, bool, error) {
	out, err := gitExec(dir, "ls-remote", "--symref", "origin", "HEAD")
	if err != nil {
		return "", false, err
	}
	for _, line := range strings.Split(out, "\n") {
		rest, found := strings.CutPrefix(line, "ref: refs/heads/")
		if !found {
			continue
		}
		name, found := strings.CutSuffix(rest, "\tHEAD")
		if !found {
			continue
		}
		if name != "" {
			return name, true, nil
		}
	}
	return "", false, nil
}
