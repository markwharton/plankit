package guard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/markwharton/plankit/internal/hookio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
)

func TestGitSubcommand(t *testing.T) {
	cases := []struct{ cmd, want string }{
		{"git commit -m x", "commit"},
		{"git push origin main", "push"},
		{"/usr/bin/git push", "push"},
		{"GIT_DIR=. git push", "push"},
		{"FOO=1 BAR=2 git rebase main", "rebase"},
		{"command git merge dev", "merge"},
		{"git -C /tmp push", "push"},
		{"git -c user.name=x commit", "commit"},
		{"git --git-dir=.git reset --hard", "reset"},
		{"git.exe push", "push"},
		{"C:/Program/git.exe push", "push"},
		{`C:\tools\git.exe commit -m x`, "commit"},
		{"git status", "status"},
		{"git", ""},
		{"gitk", ""},
		{"echo git push", ""},
		{"1FOO=x git push", ""}, // not a valid env assignment: not git
	}
	for _, tc := range cases {
		if got := gitSubcommand(tc.cmd); got != tc.want {
			t.Errorf("gitSubcommand(%q) = %q, want %q", tc.cmd, got, tc.want)
		}
	}
}

func TestIsGitMutationAcrossChains(t *testing.T) {
	yes := []string{
		"git commit -m x",
		"git add . && git commit -m x",
		"echo hi; git rebase main",
		"true || git merge dev",
		"git fetch | cat\ngit reset --hard",
		"make build |& tee log && git push",
	}
	no := []string{
		"git status && git log",
		`echo "a && git push"`,
		"echo 'git commit -m x'",
		"git checkout -b feature",
		"git pull --rebase=false",
		"ls | grep git",
	}
	for _, c := range yes {
		if !isGitMutation(c) {
			t.Errorf("isGitMutation(%q) = false, want true", c)
		}
	}
	for _, c := range no {
		if isGitMutation(c) {
			t.Errorf("isGitMutation(%q) = true, want false", c)
		}
	}
}

// scratch builds a configured repo on branch main and returns its path.
func scratch(t *testing.T, mode, push string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "t@t"}, {"config", "user.name", "t"},
	} {
		if _, err := git.Exec(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644)
	git.Exec(dir, "add", ".")
	git.Exec(dir, "commit", "-q", "-m", "first")
	cfg := config.Default("main")
	cfg.Guard.Mode = mode
	cfg.Guard.Push = push
	if err := config.Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	return dir
}

func runGuard(t *testing.T, dir, command string) (string, string) {
	t.Helper()
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	payload, _ := json.Marshal(map[string]any{
		"cwd":        dir,
		"tool_input": map[string]string{"command": command},
	})
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "guard"}, []*cli.Command{Cmd}, bytes.NewReader(payload), &out, &errw)
	if code != 0 {
		t.Fatalf("hook exit %d, must always be 0 (stderr: %s)", code, errw.String())
	}
	return out.String(), errw.String()
}

func TestBlocksMutationOnProtectedBranch(t *testing.T) {
	dir := scratch(t, "block", "off")
	out, _ := runGuard(t, dir, "git commit -m x")
	if !strings.Contains(out, `"permissionDecision":"deny"`) || !strings.Contains(out, `Branch \"main\" is protected`) {
		t.Fatalf("out = %s", out)
	}
}

func TestAskModeAsks(t *testing.T) {
	dir := scratch(t, "ask", "off")
	out, _ := runGuard(t, dir, "git merge dev")
	if !strings.Contains(out, `"permissionDecision":"ask"`) {
		t.Fatalf("out = %s", out)
	}
}

func TestOffBranchButPushPolicyStillFires(t *testing.T) {
	dir := scratch(t, "block", "block")
	git.Exec(dir, "checkout", "-q", "-b", "feature")
	out, _ := runGuard(t, dir, "git push origin feature")
	if !strings.Contains(out, `"deny"`) || !strings.Contains(out, "push blocked") {
		t.Fatalf("push policy off-branch: %s", out)
	}
}

func TestStrongestDecisionWins(t *testing.T) {
	// push:ask would ask, but the branch policy denies: deny wins.
	dir := scratch(t, "block", "ask")
	out, _ := runGuard(t, dir, "git push")
	if !strings.Contains(out, `"deny"`) || !strings.Contains(out, "is protected") {
		t.Fatalf("deny should win over ask: %s", out)
	}
}

func TestEverythingOffIsSilent(t *testing.T) {
	dir := scratch(t, "off", "off")
	out, _ := runGuard(t, dir, "git push && git commit -m x")
	if out != "" {
		t.Fatalf("out = %s, want empty", out)
	}
}

func TestNonMutationIsSilent(t *testing.T) {
	dir := scratch(t, "block", "block")
	out, _ := runGuard(t, dir, "git status")
	if out != "" {
		t.Fatalf("out = %s", out)
	}
}

