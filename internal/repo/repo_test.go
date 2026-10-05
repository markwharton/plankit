package repo

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
)

func scratch(t *testing.T, commit bool) string {
	t.Helper()
	dir := t.TempDir()
	mustGit(t, dir, "init", "-q", "-b", "main")
	mustGit(t, dir, "config", "user.email", "t@t")
	mustGit(t, dir, "config", "user.name", "t")
	if commit {
		if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		mustGit(t, dir, "add", ".")
		mustGit(t, dir, "commit", "-q", "-m", "first")
	}
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, err := git.Exec(dir, args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errw bytes.Buffer
	code := cli.RunIO(append([]string{"pk"}, args...),
		[]*cli.Command{InitCmd, StatusCmd}, nil, &out, &errw)
	return code, out.String(), errw.String()
}

// A v0.x repository is already configured, so re-running setup out of
// habit lands on the refusal. Sending that reader to "edit .pk.json"
// answers the wrong question: their hooks are firing twice.
func TestInitRefusalNamesLegacyWiring(t *testing.T) {
	dir := scratch(t, true)
	if code, out, errw := run(t, "init", "--project-dir", dir); code != cli.ExitOK {
		t.Fatalf("first init exit %d: %s%s", code, out, errw)
	}

	// Configured, nothing left over: the ordinary refusal.
	code, _, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitState || !strings.Contains(errw, "pk status") {
		t.Fatalf("ordinary refusal: code=%d errw=%q", code, errw)
	}

	// Configured, carrying a path only a v0.x pk setup created.
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "rules", "plankit"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, _, errw = run(t, "init", "--project-dir", dir)
	if code != cli.ExitState || !strings.Contains(errw, "pk help overview") {
		t.Fatalf("legacy refusal: code=%d errw=%q", code, errw)
	}
	if strings.Contains(errw, "pk status") {
		t.Fatalf("the legacy hint replaces the ordinary one: %q", errw)
	}
}

func TestInitThenStatusRoundTrips(t *testing.T) {
	dir := scratch(t, true)
	code, out, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitOK {
		t.Fatalf("init exit %d: %s%s", code, out, errw)
	}
	if out != "" {
		t.Fatalf("init has a commit path, so stdout stays empty; got %q", out)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("written config does not load: %v", err)
	}
	if cfg.Release.Branch != "main" || len(cfg.Guard.Branches) != 1 || cfg.Guard.Branches[0] != "main" {
		t.Fatalf("defaults wrong: %+v", cfg)
	}
	if _, err := os.Stat(cfg.Preserve.DirPath(dir)); !os.IsNotExist(err) {
		t.Fatal("init must not create the plans directory; preserve creates it on first use")
	}
	if got := git.LatestTag(dir); got != "" {
		t.Fatalf("init must not tag; got %q", got)
	}
	// The brief is printed in full: the session that ran init was not
	// briefed at its start. The hidden plan type stays out of it.
	for _, want := range []string{"plankit configured", "is configured in this repository", "Conventional Commits", "feat, fix,", "changelog.types"} {
		if !strings.Contains(errw, want) {
			t.Errorf("init output missing %q:\n%s", want, errw)
		}
	}
	if strings.Contains(errw, "style, plan") {
		t.Errorf("hidden plan type leaked into the brief:\n%s", errw)
	}

	code, out, errw = run(t, "status", "--project-dir", dir)
	if code != cli.ExitOK {
		t.Fatalf("status exit %d", code)
	}
	for _, want := range []string{"develop (clean)", "preserve", "manual", "block", "none yet", "0 preserved"} {
		if !strings.Contains(out, want) {
			t.Errorf("status missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(errw, "Note:") {
		t.Errorf("a bootstrapped repository has no readiness notes:\n%s", errw)
	}
}

// A fresh repository is the case the bootstrap exists for: no commits,
// the checked-out branch about to become protected. One run leaves it
// committed, tagged, and on a working branch.
func TestInitBootstrapsEmptyRepo(t *testing.T) {
	dir := scratch(t, false)
	code, out, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitOK || out != "" {
		t.Fatalf("code=%d out=%q errw=%q", code, out, errw)
	}
	subject, _ := git.Exec(dir, "log", "-1", "--format=%s")
	if subject != configureSubject {
		t.Fatalf("commit subject = %q", subject)
	}
	if n, _ := git.Exec(dir, "rev-list", "--count", "HEAD"); n != "1" {
		t.Fatalf("commit count = %s", n)
	}
	if tracked, _ := git.Exec(dir, "ls-tree", "-r", "--name-only", "HEAD"); tracked != settingsFile+"\n"+config.FileName {
		t.Fatalf("root commit tracks %q", tracked)
	}
	if git.LatestTag(dir) != "" {
		t.Fatal("init must not tag")
	}
	if b, _ := git.CurrentBranch(dir); b != "develop" {
		t.Fatalf("on %q, want develop", b)
	}
	if !git.BranchExists(dir, "main") {
		t.Fatal("main is gone")
	}
	if strings.Contains(errw, "Note:") || strings.Contains(errw, "Hint:") {
		t.Fatalf("nothing was skipped and nothing preceded, so no note or hint:\n%s", errw)
	}
}

func TestInitCommitsOnlyThePolicy(t *testing.T) {
	dir := scratch(t, true)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errw := run(t, "init", "--project-dir", dir); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	if changed, _ := git.Exec(dir, "diff", "--name-only", "HEAD~1", "HEAD"); changed != settingsFile+"\n"+config.FileName {
		t.Fatalf("commit touched %q", changed)
	}
	if clean, _ := git.Clean(dir); clean {
		t.Fatal("the unrelated change was swept into the commit")
	}
}

func TestInitKeepsAnExistingWorkingBranch(t *testing.T) {
	dir := scratch(t, true)
	mustGit(t, dir, "branch", "feature")
	code, _, _ := run(t, "init", "--project-dir", dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if b, _ := git.CurrentBranch(dir); b != "main" {
		t.Fatalf("switched to %q; a repository with its own branch keeps it", b)
	}
	if git.BranchExists(dir, "develop") {
		t.Fatal("created develop beside an existing working branch")
	}
}

func TestInitBranchFlag(t *testing.T) {
	dir := scratch(t, false)
	if code, _, errw := run(t, "init", "--project-dir", dir, "--branch", "dev"); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	if b, _ := git.CurrentBranch(dir); b != "dev" {
		t.Fatalf("on %q, want dev", b)
	}

	other := scratch(t, false)
	code, _, errw := run(t, "init", "--project-dir", other, "--branch", "main")
	if code != cli.ExitUsage || !strings.Contains(errw, "release branch") {
		t.Fatalf("--branch equal to --release: code=%d errw=%q", code, errw)
	}
	if _, err := os.Stat(config.Path(other)); !os.IsNotExist(err) {
		t.Fatal("a usage error wrote .pk.json")
	}
}

func TestInitNoCommitLeavesTheRestToTheDeveloper(t *testing.T) {
	dir := scratch(t, false)
	code, out, errw := run(t, "init", "--project-dir", dir, "--no-commit")
	if code != cli.ExitOK || out != "" {
		t.Fatalf("code=%d out=%q errw=%q", code, out, errw)
	}
	if git.HasCommits(dir) {
		t.Fatal("--no-commit committed")
	}
	if untracked, _ := git.Exec(dir, "ls-files", "--others"); untracked != settingsFile+"\n"+config.FileName {
		t.Fatalf("untracked = %q", untracked)
	}
	if git.LatestTag(dir) != "" || git.BranchExists(dir, "develop") {
		t.Fatal("tagged or branched without a commit")
	}
	for _, want := range []string{"commit .pk.json and .claude/settings.json yourself", "guard blocks", "then git switch -c develop"} {
		if !strings.Contains(errw, want) {
			t.Errorf("missing %q:\n%s", want, errw)
		}
	}
}

// Work committed before init is in the first release with everything
// after it. Init says how much and names the exact command that keeps
// it as history instead; it places no tag itself.
func TestInitWorkBeforeInitJoinsTheFirstRelease(t *testing.T) {
	dir := scratch(t, true)
	sha, _ := git.Exec(dir, "rev-parse", "--short", "HEAD")
	code, _, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if git.LatestTag(dir) != "" {
		t.Fatal("init tagged")
	}
	if subject, _ := git.Exec(dir, "log", "-1", "--format=%s"); subject != configureSubject {
		t.Fatalf("not committed: HEAD is %q", subject)
	}
	if !strings.Contains(errw, "Note: 1 commits precede plankit's and will be in the first release; to keep them as history, tag the last of them: git tag v0.0.0 "+sha) {
		t.Fatalf("note missing or wrong:\n%s", errw)
	}
	if _, _, quiet := run(t, "init", "--project-dir", scratch(t, true), "--quiet"); strings.Contains(quiet, "Note:") {
		t.Fatalf("--quiet still notes:\n%s", quiet)
	}
}

// Started on the working branch: the release branch is named, does not
// exist, and is created at the root so it carries no unreleased work.
// The policy lands on the branch checked out.
func TestInitFromWorkingBranchCreatesReleaseAtRoot(t *testing.T) {
	dir := scratch(t, false)
	mustGit(t, dir, "switch", "-q", "-c", "develop")
	for _, m := range []string{"feat: one", "fix: two"} {
		if err := os.WriteFile(filepath.Join(dir, "w.txt"), []byte(m+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		mustGit(t, dir, "add", ".")
		mustGit(t, dir, "commit", "-q", "-m", m)
	}
	root, _ := git.RootCommit(dir)
	code, _, errw := run(t, "init", "--project-dir", dir, "--release", "main")
	if code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	if b, _ := git.CurrentBranch(dir); b != "develop" {
		t.Fatalf("on %q, want develop", b)
	}
	if at, _ := git.Exec(dir, "rev-parse", "main"); at != root {
		t.Fatalf("main at %s, want the root %s", at, root)
	}
	if subject, _ := git.Exec(dir, "log", "-1", "--format=%s"); subject != configureSubject {
		t.Fatalf("policy not on develop: HEAD is %q", subject)
	}
	if !strings.Contains(errw, "created: .pk.json, .claude/settings.json, commit, branch main") || !strings.Contains(errw, "2 commits precede") {
		t.Fatalf("summary or note wrong:\n%s", errw)
	}

	// Without --release, the checked-out develop would be the release
	// branch and the working branch at once: refused, naming the fix.
	dir = scratch(t, false)
	mustGit(t, dir, "switch", "-q", "-c", "develop")
	code, _, errw = run(t, "init", "--project-dir", dir)
	if code != cli.ExitUsage || !strings.Contains(errw, "pk init --release main") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if _, err := os.Stat(config.Path(dir)); !os.IsNotExist(err) {
		t.Fatal("refused, yet .pk.json was written")
	}
}

func TestInitDryRunTouchesNothing(t *testing.T) {
	dir := scratch(t, false)
	code, out, errw := run(t, "init", "--project-dir", dir, "--dry-run")
	if code != cli.ExitOK || out != "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	if !strings.Contains(errw, "would create: .pk.json, .claude/settings.json, commit, branch develop") {
		t.Fatalf("dry run previews the whole bootstrap:\n%s", errw)
	}
	if _, err := os.Stat(config.Path(dir)); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote .pk.json")
	}
	if _, err := os.Stat(filepath.Join(dir, settingsFile)); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote the settings file")
	}
	if git.HasCommits(dir) || git.LatestTag(dir) != "" || git.BranchExists(dir, "develop") {
		t.Fatal("dry-run changed the repository")
	}
}

func TestInitJSON(t *testing.T) {
	dir := scratch(t, false)
	code, out, _ := run(t, "init", "--project-dir", dir, "--format", "json")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json: %v in %q", err, out)
	}
	if got["release"] != "main" || got["committed"] != true || got["branch"] != "develop" || got["dryRun"] != false {
		t.Fatalf("state: %v", got)
	}
	if _, has := got["baseline"]; has {
		t.Fatalf("baseline key must be gone: %v", got)
	}
	if created, _ := got["created"].([]any); len(created) != 4 || created[1] != settingsFile {
		t.Fatalf("created = %v", got["created"])
	}
}

func TestInitFlags(t *testing.T) {
	dir := scratch(t, true)
	code, _, _ := run(t, "init", "--project-dir", dir, "--release", "trunk")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	cfg, _ := config.Load(dir)
	if cfg.Release.Branch != "trunk" || cfg.Guard.Branches[0] != "trunk" {
		t.Fatalf("release flag not honored: %+v", cfg)
	}
	// The checked-out branch is not the release branch, so it is already
	// a working branch: no develop; the named release branch is created.
	if b, _ := git.CurrentBranch(dir); b != "main" || git.BranchExists(dir, "develop") || !git.BranchExists(dir, "trunk") {
		t.Fatalf("branch = %q, develop exists = %v, trunk exists = %v", b, git.BranchExists(dir, "develop"), git.BranchExists(dir, "trunk"))
	}
}

func TestInitPush(t *testing.T) {
	dir := scratch(t, false)
	bare := filepath.Join(t.TempDir(), "origin.git")
	if _, err := git.Exec(t.TempDir(), "init", "-q", "--bare", "-b", "main", bare); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "remote", "add", "origin", bare)
	code, _, errw := run(t, "init", "--project-dir", dir, "--push")
	if code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	for _, ref := range []string{"refs/heads/main", "refs/heads/develop"} {
		if _, err := git.Exec(bare, "rev-parse", "--verify", "-q", ref); err != nil {
			t.Errorf("origin lacks %s", ref)
		}
	}
	if up, _ := git.Exec(dir, "rev-parse", "--abbrev-ref", "develop@{upstream}"); up != "origin/develop" {
		t.Fatalf("develop upstream = %q", up)
	}
	if !strings.Contains(errw, "push origin main develop") {
		t.Fatalf("push not reported:\n%s", errw)
	}
}

func TestInitPushPreconditions(t *testing.T) {
	dir := scratch(t, false)
	code, _, errw := run(t, "init", "--project-dir", dir, "--push")
	if code != cli.ExitState || !strings.Contains(errw, "no origin remote") {
		t.Fatalf("without origin: code=%d errw=%q", code, errw)
	}
	if _, err := os.Stat(config.Path(dir)); !os.IsNotExist(err) {
		t.Fatal("a refused --push wrote .pk.json")
	}

	code, _, errw = run(t, "init", "--project-dir", dir, "--push", "--no-commit")
	if code != cli.ExitUsage || !strings.Contains(errw, "--no-commit") {
		t.Fatalf("--push --no-commit: code=%d errw=%q", code, errw)
	}
}

func TestStatusReadinessNotes(t *testing.T) {
	// Configured by hand: policy committed on main, no tag, no other
	// branch. Status names both missing steps.
	dir := scratch(t, true)
	if err := config.Write(dir, config.Default("main")); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "chore: configure plankit")
	code, _, errw := run(t, "status", "--project-dir", dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"no working branch besides main", "git switch -c develop", settingsFile + " lacks the entries", `"plankit@plankit": true`} {
		if !strings.Contains(errw, want) {
			t.Errorf("missing %q:\n%s", want, errw)
		}
	}
	if _, _, quiet := run(t, "status", "--project-dir", dir, "--quiet"); strings.Contains(quiet, "Note:") {
		t.Fatalf("--quiet still notes:\n%s", quiet)
	}
}

