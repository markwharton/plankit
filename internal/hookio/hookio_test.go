package hookio

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestReadInputNilAndEmpty(t *testing.T) {
	if _, err := ReadInput(nil); err != io.EOF {
		t.Fatalf("nil reader: %v, want io.EOF", err)
	}
	if _, err := ReadInput(strings.NewReader("not json")); err == nil {
		t.Fatal("malformed payload should error")
	}
}

func TestReadInputParsesPayload(t *testing.T) {
	in, err := ReadInput(strings.NewReader(`{"cwd":"/x","tool_input":{"command":"git push","file_path":"a.md"},"tool_response":"done"}`))
	if err != nil {
		t.Fatal(err)
	}
	if in.CWD != "/x" || in.ToolInput.Command != "git push" || in.ToolInput.FilePath != "a.md" {
		t.Fatalf("payload: %+v", in)
	}
	if in.ToolResponseString() != "done" {
		t.Fatalf("response: %q", in.ToolResponseString())
	}
}

func TestToolResponseObjectPassesThrough(t *testing.T) {
	in := Input{ToolResponse: json.RawMessage(`{"filePath":"/p.md"}`)}
	if got := in.ToolResponseString(); got != `{"filePath":"/p.md"}` {
		t.Fatalf("got %q", got)
	}
}

func TestResolveDirPrecedence(t *testing.T) {
	env := func(k string) string {
		if k == "CLAUDE_PROJECT_DIR" {
			return "/started-here"
		}
		return ""
	}
	noEnv := func(string) string { return "" }
	if got := ResolveDir(env, "/session-is-here", "/stated", true); got != "/stated" {
		t.Fatalf("an explicit project dir beats everything: %q", got)
	}
	if got := ResolveDir(env, "/session-is-here", "/default", false); got != "/session-is-here" {
		t.Fatalf("the payload's cwd beats where the session started: %q", got)
	}
	if got := ResolveDir(env, "", "/default", false); got != "/started-here" {
		t.Fatalf("CLAUDE_PROJECT_DIR fills in when the payload has no cwd: %q", got)
	}
	if got := ResolveDir(noEnv, "", "/default", false); got != "/default" {
		t.Fatalf("the process directory is the last resort: %q", got)
	}
}

func TestWritePermissionDecision(t *testing.T) {
	var b bytes.Buffer
	if err := WritePermissionDecision(&b, PermissionDeny, "because"); err != nil {
		t.Fatal(err)
	}
	var out struct {
		H struct {
			Event    string `json:"hookEventName"`
			Decision string `json:"permissionDecision"`
			Reason   string `json:"permissionDecisionReason"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(b.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.H.Event != "PreToolUse" || out.H.Decision != "deny" || out.H.Reason != "because" {
		t.Fatalf("decision: %+v", out.H)
	}
}

func TestWritePostToolUseOmitsEmptyContext(t *testing.T) {
	var b bytes.Buffer
	WritePostToolUse(&b, "saved", "")
	if strings.Contains(b.String(), "hookSpecificOutput") {
		t.Fatalf("empty context should omit hookSpecificOutput: %s", b.String())
	}
	b.Reset()
	WritePostToolUse(&b, "ready", "tell them")
	s := b.String()
	if !strings.Contains(s, `"hookEventName":"PostToolUse"`) || !strings.Contains(s, "tell them") {
		t.Fatalf("context form: %s", s)
	}
}
