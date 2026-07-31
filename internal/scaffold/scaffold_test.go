package scaffold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/setup"
)

// fakeGit records the git commands a run issues and answers them from a
// scripted table, so a test can drive pk init without a real repository.
type fakeGit struct {
	calls    []string
	branch   string   // answer for "branch --show-current"
	tags     string   // answer for "tag --list"
	noHEAD   bool     // rev-parse HEAD fails: repository has no commits
	dirty    bool     // status --porcelain reports changes
	noOrigin bool     // remote get-url origin fails
	heads    []string // existing local branches, for rev-parse --verify refs/heads/*
	failOn   string   // command prefix that returns an error
	// originURL overrides the origin remote; empty means a GitHub remote.
	originURL string
	// onCommit fires after an anchor commit, so a test can model the
	// repository gaining its first commit part-way through a run.
	onCommit func()

	statusCalls int
}

func (f *fakeGit) exec(dir string, args ...string) (string, error) {
	joined := strings.Join(args, " ")
	f.calls = append(f.calls, joined)
	if f.failOn != "" && strings.HasPrefix(joined, f.failOn) {
		return "", fmt.Errorf("git %s failed", f.failOn)
	}
	switch {
	case joined == "branch --show-current":
		return f.branch + "\n", nil
	case joined == "rev-parse HEAD":
		if f.noHEAD {
			return "", fmt.Errorf("no commits")
		}
		return "abc1234\n", nil
	case joined == "status --porcelain":
		// Model the real sequence: preflight sees a clean tree, then pk init
		// writes its files, so the commit step sees them as untracked.
		f.statusCalls++
		if f.dirty {
			return " M some-file\n", nil
		}
		if f.statusCalls > 1 {
			return "?? CLAUDE.md\n?? .pk.json\n", nil
		}
		return "", nil
	case joined == "remote get-url origin":
		if f.noOrigin {
			return "", fmt.Errorf("no origin")
		}
		if f.originURL != "" {
			return f.originURL + "\n", nil
		}
		return "git@github.com:markwharton/demo.git\n", nil
	case strings.HasPrefix(joined, "commit --allow-empty"):
		if f.onCommit != nil {
			f.onCommit()
		}
		return "", nil
	case strings.HasPrefix(joined, "tag --list"):
		return f.tags, nil
	case strings.HasPrefix(joined, "rev-parse --verify --quiet refs/heads/"):
		name := strings.TrimPrefix(joined, "rev-parse --verify --quiet refs/heads/")
		for _, h := range f.heads {
			if h == name {
				return "abc1234\n", nil
			}
		}
		return "", fmt.Errorf("no such branch")
	}
	return "", nil
}

func (f *fakeGit) ran(prefix string) bool {
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

// newRepo returns a temp dir containing a .git marker so git.RepoRoot resolves
// it, plus a Config wired to the real filesystem and the given fake git.
func newRepo(t *testing.T, g *fakeGit) (string, *bytes.Buffer, Config) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	var stderr bytes.Buffer
	cfg := Config{
		ProjectDir: dir,
		Stderr:     &stderr,
		GitExec:    g.exec,
		ReadFile:   os.ReadFile,
		WriteFile:  os.WriteFile,
		Stat:       os.Stat,
		MkdirAll:   os.MkdirAll,
		ReadDir:    os.ReadDir,
		Remove:     os.Remove,
		Rename:     os.Rename,
		LookPath:   func(string) (string, error) { return "/usr/local/bin/pk", nil },
	}
	return dir, &stderr, cfg
}

func readPkConfig(t *testing.T, dir string) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, ".pk.json"))
	if err != nil {
		t.Fatalf(".pk.json not written: %v", err)
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf(".pk.json is not valid JSON: %v", err)
	}
	return out
}

func TestRun_freshRepo(t *testing.T) {
	g := &fakeGit{branch: "main"}
	dir, stderr, cfg := newRepo(t, g)

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Topology written, and the modes pk setup contributes are alongside it
	// rather than having clobbered it.
	pk := readPkConfig(t, dir)
	var guard struct {
		Branches []string `json:"branches"`
		Mode     string   `json:"mode"`
	}
	if err := json.Unmarshal(pk["guard"], &guard); err != nil {
		t.Fatalf("guard section: %v", err)
	}
	if len(guard.Branches) != 1 || guard.Branches[0] != "main" {
		t.Errorf("guard.branches = %v, want [main]", guard.Branches)
	}
	if guard.Mode == "" {
		t.Error("guard.mode missing; pk setup's field merge did not survive")
	}
	var release struct {
		Branch string `json:"branch"`
	}
	if err := json.Unmarshal(pk["release"], &release); err != nil {
		t.Fatalf("release section: %v", err)
	}
	if release.Branch != "main" {
		t.Errorf("release.branch = %q, want main", release.Branch)
	}

	// Managed files and the ruleset.
	for _, rel := range []string{
		"CLAUDE.md",
		".claude/settings.json",
		".claude/rules/plankit/git-discipline.md",
		".claude/skills/ship/SKILL.md",
		".github/protect-main.json",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("%s not created", rel)
		}
	}

	// The ruleset must be postable as-is: valid JSON, no UI-export fields.
	data, err := os.ReadFile(filepath.Join(dir, ".github/protect-main.json"))
	if err != nil {
		t.Fatalf("read ruleset: %v", err)
	}
	var ruleset map[string]any
	if err := json.Unmarshal(data, &ruleset); err != nil {
		t.Fatalf("ruleset is not valid JSON: %v", err)
	}
	for _, k := range []string{"source", "source_type"} {
		if _, ok := ruleset[k]; ok {
			t.Errorf("ruleset carries %q, which the rulesets API rejects", k)
		}
	}

	// Baseline tag, branch creation, and the switch.
	if !g.ran("tag v0.0.0") {
		t.Error("v0.0.0 was not tagged")
	}
	if !g.ran("branch develop") {
		t.Error("develop was not created")
	}
	if !g.ran("switch develop") {
		t.Error("did not switch to develop")
	}
	if !g.ran("commit -m " + SetupCommitMessage) {
		t.Errorf("setup files not committed; calls: %v", g.calls)
	}
	if g.ran("push") {
		t.Error("pushed without --push")
	}
	if !strings.Contains(stderr.String(), "Nothing was pushed") {
		t.Error("summary does not say nothing was pushed")
	}
}

func TestRun_pushPublishesBranchesAndTag(t *testing.T) {
	g := &fakeGit{branch: "main"}
	_, stderr, cfg := newRepo(t, g)
	cfg.Push = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// --push publishes fully, and only after the setup files are committed:
	// pushing before the commit would publish branches carrying none of them.
	for _, want := range []string{"push -u origin main", "push origin v0.0.0", "push -u origin develop"} {
		if !g.ran(want) {
			t.Errorf("did not run git %q; calls: %v", want, g.calls)
		}
	}
	commitAt, pushAt := -1, -1
	for i, c := range g.calls {
		if strings.HasPrefix(c, "commit ") && commitAt < 0 {
			commitAt = i
		}
		if strings.HasPrefix(c, "push ") && pushAt < 0 {
			pushAt = i
		}
	}
	if commitAt < 0 || pushAt < 0 || commitAt > pushAt {
		t.Errorf("commit must precede the first push; commit at %d, push at %d", commitAt, pushAt)
	}
	if strings.Contains(stderr.String(), "Nothing was pushed") {
		t.Error("summary claims nothing was pushed")
	}
}

func TestRun_dryRunWritesNothing(t *testing.T) {
	g := &fakeGit{branch: "main"}
	dir, stderr, cfg := newRepo(t, g)
	cfg.DryRun = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	for _, rel := range []string{".pk.json", "CLAUDE.md", ".claude", ".github"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
			t.Errorf("dry run created %s", rel)
		}
	}
	for _, mutating := range []string{"tag v0.0.0", "branch develop", "switch ", "push ", "commit ", "add -A"} {
		if g.ran(mutating) {
			t.Errorf("dry run ran git %q", mutating)
		}
	}
	if !strings.Contains(stderr.String(), "Would initialize:") {
		t.Error("dry run did not print a preview")
	}
}

