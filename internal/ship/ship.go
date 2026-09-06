// Package ship implements pk ship: changelog then release in one
// invocation, so a release is a single command that can run
// unattended. It composes the two commands at their public boundary
// (cli.RunIO) rather than sharing internals, and it carries no state
// of its own: what remains to be done is derived from the Release-Tag
// trailer, the same source of truth the two halves already use. A
// pending trailer means changelog is done, so ship skips to release;
// no trailer means run both. That makes ship resumable: a run
// interrupted between the halves completes on rerun.
package ship

import (
	"fmt"

	"github.com/markwharton/plankit/internal/changelog"
	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/release"
)

// Cmd is the ship command.
var Cmd = &cli.Command{
	Name:    "ship",
	Summary: "Run changelog then release: cut the pending work as one release",
	Flags: []cli.FlagSpec{
		{Name: "bump", Type: cli.StringFlag, Usage: "Override the version bump: major, minor, or patch (passed to changelog)"},
		{Name: "dry-run", Type: cli.BoolFlag, Usage: "Rehearse: preview the section, or the release when one is already pending"},
		{Name: "exclude", Type: cli.StringFlag, Usage: "Comma-separated commit SHAs to drop from the section (passed to changelog)"},
	},
	Run: run,
}

func run(ctx *cli.Context) error {
	root, ok := git.FindRoot(ctx.ProjectDir)
	if !ok {
		return cli.Statef("not a git repository: %s", ctx.ProjectDir)
	}
	dryRun := ctx.Bool("dry-run")

	// The trailer decides what remains. Pending means changelog already
	// ran (this invocation or an earlier one), so ship resumes at
	// release rather than tripping changelog's already-pending refusal.
	_, pending, err := changelog.ReadReleaseTagTrailer(root)
	haveTrailer := err == nil

	if haveTrailer {
		if ctx.String("bump") != "" || ctx.String("exclude") != "" {
			msg.Warnf(ctx.Stderr, "--bump and --exclude apply to changelog, which already ran (Release-Tag: %s); ignoring", pending)
		}
		fmt.Fprintf(ctx.Stderr, "Release-Tag %s already pending; skipping changelog\n", pending)
	} else {
		args := []string{"pk", "changelog", "--project-dir", root}
		if b := ctx.String("bump"); b != "" {
			args = append(args, "--bump", b)
		}
		if x := ctx.String("exclude"); x != "" {
			args = append(args, "--exclude", x)
		}
		if dryRun {
			args = append(args, "--dry-run")
		}
		if ctx.Quiet {
			args = append(args, "--quiet")
		}
		if code := cli.RunIO(args, []*cli.Command{changelog.Cmd}, nil, ctx.Stdout, ctx.Stderr); code != cli.ExitOK {
			// The child already reported; only the code remains.
			return cli.Silent(code)
		}
		if dryRun {
			// A dry changelog leaves no trailer, so the release half has
			// nothing to rehearse against. Say so instead of failing.
			if !ctx.Quiet {
				msg.Notef(ctx.Stderr, "release rehearses against the changelog commit; run pk ship to execute both, or pk ship --dry-run again once a release is pending")
			}
			return nil
		}
	}

	args := []string{"pk", "release", "--project-dir", root}
	if dryRun {
		args = append(args, "--dry-run")
	}
	if ctx.Quiet {
		args = append(args, "--quiet")
	}
	if code := cli.RunIO(args, []*cli.Command{release.Cmd}, nil, ctx.Stdout, ctx.Stderr); code != cli.ExitOK {
		if !ctx.Quiet && !dryRun {
			// The changelog commit stands; rerunning ship resumes here.
			msg.Hintf(ctx.Stderr, "the release commit remains pending; rerun pk ship to retry, or pk changelog --undo to unwind")
		}
		return cli.Silent(code)
	}
	return nil
}
