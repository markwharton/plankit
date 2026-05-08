package setup

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newGitRepoDir returns a temp dir with a .git subdirectory so git.IsRepo returns true.
func newGitRepoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("setup: mkdir .git: %v", err)
	}
	return dir
}

// fakeGitExec records calls and returns canned responses.
type fakeGitExec struct {
	calls       []string
	tagList     string // output for `tag --list v* ...`
	revParseErr bool   // make `rev-parse --verify` fail
	headErr     bool   // make `rev-parse HEAD` fail (no commits)
	pushErr     bool   // make `push` fail
}

func (f *fakeGitExec) exec(dir string, args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	switch {
	case len(args) >= 2 && args[0] == "tag" && args[1] == "--list":
		return f.tagList, nil
	case len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--verify":
		if f.revParseErr {
			return "", fmt.Errorf("bad ref")
		}
		return "abc123", nil
	case len(args) == 2 && args[0] == "rev-parse" && args[1] == "HEAD":
		if f.headErr {
			return "", fmt.Errorf("fatal: ambiguous argument 'HEAD': unknown revision")
		}
		return "abc123", nil
	case len(args) >= 1 && args[0] == "push":
		if f.pushErr {
			return "", fmt.Errorf("push failed")
		}
		return "", nil
	}
	return "", nil
}

func baselineCfg(dir string, stderr *bytes.Buffer, fake *fakeGitExec) Config {
	cfg := Config{
		Stderr:       stderr,
		ProjectDir:   dir,
		PreserveMode: "manual",
		GuardMode:    "block",
		GitExec:      fake.exec,
	}
	withFS(&cfg)
	return cfg
}

func assertCallMade(t *testing.T, calls []string, want string) {
	t.Helper()
	for _, c := range calls {
		if c == want {
			return
		}
	}
	t.Errorf("missing git call %q; calls=%v", want, calls)
}

func assertCallNotMade(t *testing.T, calls []string, unwanted string) {
	t.Helper()
	for _, c := range calls {
		if c == unwanted {
			t.Errorf("unexpected git call %q; calls=%v", unwanted, calls)
		}
	}
}

func TestRun_baseline_createsV000OnHEAD(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCallMade(t, fake.calls, "tag --list v* --sort=-v:refname")
	assertCallMade(t, fake.calls, "tag v0.0.0 HEAD")
	if !strings.Contains(stderr.String(), "Tagged v0.0.0 on HEAD") {
		t.Errorf("stderr = %q, want 'Tagged v0.0.0 on HEAD'", stderr.String())
	}
}

func TestRun_baseline_noOpWhenTagExists(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{tagList: "v1.2.3\nv1.0.0"}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCallNotMade(t, fake.calls, "tag v0.0.0 HEAD")
	if !strings.Contains(stderr.String(), "Found tag v1.2.3 — already anchored") {
		t.Errorf("stderr = %q, want 'Found tag v1.2.3 — already anchored'", stderr.String())
	}
}

func TestRun_baseline_ignoresNonSemverTag(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{tagList: "v-my-thing\nvnext"}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Lookalike tags must not count as anchored — v0.0.0 should be created.
	assertCallMade(t, fake.calls, "tag v0.0.0 HEAD")
}

func TestRun_baselineAt_usesRef(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.BaselineAt = "deadbeef"

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCallMade(t, fake.calls, "rev-parse --verify deadbeef")
	assertCallMade(t, fake.calls, "tag v0.0.0 deadbeef")
	if !strings.Contains(stderr.String(), "Tagged v0.0.0 on deadbeef") {
		t.Errorf("stderr = %q, want 'Tagged v0.0.0 on deadbeef'", stderr.String())
	}
}

func TestRun_baselineAt_invalidRef(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{revParseErr: true}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.BaselineAt = "not-a-ref"

	err := Run(cfg)
	if err == nil {
		t.Fatal("Run() expected error for invalid ref, got nil")
	}
	if !strings.Contains(err.Error(), "does not resolve") {
		t.Errorf("error = %v, want 'does not resolve'", err)
	}
	assertCallNotMade(t, fake.calls, "tag v0.0.0 not-a-ref")
}

