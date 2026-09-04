// Package release implements pk release: it reads the Release-Tag
// trailer pk changelog wrote on HEAD, runs pre-flight checks, creates
// the git tag, and pushes to origin. With release.branch configured, the
// current branch fast-forwards into the release branch first; without
// it, trunk flow tags and pushes the default branch directly.
//
// Ported from v1, including the failure discipline: everything mutating
// happens under a defer that deletes an unpushed tag, rolls back the
// merge, and switches back to the source branch, so an aborted release
// leaves the repository as it found it. v2 deltas: exits follow the
// layer-0 taxonomy, ceremony prints on stderr, stdout stays empty.
package release

import (
	"errors"
	"fmt"
	"strings"

	"github.com/markwharton/plankit/internal/changelog"
	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hookio"
	"github.com/markwharton/plankit/internal/msg"
)

// Cmd is the release command.
var Cmd = &cli.Command{
	Name:    "release",
	Summary: "Tag the pending release from the Release-Tag trailer and push it to origin",
	Flags: []cli.FlagSpec{
		{Name: "dry-run", Type: cli.BoolFlag, Usage: "Validate without merging, tagging, or pushing"},
	},
	Run: run,
}

func run(ctx *cli.Context) error {
	if ctx.Format == "json" {
		return cli.Usagef("--format json is not supported by pk release")
	}
	root, ok := git.FindRoot(ctx.ProjectDir)
	if !ok {
		return cli.Statef("not a git repository: %s", ctx.ProjectDir)
	}
	w := ctx.Stderr
	dryRun := ctx.Bool("dry-run")

	sourceBranch, err := git.Exec(root, "branch", "--show-current")
	if err != nil {
		return fmt.Errorf("git branch failed: %v", err)
	}

	pk, err := config.Load(root)
	if errors.Is(err, config.ErrNotConfigured) {
		return cli.WithHint(cli.Statef("%v", err), "run pk init to configure this repository")
	}
	if err != nil {
		return cli.Statef("%v", err)
	}
	releaseBranch := pk.Release.Branch
	// A branch name can never start with -; refuse before it reaches git
	// argv, where it would parse as an option.
	if strings.HasPrefix(releaseBranch, "-") {
		return cli.Statef("invalid release.branch %q in .pk.json; branch names cannot start with -", releaseBranch)
	}
	needsMerge := releaseBranch != "" && sourceBranch != releaseBranch

	// On the release branch with merge flow configured: refuse. With no
	// other local branch this is the main-only dead-end, so point at the
	// way out rather than an instruction the person cannot follow.
	if releaseBranch != "" && sourceBranch == releaseBranch {
		err := cli.Statef("you're on the release branch %q; switch to your working branch first", releaseBranch)
		if !git.HasOtherLocalBranch(root, releaseBranch) {
			err = cli.WithHint(err, "to start one: git switch -c develop && git push -u origin develop, then pk changelog && pk release")
		}
		return err
	}

	_, tag, err := changelog.ReadReleaseTagTrailer(root)
	if err != nil {
		if errors.Is(err, changelog.ErrNoTrailer) {
			return cli.WithHint(cli.Statef("no Release-Tag trailer on HEAD"), "run pk changelog first")
		}
		return cli.Statef("%v", err)
	}

	if existing, err := git.Exec(root, "tag", "--list", tag); err != nil {
		return fmt.Errorf("git tag --list failed: %v", err)
	} else if strings.TrimSpace(existing) != "" {
		return cli.Statef("tag %s already exists locally; nothing to release", tag)
	}

	msg.Banner(w, "Release "+tag)
	fmt.Fprintln(w, "")
	msg.Section(w, "Pre-flight checks")

	if err := git.CheckCleanTree(root); err != nil {
		return cli.Statef("%v", err)
	}
	msg.Itemf(w, "Clean working tree")

	// Trunk flow only: releases publish from origin's default branch, so
	// refuse any other. Re-checks what pk changelog already refused, like
	// the branch-on-origin check below. Skipped on detached HEAD and when
	// origin advertises no HEAD symref.
	defaultVerified := false
	if releaseBranch == "" && sourceBranch != "" {
		if def, ok, derr := git.DefaultBranch(root); derr == nil && ok {
			if sourceBranch != def {
				return cli.WithHint(
					cli.Statef("you're on %q but the default branch on origin is %q; trunk flow releases from the default branch", sourceBranch, def),
					"to release this work from %s: git switch %s && git merge %s, then pk release", def, def, sourceBranch)
			}
			defaultVerified = true
		}
	}

	// Source must exist on origin: a clear error beats the cryptic fetch
	// failure a local-only branch would produce later.
	if _, err := git.Exec(root, "ls-remote", "--exit-code", "--heads", "origin", sourceBranch); err != nil {
		return cli.WithHint(cli.Statef("%s does not exist on origin", sourceBranch),
			"to push it: git push -u origin %s", sourceBranch)
	}
	msg.Itemf(w, "%s exists on origin", sourceBranch)

	// Source must not be behind origin.
	if _, err := git.Exec(root, "fetch", "origin", sourceBranch, "--quiet"); err != nil {
		return fmt.Errorf("git fetch failed: %v", err)
	}
	mergeBase, err := git.Exec(root, "merge-base", "HEAD", "origin/"+sourceBranch)
	if err != nil {
		return fmt.Errorf("git merge-base failed: %v", err)
	}
	remote, err := git.Exec(root, "rev-parse", "origin/"+sourceBranch)
	if err != nil {
		return fmt.Errorf("git rev-parse failed: %v", err)
	}
	if mergeBase != remote {
		return cli.Statef("local %s is behind origin/%s; pull first", sourceBranch, sourceBranch)
	}
	msg.Itemf(w, "Not behind origin/%s", sourceBranch)

	// Merge flow: the release branch must resolve locally or on origin
	// before the flow switches to it, and origin's copy must be an
	// ancestor of HEAD or the atomic push would be rejected. Catch both
	// here, before tagging.
	if needsMerge {
		if _, err := git.Exec(root, "fetch", "origin", releaseBranch, "--quiet"); err != nil {
			msg.Warnf(w, "failed to fetch %s from origin: %v (continuing with local state)", releaseBranch, err)
		}
		_, localErr := git.Exec(root, "rev-parse", "--verify", "--quiet", "refs/heads/"+releaseBranch)
		_, remoteErr := git.Exec(root, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+releaseBranch)
		if localErr != nil && remoteErr != nil {
			return cli.WithHint(
				cli.Statef("release branch %s does not exist locally or on origin", releaseBranch),
				"to create it: git branch %s && git push -u origin %s", releaseBranch, releaseBranch)
		}
		if remoteErr == nil {
			if _, err := git.Exec(root, "merge-base", "--is-ancestor", "origin/"+releaseBranch, "HEAD"); err != nil {
				return cli.WithHint(
					cli.Statef("origin/%s has diverged from %s; the release push would be rejected", releaseBranch, sourceBranch),
					"to reconcile, on %s: git merge origin/%s", sourceBranch, releaseBranch)
			}
		}
		msg.Itemf(w, "%s exists", releaseBranch)
	}
	msg.Itemf(w, "Release-Tag trailer: %s", tag)

	// From here, mutations happen under the rollback defer: an unpushed
	// tag is deleted, a merged release branch is reset, and the checkout
	// returns to the source branch, so an aborted release leaves the
	// repository as it found it.
	tagCreated := false
	released := false
	switchedBack := true
	preMergeHead := ""
	defer func() {
		if tagCreated && !released {
			if _, err := git.Exec(root, "tag", "-d", tag); err != nil {
				msg.Warnf(w, "failed to delete local tag %s: %v", tag, err)
			} else {
				fmt.Fprintf(w, "Cleaned up local tag %s\n", tag)
			}
		}
		if preMergeHead != "" && !released {
			if _, err := git.Exec(root, "reset", "--hard", preMergeHead); err != nil {
				msg.Warnf(w, "failed to roll back merge on %s: %v", releaseBranch, err)
			} else {
				fmt.Fprintf(w, "Rolled back merge on %s\n", releaseBranch)
			}
		}
		if !switchedBack {
			if _, err := git.Exec(root, "switch", sourceBranch); err != nil {
				msg.Warnf(w, "failed to switch back to %s: %v", sourceBranch, err)
			}
		}
	}()

	if needsMerge {
		if dryRun {
			if _, err := git.Exec(root, "merge-base", "--is-ancestor", releaseBranch, sourceBranch); err != nil {
				return cli.Statef("merge would not be fast-forward; %s has diverged from %s. Resolve on %s manually, then try again.", releaseBranch, sourceBranch, releaseBranch)
			}
			msg.Itemf(w, "Would merge %s into %s (fast-forward)", sourceBranch, releaseBranch)
		} else {
			if _, err := git.Exec(root, "switch", releaseBranch); err != nil {
				return fmt.Errorf("failed to switch to %s: %v", releaseBranch, err)
			}
			switchedBack = false
			head, err := git.Exec(root, "rev-parse", "HEAD")
			if err != nil {
				return fmt.Errorf("failed to read HEAD of %s: %v", releaseBranch, err)
			}
			preMergeHead = head
			if _, err := git.Exec(root, "merge", "--ff-only", sourceBranch); err != nil {
				return cli.Statef("merge failed; %s has diverged from %s (not fast-forward). Resolve on %s manually, then try again.", releaseBranch, sourceBranch, releaseBranch)
			}
			msg.Itemf(w, "Merged %s into %s", sourceBranch, releaseBranch)
		}
	} else if releaseBranch == "" {
		msg.Itemf(w, "Trunk flow (no release.branch in .pk.json)")
		if defaultVerified {
			msg.Itemf(w, "On %s (default branch on origin)", sourceBranch)
		} else {
			// Doubles as disclosure that the default-branch check could
			// not run (origin advertises no HEAD symref).
			msg.Itemf(w, "On %s branch", sourceBranch)
		}
	}

	// Env shared by both hooks, matching the changelog hooks' contract:
	// VERSION without the v (for stamping files), TAG with it (the ref
	// name). The tag ref itself does not exist until after preRelease.
	hookEnv := map[string]string{
		"VERSION": strings.TrimPrefix(tag, "v"),
		"TAG":     tag,
	}

	// preRelease runs before the tag exists: rehearsable in --dry-run,
	// and a hook that commits produces a commit the tag then covers. A
	// hook that needs the tag ref wants prePush.
	if h := pk.Release.Hooks.PreRelease; h != "" {
		fmt.Fprintln(w, "")
		msg.Section(w, "Pre-release hook")
		msg.Itemf(w, "%s", h)
		if err := hookio.RunScript(w, root, h, hookEnv); err != nil {
			return cli.Statef("pre-release hook failed: %v", err)
		}
		msg.Itemf(w, "Hook passed")
	}

	// The push publishes the release branch in merge flow, the source
	// branch in trunk flow; computed before the dry-run exit so the
	// rehearsal names the branch the real push would publish.
	pushBranch := sourceBranch
	if needsMerge {
		pushBranch = releaseBranch
	}

	if dryRun {
		fmt.Fprintln(w, "")
		msg.Section(w, "Dry run complete")
		msg.Itemf(w, "Would create tag %s", tag)
		msg.Itemf(w, "Would push %s and %s", pushBranch, tag)
		msg.Itemf(w, "All checks passed. Run without --dry-run to release.")
		return nil
	}

	// Tag HEAD: source HEAD in trunk flow, release-branch HEAD after the
	// fast-forward in merge flow; the same commit either way.
	if _, err := git.Exec(root, "tag", tag); err != nil {
		return fmt.Errorf("git tag failed: %v", err)
	}
	tagCreated = true
	fmt.Fprintf(w, "\nCreated local tag %s\n", tag)

	// prePush runs with the tag ref in existence (signing, artifact
	// builds keyed on the tag). A failure aborts before any push; the
	// defer removes the tag and rolls back the merge. Never runs in
	// --dry-run, which returned above: preRelease is the rehearsable slot.
	if h := pk.Release.Hooks.PrePush; h != "" {
		fmt.Fprintln(w, "")
		msg.Section(w, "Pre-push hook")
		msg.Itemf(w, "%s", h)
		if err := hookio.RunScript(w, root, h, hookEnv); err != nil {
			return cli.Statef("pre-push hook failed: %v", err)
		}
		msg.Itemf(w, "Hook passed")
	}

	fmt.Fprintln(w, "")
	msg.Section(w, "Pushing to origin")
	if _, err := git.Exec(root, "push", "--atomic", "origin", pushBranch, tag); err != nil {
		return cli.Statef("git push failed: %v", err)
	}
	released = true
	msg.Itemf(w, "Pushed %s and %s", pushBranch, tag)

	if needsMerge {
		if _, err := git.Exec(root, "switch", sourceBranch); err != nil {
			msg.Warnf(w, "failed to switch back to %s: %v", sourceBranch, err)
		}
		switchedBack = true
		if _, err := git.Exec(root, "push", "origin", sourceBranch); err != nil {
			msg.Warnf(w, "failed to push %s: %v", sourceBranch, err)
		} else {
			msg.Itemf(w, "Pushed %s", sourceBranch)
		}
	}

	fmt.Fprintln(w, "")
	msg.Banner(w, "Release "+tag+" complete")
	return nil
}
