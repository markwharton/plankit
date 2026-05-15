package preserve

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		title  string
		maxLen int
		want   string
	}{
		{"My Plan Title", 60, "my-plan-title"},
		{"Fix: bug #123 & improve UX!", 60, "fix-bug-123-improve-ux"},
		{"Caching Report: weather-data", 60, "caching-report-weather-data"},
		{"  Leading and trailing spaces  ", 60, "leading-and-trailing-spaces"},
		{"ALL CAPS TITLE", 60, "all-caps-title"},
		{"a-b-c", 60, "a-b-c"},
		{"truncated-at-boundary", 10, "truncated"},
		{"exact-len", 9, "exact-len"},
		{"", 60, ""},
		{"!!!", 60, ""},
		{"Cross-platform Claude Code hooks (staged approach)", 60, "cross-platform-claude-code-hooks-staged-approach"},
		{"Plan café design", 10, "plan-café"},
		{"Résumé builder implementation", 60, "résumé-builder-implementation"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := slugify(tt.title, tt.maxLen)
			if got != tt.want {
				t.Errorf("slugify(%q, %d) = %q, want %q", tt.title, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestExtractPlanPath(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "path in tool response",
			text: `Plan saved to /Users/mark/.claude/plans/quirky-tinkering-wreath.md`,
			want: "/Users/mark/.claude/plans/quirky-tinkering-wreath.md",
		},
		{
			name: "path in JSON context",
			text: `{"path":"/home/user/.claude/plans/my-plan.md","status":"ok"}`,
			want: "/home/user/.claude/plans/my-plan.md",
		},
		{
			name: "no plan path",
			text: "Some other text without a plan path",
			want: "",
		},
		{
			name: "empty text",
			text: "",
			want: "",
		},
		{
			name: "root user path",
			text: `Plan saved to /root/.claude/plans/idempotent-singing-cocoa.md`,
			want: "/root/.claude/plans/idempotent-singing-cocoa.md",
		},
		{
			name: "path in multiline response",
			text: "Plan approved.\nSaved to /home/user/.claude/plans/my-plan.md\nDone.",
			want: "/home/user/.claude/plans/my-plan.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPlanPath(tt.text)
			if got != tt.want {
				t.Errorf("extractPlanPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"normal heading", "# My Plan\n\nContent here", "My Plan"},
		{"heading with content before", "Some preamble\n# The Title\nMore", "The Title"},
		{"no heading", "No heading here\nJust text", "untitled plan"},
		{"empty content", "", "untitled plan"},
		{"h2 but no h1", "## Subheading\nContent", "untitled plan"},
		{"Plan: prefix preserved", "# Plan: Fix hooks merge\nContent", "Plan: Fix hooks merge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.content)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

// withFS sets the filesystem dependencies to real os implementations.
func withFS(cfg *Config) {
	cfg.ReadFile = os.ReadFile
	cfg.WriteFile = os.WriteFile
	cfg.Stat = os.Stat
	cfg.MkdirAll = os.MkdirAll
	cfg.ReadDir = os.ReadDir
	cfg.Remove = os.Remove
}

func TestRun(t *testing.T) {
	fixedTime := time.Date(2026, 3, 11, 15, 30, 0, 0, time.Local)

	t.Run("happy path", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planFile := filepath.Join(plansDir, "test-plan.md")
		planContent := "# Test Plan\n\nThis is a test plan with enough content to pass the minimum length check easily."
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()

		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		var gitCalls []string

		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				call := strings.Join(args, " ")
				gitCalls = append(gitCalls, call)
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		// Verify file was written.
		destFile := filepath.Join(projectDir, "docs", "plans", "2026-03-11-001-test-plan.md")
		content, err := os.ReadFile(destFile)
		if err != nil {
			t.Fatalf("plan file not written: %v", err)
		}
		if string(content) != planContent {
			t.Errorf("content mismatch")
		}

		// Verify git calls (no push by default).
		wantCalls := []string{
			"rev-parse --is-inside-work-tree",
			"add docs/plans/2026-03-11-001-test-plan.md",
			"diff --cached --quiet",
			"commit -m plan: Test Plan [skip ci]",
		}
		if len(gitCalls) != len(wantCalls) {
			t.Fatalf("git calls = %v, want %v", gitCalls, wantCalls)
		}
		for i, want := range wantCalls {
			if gitCalls[i] != want {
				t.Errorf("git call %d = %q, want %q", i, gitCalls[i], want)
			}
		}

		// Verify stdout has systemMessage (commit only, no push).
		if !strings.Contains(stdout.String(), "Approved plan committed:") {
			t.Errorf("stdout = %q, want commit-only systemMessage", stdout.String())
		}
	})

	t.Run("push flag pushes to origin", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planFile := filepath.Join(plansDir, "test-plan.md")
		planContent := "# Push Plan\n\nThis plan has enough content to pass the minimum length check easily."
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		var gitCalls []string

		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				call := strings.Join(args, " ")
				gitCalls = append(gitCalls, call)
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				return "", nil
			},
			Push: true,
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		// Verify push is included in git calls.
		wantCalls := []string{
			"rev-parse --is-inside-work-tree",
			"add docs/plans/2026-03-11-001-push-plan.md",
			"diff --cached --quiet",
			"commit -m plan: Push Plan [skip ci]",
			"push origin HEAD",
		}
		if len(gitCalls) != len(wantCalls) {
			t.Fatalf("git calls = %v, want %v", gitCalls, wantCalls)
		}
		for i, want := range wantCalls {
			if gitCalls[i] != want {
				t.Errorf("git call %d = %q, want %q", i, gitCalls[i], want)
			}
		}

		// Verify stdout says "committed and pushed".
		if !strings.Contains(stdout.String(), "Approved plan committed and pushed") {
			t.Errorf("stdout = %q, want pushed systemMessage", stdout.String())
		}
	})

	t.Run("short plan skipped", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planFile := filepath.Join(plansDir, "short.md")
		os.WriteFile(planFile, []byte("# Short"), 0644)

		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s"}`, planFile)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:   strings.NewReader(inputJSON),
			Stdout:  &stdout,
			Stderr:  &stderr,
			Env:     func(string) string { return "" },

			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) { t.Fatal("unexpected git call"); return "", nil },
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stdout.Len() > 0 {
			t.Errorf("expected no output for short plan, got %q", stdout.String())
		}
	})

	t.Run("plan unchanged", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planFile := filepath.Join(plansDir, "test.md")
		os.WriteFile(planFile, []byte("# Test\n\nThis plan has enough content to pass the minimum length check."), 0644)

		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				// diff --cached --quiet exits 0 when no changes.
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout.String(), "Plan unchanged") {
			t.Errorf("stdout = %q, want 'Plan unchanged'", stdout.String())
		}
	})

	t.Run("no plan path in tool_response exits silently", func(t *testing.T) {
		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":"something else","cwd":"%s"}`, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},
			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) { t.Fatal("unexpected git call"); return "", nil },
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stdout.Len() > 0 {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
	})

	t.Run("sequence increments", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planFile := filepath.Join(plansDir, "test-plan.md")
		planContent := "# Second Plan\n\nThis is a second plan with enough content to pass the minimum length check easily."
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()

		// Pre-populate docs/plans/ with an existing plan for the same date.
		destPlansDir := filepath.Join(projectDir, "docs", "plans")
		os.MkdirAll(destPlansDir, 0755)
		os.WriteFile(filepath.Join(destPlansDir, "2026-03-11-001-first-plan.md"), []byte("existing"), 0644)

		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		var addPath string

		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "add" {
					addPath = args[1]
				}
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		// Verify the file got sequence 002.
		wantPath := "docs/plans/2026-03-11-002-second-plan.md"
		if addPath != wantPath {
			t.Errorf("git add path = %q, want %q", addPath, wantPath)
		}

		destFile := filepath.Join(projectDir, wantPath)
		if _, err := os.Stat(destFile); err != nil {
			t.Errorf("plan file not written at expected path: %v", err)
		}
	})

	t.Run("duplicate content skipped", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Duplicate Plan\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "test-plan.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()

		// Pre-populate docs/plans/ with the same content.
		destPlansDir := filepath.Join(projectDir, "docs", "plans")
		os.MkdirAll(destPlansDir, 0755)
		os.WriteFile(filepath.Join(destPlansDir, "2026-03-11-001-duplicate-plan.md"), []byte(planContent), 0644)

		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "rev-parse" {
					return "", nil
				}
				t.Fatalf("unexpected git call after dedup: %v", args)
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout.String(), "Plan already preserved") {
			t.Errorf("stdout = %q, want 'Plan already preserved'", stdout.String())
		}
		if !strings.Contains(stdout.String(), "2026-03-11-001-duplicate-plan.md") {
			t.Errorf("stdout = %q, want existing filename", stdout.String())
		}
	})

	t.Run("duplicate different date not matched", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Same Content\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "test-plan.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()

		// Pre-populate with same content but different date prefix.
		destPlansDir := filepath.Join(projectDir, "docs", "plans")
		os.MkdirAll(destPlansDir, 0755)
		os.WriteFile(filepath.Join(destPlansDir, "2026-03-10-001-same-content.md"), []byte(planContent), 0644)

		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		var addCalled bool
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "add" {
					addCalled = true
				}
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !addCalled {
			t.Error("expected git add to be called (different date should not dedup)")
		}
	})

	t.Run("empty stdin without pointer exits silently", func(t *testing.T) {
		projectDir := t.TempDir()

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(""),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},
			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) { t.Fatal("unexpected git call"); return "", nil },
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stdout.Len() > 0 {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
	})

	t.Run("notify flag outputs prompt without preserving", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Feature X\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "notify-test.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s"}`, planFile)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:   strings.NewReader(inputJSON),
			Stdout:  &stdout,
			Stderr:  &stderr,
			Env:     func(string) string { return "" },

			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) { t.Fatal("unexpected git call in notify mode"); return "", nil },
			Notify:  true,
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout.String(), "Feature X") {
			t.Errorf("stdout = %q, want plan title", stdout.String())
		}
		if !strings.Contains(stdout.String(), "/preserve") {
			t.Errorf("stdout = %q, want slash command reference", stdout.String())
		}
		if !strings.Contains(stdout.String(), "additionalContext") {
			t.Errorf("stdout = %q, want additionalContext for Claude", stdout.String())
		}
		// hookEventName is required by the Claude Code hook schema whenever
		// hookSpecificOutput is present.
		if !strings.Contains(stdout.String(), `"hookEventName":"PostToolUse"`) {
			t.Errorf("stdout = %q, want hookEventName=PostToolUse in hookSpecificOutput", stdout.String())
		}
	})

	t.Run("notify writes pending-plan pointer in .git/", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Pointer Plan\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "pointer-test.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)

		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) { t.Fatal("unexpected git call in notify mode"); return "", nil },
			Notify:  true,
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		pointerFile := filepath.Join(projectDir, ".git", "pk-pending-plan")
		data, err := os.ReadFile(pointerFile)
		if err != nil {
			t.Fatalf("pointer file not written: %v", err)
		}
		got := strings.TrimSpace(string(data))
		if got != planFile {
			t.Errorf("pointer content = %q, want %q", got, planFile)
		}
	})

	t.Run("skill invocation reads pointer under race", func(t *testing.T) {
		// Regression guard: pointer must win even when a newer rival plan exists.
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		approvedContent := "# Approved Plan\n\nThe plan Session A approved and expects to preserve."
		approvedFile := filepath.Join(plansDir, "approved.md")
		os.WriteFile(approvedFile, []byte(approvedContent), 0644)
		older := time.Now().Add(-time.Hour)
		os.Chtimes(approvedFile, older, older)

		rivalContent := "# Rival Plan\n\nA plan written by Session B in a different project after Session A's approval."
		rivalFile := filepath.Join(plansDir, "rival.md")
		os.WriteFile(rivalFile, []byte(rivalContent), 0644)

		projectDir := t.TempDir()
		os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
		pointerFile := filepath.Join(projectDir, ".git", "pk-pending-plan")
		os.WriteFile(pointerFile, []byte(approvedFile+"\n"), 0644)

		var stdout, stderr bytes.Buffer
		var commitMsg string
		cfg := Config{
			Stdin:  strings.NewReader(""), // skill invocation: no stdin
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "commit" {
					commitMsg = args[2]
				}
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(commitMsg, "Approved Plan") {
			t.Errorf("commit message = %q, want to contain 'Approved Plan'", commitMsg)
		}
		if strings.Contains(commitMsg, "Rival Plan") {
			t.Errorf("commit message = %q, must not contain 'Rival Plan'", commitMsg)
		}
		if _, err := os.Stat(pointerFile); !os.IsNotExist(err) {
			t.Errorf("pointer file still exists after successful preserve: err=%v", err)
		}
	})

	t.Run("stale pointer target missing exits silently", func(t *testing.T) {
		projectDir := t.TempDir()
		os.MkdirAll(filepath.Join(projectDir, ".git"), 0755)
		pointerFile := filepath.Join(projectDir, ".git", "pk-pending-plan")
		os.WriteFile(pointerFile, []byte("/does/not/exist.md\n"), 0644)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(""),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},
			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) { t.Fatal("unexpected git call"); return "", nil },
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stdout.Len() > 0 {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
		if _, err := os.Stat(pointerFile); !os.IsNotExist(err) {
			t.Errorf("stale pointer still exists: err=%v", err)
		}
	})

	t.Run("dry-run previews without writing or committing", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Dry Run Plan\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "dry-run-test.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) {
				// Only rev-parse should be called; no add/commit/push.
				return "", nil
			},
			DryRun: true,
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		// Verify preview output on stderr.
		output := stderr.String()
		if !strings.Contains(output, "Dry Run Plan") {
			t.Errorf("stderr = %q, want plan title", output)
		}
		if !strings.Contains(output, "docs/plans/2026-03-11-001-dry-run-plan.md") {
			t.Errorf("stderr = %q, want destination path", output)
		}
		if !strings.Contains(output, "plan: Dry Run Plan [skip ci]") {
			t.Errorf("stderr = %q, want commit message preview", output)
		}

		// Verify no directory or file was created.
		destDir := filepath.Join(projectDir, "docs", "plans")
		if _, err := os.Stat(destDir); err == nil {
			t.Error("dry-run should not create the docs/plans/ directory")
		}
		destFile := filepath.Join(destDir, "2026-03-11-001-dry-run-plan.md")
		if _, err := os.Stat(destFile); err == nil {
			t.Error("dry-run should not write the plan file")
		}

		// Verify no stdout (no systemMessage).
		if stdout.Len() > 0 {
			t.Errorf("stdout = %q, want empty in dry-run mode", stdout.String())
		}
	})

	t.Run("update notice appended to systemMessage", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Update Test\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "update-test.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				return "", nil
			},
			CheckUpdate: func() string {
				return "Update available: pk v1.0.0 → v2.0.0"
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := stdout.String()
		if !strings.Contains(out, "Approved plan committed:") {
			t.Errorf("stdout = %q, want commit message", out)
		}
		if !strings.Contains(out, "Update available") {
			t.Errorf("stdout = %q, want update notice", out)
		}
	})

	t.Run("root home directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planFile := filepath.Join(plansDir, "idempotent-singing-cocoa.md")
		planContent := "# Test Plan from Web Session\n\nThis plan was created in a remote web session with root home directory."
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		var addPath string

		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "add" {
					addPath = args[1]
				}
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		wantPath := "docs/plans/2026-03-11-001-test-plan-from-web-session.md"
		if addPath != wantPath {
			t.Errorf("git add path = %q, want %q", addPath, wantPath)
		}

		destFile := filepath.Join(projectDir, wantPath)
		content, err := os.ReadFile(destFile)
		if err != nil {
			t.Fatalf("plan file not written: %v", err)
		}
		if string(content) != planContent {
			t.Errorf("content mismatch")
		}
	})

	t.Run("json object tool_response with plan path", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planFile := filepath.Join(plansDir, "my-plan.md")
		planContent := "# JSON Response Plan\n\nThis plan tests extraction from a JSON object tool_response."
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		// tool_response is a JSON object, not a string.
		inputJSON := fmt.Sprintf(`{"tool_response":{"planPath":"%s","approved":true},"cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		var addPath string

		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "add" {
					addPath = args[1]
				}
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		wantPath := "docs/plans/2026-03-11-001-json-response-plan.md"
		if addPath != wantPath {
			t.Errorf("git add path = %q, want %q", addPath, wantPath)
		}
	})

	t.Run("push failure reports local commit", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Push Fail Plan\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "push-fail.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				if args[0] == "push" {
					return "", fmt.Errorf("permission denied")
				}
				return "", nil
			},
			Push: true,
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := stdout.String()
		if !strings.Contains(out, "committed locally but push failed") {
			t.Errorf("stdout = %q, want push failure message", out)
		}
	})

	t.Run("dry-run with push shows push preview", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Dry Push Plan\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "dry-push.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) {
				return "", nil
			},
			DryRun: true,
			Push:   true,
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		output := stderr.String()
		if !strings.Contains(output, "Push:") {
			t.Errorf("stderr = %q, want push preview line", output)
		}
		if !strings.Contains(output, "git push origin HEAD") {
			t.Errorf("stderr = %q, want push command preview", output)
		}
	})

	t.Run("getwd fallback for project directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Getwd Plan\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "getwd-test.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		// No cwd in input, no CLAUDE_PROJECT_DIR — forces Getwd fallback.
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s"}`, planFile)

		var stdout, stderr bytes.Buffer
		var addPath string
		cfg := Config{
			Stdin:   strings.NewReader(inputJSON),
			Stdout:  &stdout,
			Stderr:  &stderr,
			Env:     func(string) string { return "" },

			Now:     func() time.Time { return fixedTime },
			Getwd:   func() (string, error) { return projectDir, nil },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "add" {
					addPath = args[1]
				}
				if args[0] == "diff" && args[1] == "--cached" {
					return "", fmt.Errorf("changes exist")
				}
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		wantPath := "docs/plans/2026-03-11-001-getwd-plan.md"
		if addPath != wantPath {
			t.Errorf("git add path = %q, want %q", addPath, wantPath)
		}

		destFile := filepath.Join(projectDir, wantPath)
		if _, err := os.Stat(destFile); err != nil {
			t.Errorf("plan file not written at expected path: %v", err)
		}
	})

	t.Run("not a git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Not Git Plan\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "not-git.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},

			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "rev-parse" {
					return "", fmt.Errorf("not a git repository")
				}
				t.Fatal("unexpected git call after rev-parse failure")
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stderr.String(), "not a git repository") {
			t.Errorf("stderr = %q, want git repo error", stderr.String())
		}
		if stdout.Len() > 0 {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
	})

	t.Run("no project directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# No Dir Plan\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "no-dir.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		// No cwd in input, no CLAUDE_PROJECT_DIR, Getwd fails.
		inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s"}`, planFile)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:   strings.NewReader(inputJSON),
			Stdout:  &stdout,
			Stderr:  &stderr,
			Env:     func(string) string { return "" },

			Now:     func() time.Time { return fixedTime },
			Getwd:   func() (string, error) { return "", fmt.Errorf("no working directory") },
			GitExec: func(string, ...string) (string, error) {
				t.Fatal("unexpected git call when no project directory")
				return "", nil
			},
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stderr.String(), "could not determine project directory") {
			t.Errorf("stderr = %q, want project directory error", stderr.String())
		}
		if stdout.Len() > 0 {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
	})

	t.Run("json object tool_response without plan path exits silently", func(t *testing.T) {
		projectDir := t.TempDir()
		inputJSON := fmt.Sprintf(`{"tool_response":{"status":"approved","message":"Plan complete"},"cwd":"%s"}`, projectDir)

		var stdout, stderr bytes.Buffer
		cfg := Config{
			Stdin:  strings.NewReader(inputJSON),
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},
			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) { t.Fatal("unexpected git call"); return "", nil },
		}
		withFS(&cfg)

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if stdout.Len() > 0 {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
	})
}