func TestRun_baselinePush_callsGitPushWithHEAD(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.Push = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCallMade(t, fake.calls, "push origin HEAD v0.0.0")
	if !strings.Contains(stderr.String(), "Pushed HEAD and v0.0.0 to origin") {
		t.Errorf("stderr = %q, want 'Pushed HEAD and v0.0.0 to origin'", stderr.String())
	}
}

func TestRun_baselineAtPush_pushesTagOnly(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.BaselineAt = "deadbeef"
	cfg.Push = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// With --at, HEAD is NOT pushed — only the tag.
	assertCallMade(t, fake.calls, "push origin v0.0.0")
	assertCallNotMade(t, fake.calls, "push origin HEAD v0.0.0")
	if !strings.Contains(stderr.String(), "Pushed v0.0.0 to origin") {
		t.Errorf("stderr = %q, want 'Pushed v0.0.0 to origin'", stderr.String())
	}
	if strings.Contains(stderr.String(), "HEAD and v0.0.0") {
		t.Errorf("stderr = %q, should not mention HEAD when --at is used", stderr.String())
	}
}

func TestRun_baseline_printsPushHintWhenNotPushed(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(stderr.String(), "git push origin v0.0.0") {
		t.Errorf("stderr = %q, want push hint", stderr.String())
	}
	assertCallNotMade(t, fake.calls, "push origin v0.0.0")
}

func TestRun_baseline_requiresGitRepo(t *testing.T) {
	// Temp dir without .git — git.IsRepo returns false.
	dir := t.TempDir()
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.AllowNonGit = true // let setup proceed past the IsRepo guard at the top

	err := Run(cfg)
	if err == nil || !strings.Contains(err.Error(), "--baseline requires a git repository") {
		t.Errorf("Run() error = %v, want '--baseline requires a git repository'", err)
	}
}

func TestRun_baseline_skipsTagWhenNoCommits(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{headErr: true}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertCallMade(t, fake.calls, "rev-parse HEAD")
	assertCallNotMade(t, fake.calls, "tag v0.0.0 HEAD")
	if !strings.Contains(stderr.String(), "No commits yet") {
		t.Errorf("stderr = %q, want 'No commits yet' guidance", stderr.String())
	}
	if !strings.Contains(stderr.String(), "pk setup --baseline") {
		t.Errorf("stderr = %q, want re-run hint", stderr.String())
	}
}

func TestRun_baseline_noCommitsDoesNotAffectBaselineAt(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{headErr: true}
	cfg := baselineCfg(dir, &stderr, fake)
	cfg.Baseline = true
	cfg.BaselineAt = "deadbeef"

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// --at bypasses HEAD check; rev-parse --verify validates the ref instead.
	assertCallNotMade(t, fake.calls, "rev-parse HEAD")
	assertCallMade(t, fake.calls, "rev-parse --verify deadbeef")
	assertCallMade(t, fake.calls, "tag v0.0.0 deadbeef")
}

func TestRun_noTagsTip_shownWhenNoTags(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{}
	cfg := baselineCfg(dir, &stderr, fake)
	// Baseline not set.

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(stderr.String(), "No version tags found") {
		t.Errorf("stderr = %q, want 'No version tags found' tip", stderr.String())
	}
}

func TestRun_noTagsTip_hiddenWhenTagsExist(t *testing.T) {
	dir := newGitRepoDir(t)
	var stderr bytes.Buffer
	fake := &fakeGitExec{tagList: "v0.1.0"}
	cfg := baselineCfg(dir, &stderr, fake)

	if err := Run(cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if strings.Contains(stderr.String(), "No version tags found") {
		t.Errorf("stderr = %q, tip should be hidden when tags exist", stderr.String())
	}
}
