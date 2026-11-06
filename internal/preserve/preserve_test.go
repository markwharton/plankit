package preserve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/markwharton/plankit/internal/cli"
	"github.com/markwharton/plankit/internal/config"
	"github.com/markwharton/plankit/internal/git"
	"github.com/markwharton/plankit/internal/hookio"
)

const planBody = "# Ship The Widget\n\nContext: enough substance to clear the minimum plan size threshold.\n"

func fixedNow(t *testing.T) {
	t.Helper()
	old := now
	now = func() time.Time { return time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { now = old })
}

// scratch builds a configured repo (preserve mode as given) plus a fake
// Claude plans dir holding one approved plan; returns repo and plan path.
func scratch(t *testing.T, mode, content string) (string, string) {
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
	cfg.Preserve.Mode = mode
	if err := config.Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	git.Exec(dir, "add", ".pk.json")
	git.Exec(dir, "commit", "-q", "-m", "config")

	// The plan lives where Claude Code writes it: ~/.claude/plans of a home
	// that is this test's, with no CLAUDE_CONFIG_DIR in the environment.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	claudePlans := filepath.Join(home, ".claude", "plans")
	os.MkdirAll(claudePlans, 0o755)
	planPath := filepath.Join(claudePlans, "widget.md")
	os.WriteFile(planPath, []byte(content), 0o644)
	return dir, planPath
}

// configDirPlan points CLAUDE_CONFIG_DIR at a second configuration
// directory (a second account) and writes a plan under its plans/.
func configDirPlan(t *testing.T, content string) string {
	t.Helper()
	configDir := filepath.Join(t.TempDir(), ".claude-work")
	plans := filepath.Join(configDir, "plans")
	os.MkdirAll(plans, 0o755)
	planPath := filepath.Join(plans, "widget.md")
	os.WriteFile(planPath, []byte(content), 0o644)
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)
	return planPath
}

// runPreserveIO runs the command: with a plan path, as the hook with
// that plan in the payload; without one, as a typed run with no stdin.
// Returns stdout, stderr, and the exit code.
func runPreserveIO(t *testing.T, dir, planPath string, args ...string) (string, string, int) {
	t.Helper()
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	var stdin io.Reader
	if planPath != "" {
		payload, _ := json.Marshal(map[string]any{
			"cwd":           dir,
			"tool_response": map[string]string{"filePath": planPath},
		})
		stdin = bytes.NewReader(payload)
	}
	var out, errw bytes.Buffer
	argv := append([]string{"pk", "preserve", "--project-dir", dir}, args...)
	code := cli.RunIO(argv, []*cli.Command{Cmd}, stdin, &out, &errw)
	return out.String(), errw.String(), code
}

// runPreserve is runPreserveIO for the runs that must succeed.
func runPreserve(t *testing.T, dir, planPath string, args ...string) (string, string) {
	t.Helper()
	out, errw, code := runPreserveIO(t, dir, planPath, args...)
	if code != 0 {
		t.Fatalf("hook exit %d (stderr: %s)", code, errw)
	}
	return out, errw
}

// plansDir is the default plans directory of a scratch repo.
func plansDir(dir string) string { return config.Default("main").Preserve.DirPath(dir) }

func commitCount(t *testing.T, dir string) int {
	out, err := git.Exec(dir, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	var n int
	fmt.Sscanf(out, "%d", &n)
	return n
}

// withUpstream gives the repo a bare origin and pushes main to it with
// the upstream set, so HEAD is on the remote and nothing is unpushed.
// Returns the origin path.
func withUpstream(t *testing.T, dir string) string {
	t.Helper()
	origin := t.TempDir()
	for _, step := range []struct {
		dir  string
		args []string
	}{
		{origin, []string{"init", "-q", "--bare"}},
		{dir, []string{"remote", "add", "origin", origin}},
		{dir, []string{"push", "-q", "-u", "origin", "main"}},
	} {
		if _, err := git.Exec(step.dir, step.args...); err != nil {
			t.Fatal(err)
		}
	}
	return origin
}

// TestAutoModePreservesAndCommits is the plan-only case: HEAD is on
// its upstream, so the plan commit is all the next push carries and
// the subject takes [skip ci].
func TestAutoModePreservesAndCommits(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	withUpstream(t, dir)
	before := commitCount(t, dir)

	out, _ := runPreserve(t, dir, plan)

	dest := filepath.Join(plansDir(dir), "2026-09-05-001-ship-the-widget.md")
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("preserved file: %v", err)
	}
	if !bytes.Equal(got, []byte(planBody)) {
		t.Fatal("preserved bytes differ from the approved plan")
	}
	if commitCount(t, dir) != before+1 {
		t.Fatal("expected one new commit")
	}
	subject, _ := git.Exec(dir, "log", "-1", "--format=%s")
	if subject != "plan: Ship The Widget [skip ci]" {
		t.Fatalf("subject = %q", subject)
	}
	if !strings.Contains(out, "Approved plan committed: docs/plans/2026-09-05-001-ship-the-widget.md\"") {
		t.Fatalf("systemMessage: %s", out)
	}
}