func TestRun_mkdirAllFailure(t *testing.T) {
	fixedTime := time.Date(2026, 3, 11, 15, 30, 0, 0, time.Local)
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, ".claude", "plans")
	os.MkdirAll(plansDir, 0755)

	planFile := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planFile, []byte("# Test Plan\n\nEnough content to pass the minimum length check for this test."), 0644)

	projectDir := t.TempDir()
	inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

	var stderr bytes.Buffer
	cfg := Config{
		Stdin:     strings.NewReader(inputJSON),
		Stdout:    &bytes.Buffer{},
		Stderr:    &stderr,
		Env:       func(string) string { return "" },

		Now:       func() time.Time { return fixedTime },
		GitExec:   func(dir string, args ...string) (string, error) { return "", nil },
		ReadFile:  os.ReadFile,
		Stat:      os.Stat,
		ReadDir:   os.ReadDir,
		Remove:    os.Remove,
		MkdirAll:  func(string, os.FileMode) error { return fmt.Errorf("permission denied") },
		WriteFile: os.WriteFile,
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "failed to create directory") {
		t.Errorf("stderr = %q, want 'failed to create directory'", stderr.String())
	}
}

func TestRun_writeFileFailure(t *testing.T) {
	fixedTime := time.Date(2026, 3, 11, 15, 30, 0, 0, time.Local)
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, ".claude", "plans")
	os.MkdirAll(plansDir, 0755)

	planFile := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planFile, []byte("# Test Plan\n\nEnough content to pass the minimum length check for this test."), 0644)

	projectDir := t.TempDir()
	inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

	var stderr bytes.Buffer
	cfg := Config{
		Stdin:     strings.NewReader(inputJSON),
		Stdout:    &bytes.Buffer{},
		Stderr:    &stderr,
		Env:       func(string) string { return "" },

		Now:       func() time.Time { return fixedTime },
		GitExec:   func(dir string, args ...string) (string, error) { return "", nil },
		ReadFile:  os.ReadFile,
		Stat:      os.Stat,
		ReadDir:   os.ReadDir,
		Remove:    os.Remove,
		MkdirAll:  os.MkdirAll,
		WriteFile: func(string, []byte, os.FileMode) error { return fmt.Errorf("disk full") },
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "failed to write plan") {
		t.Errorf("stderr = %q, want 'failed to write plan'", stderr.String())
	}
}