func TestStatusUnconfiguredExitsState(t *testing.T) {
	dir := scratch(t, true)
	code, out, errw := run(t, "status", "--project-dir", dir)
	if code != cli.ExitState {
		t.Fatalf("exit %d, want %d", code, cli.ExitState)
	}
	if !strings.Contains(out, "not configured") || !strings.Contains(errw, "pk init") {
		t.Fatalf("out=%q errw=%q", out, errw)
	}
}

func TestStatusNotARepo(t *testing.T) {
	code, _, errw := run(t, "status", "--project-dir", t.TempDir())
	if code != cli.ExitState || !strings.Contains(errw, "not a git repository") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
}

func TestStatusJSON(t *testing.T) {
	dir := scratch(t, true)
	run(t, "init", "--project-dir", dir)
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Preserve.Dir = "documentation/plans"
	if err := config.Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.Preserve.DirPath(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Preserve.DirPath(dir), "2026-01-01-1-x.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run(t, "status", "--project-dir", dir, "--format", "json")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	var s map[string]any
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("json: %v in %q", err, out)
	}
	if s["configured"] != true || s["plans"] != float64(1) || s["plansDir"] != "documentation/plans" || s["releaseBranch"] != "main" {
		t.Fatalf("state: %v", s)
	}
	_, text, _ := run(t, "status", "--project-dir", dir)
	if !strings.Contains(text, "documentation/plans/ immutable") || !strings.Contains(text, "1 preserved") {
		t.Fatalf("text report: %s", text)
	}

	// Unconfigured json still emits the report; exit code carries state.
	bare := scratch(t, true)
	code, out, _ = run(t, "status", "--project-dir", bare, "--format", "json")
	if code != cli.ExitState || !strings.Contains(out, `"configured":false`) {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestDefaultProjectDirWalksToRoot(t *testing.T) {
	t.Setenv("PK_PROJECT_DIR", "")
	dir := scratch(t, true)
	run(t, "init", "--project-dir", dir)
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	code, out, _ := run(t, "status")
	if code != cli.ExitOK || !strings.Contains(out, dir) {
		t.Fatalf("from subdir: code=%d out=%q", code, out)
	}
}

// The developer's settings file, with spacing no encoder would produce,
// hooks, and permissions. Init adds the two entries and nothing else
// changes, not a byte.
const developerSettings = "{\n    \"permissions\": {\n        \"allow\": [\"Bash(pk:*)\"]\n    },\n    \"hooks\":  {\"Stop\": [{\"matcher\": \"\", \"hooks\": [{\"type\": \"command\", \"command\": \"say done && true\"}]}]},\n    \"enabledPlugins\": { \"other@other\": true }\n}\n"

func writeSettings(t *testing.T, dir, content string, commit bool) {
	t.Helper()
	path := filepath.Join(dir, settingsFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if commit {
		mustGit(t, dir, "add", settingsFile)
		mustGit(t, dir, "commit", "-q", "-m", "chore: settings")
	}
}

func TestInitSplicesTheDeveloperSettings(t *testing.T) {
	dir := scratch(t, true)
	writeSettings(t, dir, developerSettings, true)
	if code, _, errw := run(t, "init", "--project-dir", dir); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	got, _ := os.ReadFile(filepath.Join(dir, settingsFile))
	want := strings.Replace(developerSettings,
		"\"enabledPlugins\": { \"other@other\": true }\n",
		"\"enabledPlugins\": { \"other@other\": true, \"plankit@plankit\": true },\n    \"extraKnownMarketplaces\": {\n        \"plankit\": {\n            \"source\": {\n                \"source\": \"url\",\n                \"url\": \"https://plankit.com/marketplace.json\"\n            }\n        }\n    }\n", 1)
	if string(got) != want {
		t.Fatalf("settings after init:\n%s\nwant\n%s", got, want)
	}
	if changed, _ := git.Exec(dir, "diff", "--name-only", "HEAD~1", "HEAD"); changed != settingsFile+"\n"+config.FileName {
		t.Fatalf("commit touched %q", changed)
	}
}

func TestInitLeavesPresentEntriesAlone(t *testing.T) {
	// Both entries present, the marketplace pointing elsewhere: that is
	// the developer's choice. Nothing is written, the commit carries
	// only the policy, and the difference is reported.
	dir := scratch(t, true)
	content := "{\"enabledPlugins\":{\"plankit@plankit\":true},\"extraKnownMarketplaces\":{\"plankit\":{\"source\":{\"source\":\"github\",\"repo\":\"me/plankit\"}}}}\n"
	writeSettings(t, dir, content, true)
	code, _, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, settingsFile)); string(got) != content {
		t.Fatalf("settings rewritten:\n%s", got)
	}
	if changed, _ := git.Exec(dir, "diff", "--name-only", "HEAD~1", "HEAD"); changed != config.FileName {
		t.Fatalf("commit touched %q", changed)
	}
	if !strings.Contains(errw, "extraKnownMarketplaces.plankit is present with another value; left as it is") {
		t.Fatalf("difference not reported:\n%s", errw)
	}
	if strings.Contains(errw, "created: .pk.json, .claude") {
		t.Fatalf("settings listed as created:\n%s", errw)
	}
}

func TestInitRefusesDirtySettings(t *testing.T) {
	dir := scratch(t, true)
	writeSettings(t, dir, "{\"model\": \"opus\"}\n", true)
	writeSettings(t, dir, "{\"model\": \"sonnet\"}\n", false)
	code, _, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitState || !strings.Contains(errw, settingsFile+" has uncommitted changes") || !strings.Contains(errw, "commit or stash") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if _, err := os.Stat(config.Path(dir)); !os.IsNotExist(err) {
		t.Fatal("refused, yet .pk.json was written")
	}
}

func TestInitRefusesMalformedSettings(t *testing.T) {
	dir := scratch(t, true)
	writeSettings(t, dir, "{\n", true)
	code, _, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitState || !strings.Contains(errw, settingsFile+":") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if _, err := os.Stat(config.Path(dir)); !os.IsNotExist(err) {
		t.Fatal("refused, yet .pk.json was written")
	}
	if clean, _ := git.Clean(dir); !clean {
		t.Fatal("refusal left the tree dirty")
	}
}

func TestStatusPluginEntriesNote(t *testing.T) {
	dir := scratch(t, true)
	if code, _, errw := run(t, "init", "--project-dir", dir); code != cli.ExitOK {
		t.Fatalf("init exit %d: %s", code, errw)
	}
	if _, _, errw := run(t, "status", "--project-dir", dir); strings.Contains(errw, "Note:") {
		t.Fatalf("init wrote the entries, so no note:\n%s", errw)
	}
	// One entry gone: the note shows only that one, as the exact JSON to
	// merge, and --quiet drops it.
	writeSettings(t, dir, "{\n  \"enabledPlugins\": {\n    \"plankit@plankit\": true\n  }\n}\n", true)
	_, _, errw := run(t, "status", "--project-dir", dir)
	if !strings.Contains(errw, "Note: "+settingsFile+" lacks the entries") || !strings.Contains(errw, `"url": "https://plankit.com/marketplace.json"`) {
		t.Fatalf("note missing:\n%s", errw)
	}
	if strings.Contains(errw, "enabledPlugins") {
		t.Fatalf("present entry shown as missing:\n%s", errw)
	}
	if _, _, quiet := run(t, "status", "--project-dir", dir, "--quiet"); strings.Contains(quiet, "Note:") {
		t.Fatalf("--quiet still notes:\n%s", quiet)
	}
	// A file that does not parse is a problem, not a note: every mode
	// says so and exits 2.
	writeSettings(t, dir, "{\n", true)
	for _, args := range [][]string{{"status"}, {"status", "--quiet"}, {"status", "--format", "json"}} {
		code, _, errw := run(t, append(args, "--project-dir", dir)...)
		if code != cli.ExitState || !strings.Contains(errw, settingsFile+":") {
			t.Fatalf("%v: code=%d errw=%q", args, code, errw)
		}
	}
}

// From a working branch, the checked-out branch is probably not the
// release branch. When origin says which one is, init refuses to guess.
func TestInitRefusesToGuardAWorkingBranch(t *testing.T) {
	dir := scratch(t, true)
	bare := filepath.Join(t.TempDir(), "origin.git")
	if _, err := git.Exec(t.TempDir(), "init", "-q", "--bare", "-b", "main", bare); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "remote", "add", "origin", bare)
	mustGit(t, dir, "push", "-q", "origin", "main")
	mustGit(t, dir, "remote", "set-head", "origin", "main")
	mustGit(t, dir, "switch", "-q", "-c", "dev")

	code, _, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitState || !strings.Contains(errw, "on dev, but origin's default branch is main") || !strings.Contains(errw, "name the branch to guard and release into: pk init --release main, or pk init --release dev") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if _, err := os.Stat(config.Path(dir)); !os.IsNotExist(err) {
		t.Fatal("refused, yet .pk.json was written")
	}

	// Named, the release branch is taken as given, and the commit lands
	// on the branch checked out.
	if code, _, errw := run(t, "init", "--project-dir", dir, "--release", "main"); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, errw)
	}
	cfg, err := config.Load(dir)
	if err != nil || cfg.Release.Branch != "main" {
		t.Fatalf("policy: %+v %v", cfg, err)
	}
	if b, _ := git.CurrentBranch(dir); b != "dev" {
		t.Fatalf("on %q, want dev", b)
	}
}