// TestMarkerLeftOffWhenUnpushedCommitsPrecedeThePlan: the plan lands on
// top of unpushed work, so the subject carries no [skip ci] and the
// preview and the response both say why.
func TestMarkerLeftOffWhenUnpushedCommitsPrecedeThePlan(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	withUpstream(t, dir)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0o644)
	git.Exec(dir, "add", "b.txt")
	git.Exec(dir, "commit", "-q", "-m", "feat: b")

	_, errw := runPreserve(t, dir, plan, "--dry-run")
	if !strings.Contains(errw, "  Commit: plan: Ship The Widget\n") || !strings.Contains(errw, "  CI:     [skip ci] left off: unpushed commits precede it\n") {
		t.Fatalf("preview: %s", errw)
	}

	out, _ := runPreserve(t, dir, plan)
	subject, _ := git.Exec(dir, "log", "-1", "--format=%s")
	if subject != "plan: Ship The Widget" {
		t.Fatalf("subject = %q", subject)
	}
	if !strings.Contains(out, "Approved plan committed: docs/plans/2026-09-05-001-ship-the-widget.md ([skip ci] left off: unpushed commits precede it)") {
		t.Fatalf("systemMessage: %s", out)
	}
}

// TestMarkerLeftOffWithoutAnUpstream: with no upstream nothing says
// what is unpushed, so the marker stays off and the response says so.
func TestMarkerLeftOffWithoutAnUpstream(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	out, _ := runPreserve(t, dir, plan)
	subject, _ := git.Exec(dir, "log", "-1", "--format=%s")
	if subject != "plan: Ship The Widget" {
		t.Fatalf("subject = %q", subject)
	}
	if !strings.Contains(out, "([skip ci] left off: the branch has no upstream)") {
		t.Fatalf("systemMessage: %s", out)
	}
}

// TestOtherChangesStayOutOfThePlanCommit: a plan commit holds the plan
// file alone. A staged addition, a staged deletion, and an unstaged edit
// are all left exactly as they were.
func TestOtherChangesStayOutOfThePlanCommit(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0o644)
	git.Exec(dir, "add", "b.txt")
	git.Exec(dir, "commit", "-q", "-m", "b")
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("c\n"), 0o644)
	git.Exec(dir, "add", "c.txt")
	git.Exec(dir, "rm", "-q", "b.txt")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("edited\n"), 0o644)
	statusBefore, _ := git.Exec(dir, "status", "--porcelain")
	before := commitCount(t, dir)

	runPreserve(t, dir, plan)

	if commitCount(t, dir) != before+1 {
		t.Fatal("expected one new commit")
	}
	files, _ := git.Exec(dir, "show", "--name-status", "--format=", "HEAD")
	if strings.TrimSpace(files) != "A\tdocs/plans/2026-09-05-001-ship-the-widget.md" {
		t.Fatalf("committed: %q", files)
	}
	if statusAfter, _ := git.Exec(dir, "status", "--porcelain"); statusAfter != statusBefore {
		t.Fatalf("status changed:\nbefore %q\nafter  %q", statusBefore, statusAfter)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "a.txt")); string(got) != "edited\n" {
		t.Fatalf("unstaged edit disturbed: %q", got)
	}
}

// TestPushCarriesOnlyThePlanCommit: --push publishes the plan commit,
// and the staged work it left out stays local and staged.
func TestPushCarriesOnlyThePlanCommit(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	origin := withUpstream(t, dir)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte("c\n"), 0o644)
	git.Exec(dir, "add", "c.txt")

	out, _ := runPreserve(t, dir, plan, "--push")

	if !strings.Contains(out, "committed and pushed") {
		t.Fatalf("out = %s", out)
	}
	head, _ := git.Exec(dir, "rev-parse", "HEAD")
	if remote, _ := git.Exec(origin, "rev-parse", "main"); remote != head {
		t.Fatalf("origin main %s, local HEAD %s", remote, head)
	}
	files, _ := git.Exec(origin, "show", "--name-only", "--format=", "main")
	if strings.TrimSpace(files) != "docs/plans/2026-09-05-001-ship-the-widget.md" {
		t.Fatalf("pushed: %q", files)
	}
	if staged, _ := git.Exec(dir, "diff", "--cached", "--name-only"); staged != "c.txt" {
		t.Fatalf("staged after push: %q", staged)
	}
}