func TestRun_idempotent(t *testing.T) {
	g := &fakeGit{branch: "main"}
	dir, _, cfg := newRepo(t, g)

	if err := Run(cfg); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	first, err := os.ReadFile(filepath.Join(dir, ".pk.json"))
	if err != nil {
		t.Fatalf("read .pk.json: %v", err)
	}

	// Second run sees the world the first one made: the tag exists and
	// develop is already a local branch.
	g2 := &fakeGit{branch: "main", tags: "v0.0.0\n", heads: []string{"develop"}}
	cfg.GitExec = g2.exec
	if err := Run(cfg); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	second, err := os.ReadFile(filepath.Join(dir, ".pk.json"))
	if err != nil {
		t.Fatalf("read .pk.json: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf(".pk.json changed on re-run:\nfirst:  %s\nsecond: %s", first, second)
	}
	if g2.ran("tag v0.0.0") {
		t.Error("re-run re-tagged v0.0.0")
	}
	if g2.ran("branch develop") {
		t.Error("re-run re-created develop")
	}
	if !g2.ran("switch develop") {
		t.Error("re-run did not leave the user on develop")
	}
}

func TestRun_preflightRefusals(t *testing.T) {
	tests := []struct {
		name    string
		git     *fakeGit
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:    "no commits",
			git:     &fakeGit{branch: "main", noHEAD: true},
			wantErr: "no commits",
		},
		{
			name:    "dirty tree",
			git:     &fakeGit{branch: "main", dirty: true},
			wantErr: "working tree is not clean",
		},
		{
			name:    "on the working branch, not the release branch",
			git:     &fakeGit{branch: "develop"},
			mutate:  func(c *Config) { c.ReleaseBranch = "main" },
			wantErr: `you are on "develop"`,
		},
		{
			name:    "detached HEAD",
			git:     &fakeGit{branch: ""},
			wantErr: "detached",
		},
		{
			name:    "push without an origin",
			git:     &fakeGit{branch: "main", noOrigin: true},
			mutate:  func(c *Config) { c.Push = true },
			wantErr: "--push needs an origin remote",
		},
		{
			name:    "source branch equals release branch",
			git:     &fakeGit{branch: "main"},
			mutate:  func(c *Config) { c.SourceBranch = "main" },
			wantErr: "they must differ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, _, cfg := newRepo(t, tt.git)
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}
			err := Run(cfg)
			if err == nil {
				t.Fatal("Run() succeeded, want refusal")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
			// A refusal must change nothing.
			for _, rel := range []string{".pk.json", "CLAUDE.md", ".claude", ".github"} {
				if _, statErr := os.Stat(filepath.Join(dir, rel)); statErr == nil {
					t.Errorf("refusal still created %s", rel)
				}
			}
		})
	}
}

func TestRun_notAGitRepo(t *testing.T) {
	g := &fakeGit{branch: "main"}
	dir := t.TempDir() // no .git marker
	var stderr bytes.Buffer
	cfg := Config{ProjectDir: dir, Stderr: &stderr, GitExec: g.exec, Stat: os.Stat}

	err := Run(cfg)
	if err == nil {
		t.Fatal("Run() succeeded outside a git repository")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("error = %q, want it to name the missing repository", err)
	}
}

func TestRun_gitFailureMidSequence(t *testing.T) {
	// The branch creation fails after the config and managed files landed.
	// The error must surface rather than being swallowed into a clean exit.
	g := &fakeGit{branch: "main", failOn: "branch develop"}
	_, _, cfg := newRepo(t, g)

	err := Run(cfg)
	if err == nil {
		t.Fatal("Run() succeeded despite a failing git branch")
	}
	if !strings.Contains(err.Error(), "failed to create branch develop") {
		t.Errorf("error = %q, want it to name the failed step", err)
	}
}

func TestRun_customBranchNames(t *testing.T) {
	g := &fakeGit{branch: "trunk"}
	dir, _, cfg := newRepo(t, g)
	cfg.ReleaseBranch = "trunk"
	cfg.SourceBranch = "work"

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	pk := readPkConfig(t, dir)
	if !strings.Contains(string(pk["release"]), `"trunk"`) {
		t.Errorf("release section = %s, want branch trunk", pk["release"])
	}
	if !strings.Contains(string(pk["guard"]), `"trunk"`) {
		t.Errorf("guard section = %s, want branches [trunk]", pk["guard"])
	}
	if !g.ran("branch work") {
		t.Error("custom source branch not created")
	}
}

func TestRun_preservesExistingPkConfigKeys(t *testing.T) {
	g := &fakeGit{branch: "main"}
	dir, _, cfg := newRepo(t, g)
	existing := []byte(`{"changelog":{"showScope":true}}`)
	if err := os.WriteFile(filepath.Join(dir, ".pk.json"), existing, 0644); err != nil {
		t.Fatalf("seed .pk.json: %v", err)
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	pk := readPkConfig(t, dir)
	if _, ok := pk["changelog"]; !ok {
		t.Error("existing changelog config was dropped")
	}
	if !strings.Contains(string(pk["changelog"]), "showScope") {
		t.Errorf("changelog section = %s, want showScope preserved", pk["changelog"])
	}
}

func TestRun_leavesAnExistingRulesetAlone(t *testing.T) {
	g := &fakeGit{branch: "main"}
	dir, _, cfg := newRepo(t, g)
	if err := os.MkdirAll(filepath.Join(dir, ".github"), 0755); err != nil {
		t.Fatalf("mkdir .github: %v", err)
	}
	custom := []byte(`{"name":"my-own-policy"}`)
	if err := os.WriteFile(filepath.Join(dir, ".github/protect-main.json"), custom, 0644); err != nil {
		t.Fatalf("seed ruleset: %v", err)
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, ".github/protect-main.json"))
	if err != nil {
		t.Fatalf("read ruleset: %v", err)
	}
	if !bytes.Equal(got, custom) {
		t.Errorf("overwrote the project's ruleset: got %s", got)
	}
}

