// Package scaffold implements pk init: making a fresh repository
// plankit-shaped in one command.
//
// It writes the branch topology into .pk.json, runs the pk setup path to
// install managed files and the v0.0.0 baseline tag, drops the GitHub
// branch-protection ruleset, and creates the source branch. With NoSetup the
// managed-file install is skipped and only the repository shape is written,
// for projects that want release management without the .claude footprint.
// The package is named scaffold rather than init because init is a Go
// keyword.
//
// Everything here is git-only, so it works on any host and offline. Applying
// the branch-protection ruleset needs an authenticated GitHub call, so pk
// writes the ruleset into the repository and prints how to apply it rather
// than reaching for gh itself.
package scaffold

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/paths"
	"github.com/markwharton/plankit/internal/readiness"
	"github.com/markwharton/plankit/internal/setup"
)

// DefaultSourceBranch is the working branch pk init creates when --source is
// not given. All work happens here; the release branch advances only via
// pk release.
const DefaultSourceBranch = "develop"

// SetupCommitMessage is the commit pk init makes for the files it writes. It
// is kept separate from the project's own first commit so that v0.0.0 anchors
// where the project's history actually begins.
const SetupCommitMessage = "chore: pk setup"

// MinimalCommitMessage is the commit pk init makes with --no-setup, where the
// pk setup step never ran and "chore: pk setup" would misdescribe the commit.
const MinimalCommitMessage = "chore: pk init"

// Config holds the injectable dependencies and flags for pk init.
type Config struct {
	ProjectDir    string
	SourceBranch  string
	ReleaseBranch string
	NoSetup       bool
	Push          bool
	DryRun        bool
	Version       string

	Stderr    io.Writer
	GitExec   func(dir string, args ...string) (string, error)
	ReadFile  func(string) ([]byte, error)
	WriteFile func(string, []byte, os.FileMode) error
	Stat      func(string) (os.FileInfo, error)
	MkdirAll  func(string, os.FileMode) error
	ReadDir   func(string) ([]os.DirEntry, error)
	Remove    func(string) error
	Rename    func(string, string) error
	LookPath  func(string) (string, error)
}

// DefaultConfig returns a Config wired to real OS resources.
func DefaultConfig() Config {
	s := setup.DefaultConfig()
	return Config{
		Stderr:    os.Stderr,
		GitExec:   git.Exec,
		ReadFile:  s.ReadFile,
		WriteFile: s.WriteFile,
		Stat:      s.Stat,
		MkdirAll:  s.MkdirAll,
		ReadDir:   s.ReadDir,
		Remove:    s.Remove,
		Rename:    s.Rename,
		LookPath:  s.LookPath,
	}
}

// Run makes the repository plankit-shaped.
//
// Two paths, decided by whether the working branch exists. It is created
// last, so its presence means a run completed; from then on pk init writes
// and commits nothing, because a commit on the release branch after the split
// is one the working branch lacks and pk release cannot fast-forward past.
// Before the split every step is a no-op when already satisfied, so a re-run
// after a partial failure completes the job rather than erroring.
func Run(cfg Config) error {
	return runShape(cfg, cfg.ProjectDir)
}

