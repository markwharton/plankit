// pk - Plan-driven development toolkit for Claude Code.
//
// Provides plan preservation and protection for Claude Code projects.
// Designed to be invoked by Claude Code hooks configured in .claude/settings.json.
//
// Commands:
//
//	pk changelog   Generate CHANGELOG.md from conventional commits, commit, and tag
//	pk preserve    PostToolUse hook: preserve approved plans in docs/plans/
//	pk protect     PreToolUse hook: block edits to docs/plans/
//	pk release     Validate and push release to origin
//	pk setup       Configure a project's .claude/settings.json
//	pk version     Print version
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/markwharton/plankit/internal/changelog"
	"github.com/markwharton/plankit/internal/preserve"
	"github.com/markwharton/plankit/internal/protect"
	"github.com/markwharton/plankit/internal/release"
	"github.com/markwharton/plankit/internal/setup"
	"github.com/markwharton/plankit/internal/update"
	"github.com/markwharton/plankit/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "changelog":
		runChangelog(os.Args[2:])
	case "preserve":
		runPreserve(os.Args[2:])
	case "release":
		runRelease(os.Args[2:])
	case "protect":
		os.Exit(protect.Run(os.Stdin, os.Stdout, os.Stderr, os.Getenv))
	case "setup":
		runSetup(os.Args[2:])
	case "version", "--version", "-v":
		runVersion()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runChangelog(args []string) {
	fs := flag.NewFlagSet("changelog", flag.ExitOnError)
	bump := fs.String("bump", "", "Override version bump: major, minor, or patch")
	dryRun := fs.Bool("dry-run", false, "Preview without writing, committing, or tagging")
	fs.Parse(args)

	cfg := changelog.DefaultConfig()
	cfg.Bump = *bump
	cfg.DryRun = *dryRun

	os.Exit(changelog.Run(cfg))
}

func runPreserve(args []string) {
	fs := flag.NewFlagSet("preserve", flag.ExitOnError)
	notify := fs.Bool("notify", false, "Notify only, do not preserve")
	fs.Parse(args)

	cfg := preserve.DefaultConfig()
	cfg.Notify = *notify
	cfg.CheckUpdate = func() string {
		latest, available := update.Check(update.DefaultConfig(version.Version()))
		if available {
			return update.FormatNotice(latest, version.Version())
		}
		return ""
	}

	os.Exit(preserve.Run(cfg))
}

func runRelease(args []string) {
	fs := flag.NewFlagSet("release", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "Validate without pushing")
	branch := fs.String("branch", "main", "Expected branch for release")
	fs.Parse(args)

	cfg := release.DefaultConfig()
	cfg.DryRun = *dryRun
	cfg.Branch = *branch

	os.Exit(release.Run(cfg))
}

func runSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	projectDir := fs.String("project-dir", ".", "Project directory (default: current directory)")
	preserveMode := fs.String("preserve", "auto", "Plan preservation mode: auto or manual")
	fs.Parse(args)

	dir := *projectDir
	if dir == "." {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	}

	// Validate preserve mode.
	switch *preserveMode {
	case "auto", "manual":
		// Valid.
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid --preserve mode %q (must be auto or manual)\n", *preserveMode)
		os.Exit(1)
	}

	if err := setup.Run(dir, os.Stderr, *preserveMode); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	printUpdateNotice()
}

func runVersion() {
	fmt.Fprintf(os.Stderr, "pk %s\n", version.Version())
	printUpdateNotice()
}

func printUpdateNotice() {
	if latest, available := update.Check(update.DefaultConfig(version.Version())); available {
		fmt.Fprintf(os.Stderr, "%s\n", update.FormatNotice(latest, version.Version()))
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "pk - Plan-driven development toolkit for Claude Code")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Hook commands (called by Claude Code, not directly):")
	fmt.Fprintln(os.Stderr, "  pk preserve [--notify]              Preserve approved plan (PostToolUse hook)")
	fmt.Fprintln(os.Stderr, "  pk protect                          Block edits to docs/plans/ (PreToolUse hook)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "User commands:")
	fmt.Fprintln(os.Stderr, "  pk changelog [--bump major|minor|patch] [--dry-run]")
	fmt.Fprintln(os.Stderr, "                                      Generate changelog, commit, and tag release")
	fmt.Fprintln(os.Stderr, "  pk release [--dry-run] [--branch main]")
	fmt.Fprintln(os.Stderr, "                                      Validate and push release to origin")
	fmt.Fprintln(os.Stderr, "  pk setup [--project-dir <dir>] [--preserve auto|manual]")
	fmt.Fprintln(os.Stderr, "                                      Configure project hooks and skills")
	fmt.Fprintln(os.Stderr, "  pk version                          Print version and check for updates")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Hook commands read JSON from stdin and write JSON to stdout.")
	fmt.Fprintln(os.Stderr, "They are designed to be called by Claude Code, not directly.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "To remove hooks, delete the \"hooks\" key from .claude/settings.json.")
}
