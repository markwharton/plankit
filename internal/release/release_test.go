package release

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// stubGitExec returns a GitExec function that dispatches based on the first arg.
func stubGitExec(handlers map[string]func(args ...string) (string, error)) func(args ...string) (string, error) {
	return func(args ...string) (string, error) {
		if h, ok := handlers[args[0]]; ok {
			return h(args...)
		}
		return "", nil
	}
}

// happyGit returns git stubs for a clean, valid release state.
func happyGit(tag, branch string) map[string]func(args ...string) (string, error) {
	return map[string]func(args ...string) (string, error){
		"tag": func(args ...string) (string, error) {
			return tag, nil
		},
		"status": func(args ...string) (string, error) {
			return "", nil
		},
		"branch": func(args ...string) (string, error) {
			return branch, nil
		},
		"fetch": func(args ...string) (string, error) {
			return "", nil
		},
		"merge-base": func(args ...string) (string, error) {
			return "abc123", nil
		},
		"rev-parse": func(args ...string) (string, error) {
			return "abc123", nil
		},
		"push": func(args ...string) (string, error) {
			return "", nil
		},
	}
}

func noConfig(_ string) ([]byte, error) {
	return nil, os.ErrNotExist
}

func TestRun_happyPath(t *testing.T) {
	var stderr bytes.Buffer
	var pushArgs []string

	git := happyGit("v1.2.3", "main")
	git["push"] = func(args ...string) (string, error) {
		pushArgs = args
		return "", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
		Branch:   "main",
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if len(pushArgs) == 0 {
		t.Fatal("push was not called")
	}
	// push origin main v1.2.3
	if pushArgs[1] != "origin" || pushArgs[2] != "main" || pushArgs[3] != "v1.2.3" {
		t.Errorf("push args = %v, want [push origin main v1.2.3]", pushArgs)
	}
	if !strings.Contains(stderr.String(), "Pushed main and v1.2.3") {
		t.Errorf("stderr missing push confirmation: %s", stderr.String())
	}
}

func TestRun_dryRun(t *testing.T) {
	var stderr bytes.Buffer
	pushCalled := false

	git := happyGit("v1.0.0", "main")
	git["push"] = func(args ...string) (string, error) {
		pushCalled = true
		return "", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
		Branch:   "main",
		DryRun:   true,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if pushCalled {
		t.Error("push should not be called in dry run")
	}
	if !strings.Contains(stderr.String(), "Dry run complete") {
		t.Errorf("stderr missing dry run message: %s", stderr.String())
	}
}

func TestRun_noTagAtHead(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("", "main")

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
		Branch:   "main",
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no version tag at HEAD") {
		t.Errorf("stderr = %q, want no version tag message", stderr.String())
	}
}

func TestRun_invalidSemver(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("v1.2", "main")

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
		Branch:   "main",
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "not valid semver") {
		t.Errorf("stderr = %q, want semver error", stderr.String())
	}
}

func TestRun_dirtyWorkingTree(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("v1.0.0", "main")
	git["status"] = func(args ...string) (string, error) {
		return " M dirty.go", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
		Branch:   "main",
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "working tree is not clean") {
		t.Errorf("stderr = %q, want dirty tree message", stderr.String())
	}
}

func TestRun_wrongBranch(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("v1.0.0", "feature-branch")

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
		Branch:   "main",
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "not on main branch") {
		t.Errorf("stderr = %q, want wrong branch message", stderr.String())
	}
}

func TestRun_behindRemote(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("v1.0.0", "main")
	git["merge-base"] = func(args ...string) (string, error) {
		return "local123", nil
	}
	git["rev-parse"] = func(args ...string) (string, error) {
		return "remote456", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
		Branch:   "main",
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "behind origin/main") {
		t.Errorf("stderr = %q, want behind remote message", stderr.String())
	}
}

func TestRun_preReleaseHook(t *testing.T) {
	var stderr bytes.Buffer
	var hookCommand string

	cfg := Config{
		Stderr:  &stderr,
		GitExec: stubGitExec(happyGit("v1.0.0", "main")),
		ReadFile: func(name string) ([]byte, error) {
			if name == ".changelog.json" {
				return []byte(`{"hooks":{"preRelease":"echo test"}}`), nil
			}
			return nil, os.ErrNotExist
		},
		RunScript: func(command string) error {
			hookCommand = command
			return nil
		},
		Branch: "main",
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if hookCommand != "echo test" {
		t.Errorf("hook command = %q, want %q", hookCommand, "echo test")
	}
}

func TestRun_preReleaseHookFailure(t *testing.T) {
	var stderr bytes.Buffer

	cfg := Config{
		Stderr:  &stderr,
		GitExec: stubGitExec(happyGit("v1.0.0", "main")),
		ReadFile: func(name string) ([]byte, error) {
			if name == ".changelog.json" {
				return []byte(`{"hooks":{"preRelease":"false"}}`), nil
			}
			return nil, os.ErrNotExist
		},
		RunScript: func(command string) error {
			return fmt.Errorf("exit status 1")
		},
		Branch: "main",
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "pre-release hook failed") {
		t.Errorf("stderr = %q, want hook failure message", stderr.String())
	}
}

func TestRun_pushFailure(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("v1.0.0", "main")
	git["push"] = func(args ...string) (string, error) {
		return "", fmt.Errorf("permission denied")
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
		Branch:   "main",
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "git push failed") {
		t.Errorf("stderr = %q, want push failure message", stderr.String())
	}
}

func TestRun_customBranch(t *testing.T) {
	var stderr bytes.Buffer
	var pushArgs []string

	git := happyGit("v1.0.0", "develop")
	git["push"] = func(args ...string) (string, error) {
		pushArgs = args
		return "", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
		Branch:   "develop",
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if pushArgs[2] != "develop" {
		t.Errorf("push branch = %q, want develop", pushArgs[2])
	}
}

func TestFindVersionTag(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{"single tag", "v1.0.0", "v1.0.0"},
		{"multiple tags", "v1.0.0\nv0.9.0", "v1.0.0"},
		{"no version tag", "some-tag", ""},
		{"empty", "", ""},
		{"mixed tags", "release-1\nv2.0.0\nother", "v2.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findVersionTag(tt.output)
			if got != tt.want {
				t.Errorf("findVersionTag(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}