func TestRun_nonGitHubOriginSkipsProtection(t *testing.T) {
	g := &fakeGit{branch: "main", originURL: "/srv/git/demo.git"}
	dir, stderr, cfg := newRepo(t, g)

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".github/protect-main.json")); err == nil {
		t.Error("wrote a ruleset for a remote that cannot use one")
	}
	if !strings.Contains(stderr.String(), "no GitHub remote") {
		t.Error("did not state that protection does not apply")
	}
}

func TestRun_releaseBranchComesFromConfigNotCurrentBranch(t *testing.T) {
	// pk init leaves you on the working branch. A re-run from there must not
	// infer that branch as the release branch and redefine the project: it
	// would rewrite release.branch and guard.branches, silently unguarding the
	// real release branch.
	g := &fakeGit{branch: "develop"}
	dir, _, cfg := newRepo(t, g)
	original := []byte(`{"guard":{"branches":["main"]},"release":{"branch":"main"}}`)
	if err := os.WriteFile(filepath.Join(dir, ".pk.json"), original, 0644); err != nil {
		t.Fatalf("seed .pk.json: %v", err)
	}
	cfg.SourceBranch = "work" // so a collision cannot mask the real check

	err := Run(cfg)
	if err == nil {
		t.Fatal("Run() succeeded from the working branch, want a refusal")
	}
	if !strings.Contains(err.Error(), `you are on "develop"`) || !strings.Contains(err.Error(), `"main"`) {
		t.Errorf("error = %q, want it to name both the current and the release branch", err)
	}

	after, readErr := os.ReadFile(filepath.Join(dir, ".pk.json"))
	if readErr != nil {
		t.Fatalf("read .pk.json: %v", readErr)
	}
	if !bytes.Equal(after, original) {
		t.Errorf(".pk.json was rewritten by a refused run:\n got %s\nwant %s", after, original)
	}
}

func TestRun_noSetupSkipsManagedFiles(t *testing.T) {
	g := &fakeGit{branch: "main"}
	dir, stderr, cfg := newRepo(t, g)
	cfg.NoSetup = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Topology only: guard.branches and release.branch, none of the modes
	// pk setup would field-merge alongside.
	pk := readPkConfig(t, dir)
	var guard struct {
		Branches []string `json:"branches"`
		Mode     string   `json:"mode"`
	}
	if err := json.Unmarshal(pk["guard"], &guard); err != nil {
		t.Fatalf("guard section: %v", err)
	}
	if len(guard.Branches) != 1 || guard.Branches[0] != "main" {
		t.Errorf("guard.branches = %v, want [main]", guard.Branches)
	}
	if guard.Mode != "" {
		t.Errorf("guard.mode = %q, want absent; --no-setup must not write hook modes", guard.Mode)
	}
	if _, ok := pk["preserve"]; ok {
		t.Error("preserve section written; --no-setup must not write hook modes")
	}
	if !strings.Contains(string(pk["release"]), `"main"`) {
		t.Errorf("release section = %s, want branch main", pk["release"])
	}

	// No managed files: this is the whole point of the flag.
	for _, rel := range []string{"CLAUDE.md", ".claude"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
			t.Errorf("--no-setup still created %s", rel)
		}
	}
	// The ruleset is repo shape, not Claude wiring, so it still lands.
	if _, err := os.Stat(filepath.Join(dir, ".github/protect-main.json")); err != nil {
		t.Error(".github/protect-main.json not created; protection is independent of --no-setup")
	}

	if !g.ran("tag v0.0.0") {
		t.Error("v0.0.0 was not tagged")
	}
	if !g.ran("commit -m " + MinimalCommitMessage) {
		t.Errorf("files not committed as %q; calls: %v", MinimalCommitMessage, g.calls)
	}
	if !g.ran("branch develop") || !g.ran("switch develop") {
		t.Error("working branch not created and switched to")
	}
	if strings.Contains(stderr.String(), "restart Claude Code") {
		t.Error("summary points at Claude Code hooks that were never installed")
	}
	if !strings.Contains(stderr.String(), "pk changelog") {
		t.Error("summary does not say release management is ready")
	}
	if !strings.Contains(stderr.String(), "pk setup") {
		t.Error("summary does not hint at pk setup for the full install later")
	}
}

