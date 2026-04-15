// pk - Plan-driven development toolkit for Claude Code.
//
// Provides plan preservation and protection for Claude Code projects.
// Designed to be invoked by Claude Code hooks configured in .claude/settings.json.
//
// Commands:
//
//	pk changelog   Generate CHANGELOG.md from conventional commits, commit, and tag version
//	pk guard       PreToolUse hook: block git mutations on protected branches
//	pk pin         Update pinned version in .claude/install-pk.sh
//	pk preserve    PostToolUse hook: preserve approved plans in docs/plans/
//	pk protect     PreToolUse hook: block edits to docs/plans/
//	pk release     Merge to release branch, validate, and push
//	pk setup       Configure a project's .claude/settings.json
//	pk status      Report plankit configuration state of a project
//	pk teardown    Remove plankit hooks, skills, and rules from a project
//	pk version     Print version (--verbose for build details)
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/markwharton/plankit/internal/changelog"
	"github.com/markwharton/plankit/internal/guard"
	"github.com/markwharton/plankit/internal/preserve"
	"github.com/markwharton/plankit/internal/protect"
	"github.com/markwharton/plankit/internal/release"
	"github.com/markwharton/plankit/internal/setup"
	"github.com/markwharton/plankit/internal/status"
	"github.com/markwharton/plankit/internal/teardown"
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
	case "guard":
		runGuard(os.Args[2:])
	case "protect":
		runProtect(os.Args[2:])
	case "setup":
		runSetup(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "teardown":
		runTeardown(os.Args[2:])
	case "pin":
		runPin(os.Args[2:])
	case "version", "--version", "-v":
		runVersion(os.Args[2:])
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runProtect(args []string) {
	fs := flag.NewFlagSet("protect", flag.ExitOnError)
	fs.Parse(args)

	os.Exit(protect.Run(protect.DefaultConfig()))
}

func runGuard(args []string) {
	fs := flag.NewFlagSet("guard", flag.ExitOnError)
	ask := fs.Bool("ask", false, "Prompt user instead of blocking")
	fs.Parse(args)

	cfg := guard.DefaultConfig()
	cfg.Ask = *ask
	os.Exit(guard.Run(cfg))
}

func runChangelog(args []string) {
	fs := flag.NewFlagSet("changelog", flag.ExitOnError)
	bump := fs.String("bump", "", "Override version bump: major, minor, or patch")
	dryRun := fs.Bool("dry-run", false, "Preview without writing or committing")
	undo := fs.Bool("undo", false, "Unwind the last pk changelog commit (must be unpushed)")
	exclude := fs.String("exclude", "", "Comma-separated commit SHAs to drop from the section (as they appear in CHANGELOG.md parentheses)")
	fs.Parse(args)

	cfg := changelog.DefaultConfig()
	cfg.Bump = *bump
	cfg.DryRun = *dryRun
	if *exclude != "" {
		cfg.Exclude = strings.Split(*exclude, ",")
	}

	if *undo {
		os.Exit(changelog.Undo(cfg))
	}
	os.Exit(changelog.Run(cfg))
}

func runPreserve(args []string) {
	fs := flag.NewFlagSet("preserve", flag.ExitOnError)
	notify := fs.Bool("notify", false, "Notify only, do not preserve")
	dryRun := fs.Bool("dry-run", false, "Preview without writing, committing, or pushing")
	push := fs.Bool("push", false, "Push to origin after committing")
	fs.Parse(args)

	cfg := preserve.DefaultConfig()
	cfg.Notify = *notify
	cfg.DryRun = *dryRun
	cfg.Push = *push
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
	dryRun := fs.Bool("dry-run", false, "Validate without merging or pushing")
	fs.Parse(args)

	cfg := release.DefaultConfig()
	cfg.DryRun = *dryRun

	os.Exit(release.Run(cfg))
}

func runSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	projectDir := fs.String("project-dir", ".", "Project directory (default: current directory)")
	preserveMode := fs.String("preserve", "manual", "Plan preservation mode: manual or auto")
	guardMode := fs.String("guard", "block", "Guard mode: block or ask")
	force := fs.Bool("force", false, "Overwrite all managed files regardless of modifications")
	allowNonGit := fs.Bool("allow-non-git", false, "Proceed even if the project directory is not a git repository")
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

	// Validate guard mode.
	switch *guardMode {
	case "block", "ask":
		// Valid.
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid --guard mode %q (must be block or ask)\n", *guardMode)
		os.Exit(1)
	}

	cfg := setup.DefaultConfig()
	cfg.ProjectDir = dir
	cfg.PreserveMode = *preserveMode
	cfg.GuardMode = *guardMode
	cfg.Force = *force
	cfg.AllowNonGit = *allowNonGit
	cfg.Version = version.Version()
	if err := setup.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	printUpdateNotice()
}

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	projectDir := fs.String("project-dir", ".", "Project directory (default: current directory)")
	brief := fs.Bool("brief", false, "One-line summary (useful for scripting)")
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

	cfg := status.DefaultConfig()
	cfg.ProjectDir = dir
	cfg.Brief = *brief
	configured, err := status.Run(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if !configured {
		os.Exit(1)
	}
}

