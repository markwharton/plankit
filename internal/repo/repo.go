// Package repo holds the repo-model commands: init writes the policy the
// model reads, status reports what the model sees. Together they prove
// internal/config and internal/git.
package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/brief"
	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/msg"
)

// InitCmd configures a repository for plankit.
var InitCmd = &cli.Command{
	Name:    "init",
	Summary: "Configure a repository for plankit",
	Flags: []cli.FlagSpec{
		{Name: "branch", Type: cli.StringFlag, Default: "develop", Usage: "Working branch to create when the release branch is the only one"},
		{Name: "dry-run", Type: cli.BoolFlag, Usage: "Preview without making any changes"},
		cli.FormatFlag,
		{Name: "no-baseline", Type: cli.BoolFlag, Usage: "Skip creating the v0.0.0 baseline tag"},
		{Name: "no-commit", Type: cli.BoolFlag, Usage: "Leave .pk.json uncommitted"},
		{Name: "push", Type: cli.BoolFlag, Usage: "Push the release branch, the tag, and the working branch to origin"},
		{Name: "release", Type: cli.StringFlag, Usage: "Release branch to guard (default: the branch currently checked out)"},
	},
	Run: runInit,
}

// configureSubject is the message of the commit init makes.
const configureSubject = "chore: configure plankit"

func runInit(ctx *cli.Context) error {
	root, ok := git.FindRoot(ctx.ProjectDir)
	if !ok {
		return cli.WithHint(
			cli.Statef("not a git repository: %s", ctx.ProjectDir),
			"run git init first; plankit configures existing repositories")
	}
	if _, err := os.Stat(config.Path(root)); err == nil {
		// A v0.x repository is configured too, so it lands here when
		// someone re-runs setup out of habit. It has a worse problem
		// than a second .pk.json, so the hint names that one instead.
		hint := fmt.Sprintf("edit %s directly, or run pk status to see the current policy", config.FileName)
		if legacyWiring(root) {
			hint = "v0.x files are still here; pk help overview says what to remove"
		}
		return cli.WithHint(cli.Statef("already configured: %s exists", config.FileName), "%s", hint)
	}

	current, err := git.CurrentBranch(root)
	if err != nil {
		return cli.Statef("cannot determine the current branch: %v", err)
	}
	release := ctx.String("release")
	if release == "" {
		release = current
	}
	working := ctx.String("branch")
	noCommit := ctx.Bool("no-commit")
	push := ctx.Bool("push")
	if working == release {
		return cli.Usagef("--branch %q is the release branch; the working branch must be another", working)
	}
	if push && noCommit {
		return cli.Usagef("--push needs the commit that --no-commit skips")
	}
	if push && !git.HasRemote(root, "origin") {
		return cli.WithHint(cli.Statef("no origin remote to push to"), "add one with git remote add origin <url>, or run without --push")
	}

	// Each step's condition is judged as if the steps before it ran, so
	// a dry run on an empty repository previews the whole bootstrap.
	hadCommits := git.HasCommits(root)
	willHaveCommit := hadCommits || !noCommit
	baseline := ""
	if !ctx.Bool("no-baseline") && git.LatestTag(root) == "" && willHaveCommit {
		baseline = "v0.0.0"
	}
	branch := ""
	if current == release && willHaveCommit && !git.HasOtherLocalBranch(root, release) {
		branch = working
	}

	dryRun := ctx.Bool("dry-run")
	created := []string{}
	do := func(label string, fn func() error) error {
		created = append(created, label)
		if dryRun {
			return nil
		}
		return fn()
	}

	cfg := config.Default(release)
	if err := do(config.FileName, func() error { return config.Write(root, cfg) }); err != nil {
		return err
	}
	// The plans directory is not created here: preserve creates it on
	// first use, so a repository that never preserves a plan never gains
	// the directory.
	if !noCommit {
		if err := do("commit", func() error { return git.CommitPaths(root, configureSubject, config.FileName) }); err != nil {
			return err
		}
	}
	if baseline != "" {
		if err := do("tag "+baseline, func() error { return git.CreateTag(root, baseline) }); err != nil {
			return err
		}
	}
	if branch != "" {
		if err := do("branch "+branch, func() error { return git.CreateBranch(root, branch) }); err != nil {
			return err
		}
	}
	pushed := []string{}
	if push {
		pushed = append(pushed, release)
		if baseline != "" {
			pushed = append(pushed, baseline)
		}
		if branch != "" {
			pushed = append(pushed, branch)
		}
		if err := do("push origin "+strings.Join(pushed, " "), func() error { return git.PushRefs(root, pushed...) }); err != nil {
			return err
		}
	}

	if ctx.Format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(map[string]any{
			"root":      root,
			"release":   release,
			"created":   created,
			"committed": !noCommit,
			"baseline":  baseline,
			"branch":    branch,
			"pushed":    pushed,
			"dryRun":    dryRun,
		})
	}
	verb := "created"
	if dryRun {
		verb = "would create"
	}
	fmt.Fprintf(ctx.Stderr, "plankit configured in %s (%s: %s)\n", root, verb, strings.Join(created, ", "))
	if ctx.Quiet {
		return nil
	}
	if baseline == "" && git.LatestTag(root) == "" && !willHaveCommit {
		msg.Notef(ctx.Stderr, "no baseline tag: the repository has no commits yet")
	}
	if noCommit {
		hint := fmt.Sprintf("commit %s yourself: guard blocks a session from doing it on %s", config.FileName, release)
		if !hadCommits {
			hint += fmt.Sprintf(", then git tag v0.0.0 && git switch -c %s", working)
		}
		msg.Hintf(ctx.Stderr, "%s", hint)
	}
	// The session that configured the repository was not briefed at its
	// start, so init hands it the brief here.
	fmt.Fprintf(ctx.Stderr, "\n%s", brief.Text(cfg))
	return nil
}

