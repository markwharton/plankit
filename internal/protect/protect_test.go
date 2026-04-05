package protect

import (
	"bytes"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		projectDir string
		wantBlock  bool
	}{
		{
			name:       "blocks file under docs/plans",
			json:       `{"tool_input":{"file_path":"/projects/foo/docs/plans/2024-01-01-test.md"}}`,
			projectDir: "/projects/foo",
			wantBlock:  true,
		},
		{
			name:       "allows file outside docs/plans",
			json:       `{"tool_input":{"file_path":"/projects/foo/src/index.ts"}}`,
			projectDir: "/projects/foo",
			wantBlock:  false,
		},
		{
			name:       "allows file in docs but not plans",
			json:       `{"tool_input":{"file_path":"/projects/foo/docs/README.md"}}`,
			projectDir: "/projects/foo",
			wantBlock:  false,
		},
		{
			name:       "blocks relative path under docs/plans",
			json:       `{"tool_input":{"file_path":"docs/plans/test.md"}}`,
			projectDir: "/projects/foo",
			wantBlock:  true,
		},
		{
			name:       "allows relative path outside docs/plans",
			json:       `{"tool_input":{"file_path":"src/index.ts"}}`,
			projectDir: "/projects/foo",
			wantBlock:  false,
		},
		{
			name:      "no tool_input",
			json:      `{"cwd":"/projects/foo"}`,
			wantBlock: false,
		},
		{
			name:      "empty file_path",
			json:      `{"tool_input":{"file_path":""}}`,
			wantBlock: false,
		},
		{
			name:      "no project dir",
			json:      `{"tool_input":{"file_path":"/projects/foo/docs/plans/test.md"}}`,
			wantBlock: false,
		},
		{
			name:       "uses cwd when CLAUDE_PROJECT_DIR not set",
			json:       `{"tool_input":{"file_path":"/projects/foo/docs/plans/test.md"},"cwd":"/projects/foo"}`,
			projectDir: "",
			wantBlock:  true,
		},
		{
			name:       "docs/plans itself is not blocked (no trailing content)",
			json:       `{"tool_input":{"file_path":"/projects/foo/docs/plans"}}`,
			projectDir: "/projects/foo",
			wantBlock:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cfg := Config{
				Stdin:  strings.NewReader(tt.json),
				Stdout: &stdout,
				Stderr: &stderr,
				Env: func(key string) string {
					if key == "CLAUDE_PROJECT_DIR" {
						return tt.projectDir
					}
					return ""
				},
			}

			exitCode := Run(cfg)

			if exitCode != 0 {
				t.Errorf("exit code = %d, want 0", exitCode)
			}

			hasBlock := strings.Contains(stdout.String(), `"decision":"block"`)
			if hasBlock != tt.wantBlock {
				t.Errorf("block = %v, want %v (stdout: %q)", hasBlock, tt.wantBlock, stdout.String())
			}
		})
	}
}

func TestIsUnderPlansDir(t *testing.T) {
	tests := []struct {
		name       string
		filePath   string
		projectDir string
		want       bool
	}{
		{"absolute under plans", "/p/docs/plans/test.md", "/p", true},
		{"absolute outside plans", "/p/src/index.ts", "/p", false},
		{"relative under plans", "docs/plans/test.md", "/p", true},
		{"relative outside plans", "src/index.ts", "/p", false},
		{"plans dir itself", "/p/docs/plans", "/p", false},
		{"nested under plans", "/p/docs/plans/sub/test.md", "/p", true},
		{"similar prefix not under plans", "/p/docs/plans-backup/test.md", "/p", false},
		{"path traversal attempt", "/p/docs/plans/../secrets/key.pem", "/p", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnderPlansDir(tt.filePath, tt.projectDir)
			if got != tt.want {
				t.Errorf("isUnderPlansDir(%q, %q) = %v, want %v", tt.filePath, tt.projectDir, got, tt.want)
			}
		})
	}
}
