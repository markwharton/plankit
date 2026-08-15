package setup

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// The GitHub ruleset pk init drops into a project, in rulesets-API shape.
// docs/protect-main.json is the same policy as exported by the GitHub UI, which
// adds "source"/"source_type" naming the repo it came from; the API rejects
// those, so this copy omits them and is postable as-is.
//
// The template names main; RenderRuleset derives the name and the protected
// ref from the project's release branch. The condition is a literal
// refs/heads/<release> rather than ~DEFAULT_BRANCH: the release branch is the
// one pk release advances and pk guard blocks, and it need not be the
// repository's default branch (a project may make its working branch the
// default so pull requests base there). ~DEFAULT_BRANCH would then follow the
// working branch and leave the release branch unguarded.
//
//go:embed template/protect-main.json
var rulesetTemplate []byte

// RulesetPath returns where WriteRuleset drops the ruleset for the given
// release branch, relative to the project root: .github/protect-main.json for
// the default. Referenced in hint text, so it is named once. A slash in the
// branch name becomes a hyphen so the file stays directly under .github/.
func RulesetPath(releaseBranch string) string {
	return filepath.ToSlash(filepath.Join(".github", rulesetName(releaseBranch)+".json"))
}

// rulesetName returns the ruleset's display name, protect-<release>.
func rulesetName(releaseBranch string) string {
	return "protect-" + strings.ReplaceAll(releaseBranch, "/", "-")
}

// RenderRuleset returns the ruleset for the given release branch: the embedded
// policy with its name and its ref_name condition derived from the branch.
func RenderRuleset(releaseBranch string) ([]byte, error) {
	if releaseBranch == "" {
		return nil, fmt.Errorf("cannot render a ruleset without a release branch")
	}
	top, err := ParseOrderedObject(rulesetTemplate)
	if err != nil {
		return nil, fmt.Errorf("embedded ruleset template is not a JSON object: %w", err)
	}
	name, err := json.Marshal(rulesetName(releaseBranch))
	if err != nil {
		return nil, err
	}
	top.Set("name", name)

	condRaw, _ := top.Get("conditions")
	conditions, err := ParseOrderedObject(condRaw)
	if err != nil {
		return nil, fmt.Errorf("embedded ruleset template has malformed conditions: %w", err)
	}
	refRaw, _ := conditions.Get("ref_name")
	refName, err := ParseOrderedObject(refRaw)
	if err != nil {
		return nil, fmt.Errorf("embedded ruleset template has malformed ref_name: %w", err)
	}
	include, err := json.Marshal([]string{"refs/heads/" + releaseBranch})
	if err != nil {
		return nil, err
	}
	refName.Set("include", include)
	refOut, err := MarshalNoHTML(refName)
	if err != nil {
		return nil, err
	}
	conditions.Set("ref_name", refOut)
	condOut, err := MarshalNoHTML(conditions)
	if err != nil {
		return nil, err
	}
	top.Set("conditions", condOut)

	return MarshalIndentNoHTML(top)
}

// WriteRuleset writes the branch-protection ruleset for the release branch
// into the project.
//
// It is a user-owned file with no SHA marker, so an existing copy is left
// alone rather than overwritten. That makes the file on disk the source of
// truth for what gets applied: a project that has customized its policy gets
// its own policy posted, not this default. Returns whether a file was written.
func WriteRuleset(cfg Config, projectDir, releaseBranch string) (bool, error) {
	rel := RulesetPath(releaseBranch)
	path := filepath.Join(projectDir, filepath.FromSlash(rel))
	// Present already, pristine or edited: either way the project's copy wins.
	if _, err := cfg.ReadFile(path); err == nil {
		return false, nil
	}
	content, err := RenderRuleset(releaseBranch)
	if err != nil {
		return false, err
	}
	if err := cfg.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, fmt.Errorf("failed to create .github directory: %w", err)
	}
	if err := cfg.WriteFile(path, content, 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", rel, err)
	}
	return true, nil
}
