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
	root, err := cli.Root(ctx.ProjectDir)
	if err != nil {
		return err
	}
	w := ctx.Stderr
	dryRun := ctx.Bool("dry-run")

	sourceBranch, err := git.CurrentBranch(root)
	if err != nil {
		return err
	}
	if sourceBranch == "" {
		return changelog.DetachedHead()
	}

	pk, err := cli.Policy(root)
	if err != nil {
		return err
	}
	releaseBranch := pk.Release.Branch
	// A branch name can never start with -; refuse before it reaches git
	// argv, where it would parse as an option.
	if strings.HasPrefix(releaseBranch, "-") {
		return cli.Statef("invalid release.branch %q in .pk.json; branch names cannot start with -", releaseBranch)
	}
	flow := pk.Release.Flow()
	needsMerge := flow == config.MergeFlow && sourceBranch != releaseBranch

	// On the release branch with merge flow configured: refuse. With no
	// other local branch this is the main-only dead-end, so point at the
	// way out rather than an instruction the person cannot follow.
	if flow == config.MergeFlow && sourceBranch == releaseBranch {
		err := cli.Statef("you're on the release branch %q; switch to your working branch first", releaseBranch)
		if !git.HasOtherLocalBranch(root, releaseBranch) {
			err = cli.WithHint(err, "%s", changelog.StartBranchHint)
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

	// A release commit keeps its trailer after it ships, so the tag, not
	// the trailer, says whether anything is left to do here.
	switch state, err := changelog.ReleaseTagState(root, tag); {
	case err != nil:
		return err
	case state == changelog.TagOnHead:
		return cli.Statef("HEAD is already tagged %s; nothing to release", tag)
	case state == changelog.TagElsewhere:
		return cli.WithHint(
			cli.Statef("tag %s already exists on another commit", tag),
			"to restage under the next version: pk changelog --undo, then pk changelog")
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
	// the branch-on-origin check below. Skipped when origin advertises
	// no HEAD symref.
	defaultVerified := false
	if flow == config.TrunkFlow {
		if defaultVerified, err = changelog.CheckTrunkBranch(root, sourceBranch); err != nil {
			return err
		}
	}

	// Source must exist on origin.
	if err := changelog.CheckOnOrigin(root, sourceBranch); err != nil {
		return err
	}
	msg.Itemf(w, "%s exists on origin", sourceBranch)

	// Source must not be behind origin.
	if _, err := git.Exec(root, "fetch", "origin", sourceBranch, "--quiet"); err != nil {
		return changelog.OriginUnreachable(err)
	}
	upToDate, err := git.IsAncestor(root, "origin/"+sourceBranch, "HEAD")
	if err != nil {
		return err
	}
	if !upToDate {
		return cli.Statef("local %s is behind origin/%s; pull first", sourceBranch, sourceBranch)
	}
	msg.Itemf(w, "Not behind origin/%s", sourceBranch)

	// Merge flow: the release branch must exist locally or on origin
	// before the flow switches to it, and origin's copy must be an
	// ancestor of HEAD or the atomic push would be rejected. Catch both
	// here, before tagging. A branch only local (pk init without --push)
	// is a first release: the release push creates it on origin.
	if needsMerge {
		onOrigin, err := git.RemoteBranchExists(root, releaseBranch)
		if err != nil {
			return changelog.OriginUnreachable(err)
		}
		switch {
		case onOrigin:
			if _, err := git.Exec(root, "fetch", "origin", releaseBranch, "--quiet"); err != nil {
				return changelog.OriginUnreachable(err)
			}
			contained, err := git.IsAncestor(root, "origin/"+releaseBranch, "HEAD")
			if err != nil {
				return err
			}
			if !contained {
				return cli.WithHint(
					cli.Statef("origin/%s has diverged from %s; the release push would be rejected", releaseBranch, sourceBranch),
					"to reconcile, on %s: git merge origin/%s", sourceBranch, releaseBranch)
			}
			msg.Itemf(w, "%s exists on origin", releaseBranch)
		case git.BranchExists(root, releaseBranch):
			msg.Itemf(w, "%s exists locally; the release push creates it on origin", releaseBranch)
		default:
			return cli.WithHint(
				cli.Statef("release branch %s does not exist locally or on origin", releaseBranch),
				"to create it: git branch %s && git push -u origin %s", releaseBranch, releaseBranch)
		}
	}
	last, _, err := changelog.LatestTag(root)
	if err != nil {
		return err
	}
	outside, err := changelog.TagOutsideHistory(root, sourceBranch, last)
	if err != nil {
		return err
	}
	if outside != "" {
		return cli.Statef("%s", outside)
	}
	if last != "" {
		msg.Itemf(w, "Last tag %s is in %s's history", last, sourceBranch)
	}
	msg.Itemf(w, "Release-Tag trailer: %s", tag)

	// From here, mutations happen under the rollback defer: an unpushed
	// tag is deleted, a merged release branch is reset, and the checkout
	// returns to the source branch, so an aborted release leaves the
	// repository as it found it.
	switchBack := func() {
		if _, err := git.Exec(root, "switch", sourceBranch); err != nil {
			msg.Warnf(w, "failed to switch back to %s: %v", sourceBranch, err)
		}
	}
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
			switchBack()
		}
	}()

	if needsMerge {
		if dryRun {
			ff, err := git.IsAncestor(root, releaseBranch, sourceBranch)
			if err != nil {
				return err
			}
			if !ff {
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
	} else if flow == config.TrunkFlow {
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
	if err := git.CreateTag(root, tag); err != nil {
		return err
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
		return cli.Statef("%v", err)
	}
	released = true
	msg.Itemf(w, "Pushed %s and %s", pushBranch, tag)

	if needsMerge {
		switchBack()
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