func TestDuplicateContentIsNotRecommitted(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	runPreserve(t, dir, plan)
	before := commitCount(t, dir)

	out, _ := runPreserve(t, dir, plan)
	if commitCount(t, dir) != before {
		t.Fatal("duplicate plan created a commit")
	}
	if !strings.Contains(out, "already preserved") {
		t.Fatalf("out = %s", out)
	}
	entries, _ := os.ReadDir(plansDir(dir))
	if len(entries) != 1 {
		t.Fatalf("plans dir has %d entries", len(entries))
	}
}

func TestSequenceIncrementsWithinADay(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	runPreserve(t, dir, plan)

	second := strings.Replace(planBody, "Ship The Widget", "Refine The Widget", 1)
	os.WriteFile(plan, []byte(second), 0o644)
	runPreserve(t, dir, plan)

	if _, err := os.Stat(filepath.Join(plansDir(dir), "2026-09-05-002-refine-the-widget.md")); err != nil {
		t.Fatalf("second plan: %v", err)
	}
}

func TestManualModeWritesPointerThenExplicitRunCommits(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "manual", planBody)
	before := commitCount(t, dir)

	out, _ := runPreserve(t, dir, plan)
	if commitCount(t, dir) != before {
		t.Fatal("manual mode must not commit")
	}
	if !strings.Contains(out, "/plankit:preserve") || !strings.Contains(out, "additionalContext") {
		t.Fatalf("manual response: %s", out)
	}
	ptr, err := os.ReadFile(filepath.Join(dir, ".git", "pk-pending-plan"))
	if err != nil || strings.TrimSpace(string(ptr)) != plan {
		t.Fatalf("pointer: %q err=%v", ptr, err)
	}

	// Explicit invocation: no stdin payload, consumes the pointer.
	out, _ = runPreserve(t, dir, "")
	if commitCount(t, dir) != before+1 {
		t.Fatal("explicit run should commit")
	}
	if !strings.Contains(out, "Approved plan committed") {
		t.Fatalf("out = %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "pk-pending-plan")); !os.IsNotExist(err) {
		t.Fatal("pointer should be consumed")
	}
}

// TestTypedRunSaysWhyWhileTheHookStaysSilent: a person who typed the
// command gets the reason on stderr, while the hook declines without a
// word, because its stdout is the PostToolUse envelope.
func TestTypedRunSaysWhyWhileTheHookStaysSilent(t *testing.T) {
	dir, plan := scratch(t, "manual", planBody)

	out, errw := runPreserve(t, dir, "")
	if out != "" {
		t.Fatalf("typed run wrote to stdout: %s", out)
	}
	if errw != "No pending plan to preserve\n" {
		t.Fatalf("typed run narration: %q", errw)
	}

	// A payload naming a plan that is gone: the hook has nothing to do.
	os.Remove(plan)
	out, errw = runPreserve(t, dir, plan)
	if out != "" || errw != "" {
		t.Fatalf("hook broke silence: out=%q errw=%q", out, errw)
	}
}

func TestStalePointerIsRemoved(t *testing.T) {
	dir, _ := scratch(t, "manual", planBody)
	ptr := filepath.Join(dir, ".git", "pk-pending-plan")
	os.WriteFile(ptr, []byte("/nowhere/gone.md\n"), 0o644)
	runPreserve(t, dir, "")
	if _, err := os.Stat(ptr); !os.IsNotExist(err) {
		t.Fatal("stale pointer should be deleted")
	}
}

func TestDiscardRemovesThePointerAndNamesThePlan(t *testing.T) {
	dir, plan := scratch(t, "manual", planBody)
	runPreserve(t, dir, plan) // the hook records the pointer
	ptr := filepath.Join(dir, ".git", "pk-pending-plan")
	if _, err := os.Stat(ptr); err != nil {
		t.Fatal("the hook should have written the pointer")
	}
	before := commitCount(t, dir)
	out, errw := runPreserve(t, dir, "", "--discard")
	if out != "" || errw != "Discarded the pending plan: "+plan+"\n" {
		t.Fatalf("out=%q errw=%q", out, errw)
	}
	if _, err := os.Stat(ptr); !os.IsNotExist(err) {
		t.Fatal("the pointer should be gone")
	}
	if commitCount(t, dir) != before {
		t.Fatal("discard must not commit")
	}
	if _, errw := runPreserve(t, dir, ""); errw != "No pending plan to preserve\n" {
		t.Fatalf("a typed run after discard: %q", errw)
	}
}

