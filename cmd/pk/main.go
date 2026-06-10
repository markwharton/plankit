// pk - Plan-driven development toolkit for Claude Code.
//
// Provides plan preservation and protection for Claude Code projects.
// Designed to be invoked by Claude Code hooks configured in .claude/settings.json.
//
// Commands:
//
//	pk changelog   Generate CHANGELOG.md from conventional commits, commit, and tag version
//	pk guard       PreToolUse hook: block git mutations on protected branches
//	pk pin         Update pinned version in a file
//	pk preserve    PostToolUse hook: preserve approved plans in docs/plans/
//	pk protect     PreToolUse hook: block edits to docs/plans/
//	pk release     Merge to release branch, validate, and push
//	pk rules       Report the always-on context footprint of .claude/rules/ and CLAUDE.md
//	pk setup       Configure a project's .claude/settings.json
//	pk status      Report plankit configuration state of a project
//	pk teardown    Remove plankit hooks, skills, and rules from a project
//	pk version     Print version (--verbose for build details)
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/changelog"
	pkgit "github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/guard"
	"github.com/markwharton/plankit/internal/preserve"
	"github.com/markwharton/plankit/internal/protect"
	"github.com/markwharton/plankit/internal/release"
	"github.com/markwharton/plankit/internal/rules"
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
	case "rules":
		runRules(os.Args[2:])
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
	// --ask and --push-guard are DEPRECATED: modes now live in .pk.json
	// (guard.mode, guard.push). They are honored as overrides only when an old
	// hook still passes them, so existing installs keep working until re-setup.
	ask := fs.Bool("ask", false, "[deprecated] force ask mode; set guard.mode in .pk.json")
	pushGuard := fs.String("push-guard", "", "[deprecated] force push policy; set guard.push in .pk.json")
	fs.Parse(args)

	cfg := guard.DefaultConfig()
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "ask":
			cfg.Ask = *ask
		case "push-guard":
			validateMode(*pushGuard, "--push-guard", "block", "ask", "off")
			cfg.PushGuard = *pushGuard
		}
	})
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
	cfg.Dir = mustGitRoot()
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
	notify := fs.Bool("notify", false, "[deprecated] force manual (notify) mode; set preserve.mode in .pk.json")
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
	cfg.Dir = mustGitRoot()
	cfg.DryRun = *dryRun

	os.Exit(release.Run(cfg))
}

func runRules(args []string) {
	fs := flag.NewFlagSet("rules", flag.ExitOnError)
	projectDir := fs.String("project-dir", ".", "Project directory (default: current directory)")
	lint := fs.Bool("lint", false, "Scan rules for hidden/Trojan-source characters instead of reporting the footprint")
	strict := fs.Bool("strict", false, "With --lint: also run plankit house-style checks (requires --lint)")
	fs.Parse(args)

	if *strict && !*lint {
		fmt.Fprintln(os.Stderr, "Error: --strict requires --lint")
		os.Exit(1)
	}

	cfg := rules.DefaultConfig()
	cfg.ProjectDir = resolveProjectDir(*projectDir)
	cfg.Version = version.Version()
	cfg.Lint = *lint
	cfg.Strict = *strict
	os.Exit(rules.Run(cfg))
}

func runSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	projectDir := fs.String("project-dir", ".", "Project directory (default: current directory)")
	preserveMode := fs.String("preserve", "", "Plan preservation mode: auto, manual, or off (default: keep existing, else manual)")
	guardMode := fs.String("guard", "", "Guard mode: block, ask, or off (default: keep existing, else block)")
	pushGuardMode := fs.String("push-guard", "", "Push-guard mode: block, ask, or off (default: keep existing, else block)")
	force := fs.Bool("force", false, "Overwrite all managed files regardless of modifications")
	allowNonGit := fs.Bool("allow-non-git", false, "Proceed even if the project directory is not a git repository")
	baseline := fs.Bool("baseline", false, "Anchor pk changelog by creating a v0.0.0 tag if none exists")
	baselineAt := fs.String("at", "", "Ref to tag as v0.0.0 (requires --baseline; defaults to HEAD)")
	push := fs.Bool("push", false, "Push the baseline tag to origin (requires --baseline)")
	fs.Parse(args)

	dir := resolveProjectDir(*projectDir)

	// Validate explicitly-provided modes. An empty value means "not specified" —
	// pk setup resolves it (existing .pk.json > migrated from old hooks > default).
	if *preserveMode != "" {
		validateMode(*preserveMode, "--preserve", "auto", "manual", "off")
	}
	if *guardMode != "" {
		validateMode(*guardMode, "--guard", "block", "ask", "off")
	}
	if *pushGuardMode != "" {
		validateMode(*pushGuardMode, "--push-guard", "block", "ask", "off")
	}

	// --at and --push are modifiers of --baseline; reject on their own.
	if !*baseline {
		if *baselineAt != "" {
			fmt.Fprintln(os.Stderr, "Error: --at requires --baseline")
			os.Exit(1)
		}
		if *push {
			fmt.Fprintln(os.Stderr, "Error: --push requires --baseline")
			os.Exit(1)
		}
	}

	cfg := setup.DefaultConfig()
	cfg.ProjectDir = dir
	cfg.PreserveMode = *preserveMode
	cfg.GuardMode = *guardMode
	cfg.PushGuardMode = *pushGuardMode
	cfg.Force = *force
	cfg.AllowNonGit = *allowNonGit
	cfg.Version = version.Version()
	cfg.Baseline = *baseline
	cfg.BaselineAt = *baselineAt
	cfg.Push = *push
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

	dir := resolveProjectDir(*projectDir)

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

	dir := resolveProjectDir(*projectDir)

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
	name := fs.String("name", "", "Identifier to match (e.g., version, __version__)")
	fs.Parse(args)
	if *file == "" || fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: pk pin --file <path> [--name <identifier>] <version>")
		os.Exit(1)
	}
	if _, ok := version.ParseSemver(fs.Arg(0)); !ok {
		fmt.Fprintf(os.Stderr, "Error: %q is not valid semver\n", fs.Arg(0))
		os.Exit(1)
	}
	var updated bool
	var err error
	if *name != "" {
		updated, err = setup.PinVersionNamed(os.ReadFile, os.WriteFile, *file, *name, fs.Arg(0))
	} else {
		updated, err = setup.PinVersion(os.ReadFile, os.WriteFile, *file, fs.Arg(0))
	}
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

	if scriptVer, found := setup.ScriptVersion(os.ReadFile, ".claude/install-pk.sh"); found {
		running := version.Version()
		pinned := strings.TrimPrefix(scriptVer, "v")
		if !version.IsDevBuild(running) && pinned != running {
			pinnedSemver, pok := version.ParseSemver(pinned)
			runningSemver, rok := version.ParseSemver(running)
			if pok && rok && pinnedSemver.Compare(runningSemver) > 0 {
				fmt.Fprintf(os.Stderr, "Note: .claude/install-pk.sh pins %s but you're running %s — run 'go install github.com/markwharton/plankit/cmd/pk@latest' to update\n", scriptVer, running)
			} else {
				fmt.Fprintf(os.Stderr, "Note: .claude/install-pk.sh pins %s but you're running %s — re-run 'pk setup' to update\n", scriptVer, running)
			}
		}
	}

	printUpdateNotice()
}