func TestRun_gitAddFailure(t *testing.T) {
	fixedTime := time.Date(2026, 3, 11, 15, 30, 0, 0, time.Local)
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, ".claude", "plans")
	os.MkdirAll(plansDir, 0755)

	planFile := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planFile, []byte("# Test Plan\n\nEnough content to pass the minimum length check for this test."), 0644)

	projectDir := t.TempDir()
	inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

	var stderr bytes.Buffer
	cfg := Config{
		Stdin:   strings.NewReader(inputJSON),
		Stdout:  &bytes.Buffer{},
		Stderr:  &stderr,
		Env: func(string) string { return "" },
		Now: func() time.Time { return fixedTime },
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "add" {
				return "", fmt.Errorf("add failed")
			}
			return "", nil
		},
		ReadFile:  os.ReadFile,
		Stat:      os.Stat,
		ReadDir:   os.ReadDir,
		Remove:    os.Remove,
		MkdirAll:  os.MkdirAll,
		WriteFile: os.WriteFile,
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "git add failed") {
		t.Errorf("stderr = %q, want 'git add failed'", stderr.String())
	}
}

func TestRun_gitCommitFailure(t *testing.T) {
	fixedTime := time.Date(2026, 3, 11, 15, 30, 0, 0, time.Local)
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, ".claude", "plans")
	os.MkdirAll(plansDir, 0755)

	planFile := filepath.Join(plansDir, "test-plan.md")
	os.WriteFile(planFile, []byte("# Test Plan\n\nEnough content to pass the minimum length check for this test."), 0644)

	projectDir := t.TempDir()
	inputJSON := fmt.Sprintf(`{"tool_response":"Plan saved to %s","cwd":"%s"}`, planFile, projectDir)

	var stderr bytes.Buffer
	cfg := Config{
		Stdin:   strings.NewReader(inputJSON),
		Stdout:  &bytes.Buffer{},
		Stderr:  &stderr,
		Env: func(string) string { return "" },
		Now: func() time.Time { return fixedTime },
		GitExec: func(dir string, args ...string) (string, error) {
			if args[0] == "diff" && args[1] == "--cached" {
				return "", fmt.Errorf("changes exist")
			}
			if args[0] == "commit" {
				return "", fmt.Errorf("commit failed")
			}
			return "", nil
		},
		ReadFile:  os.ReadFile,
		Stat:      os.Stat,
		ReadDir:   os.ReadDir,
		Remove:    os.Remove,
		MkdirAll:  os.MkdirAll,
		WriteFile: os.WriteFile,
	}

	code := Run(cfg)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "git commit failed") {
		t.Errorf("stderr = %q, want 'git commit failed'", stderr.String())
	}
}

