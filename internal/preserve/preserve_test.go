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
		{"Caching Report: helimods-api", 60, "caching-report-helimods-api"},
		{"  Leading and trailing spaces  ", 60, "leading-and-trailing-spaces"},
		{"ALL CAPS TITLE", 60, "all-caps-title"},
		{"a-b-c", 60, "a-b-c"},
		{"truncated-at-boundary", 10, "truncated"},
		{"exact-len", 9, "exact-len"},
		{"", 60, ""},
		{"!!!", 60, ""},
		{"Cross-platform Claude Code hooks (staged approach)", 60, "cross-platform-claude-code-hooks-staged-approach"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := Slugify(tt.title, tt.maxLen)
			if got != tt.want {
				t.Errorf("Slugify(%q, %d) = %q, want %q", tt.title, tt.maxLen, got, tt.want)
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
			HomeDir: func() (string, error) { return tmpDir, nil },
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

		// Verify git calls.
		wantCalls := []string{
			"rev-parse --is-inside-work-tree",
			"add docs/plans/2026-03-11-001-test-plan.md",
			"diff --cached --quiet",
			"commit -m docs: preserve approved plan -- Test Plan [skip ci]",
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

		// Verify stdout has systemMessage.
		if !strings.Contains(stdout.String(), "Approved plan committed and pushed") {
			t.Errorf("stdout = %q, want systemMessage", stdout.String())
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
			HomeDir: func() (string, error) { return tmpDir, nil },
			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) { t.Fatal("unexpected git call"); return "", nil },
		}

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
			HomeDir: func() (string, error) { return tmpDir, nil },
			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				// diff --cached --quiet exits 0 when no changes.
				return "", nil
			},
		}

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(stdout.String(), "Plan unchanged") {
			t.Errorf("stdout = %q, want 'Plan unchanged'", stdout.String())
		}
	})

	t.Run("fallback to latest plan", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		// Create two plans; the second is newer.
		old := filepath.Join(plansDir, "old.md")
		os.WriteFile(old, []byte("# Old Plan\n\nOld content that is long enough to pass the check."), 0644)
		os.Chtimes(old, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour))

		newer := filepath.Join(plansDir, "newer.md")
		os.WriteFile(newer, []byte("# Newer Plan\n\nNewer content that is long enough to pass the check."), 0644)

		projectDir := t.TempDir()
		// tool_response has no plan path.
		inputJSON := fmt.Sprintf(`{"tool_response":"something else","cwd":"%s"}`, projectDir)

		var stdout, stderr bytes.Buffer
		var commitMsg string
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
			HomeDir: func() (string, error) { return tmpDir, nil },
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

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(commitMsg, "Newer Plan") {
			t.Errorf("commit message = %q, want to contain 'Newer Plan'", commitMsg)
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
			HomeDir: func() (string, error) { return tmpDir, nil },
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
			HomeDir: func() (string, error) { return tmpDir, nil },
			Now:     func() time.Time { return fixedTime },
			GitExec: func(dir string, args ...string) (string, error) {
				if args[0] == "rev-parse" {
					return "", nil
				}
				t.Fatalf("unexpected git call after dedup: %v", args)
				return "", nil
			},
		}

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
			HomeDir: func() (string, error) { return tmpDir, nil },
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

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !addCalled {
			t.Error("expected git add to be called (different date should not dedup)")
		}
	})

	t.Run("invalid stdin falls back to latest plan", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Fallback Plan\n\nThis plan has enough content to pass the minimum length check easily."
		planFile := filepath.Join(plansDir, "fallback.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()

		var stdout, stderr bytes.Buffer
		var commitMsg string
		cfg := Config{
			Stdin:  strings.NewReader(""), // empty stdin triggers fallback
			Stdout: &stdout,
			Stderr: &stderr,
			Env: func(key string) string {
				if key == "CLAUDE_PROJECT_DIR" {
					return projectDir
				}
				return ""
			},
			HomeDir: func() (string, error) { return tmpDir, nil },
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

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(commitMsg, "Fallback Plan") {
			t.Errorf("commit message = %q, want to contain 'Fallback Plan'", commitMsg)
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
			HomeDir: func() (string, error) { return tmpDir, nil },
			Now:     func() time.Time { return fixedTime },
			GitExec: func(string, ...string) (string, error) { t.Fatal("unexpected git call in notify mode"); return "", nil },
			Notify:  true,
		}

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
			HomeDir: func() (string, error) { return tmpDir, nil },
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

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := stdout.String()
		if !strings.Contains(out, "Approved plan committed and pushed") {
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
			HomeDir: func() (string, error) { return tmpDir, nil },
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
			HomeDir: func() (string, error) { return tmpDir, nil },
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

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		wantPath := "docs/plans/2026-03-11-001-json-response-plan.md"
		if addPath != wantPath {
			t.Errorf("git add path = %q, want %q", addPath, wantPath)
		}
	})

	t.Run("json object tool_response falls back to latest plan", func(t *testing.T) {
		tmpDir := t.TempDir()
		plansDir := filepath.Join(tmpDir, ".claude", "plans")
		os.MkdirAll(plansDir, 0755)

		planContent := "# Fallback from JSON\n\nThis plan tests fallback when JSON tool_response has no plan path."
		planFile := filepath.Join(plansDir, "latest.md")
		os.WriteFile(planFile, []byte(planContent), 0644)

		projectDir := t.TempDir()
		// JSON object with no .claude/plans/ path.
		inputJSON := fmt.Sprintf(`{"tool_response":{"status":"approved","message":"Plan complete"},"cwd":"%s"}`, projectDir)

		var stdout, stderr bytes.Buffer
		var commitMsg string

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
			HomeDir: func() (string, error) { return tmpDir, nil },
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

		exitCode := Run(cfg)
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if !strings.Contains(commitMsg, "Fallback from JSON") {
			t.Errorf("commit message = %q, want to contain 'Fallback from JSON'", commitMsg)
		}
	})
}

func TestScanDestDir(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		dup, seq := scanDestDir(dir, "2026-03-11", []byte("content"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty", dup)
		}
		if seq != 1 {
			t.Errorf("nextSeq = %d, want 1", seq)
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		dup, seq := scanDestDir("/nonexistent/path", "2026-03-11", []byte("content"))
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
		dup, seq := scanDestDir(dir, "2026-03-11", []byte("new content"))
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
		dup, seq := scanDestDir(dir, "2026-03-11", []byte("new content"))
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
		dup, seq := scanDestDir(dir, "2026-03-11", []byte("new content"))
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
		dup, seq := scanDestDir(dir, "2026-03-11", []byte("new content"))
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
		dup, seq := scanDestDir(dir, "2026-03-11", []byte("new content"))
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
		dup, seq := scanDestDir(dir, "2026-03-11", content)
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
		dup, seq := scanDestDir(dir, "2026-03-11", content)
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
		dup, _ := scanDestDir(dir, "2026-03-11", []byte("much longer content that differs in size"))
		if dup != "" {
			t.Errorf("dupName = %q, want empty (different size)", dup)
		}
	})
}