func runTeardown(args []string) {
	fs := flag.NewFlagSet("teardown", flag.ExitOnError)
	projectDir := fs.String("project-dir", ".", "Project directory (default: current directory)")
	confirm := fs.Bool("confirm", false, "Actually remove (default: preview only)")
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

	cfg := teardown.DefaultConfig()
	cfg.ProjectDir = dir
	cfg.Confirm = *confirm
	if err := teardown.Run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func runPin(args []string) {
	fs := flag.NewFlagSet("pin", flag.ExitOnError)
	file := fs.String("file", "", "File containing the version pin (required)")
	fs.Parse(args)
	if *file == "" || fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: pk pin --file <path> <version>")
		os.Exit(1)
	}
	if _, ok := version.ParseSemver(fs.Arg(0)); !ok {
		fmt.Fprintf(os.Stderr, "Error: %q is not valid semver\n", fs.Arg(0))
		os.Exit(1)
	}
	updated, err := setup.PinVersion(*file, fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if updated {
		fmt.Fprintf(os.Stderr, "Pinned %s to %s\n", *file, fs.Arg(0))
	}
}

func runVersion(args []string) {
	fs := flag.NewFlagSet("version", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "Show build date and Go version")
	fs.Parse(args)

	fmt.Fprintf(os.Stderr, "pk %s\n", version.Version())
	if *verbose {
		if info := version.VerboseInfo(); info != "" {
			fmt.Fprintln(os.Stderr, info)
		}
	}

	if scriptVer, found := setup.ScriptVersion(".claude/install-pk.sh"); found {
		running := strings.TrimPrefix(version.Version(), "v")
		pinned := strings.TrimPrefix(scriptVer, "v")
		if running != "dev" && pinned != running {
			fmt.Fprintf(os.Stderr, "Note: .claude/install-pk.sh pins %s but you're running %s — re-run 'pk setup' to update\n", scriptVer, running)
		}
	}

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
	fmt.Fprintln(os.Stderr, "  pk guard [--ask]                    Block git mutations on protected branches (PreToolUse hook)")
	fmt.Fprintln(os.Stderr, "  pk preserve [--dry-run] [--push] [--notify]")
	fmt.Fprintln(os.Stderr, "                                      Preserve approved plan (PostToolUse hook)")
	fmt.Fprintln(os.Stderr, "  pk protect                          Block edits to docs/plans/ (PreToolUse hook)")
	fmt.Fprintln(os.Stderr, "  pk pin --file <path> <version>      Update pinned version in a script file (preCommit hook)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "User commands:")
	fmt.Fprintln(os.Stderr, "  pk changelog [--bump major|minor|patch] [--dry-run] [--undo] [--exclude <sha>,<sha>]")
	fmt.Fprintln(os.Stderr, "                                      Generate changelog, commit, and tag version")
	fmt.Fprintln(os.Stderr, "  pk release [--dry-run]              Read Release-Tag trailer, tag, merge, and push")
	fmt.Fprintln(os.Stderr, "  pk setup [--force] [--allow-non-git] [--project-dir <dir>] [--guard block|ask] [--preserve auto|manual]")
	fmt.Fprintln(os.Stderr, "                                      Configure project hooks and skills")
	fmt.Fprintln(os.Stderr, "  pk status [--brief] [--project-dir <dir>]")
	fmt.Fprintln(os.Stderr, "                                      Report plankit configuration state")
	fmt.Fprintln(os.Stderr, "  pk teardown [--confirm] [--project-dir <dir>]")
	fmt.Fprintln(os.Stderr, "                                      Remove plankit hooks, skills, and rules")
	fmt.Fprintln(os.Stderr, "  pk version [--verbose]              Print version and check for updates")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Hook commands read JSON from stdin and write JSON to stdout.")
	fmt.Fprintln(os.Stderr, "They are designed to be called by Claude Code, not directly.")
}