func TestDiscardWithNothingPending(t *testing.T) {
	dir, _ := scratch(t, "manual", planBody)
	out, errw := runPreserve(t, dir, "", "--discard")
	if out != "" || errw != "No pending plan to discard\n" {
		t.Fatalf("out=%q errw=%q", out, errw)
	}
}

func TestDiscardReportsAStalePointer(t *testing.T) {
	dir, _ := scratch(t, "manual", planBody)
	ptr := filepath.Join(dir, ".git", "pk-pending-plan")
	os.WriteFile(ptr, []byte("/nowhere/gone.md\n"), 0o644)
	_, errw := runPreserve(t, dir, "", "--discard")
	if errw != "Discarded the pending plan: /nowhere/gone.md (the plan file no longer exists)\n" {
		t.Fatalf("errw=%q", errw)
	}
	if _, err := os.Stat(ptr); !os.IsNotExist(err) {
		t.Fatal("the stale pointer should be gone")
	}
}

func TestDiscardDryRunKeepsThePointer(t *testing.T) {
	dir, plan := scratch(t, "manual", planBody)
	runPreserve(t, dir, plan)
	_, errw := runPreserve(t, dir, "", "--discard", "--dry-run")
	if !strings.Contains(errw, "pk preserve --dry-run:\n  Discard: "+plan+"\n") {
		t.Fatalf("errw=%q", errw)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "pk-pending-plan")); err != nil {
		t.Fatal("a dry run must keep the pointer")
	}
}

func TestDiscardRefusesPlanAndPush(t *testing.T) {
	dir, plan := scratch(t, "manual", planBody)
	for _, args := range [][]string{{"--discard", "--plan", plan}, {"--discard", "--push"}} {
		_, errw, code := runPreserveIO(t, dir, "", args...)
		if code != cli.ExitUsage || !strings.Contains(errw, "--discard removes the pending plan pointer") {
			t.Fatalf("%v: code=%d errw=%q", args, code, errw)
		}
	}
}

func TestPendingPlanIsReadWithoutRemovingIt(t *testing.T) {
	dir, plan := scratch(t, "manual", planBody)
	if _, _, ok := PendingPlan(dir); ok {
		t.Fatal("nothing pending yet")
	}
	runPreserve(t, dir, plan)
	if path, stale, ok := PendingPlan(dir); !ok || stale || path != plan {
		t.Fatalf("live pointer: path=%q stale=%v ok=%v", path, stale, ok)
	}
	ptr := filepath.Join(dir, ".git", "pk-pending-plan")
	os.WriteFile(ptr, []byte("/nowhere/gone.md\n"), 0o644)
	if path, stale, ok := PendingPlan(dir); !ok || !stale || path != "/nowhere/gone.md" {
		t.Fatalf("stale pointer: path=%q stale=%v ok=%v", path, stale, ok)
	}
	if _, err := os.Stat(ptr); err != nil {
		t.Fatal("PendingPlan must leave a stale pointer in place")
	}
}

func TestOffModeDoesNothing(t *testing.T) {
	dir, plan := scratch(t, "off", planBody)
	before := commitCount(t, dir)
	out, _ := runPreserve(t, dir, plan)
	if out != "" || commitCount(t, dir) != before {
		t.Fatalf("off mode acted: %s", out)
	}
}

func TestUnconfiguredRepoIsSilent(t *testing.T) {
	dir, plan := scratch(t, "auto", planBody)
	os.Remove(config.Path(dir))
	out, _ := runPreserve(t, dir, plan)
	if out != "" {
		t.Fatalf("unconfigured must no-op: %s", out)
	}
}