// Init prints the brief, which reads the plugin the shim named; a
// named plugin that is not there stops init before it writes. The
// comparison sentence itself is the brief's test.
func TestInitReadsThePluginTheShimNamed(t *testing.T) {
	plugin := t.TempDir()
	if err := os.MkdirAll(filepath.Join(plugin, ".claude-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plugin, ".claude-plugin", "plugin.json"), []byte(`{"version": "999.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PK_PLUGIN_ROOT", plugin)
	dir := scratch(t, true)
	code, _, errw := run(t, "init", "--project-dir", dir)
	if code != cli.ExitOK || !strings.Contains(errw, "is configured in this repository.") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	t.Setenv("PK_PLUGIN_ROOT", t.TempDir())
	dir = scratch(t, true)
	if code, _, errw := run(t, "init", "--project-dir", dir); code != cli.ExitState || !strings.Contains(errw, "plugin.json") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if _, err := os.Stat(config.Path(dir)); !os.IsNotExist(err) {
		t.Fatal("refused, yet .pk.json was written")
	}
}

// The release branch as git shows it: a hand-configured repository
// without one gets the exact command that creates it at the root, and
// in merge flow a release branch that moved past its last tag is
// counted. Trunk flow and a repository with no release yet get nothing.
func TestStatusNotesTheReleaseBranchState(t *testing.T) {
	dir := scratch(t, true)
	mustGit(t, dir, "switch", "-q", "-c", "develop")
	mustGit(t, dir, "branch", "-D", "main")
	if err := config.Write(dir, config.Default("main")); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "chore: configure plankit")
	first, _ := git.RootCommit(dir)
	_, _, errw := run(t, "status", "--project-dir", dir)
	want := "release branch main does not exist: git branch main " + first[:7]
	if !strings.Contains(errw, want) {
		t.Fatalf("missing %q:\n%s", want, errw)
	}
	// The command, run as written, ends the note.
	mustGit(t, dir, "branch", "main", first[:7])
	if _, _, errw := run(t, "status", "--project-dir", dir); strings.Contains(errw, "does not exist") {
		t.Fatalf("note after the branch exists:\n%s", errw)
	}

	// Merge flow, released once, then main moves: counted.
	dir = scratch(t, true)
	if code, _, errw := run(t, "init", "--project-dir", dir); code != cli.ExitOK {
		t.Fatalf("init: %s", errw)
	}
	mustGit(t, dir, "tag", "v0.1.0")
	if _, _, errw := run(t, "status", "--project-dir", dir); strings.Contains(errw, "not in a release") {
		t.Fatalf("level with its tag, yet noted:\n%s", errw)
	}
	mustGit(t, dir, "switch", "-q", "main")
	if err := os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-q", "-m", "fix: on main by hand")
	mustGit(t, dir, "switch", "-q", "develop")
	if _, _, errw := run(t, "status", "--project-dir", dir); !strings.Contains(errw, "main has 1 commits not in a release") {
		t.Fatalf("moved release branch not noted:\n%s", errw)
	}
	// Trunk flow: the branch moving is the normal state.
	cfg, _ := config.Load(dir)
	cfg.Release.Branch = ""
	if err := config.Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	if _, _, errw := run(t, "status", "--project-dir", dir); strings.Contains(errw, "not in a release") {
		t.Fatalf("trunk flow noted:\n%s", errw)
	}
}
