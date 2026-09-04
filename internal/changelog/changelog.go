// Package changelog implements pk changelog: it generates CHANGELOG.md
// from conventional commits since the last tag and commits the result.
// The commit body carries a Release-Tag trailer so pk release can create
// the git tag at the right moment; pk changelog --undo unwinds an
// unpushed release commit.
//
// Ported from v1. The v2 deltas: an unconfigured repository is refused
// (state exit, hint pk init) rather than defaulted; the dry-run section
// prints on stdout so it can be redirected, with all progress on stderr;
// and exits follow the layer-0 taxonomy (state 2 for refusals, usage 1,
// internal 3).
package changelog

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hookio"
	"github.com/markwharton/plankit/internal/msg"
	"github.com/markwharton/plankit/internal/version"
)

// now is stubbed in tests to pin section dates.
var now = time.Now

// Cmd is the changelog command.
var Cmd = &cli.Command{
	Name:    "changelog",
	Summary: "Generate CHANGELOG.md from conventional commits and commit it with a Release-Tag trailer",
	Flags: []cli.FlagSpec{
		{Name: "bump", Type: cli.StringFlag, Usage: "Override the version bump: major, minor, or patch"},
		{Name: "dry-run", Type: cli.BoolFlag, Usage: "Print the section to stdout without writing or committing"},
		{Name: "exclude", Type: cli.StringFlag, Usage: "Comma-separated commit SHAs to drop from the section"},
		{Name: "undo", Type: cli.BoolFlag, Usage: "Unwind the last pk changelog commit (must be unpushed)"},
	},
	Run: run,
}

// Commit is a parsed conventional commit.
type Commit struct {
	Hash     string
	Type     string
	Scope    string
	Message  string
	Breaking bool
}

// CommitGroup holds commits under one changelog section heading.
type CommitGroup struct {
	Heading string
	Items   []Commit
}

// commitRegex parses conventional subjects: type(scope)!: message
var commitRegex = regexp.MustCompile(`^(\w+)(?:\(([^)]*)\))?(!)?\s*:\s*(.+)$`)

// refLinkDefRegex matches a markdown reference link definition.
var refLinkDefRegex = regexp.MustCompile(`^\[[^\]]+\]:\s`)

const changelogHeader = `# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).
`

