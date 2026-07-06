// Package release implements the pk release command.
// It reads the Release-Tag trailer from the HEAD commit (written by pk
// changelog), creates the git tag, validates pre-flight checks, and pushes
// the release to origin. When release.branch is configured in .pk.json, it
// merges the current branch into the release branch before pushing.
package release

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/changelog"
	"github.com/markwharton/plankit/internal/config"
	pkgit "github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hooks"
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/paths"
	"github.com/markwharton/plankit/internal/readiness"
)

// Config holds injectable dependencies for testing.
type Config struct {
	Stderr    io.Writer
	Dir       string
	GitExec   func(dir string, args ...string) (string, error)
	ReadFile  func(name string) ([]byte, error)
	RunScript func(dir string, command string, env map[string]string) error

	// DryRun validates without merging or pushing.
	DryRun bool
}

// DefaultConfig returns a Config wired to real implementations.
func DefaultConfig() Config {
	return Config{
		Stderr:    os.Stderr,
		GitExec:   pkgit.Exec,
		ReadFile:  os.ReadFile,
		RunScript: hooks.RunScript,
	}
}

// Run executes the release command. Returns the process exit code.
func Run(cfg Config) int {
	// 0. Get current branch (implicit source).
	sourceBranch, err := cfg.GitExec(cfg.Dir, "branch", "--show-current")
	if err != nil {
		msg.Errorf(cfg.Stderr, "git branch failed: %v", err)
		return 1
	}
	sourceBranch = strings.TrimSpace(sourceBranch)

	// 2. Load release config from the repository root.
	releaseConf, err := loadReleaseConfig(cfg.ReadFile, filepath.Join(cfg.Dir, paths.PkConfig))
	if err != nil {
		msg.Errorf(cfg.Stderr, "%v", err)
		return 1
	}
	releaseBranch := releaseConf.Branch
	// A branch name can never start with -; refuse before it reaches git argv,
	// where it would be parsed as an option.
	if strings.HasPrefix(releaseBranch, "-") {
		msg.Errorf(cfg.Stderr, "invalid release.branch %q in .pk.json; branch names cannot start with -", releaseBranch)
		return 1
	}
	needsMerge := releaseBranch != "" && sourceBranch != releaseBranch

	// 3. If releaseBranch is configured and we're already on it, refuse.
	// With no other local branch this is the main-only dead-end: the user
	// configured merge flow but never created a working branch, so point at
	// the way out instead of an instruction they can't follow.
	if releaseBranch != "" && sourceBranch == releaseBranch {
		msg.Errorf(cfg.Stderr, "you're on the release branch %q; switch to your working branch first", releaseBranch)
		if !readiness.HasOtherLocalBranch(cfg.GitExec, cfg.Dir, releaseBranch) {
			msg.Hintf(cfg.Stderr, "To start one: git switch -c develop && git push -u origin develop")
			msg.Hintf(cfg.Stderr, "Then: pk changelog && pk release")
		}
		return 1
	}

	// 4. Read Release-Tag trailer from HEAD.
	_, tag, err := changelog.ReadReleaseTagTrailer(cfg.GitExec, cfg.Dir)
	if err != nil {
		if errors.Is(err, changelog.ErrNoTrailer) {
			msg.Errorf(cfg.Stderr, "no Release-Tag trailer on HEAD; run pk changelog first")
		} else {
			msg.Errorf(cfg.Stderr, "%v", err)
		}
		return 1
	}

	// 5. Refuse if the tag already exists locally.
	existingTag, err := cfg.GitExec(cfg.Dir, "tag", "--list", tag)
	if err != nil {
		msg.Errorf(cfg.Stderr, "git tag --list failed: %v", err)
		return 1
	}
	if strings.TrimSpace(existingTag) != "" {
		msg.Errorf(cfg.Stderr, "tag %s already exists locally; nothing to release", tag)
		return 1
	}

	// 6. Print header.
	msg.Banner(cfg.Stderr, "Release "+tag)
	fmt.Fprintln(cfg.Stderr, "")

	msg.Section(cfg.Stderr, "Pre-flight checks")

	// 6. Pre-flight: clean working tree.
	if err := pkgit.CheckCleanTree(cfg.GitExec, cfg.Dir); err != nil {
		msg.Errorf(cfg.Stderr, "%v", err)
		return 1
	}
	msg.Itemf(cfg.Stderr, "Clean working tree")

	// 7. Pre-flight: source branch exists on origin. Gives a clear error
	// for the "local-only branch" case that would otherwise surface as a
	// cryptic fetch failure.
	if _, err := cfg.GitExec(cfg.Dir, "ls-remote", "--exit-code", "--heads", "origin", sourceBranch); err != nil {
		msg.Errorf(cfg.Stderr, "%s does not exist on origin", sourceBranch)
		msg.Hintf(cfg.Stderr, "To push it: git push -u origin %s", sourceBranch)
		return 1
	}
	msg.Itemf(cfg.Stderr, "%s exists on origin", sourceBranch)

	// 7. Pre-flight: source branch not behind remote.
	_, err = cfg.GitExec(cfg.Dir, "fetch", "origin", sourceBranch, "--quiet")
	if err != nil {
		msg.Errorf(cfg.Stderr, "git fetch failed: %v", err)
		return 1
	}

	mergeBase, err := cfg.GitExec(cfg.Dir, "merge-base", "HEAD", "origin/"+sourceBranch)
	if err != nil {
		msg.Errorf(cfg.Stderr, "git merge-base failed: %v", err)
		return 1
	}

	remote, err := cfg.GitExec(cfg.Dir, "rev-parse", "origin/"+sourceBranch)
	if err != nil {
		msg.Errorf(cfg.Stderr, "git rev-parse failed: %v", err)
		return 1
	}

	mergeBase = strings.TrimSpace(mergeBase)
	remote = strings.TrimSpace(remote)
	if mergeBase != remote {
		msg.Errorf(cfg.Stderr, "local %s is behind origin/%s; pull first", sourceBranch, sourceBranch)
		return 1
	}
	msg.Itemf(cfg.Stderr, "Not behind origin/%s", sourceBranch)

	// Pre-flight: the release branch must resolve locally or on origin before
	// the merge flow tries to switch to it; otherwise the failure surfaces as
	// a raw git switch error (or a misleading not-fast-forward in dry-run).
	if needsMerge {
		_, localErr := cfg.GitExec(cfg.Dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+releaseBranch)
		_, remoteErr := cfg.GitExec(cfg.Dir, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+releaseBranch)
		if localErr != nil && remoteErr != nil {
			msg.Errorf(cfg.Stderr, "release branch %s does not exist locally or on origin", releaseBranch)
			msg.Hintf(cfg.Stderr, "To create it: git branch %s && git push -u origin %s", releaseBranch, releaseBranch)
			return 1
		}
		msg.Itemf(cfg.Stderr, "%s exists", releaseBranch)
	}

	msg.Itemf(cfg.Stderr, "Release-Tag trailer: %s", tag)

	// 8. Merge flow.
	tagCreated := false
	released := false
	switchedBack := true // default: no switch-back needed
	preMergeHead := ""   // release branch HEAD before ff merge
	defer func() {
		if tagCreated && !released {
			if _, err := cfg.GitExec(cfg.Dir, "tag", "-d", tag); err != nil {
				msg.Warnf(cfg.Stderr, "failed to delete local tag %s: %v", tag, err)
			} else {
				fmt.Fprintf(cfg.Stderr, "Cleaned up local tag %s\n", tag)
			}
		}
		if preMergeHead != "" && !released {
			if _, err := cfg.GitExec(cfg.Dir, "reset", "--hard", preMergeHead); err != nil {
				msg.Warnf(cfg.Stderr, "failed to roll back merge on %s: %v", releaseBranch, err)
			} else {
				fmt.Fprintf(cfg.Stderr, "Rolled back merge on %s\n", releaseBranch)
			}
		}
		if !switchedBack {
			if _, err := cfg.GitExec(cfg.Dir, "switch", sourceBranch); err != nil {
				msg.Warnf(cfg.Stderr, "failed to switch back to %s: %v", sourceBranch, err)
			}
		}
	}()
	if needsMerge {
		if cfg.DryRun {
			// Check fast-forward is possible without actually merging.
			_, err := cfg.GitExec(cfg.Dir, "merge-base", "--is-ancestor", releaseBranch, sourceBranch)
			if err != nil {
				msg.Errorf(cfg.Stderr, "merge would not be fast-forward; %s has diverged from %s. Resolve on %s manually, then try again.", releaseBranch, sourceBranch, releaseBranch)
				return 1
			}
			msg.Itemf(cfg.Stderr, "Would merge %s into %s (fast-forward)", sourceBranch, releaseBranch)
		} else {
			// Fetch release branch.
			if _, err := cfg.GitExec(cfg.Dir, "fetch", "origin", releaseBranch, "--quiet"); err != nil {
				msg.Warnf(cfg.Stderr, "failed to fetch %s from origin: %v (continuing with local state)", releaseBranch, err)
			}

			// Switch to release branch. Mark switchedBack=false so the
			// top-level defer switches back on any subsequent failure.
			if _, err := cfg.GitExec(cfg.Dir, "switch", releaseBranch); err != nil {
				msg.Errorf(cfg.Stderr, "failed to switch to %s: %v", releaseBranch, err)
				return 1
			}
			switchedBack = false

			// Capture release branch HEAD before merging for rollback.
			head, err := cfg.GitExec(cfg.Dir, "rev-parse", "HEAD")
			if err != nil {
				msg.Errorf(cfg.Stderr, "failed to read HEAD of %s: %v", releaseBranch, err)
				return 1
			}
			preMergeHead = strings.TrimSpace(head)

			// Merge from source branch (fast-forward only).
			if _, err := cfg.GitExec(cfg.Dir, "merge", "--ff-only", sourceBranch); err != nil {
				msg.Errorf(cfg.Stderr, "merge failed; %s has diverged from %s (not fast-forward). Resolve on %s manually, then try again.", releaseBranch, sourceBranch, releaseBranch)
				return 1
			}
			msg.Itemf(cfg.Stderr, "Merged %s into %s", sourceBranch, releaseBranch)
		}
	} else if releaseBranch == "" {
		// Trunk flow: no releaseBranch configured — tag HEAD, push current branch.
		// We already checked tag exists above. Just note the branch.
		msg.Itemf(cfg.Stderr, "Trunk flow (no release.branch in .pk.json)")
		msg.Itemf(cfg.Stderr, "On %s branch", sourceBranch)
	}

	// 9. Run preRelease hook if configured.
	if releaseConf.Hooks.PreRelease != "" {
		fmt.Fprintln(cfg.Stderr, "")
		msg.Section(cfg.Stderr, "Pre-release hook")
		msg.Itemf(cfg.Stderr, "%s", releaseConf.Hooks.PreRelease)
		if err := cfg.RunScript(cfg.Dir, releaseConf.Hooks.PreRelease, nil); err != nil {
			msg.Errorf(cfg.Stderr, "pre-release hook failed: %v", err)
			return 1
		}
		msg.Itemf(cfg.Stderr, "Hook passed")
	}

	// 10. Dry run complete (merge/trunk flows).
	if cfg.DryRun {
		fmt.Fprintln(cfg.Stderr, "")
		msg.Section(cfg.Stderr, "Dry run complete")
		msg.Itemf(cfg.Stderr, "Would create tag %s", tag)
		msg.Itemf(cfg.Stderr, "All checks passed. Run without --dry-run to release.")
		return 0
	}

	// 11. Create the local tag on HEAD (which is either source HEAD in trunk
	// flow or release-branch HEAD after the fast-forward merge — same commit).
	// Cleanup on subsequent failure is handled by the top-level defer.
	if _, err := cfg.GitExec(cfg.Dir, "tag", tag); err != nil {
		msg.Errorf(cfg.Stderr, "git tag failed: %v", err)
		return 1
	}
	tagCreated = true
	fmt.Fprintf(cfg.Stderr, "\nCreated local tag %s\n", tag)

	// 12. Push.
	fmt.Fprintln(cfg.Stderr, "")
	msg.Section(cfg.Stderr, "Pushing to origin")

	pushBranch := sourceBranch
	if needsMerge {
		pushBranch = releaseBranch
	}

	if _, err := cfg.GitExec(cfg.Dir, "push", "origin", pushBranch, tag); err != nil {
		msg.Errorf(cfg.Stderr, "git push failed: %v", err)
		return 1
	}
	released = true
	msg.Itemf(cfg.Stderr, "Pushed %s and %s", pushBranch, tag)

	// 13. Switch back and push source branch.
	if needsMerge {
		if _, err := cfg.GitExec(cfg.Dir, "switch", sourceBranch); err != nil {
			msg.Warnf(cfg.Stderr, "failed to switch back to %s: %v", sourceBranch, err)
		}
		switchedBack = true

		if _, err := cfg.GitExec(cfg.Dir, "push", "origin", sourceBranch); err != nil {
			msg.Warnf(cfg.Stderr, "failed to push %s: %v", sourceBranch, err)
		} else {
			msg.Itemf(cfg.Stderr, "Pushed %s", sourceBranch)
		}
	}

	fmt.Fprintln(cfg.Stderr, "")
	msg.Banner(cfg.Stderr, "Release "+tag+" complete")
	return 0
}

// Type aliases for config types used throughout this package.
type ReleaseSection = config.ReleaseSection

// loadReleaseConfig reads the release section from .pk.json at path.
// Returns an error if the file exists but contains malformed JSON.
func loadReleaseConfig(readFile func(string) ([]byte, error), path string) (ReleaseSection, error) {
	pk, err := config.Load(readFile, path)
	if err != nil {
		return ReleaseSection{}, err
	}
	return pk.Release, nil
}