// StatusCmd reports configuration and repository state.
// legacyWiring reports the files a v0.x pk setup copied into a
// repository, which the plugin now ships itself. Each path is pk's own
// and cannot be mistaken for a project's: while they are present, the
// copied hooks and the plugin's both fire.
func legacyWiring(root string) bool {
	for _, p := range []string{
		".claude/install-pk.sh",
		".claude/rules/plankit",
		".claude/skills/pk-configure",
	} {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			return true
		}
	}
	return false
}

var StatusCmd = &cli.Command{
	Name:    "status",
	Summary: "Report plankit configuration and repository state",
	Flags:   []cli.FlagSpec{cli.FormatFlag},
	Run:     runStatus,
}

// state is the status report; one shape serves text and json.
type state struct {
	Root       string   `json:"root"`
	Configured bool     `json:"configured"`
	Branch     string   `json:"branch,omitempty"`
	Clean      bool     `json:"clean"`
	Preserve   string   `json:"preserve,omitempty"`
	PlansDir   string   `json:"plansDir,omitempty"`
	GuardMode  string   `json:"guardMode,omitempty"`
	GuardPush  string   `json:"guardPush,omitempty"`
	GuardBreak string   `json:"guardBreaking,omitempty"`
	Guarded    []string `json:"guardedBranches,omitempty"`
	Release    string   `json:"releaseBranch,omitempty"`
	Plans      int      `json:"plans"`
	LatestTag  string   `json:"latestTag,omitempty"`
}

func runStatus(ctx *cli.Context) error {
	root, ok := git.FindRoot(ctx.ProjectDir)
	if !ok {
		return cli.Statef("not a git repository: %s", ctx.ProjectDir)
	}

	s := state{Root: root}
	cfg, err := config.Load(root)
	switch {
	case err == nil:
		s.Configured = true
		s.Preserve = cfg.Preserve.ResolvedMode()
		s.PlansDir = cfg.Preserve.ResolvedDir()
		s.Plans = countPlans(cfg.Preserve.DirPath(root))
		s.GuardMode = cfg.Guard.ResolvedMode()
		s.GuardPush = cfg.Guard.ResolvedPush()
		s.GuardBreak = cfg.Guard.ResolvedBreaking()
		s.Guarded = cfg.Guard.Branches
		s.Release = cfg.Release.Branch
	case err == config.ErrNotConfigured:
		// reported below; carries exit 2 so scripts can probe
	default:
		return cli.Statef("%v", err)
	}

	if branch, err := git.CurrentBranch(root); err == nil {
		s.Branch = branch
	}
	s.Clean, _ = git.Clean(root)
	s.LatestTag = git.LatestTag(root)

	if ctx.Format == "json" {
		if err := json.NewEncoder(ctx.Stdout).Encode(s); err != nil {
			return err
		}
		if !s.Configured {
			return silentState()
		}
		return nil
	}

	if !s.Configured {
		fmt.Fprintf(ctx.Stdout, "plankit: not configured in %s\n", root)
		return cli.WithHint(silentState(), "run pk init to write %s", config.FileName)
	}

	tree := "clean"
	if !s.Clean {
		tree = "dirty"
	}
	fmt.Fprintf(ctx.Stdout, "plankit status\n")
	line := func(k, v string) { fmt.Fprintf(ctx.Stdout, "  %-10s %s\n", k+":", v) }
	line("project", s.Root)
	line("branch", fmt.Sprintf("%s (%s)", s.Branch, tree))
	line("preserve", s.Preserve)
	line("guard", fmt.Sprintf("%s (push: %s, breaking: %s) on %s", s.GuardMode, s.GuardPush, s.GuardBreak, strings.Join(s.Guarded, ", ")))
	line("protect", s.PlansDir+"/ immutable")
	line("release", s.Release)
	line("plans", fmt.Sprintf("%d preserved", s.Plans))
	if s.LatestTag != "" {
		line("tag", s.LatestTag)
	}
	if ctx.Quiet {
		return nil
	}
	// Readiness: a repository that stopped partway through the bootstrap
	// names its own next step.
	if s.LatestTag == "" && git.HasCommits(root) {
		msg.Notef(ctx.Stderr, "no baseline tag: git tag v0.0.0 gives the release machinery a starting point")
	}
	if s.Release != "" && !git.HasOtherLocalBranch(root, s.Release) {
		msg.Notef(ctx.Stderr, "no working branch besides %s: to start one, git switch -c develop", s.Release)
	}
	return nil
}

// silentState is exit-code 2 with no message: status already printed the
// report, the code is the machine-readable part.
func silentState() error { return cli.Silent(cli.ExitState) }

func countPlans(plansDir string) int {
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			n++
		}
	}
	return n
}