func mustGitRoot() string {
	root, ok := pkgit.RepoRoot(os.Stat, resolveDir("."))
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: not a git repository")
		os.Exit(1)
	}
	return root
}

func resolveDir(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return abs
}

// validateMode exits with an "Error: invalid <flag> mode ... (must be a, b, or c)"
// message if value is not one of valid.
func validateMode(value, flagName string, valid ...string) {
	for _, v := range valid {
		if value == v {
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Error: invalid %s mode %q (must be %s)\n", flagName, value, orList(valid))
	os.Exit(1)
}

// orList renders a slice as an English "a, b, or c" list (Oxford comma).
func orList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " or " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + ", or " + items[len(items)-1]
	}
}

func resolveProjectDir(dir string) string {
	abs := resolveDir(dir)
	if root, ok := pkgit.RepoRoot(os.Stat, abs); ok {
		return root
	}
	return abs
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
	fmt.Fprintln(os.Stderr, "  pk guard [--ask] [--push-guard block|ask|off]")
	fmt.Fprintln(os.Stderr, "                                      Block git mutations on protected branches; guard pushes (PreToolUse hook)")
	fmt.Fprintln(os.Stderr, "  pk preserve [--dry-run] [--push] [--notify]")
	fmt.Fprintln(os.Stderr, "                                      Preserve approved plan (PostToolUse hook)")
	fmt.Fprintln(os.Stderr, "  pk protect                          Block edits to docs/plans/ (PreToolUse hook)")
	fmt.Fprintln(os.Stderr, "  pk pin --file <path> [--name <id>] <version>")
	fmt.Fprintln(os.Stderr, "                                      Update pinned version in a file (preCommit hook)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "User commands:")
	fmt.Fprintln(os.Stderr, "  pk changelog [--bump major|minor|patch] [--dry-run] [--undo] [--exclude <sha>,<sha>]")
	fmt.Fprintln(os.Stderr, "                                      Generate changelog, commit, and tag version")
	fmt.Fprintln(os.Stderr, "  pk release [--dry-run]              Read Release-Tag trailer, tag, merge, and push")
	fmt.Fprintln(os.Stderr, "  pk rules [--lint [--strict]] [--project-dir <dir>]")
	fmt.Fprintln(os.Stderr, "                                      Report .claude/rules/ + CLAUDE.md context footprint; --lint scans for hidden chars")
	fmt.Fprintln(os.Stderr, "  pk setup [--force] [--allow-non-git] [--project-dir <dir>] [--guard block|ask] [--preserve auto|manual]")
	fmt.Fprintln(os.Stderr, "           [--push-guard block|ask|off] [--baseline [--at <ref>] [--push]]")
	fmt.Fprintln(os.Stderr, "                                      Configure project hooks and skills; optionally anchor pk changelog")
	fmt.Fprintln(os.Stderr, "  pk status [--brief] [--project-dir <dir>]")
	fmt.Fprintln(os.Stderr, "                                      Report plankit configuration state")
	fmt.Fprintln(os.Stderr, "  pk teardown [--confirm] [--project-dir <dir>]")
	fmt.Fprintln(os.Stderr, "                                      Remove plankit hooks, skills, and rules")
	fmt.Fprintln(os.Stderr, "  pk version [--verbose]              Print version and check for updates")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Hook commands read JSON from stdin and write JSON to stdout.")
	fmt.Fprintln(os.Stderr, "They are designed to be called by Claude Code, not directly.")
}
