package changelog

import (
	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/git"
)

// The checks pk changelog and pk release both run before they act. A
// release commit that pk changelog accepts must be one pk release can
// publish, so the two commands ask the same questions here and refuse
// in the same words.

// StartBranchHint is the advice for a repository whose only branch is
// the one that cannot be worked on.
const StartBranchHint = "to start one: git switch -c develop && git push -u origin develop"

// DetachedHead is the refusal for a detached HEAD: a release commit is
// made on a branch and published from it, and a detached HEAD has none
// to check against origin, to merge, or to push.
func DetachedHead() error {
	return cli.WithHint(
		cli.Statef("HEAD is detached; a release is made on a branch and published from it"),
		"switch to the branch that holds the work: git switch <branch>")
}

// CheckTrunkBranch is the trunk-flow check: releases publish from
// origin's default branch, so any other branch is refused. The bool
// reports that origin named its default branch and branch is it; it is
// false, with no error, when origin advertises no default and the check
// could not run.
func CheckTrunkBranch(root, branch string) (bool, error) {
	def, ok, err := git.DefaultBranch(root)
	if err != nil || !ok {
		return false, nil
	}
	if branch != def {
		return false, cli.WithHint(
			cli.Statef("you're on %q but the default branch on origin is %q; trunk flow releases from the default branch", branch, def),
			"land %s on %s first (a pull request, or a merge you have decided on), then on an up-to-date %s: pk changelog && pk release", branch, def, def)
	}
	return true, nil
}

// CheckOnOrigin refuses a branch origin does not have: the release
// push needs it there, and a clear refusal beats the fetch failure a
// local-only branch would produce later.
func CheckOnOrigin(root, branch string) error {
	onOrigin, err := git.RemoteBranchExists(root, branch)
	if err != nil {
		return err
	}
	if !onOrigin {
		return cli.WithHint(
			cli.Statef("%s does not exist on origin", branch),
			"to push it: git push -u origin %s", branch)
	}
	return nil
}