func TestShortPlanIsSkipped(t *testing.T) {
	dir, plan := scratch(t, "auto", "# Tiny\n")
	out, errw := runPreserve(t, dir, plan)
	if out != "" || errw != "" {
		t.Fatalf("the hook declines a short plan in silence: out=%q errw=%q", out, errw)
	}

	// A typed run is a person asking: a pending plan under the minimum
	// is named, with the way to drop it, and the pointer stays.
	pointer := filepath.Join(dir, ".git", "pk-pending-plan")
	if err := os.WriteFile(pointer, []byte(plan+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errw, code := runPreserveIO(t, dir, "")
	if code != cli.ExitState || out != "" || !strings.Contains(errw, "below the minimum") || !strings.Contains(errw, "pk preserve --discard") {
		t.Fatalf("typed: code=%d out=%q errw=%q", code, out, errw)
	}
	if _, err := os.Stat(pointer); err != nil {
		t.Fatal("the pointer must stay until the person drops it")
	}
}

func TestCRLFPlanKeepsCleanTitle(t *testing.T) {
	fixedNow(t)
	crlf := "# Windows Plan\r\n\r\nEnough body content to clear the minimum size threshold easily.\r\n"
	dir, plan := scratch(t, "auto", crlf)
	runPreserve(t, dir, plan)
	if _, err := os.Stat(filepath.Join(plansDir(dir), "2026-09-05-001-windows-plan.md")); err != nil {
		t.Fatalf("CRLF slug: %v", err)
	}
	subject, _ := git.Exec(dir, "log", "-1", "--format=%s")
	if subject != "plan: Windows Plan" {
		t.Fatalf("subject carries CR? %q", subject)
	}
}

func TestDryRunPreviewsOnly(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	before := commitCount(t, dir)
	_, errw := runPreserve(t, dir, plan, "--dry-run")
	if !strings.Contains(errw, "2026-09-05-001-ship-the-widget.md") {
		t.Fatalf("preview: %s", errw)
	}
	if commitCount(t, dir) != before {
		t.Fatal("dry-run committed")
	}
	if _, err := os.Stat(plansDir(dir)); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote the plans dir")
	}
}

// TestPlanFlagPreservesARevisedPlanWithoutAPointer is the case --plan
// exists for: the approved plan was preserved, the file was revised
// after, and no pointer names it any more.
func TestPlanFlagPreservesARevisedPlanWithoutAPointer(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "manual", planBody)
	runPreserve(t, dir, plan) // hook: pointer written
	runPreserve(t, dir, "")   // typed: 001 committed, pointer consumed
	before := commitCount(t, dir)

	revised := planBody + "\n## Added after discussion\n\nOne more section.\n"
	os.WriteFile(plan, []byte(revised), 0o644)
	out, _ := runPreserve(t, dir, "", "--plan", plan)

	dest := filepath.Join(plansDir(dir), "2026-09-05-002-ship-the-widget.md")
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("revised plan: %v", err)
	}
	if !bytes.Equal(got, []byte(revised)) {
		t.Fatal("preserved bytes differ from the revised plan")
	}
	if commitCount(t, dir) != before+1 {
		t.Fatal("expected one new commit")
	}
	if !strings.Contains(out, "Approved plan committed: docs/plans/2026-09-05-002-ship-the-widget.md") {
		t.Fatalf("out = %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "pk-pending-plan")); !os.IsNotExist(err) {
		t.Fatal("no pointer should exist")
	}
}

// A plan is preserved when it is committed. A run whose commit failed
// leaves the file behind; the next run commits that file, once, and
// only then is the plan already preserved.
func TestAPlanWhoseCommitFailedIsCommittedOnTheNextRun(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	hook := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := commitCount(t, dir)
	// The hook reports the failure with the exit Claude Code shows; a
	// typed run exits non-zero through the frame.
	if out, errw, code := runPreserveIO(t, dir, plan); code != hookio.ExitReport || out != "" || !strings.Contains(errw, "pk preserve: git commit failed") || commitCount(t, dir) != before {
		t.Fatalf("hook: code=%d commits=%d out=%q errw=%q", code, commitCount(t, dir), out, errw)
	}
	if out, errw, code := runPreserveIO(t, dir, "", "--plan", plan); code != cli.ExitInternal || out != "" || !strings.Contains(errw, "Error: git commit failed") || commitCount(t, dir) != before {
		t.Fatalf("typed: code=%d commits=%d out=%q errw=%q", code, commitCount(t, dir), out, errw)
	}

	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	out, _ := runPreserve(t, dir, plan)
	if commitCount(t, dir) != before+1 || !strings.Contains(out, "Approved plan committed: docs/plans/2026-09-05-001-ship-the-widget.md") {
		t.Fatalf("the leftover plan was not committed: commits=%d out=%s", commitCount(t, dir), out)
	}
	if entries, _ := os.ReadDir(plansDir(dir)); len(entries) != 1 {
		t.Fatalf("plans dir has %d entries, want the one file", len(entries))
	}
	if clean, _ := git.Clean(dir); !clean {
		t.Fatal("the tree should be clean once the plan is committed")
	}

	out, _ = runPreserve(t, dir, plan)
	if commitCount(t, dir) != before+1 || !strings.Contains(out, "already preserved") {
		t.Fatalf("a committed plan is already preserved: commits=%d out=%s", commitCount(t, dir), out)
	}
}

