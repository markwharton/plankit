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
	if got := git.LatestTag(dir); got != "v0.0.0" {
		t.Fatalf("baseline tag = %q", got)
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
	for _, want := range []string{"develop (clean)", "preserve", "manual", "block", "v0.0.0", "0 preserved"} {
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
	if tracked, _ := git.Exec(dir, "ls-tree", "--name-only", "HEAD"); tracked != config.FileName {
		t.Fatalf("root commit tracks %q", tracked)
	}
	if git.LatestTag(dir) != "v0.0.0" {
		t.Fatal("no baseline tag")
	}
	if b, _ := git.CurrentBranch(dir); b != "develop" {
		t.Fatalf("on %q, want develop", b)
	}
	if !git.BranchExists(dir, "main") {
		t.Fatal("main is gone")
	}
	if strings.Contains(errw, "no baseline tag") || strings.Contains(errw, "Hint:") {
		t.Fatalf("nothing was skipped, so no note or hint:\n%s", errw)
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
	if changed, _ := git.Exec(dir, "diff", "--name-only", "HEAD~1", "HEAD"); changed != config.FileName {
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
	if untracked, _ := git.Exec(dir, "ls-files", "--others"); untracked != config.FileName {
		t.Fatalf("untracked = %q", untracked)
	}
	if git.LatestTag(dir) != "" || git.BranchExists(dir, "develop") {
		t.Fatal("tagged or branched without a commit")
	}
	for _, want := range []string{"no baseline tag", "guard blocks", "git tag v0.0.0 && git switch -c develop"} {
		if !strings.Contains(errw, want) {
			t.Errorf("missing %q:\n%s", want, errw)
		}
	}
}

func TestInitNoBaselineOnRepoWithCommits(t *testing.T) {
	dir := scratch(t, true)
	code, _, errw := run(t, "init", "--project-dir", dir, "--no-baseline")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if git.LatestTag(dir) != "" {
		t.Fatal("--no-baseline still tagged")
	}
	if subject, _ := git.Exec(dir, "log", "-1", "--format=%s"); subject != configureSubject {
		t.Fatalf("not committed: HEAD is %q", subject)
	}
	if strings.Contains(errw, "no commits yet") {
		t.Fatalf("the repository has commits; the note lies:\n%s", errw)
	}
}

func TestInitDryRunTouchesNothing(t *testing.T) {
	dir := scratch(t, false)
	code, out, errw := run(t, "init", "--project-dir", dir, "--dry-run")
	if code != cli.ExitOK || out != "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	if !strings.Contains(errw, "would create: .pk.json, commit, tag v0.0.0, branch develop") {
		t.Fatalf("dry run previews the whole bootstrap:\n%s", errw)
	}
	if _, err := os.Stat(config.Path(dir)); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote .pk.json")
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
	if got["release"] != "main" || got["committed"] != true || got["baseline"] != "v0.0.0" || got["branch"] != "develop" || got["dryRun"] != false {
		t.Fatalf("state: %v", got)
	}
	if created, _ := got["created"].([]any); len(created) != 4 {
		t.Fatalf("created = %v", got["created"])
	}
}

func TestInitFlags(t *testing.T) {
	dir := scratch(t, true)
	code, _, _ := run(t, "init", "--project-dir", dir, "--release", "trunk", "--no-baseline")
	if code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	cfg, _ := config.Load(dir)
	if cfg.Release.Branch != "trunk" || cfg.Guard.Branches[0] != "trunk" {
		t.Fatalf("release flag not honored: %+v", cfg)
	}
	// The checked-out branch is not the release branch, so it is already
	// a working branch: nothing to create.
	if b, _ := git.CurrentBranch(dir); b != "main" || git.BranchExists(dir, "develop") {
		t.Fatalf("branch = %q, develop exists = %v", b, git.BranchExists(dir, "develop"))
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
	for _, ref := range []string{"refs/heads/main", "refs/heads/develop", "refs/tags/v0.0.0"} {
		if _, err := git.Exec(bare, "rev-parse", "--verify", "-q", ref); err != nil {
			t.Errorf("origin lacks %s", ref)
		}
	}
	if up, _ := git.Exec(dir, "rev-parse", "--abbrev-ref", "develop@{upstream}"); up != "origin/develop" {
		t.Fatalf("develop upstream = %q", up)
	}
	if !strings.Contains(errw, "push origin main v0.0.0 develop") {
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
	for _, want := range []string{"no baseline tag", "git tag v0.0.0", "no working branch besides main", "git switch -c develop"} {
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
