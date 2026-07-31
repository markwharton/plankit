package setup

import (
	_ "embed"
	"fmt"
	"path/filepath"
)

// The GitHub ruleset pk init drops into a project, in rulesets-API shape.
// docs/protect-main.json is the same policy as exported by the GitHub UI, which
// adds "source"/"source_type" naming the repo it came from; the API rejects
// those, so this copy omits them and is postable as-is.
//
// The condition targets ~DEFAULT_BRANCH rather than a literal branch name, so
// the ruleset keeps protecting the right branch if the project renames it.
//
//go:embed template/protect-main.json
var rulesetTemplate []byte

// RulesetPath is where WriteRuleset drops the ruleset, relative to the
// project root. Referenced in hint text, so it is named once.
const RulesetPath = ".github/protect-main.json"

// WriteRuleset writes the branch-protection ruleset into the project.
//
// It is a user-owned file with no SHA marker, so an existing copy is left
// alone rather than overwritten. That makes the file on disk the source of
// truth for what gets applied: a project that has customized its policy gets
// its own policy posted, not this default. Returns whether a file was written.
func WriteRuleset(cfg Config, projectDir string) (bool, error) {
	path := filepath.Join(projectDir, RulesetPath)
	// Present already, pristine or edited: either way the project's copy wins.
	if _, err := cfg.ReadFile(path); err == nil {
		return false, nil
	}
	if err := cfg.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, fmt.Errorf("failed to create .github directory: %w", err)
	}
	if err := cfg.WriteFile(path, rulesetTemplate, 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", RulesetPath, err)
	}
	return true, nil
}
