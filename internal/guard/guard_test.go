package guard

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestRun_blocksCommitOnProtectedBranch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"guard":{"branches":["main"]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "main\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), `"permissionDecision":"deny"`) {
		t.Errorf("stdout = %q, want permissionDecision=deny", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"hookEventName":"PreToolUse"`) {
		t.Errorf("stdout = %q, want hookEventName=PreToolUse", stdout.String())
	}
	if !strings.Contains(stdout.String(), "protected") {
		t.Errorf("stdout = %q, want reason mentioning protected", stdout.String())
	}
}

func TestRun_allowsCommitOnUnprotectedBranch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"guard":{"branches":["main"]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "dev\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout.Len() > 0 {
		t.Errorf("stdout = %q, want empty (no block)", stdout.String())
	}
}

func TestRun_blocksPushOnProtectedBranch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git push origin main"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"guard":{"branches":["main"]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "main\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), `"permissionDecision":"deny"`) {
		t.Errorf("stdout = %q, want permissionDecision=deny", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"hookEventName":"PreToolUse"`) {
		t.Errorf("stdout = %q, want hookEventName=PreToolUse", stdout.String())
	}
}

func TestRun_allowsReadOnlyGitCommands(t *testing.T) {
	commands := []string{
		"git status",
		"git log --oneline -5",
		"git diff",
		"git branch -a",
		"git fetch origin",
		"ls -la",
		"echo hello",
	}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			input := `{"tool_input":{"command":"` + cmd + `"},"cwd":"/project"}`
			cfg := Config{
				Stdin:  strings.NewReader(input),
				Stdout: &stdout,
				Stderr: &stderr,
				Env:    func(string) string { return "" },
				ReadFile: func(name string) ([]byte, error) {
					return []byte(`{"guard":{"branches":["main"]}}`), nil
				},
				GitExec: func(dir string, args ...string) (string, error) {
					return "main\n", nil
				},
			}

			Run(cfg)
			if stdout.Len() > 0 {
				t.Errorf("stdout = %q, want empty (no block for %q)", stdout.String(), cmd)
			}
		})
	}
}

func TestRun_noGuardConfigIsNoOp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"changelog":{"types":[]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "main\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout.Len() > 0 {
		t.Errorf("stdout = %q, want empty (no guard config)", stdout.String())
	}
}

func TestRun_emptyGuardConfigIsNoOp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"guard":{"branches":[]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "main\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout.Len() > 0 {
		t.Errorf("stdout = %q, want empty (empty protected list)", stdout.String())
	}
}

func TestRun_noPkJsonIsNoOp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return nil, &testError{msg: "file not found"}
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "main\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout.Len() > 0 {
		t.Errorf("stdout = %q, want empty (no .pk.json)", stdout.String())
	}
}

func TestRun_multipleProtectedBranches(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"guard":{"branches":["main","production"]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "production\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), `"permissionDecision":"deny"`) {
		t.Errorf("stdout = %q, want permissionDecision=deny for production branch", stdout.String())
	}
}

func TestRun_askModePromptsUser(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"guard":{"branches":["main"]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "main\n", nil
		},
		Ask: true,
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), `"permissionDecision":"ask"`) {
		t.Errorf("stdout = %q, want permissionDecision=ask", stdout.String())
	}
	if !strings.Contains(stdout.String(), "emergency hotfix") {
		t.Errorf("stdout = %q, want reason mentioning emergency hotfix", stdout.String())
	}
}

