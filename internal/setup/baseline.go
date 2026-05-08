package setup

import (
	"fmt"
	"strings"

	"github.com/markwharton/plankit/internal/version"
)

// hasValidSemverTag returns the first tag matching "v*" that parses as a valid
// semver (per pk changelog's acceptance rule), or "", false if none exists.
func hasValidSemverTag(cfg Config, projectDir string) (string, bool) {
	output, err := cfg.GitExec(projectDir, "tag", "--list", "v*", "--sort=-v:refname")
	if err != nil || output == "" {
		return "", false
	}
	for _, line := range strings.Split(output, "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}
		if _, ok := version.ParseSemver(tag); ok {
			return tag, true
		}
	}
	return "", false
}

// runBaseline creates a v0.0.0 baseline tag if no valid semver tag exists.
// If cfg.BaselineAt is set, tags that ref; otherwise tags HEAD.
// If cfg.Push is set, also pushes the tag to origin.
func runBaseline(cfg Config, projectDir string) error {
	if existing, ok := hasValidSemverTag(cfg, projectDir); ok {
		fmt.Fprintf(cfg.Stderr, "Found tag %s — already anchored\n", existing)
		return nil
	}
	target := "HEAD"
	if cfg.BaselineAt != "" {
		if _, err := cfg.GitExec(projectDir, "rev-parse", "--verify", cfg.BaselineAt); err != nil {
			return fmt.Errorf("--at ref %q does not resolve", cfg.BaselineAt)
		}
		target = cfg.BaselineAt
	} else if _, err := cfg.GitExec(projectDir, "rev-parse", "HEAD"); err != nil {
		fmt.Fprintln(cfg.Stderr, "No commits yet — commit first, then anchor with:")
		fmt.Fprintln(cfg.Stderr, "  pk setup --baseline")
		fmt.Fprintln(cfg.Stderr, "  or: git tag v0.0.0")
		return nil
	}
	if _, err := cfg.GitExec(projectDir, "tag", "v0.0.0", target); err != nil {
		return fmt.Errorf("failed to create tag v0.0.0: %w", err)
	}
	fmt.Fprintf(cfg.Stderr, "Tagged v0.0.0 on %s\n", target)
	if cfg.Push {
		// When tagging HEAD (default), also push the current branch so the tagged
		// commit is reachable from a branch on origin. When --at names a specific
		// ref, push only the tag — the user chose the ref explicitly, pk doesn't
		// assume which branch goes with it.
		pushArgs := []string{"push", "origin"}
		if cfg.BaselineAt == "" {
			pushArgs = append(pushArgs, "HEAD")
		}
		pushArgs = append(pushArgs, "v0.0.0")
		if _, err := cfg.GitExec(projectDir, pushArgs...); err != nil {
			return fmt.Errorf("failed to push baseline: %w", err)
		}
		if cfg.BaselineAt == "" {
			fmt.Fprintln(cfg.Stderr, "Pushed HEAD and v0.0.0 to origin")
		} else {
			fmt.Fprintln(cfg.Stderr, "Pushed v0.0.0 to origin")
		}
	} else {
		fmt.Fprintln(cfg.Stderr, "Run 'pk setup --baseline --push' to publish, or 'git push origin v0.0.0'")
	}
	return nil
}