// A plans directory git ignores can hold no commit: preserve refuses
// before it writes anything and names the setting.
func TestAnIgnoredPlansDirectoryIsRefused(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("docs/plans/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git.Exec(dir, "add", ".gitignore")
	git.Exec(dir, "commit", "-q", "-m", "ignore")
	before := commitCount(t, dir)

	out, errw, code := runPreserveIO(t, dir, plan)
	if code != hookio.ExitReport || out != "" || !strings.Contains(errw, "ignored by git") || !strings.Contains(errw, "preserve.dir") {
		t.Fatalf("hook: code=%d out=%q errw=%q", code, out, errw)
	}
	if _, errw, code := runPreserveIO(t, dir, "", "--plan", plan); code != cli.ExitState || !strings.Contains(errw, "ignored by git") {
		t.Fatalf("typed: code=%d errw=%q", code, errw)
	}
	if _, err := os.Stat(plansDir(dir)); !os.IsNotExist(err) {
		t.Fatal("refused, yet the plans directory was written")
	}
	if commitCount(t, dir) != before {
		t.Fatal("refused, yet a commit was made")
	}
}

func TestPlanFlagIdenticalBytesReportTheExistingFile(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	runPreserve(t, dir, plan)
	before := commitCount(t, dir)

	out, _ := runPreserve(t, dir, "", "--plan", plan)
	if commitCount(t, dir) != before {
		t.Fatal("identical plan created a commit")
	}
	if !strings.Contains(out, "already preserved as docs/plans/2026-09-05-001-ship-the-widget.md") {
		t.Fatalf("out = %s", out)
	}
	entries, _ := os.ReadDir(plansDir(dir))
	if len(entries) != 1 {
		t.Fatalf("plans dir has %d entries", len(entries))
	}
}

func TestPlanFlagRefusesAPathOutsideClaudePlans(t *testing.T) {
	dir, _ := scratch(t, "manual", planBody)
	before := commitCount(t, dir)
	elsewhere := filepath.Join(t.TempDir(), "notes.md")
	os.WriteFile(elsewhere, []byte(planBody), 0o644)

	out, errw, code := runPreserveIO(t, dir, "", "--plan", elsewhere)
	if code != cli.ExitUsage {
		t.Fatalf("exit %d, want %d (stderr: %s)", code, cli.ExitUsage, errw)
	}
	if !strings.Contains(errw, elsewhere) || !strings.Contains(errw, ".claude/plans/") {
		t.Fatalf("stderr: %s", errw)
	}
	if out != "" || commitCount(t, dir) != before {
		t.Fatalf("refusal acted: out=%q", out)
	}
}

func TestPlanFlagRefusesWhenThePointerNamesAnotherPlan(t *testing.T) {
	dir, planA := scratch(t, "manual", planBody)
	runPreserve(t, dir, planA) // pointer names A
	before := commitCount(t, dir)
	planB := filepath.Join(filepath.Dir(planA), "other.md")
	os.WriteFile(planB, []byte(strings.Replace(planBody, "Ship The Widget", "Other Plan", 1)), 0o644)

	out, errw, code := runPreserveIO(t, dir, "", "--plan", planB)
	if code != cli.ExitState {
		t.Fatalf("exit %d, want %d (stderr: %s)", code, cli.ExitState, errw)
	}
	if !strings.Contains(errw, planA) || !strings.Contains(errw, planB) {
		t.Fatalf("stderr must name both files: %s", errw)
	}
	if out != "" || commitCount(t, dir) != before {
		t.Fatalf("refusal acted: out=%q", out)
	}
	ptr, err := os.ReadFile(filepath.Join(dir, ".git", "pk-pending-plan"))
	if err != nil || strings.TrimSpace(string(ptr)) != planA {
		t.Fatalf("pointer disturbed: %q err=%v", ptr, err)
	}
}

func TestPlanFlagDryRunPreviewsOnly(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "manual", planBody)
	before := commitCount(t, dir)
	out, errw := runPreserve(t, dir, "", "--plan", plan, "--dry-run")
	if !strings.Contains(errw, "2026-09-05-001-ship-the-widget.md") {
		t.Fatalf("preview: %s", errw)
	}
	if out != "" || commitCount(t, dir) != before {
		t.Fatalf("dry-run acted: out=%q", out)
	}
	if _, err := os.Stat(plansDir(dir)); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote the plans dir")
	}
}

