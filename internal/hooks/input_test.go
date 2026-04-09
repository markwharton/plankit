package hooks

import (
	"fmt"
	"strings"
	"testing"
)

func TestReadInput(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantPath    string
		wantCommand string
		wantCWD     string
		wantErr     bool
	}{
		{
			name:     "edit with file_path",
			json:     `{"tool_input":{"file_path":"/tmp/test.md"},"cwd":"/projects/foo"}`,
			wantPath: "/tmp/test.md",
			wantCWD:  "/projects/foo",
		},
		{
			name:     "edit with spaces in JSON",
			json:     `{"tool_input": {"file_path": "/tmp/test.md"}, "cwd": "/projects/foo"}`,
			wantPath: "/tmp/test.md",
			wantCWD:  "/projects/foo",
		},
		{
			name:        "bash with command",
			json:        `{"tool_input":{"command":"git commit -m 'test'"},"cwd":"/projects/foo"}`,
			wantCommand: "git commit -m 'test'",
			wantCWD:     "/projects/foo",
		},
		{
			name:    "no tool_input",
			json:    `{"cwd":"/projects/foo"}`,
			wantCWD: "/projects/foo",
		},
		{
			name:    "empty JSON",
			json:    `{}`,
			wantCWD: "",
		},
		{
			name:    "invalid JSON",
			json:    `not json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ReadInput(strings.NewReader(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantPath != "" || tt.wantCommand != "" {
				if input.ToolInput == nil {
					t.Fatal("expected ToolInput, got nil")
				}
				if input.ToolInput.FilePath != tt.wantPath {
					t.Errorf("FilePath = %q, want %q", input.ToolInput.FilePath, tt.wantPath)
				}
				if input.ToolInput.Command != tt.wantCommand {
					t.Errorf("Command = %q, want %q", input.ToolInput.Command, tt.wantCommand)
				}
			}
			if input.CWD != tt.wantCWD {
				t.Errorf("CWD = %q, want %q", input.CWD, tt.wantCWD)
			}
		})
	}
}

func TestReadInput_emptyReader(t *testing.T) {
	// Empty reader (EOF immediately) should return an error, not block.
	_, err := ReadInput(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestReadInput_largePayload(t *testing.T) {
	// Claude Code can send large tool_response payloads (plan content, file diffs).
	// Verify ReadInput handles them without truncation.
	largeContent := strings.Repeat("x", 1<<20) // 1 MB
	payload := fmt.Sprintf(`{"tool_response":%q,"cwd":"/projects/foo"}`, largeContent)

	input, err := ReadInput(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := input.ToolResponseString()
	if len(got) != 1<<20 {
		t.Errorf("tool_response length = %d, want %d", len(got), 1<<20)
	}
	if input.CWD != "/projects/foo" {
		t.Errorf("CWD = %q, want %q", input.CWD, "/projects/foo")
	}
}

func TestToolResponseString(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "string response",
			json: `{"tool_response":"Plan saved to /tmp/plan.md"}`,
			want: "Plan saved to /tmp/plan.md",
		},
		{
			name: "object response",
			json: `{"tool_response":{"key":"value"}}`,
			want: `{"key":"value"}`,
		},
		{
			name: "null response",
			json: `{"tool_response":null}`,
			want: "",
		},
		{
			name: "missing response",
			json: `{}`,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ReadInput(strings.NewReader(tt.json))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := input.ToolResponseString()
			if got != tt.want {
				t.Errorf("ToolResponseString() = %q, want %q", got, tt.want)
			}
		})
	}
}