// runShape makes an existing repository plankit-shaped.
func runShape(cfg Config, dir string) error {
	projectDir, ok := git.RepoRoot(cfg.Stat, dir)
	if !ok {
		return fmt.Errorf("this is not a git repository. Run git init first")
	}

	releaseBranch, err := resolveReleaseBranch(cfg, projectDir)
	if err != nil {
		return err
	}
	sourceBranch := cfg.SourceBranch
	if sourceBranch == "" {
		sourceBranch = DefaultSourceBranch
	}
	if sourceBranch == releaseBranch {
		return fmt.Errorf("source branch %q is the release branch; they must differ", sourceBranch)
	}

	if branchExists(cfg, projectDir, sourceBranch) {
		return runShaped(cfg, projectDir, releaseBranch, sourceBranch)
	}

	if err := preflight(cfg, projectDir, releaseBranch); err != nil {
		return err
	}

	// The ruleset is only meaningful on a GitHub remote. A remote that is
	// known to be something else would get an inert file nobody can apply, so
	// say so and skip it. No remote yet is not that case: the host is unknown,
	// GitHub is where the ruleset applies, and pk init writes it once, before
	// the branch split, so a later run cannot add it without diverging the
	// branches. Write it now and let the summary say how to apply it.
	origin, hasOrigin := originURL(cfg, projectDir)
	protect := !hasOrigin || repoSlug(origin) != ""
	if !protect {
		msg.Notef(cfg.Stderr, "no GitHub remote, so the branch-protection ruleset does not apply here")
	}

	if cfg.DryRun {
		return preview(cfg, projectDir, releaseBranch, sourceBranch, protect)
	}

	// Topology first: pk setup field-merges its modes on top, so the keys land
	// in one file rather than racing each other.
	if _, err := setup.WriteTopology(setupConfig(cfg, projectDir), projectDir, releaseBranch); err != nil {
		return err
	}
	fmt.Fprintf(cfg.Stderr, "Set release branch %s and guarded it in %s\n", releaseBranch, paths.PkConfig)

	// Managed files plus the v0.0.0 baseline tag. Push stays off here: setup
	// would publish HEAD, which is still the commit before the one carrying
	// these files. Everything is pushed below, after the commit exists.
	// Embedded, so setup does not close with tips pointing back at pk setup;
	// the summary below is ours. With --no-setup only the baseline tag is
	// wanted: release management reads the topology and the tag, never the
	// managed files, so the .claude footprint stays out of the repository.
	sc := setupConfig(cfg, projectDir)
	sc.Embedded = true
	if cfg.NoSetup {
		if err := setup.RunBaseline(sc, projectDir); err != nil {
			return err
		}
	} else {
		sc.Baseline = true
		if err := setup.Run(sc); err != nil {
			return err
		}
	}

	if protect {
		if wrote, err := setup.WriteRuleset(setupConfig(cfg, projectDir), projectDir, releaseBranch); err != nil {
			return err
		} else if wrote {
			fmt.Fprintf(cfg.Stderr, "Wrote %s\n", setup.RulesetPath(releaseBranch))
		}
	}

	// Commit before branching, so the working branch carries the setup files
	// rather than forking from the bare commit v0.0.0 anchors.
	if err := commitSetup(cfg, projectDir); err != nil {
		return err
	}

	if err := createSourceBranch(cfg, projectDir, sourceBranch); err != nil {
		return err
	}

	if cfg.Push {
		if err := publish(cfg, projectDir, releaseBranch, sourceBranch); err != nil {
			return err
		}
	}

	return summarize(cfg, projectDir, releaseBranch, sourceBranch, protect)
}

// runShaped is the path for a repository whose working branch already exists.
// It writes and commits nothing. It confirms the rest of the shape is there,
// publishes with --push, and moves off the release branch; anything else is
// pk setup's job, on the working branch.
func runShaped(cfg Config, projectDir, releaseBranch, sourceBranch string) error {
	tag, err := checkShape(cfg, projectDir, releaseBranch, sourceBranch)
	if err != nil {
		return err
	}
	current, err := currentBranch(cfg, projectDir)
	if err != nil {
		return err
	}
	if err := git.CheckCleanTree(cfg.GitExec, projectDir); err != nil {
		return err
	}
	if cfg.Push {
		if _, ok := originURL(cfg, projectDir); !ok {
			return fmt.Errorf("--push needs an origin remote, and this repository has none")
		}
	}

	fmt.Fprintf(cfg.Stderr, "Already shaped: release branch %s, working branch %s, anchored at %s\n", releaseBranch, sourceBranch, tag)

	origin, hasOrigin := originURL(cfg, projectDir)
	protect := !hasOrigin || repoSlug(origin) != ""
	if protect {
		if _, err := cfg.ReadFile(filepath.Join(projectDir, filepath.FromSlash(setup.RulesetPath(releaseBranch)))); err != nil {
			// Written on the first run only; a repository shaped before it had
			// a remote may lack it, and adding it here would mean a commit.
			msg.Notef(cfg.Stderr, "no %s in this repository; pk init writes it on the first run only", setup.RulesetPath(releaseBranch))
			protect = false
		}
	}

	if cfg.DryRun {
		return previewShaped(cfg, releaseBranch, sourceBranch, current)
	}

	if cfg.Push {
		if err := publish(cfg, projectDir, releaseBranch, sourceBranch); err != nil {
			return err
		}
	}

	if current == releaseBranch {
		if err := switchTo(cfg, projectDir, sourceBranch); err != nil {
			return err
		}
		current = sourceBranch
	}

	return summarizeShaped(cfg, projectDir, releaseBranch, sourceBranch, current, protect)
}

