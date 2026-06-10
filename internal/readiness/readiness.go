// Package readiness evaluates whether a repository is ready for the release
// commands (pk changelog, pk release) given its .pk.json configuration.
//
// The checks are offline only: they read local refs (tags and remote-tracking
// branches under refs/remotes/origin/), never the network, so they reflect
// state as of the last fetch. pk release keeps its own authoritative network
// pre-flight; readiness exists so the gaps surface in pk status before a
// release attempt fails on them.
package readiness

import (
	"fmt"
	"strings"

	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/version"
)

// Check is one readiness fact with its next step when unmet.
type Check struct {
	Label string // e.g. "baseline tag", "develop on origin"
	OK    bool
	Value string // e.g. "v0.0.0", "missing", "ok"
	Hint  string // exact next-step command, "" when OK
	Or    string // git equivalent of Hint, "" when none
}

// Ready reports whether every check passed.
func Ready(checks []Check) bool {
	for _, c := range checks {
		if !c.OK {
			return false
		}
	}
	return true
}

// Evaluate returns the release-readiness checks for the repository at dir.
// The checks are keyed to what .pk.json declares: with release.branch set
// (merge flow) it verifies the baseline tag, a working branch distinct from
// the release branch, and both branches on origin; without it (trunk flow)
// only the baseline tag and the current branch on origin. It never nags
// about layers the configuration hasn't opted into.
func Evaluate(gitExec func(dir string, args ...string) (string, error), dir string, conf config.PkConfig) []Check {
	var checks []Check

	// Baseline tag: pk changelog has nothing to diff from without one.
	if tag, ok := ValidSemverTag(gitExec, dir); ok {
		checks = append(checks, Check{Label: "baseline tag", OK: true, Value: tag})
	} else {
		checks = append(checks, Check{
			Label: "baseline tag",
			Value: "missing",
			Hint:  "To anchor at v0.0.0: pk setup --baseline --push",
			Or:    "git tag v0.0.0 && git push origin v0.0.0",
		})
	}

	branch := currentBranch(gitExec, dir)
	releaseBranch := conf.Release.Branch

	if releaseBranch != "" && branch == releaseBranch {
		// Merge flow with no working branch: the exact dead-end pk release
		// refuses with "you're on the release branch".
		checks = append(checks, Check{
			Label: "working branch",
			Value: fmt.Sprintf("on release branch %s", releaseBranch),
			Hint:  "To start one: git switch -c develop && git push -u origin develop",
		})
	} else if branch != "" {
		// Current branch on origin: pk changelog refuses local-only branches.
		checks = append(checks, originCheck(gitExec, dir, branch))
	}

	if releaseBranch != "" && branch != releaseBranch {
		checks = append(checks, originCheck(gitExec, dir, releaseBranch))
	}

	return checks
}

// originCheck verifies that branch has a remote-tracking ref on origin.
func originCheck(gitExec func(dir string, args ...string) (string, error), dir, branch string) Check {
	c := Check{Label: branch + " on origin"}
	if _, err := gitExec(dir, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+branch); err == nil {
		c.OK = true
		c.Value = "ok"
		return c
	}
	c.Value = "missing"
	c.Hint = fmt.Sprintf("To publish it: git push -u origin %s", branch)
	return c
}

// HasOtherLocalBranch reports whether the repository has a local branch
// besides branch. Failure-point hints use it to decide between "switch to
// your working branch" (one exists) and "create one" (none does).
func HasOtherLocalBranch(gitExec func(dir string, args ...string) (string, error), dir, branch string) bool {
	out, err := gitExec(dir, "branch", "--format=%(refname:short)")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if name := strings.TrimSpace(line); name != "" && name != branch {
			return true
		}
	}
	return false
}

// currentBranch returns the checked-out branch name, or "" when detached or
// on error.
func currentBranch(gitExec func(dir string, args ...string) (string, error), dir string) string {
	out, err := gitExec(dir, "branch", "--show-current")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// ValidSemverTag returns the first tag matching "v*" that parses as a valid
// semver (per pk changelog's acceptance rule), or "", false if none exists.
func ValidSemverTag(gitExec func(dir string, args ...string) (string, error), dir string) (string, bool) {
	output, err := gitExec(dir, "tag", "--list", "v*", "--sort=-v:refname")
	if err != nil || output == "" {
		return "", false
	}
	for _, line := range strings.Split(output, "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}
		if _, ok := version.ParseSemver(tag); ok {
			return tag, true
		}
	}
	return "", false
}
