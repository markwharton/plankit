package guard

import (
	"bytes"
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
			return []byte(`{"guard":{"protectedBranches":["main"]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "main\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), `"decision":"block"`) {
		t.Errorf("stdout = %q, want block decision", stdout.String())
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
			return []byte(`{"guard":{"protectedBranches":["main"]}}`), nil
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
			return []byte(`{"guard":{"protectedBranches":["main"]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "main\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), `"decision":"block"`) {
		t.Errorf("stdout = %q, want block decision", stdout.String())
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
					return []byte(`{"guard":{"protectedBranches":["main"]}}`), nil
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
			return []byte(`{"guard":{"protectedBranches":[]}}`), nil
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
			return []byte(`{"guard":{"protectedBranches":["main","production"]}}`), nil
		},
		GitExec: func(dir string, args ...string) (string, error) {
			return "production\n", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), `"decision":"block"`) {
		t.Errorf("stdout = %q, want block for production branch", stdout.String())
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
		// Compound commands.
		{"git checkout main && git merge dev", true},
		{"git status && git commit -m 'test'", true},
		{"git log; git push origin main", true},
		{"git checkout main || git merge dev", true},
		{"git status && git log", false},
		{"echo hello && ls -la", false},
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

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }
