// Package release implements the pk release command.
// It reads the Release-Tag trailer from the HEAD commit (written by pk
// changelog), creates the git tag, validates pre-flight checks, and pushes
// the release to origin. When release.branch is configured in .pk.json, it
// merges the current branch into the release branch before pushing.
package release

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	pkgit "github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/version"
)

// Config holds injectable dependencies for testing.
type Config struct {
	Stderr    io.Writer
	GitExec   func(dir string, args ...string) (string, error)
	ReadFile  func(name string) ([]byte, error)
	RunScript func(command string, env map[string]string) error

	// DryRun validates without merging or pushing.
	DryRun bool
}

// DefaultConfig returns a Config wired to real implementations.
func DefaultConfig() Config {
	return Config{
		Stderr:    os.Stderr,
		GitExec:   pkgit.Exec,
		ReadFile:  os.ReadFile,
		RunScript: defaultRunScript,
	}
}

// defaultRunScript runs a shell command with optional environment variables.
func defaultRunScript(command string, env map[string]string) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return cmd.Run()
}

// Run executes the release command. Returns the process exit code.
func Run(cfg Config) int {
	// 1. Get current branch (implicit source).
	sourceBranch, err := cfg.GitExec("", "branch", "--show-current")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git branch failed: %v\n", err)
		return 1
	}
	sourceBranch = strings.TrimSpace(sourceBranch)

	// 2. Load release config.
	releaseConf, err := loadReleaseConfig(cfg.ReadFile)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: %v\n", err)
		return 1
	}
	releaseBranch := releaseConf.Branch
	needsMerge := releaseBranch != "" && sourceBranch != releaseBranch

	// 3. If releaseBranch is configured and we're already on it, refuse.
	if releaseBranch != "" && sourceBranch == releaseBranch {
		fmt.Fprintf(cfg.Stderr, "Error: you're on the release branch %q — switch to your development branch first\n", releaseBranch)
		return 1
	}

	// 4. Read Release-Tag trailer from HEAD.
	trailerOut, err := cfg.GitExec("", "log", "-1", "--format=%(trailers:key=Release-Tag,valueonly)", "HEAD")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git log failed: %v\n", err)
		return 1
	}
	trailerValue := strings.TrimSpace(trailerOut)
	if trailerValue == "" {
		fmt.Fprintln(cfg.Stderr, "Error: no Release-Tag trailer on HEAD — run 'pk changelog' first")
		return 1
	}

	// Validate: must parse as semver and round-trip exactly (no trailing garbage).
	parsed, ok := version.ParseSemver(trailerValue)
	if !ok || parsed.String() != trailerValue {
		fmt.Fprintf(cfg.Stderr, "Error: Release-Tag trailer value %q is not valid semver\n", trailerValue)
		return 1
	}
	tag := trailerValue

	// 5. Refuse if the tag already exists locally.
	existingTag, err := cfg.GitExec("", "tag", "--list", tag)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git tag --list failed: %v\n", err)
		return 1
	}
	if strings.TrimSpace(existingTag) != "" {
		fmt.Fprintf(cfg.Stderr, "Error: tag %s already exists locally — nothing to release\n", tag)
		return 1
	}

	// 6. Print header.
	fmt.Fprintf(cfg.Stderr, "=== Release %s ===\n\n", tag)

	fmt.Fprintln(cfg.Stderr, "--- Pre-flight checks ---")

	// 6. Pre-flight: clean working tree.
	status, err := cfg.GitExec("", "status", "--porcelain")
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
	_, err = cfg.GitExec("", "fetch", "origin", sourceBranch, "--quiet")
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git fetch failed: %v\n", err)
		return 1
	}

	mergeBase, err := cfg.GitExec("", "merge-base", "HEAD", "origin/"+sourceBranch)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git merge-base failed: %v\n", err)
		return 1
	}

	remote, err := cfg.GitExec("", "rev-parse", "origin/"+sourceBranch)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git rev-parse failed: %v\n", err)
		return 1
	}

	if mergeBase != remote {
		fmt.Fprintf(cfg.Stderr, "Error: local %s is behind origin/%s — pull first\n", sourceBranch, sourceBranch)
		return 1
	}
	fmt.Fprintf(cfg.Stderr, "  Not behind origin/%s\n", sourceBranch)
	fmt.Fprintf(cfg.Stderr, "  Release-Tag trailer: %s\n", tag)

	// 8. Merge flow.
	tagCreated := false
	released := false
	switchedBack := true // default: no switch-back needed
	defer func() {
		if tagCreated && !released {
			if _, err := cfg.GitExec("", "tag", "-d", tag); err != nil {
				fmt.Fprintf(cfg.Stderr, "Warning: failed to delete local tag %s: %v\n", tag, err)
			} else {
				fmt.Fprintf(cfg.Stderr, "Cleaned up local tag %s\n", tag)
			}
		}
		if !switchedBack {
			if _, err := cfg.GitExec("", "switch", sourceBranch); err != nil {
				fmt.Fprintf(cfg.Stderr, "Warning: failed to switch back to %s: %v\n", sourceBranch, err)
			}
		}
	}()
	if needsMerge {
		if cfg.DryRun {
			// Check fast-forward is possible without actually merging.
			_, err := cfg.GitExec("", "merge-base", "--is-ancestor", releaseBranch, sourceBranch)
			if err != nil {
				fmt.Fprintf(cfg.Stderr, "Error: merge would not be fast-forward — %s has diverged from %s. Resolve on %s manually, then try again.\n", releaseBranch, sourceBranch, releaseBranch)
				return 1
			}
			fmt.Fprintf(cfg.Stderr, "  Would merge %s into %s (fast-forward)\n", sourceBranch, releaseBranch)
		} else {
			// Fetch release branch.
			cfg.GitExec("", "fetch", "origin", releaseBranch, "--quiet")

			// Switch to release branch. Mark switchedBack=false so the
			// top-level defer switches back on any subsequent failure.
			if _, err := cfg.GitExec("", "switch", releaseBranch); err != nil {
				fmt.Fprintf(cfg.Stderr, "Error: failed to switch to %s: %v\n", releaseBranch, err)
				return 1
			}
			switchedBack = false

			// Merge from source branch (fast-forward only).
			if _, err := cfg.GitExec("", "merge", "--ff-only", sourceBranch); err != nil {
				fmt.Fprintf(cfg.Stderr, "Error: merge failed — %s has diverged from %s (not fast-forward). Resolve on %s manually, then try again.\n", releaseBranch, sourceBranch, releaseBranch)
				return 1
			}
			fmt.Fprintf(cfg.Stderr, "  Merged %s into %s\n", sourceBranch, releaseBranch)
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
		if err := cfg.RunScript(releaseConf.Hooks.PreRelease, nil); err != nil {
			fmt.Fprintf(cfg.Stderr, "Error: pre-release hook failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(cfg.Stderr, "  Hook passed")
	}

	// 10. Dry run complete (merge/legacy flows).
	if cfg.DryRun {
		fmt.Fprintf(cfg.Stderr, "\n--- Dry run complete ---\n")
		fmt.Fprintf(cfg.Stderr, "  Would create tag %s\n", tag)
		fmt.Fprintf(cfg.Stderr, "  All checks passed. Run without --dry-run to release.\n")
		return 0
	}

	// 11. Create the local tag on HEAD (which is either source HEAD in legacy
	// flow or release-branch HEAD after the fast-forward merge — same commit).
	// Cleanup on subsequent failure is handled by the top-level defer.
	if _, err := cfg.GitExec("", "tag", tag); err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git tag failed: %v\n", err)
		return 1
	}
	tagCreated = true
	fmt.Fprintf(cfg.Stderr, "\n--- Created local tag %s ---\n", tag)

	// 12. Push.
	fmt.Fprintf(cfg.Stderr, "\n--- Pushing to origin ---\n")

	pushBranch := sourceBranch
	if needsMerge {
		pushBranch = releaseBranch
	}

	if _, err := cfg.GitExec("", "push", "origin", pushBranch, tag); err != nil {
		fmt.Fprintf(cfg.Stderr, "Error: git push failed: %v\n", err)
		return 1
	}
	released = true
	fmt.Fprintf(cfg.Stderr, "  Pushed %s and %s\n", pushBranch, tag)

	// 13. Switch back and push source branch.
	if needsMerge {
		if _, err := cfg.GitExec("", "switch", sourceBranch); err != nil {
			fmt.Fprintf(cfg.Stderr, "Warning: failed to switch back to %s: %v\n", sourceBranch, err)
		}
		switchedBack = true

		if _, err := cfg.GitExec("", "push", "origin", sourceBranch); err != nil {
			fmt.Fprintf(cfg.Stderr, "Warning: failed to push %s: %v\n", sourceBranch, err)
		} else {
			fmt.Fprintf(cfg.Stderr, "  Pushed %s\n", sourceBranch)
		}
	}

	fmt.Fprintf(cfg.Stderr, "\n=== Release %s complete ===\n", tag)
	return 0
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
// Returns an error if the file exists but contains malformed JSON.
func loadReleaseConfig(readFile func(string) ([]byte, error)) (ReleaseSection, error) {
	data, err := readFile(".pk.json")
	if err != nil {
		return ReleaseSection{}, nil
	}
	var cfg pkReleaseConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ReleaseSection{}, fmt.Errorf("failed to parse .pk.json: %w", err)
	}
	return cfg.Release, nil
}