func TestRun_noSetupDryRunWritesNothing(t *testing.T) {
	g := &fakeGit{branch: "main"}
	dir, stderr, cfg := newRepo(t, g)
	cfg.NoSetup = true
	cfg.DryRun = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	for _, rel := range []string{".pk.json", "CLAUDE.md", ".claude", ".github"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
			t.Errorf("dry run created %s", rel)
		}
	}
	for _, mutating := range []string{"tag v0.0.0", "branch develop", "switch ", "push ", "commit ", "add -A"} {
		if g.ran(mutating) {
			t.Errorf("dry run ran git %q", mutating)
		}
	}
	if strings.Contains(stderr.String(), "Install managed files") {
		t.Error("preview lists the managed-file install --no-setup skips")
	}
	if !strings.Contains(stderr.String(), MinimalCommitMessage) {
		t.Errorf("preview does not show the %q commit", MinimalCommitMessage)
	}
}

func TestRun_noSetupIdempotent(t *testing.T) {
	g := &fakeGit{branch: "main"}
	dir, _, cfg := newRepo(t, g)
	cfg.NoSetup = true

	if err := Run(cfg); err != nil {
		t.Fatalf("first Run() error = %v", err)
	}
	first, err := os.ReadFile(filepath.Join(dir, ".pk.json"))
	if err != nil {
		t.Fatalf("read .pk.json: %v", err)
	}

	g2 := &fakeGit{branch: "main", tags: "v0.0.0\n", heads: []string{"develop"}}
	cfg.GitExec = g2.exec
	if err := Run(cfg); err != nil {
		t.Fatalf("second Run() error = %v", err)
	}

	second, err := os.ReadFile(filepath.Join(dir, ".pk.json"))
	if err != nil {
		t.Fatalf("read .pk.json: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf(".pk.json changed on re-run:\nfirst:  %s\nsecond: %s", first, second)
	}
	if g2.ran("tag v0.0.0") {
		t.Error("re-run re-tagged v0.0.0")
	}
	if g2.ran("branch develop") {
		t.Error("re-run re-created develop")
	}
}

func TestRun_noSetupThenFullSetupUpgrades(t *testing.T) {
	// A minimal repo must upgrade cleanly: pk setup later field-merges its
	// modes alongside the preserved topology and installs the managed files.
	g := &fakeGit{branch: "main"}
	dir, stderr, cfg := newRepo(t, g)
	cfg.NoSetup = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	sc := setupConfig(cfg, dir)
	sc.Stderr = stderr
	if err := setup.Run(sc); err != nil {
		t.Fatalf("setup.Run() error = %v", err)
	}

	pk := readPkConfig(t, dir)
	var guard struct {
		Branches []string `json:"branches"`
		Mode     string   `json:"mode"`
	}
	if err := json.Unmarshal(pk["guard"], &guard); err != nil {
		t.Fatalf("guard section: %v", err)
	}
	if len(guard.Branches) != 1 || guard.Branches[0] != "main" {
		t.Errorf("guard.branches = %v, want the topology preserved", guard.Branches)
	}
	if guard.Mode == "" {
		t.Error("guard.mode missing; pk setup did not field-merge its modes in")
	}
	for _, rel := range []string{"CLAUDE.md", ".claude/settings.json"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("%s not created by the upgrade", rel)
		}
	}
}

func TestRun_explicitReleaseFlagBeatsConfig(t *testing.T) {
	g := &fakeGit{branch: "trunk"}
	dir, _, cfg := newRepo(t, g)
	if err := os.WriteFile(filepath.Join(dir, ".pk.json"), []byte(`{"release":{"branch":"main"}}`), 0644); err != nil {
		t.Fatalf("seed .pk.json: %v", err)
	}
	cfg.ReleaseBranch = "trunk"

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	pk := readPkConfig(t, dir)
	if !strings.Contains(string(pk["release"]), `"trunk"`) {
		t.Errorf("release = %s, want the --release flag to win", pk["release"])
	}
}