func TestExtractPlanPathHandlesWindowsEscapes(t *testing.T) {
	raw := json.RawMessage(`{"filePath":"C:\\Users\\m\\.claude\\plans\\p.md"}`)
	home := func() (string, error) { return "C:\\Users\\m", nil }
	dir := claudePlansDir("", home)
	if dir != "C:/Users/m/.claude/plans" {
		t.Fatalf("claudePlansDir = %q", dir)
	}
	if got := extractPlanPath(raw, dir, home); got != "C:/Users/m/.claude/plans/p.md" {
		t.Fatalf("got %q", got)
	}
	unix := func() (string, error) { return "/home/m", nil }
	legacy := json.RawMessage(`"Plan saved to /home/m/.claude/plans/x.md"`)
	if got := extractPlanPath(legacy, claudePlansDir("", unix), unix); got != "/home/m/.claude/plans/x.md" {
		t.Fatalf("legacy form: %q", got)
	}
	tilde := json.RawMessage(`{"filePath":"~/.claude/plans/t.md"}`)
	if got := extractPlanPath(tilde, claudePlansDir("", unix), unix); got != "/home/m/.claude/plans/t.md" {
		t.Fatalf("tilde form: %q", got)
	}
}

// TestClaudePlansDirFollowsClaudeConfigDir: Claude Code keeps a relocated
// configuration (a second account) under CLAUDE_CONFIG_DIR, and its
// plans under plans/ there; the default ~/.claude/plans then belongs
// to another session and must not match.
func TestClaudePlansDirFollowsClaudeConfigDir(t *testing.T) {
	unix := func() (string, error) { return "/home/m", nil }
	if got := claudePlansDir("/home/m/.claude-work", unix); got != "/home/m/.claude-work/plans" {
		t.Fatalf("configured: %q", got)
	}
	if got := claudePlansDir("~/.claude-work/", unix); got != "/home/m/.claude-work/plans" {
		t.Fatalf("tilde and trailing slash: %q", got)
	}
	if got := claudePlansDir("C:\\Users\\m\\.claude-work", unix); got != "C:/Users/m/.claude-work/plans" {
		t.Fatalf("windows: %q", got)
	}
	if got := claudePlansDir("", func() (string, error) { return "", os.ErrNotExist }); got != "" {
		t.Fatalf("no home, no variable: %q", got)
	}
	dir := claudePlansDir("/home/m/.claude-work", unix)
	work := json.RawMessage(`{"filePath":"/home/m/.claude-work/plans/p.md"}`)
	if got := extractPlanPath(work, dir, unix); got != "/home/m/.claude-work/plans/p.md" {
		t.Fatalf("work account plan: %q", got)
	}
	other := json.RawMessage(`{"filePath":"/home/m/.claude/plans/p.md"}`)
	if got := extractPlanPath(other, dir, unix); got != "" {
		t.Fatalf("the other account's plan matched: %q", got)
	}
}

func TestHookPreservesAPlanUnderClaudeConfigDir(t *testing.T) {
	fixedNow(t)
	dir, _ := scratch(t, "auto", planBody)
	plan := configDirPlan(t, planBody)
	before := commitCount(t, dir)
	runPreserve(t, dir, plan)
	if commitCount(t, dir) != before+1 {
		t.Fatal("the plan under CLAUDE_CONFIG_DIR was not preserved")
	}
	if _, err := os.Stat(filepath.Join(plansDir(dir), "2026-09-05-001-ship-the-widget.md")); err != nil {
		t.Fatal(err)
	}
}

func TestPlanFlagAcceptsClaudeConfigDirAndRefusesTheDefaultBesideIt(t *testing.T) {
	fixedNow(t)
	dir, defaultPlan := scratch(t, "manual", planBody)
	plan := configDirPlan(t, planBody)
	before := commitCount(t, dir)

	out, errw, code := runPreserveIO(t, dir, "", "--plan", defaultPlan)
	if code != cli.ExitUsage {
		t.Fatalf("the default directory's plan was accepted with CLAUDE_CONFIG_DIR set: exit %d, out=%q", code, out)
	}
	if !strings.Contains(errw, ".claude-work/plans/") || !strings.Contains(errw, "CLAUDE_CONFIG_DIR") {
		t.Fatalf("refusal must name the session's directory: %s", errw)
	}
	if commitCount(t, dir) != before {
		t.Fatal("refusal acted")
	}

	runPreserve(t, dir, "", "--plan", plan)
	if commitCount(t, dir) != before+1 {
		t.Fatal("the plan under CLAUDE_CONFIG_DIR was not preserved by --plan")
	}
}

// TestPageNamesThePointerFile: the page tells the reader where the
// pending-plan pointer lives, and that name is a constant here.
func TestConfiguredDirIsUsed(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "auto", planBody)
	cfg, _ := config.Load(dir)
	cfg.Preserve.Dir = "documentation/plans"
	if err := config.Write(dir, cfg); err != nil {
		t.Fatal(err)
	}
	git.Exec(dir, "commit", "-qam", "dir")

	out, _ := runPreserve(t, dir, plan)
	dest := filepath.Join(dir, "documentation", "plans", "2026-09-05-001-ship-the-widget.md")
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("configured dir: %v", err)
	}
	if !strings.Contains(out, "Approved plan committed: documentation/plans/2026-09-05-001-ship-the-widget.md") {
		t.Fatalf("out = %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "docs")); !os.IsNotExist(err) {
		t.Fatal("the default directory was created")
	}
	files, _ := git.Exec(dir, "show", "--name-only", "--format=", "HEAD")
	if strings.TrimSpace(files) != "documentation/plans/2026-09-05-001-ship-the-widget.md" {
		t.Fatalf("committed: %q", files)
	}
}