func run(ctx *cli.Context) error {
	if ctx.Format == "json" {
		return cli.Usagef("--format json is not supported by pk changelog")
	}
	root, ok := git.FindRoot(ctx.ProjectDir)
	if !ok {
		return cli.Statef("not a git repository: %s", ctx.ProjectDir)
	}
	cfg, err := loadConfig(root)
	if err != nil {
		return err
	}
	if ctx.Bool("undo") {
		return undo(ctx, root)
	}
	dryRun := ctx.Bool("dry-run")

	// A dirty tree is a more general failure than the branch checks, so
	// it goes first: fix the tree before worrying about where you are.
	if !dryRun {
		if err := git.CheckCleanTree(root); err != nil {
			return cli.Statef("%v", err)
		}
	}

	branch, _ := git.Exec(root, "branch", "--show-current")

	// Guarded branch: the release commit belongs on the working branch.
	for _, protected := range cfg.Guard.Branches {
		if branch == protected {
			err := cli.Statef("you're on %q which is a protected branch; switch to your development branch first", branch)
			if !git.HasOtherLocalBranch(root, branch) {
				err = cli.WithHint(err, "to start one: git switch -c develop && git push -u origin develop")
			}
			return err
		}
	}

	// Trunk flow (no release.branch): releases publish from the default
	// branch on origin, so refuse any other branch. Runs before the
	// branch-on-origin check, whose "push it" hint would otherwise walk
	// the user straight into this refusal. Skipped on detached HEAD and
	// when origin advertises no HEAD symref.
	if cfg.Release.Branch == "" && branch != "" {
		if def, ok, derr := git.DefaultBranch(root); derr == nil && ok && branch != def {
			return cli.WithHint(
				cli.Statef("you're on %q but the default branch on origin is %q; trunk flow releases from the default branch", branch, def),
				"to release this work from %s: git switch %s && git merge %s, then pk changelog && pk release", def, def, branch)
		}
	}

	// The branch must exist on origin, or pk changelog succeeds and pk
	// release fails, stranding a Release-Tag commit behind a manual push.
	if branch != "" {
		if _, err := git.Exec(root, "ls-remote", "--exit-code", "--heads", "origin", branch); err != nil {
			return cli.WithHint(
				cli.Statef("%s does not exist on origin", branch),
				"to push it: git push -u origin %s", branch)
		}
	}

	// Latest semver tag anchors the commit range and the next version.
	tagOutput, err := git.Exec(root, "tag", "--list", "v*", "--sort=-v:refname")
	if err != nil {
		return fmt.Errorf("failed to list tags: %v", err)
	}
	latestTag, baseVersion, found := latestSemverTag(tagOutput)
	if !found {
		// Origin has tags, local doesn't: common in shallow clones that
		// fetched only the working branch. Point at fetch, not baseline.
		if remoteTags, err := git.Exec(root, "ls-remote", "--tags", "origin"); err == nil && strings.TrimSpace(remoteTags) != "" {
			return cli.WithHint(cli.Statef("no version tags found locally"),
				"origin has tags; fetch them: git fetch --tags")
		}
		return cli.WithHint(cli.Statef("no version tags found"),
			"to anchor at v0.0.0: git tag v0.0.0 && git push origin v0.0.0")
	}

	logOutput, err := git.Exec(root, "log", "--format=%h%x00%s%x00%b%x00", latestTag+"..HEAD", "--reverse")
	if err != nil {
		return fmt.Errorf("failed to read git log: %v", err)
	}
	commits := parseLog(logOutput)
	if len(commits) == 0 {
		fmt.Fprintln(ctx.Stderr, "No new conventional commits found.")
		return nil
	}
	if x := ctx.String("exclude"); x != "" {
		commits = applyExclude(ctx, commits, strings.Split(x, ","))
		if len(commits) == 0 {
			fmt.Fprintln(ctx.Stderr, "No conventional commits remain after --exclude.")
			return nil
		}
	}
	fmt.Fprintf(ctx.Stderr, "Found %d conventional commit(s)\n", len(commits))

	bump, err := resolveBump(ctx.String("bump"), commits)
	if err != nil {
		return err
	}
	next := baseVersion.Bump(bump)
	nextTag := next.String()
	fmt.Fprintf(ctx.Stderr, "Generating %s\n", nextTag)

	// Refuse when HEAD already carries a Release-Tag trailer: one pending
	// release at a time.
	if !dryRun {
		if _, pending, err := ReadReleaseTagTrailer(root); err == nil {
			return cli.WithHint(
				cli.Statef("changelog for %s is already pending (HEAD has Release-Tag: %s)", pending, pending),
				"to complete the release: pk release; to undo and start over: pk changelog --undo")
		}
	}

	groups := groupCommits(commits, cfg.Changelog.Types)
	if len(groups) == 0 {
		fmt.Fprintln(ctx.Stderr, "No visible commits after grouping (all hidden or unmapped types).")
		return nil
	}
	section := formatSection(nextTag, now().Format("2006-01-02"), groups, cfg.Changelog.ShowScope)

	// Dry run: the section is the artifact, so it goes to stdout and can
	// be redirected; everything else in this command narrates on stderr.
	if dryRun {
		fmt.Fprint(ctx.Stdout, section)
		return nil
	}

	ver := strings.TrimPrefix(nextTag, "v")
	for _, vf := range cfg.Changelog.VersionFiles {
		if vf.Type != "" && vf.Type != "json" {
			return cli.Usagef("unsupported versionFile type %q for %s (only \"json\" is supported)", vf.Type, vf.Path)
		}
		if err := updateVersionFile(filepath.Join(root, vf.Path), ver); err != nil {
			return cli.Statef("failed to update %s: %v", vf.Path, err)
		}
		fmt.Fprintf(ctx.Stderr, "Updated %s\n", vf.Path)
	}
	if h := cfg.Changelog.Hooks.PostVersion; h != "" {
		fmt.Fprintln(ctx.Stderr, "Running postVersion hook...")
		if err := hookio.RunScript(ctx.Stderr, root, h, map[string]string{"VERSION": ver}); err != nil {
			return cli.Statef("postVersion hook failed: %v", err)
		}
	}

	repoURL := ""
	if remoteURL, err := git.Exec(root, "remote", "get-url", "origin"); err == nil {
		repoURL = git.ParseRepoURL(remoteURL)
	}
	changelogPath := filepath.Join(root, "CHANGELOG.md")
	existing, _ := readFile(changelogPath)
	updated := insertSection(string(existing), section)
	if repoURL != "" {
		updated = appendRefLink(updated, fmt.Sprintf("[%s]: %s/compare/%s...%s", nextTag, repoURL, latestTag, nextTag))
	}
	if err := writeFile(changelogPath, []byte(updated)); err != nil {
		return fmt.Errorf("failed to write CHANGELOG.md: %v", err)
	}

	if h := cfg.Changelog.Hooks.PreCommit; h != "" {
		fmt.Fprintln(ctx.Stderr, "Running preCommit hook...")
		if err := hookio.RunScript(ctx.Stderr, root, h, map[string]string{"VERSION": ver}); err != nil {
			return cli.Statef("preCommit hook failed: %v", err)
		}
	}

	// Commit. The body carries the Release-Tag trailer pk release reads;
	// no git tag is created here, that is pk release's moment.
	addFiles := []string{"add", changelogPath}
	for _, vf := range cfg.Changelog.VersionFiles {
		addFiles = append(addFiles, filepath.Join(root, vf.Path))
	}
	if _, err := git.Exec(root, addFiles...); err != nil {
		return fmt.Errorf("git add failed: %v", err)
	}
	// Also stage tracked files a hook modified.
	if _, err := git.Exec(root, "add", "-u"); err != nil {
		return fmt.Errorf("git add failed: %v", err)
	}
	if _, err := git.Exec(root, "commit", "-m", "chore: release "+nextTag, "--trailer", "Release-Tag: "+nextTag); err != nil {
		return fmt.Errorf("git commit failed: %v", err)
	}
	fmt.Fprintf(ctx.Stderr, "Committed %s\n", nextTag)
	if !ctx.Quiet {
		msg.Hintf(ctx.Stderr, "to tag and push: pk release")
	}
	return nil
}