func TestUnconfiguredRepoIsSilent(t *testing.T) {
	dir := scratch(t, "block", "block")
	os.Remove(config.Path(dir))
	out, _ := runGuard(t, dir, "git commit -m x")
	if out != "" {
		t.Fatalf("unconfigured must no-op: %s", out)
	}
}

func TestMalformedPayloadFailsOpen(t *testing.T) {
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "guard"}, []*cli.Command{Cmd}, strings.NewReader("not json"), &out, &errw)
	if code != 0 || out.Len() != 0 || !strings.Contains(errw.String(), "pk guard:") {
		t.Fatalf("code=%d out=%q errw=%q", code, out.String(), errw.String())
	}
}

// TestBadConfigIsReportedNotBlocking: a policy file that fails to load
// exits 1 with the message naming the key, which Claude Code shows to
// the person and then continues; guard makes no decision.
func TestBadConfigIsReportedNotBlocking(t *testing.T) {
	dir := scratch(t, "block", "block")
	os.WriteFile(config.Path(dir), []byte(`{"guard": {"mode": "blok"}}`), 0o644)
	payload, _ := json.Marshal(map[string]any{"cwd": dir, "tool_input": map[string]string{"command": "git commit -m x"}})
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "guard"}, []*cli.Command{Cmd}, bytes.NewReader(payload), &out, &errw)
	if code != hookio.ExitReport || out.Len() != 0 || !strings.Contains(errw.String(), "guard.mode") {
		t.Fatalf("code=%d out=%q errw=%q", code, out.String(), errw.String())
	}
}

func ExampleCmd() {
	fmt.Println(Cmd.Name)
	// Output: guard
}

// scratchOn is scratch on an arbitrary branch, with a config mutator.
func scratchOn(t *testing.T, branch string, mutate func(*config.PkConfig)) string {
	t.Helper()
	dir := scratch(t, "block", "block")
	if branch != "main" {
		if _, err := git.Exec(dir, "switch", "-q", "-c", branch); err != nil {
			t.Fatal(err)
		}
	}
	if mutate != nil {
		cfg, err := config.Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		mutate(cfg)
		if err := config.Write(dir, cfg); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestBreakingMarkerAsks(t *testing.T) {
	cases := []struct {
		name    string
		command string
		wantAsk bool
	}{
		{"bang subject", `git commit -m "feat!: drop the v1 keys"`, true},
		{"bang with scope", `git commit -m 'refactor(docgen)!: rewrite'`, true},
		{"breaking footer in body", `git commit -m "feat: add x" -m "BREAKING CHANGE: y is gone"`, true},
		{"breaking hyphen footer", `git commit -m 'fix: z' -m 'BREAKING-CHANGE: q'`, true},
		{"message equals form", `git commit --message="feat!: x"`, true},
		{"attached short form", `git commit -m"feat!: x"`, true},
		{"cluster short form", `git commit -am "feat!: x"`, true},
		{"amend keeps the check", `git commit --amend -m "feat!: x"`, true},
		{"plain feat", `git commit -m "feat: add x"`, false},
		{"bang not in marker position", `git commit -m "feat: use x! carefully"`, false},
		{"breaking words mid-line", `git commit -m "docs: explain the BREAKING CHANGE: footer"`, false},
		{"no inline message", `git commit -F msg.txt`, false},
		{"chained after safe command", `go test ./... && git commit -m "chore!: retire it"`, true},
	}
	for _, tc := range cases {
		dir := scratchOn(t, "develop", nil)
		out, _ := runGuard(t, dir, tc.command)
		if tc.wantAsk {
			if !strings.Contains(out, `"ask"`) || !strings.Contains(out, "breaking change") {
				t.Errorf("%s: want ask, got %q", tc.name, out)
			}
		} else if out != "" {
			t.Errorf("%s: want silent allow, got %q", tc.name, out)
		}
	}
}

func TestBreakingMarkerPrecedenceAndDial(t *testing.T) {
	// On a protected branch, the branch deny wins over the breaking ask.
	dir := scratchOn(t, "main", nil)
	out, _ := runGuard(t, dir, `git commit -m "feat!: x"`)
	if !strings.Contains(out, `"deny"`) {
		t.Fatalf("branch deny should win: %q", out)
	}

	// breaking: off restores silence for markers.
	dir = scratchOn(t, "develop", func(c *config.PkConfig) { c.Guard.Breaking = "off" })
	out, _ = runGuard(t, dir, `git commit -m "feat!: x"`)
	if out != "" {
		t.Fatalf("breaking off: want silence, got %q", out)
	}
}

func TestShellWords(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`git commit -m "feat!: a b"`, []string{"git", "commit", "-m", "feat!: a b"}},
		{`git commit -m 'one two'`, []string{"git", "commit", "-m", "one two"}},
		{`git commit -m "say \"hi\""`, []string{"git", "commit", "-m", `say "hi"`}},
		{"git commit -m \"line one\nline two\"", []string{"git", "commit", "-m", "line one\nline two"}},
	}
	for _, tc := range cases {
		got := shellWords(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("shellWords(%q) = %q, want %q", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("shellWords(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}