// checkShape confirms that a repository whose working branch exists carries
// the rest of the shape: a semver anchor and release.branch in .pk.json naming
// this release branch. Returns the anchor tag. Anything less is an established
// repository that happens to have the branch, which pk init does not shape:
// running the first-run steps there would tag and commit on the release
// branch behind the working branch's back.
func checkShape(cfg Config, projectDir, releaseBranch, sourceBranch string) (string, error) {
	refuse := func(reason string) error {
		return fmt.Errorf("branch %q already exists but the repository is not plankit-shaped (%s); pk init shapes a fresh repository, so shape this one by hand", sourceBranch, reason)
	}
	conf, err := config.Load(cfg.ReadFile, filepath.Join(projectDir, paths.PkConfig))
	if err != nil {
		return "", err
	}
	if conf.Release.Branch == "" {
		return "", refuse("no release.branch in " + paths.PkConfig)
	}
	if conf.Release.Branch != releaseBranch {
		return "", refuse(fmt.Sprintf("release.branch in %s is %q, not %q", paths.PkConfig, conf.Release.Branch, releaseBranch))
	}
	tag, ok := readiness.ValidSemverTag(cfg.GitExec, projectDir)
	if !ok {
		return "", refuse("no version tag")
	}
	return tag, nil
}

// branchExists reports whether a local branch of that name exists.
func branchExists(cfg Config, projectDir, name string) bool {
	_, err := cfg.GitExec(projectDir, "rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

// originURL returns the origin remote's URL and whether there is one.
func originURL(cfg Config, projectDir string) (string, bool) {
	out, err := cfg.GitExec(projectDir, "remote", "get-url", "origin")
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(out), true
}

// resolveReleaseBranch returns the release branch by precedence: the --release
// flag, then release.branch already in .pk.json, then the branch currently
// checked out. Same shape as pk setup's mode resolution.
//
// Reading .pk.json is what stops a re-run from redefining the project. pk init
// leaves you on the working branch, so a second run would otherwise infer that
// branch as the release branch and overwrite release.branch and guard.branches
// with it, silently unguarding the real release branch.
func resolveReleaseBranch(cfg Config, projectDir string) (string, error) {
	if cfg.ReleaseBranch != "" {
		return cfg.ReleaseBranch, nil
	}
	conf, err := config.Load(cfg.ReadFile, filepath.Join(projectDir, paths.PkConfig))
	if err != nil {
		return "", err
	}
	if conf.Release.Branch != "" {
		return conf.Release.Branch, nil
	}
	return currentBranch(cfg, projectDir)
}

// currentBranch returns the checked-out branch. A detached HEAD is an error
// rather than an empty string: pk init has to know which branch it is
// anchoring, and every caller here would have to re-check anyway.
func currentBranch(cfg Config, projectDir string) (string, error) {
	out, err := cfg.GitExec(projectDir, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("failed to read the current branch: %w", err)
	}
	branch := strings.TrimSpace(out)
	if branch == "" {
		return "", fmt.Errorf("HEAD is detached; check out your release branch, or name it with --release")
	}
	return branch, nil
}

// preflight refuses on any state that would make a partial or wrong
// initialization, before anything is written.
func preflight(cfg Config, projectDir, releaseBranch string) error {
	if _, err := cfg.GitExec(projectDir, "rev-parse", "HEAD"); err != nil {
		return fmt.Errorf("this repository has no commits. Make one first, so v0.0.0 has something to anchor to")
	}
	// Read the branch we are actually on, not the one that was configured:
	// with --release the two can differ, and that is the case worth catching.
	current, err := currentBranch(cfg, projectDir)
	if err != nil {
		return err
	}
	if current != releaseBranch {
		return fmt.Errorf("you are on %q but the release branch is %q; switch to it first", current, releaseBranch)
	}
	if err := git.CheckCleanTree(cfg.GitExec, projectDir); err != nil {
		return err
	}
	if cfg.Push {
		if _, ok := originURL(cfg, projectDir); !ok {
			return fmt.Errorf("--push needs an origin remote, and this repository has none")
		}
	}
	return nil
}

// preview prints what a real run would do, mirroring its shape in future tense.
func preview(cfg Config, projectDir, releaseBranch, sourceBranch string, protect bool) error {
	msg.Section(cfg.Stderr, "Would initialize")
	msg.Itemf(cfg.Stderr, "Set release branch %s and guard it in %s", releaseBranch, paths.PkConfig)
	if !cfg.NoSetup {
		msg.Itemf(cfg.Stderr, "Install managed files (CLAUDE.md, rules, skills, settings)")
	}
	if tag, ok := readiness.ValidSemverTag(cfg.GitExec, projectDir); ok {
		msg.Itemf(cfg.Stderr, "Leave tag %s alone; already anchored", tag)
	} else {
		msg.Itemf(cfg.Stderr, "Tag v0.0.0 on %s", releaseBranch)
	}
	if protect {
		msg.Itemf(cfg.Stderr, "Write %s", setup.RulesetPath(releaseBranch))
	}
	msg.Itemf(cfg.Stderr, "Commit those files as %q", commitMessage(cfg))
	msg.Itemf(cfg.Stderr, "Create branch %s and switch to it", sourceBranch)
	if cfg.Push {
		msg.Itemf(cfg.Stderr, "Push %s, v0.0.0, and %s to origin", releaseBranch, sourceBranch)
	}
	if protect {
		if origin, ok := originURL(cfg, projectDir); ok {
			msg.Itemf(cfg.Stderr, "Apply the branch-protection ruleset to %s", repoSlug(origin))
		} else {
			msg.Itemf(cfg.Stderr, "Apply the branch-protection ruleset once origin points at GitHub")
		}
	}
	return nil
}

// previewShaped prints what a real run on an already-shaped repository would
// do, which is at most a publish and a branch switch.
func previewShaped(cfg Config, releaseBranch, sourceBranch, current string) error {
	if !cfg.Push && current != releaseBranch {
		fmt.Fprintln(cfg.Stderr, "Nothing to do.")
		return nil
	}
	msg.Section(cfg.Stderr, "Would do")
	if cfg.Push {
		msg.Itemf(cfg.Stderr, "Push %s, v0.0.0, and %s to origin", releaseBranch, sourceBranch)
	}
	if current == releaseBranch {
		msg.Itemf(cfg.Stderr, "Switch to %s", sourceBranch)
	}
	return nil
}

// commitSetup commits everything pk init just wrote, as one commit separate
// from the project's own work. A no-op when there is nothing to commit, which
// is what makes a re-run idempotent rather than an empty-commit generator.
func commitSetup(cfg Config, projectDir string) error {
	status, err := cfg.GitExec(projectDir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to read the working tree status: %w", err)
	}
	if strings.TrimSpace(status) == "" {
		return nil
	}
	if _, err := cfg.GitExec(projectDir, "add", "-A"); err != nil {
		return fmt.Errorf("failed to stage the setup files: %w", err)
	}
	message := commitMessage(cfg)
	if _, err := cfg.GitExec(projectDir, "commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit the setup files: %w", err)
	}
	fmt.Fprintf(cfg.Stderr, "Committed the setup files as %q\n", message)
	return nil
}

// commitMessage returns the message for the commit pk init makes: the setup
// label normally, the init label when the setup step was skipped.
func commitMessage(cfg Config) string {
	if cfg.NoSetup {
		return MinimalCommitMessage
	}
	return SetupCommitMessage
}

// publish pushes everything pk init produced, in dependency order: the release
// branch first so the tagged commit is reachable, then the tag, then the
// working branch. Never partial, per the --push convention.
func publish(cfg Config, projectDir, releaseBranch, sourceBranch string) error {
	if _, err := cfg.GitExec(projectDir, "push", "-u", "origin", releaseBranch); err != nil {
		return fmt.Errorf("failed to push %s: %w", releaseBranch, err)
	}
	if _, err := cfg.GitExec(projectDir, "push", "origin", "v0.0.0"); err != nil {
		return fmt.Errorf("failed to push v0.0.0: %w", err)
	}
	if _, err := cfg.GitExec(projectDir, "push", "-u", "origin", sourceBranch); err != nil {
		return fmt.Errorf("failed to push %s: %w", sourceBranch, err)
	}
	fmt.Fprintf(cfg.Stderr, "Pushed %s, v0.0.0, and %s to origin\n", releaseBranch, sourceBranch)
	return nil
}

// createSourceBranch creates the working branch off the release branch and
// switches to it.
func createSourceBranch(cfg Config, projectDir, sourceBranch string) error {
	if !branchExists(cfg, projectDir, sourceBranch) {
		if _, err := cfg.GitExec(projectDir, "branch", sourceBranch); err != nil {
			return fmt.Errorf("failed to create branch %s: %w", sourceBranch, err)
		}
		fmt.Fprintf(cfg.Stderr, "Created branch %s\n", sourceBranch)
	}
	return switchTo(cfg, projectDir, sourceBranch)
}

// switchTo checks out the named branch.
func switchTo(cfg Config, projectDir, branch string) error {
	if _, err := cfg.GitExec(projectDir, "switch", branch); err != nil {
		return fmt.Errorf("failed to switch to %s: %w", branch, err)
	}
	return nil
}

// summarize closes a first run with where the user is and what is left to do.
func summarize(cfg Config, projectDir, releaseBranch, sourceBranch string, protect bool) error {
	fmt.Fprintf(cfg.Stderr, "\nYou are on %s. All work happens here; %s advances only via pk release.\n", sourceBranch, releaseBranch)
	summarizeProtection(cfg, projectDir, releaseBranch, protect)
	summarizePush(cfg)
	if cfg.NoSetup {
		fmt.Fprintf(cfg.Stderr, "\nRelease management is ready: pk changelog on %s, then pk release.\n", sourceBranch)
		msg.Hintf(cfg.Stderr, "To add the Claude Code wiring later: pk setup")
	} else {
		fmt.Fprintln(cfg.Stderr, "\nNext: restart Claude Code so the hooks load, then run /pk-configure.")
	}
	return nil
}

// summarizeShaped closes a run on an already-shaped repository. Nothing was
// installed, so there is no next step beyond publishing and protection.
func summarizeShaped(cfg Config, projectDir, releaseBranch, sourceBranch, current string, protect bool) error {
	if current == sourceBranch {
		fmt.Fprintf(cfg.Stderr, "\nYou are on %s. All work happens here; %s advances only via pk release.\n", sourceBranch, releaseBranch)
	} else {
		fmt.Fprintf(cfg.Stderr, "\nYou are on %s. All work happens on %s; %s advances only via pk release.\n", current, sourceBranch, releaseBranch)
	}
	summarizeProtection(cfg, projectDir, releaseBranch, protect)
	summarizePush(cfg)
	return nil
}

// summarizeProtection says how to apply the ruleset pk wrote, when one applies.
func summarizeProtection(cfg Config, projectDir, releaseBranch string, protect bool) {
	if !protect {
		return
	}
	path := setup.RulesetPath(releaseBranch)
	fmt.Fprintln(cfg.Stderr, "\nBranch protection is not applied; pk does not call GitHub.")
	msg.Hintf(cfg.Stderr, "To import it: GitHub repo Settings, Rules, New ruleset, Import a ruleset, then choose "+path)
	if origin, ok := originURL(cfg, projectDir); ok {
		if slug := repoSlug(origin); slug != "" {
			msg.Or(cfg.Stderr, fmt.Sprintf("gh api --method POST repos/%s/rulesets --input %s", slug, path))
		}
	}
}

// summarizePush says how to publish when nothing was pushed.
func summarizePush(cfg Config) {
	if cfg.Push {
		return
	}
	fmt.Fprintln(cfg.Stderr, "\nNothing was pushed.")
	msg.Hintf(cfg.Stderr, "To publish: pk init --push")
}

// repoSlug returns the owner/repo of a remote URL, or "" when the URL is not
// a recognisable host path.
func repoSlug(remoteURL string) string {
	u := git.ParseRepoURL(remoteURL)
	// ParseRepoURL leaves a non-URL remote (a local path, say) untouched.
	// Without the scheme there is no host, so there is no owner/repo to name
	// and no gh command worth printing.
	if !strings.HasPrefix(u, "https://") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(u, "https://"), "/")
	if len(parts) < 3 {
		return ""
	}
	return strings.Join(parts[len(parts)-2:], "/")
}

// setupConfig projects the init config onto the setup config, so pk init and
// pk setup share one implementation of the managed-file work.
func setupConfig(cfg Config, projectDir string) setup.Config {
	return setup.Config{
		ProjectDir: projectDir,
		Version:    cfg.Version,
		Stderr:     cfg.Stderr,
		GitExec:    cfg.GitExec,
		ReadFile:   cfg.ReadFile,
		WriteFile:  cfg.WriteFile,
		Stat:       cfg.Stat,
		MkdirAll:   cfg.MkdirAll,
		ReadDir:    cfg.ReadDir,
		Remove:     cfg.Remove,
		Rename:     cfg.Rename,
		LookPath:   cfg.LookPath,
	}
}
