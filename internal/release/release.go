// Package release implements the pk release command.
// It validates pre-flight checks and pushes the release to origin.
// When release.branch is configured in .pk.json, it merges the current
// branch into the release branch before pushing.
package release

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Config holds injectable dependencies for testing.
type Config struct {
	Stderr    io.Writer
	GitExec   func(args ...string) (string, error)
	ReadFile  func(name string) ([]byte, error)
	RunScript func(command string) error

	// DryRun validates without merging or pushing.
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
	}
}

// semverRegex validates vX.Y.Z format.
var semverRegex = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// Run executes the release command. Returns the process exit code.
func Run(cfg Config) int {
	// 1. Get current branch (implicit source).
	sourceBranch, err := cfg.GitExec("branch", "--show-current")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git branch failed: %v\n", err)
		return 1
	}
	sourceBranch = strings.TrimSpace(sourceBranch)

	// 2. Load release config.
	releaseConf := loadReleaseConfig(cfg.ReadFile)
	releaseBranch := releaseConf.Branch
	needsMerge := releaseBranch != "" && sourceBranch != releaseBranch

	// 3. If releaseBranch is configured and we're already on it, refuse.
	if releaseBranch != "" && sourceBranch == releaseBranch {
		fmt.Fprintf(cfg.Stderr, "Error: you're on the release branch %q — switch to your development branch first\n", releaseBranch)
		return 1
	}

	// 4. Find version tag at HEAD (optional in merge flow).
	tagOutput, err := cfg.GitExec("tag", "--points-at", "HEAD")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git tag failed: %v\n", err)
		return 1
	}
	tag := findVersionTag(tagOutput)

	// In legacy flow (no releaseBranch), tag is required.
	if !needsMerge && tag == "" {
		fmt.Fprintln(cfg.Stderr, "Error: no version tag at HEAD — run 'pk changelog' first")
		return 1
	}

	// Validate semver if tag exists.
	if tag != "" && !semverRegex.MatchString(tag) {
		fmt.Fprintf(cfg.Stderr, "Error: tag %s is not valid semver\n", tag)
		return 1
	}

	// 5. Print header.
	if tag != "" {
		fmt.Fprintf(cfg.Stderr, "=== Release %s ===\n\n", tag)
	} else {
		fmt.Fprintf(cfg.Stderr, "=== Release %s → %s ===\n\n", sourceBranch, releaseBranch)
	}

	fmt.Fprintln(cfg.Stderr, "--- Pre-flight checks ---")

	// 6. Pre-flight: clean working tree.
	status, err := cfg.GitExec("status", "--porcelain")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git status failed: %v\n", err)
		return 1
	}
	if status != "" {
		fmt.Fprintln(cfg.Stderr, "Error: working tree is not clean")
		return 1
	}
	fmt.Fprintln(cfg.Stderr, "  Clean working tree")

	// 7. Pre-flight: source branch not behind remote.
	_, err = cfg.GitExec("fetch", "origin", sourceBranch, "--quiet")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git fetch failed: %v\n", err)
		return 1
	}

	mergeBase, err := cfg.GitExec("merge-base", "HEAD", "origin/"+sourceBranch)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git merge-base failed: %v\n", err)
		return 1
	}

	remote, err := cfg.GitExec("rev-parse", "origin/"+sourceBranch)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git rev-parse failed: %v\n", err)
		return 1
	}

	if mergeBase != remote {
		fmt.Fprintf(cfg.Stderr, "Error: local %s is behind origin/%s — pull first\n", sourceBranch, sourceBranch)
		return 1
	}
	fmt.Fprintf(cfg.Stderr, "  Not behind origin/%s\n", sourceBranch)

	if tag != "" {
		fmt.Fprintf(cfg.Stderr, "  Tag %s exists at HEAD\n", tag)
	}

	// 8. Merge flow.
	switchedBack := true // default: no switch-back needed
	if needsMerge {
		if cfg.DryRun {
			// Check fast-forward is possible without actually merging.
			_, err := cfg.GitExec("merge-base", "--is-ancestor", releaseBranch, sourceBranch)
			if err != nil {
				fmt.Fprintf(cfg.Stderr, "Error: merge would not be fast-forward — %s has diverged from %s. Resolve on %s manually, then try again.\n", releaseBranch, sourceBranch, releaseBranch)
				return 1
			}
			fmt.Fprintf(cfg.Stderr, "  Would merge %s into %s (fast-forward)\n", sourceBranch, releaseBranch)
		} else {
			// Fetch release branch.
			cfg.GitExec("fetch", "origin", releaseBranch, "--quiet")

			// Switch to release branch.
			if _, err := cfg.GitExec("switch", releaseBranch); err != nil {
				fmt.Fprintf(cfg.Stderr, "Error: failed to switch to %s: %v\n", releaseBranch, err)
				return 1
			}

			// Deferred switch-back on failure.
			switchedBack = false
			defer func() {
				if !switchedBack {
					cfg.GitExec("switch", sourceBranch)
				}
			}()

			// Merge from source branch (fast-forward only).
			if _, err := cfg.GitExec("merge", "--ff-only", sourceBranch); err != nil {
				fmt.Fprintf(cfg.Stderr, "Error: merge failed — %s has diverged from %s (not fast-forward). Resolve on %s manually, then try again.\n", releaseBranch, sourceBranch, releaseBranch)
				return 1
			}
			fmt.Fprintf(cfg.Stderr, "  Merged %s into %s\n", sourceBranch, releaseBranch)

			// Verify tag is still at HEAD after merge (if tag exists).
			if tag != "" {
				postMergeTagOutput, err := cfg.GitExec("tag", "--points-at", "HEAD")
				if err != nil || !strings.Contains(postMergeTagOutput, tag) {
					fmt.Fprintf(cfg.Stderr, "Error: tag %s is no longer at HEAD after merge\n", tag)
					return 1
				}
			}
		}
	} else if releaseBranch == "" {
		// Legacy flow: no releaseBranch configured.
		// We already checked tag exists above. Just note the branch.
		fmt.Fprintf(cfg.Stderr, "  On %s branch\n", sourceBranch)
	}

	// 9. Run preRelease hook if configured.
	if releaseConf.Hooks.PreRelease != "" {
		fmt.Fprintf(cfg.Stderr, "\n--- Running pre-release hook ---\n")
		fmt.Fprintf(cfg.Stderr, "  %s\n", releaseConf.Hooks.PreRelease)
		if err := cfg.RunScript(releaseConf.Hooks.PreRelease); err != nil {
			fmt.Fprintf(cfg.Stderr, "Error: pre-release hook failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(cfg.Stderr, "  Hook passed")
	}

	// 10. Dry run complete.
	if cfg.DryRun {
		fmt.Fprintf(cfg.Stderr, "\n--- Dry run complete ---\n")
		fmt.Fprintf(cfg.Stderr, "  All checks passed. Run without --dry-run to push.\n")
		return 0
	}

	// 11. Push.
	fmt.Fprintf(cfg.Stderr, "\n--- Pushing to origin ---\n")

	pushBranch := sourceBranch
	if needsMerge {
		pushBranch = releaseBranch
	}

	if tag != "" {
		_, err = cfg.GitExec("push", "origin", pushBranch, tag)
	} else {
		_, err = cfg.GitExec("push", "origin", pushBranch)
	}
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git push failed: %v\n", err)
		return 1
	}

	if tag != "" {
		fmt.Fprintf(cfg.Stderr, "  Pushed %s and %s\n", pushBranch, tag)
	} else {
		fmt.Fprintf(cfg.Stderr, "  Pushed %s\n", pushBranch)
	}

	// 12. Switch back and push source branch.
	if needsMerge {
		if _, err := cfg.GitExec("switch", sourceBranch); err != nil {
			fmt.Fprintf(cfg.Stderr, "Warning: failed to switch back to %s: %v\n", sourceBranch, err)
		}
		switchedBack = true

		if _, err := cfg.GitExec("push", "origin", sourceBranch); err != nil {
			fmt.Fprintf(cfg.Stderr, "Warning: failed to push %s: %v\n", sourceBranch, err)
		} else {
			fmt.Fprintf(cfg.Stderr, "  Pushed %s\n", sourceBranch)
		}
	}

	if tag != "" {
		fmt.Fprintf(cfg.Stderr, "\n=== Release %s complete ===\n", tag)
	} else {
		fmt.Fprintf(cfg.Stderr, "\n=== Release %s → %s complete ===\n", sourceBranch, releaseBranch)
	}
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

// ReleaseHooks holds lifecycle hook commands for the release process.
type ReleaseHooks struct {
	PreRelease string `json:"preRelease,omitempty"`
}

// ReleaseSection holds the release config from .pk.json.
type ReleaseSection struct {
	Branch string       `json:"branch,omitempty"`
	Hooks  ReleaseHooks `json:"hooks,omitempty"`
}

// pkReleaseConfig reads the release section from .pk.json.
type pkReleaseConfig struct {
	Release ReleaseSection `json:"release,omitempty"`
}

// loadReleaseConfig reads the release section from .pk.json.
func loadReleaseConfig(readFile func(string) ([]byte, error)) ReleaseSection {
	data, err := readFile(".pk.json")
	if err != nil {
		return ReleaseSection{}
	}
	var cfg pkReleaseConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ReleaseSection{}
	}
	return cfg.Release
}