// undo unwinds an unpushed pk changelog commit: HEAD must carry a valid
// Release-Tag trailer, the tree must be clean, and HEAD must not be on
// the remote. git reset --hard HEAD~1 restores CHANGELOG.md and version
// files.
func undo(ctx *cli.Context, root string) error {
	_, trailerValue, err := ReadReleaseTagTrailer(root)
	if err != nil {
		return cli.Statef("%v", err)
	}
	if err := git.CheckCleanTree(root); err != nil {
		return cli.Statef("%v", err)
	}
	// No upstream means the commit cannot be on the remote: undo is safe.
	if upstream, err := git.Exec(root, "rev-parse", "--abbrev-ref", "HEAD@{upstream}"); err == nil && strings.TrimSpace(upstream) != "" {
		ahead, err := git.Exec(root, "log", "@{u}..HEAD", "--oneline")
		if err != nil {
			return fmt.Errorf("git log @{u}..HEAD failed: %v", err)
		}
		if strings.TrimSpace(ahead) == "" {
			return cli.Statef("HEAD is already on the remote; cannot undo a pushed commit")
		}
	}
	if _, err := git.Exec(root, "reset", "--hard", "HEAD~1"); err != nil {
		return fmt.Errorf("git reset failed: %v", err)
	}
	fmt.Fprintf(ctx.Stderr, "Undid release commit %s; CHANGELOG.md and version files restored\n", trailerValue)
	return nil
}

// fullConfig is what this command needs from .pk.json: its own section
// with the default type table applied, plus guard and release.
type fullConfig struct {
	Changelog config.ChangelogConfig
	Guard     config.GuardConfig
	Release   config.ReleaseSection
}

func loadConfig(root string) (*fullConfig, error) {
	pk, err := config.Load(root)
	if errors.Is(err, config.ErrNotConfigured) {
		return nil, cli.WithHint(cli.Statef("%v", err), "run pk init to configure this repository")
	}
	if err != nil {
		return nil, cli.Statef("%v", err)
	}
	if len(pk.Changelog.Types) == 0 {
		pk.Changelog.Types = config.Default("").Changelog.Types
	}
	return &fullConfig{Changelog: pk.Changelog, Guard: pk.Guard, Release: pk.Release}, nil
}

// parseCommit parses one conventional commit subject and body.
func parseCommit(hash, subject, body string) (Commit, bool) {
	m := commitRegex.FindStringSubmatch(subject)
	if m == nil {
		return Commit{}, false
	}
	c := Commit{Hash: hash, Type: m[1], Scope: m[2], Breaking: m[3] == "!", Message: m[4]}
	if !c.Breaking {
		c.Breaking = hasBreakingTrailer(body)
	}
	return c, true
}

func hasBreakingTrailer(body string) bool {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "BREAKING CHANGE:") || strings.HasPrefix(t, "BREAKING-CHANGE:") {
			return true
		}
	}
	return false
}

// parseLog splits NUL-delimited git log output (%h%x00%s%x00%b%x00).
func parseLog(output string) []Commit {
	if strings.TrimSpace(output) == "" {
		return nil
	}
	fields := strings.Split(output, "\x00")
	var commits []Commit
	for i := 0; i+2 < len(fields); i += 3 {
		hash := strings.TrimSpace(fields[i])
		subject := strings.TrimSpace(fields[i+1])
		body := strings.TrimSpace(fields[i+2])
		if hash == "" || subject == "" {
			continue
		}
		if c, ok := parseCommit(hash, subject, body); ok {
			commits = append(commits, c)
		}
	}
	return commits
}

// applyExclude drops commits whose short hash matches an exclude entry
// (exact match, as the hash appears in CHANGELOG.md parentheses).
// Unmatched entries warn but do not fail; the filter runs before bump
// detection so the bump reflects the commits that will actually appear.
func applyExclude(ctx *cli.Context, commits []Commit, excludes []string) []Commit {
	wanted := make(map[string]bool, len(excludes))
	for _, e := range excludes {
		if e = strings.TrimSpace(e); e != "" {
			wanted[e] = false
		}
	}
	if len(wanted) == 0 {
		return commits
	}
	kept := commits[:0]
	for _, c := range commits {
		if _, ok := wanted[c.Hash]; ok {
			wanted[c.Hash] = true
			continue
		}
		kept = append(kept, c)
	}
	for sha, matched := range wanted {
		if !matched {
			msg.Warnf(ctx.Stderr, "--exclude %s did not match any commit", sha)
		}
	}
	return kept
}

func detectBump(commits []Commit) int {
	bump := version.BumpPatch
	for _, c := range commits {
		if c.Breaking {
			return version.BumpMajor
		}
		if c.Type == "feat" && bump < version.BumpMinor {
			bump = version.BumpMinor
		}
	}
	return bump
}

func resolveBump(flag string, commits []Commit) (int, error) {
	switch flag {
	case "":
		return detectBump(commits), nil
	case "major":
		return version.BumpMajor, nil
	case "minor":
		return version.BumpMinor, nil
	case "patch":
		return version.BumpPatch, nil
	default:
		return 0, cli.Usagef("invalid --bump value %q (must be major, minor, or patch)", flag)
	}
}