func TestIsGitMutation(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"git commit -m 'test'", true},
		{"git push origin main", true},
		{"git merge dev", true},
		{"git rebase main", true},
		{"git reset --hard HEAD~1", true},
		{"git reset --soft HEAD~1", true},
		{"git reset", true},
		{"git push --force origin main", true},
		{"git push -f origin main", true},
		{"git status", false},
		{"git log", false},
		{"git diff", false},
		{"git branch -a", false},
		{"git fetch origin", false},
		{"echo hello", false},
		{"ls -la", false},
		{"git commit", true},
		{"git push", true},
		// Global options before the subcommand must not evade detection (hardening).
		{"git -C /tmp/x push", true},
		{"git -c user.name=x commit -m y", true},
		{"git --git-dir=.git push origin main", true},
		{"git -C /tmp/x status", false},
		// Compound commands.
		{"git checkout main && git merge dev", true},
		{"git status && git commit -m 'test'", true},
		{"git log; git push origin main", true},
		{"git checkout main || git merge dev", true},
		{"git status && git log", false},
		{"echo hello && ls -la", false},
		// Quoted operators should not split.
		{`git commit -m "a && b"`, true},
		{`git commit -m 'a || b; c'`, true},
		{`echo "hello" && git push`, true},
		{`echo "hello; world"`, false},
		{`git commit -m "fix: a && b || c"`, true},
		{`echo 'no && split' && echo 'no || split'`, false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := isGitMutation(tt.command)
			if got != tt.want {
				t.Errorf("isGitMutation(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestRun_malformedPkJsonLogsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{not json}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "main\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0 (hooks always exit 0)", code)
	}
	if !strings.Contains(stderr.String(), "failed to parse .pk.json") {
		t.Errorf("stderr = %q, want parse error message", stderr.String())
	}
	if stdout.Len() > 0 {
		t.Errorf("stdout = %q, want empty (no block on parse error)", stdout.String())
	}
}

func TestRun_revParseFailureAllowsThrough(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:  strings.NewReader(`{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/project"}`),
		Stdout: &stdout,
		Stderr: &stderr,
		Env:    func(string) string { return "" },
		ReadFile: func(name string) ([]byte, error) {
			return []byte(`{"guard":{"branches":["main"]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "", fmt.Errorf("fatal: not a git repository")
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if stdout.Len() > 0 {
		t.Errorf("stdout = %q, want empty (should not block when rev-parse fails)", stdout.String())
	}
}

// decision runs guard for a single command and returns its stdout (the permission
// decision JSON, or "" when allowed). Command must not contain double quotes.
func decision(t *testing.T, command, pkjson, branch string, ask bool, pushGuard string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cfg := Config{
		Stdin:     strings.NewReader(`{"tool_input":{"command":"` + command + `"},"cwd":"/project"}`),
		Stdout:    &stdout,
		Stderr:    &stderr,
		Env:       func(string) string { return "" },
		ReadFile:  func(string) ([]byte, error) { return []byte(pkjson), nil },
		GitExec:   func(string, ...string) (string, error) { return branch + "\n", nil },
		Ask:       ask,
		PushGuard: pushGuard,
	}
	Run(cfg)
	return stdout.String()
}

const guardMainOnly = `{"guard":{"branches":["main"]}}`

func TestRun_pushGuardBlocksPushOnUnprotectedBranch(t *testing.T) {
	out := decision(t, "git push origin feature", guardMainOnly, "feature", false, "block")
	if !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Errorf("out = %q, want deny", out)
	}
}

func TestRun_pushGuardAsksOnUnprotectedBranch(t *testing.T) {
	out := decision(t, "git push", guardMainOnly, "feature", false, "ask")
	if !strings.Contains(out, `"permissionDecision":"ask"`) {
		t.Errorf("out = %q, want ask", out)
	}
}

func TestRun_pushGuardOffAllowsPush(t *testing.T) {
	if out := decision(t, "git push", guardMainOnly, "feature", false, "off"); out != "" {
		t.Errorf("out = %q, want empty (push-guard off)", out)
	}
}

func TestRun_pushGuardIgnoresCommit(t *testing.T) {
	if out := decision(t, "git commit -m x", guardMainOnly, "feature", false, "block"); out != "" {
		t.Errorf("out = %q, want empty (push policy must not touch commit)", out)
	}
}

func TestRun_pushGuardDetectsDashCPush(t *testing.T) {
	out := decision(t, "git -C sub push", guardMainOnly, "feature", false, "block")
	if !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Errorf("out = %q, want deny (git -C push hardening)", out)
	}
}

func TestRun_protectedPushIsStrongest(t *testing.T) {
	// Branch ask + push block on a protected-branch push: strongest (deny) wins.
	out := decision(t, "git push", guardMainOnly, "main", true, "block")
	if !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Errorf("out = %q, want deny (strongest of branch-ask and push-block)", out)
	}
}

func TestIsGitPush(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"git push", true},
		{"git push origin main", true},
		{"git -C dir push", true},
		{"git commit -m x && git push", true},
		{"git commit", false},
		{"git status", false},
		{"echo git push", false},
	}
	for _, tt := range tests {
		if got := isGitPush(tt.cmd); got != tt.want {
			t.Errorf("isGitPush(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }
