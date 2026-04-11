package release

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// stubGitExec returns a GitExec function that dispatches based on the first non-dir arg.
func stubGitExec(handlers map[string]func(args ...string) (string, error)) func(dir string, args ...string) (string, error) {
	return func(dir string, args ...string) (string, error) {
		if h, ok := handlers[args[0]]; ok {
			return h(args...)
		}
		return "", nil
	}
}

// trailerFormat matches the git log --format used to extract Release-Tag.
const trailerFormat = "--format=%(trailers:key=Release-Tag,valueonly)"

// happyGit returns git stubs for a clean, valid legacy release state.
// tag is the value that will be returned for the Release-Tag trailer lookup.
// An empty tag means "no trailer present" (missing trailer error).
func happyGit(tag, branch string) map[string]func(args ...string) (string, error) {
	return map[string]func(args ...string) (string, error){
		"log": func(args ...string) (string, error) {
			// Expect: log -1 --format=%(trailers:key=Release-Tag,valueonly) HEAD
			return tag, nil
		},
		"tag": func(args ...string) (string, error) {
			// Expect: tag --list <tag>, tag <tag>, tag -d <tag>
			// Default: tag doesn't already exist, create/delete succeed.
			return "", nil
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

// happyGitMerge returns git stubs for a merge flow release state.
func happyGitMerge(tag, sourceBranch, releaseBranch string) map[string]func(args ...string) (string, error) {
	currentBranch := sourceBranch
	return map[string]func(args ...string) (string, error){
		"log": func(args ...string) (string, error) {
			return tag, nil
		},
		"tag": func(args ...string) (string, error) {
			return "", nil
		},
		"status": func(args ...string) (string, error) {
			return "", nil
		},
		"branch": func(args ...string) (string, error) {
			return currentBranch, nil
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
		"switch": func(args ...string) (string, error) {
			currentBranch = args[1]
			return "", nil
		},
		"merge": func(args ...string) (string, error) {
			return "", nil
		},
		"push": func(args ...string) (string, error) {
			return "", nil
		},
	}
}

func noConfig(_ string) ([]byte, error) {
	return nil, os.ErrNotExist
}

func mergeConfig(releaseBranch string) func(string) ([]byte, error) {
	return func(name string) ([]byte, error) {
		if name == ".pk.json" {
			return []byte(fmt.Sprintf(`{"release":{"branch":%q}}`, releaseBranch)), nil
		}
		return nil, os.ErrNotExist
	}
}

// --- Legacy flow tests (no release.branch configured) ---

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

func TestRun_missingTrailer(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("", "main")

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no Release-Tag trailer on HEAD") {
		t.Errorf("stderr = %q, want missing trailer message", stderr.String())
	}
}

func TestRun_invalidTrailerValue(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("v1.2", "main") // not valid semver

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "not valid semver") {
		t.Errorf("stderr = %q, want invalid trailer message", stderr.String())
	}
}

func TestRun_tagAlreadyExists(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("v1.2.3", "main")
	git["tag"] = func(args ...string) (string, error) {
		// tag --list v1.2.3 returns the existing tag.
		if len(args) >= 2 && args[1] == "--list" {
			return "v1.2.3", nil
		}
		return "", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "already exists locally") {
		t.Errorf("stderr = %q, want already-exists message", stderr.String())
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
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "working tree is not clean") {
		t.Errorf("stderr = %q, want dirty tree message", stderr.String())
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
			if name == ".pk.json" {
				return []byte(`{"release":{"hooks":{"preRelease":"echo test"}}}`), nil
			}
			return nil, os.ErrNotExist
		},
		RunScript: func(command string, env map[string]string) error {
			hookCommand = command
			return nil
		},
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
			if name == ".pk.json" {
				return []byte(`{"release":{"hooks":{"preRelease":"false"}}}`), nil
			}
			return nil, os.ErrNotExist
		},
		RunScript: func(command string, env map[string]string) error {
			return fmt.Errorf("exit status 1")
		},
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
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "git push failed") {
		t.Errorf("stderr = %q, want push failure message", stderr.String())
	}
}

// --- Merge flow tests (release.branch configured) ---

func TestRun_mergeFlow_happyPath(t *testing.T) {
	var stderr bytes.Buffer
	var pushCalls [][]string

	git := happyGitMerge("v1.2.3", "dev", "main")
	git["push"] = func(args ...string) (string, error) {
		pushCalls = append(pushCalls, args)
		return "", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if len(pushCalls) != 2 {
		t.Fatalf("push called %d times, want 2", len(pushCalls))
	}
	// First push: release branch + tag
	if pushCalls[0][2] != "main" || pushCalls[0][3] != "v1.2.3" {
		t.Errorf("first push = %v, want [push origin main v1.2.3]", pushCalls[0])
	}
	// Second push: source branch
	if pushCalls[1][2] != "dev" {
		t.Errorf("second push = %v, want [push origin dev]", pushCalls[1])
	}
	if !strings.Contains(stderr.String(), "Merged dev into main") {
		t.Errorf("stderr missing merge confirmation: %s", stderr.String())
	}
}

func TestRun_mergeFlow_missingTrailer(t *testing.T) {
	var stderr bytes.Buffer

	git := happyGitMerge("", "dev", "main")

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no Release-Tag trailer on HEAD") {
		t.Errorf("stderr = %q, want missing-trailer message", stderr.String())
	}
}

func TestRun_tagCleanupOnPushFailure(t *testing.T) {
	var stderr bytes.Buffer
	var tagCalls [][]string

	git := happyGit("v1.2.3", "main")
	git["tag"] = func(args ...string) (string, error) {
		tagCalls = append(tagCalls, args)
		// tag --list v1.2.3: return empty (not yet created).
		if len(args) >= 2 && args[1] == "--list" {
			return "", nil
		}
		return "", nil
	}
	git["push"] = func(args ...string) (string, error) {
		return "", fmt.Errorf("permission denied")
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: noConfig,
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}

	// Verify: tag v1.2.3 was created, then tag -d v1.2.3 was called for cleanup.
	var created, deleted bool
	for _, call := range tagCalls {
		if len(call) == 2 && call[0] == "tag" && call[1] == "v1.2.3" {
			created = true
		}
		if len(call) == 3 && call[0] == "tag" && call[1] == "-d" && call[2] == "v1.2.3" {
			deleted = true
		}
	}
	if !created {
		t.Error("expected tag v1.2.3 to be created")
	}
	if !deleted {
		t.Error("expected tag v1.2.3 to be cleaned up on push failure")
	}
}

func TestRun_mergeFlow_dryRun(t *testing.T) {
	var stderr bytes.Buffer
	pushCalled := false
	switchCalled := false

	git := happyGitMerge("v1.0.0", "dev", "main")
	git["push"] = func(args ...string) (string, error) {
		pushCalled = true
		return "", nil
	}
	git["switch"] = func(args ...string) (string, error) {
		switchCalled = true
		return "", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
		DryRun:   true,
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if pushCalled {
		t.Error("push should not be called in dry run")
	}
	if switchCalled {
		t.Error("switch should not be called in dry run")
	}
	if !strings.Contains(stderr.String(), "Would merge dev into main") {
		t.Errorf("stderr missing merge preview: %s", stderr.String())
	}
}

func TestRun_mergeFlow_alreadyOnReleaseBranch(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGit("v1.0.0", "main")

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "switch to your development branch first") {
		t.Errorf("stderr = %q, want development branch message", stderr.String())
	}
}

func TestRun_mergeFlow_mergeFails(t *testing.T) {
	var stderr bytes.Buffer
	switchedBack := false

	git := happyGitMerge("v1.0.0", "dev", "main")
	git["merge"] = func(args ...string) (string, error) {
		return "", fmt.Errorf("not fast-forward")
	}
	originalSwitch := git["switch"]
	git["switch"] = func(args ...string) (string, error) {
		if args[1] == "dev" {
			switchedBack = true
		}
		return originalSwitch(args...)
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "merge failed") {
		t.Errorf("stderr = %q, want merge failure message", stderr.String())
	}
	if !switchedBack {
		t.Error("should switch back to source branch after merge failure")
	}
}