// groupCommits groups by section, hidden types excluded, section order
// following the config's first appearance of each section.
func groupCommits(commits []Commit, types []config.TypeConfig) []CommitGroup {
	typeSection := make(map[string]string)
	hidden := make(map[string]bool)
	for _, tc := range types {
		if tc.Hidden {
			hidden[tc.Type] = true
		} else {
			typeSection[tc.Type] = tc.Section
		}
	}
	sectionCommits := make(map[string][]Commit)
	for _, c := range commits {
		if hidden[c.Type] {
			continue
		}
		if section, ok := typeSection[c.Type]; ok {
			sectionCommits[section] = append(sectionCommits[section], c)
		}
	}
	var groups []CommitGroup
	seen := make(map[string]bool)
	for _, tc := range types {
		if tc.Hidden || seen[tc.Section] {
			continue
		}
		seen[tc.Section] = true
		if items, ok := sectionCommits[tc.Section]; ok {
			groups = append(groups, CommitGroup{Heading: tc.Section, Items: items})
		}
	}
	return groups
}

// codeSpanRe matches a balanced single-backtick code span; a lone
// backtick renders literally in GFM, so its surroundings get escaped.
var codeSpanRe = regexp.MustCompile("`[^`]*`")

// escapeText escapes the two characters that change meaning in GFM
// text, & first so nothing double-escapes. Everything else renders the
// same escaped or not, so the raw CHANGELOG stays readable as text.
func escapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	return strings.ReplaceAll(s, "<", "&lt;")
}

// escapeSubject escapes the plain-text parts of a subject, leaving the
// author's backtick code spans verbatim (escaping inside one would
// surface a literal &lt;).
func escapeSubject(s string) string {
	var b strings.Builder
	last := 0
	for _, loc := range codeSpanRe.FindAllStringIndex(s, -1) {
		b.WriteString(escapeText(s[last:loc[0]]))
		b.WriteString(s[loc[0]:loc[1]])
		last = loc[1]
	}
	b.WriteString(escapeText(s[last:]))
	return b.String()
}

func formatSection(ver, date string, groups []CommitGroup, showScope bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## [%s] - %s\n", ver, date)
	for _, g := range groups {
		fmt.Fprintf(&b, "\n### %s\n\n", g.Heading)
		for _, c := range g.Items {
			prefix := ""
			if c.Breaking {
				prefix = "**BREAKING:** "
			}
			if showScope && c.Scope != "" {
				prefix += "**" + escapeSubject(c.Scope) + ":** "
			}
			fmt.Fprintf(&b, "- %s%s (%s)\n", prefix, escapeSubject(c.Message), c.Hash)
		}
	}
	return b.String()
}

// insertSection places a new version section above the first existing
// one, or starts a fresh file with the standard header.
func insertSection(existing, section string) string {
	if strings.TrimSpace(existing) == "" {
		return changelogHeader + "\n" + section
	}
	if idx := strings.Index(existing, "\n## ["); idx >= 0 {
		return existing[:idx+1] + section + "\n" + existing[idx+1:]
	}
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	return existing + "\n" + section
}

// appendRefLink appends a reference link definition: single newline when
// following another definition, double when separating from prose.
func appendRefLink(content, refLink string) string {
	if strings.Contains(content, refLink) {
		return content
	}
	s := strings.TrimRight(content, "\n")
	if refLinkDefRegex.MatchString(lastLine(s)) {
		return s + "\n" + refLink + "\n"
	}
	return s + "\n\n" + refLink + "\n"
}

func lastLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return lines[i]
		}
	}
	return ""
}

func latestSemverTag(tagOutput string) (string, version.Semver, bool) {
	for _, line := range strings.Split(tagOutput, "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}
		if sv, ok := version.ParseSemver(tag); ok {
			return tag, sv, true
		}
	}
	return "", version.Semver{}, false
}
