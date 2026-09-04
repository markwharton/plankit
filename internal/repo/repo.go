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

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/paths"
)

// PlansDir is where preserved plans live, relative to the repo root.
const PlansDir = paths.PlansRel

// InitCmd configures a repository for plankit.
var InitCmd = &cli.Command{
	Name:    "init",
	Summary: "Configure this repository: write .pk.json, create docs/plans, baseline a tag",
	Flags: []cli.FlagSpec{
		{Name: "release", Type: cli.StringFlag, Usage: "Release branch to guard (default: the branch currently checked out)"},
		{Name: "no-baseline", Type: cli.BoolFlag, Usage: "Skip creating the v0.0.0 baseline tag"},
		{Name: "dry-run", Type: cli.BoolFlag, Usage: "Preview without making any changes"},
	},
	Run: runInit,
}

func runInit(ctx *cli.Context) error {
	root, ok := git.FindRoot(ctx.ProjectDir)
	if !ok {
		return cli.WithHint(
			cli.Statef("not a git repository: %s", ctx.ProjectDir),
			"run git init first; plankit configures existing repositories")
	}
	if _, err := os.Stat(config.Path(root)); err == nil {
		return cli.WithHint(
			cli.Statef("already configured: %s exists", config.FileName),
			"edit %s directly, or run pk status to see the current policy", config.FileName)
	}

	release := ctx.String("release")
	if release == "" {
		branch, err := git.CurrentBranch(root)
		if err != nil {
			return cli.Statef("cannot determine the current branch: %v", err)
		}
		release = branch
	}

	// Baseline: release machinery diffs from the last tag, so a repo
	// without one gets v0.0.0 at HEAD. Needs a commit to point at.
	baseline := ""
	if !ctx.Bool("no-baseline") && git.LatestTag(root) == "" && git.HasCommits(root) {
		baseline = "v0.0.0"
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
	plans := paths.Plans(root)
	if _, err := os.Stat(plans); os.IsNotExist(err) {
		if err := do(PlansDir+"/", func() error {
			if err := os.MkdirAll(plans, 0o755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(plans, ".gitkeep"), nil, 0o644)
		}); err != nil {
			return err
		}
	}
	if baseline != "" {
		if err := do("tag "+baseline, func() error { return git.CreateTag(root, baseline) }); err != nil {
			return err
		}
	}

	if ctx.Format == "json" {
		return json.NewEncoder(ctx.Stdout).Encode(map[string]any{
			"root":     root,
			"release":  release,
			"created":  created,
			"baseline": baseline,
			"dryRun":   dryRun,
		})
	}
	verb := "created"
	if dryRun {
		verb = "would create"
	}
	fmt.Fprintf(ctx.Stdout, "plankit configured in %s (%s: %s)\n", root, verb, strings.Join(created, ", "))
	if baseline == "" && git.LatestTag(root) == "" && !ctx.Quiet {
		msg.Notef(ctx.Stdout, "no baseline tag: the repository has no commits yet")
	}
	if !ctx.Quiet {
		msg.Hintf(ctx.Stdout, "commit %s and %s; run pk status to review the policy", config.FileName, PlansDir)
	}
	return nil
}

// StatusCmd reports configuration and repository state.
var StatusCmd = &cli.Command{
	Name:    "status",
	Summary: "Report plankit configuration and repository state",
	Run:     runStatus,
}

// state is the status report; one shape serves text and json.
type state struct {
	Root       string   `json:"root"`
	Configured bool     `json:"configured"`
	Branch     string   `json:"branch,omitempty"`
	Clean      bool     `json:"clean"`
	Preserve   string   `json:"preserve,omitempty"`
	GuardMode  string   `json:"guardMode,omitempty"`
	GuardPush  string   `json:"guardPush,omitempty"`
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
		s.GuardMode = cfg.Guard.ResolvedMode()
		s.GuardPush = cfg.Guard.ResolvedPush()
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
	s.Plans = countPlans(root)

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
	line("guard", fmt.Sprintf("%s (push: %s) on %s", s.GuardMode, s.GuardPush, strings.Join(s.Guarded, ", ")))
	line("protect", PlansDir+"/ immutable")
	line("release", s.Release)
	line("plans", fmt.Sprintf("%d preserved", s.Plans))
	if s.LatestTag != "" {
		line("tag", s.LatestTag)
	}
	return nil
}

// silentState is exit-code 2 with no message: status already printed the
// report, the code is the machine-readable part.
func silentState() error { return cli.Silent(cli.ExitState) }

func countPlans(root string) int {
	entries, err := os.ReadDir(paths.Plans(root))
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