func TestRun_mergeFlow_pushFails_switchesBack(t *testing.T) {
	var stderr bytes.Buffer
	switchedBack := false

	git := happyGitMerge("v1.0.0", "dev", "main")
	git["push"] = func(args ...string) (string, error) {
		return "", fmt.Errorf("permission denied")
	}
	originalSwitch := git["switch"]
	git["switch"] = func(args ...string) (string, error) {
		if args[1] == "dev" {
			switchedBack = true
		}
		return originalSwitch(args...)
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !switchedBack {
		t.Error("should switch back to source branch after push failure")
	}
}

func TestRun_mergeFlow_dirtyTree(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGitMerge("v1.0.0", "dev", "main")
	git["status"] = func(args ...string) (string, error) {
		return " M dirty.go", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "working tree is not clean") {
		t.Errorf("stderr = %q, want dirty tree message", stderr.String())
	}
}

func TestRun_mergeFlow_sourceBehindRemote(t *testing.T) {
	var stderr bytes.Buffer
	git := happyGitMerge("v1.0.0", "dev", "main")
	git["merge-base"] = func(args ...string) (string, error) {
		// For the source branch behind-remote check
		if len(args) >= 3 && args[2] == "origin/dev" {
			return "local123", nil
		}
		return "abc123", nil
	}
	git["rev-parse"] = func(args ...string) (string, error) {
		if len(args) >= 2 && args[1] == "origin/dev" {
			return "remote456", nil
		}
		return "abc123", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "behind origin/dev") {
		t.Errorf("stderr = %q, want behind remote message", stderr.String())
	}
}

func TestRun_legacyFlow_noReleaseBranch(t *testing.T) {
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
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if pushArgs[2] != "develop" {
		t.Errorf("push branch = %q, want develop", pushArgs[2])
	}
}

// --- PR flow tests ---

func TestRun_pr_happyPath(t *testing.T) {
	var stderr bytes.Buffer
	var pushArgs []string
	var ghArgs []string

	git := happyGitMerge("v1.0.0", "dev", "main")
	git["push"] = func(args ...string) (string, error) {
		pushArgs = args
		return "", nil
	}
	git["remote"] = func(args ...string) (string, error) {
		return "git@github.com:owner/repo.git", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
		PR:       true,
		ExecGh: func(args ...string) (string, error) {
			ghArgs = args
			return "https://github.com/owner/repo/pull/42", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	// Push should target source branch + tag, not release branch.
	if pushArgs[1] != "origin" || pushArgs[2] != "dev" || pushArgs[3] != "v1.0.0" {
		t.Errorf("push args = %v, want [push origin dev v1.0.0]", pushArgs)
	}
	// gh should be called with correct args.
	if len(ghArgs) < 8 || ghArgs[0] != "pr" || ghArgs[3] != "main" || ghArgs[5] != "dev" {
		t.Errorf("gh args = %v, want pr create --base main --head dev ...", ghArgs)
	}
	if !strings.Contains(stderr.String(), "github.com/owner/repo/pull/42") {
		t.Errorf("stderr missing PR URL: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Release v1.0.0 pushed") {
		t.Errorf("stderr missing release pushed message: %s", stderr.String())
	}
}

func TestRun_pr_ghFails_compareURL(t *testing.T) {
	var stderr bytes.Buffer

	git := happyGitMerge("v1.0.0", "dev", "main")
	git["remote"] = func(args ...string) (string, error) {
		return "git@github.com:owner/repo.git", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
		PR:       true,
		ExecGh: func(args ...string) (string, error) {
			return "", fmt.Errorf("gh: command not found")
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (gh failure is not fatal); stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "github.com/owner/repo/compare/main...dev") {
		t.Errorf("stderr missing compare URL: %s", stderr.String())
	}
}

func TestRun_pr_noReleaseBranch(t *testing.T) {
	var stderr bytes.Buffer

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(happyGit("v1.0.0", "dev")),
		ReadFile: noConfig,
		PR:       true,
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--pr requires release.branch") {
		t.Errorf("stderr = %q, want release.branch required message", stderr.String())
	}
}

func TestRun_pr_dryRun(t *testing.T) {
	var stderr bytes.Buffer
	pushCalled := false
	ghCalled := false

	git := happyGitMerge("v1.0.0", "dev", "main")
	git["push"] = func(args ...string) (string, error) {
		pushCalled = true
		return "", nil
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
		PR:       true,
		DryRun:   true,
		ExecGh: func(args ...string) (string, error) {
			ghCalled = true
			return "", nil
		},
	}

	code := Run(cfg)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr.String())
	}
	if pushCalled {
		t.Error("push should not be called in dry run")
	}
	if ghCalled {
		t.Error("gh should not be called in dry run")
	}
	if !strings.Contains(stderr.String(), "Would push dev and v1.0.0") {
		t.Errorf("stderr missing push preview: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Would create PR: dev → main") {
		t.Errorf("stderr missing PR preview: %s", stderr.String())
	}
}

func TestRun_pr_missingTrailer(t *testing.T) {
	var stderr bytes.Buffer

	git := happyGitMerge("", "dev", "main")

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
		PR:       true,
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no Release-Tag trailer on HEAD") {
		t.Errorf("stderr = %q, want missing-trailer message", stderr.String())
	}
}

func TestRun_pr_pushFails(t *testing.T) {
	var stderr bytes.Buffer

	git := happyGitMerge("v1.0.0", "dev", "main")
	git["push"] = func(args ...string) (string, error) {
		return "", fmt.Errorf("permission denied")
	}

	cfg := Config{
		Stderr:   &stderr,
		GitExec:  stubGitExec(git),
		ReadFile: mergeConfig("main"),
		PR:       true,
	}

	code := Run(cfg)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "git push failed") {
		t.Errorf("stderr = %q, want push failure message", stderr.String())
	}
}