func TestScanDestDir(t *testing.T) {
	var fsCfg Config
	withFS(&fsCfg)

	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		dup, seq := scanDestDir(fsCfg, dir, "2026-03-11", []byte("content"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty", dup)
		}
		if seq != 1 {
			t.Errorf("nextSeq = %d, want 1", seq)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		dup, seq := scanDestDir(fsCfg, "/nonexistent/path", "2026-03-11", []byte("content"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty", dup)
		}
		if seq != 1 {
			t.Errorf("nextSeq = %d, want 1", seq)
		}
	})

	t.Run("single file no duplicate", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "2026-03-11-001-some-plan.md"), []byte("x"), 0644)
		dup, seq := scanDestDir(fsCfg, dir, "2026-03-11", []byte("new content"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty", dup)
		}
		if seq != 2 {
			t.Errorf("nextSeq = %d, want 2", seq)
		}
	})

	t.Run("multiple files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "2026-03-11-001-first.md"), []byte("x"), 0644)
		os.WriteFile(filepath.Join(dir, "2026-03-11-003-third.md"), []byte("x"), 0644)
		os.WriteFile(filepath.Join(dir, "2026-03-11-002-second.md"), []byte("x"), 0644)
		dup, seq := scanDestDir(fsCfg, dir, "2026-03-11", []byte("new content"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty", dup)
		}
		if seq != 4 {
			t.Errorf("nextSeq = %d, want 4", seq)
		}
	})

	t.Run("mixed dates", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "2026-03-10-005-old-date.md"), []byte("x"), 0644)
		os.WriteFile(filepath.Join(dir, "2026-03-11-002-today.md"), []byte("x"), 0644)
		dup, seq := scanDestDir(fsCfg, dir, "2026-03-11", []byte("new content"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty", dup)
		}
		if seq != 3 {
			t.Errorf("nextSeq = %d, want 3", seq)
		}
	})

	t.Run("four digit sequence", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "2026-03-11-999-plan.md"), []byte("x"), 0644)
		os.WriteFile(filepath.Join(dir, "2026-03-11-1000-plan.md"), []byte("x"), 0644)
		dup, seq := scanDestDir(fsCfg, dir, "2026-03-11", []byte("new content"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty", dup)
		}
		if seq != 1001 {
			t.Errorf("nextSeq = %d, want 1001", seq)
		}
	})

	t.Run("files without sequence numbers", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "2026-03-11-some-old-plan.md"), []byte("x"), 0644)
		dup, seq := scanDestDir(fsCfg, dir, "2026-03-11", []byte("new content"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty", dup)
		}
		if seq != 1 {
			t.Errorf("nextSeq = %d, want 1", seq)
		}
	})

	t.Run("exact duplicate", func(t *testing.T) {
		dir := t.TempDir()
		content := []byte("# Same Plan\n\nExact same content.")
		os.WriteFile(filepath.Join(dir, "2026-03-11-001-same-plan.md"), content, 0644)
		dup, seq := scanDestDir(fsCfg, dir, "2026-03-11", content)
		if dup != "2026-03-11-001-same-plan.md" {
			t.Errorf("dupName = %q, want matching filename", dup)
		}
		if seq != 2 {
			t.Errorf("nextSeq = %d, want 2", seq)
		}
	})

	t.Run("different date not matched as duplicate", func(t *testing.T) {
		dir := t.TempDir()
		content := []byte("# Same Plan\n\nExact same content.")
		os.WriteFile(filepath.Join(dir, "2026-03-10-001-same-plan.md"), content, 0644)
		dup, seq := scanDestDir(fsCfg, dir, "2026-03-11", content)
		if dup != "" {
			t.Errorf("dupName = %q, want empty (different date)", dup)
		}
		if seq != 1 {
			t.Errorf("nextSeq = %d, want 1", seq)
		}
	})

	t.Run("different size not matched as duplicate", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "2026-03-11-001-plan.md"), []byte("short"), 0644)
		dup, _ := scanDestDir(fsCfg, dir, "2026-03-11", []byte("much longer content that differs in size"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty (different size)", dup)
		}
	})
}
