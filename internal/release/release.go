// Package release implements the pk release command.
// It validates pre-flight checks and pushes the release tag to origin.
package release

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/markwharton/plankit/internal/changelog"
)

// Config holds injectable dependencies for testing.
type Config struct {
	Stderr    io.Writer
	GitExec   func(args ...string) (string, error)
	ReadFile  func(name string) ([]byte, error)
	RunScript func(command string) error

	// Branch is the expected branch for release (default: "main").
	Branch string
	// DryRun validates without pushing.
	DryRun bool
}

// DefaultConfig returns a Config wired to real implementations.
func DefaultConfig() Config {
	return Config{
		Stderr: os.Stderr,
		GitExec: func(args ...string) (string, error) {
			out, err := exec.Command("git", args...).CombinedOutput()
			return strings.TrimRight(string(out), "\n"), err
		},
		ReadFile: os.ReadFile,
		RunScript: func(command string) error {
			cmd := exec.Command("sh", "-c", command)
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		Branch: "main",
	}
}

// semverRegex validates vX.Y.Z format.
var semverRegex = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// Run executes the release command. Returns the process exit code.
func Run(cfg Config) int {
	// 1. Find version tag at HEAD.
	tagOutput, err := cfg.GitExec("tag", "--points-at", "HEAD")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git tag failed: %s\n", err)
		return 1
	}

	tag := findVersionTag(tagOutput)
	if tag == "" {
		fmt.Fprintln(cfg.Stderr, "Error: no version tag at HEAD — run 'pk changelog' first")
		return 1
	}

	// 2. Validate semver format.
	if !semverRegex.MatchString(tag) {
		fmt.Fprintf(cfg.Stderr, "Error: tag %s is not valid semver\n", tag)
		return 1
	}

	fmt.Fprintf(cfg.Stderr, "=== Release %s ===\n\n", tag)
	fmt.Fprintln(cfg.Stderr, "--- Pre-flight checks ---")

	// 3. Pre-flight: clean working tree.
	status, err := cfg.GitExec("status", "--porcelain")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git status failed: %s\n", err)
		return 1
	}
	if status != "" {
		fmt.Fprintln(cfg.Stderr, "Error: working tree is not clean")
		return 1
	}
	fmt.Fprintln(cfg.Stderr, "  Clean working tree")

	// 4. Pre-flight: on expected branch.
	branch, err := cfg.GitExec("branch", "--show-current")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git branch failed: %s\n", err)
		return 1
	}
	if branch != cfg.Branch {
		fmt.Fprintf(cfg.Stderr, "Error: not on %s branch (on: %s)\n", cfg.Branch, branch)
		return 1
	}
	fmt.Fprintf(cfg.Stderr, "  On %s branch\n", cfg.Branch)

	// 5. Pre-flight: not behind remote.
	_, err = cfg.GitExec("fetch", "origin", cfg.Branch, "--quiet")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git fetch failed: %s\n", err)
		return 1
	}

	mergeBase, err := cfg.GitExec("merge-base", "HEAD", "origin/"+cfg.Branch)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git merge-base failed: %s\n", err)
		return 1
	}

	remote, err := cfg.GitExec("rev-parse", "origin/" + cfg.Branch)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git rev-parse failed: %s\n", err)
		return 1
	}

	if mergeBase != remote {
		fmt.Fprintf(cfg.Stderr, "Error: local %s is behind origin/%s — pull first\n", cfg.Branch, cfg.Branch)
		return 1
	}
	fmt.Fprintf(cfg.Stderr, "  Not behind origin/%s\n", cfg.Branch)

	fmt.Fprintf(cfg.Stderr, "  Tag %s exists at HEAD\n", tag)

	// 6. Run preRelease hook if configured.
	config := changelog.LoadConfig(cfg.ReadFile)
	if config.Hooks.PreRelease != "" {
		fmt.Fprintf(cfg.Stderr, "\n--- Running pre-release hook ---\n")
		fmt.Fprintf(cfg.Stderr, "  %s\n", config.Hooks.PreRelease)
		if err := cfg.RunScript(config.Hooks.PreRelease); err != nil {
			fmt.Fprintf(cfg.Stderr, "Error: pre-release hook failed: %s\n", err)
			return 1
		}
		fmt.Fprintln(cfg.Stderr, "  Hook passed")
	}

	// 7. Push or dry-run.
	if cfg.DryRun {
		fmt.Fprintf(cfg.Stderr, "\n--- Dry run complete ---\n")
		fmt.Fprintf(cfg.Stderr, "  All checks passed. Run without --dry-run to push.\n")
		return 0
	}

	fmt.Fprintf(cfg.Stderr, "\n--- Pushing to origin ---\n")
	_, err = cfg.GitExec("push", "origin", cfg.Branch, tag)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git push failed: %s\n", err)
		return 1
	}
	fmt.Fprintf(cfg.Stderr, "  Pushed %s and %s\n", cfg.Branch, tag)

	fmt.Fprintf(cfg.Stderr, "\n=== Release %s started ===\n", tag)
	return 0
}

// findVersionTag returns the first v-prefixed tag from git output.
func findVersionTag(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "v") {
			return line
		}
	}
	return ""
}