// TestTypedRunRefusesAnUnconfiguredRepository: the hook declines in
// silence without a policy file; a person who typed the command is told
// the file is missing and how to write it.
func TestTypedRunRefusesAnUnconfiguredRepository(t *testing.T) {
	dir, plan := scratch(t, "auto", planBody)
	os.Remove(config.Path(dir))
	for _, args := range [][]string{{}, {"--plan", plan}} {
		out, errw, code := runPreserveIO(t, dir, "", args...)
		if code != cli.ExitState {
			t.Fatalf("%v: exit %d, want %d (stderr: %s)", args, code, cli.ExitState, errw)
		}
		if !strings.Contains(errw, config.FileName) || !strings.Contains(errw, "pk init") {
			t.Fatalf("%v: stderr: %s", args, errw)
		}
		if out != "" {
			t.Fatalf("%v: wrote to stdout: %s", args, out)
		}
	}
}

func TestPageNamesThePointerFile(t *testing.T) {
	page, err := os.ReadFile(filepath.Join("..", "..", "skills", "preserve", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "`.git/"+pointerFilename+"`") {
		t.Fatalf("skills/preserve/SKILL.md does not name .git/%s", pointerFilename)
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Ship The Widget", "ship-the-widget"},
		{"Fix: CRLF & paths (v2)", "fix-crlf-paths-v2"},
		{"  --- ", ""},
		{"Ünïcode Tïtle", "ünïcode-tïtle"},
	}
	for _, tc := range cases {
		if got := slugify(tc.in, 60); got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A payload that does not parse is not a typed run: the pending plan is
// left alone, nothing is committed, and the problem is reported with
// the exit Claude Code shows.
func TestAPayloadThatDoesNotParseIsReported(t *testing.T) {
	t.Setenv("CLAUDE_PROJECT_DIR", "")
	dir, plan := scratch(t, "manual", planBody)
	pointer := filepath.Join(dir, ".git", "pk-pending-plan")
	if err := os.WriteFile(pointer, []byte(plan+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := commitCount(t, dir)
	var out, errw bytes.Buffer
	code := cli.RunIO([]string{"pk", "preserve", "--project-dir", dir}, []*cli.Command{Cmd}, strings.NewReader("not json"), &out, &errw)
	if code != hookio.ExitReport || out.Len() != 0 || !strings.Contains(errw.String(), "pk preserve:") {
		t.Fatalf("code=%d out=%q errw=%q", code, out.String(), errw.String())
	}
	if commitCount(t, dir) != before {
		t.Fatal("a payload that does not parse committed the pending plan")
	}
	if _, err := os.Stat(pointer); err != nil {
		t.Fatal("the pointer must be left alone")
	}
}

// A path that enters the plans directory and climbs back out of it is
// not a plan. One that wanders and comes back is, under its clean name.
func TestPlanFlagRefusesAPathThatClimbsOutOfThePlansDirectory(t *testing.T) {
	fixedNow(t)
	dir, plan := scratch(t, "manual", planBody)
	plans := filepath.Dir(plan)
	outside := filepath.Join(filepath.Dir(plans), "outside.md")
	if err := os.WriteFile(outside, []byte(planBody), 0o644); err != nil {
		t.Fatal(err)
	}
	before := commitCount(t, dir)

	climbing := plans + "/../outside.md"
	out, errw, code := runPreserveIO(t, dir, "", "--plan", climbing)
	if code != cli.ExitUsage || !strings.Contains(errw, "is not a Claude Code plan") {
		t.Fatalf("code=%d errw=%q", code, errw)
	}
	if out != "" || commitCount(t, dir) != before {
		t.Fatalf("refusal acted: out=%q", out)
	}

	if err := os.MkdirAll(filepath.Join(plans, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	returning := plans + "/sub/../" + filepath.Base(plan)
	out, errw, code = runPreserveIO(t, dir, "", "--plan", returning)
	if code != cli.ExitOK || !strings.Contains(out, "Approved plan committed") || commitCount(t, dir) != before+1 {
		t.Fatalf("a path that returns inside is a plan: code=%d out=%q errw=%q", code, out, errw)
	}
}
