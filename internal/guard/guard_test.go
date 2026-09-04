package guard

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestBadConfigFailsOpen(t *testing.T) {
	dir := scratch(t, "block", "block")
	os.WriteFile(config.Path(dir), []byte(`{"guard": {"mode": "blok"}}`), 0o644)
	out, errw := runGuard(t, dir, "git commit -m x")
	if out != "" || !strings.Contains(errw, "guard.mode") {
		t.Fatalf("out=%q errw=%q", out, errw)
	}
}

func ExampleCmd() {
	fmt.Println(Cmd.Name)
	// Output: guard
}
